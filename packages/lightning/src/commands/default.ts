import type { command, command_opts } from '../structures/commands.ts';
import { create } from './bridge/create.ts';
import { join } from './bridge/join.ts';
import { leave } from './bridge/leave.ts';
import { status } from './bridge/status.ts';
import { toggle } from './bridge/toggle.ts';

export const default_commands = new Map([
	['help', {
		name: 'help',
		description: 'get help with the bot',
		execute: () =>
			'check out [the docs](https://williamhorning.eu.org/bolt/) for help.',
	}],
	['ping', {
		name: 'ping',
		description: 'check if the bot is alive',
		execute: ({ timestamp }: command_opts) =>
			`Pong! 🏓 ${
				Temporal.Now.instant().since(timestamp).round('millisecond')
					.total('milliseconds')
			}ms`,
	}],
	['version', {
		name: 'version',
		description: 'get the bots version',
		execute: () => 'hello from v0.8.0!',
	}],
	['bridge', {
		name: 'bridge',
		description: 'bridge commands',
		execute: () => 'take a look at the subcommands of this command',
		subcommands: [
			{
				name: 'create',
				description: 'create a new bridge',
				arguments: [{
					name: 'name',
					description: 'name of the bridge',
					required: true,
				}],
				execute: create,
			},
			{
				name: 'join',
				description: 'join an existing bridge',
				arguments: [{
					name: 'id',
					description: 'id of the bridge',
					required: true,
				}],
				execute: join,
			},
			{
				name: 'leave',
				description: 'leave the current bridge',
				execute: leave,
			},
			{
				name: 'toggle',
				description: 'toggle a setting on the current bridge',
				arguments: [{
					name: 'setting',
					description: 'setting to toggle',
					required: true,
				}],
				execute: toggle,
			},
			{
				name: 'status',
				description: 'get the status of the current bridge',
				execute: status,
			},
		],
	}],
]) as Map<string, command>;
