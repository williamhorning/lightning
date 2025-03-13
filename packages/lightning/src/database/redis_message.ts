import type { RedisClient } from '@iuioiua/redis';
import type { bridge, bridge_message } from '../structures/bridge.ts';
import { log_error } from '../structures/errors.ts';

export class redis_messages {
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

			const all_keys = await rd.sendCommand([
				'KEYS',
				'lightning-*',
			]) as string[];

			const new_data = [] as [string, bridge | bridge_message | string][];

			for (const key of all_keys) {
				const type = await rd.sendCommand(['TYPE', key]) as string;
				const action = type === 'string' ? 'GET' : 'JSON.GET';
				const value = await rd.sendCommand([action, key]) as string;

				try {
					const parsed = JSON.parse(value);

					new_data.push([key, {
						id: key.split('-')[2],
						bridge_id: parsed.id,
						channels: parsed.channels,
						messages: parsed.messages,
						name: `migrated bridge ${parsed.id}`,
						settings: {
							allow_editing: true,
							use_rawname: parsed.use_rawname,
							allow_everyone: true,
						},
					}]);
				} catch {
					new_data.push([key, value]);
				}
			}

			console.warn('[lightning-redis] do you want to continue?');

			const write = confirm('write the data to the database?');
			const env_confirm = Deno.env.get('LIGHTNING_MIGRATE_CONFIRM');

			if (write || env_confirm === 'true') {
				await rd.sendCommand(['DEL', ...all_keys]);

				const data = new_data.map((
					[key, value],
				) => [key, JSON.stringify(value)]);

				await rd.sendCommand(['MSET', ...data.flat()]);
				await rd.sendCommand(['SET', 'lightning-db-version', '0.8.0']);

				console.warn('[lightning-redis] data written to database');
				return;
			} else {
				console.warn('[lightning-redis] data not written to database');
				log_error('migration cancelled');
			}
		}
	}

	constructor(public redis: RedisClient) {}

	async get_json<T>(key: string): Promise<T | undefined> {
		const reply = await this.redis.sendCommand(['GET', key]);
		if (!reply || reply === 'OK') return;
		return JSON.parse(reply as string) as T;
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
}
