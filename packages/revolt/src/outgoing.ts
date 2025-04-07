import type { Client } from '@jersey/rvapi';
import type { DataMessageSend, SendableEmbed } from '@jersey/revolt-api-types';
import {
	type attachment,
	LightningError,
	type message,
} from '@jersey/lightning';

async function uploadAttachments(
	api: Client,
	attachments?: attachment[],
): Promise<string[] | undefined> {
	if (!attachments) return undefined;

	return (await Promise.all(
		attachments.map(async (attachment) => {
			try {
				return await api.media.upload_file(
					'attachments',
					await (await fetch(attachment.file)).blob(),
				);
			} catch (e) {
				new LightningError(e, {
					message: 'Failed to upload attachment',
					extra: { original: e },
				});

				return;
			}
		}),
	)).filter((i) => i !== undefined);
}

export async function getOutgoingMessage(
	api: Client,
	message: message,
	masquerade = true,
): Promise<DataMessageSend> {
	const attachments = await uploadAttachments(api, message.attachments);

	if (
		(!message.content || message.content.length < 1) &&
		(!message.embeds || message.embeds.length < 1) &&
		(!attachments || attachments.length < 1)
	) {
		message.content = '*empty message*';
	}

	return {
		attachments,
		content: (message.content?.length || 0) > 2000
			? `${message.content?.substring(0, 1997)}...`
			: message.content,
		embeds: message.embeds?.map((embed) => {
			const data: SendableEmbed = {
				icon_url: embed.author?.icon_url,
				url: embed.url,
				title: embed.title,
				description: embed.description ?? '',
				media: embed.image?.url,
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
