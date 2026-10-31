package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/validate"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type Role uint

const (
	RoleUser = Role(iota)
	RoleAdmin
)

type Permission uint

const (
	PermissionNone = Permission(iota)
	PermissionRead
	PermissionWrite
	PermissionOwner
)

type contextKey string

const (
	AuthCtxKey = contextKey("auth")

	userBucket       = "auth.user"
	capabilityBucket = "auth.capability"
	shortCodeBucket  = "auth.shortCodeCapability"
)

var (
	ErrUnauthenticated = errors.New("no authentication found")
	ErrUnautorized     = errors.New("unauthorized")
)

type User struct {
	ID    storage.UserID
	Email mail.Address
	Role  Role
}

func (u *User) BucketKey() []byte {
	return u.ID[:]
}

func (u *User) Validate() error {
	return errors.Join(
		validate.NotEmpty("email", u.Email.Address),
	)
}

type Resource string

func newResource(parts ...string) (Resource, error) {
	res, err := url.JoinPath("/", parts...)
	if err != nil {
		return "", fmt.Errorf("invalid resource path: parts=%q, err=%w", parts, err)
	}
	res = strings.TrimSpace(res)

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

type Scopes map[Resource]Permission

func (s Scopes) Add(perm Permission, parts ...string) {
	resource, err := newResource(parts...)
	if err != nil {
		panic(err)
	}

	s[resource] = perm
}

func prefixSearch(s Scopes, target Resource) *Permission {
	for res, perm := range s {
		str := strings.TrimSuffix(string(res), "/*")
		// Nothing trimmed, so no wildcard resource.
		if len(res) == len(str) {
			continue
		}

		if strings.HasPrefix(string(target), str) {
			return &perm
		}
	}

	return nil
}

func (s Scopes) PermissionFor(res Resource) *Permission {
	perm, ok := s[res]
	if !ok {
		return prefixSearch(s, res)
	}
	return &perm
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

func (c *Capability) Valid() bool {
	return c.RevokedAt == nil && time.Now().Before(c.ValidUntil)
}

func (c *Capability) Validate() error {
	return errors.Join(
		validate.NotEmpty("hash", c.Hash),
		validate.NotEmpty("scopes", c.Scopes),
	)
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
	perm := c.Scopes.PermissionFor(res)
	if perm != nil {
		switch *perm {
		case PermissionRead, PermissionWrite, PermissionOwner:
			return nil
		}
	}

	return fmt.Errorf("requires read scope: resource=%q capability=%q err=%w",
		res, c.Hash, ErrUnautorized)
}

func (c *CapabilityCtx) RequireWrite(res Resource) error {
	perm := c.Scopes.PermissionFor(res)
	if perm != nil {
		switch *perm {
		case PermissionWrite, PermissionOwner:
			return nil
		}
	}

	return fmt.Errorf("requires write scope: resource=%q capability=%q err=%w",
		res, c.Hash, ErrUnautorized)
}

func hash(id uuid.UUID) CapabilityHash {
	return argon2.IDKey(id[:], []byte("j3ij?aSDi3jl1"),
		3, 64*1024, 4, 32)
}

func validUntil(validFor time.Duration) time.Time {
	return time.Now().Add(validFor)
}

func generateCapability(creator User, user *User, scopes Scopes,
	validFor time.Duration) (CapabilityID, Capability, error) {
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
		ValidUntil:  validUntil(validFor),
		Scopes:      scopes,
	}

	// TODO: Use hmac on the ID
	return CapabilityID(id), capability, nil
}

func Set(ctx context.Context, tx storage.ReadTx,
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
	return context.WithValue(ctx, AuthCtxKey, capCtx), nil
}

func FromContext(ctx context.Context) (CapabilityCtx, error) {
	value := ctx.Value(AuthCtxKey)
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

func IssueCapability(tx storage.WriteTx, creator User, user *User, scopes Scopes,
	validFor time.Duration) (CapabilityID, *Capability, error) {
	id, capability, err := generateCapability(creator, user, scopes, validFor)
	if err != nil {
		return CapabilityID{}, nil, err
	}

	if err := storage.Store(tx, capabilityBucket, &capability, creator.ID); err != nil {
		return CapabilityID{}, nil, err
	}

	return id, &capability, nil
}

type shortCodeCapability struct {
	Code string
	Hash CapabilityHash
}

func (s shortCodeCapability) BucketKey() []byte {
	return []byte(s.Code)
}

func (s shortCodeCapability) Validate() error {
	return errors.Join(
		validate.NotEmpty("code", s.Code),
		validate.NotEmpty("hash", s.Hash),
	)
}

func storeWithShortCode(tx storage.WriteTx, code string,
	cap *Capability) error {
	// TODO: The code should also only be stored as hash.
	scc := shortCodeCapability{
		Code: code,
		Hash: cap.Hash,
	}
	if err := storage.Store(tx, shortCodeBucket, scc, cap.CreatedByID); err != nil {
		return fmt.Errorf("store short code: err=%w", err)
	}

	return nil
}

func IssueShortLivedCapability(tx storage.WriteTx, creator User, user *User, scopes Scopes) (CapabilityID, string, *Capability, error) {
	id, cap, err := IssueCapability(tx, creator, user, scopes, 5*time.Minute)
	if err != nil {
		return CapabilityID{}, "", nil, err
	}

	code := id.ShortCode()
	if err := storeWithShortCode(tx, code, cap); err != nil {
		return CapabilityID{}, "", nil, err
	}

	return id, code, cap, nil
}

func findCapabilityByShortCode(tx storage.ReadTx, code string) (*Capability, error) {
	mapping, err := storage.Load[shortCodeCapability](tx, shortCodeBucket, []byte(code))
	if err != nil {
		return nil, fmt.Errorf("find short code mapping: err=%w", err)
	}

	cap, err := storage.Load[*Capability](tx, capabilityBucket, mapping.Hash)
	if err != nil {
		return nil, fmt.Errorf("find by capability: err=%w", err)
	}

	if !cap.Valid() {
		return nil, fmt.Errorf("capability expired: err=%w", storage.ErrNotFound)
	}

	return cap, nil
}

func RevokeCapability(tx storage.WriteTx, cap *Capability, user User) error {
	revokedAt := time.Now()
	cap.RevokedAt = &revokedAt

	if err := storage.Store(tx, capabilityBucket, cap, user.ID); err != nil {
		return fmt.Errorf("revoke capability: hash=%q, err=%w", cap.Hash, err)
	}

	return nil
}
