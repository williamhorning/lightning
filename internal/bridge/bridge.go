package bridge

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func handleBridgeMessage(db Database, event string, data any) error {
	var bridge Bridge
	var err error
	var base lightning.BaseMessage

	switch msg := data.(type) {
	case lightning.Message:
		base = msg.BaseMessage
	case lightning.BaseMessage:
		base = msg
	default:
		return fmt.Errorf("unsupported message type: %T", data)
	}

	if event == "create_message" {
		bridge, err = db.getBridgeByChannel(base.ChannelID)
	} else {
		var bridgeMsg BridgeMessageCollection
		bridgeMsg, err = db.getMessage(base.EventID)
		if err == nil {
			bridge = Bridge{
				ID:       bridgeMsg.BridgeID,
				Name:     bridgeMsg.Name,
				Channels: bridgeMsg.Channels,
				Settings: bridgeMsg.Settings,
			}
			if bridgeMsg.ID != base.EventID && event == "edit_message" {
				return nil
			}
		}
	}

	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database", map[string]any{
			"channel": base.ChannelID,
			"event":   event,
		}, nil)
	}

	if bridge.ID == "" {
		return nil
	}

	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			if channel.IsDisabled().Read {
				bridgeLog.Debug("channel is subscribed, skipping", "channel", channel.ID, "plugin", channel.Plugin)
				return nil
			}
			break
		}
	}

	var channels []BridgeChannel
	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			continue
		}
		if channel.IsDisabled().Write {
			continue
		}
		channels = append(channels, channel)
	}

	if len(channels) == 0 {
		bridgeLog.Debug("No valid target channels, skipping", "bridge", bridge.ID)
		return nil
	}

	var bridgeMsg BridgeMessageCollection
	if event != "create_message" {
		bridgeMsg, err = db.getMessage(base.EventID)
		if err != nil {
			return lightning.LogError(err, "Failed to get bridge message", nil, nil)
		}
	}

	messages := make([]BridgeMessage, 0, len(channels)+1)
	var messagesMutex sync.Mutex
	var waitGroup sync.WaitGroup

	for _, channel := range channels {
		var replyIDs []string
		if msg, ok := data.(lightning.Message); ok && len(msg.RepliedTo) > 0 {
			bridgedMsg, err := db.getMessage(msg.RepliedTo[0])

			if err == nil {
				replyIDs = getMessageIDsForChannel(bridgedMsg, channel.ID, channel.Plugin)
			} else {
				bridgeLog.Warn("failed to get bridged message for reply", "replied_to", msg.RepliedTo[0], "error", err)
			}
		}

		var priorMessageIDs []string
		if event != "create_message" {
			priorMessageIDs = getMessageIDsForChannel(bridgeMsg, channel.ID, channel.Plugin)

			if len(priorMessageIDs) == 0 {
				bridgeLog.Debug("No prior message IDs found for channel %s in plugin %s, skipping", channel.ID, channel.Plugin)
				continue
			}
		}

		plugin, ok := lightning.Plugins.Get(channel.Plugin)
		if !ok {
			bridgeLog.Warn("plugin not found for channel", "channel", channel.ID, "plugin", channel.Plugin)
			continue
		}

		resultIDs := []string{}
		opts := &lightning.SendOptions{
			AllowEveryonePings: bridge.Settings.AllowEveryone,
			ChannelID:          channel.ID,
			ChannelData:        channel.Data,
		}

		waitGroup.Add(1)

		ch := channel

		go func() {
			defer waitGroup.Done()
			defer func() {
				if r := recover(); r != nil {
					lightning.LogError(fmt.Errorf("%v", r), "Panic in bridge message handling", map[string]any{
						"channel": ch.ID,
						"plugin":  ch.Plugin,
					}, nil)
				}
			}()

			var err error

			switch event {
			case "create_message":
				if msg, ok := data.(lightning.Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = ch.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
					}
					resultIDs, err = plugin.SendMessage(newMsg, opts)
				}
			case "edit_message":
				if msg, ok := data.(lightning.Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = ch.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
					}
					err = plugin.EditMessage(newMsg, priorMessageIDs, opts)
					resultIDs = priorMessageIDs
				}
			case "delete_message":
				err = plugin.DeleteMessage(priorMessageIDs, opts)
				resultIDs = priorMessageIDs
			}

			if err != nil {
				err := lightning.LogError(err, "Error handling bridge message", map[string]any{
					"channel": ch.ID,
					"plugin":  ch.Plugin,
					"bridge":  bridge.ID,
					"event":   event,
				}, nil)

				if err.Disable.Read || err.Disable.Write {
					bridgeLog.Warn("disabling channel due to error", "channel", ch.ID, "plugin", ch.Plugin, "bridge", bridge.ID, "error", err.ID, "disable_read", err.Disable.Read, "disable_write", err.Disable.Write)

					updatedChannels := make([]BridgeChannel, len(bridge.Channels))
					copy(updatedChannels, bridge.Channels)

					for i, c := range updatedChannels {
						if ch.ID == c.ID && ch.Plugin == c.Plugin {
							updatedChannels[i].Disabled = err.Disable
							break
						}
					}

					bridge.Channels = updatedChannels
					db.createBridge(bridge)

					msg := fmt.Sprintf("Disabling channel %s in bridge %s", ch.ID, bridge.ID)

					lightning.LogError(errors.New(msg), msg, map[string]any{
						"disable": map[string]bool{
							"read":  err.Disable.Read,
							"write": err.Disable.Write,
						},
					}, &err.Disable)
				}

				return
			}

			for _, id := range resultIDs {
				lightning.Plugins.SetHandled(ch.Plugin, id, strings.Replace(event, "_message", "", 1))
			}

			messagesMutex.Lock()
			messages = append(messages, BridgeMessage{
				ID:      resultIDs,
				Channel: ch.ID,
				Plugin:  ch.Plugin,
			})
			messagesMutex.Unlock()
		}()
	}

	messages = append(messages, BridgeMessage{
		ID:      []string{base.EventID},
		Channel: base.ChannelID,
		Plugin:  base.Plugin,
	})

	waitGroup.Wait()

	bridgeMsg = BridgeMessageCollection{
		Bridge: Bridge{
			ID:       base.EventID,
			Name:     bridge.Name,
			Channels: bridge.Channels,
			Settings: bridge.Settings,
		},
		BridgeID: bridge.ID,
		Messages: messages,
	}

	switch event {
	case "create_message", "edit_message":
		err := db.createMessage(bridgeMsg)
		return err
	case "delete_message":
		err := db.deleteMessage(bridgeMsg.ID)
		return err
	default:
		return fmt.Errorf("unknown event type: %s", event)
	}
}

func getMessageIDsForChannel(collection BridgeMessageCollection, channel string, plugin string) []string {
	for _, message := range collection.Messages {
		if message.Channel == channel && message.Plugin == plugin {
			return message.ID
		}
	}

	return []string{}
}
