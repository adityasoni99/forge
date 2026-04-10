package delivery

import (
	"context"
	"fmt"
	"testing"
)

type mockCmdRunner struct {
	calls []runCall
	idx   int
}

type runCall struct {
	output   string
	exitCode int
	err      error
}

func (m *mockCmdRunner) Run(_ context.Context, name string, args ...string) (string, int, error) {
	if m.idx >= len(m.calls) {
		return "", -1, fmt.Errorf("unexpected call #%d: %s %v", m.idx, name, args)
	}
	call := m.calls[m.idx]
	m.idx++
	return call.output, call.exitCode, call.err
}

func TestGitDeliveryPushAndPR(t *testing.T) {
	runner := &mockCmdRunner{calls: []runCall{
		{output: "", exitCode: 0},
		{output: "https://github.com/user/repo/pull/42\n", exitCode: 0},
	}}
	gd := NewGitDelivery(runner)
	result, err := gd.Deliver(context.Background(), "/tmp/workspace", "forge/run-123", DeliveryConfig{
		Remote:     "origin",
		BaseBranch: "main",
		PRTitle:    "forge: Fix the bug",
		PRBody:     "Automated by Forge",
	})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if !result.Pushed {
		t.Error("expected Pushed=true")
	}
	if !result.PRCreated {
		t.Error("expected PRCreated=true")
	}
	if result.PRURL != "https://github.com/user/repo/pull/42" {
		t.Errorf("PRURL = %q, want https://github.com/user/repo/pull/42", result.PRURL)
	}
}

func TestGitDeliveryPushFails(t *testing.T) {
	runner := &mockCmdRunner{calls: []runCall{
		{output: "rejected", exitCode: 1},
	}}
	gd := NewGitDelivery(runner)
	_, err := gd.Deliver(context.Background(), "/tmp/ws", "forge/run-1", DeliveryConfig{
		Remote: "origin",
	})
	if err == nil {
		t.Fatal("expected error when push fails")
	}
}

func TestGitDeliveryNoPR(t *testing.T) {
	runner := &mockCmdRunner{calls: []runCall{
		{output: "", exitCode: 0},
	}}
	gd := NewGitDelivery(runner)
	result, err := gd.Deliver(context.Background(), "/tmp/ws", "forge/run-1", DeliveryConfig{
		Remote: "origin",
	})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if !result.Pushed {
		t.Error("expected Pushed=true")
	}
	if result.PRCreated {
		t.Error("expected PRCreated=false when no PRTitle")
	}
}

func TestGitDeliveryPRCreateFails(t *testing.T) {
	runner := &mockCmdRunner{calls: []runCall{
		{output: "", exitCode: 0},
		{output: "error", exitCode: 1},
	}}
	gd := NewGitDelivery(runner)
	_, err := gd.Deliver(context.Background(), "/tmp/ws", "forge/run-1", DeliveryConfig{
		Remote:  "origin",
		PRTitle: "forge: test",
	})
	if err == nil {
		t.Fatal("expected error when PR creation fails")
	}
}

func TestGitDeliveryDefaultRemote(t *testing.T) {
	runner := &mockCmdRunner{calls: []runCall{
		{output: "", exitCode: 0},
	}}
	gd := NewGitDelivery(runner)
	result, err := gd.Deliver(context.Background(), "/tmp/ws", "forge/run-1", DeliveryConfig{})
	if err != nil {
		t.Fatalf("Deliver: %v", err)
	}
	if !result.Pushed {
		t.Error("expected Pushed=true with default remote")
	}
}

func TestGitDeliveryPushError(t *testing.T) {
	runner := &mockCmdRunner{calls: []runCall{
		{output: "", exitCode: 0, err: fmt.Errorf("network error")},
	}}
	gd := NewGitDelivery(runner)
	_, err := gd.Deliver(context.Background(), "/tmp/ws", "forge/run-1", DeliveryConfig{
		Remote: "origin",
	})
	if err == nil {
		t.Fatal("expected error when push returns error")
	}
}
