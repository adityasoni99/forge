package sandbox

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrPoolEmpty = errors.New("warm pool: no available containers")

// WarmContainer represents a pre-created container ready for immediate use.
type WarmContainer struct {
	ContainerID  string
	Image        string
	WorkspaceDir string
	CreatedAt    time.Time
}

// WarmPool manages a set of pre-heated containers that can be acquired
// quickly instead of cold-starting new ones.
type WarmPool interface {
	Acquire(ctx context.Context, config SandboxConfig) (*WarmContainer, error)
	Release(container *WarmContainer) error
	Shutdown(ctx context.Context) error
}

// DockerWarmPool is a WarmPool backed by Docker containers.
type DockerWarmPool struct {
	runner   CommandRunner
	poolSize int
	idle     []*WarmContainer
	mu       sync.Mutex
}

func NewDockerWarmPool(runner CommandRunner, poolSize int) *DockerWarmPool {
	return &DockerWarmPool{
		runner:   runner,
		poolSize: poolSize,
	}
}

// Preheat creates poolSize containers in the background so they are
// ready when Acquire is called.
func (p *DockerWarmPool) Preheat(ctx context.Context, config SandboxConfig) {
	for i := 0; i < p.poolSize; i++ {
		go func() {
			container, err := p.createContainer(ctx, config)
			if err != nil {
				return
			}
			p.mu.Lock()
			p.idle = append(p.idle, container)
			p.mu.Unlock()
		}()
	}
}

// Acquire returns a pre-heated container matching config.Image, or
// ErrPoolEmpty if none are available.
func (p *DockerWarmPool) Acquire(_ context.Context, config SandboxConfig) (*WarmContainer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, c := range p.idle {
		if c.Image == config.Image {
			p.idle = append(p.idle[:i], p.idle[i+1:]...)
			return c, nil
		}
	}
	return nil, ErrPoolEmpty
}

// Release resets a container's workspace and returns it to the idle pool.
// If the reset fails, the container is destroyed instead.
func (p *DockerWarmPool) Release(container *WarmContainer) error {
	ctx := context.Background()
	_, _, err := p.runner.Run(ctx, "docker", "exec", container.ContainerID, "git", "clean", "-fdx")
	if err != nil {
		p.destroyContainer(ctx, container)
		return nil
	}
	_, _, err = p.runner.Run(ctx, "docker", "exec", container.ContainerID, "git", "reset", "--hard")
	if err != nil {
		p.destroyContainer(ctx, container)
		return nil
	}

	p.mu.Lock()
	p.idle = append(p.idle, container)
	p.mu.Unlock()
	return nil
}

// Shutdown stops and removes all idle containers.
func (p *DockerWarmPool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	containers := make([]*WarmContainer, len(p.idle))
	copy(containers, p.idle)
	p.idle = nil
	p.mu.Unlock()

	for _, c := range containers {
		p.destroyContainer(ctx, c)
	}
	return nil
}

func (p *DockerWarmPool) createContainer(ctx context.Context, config SandboxConfig) (*WarmContainer, error) {
	args := []string{"create", "-v", config.WorkspaceDir + ":/workspace", config.Image}
	output, exitCode, err := p.runner.Run(ctx, "docker", args...)
	if err != nil || exitCode != 0 {
		return nil, errors.New("failed to create warm container")
	}

	containerID := strings.TrimSpace(output)
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	_, startExit, err := p.runner.Run(ctx, "docker", "start", containerID)
	if err != nil || startExit != 0 {
		p.destroyContainer(ctx, &WarmContainer{ContainerID: containerID})
		return nil, errors.New("failed to start warm container")
	}

	return &WarmContainer{
		ContainerID:  containerID,
		Image:        config.Image,
		WorkspaceDir: config.WorkspaceDir,
		CreatedAt:    time.Now(),
	}, nil
}

func (p *DockerWarmPool) destroyContainer(ctx context.Context, c *WarmContainer) {
	_, _, _ = p.runner.Run(ctx, "docker", "stop", c.ContainerID)
	_, _, _ = p.runner.Run(ctx, "docker", "rm", "-f", c.ContainerID)
}
