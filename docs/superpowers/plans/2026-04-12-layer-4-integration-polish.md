# Layer 4 Integration + Polish Implementation Plan

> **Status:** **Complete** — all tasks implemented; checkboxes below reflect completed work.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Wire Layers 1–3 into an honest end-to-end `forge run` flow, then add the docs, CI, and design reconciliation needed to make the v0.1 MVP reproducible.

**Architecture:** Keep Layer 4 narrow: fix the contract mismatch between `forge run`, `forge blueprint run`, and `scripts/sandbox-entry.sh`; add a deterministic smoke path that proves orchestration works in CI; then document and automate the exact supported workflows. Do **not** pull v0.2/v0.3 items (skills, memory, warm pools, Slack triggers) into this plan.

**Tech Stack:** Go, TypeScript, gRPC, Docker, YAML blueprints, GitHub Actions

---

## Scope decision

No prerequisite mini-plan is needed before Layer 4.

- The roadmap says the next artifact after Layer 3 is **Integration + Polish**.
- The newly added references (`seeing-like-an-agent`, `metaharness`, `gstack`, `Karpathy LLM Wiki`, `obsidian-second-brain`) are useful design input, but they map to **v0.2+** concerns, not MVP-blocking work.
- The actual MVP blocker is integration honesty: the codebase currently has a contract gap between `forge run`, `forge blueprint run`, and the Docker entrypoint.

---

## File structure and responsibility map

### Existing files to modify

- `cmd/forge/main.go`
  - Single CLI entrypoint.
  - Must become the canonical place for:
    - resolving built-in vs file blueprints,
    - applying `{{task}}` substitution,
    - local `--no-sandbox` execution,
    - clearer `forge run` / `forge blueprint run` flag parsing.
- `cmd/forge/main_test.go`
  - Existing CLI test file.
  - Extend with focused flag/contract tests instead of only coarse smoke checks.
- `blueprints/standard-implementation.yaml`
  - Built-in task blueprint.
  - Must accept task text via `{{task}}` placeholders.
- `blueprints/bug-fix.yaml`
  - Built-in bugfix blueprint.
  - Must accept task text via `{{task}}` placeholders.
- `factory/orchestrator/pipeline.go`
  - Builds sandbox command args.
  - Must preserve the chosen blueprint source (`--blueprint` vs `--blueprint-file`) and pass task text through consistently.
- `factory/orchestrator/pipeline_test.go`
  - Add focused assertions for command building and file-vs-built-in behavior.
- `scripts/sandbox-entry.sh`
  - Runtime glue inside Docker.
  - Must invoke `forge blueprint run` with the same contract the Go CLI supports.
- `tests/factory_integration_test.go`
  - Docker-backed integration test.
  - Repoint to a deterministic smoke blueprint so CI can prove the pipeline without relying on agent side effects.
- `test/integration/harness_test.go`
  - Harness e2e already exists.
  - Keep as the proof that Go engine ↔ TS harness works; add only minimal assertions if needed.
- `docs/design.md`
  - Canonical architecture doc.
  - Must stop claiming harness/factory are “not yet merged”.
- `roadmap.md`
  - Update checkpoint after Layer 4 is implemented.
- `project.md`
  - Update status and implementation snapshot after Layer 4 is implemented.

### New files to create

- `.github/workflows/ci.yml`
  - Run Go tests, TS tests, Docker build, and integration checks.
- `tests/testdata/integration-smoke.yaml`
  - Deterministic smoke blueprint for CI and local verification.
- `README.md`
  - First real quickstart and verification guide for Forge.

### Files intentionally out of scope

- `harness/src/skills/*`
- `harness/src/memory/*`
- `factory/triggers/*`
- `cmd/forged/*`

Those belong to v0.2+ and must **not** be pulled into Layer 4.

---

### Task 1: Unify blueprint source resolution and task templating

**Files:**
- Modify: `cmd/forge/main.go`
- Modify: `cmd/forge/main_test.go`
- Modify: `blueprints/standard-implementation.yaml`
- Modify: `blueprints/bug-fix.yaml`

- [x] **Step 1: Write the failing tests for built-in resolution and task templating**

