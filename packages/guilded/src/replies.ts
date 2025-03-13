import type { EmbedPayload } from '@guildedjs/api';
import type { message } from '@jersey/lightning';
import type { Client } from 'guilded.js';
import { fetch_author } from './authors.ts';

export async function fetch_reply_embed(
	msg: message,
	channel?: string,
	bot?: Client,
): Promise<EmbedPayload | undefined> {
	if (!msg.reply_id || !channel || !bot) return;

	try {
		const replied_to_message = await bot.messages.fetch(
			channel,
			msg.reply_id,
		);

		const author = await fetch_author(replied_to_message, bot);

		return {
			author: {
				name: `reply to ${author.username}`,
				icon_url: author.profile,
			},
			description: replied_to_message.content,
		};
	} catch {
		return;
	}
}
