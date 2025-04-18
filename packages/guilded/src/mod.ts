import { type Client, createClient } from '@jersey/guildapi';
import type { ServerChannel } from '@jersey/guilded-api-types';
import {
	type bridge_message_opts,
	type deleted_message,
	log_error,
	type message,
	plugin,
} from '@jersey/lightning';
import { handle_error } from './errors.ts';
import { get_incoming } from './incoming.ts';
import { get_outgoing } from './outgoing.ts';

/** options for the guilded bot */
export interface guilded_config {
	/** the token to use */
	token: string;
}

/** check if something is actually a config object, return if it is */
export function parse_config(v: unknown): guilded_config {
	if (typeof v !== 'object' || v === null) {
		log_error("guilded config isn't an object!", { without_cause: true });
	}
	if (!('token' in v) || typeof v.token !== 'string') {
		log_error("guilded token isn't a string", { without_cause: true });
	}
	return { token: v.token };
}

/** guilded support for lightning */
export default class guilded extends plugin {
	name = 'bolt-guilded';
	private client: Client;

	constructor(opts: guilded_config) {
		super();
		this.client = createClient(opts.token);
		this.setup_events();
		this.client.socket.connect();
	}

	private setup_events() {
		this.client.socket.on('ChatMessageCreated', async (data) => {
			const msg = await get_incoming(data.d.message, this.client);
			if (msg) this.emit('create_message', msg);
		}).on('ChatMessageDeleted', ({ d }) => {
			this.emit('delete_message', {
				channel_id: d.message.channelId,
				message_id: d.message.id,
				plugin: 'bolt-guilded',
				timestamp: Temporal.Instant.from(d.deletedAt),
			});
		}).on('ChatMessageUpdated', async (data) => {
			const msg = await get_incoming(data.d.message, this.client);
			if (msg) this.emit('edit_message', msg);
		}).on('ready', (data) => {
			console.log(`[guilded] ready as ${data.name} (${data.id})`);
		});
	}

	/** create a webhook in a channel */
	async setup_channel(channel_id: string): Promise<unknown> {
		try {
			const { channel: { serverId } } = await this.client.request(
				'get',
				`/channels/${channel_id}`,
				undefined,
			) as { channel: ServerChannel };

			const { webhook } = await this.client.request(
				'post',
				`/servers/${serverId}/webhooks`,
				{
					channelId: channel_id,
					name: 'Lightning Bridges',
				},
			);

			if (!webhook.id || !webhook.token) {
				throw 'failed to create webhook: missing id or token';
			}

			return { id: webhook.id, token: webhook.token };
		} catch (e) {
			return handle_error(e, channel_id);
		}
	}

	/** send a message either as the bot or using a webhook */
	async create_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		try {
			const msg = await get_outgoing(
				message,
				this.client,
				data?.settings?.allow_everyone ?? false,
			);

			if (data) {
				const webhook = data.channel.data as { id: string; token: string };

				const res = await (await fetch(
					`https://media.guilded.gg/webhooks/${webhook.id}/${webhook.token}`,
					{
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify(msg),
					},
				)).json();

				return [res.id];
			} else {
				const resp = await this.client.request(
					'post',
					`/channels/${message.channel_id}/messages`,
					msg,
				);

				return [resp.message.id];
			}
		} catch (e) {
			return handle_error(e, message.channel_id);
		}
	}

	/** edit stub function */
	// deno-lint-ignore require-await
	async edit_message(
		_message: message,
		data: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		return data.edit_ids;
	}

	/** delete messages from guilded */
	async delete_messages(messages: deleted_message[]): Promise<string[]> {
		return await Promise.all(messages.map(async (msg) => {
			try {
				await this.client.request(
					'delete', // @ts-expect-error: this is typed wrong
					`/channels/${msg.channel_id}/messages/${msg.message_id}`,
					undefined,
				);
				return msg.message_id;
			} catch (e) {
				handle_error(e, msg.channel_id, true);
				return msg.message_id;
			}
		}));
	}
}
