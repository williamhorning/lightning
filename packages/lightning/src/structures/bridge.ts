import type { deleted_message, message } from './messages.ts';

/** representation of a bridge */
export interface bridge {
	/** ulid secret used as primary key */
	id: string;
	/** user-facing name of the bridge */
	name: string;
	/** channels in the bridge */
	channels: bridge_channel[];
	/** settings for the bridge */
	settings: bridge_settings;
}

/** a channel within a bridge */
export interface bridge_channel {
	/** from the platform */
	id: string;
	/** data needed to bridge this channel */
	data: unknown;
	/** whether the channel is disabled */
	disabled: boolean;
	/** the plugin used to bridge this channel */
	plugin: string;
}

// TODO(jersey): implement allow_everyone and use_rawname settings

/** possible settings for a bridge */
export interface bridge_settings {
	/** allow editing/deletion */
	allow_editing: boolean;
	/** @everyone/@here/@room */
	allow_everyone: boolean;
	/** rawname = username */
	use_rawname: boolean;
}

/** list of settings for a bridge */
export const bridge_settings_list = [
	'allow_editing',
	'allow_everyone',
	'use_rawname',
];

/** representation of a bridged message collection */
export interface bridge_message {
	/** original message id */
	id: string;
	/** original bridge id */
	bridge_id: string;
	/** channels in the bridge */
	channels: bridge_channel[];
	/** messages bridged */
	messages: bridged_message[];
	/** settings for the bridge */
	settings: bridge_settings;
}

/** representation of an individual bridged message */
export interface bridged_message {
	/** ids of the message */
	id: string[];
	/** the channel id sent to */
	channel: string;
	/** the plugin used */
	plugin: string;
}

/** a message to be bridged */
export interface create_opts {
	/** the actual message */
	msg: message;
	/** the channel to use */
	channel: bridge_channel;
	/** message to reply to, if any */
	reply_id?: string;
}

/** a message to be edited */
export interface edit_opts {
	/** the actual message */
	msg: message;
	/** the channel to use */
	channel: bridge_channel;
	/** message to reply to, if any */
	reply_id?: string;
	/** ids of messages to edit */
	edit_ids: string[];
}

/** a message to be deleted */
export interface delete_opts {
	/** the actual deleted message */
	msg: deleted_message;
	/** the channel to use */
	channel: bridge_channel;
	/** ids of messages to delete */
	edit_ids: string[];
}
