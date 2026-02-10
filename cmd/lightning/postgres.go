package main

import (
	"context"
	"errors"
	"fmt"

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
	return p.exec(`INSERT INTO bridges (id, settings) VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET allow_everyone = EXCLUDED.allow_everyone
		WHERE bridges.allow_everyone IS DISTINCT FROM EXCLUDED.allow_everyone;`,
		bridge.ID, bridge.AllowEveryone)
}

func (p *database) insertChannel(bridge string, ch bridgeChannel) error {
	return p.exec(`INSERT INTO bridge_channels (bridge_id, channel_id, data, disabled_read, disabled_write)
		VALUES ($1, $2, $3, $4, $5);`, bridge, ch.ID, ch.Data, ch.DisabledRead, ch.DisabledWrite)
}

func (p *database) disableChannel(ch string, read, write bool, data map[string]string) error {
	return p.exec(`UPDATE bridge_channels SET disabled_read = $1, disabled_write = $2, data = $3
		WHERE channel_id = $4;`, read, write, data, ch)
}

func (p *database) deleteChannel(channel string) error {
	return p.exec(`DELETE FROM bridge_channels WHERE channel_id = $1;`, channel)
}

func (p *database) getBridge(bridgeID string) (bridge, error) {
	var result bridge

	if err := p.pool.QueryRow(context.Background(), `SELECT id, allow_everyone FROM bridges WHERE id = $1;`,
		bridgeID).Scan(&result.ID, &result.AllowEveryone); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return bridge{}, nil
		}

		return bridge{}, fmt.Errorf("failed to query allow_everyone: %w", err)
	}

	rows, err := p.pool.Query(context.Background(), `
		SELECT channel_id, COALESCE(data, '{}') as data, disabled_read, disabled_write
		FROM bridge_channels
		WHERE bridge_id = $1;
	`, bridgeID)
	if err != nil {
		return bridge{}, fmt.Errorf("query channels: %w", err)
	}
	defer rows.Close()

	channels, err := pgx.CollectRows(rows, pgx.RowToStructByName[bridgeChannel])
	if err != nil {
		return bridge{}, fmt.Errorf("collect channels: %w", err)
	}

	result.Channels = channels

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

func (p *database) insertMessage(eventID, bridgeID string, ids channelMessageSet) error {
	if _, err := p.pool.Exec(context.Background(),
		`INSERT INTO bridge_messages_v2 (event_id, bridge_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;`,
		eventID, bridgeID,
	); err != nil {
		return fmt.Errorf("failed to insert first: %w", err)
	}

	for _, channel := range ids {
		for _, msgID := range channel.MessageIDs {
			if _, err := p.pool.Exec(context.Background(),
				`INSERT INTO bridge_message_data (event_id, channel_id, message_id)
				 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING;`, eventID, channel.ChannelID, msgID,
			); err != nil {
				return fmt.Errorf("failed to insert data: %w", err)
			}
		}
	}

	return nil
}

