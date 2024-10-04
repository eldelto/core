package clockify

import (
	"fmt"
	"time"

	"github.com/eldelto/core/internal/rest"
)

type Client struct {
	Host string
	Auth rest.Authenticator
}

func NewClient(host string, auth rest.Authenticator) *Client {
	return &Client{
		Host: host,
		Auth: auth,
	}
}

type Settings struct {
	TimeZone string `json:"timeZone"`
}

type Myself struct {
	UserId      string   `json:"id"`
	WorkspaceId string   `json:"defaultWorkspace"`
	Settings    Settings `json:"settings"`
}

type TimeInterval struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Duration string    `json:"duration"`
}

type HourlyRate struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type CostRate struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type TimeEntry struct {
	ID                string       `json:"id"`
	Description       string       `json:"description"`
	TagIds            []string     `json:"tagIds"`
	UserID            string       `json:"userId"`
	Billable          bool         `json:"billable"`
	TaskID            string       `json:"taskId"`
	ProjectID         string       `json:"projectId"`
	WorkspaceID       string       `json:"workspaceId"`
	TimeInterval      TimeInterval `json:"timeInterval"`
	CustomFieldValues []string     `json:"customFieldValues"`
	Type              string       `json:"type"`
	KioskID           string       `json:"kioskId"`
	HourlyRate        HourlyRate   `json:"hourlyRate"`
	CostRate          CostRate     `json:"costRate"`
	IsLocked          bool         `json:"isLocked"`
}

func (c *Client) FetchMyself() (Myself, error) {
	response := Myself{}
	if err := rest.Get(c.Host+"api/v1/user", c.Auth, &response); err != nil {
		return Myself{}, err
	}

	return response, nil
}

func (c *Client) FetchTimeEntriesPaged(myself Myself, from time.Time, page int) ([]TimeEntry, error) {
	response := []TimeEntry{}
	if err := rest.Get(c.Host+fmt.Sprintf("api/v1/workspaces/%s/user/%s/time-entries?page=%d&start=%s", myself.WorkspaceId, myself.UserId, page, from.Format(time.RFC3339)), c.Auth, &response); err != nil {
		return []TimeEntry{}, err
	}

	return response, nil
}

func (c *Client) FetchTimeEntries(myself Myself, from time.Time) ([]TimeEntry, error) {
	total := []TimeEntry{}
	//  Page 0 and Page 1 are the same in Clockify 🤦
	for page, responseCnt := 1, 0; page == 1 || responseCnt > 0; page++ {
		res, err := c.FetchTimeEntriesPaged(myself, from, page)
		if err != nil {
			return nil, err
		}
		responseCnt = len(res)
		total = append(total, res...)
	}
	return total, nil
}
