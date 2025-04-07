import { log_error } from '../../structures/errors.ts';
import type { bridge_channel, command_opts } from '../../structures/mod.ts';

export async function bridge_add_common(
	opts: command_opts,
): Promise<string | bridge_channel> {
	const existing_bridge = await opts.bridge_data.get_bridge_by_channel(
		opts.channel_id,
	);

	if (existing_bridge) {
		return `You are already in a bridge called \`${existing_bridge.name}\`. You must leave it before being in another bridge. Try using \`${opts.prefix}bridge leave\` or \`${opts.prefix}help\` commands.`;
	}

	const plugin = opts.plugins.get(opts.plugin);

	if (!plugin) {
		log_error('Internal error: platform support not found', {
			extra: { plugin: opts.plugin },
		});
	}

	let bridge_data;

	try {
		bridge_data = await plugin.setup_channel(opts.channel_id);
	} catch (e) {
		log_error(e, {
			message: 'Failed to create bridge using plugin',
			extra: { channel: opts.channel_id, plugin_name: opts.plugin },
		});
	}

	return {
		id: opts.channel_id,
		data: bridge_data,
		disabled: false,
		plugin: opts.plugin,
	};
}
