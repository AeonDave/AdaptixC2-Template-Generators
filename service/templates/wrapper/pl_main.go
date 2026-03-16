package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	adaptix "github.com/Adaptix-Framework/axc2"
)

// ─── Constants ─────────────────────────────────────────────────────────────────

const (
	HookPre  = 0
	HookPost = 1

	BuildLogNone    = 0
	BuildLogInfo    = 1
	BuildLogError   = 2
	BuildLogSuccess = 3
)

// ─── Teamserver interface ──────────────────────────────────────────────────────

type Teamserver interface {
	TsEventHookRegister(eventType string, name string, phase int, priority int, handler func(event any) error) string
	TsAgentBuildLog(builderId string, status int, message string) error
	TsExtenderDataSave(extenderName string, key string, value []byte) error
	TsExtenderDataLoad(extenderName string, key string) ([]byte, error)
	TsServiceSendDataClient(operator string, service string, data string)
	TsServiceSendDataAll(service string, data string)
}

type PluginService struct{}

var (
	ModuleDir     string
	ServiceConfig string
	Ts            Teamserver
)

// ─── Init ──────────────────────────────────────────────────────────────────────

func InitPlugin(ts any, moduleDir string, serviceConfig string) adaptix.PluginService {
	ModuleDir = moduleDir
	ServiceConfig = serviceConfig
	Ts = ts.(Teamserver)

	// Register pipeline stages.
	initStages()

	// Hook into agent build (post-phase) so we can wrap every generated payload.
	Ts.TsEventHookRegister("agent.generate", "__NAME___wrapper", HookPost, 100, onAgentGenerate)

	return &PluginService{}
}

// ─── Pipeline stage registration ───────────────────────────────────────────────
// Add or remove stages here.  Each stage is a self-contained transformation.

func initStages() {
	/// START CODE HERE — register your pipeline stages
	/// Example:
	/// RegisterStage(Stage{
	///     Name:    "encrypt",
	///     Enabled: true,
	///     Run:     stageEncrypt,
	/// })
	/// END CODE HERE
}

// ─── Event hook: agent.generate (post) ─────────────────────────────────────────
// Called after every agent payload is generated.
// The event is a pointer to EventDataAgentGenerate with fields:
//   FileContent   []byte    — the generated payload (read/write in-place)
//   FileName      string    — output file name (can be renamed)
//   AgentName     string    — agent type name
//   ListenersName []string  — listener names used for the build
//   Config        string    — build configuration (JSON)

func onAgentGenerate(event any) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[__NAME__] hook panic recovered: %v\n", r)
		}
	}()

	v := reflect.ValueOf(event)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	s := v.Elem()

	fileContent := s.FieldByName("FileContent")
	if !fileContent.IsValid() {
		return nil
	}
	fileName := s.FieldByName("FileName")
	if !fileName.IsValid() {
		return nil
	}

	payload := fileContent.Bytes()
	if len(payload) == 0 {
		return nil
	}

	agentName := ""
	if f := s.FieldByName("AgentName"); f.IsValid() {
		agentName = f.String()
	}

	logBuild("", BuildLogInfo, fmt.Sprintf("[__NAME__] wrapping %s (%d bytes)", fileName.String(), len(payload)))

	cfg := loadConfig()

	ctx := &BuildContext{
		AgentName: agentName,
		FileName:  fileName.String(),
		ModuleDir: ModuleDir,
		Extra:     make(map[string]any),
	}

	wrapped, err := RunPipeline(payload, cfg, ctx)
	if err != nil {
		logBuild("", BuildLogError, fmt.Sprintf("[__NAME__] pipeline failed: %v", err))
		return nil // don't fail the build — wrapper is best-effort
	}

	// Write the modified payload back in-place on the event struct.
	if fileContent.CanSet() {
		fileContent.SetBytes(wrapped)
		logBuild("", BuildLogSuccess, fmt.Sprintf("[__NAME__] payload wrapped (%d → %d bytes)", len(payload), len(wrapped)))
	}

	// Optionally rename the output file (stages can set ctx.Extra["output_filename"]).
	if newName, ok := ctx.Extra["output_filename"].(string); ok && newName != "" {
		if fileName.IsValid() && fileName.CanSet() {
			fileName.SetString(newName)
		}
	}

	return nil
}

// ─── Call (service UI actions) ─────────────────────────────────────────────────

func (s *PluginService) Call(operator string, function string, args string) {
	switch function {

	case "status":
		handleStatus(operator)

	case "save_config":
		handleSaveConfig(operator, args)

	case "load_config":
		handleLoadConfig(operator)

	/// START CODE HERE — add your function cases
	/// END CODE HERE

	default:
		sendError(operator, "error", fmt.Sprintf("Unknown wrapper function: %s", function))
	}
}

// ─── Handlers ──────────────────────────────────────────────────────────────────

func handleStatus(operator string) {
	pipelineMu.RLock()
	info := make([]map[string]any, 0, len(stages))
	for _, s := range stages {
		info = append(info, map[string]any{
			"name":    s.Name,
			"enabled": s.Enabled,
		})
	}
	pipelineMu.RUnlock()

	sendSuccess(operator, "status", fmt.Sprintf("%d stages registered", len(info)))
}

func handleSaveConfig(operator string, args string) {
	var cfg map[string]string
	if err := json.Unmarshal([]byte(args), &cfg); err != nil {
		sendError(operator, "save_config", fmt.Sprintf("Invalid config JSON: %v", err))
		return
	}
	data, _ := json.Marshal(cfg)
	if err := Ts.TsExtenderDataSave("__NAME__", "config", data); err != nil {
		sendError(operator, "save_config", fmt.Sprintf("Failed to save: %v", err))
		return
	}
	sendSuccess(operator, "save_config", "Configuration saved.")
}

func handleLoadConfig(operator string) {
	data, err := Ts.TsExtenderDataLoad("__NAME__", "config")
	if err != nil {
		sendError(operator, "load_config", fmt.Sprintf("Failed to load: %v", err))
		return
	}
	sendSuccess(operator, "load_config", string(data))
}

// ─── Config persistence ────────────────────────────────────────────────────────

func loadConfig() map[string]string {
	data, err := Ts.TsExtenderDataLoad("__NAME__", "config")
	if err != nil || len(data) == 0 {
		return make(map[string]string)
	}
	var cfg map[string]string
	if json.Unmarshal(data, &cfg) != nil {
		return make(map[string]string)
	}
	return cfg
}

// ─── Helpers ───────────────────────────────────────────────────────────────────

// Result mirrors StealthPalace/axc2 response pattern for data_handler routing.
type Result struct {
	Action  string `json:"action"`
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

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

func sendError(operator string, action string, message string) {
	sendJSON(operator, Result{
		Action:  action,
		Success: false,
		Error:   message,
	})
}

func sendSuccess(operator string, action string, message string) {
	sendJSON(operator, Result{
		Action:  action,
		Success: true,
		Output:  message,
	})
}
