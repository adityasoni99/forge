# Forge

Three-layer open-source agent factory. **[project.md](project.md)** is the **base
map** for repo layout, module status, and links to all design docs.

## Quick Reference

- `core/blueprint/` -- Layer 1: Blueprint Engine (state machine, typed nodes, graph, YAML parser)
- `cmd/forge/` -- CLI entrypoint
- `blueprints/` -- Built-in YAML blueprint definitions
- `harness/` -- Layer 2: Intelligence harness (TypeScript gRPC server, adapters, context loader)
- `internal/grpcexec/` -- Go gRPC client bridging Layer 1 engine to Layer 2 harness
- `proto/forge/v1/` -- gRPC service definition (ForgeAgent)
- `factory/sandbox/` -- Docker container lifecycle for isolated runs
- `factory/workspace/` -- Git worktree creation/cleanup
- `factory/delivery/` -- Git push + PR creation via gh
- `factory/orchestrator/` -- Run pipeline (workspace -> sandbox -> delivery)

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


# Workflow Orchestration

### 1. Plan Mode Default
- Enter plan mode for ANY non-trivial task (3+ steps or architectural decisions)
- If something goes sideways, STOP and re-plan immediately – don't keep pushing
- Use plan mode for verification steps, not just building
- Write detailed specs upfront to reduce ambiguity

### 2. Subagent Strategy
- Use subagents liberally to keep main context window clean
- Offload research, exploration, and parallel analysis to subagents
- For complex problems, throw more compute at it via subagents
- One task per subagent for focused execution

### 3. Self-Improvement Loop
- After any correction from the user: update `tasks/lessons.md` with the pattern
- Write rules for yourself that prevent the same mistake
- Ruthlessly iterate on these lessons until mistake rate drops
- Review lessons at session start for relevant project

### 4. Verification Before Done
- Never mark a task complete without proving it works
- Diff behavior between main and your changes when relevant
- Ask yourself: "Would a staff engineer approve this?"
- Run tests, check logs, demonstrate correctness

### 5. Demand Elegance (Balanced)
- For non-trivial changes: pause and ask "is there a more elegant way?"
- If a fix feels hacky: "Knowing everything I know now, implement the elegant solution"
- Skip this for simple, obvious fixes – don't over-engineer
- Challenge your own work before presenting it

### 6. Autonomous Bug Fixing
- When given a bug report: just fix it. Don't ask for hand-holding
- Point at logs, errors, failing tests – then resolve them
- Zero context switching required from the user
- Go fix failing CI tests without being told how

## Task Management

1. **Plan First**: Write plan to `tasks/todo.md` with checkable items
2. **Verify Plan**: Check in before starting implementation
3. **Track Progress**: Mark items complete as you go
4. **Explain Changes**: High-level summary at each step
5. **Document Results**: Add review section to `tasks/todo.md`
6. **Capture Lessons**: Update `tasks/lessons.md` after corrections

## Core Principles

- **Simplicity First**: Make every change as simple as possible. Impact minimal code.
- **No Laziness**: Find root causes. No temporary fixes. Senior developer standards.
- **Minimal Impact**: Changes should only touch what's necessary. Avoid introducing bugs.