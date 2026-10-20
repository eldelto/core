package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type Role uint
type contextKey string

const (
	RoleAdmin = Role(iota)
	RoleUser

	authCtxKey = contextKey("auth")

	userBucket       = "auth.user"
	capabilityBucket = "auth.capability"
	shortCodeBucket  = "auth.shortCodeCapability"
)

var (
	ErrUnauthenticated = errors.New("no authentication found")
	ErrUnautorized     = errors.New("unauthorized")

	serviceUserID = storage.UserID(uuid.Nil)
	ServiceUser   = User{ID: serviceUserID}
)

type User struct {
	ID    storage.UserID
	Email mail.Address
	Role  Role
}

func (u *User) BucketKey() []byte {
	return u.ID[:]
}

type Resource string

func newResource(res string) (Resource, error) {
	res = strings.TrimSpace(res)
	if res[0] != '/' {
		return "", fmt.Errorf("resource must be an absolute path: %q", res)
	}

	switch strings.Count(res, "*") {
	case 0:
	case 1:
		if res[len(res)-1] != '*' {
			return "", fmt.Errorf("only postfix wildcards are allowed")
		}
	default:
		return "",
			fmt.Errorf("only a single postfix wildcard is allowed")

	}

	return Resource(res), nil
}

type Permission uint

const (
	PermissionNone = Permission(iota)
	PermissionRead
	PermissionWrite
	PermissionOwner
)

type Scopes map[Resource]Permission

func (s Scopes) Add(res string, perm Permission) {
	resource, err := newResource(res)
	if err != nil {
		panic(err)
	}

	s[resource] = perm
}

type CapabilityID uuid.UUID

func (id CapabilityID) String() string {
	return uuid.UUID(id).String()
}

func (id CapabilityID) ShortCode() string {
	// TODO: Use cryptographic random alpha-numeric
	return base64.RawURLEncoding.EncodeToString(id[:])
}

type CapabilityHash []byte

type Capability struct {
	Hash        CapabilityHash
	CreatedByID storage.UserID
	UserID      *storage.UserID
	ValidUntil  time.Time
	RevokedAt   *time.Time
	Scopes      Scopes
}

func (c *Capability) BucketKey() []byte {
	return c.Hash
}

type CapabilityCtx struct {
	Hash   CapabilityHash
	User   *User
	Scopes Scopes
}

func (c *CapabilityCtx) RequireUser() (User, error) {
	if c.User != nil {
		return *c.User, nil
	}

	return User{}, fmt.Errorf("requires user context: capability=%q err=%w",
		c.Hash, ErrUnautorized)
}

func (c *CapabilityCtx) RequireRead(res Resource) error {
	switch c.Scopes[res] {
	case PermissionRead:
	case PermissionWrite:
	case PermissionOwner:
	default:
		return fmt.Errorf("requires read scope: resource=%q capability=%q err=%w",
			res, c.Hash, ErrUnautorized)
	}

	return nil
}

func (c *CapabilityCtx) RequireWrite(res Resource) error {
	switch c.Scopes[res] {
	case PermissionWrite:
	case PermissionOwner:
	default:
		return fmt.Errorf("requires write scope: resource=%q capability=%q err=%w",
			res, c.Hash, ErrUnautorized)
	}

	return nil
}

func hash(id uuid.UUID) CapabilityHash {
	return argon2.IDKey(id[:], []byte("j3ij?aSDi3jl1"),
		3, 64*1024, 4, 32)
}

func validUntil(validForMinutes uint) time.Time {
	return time.Now().Add(time.Duration(validForMinutes) * time.Minute)
}

