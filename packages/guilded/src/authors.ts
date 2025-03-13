import type { message, message_author } from '@jersey/lightning';
import type { Client, Message } from 'guilded.js';

export async function fetch_author(
	msg: Message,
	bot: Client,
): Promise<message_author> {
	if (!msg.createdByWebhookId && msg.authorId !== 'Ann6LewA') {
		try {
			const author = await bot.members.fetch(
				msg.serverId!,
				msg.authorId,
			);

			return {
				username: author.nickname || author.username || author.user?.name ||
					'Guilded User',
				rawname: author.username || author.user?.name || 'Guilded User',
				id: msg.authorId,
				profile: author.user?.avatar || undefined,
			};
		} catch {
			return {
				username: 'Guilded User',
				rawname: 'GuildedUser',
				id: msg.authorId,
			};
		}
	} else if (msg.createdByWebhookId) {
		try {
			const webhook = await bot.webhooks.fetch(
				msg.serverId!,
				msg.createdByWebhookId,
			);

			return {
				username: webhook.name,
				rawname: webhook.name,
				id: webhook.id,
				profile: webhook.raw.avatar,
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

function is_valid_username(e: string): boolean {
	if (!e || e.length === 0 || e.length > 25) return false;
	return /^[a-zA-Z0-9_ ()-]*$/gms.test(e);
}

export function get_valid_username(msg: message): string {
	if (is_valid_username(msg.author.username)) {
		return msg.author.username;
	} else if (is_valid_username(msg.author.rawname)) {
		return msg.author.rawname;
	} else {
		return `${msg.author.id}`;
	}
}
