package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePasswordHash(t *testing.T) {
	password := "testPassword123!"

	// Test password hashing
	hashedPassword, err := GeneratePasswordHash(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)
	assert.NotEqual(t, password, hashedPassword)

	// Test that hashing the same password produces different results (due to salt)
	hashedPassword2, err := GeneratePasswordHash(password)
	assert.NoError(t, err)
	assert.NotEqual(t, hashedPassword, hashedPassword2)

	// Test empty password (bcrypt can hash empty passwords, so this should succeed)
	emptyHash, err := GeneratePasswordHash("")
	assert.NoError(t, err)
	assert.NotEmpty(t, emptyHash)
}

func TestVerifyCryptoPassword(t *testing.T) {
	password := "testPassword123!"

	// Hash password
	hashedPassword, err := GeneratePasswordHash(password)
	assert.NoError(t, err)

	// Test correct password
	isValid := VerifyCryptoPassword(password, hashedPassword)
	assert.True(t, isValid)

	// Test incorrect password
	isValid = VerifyCryptoPassword("wrongPassword", hashedPassword)
	assert.False(t, isValid)

	// Test empty password
	isValid = VerifyCryptoPassword("", hashedPassword)
	assert.False(t, isValid)

	// Test empty hash
	isValid = VerifyCryptoPassword(password, "")
	assert.False(t, isValid)
}

func TestGenerateToken(t *testing.T) {
	// Test default length (should be at least 16)
	token, err := GenerateToken(0)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, len(token) >= 16)

	// Test custom length
	customToken, err := GenerateToken(32)
	assert.NoError(t, err)
	assert.NotEmpty(t, customToken)

	// Test that tokens are different
	token2, err := GenerateToken(32)
	assert.NoError(t, err)
	assert.NotEqual(t, token, token2)
}

func TestGenerateSecureCryptoToken(t *testing.T) {
	// Test default length (should be at least 32)
	token, err := GenerateSecureCryptoToken(0)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, len(token) >= 32)

	// Test custom length
	customToken, err := GenerateSecureCryptoToken(16)
	assert.NoError(t, err)
	assert.NotEmpty(t, customToken)

	// Test that tokens are different
	token2, err := GenerateSecureCryptoToken(32)
	assert.NoError(t, err)
	assert.NotEqual(t, token, token2)
}

func TestAESEncryptDecrypt(t *testing.T) {
	plaintext := "This is a secret message"
	key := "this-is-a-32-byte-key-1234567890" // 32 bytes

	// Test encryption
	ciphertext, err := AESEncrypt(plaintext, key)
	assert.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEqual(t, plaintext, ciphertext)

	// Test decryption
	decryptedText, err := AESDecrypt(ciphertext, key)
	assert.NoError(t, err)
	assert.Equal(t, plaintext, decryptedText)

	// Test with different key
	wrongKey := "this-is-a-different-32-byte-key-12"
	_, err = AESDecrypt(ciphertext, wrongKey)
	assert.Error(t, err)

	// Test empty plaintext
	emptyCipher, err := AESEncrypt("", key)
	assert.NoError(t, err)
	emptyPlain, err := AESDecrypt(emptyCipher, key)
	assert.NoError(t, err)
	assert.Equal(t, "", emptyPlain)

	// Test invalid key length
	shortKey := "short"
	_, err = AESEncrypt(plaintext, shortKey)
	assert.Error(t, err)
}

func TestHMACFunctions(t *testing.T) {
	data := "message to sign"
	key := "secret-key"

	// Test HMAC-SHA256
	signature := HMACSHA256(data, key)
	assert.NotEmpty(t, signature)

	// Test that same data and key produce same signature
	signature2 := HMACSHA256(data, key)
	assert.Equal(t, signature, signature2)

	// Test different data produces different signature
	signature3 := HMACSHA256("different message", key)
	assert.NotEqual(t, signature, signature3)

	// Test different key produces different signature
	signature4 := HMACSHA256(data, "different-key")
	assert.NotEqual(t, signature, signature4)

	// Test empty data
	emptySignature := HMACSHA256("", key)
	assert.NotEmpty(t, emptySignature)

	// Test empty key
	emptyKeySignature := HMACSHA256(data, "")
	assert.NotEmpty(t, emptyKeySignature)

	// Test other HMAC functions
	md5Signature := HMACMD5(data, key)
	sha1Signature := HMACSHA1(data, key)
	sha512Signature := HMACSHA512(data, key)

	assert.NotEqual(t, signature, md5Signature)
	assert.NotEqual(t, signature, sha1Signature)
	assert.NotEqual(t, signature, sha512Signature)
}

