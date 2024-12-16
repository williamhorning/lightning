import type { lightning } from './lightning.ts';
import { LightningError } from './structures/errors.ts';
import type {
	bridge,
	bridge_channel,
	bridge_message,
	bridged_message,
	deleted_message,
	message,
} from './structures/mod.ts';

export async function bridge_message(
	lightning: lightning,
	event: 'create_message' | 'edit_message' | 'delete_message',
	data: message | deleted_message,
) {
	// get the bridge and return if it doesn't exist
	let bridge;

	if (event === 'create_message') {
		bridge = await lightning.data.get_bridge_by_channel(data.channel);
	} else {
		bridge = await lightning.data.get_message(data.id);
	}

	if (!bridge) return;

	// handle bridge settings
	if (event !== 'create_message' && bridge.settings.allow_editing !== true) {
		return;
	}

	if (bridge.settings.use_rawname && 'author' in data) {
		data.author.username = data.author.rawname;
	}

	// if the channel this event is from is disabled, return
	if (
		bridge.channels.find((channel) =>
			channel.id === data.channel && channel.plugin === data.plugin &&
			channel.disabled
		)
	) return;

	// filter out the channel this event is from and any disabled channels
	const channels = bridge.channels.filter(
		(i) => i.id !== data.channel || i.plugin !== data.plugin,
	).filter((i) => !i.disabled || !i.data);

	// if there are no more channels, return
	if (channels.length < 1) return;

	const messages = [] as bridged_message[];

	for (const channel of channels) {
		let prior_bridged_ids;

		if (event !== 'create_message') {
			prior_bridged_ids = (bridge as bridge_message).messages?.find((i) =>
				i.channel === channel.id && i.plugin === channel.plugin
			);

			if (!prior_bridged_ids) continue; // the message wasn't bridged previously
		}

		const plugin = lightning.plugins.get(channel.plugin);

		if (!plugin) {
			await disable_channel(
				channel,
				bridge,
				new LightningError(`plugin ${channel.plugin} doesn't exist`),
				lightning,
			);
			continue;
		}

		const reply_id = await get_reply_id(lightning, data, channel);

		let result_ids: string[];

		try {
			result_ids = await plugin[event]({
				channel,
				settings: bridge.settings,
				reply_id,
				edit_ids: prior_bridged_ids?.id as string[],
				msg: data as message,
			});
		} catch (e) {
			if (e instanceof LightningError && e.disable_channel) {
				await disable_channel(channel, bridge, e, lightning);
				continue;
			}

			// try sending an error message

			const err = e instanceof LightningError ? e : new LightningError(e, {
				message: `An error occurred while processing a message in the bridge.`,
			});

			try {
				result_ids = await plugin[event]({
					channel,
					settings: bridge.settings,
					reply_id,
					edit_ids: prior_bridged_ids?.id as string[],
					msg: err.msg,
				});
			} catch (e) {
				new LightningError(e, {
					message: `Failed to log error message in bridge`,
					extra: { channel, original_error: err.id },
				});

				continue;
			}
		}

		for (const result_id of result_ids) {
			sessionStorage.setItem(`${channel.plugin}-${result_id}`, '1');
		}

		messages.push({
			id: result_ids,
			channel: channel.id,
			plugin: channel.plugin,
		});
	}

	await lightning.data[event]({
		...bridge,
		id: data.id,
		messages,
		bridge_id: bridge.id,
	});
}

async function get_reply_id(
	core: lightning,
	msg: message | deleted_message,
	channel: bridge_channel,
): Promise<string | undefined> {
	if ('reply_id' in msg && msg.reply_id) {
		try {
			const bridged = await core.data.get_message(msg.reply_id);

			const bridged_message = bridged?.messages?.find((i) =>
				i.channel === channel.id && i.plugin === channel.plugin
			);

			return bridged_message?.id[0];
		} catch {
			return;
		}
	}
}

async function disable_channel(
	channel: bridge_channel,
	bridge: bridge | bridge_message,
	error: LightningError,
	lightning: lightning,
) {
	new LightningError(
		`disabling channel ${channel.id} in bridge ${bridge.id}`,
		{
			extra: { original_error: error.id },
		},
	);

	await lightning.data.edit_bridge({
		id: 'bridge_id' in bridge ? bridge.bridge_id : bridge.id,
		channels: bridge.channels.map((i) =>
			i.id === channel.id && i.plugin === channel.plugin
				? { ...i, disabled: true, data: error }
				: i
		),
		settings: bridge.settings,
	});
}
