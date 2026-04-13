# Forge — Roadmap

> **Purpose:** Single checkpoint file for tracking what's done, what's next, and
> where to resume. Reference this at the start of every new chat session.
>
> **Last updated:** 2026-04-13

---

## Version map

| Version | Theme | Status |
|---------|-------|--------|
| **v0.1** | Blueprint Engine + Harness MVP + Factory MVP + Integration | **Complete** |
| **v0.2** | Skills, tool pool, triggers, parallel runs | **In progress** |
| **v0.3** | Multi-adapter, warm pools, learning loops | Planned |
| **v1.0** | Production-ready factory, docs, community | Planned |

---

## v0.1 — MVP (current)

### Layer 1: Blueprint Engine (Go) — COMPLETE

| # | Task | Plan file | Status |
|---|------|-----------|--------|
| 0 | Project scaffolding (go mod, dirs, AGENTS.md, project.md) | `layer_1_blueprint_engine_4bd3f740` | Done |
| 1 | Core types (NodeType, NodeStatus, Node, RunState) | same | Done |
| 2 | Graph construction (AddNode, AddEdge, Validate, NextNodes) | same | Done |
| 3 | DeterministicNode (shell exec + timeout) | same | Done |
| 4 | GateNode (conditional routing) | same | Done |
| 5 | AgenticNode (pluggable AgentExecutor) | same | Done |
| 6 | Engine (state machine, gate routing, iteration guard) | same | Done |
| 7 | YAML parser (two-phase parse/build) | same | Done |
| 8 | Built-in blueprints (standard-implementation, bug-fix) | same | Done |
| 9 | CLI skeleton (validate, list, run) | same | Done |
| 10 | Coverage + cleanup (91% achieved) | same | Done |

**Extras implemented beyond plan:** `engine_parallel.go` (concurrent node execution),
`hooks.go` (lifecycle hooks), `permissions.go` (permission model). 75 tests, 91% coverage.

---

### Layer 2: Harness MVP (TypeScript) — COMPLETE

| # | Task | Plan file | Status |
|---|------|-----------|--------|
| 0 | Proto definition + Go code generation (`agent.proto`, buf) | `layer_2_harness_mvp_07ee3081` | Done |
| 1 | TypeScript project scaffolding (package.json, tsconfig, vitest) | same | Done |
| 2 | Agent adapter interface + echo adapter | same | Done |
| 3 | Context loader (AGENTS.md + `.forge/rules/`) | same | Done |
| 4 | Claude Code adapter (headless CLI wrapper) | same | Done |
| 5 | Agent service + gRPC server | same | Done |
| 6 | Go gRPC client (`GrpcAgentExecutor`, `--harness` flag) | same | Done |
| 7 | Integration test (Go engine ↔ TS harness) | same | Done |
| 8 | Coverage + cleanup (90%+ target) | same | Done |

**Extras:** 21 TS tests (100% statement coverage), 4 Go gRPC tests (94.1% coverage),
1 integration test (Go engine ↔ TS harness e2e).

**Key deliverables:** `proto/forge/v1/agent.proto`, `harness/` TypeScript package,
`internal/grpcexec/` Go gRPC client, headless Claude Code adapter.

---

### Layer 3: Factory MVP (Go) — COMPLETE

| # | Task | Plan file | Status |
|---|------|-----------|--------|
| 0 | Sandbox types + interface (types.go, sandbox.go) | `layer_3_factory_mvp_f6c28aa0` | Done |
| 1 | Docker sandbox implementation | same | Done |
| 2 | Workspace manager with git worktrees | same | Done |
| 3 | Delivery pipeline (git push + PR creation) | same | Done |
| 4 | Run pipeline orchestrator | same | Done |
| 5 | Dockerfile + sandbox entry script + Makefile | same | Done |
| 6 | CLI `forge run` command wiring | same | Done |
| 7 | Integration test (Docker sandbox e2e) | same | Done |
| 8 | Coverage + cleanup (100% achieved) | same | Done |

**Key deliverables:** `factory/sandbox/`, `factory/workspace/`, `factory/orchestrator/`,
`factory/delivery/`, Dockerfile, `scripts/sandbox-entry.sh`, `forge run "task"` CLI command.

**Extras:** 100% statement coverage across all 4 factory packages. 35+ tests total.
`CommandRunner` interface for testable Docker/git CLI calls. Git worktree isolation.

---

### Layer 4: Integration + Polish — COMPLETE

| Task | Plan file | Status |
|------|-----------|--------|
| Blueprint source resolution + task templating | `2026-04-12-layer-4-integration-polish` | Done |
| Align forge run, local mode, Docker entrypoint | same | Done |
| Deterministic smoke path for integration tests | same | Done |
| CI pipeline (GitHub Actions: Go, TS, Docker) | same | Done |
| README quickstart guide | same | Done |
| Design doc reconciliation | same | Done |

---

## v0.2 — Skills + Tool Pool + Triggers (in progress)

**Design spec:** `docs/superpowers/specs/2026-04-13-v02-skills-tools-triggers-design.md`

Delivery order: Sub-plan A → Sub-plan B → Sub-plan C

### Sub-plan A: Skills + EvalNode (Layer 1 + 2)

