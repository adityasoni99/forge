package orchestrator

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SessionEventType identifies the kind of lifecycle event recorded during a
// pipeline run.
type SessionEventType string

// Pipeline lifecycle event types.
const (
	EventWorkspaceCreated SessionEventType = "workspace_created"
	EventSandboxStarted   SessionEventType = "sandbox_started"
	EventNodeStarted      SessionEventType = "node_started"
	EventNodeCompleted    SessionEventType = "node_completed"
	EventDeliveryComplete SessionEventType = "delivery_complete"
	EventRunError         SessionEventType = "run_error"
	EventRunComplete      SessionEventType = "run_complete"
)

// SessionEvent is a single timestamped record of something that happened
// during a pipeline run.
type SessionEvent struct {
	ID        string                 `json:"id"`
	Timestamp time.Time              `json:"timestamp"`
	RunID     string                 `json:"run_id"`
	Type      SessionEventType       `json:"type"`
	NodeID    string                 `json:"node_id,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// SessionLog persists and retrieves pipeline run events.
type SessionLog interface {
	Emit(ctx context.Context, event SessionEvent) error
	GetEvents(ctx context.Context, runID string) ([]SessionEvent, error)
}

// FileSessionLog is a SessionLog backed by one JSONL file per run, stored
// under a single directory.
type FileSessionLog struct {
	dir string
}

// NewFileSessionLog returns a FileSessionLog that writes run logs into dir.
// The directory is created on first Emit if it does not already exist.
func NewFileSessionLog(dir string) *FileSessionLog {
	return &FileSessionLog{dir: dir}
}

func (f *FileSessionLog) logPath(runID string) string {
	return filepath.Join(f.dir, runID+".jsonl")
}

// Emit appends a JSON-encoded event to the run's log file, creating the
// file and parent directory as needed.
func (f *FileSessionLog) Emit(_ context.Context, event SessionEvent) error {
	if err := os.MkdirAll(f.dir, 0o755); err != nil {
		return fmt.Errorf("session: create dir: %w", err)
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("%d", event.Timestamp.UnixNano())
	}
	file, err := os.OpenFile(f.logPath(event.RunID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("session: open: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("session: marshal: %w", err)
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("session: write: %w", err)
	}
	return nil
}

// GetEvents reads all events for a given run. If the log file does not exist,
// it returns nil with no error.
func (f *FileSessionLog) GetEvents(_ context.Context, runID string) ([]SessionEvent, error) {
	file, err := os.Open(f.logPath(runID))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session: open: %w", err)
	}
	defer file.Close()

	var events []SessionEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var ev SessionEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			log.Printf("session: skip malformed event in %s: %v", runID, err)
			continue
		}
		events = append(events, ev)
	}
	return events, scanner.Err()
}
