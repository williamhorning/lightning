import { type Collection, type ConnectOptions, MongoClient } from '@db/mongo';
import { RedisClient } from '@iuioiua/r2d2';
import { ulid } from '@std/ulid';
import type { bridge } from '../structures/bridge.ts';
import { log_error } from '../structures/errors.ts';
import type { bridge_data } from './mod.ts';
import { redis_messages } from './redis_message.ts';

export type mongo_config = {
	database: ConnectOptions | string;
	redis: Deno.ConnectOptions;
};

export class mongo extends redis_messages implements bridge_data {
	static async create(opts: mongo_config) {
		const client = new MongoClient();
		await client.connect(opts.database);

		const database = client.database();
		const db_data_version = await database.collection('lightning').findOne({
			_id: 'db_data_version',
		});
		const bridge_collection_exists = (await database.listCollectionNames())
			.includes('bridges');

		if (db_data_version?.version !== '0.8.0' && bridge_collection_exists) {
			log_error(
				'Please delete the bridge collection or follow the migrations process in the documentation',
				{
					extra: {
						see:
							'https://williamhorning.eu.org/lightning/hosting/legacy-migrations',
					},
				},
			);
		} else if (!db_data_version && !bridge_collection_exists) {
			await database.collection('lightning').insertOne({
				_id: 'db_data_version',
				version: '0.8.0',
			});
			await database.createCollection('bridges');
		}

		const redis = new RedisClient(await Deno.connect(opts.redis));

		await redis_messages.migrate(redis);

		return new this(database.collection('bridges'), redis);
	}

	private constructor(
		private bridges: Collection<bridge & { _id: string }>,
		redis: RedisClient,
	) {
		super(redis);
	}

	async create_bridge(br: Omit<bridge, 'id'>): Promise<bridge> {
		const id = ulid();
		await this.bridges.insertOne({ _id: id, id, ...br });
		return { id, ...br };
	}

	async edit_bridge(br: bridge): Promise<void> {
		await this.bridges.replaceOne({ _id: br.id }, br);
	}

	async get_bridge_by_id(id: string): Promise<bridge | undefined> {
		return await this.bridges.findOne({ _id: id });
	}

	async get_bridge_by_channel(ch: string): Promise<bridge | undefined> {
		return await this.bridges.findOne({
			channels: {
				$all: [{
					$elemMatch: { id: ch },
				}],
			},
		});
	}
}
