package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Teamserver interface ──────────────────────────────────────────────────────
// Full interface used by stock adaptix_gopher listener. Use only the methods you need.

type Teamserver interface {
	TsAgentIsExists(agentId string) bool
	TsAgentCreate(agentCrc string, agentId string, beat []byte, listenerName string, ExternalIP string, Async bool) (adaptix.AgentData, error)
	TsAgentSetTick(agentId string, listenerName string) error
	TsAgentProcessData(agentId string, bodyData []byte) error
	TsAgentGetHostedAll(agentId string, maxDataSize int) ([]byte, error)
	TsAgentGetHostedTasks(agentId string, maxDataSize int) ([]byte, error)
	TsAgentUpdateDataPartial(agentId string, updateData interface{}) error

	TsTaskRunningExists(agentId string, taskId string) bool
	TsTunnelChannelExists(channelId int) bool

	TsAgentTerminalCloseChannel(terminalId string, status string) error
	TsTerminalConnExists(terminalId string) bool
	TsTerminalConnResume(agentId string, terminalId string, ioDirect bool)
	TsTerminalGetPipe(AgentId string, terminalId string) (*io.PipeReader, *io.PipeWriter, error)

	TsTunnelGetPipe(AgentId string, channelId int) (*io.PipeReader, *io.PipeWriter, error)
	TsTunnelConnectionResume(AgentId string, channelId int, ioDirect bool)
	TsTunnelConnectionClose(channelId int, writeOnly bool)
	TsTunnelConnectionHalt(channelId int, errorCode byte)
	TsTunnelConnectionData(channelId int, data []byte)
	TsTunnelConnectionAccept(tunnelId int, channelId int)

	TsConvertCpToUTF8(input string, codePage int) string
	TsConvertUTF8toCp(input string, codePage int) string
	TsWin32Error(errorCode uint) string
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

// ─── Create ────────────────────────────────────────────────────────────────────

func (p *PluginListener) Create(name string, config string, customData []byte) (adaptix.ExtenderListener, adaptix.ListenerData, []byte, error) {
	var (
		listener     *Listener
		listenerData adaptix.ListenerData
		customdData  []byte
		conf         TransportConfig
		err          error
	)

	if customData == nil {
		if "__LISTENER_TYPE__" != "internal" {
			if err = validConfig(config); err != nil {
				return nil, listenerData, customdData, err
			}
		}

		err = json.Unmarshal([]byte(config), &conf)
		if err != nil {
			return nil, listenerData, customdData, err
		}

		conf.Callback_addresses = strings.ReplaceAll(conf.Callback_addresses, " ", "")
		conf.Callback_addresses = strings.ReplaceAll(conf.Callback_addresses, "\n", ", ")
		conf.Callback_addresses = strings.TrimSuffix(conf.Callback_addresses, ", ")

		if "__LISTENER_TYPE__" == "internal" {
			if conf.Timeout < 1 {
				conf.Timeout = 10
			}
			if conf.ErrorAnswer == "" {
				conf.ErrorAnswer = "Connection error...\n"
			}
		}

		conf.Protocol = "__PROTOCOL__"
	} else {
		err = json.Unmarshal(customData, &conf)
		if err != nil {
			return nil, listenerData, customdData, err
		}
	}

	transport := &Transport__NAME_CAP__{
		Name:          name,
		Config:        conf,
		AgentConnects: NewMap(),
		JobConnects:   NewMap(),
		Active:        false,
	}

	listenerData = buildListenerData(transport)

	if transport.Config.Ssl {
		listenerData.Protocol = "mtls"
	}

	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(transport.Config)
	if err != nil {
		return nil, listenerData, customdData, err
	}
	customdData = buffer.Bytes()

	listener = &Listener{transport: transport}

	return listener, listenerData, customdData, nil
}

// ─── Start / Stop ──────────────────────────────────────────────────────────────

func (l *Listener) Start() error {
	if "__LISTENER_TYPE__" == "internal" {
		l.transport.Active = true
		return nil
	}

	return l.transport.Start(Ts)
}

func (l *Listener) Stop() error {
	if "__LISTENER_TYPE__" == "internal" {
		l.transport.Active = false
		return nil
	}

	return l.transport.Stop()
}

// ─── Edit ──────────────────────────────────────────────────────────────────────

func (l *Listener) Edit(config string) (adaptix.ListenerData, []byte, error) {
	var (
		listenerData adaptix.ListenerData
		customdData  []byte
		conf         TransportConfig
		err          error
	)

	err = json.Unmarshal([]byte(config), &conf)
	if err != nil {
		return listenerData, customdData, err
	}

	conf.Callback_addresses = strings.ReplaceAll(conf.Callback_addresses, " ", "")
	conf.Callback_addresses = strings.ReplaceAll(conf.Callback_addresses, "\n", ", ")
	conf.Callback_addresses = strings.TrimSuffix(conf.Callback_addresses, ", ")

	l.transport.Config.Callback_addresses = conf.Callback_addresses
	l.transport.Config.TcpBanner = conf.TcpBanner
	l.transport.Config.ErrorAnswer = conf.ErrorAnswer
	l.transport.Config.Timeout = conf.Timeout

	l.transport.Config.HostBind = conf.HostBind
	l.transport.Config.PortBind = conf.PortBind
	l.transport.Config.EncryptKey = conf.EncryptKey
	l.transport.Config.Ssl = conf.Ssl
	l.transport.Config.CaCert = conf.CaCert
	l.transport.Config.ServerCert = conf.ServerCert
	l.transport.Config.ServerKey = conf.ServerKey
	l.transport.Config.ClientCert = conf.ClientCert
	l.transport.Config.ClientKey = conf.ClientKey
	l.transport.Config.Debug = conf.Debug

	listenerData = buildListenerData(l.transport)
	if l.transport.Config.Ssl {
		listenerData.Protocol = "mtls"
	}

	var buffer bytes.Buffer
	err = json.NewEncoder(&buffer).Encode(l.transport.Config)
	if err != nil {
		return listenerData, customdData, err
	}
	customdData = buffer.Bytes()

	return listenerData, customdData, nil
}

// ─── GetProfile ────────────────────────────────────────────────────────────────

func (l *Listener) GetProfile() ([]byte, error) {
	var buffer bytes.Buffer

	err := json.NewEncoder(&buffer).Encode(l.transport.Config)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// ─── InternalHandler ───────────────────────────────────────────────────────────
// Called for internal listeners (bind_tcp, bind_smb). Not used for external listeners.

func (l *Listener) InternalHandler(data []byte) (string, error) {
	if "__LISTENER_TYPE__" != "internal" {
		return "", errors.New("listener is not internal")
	}

	encKey, err := hex.DecodeString(l.transport.Config.EncryptKey)
	if err != nil {
		return "", err
	}

	decrypted, err := DecryptData(data, encKey)
	if err != nil {
		return "", err
	}

	agentType, agentID, beat, err := ParseInternalAgentRegistration(decrypted)
	if err != nil {
		return "", err
	}

	if !Ts.TsAgentIsExists(agentID) {
		_, err = Ts.TsAgentCreate(agentType, agentID, beat, l.transport.Name, "", false)
		if err != nil {
			return agentID, err
		}
	}

	return agentID, nil
}

func buildListenerData(transport *Transport__NAME_CAP__) adaptix.ListenerData {
	listenerData := adaptix.ListenerData{
		BindHost:  transport.Config.HostBind,
		BindPort:  strconv.Itoa(transport.Config.PortBind),
		AgentAddr: transport.Config.Callback_addresses,
		Status:    "Stopped",
	}

	if "__LISTENER_TYPE__" == "internal" {
		listenerData.BindHost = ""
		listenerData.BindPort = ""
		listenerData.AgentAddr = "internal"
		if transport.Active {
			listenerData.Status = "Listen"
		} else {
			listenerData.Status = "Closed"
		}
		return listenerData
	}

	if transport.Active {
		listenerData.Status = "Listen"
	} else {
		listenerData.Status = "Closed"
	}

	return listenerData
}
