import type { bridge_data } from '../database/mod.ts';
import type { plugin } from './plugins.ts';
import type { message } from './messages.ts';

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
	channel_id: string;
	/** the plugin the command was run with */
	plugin: string;
	/** the time the command was sent */
	timestamp: Temporal.Instant;
	/** arguments for the command */
	args: Record<string, string>;
	/** the command prefix used */
	prefix: string;
	/** bridge data (for bridge commands) */
	bridge_data: bridge_data;
	/** plugin data */
	plugins: Map<string, plugin<unknown>>;
}

/** command execution event */
export interface create_command
	extends Pick<command_opts, 'channel_id' | 'plugin' | 'timestamp'> {
	/** the command to run */
	command: string;
	/** the subcommand, if any, to use */
	subcommand?: string;
	/** arguments, if any, to use */
	args?: Record<string, string | undefined>;
	/** the command prefix used */
	prefix?: string;
	/** extra string options */
	rest?: string[];
	/** event reply function */
	reply: (message: message) => Promise<void>;
	/** id of the associated event */
	message_id: string;
}
