// Package data provides the database used by the Lightning bridge bot.
package data

// The Database implementation used by the bridge system.
type Database interface {
	CreateBridge(bridge Bridge) error
	GetBridge(id string) (Bridge, error)
	GetBridgeByChannel(channelID string) (Bridge, error)
	CreateMessage(message BridgeMessageCollection) error
	DeleteMessage(id string) error
	GetMessage(id string) (BridgeMessageCollection, error)
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
		return nil, UnsupportedDatabaseTypeError{}
	}
}

// UnsupportedDatabaseTypeError is returned when an unsupported database type is given in configuration.
type UnsupportedDatabaseTypeError struct{}

func (UnsupportedDatabaseTypeError) Error() string {
	return "unsupported database type, must be 'postgres'"
}
