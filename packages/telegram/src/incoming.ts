import type { command, create_command, message } from '@lightning/lightning';
import type { CommandContext, Context } from 'grammy';
import { get_outgoing } from './outgoing.ts';

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
] as const;

export async function get_incoming(
	ctx: Context,
	proxy: string,
): Promise<message | undefined> {
	const msg = ctx.editedMessage ?? ctx.msg;
	if (!msg) return;
	const author = await ctx.getAuthor();
	const profile = await ctx.getUserProfilePhotos({ limit: 1 });
	const type = types.find((type) => type in msg) ?? 'unsupported';
	const base: message = {
		author: {
			username: author.user.last_name
				? `${author.user.first_name} ${author.user.last_name}`
				: author.user.first_name,
			rawname: author.user.username ?? author.user.first_name,
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
			(msg.edit_date ?? msg.date) * 1000,
		),
		plugin: 'bolt-telegram',
		reply_id: msg.reply_to_message
			? msg.reply_to_message.message_id.toString()
			: undefined,
	};

	if (type === 'unsupported') return;
	if (type === 'text') return { ...base, content: msg.text };
	if (type === 'dice') {
		return {
			...base,
			content: `${msg.dice!.emoji} ${msg.dice!.value}`,
		};
	}
	if (type === 'location') {
		return {
			...base,
			content: `https://www.openstreetmap.com/#map=18/${
				msg.location!.latitude
			}/${msg.location!.longitude}`,
		};
	}

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

export function get_command(
	ctx: CommandContext<Context>,
	cmd: command,
): create_command {
	return {
		channel_id: ctx.chat.id.toString(),
		command: cmd.name,
		message_id: ctx.msgId.toString(),
		timestamp: Temporal.Instant.fromEpochMilliseconds(ctx.msg.date * 1000),
		plugin: 'bolt-telegram',
		prefix: '/',
		args: {},
		rest: cmd.subcommands
			? ctx.match.split(' ').slice(1)
			: ctx.match.split(' '),
		subcommand: cmd.subcommands ? ctx.match.split(' ')[0] : undefined,
		reply: async (message: message) => {
			for (const msg of get_outgoing(message, false)) {
				await ctx.api[msg.function](
					ctx.chat.id,
					msg.value,
					{
						reply_parameters: { message_id: ctx.msgId },
						parse_mode: 'MarkdownV2',
					},
				);
			}
		},
	};
}
