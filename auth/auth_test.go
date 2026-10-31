package auth

import (
	"context"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"time"

	. "github.com/eldelto/core/internal/testutils"
	"github.com/eldelto/core/storage"
	"github.com/eldelto/core/web"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// TODO: Tests to implement
//
// Capability
// - RequireUser
// - RequireRead
// - RequireWrite
//
// Service
// - Login
// - SendMagicLink

func setup(t *testing.T) (*storage.Storage, *Service, *web.StubMailer) {
	path := filepath.Join(os.TempDir(), "db")
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

	config := Config{
		Host:          host,
		DefaultSender: defaultSender,
	}
	service := NewService(config, store, mailer, MagicLinkTemplate)

	return store, service, mailer
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

func generateRole() *rapid.Generator[Role] {
	return rapid.Custom(func(t *rapid.T) Role {
		raw := rapid.UintMax(uint(RoleAdmin)).Draw(t, "raw role")
		return Role(raw)
	})
}

func generatePermission() *rapid.Generator[Permission] {
	return rapid.Custom(func(t *rapid.T) Permission {
		raw := rapid.UintMax(uint(PermissionOwner)).Draw(t, "raw permission")
		return Permission(raw)
	})
}

func generateScopes() *rapid.Generator[Scopes] {
	return rapid.Custom(func(t *rapid.T) Scopes {
		max := rapid.IntMin(1).Draw(t, "scope max")
		resources := rapid.SliceOfN(rapid.StringMatching(`/[a-z/]\*?`), 1, max).
			Draw(t, "resources")

		scopes := Scopes{}
		for _, r := range resources {
			perm := generatePermission().Draw(t, "permission")
			scopes.Add(perm, r)
		}

		return scopes
	})
}

func generateUser() *rapid.Generator[User] {
	return rapid.Custom(func(t *rapid.T) User {
		return User{
			ID:    storage.UserID(generateUUID().Draw(t, "ID")),
			Email: generateEmail().Draw(t, "Email"),
			Role:  generateRole().Draw(t, "Role"),
		}
	})
}

func fakeCapability(ctx context.Context, scopes Scopes) context.Context {
	capCtx := CapabilityCtx{
		User:   &serviceUser,
		Scopes: scopes,
	}

	return context.WithValue(ctx, AuthCtxKey, capCtx)
}

func TestIssueCapability(t *testing.T) {
	store, _, _ := setup(t)
	now := time.Now()

	rapid.Check(t, func(t *rapid.T) {
		var (
			err      error
			capID    CapabilityID
			cap      *Capability
			user     = generateUser().Draw(t, "user")
			scopes   = generateScopes().Draw(t, "scopes")
			validFor = generateDuration().Draw(t, "validFor")
		)

		err = store.Write(func(tx storage.WriteTx) error {
			capID, cap, err = IssueCapability(tx, serviceUser, &user, scopes, validFor)
			return err
		})

		AssertNoError(t, err, "issue capability")
		AssertNotEquals(t, uuid.UUID{}, capID, "capability ID is not empty")
		AssertEquals(t, *cap.UserID, user.ID, "user IDs match")
		AssertEquals(t, serviceUser.ID, cap.CreatedByID, "creator IDs match")
		AssertEquals(t, scopes, cap.Scopes, "scopes match")
		AssertEquals(t, true, now.Before(cap.ValidUntil), "valid until is in the future")
		AssertEquals(t, true, cap.Valid(), "capability is valid")
	})
}

func TestLogin(t *testing.T) {
	store, service, mailer := setup(t)

	rapid.Check(t, func(t *rapid.T) {
		mailer.Reset()

		var (
			err          error
			capID        CapabilityID
			cap          *Capability
			email        = generateEmail().Draw(t, "email")
			redirectPath = rapid.String().Draw(t, "redirectPath")
		)

		ctx := fakeCapability(context.Background(), Scopes{})

		err = store.Write(func(tx storage.WriteTx) error {
			return service.sendMagicLink(ctx, tx, email, redirectPath)
		})
		AssertNoError(t, err, "send magic link")

		AssertEquals(t, 1, len(mailer.SentMails), "send one E-mail")
		mail := mailer.SentMails[0]
		AssertEquals(t, "no-reply@thy.computer", mail.From, "E-mail sender")
		AssertEquals(t, email.Address, mail.To, "E-mail recipient")

		r := regexp.MustCompile(`<h2>([^<]+)</h2>`)
		matches := r.FindAllStringSubmatch(mail.Content, 1)
		shortCode := matches[0][1]

		err = store.Write(func(tx storage.WriteTx) error {
			capID, cap, err = service.login(ctx, tx, email, shortCode)
			return err
		})
		AssertNoError(t, err, "login")

		// Find user with E-mail
		scopes := Scopes{}
		scopes.Add(PermissionRead, "/user", "*")
		ctx = fakeCapability(context.Background(), scopes)

		var users []*User
		err = store.Read(func(tx storage.ReadTx) error {
			users, err = service.listUsers(ctx, tx)
			return err
		})
		AssertNoError(t, err, "list users")

		i := slices.IndexFunc(users, func(u *User) bool {
			return u.Email.Address == email.Address
		})
		user := users[i]
		AssertNotEquals(t, uuid.UUID{}, capID, "capability ID is not empty")
		AssertEquals(t, *cap.UserID, user.ID, "user IDs match")
		AssertEquals(t, serviceUser.ID, cap.CreatedByID, "creator IDs match")

		scopes = Scopes{}
		scopes.Add(PermissionOwner, "/user", user.ID.String(), "*")
		AssertEquals(t, scopes, cap.Scopes, "scopes match")
		AssertEquals(t, true, cap.Valid(), "capability is valid")

		err = store.Read(func(tx storage.ReadTx) error {
			cap, err = findCapabilityByShortCode(tx, shortCode)
			return err
		})
		AssertError(t, err, "not find the revoked login token")
	})
}
