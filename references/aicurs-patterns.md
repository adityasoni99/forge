# aicurs — architectural patterns (reference)

Curated extraction from the **aicurs** tree: a filesystem snapshot aligned with Anthropic’s production Claude Code CLI agent (TypeScript/Bun). Use it as **design precedent**, not as a dependency.

For Forge’s system design, see [docs/design.md](../docs/design.md). For external links and narrative context, see [references.md](references.md).

---

## Preamble

### What aicurs is

aicurs is a large TypeScript codebase that mirrors the structure and behavior of the shipped agent CLI: tool registry, permission pipeline, hooks, MCP, subagents, coordinator mode, tasks, and swarm/teammate plumbing. It is useful for **reverse-engineering product patterns** into a blueprint-first engine like Forge.

### Usage caveats

- Treat this snapshot as **architectural reference**, not code to vendor or copy. Licensing, tests, and packaging (for example `package.json`) may be absent or incomplete in the snapshot you have.
- APIs and feature flags **will drift** relative to any future Claude Code release.
- Examples below are **short excerpts**; line numbers refer to the aicurs copy at extraction time.

### How to read this document

Each section follows **What** (intent) / **How** (mechanism) / **Key interfaces** (quoted code) / **Forge mapping** (takeaway). **Forge takeaway** lines match the tone of [references.md](references.md): one or two sentences on what the Go engine, TS harness, or factory layer should learn.

---

## 1. Tool orchestration

**What.** Run multiple `tool_use` blocks per assistant turn with safe parallelism: merge consecutive “concurrency-safe” tools into one parallel batch; run mutating or ambiguous tools serially; cap parallel fan-out; defer context mutations until after the parallel phase in **declaration order**.

**How.**

- `partitionToolCalls` folds the stream of `ToolUseBlock`s into batches: extend the previous batch only if **both** the previous and current tool are concurrency-safe; otherwise start a new batch.
- Safe batches use `runToolsConcurrently`, which fans out through `all(..., getMaxToolUseConcurrency())` (default cap 10 via `CLAUDE_CODE_MAX_TOOL_USE_CONCURRENCY`).
- During the concurrent phase, `contextModifier` callbacks are **queued** per `toolUseID`; after all tools in the batch finish, modifiers run **in original block order** (for loop over `blocks`).
- Serial batches apply `contextModifier` immediately per tool.

**Key interfaces**

```8:116:aicurs/services/tools/toolOrchestration.ts
function getMaxToolUseConcurrency(): number {
  return (
    parseInt(process.env.CLAUDE_CODE_MAX_TOOL_USE_CONCURRENCY || '', 10) || 10
  )
}
// ...
function partitionToolCalls(
  toolUseMessages: ToolUseBlock[],
  toolUseContext: ToolUseContext,
): Batch[] {
  return toolUseMessages.reduce((acc: Batch[], toolUse) => {
    const tool = findToolByName(toolUseContext.options.tools, toolUse.name)
    const parsedInput = tool?.inputSchema.safeParse(toolUse.input)
    const isConcurrencySafe = parsedInput?.success
      ? (() => {
          try {
            return Boolean(tool?.isConcurrencySafe(parsedInput.data))
          } catch {
            // If isConcurrencySafe throws (e.g., due to shell-quote parse failure),
            // treat as not concurrency-safe to be conservative
            return false
          }
        })()
      : false
    if (isConcurrencySafe && acc[acc.length - 1]?.isConcurrencySafe) {
      acc[acc.length - 1]!.blocks.push(toolUse)
    } else {
      acc.push({ isConcurrencySafe, blocks: [toolUse] })
    }
    return acc
  }, [])
}
```

```362:402:aicurs/Tool.ts
  isConcurrencySafe(input: z.infer<Input>): boolean
  isEnabled(): boolean
  isReadOnly(input: z.infer<Input>): boolean
```

**Forge takeaway.** Model each blueprint node with an explicit **concurrency-safe** flag (or derive it from node type + inputs); the Go engine can batch **sibling** nodes the same way aicurs batches tool calls, with a **worker cap** and **ordered context patches** after parallel segments complete.

---

