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

func TestSessionCreation(t *testing.T) {
	session, err := NewSession()
	AssertNoError(t, err, "NewSession")
	AssertNotNil(t, session, "session")

	err = session.Close()
	AssertNoError(t, err, "session.Close")
}

func TestMultiSessionCreation(t *testing.T) {
	// TODO: Fix
	t.Skip()
	s1, err := NewSession()
	AssertNoError(t, err, "NewSession")
	AssertNotNil(t, s1, "session")

	s2, err := NewSession()
	AssertNoError(t, err, "NewSession 2")
	AssertNotNil(t, s2, "session 2")

	err = s1.Close()
	AssertNoError(t, err, "s1.Close")

	err = s2.Close()
	AssertNoError(t, err, "s2.Close")
}
