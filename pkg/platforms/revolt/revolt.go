package revolt

import (
	"encoding/json"
	"time"
)

type revoltMessage struct {
	Edited       time.Time                  `json:"edited"`
	Webhook      *revoltMessageWebhook      `json:"webhook"`
	System       *revoltMessageSystem       `json:"system"`
	Reactions    map[string][]string        `json:"reactions"`
	Interactions *revoltMessageInteractions `json:"interactions"`
	Masquerade   *revoltMessageMasquerade   `json:"masquerade"`
	ID           string                     `json:"_id"`
	Nonce        string                     `json:"nonce"`
	Channel      string                     `json:"channel"`
	Author       string                     `json:"author"`
	Content      string                     `json:"content"`
	Attachments  []*revoltAttachment        `json:"attachments"`
	Embeds       []*revoltMessageEmbed      `json:"embeds"`
	Mentions     []string                   `json:"mentions"`
	Replies      []string                   `json:"replies"`
}

type revoltMessageWebhook struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

type revoltMessageSystem struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type revoltAttachment struct {
	Metadata    *revoltAttachmentMetadata `json:"metadata"`
	ID          string                    `json:"_id"`
	Tag         string                    `json:"tag"`
	Filename    string                    `json:"filename"`
	ContentType string                    `json:"content_type"`
	MessageID   string                    `json:"message_id"`
	UserID      string                    `json:"user_id"`
	ServerID    string                    `json:"server_id"`
	ObjectID    string                    `json:"object_id"`
	Size        int                       `json:"size"`
	Deleted     bool                      `json:"deleted"`
	Reported    bool                      `json:"reported"`
}

type revoltMessageEmbed struct {
	Type        string                     `json:"type"`
	URL         string                     `json:"url,omitempty"`
	OriginalURL string                     `json:"original_url,omitempty"`
	Special     *revoltMessageEmbedSpecial `json:"special,omitempty"`
	Title       string                     `json:"title,omitempty"`
	Description string                     `json:"description,omitempty"`
	Image       *revoltMessageEmbedImage   `json:"image,omitempty"`
	Video       *revoltMessageEmbedVideo   `json:"video,omitempty"`
	SiteName    string                     `json:"site_name,omitempty"`
	IconURL     string                     `json:"icon_url,omitempty"`
	Color       string                     `json:"colour,omitempty"`
}

type revoltMessageEmbedSpecial struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
	ContentType string    `json:"content_type"`
}

type revoltMessageEmbedImage struct {
	Size   string `json:"size"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type revoltMessageEmbedVideo struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type revoltAttachmentMetadata struct {
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type revoltMessageInteractions struct {
	Reactions         []string `json:"reactions"`
	RestrictReactions bool     `json:"restrict_reactions"`
}

type revoltMessageMasquerade struct {
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Color  string `json:"colour,omitempty"`
}

type revoltServerMember struct {
	JoinedAt time.Time               `json:"joined_at"`
	Nickname *string                 `json:"nickname"`
	Avatar   *revoltAttachment       `json:"avatar"`
	Timeout  *time.Time              `json:"timeout"`
	ID       revoltMemberCompositeID `json:"_id"`
	Roles    []string                `json:"roles"`
}

type revoltMemberCompositeID struct {
	User   string `json:"user"`
	Server string `json:"server"`
}

type revoltUser struct {
	Avatar        *revoltAttachment      `json:"avatar"`
	Status        *revoltUserStatus      `json:"status"`
	Profile       *revoltUserProfile     `json:"profile"`
	Flags         *int                   `json:"flags"`
	Bot           *revoltBot             `json:"bot"`
	ID            string                 `json:"_id"`
	Username      string                 `json:"username"`
	Discriminator string                 `json:"discriminator"`
	DisplayName   string                 `json:"display_name"`
	Relationship  string                 `json:"relationship"`
	Relations     []*revoltUserRelations `json:"relations"`
	Badges        int                    `json:"badges"`
	Privileged    bool                   `json:"privileged"`
	Online        bool                   `json:"online"`
}

type revoltUserRelations struct {
	ID     string `json:"_id"`
	Status string `json:"status"`
}

type revoltUserStatus struct {
	Text     string `json:"text,omitempty"`
	Presence string `json:"presence"`
}

type revoltUserProfile struct {
	Background *revoltAttachment `json:"background,omitempty"`
	Content    string            `json:"content,omitempty"`
}

type revoltBot struct {
	ID                string `json:"_id"`
	Owner             string `json:"owner"`
	Token             string `json:"token"`
	InteractionsURL   string `json:"interactions_url"`
	TermsOfServiceURL string `json:"terms_of_service_url"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
	Flags             int    `json:"flags"`
	Public            bool   `json:"public"`
	Analytics         bool   `json:"analytics"`
	Discoverable      bool   `json:"discoverable"`
}

