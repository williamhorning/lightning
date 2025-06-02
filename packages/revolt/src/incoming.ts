import type { Message } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';
import type { embed, message } from '@lightning/lightning';
import { decodeTime } from '@std/ulid';
import { fetch_author, fetch_channel, fetch_emoji } from './cache.ts';

async function get_content(
	api: Client,
	channel_id: string,
	content?: string | null,
) {
	if (!content) return;

	for (
		const match of content.matchAll(/:([0-7][0-9A-HJKMNP-TV-Z]{25}):/g)
	) {
		try {
			content = content.replace(
				match[0],
				`:${(await fetch_emoji(api, match[1])).name}:`,
			);
		} catch {
			content = content.replace(match[0], `:${match[1]}:`);
		}
	}

	for (
		const match of content.matchAll(/<@([0-7][0-9A-HJKMNP-TV-Z]{25})>/g)
	) {
		try {
			content = content.replace(
				match[0],
				`@${(await fetch_author(api, match[1], channel_id)).username}`,
			);
		} catch {
			content = content.replace(match[0], `@${match[1]}`);
		}
	}

	for (
		const match of content.matchAll(/<#([0-7][0-9A-HJKMNP-TV-Z]{25})>/g)
	) {
		try {
			const channel = await fetch_channel(api, match[1]);
			content = content.replace(
				match[0],
				`#${'name' in channel ? channel.name : `DM${channel._id}`}`,
			);
		} catch {
			content = content.replace(match[0], `#${match[1]}`);
		}
	}

	return content;
}

export async function get_incoming(
	message: Message,
	api: Client,
): Promise<message> {
	return {
		attachments: message.attachments?.map((i) => {
			return {
				file:
					`https://cdn.revoltusercontent.com/attachments/${i._id}/${i.filename}`,
				name: i.filename,
				size: i.size / 1048576,
			};
		}),
		author: await fetch_author(api, message.author, message.channel),
		channel_id: message.channel,
		content: await get_content(api, message.channel, message.content),
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
