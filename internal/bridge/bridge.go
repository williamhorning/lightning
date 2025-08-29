// Package bridge implements a bridge bot based on Lightning, the framework, for Lightning, the bot.
package bridge

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func handleBridgeMessage(bot *lightning.Bot, database Database, event eventType, data any) error {
	base, err := getBase(data)
	if err != nil {
		return err
	}

	var bridgeData bridge

	var priorMessage *bridgeMessageCollection

	if event == typeCreate {
		bridgeData, err = database.getBridgeByChannel(base.ChannelID)
	} else {
		var prior bridgeMessageCollection

		prior, err = database.getMessage(base.EventID)
		if err == nil {
			priorMessage = &prior

			if priorMessage.ID != base.EventID && event == typeEdit {
				return nil
			}

			bridgeData, err = database.getBridge(prior.BridgeID)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to get bridge for (%s in %s): %w", base.EventID, base.ChannelID, err)
	}

	if bridgeData.ID == "" {
		return nil
	}

	if bridgeData.getChannelDisabled(base.ChannelID).Read {
		slog.Debug("bridge: channel is subscribed, skipping", "channel", base.ChannelID)

		return nil
	}

	repliedTo := getRepliedToMessage(database, data)
	messages := processMessage(bot, database, &bridgeData, event, base, data, repliedTo, priorMessage)

	return setDatabase(database, event, base, &bridgeData, messages)
}

func getBase(data any) (lightning.BaseMessage, error) {
	switch msg := data.(type) {
	case lightning.EditedMessage:
		return msg.Message.BaseMessage, nil
	case lightning.Message:
		return msg.BaseMessage, nil
	case lightning.BaseMessage:
		return msg, nil
	default:
		return lightning.BaseMessage{}, unsupportedTypeError{data}
	}
}

func getMessage(data any) (lightning.Message, error) {
	switch msg := data.(type) {
	case lightning.EditedMessage:
		return *msg.Message, nil
	case lightning.Message:
		return msg, nil
	default:
		return lightning.Message{}, unsupportedTypeError{data}
	}
}

func setDatabase(
	database Database,
	event eventType,
	base lightning.BaseMessage,
	bridgeData *bridge,
	messages channelMessageArray,
) error {
	switch event {
	case typeCreate, typeEdit:
		return database.createMessage(bridgeMessageCollection{base.EventID, bridgeData.ID, messages})
	case typeDelete:
		return database.deleteMessage(base.EventID)
	default:
		return unsupportedTypeError{event}
	}
}

func processMessage(
	bot *lightning.Bot,
	database Database,
	bridgeData *bridge,
	event eventType,
	base lightning.BaseMessage,
	data any,
	repliedTo *bridgeMessageCollection,
	priorMessage *bridgeMessageCollection,
) channelMessageArray {
	messages := make(channelMessageArray, 0, len(bridgeData.Channels)+1)
	results := make(chan *channelMessage, len(bridgeData.Channels))
	waitGroup := sync.WaitGroup{}

	for _, channel := range bridgeData.Channels {
		if channel.ID == base.ChannelID || channel.Disabled.Write {
			continue
		}

		waitGroup.Go(func() {
			priorMessageIDs := priorMessage.getChannelMessageIDs(channel.ID)

			if event != typeCreate && len(priorMessageIDs) == 0 {
				return
			}

			message := handleChannel(bot, database, bridgeData, &channel, event, data,
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

	return append(messages, channelMessage{base.ChannelID, []string{base.EventID}})
}

func handleChannel(
	bot *lightning.Bot,
	database Database,
	bridgeData *bridge,
	channel *bridgeChannel,
	event eventType,
	data any,
	repliedTo *bridgeMessageCollection,
	priorMessageIDs []string,
) *channelMessage {
	defer func(channel *bridgeChannel) {
		if r := recover(); r != nil {
			slog.Error("bridge: panic in handling", "recover", r, "channel", channel.ID)
		}
	}(channel)

	opts := &lightning.SendOptions{AllowEveryonePings: bridgeData.Settings.AllowEveryone, ChannelData: channel.Data}

	var err error

	resultIDs := priorMessageIDs

	switch event {
	case typeCreate, typeEdit:
		newMessage, msgErr := getMessage(data)
		if msgErr != nil {
			slog.Warn(fmt.Errorf("unsupported message type for bridge channel: %w", msgErr).Error(),
				"channel", channel.ID, "event", event)

			return nil
		}

		newMessage.ChannelID = channel.ID
		newMessage.RepliedTo = repliedTo.getChannelMessageIDs(channel.ID)

		if event == typeCreate {
			resultIDs, err = bot.SendMessage(&newMessage, opts)
		} else if len(priorMessageIDs) != 0 {
			err = bot.EditMessage(&newMessage, priorMessageIDs, opts)
		}
	case typeDelete:
		err = bot.DeleteMessages(channel.ID, priorMessageIDs)
	default:
	}

	if err != nil {
		handleError(database, err, channel, bridgeData, event)

		return nil
	}

	return &channelMessage{channel.ID, resultIDs}
}

func getRepliedToMessage(database Database, data any) *bridgeMessageCollection {
	msg, err := getMessage(data)
	if err != nil || len(msg.RepliedTo) == 0 {
		return nil
	}

	repliedTo, err := database.getMessage(msg.RepliedTo[0])
	if err != nil {
		slog.Warn(fmt.Errorf("failed to get replied to message: %w", err).Error(), "replied_to", msg.RepliedTo[0])

		return nil
	}

	return &repliedTo
}

func handleError(database Database, err error, channel *bridgeChannel, bridgeData *bridge, event eventType) {
	disabled := &lightning.ChannelDisabled{}

	if disabler, ok := err.(lightning.ChannelDisabler); ok {
		disabled = disabler.Disable()
	}

	slog.Error(fmt.Errorf("error handling bridge message: %w", err).Error(),
		"channel", channel.ID, "bridge", bridgeData.ID, "event", event, "disable", disabled)

	if !disabled.Read && !disabled.Write {
		return
	}

	for idx, channelData := range bridgeData.Channels {
		if channelData.ID == channel.ID {
			bridgeData.Channels[idx].Disabled = *disabled

			break
		}
	}

	slog.Warn(disableChannelError{bridgeData.ID, channel.ID}.Error(), "event", event, "disable", disabled)

	if err := database.createBridge(*bridgeData); err != nil {
		slog.Warn(fmt.Errorf("bridge: failed to disable channel: %w", err).Error(),
			"channel", channel.ID, "bridge", bridgeData.ID)
	}
}
