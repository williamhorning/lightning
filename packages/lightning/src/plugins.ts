import { EventEmitter } from '@denosaurs/event';
import type { lightning } from './lightning.ts';
import type {
	deleted_message,
	message,
	message_options,
	process_result,
} from './messages.ts';
import type { run_command_options } from './commands/mod.ts';

/** the way to make a plugin */
export interface create_plugin<
	plugin_type extends plugin<plugin_type['config']>,
> {
	/** the actual constructor of the plugin */
	type: new (l: lightning, config: plugin_type['config']) => plugin_type;
	/** the configuration options for the plugin */
	config: plugin_type['config'];
	/** version(s) the plugin supports */
	support: string[];
}

/** the events emitted by a plugin */
export type plugin_events = {
	/** when a message is created */
	create_message: [message];
	/** when a message is edited */
	edit_message: [message];
	/** when a message is deleted */
	delete_message: [deleted_message];
	/** when a command is run */
	run_command: [Omit<run_command_options, 'lightning'>];
};

/** a plugin for lightning */
export abstract class plugin<cfg> extends EventEmitter<plugin_events> {
	/** access the instance of lightning you're connected to */
	lightning: lightning;
	/** access the config passed to you by lightning */
	config: cfg;
	/** the name of your plugin */
	abstract name: string;

	/** create a new plugin instance */
	static new<T extends plugin<T['config']>>(
		this: new (l: lightning, config: T['config']) => T,
		config: T['config'],
	): create_plugin<T> {
		return { type: this, config, support: ['0.7.3'] };
	}

	constructor(l: lightning, config: cfg) {
		super();
		this.lightning = l;
		this.config = config;
	}

	/** this should return the data you need to send to the channel given */
	abstract create_bridge(channel: string): Promise<unknown> | unknown;

	/** processes a message and returns information */
	abstract process_message(opts: message_options): Promise<process_result>;
}
