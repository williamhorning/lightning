import type { bridge, bridge_message } from '../structures/bridge.ts';
import { mongo, type mongo_config } from './mongo.ts';
import { postgres, type postgres_config } from './postgres.ts';
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
	config: postgres_config;
} | {
	type: 'redis';
	config: redis_config;
} | {
	type: 'mongo';
	config: mongo_config;
};

export async function create_database(
	config: database_config,
): Promise<bridge_data> {
	switch (config.type) {
		case 'postgres':
			return await postgres.create(config.config);
		case 'redis':
			return await redis.create(config.config);
		case 'mongo':
			return await mongo.create(config.config);
		default:
			throw new Error('invalid database type');
	}
}

function get_database(
	type: string,
): typeof postgres | typeof redis | typeof mongo {
	switch (type) {
		case 'postgres':
			return postgres;
		case 'redis':
			return redis;
		case 'mongo':
			return mongo;
		default:
			throw new Error('invalid database type');
	}
}

export async function handle_migration() {
	const start_type = prompt(
		'Please enter your starting database type (postgres, redis, mongo):',
	) ?? '';
	const start = await get_database(start_type).migration_get_instance();

	const end_type = prompt(
		'Please enter your ending database type (postgres, redis, mongo):',
	) ?? '';
	const end = await get_database(end_type).migration_get_instance();

	console.log('Downloading bridges...');
	let bridges = await start.migration_get_bridges();

	console.log('Setting bridges...');
	await end.migration_set_bridges(bridges);
	bridges = [];

	console.log('Downloading messages...');
	let messages = await start.migration_get_messages();

	console.log('Setting messages...');
	await end.migration_set_messages(messages);
	messages = [];

	console.log('Migration complete!');
}
