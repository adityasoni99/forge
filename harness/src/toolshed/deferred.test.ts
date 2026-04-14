import { describe, it, expect } from 'vitest';
import { DeferredToolLoader } from './deferred.js';
import type { Tool } from './types.js';

function makeTool(name: string, descLen = 50): Tool {
  return {
    name,
    description: 'x'.repeat(descLen),
    source: 'builtin',
  };
}

describe('DeferredToolLoader', () => {
  it('keeps all tools inline when under budget', () => {
    const tools = [makeTool('a', 10), makeTool('b', 10)];
    const loader = new DeferredToolLoader(1000);
    const { inline, deferred } = loader.partition(tools);
    expect(inline).toHaveLength(2);
    expect(deferred).toHaveLength(0);
  });

  it('defers tools exceeding budget', () => {
    const tools = [
      makeTool('a', 100),
      makeTool('b', 100),
      makeTool('c', 100),
    ];
    const loader = new DeferredToolLoader(60);
    const { inline, deferred } = loader.partition(tools);
    expect(inline.length).toBeGreaterThanOrEqual(1);
    expect(deferred.length).toBeGreaterThanOrEqual(1);
    expect(inline.length + deferred.length).toBe(3);
  });

  it('returns all deferred when budget is zero', () => {
    const tools = [makeTool('a', 100)];
    const loader = new DeferredToolLoader(0);
    const { inline, deferred } = loader.partition(tools);
    expect(inline).toHaveLength(0);
    expect(deferred).toHaveLength(1);
  });

  it('handles empty tool list', () => {
    const loader = new DeferredToolLoader(1000);
    const { inline, deferred } = loader.partition([]);
    expect(inline).toEqual([]);
    expect(deferred).toEqual([]);
  });

  it('estimateToolTokens returns positive number', () => {
    const loader = new DeferredToolLoader(1000);
    const tokens = loader.estimateToolTokens(makeTool('test', 100));
    expect(tokens).toBeGreaterThan(0);
  });
});
