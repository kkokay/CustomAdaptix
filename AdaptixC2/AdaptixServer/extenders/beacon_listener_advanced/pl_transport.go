package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TransportConfig struct {
	HostBind     string   `json:"host_bind"`
	PortBind     int      `json:"port_bind"`
	CallbackAddrs []string `json:"callback_addresses"`
	EncryptKey   string   `json:"encrypt_key"`
	UserAgents   []string `json:"user_agents"`
	URIs         []string `json:"uris"`
	HTTPMethod   string   `json:"http_method"`
	UseSSL       bool     `json:"ssl"`
	JitterMs     int      `json:"jitter_ms"`
	PaddingMin   int      `json:"padding_min"`
	PaddingMax   int      `json:"padding_max"`
}

type TransportHTTP struct {
	GinEngine *gin.Engine
	Server    *http.Server
	Config    TransportConfig
	Name      string
	Active    bool
	Crypto    *CryptoProvider
}

func (t *TransportHTTP) Start(ts Teamserver) error {
	var err error

	// Init crypto
	t.Crypto, err = NewCryptoProvider(t.Config.EncryptKey)
	if err != nil {
		return fmt.Errorf("crypto init failed: %v", err)
	}

	// Setup HTTP handler
	t.GinEngine.POST("/*path", func(c *gin.Context) {
		t.handleRequest(c, ts)
	})

	// Start server
	t.Server = &http.Server{
		Addr:    net.JoinHostPort(t.Config.HostBind, fmt.Sprintf("%d", t.Config.PortBind)),
		Handler: t.GinEngine,
	}

	go t.Server.ListenAndServe()
	t.Active = true

	fmt.Printf("   Started listener '%s': http://%s:%d (ChaCha20-Poly1305 + HMAC)\n",
		t.Name, t.Config.HostBind, t.Config.PortBind)

	return nil
}

func (t *TransportHTTP) Stop() error {
	if t.Server != nil {
		t.Active = false
		return t.Server.Close()
	}
	return nil
}

func (t *TransportHTTP) handleRequest(c *gin.Context, ts Teamserver) {
	// Get beat from header
	beatB64 := c.GetHeader("X-Beacon-Id")
	if beatB64 == "" {
		c.JSON(400, nil)
		return
	}

	// Decode beat
	beatData, err := base64.StdEncoding.DecodeString(beatB64)
	if err != nil {
		c.JSON(400, nil)
		return
	}

	// Decrypt with HMAC verification
	beatDecrypted, err := t.Crypto.Decrypt(beatData)
	if err != nil {
		c.JSON(400, nil)
		return
	}

	// Parse agent info
	if len(beatDecrypted) < 8 {
		c.JSON(400, nil)
		return
	}

	agentType := fmt.Sprintf("%08x", binary.BigEndian.Uint32(beatDecrypted[:4]))
	agentId := fmt.Sprintf("%08x", binary.BigEndian.Uint32(beatDecrypted[4:8]))
	beat := beatDecrypted[8:]

	// Create agent
	ts.TsAgentCreate(agentType, agentId, beat, t.Name, c.ClientIP(), true)

	// Read body
	bodyData, _ := c.GetRawData()
	if len(bodyData) > 0 {
		ts.TsAgentProcessData(agentId, bodyData)
	}

	c.JSON(200, map[string]string{"status": "ok"})
}
