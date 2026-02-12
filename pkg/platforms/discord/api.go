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
	bot.waitIfRateLimited(method + endpoint)

	req, err := http.NewRequest(method, "https://"+bot.apiHost+"/api/v"+bot.version+endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to make %s %s request: %w", method, endpoint, err)
	}

	req.Header.Add("Authorization", "Bot "+bot.token)
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("User-Agent", "DiscordBot (https://williamhorn.ing/lightning, 0.8.8)")

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

	if resp.StatusCode == http.StatusTooManyRequests && retry < 4 {
		return handleRetry(resp, bot, method, endpoint, body, contentType, out, retry)
	}

	aerr := apiError{Request: req, Response: resp}
	if err = json.NewDecoder(resp.Body).Decode(&aerr); err != nil {
		aerr.Code = apiErrorCode(resp.StatusCode)
		aerr.Message = err.Error()
	}

	return aerr
}

func handleRetry(
	resp *http.Response, bot *client, method string, endpoint string, body io.ReadSeeker,
	contentType string, out any, retry int,
) error {
	var res ratelimitResponse

	_ = json.NewDecoder(resp.Body).Decode(&res)

	delay := func() time.Duration {
		if res.RetryAfter > 0 {
			return time.Duration(res.RetryAfter * float64(time.Second))
		}

		if h := resp.Header.Get("Retry-After"); h != "" {
			if v, err := strconv.ParseFloat(h, 64); err == nil {
				return time.Duration(v * float64(time.Second))
			}
		}

		if h := resp.Header.Get("X-Ratelimit-Reset-After"); h != "" {
			if v, err := strconv.ParseFloat(h, 64); err == nil {
				return time.Duration(v * float64(time.Second))
			}
		}

		return 1 * time.Second
	}()

	reset := time.Now().Add(delay)

	if res.Global || resp.Header.Get("X-Ratelimit-Scope") == "global" {
		bot.rateMu.RLock()
		bot.rate = reset
		bot.rateMu.RUnlock()
	} else {
		bot.routeResets.Set(method+endpoint, reset)
	}

	if body != nil {
		if _, err := body.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to rewind request body on retry %s %s: %w", method, endpoint, err)
		}
	}

	time.Sleep(delay)

	return bot.makeRequest(method, endpoint, body, contentType, out, retry+1)
}

func (bot *client) waitIfRateLimited(bucket string) {
	for {
		now := time.Now()

		bot.rateMu.Lock()
		globalReset := bot.rate
		bot.rateMu.Unlock()

		var sleep time.Duration

		if globalReset.After(now) {
			sleep = time.Until(globalReset)
		}

		if reset, ok := bot.routeResets.Get(bucket); ok && reset.After(now) {
			if d := time.Until(reset); d > sleep {
				sleep = d
			}
		}

		if sleep <= 0 {
			return
		}

		time.Sleep(sleep)
	}
}
