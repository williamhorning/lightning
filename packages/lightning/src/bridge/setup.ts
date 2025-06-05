import type { core } from '../core.ts';
import { create_database, type database_config } from '../database/mod.ts';
import { create, join, subscribe, leave, status, toggle } from './commands.ts';
import { bridge_message } from './handler.ts';

export async function setup_bridge(core: core, config: database_config) {
	const database = await create_database(config);

	core.on(
		'create_message',
		(msg) => bridge_message(core, database, 'create_message', msg),
	);
	core.on(
		'edit_message',
		(msg) => bridge_message(core, database, 'edit_message', msg),
	);
	core.on(
		'delete_message',
		(msg) => bridge_message(core, database, 'delete_message', msg),
	);

	core.set_command({
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
				execute: (o) => create(database, o),
			},
			{
				name: 'join',
				description: 'join an existing bridge',
				arguments: [{
					name: 'id',
					description: 'id of the bridge',
					required: true,
				}],
				execute: (o) => join(database, o),
			},
			{
				name: 'subscribe',
				description: 'subscribe to a bridge',
				arguments: [{
					name: 'id',
					description: 'id of the bridge',
					required: true,
				}],
				execute: (o) => subscribe(database, o),
			},
			{
				name: 'leave',
				description: 'leave the current bridge',
				arguments: [{
					name: 'id',
					description: 'id of the current bridge',
					required: true,
				}],
				execute: (o) => leave(database, o),
			},
			{
				name: 'toggle',
				description: 'toggle a setting on the current bridge',
				arguments: [{
					name: 'setting',
					description: 'setting to toggle',
					required: true,
				}],
				execute: (o) => toggle(database, o),
			},
			{
				name: 'status',
				description: 'get the status of the current bridge',
				execute: (o) => status(database, o),
			},
		],
	});
}
