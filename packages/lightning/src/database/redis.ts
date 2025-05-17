import { RedisClient } from '@iuioiua/redis';
import {
	ProgressBar,
	type ProgressBarFormatter,
} from '@std/cli/unstable-progress-bar';
import { writeTextFile } from '@std/fs/unstable-write-text-file';
import type {
	bridge,
	bridge_channel,
	bridge_message,
	bridged_message,
} from '../structures/bridge.ts';
import { get_env, stdout, tcp_connect } from '../structures/cross.ts';
import { log_error } from '../structures/errors.ts';
import type { bridge_data } from './mod.ts';

export interface redis_config {
	hostname: string;
	port: number;
}

const fmt = (fmt: ProgressBarFormatter) =>
	`[redis] ${fmt.progressBar} ${fmt.styledTime()} [${fmt.value}/${fmt.max}]\n`;

export class redis implements bridge_data {
	private redis: RedisClient;
	private seven: boolean;

	static async create(
		rd_options: redis_config,
		_do_not_use = false,
	): Promise<bridge_data> {
		const rd = new RedisClient(await tcp_connect(rd_options));

		let db_data_version = await rd.sendCommand([
			'GET',
			'lightning-db-version',
		]);

		if (db_data_version === null) {
			const number_keys = await rd.sendCommand(['DBSIZE']) as number;

			if (number_keys === 0) {
				await rd.sendCommand(['SET', 'lightning-db-version', '0.8.0']);
				db_data_version = '0.8.0';
			}
		}

		if (db_data_version !== '0.8.0' && !_do_not_use) {
			console.warn(
				`[lightning-redis] migrating database from ${db_data_version} to 0.8.0`,
			);

			const instance = new this(rd, true);

			console.log('[lightning-redis] getting bridges...');

			const bridges = await instance.migration_get_bridges();

			console.log('[lightning-redis] got bridges!');

			await writeTextFile(
				'lightning-redis-migration.json',
				JSON.stringify(bridges, null, 2),
			);

			const write = confirm(
				'[lightning-redis] write the data to the database? see \`lightning-redis-migration.json\` for the data',
			);
			const env_confirm = get_env('LIGHTNING_MIGRATE_CONFIRM');

			if (write || env_confirm === 'true') {
				await instance.migration_set_bridges(bridges);

				const former_messages = await rd.sendCommand([
					'KEYS',
					'lightning-bridged-*',
				]) as string[];

				for (const key of former_messages) {
					await rd.sendCommand(['DEL', key]);
				}

				console.warn('[lightning-redis] data written to database');

				return instance;
			} else {
				log_error('[lightning-redis] data not written to database', {
					without_cause: true,
				});
			}
		} else {
			return new this(rd, _do_not_use);
		}
	}

	private constructor(
		redis: RedisClient,
		seven = false,
	) {
		this.redis = redis;
		this.seven = seven;
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
		return await this.get_bridge_by_id(channel as string);
	}

	async create_message(msg: bridge_message): Promise<void> {
		await this.redis.sendCommand([
			'SET',
			`lightning-message-${msg.id}`,
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

		const progress = new ProgressBar(stdout(), {
			max: keys.length,
			fmt,
		});

		for (const key of keys) {
			progress.add(1);
			if (!this.seven) {
				const bridge = await this.get_bridge_by_id(
					key.replace('lightning-bridge-', ''),
				);

				if (bridge) bridges.push(bridge);
			} else {
				// ignore UUIDs and ULIDs
				if (
					key.replace('lightning-bridge-', '').match(
						/[0-7][0-9A-HJKMNP-TV-Z]{25}/gm,
					) ||
					key.replace('lightning-bridge-', '').match(
						/^[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$/i,
					)
				) {
					continue;
				}

				const bridge = await this.get_json<{
					channels: bridge_channel[];
					id: string;
					messages?: bridged_message[];
				}>(key);

				if (bridge && bridge.channels) {
					bridges.push({
						id: key.replace('lightning-bridge-', ''),
						name: bridge.id,
						channels: bridge.channels,
						settings: {
							allow_everyone: false,
						},
					});
				}
			}
		}

		progress.end();

		return bridges;
	}

	async migration_set_bridges(bridges: bridge[]): Promise<void> {
		const progress = new ProgressBar(stdout(), {
			max: bridges.length,
			fmt,
		});

		for (const bridge of bridges) {
			progress.add(1);

			await this.redis.sendCommand([
				'DEL',
				`lightning-bridge-${bridge.id}`,
			]);

			for (const channel of bridge.channels) {
				await this.redis.sendCommand([
					'DEL',
					`lightning-bchannel-${channel.id}`,
				]);
			}

			if (bridge.channels.length < 2) continue;

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

		progress.end();

		await this.redis.sendCommand(['SET', 'lightning-db-version', '0.8.0']);
	}

	async migration_get_messages(): Promise<bridge_message[]> {
		const keys = await this.redis.sendCommand([
			'KEYS',
			'lightning-message-*',
		]) as string[];

		const messages = [] as bridge_message[];

		const progress = new ProgressBar(stdout(), {
			max: keys.length,
			fmt,
		});

		for (const key of keys) {
			progress.add(1);
			const message = await this.get_json<bridge_message>(key);
			if (message) messages.push(message);
		}

		progress.end();

		return messages;
	}

	async migration_set_messages(messages: bridge_message[]): Promise<void> {
		const progress = new ProgressBar(stdout(), {
			max: messages.length,
			fmt,
		});

		for (const message of messages) {
			progress.add(1);
			await this.create_message(message);
		}

		progress.end();

		await this.redis.sendCommand(['SET', 'lightning-db-version', '0.8.0']);
	}

	static async migration_get_instance(): Promise<bridge_data> {
		const hostname = prompt('Please enter your Redis hostname (localhost):') ||
			'localhost';
		const port = prompt('Please enter your Redis port (6379):') || '6379';

		return await redis.create({
			hostname,
			port: parseInt(port),
		}, true);
	}
}
