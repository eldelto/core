package worklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/gitlab"
	"github.com/eldelto/core/internal/rest"
	"golang.org/x/sync/errgroup"
)

type GitlabSink struct {
	client    *gitlab.Client
	projectID int
}

func NewGitlabSink(rawHost string, configProvider *cli.ConfigProvider) (*GitlabSink, error) {
	host, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("init GitlabSink: %w", err)
	}

	rawProjectID, err := configProvider.Get("gitlab.project-id")
	if err != nil {
		return nil, fmt.Errorf("init GitlabSink: %w", err)
	}

	projectID, err := strconv.Atoi(rawProjectID)
	if err != nil {
		return nil, fmt.Errorf("init GitlabSink: %w", err)
	}

	token, err := configProvider.Get("gitlab.api-key")
	if err != nil {
		return nil, fmt.Errorf("init GitlabSink: %w", err)
	}

	auth := &rest.HeaderAuth{
		Name:  "PRIVATE-TOKEN",
		Value: token,
	}
	client := gitlab.NewClient(host, auth)

	return &GitlabSink{
		client:    client,
		projectID: projectID,
	}, nil
}

func (s *GitlabSink) Name() string {
	return "Gitlab"
}

func (s GitlabSink) ValidIdentifier(identifier string) bool {
	_, err := url.Parse(identifier)
	if err != nil {
		return false
	}

	return strings.Contains(identifier, "git")
}

func (s *GitlabSink) findWorklogComment(issue gitlab.Issue) (*gitlab.Note, error) {
	notes, err := s.client.ListNotes(issue)
	if err != nil {
		return nil, fmt.Errorf("find worklog comment for issue '%d': %w",
			issue.IID, err)
	}

	for _, note := range notes {
		if strings.Contains(note.Body, "externalID") {
			return &note, nil
		}
	}
	return nil, nil
}

func noteToEntries(note *gitlab.Note) ([]Entry, error) {
	if note == nil {
		return []Entry{}, nil
	}

	buff := bytes.NewBufferString(note.Body)
	var entries []Entry
	if err := json.NewDecoder(buff).Decode(&entries); err != nil {
		return entries, fmt.Errorf("decode entries for Gitlab issue '%d': %w",
			note.IssueIID, err)
	}

	return entries, nil
}

func (s *GitlabSink) gitlabIssueToEntries(issue gitlab.Issue) ([]Entry, error) {
	note, err := s.findWorklogComment(issue)
	if err != nil {
		return nil, err
	}

	return noteToEntries(note)
}

func (s *GitlabSink) gitlabIssueToFilteredEntries(issue gitlab.Issue, start, end time.Time) ([]Entry, error) {
	entries, err := s.gitlabIssueToEntries(issue)
	if err != nil {
		return nil, err
	}

	filteredEntries := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.To.Before(start) || entry.From.After(end) {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	return filteredEntries, nil
}

func parallel[I any, O any](values []I, f func(I) ([]O, error)) ([]O, error) {
	resultChan := make(chan O, 10)
	result := []O{}

	go func() {
		for r := range resultChan {
			result = append(result, r)
		}
	}()

	group := errgroup.Group{}
	group.SetLimit(10)
	for _, v := range values {
		group.Go(func() error {
			results, err := f(v)
			if err == nil {
				for _, r := range results {
					resultChan <- r
				}
			}
			return err
		})
	}

	return result, group.Wait()
}

func (s *GitlabSink) FetchEntries(start, end time.Time) ([]Entry, error) {
	issues, err := s.client.ListProjectIssues(s.projectID, start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch Gitlab entries: %w", err)
	}

	return parallel(issues, func(issue gitlab.Issue) ([]Entry, error) {
		return s.gitlabIssueToFilteredEntries(issue, start, end)
	})
}

func (s *GitlabSink) IsApplicable(e Entry) bool {
	return e.Type == EntryTypeWork && e.Ticket != ""
}

func calculateTimeToAdd(actions []Action) int {
	timeToAdd := 0
	for _, a := range actions {
		durationSec := int(math.Round(a.Entry.Duration().Seconds()))

		switch a.Operation {
		case Add:
			timeToAdd += durationSec
		case Remove:
			timeToAdd -= durationSec
		default:
			panic("unknown operation type for ticket " + a.Entry.Ticket)
		}
	}

	return timeToAdd
}

func (s *GitlabSink) entryToIssue(entry Entry) (gitlab.Issue, error) {
	id, err := strconv.Atoi(entry.Ticket)
	if err != nil {
		return gitlab.Issue{}, err
	}

	return gitlab.Issue{
		IID:       id,
		ProjectID: s.projectID,
	}, nil
}

func (s *GitlabSink) updateWorklogComment(issue gitlab.Issue, actions []Action) error {
	note, err := s.findWorklogComment(issue)
	if err != nil {
		return err
	}

	entries, err := noteToEntries(note)
	if err != nil {
		return err
	}

	newEntries := make([]Entry, 0, len(entries))
outer:
	for _, entry := range entries {
		for _, a := range actions {
			if a.Operation == Remove && entryEqual(a.Entry)(entry) {
				continue outer
			}
		}

		newEntries = append(newEntries, entry)
	}

	for _, a := range actions {
		if a.Operation == Add {
			newEntries = append(newEntries, a.Entry)
		}
	}

	buff := bytes.Buffer{}
	if err := json.NewEncoder(&buff).Encode(newEntries); err != nil {
		return fmt.Errorf("encode worklog entries for ticket %q: %w", issue.ID, err)
	}

	if note == nil {
		if _, err := s.client.CreateNote(issue, buff.String()); err != nil {
			return fmt.Errorf("create worklog comment for ticket %q: %w", issue.ID, err)
		}
	} else {
		if _, err := s.client.UpdateNote(*note, buff.String()); err != nil {
			return fmt.Errorf("update worklog comment for ticket %q: %w", issue.ID, err)
		}
	}

	return nil
}

func (s *GitlabSink) updateTicket(actions []Action) error {
	issue, err := s.entryToIssue(actions[0].Entry)
	if err != nil {
		return err
	}

	timeToAdd := calculateTimeToAdd(actions)
	if timeToAdd != 0 {
		if _, err := s.client.AddTimeSpent(issue, timeToAdd); err != nil {
			return fmt.Errorf("failed to update time spent for ticket %q: %w", issue.ID, err)
		}
	}

	return s.updateWorklogComment(issue, actions)
}

func (s *GitlabSink) ProcessActions(actions []Action, localEntries []Entry) error {
	groupedActions := groupActionsByTicket(actions)

	for _, actions := range groupedActions {
		if err := s.updateTicket(actions); err != nil {
			return err
		}
	}

	return nil
}
