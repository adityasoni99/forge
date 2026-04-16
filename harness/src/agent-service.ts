import type { AgentAdapter } from './adapters/types.js';
import type { ExecuteAgentRequest, ExecuteAgentResponse } from './types.js';
import { loadContext } from './context/loader.js';
import { composePromptStack, PromptLayer, type PromptLayerEntry } from './context/prompt-stack.js';
import { SkillResolver } from './skills/resolver.js';
import type { SkillRegistry } from './skills/registry.js';

export interface AgentServiceOptions {
  skillRegistry?: SkillRegistry;
  defaultAdapter?: string;
}

export class AgentService {
  private readonly adapters: Map<string, AgentAdapter>;
  private readonly defaultAdapterName: string;
  private readonly skillResolver: SkillResolver | null;

  constructor(
    adapters: Map<string, AgentAdapter> | AgentAdapter,
    options?: AgentServiceOptions,
  ) {
    if (adapters instanceof Map) {
      this.adapters = adapters;
    } else {
      this.adapters = new Map([[adapters.name, adapters]]);
    }
    this.defaultAdapterName = options?.defaultAdapter ?? this.adapters.keys().next().value!;
    const registry = options?.skillRegistry;
    this.skillResolver = registry ? new SkillResolver(registry.all()) : null;
  }

  async executeAgent(req: ExecuteAgentRequest): Promise<ExecuteAgentResponse> {
    try {
      const config = this.parseConfig(req.config_json);
      const adapterName = req.adapter || (config.adapter as string) || this.defaultAdapterName;
      const adapter = this.adapters.get(adapterName);
      if (!adapter) {
        return { output: '', success: false, error: `unknown adapter: ${adapterName}` };
      }

      const ctx = await loadContext(req.working_directory);
      const capabilities = adapter.getCapabilities();

      const layers: PromptLayerEntry[] = [];
      if (ctx.rootDoc) {
        layers.push({ layer: PromptLayer.ProjectRules, label: ctx.rootDocName || 'project', content: ctx.rootDoc });
      }
      for (const rule of ctx.rules) {
        layers.push({ layer: PromptLayer.ProjectRules, label: rule.name, content: rule.content });
      }

      let taskPrompt = req.prompt;
      const skill = this.skillResolver?.resolve(req.prompt, config) ?? null;
      if (skill) {
        taskPrompt = `${skill.body}\n\n${taskPrompt}`;
      }

      const composedPrompt = composePromptStack(layers, taskPrompt, {
        maxTokens: capabilities.maxContextTokens,
      });

      let output = '';
      let success = true;
      let error = '';

      for await (const event of adapter.execute({
        prompt: composedPrompt,
        workingDirectory: req.working_directory,
        configJson: req.config_json,
      })) {
        if (event.type === 'done') {
          output += event.content;
        } else if (event.type === 'output') {
          output += event.content;
        } else if (event.type === 'error') {
          success = false;
          error = event.content;
        }
      }

      return { output, success, error };
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
