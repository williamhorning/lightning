package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/williamhorning/lightning/internal/bridge"
	"github.com/williamhorning/lightning/pkg/lightning"
)

func migrate(cmd *cobra.Command, args []string) {
	sourceConfig := getDatabaseConfig("source")
	destConfig := getDatabaseConfig("destination")

	fmt.Print("Do you want to proceed with migration? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" {
		fmt.Println("Migration cancelled")
		return
	}

	sourceDB, err := sourceConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "error connecting to source database", nil, nil)
		os.Exit(1)
	}

	destDB, err := destConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "error connecting to destination database", nil, nil)
		os.Exit(1)
	}

	if err := migrateBridges(sourceDB, destDB); err != nil {
		lightning.LogError(err, "error migrating bridges", nil, nil)
		os.Exit(1)
	}

	if err := migrateMessages(sourceDB, destDB); err != nil {
		lightning.LogError(err, "error migrating messages", nil, nil)
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully")
}

func getDatabaseConfig(name string) bridge.DatabaseConfig {
	fmt.Printf("Enter %s database type (postgres): ", name)
	var dbType string
	fmt.Scanln(&dbType)

	fmt.Printf("Enter %s database connection string: ", name)
	var connection string
	fmt.Scanln(&connection)

	return bridge.DatabaseConfig{Type: dbType, Connection: connection}
}

func migrateBridges(sourceDB, destDB bridge.Database) error {
	bridges, err := sourceDB.GetAllBridges()
	if err != nil {
		return err
	}
	return destDB.SetAllBridges(bridges)
}

func migrateMessages(sourceDB, destDB bridge.Database) error {
	messages, err := sourceDB.GetAllMessages()
	if err != nil {
		return err
	}
	return destDB.SetAllMessages(messages)
}
