import { type Client, createClient } from '@jersey/rvapi';
import { getIncomingMessage } from './incoming.ts';
import type { Message as APIMessage } from '@jersey/revolt-api-types';
import { handle_error } from './errors.ts';
import { getOutgoingMessage } from './outgoing.ts';
import { fetchMessage } from './cache.ts';
import { check_permissions } from './permissions.ts';
import {
	type bridge_message_opts,
	type deleted_message,
	type message,
	plugin,
} from '@jersey/lightning';

export interface RevoltOptions {
	token: string;
	user_id: string;
}

export default class RevoltPlugin extends plugin<RevoltOptions> {
	name = 'bolt-revolt';
	support = ['0.8.0-alpha.1'];
	private client: Client;

	constructor(opts: RevoltOptions) {
		super(opts);
		this.client = createClient({ token: opts.token });
		this.setupEvents();
	}

	private setupEvents() {
		this.client.bonfire.on('Message', async (data) => {
			const msg = await getIncomingMessage(data, this.client);
			if (msg) this.emit('create_message', msg);
		}).on('MessageDelete', (data) => {
			this.emit('delete_message', {
				channel_id: data.channel,
				message_id: data.id,
				plugin: this.name,
				timestamp: Temporal.Now.instant(),
			});
		}).on('MessageUpdate', async (data) => {
			let oldMessage: APIMessage;

			try {
				oldMessage = await fetchMessage(this.client, data.channel, data.id);
			} catch {
				return;
			}

			const msg = await getIncomingMessage({
				...oldMessage,
				...data,
			}, this.client);

			if (msg) this.emit('edit_message', msg);
		}).on('Ready', (data) => {
			console.log(
				`[revolt] ready as ${
					data.users.find((i) => i._id === this.config.user_id)?.username
				} in ${data.servers.length}`,
				`[revolt] invite me at https://app.revolt.chat/bot/${this.config.user_id}`
			);
		});
	}

	async setup_channel(channelID: string): Promise<unknown> {
		return await check_permissions(channelID, this.config.user_id, this.client);
	}

	async send_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		try {
			return [
				(await this.client.request(
					'post',
					`/channels/${message.channel_id}/messages`,
					await getOutgoingMessage(this.client, message, data !== undefined),
				) as APIMessage)._id,
			];
		} catch (e) {
			return handle_error(e);
		}
	}

	async edit_message(
		message: message,
		data?: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		try {
			return [
				(await this.client.request(
					'patch',
					`/channels/${message.channel_id}/messages/${
						data?.edit_ids[0] ?? message.message_id
					}`,
					await getOutgoingMessage(this.client, message, data !== undefined),
				) as APIMessage)._id,
			];
		} catch (e) {
			return handle_error(e);
		}
	}

	async delete_messages(messages: deleted_message[]): Promise<string[]> {
		const successful = [];

		for (const msg of messages) {
			try {
				await this.client.request(
					'delete',
					`/channels/${msg.channel_id}/messages/${msg.message_id}`,
					undefined,
				);
				successful.push(msg.message_id);
			} catch (e) {
				handle_error(e);
			}
		}

		return successful;
	}
}
