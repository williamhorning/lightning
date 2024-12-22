import type { embed } from '@jersey/lightning';
import type { Embed } from 'guilded.js';

export function map_embed(embed: Embed): embed {
	return {
		...embed,
		author: embed.author
			? {
				name: embed.author.name || 'embed author',
				icon_url: embed.author.iconURL || undefined,
				url: embed.author.url || undefined,
			}
			: undefined,
		image: embed.image || undefined,
		thumbnail: embed.thumbnail || undefined,
		timestamp: embed.timestamp ? Number(embed.timestamp) : undefined,
		color: embed.color || undefined,
		description: embed.description || undefined,
		fields: embed.fields.map((i) => {
			return {
				...i,
				inline: i.inline || undefined,
			};
		}),
		footer: embed.footer || undefined,
		title: embed.title || undefined,
		url: embed.url || undefined,
		video: embed.video || undefined,
	};
}
