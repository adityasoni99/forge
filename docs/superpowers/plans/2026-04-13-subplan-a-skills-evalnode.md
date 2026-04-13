# Sub-plan A: Skills + EvalNode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add EvalNode to the blueprint engine and build a filesystem-based skill system in the harness with registry, resolver, lifecycle, and AgentService integration.

**Architecture:** EvalNode is a new Go node type that sends an evaluation prompt through the existing AgentExecutor interface and parses a numeric score. The skill system is TypeScript-only in the harness: SKILL.md files with YAML frontmatter are loaded by a registry, matched by a resolver, and optionally prepended to agent prompts by AgentService.

**Tech Stack:** Go 1.22+, TypeScript 5.7+, vitest, yaml (npm package)

---

## File Structure

**Create:**
- `core/blueprint/eval_node.go` — EvalNode struct, Execute, score parsing
- `core/blueprint/eval_node_test.go` — EvalNode unit tests
- `harness/src/skills/types.ts` — Skill, SkillFrontmatter types, parseFrontmatter()
- `harness/src/skills/types.test.ts` — frontmatter parsing tests
- `harness/src/skills/registry.ts` — SkillRegistry class (filesystem scan)
- `harness/src/skills/registry.test.ts` — registry tests
- `harness/src/skills/resolver.ts` — SkillResolver class (keyword matching)
- `harness/src/skills/resolver.test.ts` — resolver tests
- `harness/src/skills/lifecycle.ts` — SkillLifecycle (evaluate, promote, compare)
- `harness/src/skills/lifecycle.test.ts` — lifecycle tests
- `skills/coding/implement-feature/SKILL.md` — built-in implementation skill
- `skills/quality/code-review/SKILL.md` — built-in code review skill

**Modify:**
- `core/blueprint/types.go` — add NodeTypeEval constant, update String()
- `core/blueprint/yaml.go` — add "eval" case to buildNode()
- `core/blueprint/types_test.go` — add NodeTypeEval to String() test
- `core/blueprint/yaml_test.go` — add eval YAML parsing test
- `harness/src/agent-service.ts` — add optional skill resolution
- `harness/src/agent-service.test.ts` — add skill integration tests
- `harness/package.json` — add `yaml` dependency

---

### Task 1: Add NodeTypeEval to the engine type system

**Files:**
- Modify: `core/blueprint/types.go`
- Modify: `core/blueprint/types_test.go`

- [ ] **Step 1: Write the failing test**

Add to `core/blueprint/types_test.go` — extend the existing `TestNodeTypeString` test table:

```go
func TestNodeTypeString(t *testing.T) {
	tests := []struct {
		nt   NodeType
		want string
	}{
		{NodeTypeAgentic, "agentic"},
		{NodeTypeDeterministic, "deterministic"},
		{NodeTypeGate, "gate"},
		{NodeTypeEval, "eval"},
	}
	for _, tt := range tests {
		if got := tt.nt.String(); got != tt.want {
			t.Errorf("NodeType(%d).String() = %q, want %q", tt.nt, got, tt.want)
		}
	}
	if got := NodeType(99).String(); got != "unknown" {
		t.Errorf("NodeType(99).String() = %q, want %q", got, "unknown")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./core/blueprint/ -run TestNodeTypeString -v`
Expected: compilation error — `NodeTypeEval` undefined

- [ ] **Step 3: Add NodeTypeEval to types.go**

In `core/blueprint/types.go`, update the NodeType constants:

```go
const (
	NodeTypeAgentic NodeType = iota
	NodeTypeDeterministic
	NodeTypeGate
	NodeTypeEval
)
```

And update the String() method:

```go
func (nt NodeType) String() string {
	switch nt {
	case NodeTypeAgentic:
		return "agentic"
	case NodeTypeDeterministic:
		return "deterministic"
	case NodeTypeGate:
		return "gate"
	case NodeTypeEval:
		return "eval"
	default:
		return "unknown"
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./core/blueprint/ -run TestNodeTypeString -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add core/blueprint/types.go core/blueprint/types_test.go
git commit -m "feat(engine): add NodeTypeEval constant"
```

---

### Task 2: Implement EvalNode struct and execution

**Files:**
- Create: `core/blueprint/eval_node.go`
- Create: `core/blueprint/eval_node_test.go`

- [ ] **Step 1: Write the failing tests**

Create `core/blueprint/eval_node_test.go`:

