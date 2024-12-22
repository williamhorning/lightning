import type { API } from '@discordjs/core';
import type { GatewayMessageUpdateDispatchData } from 'discord-api-types';
import { get_author } from '../discord_message/get_author.ts';
import { message_to_discord } from '../discord_message/mod.ts';
import type { message } from '@jersey/lightning';

export async function message(
	api: API,
	message: GatewayMessageUpdateDispatchData,
): Promise<message> {
	if (message.flags && message.flags & 128) message.content = 'Loading...';

	if (message.type === 7) message.content = '*joined on discord*';

	if (message.sticker_items) {
		if (!message.attachments) message.attachments = [];
		for (const sticker of message.sticker_items) {
			let type;
			if (sticker.format_type === 1) type = 'png';
			if (sticker.format_type === 2) type = 'apng';
			if (sticker.format_type === 3) type = 'lottie';
			if (sticker.format_type === 4) type = 'gif';
			const url = `https://media.discordapp.net/stickers/${sticker.id}.${type}`;
			const req = await fetch(url, { method: 'HEAD' });
			if (req.ok) {
				message.attachments.push({
					url,
					description: sticker.name,
					filename: `${sticker.name}.${type}`,
					size: 0,
					id: sticker.id,
					proxy_url: url,
				});
			} else {
				message.content = '*used sticker*';
			}
		}
	}

	const { name, avatar } = await get_author(api, message);

	const data = {
		author: {
			profile: avatar,
			username: name,
			rawname: message.author?.username || 'discord user',
			id: message.author?.id || message.webhook_id || '',
			color: '#5865F2',
		},
		channel: message.channel_id,
		content: (message.content?.length || 0) > 2000
			? `${message.content?.substring(0, 1997)}...`
			: message.content,
		id: message.id,
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			Number(BigInt(message.id) >> 22n) + 1420070400000,
		),
		embeds: message.embeds?.map(
			(i: Exclude<typeof message['embeds'], undefined>[0]) => {
				return {
					...i,
					timestamp: i.timestamp ? Number(i.timestamp) : undefined,
				};
			},
		),
		reply: async (msg: message) => {
			if (!data.author.id || data.author.id === '') return;
			await api.channels.createMessage(message.channel_id, {
				...(await message_to_discord(msg)),
				message_reference: {
					message_id: message.id,
				},
			});
		},
		plugin: 'bolt-discord',
		attachments: message.attachments?.map(
			(i: Exclude<typeof message['attachments'], undefined>[0]) => {
				return {
					file: i.url,
					alt: i.description,
					name: i.filename,
					size: i.size / 1048576, // bytes -> MiB
				};
			},
		),
		reply_id: message.referenced_message?.id,
	};

	return data as message;
}
