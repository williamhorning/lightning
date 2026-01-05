package discord

import (
	"errors"
	"fmt"
	"strconv"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
)

type discordAPIError struct {
	action string
	err    *discordgo.RESTError
}

func (e discordAPIError) Disable() *lightning.ChannelDisabled {
	switch e.err.Message.Code {
	case discordgo.ErrCodeUnknownChannel:
		return &lightning.ChannelDisabled{Read: true, Write: true}
	case discordgo.ErrCodeMaximumNumberOfWebhooksReached,
		discordgo.ErrCodeMissingPermissions,
		discordgo.ErrCodeUnknownWebhook,
		discordgo.ErrCodeInvalidWebhookTokenProvided:
		return &lightning.ChannelDisabled{Read: false, Write: true}
	default:
		return &lightning.ChannelDisabled{Read: false, Write: false}
	}
}

func (e discordAPIError) Error() string {
	return "failed to " + e.action + " message: Discord API Error " +
		strconv.FormatInt(int64(e.err.Message.Code), 10) + ": " + e.err.Error()
}

func getError(err error, action string) error {
	if err == nil {
		return nil
	}

	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
			return nil
		}

		return &discordAPIError{action + ": " + restErr.Message.Message, restErr}
	}

	return fmt.Errorf("failed to %s message: %w", action, err)
}
