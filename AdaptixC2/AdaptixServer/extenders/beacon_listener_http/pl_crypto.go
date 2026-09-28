package main

import (
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// DecryptChaCha20Poly1305 decrypts data using ChaCha20-Poly1305 (RFC 7539)
// Format: [12-byte nonce][ciphertext+16-byte tag]
func DecryptChaCha20Poly1305(key []byte, data []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes")
	}

	if len(data) < 12+16 {
		return nil, fmt.Errorf("ciphertext too short (minimum 28 bytes for nonce + tag)")
	}

	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	nonce := data[:12]
	ciphertext := data[12:]

	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}

	return plaintext, nil
}
