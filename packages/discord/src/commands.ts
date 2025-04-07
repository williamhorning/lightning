import type { API } from '@discordjs/core';
import type { command } from '@jersey/lightning';

export async function setup_commands(
	api: API,
	commands: command[],
): Promise<void> {
	await api.applicationCommands.bulkOverwriteGlobalCommands(
		(await api.applications.getCurrent()).id,
		commands.map((command) => {
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
		}),
	);
}
