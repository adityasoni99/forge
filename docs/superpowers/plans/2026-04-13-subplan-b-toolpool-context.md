# Sub-plan B: Tool Pool + Context Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a tool pool assembly system, deferred tool loading, tool lifecycle hooks, subagent context isolation, and `depends_on` YAML vocabulary in the blueprint engine.

**Architecture:** Tool pool is a pure TypeScript function that merges built-in and extension tools with deny-rule filtering. Deferred loading partitions tools by context budget. Subagent context forks parent context with isolated budgets. `depends_on` is syntactic sugar in Go's YAML parser that generates edges automatically.

**Tech Stack:** Go 1.22+, TypeScript 5.7+, vitest

---

## File Structure

**Create:**
- `harness/src/toolshed/types.ts` — Tool, ToolSource, PermissionContext, DenyRule types
- `harness/src/toolshed/types.test.ts` — type validation tests
- `harness/src/toolshed/pool.ts` — assembleToolPool pure function
- `harness/src/toolshed/pool.test.ts` — pool assembly tests
- `harness/src/toolshed/deferred.ts` — DeferredToolLoader (partition by budget)
- `harness/src/toolshed/deferred.test.ts` — deferred loading tests
- `harness/src/toolshed/hooks.ts` — ToolHookRegistry (pre/post hooks)
- `harness/src/toolshed/hooks.test.ts` — hook tests
- `harness/src/context/isolation.ts` — SubagentContext (forked context)
- `harness/src/context/isolation.test.ts` — isolation tests

**Modify:**
- `core/blueprint/yaml.go` — add `DependsOn` field to NodeYAML, generate edges
- `core/blueprint/yaml_test.go` — add depends_on tests

---

### Task 1: Tool types

**Files:**
- Create: `harness/src/toolshed/types.ts`
- Create: `harness/src/toolshed/types.test.ts`

- [ ] **Step 1: Create tool types**

Create `harness/src/toolshed/types.ts`:

```typescript
export type ToolSource = 'builtin' | 'extension' | 'mcp';

export interface Tool {
  name: string;
  description: string;
  source: ToolSource;
  parameters?: Record<string, unknown>;
}

export interface DenyRule {
  toolName: string;
  reason: string;
}

export interface PermissionContext {
  denyRules: DenyRule[];
  maxTools: number;
}

export function createPermissionContext(
  denyRules: DenyRule[] = [],
  maxTools = 15,
): PermissionContext {
  return { denyRules, maxTools };
}
```

- [ ] **Step 2: Write tests**

Create `harness/src/toolshed/types.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { createPermissionContext } from './types.js';
import type { Tool, DenyRule } from './types.js';

describe('Tool types', () => {
  it('creates permission context with defaults', () => {
    const ctx = createPermissionContext();
    expect(ctx.denyRules).toEqual([]);
    expect(ctx.maxTools).toBe(15);
  });

  it('creates permission context with custom values', () => {
    const rules: DenyRule[] = [{ toolName: 'shell', reason: 'unsafe' }];
    const ctx = createPermissionContext(rules, 10);
    expect(ctx.denyRules).toEqual(rules);
    expect(ctx.maxTools).toBe(10);
  });

  it('tool object has correct shape', () => {
    const tool: Tool = {
      name: 'read_file',
      description: 'Read a file',
      source: 'builtin',
    };
    expect(tool.source).toBe('builtin');
  });
});
```

- [ ] **Step 3: Run tests**

Run: `cd harness && npx vitest run src/toolshed/types.test.ts`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add harness/src/toolshed/types.ts harness/src/toolshed/types.test.ts
git commit -m "feat(harness): add tool pool types"
```

---

### Task 2: Tool pool assembly

**Files:**
- Create: `harness/src/toolshed/pool.ts`
- Create: `harness/src/toolshed/pool.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/toolshed/pool.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { assembleToolPool } from './pool.js';
import type { Tool, PermissionContext } from './types.js';
import { createPermissionContext } from './types.js';

function makeTool(name: string, source: Tool['source'] = 'builtin'): Tool {
  return { name, description: `${name} tool`, source };
}

