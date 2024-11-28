import type { ClientOptions } from '@db/postgres';
import {
	type command,
	type command_arguments,
	default_commands,
} from './commands/mod.ts';
import type { create_plugin, plugin } from './plugins.ts';
import { bridge_data } from './bridge/data.ts';
import { handle_message } from './bridge/msg.ts';
import { run_command } from './commands/run.ts';
import { handle_command_message } from './commands/run.ts';
import type { message } from './messages.ts';
import { bridge_command } from './bridge/cmd.ts';

/** configuration options for lightning */
export interface config {
	/** database options */
	postgres_options: ClientOptions;
	/** a list of plugins */
	// deno-lint-ignore no-explicit-any
	plugins?: create_plugin<any>[];
	/** the prefix used for commands */
	cmd_prefix: string;
}

/** an instance of lightning */
export class lightning {
	/** bridge data handling */
	data: bridge_data;
	/** the commands registered */
	commands: Map<string, command> = new Map(default_commands);
	/** the config used */
	config: config;
	/** the plugins loaded */
	plugins: Map<string, plugin<unknown>>;

	/** setup an instance with the given config and bridge data */
	constructor(bridge_data: bridge_data, config: config) {
		this.data = bridge_data;
		this.config = config;
		this.commands.set('bridge', bridge_command);
		this.plugins = new Map<string, plugin<unknown>>();

		for (const p of this.config.plugins || []) {
			if (p.support.some((v) => ['0.7.3'].includes(v))) {
				const plugin = new p.type(this, p.config);
				this.plugins.set(plugin.name, plugin);
				this._handle_events(plugin);
			}
		}
	}

	private async _handle_events(plugin: plugin<unknown>) {
		for await (const { name, value } of plugin) {
			await new Promise((res) => setTimeout(res, 150));

			if (sessionStorage.getItem(`${value[0].plugin}-${value[0].id}`)) continue;

			if (name === 'run_command') {
				run_command({
					...(value[0] as Omit<
						command_arguments,
						'lightning'
					>),
					lightning: this,
				});

				continue;
			}

			if (name === 'create_message') {
				handle_command_message(value[0] as message, this);
			}

			handle_message(
				this,
				value[0] as message,
				name.split('_')[0] as 'create',
			);
		}
	}

	/** create a new instance of lightning */
	static async create(config: config): Promise<lightning> {
		const data = await bridge_data.create(config.postgres_options);

		return new lightning(data, config);
	}
}
