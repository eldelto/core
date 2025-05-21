package personio

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/rest"
	"github.com/eldelto/core/internal/webdriver"
	"github.com/google/uuid"
)

// Personio doesn't accept proper RFC3339 times so we pretend that the
// given time is UTC even though it isn't. Also Personio expects the
// given times to be in the user's configured timezone which most
// probably is the same as the computer they make the API calls from.
const personioFormat = "2006-01-02T15:04:05"

type PersonioTime time.Time

func (t *PersonioTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		*t = PersonioTime(time.Time{})
		return nil
	}
	time, err := time.Parse(personioFormat, s)
	*t = PersonioTime(time)
	return err
}

func (t *PersonioTime) MarshalJSON() ([]byte, error) {
	time := time.Time(*t)
	if time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(time.Format(personioFormat)), nil
}

type EmployeeID int

type Client struct {
	loginHost      *url.URL
	host           *url.URL
	configProvider *cli.ConfigProvider
	employeeID     EmployeeID
	cookies        []http.Cookie
	dayIDs         map[string]string
}

func NewClient(loginHost, host *url.URL, configProvider *cli.ConfigProvider) *Client {
	return &Client{
		loginHost:      loginHost,
		host:           host,
		configProvider: configProvider,
		dayIDs:         map[string]string{},
	}
}

func (c *Client) authorizeViaMicrosoft(session *webdriver.Session) error {
	fmt.Println(cli.Brown("Starting authorization via Microsoft"))

	emailInput, err := session.FindElement("input[type='email']")
	if err != nil {
		return fmt.Errorf("microsoft auth: %w", err)
	}

	if err := session.Minimize(); err != nil {
		return fmt.Errorf("microsoft auth: %w", err)
	}

	// TODO: term.ReadPassword(int(syscall.Stdin))
	email, err := c.configProvider.Get("microsoft.email")
	if err != nil {
		return fmt.Errorf("e-mail for microsoft auth: %w", err)
	}
	password, err := c.configProvider.Get("microsoft.password")
	if err != nil {
		return fmt.Errorf("password for microsoft auth: %w", err)
	}
	if err := session.Maximize(); err != nil {
		return fmt.Errorf("microsoft auth: %w", err)
	}

	// Enter the user's E-mail.
	if err := session.WriteToElement(emailInput, email); err != nil {
		return fmt.Errorf("write E-mail for microsoft auth: %w", err)
	}
	if err := session.Click("input.button_primary"); err != nil {
		return fmt.Errorf("submit E-mail for microsoft auth: %w", err)
	}
	time.Sleep(2 * time.Second)

	// Enter the user's password.
	if err := session.WriteTo("input[name='passwd']", password); err != nil {
		return fmt.Errorf("write password for microsoft auth: %w", err)
	}
	if err := session.Click("input.button_primary"); err != nil {
		return fmt.Errorf("submit password for microsoft auth: %w", err)
	}
	time.Sleep(2 * time.Second)

	// Enter the user's OTP code.
	if session.HasElement("input[name='otc']") {
		if err := session.Minimize(); err != nil {
			return fmt.Errorf("microsoft auth: %w", err)
		}
		otp, err := cli.ReadInput("\nPlease enter your OTP code:\n")
		if err != nil {
			return fmt.Errorf("OTP code for microsoft auth: %w", err)
		}
		if err := session.Maximize(); err != nil {
			return fmt.Errorf("microsoft auth: %w", err)
		}

		if err := session.WriteTo("input[name='otc']", otp); err != nil {
			return fmt.Errorf("write OTP for microsoft auth: %w", err)
		}
		if err := session.Click("input.button_primary"); err != nil {
			return fmt.Errorf("submit OTP for microsoft auth: %w", err)
		}
		time.Sleep(2 * time.Second)
	}

	// Click the stay signed in prompt if it exists.
	if err := session.Click(".sign-in-box input.button_primary"); err == nil {
		time.Sleep(2 * time.Second)
	}

	return nil
}

