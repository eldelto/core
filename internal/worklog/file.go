package worklog

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	DateTimeFormat    = "2006-01-02 15:04"
	TimeFormat        = "15:04"
	OrgDateTimeFormat = "2006-01-02 Mon 15:04"
	GermanDateFormat  = "02.01.2006"
)

var (
	ticketRegex     = regexp.MustCompile(`(\w+-\d+)`)
	dateRegex       = regexp.MustCompile(`(\d{1,2}\.\d{1,2}\.\d{4})`)
	timeRegex       = regexp.MustCompile(`(\d{1,2}:\d{2}) - (\d{1,2}:\d{2})`)
	errSkippedEntry = errors.New("entry is invalid and will be skipped")
	clockPrefix     = []byte("CLOCK:")
	emptyTime       = time.Time{}
)

func parseDateTime(s string) (time.Time, error) {
	t, err := time.Parse(DateTimeFormat, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse %q as time: %w", s, err)
	}

	return t, nil
}

func parseTicket(s []byte) (string, error) {
	matches := ticketRegex.FindSubmatch(s)
	if len(matches) < 2 {
		return "", fmt.Errorf("failed to parse ticket number: %q does not contain a valid ticket number", s)
	}

	return string(matches[1]), nil
}

func parseDate(s []byte) (time.Time, error) {
	matches := dateRegex.FindSubmatch(s)
	if len(matches) < 2 {
		return time.Time{}, fmt.Errorf("failed to parse date: %q does not contain a valid date", s)
	}

	rawDate := string(matches[1])
	date, err := time.Parse(GermanDateFormat, rawDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date from %q: %w", rawDate, err)
	}

	return date, nil
}

func validate(e Entry) error {
	if e.Duration() < 0 {
		return fmt.Errorf("entry contains negative duration")
	}

	return nil
}

func parseFromCSV(r io.Reader, start, end time.Time) ([]Entry, error) {
	csvReader := csv.NewReader(r)
	entries := []Entry{}

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read headline of CSV file: %w", err)
	}

	ticketIndex := -1
	fromIndex := -1
	toIndex := -1
	for i, k := range header {
		switch k {
		case "ticket":
			ticketIndex = i
		case "from":
			fromIndex = i
		case "to":
			toIndex = i
		}
	}

	if ticketIndex < 0 || fromIndex < 0 || toIndex < 0 {
		return nil, fmt.Errorf("invalid CSV header format")
	}

	i := 1
	for {
		i++

		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV line %d: %w", i, err)
		}

		ticket, err := parseTicket([]byte(rec[ticketIndex]))
		if err != nil {
			return nil, fmt.Errorf("error on line %d: %w", i, err)
		}

		from, err := parseDateTime(rec[fromIndex])
		if err != nil {
			return nil, fmt.Errorf("error on line %d: %w", i, err)
		}
		to, err := parseDateTime(rec[toIndex])
		if err != nil {
			return nil, fmt.Errorf("error on line %d: %w", i, err)
		}

		entry := Entry{
			Ticket: ticket,
			From:   from,
			To:     to,
		}
		if entry.From.Before(start) || entry.From.After(end) {
			continue
		}
		if err := validate(entry); err != nil {
			return nil, fmt.Errorf("error on line %d: %w", i, err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func entryFromClockLine(ticket, line string, start, end time.Time) (Entry, error) {
	if len(line) < 32+len(OrgDateTimeFormat) {
		return Entry{}, errSkippedEntry
	}

	from, err := time.Parse(OrgDateTimeFormat, line[8:8+len(OrgDateTimeFormat)])
	if err != nil {
		return Entry{}, fmt.Errorf("failed to parse from date from %q", line)
	}
	if from.Before(start) || from.After(end) {
		return Entry{}, errSkippedEntry
	}

	to, err := time.Parse(OrgDateTimeFormat, line[32:32+len(OrgDateTimeFormat)])
	if err != nil {
		return Entry{}, fmt.Errorf("failed to parse to date from %q", line)
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
			ticket, _ = parseTicket(line)
		} else if bytes.HasPrefix(trimmedLine, clockPrefix) {
			if ticket == "" {
				continue
			}
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

func entryFromMarkdownLine(line []byte, date, start, end time.Time) (Entry, error) {
	ticket, err := parseTicket(line)
	if err != nil {
		return Entry{}, errSkippedEntry
	}

	matches := timeRegex.FindSubmatch(line)
	if len(matches) < 3 {
		return Entry{}, errSkippedEntry
	}

	rawFromTime := matches[1]
	rawToTime := matches[2]

	from, err := time.ParseInLocation(TimeFormat, string(rawFromTime), time.Local)
	if err != nil {
		return Entry{}, fmt.Errorf("parse from date from %q", line)
	}
	if from.Before(start) || from.After(end) {
		return Entry{}, errSkippedEntry
	}

	to, err := time.ParseInLocation(TimeFormat, string(rawToTime), time.Local)
	if err != nil {
		return Entry{}, fmt.Errorf("parse to time from %q", line)
	}

	return Entry{
		Ticket: ticket,
		To:     to,
		From:   from,
	}, nil
}

func parseFromMarkdown(r io.Reader, start, end time.Time) ([]Entry, error) {
	entries := []Entry{}
	scanner := bufio.NewScanner(r)

	i := 0
	date := time.Time{}
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

		if line[0] == '#' {
			date, _ = parseDate(trimmedLine)
		} else {
			if date == emptyTime {
				continue
			}

			entry, err := entryFromMarkdownLine(trimmedLine, date, start, end)
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

func parseFile(path string, start, end time.Time, skipUnsupported bool) ([]Entry, error) {
	r, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer r.Close()

	if strings.HasSuffix(path, ".csv") {
		return parseFromCSV(r, start, end)
	} else if strings.HasSuffix(path, ".org") {
		return parseFromOrg(r, start, end)
	} else if strings.HasSuffix(path, ".md") {
		return parseFromMarkdown(r, start, end)
	}

	if skipUnsupported {
		return []Entry{}, nil
	}

	return nil, fmt.Errorf("unsupported file type %q", path)
}

func parseDir(path string, start, end time.Time) ([]Entry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", path, err)
	}

	result := []Entry{}
	for _, e := range entries {
		if e.IsDir() || e.Name()[0] == '.' {
			continue
		}

		filePath := filepath.Join(path, e.Name())
		entries, err := parseFile(filePath, start, end, true)
		if err != nil {
			return nil, err
		}
		result = append(result, entries...)
	}

	return result, nil
}

func isDir(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open %q: %w", path, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to get stats of %q: %w", path, err)
	}

	return info.IsDir(), nil
}

type FileSource struct {
	path string
}

func NewFileSource(path string) *FileSource {
	return &FileSource{
		path: path,
	}
}

func (s *FileSource) Name() string {
	return "file source"
}

func (s *FileSource) FetchEntries(start, end time.Time) ([]Entry, error) {
	isDir, err := isDir(s.path)
	if err != nil {
		return nil, err
	}

	if isDir {
		return parseDir(s.path, start, end)
	}

	return parseFile(s.path, start, end, false)
}
