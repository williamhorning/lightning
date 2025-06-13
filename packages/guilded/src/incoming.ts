import type { Client } from '@jersey/guildapi';
import type {
	ChatMessage,
	ServerMember,
	Webhook,
} from '@jersey/guilded-api-types';
import { type attachment, cacher, type message } from '@lightning/lightning';

const member_cache = new cacher<`${string}/${string}`, ServerMember>();
const webhook_cache = new cacher<`${string}/${string}`, Webhook>();
const asset_cache = new cacher<string, attachment>(86400000);

export async function fetch_author(msg: ChatMessage, client: Client) {
	try {
		if (!msg.createdByWebhookId) {
			const key = `${msg.serverId}/${msg.createdBy}` as const;
			const author = member_cache.get(key) ?? member_cache.set(
				key,
				(await client.request(
					'get',
					`/servers/${msg.serverId}/members/${msg.createdBy}`,
					undefined,
				) as { member: ServerMember }).member,
			);

			return {
				username: author.nickname ?? author.user.name,
				rawname: author.user.name,
				id: msg.createdBy,
				profile: author.user.avatar,
			};
		} else {
			const key = `${msg.serverId}/${msg.createdByWebhookId}` as const;
			const webhook = webhook_cache.get(key) ?? webhook_cache.set(
				key,
				(await client.request(
					'get',
					`/servers/${msg.serverId}/webhooks/${msg.createdByWebhookId}`,
					undefined,
				)).webhook,
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

async function fetch_attachments(markdown: string[], client: Client) {
	const urls = markdown.map(
		(url) => (url.split('(').pop())?.split(')')[0],
	).filter((i) => i !== undefined);

	const attachments: attachment[] = [];

	for (const url of urls) {
		const cached = asset_cache.get(url);

		if (cached) {
			attachments.push(cached);
		} else {
			try {
				const signed = (await client.request('post', '/url-signatures', {
					urls: [url],
				})).urlSignatures[0];

				if (signed.retryAfter || !signed.signature) continue;

				attachments.push(asset_cache.set(signed.url, {
					name: signed.signature.split('/').pop()?.split('?')[0] ?? 'unknown',
					file: signed.signature,
					size: parseInt(
						(await fetch(signed.signature, {
							method: 'HEAD',
						})).headers.get('Content-Length') ?? '0',
					) / 1048576,
				}));
			} catch {
				continue;
			}
		}
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
	) ?? [];

	content = content?.replaceAll(
		/!\[.*\]\(https:\/\/cdn\.gldcdn\.com\/ContentMediaGenericFiles\/.*\)/gm,
		'',
	)?.replaceAll(/<(:\w+:)\d+>/g, (_, emoji) => emoji);

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
					name: embed.author.name ?? '',
				}
				: undefined,
			image: embed.image
				? {
					...embed.image,
					url: embed.image.url ?? '',
				}
				: undefined,
			thumbnail: embed.thumbnail
				? {
					...embed.thumbnail,
					url: embed.thumbnail.url ?? '',
				}
				: undefined,
			timestamp: embed.timestamp ? Number(embed.timestamp) : undefined,
		})),
		message_id: msg.id,
		plugin: 'bolt-guilded',
		reply_id: msg.replyMessageIds,
		timestamp: Temporal.Instant.from(
			msg.createdAt,
		),
	};
}
