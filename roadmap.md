# Forge — Roadmap

> **Purpose:** Single checkpoint file for tracking what's done, what's next, and
> where to resume. Reference this at the start of every new chat session.
>
> **Last updated:** 2026-04-16

---

## Version map

| Version | Theme | Status |
|---------|-------|--------|
| **v0.1** | Blueprint Engine + Harness MVP + Factory MVP + Integration | **Complete** |
| **v0.2** | Skills, tool pool, triggers, parallel runs | **Complete** |
| **v0.3** | Multi-adapter, warm pools, learning loops, agent plugin | **Complete** (Sub-plans A–E all done) |
| **v0.3.1** | Agent Plugin System (IDE integration) | **Complete** |
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

**Plan:** [`docs/superpowers/plans/2026-04-12-layer-4-integration-polish.md`](docs/superpowers/plans/2026-04-12-layer-4-integration-polish.md) — all steps checked off in the plan doc.

| Task | Plan ID | Status |
|------|---------|--------|
| Blueprint source resolution + task templating | `2026-04-12-layer-4-integration-polish` | Done |
| Align forge run, local mode, Docker entrypoint | same | Done |
| Deterministic smoke path for integration tests | same | Done |
| CI pipeline (GitHub Actions: Go, TS, Docker) | same | Done |
| README quickstart guide | same | Done |
| Design doc reconciliation | same | Done |

---

## v0.2 — Skills + Tool Pool + Triggers (complete)

**Design spec:** [`docs/superpowers/specs/2026-04-13-v02-skills-tools-triggers-design.md`](docs/superpowers/specs/2026-04-13-v02-skills-tools-triggers-design.md)

Delivery order: Sub-plan A → Sub-plan B → Sub-plan C. **All three sub-plans are complete.** v0.2 is feature-complete.

### Sub-plan A: Skills + EvalNode (Layer 1 + 2) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-13-subplan-a-skills-evalnode.md`](docs/superpowers/plans/2026-04-13-subplan-a-skills-evalnode.md)

| # | Task | Status |
|---|------|--------|
| 1 | Add NodeTypeEval to engine type system | Done |
| 2 | EvalNode struct and execution logic | Done |
| 3 | Eval node YAML parsing | Done |
| 4 | Skill types and frontmatter parser (TS) | Done |
| 5 | Skill registry (filesystem scan) | Done |
| 6 | Skill resolver (keyword matching) | Done |
| 7 | Skill lifecycle (evaluate, promote, compare) | Done |
| 8 | Integrate skills into AgentService | Done |
| 9 | Built-in skills + end-to-end YAML test | Done |

**Delivered:** `NodeTypeEval` + `EvalNode` + YAML `type: eval`; `harness/src/skills/*` (types, registry, resolver, lifecycle); `AgentService` optional skill resolution; built-in `skills/coding/implement-feature` and `skills/quality/code-review`; `tests/testdata/eval-skill-blueprint.yaml` smoke test.

### Sub-plan B: Tool Pool + Context (Layer 2 + 1) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-13-subplan-b-toolpool-context.md`](docs/superpowers/plans/2026-04-13-subplan-b-toolpool-context.md)

| # | Task | Status |
|---|------|--------|
| 1 | Tool types (TS) | Done |
| 2 | Tool pool assembly (pure function) | Done |
| 3 | Deferred tool loading (context budget) | Done |
| 4 | Tool lifecycle hooks (pre/post) | Done |
| 5 | Subagent context isolation | Done |
| 6 | YAML `depends_on` vocabulary alignment | Done |

**Delivered:** `harness/src/toolshed/*` (types, pool, deferred, hooks); `harness/src/context/isolation.ts` (SubagentContext); `core/blueprint/yaml.go` `depends_on` field + edge generation. 26 new TS tests (75 total), 4 new Go tests (88 total blueprint).

### Sub-plan C: Triggers + Parallel (Layer 3) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-13-subplan-c-triggers-parallel.md`](docs/superpowers/plans/2026-04-13-subplan-c-triggers-parallel.md)

| # | Task | Status |
|---|------|--------|
| 1 | RunRegistry (in-memory run tracking) | Done |
| 2 | RunQueue (bounded concurrency) | Done |
| 3 | TaskAssigner (rule-based adapter selection) | Done |
| 4 | Webhook HTTP handler | Done |
| 5 | forged daemon entrypoint | Done |
| 6 | Integration smoke test | Done |

**Delivered:** `factory/orchestrator/registry.go` (concurrent-safe in-memory run tracking), `queue.go` (bounded-concurrency worker pool with `PipelineExecutor` interface), `assignment.go` (rule-based adapter selection, defaults to "claude"); `factory/triggers/webhook.go` (POST/GET `/api/v1/runs` HTTP handler with `Enqueuer`/`StatusGetter` interfaces); `cmd/forged/main.go` (daemon skeleton with signal handling, graceful shutdown); integration smoke test. 32 tests total (orchestrator + triggers), all passing with `-race`.

