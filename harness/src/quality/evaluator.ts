export interface EvaluatorConfig {
  criteria: string[];
  fewShotExamples?: string;
  calibrationPath?: string;
  maxRetries: number;
}

export interface EvalResult {
  passed: boolean;
  score: number;
  feedback: string;
  retriesRemaining: number;
}

interface Executor {
  execute(ctx: unknown, prompt: string, config: unknown): Promise<string>;
}

export class SkepticalEvaluator {
  constructor(
    private readonly executor: Executor,
    private readonly config: EvaluatorConfig,
  ) {}

  async evaluate(output: string, threshold: number): Promise<EvalResult> {
    const prompt = this.buildEvalPrompt(output);
    const response = await this.executor.execute(null, prompt, {});
    const { score, feedback } = this.parseEvalResponse(response);

    return {
      passed: score >= threshold,
      score,
      feedback,
      retriesRemaining: this.config.maxRetries,
    };
  }

  private buildEvalPrompt(output: string): string {
    const criteriaList = this.config.criteria.map((c) => `- ${c}`).join('\n');
    return `You are a skeptical code evaluator. Evaluate the following output against these criteria:
${criteriaList}

Output to evaluate:
${output}

Respond in this exact format:
SCORE: <number 0-100>
FEEDBACK: <detailed feedback>`;
  }

  private parseEvalResponse(response: string): { score: number; feedback: string } {
    const scoreMatch = response.match(/^SCORE:\s*(\d+)/m);
    const feedbackMatch = response.match(/^FEEDBACK:\s*(.+)/ms);

    return {
      score: scoreMatch ? parseInt(scoreMatch[1], 10) : 0,
      feedback: feedbackMatch ? feedbackMatch[1].trim() : response,
    };
  }
}
