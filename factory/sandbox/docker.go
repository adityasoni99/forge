package sandbox

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type DockerSandbox struct {
	runner CommandRunner
	pool   WarmPool
}

// SetWarmPool attaches a warm pool. When set, Run tries Acquire first;
// on miss (ErrPoolEmpty), it falls back to a cold docker run.
func (d *DockerSandbox) SetWarmPool(pool WarmPool) {
	d.pool = pool
}

func NewDockerSandbox(runner CommandRunner) *DockerSandbox {
	if runner == nil {
		runner = &ExecRunner{}
	}
	return &DockerSandbox{runner: runner}
}

func (d *DockerSandbox) EnsureImage(ctx context.Context, config SandboxConfig) error {
	_, exitCode, err := d.runner.Run(ctx, "docker", "image", "inspect", config.Image)
	if err != nil {
		return fmt.Errorf("docker image inspect: %w", err)
	}
	if exitCode == 0 {
		return nil
	}
	_, pullExit, err := d.runner.Run(ctx, "docker", "pull", config.Image)
	if err != nil {
		return fmt.Errorf("docker pull %s: %w", config.Image, err)
	}
	if pullExit != 0 {
		return fmt.Errorf("docker pull %s: exit code %d", config.Image, pullExit)
	}
	return nil
}

func (d *DockerSandbox) Run(ctx context.Context, config SandboxConfig, command []string) (SandboxResult, error) {
	start := time.Now()

	if d.pool != nil {
		container, err := d.pool.Acquire(ctx, config)
		if err == nil {
			result, execErr := d.runInWarmContainer(ctx, container, command, start)
			_ = d.pool.Release(container)
			return result, execErr
		}
	}

	args := d.buildRunArgs(config, command)
	output, exitCode, err := d.runner.Run(ctx, "docker", args...)
	if err != nil {
		return SandboxResult{}, fmt.Errorf("docker run: %w", err)
	}
	return SandboxResult{
		ExitCode: exitCode,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}

func (d *DockerSandbox) runInWarmContainer(ctx context.Context, container *WarmContainer, command []string, start time.Time) (SandboxResult, error) {
	args := append([]string{"exec", container.ContainerID}, command...)
	output, exitCode, err := d.runner.Run(ctx, "docker", args...)
	if err != nil {
		return SandboxResult{}, fmt.Errorf("docker exec in warm container: %w", err)
	}
	return SandboxResult{
		ExitCode: exitCode,
		Output:   output,
		Duration: time.Since(start),
	}, nil
}

var secretPatterns = []string{"*_KEY", "*_SECRET", "*_TOKEN", "*_PASSWORD", "*_CREDENTIAL"}

func isSecretEnv(key string) bool {
	upper := strings.ToUpper(key)
	for _, pattern := range secretPatterns {
		if matched, _ := filepath.Match(pattern, upper); matched {
			return true
		}
	}
	return false
}

func (d *DockerSandbox) buildRunArgs(config SandboxConfig, command []string) []string {
	args := []string{"run", "--rm"}

	if config.WorkspaceDir != "" {
		args = append(args, "-v", config.WorkspaceDir+":/workspace")
	}
	for k, v := range config.Env {
		if !isSecretEnv(k) {
			args = append(args, "-e", k+"="+v)
		}
	}
	if config.CPULimit != "" {
		args = append(args, "--cpus", config.CPULimit)
	}
	if config.MemoryLimit != "" {
		args = append(args, "-m", config.MemoryLimit)
	}
	network := config.NetworkMode
	if network == "" {
		network = "none"
	}
	args = append(args, "--network", network)
	args = append(args, config.Image)
	args = append(args, command...)
	return args
}
