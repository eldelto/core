package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

type UserID struct{ uuid.UUID }
type TokenID string

func GenerateToken(length int) (TokenID, error) {
	rawToken := make([]byte, length)
	_, err := rand.Read(rawToken)
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return TokenID(base64.URLEncoding.EncodeToString(rawToken)), nil
}
