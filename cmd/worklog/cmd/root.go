package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "worklog",
	Short: "A CLI tool to sync working times between systems.",
	Long: `A CLI tool to sync working times between different systems.

Refer to the help page of the individual sub-commands for more information.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
