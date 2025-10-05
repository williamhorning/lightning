package discord

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type discordInvalidWebhookError struct {
	channelID string
}

func (discordInvalidWebhookError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: true}
}

func (err discordInvalidWebhookError) Error() string {
	return "invalid webhook data for Discord channel: " + err.channelID
}

type discordAPIError struct {
	extra   map[string]any
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
	return "Discord API Error " + strconv.Itoa(e.code) + ": " +
		fmt.Sprintf("%#+v, disable %#+v", e.extra, e.Disable()) + ": " + e.message
}

func getError(err error, extra map[string]any, message string) error {
	var restErr *discordgo.RESTError
	if errors.As(err, &restErr) {
		if restErr.Message.Code == discordgo.ErrCodeUnknownMessage {
			return nil
		}

		return &discordAPIError{extra, message + ": " + restErr.Message.Message, restErr.Message.Code}
	}

	return fmt.Errorf("discord: unknown error: %w\n\textra: %#+v", err, extra)
}
