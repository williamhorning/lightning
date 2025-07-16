package bridge

import "errors"

// ErrUnsupportedDatabaseType is returned when an unsupported database type is given in configuration.
var ErrUnsupportedDatabaseType = errors.New("unsupported database type, must be 'postgres'")

// The Database implementation used by the bridge system.
type Database interface {
	createBridge(bridge bridge) error
	getBridge(id string) (bridge, error)
	getBridgeByChannel(channelID string) (bridge, error)
	createMessage(message bridgeMessageCollection) error
	deleteMessage(id string) error
	getMessage(id string) (bridgeMessageCollection, error)
}

// DatabaseConfig is the configuration for a database used by the bridge system.
type DatabaseConfig struct {
	Type       string `toml:"type"`
	Connection string `toml:"connection"`
}

// GetDatabase returns a Database based on the configuration.
func (config DatabaseConfig) GetDatabase() (Database, error) {
	switch config.Type {
	case "postgres":
		return newPostgresDatabase(config.Connection)
	default:
		return nil, ErrUnsupportedDatabaseType
	}
}
