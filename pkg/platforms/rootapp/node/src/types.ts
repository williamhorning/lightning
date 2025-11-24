export interface Attachment {
	URL: string;
	Name: string;
	Size: bigint;
}

// BaseMessage basic metadata
export interface BaseMessage {
	Time: Date;
	EventID: string;
	ChannelID: string;
}

// ChannelDisabled represents whether to disable a channel
export interface ChannelDisabled {
	read: boolean;
	write: boolean;
}

// DeletedMessage = BaseMessage
export type DeletedMessage = BaseMessage;

// EditedMessage information
export interface EditedMessage {
	Edited: Date;
	Message: Message | null;
}

// EmbedAuthor
export interface EmbedAuthor {
	URL?: string;
	IconURL?: string;
	Name?: string;
}

// EmbedField
export interface EmbedField {
	Name: string;
	Value: string;
	Inline: boolean;
}

// EmbedFooter
export interface EmbedFooter {
	IconURL?: string;
	Text: string;
}

// Media for embed images/video
export interface Media {
	URL: string;
	Height: number;
	Width: number;
}

// Embed (Discord-style embed)
export interface Embed {
	Author?: EmbedAuthor | null;
	Footer?: EmbedFooter | null;
	Image?: Media | null;
	Thumbnail?: Media | null;
	Video?: Media | null;
	Timestamp?: string;
	Title?: string;
	URL?: string;
	Description?: string;
	Fields?: EmbedField[];
	Color?: number;
}

// Emoji
export interface Emoji {
	URL: string;
	ID: string;
	Name: string;
}

// MessageAuthor
export interface MessageAuthor {
	ID: string;
	Nickname: string;
	Username: string;
	ProfilePicture?: string;
	Color?: string;
}

// Message
export type Message = BaseMessage & {
	Author: MessageAuthor | null;
	Content: string;
	Attachments: Attachment[];
	Embeds: Embed[];
	Emoji: Emoji[];
	RepliedTo: string[];
};

// SendOptions
export interface SendOptions {
	ChannelData: Record<string, string>;
	AllowEveryonePings: boolean;
}
