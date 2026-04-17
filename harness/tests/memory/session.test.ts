import { describe, it, expect } from 'vitest';
import * as fs from 'node:fs/promises';
import { SessionEventEmitter } from '../../src/memory/session.js';

describe('SessionEventEmitter', () => {
  it('emits events to JSONL file', async () => {
    const dir = await fs.mkdtemp('/tmp/forge-session-');
    const emitter = new SessionEventEmitter(dir);

    await emitter.emit('run-1', {
      type: 'prompt_composed',
      data: { tokens: 500 },
    });
    await emitter.emit('run-1', {
      type: 'adapter_called',
      data: { adapter: 'claude' },
    });

    const events = await emitter.getEvents('run-1');
    expect(events).toHaveLength(2);
    expect(events[0].type).toBe('prompt_composed');
    expect(events[1].type).toBe('adapter_called');

    await fs.rm(dir, { recursive: true });
  });

  it('returns empty array for unknown run', async () => {
    const dir = await fs.mkdtemp('/tmp/forge-session-');
    const emitter = new SessionEventEmitter(dir);
    const events = await emitter.getEvents('nonexistent');
    expect(events).toHaveLength(0);
    await fs.rm(dir, { recursive: true });
  });

  it('isolates events by runID', async () => {
    const dir = await fs.mkdtemp('/tmp/forge-session-');
    const emitter = new SessionEventEmitter(dir);

    await emitter.emit('run-a', { type: 'run_complete' });
    await emitter.emit('run-b', { type: 'error', data: { msg: 'boom' } });

    const evA = await emitter.getEvents('run-a');
    const evB = await emitter.getEvents('run-b');
    expect(evA).toHaveLength(1);
    expect(evB).toHaveLength(1);
    expect(evA[0].type).toBe('run_complete');
    expect(evB[0].type).toBe('error');

    await fs.rm(dir, { recursive: true });
  });

  it('getEvents with afterID returns only subsequent events', async () => {
    const dir = await fs.mkdtemp('/tmp/forge-session-');
    const emitter = new SessionEventEmitter(dir);

    await emitter.emit('run-1', { type: 'prompt_composed' });
    await emitter.emit('run-1', { type: 'adapter_called' });
    await emitter.emit('run-1', { type: 'adapter_result' });

    const all = await emitter.getEvents('run-1');
    const afterFirst = await emitter.getEvents('run-1', { afterID: all[0].id });
    expect(afterFirst).toHaveLength(2);
    expect(afterFirst[0].type).toBe('adapter_called');

    await fs.rm(dir, { recursive: true });
  });
});
