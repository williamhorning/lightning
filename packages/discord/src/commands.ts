import type { API } from '@discordjs/core';
import type { command } from '@lightning/lightning';

export async function setup_commands(
	api: API,
	commands: command[],
): Promise<void> {
	const format_arguments = (args: command['arguments']) =>
		args?.map((arg) => ({
			name: arg.name,
			description: arg.description,
			type: 3,
			required: arg.required,
		})) ?? [];

	const format_subcommands = (subcommands: command['subcommands']) =>
		subcommands?.map((subcommand) => ({
			name: subcommand.name,
			description: subcommand.description,
			type: 1,
			options: format_arguments(subcommand.arguments),
		})) ?? [];

	await api.applicationCommands.bulkOverwriteGlobalCommands(
		(await api.applications.getCurrent()).id,
		commands.map((cmd) => ({
			name: cmd.name,
			type: 1,
			description: cmd.description,
			options: [
				...format_arguments(cmd.arguments),
				...format_subcommands(cmd.subcommands),
			],
		})),
	);
}
