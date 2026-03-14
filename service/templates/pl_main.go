package main

import (
	"encoding/json"
	"fmt"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Teamserver interface ──────────────────────────────────────────────────────
// Subset of Teamserver methods useful for service plugins.
// Extend as needed for your service — the full interface is defined by axc2.

type Teamserver interface {
	TsClientBroadcastChan(jsonMsg string) error
}

type PluginService struct{}

var (
	ModuleDir string
	Ts        Teamserver
)

// InitPlugin is the plugin entry-point called by the Teamserver on load.
// It receives the teamserver handle and the absolute path to the module directory.
func InitPlugin(ts any, moduleDir string) adaptix.PluginService {
	ModuleDir = moduleDir
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
		broadcast(operator, fmt.Sprintf("[!] Unknown service function: %s", function))
	}
}

// ─── Example handler ───────────────────────────────────────────────────────────

func handlePing(operator string, args string) {
	broadcast(operator, "[+] Pong! __NAME_CAP__ service is running.")
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// broadcast sends a message to the operator's client channel.
func broadcast(operator string, message string) {
	msg := map[string]string{
		"type":    "service",
		"service": "__NAME__",
		"message": message,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	_ = Ts.TsClientBroadcastChan(string(data))
}
