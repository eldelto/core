package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func URLParamAsUUID(r *http.Request, key string) (uuid.UUID, error) {
	raw := chi.URLParam(r, key)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("parsing URL param as UUID: key=%q, value=%q, err=%w",
			key, raw, err)
	}
	return id, nil
}
