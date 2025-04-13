import { Client, GatewayDispatchEvents } from '@discordjs/core';
import { REST, type RESTOptions } from '@discordjs/rest';
import { WebSocketManager } from '@discordjs/ws';
import {
	type bridge_message_opts,
	type command,
	type deleted_message,
	log_error,
	type message,
	plugin,
} from '@jersey/lightning';
import { setup_commands } from './commands.ts';
import { handle_error } from './errors.ts';
import {
	get_deleted_message,
	get_incoming_command,
	get_incoming_message,
} from './incoming.ts';
import { get_outgoing_message } from './outgoing.ts';

/** options for the discord bot */
export type discord_config = {
	/** the token for your bot */
	token: string;
};

/** check if something is actually a config object, return if it is */
export function parse_config(v: unknown): discord_config {
	if (typeof v !== 'object' || v === null) {
		log_error("discord config isn't an object!", { without_cause: true });
	}
	if (!('token' in v) || typeof v.token !== 'string') {
		log_error("discord token isn't a string", { without_cause: true });
	}
	return { token: v.token };
}

/** discord support for lightning */
export default class discord extends plugin {
	name = 'bolt-discord';
	private client: Client;

	/** create the plugin */
	constructor(cfg: discord_config) {
		super();

		const rest = new REST({
			makeRequest: fetch as RESTOptions['makeRequest'],
			version: '10',
		}).setToken(cfg.token);

		const gateway = new WebSocketManager({
			token: cfg.token,
			intents: 0 | 16813601,
			rest,
		});

		this.client = new Client({ gateway, rest });
		this.setup_events();
		gateway.connect();
	}

	private setup_events() {
		this.client.on(GatewayDispatchEvents.MessageCreate, async (data) => {
			const msg = await get_incoming_message(data);
			if (msg) this.emit('create_message', msg);
		}).on(GatewayDispatchEvents.MessageDelete, ({ data }) => {
			this.emit('delete_message', get_deleted_message(data));
		}).on(GatewayDispatchEvents.MessageDeleteBulk, ({ data }) => {
			for (const id of data.ids) {
				this.emit('delete_message', get_deleted_message({ id, ...data }));
			}
		}).on(GatewayDispatchEvents.MessageUpdate, async (data) => {
			const msg = await get_incoming_message(data);
			if (msg) this.emit('edit_message', msg);
		}).on(GatewayDispatchEvents.InteractionCreate, (data) => {
			const cmd = get_incoming_command(data);
			if (cmd) this.emit('create_command', cmd);
		}).on(GatewayDispatchEvents.Ready, async ({ data }) => {
			console.log(
				`[discord] ready as ${data.user.username}#${data.user.discriminator} in ${data.guilds.length} servers`,
				`\n[discord] invite me at https://discord.com/oauth2/authorize?client_id=${
					(await this.client.api.applications.getCurrent()).id
				}&scope=bot&permissions=8`,
			);
		});
	}

	/** setup slash commands */
	override async set_commands(commands: command[]): Promise<void> {
		await setup_commands(this.client.api, commands);
	}

	/** create a webhook */
	async setup_channel(channelID: string): Promise<unknown> {
		try {
			const { id, token } = await this.client.api.channels.createWebhook(
				channelID,
				{ name: 'lightning bridge' },
			);

			return { id, token };
		} catch (e) {
			return handle_error(e, channelID);
		}
	}

	/** send a message using the bot itself or a webhook */
	async create_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		try {
			const msg = await get_outgoing_message(
				message,
				this.client.api,
				data !== undefined,
				data?.settings?.allow_everyone ?? false,
			);

			if (data) {
				const webhook = data.channel.data as { id: string; token: string };
				return [
					(await this.client.api.webhooks.execute(
						webhook.id,
						webhook.token,
						msg,
					)).id,
				];
			} else {
				return [
					(await this.client.api.channels.createMessage(
						message.channel_id,
						msg,
					))
						.id,
				];
			}
		} catch (e) {
			return handle_error(e, message.channel_id);
		}
	}

	/** edut a message sent by webhook */
	async edit_message(
		message: message,
		data: bridge_message_opts & { edit_ids: string[] },
	): Promise<string[]> {
		try {
			const webhook = data.channel.data as { id: string; token: string };

			await this.client.api.webhooks.editMessage(
				webhook.id,
				webhook.token,
				data.edit_ids[0],
				await get_outgoing_message(
					message,
					this.client.api,
					true,
					data?.settings?.allow_everyone ?? false,
				),
			);
			return data.edit_ids;
		} catch (e) {
			return handle_error(e, data.channel.id, true);
		}
	}

	/** delete messages */
	async delete_messages(msgs: deleted_message[]): Promise<string[]> {
		return await Promise.all(
			msgs.map(async (msg) => {
				try {
					await this.client.api.channels.deleteMessage(
						msg.channel_id,
						msg.message_id,
					);
					return msg.message_id;
				} catch (e) {
					// if this doesn't throw, it's fine
					handle_error(e, msg.channel_id, true);
					return msg.message_id;
				}
			}),
		);
	}
}
