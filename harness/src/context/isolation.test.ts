import { describe, it, expect } from 'vitest';
import { SubagentContext, SubagentType } from './isolation.js';

describe('SubagentContext', () => {
  it('creates forked context with isolated budget', () => {
    const parent = {
      maxTokens: 8000,
      rules: [{ name: 'rule1', content: 'Always test' }],
      tools: ['read_file', 'write_file', 'shell'],
      fileCache: new Map([['a.ts', 'content']]),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Explore,
      maxTokens: 4000,
    });

    expect(child.maxTokens).toBe(4000);
    expect(child.rules).toEqual(parent.rules);
    expect(child.fileCache.get('a.ts')).toBe('content');
  });

  it('explore agents exclude write tools', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: ['read_file', 'write_file', 'shell', 'grep'],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Explore,
      maxTokens: 4000,
    });

    expect(child.tools).toContain('read_file');
    expect(child.tools).toContain('grep');
    expect(child.tools).not.toContain('write_file');
    expect(child.tools).not.toContain('shell');
  });

  it('implement agents keep all tools', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: ['read_file', 'write_file', 'shell'],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Implement,
      maxTokens: 6000,
    });

    expect(child.tools).toEqual(['read_file', 'write_file', 'shell']);
  });

  it('child cannot mutate parent file cache', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: [],
      fileCache: new Map([['a.ts', 'original']]),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Implement,
      maxTokens: 4000,
    });

    child.fileCache.set('b.ts', 'new file');
    expect(parent.fileCache.has('b.ts')).toBe(false);
  });

  it('review agents exclude shell but keep read/write', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: ['read_file', 'write_file', 'shell', 'grep'],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Review,
      maxTokens: 4000,
    });

    expect(child.tools).toContain('read_file');
    expect(child.tools).toContain('write_file');
    expect(child.tools).not.toContain('shell');
  });

  it('defaults maxTokens to half parent budget', () => {
    const parent = {
      maxTokens: 8000,
      rules: [],
      tools: [],
      fileCache: new Map(),
    };

    const child = SubagentContext.fork(parent, {
      type: SubagentType.Implement,
    });

    expect(child.maxTokens).toBe(4000);
  });
});
