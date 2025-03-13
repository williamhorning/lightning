import type { API, ToEventProps } from '@discordjs/core';
import type { command, create_command, lightning } from '@jersey/lightning';
import type {
	APIInteraction,
	RESTPutAPIApplicationCommandsJSONBody,
} from 'discord-api-types';
import { get_discord_message } from './messages.ts';
import type { discord_config } from './mod.ts';

export async function set_slash_commands(
	api: API,
	config: discord_config,
	lightning: lightning,
): Promise<void> {
	if (!config.slash_commands) return;

	await api.applicationCommands.bulkOverwriteGlobalCommands(
		config.application_id,
		get_slash_commands(lightning.commands.values().toArray()),
	);
}

function get_slash_commands(
	commands: command[],
): RESTPutAPIApplicationCommandsJSONBody {
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

export function get_lightning_command(
	interaction: ToEventProps<APIInteraction>,
): create_command | undefined {
	if (interaction.data.type !== 2 || interaction.data.data.type !== 1) return;

	const args: Record<string, string> = {};
	let subcommand: string | undefined;

	for (const option of interaction.data.data.options || []) {
		if (option.type === 1) {
			subcommand = option.name;
			for (const suboption of option.options ?? []) {
				if (suboption.type === 3) {
					args[suboption.name] = suboption.value;
				}
			}
		} else if (option.type === 3) {
			args[option.name] = option.value;
		}
	}

	return {
		args,
		channel: interaction.data.channel.id,
		command: interaction.data.data.name,
		id: interaction.data.id,
		plugin: 'bolt-discord',
		reply: async (msg) =>
			await interaction.api.interactions.reply(
				interaction.data.id,
				interaction.data.token,
				await get_discord_message(msg),
			),
		subcommand,
		timestamp: Temporal.Instant.fromEpochMilliseconds(
			Number(BigInt(interaction.data.id) >> 22n) + 1420070400000,
		),
	};
}
