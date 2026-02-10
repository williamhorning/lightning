package main

import (
	"errors"
	"log"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

func bridgeCreate(database *database) func(*lightning.Bot, *lightning.Message) { //nolint:cyclop,revive
	return func(bot *lightning.Bot, message *lightning.Message) {
		bridge, err := database.getBridgeByChannel(message.ChannelID)
		if err != nil {
			log.Printf("bridge: failed to get bridge from database on create: %v\n", err)

			return
		}

		if bridge.ID == "" || bridge.getChannel(message.ChannelID).DisabledRead {
			return
		}

		repliedTo := getRepliedToMessage(database, message)
		messages := []channelMessage{{ChannelID: message.ChannelID, MessageIDs: []string{message.EventID}}}
		results := make(chan channelMessage, len(bridge.Channels))
		wait := sync.WaitGroup{}

		for _, channel := range bridge.Channels {
			if channel.ID == message.ChannelID || channel.DisabledWrite {
				continue
			}

			wait.Go(func() {
				msg := *message

				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on create in %q: %v %s\n", channel.ID, r, debug.Stack())
					}
				}()

				msg.ChannelID = channel.ID
				msg.RepliedTo = repliedTo.getChannel(channel.ID)

				resultIDs, err := bot.SendMessage(&msg, &lightning.SendOptions{
					AllowEveryonePings: bridge.AllowEveryone, ChannelData: channel.Data,
				})
				if err == nil {
					results <- channelMessage{ChannelID: channel.ID, MessageIDs: resultIDs}
				} else {
					handleError(database, channel.ID, "create", err)
				}
			})
		}

		wait.Wait()
		close(results)

		for msg := range results {
			messages = append(messages, msg)
		}

		if err = database.insertMessage(message.EventID, bridge.ID, messages); err != nil {
			log.Printf("bridge: failed to create message collection: %v\n", err)
		}
	}
}

func bridgeEdit(database *database) func(*lightning.Bot, *lightning.EditedMessage) {
	return func(bot *lightning.Bot, message *lightning.EditedMessage) {
		time.Sleep(150 * time.Millisecond)

		prior, err := database.getOriginalMessage(message.EventID)
		if err != nil {
			log.Printf("bridge: failed to get message collection for previously sent message: %v\n", err)

			return
		}

		if prior == nil {
			return
		}

		bridge, err := database.getBridgeByChannel(message.ChannelID)
		if err != nil {
			log.Printf("bridge: failed to get bridge from database on edit: %v\n", err)

			return
		}

		if bridge.ID == "" || bridge.getChannel(message.ChannelID).DisabledRead {
			return
		}

		repliedTo := getRepliedToMessage(database, message.Message)
		wait := sync.WaitGroup{}

		for _, prior := range prior {
			if bridge.getChannel(prior.ChannelID).DisabledWrite {
				continue
			}

			wait.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on edit in %q: %v %s\n", prior.ChannelID, r, debug.Stack())
					}
				}()

				channel := bridge.getChannel(prior.ChannelID)
				msg := *message.Message
				msg.ChannelID = prior.ChannelID
				msg.RepliedTo = repliedTo.getChannel(prior.ChannelID)

				if _, err := bot.EditMessage(&msg, prior.MessageIDs, &lightning.SendOptions{
					AllowEveryonePings: bridge.AllowEveryone, ChannelData: channel.Data,
				}); err != nil {
					handleError(database, prior.ChannelID, "edit", err)
				}
			})
		}

		wait.Wait()
	}
}

func bridgeDelete(database *database) func(*lightning.Bot, *lightning.BaseMessage) {
	return func(bot *lightning.Bot, message *lightning.BaseMessage) {
		time.Sleep(150 * time.Millisecond)

		prior, err := database.getMessage(message.EventID)
		if err != nil {
			log.Printf("bridge: failed to get message collection for previously sent message: %v\n", err)

			return
		}

		if prior == nil {
			return
		}

		wait := sync.WaitGroup{}

		for _, prior := range prior {
			if prior.ChannelID == message.ChannelID {
				prior.MessageIDs = slices.DeleteFunc(prior.MessageIDs,
					func(id string) bool { return id == message.EventID })
			}

			if len(prior.MessageIDs) == 0 {
				continue
			}

			wait.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on delete in %q: %v %s\n", prior.ChannelID, r, debug.Stack())
					}
				}()

				if err := bot.DeleteMessages(prior.ChannelID, prior.MessageIDs); err != nil {
					handleError(database, prior.ChannelID, "delete", err)
				}
			})
		}

		wait.Wait()

		if err := database.deleteMessage(message.EventID); err != nil {
			log.Printf("bridge: failed to delete message collection: %v\n", err)
		}
	}
}

func getRepliedToMessage(database *database, msg *lightning.Message) channelMessageSet {
	if msg == nil || len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.getMessage(msg.RepliedTo[0])
	if err != nil {
		log.Printf("bridge: failed to get message collection for replies to %q: %v\n", msg.RepliedTo[0], err)

		return nil
	}

	return repliedTo
}

func handleError(database *database, channelID, event string, err error) {
	var disabled lightning.ChannelDisabled

	disabler := new(lightning.ChannelDisabler)
	if errors.As(err, disabler) {
		if result := (*disabler).Disable(); result != nil {
			disabled = *result
		}
	}

	log.Printf("bridge: failed to %s in channel %q: %v\n", event, channelID, err)

	if !disabled.Read && !disabled.Write {
		return
	}

	log.Printf("bridge: disabling channel %q: read %t write %t\n", channelID, disabled.Read, disabled.Write)

	if err := database.disableChannel(channelID, disabled.Read, disabled.Write, map[string]string{}); err != nil {
		log.Printf("bridge: failed disabling channel %q: %v\n", channelID, err)
	}
}