## 2. Permission system

**What.** Central `CanUseToolFn` (`hasPermissionsToUseTool`) composes **stacked rules** (deny / ask / tool-specific / bypass / allow), **permission modes** (including headless auto-deny), and optional **auto-mode classifier** and hooks.

**How.**

- **Rule sources** are merged from settings tiers: `PERMISSION_RULE_SOURCES` includes setting sources plus `cliArg`, `command`, `session` (`utils/permissions/permissions.ts`).
- **Inner pipeline** (`hasPermissionsToUseToolInner`): whole-tool deny → whole-tool ask (with Bash+sandbox carve-out) → `tool.checkPermissions(parsedInput, context)` → respect tool-level deny / interactive-only / content-specific ask rules / `safetyCheck` → **bypass** modes → **always allow** → default `passthrough` becomes **`ask`**.
- **Outer transformations**: `dontAsk` maps `ask` → `deny`; `shouldAvoidPermissionPrompts` runs `PermissionRequest` hooks then **`AUTO_REJECT_MESSAGE`** if still unresolved; auto mode may run classifier fast-paths before prompting.

**Key interfaces**

```109:114:aicurs/utils/permissions/permissions.ts
const PERMISSION_RULE_SOURCES = [
  ...SETTING_SOURCES,
  'cliArg',
  'command',
  'session',
] as const satisfies readonly PermissionRuleSource[]
```

```1158:1318:aicurs/utils/permissions/permissions.ts
async function hasPermissionsToUseToolInner(
  tool: Tool,
  input: { [key: string]: unknown },
  context: ToolUseContext,
): Promise<PermissionDecision> {
  // 1a. Entire tool is denied
  const denyRule = getDenyRuleForTool(appState.toolPermissionContext, tool)
  // ...
  // 2a. bypassPermissions / plan+bypass available
  const shouldBypassPermissions =
    appState.toolPermissionContext.mode === 'bypassPermissions' ||
    (appState.toolPermissionContext.mode === 'plan' &&
      appState.toolPermissionContext.isBypassPermissionsModeAvailable)
  // ...
  // 3. Convert "passthrough" to "ask"
  const result: PermissionDecision =
    toolPermissionResult.behavior === 'passthrough'
      ? { ...toolPermissionResult, behavior: 'ask' as const, /* ... */ }
      : toolPermissionResult
```

```122:137:aicurs/Tool.ts
export type ToolPermissionContext = DeepImmutable<{
  mode: PermissionMode
  additionalWorkingDirectories: Map<string, AdditionalWorkingDirectory>
  alwaysAllowRules: ToolPermissionRulesBySource
  alwaysDenyRules: ToolPermissionRulesBySource
  alwaysAskRules: ToolPermissionRulesBySource
  isBypassPermissionsModeAvailable: boolean
  // ...
  shouldAvoidPermissionPrompts?: boolean
  awaitAutomatedChecksBeforeDialog?: boolean
}>
```

**Forge takeaway.** Keep a **`PermissionChecker`** (or equivalent) at the engine boundary: layered **deny > ask > node-specific > bypass > allow**, with **policy from multiple sources** and an explicit **non-interactive** path that maps **ask → deny** unless hooks (or factory policy) decide otherwise.

---

## 3. Hooks pipeline

**What.** User- and plugin-defined hooks run at named lifecycle events. Implementations can be **shell commands**, **prompt/agent/http** integrations, **callbacks**, or **in-process functions**. Structured **JSON** on stdout controls continuation, permissions, injected context, and MCP output tweaks.

**How.**

- Canonical event names live in `HOOK_EVENTS` (`entrypoints/sdk/coreTypes.ts`). **Additional** UI-oriented hook entrypoints (e.g. `StatusLine`) appear in execution code but are not in `HOOK_EVENTS`.
- `types/hooks.ts` defines Zod schemas for sync responses: `continue`, `suppressOutput`, `stopReason`, plus `hookSpecificOutput` discriminated by `hookEventName` (e.g. `PreToolUse` may set `permissionDecision`, `updatedInput`, `additionalContext`; `PermissionRequest` may return allow/deny decisions with `updatedPermissions`).
- `utils/hooks.ts` implements spawning, timeouts, trust checks (`checkHasTrustDialogAccepted`), and async hook background execution.

