import { describe, it, expect } from 'vitest';
import { resolveConfig, DEFAULT_CONFIG } from '../../src/plugin/config.js';
import type { PluginConfig } from '../../src/plugin/types.js';

describe('resolveConfig', () => {
  it('returns defaults when no overrides', () => {
    const config = resolveConfig();
    expect(config).toEqual(DEFAULT_CONFIG);
  });

  it('merges partial overrides with defaults', () => {
    const config = resolveConfig({ defaultAdapter: 'goose' });
    expect(config.defaultAdapter).toBe('goose');
    expect(config.executionMode).toBe(DEFAULT_CONFIG.executionMode);
  });

  it('auto execution mode is the default', () => {
    const config = resolveConfig();
    expect(config.executionMode).toBe('auto');
  });

  it('respects explicit forgeBinaryPath', () => {
    const config = resolveConfig({ forgeBinaryPath: '/usr/local/bin/forge' });
    expect(config.forgeBinaryPath).toBe('/usr/local/bin/forge');
  });

  it('default harnessPort is 50051', () => {
    const config = resolveConfig();
    expect(config.harnessPort).toBe(50051);
  });
});
