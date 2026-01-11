package main

import (
	"errors"
	"log"
	"sync"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

func bridgeCreate(database *database) func(*lightning.Bot, *lightning.Message) { //nolint:cyclop,revive,nolintlint
	return func(bot *lightning.Bot, message *lightning.Message) {
		bridge, err := database.getBridgeByChannel(message.ChannelID)
		if err != nil {
			log.Printf("bridge: failed to get bridge from database on create: %v\n", err)

			return
		}

		if bridge.ID == "" || bridge.getChannelDisabled(message.ChannelID).Read {
			return
		}

		repliedTo := getRepliedToMessage(database, message)
		messages := []channelMessage{{ChannelID: message.ChannelID, MessageIDs: []string{message.EventID}}}
		results := make(chan channelMessage, len(bridge.Channels))
		wait := sync.WaitGroup{}

		for _, channel := range bridge.Channels {
			if channel.ID == message.ChannelID || channel.Disabled.Write {
				continue
			}

			wait.Go(func() {
				msg := *message

				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on message create in channel %q: %v\n", channel.ID, r)
					}
				}()

				msg.ChannelID = channel.ID
				msg.RepliedTo = repliedTo.getChannelMessageIDs(channel.ID)

				resultIDs, err := bot.SendMessage(&msg, &lightning.SendOptions{
					AllowEveryonePings: bridge.Settings.AllowEveryone, ChannelData: channel.Data,
				})
				if err == nil {
					results <- channelMessage{ChannelID: channel.ID, MessageIDs: resultIDs}
				} else {
					handleError(database, &bridge, channel.ID, "create", err)
				}
			})
		}

		wait.Wait()
		close(results)

		for msg := range results {
			messages = append(messages, msg)
		}

		if err = database.createMessage(bridgeMessageCollection{
			ID: message.EventID, BridgeID: bridge.ID, Messages: messages,
		}); err != nil {
			log.Printf("bridge: failed to create message collection: %v\n", err)
		}
	}
}

func bridgeEdit(database *database) func(*lightning.Bot, *lightning.EditedMessage) {
	return func(bot *lightning.Bot, message *lightning.EditedMessage) {
		bridge, prior, found := getPriorMessage(database, &message.BaseMessage)
		if !found {
			return
		}

		repliedTo := getRepliedToMessage(database, message.Message)
		wait := sync.WaitGroup{}

		for _, channel := range bridge.Channels {
			if channel.ID == message.ChannelID || channel.Disabled.Write {
				continue
			}

			wait.Go(func() {
				channel := channel
				msg := *message.Message

				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on message edit in channel %q: %v\n", channel.ID, r)
					}
				}()

				msg.ChannelID = channel.ID
				msg.RepliedTo = repliedTo.getChannelMessageIDs(channel.ID)

				if err := bot.EditMessage(&msg, prior.getChannelMessageIDs(channel.ID),
					&lightning.SendOptions{
						AllowEveryonePings: bridge.Settings.AllowEveryone, ChannelData: channel.Data,
					}); err != nil {
					handleError(database, bridge, channel.ID, "edit", err)
				}
			})
		}

		wait.Wait()
	}
}

func bridgeDelete(database *database) func(*lightning.Bot, *lightning.BaseMessage) {
	return func(bot *lightning.Bot, message *lightning.BaseMessage) {
		bridge, prior, found := getPriorMessage(database, message)
		if !found {
			return
		}

		wait := sync.WaitGroup{}

		for _, channel := range prior.Messages {
			if bridge.getChannelDisabled(channel.ChannelID).Write || len(channel.MessageIDs) == 0 {
				continue
			}

			wait.Go(func() {
				channel := channel

				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on message delete in channel %q: %v\n", channel.ChannelID, r)
					}
				}()

				if err := bot.DeleteMessages(channel.ChannelID, channel.MessageIDs); err != nil {
					handleError(database, bridge, channel.ChannelID, "delete", err)
				}
			})
		}

		wait.Wait()

		if err := database.deleteMessage(message.EventID); err != nil {
			log.Printf("bridge: failed to delete message collection: %v\n", err)
		}
	}
}

func getPriorMessage(database *database, base *lightning.BaseMessage) (*bridge, *bridgeMessageCollection, bool) {
	bridge, err := database.getBridgeByChannel(base.ChannelID)
	if err != nil {
		log.Printf("bridge: failed to get bridge for previously sent message: %v\n", err)

		return nil, nil, false
	}

	if bridge.ID == "" || bridge.getChannelDisabled(base.ChannelID).Read {
		return nil, nil, false
	}

	prior, err := database.getMessage(base.EventID)
	if err != nil {
		log.Printf("bridge: failed to get message collection for previously sent message: %v\n", err)

		return nil, nil, false
	}

	if prior.ID == "" {
		return nil, nil, false
	}

	return &bridge, &prior, true
}

func getRepliedToMessage(database *database, msg *lightning.Message) *bridgeMessageCollection {
	if msg == nil || len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.getMessage(msg.RepliedTo[0])
	if err != nil {
		log.Printf("bridge: failed to get message collection for replies to %q: %v\n", msg.RepliedTo[0], err)

		return nil
	}

	return &repliedTo
}

func handleError(database *database, bridge *bridge, channelID, event string, err error) {
	var disabled lightning.ChannelDisabled

	disabler := new(lightning.ChannelDisabler)
	if errors.As(err, disabler) {
		if result := (*disabler).Disable(); result != nil {
			disabled = *result
		}
	}

	log.Printf("bridge: failed to %s in channel %q (in %q): %v\n", event, channelID, bridge.ID, err)

	if !disabled.Read && !disabled.Write {
		return
	}

	log.Printf("bridge: disabling channel %q: read %t write %t\n", channelID, disabled.Read, disabled.Write)

	for i, ch := range bridge.Channels {
		if ch.ID == channelID {
			bridge.Channels[i].Disabled = disabled

			break
		}
	}

	if err := database.createBridge(*bridge); err != nil {
		log.Printf("bridge: failed disabling channel %q: %v\n", channelID, err)
	}
}
