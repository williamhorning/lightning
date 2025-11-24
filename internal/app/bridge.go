// Package app defines the lightning bridge application
package app

import (
	"errors"
	"log"
	"sync"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// Create handles and bridges new messages.
func Create(database data.Database) func(*lightning.Bot, *lightning.Message) { //nolint:cyclop,revive
	return func(bot *lightning.Bot, message *lightning.Message) {
		bridge, err := database.GetBridgeByChannel(message.ChannelID)
		if err != nil {
			log.Printf("failed to get bridge from database on create: %v\n", err)

			return
		}

		if bridge.ID == "" || bridge.GetChannelDisabled(message.ChannelID).Read {
			return
		}

		repliedTo := getRepliedToMessage(database, message)
		messages := make([]data.ChannelMessage, 0, len(bridge.Channels)+1)
		results := make(chan data.ChannelMessage, len(bridge.Channels))
		wait := sync.WaitGroup{}

		for _, channel := range bridge.Channels {
			if channel.ID == message.ChannelID || channel.Disabled.Write {
				continue
			}

			wait.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on create in channel %s: %#+v", channel.ID, r)
					}
				}()

				msg := *message
				msg.ChannelID = channel.ID
				msg.RepliedTo = repliedTo.GetChannelMessageIDs(channel.ID)

				resultIDs, err := bot.SendMessage(&msg, &lightning.SendOptions{
					AllowEveryonePings: bridge.Settings.AllowEveryone, ChannelData: channel.Data,
				})
				if err == nil {
					results <- data.ChannelMessage{ChannelID: channel.ID, MessageIDs: resultIDs}
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

		if err = database.CreateMessage(data.BridgeMessageCollection{
			ID: message.EventID, BridgeID: bridge.ID, Messages: append(messages, data.ChannelMessage{
				ChannelID: message.ChannelID, MessageIDs: []string{message.EventID},
			}),
		}); err != nil {
			log.Printf("failed to set message collection in bridge_messages on create: %v\n", err)
		}
	}
}

// Edit handles and bridges message edits.
func Edit(database data.Database) func(*lightning.Bot, *lightning.EditedMessage) {
	return func(bot *lightning.Bot, message *lightning.EditedMessage) {
		bridge, prior, found := getPriorMessage(database, &message.Message.BaseMessage)
		if !found {
			return
		}

		repliedTo := getRepliedToMessage(database, message.Message)
		wait := sync.WaitGroup{}

		for _, channel := range bridge.Channels {
			if channel.ID == message.Message.ChannelID || channel.Disabled.Write {
				continue
			}

			wait.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on edit in channel %s: %#+v", channel.ID, r)
					}
				}()

				msg := *message.Message

				msg.ChannelID = channel.ID
				msg.RepliedTo = repliedTo.GetChannelMessageIDs(channel.ID)

				if err := bot.EditMessage(&msg, prior.GetChannelMessageIDs(channel.ID),
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

// Delete handles and bridges message deletion.
func Delete(database data.Database) func(*lightning.Bot, *lightning.BaseMessage) {
	return func(bot *lightning.Bot, message *lightning.BaseMessage) {
		bridge, prior, found := getPriorMessage(database, message)
		if !found {
			return
		}

		wait := sync.WaitGroup{}

		for _, channel := range prior.Messages {
			if bridge.GetChannelDisabled(channel.ChannelID).Write || len(channel.MessageIDs) == 0 {
				continue
			}

			wait.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("bridge: panic on delete in channel %s: %#+v", channel.ChannelID, r)
					}
				}()

				if err := bot.DeleteMessages(channel.ChannelID, channel.MessageIDs); err != nil {
					handleError(database, bridge, channel.ChannelID, "delete", err)
				}
			})
		}

		wait.Wait()

		if err := database.DeleteMessage(message.EventID); err != nil {
			log.Printf("failed to set delete collection in bridge_messages on delete: %v\n", err)
		}
	}
}

func getPriorMessage(
	database data.Database, base *lightning.BaseMessage,
) (*data.Bridge, *data.BridgeMessageCollection, bool) {
	bridge, err := database.GetBridgeByChannel(base.ChannelID)
	if err != nil {
		log.Printf("failed to get bridge from database on delete: %v\n", err)

		return nil, nil, false
	}

	if bridge.ID == "" || bridge.GetChannelDisabled(base.ChannelID).Read {
		return nil, nil, false
	}

	prior, err := database.GetMessage(base.ChannelID)
	if err != nil {
		log.Printf("failed to get prior from database on delete: %v\n", err)

		return nil, nil, false
	}

	if prior.ID == "" || len(prior.Messages) < 2 {
		return nil, nil, false
	}

	return &bridge, &prior, true
}

func getRepliedToMessage(database data.Database, msg *lightning.Message) *data.BridgeMessageCollection {
	if msg == nil || len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.GetMessage(msg.RepliedTo[0])
	if err != nil {
		log.Printf("bridge: failed to get replied_to for %s: %v\n", msg.RepliedTo[0], err)

		return nil
	}

	return &repliedTo
}

func handleError(database data.Database, bridge *data.Bridge, channelID, event string, err error) {
	var disabled lightning.ChannelDisabled

	disabler := new(lightning.ChannelDisabler)
	if errors.As(err, disabler) {
		if result := (*disabler).Disable(); result != nil {
			disabled = *result
		}
	}

	log.Printf("bridge: in bridge %s on %s: %v\n", bridge.ID, event, err)

	if !disabled.Read && !disabled.Write {
		return
	}

	for i, ch := range bridge.Channels {
		if ch.ID == channelID {
			bridge.Channels[i].Disabled = disabled

			break
		}
	}

	log.Printf("bridge: disabling channel %s in bridge %s on %s: %#+v\n", bridge.ID, channelID, event, disabled)

	if err := database.CreateBridge(*bridge); err != nil {
		log.Printf("bridge: failed to disable %s in bridge %s: %v\n", channelID, bridge.ID, err)
	}
}
