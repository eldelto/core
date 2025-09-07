package worklog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/internal/gitlab"
)

type GitlabSink struct {
	client    *gitlab.Client
	projectID string
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

func (s *GitlabSink) ProcessActions(actions []Action, localEntries []Entry) error {
	// TODO: Implement
	return nil
}
