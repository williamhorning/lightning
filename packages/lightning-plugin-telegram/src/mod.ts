import {
	type lightning,
	type message_options,
	plugin,
	type process_result,
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
	bot: Bot;

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
		Deno.serve({ port: this.config.plugin_port }, (req: Request) => {
			const { pathname } = new URL(req.url);
			return fetch(
				`https://api.telegram.org/file/bot${this.bot.token}/${
					pathname.replace('/telegram/', '')
				}`,
			);
		});
	}

	/** create a bridge */
	create_bridge(channel: string): string {
		return channel;
	}

	/** process a message event */
	async process_message(opts: message_options): Promise<process_result> {
		try {
			if (opts.action === 'delete') {
				for (const id of opts.edit_id) {
					await this.bot.api.deleteMessage(
						opts.channel.id,
						Number(id),
					);
				}

				return {
					id: opts.edit_id,
					channel: opts.channel,
				};
			} else if (opts.action === 'edit') {
				const content = from_lightning(opts.message)[0];

				await this.bot.api.editMessageText(
					opts.channel.id,
					Number(opts.edit_id[0]),
					content.value,
					{
						parse_mode: 'MarkdownV2',
					},
				);

				return {
					id: opts.edit_id,
					channel: opts.channel,
				};
			} else if (opts.action === 'create') {
				const content = from_lightning(opts.message);
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

				return {
					id: messages,
					channel: opts.channel,
				};
			} else {
				throw new Error('unknown action');
			}
		} catch (e) {
			// TODO(@williamhorning): improve error handling logic
			return {
				error: e as Error,
				channel: opts.channel,
			};
		}
	}
}
