import { describe, it, expect } from 'vitest';
import { EchoAdapter } from './echo.js';

describe('EchoAdapter', () => {
  it('returns the prompt as output', async () => {
    const adapter = new EchoAdapter();
    const result = await adapter.execute({
      prompt: 'Write tests for the auth module',
      workingDirectory: '/tmp/project',
      configJson: '{}',
    });
    expect(result.output).toContain('Write tests for the auth module');
    expect(result.success).toBe(true);
  });

  it('includes config in output when present', async () => {
    const adapter = new EchoAdapter();
    const result = await adapter.execute({
      prompt: 'Plan it',
      workingDirectory: '/tmp',
      configJson: '{"model":"claude-sonnet-4-20250514"}',
    });
    expect(result.output).toContain('Plan it');
    expect(result.success).toBe(true);
  });

  it('exposes optional capabilities for adapter routing', () => {
    const adapter = new EchoAdapter();
    expect(adapter.getCapabilities?.()).toEqual({ streaming: false, interrupt: false });
  });
});
