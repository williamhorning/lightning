package bridge

const (
	sqlCreateTables = `
		CREATE TABLE IF NOT EXISTS lightning (
			prop  TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		INSERT INTO lightning (prop, value)
		VALUES ('db_data_version', '0.8.0')
		ON CONFLICT DO NOTHING;

		CREATE TABLE IF NOT EXISTS bridges (
			id       TEXT PRIMARY KEY,
			name     TEXT NOT NULL,
			channels JSONB NOT NULL,
			settings JSONB NOT NULL
		);

		:create_index:

		CREATE TABLE IF NOT EXISTS bridge_messages (
			id        TEXT PRIMARY KEY,
			name      TEXT NOT NULL,
			bridge_id TEXT NOT NULL,
			channels  JSONB NOT NULL,
			messages  JSONB NOT NULL,
			settings  JSONB NOT NULL
		);`

	sqlInsertBridge = `
		INSERT INTO bridges (id, name, channels, settings)
		VALUES (?1, ?2, ?3, ?4)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			channels = excluded.channels,
			settings = excluded.settings`

	sqlInsertMessage = `
		INSERT INTO bridge_messages (id, name, bridge_id, channels, messages, settings)
		VALUES (?1, ?2, ?3, ?4, ?5, ?6)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			channels = excluded.channels,
			messages = excluded.messages,
			settings = excluded.settings`

	sqlSelectBridgeByID = `SELECT id, name, channels, settings FROM bridges WHERE id = ?1`

	sqlSelectBridgeByChannel = `
		SELECT id, name, channels, settings FROM bridges
		WHERE EXISTS (
			SELECT 1 FROM jsonb_array_elements(channels) AS ch
			WHERE ch.value->>'id' = ?1
		);`

	sqlSelectMessage = `
		SELECT id, name, bridge_id, channels, messages, settings FROM bridge_messages
		WHERE id = $1 OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(messages) AS msg
			CROSS JOIN jsonb_array_elements_text(msg->'id') AS id_element
			WHERE id_element.value = $1
		)`

	sqlDeleteMessage = `DELETE FROM bridge_messages WHERE id = ?1`
)