type revoltChannel struct {
	Icon               *revoltAttachment             `json:"icon"`
	DefaultPermissions *revoltPermissions            `json:"default_permissions"`
	RolePermissions    map[string]*revoltPermissions `json:"role_permissions"`
	Permissions        *uint                         `json:"permissions"`
	ID                 string                        `json:"_id"`
	Server             string                        `json:"server"`
	ChannelType        revoltChannelType             `json:"channel_type"`
	Name               string                        `json:"name"`
	Description        string                        `json:"description"`
	LastMessageID      string                        `json:"last_message_id"`
	Owner              string                        `json:"owner"`
	Recipients         []string                      `json:"recipients"`
	NSFW               bool                          `json:"nsfw"`
	Active             bool                          `json:"active"`
}

type revoltChannelType string

const (
	revoltChannelTypeSavedMessages revoltChannelType = "SavedMessages"
	revoltChannelTypeText          revoltChannelType = "TextChannel"
	revoltChannelTypeVoice         revoltChannelType = "VoiceChannel"
	revoltChannelTypeDM            revoltChannelType = "DirectMessage"
	revoltChannelTypeGroup         revoltChannelType = "Group"
)

type revoltPermissions struct {
	Allow uint `json:"a"`
	Deny  uint `json:"d"`
}

type revoltMessageEditData struct {
	Content string                `json:"content,omitempty"`
	Embeds  []*revoltMessageEmbed `json:"embeds,omitempty"`
}

type revoltMessageSend struct {
	Masquerade   *revoltMessageMasquerade   `json:"masquerade,omitempty"`
	Interactions *revoltMessageInteractions `json:"interactions,omitempty"`
	Content      string                     `json:"content"`
	Attachments  []string                   `json:"attachments,omitempty"`
	Replies      []*revoltMessageReplies    `json:"replies,omitempty"`
	Embeds       []*revoltMessageEmbed      `json:"embeds,omitempty"`
}

func (m revoltMessageSend) toEdit() revoltMessageEditData {
	return revoltMessageEditData{
		Content: m.Content,
		Embeds:  m.Embeds,
	}
}

type revoltMessageReplies struct {
	ID      string `json:"id"`
	Mention bool   `json:"mention"`
}

type revoltEventMessageDelete struct {
	revoltEvent

	ID      string `json:"id"`
	Channel string `json:"channel"`
}

type revoltEventMessageUpdate struct {
	revoltEvent

	ID      string        `json:"id"`
	Channel string        `json:"channel"`
	Data    revoltMessage `json:"data"`
}

type revoltEventMessage struct {
	revoltEvent
	revoltMessage
}

type revoltEventError struct {
	revoltEvent

	Error string `json:"error"`
}

type revoltEventReady struct {
	revoltEvent

	Users    []*revoltUser         `json:"users"`
	Servers  []*revoltServer       `json:"servers"`
	Channels []*revoltChannel      `json:"channels"`
	Members  []*revoltServerMember `json:"members"`
	Emojis   []*revoltEmoji        `json:"emojis"`
}

type revoltServer struct {
	Roles map[string]*revoltServerRole `json:"roles"`

	DefaultPermissions *uint                      `json:"default_permissions"`
	Icon               *revoltAttachment          `json:"icon"`
	Banner             *revoltAttachment          `json:"banner"`
	Flags              *uint                      `json:"flags"`
	NSFW               *bool                      `json:"nsfw"`
	Analytics          *bool                      `json:"analytics"`
	Discoverable       *bool                      `json:"discoverable"`
	SystemMessages     revoltServerSystemMessages `json:"system_messages"`

	ID          string                  `json:"_id"`
	Owner       string                  `json:"owner"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Channels    []string                `json:"channels"`
	Categories  []*revoltServerCategory `json:"categories"`
}

type revoltEventBulk struct {
	revoltEvent

	V []json.RawMessage `json:"v"`
}

type revoltEvent struct {
	Type string `json:"type"`
}

type revoltEmoji struct {
	ID        string             `json:"_id"`
	Parent    *revoltEmojiParent `json:"parent"`
	CreatorID string             `json:"creator_id"`
	Name      string             `json:"name"`
	Animated  bool               `json:"animated"`
	NSFW      bool               `json:"nsfw"`
}

type revoltEmojiParent struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type revoltServerRole struct {
	Name        string             `json:"name"`
	Permissions *revoltPermissions `json:"permissions"`
	Colour      string             `json:"colour"`
	Hoist       bool               `json:"hoist"`
	Rank        int                `json:"rank"`
}

type revoltServerCategory struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Channels []string `json:"channels"`
}

type revoltServerSystemMessages struct {
	UserJoined string `json:"user_joined,omitempty"`
	UserLeft   string `json:"user_left,omitempty"`
	UserKicked string `json:"user_kicked,omitempty"`
	UserBanned string `json:"user_banned,omitempty"`
}

type revoltUploadResponse struct {
	ID string `json:"id"`
}

type revoltChannelMessageBulkDeleteData struct {
	IDs []string `json:"ids"`
}
