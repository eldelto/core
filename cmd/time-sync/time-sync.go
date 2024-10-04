package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/personio"
	"go.etcd.io/bbolt"
)

const dbPath = "time-sync.db"

func main() {
	url, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatalf("failed to open bbolt DB %q: %v", dbPath, err)
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
