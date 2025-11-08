package gitlab_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/eldelto/core/internal/gitlab"
	"github.com/eldelto/core/internal/rest"
	. "github.com/eldelto/core/internal/testutils"
)

func TestClientCalls(t *testing.T) {
	t.Skip()

	host, err := url.Parse("https://gitlab.acme.asdf")
	AssertNoError(t, err, "url.Parse")

	auth := &rest.HeaderAuth{
		Name:  "PRIVATE-TOKEN",
		Value: "gitlab-<token>",
	}
	client := gitlab.NewClient(host, auth)

	// Note: Commented tests work but change their data to frequently.
	tests := []struct {
		url  string
		call func(c *gitlab.Client) (any, error)
		want any
	}{
		{
			url: "GET /api/v4/projects/:id/issues/:id",
			call: func(c *gitlab.Client) (any, error) {
				return c.ListProjectIssues(49, time.Now(), time.Now())
			},
			want: []gitlab.Issue{{ID: 3935, ProjectID: 46}, {ID: 3894, ProjectID: 46}, {ID: 3841, ProjectID: 46}, {ID: 3840, ProjectID: 46}, {ID: 3834, ProjectID: 46}, {ID: 3832, ProjectID: 46}, {ID: 3693, ProjectID: 46}},
		},
		// {
		// 	url: "GET /api/v4/projects/:id/issues/:iid/notes",
		// 	call: func(c *gitlab.Client) (any, error) {
		// 		return c.ListNotes(gitlab.Issue{ID:419, ProjectID: 46})
		// 	},
		// 	want: []gitlab.Note{},
		// },
		// {
		// 	url: "POST /api/v4/projects/:id/issues/:iid/notes",
		// 	call: func(c *gitlab.Client) (any, error) {
		// 		return c.CreateNote(gitlab.Issue{ID:419, ProjectID: 46}, "API note")
		// 	},
		// 	want: gitlab.Note{ID:35228, ProjectID:46, IssueID:419, Body:"API note"},
		// },
		{
			url: "PUT /api/v4/projects/:id/issues/:iid/notes/:id",
			call: func(c *gitlab.Client) (any, error) {
				return c.UpdateNote(gitlab.Note{ID: 35228, ProjectID: 46, IssueID: 419}, "API note update")
			},
			want: gitlab.Note{ID: 35228, ProjectID: 46, IssueID: 419, Body: "API note update"},
		},
		// {
		// 			url: "POST /api/v4/projects/:id/issues/:iid/add_spent_time",
		// 	call: func(c *gitlab.Client) (any, error) {
		// 		return c.AddTimeSpent(gitlab.Issue{ID:419, ProjectID: 46}, 10)
		// 	},
		// 	want: gitlab.TimeSpent{TotalTimeSpentSec:10},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got, err := tt.call(client)
			AssertNoError(t, err, "client call")
			AssertEquals(t, tt.want, got, "response data")
		})
	}
}
