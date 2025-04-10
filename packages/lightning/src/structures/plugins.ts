import { EventEmitter } from '@denosaurs/event';
import type { bridge_message_opts } from './bridge.ts';
import type { deleted_message, message } from './messages.ts';
import type { command, create_command } from './commands.ts';

/** the events emitted by a plugin */
export type plugin_events = {
	/** when a message is created */
	create_message: [message];
	/** when a message is edited */
	edit_message: [message];
	/** when a message is deleted */
	delete_message: [deleted_message];
	/** when a command is run */
	create_command: [create_command];
};

/** a plugin for lightning */
export interface plugin<cfg> {
	/** set commands on the platform, if available */
	set_commands?(commands: command[]): Promise<void> | void;
}

/** a plugin for lightning */
export abstract class plugin<cfg> extends EventEmitter<plugin_events> {
	/** access the config passed to you by lightning */
	config: cfg;
	/** the name of your plugin */
	abstract name: string;
	/** the versions supported by your plugin */
	abstract support: string[];
	/** initialize a plugin with the given lightning instance and config */
	constructor(config: cfg) {
		super();
		this.config = config;
	}
	/** setup a channel to be used in a bridge */
	abstract setup_channel(channel: string): Promise<unknown> | unknown;
	/** send a message to a given channel */
	abstract send_message(
		message: message,
		opts?: bridge_message_opts,
	): Promise<string[]>;
	/** edit a message in a given channel */
	abstract edit_message(
		message: message,
		opts?: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]>;
	/** delete messages in a given channel */
	abstract delete_messages(
		messages: deleted_message[],
	): Promise<string[]>;
}
