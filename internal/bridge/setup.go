package bridge

import "github.com/williamhorning/lightning/pkg/lightning"

var bridgeLog = lightning.Log.WithPrefix("bridge")

func Setup(db Database) {
	bridgeLog.Info("Setting up bridge system")
	lightning.RegisterCommand(bridgeCommand(db))

	go func() {
		for event := range lightning.Plugins.ListenMessages() {
			if err := handleBridgeMessage(db, "create_message", event); err != nil {
				bridgeLog.Error("Failed to handle bridge message creation", "error", err, "event", event.EventID)
			}
		}
	}()

	go func() {
		for event := range lightning.Plugins.ListenEdits() {
			if err := handleBridgeMessage(db, "edit_message", event); err != nil {
				bridgeLog.Error("Failed to handle bridge message edit", "error", err, "event", event.EventID)
			}
		}
	}()

	go func() {
		for event := range lightning.Plugins.ListenDeletes() {
			if err := handleBridgeMessage(db, "delete_message", event); err != nil {
				bridgeLog.Error("Failed to handle bridge message deletion", "error", err, "event", event.EventID)
			}
		}
	}()

	bridgeLog.Info("Bridge system setup!")
}
