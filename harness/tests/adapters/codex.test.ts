import { describe, it, expect, vi, beforeEach } from 'vitest';
import { EventEmitter } from 'node:events';
import { Readable } from 'node:stream';
import * as child_process from 'node:child_process';
import { CodexAdapter } from '../../src/adapters/codex.js';
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

describe('CodexAdapter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('has name "codex"', () => {
    const adapter = new CodexAdapter();
    expect(adapter.name).toBe('codex');
  });

  it('declares capabilities', () => {
    const adapter = new CodexAdapter();
    expect(adapter.getCapabilities()).toEqual({
      streaming: false,
      interrupt: true,
      maxContextTokens: 200000,
      supportsTools: true,
      needsContextReset: false,
    });
  });

  it('spawns codex CLI with correct args and yields done event', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess('codex output', '', 0);
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new CodexAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Implement feature',
        workingDirectory: '/tmp/project',
        configJson: '{}',
      }),
    );

    expect(mockSpawn).toHaveBeenCalledWith(
      'codex',
      expect.arrayContaining(['--quiet', '--approval-mode', 'full-auto', 'Implement feature']),
      expect.objectContaining({ cwd: '/tmp/project' }),
    );
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toBe('codex output');
  });

  it('yields error event when CLI exits non-zero', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess('', 'codex: command not found', 127);
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new CodexAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Do something',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('error');
    expect(events[0].content).toContain('codex exited with code 127');
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

    const adapter = new CodexAdapter();
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
    expect(events[0].content).toContain('failed to spawn codex');
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

    const adapter = new CodexAdapter();
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
