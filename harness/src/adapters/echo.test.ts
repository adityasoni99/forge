import { describe, it, expect } from 'vitest';
import { EchoAdapter } from './echo.js';
import type { AgentEvent } from './events.js';

async function collectEvents(iter: AsyncIterable<AgentEvent>): Promise<AgentEvent[]> {
  const events: AgentEvent[] = [];
  for await (const event of iter) {
    events.push(event);
  }
  return events;
}

describe('EchoAdapter', () => {
  it('yields a done event containing prompt and workingDir', async () => {
    const adapter = new EchoAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Write tests for the auth module',
        workingDirectory: '/tmp/project',
        configJson: '{}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toContain('Write tests for the auth module');
    expect(events[0].content).toContain('/tmp/project');
  });

  it('includes prompt in output for different inputs', async () => {
    const adapter = new EchoAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Plan it',
        workingDirectory: '/tmp',
        configJson: '{"model":"claude-sonnet-4-20250514"}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toContain('Plan it');
  });

  it('has name property set to echo', () => {
    const adapter = new EchoAdapter();
    expect(adapter.name).toBe('echo');
  });

  it('returns full AgentCapabilities from getCapabilities()', () => {
    const adapter = new EchoAdapter();
    expect(adapter.getCapabilities()).toEqual({
      streaming: false,
      interrupt: false,
      maxContextTokens: 200000,
      supportsTools: false,
      needsContextReset: false,
    });
  });

  it('interrupt() resolves without error', async () => {
    const adapter = new EchoAdapter();
    await expect(adapter.interrupt()).resolves.toBeUndefined();
  });
});
