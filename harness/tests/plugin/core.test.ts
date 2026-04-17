import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ForgePluginCore } from '../../src/plugin/core.js';
import { IDEType } from '../../src/plugin/types.js';
import type { AgentAdapter, AgentAdapterRequest, AgentCapabilities } from '../../src/adapters/types.js';
import type { AgentEvent } from '../../src/adapters/events.js';

class StubAdapter implements AgentAdapter {
  readonly name: string;
  constructor(name: string) { this.name = name; }
  async *execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent> {
    yield { type: 'done', content: `[${this.name}] executed` };
  }
  getCapabilities(): AgentCapabilities {
    return { streaming: false, interrupt: false, maxContextTokens: 200000, supportsTools: false, needsContextReset: false };
  }
  async interrupt(): Promise<void> {}
}

describe('ForgePluginCore', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    delete process.env.CURSOR_TRACE_ID;
    delete process.env.CURSOR_SESSION;
    delete process.env.CLAUDE_CODE;
    delete process.env.CLAUDE_SESSION_ID;
    delete process.env.CODEIUM_SESSION;
    delete process.env.WINDSURF_SESSION;
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('initializes with detected IDE context', () => {
    process.env.CURSOR_TRACE_ID = 'test';
    const core = new ForgePluginCore();
    expect(core.ideContext.ide).toBe(IDEType.Cursor);
  });

  it('executeCommand routes run to standard-implementation', async () => {
    const core = new ForgePluginCore({
      adapters: new Map([['claude', new StubAdapter('claude')]]),
    });
    const result = await core.executeCommand('run', 'Add login feature');
    expect(result.success).toBe(true);
    expect(result.output).toContain('claude');
  });

  it('executeCommand returns error for unknown command', async () => {
    const core = new ForgePluginCore({
      adapters: new Map([['claude', new StubAdapter('claude')]]),
    });
    const result = await core.executeCommand('unknown', 'task');
    expect(result.success).toBe(false);
    expect(result.error).toContain('unknown command');
  });

  it('listCommands returns all registered commands', () => {
    const core = new ForgePluginCore();
    const commands = core.listCommands();
    expect(commands.length).toBeGreaterThanOrEqual(3);
    expect(commands.map(c => c.name)).toContain('run');
    expect(commands.map(c => c.name)).toContain('fix');
    expect(commands.map(c => c.name)).toContain('plan');
  });

  it('getStatus returns ide and adapter info', () => {
    const core = new ForgePluginCore({
      adapters: new Map([['echo', new StubAdapter('echo')]]),
    });
    const status = core.getStatus();
    expect(status.ide).toBeDefined();
    expect(status.availableAdapters).toContain('echo');
    expect(status).not.toHaveProperty('executionMode');
  });
});