func generateCapability(creator User, user *User, scopes Scopes,
	validForMinutes uint) (CapabilityID, Capability, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return CapabilityID{}, Capability{}, fmt.Errorf("generate token: %w", err)
	}

	var userID *storage.UserID
	if user != nil {
		userID = &user.ID
	}

	capability := Capability{
		Hash:        hash(id),
		CreatedByID: creator.ID,
		UserID:      userID,
		ValidUntil:  validUntil(validForMinutes),
		Scopes:      scopes,
	}

	// TODO: Use hmac on the ID
	return CapabilityID(id), capability, nil
}

func FromContext(ctx context.Context) (CapabilityCtx, error) {
	value := ctx.Value(authCtxKey)
	if value == nil {
		return CapabilityCtx{}, ErrUnauthenticated
	}

	capability, ok := value.(CapabilityCtx)
	if !ok {
		return CapabilityCtx{}, fmt.Errorf("failed to cast '%v' to type CapabilityCtx: %w",
			value, ErrUnauthenticated)
	}

	return capability, nil
}

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
	store.RegisterBucket(storage.Bucket{Name: userBucket})
	store.RegisterBucket(storage.Bucket{Name: capabilityBucket})
	store.RegisterBucket(storage.Bucket{Name: shortCodeBucket})

	return &Service{
		config:            config,
		store:             store,
		mailer:            mailer,
		magicLinkTemplate: magicLinkTemplate,
	}
}

func (s *Service) IssueCapability(ctx context.Context, tx storage.WriteTx, user *User, scopes Scopes,
	validForMinutes uint) (CapabilityID, Capability, error) {
	capCtx, err := FromContext(ctx)
	if err != nil {
		return CapabilityID{}, Capability{}, err
	}

	creator, err := capCtx.RequireUser()
	if err != nil {
		return CapabilityID{}, Capability{},
			fmt.Errorf("issue capability: %w", err)
	}

	id, capability, err := generateCapability(creator, user, scopes, validForMinutes)
	if err != nil {
		return CapabilityID{}, Capability{}, err
	}

	if err := storage.Store(tx, capabilityBucket, &capability, creator.ID); err != nil {
		return CapabilityID{}, Capability{}, err
	}

	return id, capability, nil
}

type shortCodeCapability struct {
	Code string
	Hash CapabilityHash
}

func (s shortCodeCapability) BucketKey() []byte {
	return []byte(s.Code)
}

func (s *Service) storeWithShortCode(tx storage.WriteTx, code string,
	cap *Capability) error {
	scc := shortCodeCapability{
		Code: code,
		Hash: cap.Hash,
	}
	fmt.Printf("storing: %v\n", scc)
	if err := storage.Store(tx, shortCodeBucket, scc, cap.CreatedByID); err != nil {
		return fmt.Errorf("store short code: err=%w", err)
	}

	return nil
}

func (s *Service) findCapabilityByShortCode(tx storage.ReadTx, code string) (*Capability, error) {
	mapping, err := storage.Load[shortCodeCapability](tx, shortCodeBucket, []byte(code))
	if err != nil {
		return nil, fmt.Errorf("find short code mapping: err=%w", err)
	}

	cap, err := storage.Load[*Capability](tx, capabilityBucket, mapping.Hash)
	if err != nil {
		return nil, fmt.Errorf("find by capability: err=%w", err)
	}

	return cap, nil
}

type magicLinkData struct {
	URL  *url.URL
	Code string
}

