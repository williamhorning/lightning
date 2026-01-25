package lightning

import "time"

// An Attachment on a [Message].
type Attachment struct {
	URL  string
	Name string
	Size int64
}

// BaseMessage is basic message information, such as an ID, channel, and timestamp.
type BaseMessage struct {
	Time      time.Time
	EventID   string
	ChannelID string
}

func (b *BaseMessage) setChannelID(id string) {
	b.ChannelID = id + "::" + b.ChannelID
}

// ChannelDisabled represents whether to disable a channel due to possible errors.
type ChannelDisabled struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

// ChannelDisabler is an interface that allows a channel to be disabled in an external system.
type ChannelDisabler interface {
	Disable() *ChannelDisabled
}

// A CommandArgument is a possible argument for a [Command].
type CommandArgument struct {
	Name        string
	Description string
}

// CommandOptions are provided to a [Command] executor.
type CommandOptions struct {
	BaseMessage

	Arguments map[string]string
	Author    *MessageAuthor
	Bot       *Bot
	Reply     func(message *Message, sensitive bool)
	Prefix    string
}

// A Command registered with [Bot].
type Command struct {
	Executor    func(options *CommandOptions)
	Name        string
	Description string
	Subcommands map[string]Command
	Arguments   []CommandArgument
}

// CommandEvent represents an execution of a command on a platform.
type CommandEvent struct {
	*CommandOptions

	Subcommand *string
	Command    string
	Options    []string
}

// EditedMessage is information about an edited message.
type EditedMessage struct {
	*Message

	Edited time.Time
}

// EmbedAuthor is an author on an [Embed].
type EmbedAuthor struct {
	URL     string
	IconURL string
	Name    string
}

// EmbedField is a field on an [Embed].
type EmbedField struct {
	Name   string
	Value  string
	Inline bool
}

// EmbedFooter is a footer on an [Embed].
type EmbedFooter struct {
	IconURL string
	Text    string
}

// Embed is a Discord-style embed.
type Embed struct {
	Author      *EmbedAuthor
	Footer      *EmbedFooter
	Image       *Media
	Thumbnail   *Media
	Video       *Media
	Timestamp   string
	Title       string
	URL         string
	Description string
	Fields      []EmbedField
	Color       int
}

// Emoji represents custom emoji in a [Message].
type Emoji struct {
	URL  string
	ID   string
	Name string
}

// Media represents images/videos on an [Embed].
type Media struct {
	URL    string
	Height int
	Width  int
}

// MessageAuthor is an author on an [Message].
type MessageAuthor struct {
	ID             string
	Username       string
	ProfilePicture string
	Color          string
}

// Message is a representation of a message on a platform.
type Message struct {
	BaseMessage

	Author      *MessageAuthor
	Content     string
	Attachments []Attachment
	Embeds      []Embed
	Emoji       []Emoji
	RepliedTo   []string
}

// SendOptions is possible options to use when sending a message.
type SendOptions struct {
	CommandUser        string
	ChannelData        map[string]string
	CommandResponse    bool
	AllowEveryonePings bool
}
