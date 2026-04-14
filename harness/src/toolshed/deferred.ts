import type { Tool } from './types.js';

export interface PartitionResult {
  inline: Tool[];
  deferred: Tool[];
}

export class DeferredToolLoader {
  constructor(private readonly budgetTokens: number) {}

  partition(tools: Tool[]): PartitionResult {
    const inline: Tool[] = [];
    const deferred: Tool[] = [];
    let used = 0;

    for (const tool of tools) {
      const cost = this.estimateToolTokens(tool);
      if (used + cost <= this.budgetTokens) {
        inline.push(tool);
        used += cost;
      } else {
        deferred.push(tool);
      }
    }

    return { inline, deferred };
  }

  estimateToolTokens(tool: Tool): number {
    const text = `${tool.name} ${tool.description} ${JSON.stringify(tool.parameters ?? {})}`;
    return Math.ceil(text.length / 4);
  }
}
