package main

import (
	"github.com/spf13/cobra"
)

func main() {
	cmd := (&cobra.Command{
		Use:     "lightning",
		Short:   "extensible chatbot connecting communities",
		Long:    "Lightning is an extensible chatbot that connects communities.\nDocs available at https://williamhorning.eu.org",
		Version: "0.8.0-alpha.11",
		Example: "  lightning run lightning.toml",
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "migrate",
		Short:   "migrate databases",
		Long:    "Migrate from one database to another, or from one version to another",
		Example: "  lightning migrate",
		Run:     migrate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:     "run",
		Short:   "run a lightning instance",
		Long:    "Run a lightning instance with the specified configuration file",
		Args:    cobra.RangeArgs(0, 1),
		Example: "  lightning run lightning.toml",
		Run:     run,
	})

	cmd.Execute()
}
