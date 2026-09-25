package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// CryptoProvider handles encryption/decryption operations
type CryptoProvider struct {
	keyHex string
}

// NewCryptoProvider creates a new crypto provider from hex key
func NewCryptoProvider(keyHex string) (*CryptoProvider, error) {
	if len(keyHex) != 64 { // 32 bytes = 64 hex chars
		return nil, errors.New("encryption key must be 64 hex characters (32 bytes)")
	}

	// Validate hex
	if _, err := hex.DecodeString(keyHex); err != nil {
		return nil, fmt.Errorf("invalid hex encoding for encryption key: %v", err)
	}

	return &CryptoProvider{keyHex: keyHex}, nil
}

// EncryptChaCha20Poly1305 encrypts data using ChaCha20-Poly1305
// Returns: nonce (12 bytes) + ciphertext + tag (all concatenated)
func (cp *CryptoProvider) EncryptChaCha20Poly1305(plaintext []byte) ([]byte, error) {
	key, err := hex.DecodeString(cp.keyHex)
	if err != nil {
		return nil, err
	}

	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	// Generate random nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := cipher.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptChaCha20Poly1305 decrypts data encrypted with EncryptChaCha20Poly1305
func (cp *CryptoProvider) DecryptChaCha20Poly1305(encrypted []byte) ([]byte, error) {
	if len(encrypted) < 12 {
		return nil, errors.New("encrypted data too short (need at least 12 bytes for nonce)")
	}

	key, err := hex.DecodeString(cp.keyHex)
	if err != nil {
		return nil, err
	}

	cipher, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}

	// Extract nonce and ciphertext
	nonce := encrypted[:12]
	ciphertext := encrypted[12:]

	// Decrypt
	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (authentication tag mismatch): %v", err)
	}

	return plaintext, nil
}

// EncryptRC4 - fallback to RC4 for backward compatibility
func (cp *CryptoProvider) EncryptRC4(plaintext []byte) ([]byte, error) {
	// This is deprecated but kept for agent compatibility
	// Agents built with older version will still work
	return plaintext, nil // Placeholder - use legacy RC4 from transport.go if needed
}
