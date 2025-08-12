package revolt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type revoltSocketManager struct {
	conn                  *websocket.Conn
	done                  chan struct{}
	readyHandler          func(*revoltEventReady)
	messageCreatedHandler func(*revoltEventMessage)
	messageUpdatedHandler func(*revoltEventMessageUpdate)
	messageDeletedHandler func(*revoltEventMessageDelete)
	Token                 string
	mu                    sync.RWMutex
	reconnecting          bool
	Alive                 bool
}

func revoltNewSocketManager(token string) *revoltSocketManager {
	return &revoltSocketManager{
		Token: token,
		done:  make(chan struct{}),
	}
}

func (s *revoltSocketManager) OnReady(handler func(*revoltEventReady)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.readyHandler = handler
}

func (s *revoltSocketManager) OnMessageCreated(handler func(*revoltEventMessage)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageCreatedHandler = handler
}

func (s *revoltSocketManager) OnMessageUpdated(handler func(*revoltEventMessageUpdate)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageUpdatedHandler = handler
}

func (s *revoltSocketManager) OnMessageDeleted(handler func(*revoltEventMessageDelete)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.messageDeletedHandler = handler
}

func (s *revoltSocketManager) Connect() error {
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

func (s *revoltSocketManager) connectWebsocket() error {
	header := http.Header{}
	header.Set("User-Agent", "lightning/"+lightning.VERSION)

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}

	var err error

	var resp *http.Response

	s.conn, resp, err = dialer.Dial("wss://app.revolt.chat/events?version=1&format=json&token="+s.Token, header)
	if err != nil {
		return fmt.Errorf("revolt: failed to dial WebSocket: %w", err)
	}

	err = resp.Body.Close()
	if err != nil {
		slog.Warn("revolt: failed to close websocket request body", "err", err)
	}

	s.mu.Lock()
	s.Alive = true
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.readMessages()

	return nil
}

func (s *revoltSocketManager) readMessages() {
	defer func() {
		s.mu.Lock()

		s.Alive = false
		if s.conn != nil {
			if err := s.conn.Close(); err != nil {
				slog.Warn("revolt: failed to close request body when reading messages")
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
				slog.Error("revolt: error reading from socket", "error", err)
			} else {
				slog.Debug("revolt: socket closed normally")
			}

			return
		}

		s.handleEvent(message)
	}
}

func (s *revoltSocketManager) handleReconnect() {
	attempts := 0
	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

	for {
		attempts++

		slog.Info("revolt: attempting to reconnect to WebSocket", "attempt", attempts, "backoff", backoff)
		time.Sleep(backoff)

		err := s.Connect()
		if err == nil {
			slog.Info("revolt: WebSocket reconnection successful")

			return
		}

		backoff = min(time.Duration(float64(backoff)*1.5), maxBackoff)
		slog.Error("revolt: failed to reconnect to WebSocket", "attempt", attempts, "backoff", backoff, "error", err)
	}
}

func (s *revoltSocketManager) handleEvent(message []byte) {
	var data revoltEvent
	if err := json.Unmarshal(message, &data); err != nil {
		slog.Error("revolt: error parsing WebSocket message", "error", err, "message", string(message))

		return
	}

	switch data.Type {
	case "Bulk":
		s.handleBulkEvent(message)
	case "Error":
		s.handleErrorEvent(message)
	case "Ready":
		s.handleReadyEvent(message)
	case "Message":
		s.handleMessageCreatedEvent(message)
	case "MessageUpdate":
		s.handleMessageUpdatedEvent(message)
	case "MessageDelete":
		s.handleMessageDeletedEvent(message)
	}
}

func (s *revoltSocketManager) handleBulkEvent(message []byte) {
	var bulk revoltEventBulk
	if err := json.Unmarshal(message, &bulk); err != nil {
		slog.Warn("revolt: failed to unmarshal bulk data", "error", err, "data", message)

		return
	}

	for _, event := range bulk.V {
		s.handleEvent(event)
	}
}

func (*revoltSocketManager) handleErrorEvent(message []byte) {
	var errorEvent revoltEventError
	if err := json.Unmarshal(message, &errorEvent); err != nil {
		slog.Warn("revolt: failed to unmarshal error data", "error", err, "data", message)

		return
	}

	slog.Warn("revolt: socket error", "err", errorEvent)
}

func (s *revoltSocketManager) handleReadyEvent(data []byte) {
	if s.readyHandler == nil {
		return
	}

	var welcome revoltEventReady
	if err := json.Unmarshal(data, &welcome); err != nil {
		slog.Warn("revolt: failed to unmarshal ready data", "error", err, "data", data)

		return
	}

	go s.readyHandler(&welcome)
}

func (s *revoltSocketManager) handleMessageCreatedEvent(data []byte) {
	if s.messageCreatedHandler == nil {
		return
	}

	var msg revoltEventMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Warn("revolt: failed to unmarshal message created data", "error", err, "data", data)

		return
	}

	go s.messageCreatedHandler(&msg)
}

func (s *revoltSocketManager) handleMessageUpdatedEvent(data []byte) {
	if s.messageUpdatedHandler == nil {
		return
	}

	var msg revoltEventMessageUpdate
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Warn("revolt: failed to unmarshal message update data", "error", err, "data", data)

		return
	}

	go s.messageUpdatedHandler(&msg)
}

func (s *revoltSocketManager) handleMessageDeletedEvent(data []byte) {
	if s.messageDeletedHandler == nil {
		return
	}

	var msg revoltEventMessageDelete
	if err := json.Unmarshal(data, &msg); err != nil {
		slog.Warn("revolt: failed to unmarshal message deleted data", "error", err, "data", data)

		return
	}

	go s.messageDeletedHandler(&msg)
}
