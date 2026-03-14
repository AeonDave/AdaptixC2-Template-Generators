//go:build windows

package impl

import (
	"__NAME__/protocol"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Platform-specific implementations for Windows.
// Fill in every TODO method below with your own logic.
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Platform ──────────────────────────────────────────────────────────────────

// GetCP returns the system code page.
// TODO: Implement (e.g., call kernel32.GetACP via syscall).
func (a *Agent) GetCP() uint32 {
	return 0
}

// IsElevated returns true if running with admin privileges.
// TODO: Implement (e.g., windows.GetCurrentProcessToken().IsElevated()).
func (a *Agent) IsElevated() bool {
	return false
}

// GetOsVersion returns a human-readable OS version string.
// TODO: Implement (e.g., read registry HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion).
func (a *Agent) GetOsVersion() string {
	return "Windows"
}

// NormalizePath converts a path to its canonical Windows form.
// TODO: Implement (e.g., expand %USERPROFILE%, resolve /).
func (a *Agent) NormalizePath(path string) string {
	return path
}

// ─── FileSystem ────────────────────────────────────────────────────────────────

// GetListing returns the absolute path and directory entries.
// TODO: Implement using os.ReadDir.
func (a *Agent) GetListing(path string) (string, []protocol.DirEntry, error) {
	return path, nil, nil
}

// CopyFile copies a single file from src to dst.
// TODO: Implement.
func (a *Agent) CopyFile(src, dst string) error {
	return nil
}

// CopyDir recursively copies a directory.
// TODO: Implement.
func (a *Agent) CopyDir(src, dst string) error {
	return nil
}

// ─── Execution ─────────────────────────────────────────────────────────────────

// RunShell executes a command via cmd.exe /c.
// TODO: Implement. Consider using gabemarshall/pty for ConPTY support.
func (a *Agent) RunShell(cmd string, output bool, wait bool) (string, error) {
	return "", nil
}

// ListProcesses returns a snapshot of running processes.
// TODO: Implement (e.g., CreateToolhelp32Snapshot or gopsutil).
func (a *Agent) ListProcesses() ([]protocol.ProcessEntry, error) {
	return nil, nil
}

// CaptureScreenshot takes a screenshot and returns PNG bytes.
// TODO: Implement (e.g., use GDI+ BitBlt or kbinani/screenshot).
func (a *Agent) CaptureScreenshot() ([]byte, error) {
	return nil, nil
}
