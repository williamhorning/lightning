package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

type non2xxError struct {
	code int
}

func (e *non2xxError) Error() string {
	return "received non-2xx response: " + strconv.Itoa(e.code)
}

// SetupLogging creates a logger that deals with color and webhooks.
func SetupLogging() *WebhookLogger {
	log.SetFlags(log.Ltime | log.Lshortfile)

	log.SetPrefix("")

	instance := &WebhookLogger{}

	log.SetOutput(io.MultiWriter(os.Stderr, instance))

	return instance
}

// WebhookLogger is a custom log handler that sends logs to a webhook.
type WebhookLogger struct {
	URL string
}

func (l *WebhookLogger) Write(output []byte) (int, error) {
	if l.URL == "" {
		return len(output), nil
	}

	data, err := json.Marshal(map[string]string{"content": "```ansi\n" + string(output) + "\n```"})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal log: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, l.URL, bytes.NewBuffer(data))
	if err != nil {
		return 0, fmt.Errorf("failed to create log request: %w", err)
	}

	req.Header["Content-Type"] = []string{"application/json"}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send log request: %w", err)
	}

	if err = resp.Body.Close(); err != nil {
		return 0, fmt.Errorf("failed to close log response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, &non2xxError{resp.StatusCode}
	}

	return len(output), nil
}
