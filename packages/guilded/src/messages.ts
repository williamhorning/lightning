import type { message } from '@jersey/lightning';
import type { Client } from '@jersey/guildapi';
import { fetch_attachments } from './attachments.ts';
import { fetch_author, get_valid_username } from './authors.ts';
import { get_guilded_embeds, get_lightning_embeds } from './embeds.ts';
import { fetch_reply_embed } from './replies.ts';
import type { ChatEmbed, ChatMessage } from '@jersey/guilded-api-types';

type webhook_payload = {
	content?: string;
	embeds?: ChatEmbed[];
	replyMessageIds?: string[];
	avatar_url?: string;
	username?: string;
};

export async function get_lightning_message(
	msg: ChatMessage,
	bot: Client,
): Promise<message | undefined> {
	if (!msg.serverId) return;

	let content = msg.content?.replaceAll('\n```\n```\n', '\n');

	const urls = content?.match(
		/!\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
	) || [];

	content = content?.replaceAll(
		/!\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
		'',
	);

	return {
		author: {
			...await fetch_author(msg, bot),
			color: '#F5C400',
		},
		attachments: await fetch_attachments(bot, urls),
		channel: msg.channelId,
		id: msg.id,
		timestamp: Temporal.Instant.from(
			msg.createdAt,
		),
		embeds: get_lightning_embeds(msg.embeds),
		plugin: 'bolt-guilded',
		reply: async (reply: message) => {
			await bot.request(
				'post',
				`/channels/${msg.channelId}/messages`,
				await get_guilded_message(reply),
			);
		},
		content,
		reply_id: msg.replyMessageIds && msg.replyMessageIds.length > 0
			? msg.replyMessageIds[0]
			: undefined,
	};
}

export async function get_guilded_message(
	msg: message,
	channel?: string,
	bot?: Client,
	everyone = true,
): Promise<webhook_payload> {
	const message: webhook_payload = {
		content: msg.content,
		avatar_url: msg.author.profile,
		username: get_valid_username(msg),
		embeds: get_guilded_embeds(msg.embeds),
	};

	if (msg.reply_id) {
		const embed = await fetch_reply_embed(msg, channel, bot);

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

	if (!everyone && message.content) {
		message.content = message.content.replace(/@everyone/g, '(a)everyone');
		message.content = message.content.replace(/@here/g, '(a)here');
	}

	return message;
}
