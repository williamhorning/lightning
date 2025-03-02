import type { message } from '@jersey/lightning';
import type { Client, Message } from 'guilded.js';
import { convert_msg } from '../guilded.ts';
import { get_author } from './author.ts';
import { map_embed } from './map_embed.ts';

export async function guilded_to_message(
	msg: Message,
	bot: Client,
): Promise<message | undefined> {
	if (msg.serverId === null) return;

	const author = await get_author(msg, bot);

	const timestamp = Temporal.Instant.fromEpochMilliseconds(
		msg.createdAt.valueOf(),
	);

	let content = msg.content.replaceAll('\n```\n```\n', '\n');

	// /!\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm
	const urls = content.match(
		/\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
	) || [];

	content = content.replaceAll(
		/\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
		'',
	);

	const attachments_urls = [] as [string, number][];

	try {
		const signed = await (await fetch("https://www.guilded.gg/api/v1/url-signatures", {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'Accept': 'application/json',
				'Authorization': `Bearer ${bot.token}`,
			},
			body: JSON.stringify(urls.map((url) => ({ url }))),
		})).json();

		for (const url of signed) {
			if (url.signature) {
				// TODO(jersey): store the signed url somewhere and have our own proxy, like telegram
				const resp = await fetch(url.signature, {
					method: "HEAD"
				});

				const size = parseInt(resp.headers.get('Content-Length') || '0');

				attachments_urls.push([url.url, size]);
			}
		}
	} catch {
		// ignore
	}

	return {
		author: {
			...author,
			color: '#F5C400',
		},
		attachments: attachments_urls.map(([url, size]) => {
			return {
				name: url.split('/').pop()?.split('?')[0] || 'unknown',
				file: url,
				size,
			};
		}),
		channel: msg.channelId,
		id: msg.id,
		timestamp,
		embeds: msg.embeds?.map(map_embed),
		plugin: 'bolt-guilded',
		reply: async (reply: message) => {
			await msg.reply(await convert_msg(reply));
		},
		content,
		reply_id: msg.isReply ? msg.replyMessageIds[0] : undefined,
	};
}
