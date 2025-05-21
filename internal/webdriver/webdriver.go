package webdriver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eldelto/core/internal/rest"
	"github.com/google/uuid"
)

const (
	safariDriver = "safaridriver"
	chromeDriver = "chromedriver"
)

// TODO: Look into session timeouts instead.
func withRetry(f func() error) error {
	var err error
	for range 3 {
		err = f()
		if err == nil {
			return nil
		}

		time.Sleep(1 * time.Second)
	}

	return err
}

type driver struct {
	name            string
	port            int
	applicationName string
	wg              *sync.WaitGroup
}

func appNameFromDriver(name string) string {
	switch name {
	case safariDriver:
		return "Safari"
	case chromeDriver:
		return "Google Chrome"
	default:
		err := fmt.Errorf("%q is not a supported webdriver", name)
		panic(err)
	}
}

func runDriver(name string, port int) (*driver, error) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, name, "-p", strconv.Itoa(port))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start webdriver %q on port %d: %w",
			name, port, err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// Shutdown the webdriver when nobody uses it anymore.
		wg.Wait()
		if err := cmd.Cancel(); err != nil {
			err := fmt.Errorf("stop webdriver %q: %w", name, err)
			log.Println(err)
		}
	}()

	d := driver{
		name:            name,
		port:            port,
		applicationName: appNameFromDriver(name),
		wg:              wg,
	}
	d.acquire()

	return &d, nil
}

func (d *driver) host() string {
	return "http://localhost:" + strconv.Itoa(d.port)
}

func (d *driver) acquire() {
	d.wg.Add(1)
}

func (d *driver) release() {
	d.wg.Done()
}

type createSessionRequest struct {
	SessionID    string `json:"sessionId"`
	Capabilities struct {
	} `json:"capabilities"`
}

type createSessionResponse struct {
	Value struct {
		SessionID string `json:"sessionId"`
	} `json:"value"`
}

func (d *driver) createSession() (string, error) {
	url, err := url.JoinPath(d.host(), "session")
	if err != nil {
		panic(err)
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("create session ID: %w", err)
	}

	request := createSessionRequest{
		SessionID: id.String(),
	}

	var response createSessionResponse
	if err := rest.Post(url, nil, request, &response, nil); err != nil {
		return "", fmt.Errorf("create session via webdriver %q: %w", d.name, err)
	}

	return response.Value.SessionID, nil
}

var (
	drivers        = []string{safariDriver, chromeDriver}
	runningDrivers = []driver{}
)

func detectInstalledDriver() (string, error) {
	for _, driver := range drivers {
		result, err := exec.Command("whereis", driver).Output()
		if err != nil {
			continue
		}

		parts := strings.Split(strings.TrimSpace(string(result)), ":")
		if len(parts) < 2 || parts[1] == "" {
			continue
		}

		return driver, nil
	}

	return "", fmt.Errorf("no webdriver implementation found - please install the webdriver for your preferred browser and try again")
}

func getDriver() (*driver, error) {
	// create or reuse driver
	name, err := detectInstalledDriver()
	if err != nil {
		return nil, err
	}

	for _, driver := range runningDrivers {
		if driver.name == name {
			driver.acquire()
			return &driver, nil
		}
	}

	return runDriver(name, 45595)
}

type Session struct {
	id     string
	driver *driver
}

func NewSession() (*Session, error) {
	driver, err := getDriver()
	if err != nil {
		return nil, fmt.Errorf("create webdriver session: %w", err)
	}

	// create session via API
	sessionID := ""
	err = withRetry(func() error {
		sID, err := driver.createSession()
		sessionID = sID
		return err
	})
	if err != nil {
		return nil, err
	}

	return &Session{
		id:     sessionID,
		driver: driver,
	}, nil
}

func (s *Session) Close() error {
	s.driver.release()

	url, err := url.JoinPath(s.driver.host(), "session", s.id)
	if err != nil {
		panic(err)
	}

	err = withRetry(func() error {
		return rest.Delete(url, nil)
	})
	if err != nil {
		return fmt.Errorf("close session %q via webdriver %q: %w",
			s.id, s.driver.name, err)
	}

	return nil
}

func (s *Session) sessionEndpoint(parts ...string) (string, error) {
	parts = append([]string{"session", s.id}, parts...)
	url, err := url.JoinPath(s.driver.host(), parts...)
	if err != nil {
		return "", fmt.Errorf("generate session endpoint for session %q: %w", s.id, err)
	}

	return url, nil
}

type navigateToRequest struct {
	URL string `json:"url"`
}

func (s *Session) NavigateTo(pageURL *url.URL) error {
	url, err := s.sessionEndpoint("url")
	if err != nil {
		return fmt.Errorf("open page %q: %w", url, err)
	}

	request := navigateToRequest{URL: pageURL.String()}
	if err := rest.Post(url, nil, request, nil, nil); err != nil {
		return fmt.Errorf("open page %q: %w", url, err)
	}

	return nil
}

type getURLResponse struct {
	Value string `json:"value"`
}

func (s *Session) URL() (*url.URL, error) {
	endpoint, err := s.sessionEndpoint("url")
	if err != nil {
		return nil, fmt.Errorf("get URL for session %q: %w", s.id, err)
	}

	var response getURLResponse
	if err := rest.Get(endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("get URL for session %q: %w", s.id, err)
	}

	url, err := url.Parse(response.Value)
	if err != nil {
		return nil, fmt.Errorf("get URL for session %q: %w", s.id, err)
	}

	return url, nil
}

