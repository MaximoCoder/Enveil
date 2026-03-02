package crypto

import (
	"testing"
)

// --- DeriveKey ---

func TestDeriveKeyProducesDeterministicOutput(t *testing.T) {
	salt := []byte("1234567890123456")
	key1 := DeriveKey("mypassword", salt)
	key2 := DeriveKey("mypassword", salt)

	if string(key1) != string(key2) {
		t.Fatal("DeriveKey should produce the same key for the same password and salt")
	}
}

func TestDeriveKeyDiffersWithDifferentPassword(t *testing.T) {
	salt := []byte("1234567890123456")
	key1 := DeriveKey("password1", salt)
	key2 := DeriveKey("password2", salt)

	if string(key1) == string(key2) {
		t.Fatal("DeriveKey should produce different keys for different passwords")
	}
}

func TestDeriveKeyDiffersWithDifferentSalt(t *testing.T) {
	key1 := DeriveKey("mypassword", []byte("salt1salt1salt1s"))
	key2 := DeriveKey("mypassword", []byte("salt2salt2salt2s"))

	if string(key1) == string(key2) {
		t.Fatal("DeriveKey should produce different keys for different salts")
	}
}

func TestDeriveKeyLength(t *testing.T) {
	salt := []byte("1234567890123456")
	key := DeriveKey("mypassword", salt)

	if len(key) != 32 {
		t.Fatalf("expected key length 32, got %d", len(key))
	}
}

// --- GenerateSalt ---

func TestGenerateSaltLength(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(salt) != 16 {
		t.Fatalf("expected salt length 16, got %d", len(salt))
	}
}

func TestGenerateSaltIsRandom(t *testing.T) {
	salt1, _ := GenerateSalt()
	salt2, _ := GenerateSalt()

	if string(salt1) == string(salt2) {
		t.Fatal("GenerateSalt should produce different values each time")
	}
}

// --- Encrypt / Decrypt ---

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	plaintext := "super-secret-value"

	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptProducesDifferentCiphertextEachTime(t *testing.T) {
	key := make([]byte, 32)
	plaintext := "same-value"

	c1, _ := Encrypt(plaintext, key)
	c2, _ := Encrypt(plaintext, key)

	if c1 == c2 {
		t.Fatal("Encrypt should produce different ciphertext each time due to random nonce")
	}
}

func TestDecryptFailsWithWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF

	encrypted, err := Encrypt("secret", key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Fatal("Decrypt should fail with a wrong key")
	}
}

func TestDecryptFailsWithTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)

	encrypted, err := Encrypt("secret", key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Flip the last character of the hex string
	tampered := encrypted[:len(encrypted)-1] + "x"
	_, err = Decrypt(tampered, key)
	if err == nil {
		t.Fatal("Decrypt should fail with tampered ciphertext")
	}
}

func TestDecryptFailsWithEmptyInput(t *testing.T) {
	key := make([]byte, 32)
	_, err := Decrypt("", key)
	if err == nil {
		t.Fatal("Decrypt should fail with empty input")
	}
}

// --- DeriveTransportKey ---

func TestDeriveTransportKeyLength(t *testing.T) {
	key := DeriveTransportKey("my-api-key")
	if len(key) != 32 {
		t.Fatalf("expected transport key length 32, got %d", len(key))
	}
}

func TestDeriveTransportKeyIsDeterministic(t *testing.T) {
	k1 := DeriveTransportKey("my-api-key")
	k2 := DeriveTransportKey("my-api-key")

	if string(k1) != string(k2) {
		t.Fatal("DeriveTransportKey should be deterministic")
	}
}

func TestDeriveTransportKeyDiffersForDifferentKeys(t *testing.T) {
	k1 := DeriveTransportKey("api-key-1")
	k2 := DeriveTransportKey("api-key-2")

	if string(k1) == string(k2) {
		t.Fatal("DeriveTransportKey should produce different keys for different API keys")
	}
}