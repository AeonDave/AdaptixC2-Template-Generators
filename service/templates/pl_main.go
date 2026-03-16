package main

import (
	"encoding/json"
	"fmt"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Constants ─────────────────────────────────────────────────────────────────

const (
	HookPre  = 0 // Event hook phase: before the core action
	HookPost = 1 // Event hook phase: after the core action

	BuildLogNone    = 0 // No status
	BuildLogInfo    = 1 // Informational build log
	BuildLogError   = 2 // Error build log
	BuildLogSuccess = 3 // Success build log
)

// ─── Teamserver interface ──────────────────────────────────────────────────────
// Subset of Teamserver methods useful for service plugins.
// Extend as needed for your service — the full interface is defined by axc2.

type Teamserver interface {
	// Register an event hook (e.g. "agent.generate", "agent.checkin").
	//   eventType — event name
	//   name      — unique hook identifier
	//   phase     — HookPre (0) or HookPost (1)
	//   priority  — lower runs first
	//   handler   — callback; receives event payload, returns error to abort (pre-hooks only)
	TsEventHookRegister(eventType string, name string, phase int, priority int, handler func(event any) error) string

	// Append a log line to the agent-build log visible in the UI.
	TsAgentBuildLog(builderId string, status int, message string) error

	// Persist arbitrary data scoped to this extender (survives restarts).
	TsExtenderDataSave(extenderName string, key string, value []byte) error
	TsExtenderDataLoad(extenderName string, key string) ([]byte, error)

	// Send a JSON payload to a specific operator client.
	TsServiceSendDataClient(operator string, service string, data string)
	// Send a JSON payload to ALL operator clients.
	TsServiceSendDataAll(service string, data string)
}

type PluginService struct{}

var (
	ModuleDir     string
	ServiceConfig string
	Ts            Teamserver
)

// InitPlugin is the plugin entry-point called by the Teamserver on load.
// It receives the teamserver handle, the module directory path,
// and the raw service configuration string from config.yaml.
func InitPlugin(ts any, moduleDir string, serviceConfig string) adaptix.PluginService {
	ModuleDir = moduleDir
	ServiceConfig = serviceConfig
	Ts = ts.(Teamserver)
	return &PluginService{}
}

// ─── Call ──────────────────────────────────────────────────────────────────────
// Called by the Teamserver when an operator invokes a service function.
//   operator — username of the requesting operator
//   function — name of the function to execute (defined in ax_config.axs)
//   args     — JSON-encoded arguments from the UI form

func (s *PluginService) Call(operator string, function string, args string) {
	switch function {

	case "ping":
		handlePing(operator, args)

	/// START CODE HERE — add your function cases
	/// END CODE HERE

	default:
		sendError(operator, fmt.Sprintf("Unknown service function: %s", function))
	}
}

// ─── Example handler ───────────────────────────────────────────────────────────

func handlePing(operator string, args string) {
	sendSuccess(operator, "[+] Pong! __NAME_CAP__ service is running.")
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// sendJSON marshals a value and sends it to a specific operator (or all if operator is empty).
func sendJSON(operator string, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	if operator != "" {
		Ts.TsServiceSendDataClient(operator, "__NAME__", string(data))
	} else {
		Ts.TsServiceSendDataAll("__NAME__", string(data))
	}
}

// sendError sends an error response to the operator (or broadcast if operator is empty).
func sendError(operator string, message string) {
	sendJSON(operator, map[string]any{
		"success": false,
		"error":   message,
	})
}

// sendSuccess sends a success response to the operator (or broadcast if operator is empty).
func sendSuccess(operator string, message string) {
	sendJSON(operator, map[string]any{
		"success": true,
		"output":  message,
	})
}
