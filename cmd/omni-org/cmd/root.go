package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/eldelto/core/internal/conf"
	"github.com/eldelto/core/omniorg"
	"github.com/spf13/cobra"
)

const configName = "omni.conf"

var rootCmd = &cobra.Command{
	Use:   "omni-org",
	Short: "Omnificent Emacs org-mode",
	Long: `This is a companion application for the omni-org Emacs package. It
collects tasks from various sources, unifies them and finally renders
them to an org-mode file so they can be acted upon just like regular
org-mode tasks.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(); err != nil {
			log.Fatal(err)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func commandLoop() error {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		var entryID omniorg.ID
		command := strings.TrimSpace(scanner.Text())
		parts := strings.Split(command, ":")
		if len(parts) > 1 {
			command = parts[0]
			entryID = omniorg.ID(parts[1])
		}

		var err error
		switch command {
		case "generate":
			err = omniorg.GenerateOrgFile(time.Time{}, time.Now())
			fmt.Println("Generated omni.org")
		case "taken-over":
			err = omniorg.MarkTakenOver(entryID)
		case "quit":
			return nil
		default:
			fmt.Printf("Unknown command %q\n", command)
		}
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("command loop: err=%w", err)
	}
	return nil
}

func run() error {
	if _, err := os.Stat(configName); err != nil {
		log.Fatalf("init omni-org: err=%v - exiting", err)
	}
	config := conf.NewFileConfigProvider(configName)

	if config.Exists("gitlab.token") {
		omniorg.RegisterSource(omniorg.NewGitlabSource(config))
	}

	return commandLoop()
}
