import { EventEmitter } from '@denosaurs/event';
import type { bridge_message_opts } from './bridge.ts';
import type { command, create_command } from './commands.ts';
import type { deleted_message, message } from './messages.ts';
import type { config_schema } from './validate.ts';

/** the events emitted by core/plugins */
export type events = {
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
export interface plugin {
	/** setup user-facing commands, if available */
	set_commands?(commands: command[]): Promise<void> | void;
}

/** a plugin for lightning */
export abstract class plugin extends EventEmitter<events> {
	/** the name of your plugin */
	abstract name: string;
	/** setup a channel to be used in a bridge */
	abstract setup_channel(channel: string): Promise<unknown> | unknown;
	/** send a message to a given channel */
	abstract create_message(
		message: message,
		opts?: bridge_message_opts,
	): Promise<string[]>;
	/** edit a message in a given channel */
	abstract edit_message(
		message: message,
		opts: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]>;
	/** delete messages in a given channel */
	abstract delete_messages(
		messages: deleted_message[],
	): Promise<string[]>;
}

/** the type core uses to load a module */
export interface plugin_module {
	/** the plugin constructor */
	default?: { new (cfg: unknown): plugin };
	/** the config to validate use */
	schema?: config_schema;
}
