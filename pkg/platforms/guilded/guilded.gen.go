package guilded

import "time"

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
	ChannelID             string              `json:"channelID"`
	Content               *string             `json:"content,omitempty"`
	CreatedAt             time.Time           `json:"createdAt"`
	CreatedBy             string              `json:"createdBy"`
	CreatedByWebhookID    *string             `json:"createdByWebhookID,omitempty"`
	Embeds                *[]guildedChatEmbed `json:"embeds,omitempty"`
	GroupID               *string             `json:"groupID,omitempty"`
	HiddenLinkPreviewURLs *[]string           `json:"hiddenLinkPreviewURLs,omitempty"`
	ID                    string              `json:"id"`
	IsPinned              *bool               `json:"isPinned,omitempty"`
	IsPrivate             *bool               `json:"isPrivate,omitempty"`
	IsSilent              *bool               `json:"isSilent,omitempty"`
	Mentions              *guildedMentions    `json:"mentions,omitempty"`
	ReplyMessageIDs       *[]string           `json:"replyMessageIDs,omitempty"`
	ServerID              *string             `json:"serverID,omitempty"`
	Type                  string              `json:"type"`
	UpdatedAt             *time.Time          `json:"updatedAt,omitempty"`
}

type guildedChatMessageResponse struct {
	Message guildedChatMessage `json:"message"`
}

type guildedChatMessageCreated struct {
	Message  guildedChatMessage `json:"message"`
	ServerID string             `json:"serverID"`
}

type guildedChatMessageDeleted struct {
	DeletedAt time.Time          `json:"deletedAt"`
	Message   guildedChatMessage `json:"message"`
	ServerID  string             `json:"serverID"`
}

type guildedChatMessageUpdated struct {
	Message  guildedChatMessage `json:"message"`
	ServerID string             `json:"serverID"`
}

type guildedMentions struct {
	Channels *[]struct {
		ID string `json:"id"`
	} `json:"channels,omitempty"`

	Everyone *bool `json:"everyone,omitempty"`

	Here *bool `json:"here,omitempty"`

	Roles *[]struct {
		ID int `json:"id"`
	} `json:"roles,omitempty"`

	Users *[]struct {
		ID string `json:"id"`
	} `json:"users,omitempty"`
}

type guildedPayload struct {
	Content         string             `json:"content"`
	Embeds          []guildedChatEmbed `json:"embeds,omitempty"`
	ReplyMessageIDs []string           `json:"replyMessageIDs,omitempty"`
	AvatarURL       string             `json:"avatar_url,omitempty"`
	Username        string             `json:"username,omitempty"`
}

type guildedServerChannel struct {
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	ArchivedBy *string    `json:"archivedBy,omitempty"`
	CategoryID *int       `json:"categoryID,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	CreatedBy  string     `json:"createdBy"`
	GroupID    string     `json:"groupID"`
	ID         string     `json:"id"`
	MessageID  *string    `json:"messageID,omitempty"`
	Name       string     `json:"name"`
	ParentID   *string    `json:"parentID,omitempty"`
	Priority   *int       `json:"priority,omitempty"`
	RootID     *string    `json:"rootID,omitempty"`
	ServerID   string     `json:"serverID"`
	Topic      *string    `json:"topic,omitempty"`
	Type       string     `json:"type"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	Visibility *string    `json:"visibility"`
}

type guildedServerChannelResponse struct {
	Channel guildedServerChannel `json:"channel"`
}

type guildedServerMember struct {
	IsOwner  *bool       `json:"isOwner,omitempty"`
	JoinedAt time.Time   `json:"joinedAt"`
	Nickname *string     `json:"nickname,omitempty"`
	RoleIDs  []int       `json:"roleIDs"`
	User     guildedUser `json:"user"`
}

type guildedServerMemberResponse struct {
	Member guildedServerMember `json:"member"`
}

type guildedSocketEventEnvelope struct {
	D  *map[string]any `json:"d,omitempty"`
	Op int             `json:"op"`
	S  *string         `json:"s,omitempty"`
	T  *string         `json:"t,omitempty"`
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
	Avatar    *string            `json:"avatar,omitempty"`
	Banner    *string            `json:"banner,omitempty"`
	CreatedAt time.Time          `json:"createdAt"`
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Status    *guildedUserStatus `json:"status,omitempty"`
	Type      *string            `json:"type,omitempty"`
}

type guildedUserStatus struct {
	Content *string `json:"content,omitempty"`
	EmoteID int     `json:"emoteID"`
}

type guildedWebhook struct {
	Avatar    *string    `json:"avatar,omitempty"`
	ChannelID string     `json:"channelID"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	ServerID  string     `json:"serverID"`
	Token     *string    `json:"token,omitempty"`
}

type guildedWebhookResponse struct {
	Webhook guildedWebhook `json:"webhook"`
}

type guildedWebhookExecuteResponse struct {
	ID string `json:"id"`
}

type guildedWelcomeMessage struct {
	BotID               string      `json:"botID"`
	HeartbeatIntervalMs int         `json:"heartbeatIntervalMs"`
	LastMessageID       string      `json:"lastMessageID"`
	User                guildedUser `json:"user"`
}
