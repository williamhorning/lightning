package commands

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func sendErr(err error, msg string, opts *lightning.CommandOptions) {
	slog.Error(fmt.Errorf("command error: %w", err).Error())

	if err = opts.Reply(getMessage("something went wrong :(", "uh oh! looks like you got struck by an error: "+
		msg+"\n\n```\n"+err.Error()+"\n```\nif you think this is a bug, or need more help, see the "+
		"[docs](https://williamhorning.eu.org/lightning/bridge)"), false); err != nil {
		slog.Error(fmt.Errorf("failed to send error message: %w", err).Error())
	}
}

func getTime() *string {
	str := time.Now().Format(time.RFC3339)

	return &str
}

func getMessage(title, description string) *lightning.Message {
	color := 0x487C7E
	lightningProfileURL := "https://williamhorning.eu.org/assets/clouds.jpg"

	if title == "something went wrong :(" {
		color = 0xFF0000
	}

	return &lightning.Message{Embeds: []lightning.Embed{{
		Title:       &title,
		Description: &description,
		Color:       &color,
		Footer: &lightning.EmbedFooter{
			Text:    "powered by lightning",
			IconURL: &lightningProfileURL,
		},
		Timestamp: getTime(),
	}}}
}
