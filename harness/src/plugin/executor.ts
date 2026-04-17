import { AgentService } from '../agent-service.js';
import type { AgentAdapter } from '../adapters/types.js';
import type { PluginConfig, ExecutionResult } from './types.js';

export interface ExecutorOptions {
  adapters: Map<string, AgentAdapter>;
  config: Required<PluginConfig>;
}

export interface ExecuteRequest {
  prompt: string;
  workingDirectory: string;
  adapter: string;
  blueprintName?: string;
  runId?: string;
}

export class ForgeExecutor {
  private readonly agentService: AgentService;
  private readonly config: Required<PluginConfig>;

  constructor(options: ExecutorOptions) {
    this.config = options.config;
    this.agentService = new AgentService(options.adapters, {
      defaultAdapter: options.config.defaultAdapter || undefined,
    });
  }

  async execute(req: ExecuteRequest): Promise<ExecutionResult> {
    const start = Date.now();
    try {
      const result = await this.executeDirect(req);
      result.durationMs = Date.now() - start;
      return result;
    } catch (err) {
      return {
        output: '',
        success: false,
        error: err instanceof Error ? err.message : String(err),
        mode: 'direct',
        adapter: req.adapter,
        durationMs: Date.now() - start,
      };
    }
  }

  private async executeDirect(req: ExecuteRequest): Promise<ExecutionResult> {
    const response = await this.agentService.executeAgent({
      prompt: req.prompt,
      config_json: JSON.stringify({ adapter: req.adapter }),
      working_directory: req.workingDirectory,
      blueprint_name: req.blueprintName ?? '',
      node_id: 'plugin',
      run_id: req.runId ?? `plugin-${Date.now()}`,
      adapter: req.adapter,
    });

    return {
      output: response.output,
      success: response.success,
      error: response.error || undefined,
      mode: 'direct',
      adapter: req.adapter,
      durationMs: 0,
    };
  }
}
