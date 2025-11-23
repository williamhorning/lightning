package app

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
)

// SetupLogging creates a logger that deals with color and webhooks.
func SetupLogging(url string) {
	log.SetFlags(log.Ltime | log.Lshortfile)
	log.SetPrefix("")
	log.SetOutput(io.MultiWriter(os.Stderr, &webhookLogger{url}))
}

type webhookLogger struct {
	url string
}

func (l *webhookLogger) Write(output []byte) (int, error) {
	go func() {
		if l.url == "" {
			return
		}

		data, err := json.Marshal(map[string]string{"content": "```ansi\n" + string(output) + "\n```"})
		if err != nil {
			return
		}

		req, err := http.NewRequest(http.MethodPost, l.url, bytes.NewBuffer(data))
		if err != nil {
			return
		}

		req.Header["Content-Type"] = []string{"application/json"}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}

		defer resp.Body.Close()
	}()

	return len(output), nil
}
