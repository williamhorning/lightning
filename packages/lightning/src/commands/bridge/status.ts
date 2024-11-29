import type { command_opts } from '../../structures/commands.ts';

export async function status(opts: command_opts): Promise<string> {
	const bridge = await opts.lightning.data.get_bridge_by_channel(
		opts.channel,
	);

	if (!bridge) return `You are not in a bridge`;

	let str = `Name: \`${bridge.name}\`\n\nChannels:\n`;

	for (const [i, value] of bridge.channels.entries()) {
		str += `${i + 1}. \`${value.id}\` on \`${value.plugin}\`\n`;
	}

	str += `\nSettings:\n`;

	for (const [key, value] of Object.entries(bridge.settings)) {
		str += `- \`${key}\` ${value ? '✔' : '❌'}\n`;
	}

	return str;
}
