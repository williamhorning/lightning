import { Client, type ClientOptions } from '@db/postgres';
import { ulid } from '@std/ulid';

export interface bridge {
	id: string; /* ulid */
	name: string; /* name of the bridge */
	channels: bridge_channel[]; /* channels bridged */
	settings: bridge_settings; /* settings for the bridge */
}

export interface bridge_channel {
	id: string; /* from the platform */
	data: unknown; /* data needed to bridge this channel */
	disabled: boolean; /* whether the channel is disabled */
	plugin: string; /* the plugin used to bridge this channel */
}

export interface bridge_settings {
	allow_editing: boolean; /* allow editing/deletion */
	allow_everyone: boolean; /* @everyone/@here/@room */
	use_rawname: boolean; /* rawname = username */
}

export interface bridge_message extends bridge {
	original_id: string; /* original message id */
	messages: bridged_message[]; /* bridged messages */
}

export interface bridged_message {
	id: string[]; /* message id */
	channel: string; /* channel id */
	plugin: string; /* plugin id */
}

export class bridge_data {
	private pg: Client;

	static async create(pg_options: ClientOptions): Promise<bridge_data> {
		const pg = new Client(pg_options);
		await pg.connect();

		await this.create_table(pg);

		return new bridge_data(pg);
	}

	private static async create_table(pg: Client) {
		const exists = (await pg.queryArray`SELECT relname FROM pg_class
			WHERE relname = 'bridges'`).rows.length > 0;

		if (exists) return;

		await pg.queryArray`CREATE TABLE bridges (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			channels JSONB NOT NULL,
			settings JSONB NOT NULL
		)`;
		await pg.queryArray`CREATE TABLE bridge_messages (
			original_id TEXT PRIMARY KEY,
			id TEXT NOT NULL,
			name TEXT NOT NULL,
			channels JSONB NOT NULL REFERENCES bridges(channels),
			messages JSONB NOT NULL,
			settings JSONB NOT NULL REFERENCES bridges(settings),
			CONSTRAINT fk_id FOREIGN KEY(id) REFERENCES bridges(id)
		)`;
	}

	private constructor(pg_client: Client) {
		this.pg = pg_client;
	}

	async new_bridge(bridge: Omit<bridge, 'id'>): Promise<bridge> {
		const id = ulid();

		await this.pg.queryArray`INSERT INTO bridges
			(id, name, channels, settings) VALUES
			(${id}, ${bridge.name}, ${bridge.channels}, ${bridge.settings})`;

		return { id, ...bridge };
	}

	async update_bridge(bridge: bridge): Promise<bridge> {
		await this.pg.queryArray`UPDATE bridges SET
			name = ${bridge.name},
			channels = ${bridge.channels},
			settings = ${bridge.settings}
			WHERE id = ${bridge.id}`;

		return bridge;
	}

	async get_bridge_by_id(id: string): Promise<bridge | undefined> {
		const resp = await this.pg.queryObject<bridge>`
			SELECT * FROM bridges WHERE id = ${id}`;

		return resp.rows[0];
	}

	async get_bridge_by_channel(channel: string): Promise<bridge | undefined> {
		const resp = await this.pg.queryObject<bridge>`
			SELECT * FROM bridges WHERE JSON_QUERY(channels, '$[*].id') = ${channel}`;

		return resp.rows[0];
	}

	async new_bridge_message(message: bridge_message): Promise<bridge_message> {
		await this.pg.queryArray`INSERT INTO bridge_messages
			(original_id, id, name, channels, messages, settings) VALUES
			(${message.original_id}, ${message.id}, ${message.name}, ${message.channels}, ${message.messages}, ${message.settings})`;

		return message;
	}

	async update_bridge_message(
		message: bridge_message,
	): Promise<bridge_message> {
		await this.pg.queryArray`UPDATE bridge_messages SET
			id = ${message.id},
			channels = ${message.channels},
			messages = ${message.messages},
			settings = ${message.settings}
			WHERE original_id = ${message.original_id}`;

		return message;
	}

	async delete_bridge_message(id: string): Promise<void> {
		await this.pg
			.queryArray`DELETE FROM bridge_messages WHERE original_id = ${id}`;
	}

	async get_bridge_message(id: string): Promise<bridge_message | undefined> {
		const resp = await this.pg.queryObject<bridge_message>`
			SELECT * FROM bridge_messages WHERE original_id = ${id}`;

		return resp.rows[0];
	}

	async is_bridged_message(id: string): Promise<boolean> {
		const resp = await this.pg.queryObject<bridge_message>`
			SELECT * FROM bridge_messages WHERE JSON_QUERY(messages, '$[*].id') = ${id}`;

		return resp.rows.length > 0;
	}
}
