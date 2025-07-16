package bridge

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/williamhorning/lightning/pkg/lightning"
)

const (
	sqlCreateTables = `
		CREATE TABLE IF NOT EXISTS lightning (
			prop  TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		INSERT INTO lightning (prop, value)
		VALUES ('db_data_version', '0.8.0')
		ON CONFLICT (prop) DO NOTHING;

		CREATE TABLE IF NOT EXISTS bridges (
			id       TEXT PRIMARY KEY,
			name     TEXT NOT NULL,
			channels JSONB NOT NULL,
			settings JSONB NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_channels ON bridges USING GIN (channels);

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
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			channels = EXCLUDED.channels,
			settings = EXCLUDED.settings`

	sqlInsertMessage = `
		INSERT INTO bridge_messages (id, name, bridge_id, channels, messages, settings)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			channels = EXCLUDED.channels,
			messages = EXCLUDED.messages,
			settings = EXCLUDED.settings`
)

type postgresDatabase struct {
	conn *pgxpool.Pool
}

func newPostgresDatabase(connection string) (*postgresDatabase, error) {
	conn, err := pgxpool.New(context.Background(), connection)
	if err != nil {
		return nil, lightning.LogError(err, "failed to make pgxpool", nil, nil)
	}

	_, err = conn.Exec(context.Background(), sqlCreateTables)
	if err != nil {
		conn.Close()

		return nil, lightning.LogError(err, "failed to setup schema", nil, nil)
	}

	return &postgresDatabase{conn}, nil
}

func (p *postgresDatabase) createBridge(bridge bridge) error {
	channels, channelsErr := json.Marshal(bridge.Channels)
	settings, settingsErr := json.Marshal(bridge.Settings)

	err := cmp.Or(channelsErr, settingsErr)
	if err != nil {
		return lightning.LogError(err, "failed to marshal bridge", map[string]any{"br": bridge}, nil)
	}

	_, err = p.conn.Exec(context.Background(), sqlInsertBridge, bridge.ID, bridge.Name, channels, settings)
	if err != nil {
		return lightning.LogError(err, "failed to create bridge", nil, nil)
	}

	return nil
}

func (p *postgresDatabase) getBridge(id string) (bridge, error) {
	row := p.conn.QueryRow(context.Background(), `
		SELECT id, name, channels, settings 
		FROM bridges 
		WHERE id = $1
	`, id)

	return handleBridgeRow(row)
}

func (p *postgresDatabase) getBridgeByChannel(channelID string) (bridge, error) {
	row := p.conn.QueryRow(context.Background(), `
		SELECT id, name, channels, settings FROM bridges 
		WHERE EXISTS (
			SELECT 1 FROM jsonb_array_elements(channels) AS ch
			WHERE ch->>'id' = $1
		)`, channelID)

	return handleBridgeRow(row)
}

func (p *postgresDatabase) createMessage(message bridgeMessageCollection) error {
	channels, channelsErr := json.Marshal(message.Channels)
	messages, messagesErr := json.Marshal(message.Messages)
	settings, settingsErr := json.Marshal(message.Settings)

	err := cmp.Or(channelsErr, messagesErr, settingsErr)
	if err != nil {
		return lightning.LogError(err, "failed to marshal message", map[string]any{"msg": message}, nil)
	}

	_, err = p.conn.Exec(
		context.Background(),
		sqlInsertMessage,
		message.ID,
		message.Name,
		message.BridgeID,
		channels,
		messages,
		settings,
	)
	if err != nil {
		return lightning.LogError(err, "failed to create message", nil, nil)
	}

	return nil
}

func (p *postgresDatabase) deleteMessage(id string) error {
	_, err := p.conn.Exec(context.Background(), `DELETE FROM bridge_messages WHERE id = $1`, id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return lightning.LogError(err, "failed to delete message", map[string]any{"id": id}, nil)
	}

	return nil
}

func (p *postgresDatabase) getMessage(msgID string) (bridgeMessageCollection, error) {
	row := p.conn.QueryRow(context.Background(), `
		SELECT id, name, bridge_id, channels, messages, settings FROM bridge_messages
		WHERE id = $1 OR EXISTS (
			SELECT 1 FROM jsonb_array_elements(messages) AS msg
			CROSS JOIN jsonb_array_elements_text(msg->'id') AS id_element
			WHERE id_element = $1
		)
	`, msgID)

	return handleMessageRow(row)
}

func handleBridgeRow(row pgx.Row) (bridge, error) {
	var (
		bridgeValue                bridge
		channelsJSON, settingsJSON []byte
	)

	if err := row.Scan(&bridgeValue.ID, &bridgeValue.Name, &channelsJSON, &settingsJSON); err != nil &&
		!errors.Is(err, pgx.ErrNoRows) {
		return bridge{}, lightning.LogError(err, "failed to get bridge row", nil, nil)
	} else if errors.Is(err, pgx.ErrNoRows) {
		return bridge{}, nil
	}

	if err := cmp.Or(
		json.Unmarshal(channelsJSON, &bridgeValue.Channels),
		json.Unmarshal(settingsJSON, &bridgeValue.Settings),
	); err != nil {
		return bridge{}, lightning.LogError(err, "failed to unmarshal bridge row", nil, nil)
	}

	return bridgeValue, nil
}

func handleMessageRow(row pgx.Row) (bridgeMessageCollection, error) {
	var (
		message                                  bridgeMessageCollection
		channelsJSON, messagesJSON, settingsJSON []byte
	)

	err := row.Scan(&message.ID, &message.Name, &message.BridgeID, &channelsJSON, &messagesJSON, &settingsJSON)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return bridgeMessageCollection{}, lightning.LogError(err, "failed to get message row", nil, nil)
	} else if errors.Is(err, pgx.ErrNoRows) {
		return bridgeMessageCollection{}, nil
	}

	if err := cmp.Or(
		json.Unmarshal(channelsJSON, &message.Channels),
		json.Unmarshal(messagesJSON, &message.Messages),
		json.Unmarshal(settingsJSON, &message.Settings),
	); err != nil {
		return bridgeMessageCollection{}, lightning.LogError(err, "failed to unmarshal message row", nil, nil)
	}

	return message, nil
}
