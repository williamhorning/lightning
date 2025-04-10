import { REST, type RESTOptions } from '@discordjs/rest';
import { WebSocketManager } from '@discordjs/ws';
import { Client } from '@discordjs/core';
import { GatewayDispatchEvents } from 'discord-api-types';
import {
	getDeletedMessage,
	getIncomingCommand,
	getIncomingMessage,
} from './incoming.ts';
import { handle_error } from './errors.ts';
import { getOutgoingMessage } from './outgoing.ts';
import {
	type bridge_message_opts,
	type command,
	type deleted_message,
	type message,
	plugin,
} from '@jersey/lightning';
import { setup_commands } from './commands.ts';

/** Options to use for the Discord plugin */
export interface DiscordOptions {
	/** The token to use for the bot */
	token: string;
}

export default class DiscordPlugin extends plugin<DiscordOptions> {
	name = 'bolt-discord';
	support = ['0.8.0-alpha.1'];
	private client: Client;

	constructor(config: DiscordOptions) {
		super(config);

		const rest = new REST({
			makeRequest: fetch as RESTOptions['makeRequest'],
			userAgentAppendix: `${navigator.userAgent} lightningplugindiscord/0.8.0`,
			version: '10',
		}).setToken(config.token);

		const gateway = new WebSocketManager({
			token: config.token,
			intents: 0 | 16813601,
			rest,
		});

		this.client = new Client({ gateway, rest });
		this.setup_events();
		gateway.connect();
	}

	private setup_events() {
		this.client.on(GatewayDispatchEvents.MessageCreate, async (data) => {
			const msg = await getIncomingMessage(data);
			if (msg) this.emit('create_message', msg);
		}).on(GatewayDispatchEvents.MessageDelete, ({ data }) => {
			this.emit('delete_message', getDeletedMessage(data));
		}).on(GatewayDispatchEvents.MessageDeleteBulk, ({ data }) => {
			for (const id of data.ids) {
				this.emit('delete_message', getDeletedMessage({ id, ...data }));
			}
		}).on(GatewayDispatchEvents.MessageUpdate, async (data) => {
			const msg = await getIncomingMessage(data);
			if (msg) this.emit('edit_message', msg);
		}).on(GatewayDispatchEvents.InteractionCreate, (data) => {
			const cmd = getIncomingCommand(data);
			if (cmd) this.emit('create_command', cmd);
		}).on(GatewayDispatchEvents.Ready, async ({ data }) => {
			console.log(
				`[discord] ready as ${data.user.username}#${data.user.discriminator} in ${data.guilds.length}`,
				`[discord] invite me at https://discord.com/oauth2/authorize?client_id=${
					(await this.client.api.applications.getCurrent()).id
				}&scope=bot&permissions=8`,
			);
		});
	}

	override async set_commands(commands: command[]): Promise<void> {
		await setup_commands(this.client.api, commands);
	}

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

	async send_message(
		message: message,
		data?: bridge_message_opts,
	): Promise<string[]> {
		try {
			const msg = await getOutgoingMessage(
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
				await getOutgoingMessage(
					message,
					this.client.api,
					data !== undefined,
					data?.settings?.allow_everyone ?? false,
				),
			);
			return data.edit_ids;
		} catch (e) {
			return handle_error(e, data.channel.id, true);
		}
	}

	async delete_messages(msgs: deleted_message[]): Promise<string[]> {
		const successful = [];

		for (const msg of msgs) {
			try {
				await this.client.api.channels.deleteMessage(
					msg.channel_id,
					msg.message_id,
				);
				successful.push(msg.message_id);
			} catch (e) {
				// if this doesn't throw, it's fine
				handle_error(e, msg.channel_id, true);
			}
		}

		return successful;
	}
}
