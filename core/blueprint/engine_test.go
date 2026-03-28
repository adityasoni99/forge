package blueprint

import (
	"context"
	"testing"
)

func buildLinearGraph() *Graph {
	g := NewGraph()
	executor := &mockExecutor{output: "done"}
	_ = g.AddNode(NewAgenticNode("plan", "plan", nil, executor))
	_ = g.AddNode(NewDeterministicNode("lint", "echo lint-ok"))
	_ = g.AddNode(NewDeterministicNode("commit", "echo committed"))
	_ = g.AddEdge(Edge{From: "plan", To: "lint"})
	_ = g.AddEdge(Edge{From: "lint", To: "commit"})
	_ = g.SetStartNode("plan")
	return g
}

func TestEngineLinearExecution(t *testing.T) {
	engine := NewEngine(buildLinearGraph())
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

func buildGatedGraph() *Graph {
	g := NewGraph()
	executor := &mockExecutor{output: "done"}
	_ = g.AddNode(NewAgenticNode("implement", "implement", nil, executor))
	_ = g.AddNode(NewDeterministicNode("test", "echo test-ok"))
	_ = g.AddNode(NewGateNode("test-gate", "test"))
	_ = g.AddNode(NewDeterministicNode("commit", "echo committed"))
	_ = g.AddEdge(Edge{From: "implement", To: "test"})
	_ = g.AddEdge(Edge{From: "test", To: "test-gate"})
	_ = g.AddEdge(Edge{From: "test-gate", To: "commit", Condition: "pass"})
	_ = g.AddEdge(Edge{From: "test-gate", To: "implement", Condition: "fail"})
	_ = g.SetStartNode("implement")
	return g
}

func TestEngineGatePass(t *testing.T) {
	engine := NewEngine(buildGatedGraph())
	state, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed", state.Status)
	}
	if _, ok := state.NodeResults["commit"]; !ok {
		t.Error("expected commit to be executed after gate pass")
	}
}

func TestEngineGateFailMaxIterations(t *testing.T) {
	g := NewGraph()
	executor := &mockExecutor{output: "done"}
	_ = g.AddNode(NewAgenticNode("implement", "implement", nil, executor))
	_ = g.AddNode(NewDeterministicNode("test", "exit 1")) // always fails
	_ = g.AddNode(NewGateNode("gate", "test"))
	_ = g.AddNode(NewDeterministicNode("commit", "echo ok"))
	_ = g.AddEdge(Edge{From: "implement", To: "test"})
	_ = g.AddEdge(Edge{From: "test", To: "gate"})
	_ = g.AddEdge(Edge{From: "gate", To: "commit", Condition: "pass"})
	_ = g.AddEdge(Edge{From: "gate", To: "implement", Condition: "fail"})
	_ = g.SetStartNode("implement")

	engine := NewEngine(g)
	engine.SetMaxIterations(9) // 3 loop iterations * 3 nodes = 9
	_, err := engine.Execute(context.Background())
	if err == nil {
		t.Fatal("expected max iterations error")
	}
}

func TestEngineContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	engine := NewEngine(buildLinearGraph())
	_, err := engine.Execute(ctx)
	if err == nil {
		t.Fatal("expected context cancelled error")
	}
}

func TestEngineInvalidGraph(t *testing.T) {
	g := NewGraph() // empty, no start node
	engine := NewEngine(g)
	_, err := engine.Execute(context.Background())
	if err == nil {
		t.Fatal("expected invalid graph error")
	}
}
