import { describe, it, expect, vi, beforeEach } from 'vitest';
import { EventEmitter } from 'node:events';
import { Readable } from 'node:stream';
import * as child_process from 'node:child_process';
import { ClaudeCodeAdapter } from './claude-code.js';

vi.mock('node:child_process', () => ({
	spawn: vi.fn(),
}));

describe('ClaudeCodeAdapter', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('spawns claude CLI with correct args', async () => {
		const mockSpawn = vi.mocked(child_process.spawn);
		const mockProcess = createMockProcess(
			JSON.stringify({ result: 'implemented the feature', cost_usd: 0.02 }),
			'',
			0,
		);
		mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

		const adapter = new ClaudeCodeAdapter();
		const result = await adapter.execute({
			prompt: 'Fix the bug',
			workingDirectory: '/tmp/project',
			configJson: '{}',
		});

		expect(mockSpawn).toHaveBeenCalledWith(
			'claude',
			expect.arrayContaining(['-p', 'Fix the bug', '--output-format', 'json']),
			expect.objectContaining({ cwd: '/tmp/project' }),
		);
		expect(result.success).toBe(true);
		expect(result.output).toContain('implemented the feature');
	});

	it('returns failure when CLI exits non-zero', async () => {
		const mockSpawn = vi.mocked(child_process.spawn);
		const mockProcess = createMockProcess('', 'claude: command not found', 127);
		mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

		const adapter = new ClaudeCodeAdapter();
		const result = await adapter.execute({
			prompt: 'Do something',
			workingDirectory: '/tmp',
			configJson: '{}',
		});

		expect(result.success).toBe(false);
		expect(result.error).toBeDefined();
	});

	it('returns failure when spawn emits error', async () => {
		const mockSpawn = vi.mocked(child_process.spawn);
		const proc = new EventEmitter() as EventEmitter & {
			stdout: Readable;
			stderr: Readable;
		};
		proc.stdout = Readable.from([]);
		proc.stderr = Readable.from([]);
		mockSpawn.mockReturnValue(proc as child_process.ChildProcess);

		const adapter = new ClaudeCodeAdapter();
		const promise = adapter.execute({
			prompt: 'Hi',
			workingDirectory: '/tmp',
			configJson: '{}',
		});
		setTimeout(() => proc.emit('error', new Error('ENOENT')), 5);
		const result = await promise;
		expect(result.success).toBe(false);
		expect(result.error).toContain('failed to spawn claude');
		expect(result.error).toContain('ENOENT');
	});

	it('uses raw stdout when JSON parse fails on success exit', async () => {
		const mockSpawn = vi.mocked(child_process.spawn);
		const mockProcess = createMockProcess('not-json-output', '', 0);
		mockSpawn.mockReturnValue(mockProcess as child_process.ChildProcess);

		const adapter = new ClaudeCodeAdapter();
		const result = await adapter.execute({
			prompt: 'Run',
			workingDirectory: '/tmp',
			configJson: '{}',
		});
		expect(result.success).toBe(true);
		expect(result.output).toBe('not-json-output');
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
