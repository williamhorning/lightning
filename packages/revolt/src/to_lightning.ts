import type { Channel, Embed, Message, User } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';
import type { embed, message, message_author } from '@jersey/lightning';
import { decodeTime } from '@std/ulid';
import { to_revolt } from './to_revolt.ts';
import { fetch_member } from './fetch_member.ts';

export async function to_lightning(
	api: Client,
	message: Message,
): Promise<message> {
	return {
		attachments: message.attachments?.map((i) => {
			return {
				file: `https://autumn.revolt.chat/attachments/${i._id}/${i.filename}`,
				name: i.filename,
				size: i.size,
			};
		}),
		author: await get_author(api, message.author, message.channel),
		channel: message.channel,
		content: message.content ?? undefined,
		embeds: (message.embeds as Embed[] | undefined)?.map<embed>((i) => {
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
					...(await to_revolt(
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

async function get_author(
	api: Client,
	author_id: string,
	channel_id: string,
): Promise<message_author> {
	try {
		const channel = await api.request(
			'get',
			`/channels/${channel_id}`,
			undefined,
		) as Channel;

		const author = await api.request(
			'get',
			`/users/${author_id}`,
			undefined,
		) as User;

		const author_data = {
			id: author_id,
			rawname: author.username,
			username: author.username,
			color: '#FF4654',
			profile: author.avatar
				? `https://autumn.revolt.chat/avatars/${author.avatar._id}`
				: undefined,
		};

		if (channel.channel_type !== 'TextChannel') {
			return author_data;
		} else {
			try {
				const member = await fetch_member(api, channel, author_id);

				return {
					...author_data,
					username: member.nickname ?? author_data.username,
					profile: member.avatar
						? `https://autumn.revolt.chat/avatars/${member.avatar._id}`
						: author_data.profile,
				};
			} catch {
				return author_data;
			}
		}
	} catch {
		return {
			id: author_id,
			rawname: 'RevoltUser',
			username: 'Revolt User',
			color: '#FF4654',
		};
	}
}
