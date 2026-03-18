package impl

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"__NAME__/protocol"
)

func listDirEntries(path string) (string, []protocol.DirEntry, error) {
	norm := path
	if norm == "" {
		norm = "."
	}
	abs, err := filepath.Abs(norm)
	if err != nil {
		return path, nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return abs, nil, err
	}
	out := make([]protocol.DirEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, protocol.DirEntry{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
		})
	}
	return abs, out, nil
}

func runShellCommand(shell string, args []string, capture bool, wait bool) (string, error) {
	cmd := exec.Command(shell, args...)
	if !wait {
		if capture {
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf
		}
		return "", cmd.Start()
	}
	if capture {
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	return "", cmd.Run()
}

func parseUnixPS(out []byte) []protocol.ProcessEntry {
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	procs := make([]protocol.ProcessEntry, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		pid, err1 := strconv.ParseUint(fields[0], 10, 32)
		ppid, err2 := strconv.ParseUint(fields[1], 10, 32)
		if err1 != nil || err2 != nil {
			continue
		}
		user := fields[2]
		name := strings.Join(fields[3:], " ")
		procs = append(procs, protocol.ProcessEntry{Pid: uint32(pid), PPid: uint32(ppid), Name: name, User: user, Arch: "", Session: 0})
	}
	return procs
}

func parseWindowsTasklist(out []byte) ([]protocol.ProcessEntry, error) {
	r := csv.NewReader(bytes.NewReader(out))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	procs := make([]protocol.ProcessEntry, 0, len(records))
	for _, rec := range records {
		if len(rec) < 2 {
			continue
		}
		pid, err := strconv.ParseUint(strings.TrimSpace(rec[1]), 10, 32)
		if err != nil {
			continue
		}
		session := uint32(0)
		if len(rec) > 3 {
			if v, err := strconv.ParseUint(strings.TrimSpace(rec[3]), 10, 32); err == nil {
				session = uint32(v)
			}
		}
		procs = append(procs, protocol.ProcessEntry{Pid: uint32(pid), PPid: 0, Name: strings.TrimSpace(rec[0]), User: "", Arch: "", Session: session})
	}
	return procs, nil
}

type screenshotAttempt struct {
	name string
	fn   func() ([]byte, error)
}

func captureScreenshotAttempts(attempts ...screenshotAttempt) ([]byte, error) {
	if len(attempts) == 0 {
		return nil, fmt.Errorf("no screenshot backends configured")
	}

	errs := make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		data, err := attempt.fn()
		if err == nil && len(data) > 0 {
			return data, nil
		}
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", attempt.name, err))
			continue
		}
		errs = append(errs, fmt.Sprintf("%s: empty screenshot output", attempt.name))
	}

	return nil, fmt.Errorf("screenshot capture failed (%s)", strings.Join(errs, "; "))
}

func captureScreenshotFromStdout(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("command returned no data")
	}
	return out, nil
}

func captureScreenshotFromTempFile(command string, args ...string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "adaptix-screenshot-*.png")
	if err != nil {
		return nil, err
	}
	path := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(path)

	resolved := make([]string, len(args))
	for i, arg := range args {
		resolved[i] = strings.ReplaceAll(arg, "{out}", path)
	}

	cmd := exec.Command(command, resolved...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("command created an empty screenshot file")
	}
	return data, nil
}

func singleQuotedPowerShellPath(path string) string {
	return strings.ReplaceAll(path, "'", "''")
}
