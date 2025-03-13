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
	create_plugin,
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
	// deno-lint-ignore no-explicit-any
	plugins?: create_plugin<any>[];
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
			if (plugin.support.includes('0.8.0')) {
				const plugin_instance = new plugin.type(this, plugin.config);
				this.plugins.set(plugin_instance.name, plugin_instance);
				this.handle_events(plugin_instance);
			}
		}
	}

	/** event handler */
	private async handle_events(plugin: plugin<unknown>) {
		for await (const { name, value } of plugin) {
			await new Promise((res) => setTimeout(res, 150));

			if (sessionStorage.getItem(`${value[0].plugin}-${value[0].id}`)) {
				continue;
			}

			if (name === 'create_command') {
				run_command(value[0] as create_command, this);
				continue;
			}

			if (name === 'create_message') {
				execute_text_command(value[0] as message, this);
			}

			bridge_message(this, name, value[0]);
		}
	}

	/** create a new instance of lightning */
	static async create(config: config): Promise<lightning> {
		const data = await create_database(config.database);

		return new lightning(data, config);
	}
}
