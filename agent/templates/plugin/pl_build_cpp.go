package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Build configuration ───────────────────────────────────────────────────────
// Parsed from ax_config.axs container keys.
// Adjust fields to match your C++ agent's build-time configuration UI.

type GenerateConfig struct {
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
		// For C++ agents, the profile is passed via preprocessor -DPROFILE flag.
		// GenerateProfiles should produce a hex-escaped string: \x01\x02...
		//
		// Example (see beacon_agent for a complete reference):
		//   var gc GenerateConfig
		//   json.Unmarshal([]byte(profile.AgentConfig), &gc)
		//   encryptKey := ...
		//   params := buildProfileParams(listenerMap, gc)
		//   packedParams, _ := PackArray(params)
		//   cryptParams, _ := RC4Crypt(packedParams, encryptKey)
		//   profileArray := []interface{}{len(cryptParams), cryptParams, encryptKey}
		//   packedProfile, _ := PackArray(profileArray)
		//   profileString := ""
		//   for _, b := range packedProfile {
		//       profileString += fmt.Sprintf("\\x%02x", b)
		//   }
		//   agentProfiles = append(agentProfiles, []byte(profileString))

		_ = listenerMap
	}

	return agentProfiles, nil
}

// ─── Build payload (C++ / MinGW) ──────────────────────────────────────────────
//
// Uses the implant Makefile to compile from source. Profile data is injected
// via a generated header (profile_gen.h) included through -include.

func (p *__NAME_CAP__Plugin) BuildPayload(profile adaptix.BuildProfile, agentProfiles [][]byte) ([]byte, string, error) {
	var (
		Filename string
		Payload  []byte
	)

	if len(agentProfiles) != 1 {
		return nil, "", fmt.Errorf("expected 1 agent profile, got %d", len(agentProfiles))
	}
	agentProfile := agentProfiles[0]
	agentProfileSize := len(agentProfile) / 4 // each \xNN = 4 chars

	var generateConfig GenerateConfig
	err := json.Unmarshal([]byte(profile.AgentConfig), &generateConfig)
	if err != nil {
		return nil, "", err
	}

	srcDir := ModuleDir + "/" + SrcPath
	tempDir, err := os.MkdirTemp("", "ax-*")
	if err != nil {
		return nil, "", err
	}

	// ── Architecture ───────────────────────────────────────────────────────

	var makeTarget string
	switch generateConfig.Arch {
	case "x64":
		makeTarget = "x64"
	case "x86":
		makeTarget = "x86"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported architecture: %s", generateConfig.Arch)
	}

	// ── Format ─────────────────────────────────────────────────────────────

	var makeFormat string
	switch generateConfig.Format {
	case "Exe":
		makeFormat = "exe"
	case "Service Exe":
		makeFormat = "svc"
	case "DLL":
		makeFormat = "dll"
	case "Shellcode":
		makeFormat = "shellcode"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported format: %s", generateConfig.Format)
	}

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Target: %s %s (%s)", generateConfig.Arch, generateConfig.Format, makeFormat))

	// ── Write profile header ───────────────────────────────────────────────
	// This header is included first via -include, so #define PROFILE takes
	// effect before config.cpp's #ifndef guard.

	header := fmt.Sprintf("#define PROFILE \"%s\"\n#define PROFILE_SIZE %d\n",
		string(agentProfile), agentProfileSize)

	if generateConfig.Format == "Service Exe" && generateConfig.SvcName != "" {
		svcHex := ""
		for _, c := range generateConfig.SvcName {
			svcHex += fmt.Sprintf("\\x%02x", c)
		}
		header += fmt.Sprintf("#define SERVICE_NAME \"%s\"\n", svcHex)
	}

	headerPath := tempDir + "/profile_gen.h"
	err = os.WriteFile(headerPath, []byte(header), 0644)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}

	// ── Build via Make ─────────────────────────────────────────────────────
	// BEACON=$tempDir/__NAME__ redirects output to the temp directory.
	// PROFILE_HEADER=$headerPath adds -include for the profile defines.

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO, "Compiling and linking...")

	err = Ts.TsAgentBuildExecute(profile.BuilderId, srcDir, "make", makeTarget,
		fmt.Sprintf("FORMAT=%s", makeFormat),
		fmt.Sprintf("BEACON=%s/__NAME__", tempDir),
		fmt.Sprintf("PROFILE_HEADER=%s", headerPath))
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}

	// ── Read output ────────────────────────────────────────────────────────

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == "profile_gen.h" {
			continue
		}
		buildPath := tempDir + "/" + e.Name()
		Payload, err = os.ReadFile(buildPath)
		if err != nil {
			_ = os.RemoveAll(tempDir)
			return nil, "", err
		}
		Filename = e.Name()
		break
	}
	_ = os.RemoveAll(tempDir)

	if Payload == nil {
		return nil, "", fmt.Errorf("build produced no output")
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
