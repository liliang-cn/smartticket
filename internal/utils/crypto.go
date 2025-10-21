package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/md5"
	cryptorand "crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

// Helper function for cryptographically secure random int.
func randomInt(max int) int {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// Fallback to simple random
		return mathrand.Intn(max)
	}
	return int(n.Int64())
}

// Hash utilities

// HashMD5 calculates MD5 hash.
func HashMD5(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HashSHA1 calculates SHA1 hash.
func HashSHA1(data string) string {
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HashSHA256 calculates SHA256 hash.
func HashSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HashSHA512 calculates SHA512 hash.
func HashSHA512(data string) string {
	hash := sha512.Sum512([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HMAC utilities

// HMACMD5 calculates HMAC-MD5.
func HMACMD5(data, key string) string {
	h := hmac.New(md5.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA1 calculates HMAC-SHA1.
func HMACSHA1(data, key string) string {
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA256 calculates HMAC-SHA256.
func HMACSHA256(data, key string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA512 calculates HMAC-SHA512.
func HMACSHA512(data, key string) string {
	h := hmac.New(sha512.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// Encryption utilities

// AESEncrypt encrypts data using AES-GCM.
func AESEncrypt(plaintext, key string) (string, error) {
	// Generate random nonce
	nonce := make([]byte, 12)
	if _, err := cryptorand.Read(nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Convert key to bytes
	keyBytes := []byte(key)
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		return "", fmt.Errorf("AES key must be 16, 24, or 32 bytes long")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Encrypt
	ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Combine nonce and ciphertext
	result := append(nonce, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// AESDecrypt decrypts data using AES-GCM.
func AESDecrypt(ciphertext, key string) (string, error) {
	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data) < 12 {
		return "", fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce := data[:12]
	ciphertextBytes := data[12:]

	// Convert key to bytes
	keyBytes := []byte(key)
	if len(keyBytes) != 16 && len(keyBytes) != 24 && len(keyBytes) != 32 {
		return "", fmt.Errorf("AES key must be 16, 24, or 32 bytes long")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher block: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := aesgcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// Password utilities

// GeneratePasswordHash generates a secure password hash.
func GeneratePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate password hash: %w", err)
	}
	return string(hash), nil
}

// VerifyCryptoPassword verifies a password against its hash (renamed to avoid conflict).
func VerifyCryptoPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateStrongPassword generates a strong password.
func GenerateStrongPassword(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8")
	}

	const (
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numbers   = "0123456789"
		special   = "!@#$%^&*()_+-=[]{}|;:,.<>?"
		allChars  = lowercase + uppercase + numbers + special
	)

	password := make([]byte, length)

	// Ensure at least one character from each category
	password[0] = lowercase[randomInt(len(lowercase))]
	password[1] = uppercase[randomInt(len(uppercase))]
	password[2] = numbers[randomInt(len(numbers))]
	password[3] = special[randomInt(len(special))]

	// Fill the rest with random characters
	for i := 4; i < length; i++ {
		password[i] = allChars[randomInt(len(allChars))]
	}

	// Shuffle the password
	mathrand.Shuffle(len(password), func(i, j int) {
		password[i], password[j] = password[j], password[i]
	})

	return string(password), nil
}

// GeneratePIN generates a numeric PIN.
func GeneratePIN(length int) string {
	if length < 4 {
		length = 4
	}
	if length > 10 {
		length = 10
	}

	pin := make([]byte, length)
	for i := range pin {
		pin[i] = '0' + byte(randomInt(10))
	}
	return string(pin)
}

// Token utilities

// GenerateToken generates a random token.
func GenerateToken(length int) (string, error) {
	if length < 16 {
		length = 16
	}

	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateSecureCryptoToken generates a cryptographically secure token (renamed to avoid conflict).
func GenerateSecureCryptoToken(length int) (string, error) {
	if length < 32 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateAPIToken generates an API token with prefix.
func GenerateAPIToken(prefix string, length int) (string, error) {
	token, err := GenerateSecureCryptoToken(length)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%s", prefix, token), nil
}

// GenerateSessionToken generates a session token.
func GenerateSessionToken() (string, error) {
	return GenerateSecureCryptoToken(32)
}

// GenerateCSRFToken generates a CSRF token.
func GenerateCSRFToken() (string, error) {
	return GenerateSecureCryptoToken(32)
}

// Key derivation utilities

// DeriveKey derives a key from password using PBKDF2.
func DeriveKey(password, salt string, iterations, keyLength int) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), iterations, keyLength, sha256.New)
}

// GenerateSalt generates a random salt.
func GenerateSalt(length int) ([]byte, error) {
	if length < 16 {
		length = 16
	}

	salt := make([]byte, length)
	if _, err := cryptorand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateSaltString generates a random salt as base64 string.
func GenerateSaltString(length int) (string, error) {
	salt, err := GenerateSalt(length)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// Random utilities

// GenerateRandomBytes generates cryptographically secure random bytes.
func GenerateRandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}

	bytes := make([]byte, length)
	if _, err := cryptorand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// GenerateRandomInt generates a random integer between min and max (inclusive).
func GenerateRandomInt(min, max int) (int, error) {
	if min > max {
		return 0, fmt.Errorf("min cannot be greater than max")
	}

	// Calculate range size
	rangeSize := max - min + 1

	// Generate random number
	nBig, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(rangeSize)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random int: %w", err)
	}

	return min + int(nBig.Int64()), nil
}

// GenerateRandomFloat generates a random float between 0.0 and 1.0.
func GenerateRandomFloat() (float64, error) {
	bytes, err := GenerateRandomBytes(8)
	if err != nil {
		return 0, err
	}

	// Convert bytes to uint64, then divide by 2^64
	n := uint64(bytes[0])<<56 | uint64(bytes[1])<<48 | uint64(bytes[2])<<40 | uint64(bytes[3])<<32 |
		uint64(bytes[4])<<24 | uint64(bytes[5])<<16 | uint64(bytes[6])<<8 | uint64(bytes[7])

	return float64(n) / float64(1<<63), nil
}

// Hash utilities for files

// HashFile calculates hash of file content.
func HashFile(filePath string, hashType string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	switch strings.ToLower(hashType) {
	case "md5":
		return HashMD5(string(data)), nil
	case "sha1":
		return HashSHA1(string(data)), nil
	case "sha256":
		return HashSHA256(string(data)), nil
	case "sha512":
		return HashSHA512(string(data)), nil
	default:
		return "", fmt.Errorf("unsupported hash type: %s", hashType)
	}
}

// Checksum utilities

// CalculateChecksum calculates checksum of data.
func CalculateChecksum(data []byte) uint32 {
	var checksum uint32 = 0
	for _, b := range data {
		checksum += uint32(b)
	}
	return checksum
}

// VerifyChecksum verifies checksum of data.
func VerifyChecksum(data []byte, expectedChecksum uint32) bool {
	return CalculateChecksum(data) == expectedChecksum
}

// XOR utilities

// XOREncrypt performs XOR encryption/decryption.
func XOREncrypt(data, key []byte) []byte {
	if len(key) == 0 {
		return data
	}

	result := make([]byte, len(data))
	keyLen := len(key)

	for i, b := range data {
		result[i] = b ^ key[i%keyLen]
	}

	return result
}

// CaesarCipher performs Caesar cipher encryption/decryption.
func CaesarCipher(text string, shift int) string {
	result := make([]rune, len(text))

	for i, char := range text {
		if char >= 'a' && char <= 'z' {
			result[i] = 'a' + (char-'a'+rune(shift))%26
		} else if char >= 'A' && char <= 'Z' {
			result[i] = 'A' + (char-'A'+rune(shift))%26
		} else {
			result[i] = char
		}
	}

	return string(result)
}

// Base64 utilities

// Base64Encode encodes data to base64.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes base64 data.
func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

// Base64URLEncode encodes data to base64 URL.
func Base64URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

// Base64URLDecode decodes base64 URL data.
func Base64URLDecode(data string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(data)
}

// Hex utilities

// HexEncode encodes data to hexadecimal.
func HexEncode(data []byte) string {
	return hex.EncodeToString(data)
}

// HexDecode decodes hexadecimal string.
func HexDecode(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}

// Secure comparison utilities

// SecureCompare performs constant-time comparison to prevent timing attacks.
func SecureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

// SecureCompareBytes performs constant-time comparison of byte slices.
func SecureCompareBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}
