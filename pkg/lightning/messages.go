package lightning

import "time"

var profilePicURL = "https://williamhorning.eu.org/assets/lightning/logo_color.svg"

type BaseMessage struct {
	EventID   string
	ChannelID string
	Plugin    string
	Time      time.Time
}

type Attachment struct {
	URL  string
	Name string
	Size float64
}

type Media struct {
	URL    string
	Height int
	Width  int
}

type EmbedAuthor struct {
	Name    string
	URL     *string
	IconURL *string
}

type EmbedField struct {
	Name   string
	Value  string
	Inline bool
}

type EmbedFooter struct {
	Text    string
	IconURL *string
}

type Embed struct {
	Author      *EmbedAuthor
	Color       *int
	Description *string
	Fields      []EmbedField
	Footer      *EmbedFooter
	Image       *Media
	Thumbnail   *Media
	Timestamp   *int64
	Title       *string
	URL         *string
	Video       *Media
}

type MessageAuthor struct {
	ID             string
	Nickname       string
	Username       string
	ProfilePicture *string
	Color          string
}

type Message struct {
	BaseMessage
	Author      MessageAuthor
	Content     string
	Attachments []Attachment
	Embeds      []Embed
	RepliedTo   []string
}

func CreateMessage(content string) Message {
	return Message{
		Content: content,
		Author: MessageAuthor{
			ID:             "lightning",
			Nickname:       "lightning",
			Username:       "lightning",
			ProfilePicture: &profilePicURL,
			Color:          "#487C7E",
		},
		BaseMessage: BaseMessage{
			Time:   time.Now(),
			Plugin: "lightning",
		},
	}
}
