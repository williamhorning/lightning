import {
	type bridge_message_opts,
	type deleted_message,
	type message,
	plugin,
} from '@jersey/lightning';
import { Bot } from 'grammy';
import { getIncomingMessage } from './incoming.ts';
import { getOutgoingMessage } from './outgoing.ts';

/** options for the telegram plugin */
export interface telegram_config {
	/** the token for the bot */
	token: string;
	/** the port the plugins proxy will run on */
	proxy_port: number;
	/** the publically accessible url of the plugin */
	proxy_url: string;
}

export default class TelegramPlugin extends plugin<telegram_config> {
	name = 'bolt-telegram';
	support = ['0.8.0-alpha.1'];
	private bot: Bot;

	constructor(opts: telegram_config) {
		super(opts);
		this.bot = new Bot(opts.token);
		this.bot.start();

		this.bot.on(['message', 'edited_message'], async (ctx) => {
			const msg = await getIncomingMessage(ctx, this.config.proxy_url);
			if (msg) this.emit('create_message', msg);
		});

		Deno.serve({
			port: this.config.proxy_port,
			onListen: ({ port }) => {
				console.log(
					`[telegram] proxy available at localhost:${port} or ${this.config.proxy_url}`,
				);
			},
		}, (req: Request) => {
			const { pathname } = new URL(req.url);
			return fetch(
				`https://api.telegram.org/file/bot${this.config.token}/${
					pathname.replace('/telegram/', '')
				}`,
			);
		});
	}

	setup_channel(channel: string): unknown {
		return channel;
	}

	async send_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		const messages = [];

		for (const msg of getOutgoingMessage(message, data !== undefined)) {
			const result = await this.bot.api[msg.function](
				message.channel_id,
				msg.value,
				{
					reply_parameters: message.reply_id
						? {
							message_id: Number(message.reply_id),
						}
						: undefined,
					parse_mode: 'MarkdownV2',
				},
			);

			messages.push(String(result.message_id));
		}

		return messages;
	}

	async edit_message(
		message: message,
		opts: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		await this.bot.api.editMessageText(
			opts.channel.id,
			Number(opts.edit_ids[0]),
			getOutgoingMessage(message, true)[0].value,
			{
				parse_mode: 'MarkdownV2',
			},
		);

		return opts.edit_ids;
	}

	async delete_messages(messages: deleted_message[]): Promise<string[]> {
		const successful: string[] = [];

		for (const msg of messages) {
			await this.bot.api.deleteMessage(msg.channel_id, Number(msg.message_id));
			successful.push(msg.message_id);
		}

		return successful;
	}
}
