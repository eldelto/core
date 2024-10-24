package worklog

import (
	"fmt"
	"net/url"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/clockify"
	"github.com/eldelto/core/internal/rest"
)

type ClockifySource struct {
	configProvider *cli.ConfigProvider
}

func NewClockifySource(configProvider *cli.ConfigProvider) *ClockifySource {
	return &ClockifySource{
		configProvider: configProvider,
	}
}

func (s *ClockifySource) Name() string {
	return "clockify source"
}

func (s *ClockifySource) FetchEntries(start, end time.Time) ([]Entry, error) {
	apiKey, err := s.configProvider.Get("clockify.api-key")
	if err != nil {
		return nil, fmt.Errorf("clockify API key: %w", err)
	}

	auth := &rest.HeaderAuth{
		Name:  "X-Api-Key",
		Value: apiKey,
	}

	host, err := url.Parse("https://api.clockify.me")
	if err != nil {
		return nil, fmt.Errorf("parse clockify host: %w", err)
	}
	client := &clockify.Client{Host: host, Auth: auth}

	myself, err := client.FetchMyself()
	if err != nil {
		return nil, err
	}

	timeEntries, err := client.FetchTimeEntries(myself, start, end)
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
		ticket := parseTicketNumber([]byte(v.Description))
		if ticket == "" {
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
