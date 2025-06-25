package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type ErrorConfig struct {
	Code         int
	Message      string
	DisableRead  bool
	DisableWrite bool
}

var discordErrors = map[int]ErrorConfig{
	30007: {30007, "too many webhooks in channel, try deleting some", false, true},
	30058: {30058, "too many webhooks in guild, try deleting some", false, true},
	50013: {50013, "missing permissions to make webhook", false, true},
	10003: {10003, "unknown channel, disabling channel", true, true},
	10015: {10015, "unknown message, disabling channel", false, true},
	50027: {50027, "invalid webhook token, disabling channel", false, true},
	0:     {0, "unknown RESTError, not disabling channel", false, false},
}

func getError(err error, extra map[string]any, message string) error {
	if restErr, ok := err.(*discordgo.RESTError); ok {
		if restErr.Message.Code == 10008 {
			return nil
		}

		e, found := discordErrors[restErr.Message.Code]

		if !found {
			e = discordErrors[0]
			e.Code = restErr.Message.Code
		}

		return lightning.LogError(fmt.Errorf(e.Message+": %w", err), message, extra, lightning.ReadWriteDisabled{Read: e.DisableRead, Write: e.DisableWrite})
	} else {
		return lightning.LogError(fmt.Errorf("unknown error: %w", err), message, extra, lightning.ReadWriteDisabled{})
	}
}
