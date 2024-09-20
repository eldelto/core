package webdriver

import (
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestDriverDetect(t *testing.T) {
	driver, err := detectInstalledDriver()
	AssertNoError(t, err, "detectDriver")
	AssertContains(t, driver, drivers, "Driver should be a supported one")
}
