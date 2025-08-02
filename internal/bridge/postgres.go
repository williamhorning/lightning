package bridge

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/williamhorning/lightning/pkg/lightning"
)

type postgresDatabase struct {
	db *sql.DB
}

func newPostgresDatabase(conn string) (Database, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, wrapErr(err, "create connection pool")
	}

	pgdb := &postgresDatabase{stdlib.OpenDBFromPool(pool)}

	if err := pgdb.setupDatabase(); err != nil {
		if err = pgdb.db.Close(); err != nil {
			slog.Error("failed to close database connection", "err", err)
		}

		return nil, wrapErr(err, "setup schema")
	}

	return pgdb, nil
}

func wrapErr(err error, msg string) error {
	return lightning.LogError(err, msg, nil, nil)
}

func (p *postgresDatabase) exec(query string, args ...interface{}) error {
	if _, err := p.db.ExecContext(context.Background(), query, args...); err != nil {
		return wrapErr(err, "exec failed")
	}

	return nil
}

func (p *postgresDatabase) withTx(txnfn func(*sql.Tx) error) error {
	txn, err := p.db.BeginTx(context.Background(), nil)
	if err != nil {
		return wrapErr(err, "begin tx")
	}

	defer func() {
		if err := txn.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.Warn("tx rollback failed", "err", err)
		}
	}()

	if err := txnfn(txn); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		return wrapErr(err, "commit tx")
	}

	return nil
}

func (p *postgresDatabase) createBridge(bridgeData bridge) error {
	return p.withTx(func(txn *sql.Tx) error {
		settings, err := json.Marshal(bridgeData.Settings)
		if err != nil {
			return wrapErr(err, "marshal settings")
		}

		if _, err := txn.ExecContext(context.Background(), insertBridge, bridgeData.ID, settings); err != nil {
			return wrapErr(err, "insert bridge")
		}

		if _, err := txn.ExecContext(context.Background(), deleteBridgeChannelsQuery, bridgeData.ID); err != nil {
			return wrapErr(err, "delete old channels")
		}

		for _, channel := range bridgeData.Channels {
			data, err := json.Marshal(channel.Data)
			if err != nil {
				return wrapErr(err, "marshal channel data")
			}

			disabled, err := json.Marshal(channel.Disabled)
			if err != nil {
				return wrapErr(err, "marshal channel disabled")
			}

			if _, err := txn.ExecContext(context.Background(), insertChannel,
				bridgeData.ID, channel.ID, data, disabled); err != nil {
				return wrapErr(err, "insert channel")
			}
		}

		return nil
	})
}

func (p *postgresDatabase) getBridge(brID string) (bridge, error) {
	var (
		bridgeData bridge
		settings   json.RawMessage
	)

	bridgeData.ID = brID

	if err := p.db.QueryRowContext(context.Background(), selectBridgeSettingsByID, brID).Scan(&settings); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return bridge{}, nil
		}

		return bridgeData, wrapErr(err, "query bridge settings")
	}

	if err := json.Unmarshal(settings, &bridgeData.Settings); err != nil {
		return bridgeData, wrapErr(err, "unmarshal settings")
	}

	rows, err := p.db.QueryContext(context.Background(), selectBridgeChannelsQuery, brID)
	if err != nil {
		return bridgeData, wrapErr(err, "query channels")
	}

	defer func() {
		if err := rows.Close(); err != nil {
			slog.Warn("failed to close rows", "err", err)
		}
	}()

	for rows.Next() {
		channel, err := getChannelRow(rows)
		if err != nil {
			return bridge{}, wrapErr(err, "get channel row")
		}

		bridgeData.Channels = append(bridgeData.Channels, channel)
	}

	if err := rows.Err(); err != nil {
		return bridge{}, wrapErr(err, "iterate channels")
	}

	return bridgeData, nil
}

func getChannelRow(rows *sql.Rows) (bridgeChannel, error) {
	var (
		channel        bridgeChannel
		data, disabled json.RawMessage
	)

	if err := rows.Scan(&channel.ID, &data, &disabled); err != nil {
		return bridgeChannel{}, wrapErr(err, "scan channel")
	}

	if err := json.Unmarshal(data, &channel.Data); err != nil {
		return bridgeChannel{}, wrapErr(err, "unmarshal channel data")
	}

	if err := json.Unmarshal(disabled, &channel.Disabled); err != nil {
		return bridgeChannel{}, wrapErr(err, "unmarshal disabled flag")
	}

	return channel, nil
}

func (p *postgresDatabase) getBridgeByChannel(chID string) (bridge, error) {
	var bID string

	err := p.db.QueryRowContext(context.Background(),
		selectBridgeByChannelQuery, chID).Scan(&bID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return bridge{}, nil
		}

		return bridge{}, wrapErr(err, "query bridge by channel")
	}

	return p.getBridge(bID)
}

func (p *postgresDatabase) createMessage(message bridgeMessageCollection) error {
	data, err := json.Marshal(message.Messages)
	if err != nil {
		return wrapErr(err, "marshal messages")
	}

	return p.exec(insertMessage, message.ID, message.BridgeID, data)
}

func (p *postgresDatabase) getMessage(msgID string) (bridgeMessageCollection, error) {
	var (
		message bridgeMessageCollection
		data    json.RawMessage
	)

	err := p.db.QueryRowContext(context.Background(),
		selectMessageCollectionQuery, msgID).
		Scan(&message.ID, &message.BridgeID, &data)
	if err != nil {
		return message, wrapErr(err, "query message")
	}

	if err := json.Unmarshal(data, &message.Messages); err != nil {
		return message, wrapErr(err, "unmarshal messages")
	}

	return message, nil
}

func (p *postgresDatabase) deleteMessage(id string) error {
	var realID string

	err := p.db.QueryRowContext(context.Background(), selectMessageIDQuery, id).Scan(&realID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return wrapErr(err, "query message id")
	}

	if realID != "" {
		return p.exec(deleteMessageCollectionQuery, realID)
	}

	return nil
}

func (p *postgresDatabase) setupDatabase() error {
	if err := p.exec(createTables); err != nil {
		return wrapErr(err, "create tables")
	}

	version := "0.8.1"

	err := p.db.QueryRowContext(context.Background(), selectDatabaseVersionQuery).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		if err = p.exec(insertDatabaseVersionQuery); err != nil {
			return wrapErr(err, "init db version")
		}
	} else if err != nil {
		return wrapErr(err, "get db version")
	}

	if version != "0.8.1" {
		return p.migrateDatabase(version)
	}

	return nil
}

//go:embed migration.sql
var zeroEightZeroMigrationQuery string

func (p *postgresDatabase) migrateDatabase(version string) error {
	if version != "0.8.0" {
		return UnsupportedDatabaseTypeError{}
	}

	slog.Info("migrating database", "from", version)

	return p.withTx(func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(context.Background(), zeroEightZeroMigrationQuery); err != nil {
			return wrapErr(err, "exec migration")
		}

		return nil
	})
}
