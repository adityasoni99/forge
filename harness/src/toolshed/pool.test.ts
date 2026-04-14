import { describe, it, expect } from 'vitest';
import { assembleToolPool } from './pool.js';
import type { Tool, PermissionContext } from './types.js';
import { createPermissionContext } from './types.js';

function makeTool(name: string, source: Tool['source'] = 'builtin'): Tool {
  return { name, description: `${name} tool`, source };
}

describe('assembleToolPool', () => {
  it('merges builtin and extension tools', () => {
    const builtins = [makeTool('read'), makeTool('write')];
    const extensions = [makeTool('search', 'extension')];
    const ctx = createPermissionContext();
    const result = assembleToolPool(builtins, extensions, ctx);
    expect(result).toHaveLength(3);
  });

  it('filters denied tools', () => {
    const builtins = [makeTool('read'), makeTool('shell')];
    const extensions: Tool[] = [];
    const ctx = createPermissionContext([{ toolName: 'shell', reason: 'unsafe' }]);
    const result = assembleToolPool(builtins, extensions, ctx);
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('read');
  });

  it('deduplicates: builtin wins on name clash', () => {
    const builtins = [makeTool('read', 'builtin')];
    const extensions = [makeTool('read', 'extension')];
    const ctx = createPermissionContext();
    const result = assembleToolPool(builtins, extensions, ctx);
    expect(result).toHaveLength(1);
    expect(result[0].source).toBe('builtin');
  });

  it('sorts alphabetically for cache consistency', () => {
    const builtins = [makeTool('write'), makeTool('read'), makeTool('grep')];
    const ctx = createPermissionContext();
    const result = assembleToolPool(builtins, [], ctx);
    expect(result.map((t) => t.name)).toEqual(['grep', 'read', 'write']);
  });

  it('caps at maxTools', () => {
    const builtins = Array.from({ length: 20 }, (_, i) => makeTool(`tool-${String(i).padStart(2, '0')}`));
    const ctx = createPermissionContext([], 5);
    const result = assembleToolPool(builtins, [], ctx);
    expect(result).toHaveLength(5);
  });

  it('returns empty when all tools denied', () => {
    const builtins = [makeTool('shell')];
    const ctx = createPermissionContext([{ toolName: 'shell', reason: 'no' }]);
    const result = assembleToolPool(builtins, [], ctx);
    expect(result).toEqual([]);
  });
});