**Key interfaces — hook events**

```25:52:aicurs/entrypoints/sdk/coreTypes.ts
export const HOOK_EVENTS = [
  'PreToolUse',
  'PostToolUse',
  'PostToolUseFailure',
  'Notification',
  'UserPromptSubmit',
  'SessionStart',
  'SessionEnd',
  'Stop',
  'StopFailure',
  'SubagentStart',
  'SubagentStop',
  'PreCompact',
  'PostCompact',
  'PermissionRequest',
  'PermissionDenied',
  'Setup',
  'TeammateIdle',
  'TaskCreated',
  'TaskCompleted',
  'Elicitation',
  'ElicitationResult',
  'ConfigChange',
  'WorktreeCreate',
  'WorktreeRemove',
  'InstructionsLoaded',
  'CwdChanged',
  'FileChanged',
] as const
```

*(27 event names in `HOOK_EVENTS`.)*

```49:165:aicurs/types/hooks.ts
export const syncHookResponseSchema = lazySchema(() =>
  z.object({
    continue: z.boolean().optional(),
    suppressOutput: z.boolean().optional(),
    stopReason: z.string().optional(),
    decision: z.enum(['approve', 'block']).optional(),
    // ...
    hookSpecificOutput: z
      .union([
        z.object({
          hookEventName: z.literal('PreToolUse'),
          permissionDecision: permissionBehaviorSchema().optional(),
          permissionDecisionReason: z.string().optional(),
          updatedInput: z.record(z.string(), z.unknown()).optional(),
          additionalContext: z.string().optional(),
        }),
        // ... additional hookEventName variants ...
      ])
      .optional(),
  }),
)
```

**Forge takeaway.** Expose an **`EngineHook`** surface keyed by the same **event vocabulary**, returning a small **structured decision** (continue, permission override, optional context patch). Run hooks **before** irreversible nodes and **trust-gate** hook execution when blueprints come from untrusted paths.

---

## 4. Coordinator mode

**What.** One runtime; **coordinator** sessions swap in a **dedicated system prompt** and a **minimal tool allowlist** (spawn/stop/message workers). **Workers** use the normal async-agent tool set (`ASYNC_AGENT_ALLOWED_TOOLS`), optionally simplified via `CLAUDE_CODE_SIMPLE`.

**How.**

- `COORDINATOR_MODE_ALLOWED_TOOLS` restricts the coordinator to agent orchestration tools (`constants/tools.ts`).
- `getCoordinatorSystemPrompt()` and `getCoordinatorUserContext()` document worker capabilities and workflow expectations (`coordinator/coordinatorMode.ts`).
- Gated by compile-time `feature('COORDINATOR_MODE')` and runtime `CLAUDE_CODE_COORDINATOR_MODE`.

**Key interfaces**

```104:112:aicurs/constants/tools.ts
export const COORDINATOR_MODE_ALLOWED_TOOLS = new Set([
  AGENT_TOOL_NAME,
  TASK_STOP_TOOL_NAME,
  SEND_MESSAGE_TOOL_NAME,
  SYNTHETIC_OUTPUT_TOOL_NAME,
])
```

```55:71:aicurs/constants/tools.ts
export const ASYNC_AGENT_ALLOWED_TOOLS = new Set([
  FILE_READ_TOOL_NAME,
  WEB_SEARCH_TOOL_NAME,
  TODO_WRITE_TOOL_NAME,
  GREP_TOOL_NAME,
  WEB_FETCH_TOOL_NAME,
  GLOB_TOOL_NAME,
  ...SHELL_TOOL_NAMES,
  FILE_EDIT_TOOL_NAME,
  FILE_WRITE_TOOL_NAME,
  NOTEBOOK_EDIT_TOOL_NAME,
  SKILL_TOOL_NAME,
  SYNTHETIC_OUTPUT_TOOL_NAME,
  TOOL_SEARCH_TOOL_NAME,
  ENTER_WORKTREE_TOOL_NAME,
  EXIT_WORKTREE_TOOL_NAME,
])
```

