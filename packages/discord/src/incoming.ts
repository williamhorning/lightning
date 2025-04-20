import type {
	API,
	APIInteraction,
	APIStickerItem,
	GatewayMessageDeleteDispatchData,
	GatewayMessageUpdateDispatchData,
	ToEventProps,
} from '@discordjs/core';
import type {
	attachment,
	create_command,
	deleted_message,
	message,
} from '@lightning/lightning';
import { get_outgoing_message } from './outgoing.ts';

export function get_deleted_message(
	data: GatewayMessageDeleteDispatchData,
): deleted_message {
	return {
		message_id: data.id,
		channel_id: data.channel_id,
		plugin: 'bolt-discord',
		timestamp: Temporal.Now.instant(),
	};
}

async function fetch_author(api: API, data: GatewayMessageUpdateDispatchData) {
	let profile = data.author.avatar !== null
		? `https://cdn.discordapp.com/avatars/${data.author.id}/${data.author.avatar}.png`
		: `https://cdn.discordapp.com/embed/avatars/${
			Number(BigInt(data.author.id) >> 22n) % 6
		}.png`;

	let username = data.author.global_name || data.author.username;

	if (data.guild_id) {
		try {
			const member = data.member || await api.guilds.getMember(
				data.guild_id,
				data.author.id,
			);

			if (member.avatar) {
				profile =
					`https://cdn.discordapp.com/guilds/${data.guild_id}/users/${data.author.id}/avatars/${member.avatar}.png`;
			}

			if (member.nick) username = member.nick;
		} catch {
			// safe to ignore, we already have a name and avatar
		}
	}

	return { profile, username };
}

async function fetch_stickers(
	stickers: APIStickerItem[],
): Promise<attachment[]> {
	return (await Promise.allSettled(stickers.map(async (sticker) => {
		let type;

		if (sticker.format_type === 1) type = 'png';
		if (sticker.format_type === 2) type = 'apng';
		if (sticker.format_type === 3) type = 'lottie';
		if (sticker.format_type === 4) type = 'gif';

		const url = `https://media.discordapp.net/stickers/${sticker.id}.${type}`;

		const request = await fetch(url, { method: 'HEAD' });

		return {
			file: url,
			alt: sticker.name,
			name: `${sticker.name}.${type}`,
			size: parseInt(request.headers.get('Content-Length') ?? '0', 10) /
				1048576,
		};
	}))).flatMap((i) => i.status === 'fulfilled' ? i.value : []);
}

export async function get_incoming_message(
	{ api, data }: { api: API; data: GatewayMessageUpdateDispatchData },
): Promise<message | undefined> {
	// normal messages, replies, and user joins
	if (
		data.type !== 0 &&
		data.type !== 7 &&
		data.type !== 19 &&
		data.type !== 20 &&
		data.type !== 23
	) {
		return;
	}

	const message: message = {
		attachments: [
			...data.attachments?.map(
				(i: typeof data['attachments'][0]) => {
					return {
						file: i.url,
						alt: i.description,
						name: i.filename,
						size: i.size / 1048576, // bytes -> MiB
					};
				},
			),
			...data.sticker_items ? await fetch_stickers(data.sticker_items) : [],
		],
		author: {
			rawname: data.author.username,
			id: data.author.id,
			color: '#5865F2',
			...await fetch_author(api, data),
		},
		channel_id: data.channel_id,
		content: data.type === 7
			? '*joined on discord*'
			: (data.flags || 0) & 128
			? '*loading...*'
			: data.content,
		embeds: data.embeds.map((i) => ({
			...i,
			timestamp: i.timestamp ? Number(i.timestamp) : undefined,
			video: i.video ? { ...i.video, url: i.video.url ?? '' } : undefined,
		})),
		message_id: data.id,
		plugin: 'bolt-discord',
		reply_id: data.message_reference &&
				data.message_reference.type === 0
			? data.message_reference.message_id
			: undefined,
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			Number(BigInt(data.id) >> 22n) + 1420070400000,
		),
	};

	return message;
}

export function get_incoming_command(
	interaction: ToEventProps<APIInteraction>,
): create_command | undefined {
	if (interaction.data.type !== 2 || interaction.data.data.type !== 1) return;

	const args: Record<string, string> = {};
	let subcommand: string | undefined;

	for (const option of interaction.data.data.options || []) {
		if (option.type === 1) {
			subcommand = option.name;
			for (const suboption of option.options ?? []) {
				if (suboption.type === 3) {
					args[suboption.name] = suboption.value;
				}
			}
		} else if (option.type === 3) {
			args[option.name] = option.value;
		}
	}

	return {
		args,
		channel_id: interaction.data.channel.id,
		command: interaction.data.data.name,
		message_id: interaction.data.id,
		prefix: '/',
		plugin: 'bolt-discord',
		reply: async (msg) =>
			await interaction.api.interactions.reply(
				interaction.data.id,
				interaction.data.token,
				await get_outgoing_message(msg, interaction.api, false, false),
			),
		subcommand,
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			Number(BigInt(interaction.data.id) >> 22n) + 1420070400000,
		),
	};
}
