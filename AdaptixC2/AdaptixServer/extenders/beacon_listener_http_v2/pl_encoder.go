package main

import "encoding/hex"
import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// EncodingType defines supported encoding methods
type EncodingType string

const (
	EncodingBase64   EncodingType = "base64"
	EncodingJSON     EncodingType = "json"
	EncodingBinary   EncodingType = "binary"
	EncodingHex      EncodingType = "hex"
)

// EncoderDecoder handles different encoding/decoding formats
type EncoderDecoder struct {
	encodingType EncodingType
	crypto       *CryptoProvider
}

// NewEncoderDecoder creates encoder/decoder for specified format
func NewEncoderDecoder(encoding string, crypto *CryptoProvider) (*EncoderDecoder, error) {
	encType := EncodingType(strings.ToLower(encoding))

	switch encType {
	case EncodingBase64, EncodingJSON, EncodingBinary, EncodingHex:
		return &EncoderDecoder{
			encodingType: encType,
			crypto:       crypto,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported encoding type: %s", encoding)
	}
}

// Encode encodes plaintext to specified format
func (ed *EncoderDecoder) Encode(plaintext []byte) ([]byte, error) {
	switch ed.encodingType {
	case EncodingBase64:
		return []byte(base64.StdEncoding.EncodeToString(plaintext)), nil

	case EncodingJSON:
		payload := map[string]interface{}{
			"data": base64.StdEncoding.EncodeToString(plaintext),
		}
		return json.Marshal(payload)

	case EncodingHex:
		return []byte(fmt.Sprintf("%x", plaintext)), nil

	case EncodingBinary:
		return plaintext, nil

	default:
		return nil, errors.New("unknown encoding type")
	}
}

// Decode decodes from specified format to plaintext
func (ed *EncoderDecoder) Decode(encoded []byte) ([]byte, error) {
	switch ed.encodingType {
	case EncodingBase64:
		return base64.StdEncoding.DecodeString(string(encoded))

	case EncodingJSON:
		var payload map[string]interface{}
		if err := json.Unmarshal(encoded, &payload); err != nil {
			return nil, fmt.Errorf("json decode error: %v", err)
		}

		dataStr, ok := payload["data"].(string)
		if !ok {
			return nil, errors.New("json payload missing or invalid 'data' field")
		}

		return base64.StdEncoding.DecodeString(dataStr)

	case EncodingHex:
		var hexStr string
		if err := json.Unmarshal(encoded, &hexStr); err != nil {
			hexStr = string(encoded)
		}

		// Remove spaces and decode
		hexStr = strings.ReplaceAll(hexStr, " ", "")
		return hex.DecodeString(hexStr)

	case EncodingBinary:
		return encoded, nil

	default:
		return nil, errors.New("unknown encoding type")
	}
}

// EncodeWithEncryption encrypts and encodes data
func (ed *EncoderDecoder) EncodeWithEncryption(plaintext []byte) ([]byte, error) {
	// Encrypt using ChaCha20-Poly1305
	encrypted, err := ed.crypto.EncryptChaCha20Poly1305(plaintext)
	if err != nil {
		return nil, err
	}

	// Then encode
	return ed.Encode(encrypted)
}

// DecodeWithDecryption decodes and decrypts data
func (ed *EncoderDecoder) DecodeWithDecryption(encoded []byte) ([]byte, error) {
	// First decode
	encrypted, err := ed.Decode(encoded)
	if err != nil {
		return nil, err
	}

	// Then decrypt
	return ed.crypto.DecryptChaCha20Poly1305(encrypted)
}
