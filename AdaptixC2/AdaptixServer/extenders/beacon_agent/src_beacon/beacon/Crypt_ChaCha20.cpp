#include "Crypt_ChaCha20.h"
#include <string.h>

// ============================================================================
// INTERNAL HELPER FUNCTIONS
// ============================================================================

// Rotate left (used in ChaCha20 quarter round)
static inline uint32_t rotl32(uint32_t x, int n)
{
    return (x << n) | (x >> (32 - n));
}

// Little-endian load/store
static inline uint32_t le32_load(const uint8_t *p)
{
    return ((uint32_t)p[0]        |
            ((uint32_t)p[1] <<  8) |
            ((uint32_t)p[2] << 16) |
            ((uint32_t)p[3] << 24));
}

static inline void le32_store(uint8_t *p, uint32_t x)
{
    p[0] = (uint8_t)(x      );
    p[1] = (uint8_t)(x >>  8);
    p[2] = (uint8_t)(x >> 16);
    p[3] = (uint8_t)(x >> 24);
}

// ============================================================================
// CHACHA20 CORE ALGORITHM
// ============================================================================

// ChaCha20 Quarter Round
// Mixes state[a], state[b], state[c], state[d]
static inline void chacha20_qround(
    uint32_t *state,
    int a, int b, int c, int d)
{
    // a += b; d ^= a; d <<<= 16;
    state[a] += state[b]; state[d] ^= state[a]; state[d] = rotl32(state[d], 16);
    // c += d; b ^= c; b <<<= 12;
    state[c] += state[d]; state[b] ^= state[c]; state[b] = rotl32(state[b], 12);
    // a += b; d ^= a; d <<<= 8;
    state[a] += state[b]; state[d] ^= state[a]; state[d] = rotl32(state[d], 8);
    // c += d; b ^= c; b <<<= 7;
    state[c] += state[d]; state[b] ^= state[c]; state[b] = rotl32(state[b], 7);
}

// Generate a single ChaCha20 block (64 bytes of keystream)
static void chacha20_block(ChaCha20_Context *ctx, uint8_t *output)
{
    uint32_t state[16];
    uint32_t initial_state[16];
    int i;

    // Copy state
    memcpy(state, ctx->state, sizeof(state));
    memcpy(initial_state, ctx->state, sizeof(initial_state));

    // 10 rounds (20 quarter rounds total)
    for (i = 0; i < 10; i++) {
        // Diagonal rounds
        chacha20_qround(state, 0, 4,  8, 12);
        chacha20_qround(state, 1, 5,  9, 13);
        chacha20_qround(state, 2, 6, 10, 14);
        chacha20_qround(state, 3, 7, 11, 15);

        // Column rounds
        chacha20_qround(state, 0, 5, 10, 15);
        chacha20_qround(state, 1, 6, 11, 12);
        chacha20_qround(state, 2, 7,  8, 13);
        chacha20_qround(state, 3, 4,  9, 14);
    }

    // Add initial state to working state (modulo 2^32)
    for (i = 0; i < 16; i++) {
        state[i] += initial_state[i];
        le32_store(output + (i * 4), state[i]);
    }

    // Increment counter (state[12] for IETF, state[12-13] for counter mode)
    if (++(ctx->state[12]) == 0) {
        ++(ctx->state[13]);  // Carry over on overflow
    }
}

// ============================================================================
// PUBLIC API IMPLEMENTATION
// ============================================================================

void ChaCha20_Init(
    ChaCha20_Context *ctx,
    const uint8_t *key,
    const uint8_t *nonce,
    ChaCha20_Mode mode)
{
    // Constants
    ctx->state[0] = CHACHA20_CONSTANT_0;
    ctx->state[1] = CHACHA20_CONSTANT_1;
    ctx->state[2] = CHACHA20_CONSTANT_2;
    ctx->state[3] = CHACHA20_CONSTANT_3;

    // Key (32 bytes = 8 words)
    ctx->state[4]  = le32_load(key + 0);
    ctx->state[5]  = le32_load(key + 4);
    ctx->state[6]  = le32_load(key + 8);
    ctx->state[7]  = le32_load(key + 12);
    ctx->state[8]  = le32_load(key + 16);
    ctx->state[9]  = le32_load(key + 20);
    ctx->state[10] = le32_load(key + 24);
    ctx->state[11] = le32_load(key + 28);

    // Counter and nonce
    if (mode == CHACHA20_MODE_IETF) {
        // IETF: 32-bit counter (state[12]) + 96-bit nonce (state[13-15])
        ctx->state[12] = 0;
        ctx->state[13] = le32_load(nonce + 0);
        ctx->state[14] = le32_load(nonce + 4);
        ctx->state[15] = le32_load(nonce + 8);
    } else {
        // Counter mode: 64-bit counter (state[12-13]) + 64-bit nonce (state[14-15])
        ctx->state[12] = 0;
        ctx->state[13] = 0;
        ctx->state[14] = le32_load(nonce + 0);
        ctx->state[15] = le32_load(nonce + 4);
    }

    ctx->keystream_index = CHACHA20_BLOCK_SIZE;  // Force initial block generation
}

