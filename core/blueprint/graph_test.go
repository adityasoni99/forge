package blueprint

import (
	"context"
	"testing"
)

// stubNode is a minimal Node for graph tests (no execution logic needed)
type stubNode struct {
	id       string
	nodeType NodeType
}

func (s *stubNode) ID() string     { return s.id }
func (s *stubNode) Type() NodeType { return s.nodeType }
func (s *stubNode) Execute(_ context.Context, _ *RunState) (NodeResult, error) {
	return NodeResult{Status: NodeStatusPassed}, nil
}

func (s *stubNode) IsConcurrencySafe() bool { return false }

func TestGraphAddNode(t *testing.T) {
	g := NewGraph()
	err := g.AddNode(&stubNode{id: "n1", nodeType: NodeTypeAgentic})
	if err != nil {
		t.Fatalf("AddNode() error = %v", err)
	}
	got, ok := g.GetNode("n1")
	if !ok {
		t.Fatal("GetNode(n1) not found")
	}
	if got.ID() != "n1" {
		t.Errorf("got ID = %q, want %q", got.ID(), "n1")
	}
}

func TestGraphAddNodeDuplicate(t *testing.T) {
	g := NewGraph()
	node := &stubNode{id: "n1", nodeType: NodeTypeAgentic}
	_ = g.AddNode(node)
	err := g.AddNode(node)
	if err == nil {
		t.Fatal("expected error for duplicate node")
	}
}

func TestGraphNodeCount(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	_ = g.AddNode(&stubNode{id: "b"})
	if got := g.NodeCount(); got != 2 {
		t.Errorf("NodeCount() = %d, want 2", got)
	}
}

func TestGraphAddEdge(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	_ = g.AddNode(&stubNode{id: "b"})
	err := g.AddEdge(Edge{From: "a", To: "b"})
	if err != nil {
		t.Fatalf("AddEdge() error = %v", err)
	}
	next := g.NextNodes("a", "")
	if len(next) != 1 || next[0] != "b" {
		t.Errorf("NextNodes(a, '') = %v, want [b]", next)
	}
}

func TestGraphAddEdgeInvalidSource(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "b"})
	err := g.AddEdge(Edge{From: "missing", To: "b"})
	if err == nil {
		t.Fatal("expected error for edge from missing node")
	}
}

func TestGraphAddEdgeInvalidTarget(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	err := g.AddEdge(Edge{From: "a", To: "missing"})
	if err == nil {
		t.Fatal("expected error for edge to missing node")
	}
}

func TestGraphNextNodesConditional(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "gate", nodeType: NodeTypeGate})
	_ = g.AddNode(&stubNode{id: "pass-target"})
	_ = g.AddNode(&stubNode{id: "fail-target"})
	_ = g.AddEdge(Edge{From: "gate", To: "pass-target", Condition: "pass"})
	_ = g.AddEdge(Edge{From: "gate", To: "fail-target", Condition: "fail"})

	passNext := g.NextNodes("gate", "pass")
	if len(passNext) != 1 || passNext[0] != "pass-target" {
		t.Errorf("NextNodes(gate, pass) = %v, want [pass-target]", passNext)
	}
	failNext := g.NextNodes("gate", "fail")
	if len(failNext) != 1 || failNext[0] != "fail-target" {
		t.Errorf("NextNodes(gate, fail) = %v, want [fail-target]", failNext)
	}
}

func TestGraphValidateValid(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	_ = g.AddNode(&stubNode{id: "b"})
	_ = g.AddEdge(Edge{From: "a", To: "b"})
	_ = g.SetStartNode("a")
	if err := g.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestGraphValidateNoStartNode(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for missing start node")
	}
}

func TestGraphSetStartNodeMissing(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	if err := g.SetStartNode("ghost"); err == nil {
		t.Fatal("expected error when start node id is not in graph")
	}
}

// Corrupt start reference exercises Validate's start-not-in-graph branch.
// Normal API cannot produce this state (SetStartNode checks membership).
func TestGraphValidateStartNotInNodes(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	g.startNode = "missing"
	if err := g.Validate(); err == nil {
		t.Fatal("expected error when start id is not in nodes map")
	}
}

func TestGraphValidateUnreachableNode(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "a"})
	_ = g.AddNode(&stubNode{id: "b"}) // no edge to b
	_ = g.SetStartNode("a")
	if err := g.Validate(); err == nil {
		t.Fatal("expected error for unreachable node b")
	}
}

func TestGraphValidateGateLoop(t *testing.T) {
	// Gate loops (retry) are valid: implement -> test -> gate --fail--> implement
	g := NewGraph()
	_ = g.AddNode(&stubNode{id: "impl", nodeType: NodeTypeAgentic})
	_ = g.AddNode(&stubNode{id: "test", nodeType: NodeTypeDeterministic})
	_ = g.AddNode(&stubNode{id: "gate", nodeType: NodeTypeGate})
	_ = g.AddNode(&stubNode{id: "done", nodeType: NodeTypeDeterministic})
	_ = g.AddEdge(Edge{From: "impl", To: "test"})
	_ = g.AddEdge(Edge{From: "test", To: "gate"})
	_ = g.AddEdge(Edge{From: "gate", To: "done", Condition: "pass"})
	_ = g.AddEdge(Edge{From: "gate", To: "impl", Condition: "fail"})
	_ = g.SetStartNode("impl")
	if err := g.Validate(); err != nil {
		t.Fatalf("gate loops should be valid, got: %v", err)
	}
}
