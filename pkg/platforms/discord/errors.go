package discord

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
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
	return "Discord API Error " + strconv.Itoa(e.code) + ": " + e.message
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
