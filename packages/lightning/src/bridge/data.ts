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

export interface bridge_message {
	id: string; /* original message id */
	bridge_id: string; /* bridge id */
	channels: bridge_channel[]; /* channels bridged */
	messages: bridged_message[]; /* bridged messages */
	settings: bridge_settings; /* settings for the bridge */
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

		await bridge_data.create_table(pg);

		return new bridge_data(pg);
	}

	private static async create_table(pg: Client) {
		const exists = (await pg.queryArray`SELECT relname FROM pg_class
			WHERE relname = 'bridges'`).rows.length > 0;

		if (exists) return;

		await pg.queryArray`
			CREATE TABLE bridges (
				id       TEXT PRIMARY KEY,
				name     TEXT NOT NULL,
				channels JSONB NOT NULL,
				settings JSONB NOT NULL
			);

			CREATE TABLE bridge_messages (
				id        TEXT PRIMARY KEY,
				bridge_id TEXT NOT NULL,
				channels  JSONB NOT NULL,
				messages  JSONB NOT NULL,
				settings  JSONB NOT NULL
			);
		`;
	}

	private constructor(pg_client: Client) {
		this.pg = pg_client;
	}

	async create_bridge(br: Omit<bridge, "id">): Promise<bridge> {
		const id = ulid();

		await this.pg.queryArray`
			INSERT INTO bridges (id, name, channels, settings)
			VALUES (${id}, ${br.name}, ${JSON.stringify(br.channels)}, ${JSON.stringify(br.settings)})
		`;

		return { id, ...br };
	}

	async edit_bridge(br: Omit<bridge, "name">): Promise<void> {
		await this.pg.queryArray`
			UPDATE bridges
			SET channels = ${JSON.stringify(br.channels)}, settings = ${JSON.stringify(br.settings)}
			WHERE id = ${br.id}
		`;
	}

	async get_bridge_by_id(id: string): Promise<bridge | undefined> {
		const res = await this.pg.queryObject<bridge>`
			SELECT * FROM bridges
			WHERE id = ${id}
		`;

		return res.rows[0];
	}

	async get_bridge_by_channel(ch: string): Promise<bridge | undefined> {
		const res = await this.pg.queryObject<bridge>(`
			SELECT * FROM bridges
			WHERE EXISTS (
				SELECT 1 FROM jsonb_array_elements(channels) AS ch
				WHERE ch->>'id' = '${ch}'
			)
		`);

		return res.rows[0];
	}

	async create_bridge_message(msg: bridge_message): Promise<void> {
		await this.pg.queryArray`INSERT INTO bridge_messages
			(id, bridge_id, channels, messages, settings) VALUES
			(${msg.id}, ${msg.bridge_id}, ${JSON.stringify(msg.channels)}, ${JSON.stringify(msg.messages)}, ${JSON.stringify(msg.settings)})`;
	}

	async edit_bridge_message(msg: bridge_message): Promise<void> {
		await this.pg.queryArray`
			UPDATE bridge_messages
			SET messages = ${JSON.stringify(msg.messages)}, channels = ${JSON.stringify(msg.channels)}, settings = ${JSON.stringify(msg.settings)}
			WHERE id = ${msg.id}
		`;
	}

	async delete_bridge_message({ id }: bridge_message): Promise<void> {
		await this.pg.queryArray`
			DELETE FROM bridge_messages WHERE id = ${id}
		`;
	}

	async get_bridge_message(id: string): Promise<bridge_message | undefined> {
		const res = await this.pg.queryObject<bridge_message>(`
			SELECT * FROM bridge_messages
			WHERE id = '${id}' OR EXISTS (
				SELECT 1 FROM jsonb_array_elements(messages) AS msg
				WHERE EXISTS (
					SELECT 1 FROM jsonb_array_elements(msg->'id') AS id
					WHERE id = '${id}'
				)
			)
		`);

		return res.rows[0];
	}
}
