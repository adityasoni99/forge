import * as fs from 'node:fs/promises';
import * as path from 'node:path';

export interface RuleFile {
  name: string;
  content: string;
}

export interface LoadContextOptions {
  maxTokens?: number;
}

export interface LoadedContext {
  rootDoc: string;
  rootDocName: string;
  rules: RuleFile[];
  composePrompt(taskPrompt: string): string;
}

const ROOT_DOC_CANDIDATES = ['AGENTS.md', 'CLAUDE.md', 'README.md'];
const DEFAULT_MAX_TOKENS = 8000;

export function estimateTokens(text: string): number {
  return Math.ceil(text.length / 4);
}

export async function loadContext(
  workingDir: string,
  options?: LoadContextOptions,
): Promise<LoadedContext> {
  const maxTokens = options?.maxTokens ?? DEFAULT_MAX_TOKENS;
  const rootDoc = await loadRootDoc(workingDir);
  const allRules = await loadRules(workingDir);

  const rules = fitRulesToBudget(rootDoc, allRules, maxTokens);

  return {
    rootDoc: rootDoc.content,
    rootDocName: rootDoc.name,
    rules,
    composePrompt(taskPrompt: string): string {
      return buildComposedPrompt(rootDoc, rules, taskPrompt, maxTokens);
    },
  };
}

function fitRulesToBudget(
  rootDoc: { name: string; content: string },
  allRules: RuleFile[],
  maxTokens: number,
): RuleFile[] {
  const header = rootDoc.content
    ? `=== Project Context (${rootDoc.name}) ===\n${rootDoc.content}`
    : '';
  let used = estimateTokens(header);
  const out: RuleFile[] = [];
  if (allRules.length === 0) return out;
  used += estimateTokens('=== Directory Rules ===');
  for (const rule of allRules) {
    const chunk = `--- ${rule.name} ---\n${rule.content}`;
    const cost = estimateTokens(chunk) + 2;
    if (used + cost > maxTokens) break;
    out.push(rule);
    used += cost;
  }
  return out;
}

function buildComposedPrompt(
  rootDoc: { name: string; content: string },
  rules: RuleFile[],
  taskPrompt: string,
  maxTokens: number,
): string {
  const parts: string[] = [];
  if (rootDoc.content) {
    parts.push(`=== Project Context (${rootDoc.name}) ===\n${rootDoc.content}`);
  }
  if (rules.length > 0) {
    parts.push('=== Directory Rules ===');
    for (const rule of rules) {
      parts.push(`--- ${rule.name} ---\n${rule.content}`);
    }
  }
  parts.push(`=== Task ===\n${taskPrompt}`);
  let s = parts.join('\n\n');
  while (estimateTokens(s) > maxTokens && s.length > taskPrompt.length + 20) {
    s = s.slice(0, Math.floor(s.length * 0.9));
  }
  return s;
}

async function loadRootDoc(dir: string): Promise<{ name: string; content: string }> {
  for (const candidate of ROOT_DOC_CANDIDATES) {
    try {
      const content = await fs.readFile(path.join(dir, candidate), 'utf-8');
      return { name: candidate, content };
    } catch {
      continue;
    }
  }
  return { name: '', content: '' };
}

async function loadRules(dir: string): Promise<RuleFile[]> {
  const rulesDir = path.join(dir, '.forge', 'rules');
  try {
    const entries = await fs.readdir(rulesDir);
    const mdFiles = entries.filter((e) => e.endsWith('.md')).sort();
    const rules: RuleFile[] = [];
    for (const name of mdFiles) {
      const content = await fs.readFile(path.join(rulesDir, name), 'utf-8');
      rules.push({ name, content });
    }
    return rules;
  } catch {
    return [];
  }
}
