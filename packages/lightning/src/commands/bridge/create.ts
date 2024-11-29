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
		await opts.lightning.data.create_bridge(bridge_data);
		return `Bridge created successfully! You can now join it using \`${opts.lightning.config.prefix}join ${result.id}\`. Keep this id safe, don't share it with anyone, and delete this message.`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to insert bridge into database',
			extra: bridge_data,
		});
	}
}
