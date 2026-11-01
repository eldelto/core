package auth

import (
	"strings"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
	"pgregory.net/rapid"
)

func TestRequireRead_ExactMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sourceScopes := generateScopes().Draw(t, "scopes")
		resource := generateResource().Draw(t, "resource")

		scopes := Scopes{}
		for res, perm := range sourceScopes {
			if strings.HasSuffix(string(res), "*") {
				continue
			}
			scopes[res] = perm
		}
		cap := CapabilityCtx{Scopes: scopes}

		err := cap.RequireRead(resource)
		if perm, ok := scopes[resource]; ok && perm != PermissionNone {
			AssertNoError(t, err, "reading is allowed")
		} else {
			AssertError(t, err, "reading is not allowed")
		}
	})
}

func TestRequireRead_PrefixMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := "/base"
		resource := Resource(
			base + strings.TrimRight(string(generateResource().Draw(t, "suffix")), "*"))

		matchScopes := Scopes{}
		matchScopes.Add(PermissionRead, string(base), "*")
		cap := CapabilityCtx{Scopes: matchScopes}
		AssertNoError(t, cap.RequireRead(resource), "match wildcard")

		failScopes := Scopes{}
		failScopes.Add(PermissionRead, "/other",
			string(generateResource().Draw(t, "fail prefix")))
		cap = CapabilityCtx{Scopes: failScopes}
		AssertError(t, cap.RequireRead(resource), "not match wildcard")
	})
}

func TestRequireWrite_ExactMatch(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sourceScopes := generateScopes().Draw(t, "scopes")
		resource := generateResource().Draw(t, "resource")

		scopes := Scopes{}
		for res, perm := range sourceScopes {
			if strings.HasSuffix(string(res), "*") {
				continue
			}
			scopes[res] = perm
		}
		cap := CapabilityCtx{Scopes: scopes}

		err := cap.RequireWrite(resource)
		if perm, ok := scopes[resource]; ok &&
			(perm == PermissionWrite || perm == PermissionOwner) {
			AssertNoError(t, err, "reading is allowed")
		} else {
			AssertError(t, err, "reading is not allowed")
		}
	})
}

func TestRequireUser(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		user := generateUser().Draw(t, "user")
		resource := generateResource().Draw(t, "resource")

		scopes := Scopes{}
		scopes.Add(PermissionRead, string(resource))

		cap := CapabilityCtx{Scopes: scopes}
		_, err := cap.RequireUser()
		AssertError(t, err, "has no user")

		cap = CapabilityCtx{Scopes: scopes, User: &user}
		u, err := cap.RequireUser()
		AssertNoError(t, err, "has a user")
		AssertEquals(t, user, u, "returns the same user")
	})
}
