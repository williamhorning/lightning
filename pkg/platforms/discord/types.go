package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"codeberg.org/jersey/lightning/pkg/lightning"
)

type apiError struct {
	Code     apiErrorCode   `json:"code"`
	Errors   apiFieldErrors `json:"errors,omitempty"`
	Message  string         `json:"message"`
	Request  *http.Request  `json:"-"`
	Response *http.Response `json:"-"`
}

func (a apiError) Disable() *lightning.ChannelDisabled {
	switch a.Code {
	case errUnknownChannel:
		return &lightning.ChannelDisabled{Read: true, Write: true}
	case errMaximumNumberOfWebhooksReached,
		errMissingPermissions,
		errUnknownWebhook,
		errInvalidWebhookTokenProvided:
		return &lightning.ChannelDisabled{Read: false, Write: true}
	default:
		return &lightning.ChannelDisabled{Read: false, Write: false}
	}
}

func (a apiError) Error() string {
	str := "Discord API Error: " + a.Request.Method + " " + a.Request.URL.Path + " status " +
		strconv.FormatInt(int64(a.Response.StatusCode), 10) + " code " + strconv.FormatInt(int64(a.Code), 10) + ";"

	for field, detail := range a.Errors {
		for _, e := range detail.Errors {
			str += " " + field + ": [" + e.Code + "] " + e.Message + ";"
		}
	}

	return str
}

type apiErrorCode int

const (
	errInvalidWebhookTokenProvided    apiErrorCode = 50027
	errMaximumNumberOfWebhooksReached apiErrorCode = 30007
	errMissingPermissions             apiErrorCode = 50013
	errUnknownChannel                 apiErrorCode = 10003
	errUnknownWebhook                 apiErrorCode = 10015
)

type apiErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type apiFieldErrors map[string]struct {
	Errors []apiErrorDetail `json:"_errors"` //nolint:tagliatelle
}

type allowedMentions struct {
	Parse []allowedMentionsType `json:"parse,omitempty"`
}

type allowedMentionsType string

const (
	allowedMentionRoles allowedMentionsType = "roles"
	allowedMentionUsers allowedMentionsType = "users"
)

type application struct {
	ID snowflake `json:"id,omitempty"`
}

type applicationCommand struct {
	Description string          `json:"description"`
	Name        string          `json:"name"`
	Options     []commandOption `json:"options,omitempty"`
	Type        commandType     `json:"type"`
}

type attachment struct {
	Filename string `json:"filename"`
	Size     int    `json:"size"`
	URL      string `json:"url"`
}

type buttonStyle int

const btnLink buttonStyle = 5

type channel struct {
	GuildID *snowflake `json:"guild_id,omitempty"`
	ID      snowflake  `json:"id"`
	Name    string     `json:"name,omitempty"`
}

type connState int

const (
	stateConnecting connState = iota
	stateConnected
	stateReconnecting
	stateTerminal
)

type commandOption struct {
	Description string            `json:"description"`
	Name        string            `json:"name"`
	Options     []commandOption   `json:"options,omitempty"`
	Required    bool              `json:"required,omitempty"`
	Type        commandOptionType `json:"type"`
}

type commandOptionType int

const (
	optString     commandOptionType = 3
	optSubCommand commandOptionType = 1
)

type commandType int

const commandTypeChatInput commandType = 1

type componentType int

const (
	compActionRow componentType = 1
	compButton    componentType = 2
)

type component struct {
	Components []component   `json:"components,omitempty"`
	Label      *string       `json:"label,omitempty"`
	Style      *buttonStyle  `json:"style,omitempty"`
	Type       componentType `json:"type"`
	URL        *string       `json:"url,omitempty"`
}

type discordEmoji struct {
	Animated bool      `json:"animated"`
	Guild    snowflake `json:"-"`
	ID       snowflake `json:"id"`
	Name     string    `json:"name"`
}

type discordEmojiEvent struct {
	Guild  snowflake      `json:"guild_id"`
	Emojis []discordEmoji `json:"emojis"`
}

type embed struct {
	Author      *embedAuthor `json:"author,omitempty"`
	Color       int          `json:"color,omitempty"`
	Description string       `json:"description,omitempty"`
	Fields      []embedField `json:"fields,omitempty"`
	Footer      *embedFooter `json:"footer,omitempty"`
	Image       *embedMedia  `json:"image,omitempty"`
	Thumbnail   *embedMedia  `json:"thumbnail,omitempty"`
	Timestamp   *time.Time   `json:"timestamp,omitempty"`
	Title       string       `json:"title,omitempty"`
	URL         string       `json:"url,omitempty"`
	Video       *embedMedia  `json:"video,omitempty"`
}

