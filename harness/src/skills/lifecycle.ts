import type { Skill } from './types.js';

export interface TestCase {
  input: string;
  expectedContains: string;
}

export interface TestCaseResult {
  input: string;
  passed: boolean;
  composedPrompt: string;
}

export interface EvalResult {
  passed: boolean;
  passRate: number;
  results: TestCaseResult[];
}

export interface ComparisonResult {
  winnerName: string;
  scoreA: number;
  scoreB: number;
  details: string;
}

export class SkillLifecycle {
  // Stub: checks whether expectedContains appears in the composed prompt
  // (skill body + task input). Will be replaced with LLM-powered evaluation
  // once the agent executor is wired through the lifecycle.
  async evaluate(skill: Skill, testCases: TestCase[]): Promise<EvalResult> {
    if (testCases.length === 0) {
      return { passed: true, passRate: 1.0, results: [] };
    }

    const results: TestCaseResult[] = [];
    let passCount = 0;

    for (const tc of testCases) {
      const composed = `${skill.body}\n\n=== Task ===\n${tc.input}`;
      const passed = composed.toLowerCase().includes(tc.expectedContains.toLowerCase());
      if (passed) passCount++;
      results.push({ input: tc.input, passed, composedPrompt: composed });
    }

    const passRate = passCount / testCases.length;
    return { passed: passRate === 1.0, passRate, results };
  }

  promote(skill: Skill, newVersion: string, newEvalScore: number): Skill {
    return {
      ...skill,
      version: newVersion,
      evalScore: newEvalScore,
    };
  }

  async compare(skillA: Skill, skillB: Skill, testCases: TestCase[]): Promise<ComparisonResult> {
    const resultA = await this.evaluate(skillA, testCases);
    const resultB = await this.evaluate(skillB, testCases);

    const winnerName = resultA.passRate >= resultB.passRate ? skillA.name : skillB.name;
    return {
      winnerName,
      scoreA: resultA.passRate,
      scoreB: resultB.passRate,
      details: `${skillA.name}: ${resultA.passRate.toFixed(2)} vs ${skillB.name}: ${resultB.passRate.toFixed(2)}`,
    };
  }
}
