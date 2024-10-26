package worklog

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const OrgDateTimeFormat = "2006-01-02 Mon 15:04"

var clockPrefix = []byte("CLOCK:")

func entryFromClockLine(ticket, line string, start, end time.Time) (Entry, error) {
	if len(line) < 32+len(OrgDateTimeFormat) {
		return Entry{}, errSkippedEntry
	}

	from, err := time.ParseInLocation(OrgDateTimeFormat, line[8:8+len(OrgDateTimeFormat)], time.Local)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to parse from date from %q", line)
	}
	to, err := time.ParseInLocation(OrgDateTimeFormat, line[32:32+len(OrgDateTimeFormat)], time.Local)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to parse to date from %q", line)
	}
	if to.Before(start) || from.After(end) {
		return Entry{}, errSkippedEntry
	}

	return Entry{
		Ticket: ticket,
		To:     to,
		From:   from,
	}, nil
}

func parseFromOrg(r io.Reader, start, end time.Time) ([]Entry, error) {
	entries := []Entry{}
	scanner := bufio.NewScanner(r)

	i := 0
	ticket := ""
	for scanner.Scan() {
		i++
		if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("error on line %d: failed to scan line: %w", i, err)
		}

		line := scanner.Bytes()
		if len(line) < 1 {
			continue
		}
		trimmedLine := bytes.TrimSpace(line)

		if line[0] == '*' {
			ticket = parseTicketNumber(line)
		} else if bytes.HasPrefix(trimmedLine, clockPrefix) {
			entry, err := entryFromClockLine(ticket, string(trimmedLine), start, end)
			if err != nil {
				if errors.Is(err, errSkippedEntry) {
					continue
				}
				return nil, fmt.Errorf("error on line %d: %w", i, err)
			}

			if err := validate(entry); err != nil {
				return nil, fmt.Errorf("error on line %d: %w", i, err)
			}

			entries = append(entries, entry)
		}
	}

	return entries, nil
}

type OrgSource struct {
	filePath string
}

func NewOrgSource(path string) (*OrgSource, error) {
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("org file %q could not be opened: %w", path, err)
	}

	return &OrgSource{
		filePath: path,
	}, nil
}

func (s *OrgSource) Name() string {
	return "Org Mode"
}

func (s OrgSource) ValidIdentifier(identifier string) bool {
	return strings.HasSuffix(identifier, ".org")
}

func (s *OrgSource) FetchEntries(start, end time.Time) ([]Entry, error) {
	f, err := os.Open(s.filePath)
	if err != nil {
		return nil, fmt.Errorf("fetch CSV entries: %w", err)
	}
	defer f.Close()
	return parseFromOrg(f, start, end)
}
