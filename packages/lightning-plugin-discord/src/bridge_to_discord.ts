import type { create_opts, delete_opts, edit_opts } from '@jersey/lightning';
import { message_to_discord } from './discord_message/mod.ts';
import { error_handler } from './error_handler.ts';
import type { API } from '@discordjs/core';

type data = { id: string; token: string };

export async function setup_bridge(api: API, channel: string) {
	try {
		const { id, token } = await api.channels.createWebhook(
			channel,
			{
				name: 'lightning bridge',
			},
		);

		return { id, token };
	} catch (e) {
		return error_handler(e, channel, 'setting up channel');
	}
}

export async function create_message(api: API, opts: create_opts) {
	const data = opts.channel.data as data;
	const transformed = await message_to_discord(
		opts.msg,
		api,
		opts.channel.id,
		opts.reply_id,
	);

	try {
		const res = await api.webhooks.execute(
			data.id,
			data.token,
			transformed,
		);

		return [res.id];
	} catch (e) {
		return error_handler(e, opts.channel.id, 'creating message');
	}
}

export async function edit_message(api: API, opts: edit_opts) {
	const data = opts.channel.data as data;
	const transformed = await message_to_discord(
		opts.msg,
		api,
		opts.channel.id,
		opts.reply_id,
	);

	try {
		await api.webhooks.editMessage(
			data.id,
			data.token,
			opts.edit_ids[0],
			transformed,
		);

		return opts.edit_ids;
	} catch (e) {
		return error_handler(e, opts.channel.id, 'editing message');
	}
}

export async function delete_message(api: API, opts: delete_opts) {
	const data = opts.channel.data as data;

	try {
		await api.webhooks.deleteMessage(
			data.id,
			data.token,
			opts.edit_ids[0],
		);

		return opts.edit_ids;
	} catch (e) {
		return error_handler(e, opts.channel.id, 'editing message');
	}
}
