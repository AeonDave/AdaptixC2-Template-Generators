package main

import (
	"fmt"
	"sync"
)

// ─── Pipeline Stage ────────────────────────────────────────────────────────────
// A Stage is a named transformation step applied to a payload in sequence.
// Each stage receives the payload bytes, an immutable config map, and the build
// context — and returns the (possibly modified) payload or an error.

type Stage struct {
	Name    string
	Enabled bool
	Run     func(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error)
}

// BuildContext carries shared state through the pipeline.
type BuildContext struct {
	AgentName string
	BuilderId string
	FileName  string
	ModuleDir string
	Extra     map[string]any
}

// ─── Pipeline ──────────────────────────────────────────────────────────────────

var (
	pipelineMu sync.RWMutex
	stages     []Stage
)

// RegisterStage appends a stage to the end of the pipeline.
func RegisterStage(s Stage) {
	pipelineMu.Lock()
	defer pipelineMu.Unlock()
	stages = append(stages, s)
}

// RunPipeline executes all enabled stages in order.
// It logs progress via TsAgentBuildLog and returns the final payload.
func RunPipeline(payload []byte, cfg map[string]string, ctx *BuildContext) ([]byte, error) {
	pipelineMu.RLock()
	ordered := make([]Stage, len(stages))
	copy(ordered, stages)
	pipelineMu.RUnlock()

	current := payload
	for _, s := range ordered {
		// Config can override the default enable/disable per stage.
		// Keys: "stage.<name>.enabled" = "true" | "false"
		enabled := s.Enabled
		if v, ok := cfg["stage."+s.Name+".enabled"]; ok {
			enabled = (v == "true" || v == "1")
		}
		if !enabled {
			logBuild(ctx.BuilderId, BuildLogInfo, fmt.Sprintf("[%s] skipped (disabled)", s.Name))
			continue
		}
		logBuild(ctx.BuilderId, BuildLogInfo, fmt.Sprintf("[%s] running...", s.Name))

		result, err := s.Run(current, cfg, ctx)
		if err != nil {
			logBuild(ctx.BuilderId, BuildLogError, fmt.Sprintf("[%s] failed: %v", s.Name, err))
			return nil, fmt.Errorf("stage %s: %w", s.Name, err)
		}
		logBuild(ctx.BuilderId, BuildLogInfo, fmt.Sprintf("[%s] done (%d → %d bytes)", s.Name, len(current), len(result)))
		current = result
	}
	return current, nil
}

// logBuild is a convenience wrapper — it silently no-ops if Ts is nil (unit tests).
func logBuild(builderID string, status int, message string) {
	if Ts != nil && builderID != "" {
		_ = Ts.TsAgentBuildLog(builderID, status, message)
	}
}
