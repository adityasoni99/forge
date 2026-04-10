import type {
  AgentAdapter,
  AgentAdapterRequest,
  AgentAdapterResponse,
  AgentCapabilities,
} from './types.js';

export class EchoAdapter implements AgentAdapter {
  getCapabilities(): AgentCapabilities {
    return { streaming: false, interrupt: false };
  }

  async execute(req: AgentAdapterRequest): Promise<AgentAdapterResponse> {
    return {
      output: `[echo] prompt="${req.prompt}" workingDir="${req.workingDirectory}"`,
      success: true,
    };
  }
}
