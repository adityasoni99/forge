export enum PromptLayer {
  Override = 0,
  Coordinator = 1,
  AgentSpecific = 2,
  ProjectRules = 3,
  DefaultBaseline = 4,
}

export interface PromptLayerEntry {
  layer: PromptLayer;
  label: string;
  content: string;
}

export interface PromptStackOptions {
  maxTokens: number;
}

function estimateTokens(text: string): number {
  return Math.ceil(text.length / 4);
}

export function composePromptStack(
  layers: PromptLayerEntry[],
  taskPrompt: string,
  options: PromptStackOptions,
): string {
  const sorted = [...layers].sort((a, b) => a.layer - b.layer);

  const taskTokens = estimateTokens(taskPrompt) + 10;
  let remainingBudget = options.maxTokens - taskTokens;

  const included: PromptLayerEntry[] = [];

  for (const entry of sorted) {
    const cost = estimateTokens(entry.content) + estimateTokens(entry.label) + 5;

    if (entry.layer <= PromptLayer.Coordinator) {
      included.push(entry);
      remainingBudget -= cost;
      continue;
    }

    if (cost <= remainingBudget) {
      included.push(entry);
      remainingBudget -= cost;
    }
  }

  const parts: string[] = [];
  for (const entry of included) {
    parts.push(`=== ${entry.label} ===\n${entry.content}`);
  }
  parts.push(`=== Task ===\n${taskPrompt}`);

  return parts.join('\n\n');
}