type embedAuthor struct {
	IconURL string `json:"icon_url,omitempty"`
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
}

type embedField struct {
	Inline bool   `json:"inline,omitempty"`
	Name   string `json:"name"`
	Value  string `json:"value"`
}

type embedFooter struct {
	IconURL string `json:"icon_url,omitempty"`
	Text    string `json:"text"`
}

type embedMedia struct {
	Height *int   `json:"height,omitempty"`
	URL    string `json:"url"`
	Width  *int   `json:"width,omitempty"`
}

type eventType string

const (
	eventReady             eventType = "READY"
	eventResumed           eventType = "RESUMED"
	eventChannelCreate     eventType = "CHANNEL_CREATE"
	eventChannelUpdate     eventType = "CHANNEL_UPDATE"
	eventGuildCreate       eventType = "GUILD_CREATE"
	eventGuildUpdate       eventType = "GUILD_UPDATE"
	eventRoleCreate        eventType = "GUILD_ROLE_CREATE"
	eventRoleUpdate        eventType = "GUILD_ROLE_UPDATE"
	eventEmojisUpdate      eventType = "GUILD_EMOJIS_UPDATE"
	eventMessageCreate     eventType = "MESSAGE_CREATE"
	eventMessageEdit       eventType = "MESSAGE_UPDATE"
	eventMessageDelete     eventType = "MESSAGE_DELETE"
	eventInteractionCreate eventType = "INTERACTION_CREATE"
)

type file struct {
	Cancel func()
	Name   string
	Reader io.Reader
}

type gatewayHello struct {
	Interval int `json:"heartbeat_interval"`
}

type gatewayIdentify struct {
	Token      string                    `json:"token"`
	Properties gatewayIdentifyProperties `json:"properties"`
	Intents    intent                    `json:"intents"`
}

type gatewayIdentifyProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}

