#pragma once

#include <stdint.h>
#include <string.h>
#include "hmac_sha256.h"

// ChaCha20 Stream Cipher Implementation
// RFC 7539: https://tools.ietf.org/html/rfc7539
//
// ChaCha20 is a 256-bit stream cipher that's:
// - Fast (~3x faster than AES on non-AES-NI CPUs)
// - Secure (IETF standard, no known practical attacks)
// - Simple to implement (pure arithmetic operations)
// - Patent-free
//
// Usage:
// 1. Call ChaCha20_Init() with key and nonce
// 2. Call ChaCha20_Encrypt() for each chunk of data
// 3. Counter automatically increments

typedef struct {
    uint32_t state[16];         // ChaCha20 state (64 bytes)
    uint8_t  keystream[64];     // Current keystream block
    uint32_t keystream_index;   // Current position in keystream
} ChaCha20_Context;

typedef enum {
    CHACHA20_MODE_COUNTER = 0,  // 64-bit nonce (counter in state[12-13])
    CHACHA20_MODE_IETF = 1      // 96-bit nonce (counter in state[12] only)
} ChaCha20_Mode;

// ============================================================================
// PUBLIC API
// ============================================================================

/**
 * Initialize ChaCha20 context with key and nonce
 *
 * @param ctx ChaCha20 context (must be allocated by caller)
 * @param key 32-byte encryption key (256-bit)
 * @param nonce 8-byte or 12-byte nonce depending on mode
 * @param mode CHACHA20_MODE_COUNTER (8-byte nonce) or CHACHA20_MODE_IETF (12-byte nonce)
 */
void ChaCha20_Init(
    ChaCha20_Context *ctx,
    const uint8_t *key,     // 32 bytes
    const uint8_t *nonce,   // 8 or 12 bytes
    ChaCha20_Mode mode
);

/**
 * Encrypt plaintext using ChaCha20
 * Decryption is identical (stream cipher property)
 *
 * @param ctx Initialized ChaCha20 context
 * @param plaintext Input data to encrypt (can be any length)
 * @param plaintext_len Length of input
 * @param ciphertext Output buffer (must be >= plaintext_len)
 *
 * Note: ctx->state is modified (counter incremented)
 */
void ChaCha20_Encrypt(
    ChaCha20_Context *ctx,
    const uint8_t *plaintext,
    uint32_t plaintext_len,
    uint8_t *ciphertext
);

/**
 * Decrypt ciphertext using ChaCha20
 * This is identical to encryption (stream cipher property)
 *
 * @param ctx Initialized ChaCha20 context
 * @param ciphertext Input encrypted data
 * @param ciphertext_len Length of input
 * @param plaintext Output buffer (must be >= ciphertext_len)
 */
void ChaCha20_Decrypt(
    ChaCha20_Context *ctx,
    const uint8_t *ciphertext,
    uint32_t ciphertext_len,
    uint8_t *plaintext
);

/**
 * Helper: Encrypt/decrypt data in-place
 * Modifies the input buffer directly
 *
 * @param ctx Initialized ChaCha20 context
 * @param data Data to encrypt/decrypt (modified in-place)
 * @param len Length of data
 */
void ChaCha20_Crypt(ChaCha20_Context *ctx, uint8_t *data, uint32_t len);

/**
 * Helper: Zero out sensitive data
 * @param ptr Pointer to data
 * @param len Length in bytes
 */
void ChaCha20_Zero(volatile void *ptr, size_t len);

// ============================================================================
// BEACON AGENT WRAPPER FUNCTIONS
// ============================================================================

// Simple wrappers for beacon agent compatibility
inline void EncryptChaCha20(uint8_t *plainData, uint32_t plainSize, uint8_t *key, uint8_t *nonce) {
    ChaCha20_Context ctx;
    ChaCha20_Init(&ctx, key, nonce, CHACHA20_MODE_IETF);
    ChaCha20_Encrypt(&ctx, plainData, plainSize, plainData);
    ChaCha20_Zero(&ctx, sizeof(ctx));
}

inline void DecryptChaCha20(uint8_t *cipherData, uint32_t cipherSize, uint8_t *key, uint8_t *nonce) {
    ChaCha20_Context ctx;
    ChaCha20_Init(&ctx, key, nonce, CHACHA20_MODE_IETF);
    ChaCha20_Decrypt(&ctx, cipherData, cipherSize, cipherData);
    ChaCha20_Zero(&ctx, sizeof(ctx));
}

// ============================================================================
// HMAC-SHA256 AUTHENTICATED ENCRYPTION
// ============================================================================