func TestGenerateAPIToken(t *testing.T) {
	// Test API key generation
	apiKey, err := GenerateAPIToken("sk", 32)
	assert.NoError(t, err)
	assert.NotEmpty(t, apiKey)
	assert.True(t, len(apiKey) >= 35, "API key should be at least 35 characters (prefix + token)")
	assert.True(t, strings.HasPrefix(apiKey, "sk_"), "API key should start with 'sk_'")

	// Test custom length
	customKey, err := GenerateAPIToken("pk", 16)
	assert.NoError(t, err)
	assert.NotEmpty(t, customKey)
	assert.True(t, strings.HasPrefix(customKey, "pk_"), "API key should start with 'pk_'")

	// Test that API keys are unique
	apiKey2, err := GenerateAPIToken("sk", 32)
	assert.NoError(t, err)
	assert.NotEqual(t, apiKey, apiKey2)
}

func TestGenerateStrongPassword(t *testing.T) {
	// Test password generation
	password, err := GenerateStrongPassword(12)
	assert.NoError(t, err)
	assert.Len(t, password, 12)

	// Test that generated password meets strength requirements
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	assert.True(t, hasUpper, "Password should contain uppercase letter")
	assert.True(t, hasLower, "Password should contain lowercase letter")
	assert.True(t, hasNumber, "Password should contain number")
	assert.True(t, hasSpecial, "Password should contain special character")

	// Test minimum length
	_, err = GenerateStrongPassword(6)
	assert.Error(t, err)
}

func TestGeneratePIN(t *testing.T) {
	// Test PIN generation
	pin := GeneratePIN(6)
	assert.Len(t, pin, 6)
	assert.Regexp(t, `^\d{6}$`, pin)

	// Test default length
	defaultPin := GeneratePIN(0)
	assert.Len(t, defaultPin, 4)

	// Test that PINs are different
	pin2 := GeneratePIN(6)
	assert.NotEqual(t, pin, pin2)
}

func TestBase64Encode(t *testing.T) {
	data := []byte("Hello, World!")

	// Test base64 encoding
	encoded := Base64Encode(data)
	assert.NotEmpty(t, encoded)
	assert.NotEqual(t, string(data), encoded)

	// Test known value
	knownData := []byte("test")
	knownEncoded := Base64Encode(knownData)
	assert.Equal(t, "dGVzdA==", knownEncoded)

	// Test empty data
	emptyEncoded := Base64Encode([]byte{})
	assert.Equal(t, "", emptyEncoded)
}

func TestBase64Decode(t *testing.T) {
	// Test decoding
	encoded := "SGVsbG8sIFdvcmxkIQ=="
	decoded, err := Base64Decode(encoded)
	assert.NoError(t, err)
	assert.Equal(t, "Hello, World!", string(decoded))

	// Test known value
	knownEncoded := "dGVzdA=="
	knownDecoded, err := Base64Decode(knownEncoded)
	assert.NoError(t, err)
	assert.Equal(t, "test", string(knownDecoded))

	// Test invalid base64
	_, err = Base64Decode("invalid-base64")
	assert.Error(t, err)

	// Test empty string
	decoded, err = Base64Decode("")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(decoded))
}

func TestHashFunctions(t *testing.T) {
	data := "test data"

	// Test MD5
	md5Hash := HashMD5(data)
	assert.Len(t, md5Hash, 32)
	assert.Equal(t, "eb733a00c0c9d336e65691a37ab54293", md5Hash)

	// Test SHA1
	sha1Hash := HashSHA1(data)
	assert.Len(t, sha1Hash, 40)

	// Test SHA256
	sha256Hash := HashSHA256(data)
	assert.Len(t, sha256Hash, 64)

	// Test SHA512
	sha512Hash := HashSHA512(data)
	assert.Len(t, sha512Hash, 128)

	// Test that different hashes produce different results
	assert.NotEqual(t, md5Hash, sha1Hash)
	assert.NotEqual(t, sha1Hash, sha256Hash)
	assert.NotEqual(t, sha256Hash, sha512Hash)
}

