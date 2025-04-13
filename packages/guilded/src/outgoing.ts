import type { Client } from '@jersey/guildapi';
import type { ChatEmbed } from '@jersey/guilded-api-types';
import type { message } from '@jersey/lightning';
import { fetch_author } from './incoming.ts';

type guilded_payload = {
	content?: string;
	embeds?: ChatEmbed[];
	replyMessageIds?: string[];
	avatar_url?: string;
	username?: string;
};

const username = /^[a-zA-Z0-9_ ()-]{1,25}$/ms;

function get_name(msg: message): string {
	if (username.test(msg.author.username)) {
		return msg.author.username;
	} else if (username.test(msg.author.rawname)) {
		return msg.author.rawname;
	} else {
		return `${msg.author.id}`;
	}
}

async function fetch_reply(
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

		const author = await fetch_author(replied_to.message, client);

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

export async function get_outgoing(
	msg: message,
	client: Client,
	limitMentions?: boolean,
): Promise<guilded_payload> {
	const message: guilded_payload = {
		content: msg.content,
		avatar_url: msg.author.profile,
		username: get_name(msg),
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
				timestamp: i.timestamp ? i.timestamp.toString() : undefined,
			};
		}),
	};

	if (msg.reply_id) {
		const embed = await fetch_reply(msg, client);

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
		message.content = message.content.replace(/@everyone/gi, '(a)everyone');
		message.content = message.content.replace(/@here/gi, '(a)here');
	}

	return message;
}
