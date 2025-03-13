import type { API } from '@discordjs/core';
import type { APIEmbed } from 'discord-api-types';
import { fetch_author } from './authors.ts';

export interface reply_options {
	api?: API;
	channel?: string;
	reply_id?: string;
}

export async function fetch_reply_embed(
	{ api, channel, reply_id }: reply_options,
): Promise<APIEmbed | undefined> {
	if (!api || !channel || !reply_id) return;

	try {
		const message = await api.channels.getMessage(channel, reply_id);

		const { profile, username } = await fetch_author(api, message);

		return {
			author: {
				name: `replying to ${username}`,
				icon_url: profile,
			},
			description: message.content,
		};
	} catch {
		return;
	}
}
