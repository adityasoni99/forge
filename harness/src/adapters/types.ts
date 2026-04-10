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
  maxTokens?: number;
}

export interface AgentAdapter {
  execute(req: AgentAdapterRequest): Promise<AgentAdapterResponse>;
  getCapabilities?(): AgentCapabilities;
}
