import { IDEType, type IDEContext } from './types.js';

const IDE_SIGNALS: Array<{ check: () => boolean; ide: IDEType; adapter: string }> = [
  { check: () => Boolean(process.env.CURSOR_TRACE_ID || process.env.CURSOR_SESSION), ide: IDEType.Cursor, adapter: 'cursor' },
  { check: () => Boolean(process.env.CLAUDE_CODE || process.env.CLAUDE_SESSION_ID), ide: IDEType.ClaudeCode, adapter: 'claude' },
  { check: () => Boolean(process.env.CODEIUM_SESSION || process.env.WINDSURF_SESSION), ide: IDEType.Windsurf, adapter: 'claude' },
];

export function detectIDE(workingDirectory?: string): IDEContext {
  for (const signal of IDE_SIGNALS) {
    if (signal.check()) {
      return { ide: signal.ide, defaultAdapter: signal.adapter, workingDirectory: workingDirectory ?? process.cwd() };
    }
  }
  return { ide: IDEType.Unknown, defaultAdapter: 'claude', workingDirectory: workingDirectory ?? process.cwd() };
}
