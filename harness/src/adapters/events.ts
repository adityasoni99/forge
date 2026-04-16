export type AgentEventType = 'output' | 'tool_call' | 'tool_result' | 'error' | 'done';

export interface AgentEvent {
  type: AgentEventType;
  content: string;
  metadata?: Record<string, unknown>;
}
