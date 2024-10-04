package worklog

import (
	"fmt"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/clockify"
)

func FetchClockify(authProvider AuthenticationProvider, startDate time.Time) ([]Entry, error) {
	auth, err := authProvider.Authenticator()
	if err != nil {
		return nil, err
	}

	client := &clockify.Client{Host: "https://api.clockify.me/", Auth: auth}

	myself, err := client.FetchMyself()
	if err != nil {
		return nil, err
	}

	timeEntries, err := client.FetchTimeEntries(myself, startDate)
	if err != nil {
		return nil, err
	}

	result := []Entry{}
	loc, err := time.LoadLocation(myself.Settings.TimeZone)
	if err != nil {
		msg := fmt.Sprintf("Failed to load location from clockify: %v, falling back to UTC", myself.Settings.TimeZone)
		fmt.Println(cli.Brown(msg))
		loc = time.UTC
	}
	for _, v := range timeEntries {
		ticket, err := parseTicket([]byte(v.Description))
		if err != nil {
			msg := fmt.Sprintf("failed to parse ticket for %s, skipping", v.Description)
			fmt.Println(cli.Brown(msg))
			continue
		}
		result = append(result, Entry{
			From:   v.TimeInterval.Start.In(loc),
			To:     v.TimeInterval.End.In(loc),
			Ticket: ticket,
		})
	}
	return result, nil
}
