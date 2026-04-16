package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileSessionLogEmitAndGetEvents(t *testing.T) {
	dir := t.TempDir()
	log := NewFileSessionLog(dir)
	ctx := context.Background()

	ev1 := SessionEvent{
		Timestamp: time.Now(),
		RunID:     "run-abc",
		Type:      EventWorkspaceCreated,
		Data:      map[string]interface{}{"branch": "forge/run-abc"},
	}
	ev2 := SessionEvent{
		Timestamp: time.Now(),
		RunID:     "run-abc",
		Type:      EventSandboxStarted,
		NodeID:    "build",
	}

	if err := log.Emit(ctx, ev1); err != nil {
		t.Fatalf("Emit ev1: %v", err)
	}
	if err := log.Emit(ctx, ev2); err != nil {
		t.Fatalf("Emit ev2: %v", err)
	}

	events, err := log.GetEvents(ctx, "run-abc")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Type != EventWorkspaceCreated {
		t.Errorf("events[0].Type = %q, want %q", events[0].Type, EventWorkspaceCreated)
	}
	if events[1].Type != EventSandboxStarted {
		t.Errorf("events[1].Type = %q, want %q", events[1].Type, EventSandboxStarted)
	}
}

func TestFileSessionLogGetEventsEmpty(t *testing.T) {
	dir := t.TempDir()
	log := NewFileSessionLog(dir)
	ctx := context.Background()

	events, err := log.GetEvents(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("got %d events, want 0", len(events))
	}
}

func TestFileSessionLogCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	log := NewFileSessionLog(dir)
	ctx := context.Background()

	ev := SessionEvent{RunID: "run-x", Type: EventRunComplete, Timestamp: time.Now()}
	if err := log.Emit(ctx, ev); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("expected sessions directory to be created")
	}
}

func TestFileSessionLogIsolatesRuns(t *testing.T) {
	dir := t.TempDir()
	log := NewFileSessionLog(dir)
	ctx := context.Background()

	if err := log.Emit(ctx, SessionEvent{RunID: "run-1", Type: EventRunComplete, Timestamp: time.Now()}); err != nil {
		t.Fatalf("Emit run-1: %v", err)
	}
	if err := log.Emit(ctx, SessionEvent{RunID: "run-2", Type: EventRunError, Timestamp: time.Now()}); err != nil {
		t.Fatalf("Emit run-2: %v", err)
	}

	events1, err := log.GetEvents(ctx, "run-1")
	if err != nil {
		t.Fatalf("GetEvents run-1: %v", err)
	}
	events2, err := log.GetEvents(ctx, "run-2")
	if err != nil {
		t.Fatalf("GetEvents run-2: %v", err)
	}

	if len(events1) != 1 {
		t.Errorf("run-1: got %d events, want 1", len(events1))
	}
	if len(events2) != 1 {
		t.Errorf("run-2: got %d events, want 1", len(events2))
	}
	if events1[0].Type != EventRunComplete {
		t.Errorf("run-1 type = %q, want %q", events1[0].Type, EventRunComplete)
	}
	if events2[0].Type != EventRunError {
		t.Errorf("run-2 type = %q, want %q", events2[0].Type, EventRunError)
	}
}
