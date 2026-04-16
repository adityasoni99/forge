import { describe, it, expect, vi, beforeEach } from 'vitest';
import { EventEmitter } from 'node:events';
import { Readable } from 'node:stream';
import * as child_process from 'node:child_process';
import { GooseAdapter } from '../../src/adapters/goose.js';
import type { AgentEvent } from '../../src/adapters/events.js';

vi.mock('node:child_process', () => ({
  spawn: vi.fn(),
}));

async function collectEvents(iter: AsyncIterable<AgentEvent>): Promise<AgentEvent[]> {
  const events: AgentEvent[] = [];
  for await (const event of iter) {
    events.push(event);
  }
  return events;
}

describe('GooseAdapter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('has name "goose"', () => {
    const adapter = new GooseAdapter();
    expect(adapter.name).toBe('goose');
  });

  it('declares capabilities', () => {
    const adapter = new GooseAdapter();
    expect(adapter.getCapabilities()).toEqual({
      streaming: false,
      interrupt: true,
      maxContextTokens: 128000,
      supportsTools: true,
      needsContextReset: false,
    });
  });

  it('spawns goose CLI with correct args and yields done event', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess('goose output', '', 0);
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new GooseAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Fix the bug',
        workingDirectory: '/tmp/project',
        configJson: '{}',
      }),
    );

    expect(mockSpawn).toHaveBeenCalledWith(
      'goose',
      expect.arrayContaining(['run', '--headless', '--text', 'Fix the bug']),
      expect.objectContaining({ cwd: '/tmp/project' }),
    );
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toBe('goose output');
  });

  it('yields error event when CLI exits non-zero', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess('', 'goose: command not found', 127);
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new GooseAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Do something',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('error');
    expect(events[0].content).toContain('goose exited with code 127');
  });

  it('yields error event when spawn emits error', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const proc = new EventEmitter() as EventEmitter & {
      stdout: Readable;
      stderr: Readable;
    };
    proc.stdout = Readable.from([]);
    proc.stderr = Readable.from([]);
    mockSpawn.mockReturnValue(proc as child_process.ChildProcess);

    const adapter = new GooseAdapter();
    setTimeout(() => proc.emit('error', new Error('ENOENT')), 5);
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Hi',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('error');
    expect(events[0].content).toContain('failed to spawn goose');
    expect(events[0].content).toContain('ENOENT');
  });

  it('interrupt kills the current process', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const proc = new EventEmitter() as EventEmitter & {
      stdout: Readable;
      stderr: Readable;
      kill: ReturnType<typeof vi.fn>;
    };
    proc.stdout = Readable.from([]);
    proc.stderr = Readable.from([]);
    proc.kill = vi.fn();
    mockSpawn.mockReturnValue(proc as unknown as child_process.ChildProcess);

    const adapter = new GooseAdapter();
    const iterPromise = collectEvents(
      adapter.execute({
        prompt: 'Long task',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    await new Promise((r) => setTimeout(r, 10));
    await adapter.interrupt();
    expect(proc.kill).toHaveBeenCalledWith('SIGTERM');

    proc.emit('close', 1);
    await iterPromise;
  });
});

function createMockProcess(
  stdout: string,
  stderr: string,
  exitCode: number,
): EventEmitter & { stdout: Readable; stderr: Readable } {
  const proc = new EventEmitter() as EventEmitter & {
    stdout: Readable;
    stderr: Readable;
  };
  proc.stdout = Readable.from([stdout]);
  proc.stderr = Readable.from([stderr]);

  setTimeout(() => proc.emit('close', exitCode), 10);
  return proc;
}
