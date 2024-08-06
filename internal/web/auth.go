package web

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type ctxKey string

const userIDKey = ctxKey("userID")

type SessionID string
type UserID uuid.UUID

type AuthRepository interface {
	Store(SessionID, UserID) error
	Find(SessionID) (UserID, error)
}

type Authenticator struct {
	repo AuthRepository
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			if err != http.ErrNoCookie {
				log.Printf("failed to fetch session cookie: %v", err)
			}

			http.Redirect(w, r, "/login", http.StatusFound)
			next.ServeHTTP(w, r)
			return
		}

		userID, err := a.repo.Find(SessionID(cookie.Value))
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func NewLoginController() *Controller {
	return &Controller{
		BasePath: "login",
		Handlers: map[Endpoint]Handler{
			{Method: http.MethodGet, Path: ""}:    getLoginPage(),
			{Method: http.MethodPost, Path: ""}:   login(),
			{Method: http.MethodDelete, Path: ""}: logout(),
		},
	}
}

func getLoginPage() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Implement
		return nil
	}
}

func login() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Implement
		return nil
	}
}

func logout() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Implement
		return nil
	}
}
