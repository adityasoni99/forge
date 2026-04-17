// factory/sandbox/types.go
package sandbox

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

type SandboxConfig struct {
	Image        string
	WorkspaceDir string
	Env          map[string]string
	Timeout      time.Duration
	CPULimit     string
	MemoryLimit  string
	NetworkMode  string
}

type SandboxResult struct {
	ExitCode int
	Output   string
	Duration time.Duration
}

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, int, error)
}

type ExecRunner struct{}

func NewExecRunner() *ExecRunner { return &ExecRunner{} }

func (r *ExecRunner) Run(ctx context.Context, name string, args ...string) (string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return out.String(), exitErr.ExitCode(), nil
		}
		return out.String(), -1, err
	}
	return out.String(), 0, nil
}
