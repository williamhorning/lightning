// Package stoat provides functionality to deal with the Stoat API.
package stoat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/williamhorning/lightning/internal/v2/cache"
	"github.com/williamhorning/lightning/internal/v2/workaround"
)

// Session represents a bot session on Stoat.
type Session struct {
	MessageDeleted   chan *MessageDeleteEvent
	conn             *websocket.Conn
	Ready            chan *ReadyEvent
	MessageCreated   chan *Message
	MessageUpdated   chan *MessageUpdateEvent
	Token            string
	ChannelCache     cache.Expiring[string, Channel]
	MemberCache      cache.Expiring[string, Member]
	UserCache        cache.Expiring[string, User]
	ServerEmojiCache cache.Expiring[string, []Emoji]
	EmojiCache       cache.Expiring[string, Emoji]
	ServerCache      cache.Expiring[string, Server]
	connected        atomic.Bool
}

// Get makes a request against the Stoat API.
func Get[T any](session *Session, endpoint string, key string, cacher *cache.Expiring[string, T]) *T {
	if key != "" {
		if val, ok := cacher.Get(key); ok {
			return &val
		}
	}

	body, code, err := session.Fetch(http.MethodGet, endpoint, nil, nil, nil)
	if err != nil || code != 200 {
		return nil
	}

	defer func() {
		if err = body.Close(); err != nil {
			log.Printf("internal/stoat: failed to close body: %v\n", err)
		}
	}()

	var val T

	if err = json.NewDecoder(body).Decode(&val); err != nil {
		return nil
	}

	if key != "" {
		cacher.Set(key, val)
	}

	return &val
}

// Fetch returns a request body, status code, and/or possible error from the Stoat API.
func (s *Session) Fetch(
	method, endpoint string, data any, base *string, headers map[string][]string,
) (io.ReadCloser, int, error) {
	if base == nil {
		defaultURL := "https://api.stoat.chat/0.8"
		base = &defaultURL
	}

	var body io.Reader

	if data != nil && headers["Content-Type"][0] == "application/json" {
		payload, err := json.Marshal(data)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal body: %w", err)
		}

		body = bytes.NewBuffer(payload)
	} else if reader, ok := data.(io.Reader); ok {
		body = reader
	}

	req, err := http.NewRequestWithContext(context.Background(), method, *base+endpoint, body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create %s request for %s: %w", method, endpoint, err)
	}

	req.Header = headers
	req.Header["X-Bot-Token"] = []string{s.Token}
	req.Header["User-Agent"] = []string{"rvapi/0.8.0-rc.7"}

	resp, err := workaround.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to make %s request to %s: %w", method, endpoint, err)
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

	return session.Fetch(method, endpoint, body, nil, nil)
}
