#pragma once

#include <stdint.h>

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
