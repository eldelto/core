package personio

import (
	"fmt"
	"net/http"
	"net/url"
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
const personioFormat = "2006-01-02T15:04:05Z"

type EmployeeID int

type Client struct {
	host           *url.URL
	configProvider *cli.ConfigProvider
	employeeID     EmployeeID
	cookies        []http.Cookie
	dayIDs         map[string]string
}

func NewClient(host *url.URL, configProvider *cli.ConfigProvider) *Client {
	return &Client{
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
	otp, err := cli.ReadInput("\nPlease enter your OTP code:\n")
	if err != nil {
		return fmt.Errorf("OTP code for microsoft auth: %w", err)
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
	if err := session.WriteTo("input[name='otc']", otp); err != nil {
		return fmt.Errorf("write OTP for microsoft auth: %w", err)
	}
	if err := session.Click("input.button_primary"); err != nil {
		return fmt.Errorf("submit OTP for microsoft auth: %w", err)
	}

	return nil
}

func (c *Client) Login() error {
	session, err := webdriver.NewSession()
	if err != nil {
		return fmt.Errorf("personio login: %w", err)
	}
	defer session.Close()

	url := c.host.JoinPath("login", "index")
	if err := session.NavigateTo(url); err != nil {
		return fmt.Errorf("navigate to personio login %q: %w", url, err)
	}

	element, err := session.FindElement("a[href='https://firstbird.personio.de/oauth/authorize']")
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
	if _, err := session.FindElement("a[data-test-id='navsidebar-time_tracking']"); err != nil {
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

type AttendanceAttributes struct {
	Start           time.Time `json:"start"`
	End             time.Time `json:"end"`
	Comment         string    `json:"comment"`
	AttendanceDayID string    `json:"attendance_day_id"`
	PeriodType      string    `json:"period_type"`
}

type AttendancePeriode struct {
	ID         string               `json:"id"`
	Attributes AttendanceAttributes `json:"attributes"`
}

type attendanceDay struct {
	ID         string `json:"id"`
	Attributes struct {
		Day string `json:"day"`
	} `json:"attributes"`
}

type getAttendanceResponse struct {
	Data struct {
		AttendancePeriods struct {
			Data []AttendancePeriode `json:"data"`
		} `json:"attendance_periods"`
		AttendanceDays struct {
			Data []attendanceDay `json:"data"`
		} `json:"attendance_days"`
	} `json:"data"`
}

func (c *Client) GetAttendance(employeeID EmployeeID, start, end time.Time) ([]AttendancePeriode, error) {
	endpoint := c.host.JoinPath(fmt.Sprintf("/svc/attendance-bff/attendance-calendar/%d", employeeID))
	query := endpoint.Query()
	query.Add("start_date", start.Format(time.DateOnly))
	query.Add("end_date", end.Format(time.DateOnly))
	endpoint.RawQuery = query.Encode()

	var response getAttendanceResponse
	if err := rest.GET(endpoint).
		Cookies(c.cookies).
		ResponseAs(&response).
		Run(); err != nil {
		return nil, fmt.Errorf("get attendance for employee %d: %w",
			employeeID, err)
	}

	// Cache date <-> day ID mapping for other requests.
	for _, day := range response.Data.AttendanceDays.Data {
		c.dayIDs[day.Attributes.Day] = day.ID
	}

	return response.Data.AttendancePeriods.Data, nil
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
		return "", fmt.Errorf("could not resolve day ID for date %q, cache=%v", date, c.dayIDs)
	}

	return dayID, nil
}

type attendancePeriod struct {
	ID             string  `json:"id"`
	ProjectID      *string `json:"project_id"`
	PeriodType     string  `json:"period_type"`
	LegacyBreakMin int     `json:"legacy_break_min"`
	Comment        *string `json:"comment"`
	Start          string  `json:"start"`
	End            string  `json:"end"`
}

type createAttendanceRequest struct {
	EmployeeID           EmployeeID         `json:"employee_id"`
	Periods              []attendancePeriod `json:"periods"`
	RulesViolationReason *string            `json:"rules_violation_reason"`
}

type Attendance struct {
	Start time.Time
	End   time.Time
}

func (c *Client) CreateAttendances(employeeID EmployeeID, day time.Time, attendances []Attendance) error {
	dayID, err := c.resolveDayID(day)
	if err != nil {
		return err
	}

	endpoint := c.host.JoinPath("api/v1/attendances/days", dayID)

	periods := make([]attendancePeriod, len(attendances))
	for i, attendance := range attendances {
		// TODO: Validate start/end to be equal to day.

		periodID, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("create attendance periode ID: %w", err)
		}

		periods[i] = attendancePeriod{
			ID:         periodID.String(),
			PeriodType: "work",
			Start:      attendance.Start.Format(personioFormat),
			End:        attendance.End.Format(personioFormat),
		}
	}

	request := createAttendanceRequest{
		EmployeeID: employeeID,
		Periods:    periods,
	}

	csrfToken := ""
	for _, c := range c.cookies {
		if c.Name == "XSRF-TOKEN" {
			csrfToken = c.Value
			break
		}
	}

	var response map[string]any

	if err := rest.PUT(endpoint).
		Cookies(c.cookies).
		AddHeader("x-csrf-token", csrfToken).
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
