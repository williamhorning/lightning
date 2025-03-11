import type { DataMessageSend, SendableEmbed } from '@jersey/revolt-api-types';
import {
	type attachment,
	type embed,
	LightningError,
	type message,
} from '@jersey/lightning';
import type { Client } from '@jersey/rvapi';

export async function to_revolt(
	api: Client,
	message: message,
	masquerade = true,
): Promise<DataMessageSend> {
	const attachments = await upload_attachments(api, message.attachments);
	const embeds = map_embeds(message.embeds);

	if (
		(!message.content || message.content.length < 1) &&
		(!embeds || embeds.length < 1) &&
		(!attachments || attachments.length < 1)
	) {
		message.content = '*empty message*';
	}

	return {
		attachments,
		content: (message.content?.length || 0) > 2000
			? `${message.content?.substring(0, 1997)}...`
			: message.content,
		embeds,
		replies: message.reply_id
			? [{ id: message.reply_id, mention: true }]
			: undefined,
		masquerade: masquerade
			? {
				name: message.author.username,
				avatar: message.author.profile,
				colour: message.author.color,
			}
			: undefined,
	};
}

function map_embeds(embeds?: embed[]): SendableEmbed[] | undefined {
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

async function upload_attachments(api: Client, attachments?: attachment[]) {
	if (!attachments) return undefined;

	return (await Promise.all(
		attachments.map(async (attachment) =>
			api.media.upload_file(
				'attachments',
				await (await fetch(attachment.file)).blob(),
			)
				.then((id) => [id])
				.catch((e) => {
					new LightningError(e, {
						message: 'Failed to upload attachment',
						extra: { original: e },
					});
					return [] as string[];
				})
		),
	)).flat();
}
