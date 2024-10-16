package worklog

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/eldelto/core/internal/cache"
	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/jira"
	"github.com/eldelto/core/internal/rest"
)

func jiraWorklogToEntry(worklog jira.Worklog, timeZone string) Entry {
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		msg := fmt.Sprintf("Failed to load location from jira: %v, falling back to UTC", timeZone)
		fmt.Println(cli.Brown(msg))
		loc = time.UTC
	}

	// TODO: Do we need this?
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

type JiraSink struct {
	client      *jira.Client
	auth        rest.Authenticator
	cacheMyself cache.Cacher[jira.Myself]
	cache       cache.Cacher[string]
}

func NewJiraSink(configProvider *cli.ConfigProvider) (*JiraSink, error) {
	rawHost, err := configProvider.Get("jira.host")
	if err != nil {
		return nil, fmt.Errorf("init JiraSink: %w", err)
	}

	host, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("init JiraSink: %w", err)
	}

	apiKey, err := configProvider.Get("jira.api-key")
	if err != nil {
		return nil, fmt.Errorf("init JiraSink: %w", err)
	}
	auth := rest.BearerAuth{Token: apiKey}

	return &JiraSink{
		client:      &jira.Client{Host: host},
		auth:        &auth,
		cacheMyself: cache.NewOneTime[jira.Myself](),
		cache:       cache.NewOneTime[string](),
	}, nil
}

func (s *JiraSink) myself() (jira.Myself, error) {
	defaultStruct := jira.Myself{
		Key:      "",
		TimeZone: "",
	}

	return s.cacheMyself.GetOrElse("myself", func() (jira.Myself, error) {
		myself, err := s.client.FetchMyself(s.auth)
		if err != nil {
			return defaultStruct, fmt.Errorf("resolve user ID: %w", err)
		}

		return myself, nil
	})
}

func (s *JiraSink) resolveTicketID(e Entry) (string, error) {
	return s.cache.GetOrElse(e.Ticket, func() (string, error) {
		issue, err := s.client.FetchIssue(s.auth, e.Ticket)
		if err != nil {
			return "", err
		}

		return issue.ID, nil
	})
}

func (s *JiraSink) addEntry(e Entry) error {
	myself, err := s.myself()
	if err != nil {
		return err
	}

	loc, err := time.LoadLocation(myself.TimeZone)
	if err != nil {
		msg := fmt.Sprintf("Failed to load location from jira: %v, falling back to UTC", myself.TimeZone)
		fmt.Println(cli.Brown(msg))
		loc = time.UTC
	}

	externalID, err := s.resolveTicketID(e)
	if err != nil {
		return err
	}

	request := jira.WorklogEntryRequest{
		Worker:           myself.Key,
		OriginTaskID:     externalID,
		Started:          e.From.In(loc).Format(rest.ISO8601Format),
		TimeSpentSeconds: int(math.Round(e.To.Sub(e.From).Seconds())),
		Comment:          "-",
	}

	_, err = s.client.CreateWorklogEntry(s.auth, request)
	return err
}

func (s *JiraSink) removeEntry(e Entry) error {
	if e.ExternalID == "" {
		return fmt.Errorf("worklog ID is not set for entry %v", e)
	}

	return s.client.DeleteWorklogEntry(s.auth, e.ExternalID)
}

func (s *JiraSink) Name() string {
	return "Jira"
}

func (s *JiraSink) FetchEntries(start, end time.Time) ([]Entry, error) {
	myself, err := s.myself()
	if err != nil {
		return nil, err
	}

	worklogs, err := s.client.SearchForWorklogs(s.auth, myself.Key, start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch jira entries: %w", err)
	}

	entries := make([]Entry, len(worklogs))
	for i, wl := range worklogs {
		entries[i] = jiraWorklogToEntry(wl, myself.TimeZone)
	}

	return entries, nil
}

func (s *JiraSink) IsApplicable(e Entry) bool {
	return e.Type == EntryTypeWork
}

func (s *JiraSink) ProcessActions(actions []Action, localEntries []Entry) error {
	for _, a := range actions {
		switch a.Operation {
		case Add:
			if err := s.addEntry(a.Entry); err != nil {
				return err
			}
		case Remove:
			if err := s.removeEntry(a.Entry); err != nil {
				return err
			}
		default:
			return fmt.Errorf("jira sink has no handler for operation %v", a)
		}
	}

	return nil
}
