package guilded

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func guildedMakeRequest(token, method, endpoint string, body io.Reader) (*http.Response, error) {
	url := "https://www.guilded.gg/api/v1" + endpoint

	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		wrapped := fmt.Errorf("guilded: creating request: %w\n\tendpoint: %s\n\tmethod: %s", err, endpoint, method)

		slog.Error(wrapped.Error())

		return nil, wrapped
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "lightning/"+lightning.VERSION)
	req.Header["x-guilded-bot-api-use-official-markdown"] = []string{"true"}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		wrapped := fmt.Errorf("guilded: making request: %w\n\tendpoint: %s\n\tmethod: %s", err, endpoint, method)

		slog.Error(wrapped.Error())

		return nil, wrapped
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := resp.Header.Get("Retry-After")

		if retryAfter == "" {
			retryAfter = "1000"
		}

		retryAfterDuration, err := time.ParseDuration(retryAfter + "ms")
		if err != nil {
			retryAfterDuration = time.Second
		}

		time.Sleep(retryAfterDuration)

		return guildedMakeRequest(token, method, endpoint, body)
	}

	return resp, nil
}

type guildedSocketManager struct {
	conn                  *websocket.Conn
	done                  chan struct{}
	readyHandler          func(*guildedWelcomeMessage)
	messageCreatedHandler func(*guildedChatMessageCreated)
	messageUpdatedHandler func(*guildedChatMessageUpdated)
	messageDeletedHandler func(*guildedChatMessageDeleted)
	Token                 string
	mu                    sync.RWMutex
	Alive                 bool
	reconnecting          bool
}

func guildedNewSocketManager(token string) *guildedSocketManager {
	return &guildedSocketManager{
		Token: token,
		done:  make(chan struct{}),
	}
}

func (s *guildedSocketManager) OnReady(handler func(*guildedWelcomeMessage)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.readyHandler = handler
}

func (s *guildedSocketManager) OnMessageCreated(handler func(*guildedChatMessageCreated)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageCreatedHandler = handler
}

func (s *guildedSocketManager) OnMessageUpdated(handler func(*guildedChatMessageUpdated)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageUpdatedHandler = handler
}

func (s *guildedSocketManager) OnMessageDeleted(handler func(*guildedChatMessageDeleted)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageDeletedHandler = handler
}

func (s *guildedSocketManager) Connect() error {
	s.mu.Lock()

	if s.Alive || s.reconnecting {
		s.mu.Unlock()

		return nil
	}

	s.reconnecting = true
	s.mu.Unlock()

	err := s.connectWebsocket()

	s.mu.Lock()
	s.reconnecting = false
	s.mu.Unlock()

	return err
}

func (s *guildedSocketManager) connectWebsocket() error {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+s.Token)
	header.Set("User-Agent", "lightning/"+lightning.VERSION)
	header["x-guilded-bot-api-use-official-markdown"] = []string{"true"}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}

	var err error

	var resp *http.Response

	s.conn, resp, err = dialer.Dial("wss://www.guilded.gg/websocket/v1", header)
	if err != nil {
		wrapped := fmt.Errorf("guilded: failed to dial WebSocket: %w", err)

		slog.Error(wrapped.Error(), "response", resp)

		return wrapped
	}

	err = resp.Body.Close()
	if err != nil {
		slog.Warn(fmt.Errorf("guilded: failed to close websocket request body: %w", err).Error())
	}

	s.mu.Lock()
	s.Alive = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.readMessages()

	return nil
}

func (s *guildedSocketManager) readMessages() {
	defer func() {
		s.mu.Lock()

		s.Alive = false
		if s.conn != nil {
			if err := s.conn.Close(); err != nil {
				slog.Warn("guilded: failed to close WebSocket connection when reading messages")
			}

			s.conn = nil
		}

		close(s.done)
		s.mu.Unlock()

		go s.handleReconnect()
	}()

	for {
		s.mu.RLock()

		if !s.Alive {
			s.mu.RUnlock()

			return
		}

		conn := s.conn
		s.mu.RUnlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				slog.Error(fmt.Errorf("guilded: error reading from socket: %w", err).Error())
			} else {
				slog.Debug("guilded: socket closed normally")
			}

			return
		}

		var data guildedSocketEventEnvelope
		if err := json.Unmarshal(message, &data); err != nil {
			slog.Error(fmt.Errorf("guilded: error parsing WebSocket message: %w", err).Error(), "msg", string(message))

			continue
		}

		s.handleEvent(data)
	}
}

func (s *guildedSocketManager) handleReconnect() {
	attempts := 0
	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

	for {
		attempts++

		slog.Info("guilded: attempting to reconnect to WebSocket", "attempt", attempts, "backoff", backoff)
		time.Sleep(backoff)

		err := s.Connect()
		if err == nil {
			slog.Info("guilded: WebSocket reconnection successful")

			return
		}

		backoff = min(time.Duration(float64(backoff)*1.5), maxBackoff)
		slog.Error(fmt.Errorf("guilded: failed to reconnect to WebSocket: %w", err).Error(),
			"attempt", attempts, "backoff", backoff)
	}
}

func (s *guildedSocketManager) handleEvent(data guildedSocketEventEnvelope) {
	if data.Op == 1 {
		s.handleReadyEvent(data)
	}

	if data.T == nil {
		return
	}

	switch *data.T {
	case "ChatMessageCreated":
		s.handleMessageCreatedEvent(data)
	case "ChatMessageUpdated":
		s.handleMessageUpdatedEvent(data)
	case "ChatMessageDeleted":
		s.handleMessageDeletedEvent(data)
	default:
	}
}

func (s *guildedSocketManager) handleReadyEvent(data guildedSocketEventEnvelope) {
	if s.readyHandler == nil {
		return
	}

	var welcome guildedWelcomeMessage
	if err := json.Unmarshal(data.D, &welcome); err != nil {
		slog.Warn(fmt.Errorf("guilded: failed to unmarshal ready data: %w", err).Error(), "data", data.D)

		return
	}

	go s.readyHandler(&welcome)
}

func (s *guildedSocketManager) handleMessageCreatedEvent(data guildedSocketEventEnvelope) {
	if s.messageCreatedHandler == nil {
		return
	}

	var msg guildedChatMessageCreated
	if err := json.Unmarshal(data.D, &msg); err != nil {
		slog.Warn(fmt.Errorf("guilded: failed to unmarshal ChatMessageCreated data: %w", err).Error(), "data", data.D)

		return
	}

	go s.messageCreatedHandler(&msg)
}

func (s *guildedSocketManager) handleMessageUpdatedEvent(data guildedSocketEventEnvelope) {
	if s.messageUpdatedHandler == nil {
		return
	}

	var msg guildedChatMessageUpdated
	if err := json.Unmarshal(data.D, &msg); err != nil {
		slog.Warn(fmt.Errorf("guilded: failed to unmarshal ChatMessageUpdated data: %w", err).Error(), "data", data.D)

		return
	}

	go s.messageUpdatedHandler(&msg)
}

func (s *guildedSocketManager) handleMessageDeletedEvent(data guildedSocketEventEnvelope) {
	if s.messageDeletedHandler == nil {
		return
	}

	var msg guildedChatMessageDeleted
	if err := json.Unmarshal(data.D, &msg); err != nil {
		slog.Warn(fmt.Errorf("guilded: failed to unmarshal ChatMessageDeleted data: %w", err).Error(), "data", data.D)

		return
	}

	go s.messageDeletedHandler(&msg)
}
