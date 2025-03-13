import {
	type create_opts,
	type delete_opts,
	type edit_opts,
	type lightning,
	plugin,
} from '@jersey/lightning';
import { Bot } from 'grammy';
import { get_lightning_message, get_telegram_message } from './messages.ts';
import { setup_file_proxy } from './file_proxy.ts';

/** options for the telegram plugin */
export interface telegram_config {
	/** the token for the bot */
	bot_token: string;
	/** the port the plugins proxy will run on */
	proxy_port: number;
	/** the publically accessible url of the plugin */
	proxy_url: string;
}

/** the plugin to use */
export class telegram_plugin extends plugin<telegram_config> {
	name = 'bolt-telegram';
	bot: Bot;

	constructor(l: lightning, cfg: telegram_config) {
		super(l, cfg);
		this.bot = new Bot(cfg.bot_token);
		this.bot.on('message', async (ctx) => {
			const msg = await get_telegram_message(ctx, cfg);
			if (!msg) return;
			this.emit('create_message', msg);
		});
		this.bot.on('edited_message', async (ctx) => {
			const msg = await get_telegram_message(ctx, cfg);
			if (!msg) return;
			this.emit('edit_message', msg);
		});
		// turns out it's impossible to deal with messages being deleted due to tdlib/telegram-bot-api#286
		setup_file_proxy(cfg);
		this.bot.start();
	}

	/** create a bridge */
	setup_channel(channel: string): unknown {
		return channel;
	}

	async create_message(opts: create_opts): Promise<string[]> {
		const messages = [];

		for (const msg of get_lightning_message(opts.msg)) {
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

	async edit_message(opts: edit_opts): Promise<string[]> {
		await this.bot.api.editMessageText(
			opts.channel.id,
			Number(opts.edit_ids[0]),
			get_lightning_message(opts.msg)[0].value,
			{
				parse_mode: 'MarkdownV2',
			},
		);

		return opts.edit_ids;
	}

	async delete_message(opts: delete_opts): Promise<string[]> {
		for (const id of opts.edit_ids) {
			await this.bot.api.deleteMessage(
				opts.channel.id,
				Number(id),
			);
		}

		return opts.edit_ids;
	}
}
