import type { message } from '@jersey/lightning';
import type { Client } from '@jersey/guildapi';
import { fetch_author } from './authors.ts';
import type { ChatEmbed } from '@jersey/guilded-api-types';

export async function fetch_reply_embed(
	msg: message,
	channel?: string,
	bot?: Client,
): Promise<ChatEmbed | undefined> {
	if (!msg.reply_id || !channel || !bot) return;

	try {
		const replied_to = await bot.request(
			'get',
			`/channels/${channel}/messages/${msg.reply_id}`,
			undefined,
		);

		const author = await fetch_author(replied_to.message, bot);

		return {
			author: {
				name: `reply to ${author.username}`,
				icon_url: author.profile,
			},
			description: replied_to.message.content,
		};
	} catch {
		return;
	}
}
