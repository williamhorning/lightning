import {
	type lightning,
	type message_options,
	plugin,
	type process_result,
} from '@jersey/lightning';
import type { Message } from '@jersey/revolt-api-types';
import { type Client, createClient } from '@jersey/rvapi';
import { fromrvapi, torvapi } from './messages.ts';
import { revolt_perms } from './permissions.ts';

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

	/** handle revolt events */
	private setup_events() {
		this.bot.bonfire.on('Message', async (message) => {
			if (
				message.system || !message.channel ||
				message.channel === 'undefined'
			) return;
			this.emit('create_message', await fromrvapi(this.bot, message));
		});
		this.bot.bonfire.on('MessageUpdate', async (message) => {
			if (
				message.data.system || !message.channel ||
				message.channel === 'undefined'
			) return;
			this.emit(
				'edit_message',
				await fromrvapi(this.bot, message.data as Message),
			);
		});
		this.bot.bonfire.on('MessageDelete', (message) => {
			this.emit('delete_message', {
				channel: message.channel,
				id: message.id,
				plugin: 'bolt-revolt',
				timestamp: Temporal.Now.instant(),
			});
		});
		this.bot.bonfire.on('socket_close', (info) => {
			console.warn('Revolt socket closed', info);
			this.bot = createClient(this.config);
			this.setup_events();
		});
	}

	/** create a bridge */
	async create_bridge(channel: string): Promise<string> {
		return await revolt_perms(this.bot, channel, this.config.user_id);
	}

	/** process a message */
	async process_message(opts: message_options): Promise<process_result> {
		if (opts.action === 'create') {
			try {
				const msg = await torvapi(this.bot, {
					...opts.message,
					reply_id: opts.reply_id,
				});

				const resp = (await this.bot.request(
					'post',
					`/channels/${opts.channel.id}/messages`,
					msg,
				)) as Message;

				return {
					channel: opts.channel,
					id: [resp._id],
				};
			} catch (e) {
				if (
					(e as { cause: { status: number } }).cause.status ===
						403 ||
					(e as { cause: { status: number } }).cause.status ===
						404
				) {
					return {
						channel: opts.channel,
						disable: true,
						error: e as Error,
					};
				} else {
					throw e;
				}
			}
		} else if (opts.action === 'edit') {
			await this.bot.request(
				'patch',
				`/channels/${opts.channel.id}/messages/${opts.edit_id[0]}`,
				await torvapi(this.bot, {
					...opts.message,
					reply_id: opts.reply_id,
				}),
			);

			return {
				channel: opts.channel,
				id: opts.edit_id,
			};
		} else {
			await this.bot.request(
				'delete',
				`/channels/${opts.channel.id}/messages/${opts.edit_id[0]}`,
				undefined,
			);

			return {
				channel: opts.channel,
				id: opts.edit_id,
			};
		}
	}
}
