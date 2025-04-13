import { RedisClient } from '@iuioiua/redis';
import type { bridge, bridge_message } from '../structures/bridge.ts';
import type { bridge_data } from './mod.ts';
import { log_error } from '../structures/errors.ts';

export type redis_config = Deno.ConnectOptions;

export class redis implements bridge_data {
	static async create(rd_options: Deno.ConnectOptions): Promise<bridge_data> {
		const conn = await Deno.connect(rd_options);
		const client = new RedisClient(conn);

		await this.migrate(client);

		return new this(client);
	}

	static async migrate(rd: RedisClient): Promise<void> {
		let db_data_version = await rd.sendCommand([
			'GET',
			'lightning-db-version',
		]);

		if (db_data_version === null) {
			const number_keys = await rd.sendCommand(['DBSIZE']) as number;

			if (number_keys === 0) db_data_version = '0.8.0';
		}

		if (db_data_version !== '0.8.0') {
			console.warn(
				`[lightning-redis] migrating database from ${db_data_version} to 0.8.0`,
			);

			console.log('[lightning-redis] getting keys');

			const all_keys = await rd.sendCommand([
				'KEYS',
				'lightning-*',
			]) as string[];

			console.log('[lightning-redis] got keys');

			const new_data = await Promise.all(all_keys.map(async (key: string) => {
				console.log(`[lightning-redis] migrating key ${key}`);
				const type = await rd.sendCommand(['TYPE', key]) as string;
				const value = await rd.sendCommand([
					type === 'string' ? 'GET' : 'JSON.GET',
					key,
				]) as string;

				try {
					const parsed = JSON.parse(value);
					return [
						key,
						JSON.stringify(
							{
								id: key.split('-')[2],
								bridge_id: parsed.id,
								channels: parsed.channels,
								messages: parsed.messages,
								name: parsed.id,
								settings: {
									allow_everyone: false,
								},
							} as bridge | bridge_message,
						),
					];
				} catch {
					return [key, value];
				}
			}));

			Deno.writeTextFileSync(
				'lightning-redis-migration.json',
				JSON.stringify(new_data, null, 2),
			);

			console.warn('[lightning-redis] do you want to continue?');

			const write = confirm('write the data to the database?');
			const env_confirm = Deno.env.get('LIGHTNING_MIGRATE_CONFIRM');

			if (write || env_confirm === 'true') {
				await rd.sendCommand(['DEL', ...all_keys]);
				await rd.sendCommand([
					'MSET',
					'lightning-db-version',
					'0.8.0',
					...new_data.flat(1),
				]);

				console.warn('[lightning-redis] data written to database');
			} else {
				console.warn('[lightning-redis] data not written to database');
				log_error('migration cancelled');
			}
		}
	}

	private constructor(public redis: RedisClient) {
		this.redis = redis;
	}

	async get_json<T>(key: string): Promise<T | undefined> {
		const reply = await this.redis.sendCommand(['GET', key]);
		if (!reply || reply === 'OK') return;
		return JSON.parse(reply as string) as T;
	}

	async create_bridge(br: Omit<bridge, 'id'>): Promise<bridge> {
		const id = crypto.randomUUID();

		await this.edit_bridge({ id, ...br });

		return { id, ...br };
	}

	async edit_bridge(br: bridge): Promise<void> {
		const old_bridge = await this.get_bridge_by_id(br.id);

		for (const channel of old_bridge?.channels || []) {
			await this.redis.sendCommand([
				'DEL',
				`lightning-bchannel-${channel.id}`,
			]);
		}

		await this.redis.sendCommand([
			'SET',
			`lightning-bridge-${br.id}`,
			JSON.stringify(br),
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

	async create_message(msg: bridge_message): Promise<void> {
		await this.redis.sendCommand([
			'SET',
			'lightning-message-${msg.id}',
			JSON.stringify(msg),
		]);

		for (const message of msg.messages) {
			await this.redis.sendCommand([
				'SET',
				`lightning-message-${message.id}`,
				JSON.stringify(msg),
			]);
		}
	}

	async edit_message(msg: bridge_message): Promise<void> {
		await this.create_message(msg);
	}

	async delete_message(msg: bridge_message): Promise<void> {
		await this.redis.sendCommand(['DEL', `lightning-message-${msg.id}`]);
	}

	async get_message(id: string): Promise<bridge_message | undefined> {
		return await this.get_json<bridge_message>(
			`lightning-message-${id}`,
		);
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

			for (const channel of bridge.channels) {
				await this.redis.sendCommand([
					'SET',
					`lightning-bchannel-${channel.id}`,
					bridge.id,
				]);
			}
		}
	}

	async migration_get_messages(): Promise<bridge_message[]> {
		const keys = await this.redis.sendCommand([
			'KEYS',
			'lightning-message-*',
		]) as string[];

		const messages = [] as bridge_message[];

		for (const key of keys) {
			const message = await this.get_json<bridge_message>(key);
			if (message) messages.push(message);
		}

		return messages;
	}

	async migration_set_messages(messages: bridge_message[]): Promise<void> {
		for (const message of messages) {
			await this.create_message(message);
		}

		await this.redis.sendCommand(['SET', 'lightning-db-version', '0.8.0']);
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
