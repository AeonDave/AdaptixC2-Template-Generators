//go:build darwin

package impl

import (
	"__NAME__/protocol"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Platform-specific implementations for macOS.
// Fill in every TODO method below with your own logic.
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Platform ──────────────────────────────────────────────────────────────────

// GetCP returns the terminal code page. Usually 0 on macOS.
func (a *Agent) GetCP() uint32 {
	return 0
}

// IsElevated returns true when running as root.
// TODO: Implement (e.g., os.Geteuid() == 0).
func (a *Agent) IsElevated() bool {
	return false
}

// GetOsVersion returns a human-readable OS version string.
// TODO: Implement (e.g., run sw_vers -productVersion).
func (a *Agent) GetOsVersion() string {
	return "macOS"
}

// NormalizePath converts a path to its canonical form.
// TODO: Implement (e.g., expand ~/).
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

// RunShell executes a command via /bin/sh -c.
// TODO: Implement. Consider using creack/pty for PTY support.
func (a *Agent) RunShell(cmd string, output bool, wait bool) (string, error) {
	return "", nil
}

// ListProcesses returns a snapshot of running processes.
// TODO: Implement (e.g., use gopsutil or parse ps aux).
func (a *Agent) ListProcesses() ([]protocol.ProcessEntry, error) {
	return nil, nil
}

// CaptureScreenshot takes a screenshot and returns PNG bytes.
// TODO: Implement (e.g., use screencapture command or kbinani/screenshot).
func (a *Agent) CaptureScreenshot() ([]byte, error) {
	return nil, nil
}
