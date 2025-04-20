import type { bridge_data } from '../database/mod.ts';
import { bridge_settings_list } from '../structures/bridge.ts';
import { log_error } from '../structures/errors.ts';
import type { bridge_channel, command_opts } from '../structures/mod.ts';

export async function create(
	db: bridge_data,
	opts: command_opts,
): Promise<string> {
	const result = await _add(db, opts);

	if (typeof result === 'string') return result;

	const data = {
		name: opts.args.name!,
		channels: [result],
		settings: {
			allow_editing: true,
			allow_everyone: false,
			use_rawname: false,
		},
	};

	try {
		const { id } = await db.create_bridge(data);
		return `Bridge created successfully!\nYou can now join it using \`${opts.prefix}bridge join ${id}\`.\nKeep this ID safe, don't share it with anyone, and delete this message.`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to insert bridge into database',
			extra: data,
		});
	}
}

export async function join(
	db: bridge_data,
	opts: command_opts,
): Promise<string> {
	const result = await _add(db, opts);

	if (typeof result === 'string') return result;

	const target_bridge = await db.get_bridge_by_id(
		opts.args.id!,
	);

	if (!target_bridge) {
		return `Bridge with id \`${opts.args.id}\` not found. Make sure you have the correct id.`;
	}

	target_bridge.channels.push(result);

	try {
		await db.edit_bridge(target_bridge);

		return `Bridge joined successfully!`;
	} catch (e) {
		log_error(e, {
			message: 'Failed to update bridge in database',
			extra: { target_bridge },
		});
	}
}

async function _add(
	db: bridge_data,
	opts: command_opts,
): Promise<string | bridge_channel> {
	const existing_bridge = await db.get_bridge_by_channel(
		opts.channel_id,
	);

	if (existing_bridge) {
		return `You are already in a bridge called \`${existing_bridge.name}\`. You must leave it before being in another bridge. Try using \`${opts.prefix}bridge leave\` or \`${opts.prefix}help\` commands.`;
	}

	try {
		return {
			id: opts.channel_id,
			data: await opts.plugin.setup_channel(opts.channel_id),
			disabled: false,
			plugin: opts.plugin.name,
		};
	} catch (e) {
		log_error(e, {
			message: 'Failed to create bridge using plugin',
			extra: { channel: opts.channel_id, plugin_name: opts.plugin },
		});
	}
}

export async function leave(
	db: bridge_data,
	opts: command_opts,
): Promise<string> {
	const bridge = await db.get_bridge_by_channel(
		opts.channel_id,
	);

	if (!bridge) return `You are not in a bridge`;

	if (opts.args.id !== bridge.id) {
		return `You must provide the bridge id in order to leave this bridge`;
	}

	bridge.channels = bridge.channels.filter((
		ch,
	) => ch.id !== opts.channel_id);

	try {
		await db.edit_bridge(
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

export async function status(
	db: bridge_data,
	opts: command_opts,
): Promise<string> {
	const bridge = await db.get_bridge_by_channel(
		opts.channel_id,
	);

	if (!bridge) return `You are not in a bridge`;

	let str = `Name: \`${bridge.name}\`\n\nChannels:\n`;

	for (const [i, value] of bridge.channels.entries()) {
		str += `${i + 1}. \`${value.id}\` on \`${value.plugin}\`${
			value.disabled ? ' (disabled)' : ''
		}\n`;
	}

	str += `\nSettings:\n`;

	for (
		const [key, value] of Object.entries(bridge.settings).filter(([key]) =>
			bridge_settings_list.includes(key)
		)
	) {
		str += `- \`${key}\` ${value ? '✔' : '❌'}\n`;
	}

	return str;
}

export async function toggle(
	db: bridge_data,
	opts: command_opts,
): Promise<string> {
	const bridge = await db.get_bridge_by_channel(
		opts.channel_id,
	);

	if (!bridge) return `You are not in a bridge`;

	if (!bridge_settings_list.includes(opts.args.setting!)) {
		return `That setting does not exist`;
	}

	const key = opts.args.setting as keyof typeof bridge.settings;

	bridge.settings[key] = !bridge.settings[key];

	try {
		await db.edit_bridge(
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
