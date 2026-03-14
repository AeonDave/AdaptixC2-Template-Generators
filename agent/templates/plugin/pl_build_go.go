package main

import (
	"encoding/json"
	"fmt"
	"os"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Build configuration ───────────────────────────────────────────────────────
// Parsed from ax_config.axs container keys.
// Adjust fields to match your agent's build-time configuration UI.

type GenerateConfig struct {
	Os            string `json:"os"`
	Arch          string `json:"arch"`
	Win7support   bool   `json:"win7_support"`
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

		// TODO: Parse listener-specific fields from listenerMap and agent config
		// from profile.AgentConfig. Build a profile blob (e.g. msgpack-encoded
		// Profile struct), encrypt it, and append to agentProfiles.
		//
		// Example (see gopher_agent for a complete reference):
		//   var gc GenerateConfig
		//   json.Unmarshal([]byte(profile.AgentConfig), &gc)
		//   encrypt_key, _ := listenerMap["encrypt_key"].(string)
		//   encryptKey, _ := hex.DecodeString(encrypt_key)
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

// ─── Build payload (Go implant) ────────────────────────────────────────────────

func (p *__NAME_CAP__Plugin) BuildPayload(profile adaptix.BuildProfile, agentProfiles [][]byte) ([]byte, string, error) {
	var (
		Filename string
		Payload  []byte
	)

	var (
		generateConfig GenerateConfig
		GoArch         string
		GoOs           string
		buildPath      string
	)

	err := json.Unmarshal([]byte(profile.AgentConfig), &generateConfig)
	if err != nil {
		return nil, "", err
	}

	currentDir := ModuleDir
	tempDir, err := os.MkdirTemp("", "ax-*")
	if err != nil {
		return nil, "", err
	}

	switch generateConfig.Arch {
	case "amd64":
		GoArch = "amd64"
	case "arm64":
		GoArch = "arm64"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported architecture: %s", generateConfig.Arch)
	}

	LdFlags := "-s -w"
	switch generateConfig.Os {
	case "linux":
		GoOs = "linux"
		Filename = "__NAME__.bin"
	case "macos":
		GoOs = "darwin"
		Filename = "__NAME__.bin"
	case "windows":
		GoOs = "windows"
		Filename = "__NAME__.exe"
		LdFlags += " -H=windowsgui"
	default:
		_ = os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("unsupported OS: %s", generateConfig.Os)
	}
	buildPath = tempDir + "/" + Filename

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Target: %s/%s, Output: %s", GoOs, GoArch, Filename))

	// Write profile data into config.go as Go byte array literals
	config := "package main\n\nvar encProfiles = [][]byte{\n"
	for _, p := range agentProfiles {
		config += fmt.Sprintf("\t[]byte(\"%s\"),\n", p)
	}
	config += "}\n"

	configPath := currentDir + "/" + SrcPath + "/config.go"
	err = os.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}

	// Build
	cmdBuild := fmt.Sprintf("GOWORK=off CGO_ENABLED=0 GOOS=%s GOARCH=%s __BUILD_TOOL__ -trimpath -ldflags=\"%s\" -o %s",
		GoOs, GoArch, LdFlags, buildPath)

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO, "Starting build process...")

	err = Ts.TsAgentBuildExecute(profile.BuilderId, currentDir+"/"+SrcPath, "sh", "-c", cmdBuild)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}

	Payload, err = os.ReadFile(buildPath)
	if err != nil {
		_ = os.RemoveAll(tempDir)
		return nil, "", err
	}
	_ = os.RemoveAll(tempDir)

	_ = Ts.TsAgentBuildLog(profile.BuilderId, adaptix.BUILD_LOG_INFO,
		fmt.Sprintf("Payload size: %d bytes", len(Payload)))

	return Payload, Filename, nil
}
