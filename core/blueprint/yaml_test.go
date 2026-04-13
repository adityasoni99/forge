package blueprint

import (
	"os"
	"reflect"
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

const enrichedYAML = `
name: enriched
version: "1.0"
description: "desc"
when_to_use: "When refactoring services"
start: a
hooks:
  - event: pre_node_exec
    action: log
    config:
      level: debug
nodes:
  a:
    type: agentic
    description: "Planning step"
    concurrency_safe: true
    allowed_tools:
      - read
      - grep
    max_retries: 3
    config:
      prompt: "Plan the work"
  b:
    type: deterministic
    description: "Run checks"
    max_retries: 1
    config:
      command: "echo ok"
edges:
  - from: a
    to: b
`

func TestParseEnrichedBlueprintYAML(t *testing.T) {
	bp, err := ParseBlueprintYAML([]byte(enrichedYAML))
	if err != nil {
		t.Fatalf("ParseBlueprintYAML: %v", err)
	}
	if bp.WhenToUse != "When refactoring services" {
		t.Errorf("WhenToUse = %q", bp.WhenToUse)
	}
	if len(bp.Hooks) != 1 {
		t.Fatalf("Hooks len = %d, want 1", len(bp.Hooks))
	}
	h := bp.Hooks[0]
	if h.Event != "pre_node_exec" || h.Action != "log" {
		t.Errorf("Hook = %+v", h)
	}
	if h.Config["level"] != "debug" {
		t.Errorf("Hook.Config = %#v", h.Config)
	}
	na := bp.Nodes["a"]
	if na.Description != "Planning step" {
		t.Errorf("node a Description = %q", na.Description)
	}
	if na.ConcurrencySafe == nil || !*na.ConcurrencySafe {
		t.Errorf("node a ConcurrencySafe = %v", na.ConcurrencySafe)
	}
	wantTools := []string{"read", "grep"}
	if !reflect.DeepEqual(na.AllowedTools, wantTools) {
		t.Errorf("AllowedTools = %v, want %v", na.AllowedTools, wantTools)
	}
	if na.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d", na.MaxRetries)
	}
	nb := bp.Nodes["b"]
	if nb.Description != "Run checks" {
		t.Errorf("node b Description = %q", nb.Description)
	}
	if nb.MaxRetries != 1 {
		t.Errorf("node b MaxRetries = %d", nb.MaxRetries)
	}
}

func TestBuildGraphConcurrencySafeFromYAML(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: x
nodes:
  x:
    type: agentic
    concurrency_safe: true
    config:
      prompt: "hi"
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	raw, _ := g.GetNode("x")
	an, ok := raw.(*AgenticNode)
	if !ok {
		t.Fatalf("node type %T", raw)
	}
	if !an.IsConcurrencySafe() {
		t.Error("IsConcurrencySafe() = false, want true")
	}
}

func TestBuildGraphAllowedToolsOnDeterministicErrors(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: x
nodes:
  x:
    type: deterministic
    allowed_tools:
      - shell
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
		t.Fatal("expected error for allowed_tools on deterministic node")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	bp, err := ParseBlueprintYAML([]byte(testYAML))
	if err != nil {
		t.Fatalf("ParseBlueprintYAML: %v", err)
	}
	if bp.WhenToUse != "" {
		t.Errorf("WhenToUse = %q, want empty", bp.WhenToUse)
	}
	if len(bp.Hooks) != 0 {
		t.Errorf("Hooks = %v, want empty", bp.Hooks)
	}
	plan := bp.Nodes["plan"]
	if plan.Description != "" || plan.ConcurrencySafe != nil || len(plan.AllowedTools) != 0 || plan.MaxRetries != 0 {
		t.Errorf("plan node has unexpected enriched fields: %+v", plan)
	}
	g, err := bp.BuildGraph(&mockExecutor{output: "done"})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount = %d", g.NodeCount())
	}
}

func TestBuildGraphAllowedToolsOnAgentic(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: x
nodes:
  x:
    type: agentic
    allowed_tools:
      - read
      - write
    config:
      prompt: "go"
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	raw, _ := g.GetNode("x")
	an := raw.(*AgenticNode)
	tools, ok := an.config["allowed_tools"].([]string)
	if !ok {
		t.Fatalf("allowed_tools type %T", an.config["allowed_tools"])
	}
	if !reflect.DeepEqual(tools, []string{"read", "write"}) {
		t.Errorf("allowed_tools = %v", tools)
	}
}

func TestBuildGraphEvalNode(t *testing.T) {
	yamlData := `
name: eval-test
version: "0.1"
start: evaluate
nodes:
  evaluate:
    type: eval
    config:
      prompt: "Rate the code quality"
      criteria:
        - correctness
        - readability
      threshold: 0.8
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{output: "0.9"})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	raw, ok := g.GetNode("evaluate")
	if !ok {
		t.Fatal("node 'evaluate' not found")
	}
	if raw.Type() != NodeTypeEval {
		t.Errorf("Type() = %v, want Eval", raw.Type())
	}
}

func TestBuildGraphEvalNodeMissingPrompt(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: e
nodes:
  e:
    type: eval
    config:
      threshold: 0.7
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error for eval node without prompt")
	}
}

func TestBuildGraphEvalNodeDefaultThreshold(t *testing.T) {
	yamlData := `
name: t
version: "0.1"
start: e
nodes:
  e:
    type: eval
    config:
      prompt: "Check it"
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{output: "0.75"})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	_, ok := g.GetNode("e")
	if !ok {
		t.Fatal("node 'e' not found")
	}
}
