package stoat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"codeberg.org/jersey/lightning/internal/cache"
	"github.com/gorilla/websocket"
)

type session struct {
	conn             *websocket.Conn
	messageDeleted   chan *stMessageDeleteEvent
	ready            chan *stReadyEvent
	messageCreated   chan *stMessage
	messageUpdated   chan *stMessageUpdateEvent
	token            string
	channelCache     cache.Expiring[string, stChannel]
	dmChannelCache   cache.Expiring[string, stChannel]
	memberCache      cache.Expiring[string, stMember]
	userCache        cache.Expiring[string, stUser]
	serverEmojiCache cache.Expiring[string, []stEmoji]
	emojiCache       cache.Expiring[string, stEmoji]
	serverCache      cache.Expiring[string, stServer]
	mu               sync.Mutex
}

func fetch[T any](session *session, method, endpoint, content string, data any) (*T, error) {
	var body io.Reader

	if data != nil {
		if r, ok := data.(io.Reader); ok {
			body = r
		} else {
			b, err := json.Marshal(data)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}

			body = bytes.NewBuffer(b)
		}
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header = map[string][]string{
		"Accept":      {"application/json"},
		"X-Bot-Token": {session.token},
		"User-Agent":  {"rvapi/0.8.2"},
	}

	if content != "" {
		req.Header["Content-Type"] = []string{content}
	}

	return requestLoop[T](session, req, content, data)
}

func requestLoop[T any](
	session *session, req *http.Request, content string, data any,
) (*T, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil, nil //nolint:nilnil
	case http.StatusOK, http.StatusCreated:
		var val T
		if err := json.NewDecoder(resp.Body).Decode(&val); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return &val, nil
	case http.StatusTooManyRequests:
		if content == "application/json" {
			time.Sleep(time.Second)

			return fetch[T](session, req.Method, req.URL.String(), content, data)
		}

		fallthrough
	default:
		var stoatError stError
		if err := json.NewDecoder(resp.Body).Decode(&stoatError); err != nil {
			return nil, fmt.Errorf("failed to decode error response: %w", err)
		}

		stoatError.data = data

		return nil, &stoatError
	}
}

func get[T any](session *session, endpoint, key string, cacher *cache.Expiring[string, T]) (*T, error) {
	if val, ok := cacher.Get(key); ok {
		return &val, nil
	}

	resp, err := fetch[T](session, http.MethodGet, "https://api.stoat.chat/0.8"+endpoint, "", nil)
	if err != nil {
		return nil, err
	}

	cacher.Set(key, *resp)

	return resp, nil
}

func (session *session) sendMessage(channel string, msg *stDataMessageSend) (string, error) {
	ch, err := get(session, "/channels/"+channel, channel, &session.channelCache)
	if err == nil && ch.ChannelType != "TextChannel" && msg.Masquerade != nil {
		msg.Masquerade.Colour = ""
	}

	resp, err := fetch[stMessage](session, "POST", "https://api.stoat.chat/0.8/channels/"+channel+"/messages", "", msg)
	if err != nil {
		return "", fmt.Errorf("failed to make send request: %w", err)
	}

	return resp.ID, nil
}

func (session *session) uploadFile(srcURL, filename string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}

	defer resp.Body.Close()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, resp.Body); err != nil {
		return "", fmt.Errorf("failed to copy file data to form: %w", err)
	}

	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	val, err := fetch[struct {
		ID string `json:"id"`
	}](session, http.MethodPost, "https://cdn.stoatusercontent.com/attachments", writer.FormDataContentType(), &buf)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return val.ID, nil
}
