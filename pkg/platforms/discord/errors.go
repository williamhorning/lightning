package discord

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type discordInvalidWebhookError struct{}

func (discordInvalidWebhookError) Error() string {
	return "invalid webhook data for Discord channel"
}

type discordAPIError struct {
	Message string
	Disable lightning.ChannelDisabled
	Code    int
}

func (e *discordAPIError) Error() string {
	return "Discord API Error " + strconv.Itoa(e.Code) + ": " + e.Message
}

func getError(err error, extra map[string]any, message string) error {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
			return nil
		}

		newError := &discordAPIError{
			Code:    restErr.Message.Code,
			Disable: lightning.ChannelDisabled{},
		}

		switch restErr.Message.Code {
		case discordgo.ErrCodeUnknownChannel:
			newError.Disable.Read = true
			newError.Message = "unknown channel, disabling channel"
		case discordgo.ErrCodeMaximumNumberOfWebhooksReached:
			newError.Disable.Write = true
			newError.Message = "too many webhooks in channel, try deleting some"
		case discordgo.ErrCodeMissingPermissions:
			newError.Disable.Write = true
			newError.Message = "missing permissions to make webhook"
		case discordgo.ErrCodeUnknownWebhook:
			newError.Disable.Write = true
			newError.Message = "unknown message, disabling channel"
		case discordgo.ErrCodeInvalidWebhookTokenProvided:
			newError.Disable.Write = true
			newError.Message = "invalid webhook token, disabling channel"
		default:
			newError.Message = "unknown RESTError, not disabling channel"
		}

		return lightning.LogError(
			newError,
			message,
			extra,
			&newError.Disable,
		)
	}

	return lightning.LogError(fmt.Errorf("unknown error: %w", err), message, extra, nil)
}
