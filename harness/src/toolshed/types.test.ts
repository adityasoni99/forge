import { describe, it, expect } from 'vitest';
import { createPermissionContext } from './types.js';
import type { Tool, DenyRule } from './types.js';

describe('Tool types', () => {
  it('creates permission context with defaults', () => {
    const ctx = createPermissionContext();
    expect(ctx.denyRules).toEqual([]);
    expect(ctx.maxTools).toBe(15);
  });

  it('creates permission context with custom values', () => {
    const rules: DenyRule[] = [{ toolName: 'shell', reason: 'unsafe' }];
    const ctx = createPermissionContext(rules, 10);
    expect(ctx.denyRules).toEqual(rules);
    expect(ctx.maxTools).toBe(10);
  });

  it('tool object has correct shape', () => {
    const tool: Tool = {
      name: 'read_file',
      description: 'Read a file',
      source: 'builtin',
    };
    expect(tool.source).toBe('builtin');
  });
});
