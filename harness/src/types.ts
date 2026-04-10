export interface ExecuteAgentRequest {
  prompt: string;
  config_json: string;
  working_directory: string;
  blueprint_name: string;
  node_id: string;
  run_id: string;
}

export interface ExecuteAgentResponse {
  output: string;
  success: boolean;
  error: string;
}
