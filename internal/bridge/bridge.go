// Package bridge implements a bridge bot based on Lightning, the framework, for Lightning, the bot.
package bridge

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func handleBridgeMessage(bot *lightning.Bot, database data.Database, event data.EventType, dat any) error {
	base, err := getBase(dat)
	if err != nil {
		return err
	}

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
		slog.Debug("bridge: channel is subscribed, skipping", "channel", base.ChannelID)

		return nil
	}

	repliedTo := getRepliedToMessage(database, dat)
	messages := processMessage(bot, database, &bridge, event, base, dat, repliedTo, priorMessage)

	return setDatabase(database, event, base, &bridge, messages)
}

func getBase(dat any) (lightning.BaseMessage, error) {
	switch msg := dat.(type) {
	case lightning.EditedMessage:
		return msg.Message.BaseMessage, nil
	case lightning.Message:
		return msg.BaseMessage, nil
	case lightning.BaseMessage:
		return msg, nil
	default:
		return lightning.BaseMessage{}, unsupportedTypeError{dat}
	}
}

func getMessage(dat any) (lightning.Message, error) {
	switch msg := dat.(type) {
	case lightning.EditedMessage:
		return *msg.Message, nil
	case lightning.Message:
		return msg, nil
	default:
		return lightning.Message{}, unsupportedTypeError{dat}
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
		return unsupportedTypeError{event}
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
			slog.Error("bridge: panic in handling", "recover", r, "channel", channel.ID)
		}
	}(channel)

	opts := &lightning.SendOptions{AllowEveryonePings: bridge.Settings.AllowEveryone, ChannelData: channel.Data}

	var err error

	resultIDs := priorMessageIDs

	switch event {
	case data.TypeCreate, data.TypeEdit:
		newMessage, msgErr := getMessage(dat)
		if msgErr != nil {
			slog.Warn(fmt.Errorf("unsupported message type for bridge channel: %w", msgErr).Error(),
				"channel", channel.ID, "event", event)

			return nil
		}

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
	msg, err := getMessage(dat)
	if err != nil || len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.GetMessage(msg.RepliedTo[0])
	if err != nil {
		slog.Warn(fmt.Errorf("failed to get replied to message: %w", err).Error(), "replied_to", msg.RepliedTo[0])

		return nil
	}

	return &repliedTo
}

func handleError(
	database data.Database,
	err error,
	channel *data.BridgeChannel,
	bridge *data.Bridge,
	event data.EventType,
) {
	disabled := &lightning.ChannelDisabled{}

	if disabler, ok := err.(lightning.ChannelDisabler); ok {
		disabled = disabler.Disable()
	}

	slog.Error(fmt.Errorf("error handling bridge message: %w", err).Error(),
		"channel", channel.ID, "bridge", bridge.ID, "event", event, "disable", disabled)

	if !disabled.Read && !disabled.Write {
		return
	}

	for idx, channelData := range bridge.Channels {
		if channelData.ID == channel.ID {
			bridge.Channels[idx].Disabled = *disabled

			break
		}
	}

	slog.Warn(disableChannelError{bridge.ID, channel.ID}.Error(), "event", event, "disable", disabled)

	if err := database.CreateBridge(*bridge); err != nil {
		slog.Warn(fmt.Errorf("bridge: failed to disable channel: %w", err).Error(),
			"channel", channel.ID, "bridge", bridge.ID)
	}
}
