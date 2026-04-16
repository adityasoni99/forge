import type { AgentAdapter, AgentAdapterRequest, AgentCapabilities, SyncAgentAdapter } from './types.js';
import type { AgentEvent } from './events.js';

const DEFAULT_CAPABILITIES: AgentCapabilities = {
  streaming: false,
  interrupt: false,
  maxContextTokens: 200000,
  supportsTools: false,
  needsContextReset: false,
};

export class SyncAdapterWrapper implements AgentAdapter {
  readonly name: string;
  private readonly inner: SyncAgentAdapter;
  private readonly caps: AgentCapabilities;

  constructor(name: string, inner: SyncAgentAdapter, capabilityOverrides?: Partial<AgentCapabilities>) {
    this.name = name;
    this.inner = inner;
    const partialCaps = inner.getCapabilities?.() ?? {};
    this.caps = { ...DEFAULT_CAPABILITIES, ...partialCaps, ...capabilityOverrides };
  }

  async *execute(req: AgentAdapterRequest): AsyncIterable<AgentEvent> {
    const result = await this.inner.execute(req);
    if (result.success) {
      yield { type: 'done', content: result.output };
    } else {
      yield { type: 'error', content: result.error ?? 'unknown error' };
    }
  }

  getCapabilities(): AgentCapabilities {
    return this.caps;
  }

  async interrupt(): Promise<void> {
    // sync adapters complete in a single await; nothing to interrupt
  }
}
