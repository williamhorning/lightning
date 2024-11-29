import type { API } from '@discordjs/core';
import type { command, run_command_options, lightning } from '@jersey/lightning';
import type { APIInteraction } from 'discord-api-types';
import { to_discord } from './discord.ts';
import { instant } from './lightning.ts';

export function to_command(interaction: { api: API; data: APIInteraction }, lightning: lightning) {
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
		timestamp: instant(interaction.data.id),
		lightning,
		plugin: 'bolt-discord',
		reply: async (msg) => {
			await interaction.api.interactions.reply(
				interaction.data.id,
				interaction.data.token,
				await to_discord(msg),
			);
		},
		args: opts,
	} as run_command_options;
}

export function to_intent_opts({ arguments: args, subcommands }: command) {
	const opts = [];

	if (args) {
		for (const arg of args) {
			opts.push({
				name: arg.name,
				description: arg.description,
				type: 3,
				required: arg.required,
			});
		}
	}

	if (subcommands) {
		for (const sub of subcommands) {
			opts.push({
				name: sub.name,
				description: sub.description,
				type: 1,
				options: sub.arguments?.map((opt) => ({
					name: opt.name,
					description: opt.description,
					type: 3,
					required: opt.required,
				})),
			});
		}
	}

	return opts;
}
