import { describe, it, expect } from 'vitest';
import { AgentService } from './agent-service.js';
import { EchoAdapter } from './adapters/echo.js';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as os from 'node:os';

describe('AgentService', () => {
  it('enriches prompt with context and calls adapter', async () => {
    const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-svc-'));
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), '# Rules\nAlways use TDD.');

    try {
      const service = new AgentService(new EchoAdapter());
      const response = await service.executeAgent({
        prompt: 'Fix the auth module',
        config_json: '{}',
        working_directory: tmpDir,
        blueprint_name: 'bug-fix',
        node_id: 'implement',
        run_id: 'run-1',
      });

      expect(response.success).toBe(true);
      // The echo adapter receives the composed prompt (with context)
      expect(response.output).toContain('Always use TDD');
      expect(response.output).toContain('Fix the auth module');
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it('handles adapter failure gracefully', async () => {
    const failAdapter = {
      async execute() {
        return { output: '', success: false, error: 'agent crashed' };
      },
    };
    const service = new AgentService(failAdapter);
    const response = await service.executeAgent({
      prompt: 'Do something',
      config_json: '{}',
      working_directory: '/tmp',
      blueprint_name: 'test',
      node_id: 'a',
      run_id: 'r1',
    });

    expect(response.success).toBe(false);
    expect(response.error).toContain('agent crashed');
  });

  it('returns structured error when adapter throws', async () => {
    const throwingAdapter = {
      async execute() {
        throw new Error('boom');
      },
    };
    const service = new AgentService(throwingAdapter);
    const response = await service.executeAgent({
      prompt: 'x',
      config_json: '{}',
      working_directory: '/tmp',
      blueprint_name: 'b',
      node_id: 'n',
      run_id: 'r',
    });
    expect(response.success).toBe(false);
    expect(response.error).toContain('boom');
  });

  it('stringifies non-Error throws in catch path', async () => {
    const throwingAdapter = {
      async execute() {
        throw 'string-throw';
      },
    };
    const service = new AgentService(throwingAdapter);
    const response = await service.executeAgent({
      prompt: 'x',
      config_json: '{}',
      working_directory: '/tmp',
      blueprint_name: 'b',
      node_id: 'n',
      run_id: 'r',
    });
    expect(response.success).toBe(false);
    expect(response.error).toContain('string-throw');
  });
});
