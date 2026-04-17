import type { CommandDefinition } from './types.js';

export const COMMANDS: Map<string, CommandDefinition> = new Map([
  ['run', {
    name: 'run',
    description: 'Run a task using the standard implementation blueprint',
    blueprintName: 'standard-implementation',
    promptTemplate: [
      'Execute the following task end-to-end: plan the approach, write the code,',
      'run linters and tests, fix any failures, then commit the result.',
      '',
      'Task: {{task}}',
    ].join('\n'),
  }],
  ['fix', {
    name: 'fix',
    description: 'Fix a bug using the bug-fix blueprint',
    blueprintName: 'bug-fix',
    promptTemplate: [
      'Reproduce and fix the following bug. Run tests to verify the fix,',
      'then commit the result.',
      '',
      'Bug: {{task}}',
      '{{#filePath}}File: {{filePath}}{{/filePath}}',
    ].join('\n'),
  }],
  ['plan', {
    name: 'plan',
    description: 'Create an action plan without executing it',
    blueprintName: 'standard-implementation',
    promptTemplate: [
      'Create a detailed action plan for the following task.',
      'Do NOT build or execute — only plan. Output the plan as a structured markdown document',
      'with specific file paths, code snippets, and test strategy.',
      '',
      'Task: {{task}}',
    ].join('\n'),
    planOnly: true,
  }],
]);

export function resolveCommand(name: string): CommandDefinition | undefined {
  return COMMANDS.get(name);
}

export interface PromptContext {
  filePath?: string;
  errorOutput?: string;
  selection?: string;
}

function escapeTemplateTokens(input: string): string {
  return input
    .replace(/\{\{/g, '{ {')
    .replace(/\}\}/g, '} }');
}

export function buildTaskPrompt(
  command: CommandDefinition,
  task: string,
  context?: PromptContext,
): string {
  let prompt = command.promptTemplate.replace(/\{\{task\}\}/g, escapeTemplateTokens(task));

  if (context?.filePath) {
    const safeFilePath = escapeTemplateTokens(context.filePath);
    prompt = prompt
      .replace(/\{\{#filePath\}\}(.*?)\{\{\/filePath\}\}/gs, '$1')
      .replace(/\{\{filePath\}\}/g, safeFilePath);
  } else {
    prompt = prompt.replace(/\{\{#filePath\}\}.*?\{\{\/filePath\}\}/gs, '');
  }

  if (context?.errorOutput) {
    prompt += `\n\nError output:\n${escapeTemplateTokens(context.errorOutput)}`;
  }
  if (context?.selection) {
    prompt += `\n\nSelected code:\n${escapeTemplateTokens(context.selection)}`;
  }

  return prompt.trim();
}
