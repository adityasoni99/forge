export interface SprintContract {
  doneCriteria: string[];
  verificationSteps: string[];
  acceptanceThresholds: Record<string, number>;
}

export interface ContractEvalResult {
  passed: boolean;
  feedback: string;
  failedMetrics: string[];
}

export function evaluateAgainstContract(
  metrics: Record<string, number>,
  contract: SprintContract,
): ContractEvalResult {
  const failedMetrics: string[] = [];

  for (const [metric, threshold] of Object.entries(contract.acceptanceThresholds)) {
    const actual = metrics[metric];
    if (actual === undefined || actual < threshold) {
      failedMetrics.push(`${metric}: ${actual ?? 'missing'} < ${threshold}`);
    }
  }

  return {
    passed: failedMetrics.length === 0,
    feedback: failedMetrics.length > 0
      ? `Failed metrics: ${failedMetrics.join(', ')}`
      : 'All acceptance thresholds met',
    failedMetrics,
  };
}
