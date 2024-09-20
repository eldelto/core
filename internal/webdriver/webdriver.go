package webdriver

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	safariDriver = "safaridriver"
	chromeDriver = "chromedriver"
)

type driver struct {
	name string
	port int
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
		wg:   wg,
	}
	d.acquire()

	return &d, nil
}

func (d *driver) acquire() {
	d.wg.Add(1)
}

/*func (d *driver) release() {
	d.wg.Done()
}

func (d *driver) createSession() (string, error) {
	// TODO
	return "", nil
}
*/

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

	return &Session{
		id:     "",
		driver: driver,
	}, nil
}