describe('assembleToolPool', () => {
  it('merges builtin and extension tools', () => {
    const builtins = [makeTool('read'), makeTool('write')];
    const extensions = [makeTool('search', 'extension')];
    const ctx = createPermissionContext();
    const result = assembleToolPool(builtins, extensions, ctx);
    expect(result).toHaveLength(3);
  });

  it('filters denied tools', () => {
    const builtins = [makeTool('read'), makeTool('shell')];
    const extensions: Tool[] = [];
    const ctx = createPermissionContext([{ toolName: 'shell', reason: 'unsafe' }]);
    const result = assembleToolPool(builtins, extensions, ctx);
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('read');
  });

  it('deduplicates: builtin wins on name clash', () => {
    const builtins = [makeTool('read', 'builtin')];
    const extensions = [makeTool('read', 'extension')];
    const ctx = createPermissionContext();
    const result = assembleToolPool(builtins, extensions, ctx);
    expect(result).toHaveLength(1);
    expect(result[0].source).toBe('builtin');
  });

  it('sorts alphabetically for cache consistency', () => {
    const builtins = [makeTool('write'), makeTool('read'), makeTool('grep')];
    const ctx = createPermissionContext();
    const result = assembleToolPool(builtins, [], ctx);
    expect(result.map((t) => t.name)).toEqual(['grep', 'read', 'write']);
  });

  it('caps at maxTools', () => {
    const builtins = Array.from({ length: 20 }, (_, i) => makeTool(`tool-${String(i).padStart(2, '0')}`));
    const ctx = createPermissionContext([], 5);
    const result = assembleToolPool(builtins, [], ctx);
    expect(result).toHaveLength(5);
  });

  it('returns empty when all tools denied', () => {
    const builtins = [makeTool('shell')];
    const ctx = createPermissionContext([{ toolName: 'shell', reason: 'no' }]);
    const result = assembleToolPool(builtins, [], ctx);
    expect(result).toEqual([]);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/toolshed/pool.test.ts`
Expected: FAIL — cannot import `assembleToolPool`

- [ ] **Step 3: Implement assembleToolPool**

Create `harness/src/toolshed/pool.ts`:

```typescript
import type { Tool, PermissionContext } from './types.js';

export function assembleToolPool(
  builtins: Tool[],
  extensions: Tool[],
  ctx: PermissionContext,
): Tool[] {
  const denySet = new Set(ctx.denyRules.map((r) => r.toolName));

  const byName = new Map<string, Tool>();
  for (const tool of builtins) {
    if (!denySet.has(tool.name)) {
      byName.set(tool.name, tool);
    }
  }
  for (const tool of extensions) {
    if (!denySet.has(tool.name) && !byName.has(tool.name)) {
      byName.set(tool.name, tool);
    }
  }

  const sorted = [...byName.values()].sort((a, b) => a.name.localeCompare(b.name));
  return sorted.slice(0, ctx.maxTools);
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/toolshed/pool.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/toolshed/pool.ts harness/src/toolshed/pool.test.ts
git commit -m "feat(harness): add assembleToolPool pure function"
```

---

### Task 3: Deferred tool loading

**Files:**
- Create: `harness/src/toolshed/deferred.ts`
- Create: `harness/src/toolshed/deferred.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/toolshed/deferred.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { DeferredToolLoader } from './deferred.js';
import type { Tool } from './types.js';

function makeTool(name: string, descLen = 50): Tool {
  return {
    name,
    description: 'x'.repeat(descLen),
    source: 'builtin',
  };
}

describe('DeferredToolLoader', () => {
  it('keeps all tools inline when under budget', () => {
    const tools = [makeTool('a', 10), makeTool('b', 10)];
    const loader = new DeferredToolLoader(1000);
    const { inline, deferred } = loader.partition(tools);
    expect(inline).toHaveLength(2);
    expect(deferred).toHaveLength(0);
  });

  it('defers tools exceeding budget', () => {
    const tools = [
      makeTool('a', 100),
      makeTool('b', 100),
      makeTool('c', 100),
    ];
    const loader = new DeferredToolLoader(60);
    const { inline, deferred } = loader.partition(tools);
    expect(inline.length).toBeGreaterThanOrEqual(1);
    expect(deferred.length).toBeGreaterThanOrEqual(1);
    expect(inline.length + deferred.length).toBe(3);
  });

  it('returns all deferred when budget is zero', () => {
    const tools = [makeTool('a', 100)];
    const loader = new DeferredToolLoader(0);
    const { inline, deferred } = loader.partition(tools);
    expect(inline).toHaveLength(0);
    expect(deferred).toHaveLength(1);
  });

  it('handles empty tool list', () => {
    const loader = new DeferredToolLoader(1000);
    const { inline, deferred } = loader.partition([]);
    expect(inline).toEqual([]);
    expect(deferred).toEqual([]);
  });

  it('estimateToolTokens returns positive number', () => {
    const loader = new DeferredToolLoader(1000);
    const tokens = loader.estimateToolTokens(makeTool('test', 100));
    expect(tokens).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/toolshed/deferred.test.ts`
Expected: FAIL — cannot import `DeferredToolLoader`

- [ ] **Step 3: Implement DeferredToolLoader**

Create `harness/src/toolshed/deferred.ts`:

```typescript
import type { Tool } from './types.js';

export interface PartitionResult {
  inline: Tool[];
  deferred: Tool[];
}

export class DeferredToolLoader {
  constructor(private readonly budgetTokens: number) {}

  partition(tools: Tool[]): PartitionResult {
    const inline: Tool[] = [];
    const deferred: Tool[] = [];
    let used = 0;

    for (const tool of tools) {
      const cost = this.estimateToolTokens(tool);
      if (used + cost <= this.budgetTokens) {
        inline.push(tool);
        used += cost;
      } else {
        deferred.push(tool);
      }
    }

    return { inline, deferred };
  }

  estimateToolTokens(tool: Tool): number {
    const text = `${tool.name} ${tool.description} ${JSON.stringify(tool.parameters ?? {})}`;
    return Math.ceil(text.length / 4);
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/toolshed/deferred.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/toolshed/deferred.ts harness/src/toolshed/deferred.test.ts
git commit -m "feat(harness): add DeferredToolLoader for context-aware partitioning"
```

---

### Task 4: Tool lifecycle hooks

**Files:**
- Create: `harness/src/toolshed/hooks.ts`
- Create: `harness/src/toolshed/hooks.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/toolshed/hooks.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { ToolHookRegistry } from './hooks.js';
import type { ToolInvocation, ToolResult } from './hooks.js';

describe('ToolHookRegistry', () => {
  it('runs pre-hook before tool invocation', async () => {
    const registry = new ToolHookRegistry();
    const calls: string[] = [];
    registry.registerPreHook('read_file', async (inv) => {
      calls.push(`pre:${inv.toolName}`);
      return inv;
    });

    const inv: ToolInvocation = { toolName: 'read_file', args: { path: '/a.ts' } };
    const result = await registry.runPreHooks(inv);
    expect(calls).toEqual(['pre:read_file']);
    expect(result.toolName).toBe('read_file');
  });

  it('runs post-hook after tool invocation', async () => {
    const registry = new ToolHookRegistry();
    const calls: string[] = [];
    registry.registerPostHook('write_file', async (_inv, res) => {
      calls.push(`post:${res.success}`);
      return res;
    });

    const inv: ToolInvocation = { toolName: 'write_file', args: {} };
    const res: ToolResult = { output: 'written', success: true };
    const result = await registry.runPostHooks(inv, res);
    expect(calls).toEqual(['post:true']);
    expect(result.success).toBe(true);
  });

  it('pre-hook can modify invocation', async () => {
    const registry = new ToolHookRegistry();
    registry.registerPreHook('shell', async (inv) => {
      return { ...inv, args: { ...inv.args, timeout: 30 } };
    });

    const inv: ToolInvocation = { toolName: 'shell', args: { command: 'ls' } };
    const result = await registry.runPreHooks(inv);
    expect(result.args).toEqual({ command: 'ls', timeout: 30 });
  });

  it('pre-hook can block by returning null', async () => {
    const registry = new ToolHookRegistry();
    registry.registerPreHook('shell', async () => null);

    const inv: ToolInvocation = { toolName: 'shell', args: {} };
    const result = await registry.runPreHooks(inv);
    expect(result).toBeNull();
  });

  it('ignores hooks for other tools', async () => {
    const registry = new ToolHookRegistry();
    const calls: string[] = [];
    registry.registerPreHook('read_file', async (inv) => {
      calls.push('read');
      return inv;
    });

    const inv: ToolInvocation = { toolName: 'write_file', args: {} };
    const result = await registry.runPreHooks(inv);
    expect(calls).toEqual([]);
    expect(result.toolName).toBe('write_file');
  });

  it('chains multiple pre-hooks in order', async () => {
    const registry = new ToolHookRegistry();
    const order: number[] = [];
    registry.registerPreHook('shell', async (inv) => {
      order.push(1);
      return inv;
    });
    registry.registerPreHook('shell', async (inv) => {
      order.push(2);
      return inv;
    });

    const inv: ToolInvocation = { toolName: 'shell', args: {} };
    await registry.runPreHooks(inv);
    expect(order).toEqual([1, 2]);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/toolshed/hooks.test.ts`
Expected: FAIL — cannot import `ToolHookRegistry`

- [ ] **Step 3: Implement ToolHookRegistry**

Create `harness/src/toolshed/hooks.ts`:

```typescript
export interface ToolInvocation {
  toolName: string;
  args: Record<string, unknown>;
}

export interface ToolResult {
  output: string;
  success: boolean;
  error?: string;
}

export type PreHook = (inv: ToolInvocation) => Promise<ToolInvocation | null>;
export type PostHook = (inv: ToolInvocation, res: ToolResult) => Promise<ToolResult>;

export class ToolHookRegistry {
  private preHooks = new Map<string, PreHook[]>();
  private postHooks = new Map<string, PostHook[]>();

  registerPreHook(toolName: string, hook: PreHook): void {
    const hooks = this.preHooks.get(toolName) ?? [];
    hooks.push(hook);
    this.preHooks.set(toolName, hooks);
  }

  registerPostHook(toolName: string, hook: PostHook): void {
    const hooks = this.postHooks.get(toolName) ?? [];
    hooks.push(hook);
    this.postHooks.set(toolName, hooks);
  }

  async runPreHooks(inv: ToolInvocation): Promise<ToolInvocation | null> {
    const hooks = this.preHooks.get(inv.toolName) ?? [];
    let current: ToolInvocation | null = inv;
    for (const hook of hooks) {
      if (current === null) return null;
      current = await hook(current);
    }
    return current;
  }

  async runPostHooks(inv: ToolInvocation, res: ToolResult): Promise<ToolResult> {
    const hooks = this.postHooks.get(inv.toolName) ?? [];
    let current = res;
    for (const hook of hooks) {
      current = await hook(inv, current);
    }
    return current;
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/toolshed/hooks.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/toolshed/hooks.ts harness/src/toolshed/hooks.test.ts
git commit -m "feat(harness): add ToolHookRegistry with pre/post hooks"
```

---

### Task 5: Subagent context isolation

**Files:**
- Create: `harness/src/context/isolation.ts`
- Create: `harness/src/context/isolation.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `harness/src/context/isolation.test.ts`:

```typescript
import { describe, it, expect } from 'vitest';
import { SubagentContext, SubagentType } from './isolation.js';

describe('SubagentContext', () => {
  it('creates forked context with isolated budget', () => {
    const parent = {
      maxTokens: 8000,
      rules: [{ name: 'rule1', content: 'Always test' }],
      tools: ['read_file', 'write_file', 'shell'],
      fileCache: new Map([['a.ts', 'content']]),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Explore,
      maxTokens: 4000,
    });

    expect(child.maxTokens).toBe(4000);
    expect(child.rules).toEqual(parent.rules);
    expect(child.fileCache.get('a.ts')).toBe('content');
  });

  it('explore agents exclude write tools', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: ['read_file', 'write_file', 'shell', 'grep'],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Explore,
      maxTokens: 4000,
    });

    expect(child.tools).toContain('read_file');
    expect(child.tools).toContain('grep');
    expect(child.tools).not.toContain('write_file');
    expect(child.tools).not.toContain('shell');
  });

  it('implement agents keep all tools', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: ['read_file', 'write_file', 'shell'],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Implement,
      maxTokens: 6000,
    });

    expect(child.tools).toEqual(['read_file', 'write_file', 'shell']);
  });

  it('child cannot mutate parent file cache', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: [],
      fileCache: new Map([['a.ts', 'original']]),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Implement,
      maxTokens: 4000,
    });

    child.fileCache.set('b.ts', 'new file');
    expect(parent.fileCache.has('b.ts')).toBe(false);
  });

  it('review agents exclude shell but keep read/write', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: ['read_file', 'write_file', 'shell', 'grep'],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Review,
      maxTokens: 4000,
    });

    expect(child.tools).toContain('read_file');
    expect(child.tools).toContain('write_file');
    expect(child.tools).not.toContain('shell');
  });

  it('defaults maxTokens to half parent budget', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: [],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Implement,
    });

    expect(child.maxTokens).toBe(4000);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd harness && npx vitest run src/context/isolation.test.ts`
Expected: FAIL — cannot import `SubagentContext`

- [ ] **Step 3: Implement SubagentContext**

Create `harness/src/context/isolation.ts`:

```typescript
import type { RuleFile } from './loader.js';

export enum SubagentType {
  Explore = 'explore',
  Implement = 'implement',
  Review = 'review',
}

export interface ParentContext {
  maxTokens: number;
  rules: RuleFile[];
  tools: string[];
  fileCache: Map<string, string>;
}

export interface ForkOptions {
  type: SubagentType;
  maxTokens?: number;
}

const WRITE_TOOLS = new Set(['write_file', 'shell', 'edit_file', 'create_file', 'delete_file']);
const SHELL_TOOLS = new Set(['shell', 'execute_command']);

export class SubagentContext {
  readonly maxTokens: number;
  readonly rules: RuleFile[];
  readonly tools: string[];
  readonly fileCache: Map<string, string>;
  readonly type: SubagentType;

  private constructor(
    type: SubagentType,
    maxTokens: number,
    rules: RuleFile[],
    tools: string[],
    fileCache: Map<string, string>,
  ) {
    this.type = type;
    this.maxTokens = maxTokens;
    this.rules = rules;
    this.tools = tools;
    this.fileCache = fileCache;
  }

  static fork(parent: ParentContext, options: ForkOptions): SubagentContext {
    const maxTokens = options.maxTokens ?? Math.floor(parent.maxTokens / 2);
    const rules = [...parent.rules];
    const fileCache = new Map(parent.fileCache);
    const tools = filterToolsForType(parent.tools, options.type);

    return new SubagentContext(options.type, maxTokens, rules, tools, fileCache);
  }
}

function filterToolsForType(tools: string[], type: SubagentType): string[] {
  switch (type) {
    case SubagentType.Explore:
      return tools.filter((t) => !WRITE_TOOLS.has(t) && !SHELL_TOOLS.has(t));
    case SubagentType.Review:
      return tools.filter((t) => !SHELL_TOOLS.has(t));
    case SubagentType.Implement:
      return [...tools];
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd harness && npx vitest run src/context/isolation.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add harness/src/context/isolation.ts harness/src/context/isolation.test.ts
git commit -m "feat(harness): add SubagentContext isolation with per-type tool filtering"
```

---

### Task 6: YAML `depends_on` vocabulary alignment

**Files:**
- Modify: `core/blueprint/yaml.go`
- Modify: `core/blueprint/yaml_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `core/blueprint/yaml_test.go`:

```go
func TestDependsOnGeneratesEdges(t *testing.T) {
	yamlData := `
name: depends-test
version: "0.1"
start: plan
nodes:
  plan:
    type: agentic
    config:
      prompt: "Create plan"
  implement:
    type: agentic
    depends_on:
      - plan
    config:
      prompt: "Implement"
  test:
    type: deterministic
    depends_on:
      - implement
    config:
      command: "echo test"
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{output: "ok"})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if err := g.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	next := g.NextNodes("plan", "")
	found := false
	for _, n := range next {
		if n == "implement" {
			found = true
		}
	}
	if !found {
		t.Errorf("plan -> implement edge not generated from depends_on; next = %v", next)
	}

	next2 := g.NextNodes("implement", "")
	found2 := false
	for _, n := range next2 {
		if n == "test" {
			found2 = true
		}
	}
	if !found2 {
		t.Errorf("implement -> test edge not generated from depends_on; next = %v", next2)
	}
}

func TestDependsOnCombinesWithExplicitEdges(t *testing.T) {
	yamlData := `
name: combined
version: "0.1"
start: a
nodes:
  a:
    type: deterministic
    config:
      command: "echo a"
  b:
    type: deterministic
    depends_on:
      - a
    config:
      command: "echo b"
  c:
    type: deterministic
    config:
      command: "echo c"
edges:
  - from: a
    to: c
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{output: "ok"})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}

	next := g.NextNodes("a", "")
	if len(next) < 2 {
		t.Errorf("a should have edges to both b and c; got %v", next)
	}
}

func TestDependsOnUnknownNodeErrors(t *testing.T) {
	yamlData := `
name: bad-dep
version: "0.1"
start: a
nodes:
  a:
    type: deterministic
    depends_on:
      - ghost
    config:
      command: "echo a"
edges: []
`
	bp, err := ParseBlueprintYAML([]byte(yamlData))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	_, err = bp.BuildGraph(&mockExecutor{})
	if err == nil {
		t.Fatal("expected error for depends_on referencing unknown node")
	}
}

func TestBackwardCompatibilityNoDependsOn(t *testing.T) {
	bp, err := ParseBlueprintYAML([]byte(testYAML))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	g, err := bp.BuildGraph(&mockExecutor{output: "done"})
	if err != nil {
		t.Fatalf("BuildGraph: %v", err)
	}
	if g.NodeCount() != 4 {
		t.Errorf("NodeCount = %d, want 4", g.NodeCount())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./core/blueprint/ -run "TestDependsOn" -v`
Expected: FAIL — compilation error (DependsOn field doesn't exist on NodeYAML)

- [ ] **Step 3: Add DependsOn to NodeYAML and generate edges in BuildGraph**

In `core/blueprint/yaml.go`, add the `DependsOn` field to `NodeYAML`:

```go
type NodeYAML struct {
	Type            string                 `yaml:"type"`
	Description     string                 `yaml:"description,omitempty"`
	ConcurrencySafe *bool                  `yaml:"concurrency_safe,omitempty"`
	AllowedTools    []string               `yaml:"allowed_tools,omitempty"`
	MaxRetries      int                    `yaml:"max_retries,omitempty"`
	DependsOn       []string               `yaml:"depends_on,omitempty"`
	Config          map[string]interface{} `yaml:"config"`
}
```

In the same file, update the `BuildGraph` method to generate edges from `depends_on`:

```go
func (bp *BlueprintYAML) BuildGraph(executor AgentExecutor) (*Graph, error) {
	g := NewGraph()
	if err := bp.addNodesToGraph(g, executor); err != nil {
		return nil, err
	}
	if err := bp.addDependsOnEdges(g); err != nil {
		return nil, err
	}
	for _, ey := range bp.Edges {
		if err := g.AddEdge(Edge{From: ey.From, To: ey.To, Condition: ey.Condition}); err != nil {
			return nil, err
		}
	}
	if err := g.SetStartNode(bp.Start); err != nil {
		return nil, err
	}
	return g, nil
}

func (bp *BlueprintYAML) addDependsOnEdges(g *Graph) error {
	for id, ny := range bp.Nodes {
		for _, dep := range ny.DependsOn {
			if err := g.AddEdge(Edge{From: dep, To: id}); err != nil {
				return fmt.Errorf("depends_on edge %s -> %s: %w", dep, id, err)
			}
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./core/blueprint/ -run "TestDependsOn|TestBackwardCompatibility" -v`
Expected: all PASS

- [ ] **Step 5: Run all blueprint tests**

Run: `go test ./core/blueprint/ -v`
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add core/blueprint/yaml.go core/blueprint/yaml_test.go
git commit -m "feat(engine): add depends_on YAML vocabulary for declarative edges"
```
