import type { PluginConfig } from './types.js';

export const DEFAULT_CONFIG: Required<PluginConfig> = {
  defaultAdapter: '',
  executionMode: 'auto',
  forgeBinaryPath: 'forge',
  harnessPort: 50051,
};

export function resolveConfig(overrides?: Partial<PluginConfig>): Required<PluginConfig> {
  return { ...DEFAULT_CONFIG, ...stripUndefined(overrides ?? {}) };
}

function stripUndefined(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    if (value !== undefined) result[key] = value;
  }
  return result;
}
