import { type Client, createClient } from '@jersey/guildapi';
import {
	type bridge_message_opts,
	type deleted_message,
	type message,
	plugin,
} from '@jersey/lightning';
import { getIncomingMessage } from './incoming.ts';
import { handle_error } from './errors.ts';
import type { ServerChannel } from '@jersey/guilded-api-types';
import { getOutgoingMessage } from './outgoing.ts';

/** options for the guilded plugin */
export interface GuildedOptions {
	/** the token to use */
	token: string;
}

export default class GuildedPlugin extends plugin<GuildedOptions> {
	name = 'bolt-guilded';
	private client: Client;

	constructor(opts: GuildedOptions) {
		super(opts);
		this.client = createClient(opts.token);
		this.setup_events();
		this.client.socket.connect();
	}

	private setup_events() {
		this.client.socket.on('ChatMessageCreated', async (data) => {
			const msg = await getIncomingMessage(data.d.message, this.client);
			if (msg) this.emit('create_message', msg);
		}).on('ChatMessageDeleted', ({ d }) => {
			this.emit('delete_message', {
				channel_id: d.message.channelId,
				message_id: d.message.id,
				plugin: 'bolt-guilded',
				timestamp: Temporal.Instant.from(d.deletedAt),
			});
		}).on('ChatMessageUpdated', async (data) => {
			const msg = await getIncomingMessage(data.d.message, this.client);
			if (msg) this.emit('edit_message', msg);
		}).on('ready', (data) => {
			this.log('info', `Ready as ${data.name} (${data.id})`);
		});
	}

	async setup_channel(channelID: string): Promise<unknown> {
		try {
			const { channel: { serverId } } = await this.client.request(
				'get',
				`/channels/${channelID}`,
				undefined,
			) as { channel: ServerChannel };

			const { webhook } = await this.client.request(
				'post',
				`/servers/${serverId}/webhooks`,
				{
					channelId: channelID,
					name: 'Lightning Bridges',
				},
			);

			if (!webhook.id || !webhook.token) {
				throw 'failed to create webhook: missing id or token';
			}

			return { id: webhook.id, token: webhook.token };
		} catch (e) {
			return handle_error(e, channelID);
		}
	}

	async send_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		try {
			const msg = await getOutgoingMessage(
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

	async edit_message(
		message: message,
		data?: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		// guilded webhooks don't support editing
		if (data) return data.edit_ids;

		try {
			const resp = await this.client.request(
				'put',
				`/channels/${message.channel_id}/messages/${message.message_id}`,
				await getOutgoingMessage(message, this.client, false),
			);

			return [resp.message.id];
		} catch (e) {
			return handle_error(e, message.channel_id, true);
		}
	}

	async delete_messages(messages: deleted_message[]): Promise<string[]> {
		const successful = [];

		for (const msg of messages) {
			try {
				await this.client.request(
					'delete', // @ts-expect-error: this is typed wrong
					`/channels/${opts.channel}/messages/${msg.message_id[0]}`,
					undefined,
				);
				successful.push(msg.message_id);
			} catch (e) {
				handle_error(e, msg.channel_id, true);
			}
		}

		return successful;
	}
}
