import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { detectIDE } from '../../src/plugin/ide-detect.js';
import { IDEType } from '../../src/plugin/types.js';

describe('detectIDE', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    process.env = { ...originalEnv };
    delete process.env.CURSOR_TRACE_ID;
    delete process.env.CURSOR_SESSION;
    delete process.env.CLAUDE_CODE;
    delete process.env.CLAUDE_SESSION_ID;
    delete process.env.CODEIUM_SESSION;
    delete process.env.WINDSURF_SESSION;
  });

  afterEach(() => {
    process.env = originalEnv;
  });

  it('detects Cursor from CURSOR_TRACE_ID', () => {
    process.env.CURSOR_TRACE_ID = 'abc123';
    const ctx = detectIDE();
    expect(ctx.ide).toBe(IDEType.Cursor);
    expect(ctx.defaultAdapter).toBe('cursor');
  });

  it('detects Claude Code from CLAUDE_CODE env', () => {
    process.env.CLAUDE_CODE = '1';
    const ctx = detectIDE();
    expect(ctx.ide).toBe(IDEType.ClaudeCode);
    expect(ctx.defaultAdapter).toBe('claude');
  });

  it('detects Windsurf from CODEIUM_SESSION env', () => {
    process.env.CODEIUM_SESSION = 'session-xyz';
    const ctx = detectIDE();
    expect(ctx.ide).toBe(IDEType.Windsurf);
    expect(ctx.defaultAdapter).toBe('claude');
  });

  it('returns Unknown with claude default when no IDE detected', () => {
    const ctx = detectIDE();
    expect(ctx.ide).toBe(IDEType.Unknown);
    expect(ctx.defaultAdapter).toBe('claude');
  });

  it('uses cwd as workingDirectory', () => {
    const ctx = detectIDE();
    expect(ctx.workingDirectory).toBe(process.cwd());
  });

  it('accepts workingDirectory override', () => {
    const ctx = detectIDE('/custom/dir');
    expect(ctx.workingDirectory).toBe('/custom/dir');
  });
});
