/** representation of a bridge */
export interface bridge {
	/** primary key */
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
	/** the channel's canonical id */
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
export interface bridge_message extends bridge {
	/** original bridge id */
	bridge_id: string;
	/** messages bridged */
	messages: bridged_message[];
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
