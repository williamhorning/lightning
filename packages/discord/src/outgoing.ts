import {
	AllowedMentionsTypes,
	type APIEmbed,
	type APIMessageReference,
	ButtonStyle,
	ComponentType,
	type RESTPostAPIWebhookWithTokenJSONBody,
	type RESTPostAPIWebhookWithTokenQuery,
} from 'discord-api-types';
import type { attachment, message } from '@jersey/lightning';
import type { RawFile } from '@discordjs/rest';
import type { API } from '@discordjs/core';

export interface DiscordPayload
	extends
		RESTPostAPIWebhookWithTokenJSONBody,
		RESTPostAPIWebhookWithTokenQuery {
	embeds: APIEmbed[];
	files?: RawFile[];
	message_reference?: APIMessageReference & { message_id: string },
	wait: true;
}

async function fetchReplyComponent(
	channelID: string,
	replyID?: string,
	api?: API,
) {
	try {
		if (!replyID || !api) return;

		const channel = await api.channels.get(channelID);
		const channelPath = 'guild_id' in channel
			? `${channel.guild_id}/${channelID}`
			: `@me/${channelID}`;
		const msg = await api.channels.getMessage(channelID, replyID);

		return [{
			type: ComponentType.ActionRow as const,
			components: [{
				type: ComponentType.Button as const,
				style: ButtonStyle.Link as const,
				label: `reply to ${msg.author.username}`,
				url: `https://discord.com/channels/${channelPath}/${replyID}`,
			}],
		}];
	} catch {
		// TODO(jersey): maybe log this?
		return;
	}
}

async function fetchFiles(
	attachments: attachment[] | undefined,
): Promise<RawFile[] | undefined> {
	if (!attachments) return;

	let totalSize = 0;

	return (await Promise.all(
		attachments.map(async (attachment) => {
			try {
				if (attachment.size >= 25) return;
				if (totalSize + attachment.size >= 25) return;

				const data = new Uint8Array(
					await (await fetch(attachment.file, {
						signal: AbortSignal.timeout(5000),
					})).arrayBuffer(),
				);

				const name = attachment.name ?? attachment.file?.split('/').pop()!;

				totalSize += attachment.size;

				return { data, name };
			} catch {
				return;
			}
		}),
	)).filter((i) => i !== undefined);
}

export async function getOutgoingMessage(
	msg: message,
	api: API,
	button_reply: boolean,
	limit_mentions: boolean,
): Promise<DiscordPayload> {
	const payload: DiscordPayload = {
		allowed_mentions: limit_mentions
			? { parse: [AllowedMentionsTypes.Role, AllowedMentionsTypes.User] }
			: undefined,
		avatar_url: msg.author.profile,
		// TODO(jersey): since telegram forced multiple message support, split the message into two?
		content: (msg.content?.length || 0) > 2000
			? `${msg.content?.substring(0, 1997)}...`
			: msg.content,
		components: button_reply ? await fetchReplyComponent(msg.channel_id, msg.reply_id, api) : undefined,
		embeds: (msg.embeds ?? []).map((e) => ({
			...e,
			timestamp: e.timestamp?.toString(),
		})),
		files: await fetchFiles(msg.attachments),
		message_reference: !button_reply && msg.reply_id ? { type: 0, channel_id: msg.channel_id, message_id: msg.reply_id } : undefined,
		username: msg.author.username,
		wait: true,
	};

	if (!payload.content && (!payload.embeds || payload.embeds.length === 0)) {
		// this acts like a blank message and renders nothing
		payload.content = '_ _';
	}

	return payload;
}
