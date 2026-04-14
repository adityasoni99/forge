import type { RuleFile } from './loader.js';

export enum SubagentType {
  Explore = 'explore',
  Implement = 'implement',
  Review = 'review',
}

export interface ParentContext {
  maxTokens: number;
  rules: RuleFile[];
  tools: string[];
  fileCache: Map<string, string>;
}

export interface ForkOptions {
  type: SubagentType;
  maxTokens?: number;
}

const WRITE_TOOLS = new Set(['write_file', 'shell', 'edit_file', 'create_file', 'delete_file']);
const SHELL_TOOLS = new Set(['shell', 'execute_command']);

export class SubagentContext {
  readonly maxTokens: number;
  readonly rules: RuleFile[];
  readonly tools: string[];
  readonly fileCache: Map<string, string>;
  readonly type: SubagentType;

  private constructor(
    type: SubagentType,
    maxTokens: number,
    rules: RuleFile[],
    tools: string[],
    fileCache: Map<string, string>,
  ) {
    this.type = type;
    this.maxTokens = maxTokens;
    this.rules = rules;
    this.tools = tools;
    this.fileCache = fileCache;
  }

  static fork(parent: ParentContext, options: ForkOptions): SubagentContext {
    const maxTokens = options.maxTokens ?? Math.floor(parent.maxTokens / 2);
    const rules = [...parent.rules];
    const fileCache = new Map(parent.fileCache);
    const tools = filterToolsForType(parent.tools, options.type);

    return new SubagentContext(options.type, maxTokens, rules, tools, fileCache);
  }
}

function filterToolsForType(tools: string[], type: SubagentType): string[] {
  switch (type) {
    case SubagentType.Explore:
      return tools.filter((t) => !WRITE_TOOLS.has(t) && !SHELL_TOOLS.has(t));
    case SubagentType.Review:
      return tools.filter((t) => !SHELL_TOOLS.has(t));
    case SubagentType.Implement:
      return [...tools];
  }
}
