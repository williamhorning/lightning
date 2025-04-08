import { bridge_message } from './bridge.ts';
import { default_commands } from './commands/default.ts';
import { execute_text_command, run_command } from './commands/runners.ts';
import {
	type bridge_data,
	create_database,
	type database_config,
} from './database/mod.ts';
import type {
	command,
	create_command,
	message,
	plugin,
} from './structures/mod.ts';

/** configuration options for lightning */
export interface config {
	/** error URL */
	error_url?: string;
	/** database options */
	database: database_config;
	/** a list of plugins */
	plugins?: plugin<unknown>[];
	/** the prefix used for commands */
	prefix: string;
}

/** an instance of lightning */
export class lightning {
	/** bridge data handling */
	data: bridge_data;
	/** the commands registered */
	commands: Map<string, command> = default_commands;
	/** the config used */
	config: config;
	/** the plugins loaded */
	plugins: Map<string, plugin<unknown>>;

	/** setup an instance with the given config and bridge data */
	constructor(bridge_data: bridge_data, config: config) {
		this.data = bridge_data;
		this.config = config;
		this.plugins = new Map<string, plugin<unknown>>();

		for (const plugin of this.config.plugins || []) {
			if (plugin.support.includes('0.8.0-alpha.1')) {
				this.plugins.set(plugin.name, plugin);
				if (plugin.set_commands) {
					plugin.set_commands(this.commands.values().toArray());
				}
				this.handle_events(plugin);
			}
		}
	}

	/** event handler */
	private async handle_events(plugin: plugin<unknown>) {
		for await (const { name, value } of plugin) {
			await new Promise((res) => setTimeout(res, 150));

			if (sessionStorage.getItem(`${value[0].plugin}-${value[0].message_id}`)) {
				continue;
			}

			switch (name) {
				case 'create_command':
					run_command(value[0] as create_command, this);
					break;
				case 'create_message':
					execute_text_command(value[0] as message, this);
					bridge_message(this, name, value[0]);
					break;
				case 'edit_message':
					bridge_message(this, name, value[0]);
					break;
				case 'delete_message':
					bridge_message(this, name, value[0]);
					break;
			}
		}
	}

	/** create a new instance of lightning */
	static async create(config: config): Promise<lightning> {
		const data = await create_database(config.database);

		return new lightning(data, config);
	}
}
