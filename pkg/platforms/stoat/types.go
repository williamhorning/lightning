package stoat

import "time"

type stChannel struct {
	Name            string                     `json:"name,omitempty"`
	Owner           *string                    `json:"owner,omitempty"`
	Permissions     *stPermission              `json:"permissions,omitempty"`
	Server          *string                    `json:"server,omitempty"`
	DefaultPerms    *stOverrideField           `json:"default_permissions,omitempty"`
	RolePermissions map[string]stOverrideField `json:"role_permissions,omitempty"`
	ChannelType     string                     `json:"channel_type"`
	ID              string                     `json:"_id"` //nolint:tagliatelle
	Recipients      []string                   `json:"recipients,omitempty"`
}

type stDataEditMessage struct {
	Content string            `json:"content,omitempty"`
	Embeds  []stSendableEmbed `json:"embeds,omitempty"`
}

type stDataMessageSend struct {
	Masquerade  *stMasquerade     `json:"masquerade,omitempty"`
	Content     string            `json:"content,omitempty"`
	Attachments []string          `json:"attachments,omitempty"`
	Replies     []stReplyIntent   `json:"replies,omitempty"`
	Embeds      []stSendableEmbed `json:"embeds,omitempty"`
}

type stEmbed struct {
	URL         string   `json:"url,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Image       *stMedia `json:"image,omitempty"`
	Video       *stMedia `json:"video,omitempty"`
	IconURL     *string  `json:"icon_url,omitempty"`
	Colour      string   `json:"colour,omitempty"`
}

type stEmoji struct {
	ID     string        `json:"_id"` //nolint:tagliatelle
	Parent stEmojiParent `json:"parent"`
	Name   string        `json:"name"`
}

type stEmojiParent struct {
	ID string `json:"id,omitempty"`
}

type stFile struct {
	ID       string `json:"_id"` //nolint:tagliatelle
	Tag      string `json:"tag"`
	Filename string `json:"filename"`
	Size     int    `json:"size"`
}

type stMedia struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type stMasquerade struct {
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Colour string `json:"colour,omitempty"`
}

type stMember struct {
	ID       stMemberCompositeKey `json:"_id"` //nolint:tagliatelle
	Timeout  time.Time            `json:"timeout"`
	Avatar   *stFile              `json:"avatar"`
	Nickname *string              `json:"nickname"`
	Roles    []string             `json:"roles,omitempty"`
}

type stMemberCompositeKey struct {
	Server string `json:"server"`
	User   string `json:"user"`
}

type stMessage struct {
	Edited      time.Time    `json:"edited"`
	Masquerade  stMasquerade `json:"masquerade"`
	Content     string       `json:"content"`
	Author      string       `json:"author"`
	Channel     string       `json:"channel"`
	ID          string       `json:"_id"` //nolint:tagliatelle
	Attachments []stFile     `json:"attachments,omitempty"`
	Embeds      []stEmbed    `json:"embeds,omitempty"`
	Replies     []string     `json:"replies,omitempty"`
}

type stMessageDeleteEvent struct {
	ID      string `json:"id"`
	Channel string `json:"channel"`
}

type stMessageUpdateEvent struct {
	Data stMessage `json:"data"`
}

type stOverrideField struct {
	Allow stPermission `json:"a"`
	Deny  stPermission `json:"d"`
}

type stReplyIntent struct {
	ID              string `json:"id"`
	Mention         bool   `json:"mention"`
	FailIfNotExists bool   `json:"fail_if_not_exists"`
}

type stReadyEvent struct {
	Users    []stUser    `json:"users,omitempty"`
	Servers  []stServer  `json:"servers,omitempty"`
	Channels []stChannel `json:"channels,omitempty"`
	Members  []stMember  `json:"members,omitempty"`
	Emojis   []stEmoji   `json:"emojis,omitempty"`
}

type stServer struct {
	Roles map[string]struct {
		Permissions stOverrideField `json:"permissions"`
	} `json:"roles"`
	ID                 string       `json:"_id"` //nolint:tagliatelle
	Owner              string       `json:"owner"`
	DefaultPermissions stPermission `json:"default_permissions"`
}

type stSendableEmbed struct {
	IconURL     string `json:"icon_url,omitempty"`
	URL         string `json:"url,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Media       string `json:"media,omitempty"`
	Colour      string `json:"colour,omitempty"`
}

type stUser struct {
	Avatar       *stFile `json:"avatar"`
	Relationship string  `json:"relationship"`
	ID           string  `json:"_id"` //nolint:tagliatelle
	Username     string  `json:"username"`
}
