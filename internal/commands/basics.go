// Package commands implements the commands used by the Lightning bridge bot.
package commands

import (
	"log"
	"strconv"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

// HelpCommand provides `!help`.
func HelpCommand(username string) *lightning.Command {
	return &lightning.Command{
		Name:        "help",
		Description: "get help with the bot",
		Executor: func(opts *lightning.CommandOptions) {
			msg := getMessage(
				username+" help:",
				"hi! i'm "+username+" "+lightning.VERSION+"!\n\n"+
					"available commands are:\n"+
					"- `bridge`: manage bridges between channels\n"+
					"- `help`: get help with the bot\n"+
					"- `ping`: check if the bot is alive\n\n"+
					"read the [docs](https://williamhorning.eu.org/lightning) for more help",
			)

			if err := opts.Reply(msg, false); err != nil {
				log.Printf("failed to reply to help command: %v\n", err)
			}
		},
	}
}

// PingCommand provides `!ping`.
func PingCommand() *lightning.Command {
	return &lightning.Command{
		Name:        "ping",
		Description: "check if the bot is alive",
		Executor: func(opts *lightning.CommandOptions) {
			if err := opts.Reply(getMessage("Pong! 🏓 ",
				strconv.FormatInt(time.Since(*opts.Time).Milliseconds(), 10)+"ms"), false); err != nil {
				log.Printf("failed to reply to ping command: %v\n", err)
			}
		},
	}
}
