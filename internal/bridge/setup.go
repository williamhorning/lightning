package bridge

import (
	"cmp"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

// Setup the bridge system with the given database.
func Setup(bot *lightning.Bot, database Database) {
	slog.Info("bridge: setting up")

	if err := cmp.Or(
		bot.AddCommand(&lightning.Command{
			Name:        "help",
			Description: "get help with the bot",
			Executor: func(_ lightning.CommandOptions) string {
				return "hi, i'm lightning " + lightning.VERSION + "! [docs](https://williamhorning.eu.org/lightning/)"
			},
		}),

		bot.AddCommand(&lightning.Command{
			Name:        "ping",
			Description: "check if the bot is alive",
			Executor: func(options lightning.CommandOptions) string {
				return "Pong! 🏓 " + strconv.FormatInt(time.Since(*options.Time).Milliseconds(), 10) + "ms"
			},
		}),

		bot.AddCommand(bridgeCommand(database)),
	); err != nil {
		slog.Error(fmt.Errorf("bridge: failed to add commands: %w", err).Error())
	}

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.Message) {
		if err := handleBridgeMessage(bot, database, "create", *event); err != nil {
			slog.Error(fmt.Errorf("bridge: creation failed: %w\n\tevent: %s", err, event.EventID).Error())
		}
	})

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.EditedMessage) {
		if err := handleBridgeMessage(bot, database, "edit", *event); err != nil {
			slog.Error(fmt.Errorf("bridge: editing failed: %w\n\tevent: %s", err, event.Message.EventID).Error())
		}
	})

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.DeletedMessage) {
		if err := handleBridgeMessage(bot, database, "delete", *event); err != nil {
			slog.Error(fmt.Errorf("bridge: deletion failed: %w\n\tevent: %s", err, event.EventID).Error())
		}
	})

	slog.Info("bridge: set up!")
}
