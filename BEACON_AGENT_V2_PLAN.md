# Beacon Agent v2 - ChaCha20-Poly1305 Support

## 🎯 Задача

Добавить поддержку **ChaCha20-Poly1305** в beacon_agent для работы с beacon_listener_http_v2.

---

## 📊 Текущее Состояние

### Что есть сейчас:
- ✅ `beacon_listener_http_v2` с ChaCha20-Poly1305 (Go)
- ❌ `beacon_agent` только с RC4 (C++)
- ❌ Несовместимость между v2 listener и текущим agent

### Как работает beacon_agent:
```
src_beacon/beacon/
├── ConnectorHTTP.cpp     - HTTP transport
├── ConnectorDNS.cpp      - DNS transport
├── ConnectorTCP.cpp      - TCP transport
├── Crypt.h               - Криптография (RC4)
├── Agent.cpp             - Main agent logic
└── ... (другие компоненты)
```

### Криптография в агенте:
- Используется **RC4** везде (функции `EncryptRC4`, `DecryptRC4`)
- Ключ: 16 байт
- Все данные: beat + data зашифровываются RC4

---

## 🔧 План Реализации

### Вариант 1️⃣ : Быстрая реализация (2-3 часа)

Создать **ChaCha20 поддержку** без Poly1305 (только шифрование):

**Плюсы:**
- Быстро реализуется
- Работает с listener_http_v2
- Проще портировать на Windows

**Минусы:**
- Нет проверки целостности
- Меньше OPSEC улучшений

**Файлы:**
1. `Crypt_ChaCha20.h` - ChaCha20 (только шифрование)
2. Модифицировать `ConnectorHTTP.cpp` для поддержки обоих методов
3. Конфигурация для выбора метода

### Вариант 2️⃣ : Полная реализация (5-7 часов)

Добавить **ChaCha20-Poly1305 AEAD**:

**Плюсы:**
- Full AEAD (authenticated encryption)
- Проверка целостности
- Максимальная безопасность

**Минусы:**
- Сложнее реализовать на C++
- Требует генерации nonce в агенте
- Больше кода

**Файлы:**
1. `Crypt_ChaCha20.h` - ChaCha20 + Poly1305
2. `Crypt_ChaCha20.cpp` - Implementation
3. Модифицировать коннекторы
4. Структура для nonce management

---

## 💡 Рекомендуемое Решение

**Вариант 1 + Вариант 2 Hybrid:**

1. Реализовать **ChaCha20-only** (быстро работает)
2. Добавить Poly1305 позже (опционально)
3. Конфигурируемый выбор:
   - Режим 0 = RC4 (legacy)
   - Режим 1 = ChaCha20 (new)
   - Режим 2 = ChaCha20-Poly1305 (future)

---

## 📋 Файлы для Создания/Модификации

### Новые файлы:

#### 1. `Crypt_ChaCha20.h`
```cpp
#pragma once

// ChaCha20 stream cipher
// Based on RFC 7539
// Generates keystream for XOR encryption

typedef struct {
    uint32_t state[16];
} ChaCha20_ctx;

void ChaCha20_Init(ChaCha20_ctx *ctx, uint8_t *key, uint8_t *nonce, uint32_t counter);
void ChaCha20_Block(ChaCha20_ctx *ctx, uint8_t *output);
void ChaCha20_Encrypt(uint8_t *plaintext, uint32_t len, uint8_t *key, uint8_t *nonce, uint32_t *counter);
void ChaCha20_Decrypt(uint8_t *ciphertext, uint32_t len, uint8_t *key, uint8_t *nonce, uint32_t *counter);
```

#### 2. `Crypt_ChaCha20.cpp`
```cpp
#include "Crypt_ChaCha20.h"
#include <string.h>

// ChaCha20 implementation (portable C)
// ~200 lines of code

void ChaCha20_Init(ChaCha20_ctx *ctx, uint8_t *key, uint8_t *nonce, uint32_t counter) {
    // Initialize state with:
    // - Constants
    // - Key (32 bytes)
    // - Counter (4 bytes)
    // - Nonce (8 bytes for counter mode, 12 for standard)
}

void ChaCha20_Block(ChaCha20_ctx *ctx, uint8_t *output) {
    // Generate 64-byte keystream block
}

void ChaCha20_Encrypt(uint8_t *plaintext, uint32_t len, uint8_t *key, uint8_t *nonce, uint32_t *counter) {
    // XOR plaintext with keystream
}
```

### Модифицировать:

#### 1. `ConnectorHTTP.cpp`
```cpp
// Add support for both RC4 and ChaCha20

enum EncryptionMethod {
    ENC_RC4 = 0,
    ENC_CHACHA20 = 1
};

class ConnectorHTTP {
    BYTE encryption_method = ENC_RC4;  // Default to RC4 for compatibility
    BYTE chacha20_counter[4] = {0};    // Counter for ChaCha20
    
    // Modify Exchange():
    void Exchange(BYTE* plainData, ULONG plainSize, BYTE* sessionKey) {
        if (encryption_method == ENC_CHACHA20) {
            ChaCha20_Encrypt(plainData, plainSize, sessionKey, nonce, counter);
        } else {
            EncryptRC4(plainData, plainSize, sessionKey, 16);
        }
    }
};
```

#### 2. `Agent.cpp`
```cpp
// Read encryption_method from config
// Pass to all connectors
// Default to RC4 for backward compatibility

config->encryption_method = ENC_RC4;  // Default
// Later: read from listener configuration
```

