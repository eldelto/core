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

	"github.com/eldelto/core/internal/gitlab"
)

type GitlabSink struct {
	client    *gitlab.Client
	projectID int
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
		return nil, fmt.Errorf("find worklog comment for issue %q: %w",
			issue.IID, err)
	}

	for _, note := range notes {
		if strings.Contains(note.Body, "externalID") {
			return &note, nil
		}
	}
	return nil, nil
}

func (s *GitlabSink) gitlabIssueToEntries(issue gitlab.Issue) ([]Entry, error) {
	comment, err := s.findWorklogComment(issue)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBufferString(comment.Body)
	var entries []Entry
	if err := json.NewDecoder(buff).Decode(&entries); err != nil {
		return entries, fmt.Errorf("decode entries for Gitlab issue %q: %w",
			issue.IID, err)
	}

	return entries, nil
}

func (s *GitlabSink) FetchEntries(start, end time.Time) ([]Entry, error) {
	issues, err := s.client.ListProjectIssues(s.projectID, start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch Gitlab entries: %w", err)
	}

	entries := []Entry{}
	for _, issue := range issues {
		newEntries, err := s.gitlabIssueToEntries(issue)
		if err != nil {
			return nil, err
		}
		entries = append(entries, newEntries...)
	}

	return entries, nil
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
	id, err := strconv.ParseInt(entry.Ticket, 10, 64)
	if err != nil {
		return gitlab.Issue{}, err
	}

	return gitlab.Issue{
		ID:        int(id),
		ProjectID: s.projectID,
	}, nil
}

func (s *GitlabSink) updateWorklogComment(issue gitlab.Issue, entries []Entry) error {
	note, err := s.findWorklogComment(issue)
	if err != nil {
		return err
	}

	buff := bytes.Buffer{}
	if err := json.NewEncoder(&buff).Encode(entries); err != nil {
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

func (s *GitlabSink) updateTicket(actions []Action, localEntries []Entry) error {
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

	return s.updateWorklogComment(issue, localEntries)
}

func (s *GitlabSink) ProcessActions(actions []Action, localEntries []Entry) error {
	groupedActions := groupActionsByTicket(actions)
	groupedEntries := groupEntriesByTicket(localEntries)

	for ticketID, actions := range groupedActions {
		entries := groupedEntries[ticketID]
		if err := s.updateTicket(actions, entries); err != nil {
			return err
		}
	}

	return nil
}
