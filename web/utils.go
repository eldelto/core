package web

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

func SetCookie(w http.ResponseWriter, host *url.URL, key, value string, validUntil time.Time) {
	cookie := http.Cookie{
		Name:     key,
		Value:    value,
		Path:     "/",
		Secure:   !strings.Contains(host.Hostname(), "localhost"),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  validUntil,
	}
	http.SetCookie(w, &cookie)
}

func RemoveCookie(w http.ResponseWriter, name string) {
	cookie := http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)
}
