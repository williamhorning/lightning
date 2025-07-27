package bridge

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/williamhorning/lightning/pkg/lightning"
	_ "modernc.org/sqlite" // pure go sqlite driver
)

type sqliteDatabase struct {
	db *sql.DB
}

func newSQLiteDatabase(connection string) (Database, error) {
	conn, err := sql.Open("sqlite", connection)
	if err != nil {
		return nil, lightning.LogError(err, "failed to open db", nil, nil)
	}

	instance := &sqliteDatabase{db: conn}

	_, err = conn.ExecContext(context.Background(), prepareSQLiteQuery(sqlCreateTables))
	if err != nil {
		return nil, lightning.LogError(err, "failed to create tables", nil, nil)
	}

	return instance, nil
}

func (s *sqliteDatabase) createBridge(bridgeData bridge) error {
	channels, channelsErr := json.Marshal(bridgeData.Channels)

	settings, settingsErr := json.Marshal(bridgeData.Settings)
	if err := cmp.Or(channelsErr, settingsErr); err != nil {
		return lightning.LogError(err, "failed to marshal bridge", map[string]any{"br": bridgeData}, nil)
	}

	_, err := s.db.ExecContext(context.Background(), prepareSQLiteQuery(sqlInsertBridge),
		bridgeData.ID, bridgeData.Name, channels, settings)
	if err != nil {
		return lightning.LogError(err, "failed to create bridge", nil, nil)
	}

	return nil
}

func (s *sqliteDatabase) getBridge(id string) (bridge, error) {
	return handleSQLiteBridgeRow(
		s.db.QueryRowContext(context.Background(), prepareSQLiteQuery(sqlSelectBridgeByID), id),
	)
}

func (s *sqliteDatabase) getBridgeByChannel(channelID string) (bridge, error) {
	return handleSQLiteBridgeRow(
		s.db.QueryRowContext(context.Background(), prepareSQLiteQuery(sqlSelectBridgeByChannel), channelID),
	)
}

func (s *sqliteDatabase) createMessage(message bridgeMessageCollection) error {
	channels, channelsErr := json.Marshal(message.Channels)
	messages, messagesErr := json.Marshal(message.Messages)

	settings, settingsErr := json.Marshal(message.Settings)
	if err := cmp.Or(channelsErr, messagesErr, settingsErr); err != nil {
		return lightning.LogError(err, "failed to marshal message", map[string]any{"msg": message}, nil)
	}

	_, err := s.db.ExecContext(context.Background(), prepareSQLiteQuery(sqlInsertMessage), message.ID, message.Name,
		message.BridgeID, channels, messages, settings)
	if err != nil {
		return lightning.LogError(err, "failed to create message", nil, nil)
	}

	return nil
}

func (s *sqliteDatabase) deleteMessage(id string) error {
	_, err := s.db.ExecContext(context.Background(), prepareSQLiteQuery(sqlDeleteMessage), id)
	if err != nil {
		return lightning.LogError(err, "failed to delete message", map[string]any{"id": id}, nil)
	}

	return nil
}

func (s *sqliteDatabase) getMessage(id string) (bridgeMessageCollection, error) {
	return handleSQLiteMessageRow(s.db.QueryRowContext(context.Background(), prepareSQLiteQuery(sqlSelectMessage), id))
}

func prepareSQLiteQuery(query string) string {
	query = strings.ReplaceAll(query, "JSONB", "TEXT")
	query = strings.ReplaceAll(query, ":create_index:", "")
	query = strings.ReplaceAll(
		query,
		"CROSS JOIN jsonb_array_elements_text(msg->'id') AS id_element",
		"JOIN json_each(json_extract(msg.value, '$.id')) AS id_element ON 1=1",
	)
	query = strings.ReplaceAll(query, "jsonb_array_elements", "json_each")

	return query
}

func handleSQLiteBridgeRow(row *sql.Row) (bridge, error) {
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

func handleSQLiteMessageRow(row *sql.Row) (bridgeMessageCollection, error) {
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
