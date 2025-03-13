import type { attachment, embed } from './media.ts';

/**
 * creates a message that can be sent using lightning
 * @param text the text of the message (can be markdown)
 */
export function create_message(text: string): message {
	return {
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

/** a message recieved by a plugin */
export interface message extends deleted_message {
	/** the attachments sent with the message */
	attachments?: attachment[];
	/** the author of the message */
	author: message_author;
	/** message content (can be markdown) */
	content?: string;
	/** discord-style embeds */
	embeds?: embed[];
	/** a function to reply to a message */
	reply: (message: message, optional?: unknown) => Promise<void>;
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
