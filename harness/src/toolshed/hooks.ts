export interface ToolInvocation {
  toolName: string;
  args: Record<string, unknown>;
}

export interface ToolResult {
  output: string;
  success: boolean;
  error?: string;
}

export type PreHook = (inv: ToolInvocation) => Promise<ToolInvocation | null>;
export type PostHook = (inv: ToolInvocation, res: ToolResult) => Promise<ToolResult>;

export class ToolHookRegistry {
  private preHooks = new Map<string, PreHook[]>();
  private postHooks = new Map<string, PostHook[]>();

  registerPreHook(toolName: string, hook: PreHook): void {
    const hooks = this.preHooks.get(toolName) ?? [];
    hooks.push(hook);
    this.preHooks.set(toolName, hooks);
  }

  registerPostHook(toolName: string, hook: PostHook): void {
    const hooks = this.postHooks.get(toolName) ?? [];
    hooks.push(hook);
    this.postHooks.set(toolName, hooks);
  }

  async runPreHooks(inv: ToolInvocation): Promise<ToolInvocation | null> {
    const hooks = this.preHooks.get(inv.toolName) ?? [];
    let current: ToolInvocation | null = inv;
    for (const hook of hooks) {
      if (current === null) return null;
      current = await hook(current);
    }
    return current;
  }

  async runPostHooks(inv: ToolInvocation, res: ToolResult): Promise<ToolResult> {
    const hooks = this.postHooks.get(inv.toolName) ?? [];
    let current = res;
    for (const hook of hooks) {
      current = await hook(inv, current);
    }
    return current;
  }
}
