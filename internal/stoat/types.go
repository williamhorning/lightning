package stoat

import (
	"encoding/json"
	"time"
)

// BaseEvent provides the common "type" field.
type BaseEvent struct {
	Type string `json:"type"`
}

// BulkEvent wraps multiple events into a single payload.
type BulkEvent struct {
	V []json.RawMessage `json:"v"` // slice of other events
}

// CDNFile is the Autumn representation of a file.
type CDNFile struct {
	ID string `json:"id"`
}

// Channel represents all channel types.
type Channel struct {
	Name            string                   `json:"name,omitempty"`
	Owner           *string                  `json:"owner,omitempty"`
	Permissions     *Permission              `json:"permissions,omitempty"`
	Server          *string                  `json:"server,omitempty"`
	DefaultPerms    *OverrideField           `json:"default_permissions,omitempty"`
	RolePermissions map[string]OverrideField `json:"role_permissions,omitempty"`
	ChannelType     ChannelType              `json:"channel_type"`
	ID              string                   `json:"_id"`
	Recipients      []string                 `json:"recipients,omitempty"`
}

// ChannelType is the type of a channel.
type ChannelType string

// Possible ChannelType values.
const (
	ChannelTypeSavedMessages ChannelType = "SavedMessages"
	ChannelTypeText          ChannelType = "TextChannel"
	ChannelTypeVoice         ChannelType = "VoiceChannel"
	ChannelTypeDM            ChannelType = "DirectMessage"
	ChannelTypeGroup         ChannelType = "Group"
)

// DataEditMessage describes how to edit a message.
type DataEditMessage struct {
	Content string          `json:"content"`
	Embeds  []SendableEmbed `json:"embeds,omitempty"`
}

// DataMessageSend is a message to send.
type DataMessageSend struct {
	Masquerade  *Masquerade     `json:"masquerade,omitempty"`
	Content     string          `json:"content"`
	Attachments []string        `json:"attachments,omitempty"`
	Replies     []ReplyIntent   `json:"replies,omitempty"`
	Embeds      []SendableEmbed `json:"embeds,omitempty"`
}

// Embed represents embedded rich content.
type Embed struct {
	URL         string  `json:"url,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Image       *Media  `json:"image,omitempty"`
	Video       *Media  `json:"video,omitempty"`
	IconURL     *string `json:"icon_url,omitempty"`
	Colour      string  `json:"colour,omitempty"`
}

// Emoji is a user-created emoji on Stoat.
type Emoji struct {
	ID     string      `json:"_id"`
	Parent EmojiParent `json:"parent"`
	Name   string      `json:"name"`
}

// EmojiParent represents emoji scoping.
type EmojiParent struct {
	ID string `json:"id,omitempty"`
}

// File is the representation of a file.
type File struct {
	ID       string `json:"_id"`
	Tag      string `json:"tag"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
}

// Media is a representation of an image or video.
type Media struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Masquerade is name/avatar override information.
type Masquerade struct {
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Colour string `json:"colour,omitempty"`
}

// Member in a server.
type Member struct {
	ID       MemberCompositeKey `json:"_id"`
	Timeout  time.Time          `json:"timeout"`
	Avatar   *File              `json:"avatar"`
	Nickname *string            `json:"nickname"`
	Roles    []string           `json:"roles,omitempty"`
}

// MemberCompositeKey consists of server and user id.
type MemberCompositeKey struct {
	Server string `json:"server"`
	User   string `json:"user"`
}

// Message on Stoat.
type Message struct {
	Edited      time.Time  `json:"edited"`
	Masquerade  Masquerade `json:"masquerade"`
	Content     string     `json:"content"`
	Author      string     `json:"author"`
	Channel     string     `json:"channel"`
	ID          string     `json:"_id"`
	Attachments []File     `json:"attachments,omitempty"`
	Embeds      []Embed    `json:"embeds,omitempty"`
	Replies     []string   `json:"replies,omitempty"`
}

// MessageDeleteEvent represents the deletion of a message.
type MessageDeleteEvent struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
}

// MessageUpdateEvent represents a partial message update.
type MessageUpdateEvent struct {
	Data Message `json:"data"`
}

// OptionsBulkDelete are the options to delete many messages.
type OptionsBulkDelete struct {
	IDs []string `json:"ids"`
}

// Override is a representation of a single permission override.
type Override struct {
	Allow Permission `json:"allow"`
	Deny  Permission `json:"deny"`
}

// OverrideField is a representation of a single permission override as it appears on models and in the database.
type OverrideField struct {
	Allow Permission `json:"a"`
	Deny  Permission `json:"d"`
}

// ReplyIntent specifies what this message should reply to and how.
type ReplyIntent struct {
	ID              string `json:"id"`
	Mention         bool   `json:"mention"`
	FailIfNotExists bool   `json:"fail_if_not_exists"`
}

// RelationshipStatus is the status you have with a user.
type RelationshipStatus string

// Possible RelationshipStatus values.
const (
	RelationshipNone         RelationshipStatus = "None"
	RelationshipUser         RelationshipStatus = "User"
	RelationshipFriend       RelationshipStatus = "Friend"
	RelationshipOutgoing     RelationshipStatus = "Outgoing"
	RelationshipIncoming     RelationshipStatus = "Incoming"
	RelationshipBlocked      RelationshipStatus = "Blocked"
	RelationshipBlockedOther RelationshipStatus = "BlockedOther"
)

// ReadyEvent provides an initial data snapshot.
type ReadyEvent struct {
	Users    []User    `json:"users,omitempty"`
	Servers  []Server  `json:"servers,omitempty"`
	Channels []Channel `json:"channels,omitempty"`
	Members  []Member  `json:"members,omitempty"`
	Emojis   []Emoji   `json:"emojis,omitempty"`
}

// Role represents a Role in a Stoat server.
type Role struct {
	Permissions OverrideField `json:"permissions"`
}

// Server on Stoat.
type Server struct {
	Roles              map[string]Role `json:"roles"`
	ID                 string          `json:"_id"`
	Owner              string          `json:"owner"`
	DefaultPermissions Permission      `json:"default_permissions"`
}

// SendableEmbed is a representation of a text embed before it is sent.
type SendableEmbed struct {
	IconURL     string `json:"icon_url,omitempty"`
	URL         string `json:"url,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Media       string `json:"media,omitempty"`
	Colour      string `json:"colour,omitempty"`
}

// User on Stoat.
type User struct {
	Avatar       *File              `json:"avatar"`
	Relationship RelationshipStatus `json:"relationship"`
	ID           string             `json:"_id"`
	Username     string             `json:"username"`
}
