import { describe, it, expect } from 'vitest';
import { deriveRuleFromFailure, parseRuleResponse } from '../../src/memory/failure-to-rule.js';
import type { SessionEvent } from '../../src/memory/types.js';

describe('parseRuleResponse', () => {
  it('parses structured rule from agent output', () => {
    const output = `CATEGORY: testing
NAME: always-run-tests-before-commit
BODY:
Always run the test suite before committing. Use \`make test\` or the project's
configured test command. Never commit code that has failing tests.`;

    const rule = parseRuleResponse(output, 'run-123', 'test failed');
    expect(rule).not.toBeNull();
    expect(rule!.category).toBe('testing');
    expect(rule!.name).toBe('always-run-tests-before-commit');
    expect(rule!.body).toContain('Always run the test suite');
    expect(rule!.sourceRunID).toBe('run-123');
  });

  it('returns null for unparseable output', () => {
    const rule = parseRuleResponse('random text', 'run-1', 'error');
    expect(rule).toBeNull();
  });
});

describe('deriveRuleFromFailure', () => {
  it('calls executor with failure context and returns derived rule', async () => {
    const events: SessionEvent[] = [
      { id: '1', timestamp: '', runID: 'run-1', type: 'adapter_called', data: { adapter: 'claude' } },
      { id: '2', timestamp: '', runID: 'run-1', type: 'error', data: { message: 'lint failed: unused import' } },
    ];

    const mockExecutor = {
      execute: async (_ctx: any, _prompt: string, _config: any) => {
        return `CATEGORY: linting
NAME: remove-unused-imports
BODY:
Always remove unused imports before committing. Run the linter after every change.`;
      },
    };

    const rule = await deriveRuleFromFailure(events, mockExecutor);
    expect(rule).not.toBeNull();
    expect(rule!.category).toBe('linting');
    expect(rule!.name).toBe('remove-unused-imports');
  });

  it('returns null when executor fails', async () => {
    const events: SessionEvent[] = [
      { id: '1', timestamp: '', runID: 'run-1', type: 'error', data: { message: 'crash' } },
    ];
    const mockExecutor = {
      execute: async () => { throw new Error('executor down'); },
    };
    const rule = await deriveRuleFromFailure(events, mockExecutor);
    expect(rule).toBeNull();
  });
});
