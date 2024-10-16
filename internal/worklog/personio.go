package worklog

import (
	"fmt"
	"log"
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

func actionsToAttendances(actions []Action) []personio.Attendance {
	attendances := []personio.Attendance{}
	for _, a := range actions {
		if a.Operation == Remove {
			continue
		}

		attendance := personio.Attendance{
			Start:   a.Entry.From,
			End:     a.Entry.To,
			Comment: a.Entry.Ticket,
		}
		attendances = append(attendances, attendance)
	}

	return attendances
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

/*
   Problem: Personio needs days to be updated at once, meaning I can't
   add remove single entries without knowing everything that happened
   on that day.

   PUT /attendance/:dayID updates a given day with the attendence
   periods contained in the request but can't clear all periods of
   that day.

   DELETE /attendance/:dayID/periods clears all periods.

   If there are changes for a given day:
   1. No additions => DELETE periods
   2. Overwrite remote entries with local entries (PUT)

   SyncEntries(start, end time.Time, local []Entries) error
*/

func (s *PersonioSink) UpdateAttendances(employeeID personio.EmployeeID, day time.Time, attendances []personio.Attendance) error {
	// TODO: Only remove if there is at least one remove action for this day.
	if err := s.client.RemoveAttendances(employeeID, day); err != nil {
		err := fmt.Errorf("update attendances: %w", err)
		log.Println(err)
	}

	if err := s.client.CreateAttendances(employeeID, day, attendances); err != nil {
		return fmt.Errorf("update attendances: %w", err)
	}

	return nil
}

func (s *PersonioSink) ProcessActions(actions []Action) error {
	employeeID, err := s.client.GetEmployeeID()
	if err != nil {
		return fmt.Errorf("fetch personio entries: %w", err)
	}

	dailyActions := groupByDay(actions)
	for day, actions := range dailyActions {
		attendances := actionsToAttendances(actions)
		if len(attendances) < 1 {
			if err := s.client.RemoveAttendances(employeeID, day); err != nil {
				return fmt.Errorf("remove personio actions for day %q: %w", day, err)
			}
			continue
		}

		if err := s.UpdateAttendances(employeeID, day, attendances); err != nil {
			return fmt.Errorf("update personio actions for day %q: %w", day, err)
		}
	}

	return nil
}
