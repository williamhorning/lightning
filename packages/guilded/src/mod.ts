import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	log_error,
	plugin,
} from '@jersey/lightning';
import { handle_error } from './errors.ts';
import { get_guilded_message, get_lightning_message } from './messages.ts';
import { type Client, createClient } from '@jersey/guildapi';
import type { ServerChannel } from '@jersey/guilded-api-types';

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

		this.bot = createClient(c.token);

		this.bot.socket.on('ready', (user) => {
			console.log(`[guilded] ready as ${user.name}`);
		});

		this.bot.socket.on('ChatMessageCreated', async ({ d: { message } }) => {
			const msg = await get_lightning_message(message, this.bot);
			if (msg) this.emit('create_message', msg);
		});

		this.bot.socket.on('ChatMessageUpdated', async ({ d: { message } }) => {
			const msg = await get_lightning_message(message, this.bot);
			if (msg) this.emit('edit_message', msg);
		});

		this.bot.socket.on('ChatMessageDeleted', ({ d: { message } }) => {
			this.emit('delete_message', {
				channel: message.channelId,
				id: message.id,
				plugin: 'bolt-guilded',
				timestamp: Temporal.Instant.from(message.deletedAt),
			});
		});

		this.bot.socket.connect();
	}

	async setup_channel(channel: string): Promise<unknown> {
		try {
			const { channel: { serverId } } = await this.bot.request(
				'get',
				`/channels/${channel}`,
				undefined,
			) as { channel: ServerChannel };

			const { webhook } = await this.bot.request(
				'post',
				`/servers/${serverId}/webhooks`,
				{
					channelId: channel,
					name: 'Lightning Bridges',
				},
			);

			if (!webhook.id || !webhook.token) {
				log_error('failed to create webhook: missing id or token', {
					extra: { webhook: webhook },
				});
			}

			return { id: webhook.id, token: webhook.token };
		} catch (e) {
			return handle_error(e, channel);
		}
	}

	async create_message(opts: create_opts): Promise<string[]> {
		try {
			const data = opts.channel.data as { id: string; token: string };
			const res = await (await fetch(
				`https://media.guilded.gg/webhooks/${data.id}/${data.token}`,
				{
					method: 'POST',
					body: JSON.stringify(
						await get_guilded_message(
							opts.msg,
							opts.channel.id,
							this.bot,
							opts.settings.allow_everyone,
						),
					),
				},
			)).json();

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
			await this.bot.request(
				'delete',
				// @ts-expect-error: guilded's openapi spec is really bad
				`/channels/${opts.channel}/messages/${opts.edit_ids[0]}`,
				undefined,
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, opts.channel.id, true);
		}
	}
}
