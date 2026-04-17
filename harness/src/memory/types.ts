export interface SessionEvent {
  id: string;
  timestamp: string;
  runID: string;
  type: SessionEventType;
  nodeID?: string;
  data?: Record<string, unknown>;
}

export type SessionEventType =
  | 'prompt_composed'
  | 'adapter_called'
  | 'adapter_result'
  | 'tool_invoked'
  | 'tool_result'
  | 'error'
  | 'run_complete';

export interface DerivedRule {
  category: string;
  name: string;
  body: string;
  sourceRunID: string;
  sourceError: string;
}

export interface DocFreshnessReport {
  staleFiles: StaleDoc[];
  checkedAt: string;
}

export interface StaleDoc {
  docPath: string;
  relatedCodePaths: string[];
  docLastModified: string;
  codeLastModified: string;
  staleDays: number;
}
