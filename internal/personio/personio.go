package personio

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

func otpFlow(session *webdriver.Session) error {
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

	return nil
}

func isOtpPrompt(session *webdriver.Session) bool {
	return session.HasElement("input[name='otc']")
}

func fidoFlow(session *webdriver.Session) error {
	if err := session.Click(".sign-in-box input.button_primary"); err != nil {
		return fmt.Errorf("FIDO click continue: %w", err)
	}

	if err := session.WaitForURL("https://login.microsoftonline.com/common/SAS/ProcessAuth"); err != nil {
		return fmt.Errorf("FIDO wait for completed auth: %w", err)
	}

	return nil
}

func fidoQuickFlow(session *webdriver.Session) error {
	if err := session.WaitForURLRegex(`https:\/\/login.microsoftonline.com\/[^\/]+\/auth/finish`); err != nil {
		return fmt.Errorf("FIDO wait for completed auth: %w", err)
	}

	return nil
}

func isFidoPrompt(session *webdriver.Session) bool {
	url, err := session.URL()
	if err != nil {
		err := fmt.Errorf("check FIDO URL: %w", err)
		panic(err)
	}

	return strings.Contains(url.String(), "bridge/fido")
}

func (c *Client) authorizeViaMicrosoft(session *webdriver.Session) error {
	fmt.Println(cli.Brown("Starting authorization via Microsoft"))

	emailInput, err := session.FindElement("input[type='email']")
	if err != nil {
		return fmt.Errorf("microsoft auth: %w", err)
	}

	requiresInput := !(c.configProvider.Has("microsoft.email") &&
		c.configProvider.Has("microsoft.password"))

	if requiresInput {
		if err := session.Minimize(); err != nil {
			return fmt.Errorf("microsoft auth: %w", err)
		}
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
	if requiresInput {
		if err := session.Maximize(); err != nil {
			return fmt.Errorf("microsoft auth: %w", err)
		}
	}

	// Enter the user's E-mail.
	if err := session.WriteToElement(emailInput, email); err != nil {
		return fmt.Errorf("write E-mail for microsoft auth: %w", err)
	}
	if err := session.Click("input.button_primary"); err != nil {
		return fmt.Errorf("submit E-mail for microsoft auth: %w", err)
	}
	time.Sleep(2 * time.Second)

	// The password prompt is optional as sometimes the FIDO flow
	// already starts here depending on the Entra configuration.
	if isFidoPrompt(session) {
		if err := fidoQuickFlow(session); err != nil {
			return err
		}
	} else {
		// Enter the user's password.
		if err := session.WriteTo("input[name='passwd']", password); err != nil {
			return fmt.Errorf("write password for microsoft auth: %w", err)
		}
		if err := session.Click("input.button_primary"); err != nil {
			return fmt.Errorf("submit password for microsoft auth: %w", err)
		}
		time.Sleep(2 * time.Second)

		// Dispatch between flows.
		if isOtpPrompt(session) {
			if err := otpFlow(session); err != nil {
				return err
			}
		} else if isFidoPrompt(session) {
			if err := fidoFlow(session); err != nil {
				return err
			}
		} else {
			panic("unsupported login flow")
		}
	}

	// Click the stay signed in prompt if it exists.
	if err := session.Click(".sign-in-box input.button_primary"); err == nil {
		time.Sleep(1 * time.Second)
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

type Attendance struct {
	Start   time.Time
	End     time.Time
	Comment string
}

const (
	trackAttendanceQuery       query         = "52d040f7dfa51ff8a9a3cd149cd2da58c5d15ed404f2be69e1d081bcab879c83"
	trackAttendanceOperation   operationName = "TM_TrackAttendancePeriodsOnDay_v2026071301"
	originWeb                  origin        = "ATTENDANCE_ORIGIN_WEB"
	deleteAttendancesQuery     query         = "ba5746220742d912960d7f0eb863eac69bbe9039f638f64b68a3cf348d635078"
	deleteAttendancesOperation operationName = "TM_DeleteAttendancePeriodsOnDays_v2026071501"
	periodTypeWork                           = "ATTENDANCE_PERIOD_TYPE_WORK"
)

func toGraphqlDate(t time.Time) graphqlDate {
	return graphqlDate{Year: uint(t.Year()), Month: uint(t.Month()), Day: uint(t.Day())}
}

func toGraphqlDateTime(t time.Time) graphqlDateTime {
	return graphqlDateTime{
		Date: toGraphqlDate(t),
		Time: graphqlTime{Hours: uint(t.Hour()), Minutes: uint(t.Minute())},
	}
}

func attendanceToGraphqlPeriod(personID string, a Attendance) (graphqlAttendancePeriod, error) {
	// TODO: Validate start/end to be equal to day.
	periodID, err := uuid.NewRandom()
	if err != nil {
		return graphqlAttendancePeriod{}, fmt.Errorf("create attendance periode ID: %w", err)
	}

	return graphqlAttendancePeriod{
		ID:             graphqlID{ID: periodID.String()},
		Type:           periodTypeWork,
		StartLocalTime: toGraphqlDateTime(a.Start),
		EndLocalTime:   toGraphqlDateTime(a.End),
		PersonID:       graphqlID{ID: personID},
		Origin:         originWeb,
	}, nil
}

func postGraphql[I any](c *Client, operation operationName, q query, input I) error {
	request := graphqlRequest[I]{
		Query:         q,
		OperationName: operation,
		Variables:     graphqlVariables[I]{Input: input},
	}

	var response map[string]any
	if err := rest.POST(c.host.JoinPath("graphql")).
		Cookies(c.cookies).
		RequestBody(request).
		ResponseAs(&response).
		Run(); err != nil {
		return err
	}

	if errs, ok := response["errors"]; ok {
		return fmt.Errorf("graphql errors: %v", errs)
	}

	return nil
}

type query string
type operationName string
type origin string

type graphqlID struct {
	ID string `json:"id"`
}

type graphqlDate struct {
	Year  uint `json:"year"`
	Month uint `json:"month"`
	Day   uint `json:"day"`
}

type graphqlTime struct {
	Hours   uint `json:"hours"`
	Minutes uint `json:"minutes"`
}

type graphqlDateTime struct {
	Date graphqlDate `json:"date"`
	Time graphqlTime `json:"time"`
}

type graphqlAttendancePeriod struct {
	ID             graphqlID       `json:"id"`
	Type           string          `json:"type"`
	StartLocalTime graphqlDateTime `json:"startLocalTime"`
	EndLocalTime   graphqlDateTime `json:"endLocalTime"`
	PersonID       graphqlID       `json:"personId"`
	Origin         origin          `json:"origin"`
}

type graphqlCreateInput struct {
	Origin                 origin                    `json:"origin"`
	AssumedAttributionDate graphqlDate               `json:"assumedAttributionDate"`
	PersonID               graphqlID                 `json:"personId"`
	AttendancePeriods      []graphqlAttendancePeriod `json:"attendancePeriods"`
}

type graphqlVariables[I any] struct {
	Input I `json:"input"`
}

type graphqlRequest[I any] struct {
	Query         query               `json:"query"`
	OperationName operationName       `json:"operationName"`
	Variables     graphqlVariables[I] `json:"variables"`
}

type graphqlDeleteInput struct {
	PersonID graphqlID     `json:"personId"`
	Dates    []graphqlDate `json:"dates"`
	Origin   origin        `json:"origin"`
}

func (c *Client) CreateAttendances(employeeID EmployeeID, day time.Time,
	attendances []Attendance) error {
	personID := strconv.Itoa(int(employeeID))

	periods := make([]graphqlAttendancePeriod, len(attendances))
	for i, attendance := range attendances {
		period, err := attendanceToGraphqlPeriod(personID, attendance)
		if err != nil {
			return err
		}
		periods[i] = period
	}

	input := graphqlCreateInput{
		Origin:                 originWeb,
		AssumedAttributionDate: toGraphqlDate(day),
		PersonID:               graphqlID{ID: personID},
		AttendancePeriods:      periods,
	}

	if err := postGraphql(c, trackAttendanceOperation, trackAttendanceQuery, input); err != nil {
		return fmt.Errorf("create attendance for employee %d: %w", employeeID, err)
	}

	return nil
}

func (c *Client) RemoveAttendances(employeeID EmployeeID, day time.Time) error {
	input := graphqlDeleteInput{
		PersonID: graphqlID{ID: strconv.Itoa(int(employeeID))},
		Dates:    []graphqlDate{toGraphqlDate(day)},
		Origin:   originWeb,
	}

	if err := postGraphql(c, deleteAttendancesOperation, deleteAttendancesQuery, input); err != nil {
		return fmt.Errorf("delete attendances for employee %d: %w", employeeID, err)
	}

	return nil
}
