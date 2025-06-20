package guilded

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

type guildedCloseInfo struct {
	Code   int
	Reason string
}

type guildedSocketManager struct {
	conn           *websocket.Conn
	Alive          bool
	LastMessageID  string
	ReconnectCount int
	Token          string
	listeners      map[string][]func(...any)
	mu             sync.RWMutex
	done           chan struct{}
}

func guildedNewSocketManager(token string) *guildedSocketManager {
	return &guildedSocketManager{
		Token:     token,
		listeners: make(map[string][]func(...any)),
		done:      make(chan struct{}),
	}
}

func (s *guildedSocketManager) On(event string, handler any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Convert handler to func(...any)
	var fn func(...any)
	switch h := handler.(type) {
	case func(...any):
		fn = h
	case func():
		fn = func(...any) { h() }
	case func(any):
		fn = func(args ...any) {
			if len(args) > 0 {
				h(args[0])
			}
		}
	default:
		fn = func(args ...any) {
			if len(args) > 0 {
				if f, ok := handler.(func(any)); ok {
					f(args[0])
				}
			}
		}
	}
	s.listeners[event] = append(s.listeners[event], fn)
}

func (s *guildedSocketManager) Emit(event string, args ...any) {
	s.mu.RLock()
	handlers := s.listeners[event]
	s.mu.RUnlock()

	for _, handler := range handlers {
		handler(args...)
	}
}

func (s *guildedSocketManager) Connect() error {
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
	s.Alive = true
	s.done = make(chan struct{})

	go s.readMessages()
	return nil
}

func (s *guildedSocketManager) readMessages() {
	defer func() {
		s.conn.Close()
		s.Alive = false
		close(s.done)
	}()

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			s.Emit("debug", fmt.Sprintf("Error reading from socket: %v", err))
			closeInfo := guildedCloseInfo{Code: websocket.CloseNormalClosure}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				if ce, ok := err.(*websocket.CloseError); ok {
					closeInfo.Code = ce.Code
					closeInfo.Reason = ce.Text
				}
			}
			s.Emit("close", closeInfo)
			s.handleReconnect()
			return
		}

		s.Emit("debug", fmt.Sprintf("received packet: %s", message))
		s.handleMessage(message)
	}
}

func (s *guildedSocketManager) handleMessage(message []byte) {
	var data guildedSocketEventEnvelope
	if err := json.Unmarshal(message, &data); err != nil {
		s.Emit("debug", "received invalid packet")
		return
	}

	if data.S != nil {
		s.LastMessageID = *data.S
	}

	switch data.Op {
	case guildedSocketOPSuccess:
		s.handleEvent(data)
	case guildedSocketOPWelcome:
		s.handleWelcome(data)
	case guildedSocketOPResume:
		s.Emit("debug", "received resume packet")
		s.LastMessageID = ""
	case guildedSocketOPError:
		s.handleError(data)
	case guildedSocketOPPing:
		s.handlePing()
	default:
		s.Emit("debug", "received unknown opcode")
	}
}

func (s *guildedSocketManager) handleEvent(data guildedSocketEventEnvelope) {
	if data.T == nil {
		return
	}
	eventType := *data.T
	eventJSON, _ := json.Marshal(data.D)

	var evt any
	var err error
	switch eventType {
	case "ChatMessageCreated":
		evt = &guildedChatMessageCreated{}
	case "ChatMessageUpdated":
		evt = &guildedChatMessageUpdated{}
	case "ChatMessageDeleted":
		evt = &guildedChatMessageDeleted{}
	default:
		s.Emit(eventType, data.D)
		return
	}

	if err = json.Unmarshal(eventJSON, evt); err != nil {
		s.Emit("debug", fmt.Sprintf("Failed to parse %s: %v", eventType, err))
		return
	}
	s.Emit(eventType, evt)
}

func (s *guildedSocketManager) handleWelcome(data guildedSocketEventEnvelope) {
	var welcome guildedWelcomeMessage
	welcomeJSON, _ := json.Marshal(data.D)
	if err := json.Unmarshal(welcomeJSON, &welcome); err != nil {
		s.Emit("debug", "received invalid welcome packet")
		return
	}
	s.Emit("ready", &welcome)
}

func (s *guildedSocketManager) handleError(data guildedSocketEventEnvelope) {
	s.Emit("debug", "received error packet")
	var errorData struct {
		Message string `json:"message"`
	}
	errJSON, _ := json.Marshal(data.D)
	if err := json.Unmarshal(errJSON, &errorData); err == nil {
		s.Emit("error", errors.New(errorData.Message), data)
	}
	s.LastMessageID = ""
	s.conn.Close()
}

func (s *guildedSocketManager) handlePing() {
	s.Emit("debug", "received ping packet, sending pong")
	pong := map[string]any{"op": guildedSocketOPPong}
	if pongData, err := json.Marshal(pong); err == nil {
		s.conn.WriteMessage(websocket.TextMessage, pongData)
	}
}

func (s *guildedSocketManager) handleReconnect() {
	s.Emit("debug", "disconnecting due to close")
	s.Emit("debug", "reconnecting to Guilded")
	s.Emit("reconnect")
	s.ReconnectCount++

	backoff := time.Duration(math.Min(float64(s.ReconnectCount*s.ReconnectCount), 30)) * time.Second
	time.Sleep(backoff)
	s.Connect()
}

func (s *guildedSocketManager) Close() error {
	if s.conn == nil {
		return nil
	}

	err := s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		return err
	}
	<-s.done
	return nil
}
