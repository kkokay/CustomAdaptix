package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// ChaCha20Poly1305 + HMAC-SHA256 for auth
type CryptoProvider struct {
	key      []byte
	hmacKey  []byte
}

func NewCryptoProvider(keyHex string) (*CryptoProvider, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("key must be 64 hex chars (32 bytes)")
	}

	c := &CryptoProvider{
		key:     key,
		hmacKey: key, // Use same key for HMAC
	}
	return c, nil
}

// Decrypt: [nonce(12)][ciphertext][tag(16)][hmac(32)]
func (c *CryptoProvider) Decrypt(data []byte) ([]byte, error) {
	if len(data) < 12+16+32 {
		return nil, fmt.Errorf("data too short")
	}

	// Verify HMAC
	expectedHmac := data[len(data)-32:]
	dataToVerify := data[:len(data)-32]
	actualHmac := hmac.New(sha256.New, c.hmacKey)
	actualHmac.Write(dataToVerify)
	if !hmac.Equal(actualHmac.Sum(nil), expectedHmac) {
		return nil, fmt.Errorf("HMAC verification failed")
	}

	// Extract nonce and ciphertext
	nonce := data[:12]
	ciphertext := data[12 : len(data)-32]

	cipher, _ := chacha20poly1305.New(c.key)
	plaintext, err := cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}
	return plaintext, nil
}

// Encrypt: [nonce(12)][ciphertext][tag(16)][hmac(32)]
func (c *CryptoProvider) Encrypt(plaintext, nonce []byte) ([]byte, error) {
	if len(nonce) != 12 {
		return nil, fmt.Errorf("nonce must be 12 bytes")
	}

	cipher, _ := chacha20poly1305.New(c.key)
	ciphertext := cipher.Seal(nil, nonce, plaintext, nil)

	// Add HMAC
	result := append(nonce, ciphertext...)
	h := hmac.New(sha256.New, c.hmacKey)
	h.Write(result)
	result = append(result, h.Sum(nil)...)

	return result, nil
}
