package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aditya-soni/forge/factory/delivery"
	"github.com/aditya-soni/forge/factory/sandbox"
	"github.com/aditya-soni/forge/factory/workspace"
)

// SandboxRunner abstracts sandbox image management and container execution.
type SandboxRunner interface {
	EnsureImage(ctx context.Context, config sandbox.SandboxConfig) error
	Run(ctx context.Context, config sandbox.SandboxConfig, command []string) (sandbox.SandboxResult, error)
}

// WorkspaceCreator abstracts git worktree lifecycle management.
type WorkspaceCreator interface {
	Create(ctx context.Context, repoDir, runID string) (*workspace.Workspace, error)
	Destroy(ctx context.Context, ws *workspace.Workspace) error
}

// Deliverer abstracts git push + PR creation.
type Deliverer interface {
	Deliver(ctx context.Context, workspaceDir, branch string, config delivery.DeliveryConfig) (delivery.DeliveryResult, error)
}

// Pipeline wires workspace, sandbox, and delivery into a single run lifecycle.
type Pipeline struct {
	sandbox   SandboxRunner
	workspace WorkspaceCreator
	delivery  Deliverer
}

// NewPipeline constructs a Pipeline from its three dependency interfaces.
func NewPipeline(sbx SandboxRunner, ws WorkspaceCreator, dlv Deliverer) *Pipeline {
	return &Pipeline{sandbox: sbx, workspace: ws, delivery: dlv}
}

// Execute runs the full forge pipeline: workspace → sandbox → delivery → cleanup.
func (p *Pipeline) Execute(ctx context.Context, req RunRequest) (RunResult, error) {
	start := time.Now()
	runID := generateRunID()
	result := RunResult{RunID: runID, Status: RunStatusRunning}

	ws, err := p.workspace.Create(ctx, req.RepoDir, runID)
	if err != nil {
		appendEvent(&result.Events, "workspace", "create failed", time.Since(start))
		result.Status = RunStatusFailed
		result.Error = fmt.Sprintf("workspace create: %v", err)
		result.Duration = time.Since(start)
		return result, nil
	}
	appendEvent(&result.Events, "workspace", "created "+ws.Branch, time.Since(start))
	defer func() {
		destroyStart := time.Now()
		_ = p.workspace.Destroy(ctx, ws)
		appendEvent(&result.Events, "workspace", "destroyed", time.Since(destroyStart))
	}()
	result.Branch = ws.Branch

	image := req.Image
	if image == "" {
		image = "forge:latest"
	}
	sbxConfig := sandbox.SandboxConfig{
		Image:        image,
		WorkspaceDir: ws.Dir,
		Env:          req.Env,
		Timeout:      req.Timeout,
		NetworkMode:  "none",
	}

	ensureStart := time.Now()
	if err := p.sandbox.EnsureImage(ctx, sbxConfig); err != nil {
		appendEvent(&result.Events, "sandbox", "ensure image failed", time.Since(ensureStart))
		result.Status = RunStatusFailed
		result.Error = fmt.Sprintf("ensure image: %v", err)
		result.Duration = time.Since(start)
		return result, nil
	}
	appendEvent(&result.Events, "sandbox", "image ready", time.Since(ensureStart))

	command := buildSandboxCommand(req)
	runStart := time.Now()
	sbxResult, err := p.sandbox.Run(ctx, sbxConfig, command)
	if err != nil {
		appendEvent(&result.Events, "sandbox", "docker run error", time.Since(runStart))
		result.Status = RunStatusFailed
		result.Error = fmt.Sprintf("sandbox run: %v", err)
		result.Duration = time.Since(start)
		return result, nil
	}
	appendEvent(&result.Events, "sandbox", "completed", sbxResult.Duration)
	result.Output = sbxResult.Output

	if sbxResult.ExitCode != 0 {
		result.Status = RunStatusFailed
		result.Error = fmt.Sprintf("blueprint exited with code %d", sbxResult.ExitCode)
		result.Duration = time.Since(start)
		return result, nil
	}

	if !req.NoPR {
		dlvConfig := delivery.DeliveryConfig{
			Remote:     "origin",
			BaseBranch: req.BaseBranch,
			PRTitle:    fmt.Sprintf("forge: %s", req.Task),
			PRBody:     fmt.Sprintf("Automated by Forge.\n\nBlueprint: %s\nRun ID: %s", req.BlueprintName, runID),
		}
		dlvStart := time.Now()
		dlvResult, err := p.delivery.Deliver(ctx, ws.Dir, ws.Branch, dlvConfig)
		if err != nil {
			appendEvent(&result.Events, "delivery", "failed", time.Since(dlvStart))
			result.Status = RunStatusFailed
			result.Error = fmt.Sprintf("delivery: %v", err)
			result.Duration = time.Since(start)
			return result, nil
		}
		appendEvent(&result.Events, "delivery", "pushed and PR", time.Since(dlvStart))
		result.PRURL = dlvResult.PRURL
	}

	result.Status = RunStatusPassed
	result.Duration = time.Since(start)
	return result, nil
}

func generateRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func appendEvent(events *[]RunEvent, phase, message string, d time.Duration) {
	*events = append(*events, RunEvent{
		Phase:     phase,
		Timestamp: time.Now(),
		Message:   message,
		Duration:  d,
	})
}

func buildSandboxCommand(req RunRequest) []string {
	var args []string
	if req.BlueprintName != "" {
		args = append(args, "--blueprint", req.BlueprintName)
	}
	if req.BlueprintFile != "" {
		args = append(args, "--blueprint-file", req.BlueprintFile)
	}
	if req.Task != "" {
		args = append(args, "--task", req.Task)
	}
	if req.Adapter != "" {
		args = append(args, "--adapter", req.Adapter)
	}
	return args
}
