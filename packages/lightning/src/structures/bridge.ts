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

/** possible settings for a bridge */
export interface bridge_settings {
	/** `@everyone/@here/@room` */
	allow_everyone: boolean;
}

/** list of settings for a bridge */
export const bridge_settings_list = [
	'allow_everyone',
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

/** options for a message to be bridged */
export interface bridge_message_opts {
	/** the channel to use */
	channel: bridge_channel;
	/** ids of messages to edit, if any */
	edit_ids?: string[];
	/** the settings to use */
	settings: bridge_settings;
}
