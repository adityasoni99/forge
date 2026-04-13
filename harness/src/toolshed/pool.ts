import type { Tool, PermissionContext } from './types.js';

export function assembleToolPool(
  builtins: Tool[],
  extensions: Tool[],
  ctx: PermissionContext,
): Tool[] {
  const denySet = new Set(ctx.denyRules.map((r) => r.toolName));

  const byName = new Map<string, Tool>();
  for (const tool of builtins) {
    if (!denySet.has(tool.name)) {
      byName.set(tool.name, tool);
    }
  }
  for (const tool of extensions) {
    if (!denySet.has(tool.name) && !byName.has(tool.name)) {
      byName.set(tool.name, tool);
    }
  }

  const sorted = [...byName.values()].sort((a, b) => a.name.localeCompare(b.name));
  return sorted.slice(0, ctx.maxTools);
}
