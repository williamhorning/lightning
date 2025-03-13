import type { command_opts } from './commands.ts';
import type { deleted_message, message } from './messages.ts';

/** command execution event */
export interface create_command extends Omit<Omit<command_opts, 'args'>, 'lightning'> {
	/** the command to run */
	command: string;
	/** the subcommand, if any, to use */
	subcommand?: string;
	/** arguments, if any, to use */
	args?: Record<string, string>;
	/** extra string options */
	rest?: string[];
	/** event reply function */
	reply: message['reply'];
	/** id of the associated event */
	id: string;
}

/** the events emitted by a plugin */
export type plugin_events = {
	/** when a message is created */
	create_message: [message];
	/** when a message is edited */
	edit_message: [message];
	/** when a message is deleted */
	delete_message: [deleted_message];
	/** when a command is run */
	create_command: [create_command];
};
