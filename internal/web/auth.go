package web

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/internal/boltutil"
	"github.com/eldelto/core/internal/errs"
	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

type ctxKey string

const (
	LoginPath  = "/login.html"
	authCtxKey = ctxKey("auth")
	cookieName = "session"
)

type SessionID string
type TokenID string
type UserID struct{ uuid.UUID }

type Token struct {
	ID    TokenID
	Email mail.Address
	// Unix timestamp
	ValidUntil int64
}

func (t *Token) Expired() bool {
	return t.ValidUntil < time.Now().Unix()
}

// TODO: Shouldn't the session expire at one point?
type Session struct {
	ID   SessionID
	User UserID
	Email mail.Address
}

type Auth interface {
	UserID() UserID
}

type UserAuth struct {
	User UserID
	Email mail.Address
}

func (a *UserAuth) UserID() UserID {
	return a.User
}

func GetAuth(ctx context.Context) (Auth, error) {
	value := ctx.Value(authCtxKey)
	if value == nil {
		return nil, ErrUnauthenticated
	}

	auth, ok := value.(Auth)
	if !ok {
		return nil, fmt.Errorf("failed to cast %v to type Auth", value)
	}

	return auth, nil
}

func SetAuth(ctx context.Context, auth Auth) context.Context {
	return context.WithValue(ctx, authCtxKey, auth)
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
	domain               string
	repo                 AuthRepository
	loginTemplate        *Template
	tokenCreatedTemplate *Template
	TokenCallback        func(mail.Address, TokenID) error
	RedirectTarget       string
}

func NewAuthenticator(domain string,
	redirectTarget string,
	repo AuthRepository,
	templateFS,
	assetsFS fs.FS) *Authenticator {
	templater := NewTemplater(templateFS, assetsFS)
	loginTemplate := templater.GetP("login.html")
	tokenCreatedtemplate := templater.GetP("verify.html")

	return &Authenticator{
		domain:               domain,
		repo:                 repo,
		loginTemplate:        loginTemplate,
		tokenCreatedTemplate: tokenCreatedtemplate,
		RedirectTarget:       redirectTarget,
	}
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			// TODO: Let the controller decide how to handle that?
			/*if err != http.ErrNoCookie {
				log.Printf("failed to fetch session cookie: %v", err)
			}

			http.Redirect(w, r, LoginPath, http.StatusSeeOther)
			*/

			next.ServeHTTP(w, r)
			return
		}

		session, err := a.repo.FindSession(SessionID(cookie.Value))
		if err != nil {
			log.Printf("failed to fetch session while accessing %q: %v",
				r.URL.String(), err)
			http.Redirect(w, r, LoginPath, http.StatusSeeOther)
			return
		}

		ctx := SetAuth(r.Context(), &UserAuth{
			User: session.User,
			Email: session.Email,
		})
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (a *Authenticator) forwardingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := GetAuth(r.Context())
		if err == nil {
			http.Redirect(w, r, a.RedirectTarget, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (a *Authenticator) Controller() *Controller {
	return &Controller{
		BasePath: "/auth",
		Handlers: map[Endpoint]Handler{
			{Method: http.MethodGet, Path: "login"}:      a.login(),
			{Method: http.MethodPost, Path: "token"}:     a.createToken(),
			{Method: http.MethodGet, Path: "session"}:    a.authenticate(),
			{Method: http.MethodDelete, Path: "session"}: logout(),
		},
		Middleware: []Middleware{
			a.forwardingMiddleware,
			a.Middleware,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, outerErr error) Handler {
			return func(w http.ResponseWriter, r *http.Request) error {
				log.Println(outerErr)

				msgParam := "?msg=" + url.QueryEscape(outerErr.Error())

				if errors.Is(outerErr, ErrUnauthenticated) {
					http.Redirect(w, r, LoginPath+msgParam, http.StatusSeeOther)
					return nil
				}

				http.Redirect(w, r, "/error.html"+msgParam, http.StatusSeeOther)
				return nil
			}
		},
	}
}

func (a *Authenticator) GenerateToken(length int) (TokenID, error) {
	rawToken := make([]byte, length)
	_, err := rand.Read(rawToken)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return TokenID(base64.URLEncoding.EncodeToString(rawToken)), nil
}

func (a *Authenticator) login() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return a.loginTemplate.Execute(w, nil)
	}
}

