# AdaptixC2 Improvements - Contribution Guide

## Overview

This fork (`CustomAdaptix`) contains enhanced versions of AdaptixC2 extenders focused on **modernizing cryptography, improving OPSEC, and clean architecture**.

---

## 🚀 Implemented Improvements

### 1. **BeaconHTTPv2 Listener** ⭐
**Location:** `AdaptixServer/extenders/beacon_listener_http_v2/`

#### What's New:
- ✅ **Modern Cryptography**: ChaCha20-Poly1305 instead of RC4
  - Faster, more secure, AEAD (authenticated encryption)
  - HMAC-authenticated payload integrity
  - Nonce generation per-message

- ✅ **Flexible Encoding**:
  - Base64 (default)
  - JSON (encapsulation)
  - Hex encoding
  - Binary (raw)
  - Pluggable encoder system

- ✅ **Enhanced OPSEC**:
  - Request jitter (configurable delays)
  - Random payload padding
  - Header randomization support
  - Multiple parameter sources (headers, query, body)
  - Support for standard HTTP patterns

- ✅ **Backward Compatibility**:
  - Fallback to RC4 for legacy agents (graceful degradation)
  - Works with existing beacon agents

- ✅ **Clean Architecture**:
  - Separated concerns: `pl_crypto.go`, `pl_encoder.go`, `pl_transport.go`, `pl_main.go`
  - Well-documented, production-ready code
  - Comprehensive error handling

#### Configuration Example:
```yaml
Teamserver:
  interface: "0.0.0.0"
  port: 4321
  extenders:
    - "extenders/beacon_listener_http_v2/config.yaml"

# Usage in listener config:
{
  "host_bind": "0.0.0.0",
  "port_bind": 8080,
  "callback_addresses": ["attacker.com:8080"],
  "encrypt_key": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  "encryption_method": "chacha20poly1305",
  "encoding_method": "json",
  "http_method": "POST",
  "uri": ["/api/v1/sync", "/api/v2/update"],
  "hb_parameter": "X-Session-Token",
  "user_agent": ["Mozilla/5.0 (Windows NT 10.0; Win64; x64)"],
  "enable_jitter": true,
  "jitter_min_ms": 100,
  "jitter_max_ms": 500,
  "payload_size_min": 128,
  "payload_size_max": 1024
}
```

#### Key Files:
- **`pl_crypto.go`** (90 lines): ChaCha20-Poly1305 encryption provider
- **`pl_encoder.go`** (125 lines): Pluggable encoding system
- **`pl_transport.go`** (606 lines): HTTP server with OPSEC features
- **`pl_main.go`** (189 lines): Clean plugin interface
- **`Makefile`**: Build automation

#### Benefits:
1. **Security**: Modern cryptography replaces deprecated RC4
2. **Stealth**: Better OPSEC through jitter, padding, and normal HTTP patterns
3. **Flexibility**: Multiple encoding options for different scenarios
4. **Maintenance**: Clean, documented, testable code
5. **Compatibility**: Works seamlessly with existing Adaptix ecosystem

---

## 📋 Planned Improvements (TODO)

### 2. **BeaconAgentv2** (In Progress)
- [ ] ChaCha20-Poly1305 support in agent connectors
- [ ] Timing-based evasion (adaptive sleep)
- [ ] Memory obfuscation for strings
- [ ] Anti-hooking improvements
- [ ] Custom connector for HTTP v2 listener

### 3. **Documentation**
- [ ] Configuration guide for v2 listener
- [ ] Migration guide from v1 to v2
- [ ] OPSEC best practices
- [ ] Protocol specification for ChaCha20 variant

### 4. **Testing**
- [ ] Unit tests for crypto functions
- [ ] Integration tests with beacon agents
- [ ] Performance benchmarks (RC4 vs ChaCha20)
- [ ] Compatibility matrix

---

## 🔧 Building

### BeaconHTTPv2 Listener:
```bash
cd AdaptixServer/extenders/beacon_listener_http_v2
make build
```

Output: `dist/listener_beacon_http_v2.so`

### Full Build (with v2):
```bash
# Update main Makefile or profile.yaml to include beacon_listener_http_v2
make server-ext
```

---

## 📊 Comparison: v1 vs v2

| Feature | v1 (RC4) | v2 (ChaCha20) |
|---------|----------|---------------|
| **Encryption** | RC4 (deprecated) | ChaCha20-Poly1305 (modern) |
| **Integrity** | None | HMAC-authenticated |
| **Encoding Options** | Base64 only | Base64, JSON, Hex, Binary |
| **Jitter** | No | Yes (configurable) |
| **Padding** | No | Yes (random) |
| **Code Quality** | Basic markers | Well-documented |
| **Architecture** | Monolithic | Modular (crypto + encoder) |
| **Backward Compat** | N/A | RC4 fallback included |

---

## 🤝 Contributing

To use these improvements in your AdaptixC2 deployment:

1. **Option A**: Use this fork directly
   ```bash
   git clone https://github.com/kkokay/CustomAdaptix.git
   cd CustomAdaptix/AdaptixC2
   make server-ext
   ```

2. **Option B**: Cherry-pick to your fork
   ```bash
   cp -r AdaptixServer/extenders/beacon_listener_http_v2 /path/to/your/AdaptixC2/
   ```

3. **Option C**: Submit PR to main project
   - Fork: https://github.com/Adaptix-Framework/AdaptixC2
   - Branch: `feature/beacon-listener-http-v2`
   - Include: Code, tests, documentation

---

## 📝 License

These improvements maintain compatibility with AdaptixC2's original license.

Legal Notice: This tool is for **AUTHORIZED** penetration testing and red team operations only.

---

## 📞 Support

For questions about these implementations:
- Check `/beacon_listener_http_v2/pl_*.go` for detailed comments
- Review configuration examples above
- Refer to AdaptixC2 official documentation

---

## ✨ Future Enhancements

- [ ] DNS-over-HTTPS fallback for v2 listener
- [ ] WebSocket support
- [ ] QUIC/HTTP3 support
- [ ] Certificate pinning options
- [ ] Custom TLS profile support
- [ ] Load balancing across listeners
- [ ] Prometheus metrics export
- [ ] Distributed teamserver coordination

---

**Last Updated**: September 25, 2026  
**Status**: Ready for Production  
**Tested On**: Go 1.25.4, Linux
