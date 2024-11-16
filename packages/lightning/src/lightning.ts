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

/** configuration options for lightning */
export interface config {
	/** database options */
	postgres_options: ClientOptions;
	/** a list of plugins */
	// deno-lint-ignore no-explicit-any
	plugins?: create_plugin<plugin<any>>[];
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
	/** set of processed messages */
	private processed: Set<`${string}-${string}`> = new Set();
	/** the plugins loaded */
	plugins: Map<string, plugin<unknown>>;

	/** setup an instance with the given config and bridge data */
	constructor(bridge_data: bridge_data, config: config) {
		this.data = bridge_data;
		this.config = config;
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
		for await (const event of plugin) {
			await new Promise((res) => setTimeout(res, 150));

			const id = `${event.value[0].plugin}-${event.value[0].id}` as const;

			if (!this.processed.has(id)) {
				this.processed.add(id);

				if (event.name === 'run_command') {
					run_command({
						...(event.value[0] as Omit<
							command_arguments,
							'lightning'
						>),
						lightning: this,
					});

					continue;
				}

				if (event.name === 'create_message') {
					handle_command_message(event.value[0] as message, this);
				}

				handle_message(
					this,
					event.value[0] as message,
					event.name.split('_')[0] as 'create',
				);
			} else {
				this.processed.delete(id);
			}
		}
	}

	/** create a new instance of lightning */
	static async create(config: config): Promise<lightning> {
		const data = await bridge_data.create(config.postgres_options);

		return new lightning(data, config);
	}
}
