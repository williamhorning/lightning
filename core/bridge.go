package lightning

import (
	"errors"
	"fmt"
	"strings"
)

func SetupBridge(db Database) {
	Log.Info().Msg("Setting up bridge system")
	RegisterCommand(bridgeCommand(db))

	go func() {
		for event := range ListenMessages() {
			Log.Trace().Str("event_id", event.EventID).Str("channel", event.ChannelID).Msg("Received message creation event")
			if err := handleBridgeMessage(db, "create_message", event); err != nil {
				LogError(err, "Failed to handle bridge message creation", nil, ReadWriteDisabled{})
			}
		}
	}()

	go func() {
		for event := range ListenEdits() {
			Log.Trace().Str("event_id", event.EventID).Str("channel", event.ChannelID).Msg("Received message edit event")
			if err := handleBridgeMessage(db, "edit_message", event); err != nil {
				LogError(err, "Failed to handle bridge message edit", nil, ReadWriteDisabled{})
			}
		}
	}()

	go func() {
		for event := range ListenDeletes() {
			Log.Trace().Str("event_id", event.EventID).Str("channel", event.ChannelID).Msg("Received message deletion event")
			if err := handleBridgeMessage(db, "delete_message", event); err != nil {
				LogError(err, "Failed to handle bridge message deletion", nil, ReadWriteDisabled{})
			}
		}
	}()

	Log.Info().Msg("Bridge system setup!")
}

