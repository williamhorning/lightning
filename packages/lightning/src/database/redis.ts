import { RedisClient } from '@iuioiua/redis';
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

	async migration_get_bridges(): Promise<bridge[]> {
		const keys = await this.redis.sendCommand([
			'KEYS',
			'lightning-bridge-*',
		]) as string[];

		const bridges = [] as bridge[];

		for (const key of keys) {
			const bridge = await this.get_bridge_by_id(
				key.replace('lightning-bridge-', ''),
			);

			if (bridge) bridges.push(bridge);
		}

		return bridges;
	}

	async migration_set_bridges(bridges: bridge[]): Promise<void> {
		for (const bridge of bridges) {
			await this.redis.sendCommand([
				'SET',
				`lightning-bridge-${bridge.id}`,
				JSON.stringify(bridge),
			]);
		}
	}

	static async migration_get_instance(): Promise<bridge_data> {
		const hostname = prompt('Please enter your Redis hostname (localhost):') ||
			'localhost';
		const port = prompt('Please enter your Redis port (6379):') || '6379';

		return await redis.create({
			hostname,
			port: parseInt(port),
			transport: 'tcp',
		});
	}
}
