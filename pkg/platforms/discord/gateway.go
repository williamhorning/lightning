package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

func addHandler[T any](bot *client, evt eventType, handler func(*T)) {
	bot.handlersMu.Lock()
	defer bot.handlersMu.Unlock()

	if bot.handlers == nil {
		bot.handlers = make(map[eventType][]func(any))
	}

	bot.handlers[evt] = append(bot.handlers[evt], func(v any) {
		if t, ok := v.(*T); ok {
			handler(t)
		}
	})
}

func (bot *client) connect() error {
	var url struct {
		URL string `json:"url"`
	}

	if err := bot.do("GET", "/gateway", nil, &url); err != nil {
		return err
	}

	socket, dur, err := bot.connectOnce(url.URL, "", "", 0)
	if err != nil {
		return err
	}

	go bot.run(socket, dur, url.URL)

	return nil
}

func (bot *client) run(socket *websocket.Conn, heartbeat time.Duration, gateway string) { //nolint:cyclop,revive,funlen
	var (
		state       = stateConnected
		nextBeat    <-chan time.Time
		requestBeat = make(chan struct{}, 1)
		messages    = make(chan struct {
			data []byte
			err  error
		}, 16)
		backoff      int
		seq          int64
		sessionID    string
		reconnectURL string
		activeSocket atomic.Pointer[websocket.Conn]
	)

	activeSocket.Store(socket)

	go startListener(messages, socket, &activeSocket)

	for {
		switch state {
		case stateConnecting:
			var err error

			socket, heartbeat, err = bot.connectOnce(gateway, reconnectURL, sessionID, seq)
			if err != nil {
				state = bot.closeAndTransition(socket, &seq, &sessionID, &reconnectURL, err)

				continue
			}

			activeSocket.Store(socket)
 
			go startListener(messages, socket, &activeSocket)

			backoff = 0
			nextBeat = time.After(1 * time.Second)
			state = stateConnected
		case stateConnected:
			select {
			case <-requestBeat:
				if err := bot.sendHeartbeat(socket, &seq); err != nil {
					state = bot.closeAndTransition(socket, &seq, &sessionID, &reconnectURL, err)

					continue
				}
			case <-nextBeat:
				if err := bot.sendHeartbeat(socket, &seq); err != nil {
					state = bot.closeAndTransition(socket, &seq, &sessionID, &reconnectURL, err)

					continue
				}

				nextBeat = time.After(heartbeat)
			case msg := <-messages:
				_ = socket.SetReadDeadline(time.Now().Add(heartbeat * 2))

				if msg.err != nil {
					state = bot.closeAndTransition(socket, &seq, &sessionID, &reconnectURL, msg.err)

					continue
				}

				state = bot.handleGatewayMessage(msg.data, requestBeat, &reconnectURL, &sessionID, &seq)
			}
		case stateReconnecting:
			backoff++
			time.Sleep(time.Duration(backoff*backoff) * time.Second)

			state = stateConnecting
		case stateTerminal:
			return
		default:
		}
	}
}

func startListener(messages chan struct {
	data []byte
	err  error
}, socket *websocket.Conn, active *atomic.Pointer[websocket.Conn],
) {
	for {
		if active.Load() != socket {
			return
		}

		_, data, err := socket.ReadMessage()

		messages <- struct {
			data []byte
			err  error
		}{data, err}

		if err != nil {
			return
		}
	}
}

func (bot *client) connectOnce( //nolint:revive,cyclop
	gateway, reconnectURL, sessionID string, seq int64,
) (*websocket.Conn, time.Duration, error) {
	url := gateway
	if reconnectURL != "" {
		url = reconnectURL
	}

	socket, resp, err := websocket.DefaultDialer.Dial(url+"?v="+bot.version+"&encoding=json", http.Header{
		"User-Agent": []string{"DiscordBot (https://williamhorn.ing/lightning, 0.8.6)"},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to dial websocket: %w", err)
	}

	_ = resp.Body.Close()

	_, data, err := socket.ReadMessage()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read hello message: %w", err)
	}

	var msg gatewayMessage
	if err := json.Unmarshal(data, &msg); err != nil || msg.Op != 10 {
		return nil, 0, fmt.Errorf("failed to unmarshal gateway message (op %d): %w", msg.Op, err)
	}

	var hello gatewayHello
	if err := json.Unmarshal(msg.D, &hello); err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal hello message: %w", err)
	}

	var payload []byte

	var payloadOp int

	if sessionID != "" && seq != 0 {
		payload, err = json.Marshal(gatewayResume{
			Token: bot.token, SessionID: sessionID, Sequence: seq,
		})
		payloadOp = 6
	} else {
		payload, err = json.Marshal(gatewayIdentify{
			Token: bot.token, Intents: bot.intents,
			Properties: gatewayIdentifyProperties{
				OS: "lightning", Browser: "https://williamhorn.ing/lightning",
				Device: "https://williamhorn.ing/lightning",
			},
		})
		payloadOp = 2
	}

	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal identify/resume payload: %w", err)
	}

	if err := socket.WriteJSON(gatewayMessage{Op: payloadOp, D: payload}); err != nil {
		return nil, 0, fmt.Errorf("failed to write identify/resume payload: %w", err)
	}

	return socket, time.Duration(hello.Interval) * time.Millisecond, nil
}

