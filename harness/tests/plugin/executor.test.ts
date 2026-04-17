import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ForgeExecutor } from '../../src/plugin/executor.js';
import type { PluginConfig, ExecutionResult } from '../../src/plugin/types.js';
import type { AgentAdapter, AgentAdapterRequest, AgentCapabilities } from '../../src/adapters/types.js';
import type { AgentEvent } from '../../src/adapters/events.js';

class MockAdapter implements AgentAdapter {
  readonly name = 'mock';
  lastPrompt = '';

  async *execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent> {
    this.lastPrompt = req.prompt;
    yield { type: 'done', content: `mock-output: ${req.prompt.slice(0, 30)}` };
  }

  getCapabilities(): AgentCapabilities {
    return { streaming: false, interrupt: false, maxContextTokens: 200000, supportsTools: false, needsContextReset: false };
  }

  async interrupt(): Promise<void> {}
}

describe('ForgeExecutor', () => {
  let mockAdapter: MockAdapter;
  let executor: ForgeExecutor;

  beforeEach(() => {
    mockAdapter = new MockAdapter();
    executor = new ForgeExecutor({
      adapters: new Map([['mock', mockAdapter]]),
      config: { defaultAdapter: 'mock', forgeBinaryPath: 'forge' },
    });
  });

  it('executes in direct mode via AgentService', async () => {
    const result = await executor.execute({
      prompt: 'Fix the login bug',
      workingDirectory: process.cwd(),
      adapter: 'mock',
    });
    expect(result.success).toBe(true);
    expect(result.mode).toBe('direct');
    expect(result.output).toContain('mock-output');
  });

  it('uses specified adapter', async () => {
    const result = await executor.execute({
      prompt: 'Test prompt',
      workingDirectory: process.cwd(),
      adapter: 'mock',
    });
    expect(result.adapter).toBe('mock');
  });

  it('reports error for unknown adapter', async () => {
    const result = await executor.execute({
      prompt: 'Test prompt',
      workingDirectory: process.cwd(),
      adapter: 'nonexistent',
    });
    expect(result.success).toBe(false);
    expect(result.error).toContain('unknown adapter');
  });

  it('records durationMs', async () => {
    const result = await executor.execute({
      prompt: 'Test prompt',
      workingDirectory: process.cwd(),
      adapter: 'mock',
    });
    expect(result.durationMs).toBeGreaterThanOrEqual(0);
  });
});
