import type { attachment, message } from '@jersey/lightning';
import type { Client, Message } from 'guilded.js';
import { convert_msg } from '../guilded.ts';
import { get_author } from './author.ts';
import { map_embed } from './map_embed.ts';

export async function guilded_to_message(
	msg: Message,
	bot: Client,
): Promise<message | undefined> {
	if (msg.serverId === null) return;

	let content = msg.content.replaceAll('\n```\n```\n', '\n');

	const urls = content.match(
		/!\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
	) || [];

	content = content.replaceAll(
		/!\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
		'',
	);

	return {
		author: {
			...await get_author(msg, bot),
			color: '#F5C400',
		},
		attachments: await get_attachments(bot, urls),
		channel: msg.channelId,
		id: msg.id,
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			msg.createdAt.valueOf(),
		),
		embeds: msg.embeds?.map(map_embed),
		plugin: 'bolt-guilded',
		reply: async (reply: message) => {
			await msg.reply(await convert_msg(reply));
		},
		content,
		reply_id: msg.isReply ? msg.replyMessageIds[0] : undefined,
	};
}

async function get_attachments(bot: Client, urls: string[]) {
	const attachments = [] as attachment[];

	try {
		const signed =
			await (await fetch('https://www.guilded.gg/api/v1/url-signatures', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Accept': 'application/json',
					'Authorization': `Bearer ${bot.token}`,
				},
				body: JSON.stringify({
					urls: urls.map((url) => (url.split('(').pop()?.split(')')[0])),
				}),
			})).json();

		for (const url of signed.urlSignatures || []) {
			if (url.signature) {
				const resp = await fetch(url.signature, {
					method: 'HEAD',
				});

				attachments.push({
					name: url.signature.split('/').pop()?.split('?')[0] || 'unknown',
					file: url.signature,
					size: parseInt(resp.headers.get('Content-Length') || '0') / 1048576,
				});
			}
		}
	} catch {
		// ignore
	}

	return attachments;
}
