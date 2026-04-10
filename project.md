# Forge Project

## Project overview

### Inputs

- A **task intent** (natural language or structured prompt) and/or a ticket/CI event
  (future triggers).
- A **repository** with build commands (e.g. `make lint`, `make test`) and project
  docs (`AGENTS.md`, optional `.forge/rules/*.md`).
- **Blueprint YAML** defining the workflow graph (built-in or custom).
- **Trusted configuration** for agents (API keys, MCP configs) supplied by the user;
  Forge must not hardcode secrets.

### Outputs

- **Run records**: per-node results, logs, success/failure, iteration counts.
- **Code changes** in a sandbox or working tree; **pull requests** when delivery
  is wired (post-MVP / stretch).
- **Durable artifacts**: updated docs/rules when memory and failure-to-rule features
  land (v0.3 direction).

### Constraints

- **MVP is local-first**: primary execution model uses **local Docker/Podman**, not a
  hosted Forge Cloud (see [docs/prd-forge.md](docs/prd-forge.md)).
- **Shell execution**: deterministic nodes run commands from YAML—load blueprints
  only from trusted paths.
- **Stack**: Go (Layers 1 & 3), TypeScript (Layer 2), YAML (blueprints), gRPC
  (engine ↔ harness), MCP (tools, later phases).

### Documentation index

| Document | Purpose |
|----------|---------|
| [roadmap.md](roadmap.md) | **Start here** — version map, plan status, checkpoint resume |
| [docs/design.md](docs/design.md) | Canonical architecture and ADRs |
| [docs/prd-forge.md](docs/prd-forge.md) | Product requirements, MVP scope, user stories |
| [references/references.md](references/references.md) | External links and research notes |
| [AGENTS.md](AGENTS.md) | Contributor quick reference |

### Repository map (structure reference)

Use **`project.md` + [docs/design.md](docs/design.md)** as the base map for where
everything lives and how layers connect. This section is the quick lookup table;
depth is in `docs/design.md`.

| What you need | Where it lives |
|---------------|----------------|
| Full system design, target schemas, integration picture | [docs/design.md](docs/design.md) |
| MVP scope, user stories, functional requirements | [docs/prd-forge.md](docs/prd-forge.md) |
| External reading list (Stripe, Anthropic, ECC, …) | [references/references.md](references/references.md) |
| Module dependencies, status, stories index, snapshots | This file (`project.md`) |
| Day-to-day contributor rules | [AGENTS.md](AGENTS.md) |
| **Layer 1** — blueprint engine (Go) | `core/blueprint/` |
| **Layer 1** — built-in YAML + embed | `blueprints/` (`embed.go`) |
| **CLI** | `cmd/forge/` |
| **Layer 2** — harness, adapters, gRPC server | `harness/` (MVP: echo + Claude adapters, context loader, gRPC server) |
| **gRPC** — contract + Go client | `proto/forge/v1/`, `internal/grpcexec/` |
| **Layer 3** (planned) — sandbox, triggers, orchestration, delivery | `factory/` — overview: [docs/design.md](docs/design.md) §6 |
| **Daemon** (planned) | `cmd/forged/` |
| **Built-in skills** (planned) | `skills/` |
| **Cross-package tests** (planned) | `tests/` (repo-level) |
| Local Cursor plans (not shipped in binary) | `.cursor/plans/*.plan.md` |

---

## Module dependency diagram

```text
                    +---------------------------+
                    |      cmd/forge (CLI)      |
                    |  blueprint validate|list |
                    |  |run [--harness addr]    |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    | core/blueprint            |  Layer 1: Forge Core
                    | Graph, Nodes, Engine,     |
                    | YAML, RunState,           |
                    | AgentExecutor seam        |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    | internal/grpcexec          |  gRPC bridge
                    | GrpcAgentExecutor          |
                    +-------------+--------------+
                                  |  gRPC
                    +-------------v--------------+
                    | harness/ (TypeScript)       |  Layer 2: Forge Harness
                    | gRPC server, adapters,      |  (MVP complete)
                    | context loader              |
                    +-----------------------------+
                                  ^
                                  | optional calls
                    +-------------+-------------+
                    |   factory/ (Go)           |  Layer 3: Forge Factory
                    |  sandbox, triggers,       |  (planned; not yet in repo)
                    |  orchestrator, delivery   |
                    +-------------+-------------+
                                  |
                    +-------------v-------------+
                    | Docker/Podman, git remote,  |
                    | Slack/webhook (future)      |
                    +---------------------------+
```

