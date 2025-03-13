import type { Client } from '@discordjs/core';
import { GatewayDispatchEvents } from 'discord-api-types';
import { get_lightning_command } from './commands.ts';
import { get_lightning_message } from './messages.ts';
import type { discord_plugin } from './mod.ts';

export function setup_events(
	client: Client,
	emit: discord_plugin['emit'],
): void {
	// @ts-ignore deno isn't properly handling the eventemitter code
	client.once(GatewayDispatchEvents.Ready, ({ data }) => {
		console.log(
			`[discord] ready as ${data.user.username}#${data.user.discriminator} in ${data.guilds.length} guilds`,
		);
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	client.on(GatewayDispatchEvents.MessageCreate, async (msg) => {
		emit('create_message', await get_lightning_message(msg.api, msg.data));
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	client.on(GatewayDispatchEvents.MessageUpdate, async (msg) => {
		emit('edit_message', await get_lightning_message(msg.api, msg.data));
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	client.on(GatewayDispatchEvents.MessageDelete, (msg) => {
		emit('delete_message', {
			channel: msg.data.channel_id,
			id: msg.data.id,
			plugin: 'bolt-discord',
			timestamp: Temporal.Now.instant(),
		});
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	client.on(GatewayDispatchEvents.InteractionCreate, (cmd) => {
		const command = get_lightning_command(cmd);
		if (command) emit('create_command', command);
	});
}
