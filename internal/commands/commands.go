package commands

import (
	"log"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

func sendErr(err error, msg string, opts *lightning.CommandOptions) {
	log.Printf("command error: %v\n", err)

	if err = opts.Reply(getMessage("something went wrong :(", "uh oh! looks like you got struck by an error: "+
		msg+"\n\n```\n"+err.Error()+"\n```\nif you think this is a bug, or need more help, see the "+
		"[docs](https://williamhorning.dev/lightning/bridge)"), false); err != nil {
		log.Printf("failed to reply with error to command: %v\n", err)
	}
}

func getTime() *string {
	str := time.Now().Format(time.RFC3339)

	return &str
}

func getMessage(title, description string) *lightning.Message {
	color := 0x487C7E
	lightningProfileURL := "https://williamhorning.dev/assets/lightning.png"

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
