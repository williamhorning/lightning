package discord

import (
	"errors"
	"fmt"
	"strconv"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/disgoorg/disgo/rest"
)

type snowflakeError struct {
	value   string
	disable bool
}

func (e snowflakeError) Disable() *lightning.ChannelDisabled {
	return &lightning.ChannelDisabled{Read: false, Write: e.disable}
}

func (e snowflakeError) Error() string {
	return "failed to turn into a valid snowflake: " + e.value
}

type discordAPIError struct {
	action string
	err    *rest.Error
}

func (e discordAPIError) Disable() *lightning.ChannelDisabled {
	switch e.err.Code { //nolint:exhaustive
	case rest.JSONErrorCodeUnknownChannel:
		return &lightning.ChannelDisabled{Read: true, Write: true}
	case rest.JSONErrorCodeMaximumWebhooksReached,
		rest.JSONErrorCodeLackPermissionsToPerformAction,
		rest.JSONErrorCodeUnknownWebhook,
		rest.JSONErrorCodeInvalidWebhookToken:
		return &lightning.ChannelDisabled{Read: false, Write: true}
	default:
		return &lightning.ChannelDisabled{Read: false, Write: false}
	}
}

func (e discordAPIError) Error() string {
	return "failed to " + e.action + ": Discord API Error " +
		strconv.FormatInt(int64(e.err.Code), 10) + ": " + e.err.Error()
}

func getError(err error, action string) error {
	if err == nil {
		return nil
	}

	var restErr *rest.Error
	if errors.As(err, &restErr) {
		if restErr.Code == rest.JSONErrorCodeUnknownMessage {
			return nil
		}

		return &discordAPIError{action + ": " + restErr.Message, restErr}
	}

	return fmt.Errorf("failed to %s: %w", action, err)
}