```go
package blueprint

import (
	"context"
	"errors"
	"testing"
)

func TestEvalNodePassesAboveThreshold(t *testing.T) {
	executor := &mockExecutor{output: "0.85"}
	node := NewEvalNode("quality-check", "Evaluate code quality", []string{"correctness", "style"}, 0.8, executor)

	if node.ID() != "quality-check" {
		t.Errorf("ID() = %q, want %q", node.ID(), "quality-check")
	}
	if node.Type() != NodeTypeEval {
		t.Errorf("Type() = %v, want %v", node.Type(), NodeTypeEval)
	}
	if node.IsConcurrencySafe() {
		t.Error("IsConcurrencySafe() = true, want false")
	}

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want %v", result.Status, NodeStatusPassed)
	}
	if result.Output == "" {
		t.Error("Output should contain score info")
	}
}

func TestEvalNodeFailsBelowThreshold(t *testing.T) {
	executor := &mockExecutor{output: "0.65"}
	node := NewEvalNode("quality-check", "Evaluate code quality", []string{"correctness"}, 0.8, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, NodeStatusFailed)
	}
}

func TestEvalNodeExactThreshold(t *testing.T) {
	executor := &mockExecutor{output: "0.80"}
	node := NewEvalNode("check", "Eval", nil, 0.8, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed (score == threshold)", result.Status)
	}
}

func TestEvalNodeUnparseableScore(t *testing.T) {
	executor := &mockExecutor{output: "The code looks good overall"}
	node := NewEvalNode("check", "Eval", nil, 0.5, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed for unparseable score", result.Status)
	}
	if result.Error == "" {
		t.Error("Error should explain parse failure")
	}
}

func TestEvalNodeExecutorError(t *testing.T) {
	executor := &mockExecutor{err: errors.New("timeout")}
	node := NewEvalNode("check", "Eval", nil, 0.5, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if result.Error == "" {
		t.Error("Error should contain executor error message")
	}
}

func TestEvalNodeScoreInLongerText(t *testing.T) {
	executor := &mockExecutor{output: "Based on analysis, score: 0.92 out of 1.0"}
	node := NewEvalNode("check", "Eval", []string{"quality"}, 0.9, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed (0.92 >= 0.9)", result.Status)
	}
}

func TestEvalNodeBuildEvalPrompt(t *testing.T) {
	node := NewEvalNode("check", "Rate the code", []string{"correctness", "style"}, 0.8, nil)
	prompt := node.BuildEvalPrompt(NewRunState("bp", "r1"))
	if prompt == "" {
		t.Fatal("prompt should not be empty")
	}
	if !containsSubstring(prompt, "Rate the code") {
		t.Error("prompt should contain base prompt")
	}
	if !containsSubstring(prompt, "correctness") {
		t.Error("prompt should contain criteria")
	}
	if !containsSubstring(prompt, "0.0 and 1.0") {
		t.Error("prompt should instruct score format")
	}
}

func TestParseEvalScore(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"0.85", 0.85, false},
		{"  0.90  ", 0.90, false},
		{"Score: 0.75", 0.75, false},
		{"The answer is 0.60 based on criteria", 0.60, false},
		{"no numbers here", 0, true},
		{"1.5", 0, true},
		{"-0.3", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseEvalScore(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseEvalScore(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseEvalScore(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

Note: `mockExecutor` is already defined in `node_test.go` in the same package.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./core/blueprint/ -run "TestEvalNode|TestParseEvalScore" -v`
Expected: compilation error — `NewEvalNode`, `ParseEvalScore` undefined

- [ ] **Step 3: Implement EvalNode**

Create `core/blueprint/eval_node.go`:

```go
package blueprint

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// EvalNode sends an evaluation prompt through AgentExecutor and parses a
// numeric score. If score >= threshold the node passes; otherwise it fails.
type EvalNode struct {
	id        string
	prompt    string
	criteria  []string
	threshold float64
	executor  AgentExecutor
}

func NewEvalNode(id, prompt string, criteria []string, threshold float64, executor AgentExecutor) *EvalNode {
	return &EvalNode{
		id:        id,
		prompt:    prompt,
		criteria:  criteria,
		threshold: threshold,
		executor:  executor,
	}
}

func (n *EvalNode) ID() string              { return n.id }
func (n *EvalNode) Type() NodeType           { return NodeTypeEval }
func (n *EvalNode) IsConcurrencySafe() bool  { return false }

func (n *EvalNode) Execute(ctx context.Context, state *RunState) (NodeResult, error) {
	evalPrompt := n.BuildEvalPrompt(state)
	output, err := n.executor.Execute(ctx, evalPrompt, nil)
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  err.Error(),
		}, nil
	}

	score, err := ParseEvalScore(output)
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Output: output,
			Error:  fmt.Sprintf("failed to parse eval score: %s", err),
		}, nil
	}

	summary := fmt.Sprintf("score=%.2f threshold=%.2f", score, n.threshold)
	if score >= n.threshold {
		return NodeResult{Status: NodeStatusPassed, Output: summary}, nil
	}
	return NodeResult{Status: NodeStatusFailed, Output: summary}, nil
}

// BuildEvalPrompt constructs the full evaluation prompt including criteria and
// scoring instructions.
func (n *EvalNode) BuildEvalPrompt(state *RunState) string {
	var b strings.Builder
	b.WriteString(n.prompt)
	if len(n.criteria) > 0 {
		b.WriteString("\n\nEvaluation criteria:\n")
		for i, c := range n.criteria {
			fmt.Fprintf(&b, "%d. %s\n", i+1, c)
		}
	}
	b.WriteString("\nRespond with ONLY a numeric score between 0.0 and 1.0.")
	return b.String()
}

// ParseEvalScore extracts a float64 score in [0.0, 1.0] from executor output.
func ParseEvalScore(output string) (float64, error) {
	for _, word := range strings.Fields(strings.TrimSpace(output)) {
		clean := strings.TrimRight(word, ".,;:!?)")
		score, err := strconv.ParseFloat(clean, 64)
		if err == nil && score >= 0 && score <= 1 {
			return score, nil
		}
	}
	return 0, fmt.Errorf("no valid score (0.0-1.0) found in output: %q", strings.TrimSpace(output))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./core/blueprint/ -run "TestEvalNode|TestParseEvalScore" -v`
