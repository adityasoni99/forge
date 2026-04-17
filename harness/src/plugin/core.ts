import type { AgentAdapter } from '../adapters/types.js';
import type { IDEContext, PluginConfig, ExecutionResult, CommandDefinition } from './types.js';
import { detectIDE } from './ide-detect.js';
import { resolveConfig } from './config.js';
import { resolveCommand, buildTaskPrompt, COMMANDS, type PromptContext } from './commands.js';
import { ForgeExecutor } from './executor.js';
import { EchoAdapter } from '../adapters/echo.js';

export interface ForgePluginCoreOptions {
  adapters?: Map<string, AgentAdapter>;
  config?: Partial<PluginConfig>;
  workingDirectory?: string;
}

export interface PluginStatus {
  ide: string;
  defaultAdapter: string;
  availableAdapters: string[];
  executionMode: string;
  workingDirectory: string;
}

export class ForgePluginCore {
  readonly ideContext: IDEContext;
  private readonly config: Required<PluginConfig>;
  private readonly executor: ForgeExecutor;
  private readonly adapterNames: string[];

  constructor(options?: ForgePluginCoreOptions) {
    this.ideContext = detectIDE(options?.workingDirectory);

    const adapterOverride = options?.config?.defaultAdapter || this.ideContext.defaultAdapter;
    this.config = resolveConfig({ ...options?.config, defaultAdapter: adapterOverride });

    const adapters = options?.adapters ?? new Map([['echo', new EchoAdapter()]]);
    this.adapterNames = [...adapters.keys()];

    this.executor = new ForgeExecutor({ adapters, config: this.config });
  }

  async executeCommand(
    commandName: string,
    task: string,
    context?: PromptContext,
  ): Promise<ExecutionResult> {
    const command = resolveCommand(commandName);
    if (!command) {
      return {
        output: '',
        success: false,
        error: `unknown command: ${commandName}`,
        mode: 'direct',
        adapter: this.config.defaultAdapter,
        durationMs: 0,
      };
    }

    const prompt = buildTaskPrompt(command, task, context);

    return this.executor.execute({
      prompt,
      workingDirectory: this.ideContext.workingDirectory,
      adapter: this.config.defaultAdapter,
      blueprintName: command.blueprintName,
    });
  }

  listCommands(): CommandDefinition[] {
    return [...COMMANDS.values()];
  }

  getStatus(): PluginStatus {
    return {
      ide: this.ideContext.ide,
      defaultAdapter: this.config.defaultAdapter,
      availableAdapters: this.adapterNames,
      executionMode: this.config.executionMode,
      workingDirectory: this.ideContext.workingDirectory,
    };
  }
}
