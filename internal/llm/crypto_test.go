package llm

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := make([]byte, 32) // 32 zero bytes is a valid AES-256 key for the test
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	plain := "sk-secret-12345"
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plain || enc == "" {
		t.Fatalf("ciphertext not transformed: %q", enc)
	}
	got, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plain {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plain)
	}
}

func TestEncryptNonceVaries(t *testing.T) {
	c, _ := NewCipher(make([]byte, 32))
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Fatal("expected different ciphertext per call (random nonce)")
	}
}

func TestNewCipherRejectsBadKey(t *testing.T) {
	if _, err := NewCipher(make([]byte, 16)); err == nil {
		t.Fatal("expected error for non-32-byte key")
	}
}

func TestLoadKeyFromHexAndBase64(t *testing.T) {
	hexKey := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	if _, err := LoadKey(hexKey); err != nil {
		t.Fatalf("hex key: %v", err)
	}

	b64Key := base64.StdEncoding.EncodeToString(make([]byte, 32)) // 44 chars
	if _, err := LoadKey(b64Key); err != nil {
		t.Fatalf("base64 key: %v", err)
	}
}

func TestLoadKeyRejectsWrongSize(t *testing.T) {
	if _, err := LoadKey("not-a-key"); err == nil {
		t.Fatal("expected error for malformed key")
	}
	// Valid base64 but only 16 bytes decoded → must be rejected.
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := LoadKey(short); err == nil {
		t.Fatal("expected error for 16-byte key")
	}
}

func TestDecryptRejectsBadInput(t *testing.T) {
	c, _ := NewCipher(make([]byte, 32))
	if _, err := c.Decrypt("!!!not-base64!!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	// Valid base64 but shorter than the GCM nonce → too short.
	tooShort := base64.StdEncoding.EncodeToString([]byte("short"))
	if _, err := c.Decrypt(tooShort); err == nil {
		t.Fatal("expected error for ciphertext shorter than nonce")
	}
}
