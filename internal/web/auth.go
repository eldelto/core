package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math"
	"math/big"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.etcd.io/bbolt"
)

type ctxKey string

const (
	loginPath  = "/auth/login"
	userIDKey  = ctxKey("userID")
	cookieName = "session"
)

type SessionID string
type TokenID string
type UserID struct{ uuid.UUID }

type Token struct {
	ID    TokenID
	Email mail.Address
}

type Session struct {
	ID   SessionID
	User UserID
}

func GetUserID(r *http.Request) (UserID, error) {
	value := r.Context().Value(userIDKey)
	if value == nil {
		return UserID{}, ErrUnauthenticated
	}

	userID, ok := value.(UserID)
	if !ok {
		return UserID{}, fmt.Errorf("failed to cast %v to type UserID", value)
	}

	return userID, nil
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
	loginTemplate        *template.Template
	tokenCreatedTemplate *template.Template
	TokenCallback        func(mail.Address, TokenID) error
}

func NewAuthenticator(domain string,
	repo AuthRepository,
	templateFS,
	assetsFS fs.FS) *Authenticator {
	templater := NewTemplater(templateFS, assetsFS)
	loginTemplate := templater.GetP("login.html")
	tokenCreatedtemplate := templater.GetP("token-created.html")

	return &Authenticator{
		domain:               domain,
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

		session, err := a.repo.FindSession(SessionID(cookie.Value))
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

		id := TokenID(rawToken.String())
		token := Token{
			ID:    id,
			Email: *email,
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
			return fmt.Errorf("failed to find token %q: %w %w",
				rawTokenID, err, ErrUnauthenticated)
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
			Expires:  time.Now().Add(14 * 24 * time.Hour),
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

var ErrNotFound = errors.New("element not found")

func find[T any](db *bbolt.DB, bucketName, key string) (T, error) {
	var result T

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket - bucket=%q", bucketName)
		}

		value := bucket.Get([]byte(key))
		if value == nil {
			return fmt.Errorf("failed to get value - bucket=%q, key=%q: %w",
				bucketName, key, ErrNotFound)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(value)).Decode(&result); err != nil {
			return fmt.Errorf("failed to decode value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return result, fmt.Errorf("failed to find value - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	return result, nil
}

func store[T any](db *bbolt.DB, bucketName, key string, value T) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket - bucket=%q", bucketName)
		}

		buffer := bytes.Buffer{}
		if err := gob.NewEncoder(&buffer).Encode(value); err != nil {
			return fmt.Errorf("failed to encode value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		if err := bucket.Put([]byte(key), buffer.Bytes()); err != nil {
			return fmt.Errorf("failed to persist value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to store - bucket=%q, key=%q: %w", bucketName, key, err)
	}

	return nil
}

func remove(db *bbolt.DB, bucketName, key string) error {
	err := db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketName))
		if bucket == nil {
			return fmt.Errorf("failed to get bucket - bucket=%q", bucketName)
		}

		if err := bucket.Delete([]byte(key)); err != nil {
			return fmt.Errorf("failed to delete value - bucket=%q, key=%q: %w",
				bucketName, key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to remove - bucket=%q, key=%q: %w",
			bucketName, key, err)
	}

	return nil
}

type BBoltAuthRepository struct {
	db *bbolt.DB
}

// TODO: Rather return the errors here?
func NewBBoltAuthRepository(db *bbolt.DB) *BBoltAuthRepository {
	err := db.Update(func(tx *bbolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(tokenBucket)); err != nil {
			panic(err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(emailMappingBucket)); err != nil {
			panic(err)
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(sessionBucket)); err != nil {
			panic(err)
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	return &BBoltAuthRepository{db: db}
}

func (r *BBoltAuthRepository) StoreToken(t Token) error {
	return store(r.db, tokenBucket, string(t.ID), t)
}

func (r *BBoltAuthRepository) FindToken(id TokenID) (Token, error) {
	token, err := find[Token](r.db, tokenBucket, string(id))
	if err != nil {
		return Token{}, fmt.Errorf("failed to find token %q: %w", id, err)
	}

	if err := remove(r.db, tokenBucket, string(id)); err != nil {
		return Token{}, fmt.Errorf("failed to remove token %q: %w", id, err)
	}

	return token, nil
}

func (r *BBoltAuthRepository) ResolveUserID(email mail.Address) (UserID, error) {
	userID, err := find[UserID](r.db, emailMappingBucket, email.String())
	switch {
	case errors.Is(err, ErrNotFound):
		rawUserID, err := uuid.NewRandom()
		if err != nil {
			return UserID{}, fmt.Errorf("failed to generate new user ID: %w", err)
		}
		userID = UserID{rawUserID}

		if err := store(r.db, emailMappingBucket, email.String(), userID); err != nil {
			return UserID{}, fmt.Errorf("failed to store new user ID: %w", err)
		}
	case err != nil:
		return UserID{}, fmt.Errorf("failed to map E-mail to user ID: %w", err)
	}

	return userID, nil
}

func (r *BBoltAuthRepository) StoreSession(s Session) error {
	if err := store(r.db, sessionBucket, string(s.ID), s); err != nil {
		return fmt.Errorf("failed to store session %q: %w",
			s.ID, err)
	}

	return nil
}

func (r *BBoltAuthRepository) FindSession(id SessionID) (Session, error) {
	session, err := find[Session](r.db, sessionBucket, string(id))
	if err != nil {
		return Session{}, fmt.Errorf("failed to find session %q: %w", id, err)
	}

	return session, nil
}
