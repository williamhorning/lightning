import { log_error } from '../../errors.ts';
import type { command_execute_options } from '../mod.ts';

export async function leave(opts: command_execute_options): Promise<string> {
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
        throw await log_error(
            new Error('Error updating bridge', { cause: e }),
            {
                bridge,
            },
        );
    }
}

const settings = ['allow_editing', 'allow_everyone', 'use_rawname'];

export async function toggle(opts: command_execute_options): Promise<string> {
    const bridge = await opts.lightning.data.get_bridge_by_channel(
        opts.channel,
    );

    if (!bridge) return `You are not in a bridge`;

    if (!settings.includes(opts.arguments.setting)) {
        return `That setting does not exist`;
    }

    const key = opts.arguments.setting as keyof typeof bridge.settings;

    bridge.settings[key] = !bridge.settings[key];

    try {
        await opts.lightning.data.edit_bridge(
            bridge,
        );
        return `Bridge settings updated successfully`;
    } catch (e) {
        throw await log_error(
            new Error('Error updating bridge', { cause: e }),
            {
                bridge,
            },
        );
    }
}
