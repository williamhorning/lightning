import type { API } from '@discordjs/core';
import { calculateUserDefaultAvatarIndex } from '@discordjs/rest';
import type {
	APIGuildMember,
	GatewayMessageUpdateDispatchData,
} from 'discord-api-types';

export async function fetch_author(
	api: API,
	message: GatewayMessageUpdateDispatchData,
): Promise<{ profile: string; username: string }> {
	let profile = message.author.avatar !== null
		? `https://cdn.discordapp.com/avatars/${message.author.id}/${message.author.avatar}.png`
		: `https://cdn.discordapp.com/embed/avatars/${
			calculateUserDefaultAvatarIndex(message.author.id)
		}.png`;

	let username = message.author.global_name ?? message.author.username;

	if (message.guild_id) {
		try {
			// remove type assertion once deno resolves the return type for getMember properly
			const member = message.member ?? await api.guilds.getMember(
				message.guild_id,
				message.author.id,
			) as APIGuildMember;

			if (member.avatar) {
				profile =
					`https://cdn.discordapp.com/guilds/${message.guild_id}/users/${message.author.id}/avatars/${member.avatar}.png`;
			}

			if (member.nick) username = member.nick;
		} catch {
			// safe to ignore, we already have a name and avatar
		}
	}

	return { profile, username };
}
