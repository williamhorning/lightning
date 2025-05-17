import {
	type bridge_message_opts,
	type config_schema,
	type deleted_message,
	type message,
	plugin,
} from '@lightning/lightning';
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

/** the config schema for the plugin */
export const schema: config_schema = {
	name: 'bolt-telegram',
	keys: {
		token: { type: 'string', required: true },
		proxy_port: { type: 'number', required: true },
		proxy_url: { type: 'string', required: true },
	},
};

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

		const handler = async ({ url }: { url: string }) =>
			await fetch(
				`https://api.telegram.org/file/bot${opts.token}/${
					url.replace('/telegram', '/')
				}`,
			);

		if ('Deno' in globalThis) {
			Deno.serve({ port: opts.proxy_port }, handler);
		} else if ('Bun' in globalThis) {
			// @ts-ignore: Bun.serve is not typed
			Bun.serve({
				fetch: handler,
				port: opts.proxy_port,
			});
		} else if ('process' in globalThis) {
			// deno-lint-ignore no-process-global
			process.getBuiltinModule('node:http').createServer(async (req, res) => {
				const resp = await handler(req as { url: string });
				res.writeHead(resp.status, Array.from(resp.headers.entries()));
				res.write(new Uint8Array(await resp.arrayBuffer()));
				res.end();
			});
		} else {
			throw new Error('Unsupported environment for file proxy!');
		}

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
