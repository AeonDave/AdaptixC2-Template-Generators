package main

import (
	"encoding/json"
	"fmt"
	"os"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Build configuration ───────────────────────────────────────────────────────

type GenerateConfig struct {
	Os            string `json:"os"`
	Arch          string `json:"arch"`
	ReconnTimeout string `json:"reconn_timeout"`
	ReconnCount   int    `json:"reconn_count"`
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

	switch generateConfig.Os + "/" + generateConfig.Arch {
	case "linux/amd64":
		target = "x86_64-unknown-linux-gnu"
		Filename = "__NAME__"
	case "linux/arm64":
		target = "aarch64-unknown-linux-gnu"
		Filename = "__NAME__"
	case "windows/amd64":
		target = "x86_64-pc-windows-gnu"
		Filename = "__NAME__.exe"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported target: %s/%s", generateConfig.Os, generateConfig.Arch)
	}

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Target: %s, Output: %s", target, Filename))

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

	cmdBuild := fmt.Sprintf("__BUILD_TOOL__ --release --target %s --target-dir %s", target, tempDir)
	err = Ts.TsAgentBuildExecute(profile.BuilderId, srcDir, "sh", "-c", cmdBuild)
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

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Payload: %s (%d bytes)", Filename, len(Payload)))

	return Payload, Filename, nil
}
