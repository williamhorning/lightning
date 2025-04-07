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

/** the way to make a plugin */
export interface create_plugin<
	plugin_type extends plugin<plugin_type['config']>,
> {
	/** the actual constructor of the plugin */
	type: new (config: plugin_type['config']) => plugin_type;
	/** the configuration options for the plugin */
	config: plugin_type['config'];
	/** version(s) the plugin supports */
	support: string[];
}

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
	/** create a new plugin instance */
	static new<T extends plugin<T['config']>>(
		this: new (config: T['config']) => T,
		config: T['config'],
	): create_plugin<T> {
		return { type: this, config, support: ['0.8.0-alpha.1'] };
	}
	/** initialize a plugin with the given lightning instance and config */
	constructor(config: cfg) {
		super();
		this.config = config;
	}

	/** log something to the console */
	log(type: 'info' | 'warn' | 'error', ...args: unknown[]) {
		for (const arg of args) {
			console[type](`[${this.name}]`, arg);
		}
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
