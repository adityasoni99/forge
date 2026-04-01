package blueprint

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// orderedHooks appends "hookID:event:nodeID" for each OnEvent invocation (registration order per event).
type orderedHooks struct {
	id      int
	records *[]string
}

func (o *orderedHooks) OnEvent(_ context.Context, event HookEvent, data HookData) HookResult {
	*o.records = append(*o.records, fmt.Sprintf("%d:%s:%s", o.id, event, data.NodeID))
	return DefaultHookResult()
}

func TestHooksFireInOrder(t *testing.T) {
	var records []string
	h0 := &orderedHooks{id: 0, records: &records}
	h1 := &orderedHooks{id: 1, records: &records}
	engine := NewEngine(buildLinearGraph(), "bp")
	engine.RegisterHook(h0)
	engine.RegisterHook(h1)
	_, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for i := 0; i < len(records)-1; i += 2 {
		ev0 := strings.SplitN(records[i], ":", 3)
		ev1 := strings.SplitN(records[i+1], ":", 3)
		if ev0[1] != ev1[1] {
			t.Fatalf("pair %d: events differ %q vs %q", i, ev0[1], ev1[1])
		}
		if ev0[0] != "0" || ev1[0] != "1" {
			t.Fatalf("want hook 0 then hook 1, got %q then %q", records[i], records[i+1])
		}
	}
	if len(records)%2 != 0 {
		t.Fatalf("expected even number of hook calls, got %d", len(records))
	}
}

type abortOnNodeHook struct {
	targetNode string
}

func (a *abortOnNodeHook) OnEvent(_ context.Context, event HookEvent, data HookData) HookResult {
	if event == HookPreNodeExec && data.NodeID == a.targetNode {
		return HookResult{Continue: false, Error: errors.New("blocked by hook")}
	}
	return DefaultHookResult()
}

func TestHookAbortPreNodeExec(t *testing.T) {
	engine := NewEngine(buildLinearGraph(), "bp")
	engine.RegisterHook(&abortOnNodeHook{targetNode: "lint"})
	state, err := engine.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error from hook abort")
	}
	if !strings.Contains(err.Error(), "blocked by hook") {
		t.Errorf("error = %v, want blocked by hook", err)
	}
	if state.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", state.Status)
	}
	if state.EndTime.IsZero() {
		t.Error("EndTime should be set on abort")
	}
	if _, ok := state.NodeResults["lint"]; ok {
		t.Error("lint should not have executed")
	}
	if _, ok := state.NodeResults["plan"]; !ok {
		t.Error("plan should have completed before abort")
	}
}

type runStartCompleteHook struct {
	t            *testing.T
	seenPrePlan  bool
	seenStart    bool
	seenComplete bool
}

func (r *runStartCompleteHook) OnEvent(_ context.Context, event HookEvent, data HookData) HookResult {
	switch event {
	case HookRunStart:
		if r.seenPrePlan {
			r.t.Fatalf("HookRunStart after node activity")
		}
		r.seenStart = true
	case HookPreNodeExec:
		if data.NodeID == "plan" {
			if !r.seenStart {
				r.t.Fatalf("HookPreNodeExec(plan) before HookRunStart")
			}
			r.seenPrePlan = true
		}
	case HookRunComplete:
		if !r.seenPrePlan {
			r.t.Fatalf("HookRunComplete before any node")
		}
		r.seenComplete = true
	}
	return DefaultHookResult()
}

func TestHookRunStartAndComplete(t *testing.T) {
	h := &runStartCompleteHook{t: t}
	engine := NewEngine(buildLinearGraph(), "bp")
	engine.RegisterHook(h)
	_, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !h.seenStart || !h.seenComplete {
		t.Fatalf("seenStart=%v seenComplete=%v", h.seenStart, h.seenComplete)
	}
}

type captureRunErrorHook struct {
	runErrors []HookEvent
}

func (c *captureRunErrorHook) OnEvent(_ context.Context, event HookEvent, _ HookData) HookResult {
	if event == HookRunError {
		c.runErrors = append(c.runErrors, event)
	}
	return DefaultHookResult()
}

