import { describe, it, expect } from 'vitest';
import { resolveCommand, buildTaskPrompt, COMMANDS } from '../../src/plugin/commands.js';

describe('COMMANDS', () => {
  it('has run, fix, and plan commands', () => {
    expect(COMMANDS.has('run')).toBe(true);
    expect(COMMANDS.has('fix')).toBe(true);
    expect(COMMANDS.has('plan')).toBe(true);
  });
});

describe('resolveCommand', () => {
  it('returns command by name', () => {
    const cmd = resolveCommand('run');
    expect(cmd).toBeDefined();
    expect(cmd!.blueprintName).toBe('standard-implementation');
  });

  it('returns undefined for unknown command', () => {
    expect(resolveCommand('nonexistent')).toBeUndefined();
  });

  it('fix command maps to bug-fix blueprint', () => {
    const cmd = resolveCommand('fix');
    expect(cmd!.blueprintName).toBe('bug-fix');
  });

  it('plan command is plan-only', () => {
    const cmd = resolveCommand('plan');
    expect(cmd!.planOnly).toBe(true);
  });
});

describe('buildTaskPrompt', () => {
  it('substitutes task into prompt template', () => {
    const cmd = resolveCommand('run')!;
    const prompt = buildTaskPrompt(cmd, 'Add user authentication');
    expect(prompt).toContain('Add user authentication');
  });

  it('fix command includes error context placeholder', () => {
    const cmd = resolveCommand('fix')!;
    const prompt = buildTaskPrompt(cmd, 'TypeError in auth.ts', { filePath: 'src/auth.ts' });
    expect(prompt).toContain('TypeError in auth.ts');
    expect(prompt).toContain('src/auth.ts');
  });

  it('plan command includes planning-only instruction', () => {
    const cmd = resolveCommand('plan')!;
    const prompt = buildTaskPrompt(cmd, 'Redesign the API layer');
    expect(prompt).toContain('plan');
    expect(prompt).not.toContain('implement');
  });

  it('escapes template tokens in task input to prevent injection', () => {
    const cmd = resolveCommand('run')!;
    const malicious = 'Fix {{filePath}} and {{#filePath}}injected{{/filePath}}';
    const prompt = buildTaskPrompt(cmd, malicious);
    expect(prompt).not.toContain('{{filePath}}');
    expect(prompt).not.toContain('{{#filePath}}');
    expect(prompt).toContain('{ {filePath} }');
  });

  it('escapes template tokens in context fields', () => {
    const cmd = resolveCommand('fix')!;
    const prompt = buildTaskPrompt(cmd, 'bug', {
      filePath: 'src/{{task}}.ts',
      errorOutput: 'error at {{task}}',
    });
    expect(prompt).not.toContain('{{task}}');
    expect(prompt).toContain('{ {task} }');
  });
});
