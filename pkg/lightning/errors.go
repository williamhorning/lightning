package lightning

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/oklog/ulid/v2"
)

var Log = log.NewWithOptions(os.Stderr, log.Options{TimeFormat: time.TimeOnly, Level: log.InfoLevel, ReportCaller: true})
var ErrLogErrorNilError = errors.New("LogError called with nil error. Please provide a valid error")

type ChannelDisabled struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

type LightningError struct {
	ID      string
	Disable ChannelDisabled
	Message string
}

func (e LightningError) Error() string {
	return e.Message
}

func LogError(err error, message string, extra map[string]any, disable *ChannelDisabled) LightningError {
	if err == nil {
		err = ErrLogErrorNilError
	}

	if lightningErr, ok := err.(*LightningError); ok {
		return *lightningErr
	}

	if lightningErr, ok := err.(LightningError); ok {
		return lightningErr
	}

	if disable == nil {
		disable = &ChannelDisabled{false, false}
	}

	if extra == nil {
		extra = make(map[string]any)
	}

	id := ulid.Make().String()

	Log.Error(message, "id", id, "read", disable.Read, "write", disable.Write, "error", err, "extra", extra)

	if webhook := os.Getenv("LIGHTNING_ERROR_WEBHOOK"); webhook != "" {
		body, err := json.Marshal(map[string]any{
			"content": fmt.Sprintf("Error: %s", message),
			"embeds": []map[string]any{
				{
					"title": id,
					"color": 15158332,
					"fields": []map[string]any{
						{"name": "Channel Status", "value": fmt.Sprintf("Read: %t, Write: %t", disable.Read, disable.Write), "inline": true},
						{"name": "Full Error", "value": fmt.Sprintf("```\n%s\n```", err.Error())},
					},
					"timestamp": time.Now().Format(time.RFC3339),
				},
			},
		})

		if err != nil {
			Log.Error("Error marshaling error webhook body", "error", err, "id", id)
		} else {
			resp, err := http.Post(webhook, "application/json", bytes.NewReader(body))
			if err != nil {
				Log.Error("Error sending error webhook request", "error", err, "id", id)
			} else {
				if err := resp.Body.Close(); err != nil {
					Log.Error("Error closing response body after sending error webhook", "error", err, "id", id)
				}
			}
		}
	}

	return LightningError{id, *disable, "Something went wrong! Take a look at [the docs](https://williamhorning.eu.org/lightning).\n\n```\n" + id + "\n\n" + message + "\n```"}
}
