import type { lightning } from '../lightning.ts';

/** representation of a command */
export interface command {
	/** user-facing command name */
	name: string;
	/** user-facing command description */
	description: string;
	/** possible arguments */
	arguments?: command_argument[];
	/** possible subcommands (use `${prefix}${cmd} ${subcmd}` if run as text command) */
	subcommands?: Omit<command, 'subcommands'>[];
	/** the functionality of the command, returning text */
	execute: (
		opts: command_opts,
	) => Promise<string> | string;
}

/** argument for a command */
export interface command_argument {
	/** user-facing name for the argument */
	name: string;
	/** description of the argument */
	description: string;
	/** whether the argument is required */
	required: boolean;
}

/** options passed to command#execute */
export interface command_opts {
	/** the channel the command was run in */
	channel: string;
	/** the plugin the command was run with */
	plugin: string;
	/** the time the command was sent */
	timestamp: Temporal.Instant;
	/** arguments for the command */
	args: Record<string, string>;
	/** a lightning instance */
	lightning: lightning;
}
