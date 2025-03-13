import type { DataMessageSend, Message } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';
import type { embed, message } from '@jersey/lightning';
import { decodeTime } from '@std/ulid';
import { get_author } from './author.ts';
import { upload_attachments } from './attachments.ts';
import { get_revolt_embeds } from './embeds.ts';

export async function get_lightning_message(
	api: Client,
	message: Message,
): Promise<message> {
	return {
		attachments: message.attachments?.map((i) => {
			return {
				file: `https://autumn.revolt.chat/attachments/${i._id}/${i.filename}`,
				name: i.filename,
				size: i.size / 1048576,
			};
		}),
		author: await get_author(api, message.author, message.channel),
		channel: message.channel,
		content: message.content ?? undefined,
		embeds: message.embeds?.map((i) => {
			return {
				color: 'colour' in i && i.colour
					? parseInt(i.colour.replace('#', ''), 16)
					: undefined,
				...i,
			} as embed;
		}),
		id: message._id,
		plugin: 'bolt-revolt',
		reply: async (msg: message, masquerade = true) => {
			await api.request(
				'post',
				`/channels/${message.channel}/messages`,
				{
					...(await get_revolt_message(
						api,
						{ ...msg, reply_id: message._id },
						masquerade as boolean,
					)),
				},
			);
		},
		timestamp: message.edited
			? Temporal.Instant.from(message.edited)
			: Temporal.Instant.fromEpochMilliseconds(decodeTime(message._id)),
		reply_id: message.replies?.[0] ?? undefined,
	};
}

export async function get_revolt_message(
	api: Client,
	message: message,
	masquerade = true,
): Promise<DataMessageSend> {
	const attachments = await upload_attachments(api, message.attachments);
	const embeds = get_revolt_embeds(message.embeds);

	if (
		(!message.content || message.content.length < 1) &&
		(!embeds || embeds.length < 1) &&
		(!attachments || attachments.length < 1)
	) {
		message.content = '*empty message*';
	}

	return {
		attachments,
		content: (message.content?.length || 0) > 2000
			? `${message.content?.substring(0, 1997)}...`
			: message.content,
		embeds,
		replies: message.reply_id
			? [{ id: message.reply_id, mention: true }]
			: undefined,
		masquerade: masquerade
			? {
				name: message.author.username,
				avatar: message.author.profile,
				colour: message.author.color,
			}
			: undefined,
	};
}