func TestHookRunError(t *testing.T) {
	g := NewGraph()
	executor := &mockExecutor{output: "done"}
	_ = g.AddNode(NewAgenticNode("implement", "implement", nil, executor))
	_ = g.AddNode(NewDeterministicNode("test", "exit 1"))
	_ = g.AddNode(NewGateNode("gate", "test"))
	_ = g.AddNode(NewDeterministicNode("commit", "echo ok"))
	_ = g.AddEdge(Edge{From: "implement", To: "test"})
	_ = g.AddEdge(Edge{From: "test", To: "gate"})
	_ = g.AddEdge(Edge{From: "gate", To: "commit", Condition: "pass"})
	_ = g.AddEdge(Edge{From: "gate", To: "implement", Condition: "fail"})
	_ = g.SetStartNode("implement")

	h := &captureRunErrorHook{}
	engine := NewEngine(g, "bp")
	engine.RegisterHook(h)
	engine.SetMaxIterations(9)
	_, err := engine.Execute(context.Background())
	if err == nil {
		t.Fatal("expected max iterations error")
	}
	if len(h.runErrors) == 0 {
		t.Fatal("expected at least one HookRunError")
	}
}

func TestNoHooksDefaultBehavior(t *testing.T) {
	engine := NewEngine(buildLinearGraph(), "test-blueprint")
	state, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed", state.Status)
	}
	if len(state.NodeResults) != 3 {
		t.Errorf("node results count = %d, want 3", len(state.NodeResults))
	}
	for _, id := range []string{"plan", "lint", "commit"} {
		if _, ok := state.NodeResults[id]; !ok {
			t.Errorf("missing result for %q", id)
		}
	}
}

type dataAssertHook struct {
	t *testing.T
}

func (d *dataAssertHook) OnEvent(_ context.Context, event HookEvent, data HookData) HookResult {
	if data.RunState == nil {
		d.t.Fatalf("%s: RunState is nil", event)
	}
	if data.RunState.BlueprintName != "data-bp" {
		d.t.Fatalf("wrong blueprint name")
	}
	switch event {
	case HookRunStart, HookRunComplete, HookRunError:
		// NodeID may be empty for RunStart/Complete; RunError may have node set
		if event == HookRunStart && data.NodeID != "" {
			d.t.Errorf("HookRunStart: want empty NodeID, got %q", data.NodeID)
		}
	case HookPreNodeExec:
		if data.NodeID == "" {
			d.t.Fatal("HookPreNodeExec: empty NodeID")
		}
		if data.Result != nil {
			d.t.Fatal("HookPreNodeExec: Result should be nil")
		}
	case HookPostNodeExec, HookPreEdgeTraversal:
		if data.NodeID == "" {
			d.t.Fatalf("%s: empty NodeID", event)
		}
		if data.Result == nil {
			d.t.Fatalf("%s: Result should be non-nil", event)
		}
	case HookGateEvaluated:
		if data.NodeType != NodeTypeGate {
			d.t.Fatalf("HookGateEvaluated: want gate type, got %v", data.NodeType)
		}
		if data.Result == nil {
			d.t.Fatal("HookGateEvaluated: Result nil")
		}
	}
	return DefaultHookResult()
}

func TestHookReceivesCorrectData(t *testing.T) {
	h := &dataAssertHook{t: t}
	engine := NewEngine(buildGatedGraph(), "data-bp")
	engine.RegisterHook(h)
	_, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestHookEventString(t *testing.T) {
	if HookRunStart.String() != "HookRunStart" {
		t.Errorf("HookRunStart.String() = %q", HookRunStart.String())
	}
	if HookRunError.String() != "HookRunError" {
		t.Errorf("HookRunError.String() = %q", HookRunError.String())
	}
}

func TestHookAbortDefaultError(t *testing.T) {
	block := &abortOnNodeHookNoErr{target: "plan"}
	engine := NewEngine(buildLinearGraph(), "bp")
	engine.RegisterHook(block)
	_, err := engine.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errHookAborted) {
		t.Errorf("err = %v, want errHookAborted", err)
	}
}

type abortOnNodeHookNoErr struct {
	target string
}

func (a *abortOnNodeHookNoErr) OnEvent(_ context.Context, event HookEvent, data HookData) HookResult {
	if event == HookPreNodeExec && data.NodeID == a.target {
		return HookResult{Continue: false}
	}
	return DefaultHookResult()
}
