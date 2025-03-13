import type { SendableEmbed } from '@jersey/revolt-api-types';
import type { embed } from '@jersey/lightning';

export function get_revolt_embeds(
	embeds?: embed[],
): SendableEmbed[] | undefined {
	if (!embeds) return undefined;

	return embeds.map((embed) => {
		const data: SendableEmbed = {
			icon_url: embed.author?.icon_url ?? null,
			url: embed.url ?? null,
			title: embed.title ?? null,
			description: embed.description ?? '',
			media: embed.image?.url ?? null,
			colour: embed.color ? `#${embed.color.toString(16)}` : null,
		};

		if (embed.fields) {
			for (const field of embed.fields) {
				data.description += `\n\n**${field.name}**\n${field.value}`;
			}
		}

		if (data.description?.length === 0) {
			data.description = null;
		}

		return data;
	});
}
