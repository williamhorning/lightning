import type { API } from '@discordjs/core';
import type { APIMessage } from 'discord-api-types';
import { get_author } from './get_author.ts';

export async function reply_embed(api: API, channel: string, id: string) {
	try {
		const message = await api.channels.getMessage(
			channel,
			id,
		) as APIMessage;

		const { name, avatar } = await get_author(api, message);

		return {
			author: {
				name: `replying to ${name}`,
				icon_url: avatar,
			},
			description: message.content,
		};
	} catch {
		return;
	}
}