void ChaCha20_Encrypt(
    ChaCha20_Context *ctx,
    const uint8_t *plaintext,
    uint32_t plaintext_len,
    uint8_t *ciphertext)
{
    uint32_t i, j;
    uint32_t remaining = plaintext_len;
    const uint8_t *in = plaintext;
    uint8_t *out = ciphertext;

    while (remaining > 0) {
        // Generate new keystream block if needed
        if (ctx->keystream_index >= CHACHA20_BLOCK_SIZE) {
            chacha20_block(ctx, ctx->keystream);
            ctx->keystream_index = 0;
        }

        // XOR plaintext with keystream
        uint32_t block_size = CHACHA20_BLOCK_SIZE - ctx->keystream_index;
        if (block_size > remaining) {
            block_size = remaining;
        }

        for (j = 0; j < block_size; j++) {
            out[j] = in[j] ^ ctx->keystream[ctx->keystream_index + j];
        }

        in += block_size;
        out += block_size;
        ctx->keystream_index += block_size;
        remaining -= block_size;
    }
}

void ChaCha20_Decrypt(
    ChaCha20_Context *ctx,
    const uint8_t *ciphertext,
    uint32_t ciphertext_len,
    uint8_t *plaintext)
{
    // Decryption is identical to encryption for stream ciphers
    ChaCha20_Encrypt(ctx, ciphertext, ciphertext_len, plaintext);
}

void ChaCha20_Crypt(ChaCha20_Context *ctx, uint8_t *data, uint32_t len)
{
    uint32_t i, j;
    uint32_t remaining = len;

    while (remaining > 0) {
        // Generate new keystream block if needed
        if (ctx->keystream_index >= CHACHA20_BLOCK_SIZE) {
            chacha20_block(ctx, ctx->keystream);
            ctx->keystream_index = 0;
        }

        // XOR in-place
        uint32_t block_size = CHACHA20_BLOCK_SIZE - ctx->keystream_index;
        if (block_size > remaining) {
            block_size = remaining;
        }

        for (j = 0; j < block_size; j++) {
            data[j] ^= ctx->keystream[ctx->keystream_index + j];
        }

        data += block_size;
        ctx->keystream_index += block_size;
        remaining -= block_size;
    }
}

void ChaCha20_Zero(volatile void *ptr, size_t len)
{
    volatile uint8_t *p = (volatile uint8_t *)ptr;
    size_t i;
    for (i = 0; i < len; i++) {
        p[i] = 0;
    }
}

// ============================================================================
// WRAPPER FUNCTIONS FOR COMPATIBILITY WITH BEACON AGENT
// ============================================================================

/**
 * Simple wrapper to use ChaCha20 like the existing RC4 functions
 *
 * Usage:
 *   EncryptChaCha20(data, size, key, nonce);
 */
void EncryptChaCha20(unsigned char *data, int dataLength, unsigned char *key, unsigned char *nonce)
{
    ChaCha20_Context ctx;
    ChaCha20_Init(&ctx, key, nonce, CHACHA20_MODE_IETF);
    ChaCha20_Crypt(&ctx, data, (uint32_t)dataLength);
    ChaCha20_Zero(&ctx, sizeof(ctx));
}

/**
 * Decrypt wrapper (identical to encrypt for stream ciphers)
 */
void DecryptChaCha20(unsigned char *data, int dataLength, unsigned char *key, unsigned char *nonce)
{
    // Decryption is identical to encryption
    EncryptChaCha20(data, dataLength, key, nonce);
}
