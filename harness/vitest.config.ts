import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    coverage: {
      provider: 'v8',
      include: [
        'src/agent-service.ts',
        'src/context/loader.ts',
        'src/adapters/echo.ts',
        'src/adapters/claude-code.ts',
        'src/skills/types.ts',
        'src/skills/registry.ts',
        'src/skills/resolver.ts',
        'src/skills/lifecycle.ts',
      ],
      thresholds: {
        lines: 90,
        branches: 85,
        functions: 90,
        statements: 90,
      },
    },
  },
});