**Historical note (v0.2):** Three limitations were listed here; **all are resolved in v0.3 Sub-plan A** (queue drain + `Shutdown`, `repo_url` → `GitRepoResolver`, `TaskAssigner` + `SessionLog` in `Pipeline` + `forged` wiring). See v0.3 section below.

---

## v0.3 — Learning + Multi-adapter

**Design spec:** [`docs/superpowers/specs/2026-04-15-v03-learning-multiadapter-design.md`](docs/superpowers/specs/2026-04-15-v03-learning-multiadapter-design.md)

Delivery order: **A → B → (C and D in parallel) → E** (see design spec §3).

### Sub-plan A: v0.2 Debt + Factory Hardening (Layer 3) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-15-subplan-a-v02-debt.md`](docs/superpowers/plans/2026-04-15-subplan-a-v02-debt.md)

| Deliverable | Notes |
|---------------|--------|
| `SessionLog` + `FileSessionLog` | Append-only JSONL per run; foundation for learning / restart recovery |
| `RunQueue.Shutdown` | Graceful drain (WaitGroup + two-phase wait) |
| `RepoResolver` + webhook `repo_url` | Bare-clone cache; URL validation (blocks unsafe git transports) |
| `Pipeline` options | `WithTaskAssigner`, `WithSessionLog`; session events at pipeline phases |
| `forged` production wiring | Real pipeline + `--dry-run`, `--sessions-dir`, `--repo-cache-dir`; queue + HTTP shutdown |
| Integration + docs | `TestSubplanAIntegration`; Managed Agents refs in `references/` |

**Branch merged to `main`:** `v0.3/subplan-a-debt-hardening` (commits through Sub-plan A completion).

### Sub-plan B: Multi-Adapter + Prompt Composition (Layer 2) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-15-subplan-b-multiadapter-prompt.md`](docs/superpowers/plans/2026-04-15-subplan-b-multiadapter-prompt.md)

| Deliverable | Notes |
|-------------|--------|
| `AgentEvent` + capability-aware `AgentAdapter` | Streaming `execute()`, `getCapabilities()`, `interrupt()` |
| `SyncAdapterWrapper` | Bridges sync adapters to streaming protocol |
| Echo / Claude / Goose / Codex / Cursor adapters | CLI-spawn pattern; registered in harness server |
| `composePromptStack` + `compressShellOutput` | 5-level prompt stack; shell output compression |
| `AgentService` + proto `adapter` field | Multi-adapter registry; prompt stack + adapter selection |
| Tests | Harness: 111 tests (vitest); Go `grpcexec` after `buf generate` |

### Sub-plan C: Learning Loops (Layer 2) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-15-subplan-c-learning-loops.md`](docs/superpowers/plans/2026-04-15-subplan-c-learning-loops.md)

| Deliverable | Notes |
|-------------|--------|
| `SessionEvent` types | Shared types for memory/learning (`harness/src/memory/types.ts`) |
| `SessionEventEmitter` | JSONL-per-run event log with cursor-based reads (`harness/src/memory/session.ts`) |
| Failure-to-rule pipeline | `deriveRuleFromFailure` analyzes failure chains, writes rules to `.forge/rules/` |
| Doc-gardening agent | `findStaleDocCandidates` + `blueprints/doc-gardening.yaml` blueprint |
| AgentService session wiring | Emits `prompt_composed`, `adapter_called`, `adapter_result`, `error` events |
| Tests | 13 new TS tests; 127 total harness tests |

### Sub-plan D: Quality + Permissions + Human (Layer 1+2+3) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-15-subplan-d-quality-permissions-human.md`](docs/superpowers/plans/2026-04-15-subplan-d-quality-permissions-human.md)

| Deliverable | Notes |
|-------------|--------|
| `NodeTypeHuman` + `ApprovalHandler` | Human/approval node with headless auto-deny, timeout, YAML `type: human` |
| `PermissionPipeline` | Two-phase: deterministic glob rules + async `ApprovalHandler` delegation |
| Credential isolation | Env var filtering in sandbox (`*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD`, `*_CREDENTIAL`) |
| Sprint contracts | `evaluateAgainstContract` with threshold-based pass/fail |
| Skeptical evaluator | `SkepticalEvaluator` class with SCORE/FEEDBACK parsing |
| Tests | 7 new Go tests (96 total blueprint); 5 new TS tests; 127 total harness tests |

