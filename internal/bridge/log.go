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
	"strconv"
	"strings"
	"time"

	"github.com/lmittmann/tint"
)

type non2xxError struct {
	code int
}

func (e *non2xxError) Error() string {
	return "received non-2xx response: " + strconv.Itoa(e.code)
}

// LogHandler is a custom slog handler that sends logs to a webhook and to tint.
type LogHandler struct {
	Parent slog.Handler
	URL    string
	group  []string
	attrs  []slog.Attr
	Level  slog.Level
}

// NewLogHandler creates a logger that deals with tint and webhooks.
func NewLogHandler(url string, level slog.Level) *LogHandler {
	return &LogHandler{tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	}), url, nil, nil, level}
}

// Enabled checks if the log level is enabled.
func (handler *LogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= handler.Level
}

// Handle handles the log record by sending it to the webhook and to tint.
func (handler *LogHandler) Handle(ctx context.Context, record slog.Record) error {
	return errors.Join(handler.webhookHandle(ctx, record, handler.URL), handler.Parent.Handle(ctx, record))
}

// WithGroup adds a group to the log handler.
func (handler *LogHandler) WithGroup(name string) slog.Handler {
	return &LogHandler{handler.Parent, handler.URL, append(handler.group, name), handler.attrs, handler.Level}
}

// WithAttrs adds attributes to the log handler.
func (handler *LogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogHandler{handler.Parent, handler.URL, handler.group, append(handler.attrs, attrs...), handler.Level}
}

func (handler *LogHandler) webhookHandle(ctx context.Context, record slog.Record, url string) error {
	if url == "" {
		return nil
	}

	if strings.Contains(record.Message, "connection reset by peer") {
		return nil
	}

	jsonData, err := json.Marshal(map[string]any{"embeds": []map[string]any{{
		"title":     strings.ToLower(record.Level.String()) + ": " + record.Message,
		"color":     0x487C7E,
		"fields":    handler.getFields(record),
		"footer":    map[string]string{"text": "lightning bridge logger"},
		"timestamp": record.Time.Format(time.RFC3339),
	}}})
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if err = resp.Body.Close(); err != nil {
		return fmt.Errorf("failed to close response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &non2xxError{resp.StatusCode}
	}

	return nil
}

func (handler *LogHandler) getFields(record slog.Record) []map[string]any {
	fields := make([]map[string]any, 0, len(handler.group)+len(handler.attrs)+record.NumAttrs())

	for _, group := range handler.group {
		fields = append(fields, map[string]any{"name": "Group", "value": "```\n\n" + group + "\n\n```", "inline": true})
	}

	for _, attr := range handler.attrs {
		fields = append(
			fields,
			map[string]any{"name": attr.Key, "value": "```\n\n" + attr.Value.String() + "\n\n```", "inline": true},
		)
	}

	record.Attrs(func(attr slog.Attr) bool {
		fields = append(
			fields,
			map[string]any{"name": attr.Key, "value": "```\n\n" + attr.Value.String() + "\n\n```", "inline": true},
		)

		return true
	})

	return fields
}
