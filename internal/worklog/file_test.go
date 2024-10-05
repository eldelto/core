package worklog

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
)

//go:embed test.csv
var testCSV string

//go:embed test.org
var testOrg string

func generateTestCSV(ticket, from, to string) string {
	return fmt.Sprintf("ticket,from,to\n%s,%s,%s", ticket, from, to)
}

func TestParseFromCSV(t *testing.T) {
	time.Local = time.UTC

	tests := []struct {
		name    string
		csv     string
		want    string
		wantErr bool
	}{
		{
			"valid CSV",
			testCSV,
			"[{ER-590  0 2023-12-01 09:00:00 +0000 UTC 2023-12-01 09:20:00 +0000 UTC} {HUM-123  0 2023-12-01 08:34:00 +0000 UTC 2023-12-01 12:00:00 +0000 UTC} {ER-590  0 2023-12-02 09:00:00 +0000 UTC 2023-12-02 09:20:00 +0000 UTC} {HUM-428  0 2023-12-02 07:24:00 +0000 UTC 2023-12-02 08:10:00 +0000 UTC}]",
			false,
		},
		{
			"another valid ticket",
			generateTestCSV("HUM-1", "2023-12-01 08:00", "2023-12-01 08:10"),
			"[{HUM-1  0 2023-12-01 08:00:00 +0000 UTC 2023-12-01 08:10:00 +0000 UTC}]",
			false,
		},
		{
			"no ticket number",
			generateTestCSV(" bla bla", "2023-12-01 08:00", "2023-12-01 08:10"),
			"",
			true,
		},
		{
			"invalid duration",
			generateTestCSV("HUM-1", "2023-12-01 08:00", "2023-12-01 07:10"),
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFromCSV(strings.NewReader(tt.csv), time.Time{}, time.Now())
			if tt.wantErr {
				AssertError(t, err, "ParseFromCSV")
				return
			}

			AssertNoError(t, err, "ParseFromCSV")
			AssertEquals(t, tt.want, fmt.Sprint(got), "entries")
		})
	}
}

func TestParseFromOrg(t *testing.T) {
	time.Local = time.UTC

	want := "[{HUM-13311  0 2023-12-11 13:05:00 +0000 UTC 2023-12-11 14:17:00 +0000 UTC} {HUM-13311  0 2023-12-11 11:30:00 +0000 UTC 2023-12-11 12:09:00 +0000 UTC} {HUM-13403  0 2023-12-28 13:19:00 +0000 UTC 2023-12-28 15:21:00 +0000 UTC}]"

	got, err := parseFromOrg(strings.NewReader(testOrg), time.Time{}, time.Now())
	AssertNoError(t, err, "ParseFromOrg")
	AssertEquals(t, want, fmt.Sprint(got), "entries")
}

func BenchmarkParseOrg(b *testing.B) {
	b.Skip()

	source := NewFileSource("/Users/dominicaschauer/Documents/workspace/gtd/work-notes.org")

	now := time.Now()
	oneWeekAgo := now.Add(-7 * 24 * time.Hour)
	for n := 0; n < b.N; n++ {
		_, err := source.FetchEntries(oneWeekAgo, now)
		if err != nil {
			b.Log(err)
			b.FailNow()
		}
	}
}
