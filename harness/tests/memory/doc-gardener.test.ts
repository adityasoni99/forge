import { describe, it, expect } from 'vitest';
import { findStaleDocCandidates, type GitLogEntry } from '../../src/memory/doc-gardener.js';

describe('findStaleDocCandidates', () => {
  it('flags docs when related code changed more recently', () => {
    const codeChanges: GitLogEntry[] = [
      { path: 'src/auth/login.ts', date: '2026-04-10' },
      { path: 'src/auth/signup.ts', date: '2026-04-08' },
    ];
    const docFiles: GitLogEntry[] = [
      { path: 'docs/auth.md', date: '2026-03-01' },
    ];

    const stale = findStaleDocCandidates(codeChanges, docFiles, {
      codeToDocMapping: { 'src/auth/': 'docs/auth.md' },
      staleDaysThreshold: 7,
      referenceDate: '2026-04-15',
    });

    expect(stale).toHaveLength(1);
    expect(stale[0].docPath).toBe('docs/auth.md');
    expect(stale[0].staleDays).toBeGreaterThan(30);
  });

  it('does not flag recently updated docs', () => {
    const codeChanges: GitLogEntry[] = [
      { path: 'src/auth/login.ts', date: '2026-04-10' },
    ];
    const docFiles: GitLogEntry[] = [
      { path: 'docs/auth.md', date: '2026-04-12' },
    ];

    const stale = findStaleDocCandidates(codeChanges, docFiles, {
      codeToDocMapping: { 'src/auth/': 'docs/auth.md' },
      staleDaysThreshold: 7,
      referenceDate: '2026-04-15',
    });

    expect(stale).toHaveLength(0);
  });

  it('handles empty inputs', () => {
    const stale = findStaleDocCandidates([], [], {
      codeToDocMapping: {},
      staleDaysThreshold: 7,
      referenceDate: '2026-04-15',
    });
    expect(stale).toHaveLength(0);
  });
});
