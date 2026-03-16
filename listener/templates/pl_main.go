package main

import (
	"bytes"
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
		if err = validConfig(config); err != nil {
			return nil, listenerData, customdData, err
		}

		err = json.Unmarshal([]byte(config), &conf)
		if err != nil {
			return nil, listenerData, customdData, err
		}

		conf.Callback_addresses = strings.ReplaceAll(conf.Callback_addresses, " ", "")
		conf.Callback_addresses = strings.ReplaceAll(conf.Callback_addresses, "\n", ", ")
		conf.Callback_addresses = strings.TrimSuffix(conf.Callback_addresses, ", ")

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

	listenerData = adaptix.ListenerData{
		BindHost:  transport.Config.HostBind,
		BindPort:  strconv.Itoa(transport.Config.PortBind),
		AgentAddr: conf.Callback_addresses,
		Status:    "Stopped",
	}

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
	return l.transport.Start(Ts)
}

func (l *Listener) Stop() error {
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

	listenerData = adaptix.ListenerData{
		BindHost:  l.transport.Config.HostBind,
		BindPort:  strconv.Itoa(l.transport.Config.PortBind),
		AgentAddr: l.transport.Config.Callback_addresses,
		Status:    "Listen",
	}
	if !l.transport.Active {
		listenerData.Status = "Closed"
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
// TODO: Implement if your listener_type is "internal".

func (l *Listener) InternalHandler(data []byte) (string, error) {
	return "", errors.New("not implemented")
}
