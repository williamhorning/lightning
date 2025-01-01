import { RedisClient } from '@iuioiua/r2d2';
import { ulid } from '@std/ulid';
import type { bridge } from '../structures/bridge.ts';
import type { bridge_data } from './mod.ts';
import { redis_messages } from './redis_message.ts';

export type redis_config = Deno.ConnectOptions;

export class redis extends redis_messages implements bridge_data {
	static async create(rd_options: Deno.ConnectOptions): Promise<bridge_data> {
		const conn = await Deno.connect(rd_options);
		const client = new RedisClient(conn);

		await redis_messages.migrate(client);

		return new this(client);
	}

	private constructor(redis: RedisClient) {
		super(redis);
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
