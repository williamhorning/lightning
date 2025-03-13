import { Client } from '@discordjs/core';
import { REST } from '@discordjs/rest';
import { WebSocketManager } from '@discordjs/ws';
import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	plugin,
} from '@jersey/lightning';
import { set_slash_commands } from './commands.ts';
import { handle_error } from './errors.ts';
import { setup_events } from './events.ts';
import { get_discord_message } from './messages.ts';

/** configuration for the discord plugin */
export interface discord_config {
	/** the discord bot token */
	token: string;
	/** whether to enable slash commands */
	slash_commands: boolean;
	/** discord application id */
	application_id: string;
}

/** the plugin to use */
export class discord_plugin extends plugin<discord_config> {
	name = 'bolt-discord';
	private api: Client['api'];
	private client: Client;

	constructor(l: lightning, config: discord_config) {
		super(l, config);

		// @ts-ignore the Undici type for fetch differs from Deno, but it works the same
		const rest = new REST({ version: '10', makeRequest: fetch }).setToken(
			config.token,
		);

		const gateway = new WebSocketManager({
			token: config.token,
			intents: 0 | 33281,
			rest,
		});

		// @ts-ignore Deno doesn't properly handle the AsyncEventEmitter class types, but this works
		this.client = new Client({ rest, gateway });
		this.api = this.client.api;

		set_slash_commands(this.api, config, l);
		setup_events(this.client, this.emit);
		gateway.connect();
	}

	async setup_channel(channel: string): Promise<unknown> {
		try {
			const { id, token } = await this.api.channels.createWebhook(
				channel,
				{ name: 'lightning bridge' },
			);

			return { id, token };
		} catch (e) {
			return handle_error(e, channel);
		}
	}

	async create_message(opts: create_opts): Promise<string[]> {
		const data = opts.channel.data as { id: string; token: string };

		try {
			const res = await this.api.webhooks.execute(
				data.id,
				data.token,
				await get_discord_message(
					opts.msg,
					{ api: this.api, channel: opts.channel.id, reply_id: opts.reply_id },
					opts.settings.allow_everyone,
				),
			);

			return [res.id];
		} catch (e) {
			return handle_error(e, opts.channel.id);
		}
	}

	async edit_message(opts: edit_opts): Promise<string[]> {
		const data = opts.channel.data as { id: string; token: string };

		try {
			await this.api.webhooks.editMessage(
				data.id,
				data.token,
				opts.edit_ids[0],
				await get_discord_message(
					opts.msg,
					{ api: this.api, channel: opts.channel.id, reply_id: opts.reply_id },
					opts.settings.allow_everyone,
				),
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, opts.channel.id, true);
		}
	}

	async delete_message(opts: delete_opts): Promise<string[]> {
		const data = opts.channel.data as { id: string; token: string };

		try {
			await this.api.webhooks.deleteMessage(
				data.id,
				data.token,
				opts.edit_ids[0],
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, opts.channel.id, true);
		}
	}
}
