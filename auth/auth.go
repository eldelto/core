package auth

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/validate"
	"github.com/eldelto/core/web"
	"github.com/google/uuid"
)

const (
	loginTokenValidity = 24 * 30 * time.Hour // 30 days
	sessionCookieName  = "thy_session"
)

var (
	serviceUserID = storage.UserID(uuid.Nil)
	serviceUser   = User{ID: serviceUserID}

	year = time.Hour * 24 * 365
)

type Config struct {
	Host          *url.URL
	DefaultSender *mail.Address
}

type Service struct {
	config            Config
	store             *storage.Storage
	mailer            web.Mailer
	magicLinkTemplate *template.Template
}

func NewService(config Config, store *storage.Storage,
	mailer web.Mailer, magicLinkTemplate *template.Template) *Service {
	store.RegisterBucket(storage.Bucket{
		Name:          userBucket,
		Retention:     1 * year,
		KeepRevisions: 1,
		PreCommitFuncs: []storage.PreCommitFunc{
			validate.BeforeCommit(&User{}),
		},
	})
	store.RegisterBucket(storage.Bucket{
		Name:          capabilityBucket,
		Retention:     1 * year,
		KeepRevisions: 0,
		PreCommitFuncs: []storage.PreCommitFunc{
			validate.BeforeCommit(&Capability{}),
		},
	})
	store.RegisterBucket(storage.Bucket{
		Name:          shortCodeBucket,
		Retention:     10 * time.Minute,
		KeepRevisions: 0,
		PreCommitFuncs: []storage.PreCommitFunc{
			validate.BeforeCommit(shortCodeCapability{}),
		},
	})

	return &Service{
		config:            config,
		store:             store,
		mailer:            mailer,
		magicLinkTemplate: magicLinkTemplate,
	}
}

type magicLinkData struct {
	URL  *url.URL
	Code string
}

func (s *Service) sendMagicLink(ctx context.Context, tx storage.WriteTx, email mail.Address, redirectPath string) error {
	scopes := Scopes{}
	scopes.Add(PermissionWrite, "user", email.Address, "login")
	scopes.Add(PermissionWrite, "user", email.Address, "signup")
	scopes.Add(PermissionRead, "user", email.Address)

	id, code, _, err := IssueShortLivedCapability(tx, serviceUser, nil, scopes)
	if err != nil {
		return fmt.Errorf("magic link: %w", err)
	}

	loginURL, err := url.Parse(s.config.Host.String())
	if err != nil {
		return fmt.Errorf("login URL: %w", err)
	}

	redirectPath = url.PathEscape(redirectPath)

	loginURL = loginURL.JoinPath("auth", email.Address, "verify")
	query := loginURL.Query()
	query.Set("redirect", redirectPath)
	loginURL.RawQuery = query.Encode()
	loginURL.Fragment = uuid.UUID(id).String()

	data := magicLinkData{
		URL:  loginURL,
		Code: code,
	}

	return s.mailer.Send(*s.config.DefaultSender,
		email,
		s.magicLinkTemplate,
		data)
}

func userResource(email mail.Address, parts ...string) Resource {
	parts = append([]string{email.Address}, parts...)
	path, err := url.JoinPath("/user", parts...)
	if err != nil {
		panic(err)
	}

	return Resource(path)
}

// func (s *Service) findCapability(tx storage.ReadTx, id CapabilityID) (*Capability, error) {
// 	return storage.Load[*Capability](tx, capabilityBucket, id[:])
// }

func findUserByEmail(ctx context.Context, tx storage.ReadTx,
	email mail.Address) (*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("find user by E-mail: err=%w", err)
	}

	if err := cap.RequireRead(userResource(email)); err != nil {
		return nil, err
	}

	users, err := storage.ListAll[*User](tx, userBucket)
	if err != nil {
		return nil, err
	}

	// TODO: This is highly inefficient
	for _, u := range users {
		if u.Email.Address == email.Address {
			return u, nil
		}
	}
	return nil, fmt.Errorf("no matching user found: err=%w", storage.ErrNotFound)
}

func createUser(ctx context.Context, tx storage.WriteTx,
	email mail.Address) (*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("create user: err=%w", err)
	}

	if err := cap.RequireWrite(userResource(email, "signup")); err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("create user ID: err=%w", err)
	}

	user := &User{
		ID:    storage.UserID(id),
		Email: email,
		Role:  RoleUser,
	}

	if err := storage.Store(tx, userBucket, user, user.ID); err != nil {
		return nil, fmt.Errorf("create user: err=%w", err)
	}

	return user, nil
}

func findOrCreateUser(ctx context.Context, tx storage.WriteTx,
	email mail.Address) (*User, error) {
	user, err := findUserByEmail(ctx, tx, email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			user, err = createUser(ctx, tx, email)
		}

		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *Service) login(ctx context.Context, tx storage.WriteTx,
	email mail.Address, shortCode string) (CapabilityID, *Capability, error) {
	capability, err := findCapabilityByShortCode(tx, shortCode)
	if err != nil {
		return CapabilityID{}, nil, err
	}

	if err := RevokeCapability(tx, capability, serviceUser); err != nil {
		return CapabilityID{}, nil, fmt.Errorf("revoke login token: err=%w", err)
	}

	ctx, err = Set(ctx, tx, capability)
	if err != nil {
		return CapabilityID{}, nil, err
	}

	cap, err := FromContext(ctx)
	if err != nil {
		return CapabilityID{}, nil, fmt.Errorf("login: err=%w", err)
	}

	if err := cap.RequireWrite(userResource(email, "login")); err != nil {
		return CapabilityID{}, nil, err
	}

	user, err := findOrCreateUser(ctx, tx, email)
	if err != nil {
		if err != nil {
			return CapabilityID{}, nil, fmt.Errorf("login: err=%w", err)
		}
	}

	scopes := Scopes{}
	scopes.Add(PermissionOwner, "user", user.ID.String(), "*")
	capabilityID, capability, err := IssueCapability(tx, serviceUser, user, scopes,
		loginTokenValidity)
	if err != nil {
		return capabilityID, capability, fmt.Errorf("login: err=%w", err)
	}

	return capabilityID, capability, nil
}

func (s *Service) setCapabilityCookie(w http.ResponseWriter, capID CapabilityID,
	cap Capability) {
	cookie := http.Cookie{
		Name:     sessionCookieName,
		Value:    capID.String(),
		Path:     "/",
		Secure:   !strings.Contains(s.config.Host.Hostname(), "localhost"),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  cap.ValidUntil,
	}
	http.SetCookie(w, &cookie)
}

// TODO: Move to user service
func (s *Service) listUsers(ctx context.Context, tx storage.ReadTx) ([]*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: err=%w", err)
	}

	if err := cap.RequireRead("/user"); err != nil {
		return nil, err
	}

	users, err := storage.ListAll[*User](tx, userBucket)
	if err != nil {
		return nil, fmt.Errorf("list users: err=%w", err)
	}

	return users, nil
}

// - Create long-lived capability for /user/:id/*
// - Revoke login token

// Pros
// - One auth path for everything
// - New resources are automatically protected

// Cons
// - Permissions can't be dynamically changed, needs
//   re-authentication
// - Audiences & scopes are harder to manage

// TODO
// - Magic-link login
// - Passkeys
// - Permissions (admin, user)
// - Broadcast auth service discovery when enabled
// - Open login
// - Invite only
// - First user is admin
// - Self-service logout
// - Self-service deletion
// - Self-service data export
// - Admin user management

// Shares
// Read & write permissions
// Only the resource owner is allowed to create a share
