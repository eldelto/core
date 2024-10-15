package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/worklog"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const dbPath = "time-sync.db"

var (
	sinksFlag     []string
	startDateFlag string
	endDateFlag   string
	dryRunFlag    bool
)

func parseDate(rawDate string, fallback time.Time) (time.Time, error) {
	if startDateFlag == "" {
		return fallback, nil
	}

	date, err := time.Parse(time.DateOnly, rawDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse %q as date: %w", rawDate, err)
	}

	// Truncate time values to avoid trouble with entries close to the
	// startDate along the line.
	y, m, d := date.Date()
	date = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

	return date, nil
}

func detectSource(sourcePath string, configProvider *cli.ConfigProvider) worklog.Source {
	switch sourcePath {
	case "clockify":
		return worklog.NewClockifySource(configProvider)
	default:
		return worklog.NewFileSource(sourcePath)
	}
}

func detectSinks(rawSinks []string, configProvider *cli.ConfigProvider) ([]worklog.Sink, error) {
	sinks := []worklog.Sink{}
	for _, rawSink := range rawSinks {
		switch {
		case rawSink == "stub":
			sinks = append(sinks, &worklog.StubSink{})
		case strings.Contains(rawSink, "jira"):
			sink, err := worklog.NewJiraSink(configProvider)
			if err != nil {
				return nil, err
			}
			sinks = append(sinks, sink)
		case strings.Contains(rawSink, "personio"):
			sink, err := worklog.NewPersonioSink(rawSink, configProvider)
			if err != nil {
				return nil, err
			}
			sinks = append(sinks, sink)
		default:
			return nil, fmt.Errorf("%q is not a supported sink", rawSink)
		}
	}

	return sinks, nil
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Args:  cobra.ExactArgs(1),
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

		now := time.Now().Truncate(24 * time.Hour)
		startDate, err := parseDate(startDateFlag, now.Add(-7*24*time.Hour))
		if err != nil {
			log.Fatal(err)
		}
		endDate, err := parseDate(endDateFlag, now.Add(24*time.Hour))
		if err != nil {
			log.Fatal(err)
		}

		configDir := filepath.Join(home, ".timesync")
		if err := os.Mkdir(configDir, 0751); err != nil && os.IsNotExist(err) {
			log.Fatalf("create config dir %q: %v", configDir, err)
		}

		db, err := bbolt.Open(filepath.Join(configDir, dbPath), 0600, nil)
		if err != nil {
			log.Fatalf("open bbolt database: %v", err)
		}
		defer db.Close()

		configProvider, err := cli.NewConfigProvider(db)
		if err != nil {
			log.Fatal(err)
		}

		path := os.Args[1]
		source := detectSource(path, configProvider)
		sinks, err := detectSinks(sinksFlag, configProvider)
		if err != nil {
			log.Fatal(err)
		}

		if err := worklog.Sync(source, sinks, startDate, endDate, dryRunFlag); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	syncCmd.Flags().StringArrayVar(&sinksFlag, "sink", []string{},
		`If a date in the format YYYY-MM-DD is provided, entries that are before the
given date are ignored. It defaults to today - 7 days.`)
	syncCmd.Flags().StringVar(&startDateFlag, "start-date", "",
		`If a date in the format YYYY-MM-DD is provided, entries that are before the
given date are ignored. It defaults to today - 7 days.`)
	syncCmd.Flags().StringVar(&endDateFlag, "end-date", "",
		`If a date in the format YYYY-MM-DD is provided, entries that are after the
given date are ignored. It defaults to today.`)
	syncCmd.Flags().BoolVar(&dryRunFlag, "dry-run", false,
		`If true, the command will not execute any action but only print the changes
it would make.`)
}

func main() {
	if err := syncCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
