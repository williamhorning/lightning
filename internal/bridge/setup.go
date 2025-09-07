package bridge

import (
	"fmt"
	"log/slog"

	"github.com/williamhorning/lightning/internal/commands"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// Setup the bridge system with the given database.
func Setup(bot *lightning.Bot, author *lightning.MessageAuthor, database data.Database) {
	slog.Info("bridge: setting up")

	if err := bot.AddCommand(
		commands.BridgeCommand(database), commands.HelpCommand(author.Nickname), commands.PingCommand()); err != nil {
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
