package gitlab

import (
	"net/url"
	"strconv"
	"time"

	"github.com/eldelto/core/internal/rest"
)

type Client struct {
	host *url.URL
	auth rest.Authenticator
}

func NewClient(host *url.URL, auth rest.Authenticator) *Client {
	return &Client{
		host: host,
		auth: auth,
	}
}

type Issue struct {
	ID        int `json:"id"`
	IID       int `json:"iid"`
	ProjectID int `json:"project_id"`
}

func (c *Client) ListProjectIssues(projectID int, start, end time.Time) ([]Issue, error) {
	endpoint := c.host.JoinPath("api/v4/projects", strconv.Itoa(projectID), "issues")

	query := endpoint.Query()
	query.Add("updated_after", start.Format(time.RFC3339))
	query.Add("updated_before", end.Format(time.RFC3339))
	query.Add("per_page", "100")
	endpoint.RawQuery = query.Encode()

	response := []Issue{}

	if err := rest.GET(endpoint).
		Auth(c.auth).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, err
	}

	return response, nil
}

type Note struct {
	ID        int    `json:"id"`
	ProjectID int    `json:"project_id"`
	IssueIID  int    `json:"noteable_iid"`
	Body      string `json:"body"`
}

func (c *Client) ListNotes(issue Issue) ([]Note, error) {
	endpoint := c.host.JoinPath("api/v4/projects",
		strconv.Itoa(issue.ProjectID),
		"issues",
		strconv.Itoa(issue.IID),
		"notes")
	response := []Note{}

	if err := rest.GET(endpoint).
		Auth(c.auth).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, err
	}

	return response, nil
}

type noteRequest struct {
	Body string `json:"body"`
}

func (c *Client) CreateNote(issue Issue, text string) (Note, error) {
	endpoint := c.host.JoinPath("api/v4/projects",
		strconv.Itoa(issue.ProjectID),
		"issues",
		strconv.Itoa(issue.IID),
		"notes")
	request := noteRequest{Body: text}
	response := Note{}

	if err := rest.POST(endpoint).
		Auth(c.auth).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return response, err
	}

	return response, nil
}

func (c *Client) UpdateNote(note Note, text string) (Note, error) {
	endpoint := c.host.JoinPath("api/v4/projects",
		strconv.Itoa(note.ProjectID),
		"issues",
		strconv.Itoa(note.IssueIID), // TODO: This needs to be the issue.ID
		"notes",
		strconv.Itoa(note.ID))
	request := noteRequest{Body: text}
	response := Note{}

	if err := rest.PUT(endpoint).
		Auth(c.auth).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return response, err
	}

	return response, nil
}

type spentTimeRequest struct {
	Duration string `json:"duration"`
}

type TimeSpent struct {
	TotalTimeSpentSec int `json:"total_time_spent"`
}

func (c *Client) AddTimeSpent(issue Issue, durationSec int) (TimeSpent, error) {
	endpoint := c.host.JoinPath("api/v4/projects",
		strconv.Itoa(issue.ProjectID),
		"issues",
		strconv.Itoa(issue.IID),
		"add_spent_time")
	request := spentTimeRequest{Duration: strconv.Itoa(durationSec) + "s"}
	response := TimeSpent{}

	if err := rest.POST(endpoint).
		Auth(c.auth).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return response, err
	}

	return response, nil
}

func (c *Client) ResetTimeSpent(issue Issue, durationSec int) (TimeSpent, error) {
	endpoint := c.host.JoinPath("api/v4/projects",
		strconv.Itoa(issue.ProjectID),
		"issues",
		strconv.Itoa(issue.IID),
		"reset_spent_time")
	response := TimeSpent{}

	if err := rest.POST(endpoint).
		Auth(c.auth).
		ResponseAs(&response).
		Run(); err != nil {
		return response, err
	}

	return response, nil
}
