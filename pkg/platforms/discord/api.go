package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"codeberg.org/jersey/lightning/internal/buffer"
	"codeberg.org/jersey/lightning/internal/cache"
)

func getCached[K comparable, V any](bot *client, store *cache.Expiring[K, *V], endpoint string, key K) (*V, bool) {
	if val, ok := store.Get(key); ok {
		return val, ok
	}

	var val V
	if err := bot.do("GET", endpoint, nil, &val); err != nil {
		return nil, false
	}

	store.Set(key, &val)

	return &val, true
}

func (bot *client) do(method, endpoint string, body, out any) error {
	var reader io.ReadSeeker

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body on %s %s: %w", method, endpoint, err)
		}

		reader = bytes.NewReader(data)
	}

	return bot.makeRequest(method, endpoint, reader, "application/json", out, 0)
}

func (bot *client) doMultipart(method, endpoint string, body any, files []file, out any) error {
	if len(files) == 0 {
		return bot.do(method, endpoint, body, out)
	}

	buf := new(buffer.Buffer)
	writer := multipart.NewWriter(buf)

	field, err := writer.CreateFormField("payload_json")
	if err != nil {
		return fmt.Errorf("failed to make json form field on %s %s: %w", method, endpoint, err)
	}

	if err = json.NewEncoder(field).Encode(body); err != nil {
		return fmt.Errorf("failed to marshal json body on %s %s: %w", method, endpoint, err)
	}

	for idx, file := range files {
		fileField, err := writer.CreateFormFile("files["+strconv.Itoa(idx)+"]", file.Name)
		if err != nil {
			return fmt.Errorf("failed to make form file on %s %s: %w", method, endpoint, err)
		}

		if _, err = io.Copy(fileField, file.Reader); err != nil {
			file.Cancel()

			return fmt.Errorf("failed to copy form file on %s %s: %w", method, endpoint, err)
		}

		file.Cancel()
	}

	if err = writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer on %s %s: %w", method, endpoint, err)
	}

	if _, err = buf.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to reset buffer: %w", err)
	}

	return bot.makeRequest(method, endpoint, buf, writer.FormDataContentType(), out, 0)
}

func (bot *client) makeRequest(
	method, endpoint string, body io.ReadSeeker, contentType string, out any, retry int,
) error {
	req, err := http.NewRequest(method, "https://"+bot.apiHost+"/api/v"+bot.version+endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to make %s %s request: %w", method, endpoint, err)
	}

	req.Header.Add("Authorization", "Bot "+bot.token)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("User-Agent", "DiscordBot (https://williamhorn.ing/lightning, 0.8.4)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return &apiError{Message: err.Error(), Request: req, Response: resp}
	}

	defer resp.Body.Close()

	if resp.StatusCode < 300 {
		if out != nil {
			if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
				return fmt.Errorf("failed to decode response body on %s %s: %w", method, endpoint, err)
			}
		}

		return nil
	}

	if resp.StatusCode != http.StatusTooManyRequests || retry >= 4 {
		aerr := apiError{Request: req, Response: resp}

		if err = json.NewDecoder(resp.Body).Decode(&aerr); err != nil {
			aerr.Code = apiErrorCode(resp.StatusCode)
			aerr.Message = err.Error()
		}

		return aerr
	}

	return bot.handleRetry(resp, method, endpoint, body, contentType, out, retry)
}

func (bot *client) handleRetry(
	resp *http.Response, method, endpoint string, body io.ReadSeeker, contentType string, out any, retry int,
) error {
	interval := resp.Header.Get("X-Ratelimit-Reset-After")

	duration, err := time.ParseDuration(interval + "s")
	if err != nil {
		duration = time.Second
	}

	if body != nil {
		if _, err := body.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to rewind request body on retry %s %s: %w", method, endpoint, err)
		}
	}

	time.Sleep(duration)

	return bot.makeRequest(method, endpoint, body, contentType, out, retry+1)
}
