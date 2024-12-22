import { RedisClient } from '@iuioiua/r2d2';
import { ulid } from '@std/ulid';
import type { bridge } from '../structures/bridge.ts';
import type { bridge_data } from './mod.ts';
import { redis_bridge_message_handler } from './redis_message.ts';

export type redis_config = Deno.ConnectOptions;

export class redis extends redis_bridge_message_handler implements bridge_data {
	static async create(rd_options: Deno.ConnectOptions): Promise<bridge_data> {
		const conn = await Deno.connect(rd_options);
		const client = new RedisClient(conn);
		const db_data_version = await client.sendCommand([
			'GET',
			'lightning-db-version',
		]);

		if (db_data_version !== '0.8.0') {
			console.warn(
				`[lightning-redis] migrating database from ${db_data_version} to 0.8.0`,
			);

			// TODO(jersey): use code to handle 0.7.x bridges
			// basically just need to migrate anything that starts with lightning-bridge-
			// allow_editing and use_rawname just get mvoed to the settings object
			// and then everything else is all set

			throw 'not implemented';
		}

		return new this(client);
	}

	private constructor(public redis: RedisClient) {
		super();
	}

	async create_bridge(br: Omit<bridge, 'id'>): Promise<bridge> {
		const id = ulid();

		await this.edit_bridge({ id, ...br });

		return { id, ...br };
	}

	async edit_bridge(br: bridge): Promise<void> {
		await this.redis.sendCommand([
			'SET',
			`lightning-bridge-${br.id}`,
			JSON.stringify({ ...br, name }),
		]);

		for (const channel of br.channels) {
			await this.redis.sendCommand([
				'SET',
				`lightning-bchannel-${channel.id}`,
				br.id,
			]);
		}
	}

	async get_bridge_by_id(id: string): Promise<bridge | undefined> {
		return await this.get_json<bridge>(`lightning-bridge-${id}`);
	}

	async get_bridge_by_channel(ch: string): Promise<bridge | undefined> {
		const channel = await this.redis.sendCommand([
			'GET',
			`lightning-bchannel-${ch}`,
		]);
		if (!channel || channel === 'OK') return;
		return await this.get_json<bridge>(`lightning-bridge-${channel}`);
	}
}