func (c *Client) Login() error {
	session, err := webdriver.NewSession()
	if err != nil {
		return fmt.Errorf("personio login: %w", err)
	}
	defer session.Close()

	if err := session.Maximize(); err != nil {
		return fmt.Errorf("maximise new sessio: %w", err)
	}

	url := c.loginHost.JoinPath("login", "index")
	if err := session.NavigateTo(url); err != nil {
		return fmt.Errorf("navigate to personio login %q: %w", url, err)
	}

	element, err := session.FindElement("button._social-button-oidc")
	if err != nil {
		return fmt.Errorf("locate authorize button: %w", err)
	}

	if err := session.ClickElement(element); err != nil {
		return fmt.Errorf("click authorize button: %w", err)
	}

	// Wait for the redirect to finish.
	time.Sleep(2 * time.Second)

	authProviderURL, err := session.URL()
	if err != nil {
		return fmt.Errorf("redirect URL: %w", err)
	}

	host := authProviderURL.Host
	switch host {
	case "login.microsoftonline.com":
		if err := c.authorizeViaMicrosoft(session); err != nil {
			return err
		}
	default:
		return fmt.Errorf("authentication via %q is not supported", host)
	}

	if err := session.WaitForHost(c.host.Host); err != nil {
		return fmt.Errorf("personio login callback: %w", err)
	}

	// Wait for the browser to settle after the redirect.
	time.Sleep(5 * time.Second)
	if _, err := session.FindElement("a[data-test-id='navsidebar-companyTimeline']"); err != nil {
		return fmt.Errorf("personio login callback finishing: %w", err)

	}

	cookies, err := session.Cookies()
	if err != nil {
		return fmt.Errorf("personio login: %w", err)
	}
	c.cookies = cookies

	return nil
}

type getContextResponse struct {
	Success bool `json:"success"`
	Data    struct {
		User struct {
			ID EmployeeID `json:"id"`
		} `json:"user"`
	} `json:"data"`
}

func (c *Client) GetEmployeeID() (EmployeeID, error) {
	if c.employeeID != 0 {
		return c.employeeID, nil
	}

	if c.cookies == nil {
		if err := c.Login(); err != nil {
			return 0, fmt.Errorf("personio get context: %w", err)
		}
	}

	endpoint := c.host.JoinPath("/api/v1/navigation/context")

	var response getContextResponse
	if err := rest.GET(endpoint).
		Cookies(c.cookies).
		ResponseAs(&response).
		Run(); err != nil {
		return 0, fmt.Errorf("personio get context: %w", err)
	}

	c.employeeID = response.Data.User.ID

	return c.employeeID, nil
}

type AttendancePeriode struct {
	ID      string       `json:"id"`
	Start   PersonioTime `json:"start"`
	End     PersonioTime `json:"end"`
	Comment string       `json:"comment"`
	Type    string       `json:"type"`
}

type getTimeSheetResponse struct {
	Timecards []struct {
		DayID   string              `json:"day_id"`
		Date    string              `json:"date"`
		Periods []AttendancePeriode `json:"periods"`
	} `json:"timecards"`
}

func (c *Client) GetAttendance(employeeID EmployeeID, start, end time.Time) ([]AttendancePeriode, error) {
	endpoint := c.host.JoinPath(fmt.Sprintf("/svc/attendance-bff/v1/timesheet/%d", employeeID))
	query := endpoint.Query()
	query.Add("start_date", start.Format(time.DateOnly))
	query.Add("end_date", end.Format(time.DateOnly))
	endpoint.RawQuery = query.Encode()

	var response getTimeSheetResponse
	if err := rest.GET(endpoint).
		Cookies(c.cookies).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, fmt.Errorf("get attendance for employee %d: %w",
			employeeID, err)
	}

	periods := []AttendancePeriode{}
	for _, timecard := range response.Timecards {
		if timecard.DayID == "" {
			continue
		}

		// Cache date <-> day ID mapping for other requests.
		c.dayIDs[timecard.Date] = timecard.DayID

		// This flattens multiple days in a weird way but doesn't
		// really matter at the moment.
		periods = append(periods, timecard.Periods...)
	}

	return periods, nil
}

