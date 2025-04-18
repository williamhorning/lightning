import { EventEmitter } from '@denosaurs/event';
import type {
	command,
	command_opts,
	create_command,
} from './structures/commands.ts';
import { LightningError, log_error } from './structures/errors.ts';
import { create_message, type message } from './structures/messages.ts';
import type { events, plugin, plugin_module } from './structures/plugins.ts';

export interface core_config {
	prefix?: string;
	plugins: {
		module: plugin_module;
		config: Record<string, unknown>;
	}[];
}

export class core extends EventEmitter<events> {
	private commands = new Map<string, command>([
		['help', {
			name: 'help',
			description: 'get help with the bot',
			execute: () =>
				'check out [the docs](https://williamhorning.eu.org/lightning/) for help.',
		}],
		['ping', {
			name: 'ping',
			description: 'check if the bot is alive',
			execute: ({ timestamp }: command_opts) =>
				`Pong! 🏓 ${
					Temporal.Now.instant().since(timestamp).round('millisecond')
						.total('milliseconds')
				}ms`,
		}],
		['version', {
			name: 'version',
			description: 'get the bots version',
			execute: () => 'hello from v0.8.0!',
		}],
	]);
	private plugins = new Map<string, plugin>();
	private handled = new Set<string>();
	private prefix: string;

	constructor(cfg: core_config) {
		super();
		this.prefix = cfg.prefix || '!';

		for (const { module, config } of cfg.plugins) {
			if (!module.default || !module.parse_config) {
				log_error({ ...module }, {
					message: `one or more of you plugins isn't actually a plugin!`,
					without_cause: true,
				});
			}

			const instance = new module.default(module.parse_config(config));

			this.plugins.set(instance.name, instance);
			this.handle_events(instance);
		}
	}

	set_handled(plugin: string, message_id: string): void {
		this.handled.add(`${plugin}-${message_id}`);
	}

	set_command(opts: command): void {
		this.commands.set(opts.name, opts);
	}

	get_plugin(name: string): plugin | undefined {
		return this.plugins.get(name);
	}

	private async handle_events(plugin: plugin): Promise<void> {
		for await (const { name, value } of plugin) {
			await new Promise((res) => setTimeout(res, 150));

			if (this.handled.has(`${value[0].plugin}-${value[0].message_id}`)) {
				this.handled.delete(`${value[0].plugin}-${value[0].message_id}`);
				continue;
			}

			if (name === 'create_command') {
				this.handle_command(value[0] as create_command, plugin);
			}

			if (name === 'create_message') {
				const msg = value[0] as message;

				if (msg.content?.startsWith(this.prefix)) {
					const [command, ...rest] = msg.content.replace(this.prefix, '').split(
						' ',
					);

					this.handle_command({
						...msg,
						args: {},
						command,
						prefix: this.prefix,
						reply: async (message: message) => {
							await plugin.create_message({
								...message,
								channel_id: msg.channel_id,
								reply_id: msg.message_id,
							});
						},
						rest,
					}, plugin);
				}
			}

			this.emit(name, ...value);
		}
	}

	private async handle_command(
		opts: create_command,
		plugin: plugin,
	): Promise<void> {
		let command = this.commands.get(opts.command) ?? this.commands.get('help')!;
		const subcommand_name = opts.subcommand ?? opts.rest?.shift();

		if (command.subcommands && subcommand_name) {
			const subcommand = command.subcommands.find((i) =>
				i.name === subcommand_name
			);

			if (subcommand) command = subcommand;
		}

		for (const arg of (command.arguments || [])) {
			if (!opts.args[arg.name]) {
				opts.args[arg.name] = opts.rest?.shift();
			}

			if (!opts.args[arg.name]) {
				return opts.reply(
					create_message(
						`Please provide the \`${arg.name}\` argument. Try using the \`${opts.prefix}help\` command.`,
					),
				);
			}
		}

		let resp: string | LightningError;

		try {
			resp = await command.execute({ ...opts, plugin });
		} catch (e) {
			resp = new LightningError(e, {
				message: 'An error occurred while executing the command',
				extra: { command: command.name },
			});
		}

		try {
			await opts.reply(
				resp instanceof LightningError ? resp.msg : create_message(resp),
			);
		} catch (e) {
			new LightningError(e, {
				message: 'An error occurred while sending the command response',
				extra: { command: command.name },
			});
		}
	}
}
