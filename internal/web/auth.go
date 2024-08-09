package web

import (
	"context"
	"crypto/rand"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/mail"

	"github.com/google/uuid"
)

type ctxKey string

const (
	loginPath  = "/auth/login"
	userIDKey  = ctxKey("userID")
	cookieName = "session"
)

type SessionID struct{ string }
type TokenID struct{ string }
type UserID struct{ uuid.UUID }

type Token struct {
	ID    TokenID
	Email mail.Address
}

type Session struct {
	ID   SessionID
	User UserID
}

/*
   Auth flow:

   - User enters E-mail
   - One-time token is generated and sent to the E-mail address
   - Mapping from token to E-mail is stored
   - User clicks the received login link
   - Resolve E-mail via token
   - Resolve user ID via E-mail
   - Store session cookie
*/

type AuthRepository interface {
	StoreToken(Token) error
	FindToken(TokenID) (Token, error)
	ResolveUserID(mail.Address) (UserID, error)
	StoreSession(Session) error
	FindSession(SessionID) (Session, error)
}

type Authenticator struct {
	repo                 AuthRepository
	loginTemplate        *template.Template
	tokenCreatedTemplate *template.Template
}

func NewAuthenticator(repo AuthRepository, templateFS, assetsFS fs.FS) *Authenticator {
	templater := NewTemplater(templateFS, assetsFS)
	loginTemplate := templater.GetP("login.html")
	tokenCreatedtemplate := templater.GetP("token-created.html")

	return &Authenticator{
		repo:                 repo,
		loginTemplate:        loginTemplate,
		tokenCreatedTemplate: tokenCreatedtemplate,
	}
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			if err != http.ErrNoCookie {
				log.Printf("failed to fetch session cookie: %v", err)
			}

			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}

		session, err := a.repo.FindSession(SessionID{cookie.Value})
		if err != nil {
			log.Printf("failed to fetch session: %v", err)
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, session.User)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (a *Authenticator) Controller() *Controller {
	return &Controller{
		BasePath: "/auth",
		Handlers: map[Endpoint]Handler{
			{Method: http.MethodGet, Path: "login"}:      a.getLoginPage(),
			{Method: http.MethodPost, Path: "token"}:     a.createToken(),
			{Method: http.MethodGet, Path: "session"}:    a.authenticate(),
			{Method: http.MethodDelete, Path: "session"}: logout(),
		},
	}
}

func (a *Authenticator) getLoginPage() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return a.loginTemplate.Execute(w, nil)
	}
}

func (a *Authenticator) createToken() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawToken, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			return fmt.Errorf("failed to generate random token: %w", err)
		}

		if err := r.ParseForm(); err != nil {
			return err
		}

		rawEmail := r.PostForm.Get("email")
		email, err := mail.ParseAddress(rawEmail)
		if err != nil {
			return fmt.Errorf("failed to parse %q as valid E-mail address: %w",
				rawEmail, err)
		}

		id := TokenID{rawToken.String()}
		token := Token{
			ID:    id,
			Email: *email,
		}

		if err := a.repo.StoreToken(token); err != nil {
			return fmt.Errorf("failed to store new token: %w", err)
		}

		return a.tokenCreatedTemplate.Execute(w, rawEmail)
	}
}

func (a *Authenticator) authenticate() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		rawTokenID := r.URL.Query().Get("token")
		if rawTokenID == "" {
			return ErrUnauthenticated
		}

		token, err := a.repo.FindToken(TokenID{rawTokenID})
		if err != nil {
			return fmt.Errorf("failed to find token %q: %w %w",
				rawTokenID, err, ErrUnauthenticated)
		}

		userID, err := a.repo.ResolveUserID(token.Email)
		if err != nil {
			return fmt.Errorf("failed to find user E-mail: %w %w",
				err, ErrUnauthenticated)
		}

		session := Session{
			User: userID,
		}

		if err := a.repo.StoreSession(session); err != nil {
			return fmt.Errorf("failed to store user session %q: %w %w",
				userID.String(), err, ErrUnauthenticated)
		}

		cookie := http.Cookie{
			Name:  cookieName,
			Value: session.ID.string,
			Path:  "/",
			// TODO: Switch off when domain == localhost?
			//Secure: true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, "/lists", http.StatusSeeOther)
		return nil
	}
}

func logout() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// TODO: Implement
		return nil
	}
}

type InMemoryAuthRepository struct {
	tokenMap   map[TokenID]Token
	emailMap   map[mail.Address]UserID
	sessionMap map[SessionID]Session
}

func NewInMemoryAuthRepository() *InMemoryAuthRepository {
	return &InMemoryAuthRepository{
		tokenMap:   map[TokenID]Token{},
		emailMap:   map[mail.Address]UserID{},
		sessionMap: map[SessionID]Session{},
	}
}

func (r *InMemoryAuthRepository) StoreToken(t Token) error {
	r.tokenMap[t.ID] = t
	fmt.Printf("stored token: %v\n", t)
	fmt.Printf("go to: /auth/session?token=%s\n", t.ID.string)
	return nil
}

func (r *InMemoryAuthRepository) FindToken(id TokenID) (Token, error) {
	token, ok := r.tokenMap[id]
	if !ok {
		return Token{}, fmt.Errorf("failed to find token %q", id)
	}
	delete(r.tokenMap, id)

	return token, nil
}

func (r *InMemoryAuthRepository) ResolveUserID(email mail.Address) (UserID, error) {
	userID, ok := r.emailMap[email]
	if !ok {
		rawUserID, err := uuid.NewRandom()
		if err != nil {
			return UserID{}, fmt.Errorf("failed to generate new user ID: %w", err)
		}
		userID = UserID{rawUserID}
		r.emailMap[email] = userID
	}

	return userID, nil
}

func (r *InMemoryAuthRepository) StoreSession(s Session) error {
	r.sessionMap[s.ID] = s
	return nil
}

func (r *InMemoryAuthRepository) FindSession(id SessionID) (Session, error) {
	s, ok := r.sessionMap[id]
	if !ok {
		return Session{}, fmt.Errorf("failed to find session %q", id)
	}

	return s, nil
}
