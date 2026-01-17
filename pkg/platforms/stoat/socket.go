package stoat

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

func (session *session) connect() error {
	if session.conn != nil {
		return nil
	}

	conn, resp, err := websocket.DefaultDialer.Dial(
		"wss://events.stoat.chat/?version=1&format=json&token="+session.token,
		map[string][]string{"User-Agent": {"rvapi/0.8.0"}},
	)
	if err != nil {
		return fmt.Errorf("failed to dial stoat socket: %w", err)
	}

	defer resp.Body.Close()

	session.conn = conn

	go ping(session)
	go readMessages(session)

	return nil
}

func ping(session *session) {
	for session.conn != nil {
		time.Sleep(10 * time.Second)

		session.mu.Lock()

		if session.conn != nil {
			err := session.conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"Ping"}`))
			if err != nil {
				log.Printf("stoat: failed to ping: %v\n", err)
			}
		}

		session.mu.Unlock()
	}
}

func readMessages(session *session) {
	for session.conn != nil {
		_, message, err := session.conn.ReadMessage()
		if err != nil {
			break
		}

		handleEvent(session, message)
	}

	if session.conn != nil {
		if err := session.conn.Close(); err != nil {
			log.Printf("stoat: failed to close connection: %v\n", err)
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

		log.Printf("stoat: trying reconnect #%d after %s\n", attempt, backoff.String())
	}
}

func handleEvent(session *session, message []byte) {
	var data struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(message, &data); err != nil {
		log.Printf("stoat: failed unmarshalling base socket event (%q) %v\n", string(message), err)

		return
	}

	switch data.Type {
	case "Bulk":
		handleBulkEvent(session, message)
	case "Ready":
		handleReadyEvent(session, message)
	case "Message":
		handleGenericEvent(message, session.messageCreated)
	case "MessageUpdate":
		handleGenericEvent(message, session.messageUpdated)
	case "MessageDelete":
		handleGenericEvent(message, session.messageDeleted)
	default:
	}
}

func handleBulkEvent(session *session, message []byte) {
	var bulk struct {
		V []json.RawMessage `json:"v"`
	}
	if err := json.Unmarshal(message, &bulk); err != nil {
		log.Printf("stoat: failed unmarshalling bulk socket event (%q) %v\n", string(message), err)

		return
	}

	for _, event := range bulk.V {
		handleEvent(session, event)
	}
}

func handleReadyEvent(session *session, message []byte) {
	var ready stReadyEvent
	if err := json.Unmarshal(message, &ready); err != nil {
		log.Printf("stoat: failed unmarshalling ready socket event (%q) %v\n", string(message), err)

		return
	}

	session.ready <- &ready

	for _, channel := range ready.Channels {
		session.channelCache.Set(channel.ID, channel)
	}

	for _, server := range ready.Servers {
		session.serverCache.Set(server.ID, server)
		session.serverEmojiCache.Set(server.ID, []stEmoji{})
	}

	for _, user := range ready.Users {
		session.userCache.Set(user.ID, user)
	}

	for _, member := range ready.Members {
		session.memberCache.Set(member.ID.Server+"-"+member.ID.User, member)
	}

	for _, emoji := range ready.Emojis {
		session.emojiCache.Set(emoji.ID, emoji)

		emojis, _ := session.serverEmojiCache.Get(emoji.Parent.ID)
		session.serverEmojiCache.Set(emoji.Parent.ID, append(emojis, emoji))
	}
}

func handleGenericEvent[T any](message json.RawMessage, channel chan *T) {
	var decoded T
	if err := json.Unmarshal(message, &decoded); err != nil {
		log.Printf("stoat: failed unmarshalling generic socket event (%q) %v\n", string(message), err)

		return
	}

	channel <- &decoded
}
