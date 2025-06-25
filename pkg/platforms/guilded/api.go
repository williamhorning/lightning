package guilded

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func guildedMakeRequest(token, method, endpoint string, body *io.Reader) (*http.Response, error) {
	url := "https://www.guilded.gg/api/v1" + endpoint

	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, *body)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "guildapi/0.0.5")
	req.Header.Set("x-guilded-bot-api-use-official-markdown", "true")

	return http.DefaultClient.Do(req)
}

type guildedSocketManager struct {
	conn                  *websocket.Conn
	Alive                 bool
	Token                 string
	mu                    sync.RWMutex
	done                  chan struct{}
	readyHandler          func(*guildedWelcomeMessage)
	messageCreatedHandler func(*guildedChatMessageCreated)
	messageUpdatedHandler func(*guildedChatMessageUpdated)
	messageDeletedHandler func(*guildedChatMessageDeleted)
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
	lightning.Log.Trace().Str("plugin", "guilded").Msg("Registered ready handler")
}

func (s *guildedSocketManager) OnMessageCreated(handler func(*guildedChatMessageCreated)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageCreatedHandler = handler
	lightning.Log.Trace().Str("plugin", "guilded").Msg("Registered message created handler")
}

func (s *guildedSocketManager) OnMessageUpdated(handler func(*guildedChatMessageUpdated)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageUpdatedHandler = handler
	lightning.Log.Trace().Str("plugin", "guilded").Msg("Registered message updated handler")
}

func (s *guildedSocketManager) OnMessageDeleted(handler func(*guildedChatMessageDeleted)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageDeletedHandler = handler
	lightning.Log.Trace().Str("plugin", "guilded").Msg("Registered message deleted handler")
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
	header.Set("User-Agent", "guildapi/0.0.5")
	header.Set("x-guilded-bot-api-use-official-markdown", "true")

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}

	var err error
	s.conn, _, err = dialer.Dial("wss://www.guilded.gg/websocket/v1", header)
	if err != nil {
		return err
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
			s.conn.Close()
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
				lightning.Log.Error().Err(err).Msg("Error reading from WebSocket")
			} else {
				lightning.Log.Info().Msg("WebSocket closed")
			}
			return
		}

		var data guildedSocketEventEnvelope
		if err := json.Unmarshal(message, &data); err != nil {
			lightning.Log.Error().Err(err).Msg("Error parsing WebSocket message")
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

		lightning.Log.Info().
			Int("attempt", attempts).
			Dur("backoff", backoff).
			Msg("Attempting to reconnect to Guilded WebSocket")

		time.Sleep(backoff)

		err := s.Connect()
		if err == nil {
			lightning.Log.Info().Msg("Guilded WebSocket reconnection successful")
			return
		}

		lightning.Log.Error().
			Err(err).
			Int("attempt", attempts).
			Msg("Failed to reconnect to Guilded WebSocket")

		backoff = min(time.Duration(float64(backoff)*1.5), maxBackoff)
	}
}

func (s *guildedSocketManager) handleEvent(data guildedSocketEventEnvelope) {
	if data.T == nil {
		lightning.Log.Trace().Msg("Received event with nil type")
		return
	}

	eventType := *data.T
	lightning.Log.Trace().
		Str("plugin", "guilded").
		Str("event_type", eventType).
		Msg("Processing socket event")

	switch eventType {
	case "ready":
		if s.readyHandler != nil {
			var welcome guildedWelcomeMessage
			welcomeJSON, _ := json.Marshal(data.D)
			if err := json.Unmarshal(welcomeJSON, &welcome); err != nil {
				lightning.Log.Error().Err(err).Msg("Failed to parse ready event")
				return
			}
			go s.readyHandler(&welcome)
		}
	case "ChatMessageCreated":
		if s.messageCreatedHandler != nil {
			var msg guildedChatMessageCreated
			msgJSON, _ := json.Marshal(data.D)
			if err := json.Unmarshal(msgJSON, &msg); err != nil {
				lightning.Log.Error().Err(err).Msg("Failed to parse message created event")
				return
			}
			lightning.Log.Trace().Str("plugin", "guilded").Msg("Calling message created handler")
			go s.messageCreatedHandler(&msg)
		}
	case "ChatMessageUpdated":
		if s.messageUpdatedHandler != nil {
			var msg guildedChatMessageUpdated
			msgJSON, _ := json.Marshal(data.D)
			if err := json.Unmarshal(msgJSON, &msg); err != nil {
				lightning.Log.Error().Err(err).Msg("Failed to parse message updated event")
				return
			}
			go s.messageUpdatedHandler(&msg)
		}
	case "ChatMessageDeleted":
		if s.messageDeletedHandler != nil {
			var msg guildedChatMessageDeleted
			msgJSON, _ := json.Marshal(data.D)
			if err := json.Unmarshal(msgJSON, &msg); err != nil {
				lightning.Log.Error().Err(err).Msg("Failed to parse message deleted event")
				return
			}
			go s.messageDeletedHandler(&msg)
		}
	default:
		lightning.Log.Trace().Str("event", eventType).Msg("Unhandled event type")
	}
}