func (p *database) getMessage(event string) (channelMessageSet, error) {
	ctx := context.Background()

	var eventID string

	err := p.pool.QueryRow(ctx, `
		SELECT event_id FROM bridge_message_data WHERE message_id = $1 UNION
		SELECT event_id FROM bridge_messages_v2 WHERE event_id = $1 LIMIT 1;
	`, event).Scan(&eventID)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	rows, err := p.pool.Query(ctx, `
		SELECT channel_id, array_agg(message_id ORDER BY message_id) AS ids
		FROM bridge_message_data WHERE event_id = $1 GROUP BY channel_id;
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[channelMessage])
	if err != nil {
		return nil, fmt.Errorf("failed to collect: %w", err)
	}

	return result, nil
}

func (p *database) getOriginalMessage(event string) (channelMessageSet, error) {
	var logicalEvent string

	err := p.pool.QueryRow(context.Background(), `
		SELECT event_id
		FROM bridge_message_data
		WHERE message_id = $1
		LIMIT 1;
	`, event).Scan(&logicalEvent)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}

	rows, err := p.pool.Query(context.Background(), `
		SELECT channel_id, array_agg(message_id ORDER BY message_id) AS ids
		FROM bridge_message_data WHERE event_id = $1 GROUP BY channel_id;
	`, logicalEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to select: %w", err)
	}
	defer rows.Close()

	result, err := pgx.CollectRows(rows, pgx.RowToStructByName[channelMessage])
	if err != nil {
		return nil, fmt.Errorf("failed to collect: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	return result, nil
}

func (p *database) deleteMessage(id string) error {
	if _, err := p.pool.Exec(context.Background(),
		`DELETE FROM bridge_messages_v2 WHERE event_id = (
			SELECT event_id FROM bridge_message_data WHERE message_id = $1 LIMIT 1
		);`, id); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

func (p *database) setupDatabase() error { //nolint:revive,cyclop,funlen
	if err := p.exec(`
		CREATE TABLE IF NOT EXISTS bridges (
			id TEXT PRIMARY KEY,
			allow_everyone BOOLEAN NOT NULL DEFAULT false
		);

		CREATE TABLE IF NOT EXISTS bridge_channels (
			bridge_id TEXT NOT NULL REFERENCES bridges(id) ON DELETE CASCADE,
			channel_id TEXT NOT NULL PRIMARY KEY,
			data JSONB DEFAULT '{}'::jsonb,
			disabled_read BOOLEAN NOT NULL DEFAULT false,
			disabled_write BOOLEAN NOT NULL DEFAULT false
		);

		CREATE TABLE IF NOT EXISTS bridge_messages_v2 (
			event_id   TEXT PRIMARY KEY,
			bridge_id  TEXT NOT NULL REFERENCES bridges(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);

		CREATE TABLE IF NOT EXISTS bridge_message_data (
			event_id   TEXT NOT NULL REFERENCES bridge_messages_v2(event_id) ON DELETE CASCADE,
			channel_id TEXT NOT NULL REFERENCES bridge_channels(channel_id) ON DELETE CASCADE,
			message_id TEXT NOT NULL,
			PRIMARY KEY (event_id, channel_id, message_id)
		);

		CREATE TABLE IF NOT EXISTS lightning (
			prop  TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_bridge_channels_channel_id ON bridge_channels (channel_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_channels_bridge_id ON bridge_channels (bridge_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_message_data_event ON bridge_message_data (event_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_message_data_channel ON bridge_message_data (channel_id);
		CREATE INDEX IF NOT EXISTS idx_bridge_message_data_message_id ON bridge_message_data (message_id);
		`,
	); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	var version string

	err := p.pool.QueryRow(context.Background(), `SELECT value FROM lightning WHERE prop = 'db_data_version';`).
		Scan(&version)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err = p.exec(`INSERT INTO lightning (prop, value) VALUES ('db_data_version', '0.8.4');`); err != nil {
			return fmt.Errorf("failed to set database version d0.8.4: %w", err)
		}

		return nil
	case err != nil:
		return fmt.Errorf("failed to get database version: %w", err)
	case version == "0.8.4":
		return nil
	case version == "0.8.3":
		if err := p.exec(`INSERT INTO bridge_messages_v2 (event_id, bridge_id)
			SELECT id, bridge_id FROM bridge_messages WHERE bridge_id IN (SELECT id FROM bridges)
			ON CONFLICT DO NOTHING;`); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(`INSERT INTO bridge_message_data (message_id, event_id, channel_id)
			SELECT DISTINCT msg_id AS message_id, b.id AS event_id, m.value ->> 'channel_id' AS channel_id
			FROM bridge_messages b CROSS JOIN LATERAL jsonb_array_elements(b.messages) AS m(value)
			CROSS JOIN LATERAL (SELECT jsonb_array_elements_text(m.value -> 'message_ids') AS msg_id
			WHERE jsonb_typeof(m.value -> 'message_ids') = 'array' UNION ALL
			SELECT m.value ->> 'message_ids' AS msg_id WHERE jsonb_typeof(m.value -> 'message_ids') = 'string'
			) AS msg WHERE b.bridge_id IN (SELECT id FROM bridges)
			AND (m.value ->> 'channel_id') IN (SELECT id FROM bridge_channels);`); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(
			`ALTER TABLE bridges ADD COLUMN IF NOT EXISTS allow_everyone BOOLEAN NOT NULL DEFAULT false;`,
		); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(`ALTER TABLE bridge_channels
			ADD COLUMN IF NOT EXISTS disabled_read BOOLEAN NOT NULL DEFAULT false,
			ADD COLUMN IF NOT EXISTS disabled_write BOOLEAN NOT NULL DEFAULT false,
			DROP CONSTRAINT bridge_channels_pkey,
			ADD CONSTRAINT bridge_channels_pkey PRIMARY KEY (channel_id);
		`); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(
			`UPDATE bridges SET allow_everyone = COALESCE((settings ->> 'allow_everyone')::boolean, false);`,
		); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(
			`UPDATE bridge_channels SET disabled_read = COALESCE((disabled ->> 'read')::boolean, false),
				disabled_write = COALESCE((disabled ->> 'write')::boolean, false);`,
		); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(`ALTER TABLE bridges DROP COLUMN IF EXISTS settings;`); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		if err := p.exec(`ALTER TABLE bridge_channels DROP COLUMN IF EXISTS disabled;`); err != nil {
			return fmt.Errorf("failed to migrate database version d0.8.3 → d0.8.4: %w", err)
		}

		return p.exec(`UPDATE lightning SET value = '0.8.4';`)
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
