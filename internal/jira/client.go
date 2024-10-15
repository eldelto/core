package jira

import (
	"net/url"
	"time"

	"github.com/eldelto/core/internal/rest"
)

type Client struct {
	Host *url.URL
}

type Myself struct {
	Key      string `json:"key"`
	TimeZone string `json:"timeZone"`
}

func (c *Client) FetchMyself(auth rest.Authenticator) (Myself, error) {
	endpoint := c.Host.JoinPath("rest/api/2/myself")
	response := Myself{}

	if err := rest.GET(endpoint).
		Auth(auth).
		ResponseAs(&response).
		Run(); err != nil {
		return Myself{}, err
	}

	return response, nil
}

type Issue struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

func (c *Client) FetchIssue(auth rest.Authenticator, issueKey string) (Issue, error) {
	endpoint := c.Host.JoinPath("rest/api/2/issue", issueKey)
	response := Issue{}

	if err := rest.GET(endpoint).
		Auth(auth).
		ResponseAs(&response).
		Run(); err != nil {
		return Issue{}, err
	}

	return response, nil
}

type worklogsRequest struct {
	From   string   `json:"from"`
	To     string   `json:"to"`
	Worker []string `json:"worker"`
}

type WorklogIssue struct {
	Key string `json:"key"`
	ID  int    `json:"id"`
}

type Worklog struct {
	TimeSpentSeconds int              `json:"timeSpentSeconds"`
	Issue            WorklogIssue     `json:"issue"`
	TempoWorklogID   int              `json:"tempoWorklogId"`
	Started          rest.ISO8601Time `json:"started"`
	Worker           string           `json:"worker"`
}

func (c *Client) SearchForWorklogs(auth rest.Authenticator, userId string, from, to time.Time) ([]Worklog, error) {
	endpoint := c.Host.JoinPath("rest/tempo-timesheets/4/worklogs/search")
	request := worklogsRequest{
		Worker: []string{userId},
		From:   from.Format(time.DateOnly),
		To:     to.Format(time.DateOnly),
	}
	response := []Worklog{}

	if err := rest.POST(endpoint).
		Auth(auth).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, err
	}

	return response, nil
}

type WorklogEntryRequest struct {
	Worker           string `json:"worker"`
	Comment          string `json:"comment"`
	Started          string `json:"started"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	OriginTaskID     string `json:"originTaskId"`
}

func (c *Client) CreateWorklogEntry(auth rest.Authenticator, request WorklogEntryRequest) ([]Worklog, error) {
	endpoint := c.Host.JoinPath("rest/tempo-timesheets/4/worklogs")
	response := []Worklog{}

	if err := rest.POST(endpoint).
		Auth(auth).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) DeleteWorklogEntry(auth rest.Authenticator, worklogID string) error {
	endpoint := c.Host.JoinPath("rest/tempo-timesheets/4/worklogs", worklogID)
	return rest.DELETE(endpoint).Auth(auth).Run()
}
