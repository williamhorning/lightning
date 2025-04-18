import { promptSelect } from '@std/cli/unstable-prompt-select';
import type { bridge, bridge_message } from '../structures/bridge.ts';
import { postgres } from './postgres.ts';
import { redis, type redis_config } from './redis.ts';

export interface bridge_data {
	create_bridge(br: Omit<bridge, 'id'>): Promise<bridge>;
	edit_bridge(br: bridge): Promise<void>;
	get_bridge_by_id(id: string): Promise<bridge | undefined>;
	get_bridge_by_channel(ch: string): Promise<bridge | undefined>;
	create_message(msg: bridge_message): Promise<void>;
	edit_message(msg: bridge_message): Promise<void>;
	delete_message(msg: bridge_message): Promise<void>;
	get_message(id: string): Promise<bridge_message | undefined>;
	migration_get_bridges(): Promise<bridge[]>;
	migration_get_messages(): Promise<bridge_message[]>;
	migration_set_bridges(bridges: bridge[]): Promise<void>;
	migration_set_messages(messages: bridge_message[]): Promise<void>;
}

export type database_config = {
	type: 'postgres';
	config: string;
} | {
	type: 'redis';
	config: redis_config;
};

export async function create_database(
	config: database_config,
): Promise<bridge_data> {
	switch (config.type) {
		case 'postgres':
			return await postgres.create(config.config);
		case 'redis':
			return await redis.create(config.config);
		default:
			throw new Error('invalid database type', { cause: config });
	}
}

function get_database(message: string): typeof postgres | typeof redis {
	const type = promptSelect(message, ['redis', 'postgres']);

	switch (type) {
		case 'postgres':
			return postgres;
		case 'redis':
			return redis;
		default:
			throw new Error('invalid database type!');
	}
}

export async function handle_migration() {
	const start = await get_database('Please enter your starting database type: ')
		.migration_get_instance();

	const end = await get_database('Please enter your ending database type: ')
		.migration_get_instance();

	console.log('Downloading bridges...');
	let bridges = await start.migration_get_bridges();

	console.log(`Copying ${bridges.length} bridges...`);
	await end.migration_set_bridges(bridges);
	bridges = [];

	console.log('Downloading messages...');
	let messages = await start.migration_get_messages();

	console.log(`Copying ${messages.length} messages...`);
	await end.migration_set_messages(messages);
	messages = [];

	console.log('Migration complete!');
}