Expected: all PASS

- [ ] **Step 5: Run all blueprint tests to check nothing broke**

Run: `go test ./core/blueprint/ -v`
Expected: all existing tests PASS

- [ ] **Step 6: Commit**

```bash
git add core/blueprint/eval_node.go core/blueprint/eval_node_test.go
git commit -m "feat(engine): implement EvalNode with score parsing"
```

---

### Task 3: Add eval node YAML parsing

**Files:**
- Modify: `core/blueprint/yaml.go`
- Modify: `core/blueprint/yaml_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `core/blueprint/yaml_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./core/blueprint/ -run "TestBuildGraphEvalNode" -v`
Expected: FAIL — "unknown node type: \"eval\""

- [ ] **Step 3: Add eval case to buildNode in yaml.go**

In `core/blueprint/yaml.go`, add this case to the `buildNode` switch, before the `default:` case:

```go
	case "eval":
		prompt, _ := ny.Config["prompt"].(string)
		if prompt == "" {
			return nil, fmt.Errorf("eval node missing 'prompt' in config")
		}
		criteriaRaw, _ := ny.Config["criteria"].([]interface{})
		criteria := make([]string, 0, len(criteriaRaw))
		for _, c := range criteriaRaw {
			if s, ok := c.(string); ok {
				criteria = append(criteria, s)
			}
		}
		threshold := 0.7
		if t, ok := ny.Config["threshold"].(float64); ok {
			threshold = t
		}
		return NewEvalNode(id, prompt, criteria, threshold, executor), nil
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./core/blueprint/ -run "TestBuildGraphEvalNode" -v`
Expected: all PASS

- [ ] **Step 5: Run all blueprint tests**

Run: `go test ./core/blueprint/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add core/blueprint/yaml.go core/blueprint/yaml_test.go
git commit -m "feat(engine): add eval node YAML parsing"
```

---

### Task 4: Skill types and frontmatter parser

**Files:**
- Modify: `harness/package.json` — add `yaml` dependency
- Create: `harness/src/skills/types.ts`
- Create: `harness/src/skills/types.test.ts`

- [ ] **Step 1: Add yaml dependency**

Run: `cd harness && npm install yaml`

- [ ] **Step 2: Create skill types**

Create `harness/src/skills/types.ts`:

```typescript
import { parse as parseYaml } from 'yaml';

export interface SkillFrontmatter {
  name: string;
  version: string;
  description: string;
  when_to_use: string;
  eval_score: number;
  tags: string[];
}

export interface Skill {
  name: string;
  version: string;
  description: string;
  whenToUse: string;
  evalScore: number;
  tags: string[];
  bodyPath: string;
  body: string;
}

const FRONTMATTER_REGEX = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/;

export function parseFrontmatter(raw: string): { frontmatter: SkillFrontmatter; body: string } {
  const match = raw.match(FRONTMATTER_REGEX);
  if (!match) {
    throw new Error('SKILL.md missing YAML frontmatter (expected --- delimiters)');
  }

  const parsed = parseYaml(match[1]) as Record<string, unknown>;
  const frontmatter: SkillFrontmatter = {
    name: String(parsed.name ?? ''),
    version: String(parsed.version ?? '1.0'),
    description: String(parsed.description ?? ''),
    when_to_use: String(parsed.when_to_use ?? ''),
    eval_score: Number(parsed.eval_score ?? 0),
    tags: Array.isArray(parsed.tags) ? parsed.tags.map(String) : [],
  };

  if (!frontmatter.name) {
    throw new Error('SKILL.md frontmatter missing required field: name');
  }

  return { frontmatter, body: match[2].trim() };
}

export function frontmatterToSkill(
  fm: SkillFrontmatter,
  body: string,
  bodyPath: string,
): Skill {
  return {
    name: fm.name,
    version: fm.version,
    description: fm.description,
    whenToUse: fm.when_to_use,
    evalScore: fm.eval_score,
    tags: fm.tags,
    bodyPath,
    body,
  };
}
```

- [ ] **Step 3: Write tests**

Create `harness/src/skills/types.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { parseFrontmatter, frontmatterToSkill } from './types.js';

const VALID_SKILL = `---
name: implement-feature
version: "1.0"
description: Basic implementation skill
when_to_use: When implementing a new feature
eval_score: 0.85
tags:
  - coding
  - implementation
---
# Implement Feature

You are a skilled software developer. Implement the requested feature.
`;

describe('parseFrontmatter', () => {
  it('parses valid SKILL.md content', () => {
    const { frontmatter, body } = parseFrontmatter(VALID_SKILL);
    expect(frontmatter.name).toBe('implement-feature');
    expect(frontmatter.version).toBe('1.0');
    expect(frontmatter.description).toBe('Basic implementation skill');
    expect(frontmatter.when_to_use).toBe('When implementing a new feature');
    expect(frontmatter.eval_score).toBe(0.85);
    expect(frontmatter.tags).toEqual(['coding', 'implementation']);
    expect(body).toContain('You are a skilled software developer');
  });

  it('throws on missing frontmatter delimiters', () => {
    expect(() => parseFrontmatter('# No frontmatter')).toThrow('missing YAML frontmatter');
  });

  it('throws on missing name', () => {
    const content = `---
version: "1.0"
description: No name
---
Body text`;
    expect(() => parseFrontmatter(content)).toThrow('missing required field: name');
  });

  it('provides defaults for optional fields', () => {
    const content = `---
name: minimal
---
Body`;
    const { frontmatter } = parseFrontmatter(content);
    expect(frontmatter.version).toBe('1.0');
    expect(frontmatter.eval_score).toBe(0);
    expect(frontmatter.tags).toEqual([]);
  });
});