---

## Module dependency table

| Module | Layer | Language | Depends on | Status |
|--------|-------|----------|------------|--------|
| `core/blueprint` | 1 | Go | (none) | **Complete** (engine, graph, YAML, tests) |
| `blueprints/` | 1 | YAML + `embed.go` | `core/blueprint` | **Complete** (built-ins embedded) |
| `cmd/forge` | 1 | Go | `core/blueprint`, `blueprints`, optional `internal/grpcexec` | **In progress** (CLI: validate, list, run w/ mock or gRPC harness) |
| `proto/`, `internal/grpcexec/` | 1–2 | Go | `core/blueprint` | **Complete** (ForgeAgent contract, `GrpcAgentExecutor`) |
| `harness/` | 2 | TypeScript | `proto` (contract) | **Complete** (MVP: gRPC server, echo + Claude Code adapters, context loader) |
| `factory/sandbox` | 3 | Go | `cmd/forge` or lib API | **Planned** |
| `factory/triggers` | 3 | Go | `factory/orchestrator` | **Planned** |
| `factory/orchestrator` | 3 | Go | `core/blueprint`, sandbox | **Planned** |
| `factory/delivery` | 3 | Go | git provider APIs | **Planned** |
| `cmd/forged` | 3 | Go | factory packages | **Planned** (daemon) |

---

## Module definitions and responsibilities

### Layer 1 — `core/blueprint`

- Parse and validate blueprint YAML into a **directed graph**.
- Execute **agentic** (via `AgentExecutor`), **deterministic**, and **gate** nodes.
- Maintain **`RunState`**, enforce **max iterations** on gate loops.
- Expose **`AgentExecutor`** for harness integration without importing TS.

### Layer 1 — `blueprints/`

- Ship **versioned default workflows** (`standard-implementation`, `bug-fix`, …).
- Embed FS for CLI distribution (`embed.go`).

### Layer 1 — `cmd/forge`

- Operator interface: `forge blueprint validate|list|run`.
- Eventually: `forge run`, harness address flags, JSON output mode (per PRD).

### Layer 2 — `harness/` (MVP complete)

- gRPC **ForgeAgent** service: context load → adapter → response (implemented).
- Adapters: **Echo**, **Claude Code** (headless); later Goose/Codex.
- Subsystems over time: context budget (partial), quality/eval, skills, memory, toolshed.

### Layer 3 — `factory/` (planned)

- **Sandbox**: container lifecycle, mounts, network policy, git clone/push.
- **Triggers**: CLI (done in Layer 1 first), then Slack/webhook/GitHub.
- **Orchestrator**: parallel runs, quotas, cancellation, cost/time tracking.
- **Delivery**: PRs, CI polling, human review queue policies.

---

## Stories (algorithm documentation placeholders)

*Per AGENTS.md workflow: each module’s Story belongs at the top of its code files;
this section is the project-level index.*

### `core/blueprint` — Story (summary)

**Input:** Blueprint YAML + `RunState` + `AgentExecutor`.  
**Path:** Validate graph → walk from `start` → for each node execute by type → on
gate, read prior `NodeResult` and follow `pass`/`fail` edge → stop on terminal
node or max iterations.  
**Output:** Final `RunState` with per-node results and overall status.

### `cmd/forge` — Story (summary)

**Input:** CLI args pointing to blueprint file or built-in list request.  
**Path:** Parse subcommand → read YAML if needed → validate or build graph → run
with mock executor for `run`.  
**Output:** Human-readable result or error to stderr; exit code non-zero on failure.

### `harness/` — Story (summary)

**Input:** gRPC request with prompt, working directory metadata, config JSON.  
**Path:** Load scoped context (`AGENTS.md` / rules) → compose prompt → select adapter
(Echo or Claude Code) → return consolidated agent output → structured success/failure.  
**Output:** gRPC response consumed by Go `GrpcAgentExecutor`.

### `factory/sandbox` — Story (placeholder)

**Input:** Run id, repo URL/ref, env secrets policy.  
**Path:** Start container → clone → run forge/harness inside → collect artifacts.  
**Output:** Mounted workspace changes and logs.

### `factory/orchestrator` — Story (placeholder)

**Input:** Trigger event (CLI, webhook, schedule).  
**Path:** Schedule run → allocate sandbox → supervise lifecycle → aggregate status.  
**Output:** Run handle for observability; optional PR URL.

