# Forge v0.2 Design Spec: Skills, Tools, Triggers, Parallel Runs

**Date:** 2026-04-13  
**Status:** Approved  
**Scope:** All v0.2 features, delivered as three sequential sub-plans

---

## 1. Goal

Extend Forge from a working MVP (v0.1: run blueprints with echo/Claude adapter in Docker) to a **smart factory** with skill-driven agent prompts, evaluator separation, curated tool pools, webhook triggers, and parallel run orchestration.

## 2. What v0.2 does NOT include (deferred to v0.3)

- Memory / session capture / failure-to-rule pipeline
- Multiple new adapters (Goose, Codex, Cursor)
- Warm container pools
- Full quality system (sprint contracts, Playwright MCP)
- Prompt composition stack (5-level override)

## 3. Delivery order

Three sequential sub-plans, each producing testable software:

```text
Sub-plan A: Skills + EvalNode ──► Sub-plan B: Tool Pool + Context ──► Sub-plan C: Triggers + Parallel
     (Layer 1 + 2)                        (Layer 2 + 1)                       (Layer 3)
```

Sub-plan B depends on A (tool pool references skill metadata). Sub-plan C depends on A (parallel runs benefit from EvalNode). C is independent of B.

## 4. Sub-plan A: Skills + EvalNode

### 4.1 EvalNode (Layer 1 — Go)

Add `NodeTypeEval` to the engine. An EvalNode is like an AgenticNode but with a **separate evaluator executor** and **structured grading**:

- New `NodeType` constant: `NodeTypeEval`
- New `EvalNode` struct with fields: `id`, `prompt`, `criteria []string`, `threshold float64`, `executor AgentExecutor`
- `Execute` calls the executor with a prompt that includes the criteria, then parses the response for a numeric score
- If score >= threshold, returns `NodeStatusPassed`; else `NodeStatusFailed` with structured feedback
- YAML type: `"eval"` with `config.prompt`, `config.criteria`, `config.threshold`
- The evaluator uses the same `AgentExecutor` interface as agentic nodes (same gRPC path to harness)

### 4.2 Skill registry (Layer 2 — TypeScript)

Filesystem-based skill storage following the Anthropic Agent Skills spec:

- `harness/src/skills/registry.ts` — `SkillRegistry` class
  - Scans `skills/` directory for `SKILL.md` bundles
  - Parses YAML frontmatter: `name`, `version`, `description`, `when_to_use`, `eval_score`, `tags[]`
  - Returns `Skill[]` with parsed metadata + body content path
  - Methods: `loadAll(skillsDir)`, `findByName(name)`, `findByTag(tag)`

- `harness/src/skills/resolver.ts` — `SkillResolver` class
  - Given a task description + node config, selects the best matching skill
  - Simple keyword + tag matching (no ML/embedding for v0.2)
  - Method: `resolve(taskDescription, nodeConfig) → Skill | null`

- `harness/src/skills/types.ts` — shared types
  - `Skill { name, version, description, whenToUse, evalScore, tags, bodyPath, body }`
  - `SkillFrontmatter` (parsed from YAML)

### 4.3 Skill lifecycle (Layer 2 — TypeScript)

Eval-driven skill lifecycle with quality signals:

- `harness/src/skills/lifecycle.ts` — `SkillLifecycle` class
  - `evaluate(skill, testCases) → EvalResult` — run skill against scenarios, return pass/fail + score
  - `promote(skill, newVersion) → Skill` — bump version, update eval_score in frontmatter
  - `compare(skillA, skillB, testCases) → ComparisonResult` — A/B comparison

- Quality signals stored in frontmatter: `eval_score`, `pass_rate`, `avg_tokens`

### 4.4 Skill integration with harness

- `AgentService.executeAgent()` gains an optional skill resolution step:
  - If `config_json` contains `"skill": "auto"` or `"skill": "<name>"`, resolve the skill
  - Prepend skill body to the composed prompt (before task prompt)
  - If no skill requested, behavior is unchanged (backward compatible)

### 4.5 Built-in skills directory

- Create `skills/` at repo root with 1-2 starter skills:
  - `skills/coding/implement-feature/SKILL.md` — basic implementation skill
  - `skills/quality/code-review/SKILL.md` — basic code review skill

## 5. Sub-plan B: Tool Pool + Context

### 5.1 Tool pool assembly (Layer 2 — TypeScript)

