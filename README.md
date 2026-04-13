# Forge

Forge is a three-layer open-source agent factory:

1. **Blueprint Engine (Go)** — typed graph execution with agentic, deterministic, and gate nodes
2. **Harness (TypeScript)** — gRPC adapter service for agent runners like Claude Code
3. **Factory (Go)** — Docker sandbox, git worktree isolation, and PR delivery

## Prerequisites

- Go 1.22+
- Node.js 22+
- Docker
- `gh` CLI (optional, for PR delivery)
- Claude Code CLI (optional, for real agent runs)

## Verify the repo

```bash
go test ./...
cd harness && npm ci && npm test && cd ..
make docker-build
```

## Explore built-in blueprints

```bash
go run ./cmd/forge blueprint list
```

## Validate a blueprint

```bash
go run ./cmd/forge blueprint validate ./blueprints/standard-implementation.yaml
```

## Run a local dry run

Uses the echo adapter (no Claude needed):

```bash
go run ./cmd/forge run --no-sandbox "add smoke coverage"
```

With a specific blueprint:

```bash
go run ./cmd/forge run --no-sandbox --blueprint bug-fix "fix failing parser test"
```

## Run in Docker

```bash
go run ./cmd/forge run --adapter echo --no-pr "add smoke coverage"
```

## Real Claude-backed run

To use Claude inside the sandboxed harness, make sure the Docker image contains a working
Claude Code installation and that credentials are available to the container.

```bash
go run ./cmd/forge run --adapter claude --no-pr "implement README quickstart"
```

For local execution without Docker, start the harness separately and pass its address:

```bash
# Terminal 1: start the harness
cd harness
FORGE_ADAPTER=claude FORGE_HARNESS_PORT=50051 npm start
```

```bash
# Terminal 2: run the blueprint
go run ./cmd/forge run --no-sandbox --harness 127.0.0.1:50051 --adapter claude "implement README quickstart"
```

## Architecture

See [docs/design.md](docs/design.md) for the full architecture reference.

## License

Apache 2.0
