package worklog

import (
	"fmt"
	"regexp"
	"time"
)

const DateTimeFormat = "2006-01-02 15:04"

var ticketRegex = regexp.MustCompile(`(\w+-\d+)`)

func parseDateTime(s string) (time.Time, error) {
	t, err := time.ParseInLocation(DateTimeFormat, s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse %q as time: %w", s, err)
	}

	return t, nil
}

func parseTicketNumber(s []byte) string {
	matches := ticketRegex.FindSubmatch(s)
	if len(matches) < 2 {
		return ""
	}

	return string(matches[1])
}

func validate(e Entry) error {
	if e.Duration() < 0 {
		return fmt.Errorf("entry contains negative duration")
	}

	return nil
}
