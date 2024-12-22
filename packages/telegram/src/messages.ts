import type { message } from '@jersey/lightning';
import type { Context } from 'grammy';
import type { Message } from 'grammy/types';
import { default as convert_md } from 'telegramify-markdown';
import type { telegram_config } from './mod.ts';

export function from_lightning(msg: message) {
	let content = `${msg.author.username} » ${msg.content || '_no content_'}`;

	if ((msg.embeds?.length ?? 0) > 0) {
		content = `${content}\n_this message has embeds_`;
	}

	const messages = [{
		function: 'sendMessage',
		value: convert_md(content, 'escape'),
	}] as { function: 'sendMessage' | 'sendDocument'; value: string }[];

	for (const attachment of (msg.attachments ?? [])) {
		messages.push({
			function: 'sendDocument',
			value: attachment.file,
		});
	}

	return messages;
}

export async function from_telegram(
	ctx: Context,
	cfg: telegram_config,
): Promise<message | undefined> {
	const msg = (ctx.editedMessage || ctx.msg) as Message | undefined;
	if (!msg) return;
	const type = get_message_type(msg);
	const base = await get_base_msg(ctx, msg, cfg);

	switch (type) {
		case 'text':
			return {
				...base,
				content: msg.text,
			};
		case 'document':
		case 'animation':
		case 'audio':
		case 'photo':
		case 'sticker':
		case 'video':
		case 'video_note':
		case 'voice': {
			const file_obj = type === 'photo' ? msg.photo!.slice(-1)[0] : msg[type]!;
			const file = await ctx.api.getFile(file_obj.file_id);
			if (!file.file_path) return;
			return {
				...base,
				attachments: [{
					file: `${cfg.plugin_url}/${file.file_path}`,
					size: (file.file_size ?? 0) / 1000000,
				}],
			};
		}
		case 'dice':
			return {
				...base,
				content: `${msg.dice!.emoji} ${msg.dice!.value}`,
			};
		case 'location':
			return {
				...base,
				content: `https://www.google.com/maps/search/?api=1&query=${
					msg.location!.latitude
				}%2C${msg.location!.longitude}`,
			};
		case 'unsupported':
			return;
	}
}

function get_message_type(msg: Message) {
	if ('text' in msg) return 'text';
	if ('document' in msg) return 'document';
	if ('animation' in msg) return 'animation';
	if ('audio' in msg) return 'audio';
	if ('photo' in msg) return 'photo';
	if ('sticker' in msg) return 'sticker';
	if ('video' in msg) return 'video';
	if ('video_note' in msg) return 'video_note';
	if ('voice' in msg) return 'voice';
	if ('dice' in msg) return 'dice';
	if ('location' in msg) return 'location';
	return 'unsupported';
}

async function get_base_msg(
	ctx: Context,
	msg: Message,
	cfg: telegram_config,
): Promise<message> {
	const author = await ctx.getAuthor();
	const pfps = await ctx.getUserProfilePhotos({ limit: 1 });
	return {
		author: {
			username: author.user.last_name
				? `${author.user.first_name} ${author.user.last_name}`
				: author.user.first_name,
			rawname: author.user.username || author.user.first_name,
			color: '#24A1DE',
			profile: pfps.total_count
				? `${cfg.plugin_url}/${
					(await ctx.api.getFile(pfps.photos[0][0].file_id)).file_path
				}`
				: undefined,
			id: author.user.id.toString(),
		},
		channel: msg.chat.id.toString(),
		id: msg.message_id.toString(),
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			(msg.edit_date || msg.date) * 1000,
		),
		plugin: 'bolt-telegram',
		reply: async (lmsg) => {
			for (const m of from_lightning(lmsg)) {
				await ctx.api[m.function](msg.chat.id.toString(), m.value, {
					reply_parameters: {
						message_id: msg.message_id,
					},
					parse_mode: 'MarkdownV2',
				});
			}
		},
		reply_id: msg.reply_to_message
			? msg.reply_to_message.message_id.toString()
			: undefined,
	};
}
