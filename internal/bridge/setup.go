package bridge

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

// Setup the bridge system with the given database.
func Setup(bot *lightning.Bot, database Database) {
	slog.Info("bridge: setting up")

	bot.AddCommand(lightning.Command{
		Name:        "help",
		Description: "get help with the bot",
		Executor: func(_ lightning.CommandOptions) (string, error) {
			return "hi, i'm lightning " + lightning.VERSION + "! [docs](https://williamhorning.eu.org/lightning/)", nil
		},
	})

	bot.AddCommand(lightning.Command{
		Name:        "ping",
		Description: "check if the bot is alive",
		Executor: func(options lightning.CommandOptions) (string, error) {
			return fmt.Sprintf("Pong! 🏓 %dms", (time.Since(options.Time)).Milliseconds()), nil
		},
	})

	bot.AddCommand(bridgeCommand(database))

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.Message) {
		if err := handleBridgeMessage(bot, database, "create", *event); err != nil {
			slog.Error("bridge: message creation failed", "error", err, "event", event.EventID)
		}
	})

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.EditedMessage) {
		if err := handleBridgeMessage(bot, database, "edit", *event); err != nil {
			slog.Error("bridge: message editing failed", "error", err, "event", event.Message.EventID)
		}
	})

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.DeletedMessage) {
		if err := handleBridgeMessage(bot, database, "delete", *event); err != nil {
			slog.Error("bridge: message deletion failed", "error", err, "event", event.EventID)
		}
	})

	slog.Info("bridge: set up!")
}