func (c *Client) resolveDayID(day time.Time) (string, error) {
	date := day.Format(time.DateOnly)
	dayID, ok := c.dayIDs[date]
	if ok {
		return dayID, nil
	}

	start := day.Add(-7 * 24 * time.Hour)
	end := day.Add(7 * 24 * time.Hour)
	_, err := c.GetAttendance(c.employeeID, start, end)
	if err != nil {
		return "", fmt.Errorf("resolve day ID: %w", err)
	}

	dayID, ok = c.dayIDs[date]
	if !ok {
		newId, err := uuid.NewRandom()
		if err != nil {
			return "", fmt.Errorf("resolve day ID: %w", err)
		}
		dayID = newId.String()
	}

	return dayID, nil
}

type attendancePeriod struct {
	ID         string  `json:"id"`
	ProjectID  *string `json:"project_id"`
	PeriodType string  `json:"period_type"`
	Comment    *string `json:"comment"`
	Start      string  `json:"start"`
	End        string  `json:"end"`
}

type createAttendanceRequest struct {
	EmployeeID EmployeeID         `json:"employee_id"`
	Periods    []attendancePeriod `json:"periods"`
}

type Attendance struct {
	Start   time.Time
	End     time.Time
	Comment string
}

func attendanceToPeriod(a Attendance) (attendancePeriod, error) {
	// TODO: Validate start/end to be equal to day.
	periodID, err := uuid.NewRandom()
	if err != nil {
		return attendancePeriod{}, fmt.Errorf("create attendance periode ID: %w", err)
	}

	return attendancePeriod{
		ID:         periodID.String(),
		PeriodType: "work",
		Start:      a.Start.Format(personioFormat),
		End:        a.End.Format(personioFormat),
		Comment:    &a.Comment,
	}, nil
}

func (c *Client) CreateAttendances(employeeID EmployeeID, day time.Time, attendances []Attendance) error {
	dayID, err := c.resolveDayID(day)
	if err != nil {
		return err
	}

	endpoint := c.host.JoinPath("svc/attendance-api/v1/days", dayID)

	periods := make([]attendancePeriod, len(attendances))
	for i, attendance := range attendances {
		period, err := attendanceToPeriod(attendance)
		if err != nil {
			return err
		}
		periods[i] = period
	}

	request := createAttendanceRequest{
		EmployeeID: employeeID,
		Periods:    periods,
	}

	csrfToken := ""
	for _, c := range c.cookies {
		if c.Name == "ATHENA-XSRF-TOKEN" {
			csrfToken = c.Value
			break
		}
	}

	var response map[string]any

	if err := rest.PUT(endpoint).
		Cookies(c.cookies).
		AddHeader("x-athena-xsrf-token", csrfToken).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return fmt.Errorf("create attendance for employee %d: %w", employeeID, err)
	}

	return nil
}

func (c *Client) RemoveAttendances(employeeID EmployeeID, day time.Time) error {
	dayID, err := c.resolveDayID(day)
	if err != nil {
		return err
	}

	endpoint := c.host.JoinPath("api/v1/attendances/days", dayID, "periods")

	csrfToken := ""
	for _, c := range c.cookies {
		if c.Name == "XSRF-TOKEN" {
			csrfToken = c.Value
			break
		}
	}

	if err := rest.DELETE(endpoint).
		Cookies(c.cookies).
		AddHeader("x-csrf-token", csrfToken).
		Run(); err != nil {
		return fmt.Errorf("delete attendances for employee %d: %w", employeeID, err)
	}

	return nil
}
