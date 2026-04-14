export type ToolSource = 'builtin' | 'extension' | 'mcp';

export interface Tool {
  name: string;
  description: string;
  source: ToolSource;
  parameters?: Record<string, unknown>;
}

export interface DenyRule {
  toolName: string;
  reason: string;
}

export interface PermissionContext {
  denyRules: DenyRule[];
  maxTools: number;
}

export function createPermissionContext(
  denyRules: DenyRule[] = [],
  maxTools = 15,
): PermissionContext {
  return { denyRules, maxTools };
}
