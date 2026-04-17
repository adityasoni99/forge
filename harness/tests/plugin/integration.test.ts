import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ForgePluginCore } from '../../src/plugin/core.js';
import { EchoAdapter } from '../../src/adapters/echo.js';

describe('plugin integration', () => {
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

  function createCore(): ForgePluginCore {
    return new ForgePluginCore({
      adapters: new Map([['echo', new EchoAdapter()]]),
      config: { defaultAdapter: 'echo' },
    });
  }

  it('forge_run executes with echo adapter', async () => {
    const core = createCore();
    const result = await core.executeCommand('run', 'Add user authentication');
    expect(result.success).toBe(true);
    expect(result.mode).toBe('direct');
    expect(result.output).toContain('echo');
    expect(result.output).toContain('Add user authentication');
  });

  it('forge_fix executes with file context', async () => {
    const core = createCore();
    const result = await core.executeCommand('fix', 'NullPointerException in auth', {
      filePath: 'src/auth.ts',
      errorOutput: 'TypeError: Cannot read property of null',
    });
    expect(result.success).toBe(true);
    expect(result.output).toContain('echo');
  });

  it('forge_plan executes with plan-only prompt', async () => {
    const core = createCore();
    const result = await core.executeCommand('plan', 'Redesign the API layer');
    expect(result.success).toBe(true);
    expect(result.output).toContain('plan');
  });

  it('forge_status returns plugin info', () => {
    const core = createCore();
    const status = core.getStatus();
    expect(status.availableAdapters).toContain('echo');
    expect(status).not.toHaveProperty('executionMode');
  });

  it('createForgeTools returns valid tool definitions', async () => {
    const { createForgeTools } = await import('../../src/mcp-server.js');
    const tools = createForgeTools();
    expect(tools.length).toBe(4);
    for (const tool of tools) {
      expect(tool.name).toMatch(/^forge_/);
      expect(tool.description.length).toBeGreaterThan(0);
    }
  });
});
