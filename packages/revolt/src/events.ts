import { type Client, createClient } from '@jersey/rvapi';
import type { Message } from '@jersey/revolt-api-types';
import { get_lightning_message } from './messages.ts';
import type { revolt_config, revolt_plugin } from './mod.ts';

export function setup_events(
	bot: Client,
	config: revolt_config,
	emit: revolt_plugin['emit'],
) {
	bot.bonfire.on('Ready', (ready) => {
		console.log(
			`[bolt-revolt] ready in ${ready.channels.length} channels and ${ready.servers.length} servers`,
		);
	});

	bot.bonfire.on('Message', async (msg) => {
		if (!msg.channel || msg.channel === 'undefined') return;

		emit('create_message', await get_lightning_message(bot, msg));
	});

	bot.bonfire.on('MessageUpdate', async (msg) => {
		if (!msg.channel || msg.channel === 'undefined') return;

		let oldMessage: Message;

		try {
			oldMessage = await bot.request(
				'get',
				`/channels/${msg.channel}/messages/${msg.id}`,
				undefined,
			) as Message;
		} catch {
			return;
		}

		emit(
			'edit_message',
			await get_lightning_message(bot, {
				...oldMessage,
				...msg.data,
			}),
		);
	});

	bot.bonfire.on('MessageDelete', (msg) => {
		emit('delete_message', {
			channel: msg.channel,
			id: msg.id,
			timestamp: Temporal.Now.instant(),
			plugin: 'bolt-revolt',
		});
	});

	bot.bonfire.on('socket_close', (info) => {
		console.warn('[bolt-revolt] socket closed', info);
		bot = createClient(config);
		setup_events(bot, config, emit);
	});
}
