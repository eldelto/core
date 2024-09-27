package webdriver

import (
	"net/url"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestDriverDetect(t *testing.T) {
	SkipIntegrationTest(t)

	driver, err := detectInstalledDriver()
	AssertNoError(t, err, "detectDriver")
	AssertContains(t, driver, drivers, "Driver should be a supported one")
}

func TestSessionCreation(t *testing.T) {
	SkipIntegrationTest(t)

	session, err := NewSession()
	AssertNoError(t, err, "NewSession")
	AssertNotNil(t, session, "session")

	err = session.Close()
	AssertNoError(t, err, "session.Close")
}

func TestMultiSessionCreation(t *testing.T) {
	SkipIntegrationTest(t)

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

func TestNavigation(t *testing.T) {
	SkipIntegrationTest(t)

	session, err := NewSession()
	AssertNoError(t, err, "NewSession")
	AssertNotNil(t, session, "session")
	defer session.Close()

	url, err := url.Parse("https://www.eldelto.net/")
	AssertNoError(t, err, "url.Parse")

	err = session.NavigateTo(url)
	AssertNoError(t, err, "session.NavigateTo")

	currentURL, err := session.URL()
	AssertNoError(t, err, "session.URL")
	AssertEquals(t, url, currentURL, "current URL")
}

func TestElementInteractions(t *testing.T) {
	SkipIntegrationTest(t)

	session, err := NewSession()
	AssertNoError(t, err, "NewSession")
	AssertNotNil(t, session, "session")
	defer session.Close()

	url, err := url.Parse("https://www.eldelto.net/")
	AssertNoError(t, err, "url.Parse")

	err = session.NavigateTo(url)
	AssertNoError(t, err, "session.NavigateTo")

	element, err := session.FindElement("a[href='/articles']")
	AssertNoError(t, err, "session.FindElement")
	AssertNotEquals(t, "", element.id, "element.id")

	err = session.ClickElement(element)
	AssertNoError(t, err, "session.ClickElement")
}
