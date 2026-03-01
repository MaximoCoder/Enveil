package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. These values follow the recommended standard
// for interactive use: costly enough for attackers,
// fast enough for users (~300ms on modern hardware)
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // 64MB
	argonThreads = 4
	argonKeyLen  = 32
)

// DeriveKey takes a password and a salt and produces a 32-byte master key
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)
}

// GenerateSalt generates a random 16-byte salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("error generating salt: %w", err)
	}
	return salt, nil
}

// KeyToHex converts the master key to a hexadecimal string
// which is the format SQLCipher expects
func KeyToHex(key []byte) string {
	return hex.EncodeToString(key)
}

// DeriveTransportKey derives a 32-byte AES key from the API key
// This key is used to encrypt values in transit
func DeriveTransportKey(apiKey string) []byte {
    hash := sha256.Sum256([]byte(apiKey))
    return hash[:]
}

// Encrypt encrypts a plaintext value with AES-GCM
func Encrypt(plaintext string, key []byte) (string, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("error creating cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("error creating GCM: %w", err)
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", fmt.Errorf("error generating nonce: %w", err)
    }

    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a value encrypted with Encrypt
func Decrypt(ciphertextHex string, key []byte) (string, error) {
    ciphertext, err := hex.DecodeString(ciphertextHex)
    if err != nil {
        return "", fmt.Errorf("error decoding ciphertext: %w", err)
    }

    block, err := aes.NewCipher(key)
    if err != nil {
        return "", fmt.Errorf("error creating cipher: %w", err)
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", fmt.Errorf("error creating GCM: %w", err)
    }

    if len(ciphertext) < gcm.NonceSize() {
        return "", fmt.Errorf("ciphertext too short")
    }

    nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", fmt.Errorf("error decrypting: %w", err)
    }

    return string(plaintext), nil
}