```111:118:aicurs/coordinator/coordinatorMode.ts
export function getCoordinatorSystemPrompt(): string {
  const workerCapabilities = isEnvTruthy(process.env.CLAUDE_CODE_SIMPLE)
    ? 'Workers have access to Bash, Read, and Edit tools, plus MCP tools from configured MCP servers.'
    : 'Workers have access to standard tools, MCP tools from configured MCP servers, and project skills via the Skill tool. Delegate skill invocations (e.g. /commit, /verify) to workers.'

  return `You are Claude Code, an AI assistant that orchestrates software engineering tasks across multiple workers.
```

**Forge takeaway.** **Coordinator vs worker** is **composition**: same execution engine, different **system prompt** and **allowed-tools / allowed-node** policy in YAML—no second orchestration runtime required.

---

## 5. Memory / dream system (autoDream)

**What.** Optional background **consolidation** pass (`/dream`-style) triggered only after **cheap gates** succeed: time since last run, enough touched sessions, cross-process **lock**, then a **forked read-mostly agent** with tightened Bash policy.

**How.**

- Gate order is documented at the top of `autoDream.ts`: time → session scan (with throttle) → PID lock.
- `consolidationLock.ts`: lock file **mtime** doubles as `lastConsolidatedAt`; body holds **PID**; **stale** holders (>1h) or **dead PIDs** can be reclaimed; `rollbackConsolidationLock` rewinds on failure.
- On failure after acquire, rollback restores prior mtime so the time gate can fire again later.

**Key interfaces**

```4:8:aicurs/services/autoDream/autoDream.ts
// Gate order (cheapest first):
//   1. Time: hours since lastConsolidatedAt >= minHours (one stat)
//   2. Sessions: transcript count with mtime > lastConsolidatedAt >= minSessions
//   3. Lock: no other process mid-consolidation
```

```38:84:aicurs/services/autoDream/consolidationLock.ts
export async function tryAcquireConsolidationLock(): Promise<number | null> {
  const path = lockPath()
  // ... stat + read PID ...
  if (mtimeMs !== undefined && Date.now() - mtimeMs < HOLDER_STALE_MS) {
    if (holderPid !== undefined && isProcessRunning(holderPid)) {
      return null
    }
  }
  await mkdir(getAutoMemPath(), { recursive: true })
  await writeFile(path, String(process.pid))
  // verify PID won the race
  return mtimeMs ?? 0
}
```

**Forge takeaway.** Long-running **memory consolidation** belongs in **Layer 2 (harness)** or a **factory job**: ordered **gates**, **durable lock + rollback**, and **isolated agent** with **reduced side effects**—not inside the core DAG walker.

---

## 6. Task lifecycle

**What.** Typed background work (`TaskType`) with uniform **status** transitions, **disk-backed output** (`outputFile` / `outputOffset`), SDK **`task_started`** events, and **XML task notifications** for the UI.

**How.**

- `Task.ts` defines `TaskType`, `TaskStatus`, `isTerminalTaskStatus`, and `createTaskStateBase` (includes `outputFile` from `getTaskOutputPath`).
- `utils/task/framework.ts`: `registerTask` emits structured SDK events; `generateTaskAttachments` reads **deltas** from disk; `applyTaskOffsetsAndEvictions` patches offsets against **fresh** state to avoid races; terminal tasks can be **evicted** after notification + grace windows.

**Key interfaces**

```6:20:aicurs/Task.ts
export type TaskType =
  | 'local_bash'
  | 'local_agent'
  | 'remote_agent'
  | 'in_process_teammate'
  | 'local_workflow'
  | 'monitor_mcp'
  | 'dream'

export type TaskStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'killed'
```

```104:116:aicurs/utils/task/framework.ts
  enqueueSdkEvent({
    type: 'system',
    subtype: 'task_started',
    task_id: task.id,
    tool_use_id: task.toolUseId,
    description: task.description,
    task_type: task.type,
    // ...
  })
