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

/** the options given to plugins when a message needs to be sent */
export interface create_message_opts {
	/** the action to take */
	action: 'create';
	/** the channel to send the message to */
	channel: bridge_channel;
	/** the message to send */
	message: message;
	/** the id of the message to reply to */
	reply_id?: string;
}

/** the options given to plugins when a message needs to be edited */
export interface edit_message_opts {
	/** the action to take */
	action: 'edit';
	/** the channel to send the message to */
	channel: bridge_channel;
	/** the message to send */
	message: message;
	/** the id of the message to edit */
	edit_id: string[];
	/** the id of the message to reply to */
	reply_id?: string;
}

/** the options given to plugins when a message needs to be deleted */
export interface delete_message_opts {
	/** the action to take */
	action: 'delete';
	/** the channel to send the message to */
	channel: bridge_channel;
	/** the message to send */
	message: deleted_message;
	/** the id of the message to delete */
	edit_id: string[];
	/** the id of the message to reply to */
	reply_id?: string;
}

/** the options given to plugins when a message needs to be processed */
export type message_options =
	| create_message_opts
	| edit_message_opts
	| delete_message_opts;

/** successfully processed message */
export interface processed_message {
	/** whether there was an error */
	error?: undefined;
	/** the message that was processed */
	id: string[];
	/** the channel the message was sent to */
	channel: bridge_channel;
}

/** messages not processed */
export interface unprocessed_message {
	/** the channel the message was to be sent to */
	channel: bridge_channel;
	/** whether the channel should be disabled */
	disable: boolean;
	/** the error causing this */
	error: Error;
}

/** process result */
export type process_result = processed_message | unprocessed_message;