```go
func TestParseBlueprintRunArgsBuiltin(t *testing.T) {
	harness, file, builtin, task, err := parseBlueprintRunArgs([]string{
		"--builtin", "standard-implementation",
		"--task", "add JSON logging",
		"--harness", "127.0.0.1:50051",
	})
	if err != nil {
		t.Fatalf("parseBlueprintRunArgs returned error: %v", err)
	}
	if harness != "127.0.0.1:50051" {
		t.Fatalf("harness = %q, want %q", harness, "127.0.0.1:50051")
	}
	if file != "" {
		t.Fatalf("file = %q, want empty", file)
	}
	if builtin != "standard-implementation" {
		t.Fatalf("builtin = %q, want %q", builtin, "standard-implementation")
	}
	if task != "add JSON logging" {
		t.Fatalf("task = %q, want %q", task, "add JSON logging")
	}
}

func TestResolveBlueprintDataBuiltinTemplate(t *testing.T) {
	data, label, err := resolveBlueprintData("", "standard-implementation", "add JSON logging")
	if err != nil {
		t.Fatalf("resolveBlueprintData returned error: %v", err)
	}
	if label != "standard-implementation" {
		t.Fatalf("label = %q, want %q", label, "standard-implementation")
	}
	got := string(data)
	if !strings.Contains(got, "add JSON logging") {
		t.Fatalf("resolved blueprint missing task substitution: %s", got)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/forge -run 'TestParseBlueprintRunArgsBuiltin|TestResolveBlueprintDataBuiltinTemplate' -v`

Expected: FAIL with compile errors because `parseBlueprintRunArgs` does not return `builtin` / `task`, and `resolveBlueprintData` does not exist yet.

- [x] **Step 3: Implement built-in blueprint loading plus `{{task}}` substitution**

```go
func parseBlueprintRunArgs(args []string) (harnessAddr, file, builtin, task string, err error) {
	var files []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--harness":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--harness requires an address")
			}
			i++
			harnessAddr = args[i]
		case "--builtin":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--builtin requires a blueprint name")
			}
			i++
			builtin = args[i]
		case "--task":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--task requires text")
			}
			i++
			task = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", "", "", fmt.Errorf("unknown flag: %s", args[i])
			}
			files = append(files, args[i])
		}
	}
	if builtin != "" && len(files) > 0 {
		return "", "", "", "", fmt.Errorf("use either --builtin or a blueprint file, not both")
	}
	if builtin == "" && len(files) != 1 {
		return "", "", "", "", fmt.Errorf("expected exactly one blueprint file or --builtin <name>")
	}
	if len(files) == 1 {
		file = files[0]
	}
	return harnessAddr, file, builtin, task, nil
}

func resolveBlueprintData(file, builtin, task string) ([]byte, string, error) {
	var data []byte
	var err error
	label := file

	switch {
	case file != "":
		data, err = os.ReadFile(file)
	case builtin != "":
		data, err = blueprints.BuiltIn.ReadFile(builtin + ".yaml")
		label = builtin
	default:
		return nil, "", fmt.Errorf("no blueprint source provided")
	}
	if err != nil {
		return nil, "", err
	}
	if task != "" {
		data = []byte(strings.ReplaceAll(string(data), "{{task}}", task))
	}
	return data, label, nil
}
```

```yaml
name: standard-implementation
version: "0.1"
description: "Standard implementation: plan, implement, lint, test, commit"
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
```

```yaml
name: bug-fix
version: "0.1"
description: "Bug fix: reproduce, fix, test, commit"
start: reproduce
nodes:
  reproduce:
    type: agentic
    config:
      prompt: "Write a failing test that reproduces this bug: {{task}}"
  fix:
    type: agentic
    config:
      prompt: "Fix the bug described here: {{task}}. The reproducing test must pass."
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/forge -run 'TestParseBlueprintRunArgsBuiltin|TestResolveBlueprintDataBuiltinTemplate' -v`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add cmd/forge/main.go cmd/forge/main_test.go blueprints/standard-implementation.yaml blueprints/bug-fix.yaml
git commit -m "feat: support built-in blueprint task templating"
```

---

### Task 2: Align `forge run`, local mode, and the Docker entrypoint

**Files:**
- Modify: `cmd/forge/main.go`
- Modify: `cmd/forge/main_test.go`
- Modify: `factory/orchestrator/pipeline.go`
- Modify: `factory/orchestrator/pipeline_test.go`
- Modify: `scripts/sandbox-entry.sh`

- [x] **Step 1: Write the failing tests for run-mode contract alignment**

```go
func TestBuildSandboxCommandUsesBlueprintFile(t *testing.T) {
	req := RunRequest{
		BlueprintFile: "tests/testdata/integration-smoke.yaml",
		Task:          "smoke task",
		Adapter:       "echo",
	}
	got := buildSandboxCommand(req)
	want := []string{
		"--blueprint-file", "tests/testdata/integration-smoke.yaml",
		"--task", "smoke task",
		"--adapter", "echo",
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("buildSandboxCommand mismatch (-want +got):\n%s", diff)
	}
}

