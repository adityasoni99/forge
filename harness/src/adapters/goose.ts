import { spawn, type ChildProcess } from 'node:child_process';
import type { AgentAdapter, AgentAdapterRequest, AgentCapabilities } from './types.js';
import type { AgentEvent } from './events.js';

export class GooseAdapter implements AgentAdapter {
  readonly name = 'goose';
  private readonly binary: string;
  private currentProcess: ChildProcess | null = null;

  constructor(binary = 'goose') {
    this.binary = binary;
  }

  async *execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent> {
    const result = await this.runProcess(
      ['run', '--headless', '--text', req.prompt],
      req.workingDirectory,
    );
    if (result.success) {
      yield { type: 'done', content: result.output };
    } else {
      yield { type: 'error', content: result.error };
    }
  }

  getCapabilities(): AgentCapabilities {
    return {
      streaming: false,
      interrupt: true,
      maxContextTokens: 128000,
      supportsTools: true,
      needsContextReset: false,
    };
  }

  async interrupt(): Promise<void> {
    if (this.currentProcess) {
      this.currentProcess.kill('SIGTERM');
      this.currentProcess = null;
    }
  }

  private runProcess(args: string[], cwd: string): Promise<{ output: string; success: boolean; error: string }> {
    return new Promise((resolve) => {
      const proc = spawn(this.binary, args, { cwd, env: { ...process.env } });
      this.currentProcess = proc;
      let stdout = '';
      let stderr = '';
      proc.stdout?.on('data', (chunk: Buffer) => { stdout += chunk.toString(); });
      proc.stderr?.on('data', (chunk: Buffer) => { stderr += chunk.toString(); });
      proc.on('close', (code) => {
        this.currentProcess = null;
        if (code !== 0) {
          resolve({ output: '', success: false, error: `goose exited with code ${code}: ${stderr || stdout}` });
        } else {
          resolve({ output: stdout.trim(), success: true, error: '' });
        }
      });
      proc.on('error', (err) => {
        this.currentProcess = null;
        resolve({ output: '', success: false, error: `failed to spawn goose: ${err.message}` });
      });
    });
  }
}
