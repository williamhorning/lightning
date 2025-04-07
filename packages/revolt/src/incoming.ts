import type { Client } from '@jersey/rvapi';
import type { Message as APIMessage } from '@jersey/revolt-api-types';
import { decodeTime } from '@std/ulid';
import type { embed, message } from '@jersey/lightning';
import { fetchAuthor } from './cache.ts';

export async function getIncomingMessage(
	message: APIMessage,
	api: Client,
): Promise<message> {
	return {
		attachments: message.attachments?.map((i) => {
			return {
				file: `https://autumn.revolt.chat/attachments/${i._id}/${i.filename}`,
				name: i.filename,
				size: i.size / 1048576,
			};
		}),
		author: await fetchAuthor(api, message.author, message.channel),
		channel_id: message.channel,
		content: message.content ?? undefined,
		embeds: message.embeds?.map((i) => {
			return {
				color: 'colour' in i && i.colour
					? parseInt(i.colour.replace('#', ''), 16)
					: undefined,
				...i,
			} as embed;
		}),
		message_id: message._id,
		plugin: 'bolt-revolt',
		timestamp: message.edited
			? Temporal.Instant.from(message.edited)
			: Temporal.Instant.fromEpochMilliseconds(decodeTime(message._id)),
		reply_id: message.replies?.[0] ?? undefined,
	};
}
