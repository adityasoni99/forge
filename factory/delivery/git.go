package delivery

import (
	"context"
	"fmt"
	"strings"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, int, error)
}

type GitDelivery struct {
	runner CommandRunner
}

func NewGitDelivery(runner CommandRunner) *GitDelivery {
	return &GitDelivery{runner: runner}
}

func (g *GitDelivery) Deliver(ctx context.Context, workspaceDir, branch string, config DeliveryConfig) (DeliveryResult, error) {
	remote := config.Remote
	if remote == "" {
		remote = "origin"
	}

	_, exitCode, err := g.runGit(ctx, workspaceDir, "push", remote, branch)
	if err != nil || exitCode != 0 {
		return DeliveryResult{}, fmt.Errorf("git push failed (exit %d): %w", exitCode, err)
	}

	result := DeliveryResult{Branch: branch, Pushed: true}

	if config.PRTitle == "" {
		return result, nil
	}

	args := []string{"pr", "create",
		"--title", config.PRTitle,
		"--body", config.PRBody,
		"--head", branch,
	}
	if config.BaseBranch != "" {
		args = append(args, "--base", config.BaseBranch)
	}
	output, exitCode, err := g.runInDir(ctx, workspaceDir, "gh", args...)
	if err != nil || exitCode != 0 {
		return result, fmt.Errorf("gh pr create failed (exit %d): %w", exitCode, err)
	}
	result.PRURL = strings.TrimSpace(output)
	result.PRCreated = true
	return result, nil
}

func (g *GitDelivery) runGit(ctx context.Context, dir string, args ...string) (string, int, error) {
	fullArgs := append([]string{"-C", dir}, args...)
	return g.runner.Run(ctx, "git", fullArgs...)
}

func (g *GitDelivery) runInDir(ctx context.Context, dir, cmd string, args ...string) (string, int, error) {
	return g.runner.Run(ctx, cmd, args...)
}
