import type { EmbedPayload } from '@guildedjs/api';
import type { embed } from '@jersey/lightning';
import type { Embed } from 'guilded.js';

export function get_lightning_embeds(embeds?: Embed[]): embed[] | undefined {
	if (!embeds) return;

	return embeds.map((embed) => ({
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
	}));
}

export function get_guilded_embeds(
	embeds?: embed[],
): EmbedPayload[] | undefined {
	if (!embeds) return;

	return embeds.flatMap<EmbedPayload>((embed) => {
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
	});
}
