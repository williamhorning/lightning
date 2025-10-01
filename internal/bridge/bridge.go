// Package bridge implements a bridge bot based on Lightning, the framework, for Lightning, the bot.
package bridge

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func handleBridgeMessage(bot *lightning.Bot, database data.Database, event data.EventType, dat any) error {
	base := getBase(dat)

	var err error

	var bridge data.Bridge

	var priorMessage *data.BridgeMessageCollection

	if event == data.TypeCreate {
		bridge, err = database.GetBridgeByChannel(base.ChannelID)
	} else {
		var prior data.BridgeMessageCollection

		prior, err = database.GetMessage(base.EventID)
		if err == nil {
			priorMessage = &prior

			if priorMessage.ID != base.EventID && event == data.TypeEdit {
				return nil
			}

			bridge, err = database.GetBridge(prior.BridgeID)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get bridge for (%s in %s): %w", base.EventID, base.ChannelID, err)
	}

	if bridge.ID == "" {
		return nil
	}

	if bridge.GetChannelDisabled(base.ChannelID).Read {
		return nil
	}

	repliedTo := getRepliedToMessage(database, dat)
	messages := processMessage(bot, database, &bridge, event, base, dat, repliedTo, priorMessage)

	return setDatabase(database, event, base, &bridge, messages)
}

func getBase(dat any) lightning.BaseMessage {
	switch msg := dat.(type) {
	case lightning.EditedMessage:
		return msg.Message.BaseMessage
	case lightning.Message:
		return msg.BaseMessage
	case lightning.BaseMessage:
		return msg
	default:
		return lightning.BaseMessage{}
	}
}

func getMessage(dat any) lightning.Message {
	switch msg := dat.(type) {
	case lightning.EditedMessage:
		return *msg.Message
	case lightning.Message:
		return msg
	default:
		return lightning.Message{}
	}
}

func setDatabase(
	database data.Database,
	event data.EventType,
	base lightning.BaseMessage,
	bridge *data.Bridge,
	messages []data.ChannelMessage,
) error {
	switch event {
	case data.TypeCreate, data.TypeEdit:
		if err := database.CreateMessage(
			data.BridgeMessageCollection{ID: base.EventID, BridgeID: bridge.ID, Messages: messages},
		); err != nil {
			return fmt.Errorf("setDatabase failed: %w", err)
		}
	case data.TypeDelete:
		if err := database.DeleteMessage(base.EventID); err != nil {
			return fmt.Errorf("setDatabase failed: %w", err)
		}
	default:
	}

	return nil
}

func processMessage(
	bot *lightning.Bot,
	database data.Database,
	bridge *data.Bridge,
	event data.EventType,
	base lightning.BaseMessage,
	dat any,
	repliedTo *data.BridgeMessageCollection,
	priorMessage *data.BridgeMessageCollection,
) []data.ChannelMessage {
	messages := make([]data.ChannelMessage, 0, len(bridge.Channels)+1)
	results := make(chan *data.ChannelMessage, len(bridge.Channels))
	waitGroup := sync.WaitGroup{}

	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID || channel.Disabled.Write {
			continue
		}

		waitGroup.Go(func() {
			priorMessageIDs := priorMessage.GetChannelMessageIDs(channel.ID)

			if event != data.TypeCreate && len(priorMessageIDs) == 0 {
				return
			}

			message := handleChannel(bot, database, bridge, &channel, event, dat,
				repliedTo, priorMessageIDs)

			if message != nil {
				results <- message
			}
		})
	}

	waitGroup.Wait()
	close(results)

	for message := range results {
		messages = append(messages, *message)
	}

	return append(messages, data.ChannelMessage{ChannelID: base.ChannelID, MessageIDs: []string{base.EventID}})
}

func handleChannel(
	bot *lightning.Bot,
	database data.Database,
	bridge *data.Bridge,
	channel *data.BridgeChannel,
	event data.EventType,
	dat any,
	repliedTo *data.BridgeMessageCollection,
	priorMessageIDs []string,
) *data.ChannelMessage {
	defer func(channel *data.BridgeChannel) {
		if r := recover(); r != nil {
			log.Printf("bridge: panic in handling %s: %#+v", channel.ID, r)
		}
	}(channel)

	opts := &lightning.SendOptions{AllowEveryonePings: bridge.Settings.AllowEveryone, ChannelData: channel.Data}

	var err error

	resultIDs := priorMessageIDs

	switch event {
	case data.TypeCreate, data.TypeEdit:
		newMessage := getMessage(dat)
		newMessage.ChannelID = channel.ID
		newMessage.RepliedTo = repliedTo.GetChannelMessageIDs(channel.ID)

		if event == data.TypeCreate {
			resultIDs, err = bot.SendMessage(&newMessage, opts)
		} else if len(priorMessageIDs) != 0 {
			err = bot.EditMessage(&newMessage, priorMessageIDs, opts)
		}
	case data.TypeDelete:
		err = bot.DeleteMessages(channel.ID, priorMessageIDs)
	default:
	}

	if err != nil {
		handleError(database, err, channel, bridge, event)

		return nil
	}

	return &data.ChannelMessage{ChannelID: channel.ID, MessageIDs: resultIDs}
}

func getRepliedToMessage(database data.Database, dat any) *data.BridgeMessageCollection {
	msg := getMessage(dat)

	if len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.GetMessage(msg.RepliedTo[0])
	if err != nil {
		log.Printf("bridge: failed to get replied_to for %s: %v\n", msg.RepliedTo[0], err)

		return nil
	}

	return &repliedTo
}

func handleError(
	database data.Database, err error, channel *data.BridgeChannel, bridge *data.Bridge, event data.EventType,
) {
	var disabled lightning.ChannelDisabled

	disabler := new(lightning.ChannelDisabler)
	if errors.As(err, disabler) {
		if result := (*disabler).Disable(); result != nil {
			disabled = *result
		}
	}

	log.Printf("bridge: error in channel %s in bridge %s on %s: %v\n", channel.ID, bridge.ID, event, err)

	if !disabled.Read && !disabled.Write {
		return
	}

	for idx, channelData := range bridge.Channels {
		if channelData.ID == channel.ID {
			bridge.Channels[idx].Disabled = disabled

			break
		}
	}

	log.Printf("bridge: disabling channel %s in bridge %s on %s\n\tdisable: %#+v\n",
		bridge.ID, channel.ID, event, disabled)

	if err := database.CreateBridge(*bridge); err != nil {
		log.Printf("bridge: failed to disable %s in bridge %s: %v\n", channel.ID, bridge.ID, err)
	}
}
