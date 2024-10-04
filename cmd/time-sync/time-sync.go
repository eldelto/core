package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/jira"
	"github.com/eldelto/core/internal/personio"
	"github.com/eldelto/core/internal/worklog"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const dbPath = "time-sync.db"

var (
	clearCredentialsFlag = false
	dryRunFlag           = false
	startDateFlag        = ""
)

var clockifyCmd = &cobra.Command{
	Use:   "clockify",
	Args:  cobra.MatchAll(),
	Short: "Uses clockify to sync time entries with Jira Tempo entries",
	Long: `The command is idempotent and can be run multiple times without creating any duplicate entries.
Configuration (e.g. user credentials) will be stored in a local bbolt database located in $HOME/.hodge/hodge.db.
Timezone information will be fetched from Jira and entries are matched accordingly.
It fetches synced time entries from Clockify and syncs them with Jira Tempo.
The description of the time entry must contain the ticket number e.g. ER-590 or HUM-1234.
It will skip unparseable entries and prints a warning.
You can find our api credentials here:
https://app.clockify.me/user/settings
`,
	Run: func(cmd *cobra.Command, args []string) {
		home, ok := os.LookupEnv("HOME")
		if !ok {
			log.Fatal("could not resolve $HOME environment variable")
		}

		startDate := time.Now().Add(-7 * 24 * time.Hour)
		if startDateFlag != "" {
			date, err := time.Parse(time.DateOnly, startDateFlag)
			if err != nil {
				log.Fatalf("failed to parse startDate %q: %v", startDateFlag, err)
			}
			startDate = date
		}
		// Truncate time values to avoid trouble with entries close to the
		// startDate along the line.
		y, m, d := startDate.Date()
		startDate = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

		configDir := filepath.Join(home, ".hodge")
		if err := os.Mkdir(configDir, 0751); err != nil && os.IsNotExist(err) {
			log.Fatalf("failed to create config dir %q: %v", configDir, err)
		}

		db, err := bbolt.Open(filepath.Join(configDir, "hodge.db"), 0751, nil)
		if err != nil {
			log.Fatalf("failed to open bbolt database: %v", err)
		}
		defer db.Close()

		clockifyAuthProvider := worklog.NewCustomAuthenticationProvider(db, "clockify", &worklog.HeaderAuth{})
		if clearCredentialsFlag {
			if err := clockifyAuthProvider.Clear(); err != nil {
				log.Fatalf("failed to clear credentials: %v", err)
			}
		}

		entries, err := worklog.FetchClockify(clockifyAuthProvider, startDate)
		if err != nil {
			log.Fatal(err)
		}

		client := &jira.Client{Host: "https://<jira-host>"}

		jiraAuthProvider := worklog.NewCustomAuthenticationProvider(db, "jira", &worklog.HeaderAuth{})
		if clearCredentialsFlag {
			if err := jiraAuthProvider.Clear(); err != nil {
				log.Fatalf("failed to clear credentials: %v", err)
			}
		}
		service := worklog.NewService(client, jiraAuthProvider)

		actions, err := service.SyncDryRun(entries)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println()
		worklog.PrettyPrintActions(actions)
		if dryRunFlag || len(actions) < 1 {
			return
		}

		answer, err := cli.ReadInput("Continue syncing? [Y/n]\n")
		if err != nil {
			log.Fatal(err)
		}
		if answer != "" && strings.ToLower(answer) != "y" {
			return
		}

		if err := service.Execute(actions); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	clockifyCmd.Flags().BoolVarP(&clearCredentialsFlag, "clearCredentals", "c", false,
		`If set, the saved user credentials will be cleared and you will be
prompted to enter them again.`)
	clockifyCmd.Flags().BoolVarP(&dryRunFlag, "dryRun", "d", false,
		`If set, the tool will only print out the actions it would take to sync
the worklogs but not actually do it.`)
	clockifyCmd.Flags().StringVar(&startDateFlag, "startDate", "",
		`If a date in the format YYYY-MM-DD is provided, entries that are before the
given date are ignored. It defaults to today - 7 days.`)
	worklogCmd.AddCommand(clockifyCmd)
}

func run() error {
	url, err := url.Parse(os.Args[1])
	if err != nil {
		return err
	}

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return err
	}
	defer db.Close()

	configProvider, err := cli.NewConfigProvider(db)
	if err != nil {
		log.Fatal(err)
	}

	client := personio.NewClient(url, configProvider)
	if err := client.Login(); err != nil {
		log.Fatal(err)
	}

	employeeID, err := client.GetEmployeeID()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(employeeID)

	now := time.Now()
	oneWeekAgo := now.Add(-14 * 24 * time.Hour)
	attendance, err := client.GetAttendance(employeeID, oneWeekAgo, now)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(attendance)
}
