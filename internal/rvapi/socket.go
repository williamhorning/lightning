package rvapi

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Connect to the Stoat socket.
func (s *Session) Connect() error {
	if s.connected.Load() {
		return nil
	}

	conn, resp, err := websocket.DefaultDialer.Dial(
		"wss://app.stoat.chat/events?version=1&format=json&token="+s.Token,
		map[string][]string{"User-Agent": {"rvapi/0.8.0-rc.2"}},
	)
	if err != nil {
		return fmt.Errorf("rvapi: failed to dial: %w", err)
	}

	if err = resp.Body.Close(); err != nil {
		log.Printf("rvapi: failed to close body: %v\n", err)
	}

	s.conn = conn
	s.connected.Swap(true)

	go ping(s)
	go readMessages(s)

	return nil
}

func ping(session *Session) {
	for session.connected.Load() && session.conn != nil {
		time.Sleep(10 * time.Second)

		err := session.conn.WriteMessage(websocket.TextMessage, []byte("{\"type\":\"Ping\"}"))
		if err != nil {
			log.Printf("rvapi: error pinging: %v\n", err)
		}
	}
}

func readMessages(session *Session) {
	for session.connected.Load() && session.conn != nil {
		_, message, err := session.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				log.Printf("rvapi: error reading socket: %v\n", err)
			}

			break
		}

		handleEvent(session, message)
	}

	session.connected.Store(false)

	if session.conn != nil {
		if err := session.conn.Close(); err != nil {
			log.Printf("rvapi: failed to close connection: %v\n", err)
		}

		session.conn = nil
	}

	go handleReconnect(session.Connect)
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

		log.Printf("rvapi: attempting reconnect #%d after %s\n", attempt, backoff.String())
	}
}

func handleEvent(session *Session, message []byte) {
	var data BaseEvent
	if err := json.Unmarshal(message, &data); err != nil {
		log.Printf("rvapi: failed unmarshaling event wrapper: %v\n\tdata: %s\n", err, string(message))

		return
	}

	switch data.Type {
	case "Bulk":
		handleBulkEvent(session, message)
	case "Ready":
		handleReadyEvent(session, message)
	case "Message":
		handleGenericEvent(message, session.MessageCreated)
	case "MessageUpdate":
		handleGenericEvent(message, session.MessageUpdated)
	case "MessageDelete":
		handleGenericEvent(message, session.MessageDeleted)
	default:
	}
}

func handleBulkEvent(session *Session, message []byte) {
	var bulk BulkEvent
	if err := json.Unmarshal(message, &bulk); err != nil {
		log.Printf("rvapi: failed unmarshaling bulk event: %v\n\tdata: %s\n", err, string(message))

		return
	}

	for _, event := range bulk.V {
		handleEvent(session, event)
	}
}

func handleReadyEvent(session *Session, message []byte) {
	var ready ReadyEvent
	if err := json.Unmarshal(message, &ready); err != nil {
		log.Printf("rvapi: failed unmarshaling ready event: %v\n\tdata: %s\n", err, string(message))

		return
	}

	session.Ready <- &ready

	for _, channel := range ready.Channels {
		session.ChannelCache.Set(channel.ID, channel)
	}

	for _, server := range ready.Servers {
		session.ServerCache.Set(server.ID, server)
		session.ServerEmojiCache.Set(server.ID, []Emoji{})
	}

	for _, user := range ready.Users {
		session.UserCache.Set(user.ID, user)
	}

	for _, member := range ready.Members {
		session.MemberCache.Set(member.ID.Server+"-"+member.ID.User, member)
	}

	for _, emoji := range ready.Emojis {
		session.EmojiCache.Set(emoji.ID, emoji)

		emojis, _ := session.ServerEmojiCache.Get(emoji.Parent.ID)
		session.ServerEmojiCache.Set(emoji.Parent.ID, append(emojis, emoji))
	}
}

func handleGenericEvent[T any](message []byte, channel chan *T) {
	var decoded T
	if err := json.Unmarshal(message, &decoded); err != nil {
		log.Printf("rvapi: failed unmarshaling generic event: %v\n\tdata: %s\n", err, string(message))

		return
	}

	channel <- &decoded
}
