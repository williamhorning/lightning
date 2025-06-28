package lightning

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	ctx  context.Context
}

func newPostgresDatabase(connection string) (Database, error) {
	ctx := context.Background()
	conn, err := pgxpool.New(ctx, connection)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, sqlCreateTables)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return &postgresDatabase{conn, ctx}, nil
}

func (p *postgresDatabase) createBridge(bridge Bridge) error {
	channels, err := json.Marshal(bridge.Channels)
	if err != nil {
		return err
	}

	settings, err := json.Marshal(bridge.Settings)
	if err != nil {
		return err
	}

	_, err = p.conn.Exec(p.ctx, sqlInsertBridge, bridge.ID, bridge.Name, channels, settings)
	return err
}

func (p *postgresDatabase) getBridge(id string) (Bridge, error) {
	row := p.conn.QueryRow(p.ctx, `
		SELECT id, name, channels, settings 
		FROM bridges 
		WHERE id = $1
	`, id)
	return handleBridgeRow(row)
}

func (p *postgresDatabase) getBridgeByChannel(channelID string) (Bridge, error) {
	row := p.conn.QueryRow(p.ctx, `
        SELECT * FROM bridges 
        WHERE channels @> jsonb_build_array(jsonb_build_object('id', $1))
	`, channelID)
	return handleBridgeRow(row)
}

func (p *postgresDatabase) createMessage(message BridgeMessageCollection) error {
	channels, err := json.Marshal(message.Channels)
	if err != nil {
		return err
	}

	messages, err := json.Marshal(message.Messages)
	if err != nil {
		return err
	}

	settings, err := json.Marshal(message.Settings)
	if err != nil {
		return err
	}

	_, err = p.conn.Exec(p.ctx, sqlInsertMessage, message.ID, message.Name, message.BridgeID, channels, messages, settings)
	return err
}

func (p *postgresDatabase) deleteMessage(id string) error {
	_, err := p.conn.Exec(p.ctx, `DELETE FROM bridge_messages WHERE id = $1`, id)
	return err
}

func (p *postgresDatabase) getMessage(id string) (BridgeMessageCollection, error) {
	row := p.conn.QueryRow(p.ctx, `
		SELECT * FROM bridge_messages
		WHERE id = $1 OR jsonb_path_exists(messages, '$[*].id ? (@ == $1)', $1)
	`, id)
	return handleMessageRow(row)
}

func (p *postgresDatabase) GetAllBridges() ([]Bridge, error) {
	defer startSpinner().Stop()

	rows, err := p.conn.Query(p.ctx, `SELECT id, name, channels, settings FROM bridges`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bridges []Bridge
	for rows.Next() {
		bridge, err := handleBridgeRow(rows)
		if err != nil {
			return nil, err
		}
		bridges = append(bridges, bridge)
	}

	return bridges, rows.Err()
}

func (p *postgresDatabase) GetAllMessages() ([]BridgeMessageCollection, error) {
	defer startSpinner().Stop()

	rows, err := p.conn.Query(p.ctx, `
		SELECT id, name, bridge_id, channels, messages, settings
		FROM bridge_messages
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []BridgeMessageCollection
	for rows.Next() {
		message, err := handleMessageRow(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}

	return messages, rows.Err()
}

func (p *postgresDatabase) SetAllBridges(bridges []Bridge) error {
	defer startSpinner().Stop()

	for _, bridge := range bridges {
		if err := p.createBridge(bridge); err != nil {
			return err
		}
	}
	return nil
}

func (p *postgresDatabase) SetAllMessages(messages []BridgeMessageCollection) error {
	defer startSpinner().Stop()

	batch := &pgx.Batch{}
	for _, message := range messages {
		channels, err := json.Marshal(message.Channels)
		if err != nil {
			return err
		}

		messagesJSON, err := json.Marshal(message.Messages)
		if err != nil {
			return err
		}

		settings, err := json.Marshal(message.Settings)
		if err != nil {
			return err
		}

		batch.Queue(sqlInsertMessage, message.ID, message.Name, message.BridgeID, channels, messagesJSON, settings)
	}

	br := p.conn.SendBatch(p.ctx, batch)
	defer br.Close()

	for range messages {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func handleBridgeRow(row pgx.Row) (Bridge, error) {
	var bridge Bridge
	var channelsJSON, settingsJSON []byte

	if err := row.Scan(&bridge.ID, &bridge.Name, &channelsJSON, &settingsJSON); err != nil {
		return Bridge{}, err
	}

	if err := json.Unmarshal(channelsJSON, &bridge.Channels); err != nil {
		return Bridge{}, err
	}

	if err := json.Unmarshal(settingsJSON, &bridge.Settings); err != nil {
		return Bridge{}, err
	}

	return bridge, nil
}

func handleMessageRow(row pgx.Row) (BridgeMessageCollection, error) {
	var message BridgeMessageCollection
	var channelsJSON, messagesJSON, settingsJSON []byte

	if err := row.Scan(&message.ID, &message.Name, &message.BridgeID, &channelsJSON, &messagesJSON, &settingsJSON); err != nil {
		return BridgeMessageCollection{}, err
	}

	if err := json.Unmarshal(channelsJSON, &message.Channels); err != nil {
		return BridgeMessageCollection{}, err
	}

	if err := json.Unmarshal(messagesJSON, &message.Messages); err != nil {
		return BridgeMessageCollection{}, err
	}

	if err := json.Unmarshal(settingsJSON, &message.Settings); err != nil {
		return BridgeMessageCollection{}, err
	}

	return message, nil
}
