import type { RedisClient } from '@iuioiua/r2d2';
import type {
	bridge_channel,
	bridge_message,
	bridged_message,
} from '../structures/bridge.ts';

interface redis_bridge_message {
	allow_editing: boolean;
	channels: bridge_channel[];
	id: string;
	messages?: bridged_message[];
	use_rawname: boolean;
}

// TODO(jersey): the redis_bridge_message structure sucks and if we're doing migrations
//               (see ./redis), there's no point to not use the bridge_message structure

export abstract class redis_bridge_message_handler {
	abstract redis: RedisClient;

	async get_json<T>(key: string): Promise<T | undefined> {
		const reply = await this.redis.sendCommand(['GET', key]);
		if (!reply || reply === 'OK') return;
		return JSON.parse(reply as string) as T;
	}

	async create_message(msg: bridge_message): Promise<void> {
		const redis_msg: redis_bridge_message = {
			allow_editing: msg.settings.allow_editing,
			channels: msg.channels,
			id: msg.bridge_id,
			use_rawname: msg.settings.use_rawname,
			messages: msg.messages,
		};

		await this.redis.sendCommand([
			'SET',
			'lightning-message-${msg.id}',
			JSON.stringify(redis_msg),
		]);

		for (const message of msg.messages) {
			await this.redis.sendCommand([
				'SET',
				`lightning-message-${message.id}`,
				JSON.stringify(redis_msg),
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
		const msg = await this.get_json<redis_bridge_message>(
			`lightning-message-${id}`,
		);
		if (!msg) return;

		return {
			bridge_id: msg.id,
			channels: msg.channels,
			messages: msg.messages || [],
			settings: {
				allow_editing: msg.allow_editing,
				use_rawname: msg.use_rawname,
				allow_everyone: true,
			},
			id,
		};
	}
}
