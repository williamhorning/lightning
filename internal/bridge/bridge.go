package bridge

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func handleBridgeMessage(db Database, event string, data any) error {
	lightning.Log.Trace().Str("event", event).Interface("data_type", data).Msg("Handling bridge message")

	var bridge Bridge
	var err error
	var base lightning.BaseMessage

	switch msg := data.(type) {
	case lightning.Message:
		base = msg.BaseMessage
		lightning.Log.Trace().Str("channel", base.ChannelID).Str("plugin", base.Plugin).Str("event_id", base.EventID).Msg("Processing Message type")
	case lightning.BaseMessage:
		base = msg
		lightning.Log.Trace().Str("channel", base.ChannelID).Str("plugin", base.Plugin).Str("event_id", base.EventID).Msg("Processing BaseMessage type")
	default:
		return fmt.Errorf("unsupported message type: %T", data)
	}

	if event == "create_message" {
		lightning.Log.Trace().Str("channel", base.ChannelID).Str("event", event).Msg("Getting bridge by channel for new message")
		bridge, err = db.getBridgeByChannel(base.ChannelID)
	} else {
		lightning.Log.Trace().Str("event_id", base.EventID).Str("event", event).Msg("Getting bridge message collection for existing message")
		var bridgeMsg BridgeMessageCollection
		bridgeMsg, err = db.getMessage(base.EventID)
		if err == nil {
			bridge = Bridge{
				ID:       bridgeMsg.BridgeID,
				Name:     bridgeMsg.Name,
				Channels: bridgeMsg.Channels,
				Settings: bridgeMsg.Settings,
			}
			lightning.Log.Trace().Str("bridge_id", bridge.ID).Str("event_id", base.EventID).Msg("Retrieved bridge information")
			if bridgeMsg.ID != base.EventID && event == "edit_message" {
				lightning.Log.Trace().Str("event_id", base.EventID).Msg("Message is not the original, skipping bridge edit handling")
				return nil
			}
		}
	}

	if err != nil {
		return lightning.LogError(err, "Failed to get bridge from database", map[string]any{
			"channel": base.ChannelID,
			"event":   event,
		}, lightning.ChannelDisabled{})
	}

	if bridge.ID == "" {
		lightning.Log.Trace().Str("channel", base.ChannelID).Msg("No bridge found for channel, skipping")
		return nil
	}

	lightning.Log.Trace().Str("bridge_id", bridge.ID).Str("channel", base.ChannelID).Int("channel_count", len(bridge.Channels)).Msg("Processing message for bridge")

	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			if channel.IsDisabled().Read {
				lightning.Log.Debug().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Channel is subscribed (read disabled), skipping")
				return nil
			}
			break
		}
	}

	var channels []BridgeChannel
	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Skipping source channel")
			continue
		}
		if channel.IsDisabled().Write {
			lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Channel has write disabled, skipping")
			continue
		}
		channels = append(channels, channel)
	}

	if len(channels) == 0 {
		lightning.Log.Debug().Str("bridge_id", bridge.ID).Msg("No valid target channels found, skipping")
		return nil
	}

	lightning.Log.Trace().Str("bridge_id", bridge.ID).Int("target_channels", len(channels)).Msg("Processing message for target channels")

	var bridgeMsg BridgeMessageCollection
	if event != "create_message" {
		bridgeMsg, err = db.getMessage(base.EventID)
		if err != nil {
			return lightning.LogError(err, "Failed to get bridge message", nil, lightning.ChannelDisabled{})
		}
	}

	messages := make([]BridgeMessage, 0, len(channels)+1)
	var messagesMutex sync.Mutex
	var waitGroup sync.WaitGroup

	for _, channel := range channels {
		lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Str("event", event).Msg("Processing channel")

		var replyIDs []string
		if msg, ok := data.(lightning.Message); ok && len(msg.RepliedTo) > 0 {
			lightning.Log.Trace().Strs("replied_to", msg.RepliedTo).Msg("Processing replyIDs from message")
			bridgedMsg, err := db.getMessage(msg.RepliedTo[0])

			if err == nil {
				replyIDs = getMessageIDsForChannel(bridgedMsg, channel.ID, channel.Plugin)
				if len(replyIDs) > 0 {
					lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("reply_ids", replyIDs).Msg("Found reply IDs for channel")
				}
			} else {
				lightning.Log.Debug().Err(err).Str("message_id", msg.RepliedTo[0]).Msg("Failed to get bridged message for reply")
			}
		}

		var priorMessageIDs []string
		if event != "create_message" {
			priorMessageIDs = getMessageIDsForChannel(bridgeMsg, channel.ID, channel.Plugin)

			if len(priorMessageIDs) == 0 {
				lightning.Log.Debug().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("No prior message IDs found, skipping")
				continue
			}

			lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Using prior message IDs")
		}

		lightning.Log.Trace().Str("plugin", channel.Plugin).Msg("Getting plugin")
		plugin, ok := lightning.Plugins.Get(channel.Plugin)
		if !ok {
			lightning.Log.Debug().Str("plugin", channel.Plugin).Msg("Plugin not found, skipping channel")
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
					}, lightning.ChannelDisabled{})
				}
			}()

			var err error
			lightning.Log.Trace().Str("event", event).Str("channel", ch.ID).Str("plugin", ch.Plugin).Msg("Handling event")

			switch event {
			case "create_message":
				if msg, ok := data.(lightning.Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = ch.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
						lightning.Log.Trace().Strs("reply_ids", replyIDs).Msg("Setting reply IDs for new message")
					}
					lightning.Log.Trace().Str("channel", ch.ID).Str("plugin", ch.Plugin).Msg("Sending message")
					resultIDs, err = plugin.SendMessage(newMsg, opts)
					if err == nil {
						lightning.Log.Trace().Str("channel", ch.ID).Str("plugin", ch.Plugin).Strs("message_ids", resultIDs).Msg("Message sent successfully")
					}
				}
			case "edit_message":
				if msg, ok := data.(lightning.Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = ch.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
						lightning.Log.Trace().Strs("reply_ids", replyIDs).Msg("Setting reply IDs for edit")
					}
					lightning.Log.Trace().Str("channel", ch.ID).Str("plugin", ch.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Editing message")
					err = plugin.EditMessage(newMsg, priorMessageIDs, opts)
					resultIDs = priorMessageIDs
					if err == nil {
						lightning.Log.Trace().Str("channel", ch.ID).Str("plugin", ch.Plugin).Strs("message_ids", resultIDs).Msg("Message edited successfully")
					}
				}
			case "delete_message":
				lightning.Log.Trace().Str("channel", ch.ID).Str("plugin", ch.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Deleting message")
				err = plugin.DeleteMessage(priorMessageIDs, opts)
				resultIDs = priorMessageIDs
				if err == nil {
					lightning.Log.Trace().Str("channel", ch.ID).Str("plugin", ch.Plugin).Strs("message_ids", resultIDs).Msg("Message deleted successfully")
				}
			}

			if err != nil {
				err := lightning.LogError(err, "Error handling bridge message", map[string]any{
					"channel": ch.ID,
					"plugin":  ch.Plugin,
					"bridge":  bridge.ID,
					"event":   event,
				}, lightning.ChannelDisabled{})

				if err.Disable.Read || err.Disable.Write {
					lightning.Log.Warn().Str("error", err.ID).Str("channel", ch.ID).Str("plugin", ch.Plugin).Bool("disable_read", err.Disable.Read).Bool("disable_write", err.Disable.Write).Msg("Disabling channel functionality due to error")

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
					}, err.Disable)
				}

				return
			}

			for _, id := range resultIDs {
				lightning.Plugins.SetHandled(ch.Plugin, id, strings.Replace(event, "_message", "", 1))
				lightning.Log.Trace().Str("plugin", ch.Plugin).Str("message_id", id).Msg("Marked message as handled")
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

	lightning.Log.Trace().Str("bridge_id", bridge.ID).Str("event", event).Int("message_count", len(messages)).Msg("Creating bridge message collection")

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
