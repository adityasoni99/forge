import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { loadContext } from './loader.js';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as os from 'node:os';

describe('loadContext', () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-ctx-'));
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it('loads AGENTS.md from project root', async () => {
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), '# Project\nKey info here.');
    const ctx = await loadContext(tmpDir);
    expect(ctx.rootDoc).toContain('Key info here.');
    expect(ctx.rootDocName).toBe('AGENTS.md');
  });

  it('falls back to CLAUDE.md if no AGENTS.md', async () => {
    fs.writeFileSync(path.join(tmpDir, 'CLAUDE.md'), '# Claude Config');
    const ctx = await loadContext(tmpDir);
    expect(ctx.rootDoc).toContain('Claude Config');
    expect(ctx.rootDocName).toBe('CLAUDE.md');
  });

  it('falls back to README.md if no AGENTS.md or CLAUDE.md', async () => {
    fs.writeFileSync(path.join(tmpDir, 'README.md'), '# Readme only');
    const ctx = await loadContext(tmpDir);
    expect(ctx.rootDoc).toContain('Readme only');
    expect(ctx.rootDocName).toBe('README.md');
  });

  it('loads .forge/rules/*.md files', async () => {
    fs.mkdirSync(path.join(tmpDir, '.forge', 'rules'), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, '.forge', 'rules', 'auth.md'), 'Auth rules.');
    fs.writeFileSync(path.join(tmpDir, '.forge', 'rules', 'api.md'), 'API rules.');
    const ctx = await loadContext(tmpDir);
    expect(ctx.rules).toHaveLength(2);
    expect(ctx.rules[0].name).toBe('api.md'); // sorted
    expect(ctx.rules[1].name).toBe('auth.md');
  });

  it('returns empty context for bare directory', async () => {
    const ctx = await loadContext(tmpDir);
    expect(ctx.rootDoc).toBe('');
    expect(ctx.rules).toHaveLength(0);
  });

  it('composePrompt prepends context to task prompt', async () => {
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), '# Conventions\nUse TDD.');
    const ctx = await loadContext(tmpDir);
    const composed = ctx.composePrompt('Fix the login bug');
    expect(composed).toContain('# Conventions');
    expect(composed).toContain('Fix the login bug');
    expect(composed.indexOf('Conventions')).toBeLessThan(composed.indexOf('Fix the login bug'));
  });

  it('respects maxTokens by omitting rules when budget is tight', async () => {
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), 'ROOT');
    fs.mkdirSync(path.join(tmpDir, '.forge', 'rules'), { recursive: true });
    const longRule = 'x'.repeat(400);
    fs.writeFileSync(path.join(tmpDir, '.forge', 'rules', 'a.md'), longRule);
    fs.writeFileSync(path.join(tmpDir, '.forge', 'rules', 'b.md'), longRule);
    const ctx = await loadContext(tmpDir, { maxTokens: 80 });
    const composed = ctx.composePrompt('Task');
    expect(composed).toContain('ROOT');
    expect(composed).toContain('Task');
    expect(composed).not.toContain(longRule);
  });

  it('includes directory rules in composed prompt when rules fit budget', async () => {
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), 'ROOT');
    fs.mkdirSync(path.join(tmpDir, '.forge', 'rules'), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, '.forge', 'rules', 'z.md'), 'Rule Z');
    const ctx = await loadContext(tmpDir, { maxTokens: 2000 });
    const composed = ctx.composePrompt('Do the thing');
    expect(composed).toContain('=== Directory Rules ===');
    expect(composed).toContain('--- z.md ---');
    expect(composed).toContain('Rule Z');
  });

  it('truncates composed prompt when join overhead exceeds maxTokens', async () => {
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), 'R'.repeat(200));
    fs.mkdirSync(path.join(tmpDir, '.forge', 'rules'), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, '.forge', 'rules', 'a.md'), 'Z'.repeat(120));
    const ctx = await loadContext(tmpDir, { maxTokens: 95 });
    const composed = ctx.composePrompt('TASK');
    expect(composed.length).toBeGreaterThan(0);
    expect(composed).toContain('TASK');
    expect(composed.length).toBeLessThan(2000);
  });

  it('iterates truncation until within token budget', async () => {
    fs.writeFileSync(path.join(tmpDir, 'AGENTS.md'), 'Q'.repeat(5000));
    const ctx = await loadContext(tmpDir, { maxTokens: 50 });
    const composed = ctx.composePrompt('T');
    expect(composed.length).toBeGreaterThan(24);
    expect(composed).toContain('T');
    expect(composed.length).toBeLessThan(5000);
  });
});
