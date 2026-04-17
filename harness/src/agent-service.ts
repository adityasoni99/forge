import type { AgentAdapter } from './adapters/types.js';
import type { ExecuteAgentRequest, ExecuteAgentResponse } from './types.js';
import { loadContext } from './context/loader.js';
import { composePromptStack, PromptLayer, type PromptLayerEntry } from './context/prompt-stack.js';
import { SkillResolver } from './skills/resolver.js';
import type { SkillRegistry } from './skills/registry.js';
import { SessionEventEmitter } from './memory/session.js';
import type { SessionEventType } from './memory/types.js';

export interface AgentServiceOptions {
  skillRegistry?: SkillRegistry;
  defaultAdapter?: string;
  sessionEmitter?: SessionEventEmitter;
}

export class AgentService {
  private readonly adapters: Map<string, AgentAdapter>;
  private readonly defaultAdapterName: string;
  private readonly skillResolver: SkillResolver | null;
  private readonly sessionEmitter: SessionEventEmitter | null;

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
    this.sessionEmitter = options?.sessionEmitter ?? null;
  }

  async executeAgent(req: ExecuteAgentRequest): Promise<ExecuteAgentResponse> {
    const runID = req.run_id || 'unknown';
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

      await this.emitSession(runID, {
        type: 'prompt_composed',
        data: { tokens: Math.ceil(composedPrompt.length / 4) },
      });

      let output = '';
      let success = true;
      let error = '';

      await this.emitSession(runID, {
        type: 'adapter_called',
        data: { adapter: adapterName },
      });

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

      await this.emitSession(runID, {
        type: 'adapter_result',
        data: { success, outputLength: output.length },
      });

      return { output, success, error };
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      await this.emitSession(runID, { type: 'error', data: { message } });
      return { output: '', success: false, error: message };
    }
  }

  private async emitSession(
    runID: string,
    options: { type: SessionEventType; nodeID?: string; data?: Record<string, unknown> },
  ): Promise<void> {
    if (!this.sessionEmitter) return;
    try {
      await this.sessionEmitter.emit(runID, options);
    } catch {
      // best-effort: session logging should not break execution
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
