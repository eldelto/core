package worklog

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

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

		ticket := parseTicketNumber([]byte(rec[ticketIndex]))

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
		if entry.To.Before(start) || entry.From.After(end) {
			continue
		}
		if err := validate(entry); err != nil {
			return nil, fmt.Errorf("error on line %d: %w", i, err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

type CSVSink struct {
	filePath string
}

func NewCSVSink(path string) (*CSVSink, error) {
	return &CSVSink{
		filePath: path,
	}, nil
}

func (s *CSVSink) Name() string {
	return "CSV"
}

func (s CSVSink) ValidIdentifier(identifier string) bool {
	return strings.HasSuffix(identifier, ".csv")
}

func (s *CSVSink) FetchEntries(start, end time.Time) ([]Entry, error) {
	f, err := os.Open(s.filePath)
	if err != nil {
		return []Entry{}, nil
	}
	defer f.Close()
	return parseFromCSV(f, start, end)
}

func (s *CSVSink) IsApplicable(e Entry) bool {
	return true
}

func (s *CSVSink) ProcessActions(actions []Action, localEntries []Entry) error {
	f, err := os.Create(s.filePath)
	if err != nil {
		return fmt.Errorf("process CSV sink actions: %w", err)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write([]string{"ticket", "from", "to"}); err != nil {
		return fmt.Errorf("write CSV sink headline: %w", err)
	}

	for _, a := range actions {
		switch a.Operation {
		case Add:
			row := []string{
				a.Entry.Ticket,
				a.Entry.From.Format(DateTimeFormat),
				a.Entry.To.Format(DateTimeFormat),
			}
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("write CSV sink row: %w", err)
			}
		case Remove:
		default:
			return fmt.Errorf("csv sink has no handler for operation %v", a)
		}
	}

	return nil
}
