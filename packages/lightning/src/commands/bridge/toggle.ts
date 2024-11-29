import type { command_opts } from '../../structures/commands.ts';
import { log_error } from '../../structures/errors.ts';
import { bridge_settings_list } from '../../structures/bridge.ts';

export async function toggle(opts: command_opts): Promise<string> {
	const bridge = await opts.lightning.data.get_bridge_by_channel(
		opts.channel,
	);

	if (!bridge) return `You are not in a bridge`;

	if (!bridge_settings_list.includes(opts.args.setting)) {
		return `That setting does not exist`;
	}

	const key = opts.args.setting as keyof typeof bridge.settings;

	bridge.settings[key] = !bridge.settings[key];

	try {
		await opts.lightning.data.edit_bridge(
			bridge,
		);
		return `Bridge settings updated successfully`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to update bridge in database',
			extra: { bridge },
		});
	}
}
