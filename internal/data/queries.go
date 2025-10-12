package data

const (
	createTables = `
		CREATE TABLE IF NOT EXISTS bridges (
			id TEXT PRIMARY KEY,
			settings JSONB NOT NULL DEFAULT '{"allow_everyone": false}'::jsonb
		);

		CREATE TABLE IF NOT EXISTS bridge_channels (
			bridge_id TEXT NOT NULL REFERENCES bridges(id) ON DELETE CASCADE,
			channel_id TEXT NOT NULL,
			data JSONB DEFAULT '{}'::jsonb,
			disabled JSONB NOT NULL DEFAULT '{"read": false, "write": false}'::jsonb,
			PRIMARY KEY (bridge_id, channel_id)
		);

		CREATE TABLE IF NOT EXISTS bridge_messages (
			id TEXT PRIMARY KEY,
			bridge_id TEXT NOT NULL REFERENCES bridges(id) ON DELETE CASCADE,
			messages JSONB NOT NULL DEFAULT '[]'::jsonb
		);

		CREATE TABLE IF NOT EXISTS lightning (
			prop  TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_bridge_channels_channel_id ON bridge_channels (channel_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_channels_bridge_id ON bridge_channels (bridge_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_messages_bridge_id ON bridge_messages (bridge_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_messages_gin ON bridge_messages USING GIN (messages jsonb_path_ops);`

	insertBridge = `
		INSERT INTO bridges (id, settings) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET settings = EXCLUDED.settings
		WHERE bridges.settings IS DISTINCT FROM EXCLUDED.settings;`

	insertChannel = `INSERT INTO bridge_channels (bridge_id, channel_id, data, disabled) VALUES ($1, $2, $3, $4);`

	insertMessage = `
		INSERT INTO bridge_messages (id, bridge_id, messages)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET messages = EXCLUDED.messages, bridge_id = EXCLUDED.bridge_id
		WHERE bridge_messages.messages IS DISTINCT FROM EXCLUDED.messages;`

	selectBridgeSettingsByID = `SELECT settings FROM bridges WHERE id = $1;`

	selectBridgeByChannelQuery = `SELECT bridge_id FROM bridge_channels WHERE channel_id = $1;`

	selectBridgeChannelsQuery = `
		SELECT channel_id, COALESCE(data, '{}'), disabled FROM bridge_channels WHERE bridge_id = $1;`

	selectMessageCollectionQuery = `
		SELECT id, bridge_id, messages FROM bridge_messages
		WHERE messages @> format('[{"message_ids":["%s"]}]', $1::text)::jsonb LIMIT 1;`

	selectMessageIDQuery = `
		SELECT id FROM bridge_messages
		WHERE messages @> format('[{"message_ids":["%s"]}]', $1::text)::jsonb LIMIT 1;`

	deleteBridgeChannelsQuery = `DELETE FROM bridge_channels WHERE bridge_id = $1;`

	deleteMessageCollectionQuery = `DELETE FROM bridge_messages WHERE id = $1;`

	selectDatabaseVersionQuery = `SELECT value FROM lightning WHERE prop = 'db_data_version';`

	insertDatabaseVersionQuery = `INSERT INTO lightning (prop, value) VALUES ('db_data_version', '0.8.2');`
)