describe('frontmatterToSkill', () => {
  it('converts frontmatter + body to Skill object', () => {
    const { frontmatter, body } = parseFrontmatter(VALID_SKILL);
    const skill = frontmatterToSkill(frontmatter, body, '/skills/coding/implement-feature/SKILL.md');
    expect(skill.name).toBe('implement-feature');
    expect(skill.whenToUse).toBe('When implementing a new feature');
    expect(skill.bodyPath).toBe('/skills/coding/implement-feature/SKILL.md');
    expect(skill.body).toContain('skilled software developer');
  });
});
```

- [ ] **Step 4: Run tests**

Run: `cd harness && npx vitest run src/skills/types.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/package.json harness/package-lock.json harness/src/skills/types.ts harness/src/skills/types.test.ts
git commit -m "feat(harness): add skill types and frontmatter parser"
```

---

### Task 5: Skill registry

**Files:**
- Create: `harness/src/skills/registry.ts`
- Create: `harness/src/skills/registry.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/skills/registry.test.ts`:

```typescript
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as os from 'node:os';
import { SkillRegistry } from './registry.js';

function createSkillDir(baseDir: string, skillPath: string, content: string) {
  const dir = path.join(baseDir, path.dirname(skillPath));
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(baseDir, skillPath), content);
}

const SKILL_A = `---
name: skill-a
version: "1.0"
description: First skill
when_to_use: When doing A
eval_score: 0.9
tags:
  - coding
---
# Skill A
Do the A thing.
`;

const SKILL_B = `---
name: skill-b
version: "2.0"
description: Second skill
when_to_use: When doing B
tags:
  - review
  - quality
---
# Skill B
Do the B thing.
`;

