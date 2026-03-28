# Forge

Three-layer open-source agent factory. See project.md for full architecture.

## Quick Reference

- `core/blueprint/` -- Layer 1: Blueprint Engine (state machine, typed nodes, graph, YAML parser)
- `cmd/forge/` -- CLI entrypoint
- `blueprints/` -- Built-in YAML blueprint definitions
- `harness/` -- Layer 2: Intelligence harness (TypeScript, future)
- `factory/` -- Layer 3: Infrastructure factory (Go, future)

## Key Concepts

- **Blueprint**: YAML-defined DAG of nodes representing an agent workflow
- **Node Types**: AgenticNode (LLM agent), DeterministicNode (shell command), GateNode (conditional routing)
- **Engine**: Walks the graph, executes nodes, tracks RunState, enforces iteration limits
- **AgentExecutor**: Interface for plugging in any AI agent (Layer 2 provides implementations via gRPC)

## Conventions

- Google Go Style Guide; tabs for indentation (gofmt standard)
- TDD: write failing test first, then implement
- Files under `core/blueprint/` should stay focused (<300 lines each)
- Security: blueprints must come from trusted sources only; DeterministicNode executes shell commands from blueprint YAML