### Sub-plan E: Container Warm Pools + Lazy Provisioning (Layer 3) — **COMPLETE**

**Plan:** [`docs/superpowers/plans/2026-04-15-subplan-e-warm-pools.md`](docs/superpowers/plans/2026-04-15-subplan-e-warm-pools.md)

| Deliverable | Notes |
|-------------|--------|
| `SandboxError` type | Structured container failure errors with `Unwrap()` support (`factory/sandbox/errors.go`) |
| `WarmPool` interface + `DockerWarmPool` | Pre-heated container pool with Acquire/Release/Shutdown; mutex-guarded idle pool (`factory/sandbox/pool.go`) |
| Lazy sandbox provisioning | `lazySandboxRunner` wrapper defers `EnsureImage` to first `Run()`; `WithLazySandbox` pipeline option |
| `DockerSandbox.SetWarmPool` | Acquire-first/cold-fallback pattern in `Run()`; warm container exec via `docker exec` |
| Daemon wiring | `--warm-pool-size`, `--warm-pool-image`, `--lazy-sandbox` flags; pool shutdown on SIGTERM |
| Container-as-cattle | Non-zero exit codes → `RunStatusFailed`, not pipeline crashes; verified for OOM/SIGKILL (exit 137) |
| Tests | 24 sandbox tests (94.3% coverage); 37 orchestrator tests (94.5% coverage); 127 harness tests; all pass with `-race` |

**v0.3 feature-complete.** All five sub-plans (A through E) delivered and merged. Next milestone: v0.3.1 (Agent Plugin System) or v1.0.

---

## v0.3 — Feature backlog (all delivered)

All v0.3 features have been implemented across Sub-plans A–E:

| Feature | Sub-plan | Status |
|---------|----------|--------|
| Memory / session capture | C | Done |
| Failure-to-rule pipeline | C | Done |
| Doc-gardening agent | C | Done |
| Multiple agent adapters (Goose, Codex, Cursor) | B | Done |
| Container warm pools | E | Done |
| Full quality gate system (sprint contracts) | D | Done |
| Prompt composition stack (5-level override) | B | Done |
| Permission pipeline (deterministic + async) | D | Done |
| Human/approval node in blueprint engine | D | Done |
| Shell output compression at tool boundary | B | Done |
| **Agent plugin system** | v0.3.1 | **Done** |

---

## v0.3.1 — Agent Plugin System

| Deliverable | Notes |
|-------------|-------|
| Plugin types + config | Zero-config defaults, `.forge/plugin.yaml` override |
| IDE detection | Cursor / Claude Code / Windsurf from env vars |
| Command registry | `forge_run`, `forge_fix`, `forge_plan` mapping to blueprints |
| ForgeExecutor | Direct `AgentService` execution, prompt composition |
| ForgePluginCore | Orchestrates IDE detection, config, command routing |
| MCP server | stdio transport, `@modelcontextprotocol/sdk`, 4 tools |
| CLI installer | `forge plugin install --ide auto\|cursor\|claude-code\|windsurf` |

---

## v1.0 — Production-ready (planned)

| Feature | Layer | Source |
|---------|-------|--------|
| Built-in skills (planning, coding, quality, context) | 2 | Master plan |
| Full documentation suite | — | Master plan |
| Community skill marketplace | 2 | OpenSpace prior art |
| Webhook + GitHub Issues + Slack/Discord triggers | 3 | Master plan, Archon, cc-connect |
| Human review queue + auto-merge policies | 3 | Master plan |
| Run tracing + token analytics dashboard | 3 | Master plan |
| Daemon mode (`forged`) production hardening | 3 | Master plan |
| Durable run store (Postgres) replacing in-memory registry | 3 | Multica |
| WebSocket/SSE live run progress streaming | 3 | Multica |
| Web UI for workflow management and monitoring | 3 | Archon, PentAGI |
| Pluggable observability backends (Grafana, Langfuse) | 3 | PentAGI |
| Repository code intelligence graph (AST + deps) | 2 | graphify, code-review-graph |
| Self-evolving skills from usage telemetry | 2 | OpenSpace |
| MCP-native skill packaging and distribution | 2 | OpenSpace |
| Portable agent project manifest (import/export) | All | gitagent |
| Outer-loop harness optimization (versioned candidates) | 2 | metaharness |
| RL / prompt-policy optimization (experimental) | 2 | Agent Lightning |
| Plugin marketplace — community-contributed plugins, versioned distribution, one-command install across IDEs | All | obra/superpowers, OpenSpace |

---

## Plan file index

