import { spawn } from 'node:child_process';
import type { AgentAdapter, AgentAdapterRequest, AgentAdapterResponse } from './types.js';

export class ClaudeCodeAdapter implements AgentAdapter {
	private readonly claudeBinary: string;

	constructor(claudeBinary = 'claude') {
		this.claudeBinary = claudeBinary;
	}

	async execute(req: AgentAdapterRequest): Promise<AgentAdapterResponse> {
		const args = ['-p', req.prompt, '--output-format', 'json'];

		return new Promise((resolve) => {
			const proc = spawn(this.claudeBinary, args, {
				cwd: req.workingDirectory,
				env: { ...process.env },
			});

			let stdout = '';
			let stderr = '';

			proc.stdout?.on('data', (chunk: Buffer) => {
				stdout += chunk.toString();
			});
			proc.stderr?.on('data', (chunk: Buffer) => {
				stderr += chunk.toString();
			});

			proc.on('close', (code) => {
				if (code !== 0) {
					resolve({
						output: '',
						success: false,
						error: `claude exited with code ${code}: ${stderr || stdout}`,
					});
					return;
				}

				try {
					const parsed = JSON.parse(stdout) as { result?: string };
					resolve({
						output: parsed.result ?? stdout,
						success: true,
					});
				} catch {
					resolve({ output: stdout, success: true });
				}
			});

			proc.on('error', (err) => {
				resolve({
					output: '',
					success: false,
					error: `failed to spawn claude: ${err.message}`,
				});
			});
		});
	}
}
