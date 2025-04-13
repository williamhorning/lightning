import type { Client } from '@jersey/guildapi';
import type { ChatMessage, ServerMember } from '@jersey/guilded-api-types';
import type { attachment, message } from '@jersey/lightning';

export async function fetch_author(msg: ChatMessage, client: Client) {
	try {
		if (!msg.createdByWebhookId) {
			const { member: author } = await client.request(
				'get',
				`/servers/${msg.serverId}/members/${msg.createdBy}`,
				undefined,
			) as { member: ServerMember };

			return {
				username: author.nickname || author.user.name,
				rawname: author.user.name,
				id: msg.createdBy,
				profile: author.user.avatar || undefined,
			};
		} else {
			const { webhook } = await client.request(
				'get',
				`/servers/${msg.serverId}/webhooks/${msg.createdByWebhookId}`,
				undefined,
			);

			return {
				username: webhook.name,
				rawname: webhook.name,
				id: webhook.id,
				profile: webhook.avatar,
			};
		}
	} catch {
		return {
			username: 'Guilded User',
			rawname: 'GuildedUser',
			id: msg.createdByWebhookId ?? msg.createdBy,
		};
	}
}

async function fetch_attachments(urls: string[], client: Client) {
	const attachments: attachment[] = [];

	try {
		const signed = await client.request('post', '/url-signatures', {
			urls: urls.map(
				(url) => (url.split('(').pop())?.split(')')[0],
			).filter((i) => i !== undefined),
		});

		for (const url of signed.urlSignatures) {
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

export async function get_incoming(
	msg: ChatMessage,
	client: Client,
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
		attachments: await fetch_attachments(urls, client),
		author: {
			...await fetch_author(msg, client),
			color: '#F5C400',
		},
		channel_id: msg.channelId,
		content,
		embeds: msg.embeds?.map((embed) => ({
			...embed,
			author: embed.author
				? {
					...embed.author,
					name: embed.author.name || '',
				}
				: undefined,
			image: embed.image
				? {
					...embed.image,
					url: embed.image.url || '',
				}
				: undefined,
			thumbnail: embed.thumbnail
				? {
					...embed.thumbnail,
					url: embed.thumbnail.url || '',
				}
				: undefined,
			timestamp: embed.timestamp ? Number(embed.timestamp) : undefined,
		})),
		message_id: msg.id,
		plugin: 'bolt-guilded',
		reply_id: msg.replyMessageIds && msg.replyMessageIds.length > 0
			? msg.replyMessageIds[0]
			: undefined,
		timestamp: Temporal.Instant.from(
			msg.createdAt,
		),
	};
}
