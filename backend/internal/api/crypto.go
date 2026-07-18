package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
)

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
