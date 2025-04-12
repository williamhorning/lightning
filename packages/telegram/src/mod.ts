import {
	type bridge_message_opts,
	type deleted_message,
	type message,
	plugin,
} from '@jersey/lightning';
import { Bot } from 'grammy';
import { getIncomingMessage } from './incoming.ts';
import { getOutgoingMessage } from './outgoing.ts';
import { type InferOutput, number, object, string } from '@valibot/valibot';

/** Options for the Revolt plugin */
export const config = object({
	/** The token to use for the bot */
	token: string(),
	/** The port the file proxy should run on */
	proxy_port: number(),
	/** The publically accessible url of the plugin */
	proxy_url: string(),
});

export default class TelegramPlugin extends plugin {
	name = 'bolt-telegram';
	private bot: Bot;

	constructor(opts: InferOutput<typeof config>) {
		super();
		this.bot = new Bot(opts.token);
		this.bot.start();

		this.bot.on(['message', 'edited_message'], async (ctx) => {
			const msg = await getIncomingMessage(ctx, opts.proxy_url);
			if (msg) this.emit('create_message', msg);
		});

		Deno.serve({
			port: opts.proxy_port,
			onListen: ({ port }) => {
				console.log(
					`[telegram] proxy available at localhost:${port} or ${opts.proxy_url}`,
				);
			},
		}, (req: Request) => {
			const { pathname } = new URL(req.url);
			return fetch(
				`https://api.telegram.org/file/bot${opts.token}/${
					pathname.replace('/telegram/', '')
				}`,
			);
		});
	}

	setup_channel(channel: string): unknown {
		return channel;
	}

	async create_message(
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
