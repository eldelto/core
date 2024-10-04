package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/personio"
	"github.com/eldelto/core/internal/worklog"
	"github.com/spf13/cobra"
	"go.etcd.io/bbolt"
)

const dbPath = "time-sync.db"

func detectSource(sourcePath string, configProvider *cli.ConfigProvider) worklog.Source {
	switch sourcePath {
	case "clockify":
		return worklog.NewClockifySource(configProvider)
	default:
		return worklog.NewFileSource(sourcePath)
	}
}

var rootCmd = &cobra.Command{
	Use:   "clockify",
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

		// startDate := time.Now().Add(-7 * 24 * time.Hour)
		// if startDateFlag != "" {
		// 	date, err := time.Parse(time.DateOnly, startDateFlag)
		// 	if err != nil {
		// 		log.Fatalf("failed to parse startDate %q: %v", startDateFlag, err)
		// 	}
		// 	startDate = date
		// }

		// Truncate time values to avoid trouble with entries close to the
		// startDate along the line.
		// y, m, d := startDate.Date()
		// startDate = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)

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
		sinks := []worklog.Sink{&worklog.StubSink{}}
		now := time.Now()
		oneWeekAgo := now.Add(-14 * 24 * time.Hour)

		if err := worklog.DryRun(source, sinks, oneWeekAgo, now); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
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
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
