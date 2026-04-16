import type { AgentEvent } from './events.js';

export interface AgentAdapterRequest {
  prompt: string;
  workingDirectory: string;
  configJson: string;
}

export interface AgentAdapterResponse {
  output: string;
  success: boolean;
  error?: string;
}

export interface AgentCapabilities {
  streaming: boolean;
  interrupt: boolean;
  maxContextTokens: number;
  supportsTools: boolean;
  supportedToolNames?: string[];
  needsContextReset: boolean;
}

export interface AgentAdapter {
  readonly name: string;
  execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent>;
  getCapabilities(): AgentCapabilities;
  interrupt(): Promise<void>;
}

export type { AgentAdapterRequest as LegacyAdapterRequest };
export type { AgentAdapterResponse as LegacyAdapterResponse };

export interface SyncAgentAdapter {
  execute(req: AgentAdapterRequest): Promise<AgentAdapterResponse>;
  getCapabilities?(): Partial<AgentCapabilities>;
}
