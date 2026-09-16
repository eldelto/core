package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"time"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/google/uuid"
)

type Role uint

const (
	RoleAdmin = Role(iota)
	RoleUser

	authCtxKey = "auth"

	userBucket       = "auth.user"
	capabilityBucket = "auth.capability"
)

var (
	ErrUnauthenticated = errors.New("no authentication found")
	ServiceUserID      = storage.UserID(uuid.Nil[:])
)

type User struct {
	ID    storage.UserID
	Email mail.Address
	Role  Role
}

type Resource string
type Permission string

type Scope struct {
	Resource    Resource
	Permissions []Permission
}

type CapabilityID uuid.UUID
type CapabilityHash string

type Capability struct {
	Hash       CapabilityHash
	UserID     *storage.UserID
	ValidUntil time.Time
	RevokedAt  *time.Time
	Scopes     []Scope
}

func (c *Capability) BucketKey() []byte {
	return []byte(c.Hash)
}

func GenerateCapability(u *User, scopes []Scope, validMinutes uint) (CapabilityID, Capability, error) {
	rawCapability := make([]byte, length)
	_, err := rand.Read(rawCapability)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return CapabilityID(base64.URLEncoding.EncodeToString(rawCapability)), nil
}

func FromContext(ctx context.Context) (Capability, error) {
	value := ctx.Value(authCtxKey)
	if value == nil {
		return Capability{}, ErrUnauthenticated
	}

	token, ok := value.(Capability)
	if !ok {
		return Capability{}, fmt.Errorf("failed to cast '%v' to type Capability: %w",
			value, ErrUnauthenticated)
	}

	return token, nil
}

func Set(ctx context.Context, token Capability) context.Context {
	return context.WithValue(ctx, authCtxKey, token)
}

type Service struct {
	store  storage.Storage
	mailer web.Mailer
	host   *url.URL
}

func NewService(store storage.Storage, mailer web.Mailer) *Service {
	store.RegisterBucket(storage.Bucket{Name: userBucket})
	store.RegisterBucket(storage.Bucket{Name: capabilityBucket})

	return &Service{
		store:  store,
		mailer: mailer,
	}
}

func generateLoginCapability(email mail.Address) (CapabilityID, Capability, error) {
	return GenerateCapability(nil,
		[]Scope{{
			Resource:    "user",
			Permissions: []Permission{"signup", "login"}},
		}, 5)
}

func (s *Service) SendMagicLink(ctx context.Context, email mail.Address, redirectPath string) error {
	capabilityID, capability, err := generateLoginCapability(email)
	if err != nil {
		return fmt.Errorf("magic link: %w", err)
	}

	loginURL, err := url.Parse(s.host.String())
	if err != nil {
		return fmt.Errorf("login URL: %w", err)
	}

	redirectPath = url.PathEscape(redirectPath)
	query := loginURL.Query()
	query.Set("redirect", redirectPath)
	loginURL.RawQuery = query.Encode()

	s.host.Fragment = uuid.UUID(capabilityID).String()

	return s.store.Write(func(tx storage.WriteTx) error {
		err := storage.Store(tx, capabilityBucket, &capability,
			ServiceUserID)
		if err != nil {
			return fmt.Errorf("store login capability: %w", err)
		}

		return s.mailer.Send(s.mailer.DefaultSender,
			email,
			magicLinkTemplate,
			loginURL)
	})
}

// func (s *Service)

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
// - First use is admin
// - Self-service logout
// - Self-service deletion
// - Self-service data export
// - Admin user management

// Shares
// Read & write permissions
// Only the resource owner is allowed to create a share
