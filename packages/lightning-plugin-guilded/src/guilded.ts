import type { embed, message } from '@jersey/lightning';
import type { Client, EmbedPayload, WebhookMessageContent } from 'guilded.js';

type guilded_msg = Exclude<WebhookMessageContent, string> & {
	replyMessageIds?: string[];
};

export async function convert_msg(
	msg: message,
	channel?: string,
	bot?: Client,
	allow_everyone = true,
): Promise<guilded_msg> {
	const message = {
		content: msg.content,
		avatar_url: msg.author.profile,
		username: get_valid_username(msg),
		embeds: [
			...fix_embed(msg.embeds),
			...(await get_reply_embeds(msg, channel, bot)),
		],
	} as guilded_msg;

	if (msg.reply_id) message.replyMessageIds = [msg.reply_id];

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

	if (message.embeds?.length === 0 || !message.embeds) delete message.embeds;

	if (!message.content && !message.embeds) message.content = '*empty message*';

	if (!allow_everyone && message.content) {
		message.content = message.content.replace(/@everyone/g, '(a)everyone');
		message.content = message.content.replace(/@here/g, '(a)here');
	}

	return message;
}

function get_valid_username(msg: message) {
	function valid(e: string) {
		if (!e || e.length === 0 || e.length > 25) return false;
		return /^[a-zA-Z0-9_ ()-]*$/gms.test(e);
	}

	if (valid(msg.author.username)) {
		return msg.author.username;
	} else if (valid(msg.author.rawname)) {
		return msg.author.rawname;
	} else {
		return `${msg.author.id}`;
	}
}

async function get_reply_embeds(
	msg: message,
	channel?: string,
	bot?: Client,
) {
	if (!msg.reply_id || !channel || !bot) return [];
	try {
		const msg_replied_to = await bot.messages.fetch(
			channel,
			msg.reply_id,
		);
		let author;
		if (!msg_replied_to.createdByWebhookId) {
			author = await bot.members.fetch(
				msg_replied_to.serverId!,
				msg_replied_to.authorId,
			);
		}
		return [
			{
				author: {
					name: `reply to ${author?.nickname || author?.username || 'a user'}`,
					icon_url: author?.user?.avatar || undefined,
				},
				description: msg_replied_to.content,
			},
			...(msg_replied_to.embeds || []),
		] as EmbedPayload[];
	} catch {
		return [];
	}
}

function fix_embed(embeds: embed[] = []) {
	return embeds.flatMap((embed) => {
		Object.keys(embed).forEach((key) => {
			embed[key as keyof embed] === null
				? (embed[key as keyof embed] = undefined)
				: embed[key as keyof embed];
		});
		if (!embed.description || embed.description === '') return [];
		return [
			{
				...embed,
				timestamp: embed.timestamp ? String(embed.timestamp) : undefined,
			},
		];
	}) as (EmbedPayload & { timestamp: string })[];
}
