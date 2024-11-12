import type { lightning } from '../lightning.ts';
import type { message } from '../messages.ts';

/** arguments passed to a command */
export interface command_arguments {
	/** the name of the command */
	cmd: string;
	/** the subcommand being run, if any */
	subcmd?: string;
	/** the channel its being run in */
	channel: string;
	/** the plugin its being run on */
	plugin: string;
	/** the id of the associated event */
	id: string;
	/** timestamp given */
	timestamp: Temporal.Instant;
	/** options passed by the user */
	opts: Record<string, string>;
	/** the function to reply to the command */
	reply: (message: message, optional?: unknown) => Promise<void>;
	/** the instance of lightning the command is ran against */
	lightning: lightning;
}

/** options when parsing a command */
// TODO(jersey): make the options more flexible
export interface command_options {
	/** this will be the key passed to options.opts in the execute function */
	argument_name?: string;
	/** whether or not the argument provided is required */
	argument_required?: boolean;
	/** an array of commands that show as subcommands */
	subcommands?: command[];
}

/** commands are a way for users to interact with the bot */
export interface command {
	/** the name of the command */
	name: string;
	/** an optional description */
	description?: string;
	/** options when parsing the command */
	options?: command_options;
	/** a function that returns a message */
	execute: (options: command_arguments) => Promise<string> | string;
}

export const default_commands = [
	[
		'help',
		{
			name: 'help',
			description: 'get help',
			execute: () =>
				'check out [the docs](https://williamhorning.eu.org/bolt/) for help.',
		},
	],
	[
		'version',
		{
			name: 'version',
			description: 'get the bots version',
			execute: () => 'hello from v0.7.4!',
		},
	],
	[
		'ping',
		{
			name: 'ping',
			description: 'pong',
			execute: ({ timestamp }) =>
				`Pong! 🏓 ${
					Temporal.Now.instant()
						.since(timestamp)
						.total('milliseconds')
				}ms`,
		},
	],
] as [string, command][];
