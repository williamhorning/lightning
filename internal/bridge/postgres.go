package bridge

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" // driver for postgres
	"github.com/williamhorning/lightning/pkg/lightning"
	_ "modernc.org/sqlite" // pure go sqlite driver
)

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

type sqlDatabase struct {
	db     *sql.DB
	driver string
}

func newSQLDatabase(driver, connection string) (*sqlDatabase, error) {
	conn, err := sql.Open(driver, connection)
	if err != nil {
		return nil, lightning.LogError(err, "failed to open db", nil, nil)
	}

	conn.SetMaxIdleConns(0)

	instance := &sqlDatabase{db: conn, driver: driver}

	_, err = conn.ExecContext(context.Background(), instance.prepareQuery(sqlCreateTables))
	if err != nil {
		return nil, lightning.LogError(err, "failed to create tables", nil, nil)
	}

	return instance, nil
}

func (s *sqlDatabase) createBridge(bridgeData bridge) error {
	channels, channelsErr := json.Marshal(bridgeData.Channels)

	settings, settingsErr := json.Marshal(bridgeData.Settings)
	if err := cmp.Or(channelsErr, settingsErr); err != nil {
		return lightning.LogError(err, "failed to marshal bridge", map[string]any{"br": bridgeData}, nil)
	}

	_, err := s.db.ExecContext(context.Background(), s.prepareQuery(sqlInsertBridge),
		bridgeData.ID, bridgeData.Name, channels, settings)
	if err != nil {
		return lightning.LogError(err, "failed to create bridge", nil, nil)
	}

	return nil
}

func (s *sqlDatabase) getBridge(bridgeID string) (bridge, error) {
	return handleBridgeRow(s.db.QueryRowContext(context.Background(), s.prepareQuery(sqlSelectBridgeByID), bridgeID))
}

func (s *sqlDatabase) getBridgeByChannel(channelID string) (bridge, error) {
	return handleBridgeRow(
		s.db.QueryRowContext(context.Background(), s.prepareQuery(sqlSelectBridgeByChannel), channelID),
	)
}

func (s *sqlDatabase) createMessage(message bridgeMessageCollection) error {
	channels, channelsErr := json.Marshal(message.Channels)
	messages, messagesErr := json.Marshal(message.Messages)

	settings, settingsErr := json.Marshal(message.Settings)
	if err := cmp.Or(channelsErr, messagesErr, settingsErr); err != nil {
		return lightning.LogError(err, "failed to marshal message", map[string]any{"msg": message}, nil)
	}

	_, err := s.db.ExecContext(context.Background(), s.prepareQuery(sqlInsertMessage), message.ID, message.Name,
		message.BridgeID, channels, messages, settings)
	if err != nil {
		return lightning.LogError(err, "failed to create message", nil, nil)
	}

	return nil
}

func (s *sqlDatabase) deleteMessage(messageID string) error {
	_, err := s.db.ExecContext(context.Background(), s.prepareQuery(sqlDeleteMessage), messageID)
	if err != nil {
		return lightning.LogError(err, "failed to delete message", map[string]any{"id": messageID}, nil)
	}

	return nil
}

func (s *sqlDatabase) getMessage(msgID string) (bridgeMessageCollection, error) {
	return handleMessageRow(s.db.QueryRowContext(context.Background(), s.prepareQuery(sqlSelectMessage), msgID))
}

func (s *sqlDatabase) prepareQuery(query string) string {
	switch s.driver {
	case "pgx":
		query = strings.ReplaceAll(query, "?", "$")
		query = strings.ReplaceAll(
			query,
			":create_index:",
			"CREATE INDEX IF NOT EXISTS idx_channels ON bridges USING GIN (channels);",
		)
	case "sqlite":
		query = strings.ReplaceAll(query, "JSONB", "TEXT")
		query = strings.ReplaceAll(query, ":create_index:", "")
		query = strings.ReplaceAll(
			query,
			"CROSS JOIN jsonb_array_elements_text(msg->'id') AS id_element",
			"JOIN json_each(json_extract(msg.value, '$.id')) AS id_element ON 1=1",
		)
		query = strings.ReplaceAll(query, "jsonb_array_elements", "json_each")
	}

	return query
}

func handleBridgeRow(row *sql.Row) (bridge, error) {
	var bridgeData bridge

	var channelsData, settingsData []byte

	err := row.Scan(&bridgeData.ID, &bridgeData.Name, &channelsData, &settingsData)
	if errors.Is(err, sql.ErrNoRows) {
		return bridge{}, nil
	} else if err != nil {
		return bridge{}, lightning.LogError(err, "failed to scan bridge row", nil, nil)
	}

	if err := cmp.Or(
		json.Unmarshal(channelsData, &bridgeData.Channels),
		json.Unmarshal(settingsData, &bridgeData.Settings),
	); err != nil {
		return bridge{}, lightning.LogError(err, "failed to unmarshal bridge json", nil, nil)
	}

	return bridgeData, nil
}

func handleMessageRow(row *sql.Row) (bridgeMessageCollection, error) {
	var messages bridgeMessageCollection

	var channelsData, messagesData, settingsData []byte

	err := row.Scan(&messages.ID, &messages.Name, &messages.BridgeID, &channelsData, &messagesData, &settingsData)
	if errors.Is(err, sql.ErrNoRows) {
		return bridgeMessageCollection{}, nil
	} else if err != nil {
		return bridgeMessageCollection{}, lightning.LogError(err, "failed to scan message row", nil, nil)
	}

	if err := cmp.Or(
		json.Unmarshal(channelsData, &messages.Channels),
		json.Unmarshal(messagesData, &messages.Messages),
		json.Unmarshal(settingsData, &messages.Settings),
	); err != nil {
		return bridgeMessageCollection{}, lightning.LogError(err, "failed to unmarshal message json", nil, nil)
	}

	return messages, nil
}