func (s *Service) SendMagicLink(ctx context.Context, tx storage.WriteTx, email mail.Address, redirectPath string) error {
	scopes := Scopes{}
	scopes.Add(string(userResource(email, "login")), PermissionWrite)
	ctx = fakeCapability(ctx, scopes)

	scopes = Scopes{}
	resource, err := url.JoinPath("/user", email.Address, "login")
	if err != nil {
		return err
	}

	scopes.Add(resource, PermissionWrite)

	id, cap, err := s.IssueCapability(ctx, tx, nil, scopes, 5)
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

	code := id.ShortCode()
	if err := s.storeWithShortCode(tx, code, &cap); err != nil {
		return err
	}

	data := magicLinkData{
		URL:  loginURL,
		Code: code,
	}
	fmt.Println(data)

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

func (s *Service) Set(ctx context.Context, tx storage.ReadTx,
	capability *Capability) (context.Context, error) {
	var user *User
	if capability.UserID != nil {
		u, err := storage.Load[*User](tx, userBucket, capability.UserID[:])
		if err != nil {
			return ctx, fmt.Errorf("set capability context: capability= %q err=%w",
				capability.Hash, err)
		}
		user = u
	}

	capCtx := CapabilityCtx{
		Hash:   capability.Hash,
		User:   user,
		Scopes: capability.Scopes,
	}
	return context.WithValue(ctx, authCtxKey, capCtx), nil
}

func (s *Service) FindUserByEmail(ctx context.Context, tx storage.ReadTx,
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

func (s *Service) createUser(ctx context.Context, tx storage.WriteTx,
	email mail.Address) (*User, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("create user: err=%w", err)
	}

	if err := cap.RequireWrite(userResource(email)); err != nil {
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

const loginTokenValidity = 60 * 24 * 30 // 30 days

func (s *Service) Login(ctx context.Context, tx storage.WriteTx,
	email mail.Address) (CapabilityID, Capability, error) {
	cap, err := FromContext(ctx)
	if err != nil {
		return CapabilityID{}, Capability{}, fmt.Errorf("login: err=%w", err)
	}

	fmt.Println(cap.Scopes)
	if err := cap.RequireWrite(userResource(email, "login")); err != nil {
		return CapabilityID{}, Capability{}, err
	}

	scopes := Scopes{}
	scopes.Add(string(userResource(email)), PermissionWrite)
	ctx = fakeCapability(ctx, scopes)

	user, err := s.FindUserByEmail(ctx, tx, email)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			user, err = s.createUser(ctx, tx, email)
		}

		if err != nil {
			return CapabilityID{}, Capability{}, fmt.Errorf("login: err=%w", err)
		}
	}

	scopes = Scopes{}
	resource, err := url.JoinPath("/user", user.ID.String(), "*")
	if err != nil {
		panic(err)
	}

	scopes.Add(resource, PermissionOwner)
	capabilityID, capability, err := s.IssueCapability(ctx, tx, user, scopes, loginTokenValidity)
	if err != nil {
		return capabilityID, capability, fmt.Errorf("login: err=%w", err)
	}

	return capabilityID, capability, nil
}

func fakeCapability(ctx context.Context, scopes Scopes) context.Context {
	capCtx := CapabilityCtx{
		User:   &ServiceUser,
		Scopes: scopes,
	}

	return context.WithValue(ctx, authCtxKey, capCtx)
}

type loginData struct {
	Host  *url.URL
	Token CapabilityID
}

func (s *Service) SendLoginEmail(ctx context.Context, email mail.Address,
	id CapabilityID) error {
	cap, err := FromContext(ctx)
	if err != nil {
		return fmt.Errorf("send login E-mail: err=%w", err)
	}

	if err := cap.RequireWrite(userResource(email, "login")); err != nil {
		return err
	}

	data := loginData{
		Host:  s.config.Host,
		Token: id,
	}

	if err := s.mailer.Send(*s.config.DefaultSender, email, s.magicLinkTemplate, data); err != nil {
		return fmt.Errorf("send login E-mail: err=%w", err)
	}
	return nil
}

const sessionCookieName = "thy_session"

func (s *Service) setCapabilityCookie(w http.ResponseWriter, capID CapabilityID, cap Capability) {
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

func (s *Service) ListUsers(ctx context.Context, tx storage.ReadTx) ([]*User, error) {
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

// Modules
// - Users (identity & permissions)
// - Auth (tokens & bootstrap)

// Flow
// - User enters E-mail
// - Create short-lived capability for /auth/user/:id
// - Send link with token in fragment (is this safe?)
// - User clicks
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
