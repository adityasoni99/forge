package blueprint

import (
	"context"
	"strings"
	"testing"
)

// mockPermissionChecker returns decision for onlyNodeID when set; other nodes get Allow.
// When onlyNodeID is empty, every node gets decision.
type mockPermissionChecker struct {
	decision   PermissionDecision
	onlyNodeID string
}

func (m *mockPermissionChecker) Check(_ context.Context, node Node, _ *RunState) (PermissionDecision, error) {
	if m.onlyNodeID != "" && node.ID() != m.onlyNodeID {
		return PermissionAllow, nil
	}
	return m.decision, nil
}

type trackExecuteNode struct {
	id    string
	execs *int
}

func (n *trackExecuteNode) ID() string     { return n.id }
func (n *trackExecuteNode) Type() NodeType { return NodeTypeDeterministic }

func (n *trackExecuteNode) IsConcurrencySafe() bool { return true }

func (n *trackExecuteNode) Execute(context.Context, *RunState) (NodeResult, error) {
	*n.execs++
	return NodeResult{Status: NodeStatusPassed}, nil
}

func TestPermissionDenyBlocksExecution(t *testing.T) {
	var aExecs, bExecs, cExecs int
	g := NewGraph()
	_ = g.AddNode(&trackExecuteNode{id: "a", execs: &aExecs})
	_ = g.AddNode(&trackExecuteNode{id: "b", execs: &bExecs})
	_ = g.AddNode(&trackExecuteNode{id: "c", execs: &cExecs})
	_ = g.AddEdge(Edge{From: "a", To: "b"})
	_ = g.AddEdge(Edge{From: "b", To: "c"})
	_ = g.SetStartNode("a")

	eng := NewEngine(g, "perm-test")
	eng.SetPermissionChecker(&mockPermissionChecker{
		decision:   PermissionDeny,
		onlyNodeID: "b",
	})

	_, err := eng.Execute(context.Background())
	if err == nil {
		t.Fatal("expected permission error")
	}
	if !strings.Contains(err.Error(), `permission denied for node "b"`) {
		t.Fatalf("error = %q, want permission denied for b", err.Error())
	}
	if aExecs != 1 {
		t.Errorf("node a executions = %d, want 1", aExecs)
	}
	if bExecs != 0 {
		t.Errorf("node b executions = %d, want 0 (denied)", bExecs)
	}
	if cExecs != 0 {
		t.Errorf("node c executions = %d, want 0", cExecs)
	}
}

func TestPermissionAllowProceeds(t *testing.T) {
	engine := NewEngine(buildLinearGraph(), "test-blueprint")
	engine.SetPermissionChecker(TrustedSourceChecker{})
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

func TestPermissionAskHeadlessDenies(t *testing.T) {
	var execs int
	g := NewGraph()
	_ = g.AddNode(&trackExecuteNode{id: "solo", execs: &execs})
	_ = g.SetStartNode("solo")

	eng := NewEngine(g, "perm-test")
	eng.SetPermissionChecker(&mockPermissionChecker{decision: PermissionAsk})
	eng.SetHeadless(true)

	_, err := eng.Execute(context.Background())
	if err == nil {
		t.Fatal("expected headless permission error")
	}
	if !strings.Contains(err.Error(), `permission denied (headless mode) for node "solo"`) {
		t.Fatalf("error = %q", err.Error())
	}
	if execs != 0 {
		t.Errorf("executions = %d, want 0", execs)
	}
}

func TestPermissionAskNonHeadlessAllows(t *testing.T) {
	var execs int
	g := NewGraph()
	_ = g.AddNode(&trackExecuteNode{id: "solo", execs: &execs})
	_ = g.SetStartNode("solo")

	eng := NewEngine(g, "perm-test")
	eng.SetPermissionChecker(&mockPermissionChecker{decision: PermissionAsk})
	eng.SetHeadless(false)

	state, err := eng.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if execs != 1 {
		t.Errorf("executions = %d, want 1", execs)
	}
	if state.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed", state.Status)
	}
}

func TestNilCheckerAllowsAll(t *testing.T) {
	engine := NewEngine(buildLinearGraph(), "test-blueprint")
	if engine.permissionChecker != nil {
		t.Fatal("NewEngine should leave permissionChecker nil")
	}
	state, err := engine.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if state.Status != NodeStatusPassed || len(state.NodeResults) != 3 {
		t.Fatalf("want full linear run: status=%v results=%d", state.Status, len(state.NodeResults))
	}
}

func TestPermissionDecisionString(t *testing.T) {
	tests := []struct {
		d    PermissionDecision
		want string
	}{
		{PermissionAllow, "allow"},
		{PermissionDeny, "deny"},
		{PermissionAsk, "ask"},
		{PermissionDecision(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.d.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.d, got, tt.want)
		}
	}
}
