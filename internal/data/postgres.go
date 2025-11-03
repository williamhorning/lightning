package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresDatabase struct {
	pool *pgxpool.Pool
}

func newPostgresDatabase(conn string) (Database, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, fmt.Errorf("failed to make connection pool: %w", err)
	}

	database := &postgresDatabase{pool}

	if err := database.setupDatabase(); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to setup schema: %w", err)
	}

	return database, nil
}

func (p *postgresDatabase) CreateBridge(bridge Bridge) error {
	return p.withTx(func(ctx context.Context, txn pgx.Tx) error {
		settings, err := json.Marshal(bridge.Settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}

		if _, err := txn.Exec(ctx, insertBridge, bridge.ID, settings); err != nil {
			return fmt.Errorf("insert bridge: %w", err)
		}

		if _, err := txn.Exec(ctx, deleteBridgeChannelsQuery, bridge.ID); err != nil {
			return fmt.Errorf("delete old channels: %w", err)
		}

		for _, channel := range bridge.Channels {
			data, err := json.Marshal(channel.Data)
			if err != nil {
				return fmt.Errorf("marshal channel data: %w", err)
			}

			disabled, err := json.Marshal(channel.Disabled)
			if err != nil {
				return fmt.Errorf("marshal channel disabled: %w", err)
			}

			if _, err := txn.Exec(ctx, insertChannel, bridge.ID, channel.ID, data, disabled); err != nil {
				return fmt.Errorf("insert channel: %w", err)
			}
		}

		return nil
	})
}

func (p *postgresDatabase) GetBridge(bridgeID string) (Bridge, error) {
	var bridge Bridge

	bridge.ID = bridgeID

	var settings json.RawMessage
	if err := p.pool.QueryRow(context.Background(), selectBridgeSettingsByID, bridgeID).Scan(&settings); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Bridge{}, nil
		}

		return Bridge{}, fmt.Errorf("query bridge settings: %w", err)
	}

	if err := json.Unmarshal(settings, &bridge.Settings); err != nil {
		return Bridge{}, fmt.Errorf("unmarshal settings: %w", err)
	}

	rows, err := p.pool.Query(context.Background(), selectBridgeChannelsQuery, bridgeID)
	if err != nil {
		return Bridge{}, fmt.Errorf("query channels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return Bridge{}, err
		}

		bridge.Channels = append(bridge.Channels, ch)
	}

	if err := rows.Err(); err != nil {
		return Bridge{}, fmt.Errorf("iterate channels: %w", err)
	}

	return bridge, nil
}

func (p *postgresDatabase) GetBridgeByChannel(channelID string) (Bridge, error) {
	var bridgeID string

	err := p.pool.QueryRow(context.Background(), selectBridgeByChannelQuery, channelID).Scan(&bridgeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bridge{}, nil
	} else if err != nil {
		return Bridge{}, fmt.Errorf("query bridge by channel: %w", err)
	}

	return p.GetBridge(bridgeID)
}

func (p *postgresDatabase) CreateMessage(msg BridgeMessageCollection) error {
	data, err := json.Marshal(msg.Messages)
	if err != nil {
		return fmt.Errorf("marshal messages: %w", err)
	}

	return p.exec(insertMessage, msg.ID, msg.BridgeID, data)
}

func (p *postgresDatabase) GetMessage(msgID string) (BridgeMessageCollection, error) {
	var (
		msg  BridgeMessageCollection
		data string
	)

	err := p.pool.QueryRow(context.Background(), selectMessageCollectionQuery, msgID).
		Scan(&msg.ID, &msg.BridgeID, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return BridgeMessageCollection{}, nil
	} else if err != nil {
		return BridgeMessageCollection{}, fmt.Errorf("query message: %w", err)
	}

	if err := json.Unmarshal([]byte(data), &msg.Messages); err != nil {
		return BridgeMessageCollection{}, fmt.Errorf("unmarshal messages: %w", err)
	}

	return msg, nil
}

func (p *postgresDatabase) DeleteMessage(id string) error {
	var realID string

	err := p.pool.QueryRow(context.Background(), selectMessageIDQuery, id).Scan(&realID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return fmt.Errorf("query message ID: %w", err)
	}

	return p.exec(deleteMessageCollectionQuery, realID)
}

func (p *postgresDatabase) setupDatabase() error {
	if err := p.exec(createTables); err != nil {
		return fmt.Errorf("create tables: %w", err)
	}

	var version string

	err := p.pool.QueryRow(context.Background(), selectDatabaseVersionQuery).Scan(&version)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err = p.exec(insertDatabaseVersionQuery); err != nil {
			return fmt.Errorf("init version: %w", err)
		}

		return nil
	case err != nil:
		return fmt.Errorf("get db version: %w", err)
	case version == "0.8.2":
		return nil
	case version == "0.8.1":
		return p.exec(`UPDATE lightning SET value='0.8.2' WHERE prop='db_data_version';`)
	default:
		log.Println("migration from versions before v0.8.0-beta.8 not supported.")

		return UnsupportedDatabaseTypeError{}
	}
}

func (p *postgresDatabase) exec(query string, args ...any) error {
	_, err := p.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}

	return nil
}

func (p *postgresDatabase) withTx(txnfn func(context.Context, pgx.Tx) error) error {
	txn, err := p.pool.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err := txn.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("txn rollback failed: %v", err)
		}
	}()

	if err := txnfn(context.Background(), txn); err != nil {
		return err
	}

	if err := txn.Commit(context.Background()); err != nil {
		return fmt.Errorf("failed committing txn: %w", err)
	}

	return nil
}

func scanChannel(rows pgx.Rows) (BridgeChannel, error) {
	var (
		channel   BridgeChannel
		data, dis []byte
	)
	if err := rows.Scan(&channel.ID, &data, &dis); err != nil {
		return channel, fmt.Errorf("scan channel row: %w", err)
	}

	if err := json.Unmarshal(data, &channel.Data); err != nil {
		return channel, fmt.Errorf("unmarshal channel data: %w", err)
	}

	if err := json.Unmarshal(dis, &channel.Disabled); err != nil {
		return channel, fmt.Errorf("unmarshal disabled: %w", err)
	}

	return channel, nil
}