func (bot *client) handleGatewayMessage(
	data []byte, beat chan struct{}, reconnectURL, sessionID *string, seq *int64,
) connState {
	var msg gatewayMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return stateReconnecting
	}

	if msg.S != nil {
		*seq = *msg.S
	}

	switch msg.Op {
	case 0:
		bot.dispatch(msg.T, msg.D, reconnectURL, sessionID)
	case 1:
		beat <- struct{}{}
	case 7:
		return stateReconnecting
	case 9:
		var d bool
		if err := json.Unmarshal(msg.D, &d); err != nil || !d {
			*seq = 0
			*sessionID = ""
			*reconnectURL = ""
		}

		return stateReconnecting
	default:
	}

	return stateConnected
}

func (*client) sendHeartbeat(socket *websocket.Conn, seq *int64) error {
	d, err := json.Marshal(*seq)
	if err != nil {
		d = []byte("null")
	}

	if err := socket.WriteJSON(gatewayMessage{Op: 1, D: d}); err != nil {
		return fmt.Errorf("failed to write heartbeat: %w", err)
	}

	return nil
}

func (bot *client) dispatch( //nolint:revive,cyclop,funlen
	evt eventType, data json.RawMessage, reconnectURL, sessionID *string,
) {
	var payload any

	switch evt {
	case eventReady:
		var ready readyEvent
		if json.Unmarshal(data, &ready) != nil {
			return
		}

		bot.users.Set(string(ready.User.ID), &ready.User)
		bot.application = &ready.Application
		*sessionID = ready.SessionID
		*reconnectURL = ready.ResumeURL
		payload = &ready
	case eventResumed:
		log.Printf("%s: connection resumed", bot.product)

		return
	case eventMessageCreate, eventMessageEdit:
		var msg message
		if json.Unmarshal(data, &msg) != nil {
			return
		}

		bot.messages.Set(string(msg.ID), &msg)
		payload = &msg
	case eventMessageDelete:
		var del messageDelete
		if json.Unmarshal(data, &del) != nil {
			return
		}

		payload = &del
	case eventChannelCreate, eventChannelUpdate:
		var chn channel
		if json.Unmarshal(data, &chn) != nil {
			return
		}

		bot.channels.Set(string(chn.ID), &chn)
		payload = &chn
	case eventGuildCreate, eventGuildUpdate:
		var gui guild
		if json.Unmarshal(data, &gui) != nil || gui.Unavailable {
			return
		}

		for idx := range gui.Roles {
			bot.roles.Set(string(gui.Roles[idx].ID), &gui.Roles[idx])
		}

		bot.guilds.Set(gui.ID, &gui)
		payload = &gui
	case eventRoleCreate, eventRoleUpdate:
		var rol roleEvent
		if json.Unmarshal(data, &rol) != nil {
			return
		}

		bot.roles.Set(string(rol.Role.ID), &rol.Role)
		payload = &rol
	case eventEmojisUpdate:
		var emj discordEmojiEvent
		if json.Unmarshal(data, &emj) != nil {
			return
		}

		bot.emojis.Set(emj.Guild, &emj.Emojis)
		payload = &emj
	case eventInteractionCreate:
		var v interactionCreateEvent
		if json.Unmarshal(data, &v) != nil {
			return
		}

		payload = &v
	default:
		return
	}

	bot.handlersMu.RLock()
	handlers := bot.handlers[evt]
	bot.handlersMu.RUnlock()

	for _, h := range handlers {
		h(payload)
	}
}

func (bot *client) closeAndTransition(
	socket *websocket.Conn, seq *int64, sessionID, reconnectURL *string, err error,
) connState {
	_ = socket.Close()

	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) {
		log.Printf("%s: socket error, will try reconnecting: %v", bot.product, err)

		return stateReconnecting
	}

	log.Printf("%s: socket close, will try reconnecting: %v", bot.product, err)

	switch closeErr.Code {
	case 4009:
		*seq = 0
		*sessionID = ""
		*reconnectURL = ""

		return stateReconnecting
	case 4004, 4010, 4011, 4012, 4013, 4014:
		return stateTerminal
	default:
		return stateReconnecting
	}
}
