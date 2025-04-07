import type { command_opts } from '../../structures/commands.ts';
import { log_error } from '../../structures/errors.ts';
import { bridge_add_common } from './_internal.ts';

export async function create(
	opts: command_opts,
): Promise<string> {
	const result = await bridge_add_common(opts);

	if (typeof result === 'string') return result;

	const bridge_data = {
		name: opts.args.name,
		channels: [result],
		settings: {
			allow_editing: true,
			allow_everyone: false,
			use_rawname: false,
		},
	};

	try {
		const { id } = await opts.bridge_data.create_bridge(bridge_data);
		return `Bridge created successfully!\nYou can now join it using \`${opts.prefix}bridge join ${id}\`.\nKeep this ID safe, don't share it with anyone, and delete this message.`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to insert bridge into database',
			extra: bridge_data,
		});
	}
}
