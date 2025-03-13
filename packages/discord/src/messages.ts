import type { API } from '@discordjs/core';
import type { RawFile } from '@discordjs/rest';
import type { message } from '@jersey/lightning';
import {
	AllowedMentionsTypes,
	type APIEmbed,
	type GatewayMessageUpdateDispatchData,
	type RESTPostAPIWebhookWithTokenJSONBody,
	type RESTPostAPIWebhookWithTokenQuery,
} from 'discord-api-types';
import { fetch_author } from './authors.ts';
import { fetch_files } from './files.ts';
import { fetch_reply_embed, type reply_options } from './replies.ts';
import { fetch_sticker_attachments } from './stickers.ts';

export interface discord_webhook_payload
	extends
		RESTPostAPIWebhookWithTokenJSONBody,
		RESTPostAPIWebhookWithTokenQuery {
	embeds: APIEmbed[];
	files?: RawFile[];
	wait: true;
}

export async function get_discord_message(
	msg: message,
	reply?: reply_options,
	limit_mentions?: boolean,
): Promise<discord_webhook_payload> {
	const payload: discord_webhook_payload = {
		allowed_mentions: limit_mentions
			? { parse: [AllowedMentionsTypes.Role, AllowedMentionsTypes.User] }
			: undefined,
		avatar_url: msg.author.profile,
		// TODO(jersey): since telegram forced multiple message support, split the message into two?
		content: (msg.content?.length || 0) > 2000
			? `${msg.content?.substring(0, 1997)}...`
			: msg.content,
		embeds: (msg.embeds ?? []).map((e) => ({
			...e,
			timestamp: e.timestamp?.toString(),
		})),
		files: await fetch_files(msg.attachments),
		username: msg.author.username,
		wait: true,
	};

	if (reply) {
		const embed = await fetch_reply_embed(reply);

		if (embed) payload.embeds.push(embed);
	}

	if (!payload.content && (!payload.embeds || payload.embeds.length === 0)) {
		// this acts like a blank message and renders nothing
		payload.content = '_ _';
	}

	return payload;
}

export async function get_lightning_message(
	api: API,
	message: GatewayMessageUpdateDispatchData,
): Promise<message> {
	if (message.flags && message.flags & 128) message.content = '*loading...*';

	if (message.type === 7) message.content = '*joined on discord*';

	if (message.sticker_items) {
		if (!message.attachments) message.attachments = [];
		const stickers = await fetch_sticker_attachments(message.sticker_items);
		if (stickers) message.attachments.push(...stickers);
	}

	return {
		attachments: message.attachments?.map(
			(i: typeof message['attachments'][0]) => {
				return {
					file: i.url,
					alt: i.description,
					name: i.filename,
					size: i.size / 1048576, // bytes -> MiB
				};
			},
		),
		author: {
			rawname: message.author.username,
			id: message.author.id,
			color: '#5865F2',
			...await fetch_author(api, message),
		},
		channel: message.channel_id,
		content: message.content,
		embeds: message.embeds.map((i) => ({
			...i,
			timestamp: i.timestamp ? Number(i.timestamp) : undefined,
			video: i.video ? { ...i.video, url: i.video.url ?? '' } : undefined,
		})),
		id: message.id,
		plugin: 'bolt-discord',
		reply_id: message.referenced_message?.id,
		reply: async (msg: message) => {
			await api.channels.createMessage(message.channel_id, {
				...(await get_discord_message(msg)),
				message_reference: { message_id: message.id },
			});
		},
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			Number(BigInt(message.id) >> 22n) + 1420070400000,
		),
	};
}
