import type { bridge_channel } from '../../bridge/data.ts';
import { logError } from '../../errors.ts';
import type { command_execute_options } from '../mod.ts';

export async function create(
    opts: command_execute_options,
): Promise<string> {
    const result = await _lightning_bridge_add_common(opts);

    if (typeof result === 'string') return result;

    const bridge_data = {
        name: opts.arguments.name,
        channels: [result],
        settings: {
            allow_editing: true,
            allow_everyone: false,
            use_rawname: false,
        },
    };

    try {
        await opts.lightning.data.create_bridge(bridge_data);
        return `Bridge created successfully! You can now join it using \`${opts.lightning.config.cmd_prefix}join ${result.id}\`. Keep this id safe, don't share it with anyone, and delete this message.`;
    } catch (e) {
        logError(e, {
            message: 'Failed to insert bridge into database',
            extra: bridge_data
        });
    }
}

export async function join(
    opts: command_execute_options,
): Promise<string> {
    const result = await _lightning_bridge_add_common(opts);

    if (typeof result === 'string') return result;

    const target_bridge = await opts.lightning.data.get_bridge_by_id(
        opts.arguments.id,
    );

    if (!target_bridge) {
        return `Bridge with id \`${opts.arguments.id}\` not found. Make sure you have the correct id.`;
    }

    target_bridge.channels.push(result);

    try {
        await opts.lightning.data.edit_bridge(target_bridge);

        return `Bridge joined successfully!`;
    } catch (e) {
        logError(e, {
            message: 'Failed to update bridge in database',
            extra: { target_bridge }
        })
    }
}

async function _lightning_bridge_add_common(
    opts: command_execute_options,
): Promise<string | bridge_channel> {
    const existing_bridge = await opts.lightning.data.get_bridge_by_channel(
        opts.channel,
    );

    if (existing_bridge) {
        return `You are already in a bridge called \`${existing_bridge.name}\`. You must leave it before being in another bridge. Try using \`${opts.lightning.config.cmd_prefix}leave\` or \`${opts.lightning.config.cmd_prefix}help\` commands.`
    }

    const plugin = opts.lightning.plugins.get(opts.plugin);

    if (!plugin) {
        logError('Internal error: platform support not found', {
            extra: { plugin: opts.plugin }
        })
    }

    let bridge_data;

    try {
        bridge_data = await plugin.setup_channel(opts.channel);
    } catch (e) {
        logError(e, {
            message: 'Failed to create bridge using plugin',
            extra: { channel: opts.channel, plugin_name: opts.plugin }
        })
    }

    return {
        id: opts.channel,
        data: bridge_data,
        disabled: false,
        plugin: opts.plugin,
    };
}
