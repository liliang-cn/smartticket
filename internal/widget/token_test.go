package widget

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestIssueParseToken_RoundTrip(t *testing.T) {
	const secret = "test-secret-value"
	const ticketID uint = 42

	tok, err := IssueToken(ticketID, secret)
	require.NoError(t, err)
	require.NotEmpty(t, tok)

	got, err := ParseToken(tok, secret)
	require.NoError(t, err)
	require.Equal(t, ticketID, got)
}

func TestParseToken_WrongSecret(t *testing.T) {
	tok, err := IssueToken(7, "correct-secret")
	require.NoError(t, err)

	_, err = ParseToken(tok, "wrong-secret")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseToken_MalformedToken(t *testing.T) {
	_, err := ParseToken("not.a.jwt", "secret")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseToken_EmptyToken(t *testing.T) {
	_, err := ParseToken("", "secret")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestParseToken_ExpiredToken(t *testing.T) {
	// Manually build an already-expired token.
	claims := tokenClaims{
		TicketID: 99,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	require.NoError(t, err)

	_, err = ParseToken(tok, "secret")
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestIssueToken_EmptySecret(t *testing.T) {
	_, err := IssueToken(1, "")
	require.Error(t, err)
}
