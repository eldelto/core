package web

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
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

type InMemoryAuthRepository struct {
	data map[SessionID]UserID
}

func NewInMemoryAuthRepository() *InMemoryAuthRepository {
	return &InMemoryAuthRepository{
		data: map[SessionID]UserID{},
	}
}

func (r *InMemoryAuthRepository) Store(sid SessionID, uid UserID) error {
	r.data[sid] = uid
	return nil
}

func (r *InMemoryAuthRepository) Find(sid SessionID) (UserID, error) {
	uid, ok := r.data[sid]
	if !ok {
		return UserID{}, fmt.Errorf("failed to find session %q", sid)
	}

	return uid, nil
}

type Authenticator struct {
	repo          AuthRepository
	loginTemplate *template.Template
}

func NewAuthenticator(repo AuthRepository, templateFS, assetsFS fs.FS) *Authenticator {
	templater := NewTemplater(templateFS, assetsFS)
	loginTemplate := templater.GetP("login.html")

	return &Authenticator{
		repo:          repo,
		loginTemplate: loginTemplate,
	}
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

func (a *Authenticator) Controller() *Controller {
	return &Controller{
		BasePath: "/login",
		Handlers: map[Endpoint]Handler{
			{Method: http.MethodGet, Path: ""}:    a.getLoginPage(),
			{Method: http.MethodPost, Path: ""}:   login(),
			{Method: http.MethodDelete, Path: ""}: logout(),
		},
	}
}

func (a *Authenticator) getLoginPage() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Implement
		return a.loginTemplate.Execute(w, nil)
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
