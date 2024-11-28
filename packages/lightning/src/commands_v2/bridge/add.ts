import type { bridge_channel } from '../../bridge/data.ts';
import { log_error } from '../../errors.ts';
import { create_message, type message } from '../../messages.ts';
import type { command_execute_options } from '../mod.ts';

export async function create(opts: command_execute_options): Promise<message> {
    const result = await _lightning_bridge_add_common(opts, 'name');

    if (!('data' in result)) return result;

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
        return create_message(
            `Bridge created successfully! You can now join it using \`${opts.lightning.config.cmd_prefix}join ${result.id}\`. Keep this id safe, don't share it with anyone, and delete this message.`,
        );
    } catch (e) {
        return (await log_error(
            new Error('Failed to insert bridge into database', { cause: e }),
            bridge_data,
        )).message;
    }
}

export async function join(opts: command_execute_options): Promise<message> {
    const target_bridge = await opts.lightning.data.get_bridge_by_id(
        opts.arguments.id,
    );

    if (!target_bridge) {
        return create_message(
            `Bridge with id \`${opts.arguments.id}\` not found. Make sure you have the correct id.`,
        );
    }

    const result = await _lightning_bridge_add_common(opts, 'id');

    if (!('data' in result)) return result;

    target_bridge.channels.push(result);

    try {
        await opts.lightning.data.edit_bridge(target_bridge);

        return create_message(
            `Bridge joined successfully!`,
        );
    } catch (e) {
        return (await log_error(
            new Error('Failed to update bridge in database', {
                cause: e,
            }),
            {
                bridge: target_bridge,
            },
        )).message;
    }
}

async function _lightning_bridge_add_common(
    opts: command_execute_options,
    option_name: 'name' | 'id',
): Promise<message | bridge_channel> {
    const existing_bridge = await opts.lightning.data.get_bridge_by_channel(
        opts.channel,
    );

    if (existing_bridge) {
        return create_message(
            `You are already in a bridge called \`${existing_bridge.name}\`. You must leave it before being in another bridge. Try using \`${opts.lightning.config.cmd_prefix}leave\` or \`${opts.lightning.config.cmd_prefix}help\` commands.`,
        );
    }

    if (!opts.arguments[option_name]) {
        return create_message(
            `Please provide the \`${option_name}\` argument. Try using \`${opts.lightning.config.cmd_prefix}help\` command.`,
        );
    }

    const plugin = opts.lightning.plugins.get(opts.plugin);

    if (!plugin) {
        return (await log_error(
            new Error('Internal error: platform support not found'),
            {
                plugin: opts.plugin,
            },
        )).message;
    }

    let bridge_data;

    try {
        bridge_data = await plugin.create_bridge(opts.channel);
    } catch (e) {
        return (await log_error(
            new Error('Failed to create bridge using plugin', { cause: e }),
            {
                channel: opts.channel,
                plugin_name: opts.plugin,
            },
        )).message;
    }

    return {
        id: opts.channel,
        data: bridge_data,
        disabled: false,
        plugin: opts.plugin,
    };
}