func TestForgeRunNoSandboxRequiresHarnessForClaude(t *testing.T) {
	cmd := forgeCmd(t, "run", "--no-sandbox", "--adapter", "claude", "ship Layer 4")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected failure, got success: %s", out)
	}
	if !strings.Contains(string(out), "--harness is required") {
		t.Fatalf("expected harness guidance, got: %s", out)
	}
}
```

- [x] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/forge ./factory/orchestrator -run 'TestBuildSandboxCommandUsesBlueprintFile|TestForgeRunNoSandboxRequiresHarnessForClaude' -v`

Expected: FAIL because `forge run` does not support `--blueprint-file` / `--harness`, and the current local `--no-sandbox` branch returns early.

- [x] **Step 3: Implement one shared run contract**

```go
func cmdForgeRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	blueprintName := fs.String("blueprint", "standard-implementation", "Built-in blueprint name")
	blueprintFile := fs.String("blueprint-file", "", "Path to blueprint YAML file")
	harnessAddr := fs.String("harness", "", "Harness gRPC address for local runs")
	noSandbox := fs.Bool("no-sandbox", false, "Run without Docker sandbox")
	noPR := fs.Bool("no-pr", false, "Skip PR creation")
	adapter := fs.String("adapter", "echo", "Agent adapter (echo, claude)")
	image := fs.String("image", "forge:latest", "Docker image for sandbox")
	baseBranch := fs.String("base-branch", "main", "Base branch for PR")
	fs.Parse(args)

	task := strings.Join(fs.Args(), " ")
	if task == "" {
		fmt.Fprintln(os.Stderr, "usage: forge run [flags] \"task description\"")
		os.Exit(1)
	}
	if *blueprintFile != "" && *blueprintName != "" && *blueprintName != "standard-implementation" {
		fmt.Fprintln(os.Stderr, "use either --blueprint or --blueprint-file")
		os.Exit(1)
	}

	if *noSandbox {
		if *adapter == "claude" && *harnessAddr == "" {
			fmt.Fprintln(os.Stderr, "--harness is required for --no-sandbox with --adapter claude")
			os.Exit(1)
		}
		runBlueprintSource(*blueprintFile, *blueprintName, task, *harnessAddr)
		return
	}

	req := orchestrator.RunRequest{
		Task:          task,
		BlueprintName: *blueprintName,
		BlueprintFile: *blueprintFile,
		RepoDir:       cwd,
		Adapter:       *adapter,
		Image:         *image,
		NoPR:          *noPR,
		BaseBranch:    *baseBranch,
	}
}
```

```bash
RUN_ARGS="blueprint run"
if [ -n "$BLUEPRINT_FILE" ]; then
  RUN_ARGS="$RUN_ARGS $BLUEPRINT_FILE"
elif [ -n "$BLUEPRINT" ]; then
  RUN_ARGS="$RUN_ARGS --builtin $BLUEPRINT"
fi
if [ -n "$TASK" ]; then
  RUN_ARGS="$RUN_ARGS --task $TASK"
fi
RUN_ARGS="$RUN_ARGS --harness localhost:$HARNESS_PORT"
```

