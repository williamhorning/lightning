import type { Client } from '@jersey/guildapi';
import type { message } from '@jersey/lightning';
import type { ChatEmbed } from '@jersey/guilded-api-types';
import { fetchAuthor } from './incoming.ts';

type GuildedPayload = {
	content?: string;
	embeds?: ChatEmbed[];
	replyMessageIds?: string[];
	avatar_url?: string;
	username?: string;
};

const usernameRegex = /^[a-zA-Z0-9_ ()-]{1,25}$/ms;

function getUsername(msg: message): string {
	if (usernameRegex.test(msg.author.username)) {
		return msg.author.username;
	} else if (usernameRegex.test(msg.author.rawname)) {
		return msg.author.rawname;
	} else {
		return `${msg.author.id}`;
	}
}

async function fetchReplyEmbed(
	msg: message,
	client: Client,
): Promise<ChatEmbed | undefined> {
	if (!msg.reply_id) return;

	try {
		const replied_to = await client.request(
			'get',
			`/channels/${msg.channel_id}/messages/${msg.reply_id}`,
			undefined,
		);

		const author = await fetchAuthor(replied_to.message, client);

		return {
			author: {
				name: `reply to ${author.username}`,
				icon_url: author.profile,
			},
			description: replied_to.message.content,
		};
	} catch {
		return;
	}
}

export async function getOutgoingMessage(
	msg: message,
	client: Client,
	limitMentions?: boolean,
): Promise<GuildedPayload> {
	const message: GuildedPayload = {
		content: msg.content,
		avatar_url: msg.author.profile,
		username: getUsername(msg),
		embeds: msg.embeds?.map((i) => {
			return {
				...i,
				fields: i.fields
					? i.fields.map((j) => {
						return {
							...j,
							inline: j.inline ?? false,
						};
					})
					: undefined,
				timestamp: i.timestamp ? String(i.timestamp) : undefined,
			};
		}),
	};

	if (msg.reply_id) {
		const embed = await fetchReplyEmbed(msg, client);

		if (embed) {
			if (!message.embeds) message.embeds = [];
			message.embeds.push(embed);
		}
	}

	if (msg.attachments?.length) {
		if (!message.embeds) message.embeds = [];
		message.embeds.push({
			title: 'attachments',
			description: msg.attachments
				.slice(0, 5)
				.map((a) => {
					return `![${a.alt || a.name}](${a.file})`;
				})
				.join('\n'),
		});
	}

	if (!message.content && !message.embeds) message.content = '\u2800';

	if (limitMentions && message.content) {
		message.content = message.content.replace(/@everyone/g, '(a)everyone');
		message.content = message.content.replace(/@here/g, '(a)here');
	}

	return message;
}
