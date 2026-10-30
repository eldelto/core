package auth_test

import (
	"context"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/eldelto/core/auth"
	. "github.com/eldelto/core/internal/testutils"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// TODO: Tests to implement
//
// Capability
// - Valid
// - RequireUser
// - RequireRead
// - RequireWrite
//
// Service
// - IssueCapability
// - SendMagicLink

func setup(t *testing.T) (*storage.Storage, *auth.Service) {
	path := filepath.Join(os.TempDir(), "auth.db")
	store := storage.Require(t.Context(), path)
	t.Cleanup(func() { store.Close() })
	
	host, err := url.Parse("localhost:9999")
	if err != nil {
		t.Fatal(err)
	}

	mailer := &web.StubMailer{}
	defaultSender, err := mail.ParseAddress("no-reply@thy.computer")
	if err != nil {
		t.Fatal(err)
	}

	config := auth.Config{
		Host:          host,
		DefaultSender: defaultSender,
	}
	service := auth.NewService(config, store, mailer, auth.MagicLinkTemplate)

	return store, service
}

func generateUUID() *rapid.Generator[uuid.UUID] {
	return rapid.Custom(func(t *rapid.T) uuid.UUID {
		bs := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "UUID bytes")
		var id [16]byte
		copy(id[:], bs[:16])
		return uuid.UUID(id)
	})
}

func generateEmail() *rapid.Generator[mail.Address] {
	return rapid.Custom(func(t *rapid.T) mail.Address {
		raw := rapid.StringMatching(`[a-zA-Z0-9]+@[a-zA-Z0-9]+\.[a-zA-Z]+`).
			Draw(t, "raw E-mail")
		email, err := mail.ParseAddress(raw)
		if err != nil {
			t.Fatalf("E-mail generator: raw=%q, err=%v", raw, err)
		}
		return *email
	})
}

func generateDuration() *rapid.Generator[time.Duration] {
	return rapid.Custom(func(t *rapid.T) time.Duration {
		return time.Duration(rapid.Int64Min(int64(time.Second)).Draw(t, "duration"))
	})
}

func generateRole() *rapid.Generator[auth.Role] {
	return rapid.Custom(func(t *rapid.T) auth.Role {
		raw := rapid.UintMax(uint(auth.RoleAdmin)).Draw(t, "raw role")
		return auth.Role(raw)
	})
}

func generatePermission() *rapid.Generator[auth.Permission] {
	return rapid.Custom(func(t *rapid.T) auth.Permission {
		raw := rapid.UintMax(uint(auth.PermissionOwner)).Draw(t, "raw permission")
		return auth.Permission(raw)
	})
}

func generateScopes() *rapid.Generator[auth.Scopes] {
	return rapid.Custom(func(t *rapid.T) auth.Scopes {
		max := rapid.IntMin(1).Draw(t, "scope max")
		resources := rapid.SliceOfN(rapid.StringMatching(`/[a-z/]\*?`), 1, max).
			Draw(t, "resources")
		
		scopes := auth.Scopes{}
		for _, r := range resources {
			perm := generatePermission().Draw(t, "permission")
			scopes.Add(perm, r)
		}
		
		return scopes
	})
}

func generateUser() *rapid.Generator[auth.User] {
	return rapid.Custom(func(t *rapid.T) auth.User {
		return auth.User{
			ID: storage.UserID(generateUUID().Draw(t, "ID")),
			Email: generateEmail().Draw(t, "Email"),
			Role: generateRole().Draw(t, "Role"),
		}
	})
}

func fakeCapability(ctx context.Context, scopes auth.Scopes) context.Context {
	capCtx := auth.CapabilityCtx{
		User:   &auth.ServiceUser,
		Scopes: scopes,
	}

	return context.WithValue(ctx, auth.AuthCtxKey, capCtx)
}

func TestIssueCapability(t *testing.T) {
	store, service := setup(t)
	now := time.Now()
	
	rapid.Check(t, func(t *rapid.T) {
		var (
			err error
			capID auth.CapabilityID
			cap auth.Capability
			user = generateUser().Draw(t, "user")
			scopes = generateScopes().Draw(t, "scopes")
			validFor = generateDuration().Draw(t, "validFor")
		)

		ctx := fakeCapability(context.Background(), auth.Scopes{})
		
		err = store.Write(func(tx storage.WriteTx) error {
			capID, cap, err = service.IssueCapability(ctx, tx, &user, scopes, validFor)
			return err
		})
		
		AssertNoError(t, err, "issue capability")
		AssertNotEquals(t, uuid.UUID{}, capID, "capability ID is not empty")
		AssertEquals(t, *cap.UserID, user.ID, "user IDs match")
		AssertEquals(t, auth.ServiceUser.ID, cap.CreatedByID, "creator IDs match")
		AssertEquals(t, scopes, cap.Scopes, "scopes match")
		AssertEquals(t, true, now.Before(cap.ValidUntil), "valid until is in the future")
		AssertEquals(t, true, cap.Valid(), "capability is valid")
	})
}
