import { describe, it, expect } from 'vitest';
import { SyncAdapterWrapper } from '../../src/adapters/sync-wrapper.js';
import type { SyncAgentAdapter, AgentAdapterRequest } from '../../src/adapters/types.js';

const mockSync: SyncAgentAdapter = {
  async execute(req: AgentAdapterRequest) {
    return { output: `echo: ${req.prompt}`, success: true };
  },
  getCapabilities() {
    return { streaming: false, interrupt: false };
  },
};

describe('SyncAdapterWrapper', () => {
  it('yields a done event with output', async () => {
    const wrapper = new SyncAdapterWrapper('test', mockSync);
    const events = [];
    for await (const ev of wrapper.execute({ prompt: 'hello', workingDirectory: '/tmp', configJson: '{}' })) {
      events.push(ev);
    }
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toBe('echo: hello');
  });

  it('yields error event on failure', async () => {
    const failSync: SyncAgentAdapter = {
      async execute() {
        return { output: '', success: false, error: 'boom' };
      },
    };
    const wrapper = new SyncAdapterWrapper('fail', failSync);
    const events = [];
    for await (const ev of wrapper.execute({ prompt: 'x', workingDirectory: '/tmp', configJson: '{}' })) {
      events.push(ev);
    }
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('error');
    expect(events[0].content).toBe('boom');
  });

  it('exposes capabilities with defaults', () => {
    const wrapper = new SyncAdapterWrapper('test', mockSync);
    const caps = wrapper.getCapabilities();
    expect(caps.streaming).toBe(false);
    expect(caps.interrupt).toBe(false);
    expect(caps.maxContextTokens).toBeGreaterThan(0);
    expect(caps.needsContextReset).toBe(false);
  });

  it('has a name property', () => {
    const wrapper = new SyncAdapterWrapper('my-adapter', mockSync);
    expect(wrapper.name).toBe('my-adapter');
  });
});
