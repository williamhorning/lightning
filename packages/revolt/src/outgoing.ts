import type { DataMessageSend, SendableEmbed } from '@jersey/revolt-api-types';
import type { Client } from '@jersey/rvapi';
import { LightningError, type message } from '@lightning/lightning';

export async function get_outgoing(
	api: Client,
	message: message,
	masquerade = true,
): Promise<DataMessageSend> {
	const attachments = (await Promise.all(
		message.attachments?.map(async (attachment) => {
			try {
				const file = await (await fetch(attachment.file)).blob();
				if (file.size < 1) return;
				return await api.media.upload_file('attachments', file);
			} catch (e) {
				new LightningError(e, {
					message: 'Failed to upload attachment',
					extra: { original: e },
				});

				return;
			}
		}) ?? [],
	)).filter((i) => i !== undefined);

	if (
		(!message.content || message.content.length < 1) &&
		(!message.embeds || message.embeds.length < 1) &&
		(!attachments || attachments.length < 1)
	) {
		message.content = '*empty message*';
	}

	return {
		attachments,
		content: (message.content?.length ?? 0) > 2000
			? `${message.content?.substring(0, 1997)}...`
			: message.content,
		embeds: message.embeds?.map((embed) => {
			const data: SendableEmbed = {
				icon_url: embed.author?.icon_url,
				url: embed.url,
				title: embed.title,
				description: embed.description ?? '',
				media: embed.image?.url,
				colour: embed.color
					? `#${embed.color.toString(16).padStart(6, '0')}`
					: undefined,
			};

			for (const field of embed.fields ?? []) {
				data.description += `\n\n**${field.name}**\n${field.value}`;
			}

			if (data.description?.length === 0) data.description = undefined;

			return data;
		}),
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
