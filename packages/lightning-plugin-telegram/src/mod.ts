import {
	type create_message_opts,
	type delete_message_opts,
	type edit_message_opts,
	type lightning,
	plugin,
} from '@jersey/lightning';
import { Bot } from 'grammy';
import { from_lightning, from_telegram } from './messages.ts';

/** options for the telegram plugin */
export type telegram_config = {
	/** the token for the bot */
	bot_token: string;
	/** the port the plugins proxy will run on */
	plugin_port: number;
	/** the publically accessible url of the plugin */
	plugin_url: string;
};

/** the plugin to use */
export class telegram_plugin extends plugin<telegram_config> {
	name = 'bolt-telegram';
	private bot: Bot;

	constructor(l: lightning, cfg: telegram_config) {
		super(l, cfg);
		this.bot = new Bot(cfg.bot_token);
		this.bot.on('message', async (ctx) => {
			const msg = await from_telegram(ctx, cfg);
			if (!msg) return;
			this.emit('create_message', msg);
		});
		this.bot.on('edited_message', async (ctx) => {
			const msg = await from_telegram(ctx, cfg);
			if (!msg) return;
			this.emit('edit_message', msg);
		});
		// turns out it's impossible to deal with messages being deleted due to tdlib/telegram-bot-api#286
		this.serve_proxy();
		this.bot.start();
	}

	private serve_proxy() {
		Deno.serve({
			port: this.config.plugin_port,
			onListen: (addr) => {
				console.log(
					`bolt-telegram: file proxy listening on http://localhost:${addr.port}`,
					`bolt-telegram: also available at: ${this.config.plugin_url}`,
				);
			},
		}, (req: Request) => {
			const { pathname } = new URL(req.url);
			return fetch(
				`https://api.telegram.org/file/bot${this.bot.token}/${
					pathname.replace('/telegram/', '')
				}`,
			);
		});
	}

	/** create a bridge */
	setup_channel(channel: string) {
		return channel;
	}

	async create_message(opts: create_message_opts) {
		const content = from_lightning(opts.msg);
		const messages = [];

		for (const msg of content) {
			const result = await this.bot.api[msg.function](
				opts.channel.id,
				msg.value,
				{
					reply_parameters: opts.reply_id
						? {
							message_id: Number(opts.reply_id),
						}
						: undefined,
					parse_mode: 'MarkdownV2',
				},
			);

			messages.push(String(result.message_id));
		}

		return messages;
	}

	async edit_message(opts: edit_message_opts) {
		const content = from_lightning(opts.msg)[0];

		await this.bot.api.editMessageText(
			opts.channel.id,
			Number(opts.edit_ids[0]),
			content.value,
			{
				parse_mode: 'MarkdownV2',
			},
		);

		return opts.edit_ids;
	}

	async delete_message(opts: delete_message_opts) {
		for (const id of opts.edit_ids) {
			await this.bot.api.deleteMessage(
				opts.channel.id,
				Number(id),
			);
		}

		return opts.edit_ids;
	}
}