// Encrypt with ChaCha20 and append HMAC-SHA256
// Output format: [nonce(12)][ciphertext][tag(16)][hmac(32)]
// Total: 12 + plainSize + 16 + 32 = plainSize + 60 bytes
//
// Returns: 0 on success, -1 on error
inline int EncryptChaCha20WithHmac(
    const uint8_t *plainData, uint32_t plainSize,
    const uint8_t *key, uint32_t keySize,
    const uint8_t *nonce, uint32_t nonceSize,
    const uint8_t *hmacKey, uint32_t hmacKeySize,
    uint8_t *output, uint32_t *outputSize)
{
    if (!plainData || !key || !nonce || !hmacKey || !output || !outputSize) {
        return -1;
    }

    // Minimum output size: nonce(12) + plaintext + tag(16) + hmac(32)
    uint32_t requiredSize = nonceSize + plainSize + 16 + 32;
    if (*outputSize < requiredSize) {
        return -1;
    }

    uint32_t offset = 0;

    // Write nonce
    memcpy(output, nonce, nonceSize);
    offset += nonceSize;

    // Encrypt plaintext
    ChaCha20_Context ctx;
    ChaCha20_Init(&ctx, key, nonce, CHACHA20_MODE_IETF);
    ChaCha20_Encrypt(&ctx, plainData, plainSize, &output[offset]);
    offset += plainSize;

    // Skip tag position (16 bytes for Poly1305, not computed here)
    offset += 16;

    // Compute HMAC over nonce + ciphertext + tag
    // For BeaconAdvanced, HMAC is computed over: nonce || ciphertext || (empty tag placeholder)
    uint8_t hmac_input[12 + 4096 + 16];  // Max sizes
    uint32_t hmac_input_len = nonceSize + plainSize + 16;
    if (hmac_input_len > sizeof(hmac_input)) {
        ChaCha20_Zero(&ctx, sizeof(ctx));
        return -1;
    }

    memcpy(&hmac_input[0], nonce, nonceSize);
    memcpy(&hmac_input[nonceSize], &output[nonceSize], plainSize);
    memset(&hmac_input[nonceSize + plainSize], 0, 16);  // Tag placeholder

    // Compute HMAC
    hmac_sha256(hmacKey, hmacKeySize, hmac_input, hmac_input_len, &output[offset]);
    offset += 32;

    *outputSize = offset;
    ChaCha20_Zero(&ctx, sizeof(ctx));
    return 0;
}

// Verify HMAC and decrypt ChaCha20
// Input format: [nonce(12)][ciphertext][tag(16)][hmac(32)]
//
// Returns: 0 on success (HMAC verified), -1 on error or verification failure
inline int DecryptChaCha20WithHmac(
    const uint8_t *input, uint32_t inputSize,
    const uint8_t *key, uint32_t keySize,
    const uint8_t *hmacKey, uint32_t hmacKeySize,
    uint8_t *plainData, uint32_t *plainSize)
{
    if (!input || !key || !hmacKey || !plainData || !plainSize) {
        return -1;
    }

    // Minimum input size: nonce(12) + ciphertext(1) + tag(16) + hmac(32) = 61 bytes
    if (inputSize < 61) {
        return -1;
    }

    uint32_t offset = 0;
    uint32_t nonceSize = 12;
    uint32_t cipherSize = inputSize - nonceSize - 16 - 32;

    // Extract nonce
    const uint8_t *nonce = input;
    offset += nonceSize;

    // Extract ciphertext
    const uint8_t *ciphertext = &input[offset];
    offset += cipherSize;

    // Extract tag (placeholder, not used)
    offset += 16;

    // Extract HMAC
    const uint8_t *expected_hmac = &input[offset];

    // Verify HMAC
    uint8_t hmac_input[12 + 4096 + 16];
    uint32_t hmac_input_len = nonceSize + cipherSize + 16;
    if (hmac_input_len > sizeof(hmac_input)) {
        return -1;
    }

    memcpy(&hmac_input[0], nonce, nonceSize);
    memcpy(&hmac_input[nonceSize], ciphertext, cipherSize);
    memset(&hmac_input[nonceSize + cipherSize], 0, 16);

    if (!hmac_sha256_verify(hmacKey, hmacKeySize, hmac_input, hmac_input_len, expected_hmac)) {
        return -1;  // HMAC verification failed
    }

    // Decrypt
    ChaCha20_Context ctx;
    ChaCha20_Init(&ctx, key, nonce, CHACHA20_MODE_IETF);
    ChaCha20_Decrypt(&ctx, ciphertext, cipherSize, plainData);

    *plainSize = cipherSize;
    ChaCha20_Zero(&ctx, sizeof(ctx));
    return 0;
}

// ============================================================================
// CONSTANTS
// ============================================================================

#define CHACHA20_KEY_SIZE       32  // 256 bits
#define CHACHA20_NONCE_SIZE_64  8   // 64-bit nonce (counter mode)
#define CHACHA20_NONCE_SIZE_96  12  // 96-bit nonce (IETF standard)
#define CHACHA20_BLOCK_SIZE     64  // Keystream block = 64 bytes
#define CHACHA20_STATE_SIZE     16  // State = 16 uint32_t words

// ChaCha20 constants (RFC 7539)
#define CHACHA20_CONSTANT_0 0x61707865UL   // "expa"
#define CHACHA20_CONSTANT_1 0x3320646eUL   // "nd 3"
#define CHACHA20_CONSTANT_2 0x79622d32UL   // "2-by"
#define CHACHA20_CONSTANT_3 0x6b206574UL   // "te k"

// ============================================================================
// INTEGRATION WITH BEACON AGENT
// ============================================================================

// Usage example in ConnectorHTTP.cpp:
//
//   ChaCha20_Context ctx;
//   uint8_t key[32];        // 32-byte key from listener
//   uint8_t nonce[12];      // 12-byte nonce (generate random or use counter)
//
//   // Initialize
//   ChaCha20_Init(&ctx, key, nonce, CHACHA20_MODE_IETF);
//
//   // Encrypt data
//   uint8_t ciphertext[256];
//   ChaCha20_Encrypt(&ctx, plaintext, plaintext_len, ciphertext);
//
//   // Decrypt is identical
//   ChaCha20_Decrypt(&ctx, ciphertext, ciphertext_len, plaintext);
//
//   // Clean up
//   ChaCha20_Zero(&ctx, sizeof(ctx));