| # | Task | Status |
|---|------|--------|
| 1 | Add NodeTypeEval to engine type system | Not started |
| 2 | EvalNode struct and execution logic | Not started |
| 3 | Eval node YAML parsing | Not started |
| 4 | Skill types and frontmatter parser (TS) | Not started |
| 5 | Skill registry (filesystem scan) | Not started |
| 6 | Skill resolver (keyword matching) | Not started |
| 7 | Skill lifecycle (evaluate, promote, compare) | Not started |
| 8 | Integrate skills into AgentService | Not started |
| 9 | Built-in skills + end-to-end YAML test | Not started |

### Sub-plan B: Tool Pool + Context (Layer 2 + 1)

| # | Task | Status |
|---|------|--------|
| 1 | Tool types (TS) | Not started |
| 2 | Tool pool assembly (pure function) | Not started |
| 3 | Deferred tool loading (context budget) | Not started |
| 4 | Tool lifecycle hooks (pre/post) | Not started |
| 5 | Subagent context isolation | Not started |
| 6 | YAML `depends_on` vocabulary alignment | Not started |

### Sub-plan C: Triggers + Parallel (Layer 3)

| # | Task | Status |
|---|------|--------|
| 1 | RunRegistry (in-memory run tracking) | Not started |
| 2 | RunQueue (bounded concurrency) | Not started |
| 3 | TaskAssigner (rule-based adapter selection) | Not started |
| 4 | Webhook HTTP handler | Not started |
| 5 | forged daemon entrypoint | Not started |
| 6 | Integration smoke test | Not started |

---

## v0.3 — Learning + Multi-adapter (planned)

| Feature | Layer | Source |
|---------|-------|--------|
| Memory / session capture | 2 | Master plan |
| Failure-to-rule pipeline | 2 | Master plan |
| Doc-gardening agent | 2 | Master plan |
| Multiple agent adapters (Goose, Codex, Cursor) | 2 | Master plan |
| Container warm pools | 3 | Master plan |
| Full quality gate system (sprint contracts) | 2 | Master plan |
| Prompt composition stack (5-level override) | 2 | design.md §5.3 |
| Permission pipeline (deterministic + async) | 3 | design.md §11 |

---

## v1.0 — Production-ready (planned)

| Feature | Layer | Source |
|---------|-------|--------|
| Built-in skills (planning, coding, quality, context) | 2 | Master plan |
| Full documentation suite | — | Master plan |
| Community skill marketplace | 2 | OpenSpace prior art |
| Webhook + GitHub Issues triggers | 3 | Master plan |
| Human review queue + auto-merge policies | 3 | Master plan |
| Run tracing + token analytics dashboard | 3 | Master plan |
| Daemon mode (`forged`) | 3 | Master plan |

---

## Plan file index

| Plan | ID | Layer | Status |
|------|----|-------|--------|
| Forge Agent Factory (master) | `forge_agent_factory_86e877d4` | All | Reference doc |
| Layer 1: Blueprint Engine | `layer_1_blueprint_engine_4bd3f740` | 1 | **Complete** |
| Layer 2: Harness MVP | `layer_2_harness_mvp_07ee3081` | 2 | **Complete** |
| Layer 3: Factory MVP | `layer_3_factory_mvp_f6c28aa0` | 3 | **Complete** |
| Layer 4: Integration + Polish | `2026-04-12-layer-4-integration-polish` | 1–3 | **Complete** |
| v0.2 Design Spec | `2026-04-13-v02-skills-tools-triggers-design` | All | Reference doc |
| v0.2 Sub-plan A: Skills + EvalNode | `2026-04-13-subplan-a-skills-evalnode` | 1–2 | Not started |
| v0.2 Sub-plan B: Tool Pool + Context | `2026-04-13-subplan-b-toolpool-context` | 1–2 | Not started |
| v0.2 Sub-plan C: Triggers + Parallel | `2026-04-13-subplan-c-triggers-parallel` | 3 | Not started |

v0.1 plans: `.cursor/plans/*.plan.md`
v0.2 plans: `docs/superpowers/plans/2026-04-13-subplan-*.md`

---

## Design documents

| Document | Purpose |
|----------|---------|
| [docs/design.md](docs/design.md) | Canonical architecture, ADRs, layer specs |
| [docs/prd-forge.md](docs/prd-forge.md) | Product requirements, MVP scope, user stories |
| [project.md](project.md) | Module map, dependency diagram, stories, status |
| [references/references.md](references/references.md) | Curated external links and research |
| [references/referlinks.md](references/referlinks.md) | Audit trail of all researched links |
| [AGENTS.md](AGENTS.md) | Contributor quick reference and conventions |

---

## How to resume

1. Open this file at the start of a new chat.
2. Find the **first "Not started"** task in the current version (v0.1).
3. Open the corresponding plan file (`.cursor/plans/<id>.plan.md`).
4. Reference `project.md` for module map and `docs/design.md` for architecture.
5. Begin implementing task-by-task per the plan's instructions.

**Current checkpoint:** v0.1 MVP complete. v0.2 design spec + 3 sub-plans written.
**Next action:** Execute Sub-plan A (Skills + EvalNode) using subagent-driven development.
