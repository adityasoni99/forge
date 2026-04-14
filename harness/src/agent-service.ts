import type { AgentAdapter } from './adapters/types.js';
import type { ExecuteAgentRequest, ExecuteAgentResponse } from './types.js';
import { loadContext } from './context/loader.js';
import { SkillResolver } from './skills/resolver.js';
import type { SkillRegistry } from './skills/registry.js';

export interface AgentServiceOptions {
  skillRegistry?: SkillRegistry;
}

export class AgentService {
  private readonly skillResolver: SkillResolver | null;

  constructor(
    private readonly adapter: AgentAdapter,
    options?: AgentServiceOptions,
  ) {
    const registry = options?.skillRegistry;
    this.skillResolver = registry ? new SkillResolver(registry.all()) : null;
  }

  async executeAgent(req: ExecuteAgentRequest): Promise<ExecuteAgentResponse> {
    try {
      const ctx = await loadContext(req.working_directory);
      let prompt = req.prompt;

      const config = this.parseConfig(req.config_json);
      const skill = this.skillResolver?.resolve(req.prompt, config) ?? null;
      if (skill) {
        prompt = `${skill.body}\n\n${prompt}`;
      }

      const composedPrompt = ctx.composePrompt(prompt);

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

  private parseConfig(json: string): Record<string, unknown> {
    try {
      return JSON.parse(json) as Record<string, unknown>;
    } catch {
      return {};
    }
  }
}
