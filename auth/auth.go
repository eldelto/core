package auth

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"net/url"
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
	LoginPath     string
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

func (s *Service) sendMagicLink(ctx context.Context, tx storage.WriteTx, email mail.Address) error {
	scopes := Scopes{}
	scopes.Add(PermissionWrite, "user", email.Address, "login")
	scopes.Add(PermissionWrite, "user", email.Address, "signup")
	scopes.Add(PermissionRead, "user", email.Address)

	_, code, _, err := IssueShortLivedCapability(tx, serviceUser, nil, scopes)
	if err != nil {
		return fmt.Errorf("magic link: %w", err)
	}

	return s.mailer.Send(*s.config.DefaultSender,
		email,
		s.magicLinkTemplate,
		code)
}

func userEmailResource(email mail.Address, parts ...string) Resource {
	parts = append([]string{email.Address}, parts...)
	path, err := url.JoinPath("/user", parts...)
	if err != nil {
		panic(err)
	}

	return Resource(path)
}

func findUserByEmail(ctx context.Context, tx storage.ReadTx,
	email mail.Address) (*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("find user by E-mail: err=%w", err)
	}

	if err := cap.RequireRead(userEmailResource(email)); err != nil {
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

func determineUserRole(tx storage.ReadTx) (Role, error) {
	count, err := storage.Count(tx, userBucket)
	if err != nil {
		return 0, err
	}

	fmt.Printf("count=%d\n", count)

	if count == 0 {
		return RoleAdmin, nil
	}
	return RoleUser, nil
}

func createUser(ctx context.Context, tx storage.WriteTx,
	email mail.Address) (*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("create user: err=%w", err)
	}

	if err := cap.RequireWrite(userEmailResource(email, "signup")); err != nil {
		return nil, err
	}

	id, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("create user ID: err=%w", err)
	}

	role, err := determineUserRole(tx)
	if err != nil {
		return nil, fmt.Errorf("create user ID: err=%w", err)
	}

	user := &User{
		ID:    storage.UserID(id),
		Email: email,
		Role:  role,
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
			return createUser(ctx, tx, email)
		}

		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

func createUserScopes(u *User) Scopes {
	scopes := Scopes{}
	switch u.Role {
	case RoleUser:
		scopes.Add(PermissionOwner, "user", u.ID.String(), "*")

	case RoleAdmin:
		scopes.Add(PermissionOwner, "user", u.ID.String(), "*")
		scopes.Add(PermissionWrite, "user")
	}

	return scopes
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

	if err := cap.RequireWrite(userEmailResource(email, "login")); err != nil {
		return CapabilityID{}, nil, err
	}

	user, err := findOrCreateUser(ctx, tx, email)
	if err != nil {
		if err != nil {
			return CapabilityID{}, nil, fmt.Errorf("login: err=%w", err)
		}
	}

	scopes := createUserScopes(user)
	capabilityID, capability, err := IssueCapability(tx, serviceUser, user, scopes,
		loginTokenValidity)
	if err != nil {
		return capabilityID, capability, fmt.Errorf("login: err=%w", err)
	}

	return capabilityID, capability, nil
}

func listUserCapabilities(tx storage.ReadTx, id storage.UserID) ([]*Capability, error) {
	caps, err := storage.ListAll[*Capability](tx, capabilityBucket)
	if err != nil {
		return nil, fmt.Errorf("list user capabilities: err=%w, userID=%q", err, id)
	}

	// TODO: This is highly inefficient
	res := []*Capability{}
	for _, c := range caps {
		if c.UserID != nil && *c.UserID == id {
			res = append(res, c)
		}
	}

	return res, nil
}

func revokeAllUserCapabilities(tx storage.WriteTx, user User) error {
	caps, err := listUserCapabilities(tx, user.ID)
	if err != nil {
		return fmt.Errorf("revoke user capabilities: err=%w", err)
	}

	for _, c := range caps {
		if err := RevokeCapability(tx, c, user); err != nil {
			return fmt.Errorf("revoke user capabilities: err=%w, userID=%q", err, user.ID)
		}
	}

	return nil
}

func (s *Service) logout(ctx context.Context, tx storage.WriteTx,
	id storage.UserID) error {
	cap, err := FromContext(ctx)
	if err != nil {
		return fmt.Errorf("logout: err=%w", err)
	}

	if err := cap.RequireWrite(userResource(id, "logout")); err != nil {
		return err
	}
	user, err := cap.RequireUser()
	if err != nil {
		return fmt.Errorf("logout: err=%w", err)
	}

	return revokeAllUserCapabilities(tx, user)
}

func (s *Service) setCapabilityCookie(w http.ResponseWriter, capID CapabilityID,
	cap Capability) {
	web.SetCookie(w, s.config.Host, sessionCookieName, capID.String(), cap.ValidUntil)
}

func redirectToLogin(w http.ResponseWriter, r *http.Request, path string) {
	web.RemoveCookie(w, sessionCookieName)
	http.Redirect(w, r, path, http.StatusSeeOther)
}

func (s *Service) fromCookie(r *http.Request) (CapabilityID, *Capability, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return CapabilityID{}, nil, fmt.Errorf("no session cookie found: resource=%q, err=%w",
			r.URL.String(), err)
	}

	rawID, err := uuid.Parse(cookie.Value)
	if err != nil {
		return CapabilityID{}, nil, fmt.Errorf("malformed capability ID: ID=%q, resource=%q, err=%w",
			cookie.Value, r.URL.String(), err)
	}
	capID := CapabilityID(rawID)

	hash := hash(capID)
	var cap *Capability
	err = s.store.Read(func(tx storage.ReadTx) error {
		cap, err = s.findCapability(tx, hash)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return capID, cap, fmt.Errorf("set capability context: ID=%q, resource=%q, err=%w",
			capID, r.URL.String(), err)
	}

	return capID, cap, nil
}

func (s *Service) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capID, cap, err := s.fromCookie(r)
			if err != nil {
				log.Println(err)
				redirectToLogin(w, r, s.config.LoginPath)
				return
			}

			ctx := r.Context()
			err = s.store.Read(func(tx storage.ReadTx) error {
				ctx, err = Set(ctx, tx, cap)
				return err
			})
			if err != nil {
				log.Printf("set capability context: ID=%q, resource=%q, err=%v",
					capID, r.URL.String(), err)
				redirectToLogin(w, r, s.config.LoginPath)
				return
			}

			capCtx, err := FromContext(ctx)
			if err != nil {
				log.Printf("retrieve capability context: ID=%q, resource=%q, err=%v",
					capID, r.URL.String(), err)
				redirectToLogin(w, r, s.config.LoginPath)
				return
			}

			if err := capCtx.RequireRead(Resource(r.URL.Path)); err != nil {
				log.Printf("missing read permissions: ID=%q, resource=%q, err=%v",
					capID, r.URL.String(), err)
				http.Redirect(w, r, "/error/forbidden", http.StatusSeeOther)
				return
			}

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
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