func (a *Authenticator) createToken() Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if err := r.ParseForm(); err != nil {
			return err
		}

		rawEmail := r.PostForm.Get("email")
		email, err := mail.ParseAddress(rawEmail)
		if err != nil {
			return fmt.Errorf("failed to parse %q as valid E-mail address: %w",
				rawEmail, err)
		}

		id, err := a.GenerateToken(6)
		if err != nil {
			return err
		}

		token := Token{
			ID:         id,
			Email:      *email,
			ValidUntil: time.Now().Add(15 * time.Minute).Unix(),
		}

		if err := a.repo.StoreToken(token); err != nil {
			return fmt.Errorf("failed to store new token: %w", err)
		}

		if a.TokenCallback != nil {
			if err := a.TokenCallback(*email, id); err != nil {
				return fmt.Errorf("failed to execute token callback: %w", err)
			}
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

		token, err := a.repo.FindToken(TokenID(rawTokenID))
		if err != nil {
			return fmt.Errorf("authentication token %q not found: %w %w",
				rawTokenID, err, ErrUnauthenticated)
		}

		if token.Expired() {
			return fmt.Errorf("authentication token %q expired: %w",
				rawTokenID, ErrUnauthenticated)
		}

		userID, err := a.repo.ResolveUserID(token.Email)
		if err != nil {
			return fmt.Errorf("failed to find user E-mail: %w %w",
				err, ErrUnauthenticated)
		}

		sessionID, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			return fmt.Errorf("failed to generate random session ID: %w", err)
		}
		session := Session{
			ID:   SessionID(sessionID.String()),
			User: userID,
			Email: token.Email,
		}

		if err := a.repo.StoreSession(session); err != nil {
			return fmt.Errorf("failed to store user session %q: %w %w",
				userID.String(), err, ErrUnauthenticated)
		}

		cookie := http.Cookie{
			Name:     cookieName,
			Value:    string(session.ID),
			Path:     "/",
			Secure:   !strings.Contains(a.domain, "localhost"),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Now().Add(90 * 24 * time.Hour),
		}
		http.SetCookie(w, &cookie)

		http.Redirect(w, r, a.RedirectTarget, http.StatusSeeOther)
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

const (
	tokenBucket        = "auth.tokens"
	emailMappingBucket = "auth.emailMapping"
	sessionBucket      = "auth.sessions"
)

type BBoltAuthRepository struct {
	db *bbolt.DB
}

// TODO: Rather return the errors here?
func NewBBoltAuthRepository(db *bbolt.DB) *BBoltAuthRepository {
	if err := boltutil.EnsureBucketExists(db, tokenBucket); err != nil {
		panic(err)
	}
	if err := boltutil.EnsureBucketExists(db, emailMappingBucket); err != nil {
		panic(err)
	}
	if err := boltutil.EnsureBucketExists(db, sessionBucket); err != nil {
		panic(err)
	}

	return &BBoltAuthRepository{db: db}
}

func (r *BBoltAuthRepository) StoreToken(t Token) error {
	return boltutil.Store(r.db, tokenBucket, string(t.ID), t)
}

func (r *BBoltAuthRepository) FindToken(id TokenID) (Token, error) {
	token, err := boltutil.Find[Token](r.db, tokenBucket, string(id))
	if err != nil {
		return Token{}, fmt.Errorf("failed to find token %q: %w", id, err)
	}

	if err := boltutil.Remove(r.db, tokenBucket, string(id)); err != nil {
		return Token{}, fmt.Errorf("failed to remove token %q: %w", id, err)
	}

	return token, nil
}

func (r *BBoltAuthRepository) ResolveUserID(email mail.Address) (UserID, error) {
	userID, err := boltutil.Find[UserID](r.db, emailMappingBucket, email.String())
	switch {
	case errors.Is(err, &errs.ErrNotFound{}):
		rawUserID, err := uuid.NewRandom()
		if err != nil {
			return UserID{}, fmt.Errorf("failed to generate new user ID: %w", err)
		}
		userID = UserID{rawUserID}

		if err := boltutil.Store(r.db, emailMappingBucket, email.String(), userID); err != nil {
			return UserID{}, fmt.Errorf("failed to store new user ID: %w", err)
		}
	case err != nil:
		return UserID{}, fmt.Errorf("failed to map E-mail to user ID: %w", err)
	}

	return userID, nil
}

func (r *BBoltAuthRepository) StoreSession(s Session) error {
	if err := boltutil.Store(r.db, sessionBucket, string(s.ID), s); err != nil {
		return fmt.Errorf("failed to store session %q: %w",
			s.ID, err)
	}

	return nil
}

func (r *BBoltAuthRepository) FindSession(id SessionID) (Session, error) {
	session, err := boltutil.Find[Session](r.db, sessionBucket, string(id))
	if err != nil {
		return Session{}, fmt.Errorf("failed to find session %q: %w", id, err)
	}

	return session, nil
}
