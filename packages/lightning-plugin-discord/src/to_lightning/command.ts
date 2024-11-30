import type { API } from '@discordjs/core';
import type { APIInteraction } from 'discord-api-types';
import type { create_command, lightning } from '@jersey/lightning';
import { message_to_discord } from '../discord_message/mod.ts';

export function command_to(
    interaction: { api: API; data: APIInteraction },
    lightning: lightning,
) {
    if (interaction.data.type !== 2 || interaction.data.data.type !== 1) return;
    const opts = {} as Record<string, string>;
    let subcmd;

    for (const opt of interaction.data.data.options || []) {
        if (opt.type === 1) subcmd = opt.name;
        if (opt.type === 3) opts[opt.name] = opt.value;
    }

    return {
        command: interaction.data.data.name,
        subcommand: subcmd,
        channel: interaction.data.channel.id,
        id: interaction.data.id,
        timestamp: Temporal.Instant.fromEpochMilliseconds(
            Number(BigInt(interaction.data.id) >> 22n) + 1420070400000,
        ),
        lightning,
        plugin: 'bolt-discord',
        reply: async (msg) => {
            await interaction.api.interactions.reply(
                interaction.data.id,
                interaction.data.token,
                await message_to_discord(msg),
            );
        },
        args: opts,
    } as create_command;
}
