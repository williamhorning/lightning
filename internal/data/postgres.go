package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

type postgresDatabase struct {
	db *sql.DB
}

func newPostgresDatabase(conn string) (Database, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, fmt.Errorf("failed to make connection pool: %w", err)
	}

	pgdb := &postgresDatabase{stdlib.OpenDBFromPool(pool)}

	if err := pgdb.setupDatabase(); err != nil {
		if closeErr := pgdb.db.Close(); closeErr != nil {
			log.Printf("data: failed to close connection: %v\n", err)
		}

		return nil, fmt.Errorf("failed to setup schema: %w", err)
	}

	return pgdb, nil
}

func (p *postgresDatabase) CreateBridge(bridgeData Bridge) error {
	return p.withTx(func(txn *sql.Tx) error {
		settings, err := json.Marshal(bridgeData.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}

		if _, err := txn.ExecContext(context.Background(), insertBridge, bridgeData.ID, settings); err != nil {
			return fmt.Errorf("failed to insert bridge: %w", err)
		}

		if _, err := txn.ExecContext(context.Background(), deleteBridgeChannelsQuery, bridgeData.ID); err != nil {
			return fmt.Errorf("failed to delete old channels: %w", err)
		}

		for _, channel := range bridgeData.Channels {
			data, err := json.Marshal(channel.Data)
			if err != nil {
				return fmt.Errorf("failed to marshal channel data: %w", err)
			}

			disabled, err := json.Marshal(channel.Disabled)
			if err != nil {
				return fmt.Errorf("failed to marshal channel disable information: %w", err)
			}

			if _, err := txn.ExecContext(context.Background(), insertChannel,
				bridgeData.ID, channel.ID, data, disabled); err != nil {
				return fmt.Errorf("failed to insert channel: %w", err)
			}
		}

		return nil
	})
}

func (p *postgresDatabase) GetBridge(brID string) (Bridge, error) {
	var (
		bridgeData Bridge
		settings   json.RawMessage
	)

	bridgeData.ID = brID

	if err := p.db.QueryRowContext(context.Background(), selectBridgeSettingsByID, brID).Scan(&settings); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Bridge{}, nil
		}

		return Bridge{}, fmt.Errorf("failed to query bridge settings: %w", err)
	}

	if err := json.Unmarshal(settings, &bridgeData.Settings); err != nil {
		return Bridge{}, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	rows, err := p.db.QueryContext(context.Background(), selectBridgeChannelsQuery, brID)
	if err != nil {
		return Bridge{}, fmt.Errorf("failed to query channels: %w", err)
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("data: failed to close rows: %v\n", err)
		}
	}()

	for rows.Next() {
		channel, err := getChannelRow(rows)
		if err != nil {
			return Bridge{}, fmt.Errorf("failed to get channels: %w", err)
		}

		bridgeData.Channels = append(bridgeData.Channels, channel)
	}

	if err := rows.Err(); err != nil {
		return Bridge{}, fmt.Errorf("failed to iterate channels: %w", err)
	}

	return bridgeData, nil
}

func (p *postgresDatabase) GetBridgeByChannel(chID string) (Bridge, error) {
	var bID string

	err := p.db.QueryRowContext(context.Background(),
		selectBridgeByChannelQuery, chID).Scan(&bID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Bridge{}, nil
		}

		return Bridge{}, fmt.Errorf("failed to query channel in bridge: %w", err)
	}

	return p.GetBridge(bID)
}

func (p *postgresDatabase) CreateMessage(message BridgeMessageCollection) error {
	data, err := json.Marshal(message.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	return p.exec(insertMessage, message.ID, message.BridgeID, data)
}

func (p *postgresDatabase) GetMessage(msgID string) (BridgeMessageCollection, error) {
	var (
		message BridgeMessageCollection
		data    sql.NullString
	)

	err := p.db.QueryRowContext(context.Background(),
		selectMessageCollectionQuery, msgID).
		Scan(&message.ID, &message.BridgeID, &data)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return BridgeMessageCollection{}, fmt.Errorf("failed to query message: %w", err)
	} else if errors.Is(err, sql.ErrNoRows) {
		return BridgeMessageCollection{}, nil
	}

	if err := json.Unmarshal([]byte(data.String), &message.Messages); err != nil {
		return BridgeMessageCollection{}, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return message, nil
}

func (p *postgresDatabase) DeleteMessage(id string) error {
	var realID string

	err := p.db.QueryRowContext(context.Background(), selectMessageIDQuery, id).Scan(&realID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to query message: %w", err)
	}

	if realID != "" {
		return p.exec(deleteMessageCollectionQuery, realID)
	}

	return nil
}

func (p *postgresDatabase) setupDatabase() error {
	if err := p.exec(createTables); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	version := "0.8.1"

	err := p.db.QueryRowContext(context.Background(), selectDatabaseVersionQuery).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		if err = p.exec(insertDatabaseVersionQuery); err != nil {
			return fmt.Errorf("failed to get init version: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to get db version: %w", err)
	}

	if version != "0.8.1" {
		if version == "0.8.0" {
			log.Println("data: migration from 0.8.0 to 0.8.1 isn't supported. use 0.8.0-beta.8 to migrate")
		}

		return UnsupportedDatabaseTypeError{}
	}

	return nil
}

func (p *postgresDatabase) exec(query string, args ...any) error {
	if _, err := p.db.ExecContext(context.Background(), query, args...); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}

func (p *postgresDatabase) withTx(txnfn func(*sql.Tx) error) error {
	txn, err := p.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to begin txn: %w", err)
	}

	defer func() {
		if err := txn.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Printf("data: txn rollback failed: %v\n", err)
		}
	}()

	if err := txnfn(txn); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return fmt.Errorf("failed to commit txn: %w", err)
	}

	return nil
}

func getChannelRow(rows *sql.Rows) (BridgeChannel, error) {
	var (
		channel        BridgeChannel
		data, disabled json.RawMessage
	)

	if err := rows.Scan(&channel.ID, &data, &disabled); err != nil {
		return BridgeChannel{}, fmt.Errorf("failed to scan channel row: %w", err)
	}

	if err := json.Unmarshal(data, &channel.Data); err != nil {
		return BridgeChannel{}, fmt.Errorf("failed to unmarshal channel data: %w", err)
	}

	if err := json.Unmarshal(disabled, &channel.Disabled); err != nil {
		return BridgeChannel{}, fmt.Errorf("failed to unmarshal disabled information: %w", err)
	}

	return channel, nil
}
