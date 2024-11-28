import { create_message, type message } from '../../messages.ts';
import type { command_execute_options } from '../mod.ts';

export async function status(opts: command_execute_options): Promise<message> {
    const bridge = await opts.lightning.data.get_bridge_by_channel(
        opts.channel,
    );

    if (!bridge) return create_message(`You are not in a bridge`);

    let str = `*Bridge status:*\n\n`;
    str += `**Name:** ${bridge.name}\n`;
    str += `**Channels:**\n`;

    for (const [i, value] of bridge.channels.entries()) {
        str += `${i + 1}. \`${value.id}\` on \`${value.plugin}\`\n`;
    }

    str += `\n**Settings:**\n`;

    for (const [key, value] of Object.entries(bridge.settings)) {
        str += `\`${key}: ${value}\`\n`;
    }

    return create_message(str);
}