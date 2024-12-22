import { ulid } from '@std/ulid';
import type { bridge } from '../structures/bridge.ts';
import type { bridge_data } from './mod.ts';
import { type Collection, type ConnectOptions, MongoClient } from '@db/mongo';
import { redis_bridge_message_handler } from './redis_message.ts';
import { RedisClient } from '@iuioiua/r2d2';

export type mongo_config = {
    database: ConnectOptions | string;
    redis: Deno.ConnectOptions;
};

export class mongo extends redis_bridge_message_handler implements bridge_data {
	static async create(opts: mongo_config) {
		const client = new MongoClient();
		await client.connect(opts.database);

        const database = client.database();
        const db_data_version = await database.collection('lightning').findOne({ _id: 'db_data_version' });
        const bridge_collection_exists = (await database.listCollectionNames()).includes('bridges');

        if (db_data_version?.version !== '0.8.0' && bridge_collection_exists) {
            const version = db_data_version?.version ?? 'unknown';

            console.warn(`[lightning-mongo] migrating database from ${version} to 0.8.0`);
            
            // TODO(jersey): use code to feature detect the version if not present and then migrate
			// it may be worth it to just allow migrations from the last version before redisforeverything
			// and have anything prior use the migration script from back then

            throw "not implemented";
        }

        const redis = new RedisClient(await Deno.connect(opts.redis));

		return new this(database.collection('bridges'), redis);
	}

	private constructor(
		private bridges: Collection<bridge & { _id: string }>,
        public redis: RedisClient,
	) {
        super();
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
