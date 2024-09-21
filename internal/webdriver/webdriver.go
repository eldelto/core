package webdriver

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/eldelto/core/internal/rest"
	"github.com/google/uuid"
)

const (
	safariDriver = "safaridriver"
	chromeDriver = "chromedriver"
)

type driver struct {
	name string
	port int
	maxSessions uint
	wg   *sync.WaitGroup
}

func runDriver(name string, port int) (*driver, error) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, name, "-p", strconv.Itoa(port))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start webdriver %q on port %d: %w",
			name, port, err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// Shutdown the webdriver when nobody uses it anymore.
		wg.Wait()
		if err := cmd.Cancel(); err != nil {
			err := fmt.Errorf("failed to stop webdriver %q: %w", name, err)
			log.Println(err)
		}
	}()

	d := driver{
		name: name,
		port: port,
		maxSessions: 1,
		wg:   wg,
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
		return "", fmt.Errorf("failed to create session ID: %w", err)
	}

	request := createSessionRequest{
		SessionID: id.String(),
	}

	var response createSessionResponse
	if err := rest.Post(url, nil, request, &response, nil); err != nil {
		return "", fmt.Errorf("failed to create new session via webdriver %q: %w", d.name, err)
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
		return nil, fmt.Errorf("failed to create new webdriver session: %w", err)
	}

	// create session via API
	sessionId, err := driver.createSession()
	if err != nil {
		return nil, err
	}

	return &Session{
		id:     sessionId,
		driver: driver,
	}, nil
}

func (s *Session) Close() error {
	s.driver.release()

	url, err := url.JoinPath(s.driver.host(), "session", s.id)
	if err != nil {
		panic(err)
	}

	if err := rest.Delete(url, nil); err != nil {
		return fmt.Errorf("failed to close session %q via webdriver %q: %w",
			s.id, s.driver.name, err)
	}

	return nil
}
