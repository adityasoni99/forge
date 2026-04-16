import { describe, it, expect } from 'vitest';
import { composePromptStack, PromptLayer, type PromptLayerEntry } from '../../src/context/prompt-stack.js';

describe('composePromptStack', () => {
  const baseLayers: PromptLayerEntry[] = [
    { layer: PromptLayer.DefaultBaseline, label: 'default', content: 'You are Forge.' },
    { layer: PromptLayer.ProjectRules, label: 'AGENTS.md', content: 'Follow TDD.' },
    { layer: PromptLayer.Override, label: 'org', content: 'Never leak secrets.' },
  ];

  it('orders layers by priority (Override first)', () => {
    const result = composePromptStack(baseLayers, 'Fix the bug', { maxTokens: 10000 });
    const overrideIdx = result.indexOf('Never leak secrets.');
    const defaultIdx = result.indexOf('You are Forge.');
    expect(overrideIdx).toBeLessThan(defaultIdx);
  });

  it('includes task prompt', () => {
    const result = composePromptStack(baseLayers, 'Fix the bug', { maxTokens: 10000 });
    expect(result).toContain('Fix the bug');
  });

  it('truncates from bottom when over budget', () => {
    const result = composePromptStack(baseLayers, 'task', { maxTokens: 30 });
    expect(result).toContain('Never leak secrets.');
    expect(result).toContain('task');
  });

  it('never truncates Override layer', () => {
    const layers: PromptLayerEntry[] = [
      { layer: PromptLayer.Override, label: 'org', content: 'A'.repeat(100) },
      { layer: PromptLayer.DefaultBaseline, label: 'default', content: 'B'.repeat(100) },
    ];
    const result = composePromptStack(layers, 'task', { maxTokens: 40 });
    expect(result).toContain('A'.repeat(100));
  });

  it('returns task even with zero-budget layers', () => {
    const result = composePromptStack([], 'just the task', { maxTokens: 100 });
    expect(result).toContain('just the task');
  });
});
