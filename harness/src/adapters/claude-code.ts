import { spawn, type ChildProcess } from 'node:child_process';
import type { AgentAdapter, AgentAdapterRequest, AgentCapabilities } from './types.js';
import type { AgentEvent } from './events.js';

export class ClaudeCodeAdapter implements AgentAdapter {
  readonly name = 'claude';
  private readonly claudeBinary: string;
  private currentProcess: ChildProcess | null = null;

  constructor(claudeBinary = 'claude') {
    this.claudeBinary = claudeBinary;
  }

  async *execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent> {
    const args = ['-p', req.prompt, '--output-format', 'json'];
    const result = await this.runProcess(args, req.workingDirectory);

    if (result.success) {
      yield { type: 'done', content: result.output };
    } else {
      yield { type: 'error', content: result.error };
    }
  }

  getCapabilities(): AgentCapabilities {
    return {
      streaming: true,
      interrupt: true,
      maxContextTokens: 200000,
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
      const proc = spawn(this.claudeBinary, args, {
        cwd,
        env: { ...process.env },
      });
      this.currentProcess = proc;

      let stdout = '';
      let stderr = '';

      proc.stdout?.on('data', (chunk: Buffer) => { stdout += chunk.toString(); });
      proc.stderr?.on('data', (chunk: Buffer) => { stderr += chunk.toString(); });

      proc.on('close', (code) => {
        this.currentProcess = null;
        if (code !== 0) {
          resolve({ output: '', success: false, error: `claude exited with code ${code}: ${stderr || stdout}` });
          return;
        }
        try {
          const parsed = JSON.parse(stdout) as { result?: string };
          resolve({ output: parsed.result ?? stdout, success: true, error: '' });
        } catch {
          resolve({ output: stdout, success: true, error: '' });
        }
      });

      proc.on('error', (err) => {
        this.currentProcess = null;
        resolve({ output: '', success: false, error: `failed to spawn claude: ${err.message}` });
      });
    });
  }
}
