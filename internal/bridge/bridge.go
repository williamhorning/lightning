package bridge

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func Setup(db Database) {
	lightning.Log.Info().Msg("Setting up bridge system")
	lightning.RegisterCommand(bridgeCommand(db))

	go func() {
		for event := range lightning.Plugins.ListenMessages() {
			lightning.Log.Trace().Str("event_id", event.EventID).Str("channel", event.ChannelID).Msg("Received message creation event")
			if err := handleBridgeMessage(db, "create_message", event); err != nil {
				lightning.LogError(err, "Failed to handle bridge message creation", nil, lightning.ChannelDisabled{})
			}
		}
	}()

	go func() {
		for event := range lightning.Plugins.ListenEdits() {
			lightning.Log.Trace().Str("event_id", event.EventID).Str("channel", event.ChannelID).Msg("Received message edit event")
			if err := handleBridgeMessage(db, "edit_message", event); err != nil {
				lightning.LogError(err, "Failed to handle bridge message edit", nil, lightning.ChannelDisabled{})
			}
		}
	}()

	go func() {
		for event := range lightning.Plugins.ListenDeletes() {
			lightning.Log.Trace().Str("event_id", event.EventID).Str("channel", event.ChannelID).Msg("Received message deletion event")
			if err := handleBridgeMessage(db, "delete_message", event); err != nil {
				lightning.LogError(err, "Failed to handle bridge message deletion", nil, lightning.ChannelDisabled{})
			}
		}
	}()

	lightning.Log.Info().Msg("Bridge system setup!")
}

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

		if db.hasMessage(base.EventID) {
			lightning.Log.Trace().Str("channel", base.ChannelID).Str("message", base.EventID).Msg("Skipping duplicate message")
			return nil
		}

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

	var replyChainMap map[string]map[string][]string
	if msg, ok := data.(lightning.Message); ok && len(msg.RepliedTo) > 0 {
		lightning.Log.Trace().Strs("replied_to", msg.RepliedTo).Msg("Processing reply chain")
		replyChainMap = make(map[string]map[string][]string)

		for _, replyID := range msg.RepliedTo {
			bridgedReply, err := db.getMessage(replyID)
			if err != nil {
				lightning.Log.Trace().Str("reply_id", replyID).Err(err).Msg("Failed to get bridged reply")
				continue
			}

			for _, replyMsg := range bridgedReply.Messages {
				if len(replyMsg.ID) == 0 {
					continue
				}

				if replyChainMap[replyMsg.Plugin] == nil {
					replyChainMap[replyMsg.Plugin] = make(map[string][]string)
				}

				replyChainMap[replyMsg.Plugin][replyMsg.Channel] = append(
					replyChainMap[replyMsg.Plugin][replyMsg.Channel],
					replyMsg.ID[0],
				)

				lightning.Log.Trace().
					Str("plugin", replyMsg.Plugin).
					Str("channel", replyMsg.Channel).
					Str("reply_id", replyMsg.ID[0]).
					Msg("Mapped reply ID")
			}
		}
	}

	var messagesMutex sync.Mutex

	for _, channel := range channels {
		lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Str("event", event).Msg("Processing channel")

		var priorMessageIDs []string
		if event != "create_message" {
			for _, msg := range bridgeMsg.Messages {
				if msg.Channel == channel.ID && msg.Plugin == channel.Plugin {
					priorMessageIDs = msg.ID
					break
				}
			}

			if len(priorMessageIDs) == 0 {
				if bridgeMsg.ID == base.EventID {
					lightning.Log.Trace().Str("event_id", base.EventID).Msg("Using bridge message collection ID as prior message ID")
					priorMessageIDs = []string{bridgeMsg.ID}
				} else {
					lightning.Log.Debug().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("No prior message IDs found, skipping")
					continue
				}
			}
		}

		lightning.Log.Trace().Str("plugin", channel.Plugin).Msg("Getting plugin")
		plugin, ok := lightning.Plugins.Get(channel.Plugin)
		if !ok {
			lightning.Log.Debug().Str("plugin", channel.Plugin).Msg("Plugin not found, skipping channel")
			continue
		}

		var replyIDs []string
		if pluginMap, ok := replyChainMap[channel.Plugin]; ok {
			if channelReplies, ok := pluginMap[channel.ID]; ok && len(channelReplies) > 0 {
				replyIDs = channelReplies
				lightning.Log.Trace().
					Str("channel", channel.ID).
					Str("plugin", channel.Plugin).
					Strs("reply_ids", replyIDs).
					Msg("Using cached reply IDs")
			}
		}

		var resultIDs []string
		opts := &lightning.SendOptions{
			AllowEveryonePings: bridge.Settings.AllowEveryone,
			ChannelID:          channel.ID,
			ChannelData:        channel.Data,
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					lightning.LogError(fmt.Errorf("%v", r), "Panic in bridge message handling", map[string]any{
						"channel": channel.ID,
						"plugin":  channel.Plugin,
					}, lightning.ChannelDisabled{})
				}
			}()

			var err error
			lightning.Log.Trace().Str("event", event).Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Handling event")

			switch event {
			case "create_message":
				if msg, ok := data.(lightning.Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = channel.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
						lightning.Log.Trace().Strs("reply_ids", replyIDs).Msg("Setting reply IDs for new message")
					}
					lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Sending message")
					resultIDs, err = plugin.SendMessage(newMsg, opts)
					if err == nil {
						lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("message_ids", resultIDs).Msg("Message sent successfully")
					}
				}
			case "edit_message":
				if msg, ok := data.(lightning.Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = channel.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
						lightning.Log.Trace().Strs("reply_ids", replyIDs).Msg("Setting reply IDs for edit")
					}
					lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Editing message")
					err = plugin.EditMessage(newMsg, priorMessageIDs, opts)
					resultIDs = priorMessageIDs
					if err == nil {
						lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("message_ids", resultIDs).Msg("Message edited successfully")
					}
				}
			case "delete_message":
				lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Deleting message")
				err = plugin.DeleteMessage(priorMessageIDs, opts)
				resultIDs = priorMessageIDs
				if err == nil {
					lightning.Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("message_ids", resultIDs).Msg("Message deleted successfully")
				}
			}

			if err != nil {
				err := lightning.LogError(err, "Error handling bridge message", map[string]any{
					"channel": channel.ID,
					"plugin":  channel.Plugin,
					"bridge":  bridge.ID,
					"event":   event,
				}, lightning.ChannelDisabled{})

				if err.Disable.Read || err.Disable.Write {
					lightning.Log.Warn().Str("channel", channel.ID).Str("plugin", channel.Plugin).Bool("disable_read", err.Disable.Read).Bool("disable_write", err.Disable.Write).Msg("Disabling channel functionality due to error")

					updatedChannels := make([]BridgeChannel, len(bridge.Channels))
					copy(updatedChannels, bridge.Channels)

					for i, ch := range updatedChannels {
						if ch.ID == channel.ID && ch.Plugin == channel.Plugin {
							updatedChannels[i].Disabled = err.Disable
							break
						}
					}

					bridge.Channels = updatedChannels
					db.createBridge(bridge)

					msg := fmt.Sprintf("Disabling channel %s in bridge %s", channel.ID, bridge.ID)

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
				lightning.Plugins.SetHandled(channel.Plugin, id, strings.Replace(event, "_message", "", 1))
				lightning.Log.Trace().Str("plugin", channel.Plugin).Str("message_id", id).Msg("Marked message as handled")
			}

			messagesMutex.Lock()
			messages = append(messages, BridgeMessage{
				ID:      resultIDs,
				Channel: channel.ID,
				Plugin:  channel.Plugin,
			})
			messagesMutex.Unlock()
		}()
	}

	messages = append(messages, BridgeMessage{
		ID:      []string{base.EventID},
		Channel: base.ChannelID,
		Plugin:  base.Plugin,
	})

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