| Plan | ID | Path | Layer | Status |
|------|----|------|-------|--------|
| Forge Agent Factory (master) | `forge_agent_factory_86e877d4` | `.cursor/plans/forge_agent_factory_86e877d4.plan.md` | All | Reference doc |
| Layer 1: Blueprint Engine | `layer_1_blueprint_engine_4bd3f740` | `.cursor/plans/layer_1_blueprint_engine_4bd3f740.plan.md` | 1 | **Complete** |
| Layer 2: Harness MVP | `layer_2_harness_mvp_07ee3081` | `.cursor/plans/layer_2_harness_mvp_07ee3081.plan.md` | 2 | **Complete** |
| Layer 3: Factory MVP | `layer_3_factory_mvp_f6c28aa0` | `.cursor/plans/layer_3_factory_mvp_f6c28aa0.plan.md` | 3 | **Complete** |
| Layer 4: Integration + Polish | `2026-04-12-layer-4-integration-polish` | `docs/superpowers/plans/2026-04-12-layer-4-integration-polish.md` | 1–3 | **Complete** |
| v0.2 Design Spec | `2026-04-13-v02-skills-tools-triggers-design` | `docs/superpowers/specs/2026-04-13-v02-skills-tools-triggers-design.md` | All | Reference doc |
| v0.2 Sub-plan A: Skills + EvalNode | `2026-04-13-subplan-a-skills-evalnode` | `docs/superpowers/plans/2026-04-13-subplan-a-skills-evalnode.md` | 1–2 | **Complete** |
| v0.2 Sub-plan B: Tool Pool + Context | `2026-04-13-subplan-b-toolpool-context` | `docs/superpowers/plans/2026-04-13-subplan-b-toolpool-context.md` | 1–2 | **Complete** |
| v0.2 Sub-plan C: Triggers + Parallel | `2026-04-13-subplan-c-triggers-parallel` | `docs/superpowers/plans/2026-04-13-subplan-c-triggers-parallel.md` | 3 | **Complete** |
| v0.3 Design Spec | `2026-04-15-v03-learning-multiadapter-design` | `docs/superpowers/specs/2026-04-15-v03-learning-multiadapter-design.md` | All | Reference doc |
| v0.3 Sub-plan A: Debt + Factory Hardening | `2026-04-15-subplan-a-v02-debt` | `docs/superpowers/plans/2026-04-15-subplan-a-v02-debt.md` | 3 | **Complete** |
| v0.3 Sub-plan B: Multi-Adapter + Prompt | `2026-04-15-subplan-b-multiadapter-prompt` | `docs/superpowers/plans/2026-04-15-subplan-b-multiadapter-prompt.md` | 2 | **Complete** |
| v0.3 Sub-plan C: Learning Loops | `2026-04-15-subplan-c-learning-loops` | `docs/superpowers/plans/2026-04-15-subplan-c-learning-loops.md` | 2 | **Complete** |
| v0.3 Sub-plan D: Quality + Permissions + Human | `2026-04-15-subplan-d-quality-permissions-human` | `docs/superpowers/plans/2026-04-15-subplan-d-quality-permissions-human.md` | 1–3 | **Complete** |
| v0.3 Sub-plan E: Warm Pools | `2026-04-15-subplan-e-warm-pools` | `docs/superpowers/plans/2026-04-15-subplan-e-warm-pools.md` | 3 | **Complete** |

v0.1 layer plans: `.cursor/plans/*.plan.md`
v0.1 Layer 4 + v0.2 implementation plans: `docs/superpowers/plans/*.md`

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
2. For **v0.3 planning**, review the v0.3 feature table above and create a design spec + sub-plans.
3. For historical **v0.1 layer plans**, use `.cursor/plans/<id>.plan.md` from the index table.
4. For **v0.2 plans**, see `docs/superpowers/plans/2026-04-13-subplan-*.md`.
5. For **v0.3 plans**, see `docs/superpowers/specs/2026-04-15-v03-learning-multiadapter-design.md` and `docs/superpowers/plans/2026-04-15-subplan-*.md`.
6. Reference `project.md` for module map and `docs/design.md` for architecture.

**Current checkpoint:** v0.1 MVP complete. **v0.2 complete.** **v0.3 complete** (all five sub-plans A–E merged to `main`): Sub-plan A — SessionLog, queue shutdown, repo resolver, pipeline wiring; Sub-plan B — multi-adapter harness, prompt stack, compression, proto `adapter` field; Sub-plan C — learning loops (session capture, failure-to-rule, doc-gardening); Sub-plan D — HumanNode, permission pipeline, credential isolation, quality gates; Sub-plan E — SandboxError, WarmPool, lazy provisioning, daemon wiring, container-as-cattle. 127 harness tests, 96+ Go tests.
**Next action:** Begin **v1.0** planning. v0.3.1 (Agent Plugin System) is complete — MCP server, IDE detection, command registry, CLI installer all shipped.
