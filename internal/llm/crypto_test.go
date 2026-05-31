package llm

import "testing"

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
}
