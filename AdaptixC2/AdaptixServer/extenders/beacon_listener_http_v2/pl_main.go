package main

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	adaptix "github.com/Adaptix-Framework/axc2"
	"github.com/gin-gonic/gin"
)

type Teamserver interface {
	TsAgentIsExists(agentId string) bool
	TsAgentCreate(agentCrc string, agentId string, beat []byte, listenerName string, ExternalIP string, Async bool) (adaptix.AgentData, error)
	TsAgentProcessData(agentId string, bodyData []byte) error
	TsAgentSetTick(agentId string, listenerName string) error
	TsAgentGetHostedAll(agentId string, maxDataSize int) ([]byte, error)
}

type PluginListener struct{}

var (
	ModuleDir       string
	ListenerDataDir string
	Ts              Teamserver
)

// InitPlugin initializes the listener plugin
func InitPlugin(ts any, moduleDir string, listenerDir string) adaptix.PluginListener {
	ModuleDir = moduleDir
	ListenerDataDir = listenerDir
	Ts = ts.(Teamserver)
	return &PluginListener{}
}

// Create instantiates a new HTTP listener with configuration
func (p *PluginListener) Create(name string, config string, customData []byte) (adaptix.ExtenderListener, adaptix.ListenerData, []byte, error) {
	var (
		listener     *Listener
		listenerData adaptix.ListenerData
		conf         TransportConfig
		customdData  []byte
		err          error
	)

	if customData == nil {
		if err = validConfig(config); err != nil {
			return nil, listenerData, customdData, err
		}

		if err = json.Unmarshal([]byte(config), &conf); err != nil {
			return nil, listenerData, customdData, err
		}

		// Normalize request headers
		conf.RequestHeaders = strings.TrimRight(conf.RequestHeaders, " \n\t\r") + "\n"
		conf.RequestHeaders = strings.ReplaceAll(conf.RequestHeaders, "\n", "\r\n")

		// Parse server response headers
		conf.ResponseHeaders = make(map[string]string)
		for _, line := range strings.Split(conf.Server_headers, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			conf.ResponseHeaders[key] = value
		}

		conf.Protocol = "http"

		// Set defaults
		if conf.EncryptKey == "" {
			conf.EncryptKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		}
		if conf.EncryptionMethod == "" {
			conf.EncryptionMethod = "chacha20poly1305"
		}
		if conf.EncodingMethod == "" {
			conf.EncodingMethod = "base64"
		}

	} else {
		if err = json.Unmarshal(customData, &conf); err != nil {
			return nil, listenerData, customdData, err
		}
		// Set defaults from customData too
		if conf.EncryptKey == "" {
			conf.EncryptKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		}
		if conf.EncryptionMethod == "" {
			conf.EncryptionMethod = "chacha20poly1305"
		}
		if conf.EncodingMethod == "" {
			conf.EncodingMethod = "base64"
		}
	}

	transport := &TransportHTTP{
		GinEngine: gin.New(),
		Name:      name,
		Config:    conf,
		Active:    false,
	}

	listenerData = adaptix.ListenerData{
		BindHost:  transport.Config.HostBind,
		BindPort:  strconv.Itoa(transport.Config.PortBind),
		AgentAddr: strings.Join(conf.Callback_addresses, ", "),
		Status:    "Stopped",
	}

	if transport.Config.Ssl {
		listenerData.Protocol = "https"
	}

	var buffer bytes.Buffer
	if err = json.NewEncoder(&buffer).Encode(transport.Config); err != nil {
		return nil, listenerData, customdData, err
	}
	customdData = buffer.Bytes()

	listener = &Listener{transport: transport}

	return listener, listenerData, customdData, nil
}

func (l *Listener) Start() error {
	return l.transport.Start(Ts)
}

func (l *Listener) Edit(config string) (adaptix.ListenerData, []byte, error) {
	var (
		listenerData adaptix.ListenerData
		conf         TransportConfig
		customdData  []byte
		err          error
	)

	if err = json.Unmarshal([]byte(config), &conf); err != nil {
		return listenerData, customdData, err
	}

	conf.RequestHeaders = strings.TrimRight(conf.RequestHeaders, " \n\t\r") + "\n"
	conf.RequestHeaders = strings.ReplaceAll(conf.RequestHeaders, "\n", "\r\n")

	// Update editable fields
	l.transport.Config.Callback_addresses = conf.Callback_addresses
	l.transport.Config.UserAgent = conf.UserAgent
	l.transport.Config.Uri = conf.Uri
	l.transport.Config.ParameterName = conf.ParameterName
	l.transport.Config.TrustXForwardedFor = conf.TrustXForwardedFor
	l.transport.Config.HostHeader = conf.HostHeader
	l.transport.Config.RequestHeaders = conf.RequestHeaders
	l.transport.Config.WebPageError = conf.WebPageError
	l.transport.Config.WebPageOutput = conf.WebPageOutput
	l.transport.Config.EnableJitter = conf.EnableJitter
	l.transport.Config.JitterMinMs = conf.JitterMinMs
	l.transport.Config.JitterMaxMs = conf.JitterMaxMs
	l.transport.Config.PayloadSizeMin = conf.PayloadSizeMin
	l.transport.Config.PayloadSizeMax = conf.PayloadSizeMax

	listenerData = adaptix.ListenerData{
		BindHost:  l.transport.Config.HostBind,
		BindPort:  strconv.Itoa(l.transport.Config.PortBind),
		AgentAddr: strings.Join(l.transport.Config.Callback_addresses, ", "),
		Status:    "Listen",
	}
	if !l.transport.Active {
		listenerData.Status = "Closed"
	}

	var buffer bytes.Buffer
	if err = json.NewEncoder(&buffer).Encode(l.transport.Config); err != nil {
		return listenerData, customdData, err
	}
	customdData = buffer.Bytes()

	return listenerData, customdData, nil
}

func (l *Listener) Stop() error {
	return l.transport.Stop()
}

func (l *Listener) GetProfile() ([]byte, error) {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(l.transport.Config); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (l *Listener) InternalHandler(data []byte) (string, error) {
	return "", nil
}
