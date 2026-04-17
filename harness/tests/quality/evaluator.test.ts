import { describe, it, expect } from 'vitest';
import { SkepticalEvaluator } from '../../src/quality/evaluator.js';

describe('SkepticalEvaluator', () => {
  it('returns pass when score meets threshold', async () => {
    const mockExecutor = {
      execute: async () => 'SCORE: 85\nFEEDBACK: Good implementation with minor style issues.',
    };
    const evaluator = new SkepticalEvaluator(mockExecutor, {
      criteria: ['correctness', 'maintainability'],
      maxRetries: 2,
    });
    const result = await evaluator.evaluate('function add(a,b) { return a+b; }', 70);
    expect(result.passed).toBe(true);
    expect(result.score).toBe(85);
  });

  it('returns fail when score below threshold', async () => {
    const mockExecutor = {
      execute: async () => 'SCORE: 40\nFEEDBACK: Missing error handling and tests.',
    };
    const evaluator = new SkepticalEvaluator(mockExecutor, {
      criteria: ['correctness'],
      maxRetries: 2,
    });
    const result = await evaluator.evaluate('broken code', 70);
    expect(result.passed).toBe(false);
    expect(result.score).toBe(40);
    expect(result.feedback).toContain('Missing error handling');
  });
});
