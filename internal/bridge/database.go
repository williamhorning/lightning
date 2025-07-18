package bridge

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
	case "postgres", "pgx":
		return newSQLDatabase("pgx", config.Connection)
	case "sqlite":
		return newSQLDatabase("sqlite", config.Connection)
	default:
		return nil, UnsupportedDatabaseTypeError{}
	}
}

// UnsupportedDatabaseTypeError is returned when an unsupported database type is given in configuration.
type UnsupportedDatabaseTypeError struct{}

func (UnsupportedDatabaseTypeError) Error() string {
	return "unsupported database type, must be 'postgres', 'pgx' (postgres), or 'sqlite'"
}
