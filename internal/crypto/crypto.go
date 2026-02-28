package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

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