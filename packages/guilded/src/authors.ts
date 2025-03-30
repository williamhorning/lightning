import type { message, message_author } from '@jersey/lightning';
import type { Client } from '@jersey/guildapi';
import type { ChatMessage, ServerMember } from '@jersey/guilded-api-types';

export async function fetch_author(
	msg: ChatMessage,
	bot: Client,
): Promise<message_author> {
	try {
		if (!msg.createdByWebhookId) {
			const author = await bot.request(
				'get',
				`/servers/${msg.serverId}/members/${msg.createdBy}`,
				undefined,
			) as ServerMember;

			return {
				username: author.nickname || author.user.name,
				rawname: author.user.name,
				id: msg.createdBy,
				profile: author.user.avatar || undefined,
			};
		} else {
			const { webhook } = await bot.request(
				'get',
				`/servers/${msg.serverId}/webhooks/${msg.createdByWebhookId}`,
				undefined,
			);

			return {
				username: webhook.name,
				rawname: webhook.name,
				id: webhook.id,
				profile: webhook.avatar,
			};
		}
	} catch {
		return {
			username: 'Guilded User',
			rawname: 'GuildedUser',
			id: msg.createdByWebhookId ?? msg.createdBy,
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
