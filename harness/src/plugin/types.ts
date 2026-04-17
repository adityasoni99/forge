export enum IDEType {
  Cursor = 'cursor',
  ClaudeCode = 'claude-code',
  Windsurf = 'windsurf',
  Unknown = 'unknown',
}

export interface IDEContext {
  ide: IDEType;
  defaultAdapter: string;
  workingDirectory: string;
}

export interface PluginConfig {
  defaultAdapter?: string;
  executionMode?: 'direct' | 'pipeline' | 'auto';
  forgeBinaryPath?: string;
  harnessPort?: number;
}

export interface CommandDefinition {
  name: string;
  description: string;
  blueprintName: string;
  promptTemplate: string;
  planOnly?: boolean;
}

export interface ExecutionResult {
  output: string;
  success: boolean;
  error?: string;
  mode: 'direct' | 'pipeline';
  adapter: string;
  durationMs: number;
}
