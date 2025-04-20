import type { Client } from '@jersey/guildapi';
import type {
	ChatMessage,
	ServerMember,
	Webhook,
} from '@jersey/guilded-api-types';
import type { attachment, message } from '@lightning/lightning';

class cacher<K extends string, V> {
	private map = new Map<K, {
		value: V;
		expiry: number;
	}>();
	public expiry = 30000;
	get(key: K): V | undefined {
		const time = Temporal.Now.instant().epochMilliseconds;
		const v = this.map.get(key);

		if (v && v.expiry >= time) return v.value;
	}
	set(key: K, val: V): V {
		const time = Temporal.Now.instant().epochMilliseconds;
		this.map.set(key, { value: val, expiry: time + this.expiry });
		return val;
	}
}

const member_cache = new cacher<`${string}/${string}`, ServerMember>();
const webhook_cache = new cacher<`${string}/${string}`, Webhook>();
const asset_cache = new cacher<string, attachment>();
asset_cache.expiry = 86400000; // 1 day!

export async function fetch_author(msg: ChatMessage, client: Client) {
	try {
		if (!msg.createdByWebhookId) {
			const author = member_cache.get(`${msg.serverId}/${msg.createdBy}`) ??
				member_cache.set(
					`${msg.serverId}/${msg.createdBy}`,
					(await client.request(
						'get',
						`/servers/${msg.serverId}/members/${msg.createdBy}`,
						undefined,
					) as { member: ServerMember }).member,
				);

			return {
				username: author.nickname || author.user.name,
				rawname: author.user.name,
				id: msg.createdBy,
				profile: author.user.avatar || undefined,
			};
		} else {
			const webhook = webhook_cache.get(
				`${msg.serverId}/${msg.createdByWebhookId}`,
			) ??
				webhook_cache.set(
					`${msg.serverId}/${msg.createdByWebhookId}`,
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
					name: signed.signature.split('/').pop()?.split('?')[0] || 'unknown',
					file: signed.signature,
					size: parseInt(
						(await fetch(signed.signature, {
							method: 'HEAD',
						})).headers.get('Content-Length') || '0',
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
