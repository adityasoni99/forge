import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import type { SessionEvent, DerivedRule } from './types.js';

export interface RuleExecutor {
  execute(ctx: unknown, prompt: string, config: unknown): Promise<string>;
}

export function buildFailureAnalysisPrompt(events: SessionEvent[]): string {
  const errorEvents = events.filter((e) => e.type === 'error');
  const lastError = errorEvents[errorEvents.length - 1];
  const errorMsg = lastError?.data?.message ?? lastError?.data?.msg ?? 'unknown error';

  const eventSummary = events
    .map((e) => `[${e.type}] ${e.nodeID ?? ''} ${JSON.stringify(e.data ?? {})}`)
    .join('\n');

  return `Analyze this agent run failure and derive a rule that would prevent it in the future.

## Failure chain (session events):
${eventSummary}

## Final error:
${errorMsg}

Respond in this exact format:
CATEGORY: <one-word category like testing, linting, security, architecture>
NAME: <kebab-case rule name>
BODY:
<The rule text in markdown. 2-5 sentences. Actionable and specific.>`;
}

export function parseRuleResponse(
  output: string,
  runID: string,
  errorSummary: string,
): DerivedRule | null {
  const categoryMatch = output.match(/^CATEGORY:\s*(.+)$/m);
  const nameMatch = output.match(/^NAME:\s*(.+)$/m);
  const bodyMatch = output.match(/^BODY:\n([\s\S]+)$/m);

  if (!categoryMatch || !nameMatch || !bodyMatch) {
    return null;
  }

  return {
    category: categoryMatch[1].trim().toLowerCase(),
    name: nameMatch[1].trim(),
    body: bodyMatch[1].trim(),
    sourceRunID: runID,
    sourceError: errorSummary,
  };
}

export async function deriveRuleFromFailure(
  events: SessionEvent[],
  executor: RuleExecutor,
): Promise<DerivedRule | null> {
  try {
    const prompt = buildFailureAnalysisPrompt(events);
    const output = await executor.execute(null, prompt, {});

    const errorEvents = events.filter((e) => e.type === 'error');
    const lastError = errorEvents[errorEvents.length - 1];
    const errorSummary = String(lastError?.data?.message ?? lastError?.data?.msg ?? 'unknown');
    const runID = events[0]?.runID ?? 'unknown';

    return parseRuleResponse(output, runID, errorSummary);
  } catch {
    return null;
  }
}

export async function writeRuleToRepo(
  repoDir: string,
  rule: DerivedRule,
): Promise<string> {
  const rulesDir = path.join(repoDir, '.forge', 'rules', rule.category);
  await fs.mkdir(rulesDir, { recursive: true });

  const filePath = path.join(rulesDir, `${rule.name}.md`);
  const content = `# ${rule.name}

> Derived from run ${rule.sourceRunID} (error: ${rule.sourceError})

${rule.body}
`;
  await fs.writeFile(filePath, content, 'utf-8');
  return filePath;
}
