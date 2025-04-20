import type { message } from '@lightning/lightning';
import type { Context } from 'grammy';

const types = [
	'text',
	'dice',
	'location',
	'document',
	'animation',
	'audio',
	'photo',
	'sticker',
	'video',
	'video_note',
	'voice',
	'unsupported',
] as const;

export async function get_incoming(
	ctx: Context,
	proxy: string,
): Promise<message | undefined> {
	const msg = ctx.editedMessage || ctx.msg;
	if (!msg) return;
	const author = await ctx.getAuthor();
	const profile = await ctx.getUserProfilePhotos({ limit: 1 });
	const type = types.find((type) => type in msg) ?? 'unsupported';
	const base: message = {
		author: {
			username: author.user.last_name
				? `${author.user.first_name} ${author.user.last_name}`
				: author.user.first_name,
			rawname: author.user.username || author.user.first_name,
			color: '#24A1DE',
			profile: profile.total_count
				? `${proxy}/${
					(await ctx.api.getFile(profile.photos[0][0].file_id)).file_path
				}`
				: undefined,
			id: author.user.id.toString(),
		},
		channel_id: msg.chat.id.toString(),
		message_id: msg.message_id.toString(),
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			(msg.edit_date || msg.date) * 1000,
		),
		plugin: 'bolt-telegram',
		reply_id: msg.reply_to_message
			? msg.reply_to_message.message_id.toString()
			: undefined,
	};

	switch (type) {
		case 'text':
			return {
				...base,
				content: msg.text,
			};
		case 'dice':
			return {
				...base,
				content: `${msg.dice!.emoji} ${msg.dice!.value}`,
			};
		case 'location':
			return {
				...base,
				content: `https://www.openstreetmap.com/#map=18/${
					msg.location!.latitude
				}/${msg.location!.longitude}`,
			};
		case 'unsupported':
			return;
		default: {
			const file = await ctx.api.getFile(
				(type === 'photo' ? msg.photo!.slice(-1)[0] : msg[type]!).file_id,
			);

			if (!file.file_path) return;

			return {
				...base,
				attachments: [{
					file: `${proxy}/${file.file_path}`,
					name: file.file_path,
					size: (file.file_size ?? 0) / 1048576,
				}],
			};
		}
	}
}