```

**Forge takeaway.** Extend Forge **`RunState`** with **task registry**, **terminal-state guards**, and **spill-to-disk** streaming for long outputs; treat **notifications** as a separate channel from **node completion**.

---

## 7. MCP integration

**What.** Connect to MCP servers over **multiple transports** (stdio, SSE, streamable HTTP, WebSocket, IDE variants, session ingress), **discover tools**, normalize **fully qualified names**, enforce **timeouts**, and surface **progress** to tools.

**How.**

- `connectToServer` branches on `serverRef.type` and constructs the appropriate MCP SDK transport (`services/mcp/client.ts`).
- `fetchToolsForClient` calls `tools/list`, sanitizes Unicode, builds `Tool` objects with `buildMcpToolName`, optional **skip-prefix** mode for SDK servers, caps descriptions (`MAX_MCP_DESCRIPTION_LENGTH`), and maps MCP **annotations** to `isConcurrencySafe` / `isReadOnly` / destructive hints.
- `getMcpToolTimeoutMs` reads `MCP_TOOL_TIMEOUT` with a very large default.

**Key interfaces**

```595:677:aicurs/services/mcp/client.ts
export const connectToServer = memoize(
  async (
    name: string,
    serverRef: ScopedMcpServerConfig,
    serverStats?: { /* ... */ },
  ): Promise<MCPServerConnection> => {
    // ...
      if (serverRef.type === 'sse') {
        // ...
        transport = new SSEClientTransport(
          new URL(serverRef.url),
          transportOptions,
        )
```

```1743:1797:aicurs/services/mcp/client.ts
export const fetchToolsForClient = memoizeWithLRU(
  async (client: MCPServerConnection): Promise<Tool[]> => {
    // ...
      const result = (await client.client.request(
        { method: 'tools/list' },
        ListToolsResultSchema,
      )) as ListToolsResult
      // ...
          const fullyQualifiedName = buildMcpToolName(client.name, tool.name)
          return {
            ...MCPTool,
            name: skipPrefix ? tool.name : fullyQualifiedName,
            mcpInfo: { serverName: client.name, toolName: tool.name },
            // ...
            isConcurrencySafe() {
              return tool.annotations?.readOnlyHint ?? false
            },
```

```208:228:aicurs/services/mcp/client.ts
const DEFAULT_MCP_TOOL_TIMEOUT_MS = 100_000_000
function getMcpToolTimeoutMs(): number {
  return (
    parseInt(process.env.MCP_TOOL_TIMEOUT || '', 10) ||
    DEFAULT_MCP_TOOL_TIMEOUT_MS
  )
}
```

**Forge takeaway.** The **TS harness** should own **MCP connection lifecycle**, **tool name normalization**, and **strict timeouts**; the Go engine sees **stable tool IDs** and **typed progress** callbacks.

---

## 8. Subagent spawning

**What.** `runAgent` builds an **isolated** `ToolUseContext` via `createSubagentContext`, optionally **forks parent messages**, merges **MCP clients/tools**, applies **permission overlays** (`allowedTools` replaces session allow list while preserving `cliArg`), runs **`query()`** recursively, persists **sidechain transcripts**, and **cleans up** MCP + session hooks in `finally`.

**How.**

- Async agents get an **unlinked** `AbortController`; sync agents share the parent controller.
- `agentGetAppState` can override **permission mode**, set `shouldAvoidPermissionPrompts`, `awaitAutomatedChecksBeforeDialog`, and **scoped allow rules** when `allowedTools` is passed.
- `initializeAgentMcpServers` adds per-agent MCP servers with explicit **cleanup** for inline-created clients only.

**Key interfaces**

```412:478:aicurs/tools/AgentTool/runAgent.ts
    const agentGetAppState = () => {
      const state = toolUseContext.getAppState()
      let toolPermissionContext = state.toolPermissionContext
      // Override permission mode if agent defines one (unless parent is bypassPermissions, acceptEdits, or auto)
      // ...
      if (allowedTools !== undefined) {
        toolPermissionContext = {
          ...toolPermissionContext,
          alwaysAllowRules: {
            cliArg: state.toolPermissionContext.alwaysAllowRules.cliArg,
            session: [...allowedTools],
          },
        }
      }
```

```697:714:aicurs/tools/AgentTool/runAgent.ts
  const agentToolUseContext = createSubagentContext(toolUseContext, {
    options: agentOptions,
    agentId,
    agentType: agentDefinition.agentType,
    messages: initialMessages,
    readFileState: agentReadFileState,
    abortController: agentAbortController,
    getAppState: agentGetAppState,
    shareSetAppState: !isAsync,
    // ...
  })
```

```816:821:aicurs/tools/AgentTool/runAgent.ts
  } finally {
    await mcpCleanup()
    if (agentDefinition.hooks) {
      clearSessionHooks(rootSetAppState, agentId)
    }
```

**Forge takeaway.** **`AgentExecutor`** implementations should support **nested runs** with **forked context**, **permission overlay**, **transcript isolation**, and **deterministic teardown** (tools + hooks + caches).

---

## 9. Context assembly

**What.** **Memoized** “user” and “system” context fragments (git snapshot, CLAUDE.md injection, date line) assembled in parallel where possible; optional **cache-breaker** injection clears memoization when debugging.

**How.**

- `getSystemContext` and `getUserContext` are `memoize`d functions (`context.ts`).
- `getGitStatus` internally uses `Promise.all` for branch, status, log, and user name.
- `setSystemPromptInjection` clears both caches when injection changes.

**Key interfaces**

```29:34:aicurs/context.ts
export function setSystemPromptInjection(value: string | null): void {
  systemPromptInjection = value
  // Clear context caches immediately when injection changes
  getUserContext.cache.clear?.()
  getSystemContext.cache.clear?.()
}
```

```116:149:aicurs/context.ts
export const getSystemContext = memoize(
  async (): Promise<{
    [k: string]: string
  }> => {
    const gitStatus =
      isEnvTruthy(process.env.CLAUDE_CODE_REMOTE) ||
      !shouldIncludeGitInstructions()
        ? null
        : await getGitStatus()
    // ...
    return {
      ...(gitStatus && { gitStatus }),
      ...(feature('BREAK_CACHE_COMMAND') && injection
        ? { cacheBreaker: `[CACHE_BREAKER: ${injection}]` }
        : {}),
    }
  },
)
```

**Forge takeaway.** The harness should build a **structured `ExecutionContext`** (paths, memory snippets, environment, policy) with **explicit cache keys** for prompt caching—not ad hoc string concatenation per turn.

---

## 10. Swarm / teammates

**What.** **Pluggable backends** (tmux, iTerm2, in-process) for pane-based teammate UIs; **file-backed JSON mailboxes** for cross-agent messages with **lockfile retries**.

**How.**

- `BackendType` and `PaneBackend` describe capability detection, pane creation, and command injection (`utils/swarm/backends/types.ts`).
- `registry.ts` caches detection, supports **in-process fallback** when no pane backend is available, and lazy-registers concrete backend classes.
- `teammateMailbox.ts` stores inbox arrays under `~/.claude/teams/...`; `writeToMailbox` uses `proper-lockfile` async API with **retry/backoff** (`LOCK_OPTIONS`).

**Key interfaces**

```3:9:aicurs/utils/swarm/backends/types.ts
export type BackendType = 'tmux' | 'iterm2' | 'in-process'
```

```31:41:aicurs/utils/teammateMailbox.ts
const LOCK_OPTIONS = {
  retries: {
    retries: 10,
    minTimeout: 5,
    maxTimeout: 100,
  },
}
```

**Forge takeaway.** **Layer 3 (factory)** can swap **process/pane** implementations while keeping a **stable mailbox + message schema** for multi-agent workflows.

---

## 11. Skills and plugins

**What.** Skills are **markdown packages** with **YAML frontmatter** (metadata, allowed tools, hooks, paths, model hints). **Bundled** skills can ship **extra files** extracted lazily to disk on first use.

**How.**

- `loadSkillsDir.ts` parses frontmatter (`parseFrontmatter`), optional **hooks** (`HooksSchema`), path globs for scoping, and produces `Command` objects with `getPromptForCommand` that substitutes arguments, expands `${CLAUDE_SKILL_DIR}`, and (for trusted sources) runs inline shell fragments.
- `bundledSkills.ts` registers skills programmatically; when `files` is set, `getPromptForCommand` is wrapped so **extraction** happens once per process (`extractionPromise` memoization).

**Key interfaces**

```15:41:aicurs/skills/bundledSkills.ts
export type BundledSkillDefinition = {
  name: string
  description: string
  // ...
  hooks?: HooksSettings
  context?: 'inline' | 'fork'
  agent?: string
  files?: Record<string, string>
  getPromptForCommand: (
    args: string,
    context: ToolUseContext,
  ) => Promise<ContentBlockParam[]>
}
```

```59:73:aicurs/skills/bundledSkills.ts
    let extractionPromise: Promise<string | null> | undefined
    const inner = definition.getPromptForCommand
    getPromptForCommand = async (args, ctx) => {
      extractionPromise ??= extractBundledSkillFiles(definition.name, files)
      const extractedDir = await extractionPromise
      const blocks = await inner(args, ctx)
      if (extractedDir === null) return blocks
      return prependBaseDir(blocks, extractedDir)
    }
```

**Forge takeaway.** Blueprint **metadata** can mirror skills: **frontmatter** for triggers, **tool allowlists**, and **hook matchers**, with **lazy artifact materialization** for large references.

---

## 12. Streaming tool execution

**What.** While the model **streams** `tool_use` blocks, start execution as soon as **concurrency rules** allow: parallel **safe** tools; **exclusive** access for unsafe tools; preserve **user-visible ordering** for finalized results; **progress** messages can outpace completion.

**How.**

- `StreamingToolExecutor.addTool` parses input to compute `isConcurrencySafe` (same conservative try/catch pattern as batch orchestration).
- `canExecuteTool` allows parallel execution only if **no unsafe tool is executing**; queue processing **stops at the first blocked unsafe tool** to keep order.
- Bash failures set `hasErrored` and abort a **sibling** `AbortController` to cancel related subprocesses; `discard()` supports streaming fallback.

**Key interfaces**

```34:38:aicurs/services/tools/StreamingToolExecutor.ts
/**
 * Executes tools as they stream in with concurrency control.
 * - Concurrent-safe tools can execute in parallel with other concurrent-safe tools
 * - Non-concurrent tools must execute alone (exclusive access)
 * - Results are buffered and emitted in the order tools were received
 */
```

```129:150:aicurs/services/tools/StreamingToolExecutor.ts
  private canExecuteTool(isConcurrencySafe: boolean): boolean {
    const executingTools = this.tools.filter(t => t.status === 'executing')
    return (
      executingTools.length === 0 ||
      (isConcurrencySafe && executingTools.every(t => t.isConcurrencySafe))
    )
  }

  private async processQueue(): Promise<void> {
    for (const tool of this.tools) {
      if (tool.status !== 'queued') continue

      if (this.canExecuteTool(tool.isConcurrencySafe)) {
        await this.executeTool(tool)
      } else {
        if (!tool.isConcurrencySafe) break
      }
    }
  }
```

**Forge takeaway.** If Layer 2 streams LLM output into the engine, reuse the same **safe/unsafe classification** and **ordered result buffer** semantics—**do not** require all tool calls to arrive before scheduling.

---

## 13. Feature flags

**What.** Two layers: **compile-time** stripping via `feature('FLAG')` from `bun:bundle`, and **runtime** experiments/configuration via **GrowthBook** (`getFeatureValue_*`, `checkStatsigFeatureGate_*`).

**How.**

- `main.tsx` imports `feature` from `bun:bundle` and wraps optional modules in **ternary requires** (dead-code elimination for unused product modes).
- `permissions.ts`, `client.ts`, `autoDream.ts`, and others call GrowthBook helpers for dynamic gates and numeric config (e.g. autoDream thresholds).

**Key interfaces**

```21:36:aicurs/main.tsx
import { feature } from 'bun:bundle';
// ...
import { hasGrowthBookEnvOverride, initializeGrowthBook, refreshGrowthBookAfterAuthChange } from './services/analytics/growthbook.js';
```

```74:81:aicurs/main.tsx
const coordinatorModeModule = feature('COORDINATOR_MODE') ? require('./coordinator/coordinatorMode.js') as typeof import('./coordinator/coordinatorMode.js') : null;
const assistantModule = feature('KAIROS') ? require('./assistant/index.js') as typeof import('./assistant/index.js') : null;
const kairosGate = feature('KAIROS') ? require('./assistant/gate.js') as typeof import('./assistant/gate.js') : null;
```

```58:64:aicurs/utils/permissions/permissions.ts
const classifierDecisionModule = feature('TRANSCRIPT_CLASSIFIER')
  ? (require('./classifierDecision.js') as typeof import('./classifierDecision.js'))
  : null
```

```73:92:aicurs/services/autoDream/autoDream.ts
function getConfig(): AutoDreamConfig {
  const raw =
    getFeatureValue_CACHED_MAY_BE_STALE<Partial<AutoDreamConfig> | null>(
      'tengu_onyx_plover',
      null,
    )
  return {
    minHours: /* validated */ DEFAULTS.minHours,
    minSessions: /* validated */ DEFAULTS.minSessions,
  }
}
```

**Forge takeaway.** Map **compile-time** aicurs flags to **Go build tags** or linker flags for binary size; map **GrowthBook**-style knobs to **env + config files** in the factory, with **schema validation** (as autoDream does for remote JSON).

---

## Cross-cutting themes

| Theme | Where it shows up | Implication for Forge |
|--------|-------------------|-------------------------|
| **Layered policy** | Permissions + hooks + tool `checkPermissions` | Single **decision pipeline** with explicit ordering |
| **Concurrency metadata** | `isConcurrencySafe`, MCP `readOnlyHint` | First-class **parallelism hints** on nodes/tools |
| **Isolation** | Subagent context, disk transcripts, task output files | **Fork state**, don’t alias mutable session |
| **Durable side channels** | Tasks, mailboxes, locks | **Spill heavy data to disk**; keep engine state small |
| **Progress + ordering** | Streaming executor, task deltas | **Stream progress**; **finalize** in deterministic order |
| **Feature gating** | `bun:bundle` + GrowthBook | **Static** vs **runtime** flags split |

---

## Appendix — aicurs file index

Paths below are relative to the aicurs repository root.

| Area | Primary files |
|------|----------------|
| Tool orchestration | `services/tools/toolOrchestration.ts`, `utils/generators.ts` (`all`) |
| Tool surface | `Tool.ts` |
| Permissions | `utils/permissions/permissions.ts`, `utils/permissions/PermissionResult.ts`, `utils/permissions/PermissionRule.ts` |
| Hooks | `utils/hooks.ts`, `types/hooks.ts`, `entrypoints/sdk/coreTypes.ts` |
| Coordinator | `coordinator/coordinatorMode.ts`, `constants/tools.ts` |
| Auto dream | `services/autoDream/autoDream.ts`, `services/autoDream/consolidationLock.ts` |
| Tasks | `Task.ts`, `utils/task/framework.ts`, `utils/task/diskOutput.ts` |
| MCP | `services/mcp/client.ts`, `services/mcp/mcpStringUtils.ts`, `services/mcp/normalization.ts` |
| Subagents | `tools/AgentTool/runAgent.ts`, `utils/forkedAgent.ts` |
| Context | `context.ts` |
| Swarm | `utils/swarm/backends/types.ts`, `utils/swarm/backends/registry.ts`, `utils/teammateMailbox.ts` |
| Skills | `skills/loadSkillsDir.ts`, `skills/bundledSkills.ts` |
| Streaming tools | `services/tools/StreamingToolExecutor.ts` |
| Feature flags / startup | `main.tsx`, `services/analytics/growthbook.ts` (referenced throughout) |

---

## Maintenance

When updating this reference after refreshing the aicurs snapshot:

1. Re-verify excerpts against current files (line numbers will shift).
2. Prefer **pattern-level** descriptions over full algorithm dumps.
3. Keep **Forge takeaway** lines actionable for `docs/design.md` and Go API design.
