import type { message } from '@jersey/lightning';
import type { API } from '@discordjs/core';
import type {
	RESTPostAPIWebhookWithTokenJSONBody,
	RESTPostAPIWebhookWithTokenQuery,
} from 'discord-api-types';
import type { RawFile } from '@discordjs/rest';
import { reply_embed } from './reply_embed.ts';
import { files_up_to_25MiB } from './files.ts';

export interface discord_message_send
	extends
		RESTPostAPIWebhookWithTokenJSONBody,
		RESTPostAPIWebhookWithTokenQuery {
	files?: RawFile[];
	wait: true;
}

export async function message_to_discord(
	msg: message,
	api?: API,
	channel?: string,
	reply_id?: string,
): Promise<discord_message_send> {
	const discord: discord_message_send = {
		avatar_url: msg.author.profile,
		content: (msg.content?.length || 0) > 2000
			? `${msg.content?.substring(0, 1997)}...`
			: msg.content,
		embeds: msg.embeds?.map((e) => {
			return { ...e, timestamp: e.timestamp?.toString() };
		}),
		username: msg.author.username,
		wait: true,
	};

	if (api && channel && reply_id) {
		const embed = await reply_embed(api, channel, reply_id);
		if (embed) {
			if (!discord.embeds) discord.embeds = [];
			discord.embeds.push(embed);
		}
	}

	discord.files = await files_up_to_25MiB(msg.attachments);

	if (!discord.content && (!discord.embeds || discord.embeds.length === 0)) {
		discord.content = '*empty message*';
	}

	return discord;
}
