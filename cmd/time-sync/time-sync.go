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

func run() error {
	url, err := url.Parse(os.Args[1])
	if err != nil {
		return err
	}

	db, err := bbolt.Open("time-sync.db", 0600, nil)
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

	day := time.Date(2024, 10, 1, 8, 0, 0, 0, time.Local)
	attendances := []personio.Attendance{
		{
			Start: time.Date(2024, 10, 1, 8, 0, 0, 0, time.Local),
			End:   time.Date(2024, 10, 1, 12, 0, 0, 0, time.Local),
		},
		{
			Start: time.Date(2024, 10, 1, 13, 0, 0, 0, time.Local),
			End:   time.Date(2024, 10, 1, 15, 0, 0, 0, time.Local),
		},
	}
	if err := client.CreateAttendances(employeeID, day, attendances); err != nil {
		log.Fatal(err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
