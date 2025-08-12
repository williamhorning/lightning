package guilded

import (
	"encoding/json"
	"time"

	"github.com/williamhorning/lightning/pkg/lightning"
)

type guildedChatEmbedAuthor struct {
	IconURL *string `json:"icon_url,omitempty"`
	Name    *string `json:"name,omitempty"`
	URL     *string `json:"url,omitempty"`
}

type guildedChatEmbedField struct {
	Inline *bool  `json:"inline,omitempty"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

type guildedChatEmbedFooter struct {
	IconURL *string `json:"icon_url,omitempty"`
	Text    string  `json:"text"`
}

type guildedChatEmbedMedia struct {
	URL *string `json:"url,omitempty"`
}

type guildedChatEmbed struct {
	Author      *guildedChatEmbedAuthor  `json:"author,omitempty"`
	Color       *int                     `json:"color,omitempty"`
	Description *string                  `json:"description,omitempty"`
	Fields      *[]guildedChatEmbedField `json:"fields,omitempty"`
	Footer      *guildedChatEmbedFooter  `json:"footer,omitempty"`
	Image       *guildedChatEmbedMedia   `json:"image,omitempty"`
	Thumbnail   *guildedChatEmbedMedia   `json:"thumbnail,omitempty"`
	Timestamp   *time.Time               `json:"timestamp,omitempty"`
	Title       *string                  `json:"title,omitempty"`
	URL         *string                  `json:"url,omitempty"`
}

type guildedChatMessage struct {
	CreatedAt             time.Time           `json:"createdAt"`
	Content               *string             `json:"content,omitempty"`
	CreatedByWebhookID    *string             `json:"createdByWebhookId,omitempty"`
	Embeds                *[]guildedChatEmbed `json:"embeds,omitempty"`
	GroupID               *string             `json:"groupId,omitempty"`
	HiddenLinkPreviewURLs *[]string           `json:"hiddenLinkPreviewURLs,omitempty"`
	IsPinned              *bool               `json:"isPinned,omitempty"`
	IsPrivate             *bool               `json:"isPrivate,omitempty"`
	IsSilent              *bool               `json:"isSilent,omitempty"`
	ReplyMessageIDs       *[]string           `json:"replyMessageIds,omitempty"`
	ServerID              *string             `json:"serverId,omitempty"`
	UpdatedAt             *time.Time          `json:"updatedAt,omitempty"`
	ChannelID             string              `json:"channelId"`
	CreatedBy             string              `json:"createdBy"`
	ID                    string              `json:"id"`
	Type                  string              `json:"type"`
}

type guildedChatMessageResponse struct {
	Message guildedChatMessage `json:"message"`
}

type guildedChatMessageCreated struct {
	Message  guildedChatMessage `json:"message"`
	ServerID string             `json:"serverId"`
}

type guildedChatMessageDeleted struct {
	DeletedAt time.Time          `json:"deletedAt"`
	Message   guildedChatMessage `json:"message"`
	ServerID  string             `json:"serverId"`
}

type guildedChatMessageUpdated struct {
	Message  guildedChatMessage `json:"message"`
	ServerID string             `json:"serverId"`
}

type guildedPayload struct {
	Content         string             `json:"content"`
	AvatarURL       string             `json:"avatar_url,omitempty"`
	Username        string             `json:"username,omitempty"`
	Embeds          []guildedChatEmbed `json:"embeds,omitempty"`
	ReplyMessageIDs []string           `json:"replyMessageIds,omitempty"`
}

type guildedServerChannel struct {
	CreatedAt  time.Time  `json:"createdAt"`
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	ArchivedBy *string    `json:"archivedBy,omitempty"`
	CategoryID *int       `json:"categoryId,omitempty"`
	MessageID  *string    `json:"messageId,omitempty"`
	ParentID   *string    `json:"parentId,omitempty"`
	Priority   *int       `json:"priority,omitempty"`
	RootID     *string    `json:"rootId,omitempty"`
	Topic      *string    `json:"topic,omitempty"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	Visibility *string    `json:"visibility"`
	CreatedBy  string     `json:"createdBy"`
	GroupID    string     `json:"groupId"`
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	ServerID   string     `json:"serverId"`
	Type       string     `json:"type"`
}

type guildedServerChannelResponse struct {
	Channel guildedServerChannel `json:"channel"`
}

type guildedServerMember struct {
	User     guildedUser `json:"user"`
	JoinedAt time.Time   `json:"joinedAt"`
	IsOwner  *bool       `json:"isOwner,omitempty"`
	Nickname *string     `json:"nickname,omitempty"`
	RoleIDs  []int       `json:"roleIds"`
}

func (sm *guildedServerMember) toAuthor() lightning.MessageAuthor {
	nickname := sm.User.Name

	if sm.Nickname != nil {
		nickname = *sm.Nickname
	}

	return lightning.MessageAuthor{
		Nickname:       nickname,
		Username:       sm.User.Name,
		ID:             sm.User.ID,
		ProfilePicture: sm.User.Avatar,
	}
}

type guildedServerMemberResponse struct {
	Member guildedServerMember `json:"member"`
}

type guildedSocketEventEnvelope struct {
	S  *string         `json:"s,omitempty"`
	T  *string         `json:"t,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
	Op int             `json:"op"`
}

type guildedURLSignature struct {
	RetryAfter *int    `json:"retryAfter,omitempty"`
	Signature  *string `json:"signature,omitempty"`
	URL        string  `json:"url"`
}

type guildedURLSignatureResponse struct {
	URLSignatures []guildedURLSignature `json:"urlSignatures"`
}

type guildedUser struct {
	CreatedAt time.Time          `json:"createdAt"`
	Avatar    *string            `json:"avatar,omitempty"`
	Banner    *string            `json:"banner,omitempty"`
	Status    *guildedUserStatus `json:"status,omitempty"`
	Type      *string            `json:"type,omitempty"`
	ID        string             `json:"id"`
	Name      string             `json:"name"`
}

type guildedUserStatus struct {
	Content *string `json:"content,omitempty"`
	EmoteID int     `json:"emoteId"`
}

type guildedWebhook struct {
	Avatar *string `json:"avatar,omitempty"`
	Token  *string `json:"token,omitempty"`
	ID     string  `json:"id"`
	Name   string  `json:"name"`
}

func (wh *guildedWebhook) toAuthor() lightning.MessageAuthor {
	return lightning.MessageAuthor{
		Nickname:       wh.Name,
		Username:       wh.Name,
		ID:             wh.ID,
		ProfilePicture: wh.Avatar,
	}
}

type guildedWebhookResponse struct {
	Webhook guildedWebhook `json:"webhook"`
}

type guildedWebhookExecuteResponse struct {
	ID string `json:"id"`
}

type guildedWelcomeMessage struct {
	User                guildedUser `json:"user"`
	BotID               string      `json:"botId"`
	LastMessageID       string      `json:"lastMessageId"`
	HeartbeatIntervalMs int         `json:"heartbeatIntervalMs"`
}
