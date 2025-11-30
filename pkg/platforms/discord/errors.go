package discord

import (
	"errors"
	"fmt"
	"strconv"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/bwmarrin/discordgo"
)

type discordAPIError struct {
	message string
	code    int
}

func (e discordAPIError) Disable() *lightning.ChannelDisabled {
	switch e.code {
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
	return "Discord API Error " + strconv.FormatInt(int64(e.code), 10) + ": " + e.message
}

func getError(err error, message string) error {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
			return nil
		}

		return &discordAPIError{message + ": " + restErr.Message.Message, restErr.Message.Code}
	}

	return fmt.Errorf("%s: %w", message, err)
}
