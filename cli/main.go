package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v3"
	"github.com/williamhorning/lightning"
	_ "github.com/williamhorning/lightning/discord"
	_ "github.com/williamhorning/lightning/guilded"
	_ "github.com/williamhorning/lightning/revolt"
	_ "github.com/williamhorning/lightning/telegram"
)

func main() {
	(&cli.Command{
		Name:                  "lightning",
		Usage:                 "extensible chatbot connecting communities",
		Version:               "0.8.0-alpha.7",
		DefaultCommand:        "help",
		EnableShellCompletion: true,
		Authors:               []any{"William Horning", "Lightning contributors"},
		Copyright:             "2025 William Horning and contributors.\nAvailible under the MIT license",
		Commands: []*cli.Command{
			{
				Name:   "migrate",
				Usage:  "migrate databases",
				Action: migrate,
			},
			{
				Name:  "run",
				Usage: "run a lightning instance",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name:      "config",
						UsageText: "the path to the configuration file",
						Value:     "lightning.toml",
						Config:    cli.StringConfig{TrimSpace: true},
					},
				},
				Action: run,
			},
		},
	}).Run(context.Background(), os.Args)
}

func run(ctx context.Context, c *cli.Command) error {
	config, err := lightning.LoadConfig(c.StringArg("config"))
	if err != nil {
		lightning.LogError(err, "something went wrong with loading the config", nil, lightning.ReadWriteDisabled{})
		os.Exit(1)
	}

	db, err := config.DatabaseConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "something went wrong with setting up the database", nil, lightning.ReadWriteDisabled{})
		os.Exit(1)
	}

	lightning.SetupBridge(db)

	quitChannel := make(chan os.Signal, 1)
	signal.Notify(quitChannel, syscall.SIGINT, syscall.SIGTERM)
	<-quitChannel

	lightning.LogError(errors.New("lightning instance stopped"), "lightning instance stopped", nil, lightning.ReadWriteDisabled{})
	return nil
}

func migrate(ctx context.Context, c *cli.Command) error {
	sourceConfig := getDatabaseConfig("source")
	destConfig := getDatabaseConfig("destination")

	fmt.Print("Do you want to proceed with migration? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" {
		fmt.Println("Migration cancelled")
		return nil
	}

	sourceDB, err := sourceConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "error connecting to source database", nil, lightning.ReadWriteDisabled{})
		os.Exit(1)
	}

	destDB, err := destConfig.GetDatabase()
	if err != nil {
		lightning.LogError(err, "error connecting to destination database", nil, lightning.ReadWriteDisabled{})
		os.Exit(1)
	}

	if err := migrateBridges(sourceDB, destDB); err != nil {
		lightning.LogError(err, "error migrating bridges", nil, lightning.ReadWriteDisabled{})
		os.Exit(1)
	}

	if err := migrateMessages(sourceDB, destDB); err != nil {
		lightning.LogError(err, "error migrating messages", nil, lightning.ReadWriteDisabled{})
		os.Exit(1)
	}

	fmt.Println("Migration completed successfully")
	return nil
}

func getDatabaseConfig(name string) lightning.DatabaseConfig {
	fmt.Printf("Enter %s database type (postgres/redis): ", name)
	var dbType string
	fmt.Scanln(&dbType)

	fmt.Printf("Enter %s database connection string: ", name)
	var connection string
	fmt.Scanln(&connection)

	return lightning.DatabaseConfig{Type: dbType, Connection: connection}
}

func migrateBridges(sourceDB, destDB lightning.Database) error {
	bridges, err := sourceDB.GetAllBridges()
	if err != nil {
		return err
	}
	return destDB.SetAllBridges(bridges)
}

func migrateMessages(sourceDB, destDB lightning.Database) error {
	messages, err := sourceDB.GetAllMessages()
	if err != nil {
		return err
	}
	return destDB.SetAllMessages(messages)
}
