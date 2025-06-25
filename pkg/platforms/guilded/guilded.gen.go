package guilded

import (
	"time"
)

type guildedChatEmbedAuthor struct {
	IconUrl *string `json:"icon_url,omitempty"`
	Name    *string `json:"name,omitempty"`
	Url     *string `json:"url,omitempty"`
}

type guildedChatEmbed struct {
	Author      *guildedChatEmbedAuthor `json:"author,omitempty"`
	Color       *int                    `json:"color,omitempty"`
	Description *string                 `json:"description,omitempty"`
	Fields      *[]struct {
		Inline *bool  `json:"inline,omitempty"`
		Name   string `json:"name"`
		Value  string `json:"value"`
	} `json:"fields,omitempty"`
	Footer *struct {
		IconUrl *string `json:"icon_url,omitempty"`
		Text    string  `json:"text"`
	} `json:"footer,omitempty"`
	Image *struct {
		Url *string `json:"url,omitempty"`
	} `json:"image,omitempty"`
	Thumbnail *struct {
		Url *string `json:"url,omitempty"`
	} `json:"thumbnail,omitempty"`
	Timestamp *time.Time `json:"timestamp,omitempty"`
	Title     *string    `json:"title,omitempty"`
	Url       *string    `json:"url,omitempty"`
}

type guildedChatMessage struct {
	ChannelId             string              `json:"channelId"`
	Content               *string             `json:"content,omitempty"`
	CreatedAt             time.Time           `json:"createdAt"`
	CreatedBy             string              `json:"createdBy"`
	CreatedByWebhookId    *string             `json:"createdByWebhookId,omitempty"`
	Embeds                *[]guildedChatEmbed `json:"embeds,omitempty"`
	GroupId               *string             `json:"groupId,omitempty"`
	HiddenLinkPreviewUrls *[]string           `json:"hiddenLinkPreviewUrls,omitempty"`
	Id                    string              `json:"id"`
	IsPinned              *bool               `json:"isPinned,omitempty"`
	IsPrivate             *bool               `json:"isPrivate,omitempty"`
	IsSilent              *bool               `json:"isSilent,omitempty"`
	Mentions              *guildedMentions    `json:"mentions,omitempty"`
	ReplyMessageIds       *[]string           `json:"replyMessageIds,omitempty"`
	ServerId              *string             `json:"serverId,omitempty"`
	Type                  string              `json:"type"`
	UpdatedAt             *time.Time          `json:"updatedAt,omitempty"`
}

type guildedChatMessageCreated struct {
	Message  guildedChatMessage `json:"message"`
	ServerId string             `json:"serverId"`
}

type guildedChatMessageDeleted struct {
	DeletedAt time.Time          `json:"deletedAt"`
	Message   guildedChatMessage `json:"message"`
	ServerId  string             `json:"serverId"`
}

type guildedChatMessageUpdated struct {
	Message  guildedChatMessage `json:"message"`
	ServerId string             `json:"serverId"`
}

type guildedMentions struct {
	Channels *[]struct {
		Id string `json:"id"`
	} `json:"channels,omitempty"`

	Everyone *bool `json:"everyone,omitempty"`

	Here *bool `json:"here,omitempty"`

	Roles *[]struct {
		Id int `json:"id"`
	} `json:"roles,omitempty"`

	Users *[]struct {
		Id string `json:"id"`
	} `json:"users,omitempty"`
}

type guildedServerChannel struct {
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	ArchivedBy *string    `json:"archivedBy,omitempty"`
	CategoryId *int       `json:"categoryId,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	CreatedBy  string     `json:"createdBy"`
	GroupId    string     `json:"groupId"`
	Id         string     `json:"id"`
	MessageId  *string    `json:"messageId,omitempty"`
	Name       string     `json:"name"`
	ParentId   *string    `json:"parentId,omitempty"`
	Priority   *int       `json:"priority,omitempty"`
	RootId     *string    `json:"rootId,omitempty"`
	ServerId   string     `json:"serverId"`
	Topic      *string    `json:"topic,omitempty"`
	Type       string     `json:"type"`
	UpdatedAt  *time.Time `json:"updatedAt,omitempty"`
	Visibility *string    `json:"visibility"`
}

type guildedServerMember struct {
	IsOwner  *bool       `json:"isOwner,omitempty"`
	JoinedAt time.Time   `json:"joinedAt"`
	Nickname *string     `json:"nickname,omitempty"`
	RoleIds  []int       `json:"roleIds"`
	User     guildedUser `json:"user"`
}

type guildedSocketEventEnvelope struct {
	D  *map[string]any `json:"d,omitempty"`
	Op int             `json:"op"`
	S  *string         `json:"s,omitempty"`
	T  *string         `json:"t,omitempty"`
}

type guildedUrlSignature struct {
	RetryAfter *int    `json:"retryAfter,omitempty"`
	Signature  *string `json:"signature,omitempty"`
	Url        string  `json:"url"`
}

type guildedUser struct {
	Avatar    *string            `json:"avatar,omitempty"`
	Banner    *string            `json:"banner,omitempty"`
	CreatedAt time.Time          `json:"createdAt"`
	Id        string             `json:"id"`
	Name      string             `json:"name"`
	Status    *guildedUserStatus `json:"status,omitempty"`
	Type      *string            `json:"type,omitempty"`
}

type guildedUserStatus struct {
	Content *string `json:"content,omitempty"`
	EmoteId int     `json:"emoteId"`
}

type guildedWebhook struct {
	Avatar    *string    `json:"avatar,omitempty"`
	ChannelId string     `json:"channelId"`
	CreatedAt time.Time  `json:"createdAt"`
	CreatedBy string     `json:"createdBy"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	ServerId  string     `json:"serverId"`
	Token     *string    `json:"token,omitempty"`
}

type guildedWelcomeMessage struct {
	BotId               string      `json:"botId"`
	HeartbeatIntervalMs int         `json:"heartbeatIntervalMs"`
	LastMessageId       string      `json:"lastMessageId"`
	User                guildedUser `json:"user"`
}
