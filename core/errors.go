package lightning

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog"
)

var (
	Log                 = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "Jan 02 15:04:05"}).With().Timestamp().Logger().Level(zerolog.InfoLevel)
	ErrLogErrorNilError = errors.New("LogError called with nil error. Please provide a valid error")
)

type LightningError struct {
	Disable ReadWriteDisabled
	Message string
}

func (e LightningError) Error() string {
	return e.Message
}

func LogError(err error, message string, extra map[string]any, disable ReadWriteDisabled) LightningError {
	if lightningErr, ok := err.(*LightningError); ok {
		return *lightningErr
	}

	if lightningErr, ok := err.(LightningError); ok {
		return lightningErr
	}

	if err == nil {
		err = ErrLogErrorNilError
	}

	if extra == nil {
		extra = make(map[string]any)
	}

	id := ulid.Make().String()

	Log.Error().
		Str("id", id).
		Str("message", message).
		Bool("read_disabled", disable.Read).
		Bool("write_disabled", disable.Write).
		Fields(extra).
		Err(err).Msg("[lightning] error")

	fmt.Fprintf(os.Stderr, "%+v\n", err)

	if os.Getenv("LIGHTNING_ERROR_WEBHOOK") != "" {
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
			Log.Error().Err(err).Msg("Error marshaling error webhook body")
		} else {
			resp, err := http.Post(os.Getenv("LIGHTNING_ERROR_WEBHOOK"), "application/json", bytes.NewReader(body))
			if err != nil {
				Log.Error().Err(err).Msg("Error sending error webhook request")
			} else {
				resp.Body.Close()
			}
		}
	}

	return LightningError{disable, "Something went wrong! Take a look at [the docs](https://williamhorning.eu.org/lightning).\n\n```\n" + id + "\n\n" + message + "\n```"}
}
