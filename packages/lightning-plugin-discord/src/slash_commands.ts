import type { command, lightning } from '@jersey/lightning';
import type { API } from '@discordjs/core';
import type { discord_config } from './mod.ts';

export async function setup_slash_commands(
    api: API,
    config: discord_config,
    lightning: lightning,
) {
    if (!config.slash_commands) return;

    const commands = lightning.commands.values().toArray();

    await api.applicationCommands.bulkOverwriteGlobalCommands(
        config.application_id,
        commands_to_discord(commands)
    );
}

function commands_to_discord(commands: command[]) {
    return commands.map((command) => {
        const opts = [];

        if (command.arguments) {
            for (const argument of command.arguments) {
                opts.push({
                    name: argument.name,
                    description: argument.description,
                    type: 3,
                    required: argument.required,
                });
            }
        }

        if (command.subcommands) {
            for (const subcommand of command.subcommands) {
                opts.push({
                    name: subcommand.name,
                    description: subcommand.description,
                    type: 1,
                    options: subcommand.arguments?.map((opt) => ({
                        name: opt.name,
                        description: opt.description,
                        type: 3,
                        required: opt.required,
                    })),
                });
            }
        }

        return {
            name: command.name,
            type: 1,
            description: command.description,
            options: opts,
        };
    });
}
