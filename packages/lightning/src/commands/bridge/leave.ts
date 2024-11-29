import type { command_opts } from '../../structures/commands.ts';
import { log_error } from '../../structures/errors.ts';

export async function leave(opts: command_opts): Promise<string> {
	const bridge = await opts.lightning.data.get_bridge_by_channel(
		opts.channel,
	);

	if (!bridge) return `You are not in a bridge`;

	bridge.channels = bridge.channels.filter((
		ch,
	) => ch.id !== opts.channel);

	try {
		await opts.lightning.data.edit_bridge(
			bridge,
		);
		return `Bridge left successfully`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to update bridge in database',
			extra: { bridge },
		});
	}
}
