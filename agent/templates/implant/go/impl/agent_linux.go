//go:build linux

package impl

import (
	"__NAME__/protocol"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Platform-specific implementations for Linux.
// Fill in every TODO method below with your own logic.
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Platform ──────────────────────────────────────────────────────────────────

// GetCP returns the terminal code page. Usually 0 on Linux.
func (a *Agent) GetCP() uint32 {
	// TODO: Implement if you need a specific code page.
	return 0
}

// IsElevated returns true when running as root.
// TODO: Implement privilege detection (e.g., os.Geteuid() == 0).
func (a *Agent) IsElevated() bool {
	return false
}

// GetOsVersion returns a human-readable OS version string.
// TODO: Implement (e.g., read /etc/os-release PRETTY_NAME).
func (a *Agent) GetOsVersion() string {
	return "Linux"
}

// NormalizePath converts a path to its canonical form.
// TODO: Implement (e.g., expand ~/ to home directory).
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
// TODO: Implement using io.Copy or similar.
func (a *Agent) CopyFile(src, dst string) error {
	return nil
}

// CopyDir recursively copies a directory.
// TODO: Implement using filepath.Walk.
func (a *Agent) CopyDir(src, dst string) error {
	return nil
}

// ─── Execution ─────────────────────────────────────────────────────────────────

// RunShell executes a shell command via /bin/sh -c.
// TODO: Implement. Consider using PTY (creack/pty) for interactive shells.
func (a *Agent) RunShell(cmd string, output bool, wait bool) (string, error) {
	return "", nil
}

// ListProcesses returns a snapshot of running processes.
// TODO: Implement (e.g., read /proc or use gopsutil).
func (a *Agent) ListProcesses() ([]protocol.ProcessEntry, error) {
	return nil, nil
}

// CaptureScreenshot takes a screenshot and returns PNG bytes.
// TODO: Implement (e.g., use kbinani/screenshot or xdotool).
func (a *Agent) CaptureScreenshot() ([]byte, error) {
	return nil, nil
}
