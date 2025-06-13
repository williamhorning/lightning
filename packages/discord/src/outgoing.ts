import {
	AllowedMentionsTypes,
	type API,
	type APIEmbed,
	type DescriptiveRawFile,
	type RESTPostAPIWebhookWithTokenJSONBody,
	type RESTPostAPIWebhookWithTokenQuery,
} from '@discordjs/core';
import type { attachment, message } from '@lightning/lightning';

export interface discord_payload
	extends
		RESTPostAPIWebhookWithTokenJSONBody,
		RESTPostAPIWebhookWithTokenQuery {
	embeds: APIEmbed[];
	files?: DescriptiveRawFile[];
	message_reference?: { type: number; channel_id: string; message_id: string };
	wait: true;
}

async function fetch_reply(
	channelID: string,
	replies?: string[],
	api?: API,
) {
	try {
		if (!replies || !api) return;

		const channel = await api.channels.get(channelID);
		const channelPath = 'guild_id' in channel
			? `${channel.guild_id}/${channelID}`
			: `@me/${channelID}`;

		return [{
			type: 1,
			components: await Promise.all(
				replies.slice(0, 5).map(async (reply) => ({
					type: 1 as const,
					components: [{
						type: 2 as const,
						style: 5 as const,
						label: `reply to ${
							(await api.channels.getMessage(channelID, reply)).author.username
						}`,
						url: `https://discord.com/channels/${channelPath}/${replies}`,
					}],
				})),
			),
		}];
	} catch {
		return;
	}
}

async function fetch_files(
	api: API,
	channel_id: string,
	attachments: attachment[] | undefined,
): Promise<DescriptiveRawFile[] | undefined> {
	if (!attachments) return;

	let attachment_max = 10;

	try {
		const channel = await api.channels.get(channel_id);
		if ('guild_id' in channel && channel.guild_id) {
			const server = await api.guilds.get(channel.guild_id, { with_counts: false });
			if (server.premium_tier === 2) attachment_max = 50;
			if (server.premium_tier === 3) attachment_max = 100;
		}
	} catch {
		// If we can't get the server's attachment limit, default to 10MB
	}

	return (await Promise.all(
		attachments.map(async (attachment) => {
			try {
				if (attachment.size >= attachment_max) return;
				return {
					data: new Uint8Array(
						await (await fetch(attachment.file, {
							signal: AbortSignal.timeout(5000),
						})).arrayBuffer(),
					),
					name: attachment.name ?? attachment.file?.split('/').pop()!,
				};
			} catch {
				return;
			}
		}),
	)).filter((i) => i !== undefined);
}

export async function get_outgoing_message(
	msg: message,
	api: API,
	button_reply: boolean,
	limit_mentions: boolean,
): Promise<discord_payload> {
	const payload: discord_payload = {
		allowed_mentions: limit_mentions
			? { parse: [AllowedMentionsTypes.Role, AllowedMentionsTypes.User] }
			: undefined,
		avatar_url: msg.author.profile,
		// TODO(jersey): since telegram forced multiple message support, split the message into two?
		content: (msg.content?.length || 0) > 2000
			? `${msg.content?.substring(0, 1997)}...`
			: msg.content,
		components: button_reply
			? await fetch_reply(msg.channel_id, msg.reply_id, api)
			: undefined,
		embeds: (msg.embeds ?? []).map((e) => ({
			...e,
			timestamp: e.timestamp?.toString(),
		})),
		files: await fetch_files(api, msg.channel_id, msg.attachments),
		message_reference: !button_reply && msg.reply_id
			? { type: 0, channel_id: msg.channel_id, message_id: msg.reply_id[0] }
			: undefined,
		username: msg.author.username,
		wait: true,
	};

	if (!payload.content && (!payload.embeds || payload.embeds.length === 0)) {
		// this acts like a blank message and renders nothing
		payload.content = '_ _';
	}

	return payload;
}
