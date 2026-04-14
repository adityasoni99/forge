# Forge

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5.x-3178C6?logo=typescript&logoColor=white)](https://www.typescriptlang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Tests](https://img.shields.io/badge/tests-160%2B_passing-brightgreen)]()
[![Coverage](https://img.shields.io/badge/coverage-91%25_Go_|_100%25_TS-brightgreen)]()

**Turn any AI coding agent into an autonomous factory that ships mergeable pull requests.**

Forge is a three-layer open-source system that wraps interactive AI assistants -- Claude Code, Goose, Codex, or any agent you build -- in an industrial harness with typed workflow graphs, quality gates, Docker isolation, and automated PR delivery. Give it a task; get back a tested, linted pull request.

---

## Why Forge

- **Agent-agnostic.** Forge doesn't replace your agent -- it orchestrates it. Swap Claude Code for Goose or Codex by changing one adapter flag. The `AgentExecutor` interface means any agent that can take a prompt and return output is a first-class citizen.

- **Deterministic quality gates.** Every AI-generated change passes through lint and test gates enforced by a typed state machine -- not by hoping the agent remembers to run tests. Gates loop back to the agent on failure with bounded retries.

- **Factory-scale isolation.** Each run gets its own git worktree and Docker container with configurable resource limits and network policy. Run dozens of tasks in parallel without cross-contamination.

- **Declarative workflows.** Define agent pipelines in YAML -- plan, implement, lint, test, commit -- as directed graphs. No imperative glue code. Compose, version, and share blueprints like infrastructure-as-code.

---

## Architecture

```text
+------------------------------------------------------------------+
|                        Layer 3: Factory (Go)                     |
|  Docker sandbox | git worktrees | PR delivery | run queue        |
|  webhook triggers | forged daemon | task assignment              |
+----------------------------------+-------------------------------+
                                   |
                                   v
+----------------------------------+-------------------------------+
|                   Layer 2: Intelligence Harness (TypeScript)      |
|  gRPC agent service | adapters (Claude, Echo, ...) | context     |
|  skill system | tool pool | quality gates                        |
+----------------------------------+-------------------------------+
                                   |
                                   v
+----------------------------------+-------------------------------+
|                   Layer 1: Blueprint Engine (Go)                 |
|  YAML graph parser | typed state machine | agentic nodes         |
|  deterministic nodes | gate nodes | eval nodes | iteration guard |
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
- **OpenAI harness engineering** -- AGENTS.md as table of contents, layered architecture, linter output as remediation signal

---

## Features

### Layer 1: Blueprint Engine

- Typed DAG engine with four node types: **agentic** (LLM agent), **deterministic** (shell commands), **gate** (conditional pass/fail routing), **eval** (evaluation/grading)
- Two-phase YAML parser with `depends_on` vocabulary for declaring node dependencies
- Gate nodes enforce bounded iteration limits -- no infinite retry loops
- `AgentExecutor` interface decouples the engine from any specific agent implementation
- Built-in blueprints: `standard-implementation` (plan -> implement -> lint -> test -> commit) and `bug-fix` (reproduce -> fix -> test -> commit)
- Concurrent node execution for independent graph branches

### Layer 2: Intelligence Harness

- gRPC `ForgeAgent` service consumed by the Go engine via `GrpcAgentExecutor`
- Agent adapters: **Echo** (testing), **Claude Code** (headless CLI)
- Context loader: reads `AGENTS.md` and `.forge/rules/` to compose scoped prompts
- Skill system: filesystem-scanned registry, keyword-based resolver, lifecycle management (evaluate, promote, compare)
- Tool pool: assembly with deferred loading against a context budget, pre/post execution hooks
- Subagent context isolation for parallel agent invocations

### Layer 3: Factory

- Docker sandbox with volume mounts, environment injection, resource limits, and network policy
- Git worktree manager for branch-isolated parallel runs
- PR delivery pipeline: `git push` + `gh pr create`
- Run queue with bounded concurrency (configurable `maxParallel` workers)
- In-memory run registry with concurrent-safe state tracking
- Rule-based task assigner for adapter selection
- Webhook HTTP trigger: `POST /api/v1/runs` to enqueue, `GET /api/v1/runs/:id` to poll
- `forged` daemon with signal handling and graceful shutdown

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
git clone https://github.com/aditya-soni/forge.git
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

### Claude Code agent run

Full autonomous pipeline with a real agent:

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

### Daemon mode (webhook-triggered)

Run `forged` as a long-lived service accepting tasks via HTTP:

```bash
go run ./cmd/forged -port 8080 -max-parallel 4
```

```bash
# Enqueue a run
curl -X POST http://localhost:8080/api/v1/runs \
  -H "Content-Type: application/json" \
  -d '{"task": "add unit tests for the parser module", "adapter": "claude"}'

# Poll status
curl http://localhost:8080/api/v1/runs/<run-id>
```

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

---

## Project Status

| Version | Theme | Status | Highlights |
|---------|-------|--------|------------|
| **v0.1** | Engine + Harness + Factory MVP | **Shipped** | Typed graph engine, YAML parser, gRPC harness, echo + Claude adapters, Docker sandbox, git worktrees, PR delivery, CI/CD, 75+ Go tests (91% coverage), 21 TS tests (100% coverage) |
| **v0.2** | Skills + Tool Pool + Triggers | **Shipped** | EvalNode, skill system (registry/resolver/lifecycle), tool pool with deferred loading, subagent context isolation, webhook triggers, run queue with bounded concurrency, `forged` daemon, 88+ Go tests, 75+ TS tests |
| **v0.3** | Learning + Multi-adapter | Planned | Memory/session capture, failure-to-rule pipeline, Goose/Codex/Cursor adapters, container warm pools, quality gate system |
| **v1.0** | Production-ready | Planned | Built-in skill marketplace, full docs, GitHub Issues triggers, human review queue, run tracing dashboard |

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