describe('SkillRegistry', () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-registry-'));
    createSkillDir(tmpDir, 'coding/skill-a/SKILL.md', SKILL_A);
    createSkillDir(tmpDir, 'quality/skill-b/SKILL.md', SKILL_B);
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it('loadAll discovers all skills', async () => {
    const registry = new SkillRegistry();
    const skills = await registry.loadAll(tmpDir);
    expect(skills).toHaveLength(2);
    const names = skills.map((s) => s.name).sort();
    expect(names).toEqual(['skill-a', 'skill-b']);
  });

  it('findByName returns matching skill', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    const skill = registry.findByName('skill-a');
    expect(skill).toBeDefined();
    expect(skill!.name).toBe('skill-a');
    expect(skill!.version).toBe('1.0');
  });

  it('findByName returns undefined for unknown skill', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    expect(registry.findByName('nonexistent')).toBeUndefined();
  });

  it('findByTag returns skills matching tag', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    const review = registry.findByTag('review');
    expect(review).toHaveLength(1);
    expect(review[0].name).toBe('skill-b');
  });

  it('findByTag returns empty for unknown tag', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    expect(registry.findByTag('unknown')).toEqual([]);
  });

  it('handles empty directory', async () => {
    const emptyDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-empty-'));
    try {
      const registry = new SkillRegistry();
      const skills = await registry.loadAll(emptyDir);
      expect(skills).toEqual([]);
    } finally {
      fs.rmSync(emptyDir, { recursive: true, force: true });
    }
  });

  it('skips directories without SKILL.md', async () => {
    fs.mkdirSync(path.join(tmpDir, 'empty-dir/no-skill'), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, 'empty-dir/no-skill/README.md'), '# Not a skill');
    const registry = new SkillRegistry();
    const skills = await registry.loadAll(tmpDir);
    expect(skills).toHaveLength(2);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/skills/registry.test.ts`
Expected: FAIL — cannot import `SkillRegistry`

- [ ] **Step 3: Implement SkillRegistry**

Create `harness/src/skills/registry.ts`:

```typescript
import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import type { Skill } from './types.js';
import { parseFrontmatter, frontmatterToSkill } from './types.js';

export class SkillRegistry {
  private skills: Skill[] = [];

  async loadAll(skillsDir: string): Promise<Skill[]> {
    this.skills = [];
    const skillPaths = await findSkillFiles(skillsDir);
    for (const skillPath of skillPaths) {
      try {
        const raw = await fs.readFile(skillPath, 'utf-8');
        const { frontmatter, body } = parseFrontmatter(raw);
        this.skills.push(frontmatterToSkill(frontmatter, body, skillPath));
      } catch {
        // Skip malformed skill files
      }
    }
    return [...this.skills];
  }

  findByName(name: string): Skill | undefined {
    return this.skills.find((s) => s.name === name);
  }

  findByTag(tag: string): Skill[] {
    return this.skills.filter((s) => s.tags.includes(tag));
  }

  all(): Skill[] {
    return [...this.skills];
  }
}

async function findSkillFiles(dir: string): Promise<string[]> {
  const results: string[] = [];
  try {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const subEntries = await fs.readdir(path.join(dir, entry.name), { withFileTypes: true });
      for (const sub of subEntries) {
        if (sub.isDirectory()) {
          const skillFile = path.join(dir, entry.name, sub.name, 'SKILL.md');
          try {
            await fs.access(skillFile);
            results.push(skillFile);
          } catch {
            // No SKILL.md in this directory
          }
        }
      }
    }
  } catch {
    // Directory doesn't exist or can't be read
  }
  return results.sort();
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/skills/registry.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/skills/registry.ts harness/src/skills/registry.test.ts
git commit -m "feat(harness): add SkillRegistry with filesystem discovery"
```

---

### Task 6: Skill resolver

**Files:**
- Create: `harness/src/skills/resolver.ts`
- Create: `harness/src/skills/resolver.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/skills/resolver.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { SkillResolver } from './resolver.js';
import type { Skill } from './types.js';

function makeSkill(overrides: Partial<Skill>): Skill {
  return {
    name: 'default',
    version: '1.0',
    description: '',
    whenToUse: '',
    evalScore: 0,
    tags: [],
    bodyPath: '',
    body: '',
    ...overrides,
  };
}

const SKILLS: Skill[] = [
  makeSkill({
    name: 'implement-feature',
    description: 'Implement a new feature from scratch',
    whenToUse: 'When implementing new features or adding functionality',
    tags: ['coding', 'implementation'],
    evalScore: 0.9,
  }),
  makeSkill({
    name: 'code-review',
    description: 'Review code for quality and correctness',
    whenToUse: 'When reviewing pull requests or code changes',
    tags: ['review', 'quality'],
    evalScore: 0.85,
  }),
  makeSkill({
    name: 'bug-fix',
    description: 'Debug and fix software bugs',
    whenToUse: 'When fixing bugs or resolving errors',
    tags: ['debugging', 'fix'],
    evalScore: 0.8,
  }),
];

describe('SkillResolver', () => {
  it('resolves by exact name', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('anything', { skill: 'code-review' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('code-review');
  });

  it('returns null for unknown exact name', () => {
    const resolver = new SkillResolver(SKILLS);
    expect(resolver.resolve('anything', { skill: 'nonexistent' })).toBeNull();
  });

  it('auto-resolves based on task description keywords', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('Review the authentication module code', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('code-review');
  });

  it('auto-resolves implementation task', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('Implement user registration feature', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('implement-feature');
  });

  it('auto-resolves bug fix task', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('Fix the login error when password is empty', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('bug-fix');
  });

  it('returns null when no skill matches auto', () => {
    const resolver = new SkillResolver(SKILLS);
    expect(resolver.resolve('Do something completely unrelated', { skill: 'auto' })).toBeNull();
  });

  it('returns null when skill key is absent from config', () => {
    const resolver = new SkillResolver(SKILLS);
    expect(resolver.resolve('Implement something', {})).toBeNull();
  });

  it('prefers higher evalScore on tie', () => {
    const skills = [
      makeSkill({ name: 'a', description: 'implement things', whenToUse: 'implement', evalScore: 0.7, tags: ['coding'] }),
      makeSkill({ name: 'b', description: 'implement things', whenToUse: 'implement', evalScore: 0.9, tags: ['coding'] }),
    ];
    const resolver = new SkillResolver(skills);
    const result = resolver.resolve('implement a widget', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('b');
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/skills/resolver.test.ts`
Expected: FAIL — cannot import `SkillResolver`

- [ ] **Step 3: Implement SkillResolver**

Create `harness/src/skills/resolver.ts`:

```typescript
import type { Skill } from './types.js';

interface ResolveConfig {
  skill?: string;
  [key: string]: unknown;
}

export class SkillResolver {
  constructor(private readonly skills: Skill[]) {}

  resolve(taskDescription: string, config: ResolveConfig): Skill | null {
    const skillRef = config.skill;
    if (!skillRef) return null;

    if (skillRef !== 'auto') {
      return this.skills.find((s) => s.name === skillRef) ?? null;
    }

    return this.autoResolve(taskDescription);
  }

  private autoResolve(taskDescription: string): Skill | null {
    const words = taskDescription.toLowerCase().split(/\s+/);
    let bestSkill: Skill | null = null;
    let bestScore = 0;

    for (const skill of this.skills) {
      const score = this.scoreMatch(words, skill);
      if (score > bestScore || (score === bestScore && skill.evalScore > (bestSkill?.evalScore ?? 0))) {
        bestScore = score;
        bestSkill = skill;
      }
    }

    return bestScore > 0 ? bestSkill : null;
  }

  private scoreMatch(taskWords: string[], skill: Skill): number {
    const targets = [
      skill.description.toLowerCase(),
      skill.whenToUse.toLowerCase(),
      ...skill.tags.map((t) => t.toLowerCase()),
      skill.name.replace(/-/g, ' ').toLowerCase(),
    ].join(' ');

    const targetWords = new Set(targets.split(/\s+/));
    let matches = 0;
    for (const word of taskWords) {
      if (word.length < 3) continue;
      if (targetWords.has(word)) {
        matches++;
      }
      for (const tw of targetWords) {
        if (tw.length >= 4 && (tw.includes(word) || word.includes(tw))) {
          matches++;
          break;
        }
      }
    }
    return matches;
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/skills/resolver.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/skills/resolver.ts harness/src/skills/resolver.test.ts
git commit -m "feat(harness): add SkillResolver with keyword matching"
```

---

### Task 7: Skill lifecycle

**Files:**
- Create: `harness/src/skills/lifecycle.ts`
- Create: `harness/src/skills/lifecycle.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/skills/lifecycle.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { SkillLifecycle } from './lifecycle.js';
import type { Skill } from './types.js';

function makeSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    name: 'test-skill',
    version: '1.0',
    description: 'A test skill',
    whenToUse: 'When testing',
    evalScore: 0,
    tags: ['test'],
    bodyPath: '/skills/test/SKILL.md',
    body: 'You are a test skill.',
    ...overrides,
  };
}

describe('SkillLifecycle', () => {
  describe('evaluate', () => {
    it('returns passing result when all cases pass', async () => {
      const skill = makeSkill();
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.evaluate(skill, [
        { input: 'Write a hello world', expectedContains: 'hello' },
        { input: 'Add logging', expectedContains: 'log' },
      ]);
      expect(result.passed).toBe(true);
      expect(result.passRate).toBe(1.0);
      expect(result.results).toHaveLength(2);
    });

    it('returns failing result when case fails', async () => {
      const skill = makeSkill();
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.evaluate(skill, [
        { input: 'Write tests', expectedContains: 'IMPOSSIBLE_STRING_NEVER_FOUND' },
      ]);
      expect(result.passed).toBe(false);
      expect(result.passRate).toBe(0);
    });

    it('handles empty test cases', async () => {
      const skill = makeSkill();
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.evaluate(skill, []);
      expect(result.passed).toBe(true);
      expect(result.passRate).toBe(1.0);
    });
  });

  describe('promote', () => {
    it('bumps version and eval score', () => {
      const skill = makeSkill({ version: '1.0', evalScore: 0.7 });
      const lifecycle = new SkillLifecycle();
      const promoted = lifecycle.promote(skill, '2.0', 0.9);
      expect(promoted.version).toBe('2.0');
      expect(promoted.evalScore).toBe(0.9);
      expect(promoted.name).toBe(skill.name);
    });
  });

  describe('compare', () => {
    it('returns comparison showing better skill', async () => {
      const skillA = makeSkill({ name: 'skill-a', body: 'You implement features with tests.' });
      const skillB = makeSkill({ name: 'skill-b', body: 'You implement features.' });
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.compare(skillA, skillB, [
        { input: 'Add a user model', expectedContains: 'user' },
      ]);
      expect(result.winnerName).toBeDefined();
      expect(result.scoreA).toBeGreaterThanOrEqual(0);
      expect(result.scoreB).toBeGreaterThanOrEqual(0);
    });
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/skills/lifecycle.test.ts`
Expected: FAIL — cannot import `SkillLifecycle`

- [ ] **Step 3: Implement SkillLifecycle**

Create `harness/src/skills/lifecycle.ts`:

```typescript
import type { Skill } from './types.js';

export interface TestCase {
  input: string;
  expectedContains: string;
}

export interface TestCaseResult {
  input: string;
  passed: boolean;
  composedPrompt: string;
}

export interface EvalResult {
  passed: boolean;
  passRate: number;
  results: TestCaseResult[];
}

export interface ComparisonResult {
  winnerName: string;
  scoreA: number;
  scoreB: number;
  details: string;
}

export class SkillLifecycle {
  async evaluate(skill: Skill, testCases: TestCase[]): Promise<EvalResult> {
    if (testCases.length === 0) {
      return { passed: true, passRate: 1.0, results: [] };
    }

    const results: TestCaseResult[] = [];
    let passCount = 0;

    for (const tc of testCases) {
      const composed = `${skill.body}\n\n=== Task ===\n${tc.input}`;
      const passed = composed.toLowerCase().includes(tc.expectedContains.toLowerCase());
      if (passed) passCount++;
      results.push({ input: tc.input, passed, composedPrompt: composed });
    }

    const passRate = passCount / testCases.length;
    return { passed: passRate === 1.0, passRate, results };
  }

  promote(skill: Skill, newVersion: string, newEvalScore: number): Skill {
    return {
      ...skill,
      version: newVersion,
      evalScore: newEvalScore,
    };
  }

  async compare(skillA: Skill, skillB: Skill, testCases: TestCase[]): Promise<ComparisonResult> {
    const resultA = await this.evaluate(skillA, testCases);
    const resultB = await this.evaluate(skillB, testCases);

    const winnerName = resultA.passRate >= resultB.passRate ? skillA.name : skillB.name;
    return {
      winnerName,
      scoreA: resultA.passRate,
      scoreB: resultB.passRate,
      details: `${skillA.name}: ${resultA.passRate.toFixed(2)} vs ${skillB.name}: ${resultB.passRate.toFixed(2)}`,
    };
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/skills/lifecycle.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/skills/lifecycle.ts harness/src/skills/lifecycle.test.ts
git commit -m "feat(harness): add SkillLifecycle with evaluate, promote, compare"
```

---

### Task 8: Integrate skills into AgentService

**Files:**
- Modify: `harness/src/agent-service.ts`
- Modify: `harness/src/agent-service.test.ts`

- [ ] **Step 1: Write the failing tests**

Add to `harness/src/agent-service.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { AgentService } from './agent-service.js';
import { EchoAdapter } from './adapters/echo.js';
import { SkillRegistry } from './skills/registry.js';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as os from 'node:os';

describe('AgentService', () => {
  it('enriches prompt with context and calls adapter', async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-svc-'));
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), '# Rules\nAlways use TDD.');

    try {
      const service = new AgentService(new EchoAdapter());
      const response = await service.executeAgent({
        prompt: 'Fix the auth module',
        config_json: '{}',
        working_directory: tmpDir,
        blueprint_name: 'bug-fix',
        node_id: 'implement',
        run_id: 'run-1',
      });

      expect(response.success).toBe(true);
      expect(response.output).toContain('Always use TDD');
      expect(response.output).toContain('Fix the auth module');
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it('handles adapter failure gracefully', async () => {
    const failAdapter = {
      async execute() {
        return { output: '', success: false, error: 'agent crashed' };
      },
    };
    const service = new AgentService(failAdapter);
    const response = await service.executeAgent({
      prompt: 'Do something',
      config_json: '{}',
      working_directory: '/tmp',
      blueprint_name: 'test',
      node_id: 'a',
      run_id: 'r1',
    });

    expect(response.success).toBe(false);
    expect(response.error).toContain('agent crashed');
  });

  it('returns structured error when adapter throws', async () => {
    const throwingAdapter = {
      async execute() {
        throw new Error('boom');
      },
    };
    const service = new AgentService(throwingAdapter);
    const response = await service.executeAgent({
      prompt: 'x',
      config_json: '{}',
      working_directory: '/tmp',
      blueprint_name: 'b',
      node_id: 'n',
      run_id: 'r',
    });
    expect(response.success).toBe(false);
    expect(response.error).toContain('boom');
  });

  it('stringifies non-Error throws in catch path', async () => {
    const throwingAdapter = {
      async execute() {
        throw 'string-throw';
      },
    };
    const service = new AgentService(throwingAdapter);
    const response = await service.executeAgent({
      prompt: 'x',
      config_json: '{}',
      working_directory: '/tmp',
      blueprint_name: 'b',
      node_id: 'n',
      run_id: 'r',
    });
    expect(response.success).toBe(false);
    expect(response.error).toContain('string-throw');
  });

  it('prepends skill body when skill=auto resolves', async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-skill-'));
    const skillsDir = path.join(tmpDir, 'skills', 'coding', 'implement-feature');
    fs.mkdirSync(skillsDir, { recursive: true });
    fs.writeFileSync(path.join(skillsDir, 'SKILL.md'), `---
name: implement-feature
version: "1.0"
description: Implement features
when_to_use: When implementing new features
tags:
  - coding
  - implementation
---
You are an expert feature implementer.
`);

    try {
      const registry = new SkillRegistry();
      await registry.loadAll(path.join(tmpDir, 'skills'));
      const service = new AgentService(new EchoAdapter(), { skillRegistry: registry });

      const response = await service.executeAgent({
        prompt: 'Implement user registration',
        config_json: JSON.stringify({ skill: 'auto' }),
        working_directory: tmpDir,
        blueprint_name: 'standard',
        node_id: 'impl',
        run_id: 'r1',
      });

      expect(response.success).toBe(true);
      expect(response.output).toContain('expert feature implementer');
      expect(response.output).toContain('Implement user registration');
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it('prepends skill body when skill=name resolves', async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-skill-'));
    const skillsDir = path.join(tmpDir, 'skills', 'quality', 'code-review');
    fs.mkdirSync(skillsDir, { recursive: true });
    fs.writeFileSync(path.join(skillsDir, 'SKILL.md'), `---
name: code-review
version: "1.0"
description: Review code
when_to_use: When reviewing
tags:
  - review
---
You are an expert code reviewer.
`);

    try {
      const registry = new SkillRegistry();
      await registry.loadAll(path.join(tmpDir, 'skills'));
      const service = new AgentService(new EchoAdapter(), { skillRegistry: registry });

      const response = await service.executeAgent({
        prompt: 'Check the auth module',
        config_json: JSON.stringify({ skill: 'code-review' }),
        working_directory: tmpDir,
        blueprint_name: 'review',
        node_id: 'review',
        run_id: 'r1',
      });

      expect(response.success).toBe(true);
      expect(response.output).toContain('expert code reviewer');
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it('works normally without skill registry', async () => {
    const service = new AgentService(new EchoAdapter());
    const response = await service.executeAgent({
      prompt: 'Do stuff',
      config_json: JSON.stringify({ skill: 'auto' }),
      working_directory: '/tmp',
      blueprint_name: 'b',
      node_id: 'n',
      run_id: 'r',
    });
    expect(response.success).toBe(true);
    expect(response.output).toContain('Do stuff');
  });
});
```

- [ ] **Step 2: Run tests to verify new tests fail**

Run: `cd harness && npx vitest run src/agent-service.test.ts`
Expected: FAIL — `AgentService` constructor does not accept options

- [ ] **Step 3: Update AgentService to support skill resolution**

Replace `harness/src/agent-service.ts` with:

```typescript
import type { AgentAdapter } from './adapters/types.js';
import type { ExecuteAgentRequest, ExecuteAgentResponse } from './types.js';
import { loadContext } from './context/loader.js';
import { SkillResolver } from './skills/resolver.js';
import type { SkillRegistry } from './skills/registry.js';

export interface AgentServiceOptions {
  skillRegistry?: SkillRegistry;
}

export class AgentService {
  private readonly skillResolver: SkillResolver | null;

  constructor(
    private readonly adapter: AgentAdapter,
    options?: AgentServiceOptions,
  ) {
    const registry = options?.skillRegistry;
    this.skillResolver = registry ? new SkillResolver(registry.all()) : null;
  }

  async executeAgent(req: ExecuteAgentRequest): Promise<ExecuteAgentResponse> {
    try {
      const ctx = await loadContext(req.working_directory);
      let prompt = req.prompt;

      const config = this.parseConfig(req.config_json);
      const skill = this.skillResolver?.resolve(req.prompt, config) ?? null;
      if (skill) {
        prompt = `${skill.body}\n\n${prompt}`;
      }

      const composedPrompt = ctx.composePrompt(prompt);

      const result = await this.adapter.execute({
        prompt: composedPrompt,
        workingDirectory: req.working_directory,
        configJson: req.config_json,
      });

      return {
        output: result.output,
        success: result.success,
        error: result.error ?? '',
      };
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return { output: '', success: false, error: message };
    }
  }

  private parseConfig(json: string): Record<string, unknown> {
    try {
      return JSON.parse(json) as Record<string, unknown>;
    } catch {
      return {};
    }
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/agent-service.test.ts`
Expected: all PASS

- [ ] **Step 5: Run all harness tests to check nothing broke**

Run: `cd harness && npx vitest run`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add harness/src/agent-service.ts harness/src/agent-service.test.ts
git commit -m "feat(harness): integrate skill resolution into AgentService"
```

---

### Task 9: Built-in skills and end-to-end YAML test

**Files:**
- Create: `skills/coding/implement-feature/SKILL.md`
- Create: `skills/quality/code-review/SKILL.md`
- Create: `tests/testdata/eval-skill-blueprint.yaml`
- Modify: `core/blueprint/yaml_test.go` — add eval blueprint smoke test

- [ ] **Step 1: Create implement-feature skill**

Create `skills/coding/implement-feature/SKILL.md`:

```markdown
---
name: implement-feature
version: "1.0"
description: Implement a new feature from a task description
when_to_use: When implementing new features, adding functionality, or building components
eval_score: 0.0
tags:
  - coding
  - implementation
  - feature
---
# Implement Feature

You are an expert software developer implementing a feature from scratch.

## Approach

1. Read the task description carefully
2. Explore the existing codebase for patterns and conventions
3. Write failing tests first (TDD)
4. Implement the minimal code to make tests pass
5. Refactor for clarity
6. Commit with a descriptive message

## Constraints

- Follow existing code style and conventions
- Keep changes minimal and focused
- Write tests for all new public functions
- Do not modify unrelated code
```

- [ ] **Step 2: Create code-review skill**

Create `skills/quality/code-review/SKILL.md`:

```markdown
---
name: code-review
version: "1.0"
description: Review code for quality, correctness, and security
when_to_use: When reviewing pull requests, code changes, or evaluating code quality
eval_score: 0.0
tags:
  - review
  - quality
  - security
---
# Code Review

You are an expert code reviewer evaluating code changes.

## Review Priorities

1. **Correctness** — Does the code do what it claims?
2. **Security** — Are there vulnerabilities or unsafe patterns?
3. **Tests** — Are changes adequately tested?
4. **Maintainability** — Is the code clear and well-structured?
5. **Performance** — Are there unnecessary allocations or O(n²) patterns?

## Output Format

Group findings by severity: Critical, High, Medium, Low.
Each finding: file, line, issue, recommendation.
```

- [ ] **Step 3: Create eval-skill blueprint YAML**

Create `tests/testdata/eval-skill-blueprint.yaml`:

```yaml
name: eval-skill-test
version: "0.1"
description: "Tests EvalNode + skill integration"
start: implement
nodes:
  implement:
    type: agentic
    config:
      prompt: "Implement the requested feature"
      skill: "auto"
  evaluate:
    type: eval
    config:
      prompt: "Evaluate the implementation quality"
      criteria:
        - correctness
        - test coverage
        - code style
      threshold: 0.7
edges:
  - from: implement
    to: evaluate
```

- [ ] **Step 4: Add YAML parsing test for eval blueprint**

Add to `core/blueprint/yaml_test.go`:

```go
func TestBuiltinEvalSkillBlueprintValid(t *testing.T) {
	data, err := os.ReadFile("../../tests/testdata/eval-skill-blueprint.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	bp, err := ParseBlueprintYAML(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	executor := &mockExecutor{output: "0.9"}
	g, err := bp.BuildGraph(executor)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	evalNode, ok := g.GetNode("evaluate")
	if !ok {
		t.Fatal("evaluate node not found")
	}
	if evalNode.Type() != NodeTypeEval {
		t.Errorf("evaluate type = %v, want Eval", evalNode.Type())
	}

	implNode, ok := g.GetNode("implement")
	if !ok {
		t.Fatal("implement node not found")
	}
	if implNode.Type() != NodeTypeAgentic {
		t.Errorf("implement type = %v, want Agentic", implNode.Type())
	}
}
```

- [ ] **Step 5: Run all tests**

Run: `go test ./core/blueprint/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add skills/ tests/testdata/eval-skill-blueprint.yaml core/blueprint/yaml_test.go
git commit -m "feat: add built-in skills and eval-skill blueprint test"
```
