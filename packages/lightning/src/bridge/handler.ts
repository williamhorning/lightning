import type { core } from '../core.ts';
import type { bridge_data } from '../database/mod.ts';
import type { bridge_message, bridged_message } from '../structures/bridge.ts';
import { LightningError } from '../structures/errors.ts';
import type { deleted_message, message } from '../structures/messages.ts';

export async function bridge_message(
	core: core,
	bridge_data: bridge_data,
	event: 'create_message' | 'edit_message' | 'delete_message',
	data: message | deleted_message,
) {
	const bridge = event === 'create_message'
		? await bridge_data.get_bridge_by_channel(data.channel_id)
		: await bridge_data.get_message(data.message_id);

	if (!bridge) return;

	// if the channel is disabled, return
	if (
		bridge.channels.some(
			(channel) =>
				channel.id === data.channel_id &&
				channel.plugin === data.plugin &&
				channel.disabled,
		)
	) return;

	// remove ourselves & disabled channels
	const channels = bridge.channels.filter((channel) =>
		(channel.id !== data.channel_id || channel.plugin !== data.plugin) &&
		(!channel.disabled || !channel.data)
	);

	// if there aren't any left, return
	if (channels.length < 1) return;

	const messages: bridged_message[] = [];

	for (const channel of channels) {
		const prior_bridged_ids = event === 'create_message'
			? undefined
			: (bridge as bridge_message).messages.find((i) =>
				i.channel === channel.id && i.plugin === channel.plugin
			);

		if (event !== 'create_message' && !prior_bridged_ids) continue;

		const plugin = core.get_plugin(channel.plugin)!;

		let reply_id: string | undefined;

		if ('reply_id' in data && data.reply_id) {
			try {
				const bridged = await bridge_data.get_message(data.reply_id);

				reply_id = bridged?.messages?.find((message) =>
					message.channel === channel.id && message.plugin === channel.plugin
				)?.id[0];
			} catch {
				reply_id = undefined;
			}
		}

		try {
			let result_ids: string[];

			switch (event) {
				case 'create_message':
				case 'edit_message':
					result_ids = await plugin[event](
						{
							...(data as message),
							reply_id,
							channel_id: channel.id,
							message_id: prior_bridged_ids?.id[0] ?? '',
						},
						{
							channel,
							settings: bridge.settings,
							edit_ids: prior_bridged_ids?.id as string[],
						},
					);
					break;
				case 'delete_message':
					result_ids = await plugin.delete_messages(
						prior_bridged_ids!.id.map((id) => ({
							...(data as deleted_message),
							message_id: id,
							channel_id: channel.id,
						})),
					);
			}

			result_ids.forEach((id) => core.set_handled(channel.plugin, id));

			messages.push({
				id: result_ids,
				channel: channel.id,
				plugin: channel.plugin,
			});
		} catch (e) {
			const err = new LightningError(e, {
				message: `An error occurred while processing a message in the bridge`,
			});

			if (err.disable_channel) {
				new LightningError(
					`disabling channel ${channel.id} in bridge ${bridge.id}`,
					{
						extra: { original_error: err.id },
					},
				);

				await bridge_data.edit_bridge({
					...bridge,
					channels: bridge.channels.map((ch) =>
						ch.id === channel.id && ch.plugin === channel.plugin
							? { ...ch, disabled: true }
							: ch
					),
				});
			}
		}
	}

	await bridge_data[event]({
		...bridge,
		id: data.message_id,
		messages,
		bridge_id: bridge.id,
	});
}
