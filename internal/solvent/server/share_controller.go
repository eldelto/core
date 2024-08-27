package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/eldelto/core/internal/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func NewShareController() *web.Controller {
	return &web.Controller{
		BasePath: "/shared",
		Handlers: map[web.Endpoint]web.Handler{
			{Method: http.MethodGet, Path: "/user/{userID}/list/{listID}"}: receiveShare(),
		},
		Middleware: []web.HandlerProvider{
			web.ContentTypeMiddleware(web.ContentTypeHTML),
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, outerErr error) web.Handler {

			return func(w http.ResponseWriter, r *http.Request) error {
				log.Println(outerErr)
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set(web.ContentTypeHeader, web.ContentTypeHTML)
				_, err := io.WriteString(w, outerErr.Error())
				return err
			}
		},
	}
}

func receiveShare() web.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawID := chi.URLParam(r, "userID")
		userID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		rawID = chi.URLParam(r, "listID")
		listID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("failed to parse %q as UUID: %w", rawID, err)
		}

		token := r.URL.Query().Get("t")
		if token == "" {
			return fmt.Errorf("failed to get token for list share %q", listID)
		}

		cookie := http.Cookie{
			Name:  "share-" + listID.String(),
			Value: userID.String() + ":" + token,
			Path:  "/",
			//Secure:   !strings.Contains(a.domain, "localhost"),
			Secure:   false,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(14 * 24 * time.Hour),
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/lists/"+listID.String(), http.StatusSeeOther)
		return nil
	}
}
