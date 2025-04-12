import { Client, type ClientOptions } from '@db/postgres';
import { ulid } from '@std/ulid';
import type { bridge, bridge_message } from '../structures/bridge.ts';
import type { bridge_data } from './mod.ts';

export type { ClientOptions as postgres_config };

export class postgres implements bridge_data {
	static async create(pg_url: string): Promise<bridge_data> {
		const pg = new Client(pg_url);

		await pg.connect();
		await postgres.setup_schema(pg);

		return new this(pg);
	}

	private static async setup_schema(pg: Client) {
		await pg.queryArray`
            CREATE TABLE IF NOT EXISTS lightning (
                prop  TEXT PRIMARY KEY,
                value TEXT NOT NULL
            );

            INSERT INTO lightning (prop, value)
            VALUES ('db_data_version', '0.8.0')
            /* the database shouldn't have been created before 0.8.0 so this is safe */
            ON CONFLICT (prop) DO NOTHING;

            CREATE TABLE IF NOT EXISTS bridges (
                id       TEXT PRIMARY KEY,
                name     TEXT NOT NULL,
                channels JSONB NOT NULL,
                settings JSONB NOT NULL
            );

            CREATE TABLE IF NOT EXISTS bridge_messages (
                id        TEXT PRIMARY KEY,
				name      TEXT NOT NULL,
                bridge_id TEXT NOT NULL,
                channels  JSONB NOT NULL,
                messages  JSONB NOT NULL,
                settings  JSONB NOT NULL
            );
        `;
	}

	private constructor(private pg: Client) {}

	async create_bridge(br: Omit<bridge, 'id'>): Promise<bridge> {
		const id = ulid();

		await this.pg.queryArray`
            INSERT INTO bridges (id, name, channels, settings)
            VALUES (${id}, ${br.name}, ${JSON.stringify(br.channels)}, ${
			JSON.stringify(br.settings)
		})
        `;

		return { id, ...br };
	}

	async edit_bridge(br: bridge): Promise<void> {
		await this.pg.queryArray`
            UPDATE bridges
            SET channels = ${JSON.stringify(br.channels)},
                settings = ${JSON.stringify(br.settings)}
            WHERE id = ${br.id}
        `;
	}

	async get_bridge_by_id(id: string): Promise<bridge | undefined> {
		const res = await this.pg.queryObject<bridge>`
            SELECT * FROM bridges WHERE id = ${id}
        `;

		return res.rows[0];
	}

	async get_bridge_by_channel(ch: string): Promise<bridge | undefined> {
		const res = await this.pg.queryObject<bridge>(`
            SELECT * FROM bridges WHERE EXISTS (
                SELECT 1 FROM jsonb_array_elements(channels) AS ch
                WHERE ch->>'id' = '${ch}'
            )
        `);

		return res.rows[0];
	}

	async create_message(msg: bridge_message): Promise<void> {
		await this.pg.queryArray`INSERT INTO bridge_messages
            (id, name, bridge_id, channels, messages, settings) VALUES
            (${msg.id}, ${msg.name}, ${msg.bridge_id}, ${
			JSON.stringify(msg.channels)
		}, ${JSON.stringify(msg.messages)}, ${JSON.stringify(msg.settings)})`;
	}

	async edit_message(msg: bridge_message): Promise<void> {
		await this.pg.queryArray`
            UPDATE bridge_messages
            SET messages = ${JSON.stringify(msg.messages)},
                channels = ${JSON.stringify(msg.channels)},
                settings = ${JSON.stringify(msg.settings)}
            WHERE id = ${msg.id}
        `;
	}

	async delete_message({ id }: bridge_message): Promise<void> {
		await this.pg.queryArray`
            DELETE FROM bridge_messages WHERE id = ${id}
        `;
	}

	async get_message(id: string): Promise<bridge_message | undefined> {
		const res = await this.pg.queryObject<bridge_message>(`
            SELECT * FROM bridge_messages
            WHERE id = '${id}' OR EXISTS (
                SELECT 1 FROM jsonb_array_elements(messages) AS msg
				CROSS JOIN jsonb_array_elements_text(msg->'id') AS id_element
                WHERE id_element = '${id}'
            )
        `);

		return res.rows[0];
	}

	async migration_get_bridges(): Promise<bridge[]> {
		const res = await this.pg.queryObject<bridge>(`
            SELECT * FROM bridges
        `);

		return res.rows;
	}

	async migration_get_messages(): Promise<bridge_message[]> {
		const res = await this.pg.queryObject<bridge_message>(`
            SELECT * FROM bridge_messages
        `);

		return res.rows;
	}

	async migration_set_messages(messages: bridge_message[]): Promise<void> {
		for (const msg of messages) {
			try {
				await this.create_message(msg);
			} catch {
				console.warn(`failed to insert message ${msg.id}`);
			}
		}
	}

	async migration_set_bridges(bridges: bridge[]): Promise<void> {
		for (const br of bridges) {
			await this.pg.queryArray`
                INSERT INTO bridges (id, name, channels, settings)
                VALUES (${br.id}, ${br.name}, ${JSON.stringify(br.channels)}, ${
				JSON.stringify(br.settings)
			})
            `;
		}
	}

	static async migration_get_instance(): Promise<bridge_data> {
		const pg_url = prompt(
			'Please enter your Postgres connection string (postgres://localhost):',
		) ||
			'postgres://localhost';

		return await postgres.create(pg_url);
	}
}