func TestSecureCompare(t *testing.T) {
	// Test equal strings
	assert.True(t, SecureCompare("hello", "hello"))

	// Test different strings
	assert.False(t, SecureCompare("hello", "world"))

	// Test different lengths
	assert.False(t, SecureCompare("hello", "hello!"))

	// Test empty strings
	assert.True(t, SecureCompare("", ""))
	assert.False(t, SecureCompare("", "not empty"))
}

func TestGenerateRandomBytes(t *testing.T) {
	// Test random bytes generation
	bytes, err := GenerateRandomBytes(16)
	assert.NoError(t, err)
	assert.Len(t, bytes, 16)

	// Test that multiple calls produce different results
	bytes2, err := GenerateRandomBytes(16)
	assert.NoError(t, err)
	assert.NotEqual(t, bytes, bytes2)

	// Test zero length
	zeroBytes, err := GenerateRandomBytes(0)
	assert.NoError(t, err)
	assert.Nil(t, zeroBytes)

	// Test negative length
	negBytes, err := GenerateRandomBytes(-1)
	assert.NoError(t, err)
	assert.Nil(t, negBytes)
}

func TestGenerateRandomInt(t *testing.T) {
	// Test random int generation
	num, err := GenerateRandomInt(1, 10)
	assert.NoError(t, err)
	assert.True(t, num >= 1 && num <= 10)

	// Test multiple calls produce varied results
	results := make(map[int]bool)
	for i := 0; i < 100; i++ {
		num, err := GenerateRandomInt(1, 5)
		assert.NoError(t, err)
		assert.True(t, num >= 1 && num <= 5)
		results[num] = true
	}

	// Should get at least some different values
	assert.True(t, len(results) > 1)

	// Test single value range
	single, err := GenerateRandomInt(5, 5)
	assert.NoError(t, err)
	assert.Equal(t, 5, single)

	// Test invalid range
	_, err = GenerateRandomInt(10, 1)
	assert.Error(t, err)
}

func TestGenerateSalt(t *testing.T) {
	// Test salt generation
	salt, err := GenerateSalt(16)
	assert.NoError(t, err)
	assert.Len(t, salt, 16)

	// Test default length
	defaultSalt, err := GenerateSalt(0)
	assert.NoError(t, err)
	assert.Len(t, defaultSalt, 16)

	// Test that multiple calls produce different results
	salt2, err := GenerateSalt(16)
	assert.NoError(t, err)
	assert.NotEqual(t, salt, salt2)

	// Test salt string generation
	saltStr, err := GenerateSaltString(16)
	assert.NoError(t, err)
	assert.NotEmpty(t, saltStr)

	// Test that string salt is base64 encoded
	_, err = Base64Decode(saltStr)
	assert.NoError(t, err)
}

func TestXOREncrypt(t *testing.T) {
	data := []byte("Hello, World!")
	key := []byte("secret")

	// Test encryption
	encrypted := XOREncrypt(data, key)
	assert.NotEqual(t, data, encrypted)

	// Test decryption (XOR is symmetric)
	decrypted := XOREncrypt(encrypted, key)
	assert.Equal(t, data, decrypted)

	// Test empty key
	emptyKey := XOREncrypt(data, []byte{})
	assert.Equal(t, data, emptyKey)

	// Test empty data
	emptyData := XOREncrypt([]byte{}, key)
	assert.Equal(t, []byte{}, emptyData)
}

func TestCaesarCipher(t *testing.T) {
	// Test encryption
	plaintext := "Hello, World!"
	encrypted := CaesarCipher(plaintext, 3)
	assert.Equal(t, "Khoor, Zruog!", encrypted)

	// Test decryption (negative shift)
	decrypted := CaesarCipher(encrypted, -3)
	assert.Equal(t, plaintext, decrypted)

	// Test with larger shift
	encrypted2 := CaesarCipher(plaintext, 29) // 29 = 3 (mod 26)
	assert.Equal(t, encrypted, encrypted2)

	// Test empty string
	empty := CaesarCipher("", 5)
	assert.Equal(t, "", empty)

	// Test numbers and special characters (should remain unchanged)
	mixed := CaesarCipher("ABC123!@#", 2)
	assert.Equal(t, "CDE123!@#", mixed)
}
