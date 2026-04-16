import { describe, it, expect, vi, beforeEach } from 'vitest';
import { EventEmitter } from 'node:events';
import { Readable } from 'node:stream';
import * as child_process from 'node:child_process';
import { ClaudeCodeAdapter } from './claude-code.js';
import type { AgentEvent } from './events.js';

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

describe('ClaudeCodeAdapter', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('spawns claude CLI with correct args and yields done event', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess(
      JSON.stringify({ result: 'implemented the feature', cost_usd: 0.02 }),
      '',
      0,
    );
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new ClaudeCodeAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Fix the bug',
        workingDirectory: '/tmp/project',
        configJson: '{}',
      }),
    );

    expect(mockSpawn).toHaveBeenCalledWith(
      'claude',
      expect.arrayContaining(['-p', 'Fix the bug', '--output-format', 'json']),
      expect.objectContaining({ cwd: '/tmp/project' }),
    );
    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toContain('implemented the feature');
  });

  it('yields error event when CLI exits non-zero', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess('', 'claude: command not found', 127);
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new ClaudeCodeAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Do something',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('error');
    expect(events[0].content).toContain('claude exited with code 127');
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

    const adapter = new ClaudeCodeAdapter();
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
    expect(events[0].content).toContain('failed to spawn claude');
    expect(events[0].content).toContain('ENOENT');
  });

  it('yields done with raw stdout when JSON parse fails on success exit', async () => {
    const mockSpawn = vi.mocked(child_process.spawn);
    const mockProcess = createMockProcess('not-json-output', '', 0);
    mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

    const adapter = new ClaudeCodeAdapter();
    const events = await collectEvents(
      adapter.execute({
        prompt: 'Run',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe('done');
    expect(events[0].content).toBe('not-json-output');
  });

  it('has name property set to claude', () => {
    const adapter = new ClaudeCodeAdapter();
    expect(adapter.name).toBe('claude');
  });

  it('returns full AgentCapabilities from getCapabilities()', () => {
    const adapter = new ClaudeCodeAdapter();
    expect(adapter.getCapabilities()).toEqual({
      streaming: true,
      interrupt: true,
      maxContextTokens: 200000,
      supportsTools: true,
      needsContextReset: false,
    });
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

    const adapter = new ClaudeCodeAdapter();
    const iterPromise = collectEvents(
      adapter.execute({
        prompt: 'Long task',
        workingDirectory: '/tmp',
        configJson: '{}',
      }),
    );

    // Give the execute call time to spawn the process
    await new Promise((r) => setTimeout(r, 10));
    await adapter.interrupt();
    expect(proc.kill).toHaveBeenCalledWith('SIGTERM');

    // Clean up: emit close so the promise resolves
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
