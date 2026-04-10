import type { AgentAdapter } from './adapters/types.js';
import type { ExecuteAgentRequest, ExecuteAgentResponse } from './types.js';
import { loadContext } from './context/loader.js';

export class AgentService {
  constructor(private readonly adapter: AgentAdapter) {}

  async executeAgent(req: ExecuteAgentRequest): Promise<ExecuteAgentResponse> {
    try {
      const ctx = await loadContext(req.working_directory);
      const composedPrompt = ctx.composePrompt(req.prompt);

      const result = await this.adapter.execute({
        prompt: composedPrompt,
        workingDirectory: req.working_directory,
        configJson: req.config_json,
      });

      return {
        output: result.output,
        success: result.success,
        error: result.error ?? '',
      };
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return { output: '', success: false, error: message };
    }
  }
}
