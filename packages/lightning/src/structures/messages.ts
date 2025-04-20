/**
 * creates a message that can be sent using lightning
 * @param text the text of the message (can be markdown)
 */
export function create_message(text: string): message {
	return {
		author: {
			username: 'lightning',
			profile:
				'https://williamhorning.eu.org/assets/lightning/logo_monocolor_dark.svg',
			rawname: 'lightning',
			id: 'lightning',
		},
		content: text,
		channel_id: '',
		message_id: '',
		timestamp: Temporal.Now.instant(),
		plugin: 'lightning',
	};
}

/** attachments within a message */
export interface attachment {
	/** alt text for images */
	alt?: string;
	/** a URL pointing to the file */
	file: string;
	/** the file's name */
	name?: string;
	/** whether or not the file has a spoiler */
	spoiler?: boolean;
	/** file size in MiB */
	size: number;
}

/** a representation of a message that has been deleted */
export interface deleted_message {
	/** the message's id */
	message_id: string;
	/** the channel the message was sent in */
	channel_id: string;
	/** the plugin that received the message */
	plugin: string;
	/** the time the message was sent/edited as a temporal instant */
	timestamp: Temporal.Instant;
}

/** a discord-style embed */
export interface embed {
	/** the author of the embed */
	author?: {
		/** the name of the author */
		name: string;
		/** the url of the author */
		url?: string;
		/** the icon of the author */
		icon_url?: string;
	};
	/** the color of the embed */
	color?: number;
	/** the text in an embed */
	description?: string;
	/** fields within the embed */
	fields?: {
		/** the name of the field */
		name: string;
		/** the value of the field */
		value: string;
		/** whether or not the field is inline */
		inline?: boolean;
	}[];
	/** a footer shown in the embed */
	footer?: {
		/** the footer text */
		text: string;
		/** the icon of the footer */
		icon_url?: string;
	};
	/** an image shown in the embed */
	image?: media;
	/** a thumbnail shown in the embed */
	thumbnail?: media;
	/** the time (in epoch ms) shown in the embed */
	timestamp?: number;
	/** the title of the embed */
	title?: string;
	/** a site linked to by the embed */
	url?: string;
	/** a video inside of the embed */
	video?: media;
}

/** media inside of an embed */
export interface media {
	/** the height of the media */
	height?: number;
	/** the url of the media */
	url: string;
	/** the width of the media */
	width?: number;
}

/** a message received by a plugin */
export interface message extends deleted_message {
	/** the attachments sent with the message */
	attachments?: attachment[];
	/** the author of the message */
	author: message_author;
	/** message content (can be markdown) */
	content?: string;
	/** discord-style embeds */
	embeds?: embed[];
	/** the id of the message replied to */
	reply_id?: string;
}

/** an author of a message */
export interface message_author {
	/** the nickname of the author */
	username: string;
	/** the author's username */
	rawname: string;
	/** a url pointing to the authors profile picture */
	profile?: string;
	/** the author's id */
	id: string;
	/** the color of an author */
	color?: string;
}
