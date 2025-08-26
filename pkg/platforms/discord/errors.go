package discord

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type discordInvalidWebhookError struct {
	ChannelID string
}

func (discordInvalidWebhookError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (err discordInvalidWebhookError) Error() string {
	return "invalid webhook data for Discord channel: " + err.ChannelID
}

type discordAPIError struct {
	Message string
	Code    int
}

func (e discordAPIError) Disable() *lightning.ChannelDisabled {
	switch e.Code {
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
			Message: message + ": " + restErr.Message.Message,
		}

		slog.Error("discord: API error", "code", restErr.Message.Code, "message", restErr.Message.Message,
			"extra", extra, "disable", newError.Disable())

		return newError
	}

	slog.Error("discord: unknown error", "error", err, "extra", extra)

	return fmt.Errorf("discord: unknown error: %w", err)
}