func handleBridgeMessage(db Database, event string, data any) error {
	Log.Trace().Str("event", event).Interface("data_type", data).Msg("Handling bridge message")

	var bridge Bridge
	var err error
	var base BaseMessage

	switch msg := data.(type) {
	case Message:
		base = msg.BaseMessage
		Log.Trace().Str("channel", base.ChannelID).Str("plugin", base.Plugin).Str("event_id", base.EventID).Msg("Processing Message type")
	case BaseMessage:
		base = msg
		Log.Trace().Str("channel", base.ChannelID).Str("plugin", base.Plugin).Str("event_id", base.EventID).Msg("Processing BaseMessage type")
	default:
		return fmt.Errorf("unsupported message type: %T", data)
	}

	if event == "create_message" {
		Log.Trace().Str("channel", base.ChannelID).Str("event", event).Msg("Getting bridge by channel for new message")
		bridge, err = db.getBridgeByChannel(base.ChannelID)
	} else {
		Log.Trace().Str("event_id", base.EventID).Str("event", event).Msg("Getting bridge message collection for existing message")
		var bridgeMsg BridgeMessageCollection
		bridgeMsg, err = db.getMessage(base.EventID)
		if err == nil {
			bridge = Bridge{
				ID:       bridgeMsg.BridgeID,
				Name:     bridgeMsg.Name,
				Channels: bridgeMsg.Channels,
				Settings: bridgeMsg.Settings,
			}
			Log.Trace().Str("bridge_id", bridge.ID).Str("event_id", base.EventID).Msg("Retrieved bridge information")
		}
	}

	if err != nil {
		return LogError(err, "Failed to get bridge from database", map[string]any{
			"channel": base.ChannelID,
			"event":   event,
		}, ReadWriteDisabled{})
	}

	if bridge.ID == "" {
		Log.Trace().Str("channel", base.ChannelID).Msg("No bridge found for channel, skipping")
		return nil
	}

	Log.Trace().Str("bridge_id", bridge.ID).Str("channel", base.ChannelID).Int("channel_count", len(bridge.Channels)).Msg("Processing message for bridge")

	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			if channel.IsDisabled().Read {
				Log.Debug().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Channel is subscribed (read disabled), skipping")
				return nil
			}
			break
		}
	}

	var channels []BridgeChannel
	for _, channel := range bridge.Channels {
		if channel.ID == base.ChannelID && channel.Plugin == base.Plugin {
			Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Skipping source channel")
			continue
		}
		if channel.IsDisabled().Write {
			Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Channel has write disabled, skipping")
			continue
		}
		channels = append(channels, channel)
	}

	if len(channels) == 0 {
		Log.Debug().Str("bridge_id", bridge.ID).Msg("No valid target channels found, skipping")
		return nil
	}

	Log.Trace().Str("bridge_id", bridge.ID).Int("target_channels", len(channels)).Msg("Processing message for target channels")

	var messages []BridgeMessage

	for _, channel := range channels {
		Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Str("event", event).Msg("Processing channel")

		var priorMessageIDs []string
		if event != "create_message" {
			Log.Trace().Str("event_id", base.EventID).Msg("Looking up prior message IDs")
			bridgeMsg, _ := db.getMessage(base.EventID)
			for _, msg := range bridgeMsg.Messages {
				if msg.Channel == channel.ID && msg.Plugin == channel.Plugin {
					priorMessageIDs = msg.ID
					Log.Trace().Strs("prior_ids", priorMessageIDs).Msg("Found prior message IDs")
					break
				}
			}

			if len(priorMessageIDs) == 0 {
				if bridgeMsg.ID == base.EventID {
					Log.Trace().Str("event_id", base.EventID).Msg("Using bridge message collection ID as prior message ID")
					priorMessageIDs = []string{bridgeMsg.ID}
				} else {
					Log.Debug().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("No prior message IDs found, skipping")
					continue
				}
			}
		}

		Log.Trace().Str("plugin", channel.Plugin).Msg("Getting plugin")
		plugin, ok := GetPlugin(channel.Plugin)
		if !ok {
			Log.Debug().Str("plugin", channel.Plugin).Msg("Plugin not found, skipping channel")
			continue
		}

		var replyIDs []string
		if msg, ok := data.(Message); ok && len(msg.RepliedTo) > 0 {
			Log.Trace().Strs("replied_to", msg.RepliedTo).Msg("Processing reply chain")
			for _, replyID := range msg.RepliedTo {
				bridgedReply, err := db.getMessage(replyID)
				if err != nil {
					Log.Trace().Str("reply_id", replyID).Err(err).Msg("Failed to get bridged reply")
					continue
				}

				for _, replyMsg := range bridgedReply.Messages {
					if replyMsg.Channel == channel.ID && replyMsg.Plugin == channel.Plugin && len(replyMsg.ID) > 0 {
						replyIDs = append(replyIDs, replyMsg.ID[0])
						Log.Trace().Str("reply_id", replyMsg.ID[0]).Msg("Added reply ID")
					}
				}
			}
		}

		var resultIDs []string
		opts := &BridgeMessageOptions{
			Channel:  channel,
			Settings: bridge.Settings,
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					LogError(fmt.Errorf("%v", r), "Panic in bridge message handling", map[string]any{
						"channel": channel.ID,
						"plugin":  channel.Plugin,
					}, ReadWriteDisabled{})
				}
			}()

			var err error
			Log.Trace().Str("event", event).Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Handling event")

			switch event {
			case "create_message":
				if msg, ok := data.(Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = channel.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
						Log.Trace().Strs("reply_ids", replyIDs).Msg("Setting reply IDs for new message")
					}
					Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Msg("Sending message")
					resultIDs, err = plugin.SendMessage(newMsg, opts)
					if err == nil {
						Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("message_ids", resultIDs).Msg("Message sent successfully")
					}
				}
			case "edit_message":
				if msg, ok := data.(Message); ok {
					newMsg := msg
					newMsg.BaseMessage.ChannelID = channel.ID
					if len(replyIDs) > 0 {
						newMsg.RepliedTo = replyIDs
						Log.Trace().Strs("reply_ids", replyIDs).Msg("Setting reply IDs for edit")
					}
					Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Editing message")
					err = plugin.EditMessage(newMsg, priorMessageIDs, opts)
					resultIDs = priorMessageIDs
					if err == nil {
						Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("message_ids", resultIDs).Msg("Message edited successfully")
					}
				}
			case "delete_message":
				Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("prior_ids", priorMessageIDs).Msg("Deleting message")
				err = plugin.DeleteMessage(priorMessageIDs, opts)
				resultIDs = priorMessageIDs
				if err == nil {
					Log.Trace().Str("channel", channel.ID).Str("plugin", channel.Plugin).Strs("message_ids", resultIDs).Msg("Message deleted successfully")
				}
			}

			if err != nil {
				err := LogError(err, "Error handling bridge message", map[string]any{
					"channel": channel.ID,
					"plugin":  channel.Plugin,
					"bridge":  bridge.ID,
					"event":   event,
				}, ReadWriteDisabled{})

				if err.Disable.Read || err.Disable.Write {
					Log.Warn().Str("channel", channel.ID).Str("plugin", channel.Plugin).Bool("disable_read", err.Disable.Read).Bool("disable_write", err.Disable.Write).Msg("Disabling channel functionality due to error")

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

					LogError(errors.New(msg), msg, map[string]any{
						"disable": map[string]bool{
							"read":  err.Disable.Read,
							"write": err.Disable.Write,
						},
					}, err.Disable)
				}

				return
			}

			for _, id := range resultIDs {
				setHandled(channel.Plugin, id, strings.Replace(event, "_message", "", 1))
				Log.Trace().Str("plugin", channel.Plugin).Str("message_id", id).Msg("Marked message as handled")
			}

			messages = append(messages, BridgeMessage{
				ID:      resultIDs,
				Channel: channel.ID,
				Plugin:  channel.Plugin,
			})
		}()
	}

	messages = append(messages, BridgeMessage{
		ID:      []string{base.EventID},
		Channel: base.ChannelID,
		Plugin:  base.Plugin,
	})

	Log.Trace().Str("bridge_id", bridge.ID).Str("event", event).Int("message_count", len(messages)).Msg("Creating bridge message collection")

	bridgeMsg := BridgeMessageCollection{
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
