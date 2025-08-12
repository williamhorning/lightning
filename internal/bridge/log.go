package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

// BotError is the wrapper for any error encountered by a [Bot].
type BotError struct {
	disable *lightning.ChannelDisabled

	underlying error
	message    string
}

// Disable implements the lightning.ChannelDisabler interface for BotError.
func (botErr BotError) Disable() *lightning.ChannelDisabled {
	return botErr.disable
}

func (botErr BotError) Error() string {
	return botErr.message
}

func (botErr BotError) Unwrap() error {
	return botErr.underlying
}

// LogError logs a given error, with an addition message and extra information.
func LogError(err error, message string, extra map[string]any, disable *lightning.ChannelDisabled) BotError {
	var lightningErr BotError
	if errors.As(err, &lightningErr) {
		return lightningErr
	}

	if disabler, ok := err.(lightning.ChannelDisabler); ok {
		disable = disabler.Disable()
	}

	if disable == nil {
		disable = &lightning.ChannelDisabled{Read: false, Write: false}
	}

	errorID := time.Now().Format("15:04:05.000000")

	slog.Error("lightning error: "+message,
		"id", errorID, "read", disable.Read, "write", disable.Write, "error", err, "extra", extra)

	debug.PrintStack()

	lightningError := BotError{
		disable,
		err,
		"Something went wrong!\n\n```\n" + errorID + "\n\n" + message + "\n```",
	}

	webhookLog(errorID, lightningError)

	return lightningError
}

func webhookLog(errorID string, botErr BotError) {
	webhook := os.Getenv("LIGHTNING_ERROR_WEBHOOK")

	if webhook == "" {
		return
	}

	if botErr.underlying == nil {
		botErr.underlying = fmt.Errorf("underlying is nil: %w", errors.ErrUnsupported)
	}

	body, err := json.Marshal(map[string]any{
		"content": botErr.message,
		"embeds": []map[string]any{
			{
				"title": errorID,
				"fields": []map[string]any{
					{
						"name": "Channel Status",
						"value": "Read: " + strconv.FormatBool(botErr.disable.Read) +
							", Write: " + strconv.FormatBool(botErr.disable.Write),
					},
					{"name": "Full Error", "value": "```\n" + botErr.underlying.Error() + "\n```"},
				},
			},
		},
	})
	if err != nil {
		slog.Error("lightning: failed to marshal webhook body", "error", err, "id", errorID)

		return
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhook,
		bytes.NewReader(body),
	)
	if err != nil {
		slog.Error("lightning: failed to create webhook request", "error", err, "id", errorID)

		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("lightning: error sending webhook request", "error", err, "id", errorID)

		return
	}

	if err := resp.Body.Close(); err != nil {
		slog.Error("lightning: error closing error webhook body", "error", err, "id", errorID)
	}
}
