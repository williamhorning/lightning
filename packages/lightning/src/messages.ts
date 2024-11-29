import type { bridge_channel } from './bridge/data.ts';

/**
 * creates a message that can be sent using lightning
 * @param text the text of the message (can be markdown)
 */
export function create_message(text: string): message {
	const data = {
		author: {
			username: 'lightning',
			profile: 'https://williamhorning.eu.org/assets/lightning.png',
			rawname: 'lightning',
			id: 'lightning',
		},
		content: text,
		channel: '',
		id: '',
		reply: async () => {},
		timestamp: Temporal.Now.instant(),
		plugin: 'lightning',
	};
	return data;
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
	/** file size */
	size: number;
}

/** a representation of a message that has been deleted */
export interface deleted_message {
	/** the message's id */
	id: string;
	/** the channel the message was sent in */
	channel: string;
	/** the plugin that recieved the message */
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

/** a message recieved by a plugin */
export interface message extends deleted_message {
	/** the attachments sent with the message */
	attachments?: attachment[];
	/** the author of the message */
	author: {
		/** the nickname of the author */
		username: string;
		/** the author's username */
		rawname: string;
		/** a url pointing to the authors profile picture */
		profile?: string;
		/** a url pointing to the authors banner */
		banner?: string;
		/** the author's id */
		id: string;
		/** the color of an author */
		color?: string;
	};
	/** message content (can be markdown) */
	content?: string;
	/** discord-style embeds */
	embeds?: embed[];
	/** a function to reply to a message */
	reply: (message: message, optional?: unknown) => Promise<void>;
	/** the id of the message replied to */
	reply_id?: string;
}

/** a message to be bridged */
export interface create_message_opts {
	msg: message,
	channel: bridge_channel,
	reply_id?: string,
}

/** a message to be edited */
export interface edit_message_opts {
	msg: message,
	channel: bridge_channel,
	reply_id?: string,
	edit_ids: string[],
}

/** a message to be deleted */
export interface delete_message_opts {
	msg: deleted_message,
	channel: bridge_channel,
	edit_ids: string[],
}