package lightning

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
	"time"
)

// LogError logs a given error, with an addition message and extra information.
func LogError(err error, message string, extra map[string]any, disable *ChannelDisabled) BotError {
	if err == nil {
		err = nilLogError{}
	}

	var lightningErr BotError
	if errors.As(err, &lightningErr) {
		return lightningErr
	}

	if disable == nil {
		disable = &ChannelDisabled{false, false}
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

	body, err := json.Marshal(map[string]any{
		"content": botErr.message,
		"embeds": []map[string]any{
			{
				"title": errorID,
				"fields": []map[string]any{
					{
						"name":  "Channel Status",
						"value": fmt.Sprintf("Read: %t, Write: %t", botErr.Disable.Read, botErr.Disable.Write),
					},
					{"name": "Full Error", "value": fmt.Sprintf("```\n%s\n```", botErr.Unwrap().Error())},
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
