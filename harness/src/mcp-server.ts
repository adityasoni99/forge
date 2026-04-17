import { fileURLToPath } from 'node:url';
import { McpServer } from '@modelcontextprotocol/sdk/server/mcp.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import { z } from 'zod';
import { ForgePluginCore } from './plugin/core.js';
import { EchoAdapter } from './adapters/echo.js';
import { ClaudeCodeAdapter } from './adapters/claude-code.js';
import { GooseAdapter } from './adapters/goose.js';
import { CodexAdapter } from './adapters/codex.js';
import { CursorAdapter } from './adapters/cursor.js';
import type { AgentAdapter } from './adapters/types.js';

export interface ToolDefinition {
  name: string;
  description: string;
  inputSchema: { properties: Record<string, unknown>; required?: string[] };
}

export const FORGE_TOOLS: readonly ToolDefinition[] = Object.freeze([
  {
    name: 'forge_run',
    description: 'Run a task end-to-end: plan, implement, test, commit',
    inputSchema: {
      properties: {
        task: { type: 'string', description: 'The task to execute (e.g. "Add user authentication")' },
      },
      required: ['task'],
    },
  },
  {
    name: 'forge_fix',
    description: 'Reproduce and fix a bug, then test and commit',
    inputSchema: {
      properties: {
        task: { type: 'string', description: 'Bug description or error message' },
        file_path: { type: 'string', description: 'Optional file path where the bug occurs' },
        error_output: { type: 'string', description: 'Optional error output or stack trace' },
      },
      required: ['task'],
    },
  },
  {
    name: 'forge_plan',
    description: 'Create an implementation plan without executing',
    inputSchema: {
      properties: {
        task: { type: 'string', description: 'Feature or task to plan' },
      },
      required: ['task'],
    },
  },
  {
    name: 'forge_status',
    description: 'Show Forge plugin status: detected IDE, adapter, configuration',
    inputSchema: { properties: {}, required: [] },
  },
]);

export function createForgeTools(): readonly ToolDefinition[] {
  return FORGE_TOOLS;
}

function buildAdapterMap(): Map<string, AgentAdapter> {
  const adapters = new Map<string, AgentAdapter>();
  adapters.set('echo', new EchoAdapter());
  adapters.set('claude', new ClaudeCodeAdapter());
  adapters.set('goose', new GooseAdapter());
  adapters.set('codex', new CodexAdapter());
  adapters.set('cursor', new CursorAdapter());
  return adapters;
}

function toolDesc(name: string): string {
  return FORGE_TOOLS.find(t => t.name === name)!.description;
}

export function createMcpServer(): McpServer {
  const mcp = new McpServer({ name: 'forge', version: '0.3.1' });
  const adapters = buildAdapterMap();
  const core = new ForgePluginCore({ adapters });

  mcp.tool('forge_run', toolDesc('forge_run'), {
    task: z.string().min(1).describe('The task to execute'),
  }, async ({ task }) => {
    const result = await core.executeCommand('run', task);
    return {
      content: [{ type: 'text' as const, text: result.success ? result.output : `Error: ${result.error}` }],
      isError: !result.success,
    };
  });

  mcp.tool('forge_fix', toolDesc('forge_fix'), {
    task: z.string().min(1).describe('Bug description or error message'),
    file_path: z.string().optional().describe('File path where the bug occurs'),
    error_output: z.string().optional().describe('Error output or stack trace'),
  }, async ({ task, file_path, error_output }) => {
    const result = await core.executeCommand('fix', task, {
      filePath: file_path,
      errorOutput: error_output,
    });
    return {
      content: [{ type: 'text' as const, text: result.success ? result.output : `Error: ${result.error}` }],
      isError: !result.success,
    };
  });

  mcp.tool('forge_plan', toolDesc('forge_plan'), {
    task: z.string().min(1).describe('Feature or task to plan'),
  }, async ({ task }) => {
    const result = await core.executeCommand('plan', task);
    return {
      content: [{ type: 'text' as const, text: result.success ? result.output : `Error: ${result.error}` }],
      isError: !result.success,
    };
  });

  mcp.tool('forge_status', toolDesc('forge_status'), {}, async () => {
    const status = core.getStatus();
    const text = [
      `IDE: ${status.ide}`,
      `Adapter: ${status.defaultAdapter}`,
      `Available adapters: ${status.availableAdapters.join(', ')}`,
      `Working directory: ${status.workingDirectory}`,
    ].join('\n');
    return { content: [{ type: 'text' as const, text }] };
  });

  return mcp;
}

async function main(): Promise<void> {
  const server = createMcpServer();
  const transport = new StdioServerTransport();
  await server.connect(transport);
}

const isEntry = process.argv[1] === fileURLToPath(import.meta.url);
if (isEntry || process.argv.includes('--mcp')) {
  main().catch((err) => {
    console.error('Forge MCP server error:', err);
    process.exit(1);
  });
}
