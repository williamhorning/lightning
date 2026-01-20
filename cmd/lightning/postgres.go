package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type database struct {
	pool *pgxpool.Pool
}

func newDatabase(conn string) (*database, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, fmt.Errorf("failed to make connection pool: %w", err)
	}

	database := &database{pool}

	if err = database.setupDatabase(); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to setup schema: %w", err)
	}

	return database, nil
}

func (p *database) createBridge(bridge bridge) error {
	txn, err := p.pool.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to start txn: %w", err)
	}

	defer func() {
		if err := txn.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("bridge: failed to rollback txn: %v\n", err)
		}
	}()

	settings, err := json.Marshal(bridge.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if _, err = txn.Exec(context.Background(), `
		INSERT INTO bridges (id, settings) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET settings = EXCLUDED.settings
		WHERE bridges.settings IS DISTINCT FROM EXCLUDED.settings;`,
		bridge.ID, settings); err != nil {
		return fmt.Errorf("failed inserting bridge: %w", err)
	}

	if _, err = txn.Exec(context.Background(), `DELETE FROM bridge_channels WHERE bridge_id = $1;`,
		bridge.ID); err != nil {
		return fmt.Errorf("failed deleting old channels: %w", err)
	}

	if err := setChannels(bridge, txn); err != nil {
		return err
	}

	if err := txn.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed committing txn: %w", err)
	}

	return nil
}

func setChannels(bridge bridge, txn pgx.Tx) error {
	for _, channel := range bridge.Channels {
		data, err := json.Marshal(channel.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal channel data: %w", err)
		}

		disabled, err := json.Marshal(channel.Disabled)
		if err != nil {
			return fmt.Errorf("failed to marshal channel disabled: %w", err)
		}

		if _, err := txn.Exec(context.Background(),
			`INSERT INTO bridge_channels (bridge_id, channel_id, data, disabled) VALUES ($1, $2, $3, $4);`,
			bridge.ID, channel.ID, data, disabled); err != nil {
			return fmt.Errorf("failed to insert channel: %w", err)
		}
	}

	return nil
}

func (p *database) getBridge(bridgeID string) (bridge, error) {
	var result bridge

	result.ID = bridgeID

	var settings json.RawMessage
	if err := p.pool.QueryRow(context.Background(), `SELECT settings FROM bridges WHERE id = $1;`,
		bridgeID).Scan(&settings); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return bridge{}, nil
		}

		return bridge{}, fmt.Errorf("failed to query bridge settings: %w", err)
	}

	if err := json.Unmarshal(settings, &result.Settings); err != nil {
		return bridge{}, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	rows, err := p.pool.Query(context.Background(), `SELECT channel_id, COALESCE(data, '{}'), disabled
		FROM bridge_channels WHERE bridge_id = $1;`, bridgeID)
	if err != nil {
		return bridge{}, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return bridge{}, err
		}

		result.Channels = append(result.Channels, ch)
	}

	if err := rows.Err(); err != nil {
		return bridge{}, fmt.Errorf("failed to iterate channels: %w", err)
	}

	return result, nil
}

func (p *database) getBridgeByChannel(channelID string) (bridge, error) {
	var bridgeID string

	err := p.pool.QueryRow(context.Background(), `SELECT bridge_id FROM bridge_channels WHERE channel_id = $1;`,
		channelID).Scan(&bridgeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return bridge{}, nil
	} else if err != nil {
		return bridge{}, fmt.Errorf("failed to query bridge by channel: %w", err)
	}

	return p.getBridge(bridgeID)
}

func (p *database) createMessage(msg bridgeMessageCollection) error {
	data, err := json.Marshal(msg.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	return p.exec(`
		INSERT INTO bridge_messages (id, bridge_id, messages)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE
		SET messages = EXCLUDED.messages, bridge_id = EXCLUDED.bridge_id
		WHERE bridge_messages.messages IS DISTINCT FROM EXCLUDED.messages;`,
		msg.ID, msg.BridgeID, data)
}

func (p *database) getMessage(msgID string) (bridgeMessageCollection, error) {
	var (
		msg  bridgeMessageCollection
		data string
	)

	err := p.pool.QueryRow(context.Background(), `
		SELECT id, bridge_id, messages FROM bridge_messages
		WHERE messages @> format('[{"message_ids":["%s"]}]', $1::text)::jsonb LIMIT 1;`,
		msgID).Scan(&msg.ID, &msg.BridgeID, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return bridgeMessageCollection{}, nil
	} else if err != nil {
		return bridgeMessageCollection{}, fmt.Errorf("failed to query message: %w", err)
	}

	if err := json.Unmarshal([]byte(data), &msg.Messages); err != nil {
		return bridgeMessageCollection{}, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return msg, nil
}

func (p *database) deleteMessage(id string) error {
	var realID string

	err := p.pool.QueryRow(context.Background(), `
		SELECT id FROM bridge_messages WHERE messages @> format('[{"message_ids":["%s"]}]', $1::text)::jsonb LIMIT 1;`,
		id).Scan(&realID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to query message ID: %w", err)
	}

	return p.exec(`DELETE FROM bridge_messages WHERE id = $1;`, realID)
}

func (p *database) setupDatabase() error {
	if err := p.exec(`
		CREATE TABLE IF NOT EXISTS bridges (
			id TEXT PRIMARY KEY,
			settings JSONB NOT NULL DEFAULT '{"allow_everyone": false}'::jsonb
		);

		CREATE TABLE IF NOT EXISTS bridge_channels (
			bridge_id TEXT NOT NULL REFERENCES bridges(id) ON DELETE CASCADE,
			channel_id TEXT NOT NULL UNIQUE,
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
		CREATE INDEX IF NOT EXISTS idx_bridge_messages_gin ON bridge_messages USING GIN (messages jsonb_path_ops);`,
	); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	var version string

	err := p.pool.QueryRow(context.Background(), `SELECT value FROM lightning WHERE prop = 'db_data_version';`).
		Scan(&version)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err = p.exec(`INSERT INTO lightning (prop, value) VALUES ('db_data_version', '0.8.3');`); err != nil {
			return fmt.Errorf("failed to set database version d0.8.3: %w", err)
		}

		return nil
	case err != nil:
		return fmt.Errorf("failed to get database version: %w", err)
	case version == "0.8.3":
		return nil
	case version == "0.8.1" || version == "0.8.3":
		if err := p.exec(`UPDATE bridge_channels SET data = CASE WHEN jsonb_typeof(data) = 'object' THEN
			(SELECT jsonb_object_agg(key, value) FROM jsonb_each_text(data)) ELSE NULL END;`); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.2 → d0.8.3: %w", err)
		}

		return p.exec(`UPDATE lightning SET value='0.8.3' WHERE prop='db_data_version';`)
	default:
		return unsupportedDatabaseVersionError{}
	}
}

func (p *database) exec(query string, args ...any) error {
	_, err := p.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func scanChannel(rows pgx.Rows) (bridgeChannel, error) {
	var (
		channel   bridgeChannel
		data, dis json.RawMessage
	)
	if err := rows.Scan(&channel.ID, &data, &dis); err != nil {
		return channel, fmt.Errorf("failed to scan channel row: %w", err)
	}

	if err := json.Unmarshal(data, &channel.Data); err != nil {
		return channel, fmt.Errorf("failed to unmarshal channel data: %w", err)
	}

	if err := json.Unmarshal(dis, &channel.Disabled); err != nil {
		return channel, fmt.Errorf("failed to unmarshal disabled: %w", err)
	}

	return channel, nil
}
