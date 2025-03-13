import type { Client } from 'guilded.js';
import { get_lightning_message } from './messages.ts';
import type { guilded_plugin } from './mod.ts';

export function setup_events(bot: Client, emit: guilded_plugin['emit']) {
	// @ts-ignore deno isn't properly handling the eventemitter code
	bot.on('ready', () => {
		console.log(`[bolt-guilded] ready as ${bot.user?.name}`);
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	bot.on('messageCreated', async (message) => {
		const msg = await get_lightning_message(message, bot);
		if (msg) emit('create_message', msg);
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	bot.on('messageUpdated', async (message) => {
		const msg = await get_lightning_message(message, bot);
		if (msg) emit('edit_message', msg);
	});
	// @ts-ignore deno isn't properly handling the eventemitter code
	bot.on('messageDeleted', (del) => {
		emit('delete_message', {
			channel: del.channelId,
			id: del.id,
			plugin: 'bolt-guilded',
			timestamp: Temporal.Instant.from(del.deletedAt),
		});
	});
	// @ts-ignore deno isn't dealing with the import
	bot.ws.emitter.on('exit', () => {
		bot.ws.connect();
	});
}
