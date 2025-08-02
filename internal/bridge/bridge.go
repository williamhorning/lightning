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
		return lightning.LogError(err, "failed to get bridge", map[string]any{"base": base}, nil)
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
		return msg.Message, nil
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
	messageMutex := sync.Mutex{}
	waitGroup := sync.WaitGroup{}

	for _, channel := range bridgeData.Channels {
		if channel.ID == base.ChannelID || channel.Disabled.Write {
			continue
		}

		waitGroup.Add(1)

		go func(channelCopy *bridgeChannel) {
			priorMessageIDs := priorMessage.getChannelMessageIDs(channelCopy.ID)

			if event != typeCreate && len(priorMessageIDs) == 0 {
				return
			}

			defer waitGroup.Done()

			message := handleChannel(bot, database, bridgeData, channelCopy, event, data,
				repliedTo, priorMessageIDs)

			if message != nil {
				messageMutex.Lock()
				defer messageMutex.Unlock()

				messages = append(messages, *message)
			}
		}(&channel)
	}

	waitGroup.Wait()

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
			slog.Error("bridge: panic in handling", "err", lightning.LogError(fmt.Errorf("%v", r), //nolint:err113
				"panic in bridge handling", map[string]any{"channel": channel.ID}, nil))
		}
	}(channel)

	opts := &lightning.SendOptions{AllowEveryonePings: bridgeData.Settings.AllowEveryone, ChannelData: channel.Data}

	var err error

	resultIDs := priorMessageIDs

	switch event {
	case typeCreate, typeEdit:
		newMessage, msgErr := getMessage(data)
		if msgErr != nil {
			slog.Warn("unsupported message type for bridge channel", "err", lightning.LogError(
				err, "unsupported message type for bridge channel",
				map[string]any{"channel": channel.ID, "event": event}, nil))

			return nil
		}

		newMessage.ChannelID = channel.ID
		newMessage.RepliedTo = repliedTo.getChannelMessageIDs(channel.ID)

		if event == typeCreate {
			resultIDs, err = bot.SendMessage(newMessage, opts)
		} else if len(priorMessageIDs) != 0 {
			err = bot.EditMessage(newMessage, priorMessageIDs, opts)
		}
	case typeDelete:
		err = bot.DeleteMessages(channel.ID, priorMessageIDs)
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
		slog.Warn("bridge: failed to get replied to message", "err", lightning.LogError(
			err, "Failed to get replied to message", map[string]any{"replied_to": msg.RepliedTo[0]}, nil))

		return nil
	}

	return &repliedTo
}

func handleError(database Database, err error, channel *bridgeChannel, bridgeData *bridge, event eventType) {
	lightningErr := lightning.LogError(err, "error handling bridge message",
		map[string]any{"channel": channel.ID, "bridge": bridgeData.ID, "event": event}, nil)
	if !lightningErr.Disable.Read && !lightningErr.Disable.Write {
		return
	}

	for idx, channelData := range bridgeData.Channels {
		if channelData.ID == channel.ID {
			bridgeData.Channels[idx].Disabled = *lightningErr.Disable

			break
		}
	}

	msg := disableChannelError{bridgeData.ID, channel.ID}

	slog.Warn("bridge: disabling channel due to error", "channel", channel.ID,
		"bridge", bridgeData.ID, "event", event, "err",
		lightning.LogError(msg, msg.Error(), map[string]any{"disable": lightningErr.Disable}, lightningErr.Disable))

	if err := database.createBridge(*bridgeData); err != nil {
		slog.Warn("bridge: failed to disable channel", "err", lightning.LogError(
			err, "Failed to update bridge with disabled channel", map[string]any{
				"bridge": bridgeData.ID, "channel": channel.ID,
			}, nil))
	}
}
