// Package rvapi implements the Stoat API
package rvapi

import (
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/williamhorning/lightning/internal/cache"
)

// VERSION is the version of the rvapi library. Currently, it's the same as the
// version of lightning in the repo, though it may change.
const VERSION = "0.8.0-rc.5"

// Session represents the Stoat API session a bot may have.
type Session struct {
	MessageDeleted   chan *MessageDeleteEvent
	conn             *websocket.Conn
	Ready            chan *ReadyEvent
	MessageCreated   chan *MessageEvent
	MessageUpdated   chan *MessageUpdateEvent
	Token            string
	ChannelCache     cache.Expiring[string, Channel]
	MemberCache      cache.Expiring[string, Member]
	UserCache        cache.Expiring[string, User]
	ServerEmojiCache cache.Expiring[string, []Emoji]
	EmojiCache       cache.Expiring[string, Emoji]
	ServerCache      cache.Expiring[string, Server]
	connected        atomic.Bool
}
