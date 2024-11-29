import type { command_opts } from '../../structures/commands.ts';
import { log_error } from '../../structures/errors.ts';
import { bridge_add_common } from './_internal.ts';

export async function join(
	opts: command_opts,
): Promise<string> {
	const result = await bridge_add_common(opts);

	if (typeof result === 'string') return result;

	const target_bridge = await opts.lightning.data.get_bridge_by_id(
		opts.args.id,
	);

	if (!target_bridge) {
		return `Bridge with id \`${opts.args.id}\` not found. Make sure you have the correct id.`;
	}

	target_bridge.channels.push(result);

	try {
		await opts.lightning.data.edit_bridge(target_bridge);

		return `Bridge joined successfully!`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to update bridge in database',
			extra: { target_bridge },
		});
	}
}
