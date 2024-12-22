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
