# Forge

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Tests](https://img.shields.io/badge/tests-379_passing-brightgreen)]()
[![Coverage](https://img.shields.io/badge/coverage-94%25_Go_|_100%25_TS-brightgreen)]()

**Turn any AI coding agent into an autonomous factory that ships mergeable pull requests.**

Forge is a three-layer open-source system that wraps interactive AI assistants -- Claude Code, Goose, Codex, Cursor, or any agent you build -- in an industrial harness with typed workflow graphs, quality gates, Docker isolation, learning from failures, and automated PR delivery. Give it a task; get back a tested, linted pull request.

---

## Why Forge

- **Agent-agnostic.** Forge doesn't replace your agent -- it orchestrates it. Swap Claude Code for Goose, Codex, or Cursor by changing one adapter flag. Each adapter declares its capabilities (streaming, max tokens, tool support), and Forge adapts automatically.

- **Deterministic quality gates.** Every AI-generated change passes through lint and test gates enforced by a typed state machine -- not by hoping the agent remembers to run tests. Gates loop back to the agent on failure with bounded retries. Sprint contracts and a skeptical evaluator system provide configurable acceptance criteria.

- **Self-improving.** Failed runs automatically derive prevention rules written to `.forge/rules/`. A doc-gardening agent detects stale documentation. Session event logs enable cross-run learning. The system gets smarter with every failure.

- **Factory-scale isolation.** Each run gets its own git worktree and Docker container with configurable resource limits, network policy, and credential isolation. Warm container pools and lazy provisioning minimize startup latency. Run dozens of tasks in parallel without cross-contamination.

- **Declarative workflows.** Define agent pipelines in YAML -- plan, implement, lint, test, human review, commit -- as directed graphs. Human/approval nodes block for input in interactive mode and auto-deny in CI. No imperative glue code. Compose, version, and share blueprints like infrastructure-as-code.

---

## Architecture

```text
+------------------------------------------------------------------+
|                        Layer 3: Factory (Go)                     |
|  Docker sandbox | warm container pools | git worktrees           |
|  PR delivery | run queue | webhook triggers | forged daemon      |
|  session event log | lazy provisioning | task assignment          |
+----------------------------------+-------------------------------+
                                   |
                                   v
+----------------------------------+-------------------------------+
|                   Layer 2: Intelligence Harness (TypeScript)      |
|  gRPC agent service | multi-adapter (Claude, Goose, Codex, ...)  |
|  prompt composition stack | skill system | tool pool              |
|  learning loops | quality gates | shell output compression       |
+----------------------------------+-------------------------------+
                                   |
                                   v
+----------------------------------+-------------------------------+
|                   Layer 1: Blueprint Engine (Go)                 |
|  YAML graph parser | typed state machine | agentic nodes         |
|  deterministic nodes | gate nodes | eval nodes | human nodes     |
|  permission pipeline | iteration guard                           |
+------------------------------------------------------------------+
```

Each layer is independently useful:

| Layer | Standalone use case |
|-------|---------------------|
| **1 -- Blueprint Engine** | CI/CD pipeline orchestration, workflow automation, any typed DAG execution |
| **2 -- Harness** | Standalone gRPC service wrapping any agent with context engineering and skills |
| **3 -- Factory** | Sandboxed code generation at scale with PR delivery, even with a custom engine |

---

## Why Forge over alternatives

| Capability | Forge | Archon | ChatDev | Multica |
|------------|-------|--------|---------|---------|
| Architecture | Three independent layers | Monolithic Python | Single multi-agent process | Managed platform |
| Engine language | Compiled Go -- single binary, low overhead | Python runtime | Python runtime | Proprietary |
| Agent support | Any agent via adapter interface | Coupled to specific LLMs | Fixed role set | Platform-specific |
| Sandboxing | Docker + network policy + resource limits | None | None (in-process) | Cloud-managed |
| Quality gates | First-class gate nodes with bounded retries | Manual | No enforcement | Imperative checks |
| Workflow definition | Declarative YAML blueprints | Python code | Role config | Imperative routing |
| Delivery | Automated PR creation via `gh` | Manual | Directory output | Platform-specific |
| Isolation | Git worktree per run | Shared state | Shared state | Cloud-managed |
| Deployment | Self-hosted, no cloud dependency | Self-hosted | Self-hosted | SaaS only |
| License | Apache 2.0 | Open source | Open source | Proprietary |

### What three-layer independence gives you

**Progressive adoption.** Start with `forge blueprint run` locally to validate workflows. Graduate to `forge run` with Docker sandboxing. Scale to daemon mode (`forged`) with webhook triggers and bounded-concurrency run queues.

**Separation of concerns.** Engine reliability evolves independently from AI intelligence, which evolves independently from infrastructure. A bug in your agent adapter never touches the state machine. A change to sandboxing never touches prompt composition.

**Mix and match.** Use Layer 1 alone as a general-purpose workflow engine. Use Layer 2 as a standalone harness service behind your own orchestrator. Add Layer 3 only when you need factory-scale isolation and delivery.

### Design lineage

Forge synthesizes proven patterns from production agent systems:

- **Stripe Minions** -- blueprint-as-state-machine, curated tool subsets, bounded CI retries
- **Anthropic harness design** -- planner/generator/evaluator separation, sprint contracts, context resets
- **Anthropic Managed Agents** -- brain-hands-session decoupling, containers as cattle, lazy provisioning, durable session event logs, credential isolation
- **OpenAI harness engineering** -- AGENTS.md as table of contents, layered architecture, linter output as remediation signal

---

## Features

### Layer 1: Blueprint Engine

- Typed DAG engine with five node types: **agentic** (LLM agent), **deterministic** (shell commands), **gate** (conditional pass/fail routing), **eval** (evaluation/grading), **human** (approval/review gates)
- Two-phase YAML parser with `depends_on` vocabulary for declaring node dependencies
- Gate nodes enforce bounded iteration limits -- no infinite retry loops
- **Human/approval nodes** with configurable timeout, headless auto-deny for CI, and pluggable `ApprovalHandler` (CLI, webhook, Slack)
- **Permission pipeline**: two-phase (deterministic glob rules + async human escalation) with credential isolation
- `AgentExecutor` interface decouples the engine from any specific agent implementation
- Built-in blueprints: `standard-implementation` (plan -> implement -> lint -> test -> commit) and `bug-fix` (reproduce -> fix -> test -> commit)
- Concurrent node execution for independent graph branches

### Layer 2: Intelligence Harness

- gRPC `ForgeAgent` service consumed by the Go engine via `GrpcAgentExecutor`
- **Multi-adapter system**: capability-aware streaming protocol with 5 adapters -- **Echo** (testing), **Claude Code**, **Goose**, **Codex**, **Cursor** -- each declaring capabilities (streaming, interrupt, max tokens, tool support)
- **Prompt composition stack**: 5-level priority system (override > coordinator > agent-specific > project rules > default baseline) with budget-aware truncation based on adapter capabilities
- **Shell output compression**: keeps error/warning lines, headers, summaries; collapses verbose middle sections
- Context loader: reads `AGENTS.md` and `.forge/rules/` to compose scoped prompts
- **Learning loops**: durable session event capture, failure-to-rule pipeline (failed runs automatically derive prevention rules written to `.forge/rules/`), doc-gardening agent for stale documentation detection
- **Quality gate system**: sprint contract negotiation, skeptical evaluator with configurable criteria, retry + human escalation
- Skill system: filesystem-scanned registry, keyword-based resolver, lifecycle management (evaluate, promote, compare)
- Tool pool: assembly with deferred loading against a context budget, pre/post execution hooks
- Subagent context isolation for parallel agent invocations

### Layer 3: Factory

- Docker sandbox with volume mounts, environment injection, resource limits, network policy, and **credential isolation** (env vars matching `*_KEY`, `*_SECRET`, `*_TOKEN`, `*_PASSWORD` are filtered unless allow-listed)
- **Container warm pools**: pre-heated Docker containers for fast acquisition (target: under 10s); `Acquire`/`Release`/`Shutdown` lifecycle with workspace reset on return; configurable pool size via `--warm-pool-size`
- **Lazy sandbox provisioning**: defers `EnsureImage` until the first sandbox-bound node runs, removing container setup from the critical path (follows Anthropic's ~60% p50 TTFT improvement pattern)
- **Container-as-cattle error handling**: container failures (OOM, crash, timeout) are `NodeResult{Failed}`, not pipeline crashes; failed containers are discarded, not debugged
- Git worktree manager for branch-isolated parallel runs
- PR delivery pipeline: `git push` + `gh pr create`
- Run queue with bounded concurrency (configurable `maxParallel` workers) and **graceful shutdown** (`Shutdown` drains in-flight work, stops warm pool containers)
- In-memory run registry with concurrent-safe state tracking
- Rule-based task assigner wired into the pipeline (default adapter when none specified)
- Durable **session event log** (`SessionLog` / file JSONL) for run lifecycle events, enabling learning loops and restart recovery
- Webhook HTTP trigger: `POST /api/v1/runs` to enqueue, `GET /api/v1/runs/:id` to poll; optional **`repo_url`** resolved to a cached bare clone via `GitRepoResolver`
- `forged` daemon: production pipeline wiring (Docker + worktree + delivery), or **`--dry-run`** for log-only runs; flags for warm pools, lazy sandbox, session log dir, and repo cache; signal handling with queue + pool + HTTP shutdown

---

## Quick Start

### Prerequisites

| Tool | Required | Notes |
|------|----------|-------|
| Go 1.22+ | Yes | Engine and factory |
| Node.js 22+ | Yes | Harness |
| Docker | Yes | Sandbox isolation |
| `gh` CLI | Optional | PR delivery |
| Claude Code CLI | Optional | Real agent runs |

### Build and verify

```bash
git clone https://github.com/adityasoni99/forge.git
cd forge

go test ./...
cd harness && npm ci && npm test && cd ..
make docker-build
```

### Explore blueprints

```bash
go run ./cmd/forge blueprint list
go run ./cmd/forge blueprint validate ./blueprints/standard-implementation.yaml
```

---

## Usage

### Local dry run (no Docker, no agent)

Uses the echo adapter for fast iteration on workflows:

```bash
go run ./cmd/forge run --no-sandbox "add input validation to the API handler"
```

With a specific blueprint:

```bash
go run ./cmd/forge run --no-sandbox --blueprint bug-fix "fix the nil pointer in ParseConfig"
```

### Docker-sandboxed run

Runs inside an isolated container with the echo adapter:

```bash
go run ./cmd/forge run --adapter echo --no-pr "add retry logic to the HTTP client"
```

### Full agent run

Full autonomous pipeline with a real agent (Claude Code, Goose, Codex, or Cursor):

```bash
go run ./cmd/forge run --adapter claude --no-pr "implement the caching layer from the design doc"
```

Or run the harness separately for development:

```bash
# Terminal 1: start the harness
cd harness
FORGE_ADAPTER=claude FORGE_HARNESS_PORT=50051 npm start
```

```bash
# Terminal 2: run the blueprint against the harness
go run ./cmd/forge run --no-sandbox --harness 127.0.0.1:50051 --adapter claude "implement the caching layer"
```

### Multi-adapter selection

Forge supports multiple AI agents. Choose one per run:

```bash
go run ./cmd/forge run --adapter claude "implement the caching layer"
go run ./cmd/forge run --adapter goose "fix the nil pointer in ParseConfig"
go run ./cmd/forge run --adapter codex "add retry logic to the HTTP client"
```

The `TaskAssigner` automatically selects an adapter when none is specified. Configure via `FORGE_ADAPTER` env var or blueprint config.

### Daemon mode (webhook-triggered)

Run `forged` as a long-lived service accepting tasks via HTTP. By default it runs the **full factory pipeline** (Docker sandbox, git worktree, optional PR). Use **`--dry-run`** to log tasks without Docker/git (useful for smoke tests):

```bash
go run ./cmd/forged -port 8080 -max-parallel 4
```

```bash
# Log-only pipeline (no Docker/git)
go run ./cmd/forged --dry-run -port 8080
```

#### Warm pools and lazy provisioning

Pre-heat Docker containers for faster run startup. Lazy provisioning defers container setup until the first sandbox-bound node (enabled by default):

```bash
# Pre-heat 3 warm containers
go run ./cmd/forged -port 8080 -warm-pool-size 3 -warm-pool-image forge:latest

# Disable lazy provisioning (eager sandbox setup)
go run ./cmd/forged -port 8080 -lazy-sandbox=false
```

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `-port` | `FORGED_PORT` | `8080` | HTTP listen port |
| `-max-parallel` | `FORGED_MAX_PARALLEL` | `2` | Max concurrent runs |
| `-warm-pool-size` | `FORGED_WARM_POOL_SIZE` | `0` (disabled) | Pre-heated container count |
| `-warm-pool-image` | `FORGED_WARM_POOL_IMAGE` | `forge:latest` | Image for warm containers |
| `-lazy-sandbox` | — | `true` | Defer sandbox setup until first use |
| `-sessions-dir` | `FORGED_SESSIONS_DIR` | `.forge/sessions` | Session JSONL directory |
| `-repo-cache-dir` | `FORGED_REPO_CACHE` | `.forge/repo-cache` | Bare clone cache |
| `-dry-run` | — | `false` | Log-only pipeline (no Docker/git) |

```bash
# Enqueue a run
curl -X POST http://localhost:8080/api/v1/runs \
  -H "Content-Type: application/json" \
  -d '{"task": "add unit tests for the parser module", "adapter": "claude"}'

# Poll status
curl http://localhost:8080/api/v1/runs/<run-id>
```

### IDE Plugin (MCP)

Install the Forge plugin for your IDE:

```bash
# Auto-detect IDE
forge plugin install

# Or specify explicitly
forge plugin install --ide cursor
forge plugin install --ide claude-code
forge plugin install --ide windsurf
```

After installation, restart your IDE. The following MCP tools become available:

- **forge_run** — Execute a task end-to-end (plan, implement, test, commit)
- **forge_fix** — Reproduce and fix a bug
- **forge_plan** — Create an implementation plan without executing
- **forge_status** — Check plugin configuration and status

---

## Blueprint anatomy

Blueprints are declarative YAML workflow graphs. Here is the built-in `standard-implementation` blueprint:

```yaml
name: standard-implementation
version: "0.1"
start: plan
nodes:
  plan:
    type: agentic
    config:
      prompt: "Analyze the task and create a detailed implementation plan for: {{task}}"
  implement:
    type: agentic
    config:
      prompt: "Implement the requested task: {{task}}. Write clean, tested code."
  lint:
    type: deterministic
    config:
      command: "make lint"
  lint_gate:
    type: gate
    config:
      check_node: lint
  test:
    type: deterministic
    config:
      command: "make test"
  test_gate:
    type: gate
    config:
      check_node: test
  commit:
    type: deterministic
    config:
      command: "git add -A && git commit -m 'feat: implement task'"
edges:
  - { from: plan, to: implement }
  - { from: implement, to: lint }
  - { from: lint, to: lint_gate }
  - { from: lint_gate, to: test, condition: pass }
  - { from: lint_gate, to: implement, condition: fail }  # retry on lint failure
  - { from: test, to: test_gate }
  - { from: test_gate, to: commit, condition: pass }
  - { from: test_gate, to: implement, condition: fail }  # retry on test failure
```

Gate nodes route `pass` to the next stage and `fail` back to the agent for correction, with bounded iteration limits preventing infinite loops.

### Human approval nodes

Blueprints can include human review gates that block until approved:

```yaml
nodes:
  approve_pr:
    type: human
    config:
      prompt: "Review and approve the generated PR"
      timeout: 3600
edges:
  - { from: test_gate, to: approve_pr, condition: pass }
  - { from: approve_pr, to: commit, condition: pass }
  - { from: approve_pr, to: implement, condition: fail }
```

In interactive mode, the handler blocks for human input. In headless/CI mode (`forged`), human nodes auto-deny to prevent blocking.

---

## Project Status

| Version | Theme | Status | Highlights |
|---------|-------|--------|------------|
| **v0.1** | Engine + Harness + Factory MVP | **Shipped** | Typed graph engine, YAML parser, gRPC harness, echo + Claude adapters, Docker sandbox, git worktrees, PR delivery, CI/CD |
| **v0.2** | Skills + Tool Pool + Triggers | **Shipped** | EvalNode, skill system, tool pool with deferred loading, webhook triggers, run queue, `forged` daemon |
| **v0.3** | Learning + Multi-adapter | **Shipped** | Multi-adapter (Claude, Goose, Codex, Cursor), prompt composition stack, learning loops (session capture, failure-to-rule, doc gardening), human/approval nodes, permission pipeline, credential isolation, quality gates (sprint contracts, skeptical evaluator), warm container pools, lazy provisioning, container-as-cattle error handling. 96+ Go tests (94% coverage), 127 TS tests |
| **v0.3.1** | Agent Plugin System | **Shipped** | MCP server (stdio, `@modelcontextprotocol/sdk`), IDE detection (Cursor / Claude Code / Windsurf), command registry (`forge_run`, `forge_fix`, `forge_plan`, `forge_status`), CLI installer (`forge plugin install`), template injection protection, single-source tool defs. 211 Go + 168 TS tests |
| **v1.0** | Production-ready | Planned | Durable run store (Postgres), WebSocket streaming, web UI, skill marketplace, GitHub Issues triggers, human review queue, run tracing dashboard |

See [roadmap.md](roadmap.md) for the full version map and implementation details.

---

## Documentation

| Document | Purpose |
|----------|---------|
| [docs/design.md](docs/design.md) | Architecture reference and ADRs |
| [docs/prd-forge.md](docs/prd-forge.md) | Product requirements and user stories |
| [project.md](project.md) | Module map, dependency diagram, implementation status |
| [roadmap.md](roadmap.md) | Version plan, checkpoints, plan file index |
| [AGENTS.md](AGENTS.md) | Contributor quick reference |

---

## Contributing

Forge follows TDD and the workflow described in [AGENTS.md](AGENTS.md). To contribute:

1. Fork the repo and create a feature branch
2. Write failing tests first, then implement
3. Ensure `go test ./...` and `cd harness && npm test` pass
4. Keep files under 300 lines where possible
5. Open a PR with a clear description of the change

---

## License

[Apache 2.0](https://opensource.org/licenses/Apache-2.0)
