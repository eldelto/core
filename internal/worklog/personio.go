package worklog

import (
	"fmt"
	"net/url"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/personio"
	"github.com/eldelto/core/internal/util"
)

func attendanceToEntry(a personio.AttendancePeriode) Entry {
	entryType := EntryTypeWork
	if a.Attributes.PeriodType == "break" {
		entryType = EntryTypeBreak
	}

	return Entry{
		Ticket:     a.Attributes.Comment,
		ExternalID: a.ID,
		Type:       entryType,
		From:       util.SetLocation(a.Attributes.Start, time.Local),
		To:         util.SetLocation(a.Attributes.End, time.Local),
	}
}

func attendancesToEntries(attendances []personio.AttendancePeriode) []Entry {
	entries := make([]Entry, len(attendances))
	for i := range attendances {
		entries[i] = attendanceToEntry(attendances[i])
	}

	return entries
}

func entriesToAttendances(entries []Entry) []personio.Attendance {
	attendances := []personio.Attendance{}
	for _, e := range entries {
		attendance := personio.Attendance{
			Start:   e.From,
			End:     e.To,
			Comment: e.Ticket,
		}
		attendances = append(attendances, attendance)
	}

	return attendances
}

func addActionCount(actions []Action) int {
	count := 0
	for _, a := range actions {
		if a.Operation == Add {
			count++
		}
	}
	return count
}

type PersonioSink struct {
	client *personio.Client
}

func NewPersonioSink(rawHost string, configProvider *cli.ConfigProvider) (*PersonioSink, error) {
	host, err := url.Parse(rawHost)
	if err != nil {
		return nil, fmt.Errorf("init PersonioSink: %w", err)
	}

	client := personio.NewClient(host, configProvider)

	return &PersonioSink{
		client: client,
	}, nil
}

func (s *PersonioSink) Name() string {
	return "Personio"
}

func (s *PersonioSink) FetchEntries(start, end time.Time) ([]Entry, error) {
	employeeID, err := s.client.GetEmployeeID()
	if err != nil {
		return nil, fmt.Errorf("fetch personio entries: %w", err)
	}

	attendances, err := s.client.GetAttendance(employeeID, start, end)
	if err != nil {
		return nil, fmt.Errorf("fetch personio entries: %w", err)
	}

	return attendancesToEntries(attendances), nil
}

func (s *PersonioSink) IsApplicable(e Entry) bool {
	return e.Type == EntryTypeWork
}

func (s *PersonioSink) ProcessActions(actions []Action, localEntries []Entry) error {
	employeeID, err := s.client.GetEmployeeID()
	if err != nil {
		return fmt.Errorf("fetch personio entries: %w", err)
	}

	dailyActions := groupActionsByDay(actions)
	dailyEntries := groupEntriesByDay(localEntries)
	for day, actions := range dailyActions {
		if addActionCount(actions) < 1 {
			if err := s.client.RemoveAttendances(employeeID, day); err != nil {
				return fmt.Errorf("remove personio actions for day %q: %w", day, err)
			}
			continue
		}

		attendances := entriesToAttendances(dailyEntries[day])
		if err := s.client.CreateAttendances(employeeID, day, attendances); err != nil {
			return fmt.Errorf("update personio actions for day %q: %w", day, err)
		}
	}

	return nil
}
