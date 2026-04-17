import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import type { SessionEvent, SessionEventType } from './types.js';

export interface EmitOptions {
  type: SessionEventType;
  nodeID?: string;
  data?: Record<string, unknown>;
}

export interface GetEventsOptions {
  afterID?: string;
}

export class SessionEventEmitter {
  constructor(private readonly sessionsDir: string) {}

  private logPath(runID: string): string {
    return path.join(this.sessionsDir, `${runID}.jsonl`);
  }

  async emit(runID: string, options: EmitOptions): Promise<SessionEvent> {
    await fs.mkdir(this.sessionsDir, { recursive: true });

    const event: SessionEvent = {
      id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      timestamp: new Date().toISOString(),
      runID,
      type: options.type,
      nodeID: options.nodeID,
      data: options.data,
    };

    const line = JSON.stringify(event) + '\n';
    await fs.appendFile(this.logPath(runID), line, 'utf-8');
    return event;
  }

  async getEvents(runID: string, options?: GetEventsOptions): Promise<SessionEvent[]> {
    try {
      const content = await fs.readFile(this.logPath(runID), 'utf-8');
      const lines = content.trim().split('\n').filter(Boolean);
      let events = lines.map((line) => JSON.parse(line) as SessionEvent);

      if (options?.afterID) {
        const idx = events.findIndex((e) => e.id === options.afterID);
        if (idx >= 0) {
          events = events.slice(idx + 1);
        }
      }

      return events;
    } catch {
      return [];
    }
  }
}
