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

			bridgeData = bridge{
				ID:       priorMessage.BridgeID,
				Name:     priorMessage.Name,
				Channels: priorMessage.Channels,
				Settings: priorMessage.Settings,
			}

			if priorMessage.ID != base.EventID && event == typeEdit {
				return nil
			}
		}
	}

	if err != nil {
		return lightning.LogError(err, "failed to get bridge", map[string]any{"base": base}, nil)
	}

	if bridgeData.ID == "" {
		return nil
	}

	if bridgeData.getChannelDisabled(base.ChannelID, base.Plugin).Read {
		slog.Debug("bridge: channel is subscribed, skipping", "channel", base.ChannelID, "plugin", base.Plugin)

		return nil
	}

	repliedTo := getRepliedToMessage(database, data)
	messages := processMessage(bot, database, &bridgeData, event, base, data, repliedTo, priorMessage)

	return setDatabase(database, event, base, &bridgeData, messages)
}

func getBase(data any) (lightning.BaseMessage, error) {
	switch msg := data.(type) {
	case lightning.EditedMessage:
		msg.Message.Plugin = "bolt-" + msg.Message.Plugin

		return msg.Message.BaseMessage, nil
	case lightning.Message:
		msg.Plugin = "bolt-" + msg.Plugin

		return msg.BaseMessage, nil
	case lightning.BaseMessage:
		msg.Plugin = "bolt-" + msg.Plugin

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
	messages []bridgeMessage,
) error {
	switch event {
	case typeCreate, typeEdit:
		return database.createMessage(bridgeMessageCollection{
			bridge: bridge{
				ID:       base.EventID,
				Name:     bridgeData.Name,
				Channels: bridgeData.Channels,
				Settings: bridgeData.Settings,
			},
			BridgeID: bridgeData.ID,
			Messages: messages,
		})
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
) []bridgeMessage {
	messages := make([]bridgeMessage, 0, len(bridgeData.Channels)+1)
	messageMutex := sync.Mutex{}
	waitGroup := sync.WaitGroup{}

	for _, channel := range bridgeData.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			continue
		}

		if channel.isDisabled().Write {
			continue
		}

		waitGroup.Add(1)

		go func(channelCopy *bridgeChannel) {
			var priorMessageIDs []string
			if event != typeCreate {
				priorMessageIDs = priorMessage.getChannelMessageIDs(channelCopy.ID, channelCopy.Plugin)

				if len(priorMessageIDs) == 0 {
					return
				}
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

	return append(messages, bridgeMessage{
		ID:      []string{base.EventID},
		Channel: base.ChannelID,
		Plugin:  base.Plugin,
	})
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
) *bridgeMessage {
	defer func(channel *bridgeChannel) {
		if r := recover(); r != nil {
			slog.Error("bridge: panic in handling", "err", lightning.LogError(fmt.Errorf("%v", r), //nolint:err113
				"panic in bridge handling", map[string]any{"channel": channel.ID, "plugin": channel.Plugin}, nil))
		}
	}(channel)

	opts := &lightning.SendOptions{AllowEveryonePings: bridgeData.Settings.AllowEveryone, ChannelData: channel.Data}

	var resultIDs []string

	var err error

	switch event {
	case typeCreate, typeEdit:
		newMessage, msgErr := getMessage(data)
		if msgErr != nil {
			slog.Warn("unsupported message type for bridge channel", "err", lightning.LogError(
				err, "unsupported message type for bridge channel",
				map[string]any{"channel": channel.ID, "plugin": channel.Plugin, "event": event}, nil))

			return nil
		}

		newMessage.ChannelID = channel.ID
		newMessage.Plugin = channel.Plugin[5:]
		newMessage.RepliedTo = repliedTo.getChannelMessageIDs(channel.ID, channel.Plugin)

		if event == typeCreate {
			resultIDs, err = bot.SendMessage(newMessage, opts)
		} else {
			resultIDs = priorMessageIDs

			if len(priorMessageIDs) != 0 {
				err = bot.EditMessage(newMessage, priorMessageIDs, opts)
			}
		}
	case typeDelete:
		resultIDs = priorMessageIDs
		err = bot.DeleteMessages(channel.Plugin[5:], channel.ID, priorMessageIDs)
	}

	if err != nil {
		handleError(database, err, channel, bridgeData, event)

		return nil
	}

	return &bridgeMessage{channel.ID, channel.Plugin, resultIDs}
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
	lightningErr := lightning.LogError(err, "error handling bridge message", map[string]any{
		"channel": channel.ID,
		"plugin":  channel.Plugin,
		"bridge":  bridgeData.ID,
		"event":   event,
	}, nil)
	if !lightningErr.Disable.Read && !lightningErr.Disable.Write {
		return
	}

	updatedChannels := make([]bridgeChannel, 0, len(bridgeData.Channels))

	for _, ch := range bridgeData.Channels {
		if ch.ID == channel.ID && ch.Plugin == channel.Plugin {
			ch.Disabled = lightningErr.Disable
		}

		updatedChannels = append(updatedChannels, ch)
	}

	bridgeData.Channels = updatedChannels

	msg := disableChannelError{bridgeData.ID, channel.ID, channel.Plugin}

	slog.Warn("bridge: disabling channel due to error", "channel", channel.ID,
		"plugin", channel.Plugin, "bridge", bridgeData.ID, "event", event, "err",
		lightning.LogError(msg, msg.Error(), map[string]any{"disable": lightningErr.Disable}, lightningErr.Disable))

	if err := database.createBridge(*bridgeData); err != nil {
		slog.Warn("bridge: failed to disable channel", "err", lightning.LogError(
			err, "Failed to update bridge with disabled channel", map[string]any{
				"bridge": bridgeData.ID, "channel": channel.ID, "plugin": channel.Plugin,
			}, nil))
	}
}
