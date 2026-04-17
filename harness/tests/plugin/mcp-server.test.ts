import { describe, it, expect } from 'vitest';
import { createForgeTools, FORGE_TOOLS } from '../../src/mcp-server.js';

describe('FORGE_TOOLS', () => {
  it('is a frozen array (single source of truth)', () => {
    expect(Object.isFrozen(FORGE_TOOLS)).toBe(true);
  });

  it('contains definitions for run, fix, plan, status', () => {
    const names = FORGE_TOOLS.map(t => t.name);
    expect(names).toContain('forge_run');
    expect(names).toContain('forge_fix');
    expect(names).toContain('forge_plan');
    expect(names).toContain('forge_status');
  });
});

describe('createForgeTools', () => {
  it('returns the same FORGE_TOOLS reference', () => {
    expect(createForgeTools()).toBe(FORGE_TOOLS);
  });

  it('forge_run tool has task parameter', () => {
    const tools = createForgeTools();
    const runTool = tools.find(t => t.name === 'forge_run')!;
    expect(runTool.inputSchema.properties).toHaveProperty('task');
  });

  it('forge_fix tool has optional filePath parameter', () => {
    const tools = createForgeTools();
    const fixTool = tools.find(t => t.name === 'forge_fix')!;
    expect(fixTool.inputSchema.properties).toHaveProperty('task');
    expect(fixTool.inputSchema.properties).toHaveProperty('file_path');
  });

  it('forge_status tool has no required parameters', () => {
    const tools = createForgeTools();
    const statusTool = tools.find(t => t.name === 'forge_status')!;
    expect(statusTool.inputSchema.required ?? []).toEqual([]);
  });
});
