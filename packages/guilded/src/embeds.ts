import type { ChatEmbed } from '@jersey/guilded-api-types';
import type { embed } from '@jersey/lightning';

export function get_lightning_embeds(
	embeds?: ChatEmbed[],
): embed[] | undefined {
	if (!embeds) return;

	return embeds.map((embed) => ({
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
	}));
}

export function get_guilded_embeds(
	embeds?: embed[],
): ChatEmbed[] | undefined {
	if (!embeds) return;

	return embeds.map((i) => {
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
			timestamp: i.timestamp ? String(i.timestamp) : undefined,
		};
	});
}