### `factory/delivery` — Story (placeholder)

**Input:** Successful run with git commits on a branch.  
**Path:** Push branch → open PR from template → apply labels/reviewers.  
**Output:** PR link or actionable error.

---

## Implementation order

1. **`core/blueprint`** — types → graph → deterministic node → gate → agentic +
   mock tests → engine → YAML → tests (**done**).
2. **`blueprints/`** + **`embed.go`** — built-in YAML (**done**).
3. **`cmd/forge`** — validate, list, run (**in progress**; extend per PRD).
4. **`proto/` + buf + Go codegen** — agent service contract (**done**).
5. **`harness/` TS** — server, context loader, adapters, tests (**MVP done**).
6. **`internal/grpcexec/`** — `AgentExecutor` over gRPC (**done**).
7. **`factory/sandbox`** — Docker MVP, documented policy defaults.
8. **`factory/orchestrator` + `factory/triggers` + `factory/delivery`** — parallel
   runs, Slack/webhook, PR creation.
9. **`cmd/forged`** — optional daemon for triggers and worker pools.
10. **Integration tests** — end-to-end: CLI → harness → sample repo.

---

## File structure (current vs planned)

### Current (repository snapshot)

```text
forge/
  AGENTS.md
  project.md
  go.mod
  go.sum
  Makefile
  cmd/forge/
    main.go
    main_test.go
  core/blueprint/
    types.go
    graph.go
    node.go
    engine.go
    yaml.go
    *_test.go
  blueprints/
    standard-implementation.yaml
    bug-fix.yaml
    embed.go
  docs/
    design.md
    prd-forge.md
    superpowers/plans/        (empty placeholder dir)
  references/
    references.md
  .cursor/plans/               (local planning docs; not runtime)
```

### Target tree (full factory — from design)

Aligned with [docs/design.md](docs/design.md) §14. Paths without files today are
**planned**.

```text
forge/
├── cmd/
│   ├── forge/main.go              # CLI (today: blueprint validate|list|run)
│   └── forged/main.go             # Factory daemon (planned)
├── core/
│   ├── blueprint/                 # Layer 1 engine (implemented)
│   │   ├── engine.go, graph.go, node.go, yaml.go, types.go
│   │   └── compose.go             # Blueprint composition (planned)
│   └── types/                     # Shared Go types (planned if split)
├── harness/                       # Layer 2 — TypeScript (planned)
│   ├── adapters/                  # claude-code, goose, codex, direct-llm, cursor
│   ├── context/
│   ├── quality/
│   ├── skills/
│   ├── memory/
│   └── toolshed/
├── factory/                       # Layer 3 — Go (planned)
│   ├── sandbox/
│   ├── triggers/
│   ├── orchestrator/
│   └── delivery/
├── proto/forge/v1/                # gRPC contract (planned)
├── internal/grpcexec/             # Go gRPC AgentExecutor (planned)
├── blueprints/                    # Built-in YAML (+ embed.go)
├── skills/                        # Built-in SKILL bundles (planned)
├── docs/
│   ├── design.md
│   ├── prd-forge.md
│   └── specs/                     # deeper specs (optional)
├── references/
│   └── references.md
├── tests/                         # Repo-level integration tests (planned)
├── project.md
└── AGENTS.md
```

### Planned additions (short list)

```text
  proto/forge/v1/agent.proto
  internal/grpcexec/
  harness/
  factory/{sandbox,triggers,orchestrator,delivery}
  cmd/forged/
  core/blueprint/compose.go
  skills/
  tests/
```

---

## Implementation status

| Area | Status | Notes |
|------|--------|-------|
| Blueprint engine | **Done** | Agentic, deterministic, gate nodes; iteration guard |
| Built-in blueprints | **Done** | `standard-implementation`, `bug-fix` |
| CLI | **In progress** | `run` supports mock executor or `--harness` gRPC address |
| Harness (TS) | **Done** | MVP: echo + Claude adapters, context loader, gRPC server |
| Factory (Go) | **Not started** | See `docs/design.md` Layer 3 |
| Docs suite | **Done** | `docs/design.md`, `docs/prd-forge.md`, `references/references.md` |

---

## Maintenance (AGENTS.md checklist)

- Update this file when **modules** are added or renamed.
- Update **Stories** in code files first; mirror summaries here if the module is new.
- Keep **implementation status** honest after each milestone.
- Record **design decisions** in [docs/design.md](docs/design.md) ADR table.
