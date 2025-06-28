package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	(&cli.Command{
		Name:           "lightning",
		Usage:          "extensible chatbot connecting communities",
		Version:        "0.8.0-alpha.10",
		DefaultCommand: "help",
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