func (s *Session) WaitForHost(host string) error {
	return withRetry(func() error {
		url, err := s.URL()
		if err != nil {
			return fmt.Errorf("wait for host %q: %w", host, err)
		}

		if url.Host != host {
			return fmt.Errorf("host %q did not match expected %q", url.Host, host)
		}
		return nil
	})
}

type cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type getCookiesResponse struct {
	Value []cookie `json:"value"`
}

func (s *Session) Cookies() ([]http.Cookie, error) {
	endpoint, err := s.sessionEndpoint("cookie")
	if err != nil {
		return nil, fmt.Errorf("get cookies for session %q: %w", s.id, err)
	}

	var response getCookiesResponse
	if err := rest.Get(endpoint, nil, &response); err != nil {
		return nil, fmt.Errorf("get cookies for session %q: %w", s.id, err)
	}

	cookies := make([]http.Cookie, len(response.Value))
	for i, c := range response.Value {
		cookies[i] = http.Cookie{
			Name:  c.Name,
			Value: c.Value,
		}
	}

	return cookies, nil
}

func focusApplication(name string) error {
	osName := runtime.GOOS
	switch osName {
	case "darwin":
		err := exec.Command("osascript", "-e",
			fmt.Sprintf("tell application \"%s\" to activate", name)).Run()
		if err != nil {
			return fmt.Errorf("focus application for OS %q: %w", osName, err)
		}
	default:
		err := fmt.Errorf("focus application is not supported for OS %q", osName)
		panic(err)
	}

	return nil
}

func (s *Session) Maximize() error {
	endpoint, err := s.sessionEndpoint("window", "maximize")
	if err != nil {
		return fmt.Errorf("maximize session %q: %w", s.id, err)
	}

	if err := focusApplication(s.driver.applicationName); err != nil {
		return fmt.Errorf("maximize session %q: %w", s.id, err)
	}

	if err := rest.Post(endpoint, nil, struct{}{}, nil, nil); err != nil {
		return fmt.Errorf("maximize session %q: %w", s.id, err)
	}

	return nil
}

func (s *Session) Minimize() error {
	endpoint, err := s.sessionEndpoint("window", "minimize")
	if err != nil {
		return fmt.Errorf("minimize session %q: %w", s.id, err)
	}

	if err := rest.Post(endpoint, nil, struct{}{}, nil, nil); err != nil {
		return fmt.Errorf("minimize session %q: %w", s.id, err)
	}

	return nil
}

type Element struct {
	id string
}

type findElementRequest struct {
	Using string `json:"using"`
	Value string `json:"value"`
}

type findElementResponse struct {
	Value map[string]string
}

func (s *Session) findElement(cssSelector string) (Element, error) {
	url, err := s.sessionEndpoint("element")
	if err != nil {
		return Element{}, fmt.Errorf("find element %q with selector %q: %w",
			url, cssSelector, err)
	}

	request := findElementRequest{
		Using: "css selector",
		Value: cssSelector,
	}

	var response findElementResponse

	err = withRetry(func() error {
		if err := rest.Post(url, nil, request, &response, nil); err != nil {
			return fmt.Errorf("find element %q with selector %q: %w",
				url, cssSelector, err)
		}
		return nil
	})
	if err != nil {
		return Element{}, err
	}

	for _, v := range response.Value {
		return Element{id: v}, nil
	}

	return Element{}, fmt.Errorf("no element matching %q found", cssSelector)
}

func (s *Session) FindElement(cssSelector string) (Element, error) {
	var element Element
	err := withRetry(func() error {
		e, err := s.findElement(cssSelector)
		if err != nil {
			return err
		}

		element = e
		return nil
	})

	return element, err
}

func (s *Session) HasElement(cssSelector string) bool {
	_, err := s.findElement(cssSelector)
	return err == nil
}

func (s *Session) ClickElement(element Element) error {
	url, err := s.sessionEndpoint("element", element.id, "click")
	if err != nil {
		return fmt.Errorf("click element %q: %w", element.id, err)
	}

	if err := rest.Post(url, nil, nil, nil, nil); err != nil {
		return fmt.Errorf("click element %q: %w", element.id, err)
	}

	return nil
}

func (s *Session) Click(cssSelector string) error {
	element, err := s.FindElement(cssSelector)
	if err != nil {
		return err
	}

	return s.ClickElement(element)
}

type writeToElementRequest struct {
	Text string `json:"text"`
}

func (s *Session) WriteToElement(element Element, text string) error {
	url, err := s.sessionEndpoint("element", element.id, "value")
	if err != nil {
		return fmt.Errorf("write to element %q: %w", element.id, err)
	}

	request := writeToElementRequest{
		Text: text,
	}

	if err := rest.Post(url, nil, request, nil, nil); err != nil {
		return fmt.Errorf("write to element %q: %w", element.id, err)
	}

	return nil
}

func (s *Session) WriteTo(cssSelector, text string) error {
	element, err := s.FindElement(cssSelector)
	if err != nil {
		return err
	}

	return s.WriteToElement(element, text)
}
