import type { Channel, User } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';
import type { message_author } from '@jersey/lightning';
import { fetch_member } from './member.ts';

export async function get_author(
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

		if (channel.channel_type !== 'TextChannel') return author_data;

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
	} catch {
		return {
			id: author_id,
			rawname: 'RevoltUser',
			username: 'Revolt User',
			color: '#FF4654',
		};
	}
}