- `harness/src/toolshed/pool.ts` — `assembleToolPool(permissionContext, extensionTools) → Tool[]`
  - Pure function: merge built-in tools with extension/MCP tools
  - Deny rules filter first, then deduplication (built-ins win on name clash)
  - Stable alphabetical sort for prompt cache consistency
  - Returns curated tool list (target: ~15 per task, configurable)

- `harness/src/toolshed/types.ts` — `Tool`, `ToolSource`, `PermissionContext`

### 5.2 Deferred tool loading (Layer 2 — TypeScript)

- `harness/src/toolshed/deferred.ts` — `DeferredToolLoader`
  - When tool definitions exceed configurable threshold of context window (default 15%)
  - Move lower-priority tools behind a search mechanism
  - Method: `partition(tools, budget) → { inline: Tool[], deferred: Tool[] }`

### 5.3 Tool lifecycle hooks (Layer 2 — TypeScript)

- `harness/src/toolshed/hooks.ts` — `ToolHookRegistry`
  - Pre/post hooks for tool invocations
  - `registerPreHook(toolName, fn)`, `registerPostHook(toolName, fn)`
  - Hooks can modify request/response or block execution

### 5.4 Subagent context isolation (Layer 2 — TypeScript)

- `harness/src/context/isolation.ts` — `SubagentContext`
  - Fork parent context with isolated token budget, permission mode
  - Share immutable state: file cache, tool registry snapshot
  - Drop incomplete tool pairs from message history
  - Per-type context trimming (skip git status for explore-only agents)

### 5.5 YAML vocabulary alignment (Layer 1 — Go)

- Add `depends_on` field to `NodeYAML` and `EdgeYAML`
  - `depends_on: [node_a, node_b]` as syntactic sugar for edges
  - Parser generates edges from `depends_on` in addition to explicit `edges[]`
  - Backward compatible: existing blueprints still work

## 6. Sub-plan C: Triggers + Parallel

### 6.1 Webhook trigger API (Layer 3 — Go)

- `factory/triggers/webhook.go` — HTTP server
  - `POST /api/v1/runs` — accept run request JSON, enqueue to run queue
  - `GET /api/v1/runs/:id` — status polling
  - Request shape: `{ task, blueprint, adapter, repo_url, base_branch, no_pr }`
  - Returns: `{ run_id, status }`

- `cmd/forged/main.go` — daemon entrypoint
  - Starts webhook server + run queue consumer
  - Configurable via env vars: `FORGED_PORT`, `FORGED_MAX_PARALLEL`, `FORGED_IMAGE`

### 6.2 Parallel runs (Layer 3 — Go)

- `factory/orchestrator/queue.go` — `RunQueue`
  - Bounded channel-based queue with configurable concurrency
  - Methods: `Enqueue(RunRequest) → RunID`, `Status(RunID) → RunResult`, `Wait(RunID)`
  - Each dequeued request runs through the existing `Pipeline.Execute()`
  - Respects `maxParallel` concurrency limit

- `factory/orchestrator/registry.go` — `RunRegistry`
  - In-memory map of RunID → RunResult (extensible to Redis/DB later)
  - Methods: `Register(RunID)`, `Update(RunID, RunResult)`, `Get(RunID)`

### 6.3 Task assignment (Layer 3 — Go)

- `factory/orchestrator/assignment.go` — `TaskAssigner`
  - Given a RunRequest, select the best adapter based on task type
  - Simple rule-based for v0.2 (e.g., "if task mentions review → code-review skill")
  - Method: `Assign(RunRequest) → adapter string`

## 7. Key design decisions

| Decision | Rationale |
|----------|-----------|
| Skills are filesystem YAML, not a DB | Git-native, reviewable, works offline |
| EvalNode uses same AgentExecutor interface | No new gRPC contract needed; evaluator is just another prompt |
| Tool pool is a pure function | Easy to test, no side effects, predictable |
| Webhook before Slack | Composable foundation; Slack adapter is a thin POST |
| RunQueue is channel-based | Simple Go concurrency; no external dependencies |
| `depends_on` is syntactic sugar | Backward compatible; doesn't replace edges |

## 8. Success criteria

- `forge run` with `--skill auto` resolves and applies a skill to the prompt
- EvalNode in a blueprint gates implementation quality
- Tool pool limits tools to configurable count per task
- `forged` daemon accepts webhook POSTs and runs blueprints in parallel
- All existing v0.1 tests continue to pass
- Each sub-plan has 90%+ test coverage on new code
