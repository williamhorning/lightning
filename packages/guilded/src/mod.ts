import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	log_error,
	plugin,
} from '@jersey/lightning';
import { Client, WebhookClient } from 'guilded.js';
import { error_handler } from './error_handler.ts';
import { convert_msg } from './guilded.ts';
import { guilded_to_message } from './guilded_message/mod.ts';

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
		this.setup_events();
		this.bot.login();
	}

	private setup_events() {
		this.bot.on('ready', () => {
			console.log(`[bolt-guilded] logged in as ${this.bot.user?.name}`);
		});
		this.bot.on('messageCreated', async (message) => {
			const msg = await guilded_to_message(message, this.bot);
			if (msg) this.emit('create_message', msg);
		});
		this.bot.on('messageUpdated', async (message) => {
			const msg = await guilded_to_message(message, this.bot);
			if (msg) this.emit('edit_message', msg);
		});
		this.bot.on('messageDeleted', (del) => {
			this.emit('delete_message', {
				channel: del.channelId,
				id: del.id,
				plugin: 'bolt-guilded',
				timestamp: Temporal.Instant.from(del.deletedAt),
			});
		});
		this.bot.ws.emitter.on('exit', () => {
			this.bot.ws.connect();
		});
	}

	async setup_channel(channel: string): Promise<unknown> {
		try {
			// TODO(jersey): it may be worth it to add server/guild id to the message type...
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
			return error_handler(e, channel, 'creating webhook');
		}
	}

	async create_message(opts: create_opts): Promise<string[]> {
		try {
			const webhook = new WebhookClient(
				opts.channel.data as { id: string; token: string },
			);

			const res = await webhook.send(
				await convert_msg(
					opts.msg,
					opts.channel.id,
					this.bot,
					opts.settings.allow_everyone,
				),
			);

			return [res.id];
		} catch (e) {
			return error_handler(e, opts.channel.id, 'creating message');
		}
	}

	// deno-lint-ignore require-await
	async edit_message(opts: edit_opts): Promise<string[]> {
		// guilded does not support editing messages
		return opts.edit_ids;
	}

	async delete_message(opts: delete_opts): Promise<string[]> {
		try {
			await this.bot.messages.delete(opts.channel.id, opts.edit_ids[0]);

			return opts.edit_ids;
		} catch (e) {
			return error_handler(e, opts.channel.id, 'deleting message');
		}
	}
}
