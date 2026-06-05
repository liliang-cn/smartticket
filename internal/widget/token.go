// Package widget provides the embeddable web-chat widget backend: session
// management, conversation token issuance, and message relay.
package widget

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned by ParseToken when the token is malformed,
// expired, or was signed with a different secret.
var ErrInvalidToken = errors.New("invalid or expired conversation token")

// tokenClaims are the JWT claims stored inside a conversation token.
type tokenClaims struct {
	TicketID uint `json:"tid"`
	jwt.RegisteredClaims
}

// IssueToken mints a signed HMAC-SHA256 JWT for the given ticketID. The token
// expires after 7 days. The caller supplies the server secret so this function
// is side-effect-free and easily testable.
func IssueToken(ticketID uint, secret string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("widget.IssueToken: secret must not be empty")
	}
	claims := tokenClaims{
		TicketID: ticketID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("widget.IssueToken: %w", err)
	}
	return signed, nil
}

// ParseToken validates a conversation token and returns the ticketID it encodes.
// Returns ErrInvalidToken for any validation failure (tampered, expired, wrong
// secret) so callers never inadvertently leak parse-error details.
func ParseToken(token, secret string) (uint, error) {
	if token == "" || secret == "" {
		return 0, ErrInvalidToken
	}

	parsed, err := jwt.ParseWithClaims(token, &tokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil || !parsed.Valid {
		return 0, ErrInvalidToken
	}

	claims, ok := parsed.Claims.(*tokenClaims)
	if !ok || claims.TicketID == 0 {
		return 0, ErrInvalidToken
	}
	return claims.TicketID, nil
}
