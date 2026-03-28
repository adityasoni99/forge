# Forge Project

## Overview

Forge wraps any AI coding agent in an industrial harness. Three layers:
Layer 1 (Core) orchestrates, Layer 2 (Harness) adds intelligence, Layer 3 (Factory) provides infrastructure.

## Module Dependency Diagram

    +------------------+
    |  cmd/forge (CLI) |
    +--------+---------+
             |
    +--------v---------+
    | core/blueprint   |  Layer 1: Engine, Graph, Nodes, YAML
    +--------+---------+
             |
    (AgentExecutor interface -- seam for Layer 2)

## Modules

| Module | Language | Status | Depends On |
|--------|----------|--------|------------|
| core/blueprint | Go | In Progress | (none) |
| harness/ | TypeScript | Planned | core/blueprint via gRPC |
| factory/ | Go | Planned | core/blueprint, harness via gRPC |
| cmd/forge | Go | In Progress | core/blueprint |

## Implementation Order

1. core/blueprint (types -> graph -> nodes -> engine -> yaml)
2. cmd/forge (CLI skeleton)
3. harness/ (Plan 2)
4. factory/ (Plan 3)
5. Integration (Plan 4)
