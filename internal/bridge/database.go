package bridge

import (
	"errors"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

var ErrUnsupportedDatabaseType = errors.New("unsupported database type, must be 'postgres' or 'redis'")

type Database interface {
	createBridge(bridge Bridge) error
	getBridge(id string) (Bridge, error)
	getBridgeByChannel(channelID string) (Bridge, error)
	createMessage(message BridgeMessageCollection) error
	deleteMessage(id string) error
	getMessage(id string) (BridgeMessageCollection, error)
	GetAllBridges() ([]Bridge, error)
	GetAllMessages() ([]BridgeMessageCollection, error)
	SetAllBridges(bridges []Bridge) error
	SetAllMessages(messages []BridgeMessageCollection) error
}

type DatabaseConfig struct {
	Type       string `toml:"type"`
	Connection string `toml:"connection"`
}

func (config DatabaseConfig) GetDatabase() (Database, error) {
	switch config.Type {
	case "postgres":
		return newPostgresDatabase(config.Connection)
	case "redis":
		return newRedisDatabase(config.Connection)
	default:
		return nil, ErrUnsupportedDatabaseType
	}
}

func startSpinner() *spinner.Spinner {
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
	spin.Start()
	return spin
}
