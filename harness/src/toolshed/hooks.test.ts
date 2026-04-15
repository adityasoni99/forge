import { describe, it, expect } from 'vitest';
import { ToolHookRegistry } from './hooks.js';
import type { ToolInvocation, ToolResult } from './hooks.js';

describe('ToolHookRegistry', () => {
  it('runs pre-hook before tool invocation', async () => {
    const registry = new ToolHookRegistry();
    const calls: string[] = [];
    registry.registerPreHook('read_file', async (inv) => {
      calls.push(`pre:${inv.toolName}`);
      return inv;
    });

    const inv: ToolInvocation = { toolName: 'read_file', args: { path: '/a.ts' } };
    const result = await registry.runPreHooks(inv);
    expect(calls).toEqual(['pre:read_file']);
    expect(result).not.toBeNull();
    expect(result!.toolName).toBe('read_file');
  });

  it('runs post-hook after tool invocation', async () => {
    const registry = new ToolHookRegistry();
    const calls: string[] = [];
    registry.registerPostHook('write_file', async (_inv, res) => {
      calls.push(`post:${res.success}`);
      return res;
    });

    const inv: ToolInvocation = { toolName: 'write_file', args: {} };
    const res: ToolResult = { output: 'written', success: true };
    const result = await registry.runPostHooks(inv, res);
    expect(calls).toEqual(['post:true']);
    expect(result.success).toBe(true);
  });

  it('pre-hook can modify invocation', async () => {
    const registry = new ToolHookRegistry();
    registry.registerPreHook('shell', async (inv) => {
      return { ...inv, args: { ...inv.args, timeout: 30 } };
    });

    const inv: ToolInvocation = { toolName: 'shell', args: { command: 'ls' } };
    const result = await registry.runPreHooks(inv);
    expect(result).not.toBeNull();
    expect(result!.args).toEqual({ command: 'ls', timeout: 30 });
  });

  it('pre-hook can block by returning null', async () => {
    const registry = new ToolHookRegistry();
    registry.registerPreHook('shell', async () => null);

    const inv: ToolInvocation = { toolName: 'shell', args: {} };
    const result = await registry.runPreHooks(inv);
    expect(result).toBeNull();
  });

  it('ignores hooks for other tools', async () => {
    const registry = new ToolHookRegistry();
    const calls: string[] = [];
    registry.registerPreHook('read_file', async (inv) => {
      calls.push('read');
      return inv;
    });

    const inv: ToolInvocation = { toolName: 'write_file', args: {} };
    const result = await registry.runPreHooks(inv);
    expect(calls).toEqual([]);
    expect(result).not.toBeNull();
    expect(result!.toolName).toBe('write_file');
  });

  it('chains multiple pre-hooks in order', async () => {
    const registry = new ToolHookRegistry();
    const order: number[] = [];
    registry.registerPreHook('shell', async (inv) => {
      order.push(1);
      return inv;
    });
    registry.registerPreHook('shell', async (inv) => {
      order.push(2);
      return inv;
    });

    const inv: ToolInvocation = { toolName: 'shell', args: {} };
    await registry.runPreHooks(inv);
    expect(order).toEqual([1, 2]);
  });
});
