import {
	type lightning,
	type message_options,
	plugin,
	type process_result,
} from '@jersey/lightning';
import { Client, WebhookClient } from 'guilded.js';
import { convert_msg, create_webhook } from './guilded.ts';
import { tocore } from './messages.ts';

/** options for the guilded plugin */
export interface guilded_config {
	/** the token to use */
	token: string;
}

/** the plugin to use */
export class guilded_plugin extends plugin<guilded_config> {
	bot: Client;
	name = 'bolt-guilded';

	constructor(l: lightning, c: guilded_config) {
		super(l, c);
		const h = {
			headers: {
				'x-guilded-bot-api-use-official-markdown': 'true',
			},
		};
		this.bot = new Client({ token: c.token, rest: h, ws: h });
		this.setup_events();
		this.bot.login();
	}

	private setup_events() {
		this.bot.on('messageCreated', async (message) => {
			const msg = await tocore(message, this);
			if (msg) this.emit('create_message', msg);
		});
		this.bot.on('messageUpdated', async (message) => {
			const msg = await tocore(message, this);
			if (msg) this.emit('edit_message', msg);
		});
		this.bot.on('messageDeleted', (del) => {
			this.emit('delete_message', {
				channel: del.channelId,
				id: del.id,
				plugin: 'bolt-guilded',
				timestamp: Temporal.Instant.from(del.deletedAt),
			});
		});
		this.bot.ws.emitter.on('exit', () => {
			this.bot.ws.connect();
		});
	}

	/** create a bridge in this channel */
	create_bridge(channel: string): Promise<{ id: string; token: string }> {
		return create_webhook(this.bot, channel, this.config.token);
	}

	async process_message(opts: message_options): Promise<process_result> {
		try {
			if (opts.action === 'create') {
				try {
					const { id } = await new WebhookClient(
						opts.channel.data as { token: string; id: string },
					).send(
						await convert_msg(opts.message, opts.channel.id, this),
					);

					return {
						id: [id],
						channel: opts.channel,
					};
				} catch (e) {
					if (
						(e as { response: { status: number } }).response
							.status === 404
					) {
						return {
							channel: opts.channel,
							disable: true,
							error: new Error('webhook not found!'),
						};
					} else if (
						(e as { response: { status: number } }).response
							.status === 403
					) {
						return {
							channel: opts.channel,
							disable: true,
							error: new Error('no permission to send messages!'),
						};
					} else {
						throw e;
					}
				}
			} else if (opts.action === 'delete') {
				const msg = await this.bot.messages.fetch(
					opts.channel.id,
					opts.edit_id[0],
				);

				await msg.delete();

				return {
					channel: opts.channel,
					id: opts.edit_id,
				};
			} else {
				return {
					channel: opts.channel,
					id: opts.edit_id,
				};
			}
		} catch (e) {
			// TODO(@williamhorning): improve error handling
			return {
				channel: opts.channel,
				error: e as Error,
				disable: false,
			};
		}
	}
}
