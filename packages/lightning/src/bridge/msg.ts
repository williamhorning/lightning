import type { lightning } from '../lightning.ts';
import { log_error } from '../errors.ts';
import type {
	deleted_message,
	message,
	unprocessed_message,
} from '../messages.ts';
import type {
	bridge,
	bridge_channel,
	bridge_message,
	bridged_message,
} from './data.ts';

export async function handle_message(
	core: lightning,
	msg: message | deleted_message,
	type: 'create' | 'edit' | 'delete',
): Promise<void> {
	const br = type === 'create'
		? await core.data.get_bridge_by_channel(msg.channel)
		: await core.data.get_bridge_message(msg.id);

	if (!br) return;

	if (type !== 'create' && br.settings.allow_editing !== true) return;

	if (
		br.channels.find((i) =>
			i.id === msg.channel && i.plugin === msg.plugin && i.disabled
		)
	) return;

	const channels = br.channels.filter(
		(i) => i.id !== msg.channel || i.plugin !== msg.plugin,
	);

	if (channels.length < 1) return;

	const messages = [] as bridged_message[];

	for (const ch of channels) {
		if (!ch.data || ch.disabled) continue;

		const bridged_id = (br as Partial<bridge_message>).messages?.find((i) =>
			i.channel === ch.id && i.plugin === ch.plugin
		);

		if ((type !== 'create' && !bridged_id)) {
			continue;
		}

		const plugin = core.plugins.get(ch.plugin);

		if (!plugin) {
			await disable_channel(
				ch,
				br,
				core,
				(await log_error(
					new Error(`plugin ${ch.plugin} doesn't exist`),
					{ channel: ch, bridged_id },
				)).cause,
			);

			continue;
		}

		const reply_id = await get_reply_id(core, msg as message, ch);

		let res;

		try {
			res = await plugin.process_message({
				action: type as 'edit',
				channel: ch,
				message: msg as message,
				edit_id: bridged_id?.id as string[],
				reply_id,
			});

			if (res.error) throw res.error;
		} catch (e) {
			if (type === 'delete') continue;

			if ((res as unprocessed_message).disable) {
				await disable_channel(ch, br, core, e);

				continue;
			}

			const err = await log_error(e, {
				channel: ch,
				bridged_id,
				message: msg,
			});

			try {
				res = await plugin.process_message({
					action: type as 'edit',
					channel: ch,
					message: err.message as message,
					edit_id: bridged_id?.id as string[],
					reply_id,
				});

				if (res.error) throw res.error;
			} catch (e) {
				await log_error(
					new Error(`failed to log error`, { cause: e }),
					{ channel: ch, bridged_id, original_id: err.id },
				);

				continue;
			}
		}

		for (const id of res.id) {
			sessionStorage.setItem(`${ch.plugin}-${id}`, '1');
		}

		messages.push({
			id: res.id,
			channel: ch.id,
			plugin: ch.plugin,
		});
	}

	const method = type === 'create' ? 'new' : 'update';

	await core.data[`${method}_bridge_message`]({
		...br,
		id: msg.id,
		messages,
		bridge_id: br.id,
	});
}

async function disable_channel(
	channel: bridge_channel,
	bridge: bridge | bridge_message,
	core: lightning,
	error: unknown,
): Promise<void> {
	await log_error(error, { channel, bridge });

	await core.data.update_bridge({
		id: "bridge_id" in bridge ? bridge.bridge_id : bridge.id,
		channels: bridge.channels.map((i) =>
			i.id === channel.id && i.plugin === channel.plugin
				? { ...i, disabled: true, data: error }
				: i
		),
		settings: bridge.settings
	});
}

async function get_reply_id(
	core: lightning,
	msg: message,
	channel: bridge_channel,
): Promise<string | undefined> {
	if (msg.reply_id) {
		try {
			const bridged = await core.data.get_bridge_message(msg.reply_id);

			if (!bridged) return;

			const br_ch = bridged.channels.find((i) =>
				i.id === channel.id && i.plugin === channel.plugin
			);

			if (!br_ch) return;

			return br_ch.id;
		} catch {
			return;
		}
	}
}
