import {
	type bridge_message_opts,
	type deleted_message,
	log_error,
	type message,
	plugin,
} from '@lightning/lightning';
import { Application } from '@oak/oak/application';
import { proxy } from '@oak/oak/proxy';
import { Bot } from 'grammy';
import { get_incoming } from './incoming.ts';
import { get_outgoing } from './outgoing.ts';

/** options for telegram */
export type telegram_config = {
	/** the token for the bot */
	token: string;
	/** the port the file proxy will run on */
	proxy_port: number;
	/** the publicly accessible url of the file proxy */
	proxy_url: string;
};

/** check if something is actually a config object, return if it is */
export function parse_config(v: unknown): telegram_config {
	if (typeof v !== 'object' || v === null) {
		log_error("telegram config isn't an object!", { without_cause: true });
	}
	if (!('token' in v) || typeof v.token !== 'string') {
		log_error("telegram token isn't a string", { without_cause: true });
	}
	if (!('proxy_port' in v) || typeof v.proxy_port !== 'number') {
		log_error("telegram proxy port isn't a number", { without_cause: true });
	}
	if (!('proxy_url' in v) || typeof v.proxy_url !== 'string') {
		log_error("telegram proxy url isn't a string", { without_cause: true });
	}
	return { token: v.token, proxy_port: v.proxy_port, proxy_url: v.proxy_url };
}

/** telegram support for lightning */
export default class telegram extends plugin {
	name = 'bolt-telegram';
	private bot: Bot;

	/** setup telegram and its file proxy */
	constructor(opts: telegram_config) {
		super();
		this.bot = new Bot(opts.token);
		this.bot.start();

		this.bot.on(['message', 'edited_message'], async (ctx) => {
			const msg = await get_incoming(ctx, opts.proxy_url);
			if (msg) this.emit('create_message', msg);
		});

		const app = new Application().use(
			proxy(`https://api.telegram.org/file/bot${opts.token}/`, {
				map: (path) => path.replace('/telegram/', ''),
			}),
		);

		app.listen({ port: opts.proxy_port });

		console.log(
			`[telegram] proxy available at localhost:${opts.proxy_port} or ${opts.proxy_url}`,
		);
	}

	/** stub for setup_channel */
	setup_channel(channel: string): unknown {
		return channel;
	}

	/** send a message in a channel */
	async create_message(
		message: message,
		data: bridge_message_opts,
	): Promise<string[]> {
		const messages = [];

		for (const msg of get_outgoing(message, data !== undefined)) {
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

	/** edit a message in a channel */
	async edit_message(
		message: message,
		opts: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		await this.bot.api.editMessageText(
			opts.channel.id,
			Number(opts.edit_ids[0]),
			get_outgoing(message, true)[0].value,
			{
				parse_mode: 'MarkdownV2',
			},
		);

		return opts.edit_ids;
	}

	/** delete messages in a channel */
	async delete_messages(messages: deleted_message[]): Promise<string[]> {
		return await Promise.all(
			messages.map(async (msg) => {
				await this.bot.api.deleteMessage(
					msg.channel_id,
					Number(msg.message_id),
				);
				return msg.message_id;
			}),
		);
	}
}
