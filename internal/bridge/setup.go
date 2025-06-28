package bridge

import "github.com/williamhorning/lightning/pkg/lightning"

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
