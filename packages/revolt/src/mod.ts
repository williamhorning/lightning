import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	plugin,
} from '@jersey/lightning';
import { type Client, createClient } from '@jersey/rvapi';
import type { Message } from '@jersey/revolt-api-types';
import { handle_error } from './error_handler.ts';
import { check_permissions } from './permissions.ts';
import { to_revolt } from './to_revolt.ts';
import { to_lightning } from './to_lightning.ts';

/** the config for the revolt plugin */
export interface revolt_config {
	/** the token for the revolt bot */
	token: string;
	/** the user id for the bot */
	user_id: string;
}

/** the plugin to use */
export class revolt_plugin extends plugin<revolt_config> {
	bot: Client;
	name = 'bolt-revolt';

	constructor(l: lightning, config: revolt_config) {
		super(l, config);
		this.bot = createClient(config);
		this.setup_events();
	}

	private setup_events() {
		this.bot.bonfire.on('Ready', (ready) => {
			console.log(`[bolt-revolt] ready in ${ready.channels.length} channels`);
			console.log(`[bolt-revolt] and ${ready.servers.length} servers`);
		});

		this.bot.bonfire.on('Message', async (msg) => {
			if (!msg.channel || msg.channel === 'undefined') return;

			this.emit('create_message', await to_lightning(this.bot, msg));
		});

		this.bot.bonfire.on('MessageUpdate', async (msg) => {
			if (!msg.channel || msg.channel === 'undefined') return;

			this.emit(
				'edit_message',
				await to_lightning(this.bot, msg.data as Message),
			);
		});

		this.bot.bonfire.on('MessageDelete', (msg) => {
			this.emit('delete_message', {
				channel: msg.channel,
				id: msg.id,
				timestamp: Temporal.Now.instant(),
				plugin: 'bolt-revolt',
			});
		});

		this.bot.bonfire.on('socket_close', (info) => {
			console.warn('[bolt-revolt] socket closed', info);
			this.bot = createClient(this.config);
			this.setup_events();
		});
	}

	async setup_channel(channel: string) {
		return await check_permissions(channel, this.bot, this.config.user_id);
	}

	async create_message(opts: create_opts) {
		try {
			const { _id } = (await this.bot.request(
				'post',
				`/channels/${opts.channel.id}/messages`,
				await to_revolt(this.bot, opts.msg, true),
			)) as Message;

			return [_id];
		} catch (e) {
			return handle_error(e);
		}
	}

	async edit_message(opts: edit_opts) {
		try {
			await this.bot.request(
				'patch',
				`/channels/${opts.channel.id}/messages/${opts.edit_ids[0]}`,
				await to_revolt(this.bot, opts.msg, true),
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, true);
		}
	}

	async delete_message(opts: delete_opts) {
		try {
			await this.bot.request(
				'delete',
				`/channels/${opts.channel.id}/messages/${opts.edit_ids[0]}`,
				undefined,
			);

			return opts.edit_ids;
		} catch (e) {
			return handle_error(e, true);
		}
	}
}
