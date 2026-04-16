import type { AgentAdapter, AgentAdapterRequest, AgentCapabilities } from './types.js';
import type { AgentEvent } from './events.js';

export class EchoAdapter implements AgentAdapter {
  readonly name = 'echo';

  async *execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent> {
    yield {
      type: 'done',
      content: `[echo] prompt="${req.prompt}" workingDir="${req.workingDirectory}"`,
    };
  }

  getCapabilities(): AgentCapabilities {
    return {
      streaming: false,
      interrupt: false,
      maxContextTokens: 200000,
      supportsTools: false,
      needsContextReset: false,
    };
  }

  async interrupt(): Promise<void> {}
}
