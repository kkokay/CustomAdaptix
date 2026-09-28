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

func InitPlugin(ts any, moduleDir string, listenerDir string) adaptix.PluginListener {
	ModuleDir = moduleDir
	ListenerDataDir = listenerDir
	Ts = ts.(Teamserver)
	return &PluginListener{}
}

func (p *PluginListener) Create(name string, config string, customData []byte) (adaptix.ExtenderListener, adaptix.ListenerData, []byte, error) {
	var listener *Listener
	var listenerData adaptix.ListenerData
	var conf TransportConfig
	var err error

	if customData == nil {
		if err = json.Unmarshal([]byte(config), &conf); err != nil {
			return nil, listenerData, nil, err
		}
	} else {
		if err = json.Unmarshal(customData, &conf); err != nil {
			return nil, listenerData, nil, err
		}
	}

	transport := &TransportHTTP{
		GinEngine: gin.New(),
		Name:      name,
		Config:    conf,
		Active:    false,
	}

	listenerData = adaptix.ListenerData{
		BindHost:  conf.HostBind,
		BindPort:  strconv.Itoa(conf.PortBind),
		AgentAddr: strings.Join(conf.CallbackAddrs, ", "),
		Status:    "Stopped",
	}

	if conf.UseSSL {
		listenerData.Protocol = "https"
	}

	var buffer bytes.Buffer
	json.NewEncoder(&buffer).Encode(transport.Config)

	listener = &Listener{transport: transport}
	return listener, listenerData, buffer.Bytes(), nil
}

func (l *Listener) Start() error {
	return l.transport.Start(Ts)
}

func (l *Listener) Edit(config string) (adaptix.ListenerData, []byte, error) {
	var conf TransportConfig
	var buffer bytes.Buffer
	json.Unmarshal([]byte(config), &conf)
	l.transport.Config = conf

	json.NewEncoder(&buffer).Encode(l.transport.Config)

	listenerData := adaptix.ListenerData{
		BindHost:  l.transport.Config.HostBind,
		BindPort:  strconv.Itoa(l.transport.Config.PortBind),
		AgentAddr: strings.Join(l.transport.Config.CallbackAddrs, ", "),
		Status:    "Listen",
	}
	if !l.transport.Active {
		listenerData.Status = "Closed"
	}

	return listenerData, buffer.Bytes(), nil
}

func (l *Listener) Stop() error {
	return l.transport.Stop()
}

func (l *Listener) GetProfile() ([]byte, error) {
	var buffer bytes.Buffer
	json.NewEncoder(&buffer).Encode(l.transport.Config)
	return buffer.Bytes(), nil
}

func (l *Listener) InternalHandler(data []byte) (string, error) {
	return "", nil
}

type Listener struct {
	transport *TransportHTTP
}
