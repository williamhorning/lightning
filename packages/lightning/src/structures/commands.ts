import type { message } from './messages.ts';
import type { plugin } from './plugins.ts';

/** representation of a command */
export interface command {
	/** user-facing command name */
	name: string;
	/** user-facing command description */
	description: string;
	/** possible arguments */
	arguments?: command_argument[];
	/** possible subcommands (use `${prefix}${cmd} ${subcommand}` if run as text command) */
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

/** options given to a command */
export interface command_opts {
	/** arguments to use */
	args: Record<string, string | undefined>;
	/** the channel the command was run in */
	channel_id: string;
	/** the plugin the command was run with */
	plugin: plugin;
	/** the command prefix used */
	prefix: string;
	/** the time the command was sent */
	timestamp: Temporal.Instant;
}

/** options used for a command event */
export interface create_command extends Omit<command_opts, 'plugin'> {
	/** the command to run */
	command: string;
	/** id of the associated event */
	message_id: string;
	/** the plugin id used to run this with */
	plugin: string;
	/** other, additional, options */
	rest?: string[];
	/** event reply function */
	reply: (message: message) => Promise<void>;
	/** the subcommand, if any, to use */
	subcommand?: string;
}
