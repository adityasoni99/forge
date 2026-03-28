package blueprint

import (
	"os"
	"testing"
)

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

func TestParseBlueprintYAMLMissingName(t *testing.T) {
	yamlData := `
version: "0.1"
start: a
nodes:
  a:
    type: deterministic
    config:
      command: "echo"
edges: []
`
	_, err := ParseBlueprintYAML([]byte(yamlData))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseBlueprintYAMLMissingStart(t *testing.T) {
	yamlData := `
name: no-start
version: "0.1"
nodes:
  a:
    type: deterministic
    config:
      command: "echo"
edges: []
`
	_, err := ParseBlueprintYAML([]byte(yamlData))
	if err == nil {
		t.Fatal("expected error for missing start")
	}
}

func TestParseBlueprintYAMLNoNodes(t *testing.T) {
	yamlData := `
name: empty
version: "0.1"
start: nowhere
nodes: {}
edges: []
`
	_, err := ParseBlueprintYAML([]byte(yamlData))
	if err == nil {
		t.Fatal("expected error when blueprint has no nodes")
	}
}

func TestBuildGraphAgenticMissingOrEmptyPrompt(t *testing.T) {
	cases := []string{
		`
name: t
version: "0.1"
start: x
nodes:
  x:
    type: agentic
    config: {}
edges: []
`,
		`
name: t
version: "0.1"
start: x
nodes:
  x:
    type: agentic
    config:
      prompt: ""
edges: []
`,
	}
	for i, yamlData := range cases {
		bp, err := ParseBlueprintYAML([]byte(yamlData))
		if err != nil {
			t.Fatalf("case %d parse: %v", i, err)
		}
		_, err = bp.BuildGraph(&mockExecutor{})
		if err == nil {
			t.Fatalf("case %d: expected error for agentic node without prompt", i)
		}
	}
}

func TestBuildGraphDeterministicMissingCommand(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: x
nodes:
  x:
    type: deterministic
    config: {}
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error for deterministic node without command")
	}
}

func TestBuildGraphGateMissingCheckNode(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: g
nodes:
  g:
    type: gate
    config: {}
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error for gate node without check_node")
	}
}

func TestBuildGraphStartNodeNotInBlueprint(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: missing
nodes:
  a:
    type: deterministic
    config:
      command: "echo"
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error when start is not a defined node")
	}
}

func TestBuildGraphEdgeTargetMissing(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: a
nodes:
  a:
    type: deterministic
    config:
      command: "echo"
edges:
  - from: a
    to: ghost
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error when edge references unknown target node")
	}
}

func TestBuildGraphEdgeSourceMissing(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: a
nodes:
  a:
    type: deterministic
    config:
      command: "echo"
edges:
  - from: ghost
    to: a
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error when edge references unknown source node")
	}
}

func TestBlueprintYAMLAddNodesDuplicateID(t *testing.T) {
	bp := &BlueprintYAML{
		Nodes: map[string]NodeYAML{
			"x": {Type: "deterministic", Config: map[string]interface{}{"command": "echo"}},
		},
	}
	g := NewGraph()
	if err := g.AddNode(&stubNode{id: "x"}); err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if err := bp.addNodesToGraph(g, &mockExecutor{}); err == nil {
		t.Fatal("expected error when graph already contains node id from blueprint")
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

func TestBuiltinBlueprintsValid(t *testing.T) {
	files := []string{
		"../../blueprints/standard-implementation.yaml",
		"../../blueprints/bug-fix.yaml",
	}
	executor := &mockExecutor{output: "ok"}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		bp, err := ParseBlueprintYAML(data)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		g, err := bp.BuildGraph(executor)
		if err != nil {
			t.Fatalf("build graph %s: %v", f, err)
		}
		if err := g.Validate(); err != nil {
			t.Fatalf("validate %s: %v", f, err)
		}
	}
}
