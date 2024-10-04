package worklog

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/eldelto/core/internal/cache"
	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/jira"
	"github.com/eldelto/core/internal/rest"
	"golang.org/x/sync/errgroup"
)

func jiraWorklogToEntry(worklog jira.Worklog, timeZone string) Entry {
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		msg := fmt.Sprintf("Failed to load location from jira: %v, falling back to UTC", timeZone)
		fmt.Println(cli.Brown(msg))
		loc = time.UTC
	}

	fromNoTimeZone := time.Time(worklog.Started)
	from := time.Date(fromNoTimeZone.Year(), fromNoTimeZone.Month(), fromNoTimeZone.Day(), fromNoTimeZone.Hour(), fromNoTimeZone.Minute(), fromNoTimeZone.Second(), fromNoTimeZone.Nanosecond(), loc)

	toNoTimeZone := time.Time(worklog.Started).Add(time.Duration(worklog.TimeSpentSeconds) * time.Second)
	to := time.Date(toNoTimeZone.Year(), toNoTimeZone.Month(), toNoTimeZone.Day(), toNoTimeZone.Hour(), toNoTimeZone.Minute(), toNoTimeZone.Second(), toNoTimeZone.Nanosecond(), loc)

	return Entry{
		Ticket:     worklog.Issue.Key,
		ExternalID: strconv.FormatInt(int64(worklog.TempoWorklogID), 10),
		From:       from,
		To:         to,
	}
}

type AuthenticationProvider interface {
	Authenticator() (rest.Authenticator, error)
}

const maxParallel = 8

type Service struct {
	client      *jira.Client
	auth        rest.Authenticator
	cacheMyself cache.Cacher[jira.Myself]
	cache       cache.Cacher[string]
}

func NewService(client *jira.Client, auth rest.Authenticator) *Service {
	return &Service{
		client:      client,
		auth:        auth,
		cacheMyself: cache.NewOneTime[jira.Myself](),
		cache:       cache.NewOneTime[string](),
	}
}

func (s *Service) myself() (jira.Myself, error) {
	defaultStruct := jira.Myself{
		Key:      "",
		TimeZone: "",
	}

	return s.cacheMyself.GetOrElse("myself", func() (jira.Myself, error) {
		myself, err := s.client.FetchMyself(s.auth)
		if err != nil {
			return defaultStruct, fmt.Errorf("failed to resolve user ID: %w", err)
		}

		return myself, nil
	})
}

func (s *Service) fetchRemoteEntries(day time.Time) ([]Entry, error) {
	myself, err := s.myself()
	if err != nil {
		return nil, err
	}

	worklogs, err := s.client.SearchForWorklogs(s.auth, myself.Key, day, day)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote entries for day %v: %w", day, err)
	}

	entries := make([]Entry, len(worklogs))
	for i, wl := range worklogs {
		entries[i] = jiraWorklogToEntry(wl, myself.TimeZone)
	}

	return entries, nil
}

func (s *Service) resolveTicketID(e Entry) (string, error) {
	return s.cache.GetOrElse(e.Ticket, func() (string, error) {
		issue, err := s.client.FetchIssue(s.auth, e.Ticket)
		if err != nil {
			return "", err
		}

		return issue.ID, nil
	})
}

func (s *Service) addEntry(e Entry) error {
	myself, err := s.myself()
	if err != nil {
		return err
	}

	externalID, err := s.resolveTicketID(e)
	if err != nil {
		return err
	}

	request := jira.WorklogEntryRequest{
		Worker:           myself.Key,
		OriginTaskID:     externalID,
		Started:          e.From.Format(rest.ISO8601Format),
		TimeSpentSeconds: int(math.Round(e.To.Sub(e.From).Seconds())),
		Comment:          "-",
	}

	_, err = s.client.CreateWorklogEntry(s.auth, request)
	return err
}

func (s *Service) removeEntry(e Entry) error {
	if e.ExternalID == "" {
		return fmt.Errorf("worklog ID is not set for entry %v", e)
	}

	return s.client.DeleteWorklogEntry(s.auth, e.ExternalID)
}

func (s *Service) executeAction(action Action) error {
	switch action.Operation {
	case Add:
		return s.addEntry(action.Entry)
	case Remove:
		return s.removeEntry(action.Entry)
	default:
		return fmt.Errorf("failed to execute unknown action %v", action)
	}
}

func (s *Service) Execute(actions map[time.Time][]Action) error {
	g := errgroup.Group{}
	g.SetLimit(maxParallel)

	for d, actions := range actions {
		day := d
		fmt.Printf("Syncing %s ...\n", day.Format(time.DateOnly))
		for _, a := range actions {
			action := a
			g.Go(func() error { return s.executeAction(action) })
		}
		fmt.Printf("Synced %s\n", day.Format(time.DateOnly))
	}

	return g.Wait()
}
