package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignIsStableAndVerifiable(t *testing.T) {
	body := []byte(`{"event":"ticket.created"}`)
	secret := "whsec_test"
	sig := Sign(body, secret)
	require.True(t, len(sig) > len("sha256="))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	require.Equal(t, want, sig)
}
