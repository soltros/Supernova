package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

var ErrSubsonicCredentialKeyUnavailable = errors.New("SUBSONIC_CREDENTIAL_KEY must contain at least 32 characters")

// EncryptPassword encrypts plaintext using AES-GCM and returns a hex-encoded ciphertext.
// The key parameter may be any length; we derive a 32-byte AES-256 key with SHA-256.
func EncryptPassword(plaintext string, key []byte) (string, error) {
	derived := sha256.Sum256(key) // ensure 32 bytes
	block, err := aes.NewCipher(derived[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ct), nil
}

// DecryptPassword decodes the hex-encoded ciphertext and decrypts it using AES-GCM.
func DecryptPassword(encHex string, key []byte) (string, error) {
	data, err := hex.DecodeString(encHex)
	if err != nil {
		return "", err
	}
	derived := sha256.Sum256(key)
	block, err := aes.NewCipher(derived[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}


// SubsonicCredentialKey returns the dedicated reversible-credential key.
// JWT_SECRET is intentionally not used for new encryption so JWT rotation does
// not invalidate Subsonic token/salt authentication.
func SubsonicCredentialKey() ([]byte, error) {
	key := []byte(os.Getenv("SUBSONIC_CREDENTIAL_KEY"))
	if len(key) < 32 {
		return nil, ErrSubsonicCredentialKeyUnavailable
	}
	return key, nil
}

func EncryptSubsonicPassword(plaintext string) (string, error) {
	key, err := SubsonicCredentialKey()
	if err != nil {
		return "", err
	}
	return EncryptPassword(plaintext, key)
}

// DecryptSubsonicPassword first tries the dedicated key. For upgrades, it then
// tries the historical JWT_SECRET derivation and reports legacy=true so callers
// can re-encrypt with SUBSONIC_CREDENTIAL_KEY when it is available.
func DecryptSubsonicPassword(encHex string) (plaintext string, legacy bool, err error) {
	if key, keyErr := SubsonicCredentialKey(); keyErr == nil {
		if plain, decryptErr := DecryptPassword(encHex, key); decryptErr == nil {
			return plain, false, nil
		}
	}

	legacyKey := []byte(os.Getenv("JWT_SECRET"))
	if len(legacyKey) >= 32 {
		if plain, decryptErr := DecryptPassword(encHex, legacyKey); decryptErr == nil {
			return plain, true, nil
		}
	}
	return "", false, fmt.Errorf("unable to decrypt stored Subsonic credential")
}
