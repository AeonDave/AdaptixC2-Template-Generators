package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Build configuration ───────────────────────────────────────────────────────

type GenerateConfig struct {
	Os            string `json:"os"`
	Arch          string `json:"arch"`
	Format        string `json:"format"`
	SvcName       string `json:"svc_name"`
	ReconnTimeout string `json:"reconn_timeout"`
	ReconnCount   int    `json:"reconn_count"`
	// Kill Date — set via "Set kill date" checkbox in ax_config UI.
	// Remove these fields (and the matching container.put lines) if not needed.
	IsKillDate bool   `json:"is_killdate"`
	KillDate   string `json:"kill_date"`
	KillTime   string `json:"kill_time"`
	// Working Time — set via "Set working time" checkbox in ax_config UI.
	// Remove these fields (and the matching container.put lines) if not needed.
	IsWorkTime bool   `json:"is_workingtime"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
}

var SrcPath = "src___NAME__"

// ─── Profile generation ────────────────────────────────────────────────────────

func (p *__NAME_CAP__Plugin) GenerateProfiles(profile adaptix.BuildProfile) ([][]byte, error) {
	var agentProfiles [][]byte

	for _, transportProfile := range profile.ListenerProfiles {

		var listenerMap map[string]any
		if err := json.Unmarshal(transportProfile.Profile, &listenerMap); err != nil {
			return nil, err
		}

		// TODO: Parse listener fields and build a profile blob.
		// For Rust agents, the profile is written into src/config.rs before cargo build.
		//
		// Example:
		//   encryptKey := ...
		//   profileData, _ := msgpack.Marshal(myProfile)
		//   extHandler := __NAME_CAP__Extender{}
		//   profileData, _ = extHandler.Encrypt(profileData, encryptKey)
		//   profileData = append(encryptKey, profileData...)
		//   profileString := ""
		//   for _, b := range profileData {
		//       profileString += fmt.Sprintf("\\x%02x", b)
		//   }
		//   agentProfiles = append(agentProfiles, []byte(profileString))

		_ = listenerMap
	}

	return agentProfiles, nil
}

// ─── Build payload (Rust / Cargo) ─────────────────────────────────────────────

func (p *__NAME_CAP__Plugin) BuildPayload(profile adaptix.BuildProfile, agentProfiles [][]byte) ([]byte, string, error) {
	var (
		Filename string
		Payload  []byte
	)

	var (
		generateConfig GenerateConfig
		target         string
	)

	err := json.Unmarshal([]byte(profile.AgentConfig), &generateConfig)
	if err != nil {
		return nil, "", err
	}

	srcDir := ModuleDir + "/" + SrcPath
	tempDir, err := os.MkdirTemp("", "ax-*")
	if err != nil {
		return nil, "", err
	}

	var cargoFlags string // extra cargo flags per format

	switch generateConfig.Os + "/" + generateConfig.Arch {
	case "linux/amd64":
		target = "x86_64-unknown-linux-gnu"
	case "linux/arm64":
		target = "aarch64-unknown-linux-gnu"
	case "windows/amd64":
		target = "x86_64-pc-windows-gnu"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported target: %s/%s", generateConfig.Os, generateConfig.Arch)
	}

	// ── Format → cargo flags + output filename ─────────────────────────────
	format := generateConfig.Format
	if format == "" {
		// Legacy/default: pick by OS
		switch generateConfig.Os {
		case "windows":
			format = "Exe"
		case "linux":
			format = "Binary"
		default:
			format = "Binary Mach-O"
		}
	}

	switch format {
	case "Exe", "Binary", "Binary Mach-O":
		if generateConfig.Os == "windows" {
			Filename = "__NAME__.exe"
		} else {
			Filename = "__NAME__"
		}
	case "Service Exe":
		Filename = "__NAME___svc.exe"
		cargoFlags = " --features service"
	case "DLL":
		Filename = "__NAME__.dll"
		cargoFlags = " --lib"
	case "Shellcode":
		Filename = "__NAME__.dll" // build DLL first, then sRDI
		cargoFlags = " --lib"
	case "Shared Object (.so)":
		Filename = "lib__NAME__.so"
		cargoFlags = " --lib"
	case "Dynamic Library (.dylib)":
		Filename = "lib__NAME__.dylib"
		cargoFlags = " --lib"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Target: %s, Format: %s, Output: %s", target, format, Filename))

	// Write profile data into src/config.rs
	config := "pub static ENC_PROFILES: &[&[u8]] = &[\n"
	for _, p := range agentProfiles {
		config += fmt.Sprintf("    b\"%s\",\n", p)
	}
	config += "];\n"

	configPath := srcDir + "/src/config.rs"
	err = os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}

	// Build
	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO, "Starting cargo build...")

	cmdBuild := fmt.Sprintf("__BUILD_TOOL__ --target %s --target-dir %s%s", target, tempDir, cargoFlags)

	shellCmd := fmt.Sprintf(`[ -f "$HOME/.cargo/env" ] && . "$HOME/.cargo/env"; %s`, cmdBuild)
	err = Ts.TsAgentBuildExecute(profile.BuilderId, srcDir, "sh", "-c", shellCmd)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}

	// Read output
	outputPath := fmt.Sprintf("%s/%s/release/%s", tempDir, target, Filename)
	Payload, err = os.ReadFile(outputPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}
	_ = os.RemoveAll(tempDir)

	// ── Post-processing: sRDI for Shellcode format ──────────────────────
	if format == "Shellcode" {
		_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
			fmt.Sprintf("Applying sRDI conversion (%d byte DLL)...", len(Payload)))
		Payload, err = DllToShellcode(Payload)
		if err != nil {
			return nil, "", err
		}
		Filename = "__NAME__.bin"
	}

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Payload: %s (%d bytes)", Filename, len(Payload)))

	return Payload, Filename, nil
}

// ─── Helper: parse kill date + time → Unix timestamp ───────────────────────
// Returns 0 when the checkbox is unchecked.
// Date format from ax_config dateline: "DD.MM.YYYY", time: "HH:MM".

func parseKillDate(enabled bool, dateStr, timeStr string) int64 {
	if !enabled || dateStr == "" {
		return 0
	}
	if timeStr == "" {
		timeStr = "00:00"
	}
	t, err := time.Parse("02.01.2006 15:04", dateStr+" "+timeStr)
	if err != nil {
		return 0
	}
	return t.Unix()
}

// ─── Helper: parse "HH:MM" → integer HHMM ─────────────────────────────────
// Returns 0 when the checkbox is unchecked.

func parseTimeHHMM(enabled bool, s string) int {
	if !enabled || s == "" {
		return 0
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0
	}
	return t.Hour()*100 + t.Minute()
}
