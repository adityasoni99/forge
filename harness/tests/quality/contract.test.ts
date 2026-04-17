import { describe, it, expect } from 'vitest';
import { type SprintContract, evaluateAgainstContract } from '../../src/quality/contract.js';

describe('evaluateAgainstContract', () => {
  const contract: SprintContract = {
    doneCriteria: ['All tests pass', 'No linter errors'],
    verificationSteps: ['Run test suite', 'Run linter'],
    acceptanceThresholds: { testCoverage: 80, lintScore: 95 },
  };

  it('passes when all criteria met', () => {
    const result = evaluateAgainstContract(
      { testCoverage: 90, lintScore: 98 },
      contract,
    );
    expect(result.passed).toBe(true);
  });

  it('fails when threshold not met', () => {
    const result = evaluateAgainstContract(
      { testCoverage: 70, lintScore: 98 },
      contract,
    );
    expect(result.passed).toBe(false);
    expect(result.feedback).toContain('testCoverage');
  });

  it('fails when metric missing', () => {
    const result = evaluateAgainstContract(
      { testCoverage: 90 },
      contract,
    );
    expect(result.passed).toBe(false);
  });
});