- [x] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/forge ./factory/orchestrator -run 'TestBuildSandboxCommandUsesBlueprintFile|TestForgeRunNoSandboxRequiresHarnessForClaude' -v`

Expected: PASS

- [x] **Step 5: Commit**

```bash
git add cmd/forge/main.go cmd/forge/main_test.go factory/orchestrator/pipeline.go factory/orchestrator/pipeline_test.go scripts/sandbox-entry.sh
git commit -m "feat: align forge run with sandbox and local execution"
```

---

### Task 3: Add a deterministic smoke path for Layer 4 verification

**Files:**
- Create: `tests/testdata/integration-smoke.yaml`
- Modify: `tests/factory_integration_test.go`
- Modify: `test/integration/harness_test.go`

- [x] **Step 1: Write the failing smoke integration test**

```go
func TestFactoryIntegrationSmokeBlueprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping factory integration test")
	}

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	copyFile(t,
		filepath.Join("testdata", "integration-smoke.yaml"),
		filepath.Join(repoDir, "integration-smoke.yaml"),
	)

	wsMgr := workspace.NewManager()
	ws, err := wsMgr.Create(context.Background(), repoDir, "integration-test")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	defer wsMgr.Destroy(context.Background(), ws)

	runner := &sandbox.ExecRunner{}
	sbx := sandbox.NewDockerSandbox(runner)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := sbx.Run(ctx, sandbox.SandboxConfig{
		Image:        "forge:latest",
		WorkspaceDir: ws.Dir,
		Env:          map[string]string{"FORGE_ADAPTER": "echo"},
		NetworkMode:  "none",
	}, []string{"--blueprint-file", "integration-smoke.yaml", "--task", "smoke task"})
	if err != nil {
		t.Fatalf("sandbox run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, output = %s", result.ExitCode, result.Output)
	}
}
```

- [x] **Step 2: Run the test to verify it fails**

Run: `go test ./tests -run TestFactoryIntegrationSmokeBlueprint -v`

Expected: FAIL because the smoke blueprint file does not exist yet, and/or `sandbox-entry.sh` and `forge blueprint run` do not agree on task/file handling.

- [x] **Step 3: Create the deterministic smoke blueprint**

```yaml
name: integration-smoke
version: "0.1"
description: "Deterministic smoke blueprint for Layer 4 integration tests"
start: create_file
nodes:
  create_file:
    type: deterministic
    config:
      command: "printf 'smoke-ok\\n' > smoke.txt"
  verify_file:
    type: deterministic
    config:
      command: "test -f smoke.txt && grep -q 'smoke-ok' smoke.txt"
edges:
  - from: create_file
    to: verify_file
```

- [x] **Step 4: Re-run the integration tests**

Run: `go test ./test/integration ./tests -v`

Expected:
- `test/integration` PASS (existing harness e2e still green)
- `tests` PASS when Docker is available and `forge:latest` has been built

- [x] **Step 5: Commit**

```bash
git add tests/testdata/integration-smoke.yaml tests/factory_integration_test.go test/integration/harness_test.go
git commit -m "test: add deterministic layer 4 smoke coverage"
```

---

### Task 4: Add CI that proves Go, TypeScript, and Docker all still fit together

**Files:**
- Create: `.github/workflows/ci.yml`
- Modify: `Makefile`

- [x] **Step 1: Write the workflow file first**

```yaml
name: ci

on:
  push:
    branches: [main]
  pull_request:

jobs:
  go-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: go test ./...

  harness-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: harness/package-lock.json
      - run: npm ci
        working-directory: harness
      - run: npm test
        working-directory: harness
      - run: npm run build
        working-directory: harness

  docker-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
          cache-dependency-path: harness/package-lock.json
      - run: npm ci
        working-directory: harness
      - run: make docker-build
```

- [x] **Step 2: Add a stable Make target for CI**

```make
.PHONY: ci test-go test-ts docker-build

test-go:
	go test ./...

test-ts:
	cd harness && npm test

ci: test-go test-ts docker-build
```

- [x] **Step 3: Run the local verification commands**

Run:

```bash
go test ./...
cd harness && npm test
make docker-build
```

Expected:
- Go tests PASS
- TS tests PASS
- Docker image `forge:latest` builds successfully

- [x] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml Makefile
git commit -m "ci: add layer 4 validation workflow"
```

---

### Task 5: Write the real README quickstart

**Files:**
- Create: `README.md`

- [x] **Step 1: Write the README skeleton with only supported workflows**

```markdown
# Forge

Forge is a three-layer open-source agent factory:

1. **Blueprint Engine (Go)** — typed graph execution with agentic, deterministic, and gate nodes
2. **Harness (TypeScript)** — gRPC adapter service for agent runners like Claude Code
3. **Factory (Go)** — Docker sandbox, git worktree isolation, and PR delivery

## Prerequisites

- Go
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

```bash
go run ./cmd/forge run --no-sandbox --blueprint bug-fix "fix failing parser test"
```

## Run in Docker with the harness inside the sandbox

```bash
go run ./cmd/forge run --adapter echo --no-pr "add smoke coverage"
```
```
```

- [x] **Step 2: Replace any dishonest quickstart command with one that is actually supported**

```markdown
## Real Claude-backed run

