package rvapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Get creates a request, setting the pointer to the body, and returning encountered errors.
func Get[T any](s *Session, path string, val *T) error {
	body, code, err := s.Fetch(http.MethodGet, path, nil)
	if err != nil || code != 200 {
		return err
	}

	defer func() {
		if err = body.Close(); err != nil {
			log.Printf("rvapi: failed to close body: %v\n", err)
		}
	}()

	bytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("rvapi: failed to read body: %w", err)
	}

	if err = json.Unmarshal(bytes, val); err != nil {
		return fmt.Errorf("rvapi: failed to unmarshal body: %w", err)
	}

	return nil
}

// Fetch returns a request body, status code, and/or possible error from the Stoat API.
func (s *Session) Fetch(method, endpoint string, body io.Reader) (io.ReadCloser, int, error) {
	url := "https://api.stoat.chat/0.8" + endpoint

	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		return nil, 0, fmt.Errorf("rvapi: failed to create request: %w\n\tendpoint: %s\n\tmethod: %s",
			err, endpoint, method)
	}

	req.Header["X-Bot-Token"] = []string{s.Token}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["User-Agent"] = []string{"rvapi/0.8.0-rc.7"}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("rvapi: failed to make request: %w\n\tendpoint: %s\n\tmethod: %s",
			err, endpoint, method)
	}

	if method != http.MethodGet && resp.StatusCode == http.StatusTooManyRequests {
		return handleRatelimiting(s, resp, method, endpoint, body)
	}

	return resp.Body, resp.StatusCode, nil
}

func handleRatelimiting(
	session *Session,
	resp *http.Response,
	method, endpoint string,
	body io.Reader,
) (io.ReadCloser, int, error) {
	retryAfter, ok := resp.Header["X-Ratelimit-Retry-After"]

	if !ok || len(retryAfter) == 0 {
		retryAfter = []string{"1000"}
	}

	retryAfterDuration, err := time.ParseDuration(retryAfter[0] + "ms")
	if err != nil {
		retryAfterDuration = time.Second
	}

	time.Sleep(retryAfterDuration)

	return session.Fetch(method, endpoint, body)
}
