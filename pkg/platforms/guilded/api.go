package guilded

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
	"github.com/gorilla/websocket"
)

func guildedMakeRequest(token, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, "https://www.guilded.gg/api/v1"+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("guilded: creating request: %w\n\tendpoint: %s\n\tmethod: %s", err, endpoint, method)
	}

	req.Header = http.Header{
		"Authorization": {"Bearer " + token},
		"Content-Type":  {"application/json"},
		"User-Agent":    {"lightning/" + lightning.VERSION},
		"x-guilded-bot-api-use-official-markdown": {"true"},
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("guilded: making request: %w\n\tendpoint: %s\n\tmethod: %s", err, endpoint, method)
	}

	if resp.StatusCode != http.StatusTooManyRequests {
		return resp, nil
	}

	retry := resp.Header.Get("Retry-After")
	if retry == "" {
		retry = "1000"
	}

	dur, _ := time.ParseDuration(retry + "ms")
	if dur == 0 {
		dur = time.Second
	}

	time.Sleep(dur)

	return guildedMakeRequest(token, method, endpoint, body)
}

type session struct {
	conn           *websocket.Conn
	messageDeleted chan *guildedChatMessageDeleted
	messageCreated chan *guildedChatMessageWrapper
	messageUpdated chan *guildedChatMessageWrapper
	token          string
	connected      atomic.Bool
}

func (s *session) connect() error {
	if s.connected.Load() {
		return nil
	}

	conn, resp, err := websocket.DefaultDialer.Dial(
		"wss://www.guilded.gg/websocket/v1",
		http.Header{
			"Authorization": {"Bearer " + s.token},
			"User-Agent":    {"lightning" + lightning.VERSION},
			"x-guilded-bot-api-use-official-markdown": {"true"},
		},
	)
	if err != nil {
		return fmt.Errorf("guilded: failed to dial: %w", err)
	}

	defer resp.Body.Close()

	s.conn = conn
	s.connected.Store(true)

	go readMessages(s)

	return nil
}

func readMessages(session *session) {
	for session.connected.Load() && session.conn != nil {
		_, reader, err := session.conn.NextReader()
		if err != nil {
			break
		}

		var data guildedSocketEventEnvelope
		if json.NewDecoder(reader).Decode(&data) != nil {
			return
		}

		switch data.T {
		case "ChatMessageCreated":
			handleGenericEvent(data.D, session.messageCreated)
		case "ChatMessageUpdated":
			handleGenericEvent(data.D, session.messageUpdated)
		case "ChatMessageDeleted":
			handleGenericEvent(data.D, session.messageDeleted)
		default:
		}
	}

	go handleReconnect(session)
}

func handleReconnect(session *session) {
	session.connected.Store(false)

	if session.conn != nil {
		defer session.conn.Close()

		session.conn = nil
	}

	for attempt, backoff := 1, 100*time.Millisecond; ; attempt++ {
		time.Sleep(backoff)

		if session.connect() == nil {
			return
		}

		backoff = min(time.Duration(float64(backoff)*1.5), time.Second)

		log.Printf("guilded: reconnect #%d after %s\n", attempt, backoff)
	}
}

func handleGenericEvent[T any](bytes json.RawMessage, events chan *T) {
	var d T
	if json.Unmarshal(bytes, &d) != nil {
		return
	}

	events <- &d
}
