package jira_test

import (
	"testing"
	"time"

	"github.com/eldelto/core/internal/jira"
	"github.com/eldelto/core/internal/rest"
	. "github.com/eldelto/core/internal/testutils"
)

func TestClientCalls(t *testing.T) {
	// Skipped because nobody volunteered to check-in their credentials 🥲
	t.Skip()

	auth := &rest.BasicAuth{
		Username: "dominic.aschauer",
		Password: "<password>",
	}
	client := &jira.Client{Host: "https://jira.tmp.com"}

	tests := []struct {
		url  string
		call func(c *jira.Client, auth rest.Authenticator) (any, error)
		want any
	}{
		{
			url: "/rest/api/2/myself",
			call: func(c *jira.Client, auth rest.Authenticator) (any, error) {
				return c.FetchMyself(auth)
			},
			want: jira.Myself{Key: "JIRAUSER17870"},
		},
		{
			url: "/rest/api/2/issue",
			call: func(c *jira.Client, auth rest.Authenticator) (any, error) {
				return c.FetchIssue(auth, "ER-590")
			},
			want: jira.Issue{Key: "ER-590", ID: "433889"},
		},
		{
			url: "/rest/tempo-timesheets/4/worklogs/search",
			call: func(c *jira.Client, auth rest.Authenticator) (any, error) {
				return c.SearchForWorklogs(auth, "JIRAUSER17870",
					time.Date(2023, 11, 26, 0, 0, 0, 0, time.UTC),
					time.Date(2023, 11, 27, 0, 0, 0, 0, time.UTC),
				)
			},
			want: []jira.Worklog{
				{
					TimeSpentSeconds: 1140,
					Issue:            jira.WorklogIssue{Key: "ER-590", ID: 433889},
					TempoWorklogID:   809761,
					Started:          rest.ISO8601Time(time.Date(2023, 11, 27, 15, 4, 0, 0, time.UTC)), //"2023-11-27 15:04:00.000",
					Worker:           "JIRAUSER17870",
				},
				{
					TimeSpentSeconds: 10800,
					Issue:            jira.WorklogIssue{Key: "HUM-13205", ID: 497680},
					TempoWorklogID:   809764,
					Started:          rest.ISO8601Time(time.Date(2023, 11, 27, 15, 4, 0, 0, time.UTC)), //"2023-11-27 15:04:00.000",
					Worker:           "JIRAUSER17870",
				},
				{
					TimeSpentSeconds: 3000,
					Issue:            jira.WorklogIssue{Key: "HUM-13318", ID: 510026},
					TempoWorklogID:   809765,
					Started:          rest.ISO8601Time(time.Date(2023, 11, 27, 15, 4, 0, 0, time.UTC)), //"2023-11-27 15:04:00.000",
					Worker:           "JIRAUSER17870",
				},
			},
		},
		// Works but the TempoWorklogID increases each time.
		// {
		// 	url: "/rest/tempo-timesheets/4/worklogs",
		// 	call: func(c *jira.Client, auth rest.Authenticator) (any, error) {
		// 		request := jira.WorklogEntryRequest{
		// 			Worker:           "JIRAUSER17870",
		// 			OriginTaskID:     "433889",
		// 			TimeSpentSeconds: 1,
		// 			Comment:          "test",
		// 			Started:          now.Format(rest.ISO8601Format),
		// 		}
		// 		return c.CreateWorklogEntry(auth, request)
		// 	},
		// 	want: []jira.Worklog{
		// 		{
		// 			TimeSpentSeconds: 1,
		// 			Issue:            jira.WorklogIssue{Key: "ER-590", ID: 433889},
		// 			TempoWorklogID:   822983,
		// 			Started:          now.Format(rest.ISO8601Format),
		// 			Worker:           "JIRAUSER17870",
		// 		},
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := tt.call(client, auth)
			AssertNoError(t, err, "client call")
			AssertEquals(t, tt.want, got, "response data")
		})
	}
}
