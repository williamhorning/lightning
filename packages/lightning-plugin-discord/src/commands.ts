import type { API } from '@discordjs/core';
import type { command, command_arguments } from '@jersey/lightning';
import type { APIInteraction } from 'discord-api-types';
import { to_discord } from './discord.ts';
import { instant } from './lightning.ts';

// TODO(jersey): migrate over to commands_v2

export function to_command(interaction: { api: API; data: APIInteraction }) {
	if (interaction.data.type !== 2 || interaction.data.data.type !== 1) return;
	const opts = {} as Record<string, string>;
	let subcmd = '';

	for (const opt of interaction.data.data.options || []) {
		if (opt.type === 1) subcmd = opt.name;
		if (opt.type === 3) opts[opt.name] = opt.value;
	}

	return {
		cmd: interaction.data.data.name,
		subcmd,
		reply: async (msg) => {
			await interaction.api.interactions.reply(
				interaction.data.id,
				interaction.data.token,
				await to_discord(msg),
			);
		},
		channel: interaction.data.channel.id,
		plugin: 'bolt-discord',
		opts,
		timestamp: instant(interaction.data.id),
	} as command_arguments;
}

export function to_intent_opts({ options }: command) {
	const opts = [];

	if (options?.argument_name) {
		opts.push({
			name: options.argument_name,
			description: 'option to pass to this command',
			type: 3,
			required: options.argument_required,
		});
	}

	if (options?.subcommands) {
		opts.push(
			...options.subcommands.map((i) => {
				return {
					name: i.name,
					description: i.description || i.name,
					type: 1,
					options: i.options?.argument_name
						? [
							{
								name: i.options.argument_name,
								description: i.options.argument_name,
								type: 3,
								required: i.options.argument_required || false,
							},
						]
						: undefined,
				};
			}),
		);
	}

	return opts;
}