type gatewayMessage struct {
	Op int             `json:"op"`
	S  *int64          `json:"s,omitempty"`
	T  eventType       `json:"t,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

type gatewayResume struct {
	Token     string `json:"token"`
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
}

type guild struct {
	ID          snowflake   `json:"id"`
	Unavailable bool        `json:"unavailable,omitempty"`
	PremiumTier premiumTier `json:"premium_tier,omitempty"`
	Roles       []role      `json:"roles,omitempty"`
	OwnerID     string      `json:"owner_id"`
}

type intent int

const (
	intentGuilds                 intent = 1 << 0
	intentGuildModeration        intent = 1 << 2
	intentGuildEmojis            intent = 1 << 3
	intentGuildIntegrations      intent = 1 << 4
	intentGuildWebhooks          intent = 1 << 5
	intentGuildInvites           intent = 1 << 6
	intentGuildVoiceStates       intent = 1 << 7
	intentGuildMessages          intent = 1 << 9
	intentGuildMessageReactions  intent = 1 << 10
	intentGuildMessageTyping     intent = 1 << 11
	intentDirectMessages         intent = 1 << 12
	intentDirectMessageReactions intent = 1 << 13
	intentDirectMessageTyping    intent = 1 << 14
	intentMessageContent         intent = 1 << 15

	intentsNotPrivileged = intentGuilds |
		intentGuildModeration |
		intentGuildEmojis |
		intentGuildIntegrations |
		intentGuildWebhooks |
		intentGuildInvites |
		intentGuildVoiceStates |
		intentGuildMessages |
		intentGuildMessageReactions |
		intentGuildMessageTyping |
		intentDirectMessages |
		intentDirectMessageReactions |
		intentDirectMessageTyping
)

type interactionCreateEvent struct {
	User      *user            `json:"user,omitempty"`
	Member    *member          `json:"member,omitempty"`
	ChannelID *snowflake       `json:"channel_id,omitempty"`
	GuildID   *snowflake       `json:"guild_id,omitempty"`
	Data      *interactionData `json:"data,omitempty"`
	ID        snowflake        `json:"id"`
	Token     string           `json:"token"`
	Type      interactionType  `json:"type"`
}

func (i *interactionCreateEvent) getUser() *user {
	if i.User != nil {
		return i.User
	}

	if i.Member != nil && i.Member.User != nil {
		return i.Member.User
	}

	return nil
}

type interactionData struct {
	Name    string              `json:"name"`
	Options []interactionOption `json:"options,omitempty"`
}

type interactionOption struct {
	Name    string              `json:"name"`
	Options []interactionOption `json:"options,omitempty"`
	Type    commandOptionType   `json:"type"`
	Value   string              `json:"value,omitempty"`
}

type interactionResponse struct {
	Data *interactionResponseData `json:"data,omitempty"`
	Type interactionResponseType  `json:"type,omitempty"`
}

type interactionResponseData struct {
	AllowedMentions *allowedMentions `json:"allowed_mentions,omitempty"`
	Components      []component      `json:"components,omitempty"`
	Content         string           `json:"content,omitempty"`
	Embeds          []embed          `json:"embeds,omitempty"`
	Files           []file           `json:"-"`
	Flags           messageFlags     `json:"flags,omitempty"`
}

type interactionResponseType int

type interactionType int

const interactionApplicationCommand interactionType = 2

type member struct {
	Avatar *string  `json:"avatar,omitempty"`
	Nick   *string  `json:"nick,omitempty"`
	User   *user    `json:"user,omitempty"`
	Roles  []string `json:"roles,omitempty"`
}

func (m *member) avatarURL(client *client, guild snowflake) string {
	if m == nil {
		return ""
	}

	if m.Avatar != nil && *m.Avatar != "" {
		hash := *m.Avatar

		if strings.HasPrefix(hash, "a_") {
			return "https://" + client.cdnHost + "/guilds/" +
				string(guild) + "/users/" + string(m.User.ID) +
				"/avatars/" + hash + ".gif"
		}

		return "https://" + client.cdnHost + "/guilds/" +
			string(guild) + "/users/" + string(m.User.ID) +
			"/avatars/" + hash + ".webp"
	}

	return m.User.avatarURL(client)
}

func (m *member) displayName() string {
	if m == nil {
		return ""
	}

	if m.Nick != nil && *m.Nick != "" {
		return *m.Nick
	}

	return m.User.displayName()
}

type message struct {
	AllowedMentions  *allowedMentions  `json:"allowed_mentions,omitempty"`
	Attachments      []attachment      `json:"attachments,omitempty"`
	Author           user              `json:"author"`
	ChannelID        snowflake         `json:"channel_id,omitempty"`
	Content          string            `json:"content,omitempty"`
	EditedTimestamp  *time.Time        `json:"edited_timestamp,omitempty"`
	Embeds           []embed           `json:"embeds,omitempty"`
	GuildID          *snowflake        `json:"guild_id,omitempty"`
	ID               snowflake         `json:"id,omitempty"`
	Member           *member           `json:"member,omitempty"`
	MessageReference *messageReference `json:"message_reference,omitempty"`
	MessageSnapshots []messageSnapshot `json:"message_snapshots,omitempty"`
	StickerItems     []stickerItem     `json:"sticker_items,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
	Type             messageType       `json:"type,omitempty"`
	WebhookID        *snowflake        `json:"webhook_id,omitempty"`
}

type messageEdit struct {
	AllowedMentions *allowedMentions `json:"allowed_mentions,omitempty"`
	Components      *[]component     `json:"components,omitempty"`
	Content         *string          `json:"content,omitempty"`
	Embeds          *[]embed         `json:"embeds,omitempty"`
	Flags           messageFlags     `json:"flags,omitempty"`
}

type messageDelete struct {
	ID        snowflake `json:"id"`
	ChannelID snowflake `json:"channel_id"`
	GuildID   snowflake `json:"guild_id,omitempty"`
}

type messageFlags int

const messageFlagsEphemeral messageFlags = 1 << 6

type messageReference struct {
	ChannelID       snowflake            `json:"channel_id,omitempty"`
	FailIfNotExists bool                 `json:"fail_if_not_exists,omitempty"`
	MessageID       snowflake            `json:"message_id,omitempty"`
	Type            messageReferenceType `json:"type"`
}

type messageReferenceType int

const (
	defaultReference messageReferenceType = 0
	forwardReference messageReferenceType = 1
)

type messageSend struct {
	AllowedMentions *allowedMentions  `json:"allowed_mentions,omitempty"`
	Components      []component       `json:"components,omitempty"`
	Content         string            `json:"content,omitempty"`
	Embeds          []embed           `json:"embeds,omitempty"`
	Files           []file            `json:"-"`
	Flags           messageFlags      `json:"flags,omitempty"`
	Reference       *messageReference `json:"message_reference,omitempty"`
}

type messageSnapshot struct {
	Content string `json:"content,omitempty"`
}

type messageType int

