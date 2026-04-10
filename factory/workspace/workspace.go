package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Workspace struct {
	Dir     string
	Branch  string
	RepoDir string
}

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) Create(ctx context.Context, repoDir, runID string) (*Workspace, error) {
	branch := "forge/run-" + runID
	worktreeDir := filepath.Join(os.TempDir(), "forge-workspace-"+runID)

	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", branch, worktreeDir)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return &Workspace{
		Dir:     worktreeDir,
		Branch:  branch,
		RepoDir: repoDir,
	}, nil
}

func (m *Manager) Destroy(ctx context.Context, ws *Workspace) error {
	os.RemoveAll(ws.Dir)
	cmd := exec.CommandContext(ctx, "git", "worktree", "prune")
	cmd.Dir = ws.RepoDir
	cmd.Run()
	cmd = exec.CommandContext(ctx, "git", "branch", "-D", ws.Branch)
	cmd.Dir = ws.RepoDir
	cmd.Run()
	return nil
}
