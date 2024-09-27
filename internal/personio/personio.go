package personio

import (
	"fmt"
	"net/url"
	"time"

	"github.com/eldelto/core/internal/cli"
	"github.com/eldelto/core/internal/webdriver"
)

func authorizeViaMicrosoft(session *webdriver.Session) error {
	fmt.Println(cli.Brown("Starting authorization via Microsoft"))

	emailInput, err := session.FindElement("input[type='email']")
	if err != nil {
		return fmt.Errorf("microsoft auth: %w", err)
	}

	if err := session.Minimize(); err != nil {
		return fmt.Errorf("microsoft auth: %w", err)
	}

	// TODO: term.ReadPassword(int(syscall.Stdin))
	email, err := cli.ReadInput("\nPlease enter your E-mail address:\n")
	if err != nil {
		return fmt.Errorf("e-mail for microsoft auth: %w", err)
	}
	password, err := cli.ReadInput("\nPlease enter your password:\n")
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

type Client struct {
	Host *url.URL
}

func (c *Client) Login() error {
	session, err := webdriver.NewSession()
	if err != nil {
		return fmt.Errorf("personio login: %w", err)
	}
	defer session.Close()

	url := c.Host.JoinPath("login", "index")
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
		if err := authorizeViaMicrosoft(session); err != nil {
			return err
		}
	default:
		return fmt.Errorf("authentication via %q is not supported", host)
	}

	if err := session.WaitForHost(c.Host.Host); err != nil {
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

	fmt.Println(cookies)

	return nil
}
