package guilded

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"sync/atomic"
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

type session struct {
	conn           *websocket.Conn
	ready          chan *guildedWelcomeMessage
	messageDeleted chan *guildedChatMessageDeleted
	messageCreated chan *guildedChatMessageCreated
	messageUpdated chan *guildedChatMessageUpdated
	token          string
	connected      atomic.Bool
}

func (s *session) connect() error {
	if s.connected.Load() {
		return nil
	}

	conn, resp, err := websocket.DefaultDialer.Dial(
		"wss://www.guilded.gg/websocket/v1",
		map[string][]string{
			"Authorization": {"Bearer " + s.token},
			"User-Agent":    {"lightning" + lightning.VERSION},
			"x-guilded-bot-api-use-official-markdown": {"true"},
		},
	)
	if err != nil {
		return fmt.Errorf("guilded: failed to dial: %w", err)
	}

	if err = resp.Body.Close(); err != nil {
		log.Printf("guilded: failed to close body: %v\n", err)
	}

	s.conn = conn
	s.connected.Swap(true)

	go readMessages(s)

	return nil
}

func readMessages(session *session) {
	for session.connected.Load() && session.conn != nil {
		_, message, err := session.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("guilded: error reading socket: %v\n", err)
			}

			break
		}

		handleEvent(session, message)
	}

	session.connected.Store(false)

	if session.conn != nil {
		if err := session.conn.Close(); err != nil {
			log.Printf("guilded: failed to close connection: %v\n", err)
		}

		session.conn = nil
	}

	go handleReconnect(session.connect)
}

func handleReconnect(connect func() error) {
	attempt := 0
	backoff := 100 * time.Millisecond

	for {
		attempt++

		time.Sleep(backoff)

		if connect() == nil {
			return
		}

		backoff = min(time.Duration(float64(backoff)*1.5), time.Second)

		log.Printf("guilded: attempting reconnect #%d after %s\n", attempt, backoff.String())
	}
}

func handleEvent(session *session, message []byte) {
	var data guildedSocketEventEnvelope
	if err := json.Unmarshal(message, &data); err != nil {
		log.Printf("guilded: failed unmarshaling event wrapper: %v\n\tdata: %s\n", err, string(message))

		return
	}

	if data.Op == 1 {
		handleGenericEvent(&data, session.ready)

		return
	}

	if data.T == nil {
		return
	}

	switch *data.T {
	case "ChatMessageCreated":
		handleGenericEvent(&data, session.messageCreated)
	case "ChatMessageUpdated":
		handleGenericEvent(&data, session.messageUpdated)
	case "ChatMessageDeleted":
		handleGenericEvent(&data, session.messageDeleted)
	default:
	}
}

func handleGenericEvent[T any](data *guildedSocketEventEnvelope, channel chan *T) {
	var decoded T
	if err := json.Unmarshal(data.D, &decoded); err != nil {
		log.Printf("guilded: failed unmarshaling event: %v\n\tdata: %s\n", err, string(data.D))

		return
	}

	channel <- &decoded
}
