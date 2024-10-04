package jira

import (
	"time"

	"github.com/eldelto/core/internal/rest"
)

type Client struct {
	Host string
}

type Myself struct {
	Key      string `json:"key"`
	TimeZone string `json:"timeZone"`
}

func (c *Client) FetchMyself(auth rest.Authenticator) (Myself, error) {
	response := Myself{}
	if err := rest.Get(c.Host+"/rest/api/2/myself", auth, &response); err != nil {
		return Myself{}, err
	}

	return response, nil
}

type Issue struct {
	Key string `json:"key"`
	ID  string `json:"id"`
}

func (c *Client) FetchIssue(auth rest.Authenticator, issueKey string) (Issue, error) {
	response := Issue{}
	if err := rest.Get(c.Host+"/rest/api/2/issue/"+issueKey, auth, &response); err != nil {
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
	request := worklogsRequest{
		Worker: []string{userId},
		From:   from.Format(time.DateOnly),
		To:     to.Format(time.DateOnly),
	}

	response := []Worklog{}

	if err := rest.Post(c.Host+"/rest/tempo-timesheets/4/worklogs/search", auth, request, &response, nil); err != nil {
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
	response := []Worklog{}

	if err := rest.Post(c.Host+"/rest/tempo-timesheets/4/worklogs", auth, request, &response, nil); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) DeleteWorklogEntry(auth rest.Authenticator, worklogID string) error {
	return rest.Delete(c.Host+"/rest/tempo-timesheets/4/worklogs/"+worklogID, auth)
}
