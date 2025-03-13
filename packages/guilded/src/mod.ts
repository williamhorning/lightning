import { WebhookClient } from '@guildedjs/api';
import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	log_error,
	plugin,
} from '@jersey/lightning';
import { Client } from 'guilded.js';
import { handle_error } from './errors.ts';
import { setup_events } from './events.ts';
import { get_guilded_message } from './messages.ts';

/** options for the guilded plugin */
export interface guilded_config {
	/** the token to use */
	token: string;
}

/** the plugin to use */
export class guilded_plugin extends plugin<guilded_config> {
	name = 'bolt-guilded';
	bot: Client;

	constructor(l: lightning, c: guilded_config) {
		super(l, c);

		const opts = {
			headers: {
				'x-guilded-bot-api-use-official-markdown': 'true',
			},
		};

		this.bot = new Client({ token: c.token, ws: opts, rest: opts });

		setup_events(this.bot, this.emit);
		this.bot.login();
	}

	async setup_channel(channel: string): Promise<unknown> {
		try {
			const { serverId } = await this.bot.channels.fetch(channel);
			const webhook = await this.bot.webhooks.create(serverId, {
				channelId: channel,
				name: 'Lightning Bridges',
			});
			if (!webhook.id || !webhook.token) {
				log_error('failed to create webhook: missing id or token', {
					extra: { webhook: webhook.raw },
				});
			}

			return { id: webhook.id, token: webhook.token };
		} catch (e) {
			return handle_error(e, channel);
		}
	}

	async create_message(opts: create_opts): Promise<string[]> {
		try {
			const webhook = new WebhookClient(
				opts.channel.data as { id: string; token: string },
			);

			const res = await webhook.send(
				await get_guilded_message(
					opts.msg,
					opts.channel.id,
					this.bot,
					opts.settings.allow_everyone,
				),
			);

			return [res.id];
		} catch (e) {
			return handle_error(e, opts.channel.id);
		}
	}

	// guilded doesn't support editing messages
	// deno-lint-ignore require-await
	async edit_message(opts: edit_opts): Promise<string[]> {
		return opts.edit_ids;
	}

	async delete_message(opts: delete_opts): Promise<string[]> {
		try {
			await this.bot.messages.delete(opts.channel.id, opts.edit_ids[0]);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, opts.channel.id, true);
		}
	}
}
