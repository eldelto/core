package clockify

import (
	"net/url"
	"strconv"
	"time"

	"github.com/eldelto/core/internal/rest"
)

// Clockify doesn't accept proper RFC3339 times so we pretend that the
// given time is UTC even though it isn't. Also Clockify expects the
// given times to be in the user's configured timezone which most
// probably is the same as the computer they make the API calls from.
const clockifyFormat = "2006-01-02T15:04:05Z"

type Client struct {
	Host *url.URL
	Auth rest.Authenticator
}

func NewClient(host *url.URL, auth rest.Authenticator) *Client {
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
	endpoint := c.Host.JoinPath("api/v1/user")
	response := Myself{}
	if err := rest.GET(endpoint).
		Auth(c.Auth).
		ResponseAs(&response).
		Run(); err != nil {
		return Myself{}, err
	}

	return response, nil
}

func (c *Client) FetchTimeEntriesPaged(myself Myself, start, end time.Time, page int) ([]TimeEntry, error) {
	endpoint := c.Host.JoinPath("api/v1/workspaces", myself.WorkspaceId, "user", myself.UserId, "time-entries")
	query := endpoint.Query()
	query.Add("start", start.Format(clockifyFormat))
	query.Add("end", end.Format(clockifyFormat))
	query.Add("page", strconv.Itoa(page))
	endpoint.RawQuery = query.Encode()

	response := []TimeEntry{}
	if err := rest.GET(endpoint).
		Auth(c.Auth).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) FetchTimeEntries(myself Myself, start, end time.Time) ([]TimeEntry, error) {
	total := []TimeEntry{}
	//  Page 0 and Page 1 are the same in Clockify 🤦
	for page, responseCnt := 1, 0; page == 1 || responseCnt > 0; page++ {
		res, err := c.FetchTimeEntriesPaged(myself, start, end, page)
		if err != nil {
			return nil, err
		}
		responseCnt = len(res)
		total = append(total, res...)
	}
	return total, nil
}
