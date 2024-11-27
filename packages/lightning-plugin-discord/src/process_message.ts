import type { API } from '@discordjs/core';
import type { message_options } from '@jersey/lightning';
import { to_discord } from './discord.ts';

export async function process_message(api: API, opts: message_options) {
	const data = opts.channel.data as { token: string; id: string };

	if (opts.action !== 'delete') {
		let replied_message;

		if (opts.reply_id) {
			try {
				replied_message = await api.channels
					.getMessage(opts.channel.id, opts.reply_id);
			} catch {
				// safe to ignore
			}
		}

		const msg = await to_discord(
			opts.message,
			replied_message,
		);

		try {
			let wh;

			if (opts.action === 'edit') {
				wh = await api.webhooks.editMessage(
					data.id,
					data.token,
					opts.edit_id[0],
					msg,
				);
			} else {
				wh = await api.webhooks.execute(
					data.id,
					data.token,
					msg,
				);
			}

			return {
				id: [wh.id],
				channel: opts.channel,
			};
		} catch (e) {
			if (
				(e as { status: number }).status === 404 &&
				opts.action !== 'edit'
			) {
				return {
					channel: opts.channel,
					error: e as Error,
					disable: true,
				};
			} else {
				throw e;
			}
		}
	} else {
		await api.webhooks.deleteMessage(
			data.id,
			data.token,
			opts.edit_id[0],
		);

		return {
			id: opts.edit_id,
			channel: opts.channel,
		};
	}
}
