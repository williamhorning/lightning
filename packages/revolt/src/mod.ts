import type { Message } from '@jersey/revolt-api-types';
import { Bonfire, type Client, createClient } from '@jersey/rvapi';
import {
	type bridge_message_opts,
	type config_schema,
	type deleted_message,
	type message,
	plugin,
} from '@lightning/lightning';
import { fetch_message } from './cache.ts';
import { handle_error } from './errors.ts';
import { get_incoming } from './incoming.ts';
import { get_outgoing } from './outgoing.ts';
import { check_permissions } from './permissions.ts';

/** the config for the revolt bot */
export interface revolt_config {
	/** the token for the revolt bot */
	token: string;
	/** the user id for the bot */
	user_id: string;
}

/** the config schema for the revolt plugin */
export const schema: config_schema = {
	name: 'bolt-revolt',
	keys: {
		token: { type: 'string', required: true },
		user_id: { type: 'string', required: true },
	},
};

/** revolt support for lightning */
export default class revolt extends plugin {
	name = 'bolt-revolt';
	private client: Client;
	private user_id: string;

	/** setup revolt using these options */
	constructor(opts: revolt_config) {
		super();
		this.client = createClient({ token: opts.token });
		this.user_id = opts.user_id;
		this.setup_events(opts);
	}

	private setup_events(opts: revolt_config) {
		this.client.bonfire.on('Message', async (data) => {
			const msg = await get_incoming(data, this.client);
			if (msg) this.emit('create_message', msg);
		}).on('MessageDelete', (data) => {
			this.emit('delete_message', {
				channel_id: data.channel,
				message_id: data.id,
				plugin: 'bolt-revolt',
				timestamp: Temporal.Now.instant(),
			});
		}).on('MessageUpdate', async (data) => {
			try {
				const msg = await get_incoming({
					...await fetch_message(this.client, data.channel, data.id),
					...data,
				}, this.client);

				if (msg) this.emit('edit_message', msg);
			} catch {
				return;
			}
		}).on('Ready', (data) => {
			console.log(
				`[revolt] ready as ${
					data.users.find((i) => i._id === this.user_id)?.username
				} in ${data.servers.length} servers`,
				`\n[revolt] invite me at https://app.revolt.chat/bot/${this.user_id}`,
			);
		}).on('socket_close', () => {
			this.client.bonfire = new Bonfire({
				token: opts.token,
				url: 'wss://ws.revolt.chat',
			});
			this.setup_events(opts);
		});
	}

	/** ensure masquerading will work in that channel */
	async setup_channel(channel_id: string): Promise<unknown> {
		return await check_permissions(channel_id, this.user_id, this.client);
	}

	/** send a message to a channel */
	async create_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		try {
			return [
				(await this.client.request(
					'post',
					`/channels/${message.channel_id}/messages`,
					await get_outgoing(this.client, message, data !== undefined),
				) as Message)._id,
			];
		} catch (e) {
			return handle_error(e);
		}
	}

	/** edit a message in a channel */
	async edit_message(
		message: message,
		data: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		try {
			return [
				(await this.client.request(
					'patch',
					`/channels/${message.channel_id}/messages/${data.edit_ids[0]}`,
					await get_outgoing(this.client, message, true),
				) as Message)._id,
			];
		} catch (e) {
			return handle_error(e, true);
		}
	}

	/** delete messages in a channel */
	async delete_messages(messages: deleted_message[]): Promise<string[]> {
		return await Promise.all(
			messages.map(async (msg) => {
				try {
					await this.client.request(
						'delete',
						`/channels/${msg.channel_id}/messages/${msg.message_id}`,
						undefined,
					);
					return msg.message_id;
				} catch (e) {
					handle_error(e, true);
					return msg.message_id;
				}
			}),
		);
	}
}
