package blueprint

import "testing"

const testYAML = `
name: test-blueprint
version: "0.1"
description: "A test blueprint"
start: plan
nodes:
  plan:
    type: agentic
    config:
      prompt: "Create a plan"
  lint:
    type: deterministic
    config:
      command: "echo lint-ok"
  lint_gate:
    type: gate
    config:
      check_node: lint
  commit:
    type: deterministic
    config:
      command: "echo committed"
edges:
  - from: plan
    to: lint
  - from: lint
    to: lint_gate
  - from: lint_gate
    to: commit
    condition: pass
  - from: lint_gate
    to: plan
    condition: fail
`

func TestParseBlueprintYAML(t *testing.T) {
	bp, err := ParseBlueprintYAML([]byte(testYAML))
	if err != nil {
		t.Fatalf("ParseBlueprintYAML() error = %v", err)
	}
	if bp.Name != "test-blueprint" {
		t.Errorf("Name = %q, want %q", bp.Name, "test-blueprint")
	}
	if bp.Start != "plan" {
		t.Errorf("Start = %q, want %q", bp.Start, "plan")
	}
	if len(bp.Nodes) != 4 {
		t.Errorf("Nodes count = %d, want 4", len(bp.Nodes))
	}
	if len(bp.Edges) != 4 {
		t.Errorf("Edges count = %d, want 4", len(bp.Edges))
	}
	if bp.Nodes["plan"].Type != "agentic" {
		t.Errorf("plan type = %q, want agentic", bp.Nodes["plan"].Type)
	}
}

func TestParseBlueprintYAMLInvalid(t *testing.T) {
	_, err := ParseBlueprintYAML([]byte("not: valid: yaml: ["))
	if err == nil {
		t.Fatal("expected parse error for invalid YAML")
	}
}

func TestBuildGraph(t *testing.T) {
	bp, _ := ParseBlueprintYAML([]byte(testYAML))
	executor := &mockExecutor{output: "done"}
	g, err := bp.BuildGraph(executor)
	if err != nil {
		t.Fatalf("BuildGraph() error = %v", err)
	}
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount = %d, want 4", g.NodeCount())
	}
	if g.StartNode() != "plan" {
		t.Errorf("StartNode = %q, want plan", g.StartNode())
	}

	// Verify node types
	plan, _ := g.GetNode("plan")
	if plan.Type() != NodeTypeAgentic {
		t.Errorf("plan type = %v, want Agentic", plan.Type())
	}
	lint, _ := g.GetNode("lint")
	if lint.Type() != NodeTypeDeterministic {
		t.Errorf("lint type = %v, want Deterministic", lint.Type())
	}
	gate, _ := g.GetNode("lint_gate")
	if gate.Type() != NodeTypeGate {
		t.Errorf("lint_gate type = %v, want Gate", gate.Type())
	}

	if err := g.Validate(); err != nil {
		t.Fatalf("graph validation failed: %v", err)
	}
}

func TestBuildGraphUnknownNodeType(t *testing.T) {
	yamlData := `
name: bad
version: "0.1"
start: x
nodes:
  x:
    type: alien
edges: []
`
	bp, _ := ParseBlueprintYAML([]byte(yamlData))
	_, err := bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error for unknown node type")
	}
}
