import { describe, it, expect } from 'vitest';
import { compressShellOutput } from '../../src/context/compression.js';

describe('compressShellOutput', () => {
  it('returns short output unchanged', () => {
    const result = compressShellOutput('hello world', { maxLines: 50 });
    expect(result.compressed).toBe('hello world');
    expect(result.ratio).toBe(1);
  });

  it('compresses long output', () => {
    const lines = Array.from({ length: 200 }, (_, i) => `line ${i}`).join('\n');
    const result = compressShellOutput(lines, { maxLines: 30 });
    expect(result.compressedLines).toBeLessThanOrEqual(35);
    expect(result.originalLines).toBe(200);
    expect(result.compressed).toContain('lines omitted');
  });

  it('keeps error lines', () => {
    const lines = [
      'ok 1',
      'ok 2',
      ...Array.from({ length: 100 }, (_, i) => `ok ${i + 3}`),
      'FAIL: test_foo expected 1 got 2',
      'Error: assertion failed',
      'ok final',
    ].join('\n');
    const result = compressShellOutput(lines, { maxLines: 20, keepErrorLines: true });
    expect(result.compressed).toContain('FAIL: test_foo');
    expect(result.compressed).toContain('Error: assertion failed');
  });

  it('handles empty input', () => {
    const result = compressShellOutput('');
    expect(result.compressed).toBe('');
    expect(result.originalLines).toBe(0);
  });
});
