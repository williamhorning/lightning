import type { message_author } from '@jersey/lightning';
import type { Client, Message } from 'guilded.js';

export async function get_author(
	msg: Message,
	bot: Client,
): Promise<message_author> {
	if (!msg.createdByWebhookId && msg.authorId !== 'Ann6LewA') {
		try {
			const au = await bot.members.fetch(
				msg.serverId!,
				msg.authorId,
			);

			return {
				username: au.nickname || au.username || au.user?.name || 'Guilded User',
				rawname: au.username || au.user?.name || 'Guilded User',
				id: msg.authorId,
				profile: au.user?.avatar || undefined,
			};
		} catch {
			return {
				username: 'Guilded User',
				rawname: 'GuildedUser',
				id: msg.authorId,
			};
		}
	} else if (msg.createdByWebhookId) {
		// try to fetch webhook?
		try {
			const wh = await bot.webhooks.fetch(
				msg.serverId!,
				msg.createdByWebhookId,
			);

			return {
				username: wh.name,
				rawname: wh.name,
				id: wh.id,
				profile: wh.raw.avatar,
			};
		} catch {
			return {
				username: 'Guilded Webhook',
				rawname: 'GuildedWebhook',
				id: msg.createdByWebhookId,
			};
		}
	} else {
		return {
			username: 'Guilded User',
			rawname: 'GuildedUser',
			id: msg.authorId,
		};
	}
}
