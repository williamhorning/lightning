package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

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
	go func() {
		if l.URL == "" {
			return
		}

		data, err := json.Marshal(map[string]string{"content": "```ansi\n" + string(output) + "\n```"})
		if err != nil {
			return
		}

		req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, l.URL, bytes.NewBuffer(data))
		if err != nil {
			return
		}

		req.Header["Content-Type"] = []string{"application/json"}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}

		if err := resp.Body.Close(); err != nil {
			return
		}
	}()

	return len(output), nil
}
