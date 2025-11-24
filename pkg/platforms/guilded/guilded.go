package guilded

import (
	"encoding/json"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type guildedChatMessage struct {
	CreatedAt          time.Time         `json:"createdAt"`
	Content            string            `json:"content,omitempty"`
	CreatedByWebhookID string            `json:"createdByWebhookId,omitempty"`
	ReplyMessageIDs    []string          `json:"replyMessageIds,omitempty"`
	ServerID           *string           `json:"serverId,omitempty"`
	UpdatedAt          time.Time         `json:"updatedAt,omitempty"`
	ChannelID          string            `json:"channelId"`
	CreatedBy          string            `json:"createdBy"`
	ID                 string            `json:"id"`
	Embeds             []lightning.Embed `json:"embeds,omitempty"`
}

type guildedChatMessageWrapper struct {
	Message guildedChatMessage `json:"message"`
}

type guildedChatMessageDeleted struct {
	DeletedAt time.Time          `json:"deletedAt"`
	Message   guildedChatMessage `json:"message"`
}

type guildedPayload struct {
	Content         string            `json:"content,omitempty"`
	AvatarURL       string            `json:"avatar_url,omitempty"`
	Username        string            `json:"username,omitempty"`
	Embeds          []lightning.Embed `json:"embeds,omitempty"`
	ReplyMessageIDs []string          `json:"replyMessageIds,omitempty"`
}

type guildedServerMember struct {
	Nickname *string     `json:"nickname,omitempty"`
	User     guildedUser `json:"user"`
}

type guildedSocketEventEnvelope struct {
	T  string          `json:"t"`
	D  json.RawMessage `json:"d"`
	Op int             `json:"op"`
}

type guildedURLSignature struct {
	RetryAfter *int    `json:"retryAfter,omitempty"`
	Signature  *string `json:"signature,omitempty"`
}

type guildedURLSignatureResponse struct {
	URLSignatures []guildedURLSignature `json:"urlSignatures"`
}

type guildedUser struct {
	Avatar string `json:"avatar,omitempty"`
	ID     string `json:"id"`
	Name   string `json:"name"`
}
