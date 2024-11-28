import { log_error } from '../../errors.ts';
import { create_message, type message } from '../../messages.ts';
import type { command_execute_options } from '../mod.ts';

export async function leave(opts: command_execute_options): Promise<message> {
    const bridge = await opts.lightning.data.get_bridge_by_channel(
        opts.channel,
    );

    if (!bridge) return create_message(`You are not in a bridge`);

    bridge.channels = bridge.channels.filter((
        ch,
    ) => ch.id !== opts.channel);

    try {
        await opts.lightning.data.edit_bridge(
            bridge,
        );
        return create_message(`Bridge left successfully`);
    } catch (e) {
        return (await log_error(
            new Error('Error updating bridge', { cause: e }),
            {
                bridge,
            },
        )).message;
    }
}

const settings = ['allow_editing', 'allow_everyone', 'use_rawname'];

export async function toggle(opts: command_execute_options): Promise<message> {
    const bridge = await opts.lightning.data.get_bridge_by_channel(
        opts.channel,
    );

    if (!bridge) return create_message(`You are not in a bridge`);

    if (!settings.includes(opts.arguments.setting)) {
        return create_message(`That setting does not exist`);
    }

    const key = opts.arguments.setting as keyof typeof bridge.settings;

    bridge.settings[key] = !bridge.settings[key];

    try {
        await opts.lightning.data.edit_bridge(
            bridge,
        );
        return create_message(`Bridge settings updated successfully`);
    } catch (e) {
        return (await log_error(
            new Error('Error updating bridge', { cause: e }),
            {
                bridge,
            },
        )).message;
    }
}
