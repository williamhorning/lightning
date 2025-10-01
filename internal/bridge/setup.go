package bridge

import (
	"log"

	"github.com/williamhorning/lightning/internal/commands"
	"github.com/williamhorning/lightning/internal/data"
	"github.com/williamhorning/lightning/pkg/lightning"
)

// Setup the bridge system with the given database.
func Setup(bot *lightning.Bot, author *lightning.MessageAuthor, database data.Database) {
	if err := bot.AddCommand(
		commands.BridgeCommand(database), commands.HelpCommand(author.Nickname), commands.PingCommand()); err != nil {
		log.Printf("bridge: failed to add commands: %v\n", err)
	}

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.Message) {
		if err := handleBridgeMessage(bot, database, "create", *event); err != nil {
			log.Printf("bridge: creation failed: %v\n\tevent: %s\n", err, event.EventID)
		}
	})

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.EditedMessage) {
		if err := handleBridgeMessage(bot, database, "edit", *event); err != nil {
			log.Printf("bridge: editing failed: %v\n\tevent: %s\n", err, event.Message.EventID)
		}
	})

	bot.AddHandler(func(_ *lightning.Bot, event *lightning.DeletedMessage) {
		if err := handleBridgeMessage(bot, database, "delete", *event); err != nil {
			log.Printf("bridge: deletion failed: %v\n\tevent: %s\n", err, event.EventID)
		}
	})

	log.Println("bridge: set up!")
}