To use Claude inside the sandboxed harness, make sure the image contains a working
Claude Code installation and that credentials are available to the container.

```bash
go run ./cmd/forge run --adapter claude --no-pr "implement README quickstart"
```

If you want local execution without Docker, start the harness separately and pass
its address explicitly:

```bash
cd harness
FORGE_ADAPTER=claude FORGE_HARNESS_PORT=50051 npm start
```

```bash
go run ./cmd/forge run --no-sandbox --harness 127.0.0.1:50051 --adapter claude "implement README quickstart"
```
```

- [x] **Step 3: Review the README commands by actually running the safe ones**

Run:

```bash
go run ./cmd/forge blueprint list
go run ./cmd/forge blueprint validate ./blueprints/standard-implementation.yaml
```

Expected:
- blueprint list prints built-in blueprint names
- blueprint validate reports the YAML as valid

- [x] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add honest layer 4 quickstart"
```

---

### Task 6: Reconcile the design docs with implemented reality

**Files:**
- Modify: `docs/design.md`
- Modify: `roadmap.md`
- Modify: `project.md`

- [x] **Step 1: Write the failing doc assertions as grep checks**

Run:

```bash
rg -n "not yet merged|no top-level `factory/`|Not started" docs/design.md roadmap.md project.md
```

Expected:
- `docs/design.md` still contains stale “not yet merged” / “no top-level factory” language
- `roadmap.md` still shows Layer 4 as not started until this work lands

- [x] **Step 2: Update the stale architecture status text**

```markdown
**Status:** implemented in `harness/` with gRPC server, echo + Claude adapters,
context loading, and Go bridge in `internal/grpcexec/`.
```

```markdown
**Status:** implemented in `factory/` with sandbox, workspace, orchestrator, and
delivery packages; Layer 4 finishes end-to-end wiring, CI, and quickstart polish.
```

```markdown
| **v0.1** | Blueprint Engine + Harness MVP + Factory MVP + Integration + Polish | Complete |
```

- [x] **Step 3: Run the doc sanity checks**

Run:

```bash
rg -n "not yet merged|no top-level `factory/`" docs/design.md
go test ./...
```

Expected:
- No stale implementation-status phrases remain in `docs/design.md`
- Go tests still PASS after doc edits

- [x] **Step 4: Commit**

```bash
git add docs/design.md roadmap.md project.md
git commit -m "docs: reconcile layer 4 implementation status"
```

---

## Final verification pass

- [x] **Step 1: Run the full Layer 4 verification sequence**

```bash
go test ./...
cd harness && npm test && npm run build && cd ..
make docker-build
```

Expected:
- Go tests PASS
- TS tests PASS
- Docker image builds successfully

- [x] **Step 2: Run the manual CLI verification checklist**

```bash
go run ./cmd/forge blueprint list
go run ./cmd/forge blueprint validate ./blueprints/bug-fix.yaml
go run ./cmd/forge run --no-sandbox "document Layer 4"
```

Expected:
- built-ins list successfully
- blueprint validation succeeds
- local no-sandbox path either runs with echo or prints a clear actionable error for unsupported combinations

- [x] **Step 3: Update roadmap checkpoint**

```markdown
**Current checkpoint:** Layers 1–4 complete. **Next action:** Start v0.2 planning
(skills, tool pool assembly, EvalNode, Slack trigger, parallel runs).
```

- [x] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: complete layer 4 integration and polish"
```

---

## Self-review

### Spec coverage

- Layer 4 e2e wiring: covered by Tasks 1–3.
- README quickstart: covered by Task 5.
- CI pipeline: covered by Task 4.
- ADR / design status updates: covered by Task 6.
- No extra pre-Layer-4 plan: explicitly handled in the scope decision.

### Placeholder scan

- No `TODO`, `TBD`, or “similar to above” placeholders.
- Every code-changing step includes concrete code.
- Every verification step includes an exact command and expected result.

### Type consistency

- Blueprint source is consistently modeled as either `--builtin` / `--blueprint` or `--blueprint-file`.
- Task text is consistently represented as `task` and substituted as `{{task}}`.
- Local real-agent execution consistently requires `--harness` for non-echo adapters.

---

Plan complete and saved to `docs/superpowers/plans/2026-04-12-layer-4-integration-polish.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
