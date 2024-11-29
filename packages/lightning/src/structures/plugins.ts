import { EventEmitter } from '@denosaurs/event';
import type { lightning } from '../lightning.ts';
import type {
	create_message_opts,
	delete_message_opts,
	edit_message_opts,
} from './bridge.ts';
import type { plugin_events } from './events.ts';

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
		return { type: this, config, support: ['0.8.0'] };
	}
	/** initialize a plugin with the given lightning instance and config */
	constructor(l: lightning, config: cfg) {
		super();
		this.lightning = l;
		this.config = config;
	}
	/** setup a channel to be used in a bridge */
	abstract setup_channel(channel: string): Promise<unknown> | unknown;
	/** send a message to a given channel */
	abstract create_message(
		opts: create_message_opts,
	): Promise<string[]>;
	/** edit a message in a given channel */
	abstract edit_message(
		opts: edit_message_opts,
	): Promise<string[]>;
	/** delete a message in a given channel */
	abstract delete_message(
		opts: delete_message_opts,
	): Promise<string[]>;
}
