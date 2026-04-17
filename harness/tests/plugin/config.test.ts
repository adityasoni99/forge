import { describe, it, expect } from 'vitest';
import { resolveConfig, DEFAULT_CONFIG } from '../../src/plugin/config.js';

describe('resolveConfig', () => {
  it('returns defaults when no overrides', () => {
    const config = resolveConfig();
    expect(config).toEqual(DEFAULT_CONFIG);
  });

  it('merges partial overrides with defaults', () => {
    const config = resolveConfig({ defaultAdapter: 'goose' });
    expect(config.defaultAdapter).toBe('goose');
    expect(config.forgeBinaryPath).toBe(DEFAULT_CONFIG.forgeBinaryPath);
  });

  it('respects explicit forgeBinaryPath', () => {
    const config = resolveConfig({ forgeBinaryPath: '/usr/local/bin/forge' });
    expect(config.forgeBinaryPath).toBe('/usr/local/bin/forge');
  });

  it('default forgeBinaryPath is forge', () => {
    const config = resolveConfig();
    expect(config.forgeBinaryPath).toBe('forge');
  });

  it('does not include removed fields (executionMode, harnessPort)', () => {
    const config = resolveConfig() as Record<string, unknown>;
    expect(config).not.toHaveProperty('executionMode');
    expect(config).not.toHaveProperty('harnessPort');
  });
});