#### 3. `ConnectorDNS.cpp`, `ConnectorTCP.cpp`, `ConnectorSMB.cpp`
```cpp
// Similar changes to HTTP connector
// Support both encryption methods
```

---

## 🔄 Integration Flow

```
┌─────────────────────────────────────┐
│  beacon_listener_http_v2 (Go)       │
│  - ChaCha20-Poly1305                │
│  - Config: encryption_method=1      │
└──────────────┬──────────────────────┘
               │
        ┌──────▼────────┐
        │ HTTPS Request │
        │ Beat (encrypted)
        └──────┬────────┘
               │
┌──────────────▼──────────────────────┐
│  beacon_agent (C++)                 │
│  - Receives encryption_method in    │
│    beat or first response           │
│  - Switches to ChaCha20             │
│  - All future data encrypted        │
└─────────────────────────────────────┘
```

---

## 📝 Implementation Checklist

### Phase 1: ChaCha20 Core (2 hours)
- [ ] Create `Crypt_ChaCha20.h` header
- [ ] Implement `ChaCha20_Init()` function
- [ ] Implement `ChaCha20_Block()` function
- [ ] Implement `ChaCha20_Encrypt()` wrapper
- [ ] Test ChaCha20 in isolation

### Phase 2: Agent Integration (2 hours)
- [ ] Add `encryption_method` to config struct
- [ ] Modify `ConnectorHTTP.cpp` to support both methods
- [ ] Add nonce/counter management
- [ ] Update header creation for v2 listener

### Phase 3: Compatibility (1 hour)
- [ ] Ensure RC4 still works (default)
- [ ] Test with existing agents (RC4 mode)
- [ ] Test with v2 listener (ChaCha20 mode)
- [ ] Fallback logic if listener returns error

### Phase 4: Polish (1 hour)
- [ ] Documentation
- [ ] Example configurations
- [ ] Build instructions
- [ ] Testing guide

---

## 🛠️ Implementation Details

### ChaCha20 Algorithm (Simple Version)

```cpp
// Constants
const uint32_t CHACHA20_CONSTANT[4] = {
    0x61707865, 0x3320646e, 0x79622d32, 0x6b206574
};

// Initialize state
void ChaCha20_Init(ctx, key, nonce) {
    state[0-3]   = CONSTANTS
    state[4-11]  = KEY (32 bytes / 8 words)
    state[12]    = COUNTER_LOW
    state[13]    = COUNTER_HIGH
    state[14-15] = NONCE (8 bytes / 2 words) or NONCE (12 bytes)
}

// Generate keystream
void ChaCha20_Block(ctx, output) {
    working_state = copy(ctx->state)
    for i = 0 to 9:
        working_state = ChaCha20_Quarter_Round(working_state)
    output = working_state XOR initial_state
    increment counter
}

// Encrypt: plaintext XOR keystream
void ChaCha20_Encrypt(plaintext, key, nonce) {
    keystream = ChaCha20_Block()
    ciphertext = plaintext XOR keystream
}
```

### Configuration Transfer

Agent needs to know which encryption to use. Options:

**Option A: Static Config**
```cpp
// Hardcoded in agent build
#define ENCRYPTION_METHOD ENC_CHACHA20
```

**Option B: Dynamic from Listener**
```cpp
// Listener sends in first response header
// E.g., "X-Crypto-Method: chacha20"
// Agent parses and switches
```

**Option C: Auto-detect (Recommended)**
```cpp
// Try ChaCha20 first
// If listener rejects (error response)
// Fall back to RC4
```

---

## 🔐 Security Considerations

1. **Nonce Management**: 
   - ChaCha20 requires nonce + counter
   - Counter = 4 bytes (auto-increment)
   - Nonce = 8 or 12 bytes (unique per connection)

2. **Key Derivation**:
   - Same 32-byte key from listener config
   - No additional key derivation needed (unlike TLS)

3. **Backward Compatibility**:
   - Default to RC4
   - Only use ChaCha20 when explicitly configured
   - No breaking changes to protocol

4. **Windows Compatibility**:
   - No Windows crypto APIs needed
   - Portable C implementation works on all platforms

---

## 📦 Dependencies

- **No external libraries needed**
- Pure C/C++ implementation
- ~300 lines of code total

---

## 📚 References

- RFC 7539: ChaCha20 and Poly1305 (https://tools.ietf.org/html/rfc7539)
- ChaCha20 implementations: libsodium, WireGuard
- Portable C example: https://github.com/jedisct1/libsodium

---

## ⏱️ Timeline

- **Today**: Create plan ✅
- **Tomorrow**: Implement ChaCha20 core + agent integration
- **Next Day**: Testing + documentation
- **Final**: Commit to repository

---

## 🎯 Success Criteria

1. ✅ Beacon agent builds without errors
2. ✅ Agent connects to beacon_listener_http_v2
3. ✅ Encryption/decryption works correctly
4. ✅ RC4 backward compatibility maintained
5. ✅ Documentation provided
6. ✅ Example configs included

---

## ❓ FAQ

**Q: Why not use Windows CryptoAPI?**
A: Windows CryptoAPI doesn't have ChaCha20. Need portable solution.

**Q: Will this slow down the agent?**
A: ChaCha20 is actually faster than RC4 on modern CPUs.

**Q: What about Poly1305 (the MAC)?**
A: Can be added later. ChaCha20 alone is secure for confidentiality.

**Q: Compatibility with old listeners?**
A: Yes, RC4 is default. Only uses ChaCha20 when configured.

---

**Status**: Ready to implement  
**Complexity**: Medium  
**Time**: 5-7 hours total  
**Value**: High (makes v2 listener functional)