const (
	messageTypeDefault            messageType = 0
	messageTypeReply              messageType = 19
	messageTypeChatInputCommand   messageType = 20
	messageTypeContextMenuCommand messageType = 23
)

type permissionCheckError struct {
	text string
}

func (p *permissionCheckError) Error() string {
	return "failed to " + p.text
}

type premiumTier int

const (
	premiumNone premiumTier = 0
	premium1    premiumTier = 1
	premium2    premiumTier = 2
	premium3    premiumTier = 3
)

type readyEvent struct {
	Guilds      []guild     `json:"guilds"`
	SessionID   string      `json:"session_id"`
	User        user        `json:"user"`
	ResumeURL   string      `json:"resume_gateway_url"`
	Application application `json:"application"`
}

type role struct {
	ID          snowflake `json:"id"`
	Name        string    `json:"name"`
	Permissions string    `json:"permissions"`
}

type roleEvent struct {
	GuildID snowflake `json:"guild_id"`
	Role    role      `json:"role"`
}

type snowflake string

func (s *snowflake) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)

	if bytes.Equal(data, []byte("null")) {
		*s = ""

		return nil
	}

	if len(data) > 0 && data[0] == '"' {
		str, err := strconv.Unquote(string(data))
		if err != nil {
			return fmt.Errorf("failed to unquote: %w", err)
		}

		*s = snowflake(str)

		return nil
	}

	*s = snowflake(data)

	return nil
}

type stickerFormat int

const (
	stickerPNG    stickerFormat = 1
	stickerAPNG   stickerFormat = 2
	stickerLottie stickerFormat = 3
	stickerGIF    stickerFormat = 4
)

type stickerItem struct {
	FormatType stickerFormat `json:"format_type"`
	ID         string        `json:"id"`
	Name       string        `json:"name"`
}

type user struct {
	Avatar        *string   `json:"avatar,omitempty"`
	Discriminator *string   `json:"discriminator,omitempty"`
	GlobalName    *string   `json:"global_name,omitempty"`
	ID            snowflake `json:"id"`
	Username      string    `json:"username"`
}

func (u *user) avatarURL(client *client) string {
	if u == nil {
		return ""
	}

	if u.Avatar != nil && *u.Avatar != "" {
		hash := *u.Avatar

		if strings.HasPrefix(hash, "a_") {
			return "https://" + client.cdnHost + "/avatars/" +
				string(u.ID) + "/" + hash + ".gif"
		}

		return "https://" + client.cdnHost + "/avatars/" +
			string(u.ID) + "/" + hash + ".webp"
	}

	return u.defaultAvatarURL(client)
}

func (u *user) defaultAvatarURL(client *client) string {
	if u == nil {
		return ""
	}

	var index uint64

	if u.Discriminator != nil && *u.Discriminator != "" {
		if di, err := strconv.ParseUint(*u.Discriminator, 10, 64); err == nil {
			index = di % 5
		}
	} else {
		if idInt, err := strconv.ParseUint(string(u.ID), 10, 64); err == nil {
			index = (idInt >> 22) % 6
		}
	}

	return "https://" + client.cdnHost + "/embed/avatars/" + strconv.FormatUint(index, 10) + ".webp"
}

func (u *user) displayName() string {
	if u == nil {
		return ""
	}

	if u.GlobalName != nil && *u.GlobalName != "" {
		return *u.GlobalName
	}

	return u.Username
}

type webhook struct {
	ApplicationID snowflake `json:"application_id,omitempty"`
	ID            snowflake `json:"id"`
	Token         string    `json:"token,omitempty"`
}

type webhookEditMessagePayload struct {
	AllowedMentions *allowedMentions `json:"allowed_mentions,omitempty"`
	Components      []component      `json:"components,omitempty"`
	Content         *string          `json:"content,omitempty"`
	Embeds          []embed          `json:"embeds,omitempty"`
	Flags           messageFlags     `json:"flags,omitempty"`
}

type webhookExecutePayload struct {
	AllowedMentions *allowedMentions `json:"allowed_mentions,omitempty"`
	Attachments     []attachment     `json:"attachments,omitempty"`
	AvatarURL       string           `json:"avatar_url,omitempty"`
	Components      []component      `json:"components,omitempty"`
	Content         string           `json:"content,omitempty"`
	Embeds          []embed          `json:"embeds,omitempty"`
	Files           []file           `json:"-"`
	Flags           messageFlags     `json:"flags,omitempty"`
	Username        string           `json:"username,omitempty"`
}
