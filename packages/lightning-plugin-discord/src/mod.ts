import { Client } from '@discordjs/core';
import { REST } from '@discordjs/rest';
import { WebSocketManager } from '@discordjs/ws';
import {
	type lightning,
	type message_options,
	plugin,
	type process_result,
} from '@jersey/lightning';
import { GatewayDispatchEvents } from 'discord-api-types';
import { to_command, to_intent_opts } from './commands.ts';
import { to_message } from './lightning.ts';
import { process_message } from './process_message.ts';

/** options for the discord plugin */
export type discord_config = {
	/** your bot's application id */
	app_id: string;
	/** the token for your bot */
	token: string;
	/** whether or not to enable slash commands */
	slash_cmds?: boolean;
};

/** the plugin to use */
export class discord_plugin extends plugin<discord_config> {
	bot: Client;
	name = 'bolt-discord';

	/** setup the plugin */
	constructor(l: lightning, config: discord_config) {
		super(l, config);
		this.config = config;
		this.bot = this.setup_client();
		this.setup_events();
		this.setup_commands();
	}

	private setup_client() {
		const rest = new REST({
			version: '10',
			/* @ts-ignore this works */
			makeRequest: fetch,
		}).setToken(this.config.token);

		const gateway = new WebSocketManager({
			rest,
			token: this.config.token,
			intents: 0 | 33281,
		});

		gateway.connect();

		// @ts-ignore this works?
		return new Client({ rest, gateway });
	}

	private setup_events() {
		this.bot.on(GatewayDispatchEvents.MessageCreate, async (msg) => {
			this.emit('create_message', await to_message(msg.api, msg.data));
		});

		this.bot.on(GatewayDispatchEvents.MessageUpdate, async (msg) => {
			this.emit('edit_message', await to_message(msg.api, msg.data));
		});

		this.bot.on(GatewayDispatchEvents.MessageDelete, async (msg) => {
			this.emit('delete_message', await to_message(msg.api, msg.data));
		});

		this.bot.on(GatewayDispatchEvents.InteractionCreate, (interaction) => {
			const cmd = to_command(interaction, this.lightning);
			if (cmd) this.emit('run_command', cmd);
		});
	}

	private setup_commands() {
		if (!this.config.slash_cmds) return;

		this.bot.api.applicationCommands.bulkOverwriteGlobalCommands(
			this.config.app_id,
			[...this.lightning.commands.values()].map((command) => {
				return {
					name: command.name,
					type: 1,
					description: command.description || 'a command',
					options: to_intent_opts(command),
				};
			}),
		);
	}

	/** creates a webhook in the channel for a bridge */
	async create_bridge(
		channel: string,
	): Promise<{ id: string; token?: string }> {
		const { id, token } = await this.bot.api.channels.createWebhook(
			channel,
			{
				name: 'lightning bridge',
			},
		);

		return { id, token };
	}

	/** process a message event */
	async process_message(opts: message_options): Promise<process_result> {
		return await process_message(this.bot.api, opts);
	}
}
