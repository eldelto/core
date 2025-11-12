package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/worklog"
	"github.com/spf13/cobra"
)

const dbPath = "worklog.db"

var (
	sinksFlag     []string
	startDateFlag string
	endDateFlag   string
	dryRunFlag    bool
)

func parseDate(rawDate string, fallback time.Time) (time.Time, error) {
	if rawDate == "" {
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

func detectSource(sourcePath string, configProvider *cli.ConfigProvider) (worklog.Source, error) {
	var source worklog.Source
	var err error
	switch {
	case worklog.StubSink{}.ValidIdentifier(sourcePath):
		source = &worklog.StubSink{}
	case worklog.OrgSource{}.ValidIdentifier(sourcePath):
		source, err = worklog.NewOrgSource(sourcePath)
	case worklog.CSVSink{}.ValidIdentifier(sourcePath):
		source, err = worklog.NewCSVSink(sourcePath)
	case worklog.ClockifySource{}.ValidIdentifier(sourcePath):
		source = worklog.NewClockifySource(configProvider)
	case worklog.JiraSink{}.ValidIdentifier(sourcePath):
		source, err = worklog.NewJiraSink(sourcePath, configProvider)
	case worklog.PersonioSink{}.ValidIdentifier(sourcePath):
		source, err = worklog.NewPersonioSink(sourcePath, configProvider)
	case worklog.GitlabSink{}.ValidIdentifier(sourcePath):
		source, err = worklog.NewPersonioSink(sourcePath, configProvider)
	default:
		source = worklog.NewFileSource(sourcePath)
	}

	return source, err
}

func detectSink(rawSink string, configProvider *cli.ConfigProvider) (worklog.Sink, error) {
	var sink worklog.Sink
	var err error
	switch {
	case worklog.StubSink{}.ValidIdentifier(rawSink):
		sink = &worklog.StubSink{}
	case worklog.CSVSink{}.ValidIdentifier(rawSink):
		sink, err = worklog.NewCSVSink(rawSink)
	case worklog.JiraSink{}.ValidIdentifier(rawSink):
		sink, err = worklog.NewJiraSink(rawSink, configProvider)
	case worklog.PersonioSink{}.ValidIdentifier(rawSink):
		sink, err = worklog.NewPersonioSink(rawSink, configProvider)
	case worklog.GitlabSink{}.ValidIdentifier(rawSink):
		sink, err = worklog.NewGitlabSink(rawSink, configProvider)
	default:
		return nil, fmt.Errorf("%q is not a supported sink", rawSink)
	}

	return sink, err
}

func detectSinks(rawSinks []string, configProvider *cli.ConfigProvider) ([]worklog.Sink, error) {
	sinks := []worklog.Sink{}
	for _, rawSink := range rawSinks {
		sink, err := detectSink(rawSink, configProvider)
		if err != nil {
			return nil, err
		}

		sinks = append(sinks, sink)
	}

	return sinks, nil
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Args:  cobra.ExactArgs(1),
	Short: "Sync worklogs from the given source to all provided sinks.",
	Long: `Sync worklogs from the given source to all provided sinks.

This command tries to detect the source of your worklog entries based
on the first argument. Currently supported sinks are:

  - Org-mode file
  - CSV file (with columns: ticket, from, to)
  - Directory containing above mentioned file types
  - 'clockify' (fetches data from https://clockify.me)

Sinks are all other places that you want to synchronize with your
worklog source. Sinks are auto-detected as well and there exist
implementations for:

 - Jira Tempo - if you provide a URL to a Jira instance
   (e.g. https://jira.acme.com)
 - Personio - if you provide a URL to a Personio instance
   (e.g. https://acme.personio.de)

During the worklog sync you may be asked to provide additional
credentials to complete the process. All credentials are stored in a
local database in $HOME/.worklog/worklog.db.

If an entry is unparseable, it will be skipped and a warning printed
to either stdout or stderr depending on the severity.

The sync command is idempotent and can be run multiple times without
creating duplicate entries. `,
	Run: func(cmd *cobra.Command, args []string) {
		now := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
		startDate, err := parseDate(startDateFlag, now.Add(-7*24*time.Hour))
		if err != nil {
			log.Fatal(err)
		}
		endDate, err := parseDate(endDateFlag, now)
		if err != nil {
			log.Fatal(err)
		}

		configProvider, err := initConfigProvider()
		if err != nil {
			log.Fatal(err)
		}

		path := args[0]
		source, err := detectSource(path, configProvider)
		if err != nil {
			log.Fatal(err)
		}

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
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringArrayVar(&sinksFlag, "sink", []string{},
		`One of possibly many sinks to synchronize to. E.g. 'https://jira.acme.com'
to synchronize with Jira Tempo.`)
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
