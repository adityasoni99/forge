package blueprint

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func buildFanInConcurrentGraph() *Graph {
	g := NewGraph()
	_ = g.AddNode(NewDeterministicNode("start", "echo start"))
	_ = g.AddNode(NewDeterministicNode("b1", "echo b1-out"))
	_ = g.AddNode(NewDeterministicNode("b2", "echo b2-out"))
	_ = g.AddNode(NewDeterministicNode("b3", "echo b3-out"))
	_ = g.AddNode(NewDeterministicNode("final", "echo final"))
	_ = g.AddEdge(Edge{From: "start", To: "b1"})
	_ = g.AddEdge(Edge{From: "start", To: "b2"})
	_ = g.AddEdge(Edge{From: "start", To: "b3"})
	_ = g.AddEdge(Edge{From: "b1", To: "final"})
	_ = g.AddEdge(Edge{From: "b2", To: "final"})
	_ = g.AddEdge(Edge{From: "b3", To: "final"})
	_ = g.SetStartNode("start")
	return g
}

func TestConcurrentExecution(t *testing.T) {
	engine := NewEngine(buildFanInConcurrentGraph(), "concurrent-fan-in")
	state, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Status != NodeStatusPassed {
		t.Fatalf("Status = %v, want Passed", state.Status)
	}
	for _, id := range []string{"start", "b1", "b2", "b3", "final"} {
		if _, ok := state.NodeResults[id]; !ok {
			t.Errorf("missing result for %q", id)
		}
	}
}

func TestConcurrentFallbackToSequential(t *testing.T) {
	g := NewGraph()
	exec := &mockExecutor{output: "ok"}
	agent := NewAgenticNode("agent", "p", nil, exec)
	_ = g.AddNode(NewDeterministicNode("start", "echo s"))
	_ = g.AddNode(NewDeterministicNode("det", "echo det"))
	_ = g.AddNode(agent)
	_ = g.AddNode(NewDeterministicNode("done", "echo done"))
	_ = g.AddEdge(Edge{From: "start", To: "det"})
	_ = g.AddEdge(Edge{From: "start", To: "agent"})
	_ = g.AddEdge(Edge{From: "det", To: "done"})
	_ = g.SetStartNode("start")

	engine := NewEngine(g, "fallback")
	state, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if _, ok := state.NodeResults["det"]; !ok {
		t.Fatal("expected sequential first branch (det) to run")
	}
	if _, ok := state.NodeResults["agent"]; ok {
		t.Fatal("expected agent branch not to run when falling back to first edge only")
	}
}

// errorExecuteConcurrentNode is like errorExecuteNode but marked concurrency-safe for parallel-path tests.
type errorExecuteConcurrentNode struct{ id string }

func (n *errorExecuteConcurrentNode) ID() string     { return n.id }
func (n *errorExecuteConcurrentNode) Type() NodeType { return NodeTypeDeterministic }
func (n *errorExecuteConcurrentNode) Execute(context.Context, *RunState) (NodeResult, error) {
	return NodeResult{}, errConcurrentTestFail
}
func (n *errorExecuteConcurrentNode) IsConcurrencySafe() bool { return true }

var errConcurrentTestFail = errors.New("execute failed (concurrent test)")

func TestConcurrentErrorCancelsOthers(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(NewDeterministicNode("start", "echo s"))
	_ = g.AddNode(NewDeterministicNode("ok1", "echo a"))
	_ = g.AddNode(&errorExecuteConcurrentNode{id: "bad"})
	_ = g.AddNode(NewDeterministicNode("ok2", "echo b"))
	_ = g.AddNode(NewDeterministicNode("join", "echo j"))
	_ = g.AddEdge(Edge{From: "start", To: "ok1"})
	_ = g.AddEdge(Edge{From: "start", To: "bad"})
	_ = g.AddEdge(Edge{From: "start", To: "ok2"})
	_ = g.AddEdge(Edge{From: "ok1", To: "join"})
	_ = g.AddEdge(Edge{From: "bad", To: "join"})
	_ = g.AddEdge(Edge{From: "ok2", To: "join"})
	_ = g.SetStartNode("start")

	engine := NewEngine(g, "err-cancel")
	_, err := engine.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error from parallel branch")
	}
	if !strings.Contains(err.Error(), "bad") && !errors.Is(err, errConcurrentTestFail) {
		t.Errorf("error = %v, want failure from bad node", err)
	}
}

func TestConcurrentRunStateMergeOrder(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(NewDeterministicNode("start", "echo s"))
	_ = g.AddNode(NewDeterministicNode("slow", "sleep 0.08; echo slow"))
	_ = g.AddNode(NewDeterministicNode("fast", "sleep 0.01; echo fast"))
	_ = g.AddNode(NewDeterministicNode("mid", "sleep 0.04; echo mid"))
	_ = g.AddNode(NewDeterministicNode("join", "echo j"))
	_ = g.AddEdge(Edge{From: "start", To: "slow"})
	_ = g.AddEdge(Edge{From: "start", To: "fast"})
	_ = g.AddEdge(Edge{From: "start", To: "mid"})
	_ = g.AddEdge(Edge{From: "slow", To: "join"})
	_ = g.AddEdge(Edge{From: "fast", To: "join"})
	_ = g.AddEdge(Edge{From: "mid", To: "join"})
	_ = g.SetStartNode("start")

	var mu sync.Mutex
	var applyOrder []string
	testRecordParallelApply = func(id string) {
		mu.Lock()
		applyOrder = append(applyOrder, id)
		mu.Unlock()
	}
	t.Cleanup(func() { testRecordParallelApply = nil })

	engine := NewEngine(g, "merge-order")
	_, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	want := []string{"slow", "fast", "mid"}
	if len(applyOrder) != len(want) {
		t.Fatalf("applyOrder = %v, want len %d", applyOrder, len(want))
	}
	for i, id := range want {
		if applyOrder[i] != id {
			t.Fatalf("applyOrder[%d] = %q, want %q (merge uses declaration order, not completion order)", i, applyOrder[i], id)
		}
	}
}

func TestSingleNextNodeNoConcurrencyOverhead(t *testing.T) {
	engine := NewEngine(buildLinearGraph(), "linear")
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
}
