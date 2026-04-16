import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { AgentService } from './agent-service.js';
import { EchoAdapter } from './adapters/echo.js';
import { ClaudeCodeAdapter } from './adapters/claude-code.js';
import { GooseAdapter } from './adapters/goose.js';
import { CodexAdapter } from './adapters/codex.js';
import { CursorAdapter } from './adapters/cursor.js';
import type { AgentAdapter } from './adapters/types.js';
import type { ExecuteAgentRequest } from './types.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const PROTO_PATH = path.resolve(__dirname, '../../proto/forge/v1/agent.proto');

export function createServer(service: AgentService): grpc.Server {
  const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
    keepCase: true,
    longs: String,
    enums: String,
    defaults: true,
    oneofs: true,
  });

  const proto = grpc.loadPackageDefinition(packageDefinition) as any;
  const server = new grpc.Server();

  server.addService(proto.forge.v1.ForgeAgent.service, {
    ExecuteAgent: async (
      call: grpc.ServerUnaryCall<any, any>,
      callback: grpc.sendUnaryData<any>,
    ) => {
      try {
        const response = await service.executeAgent(
          call.request as ExecuteAgentRequest,
        );
        callback(null, response);
      } catch (err) {
        callback({
          code: grpc.status.INTERNAL,
          message: err instanceof Error ? err.message : String(err),
        });
      }
    },
  });

  return server;
}

function isInvokedDirectly(): boolean {
  const entry = process.argv[1];
  if (!entry) {
    return false;
  }
  try {
    return import.meta.url === pathToFileURL(path.resolve(entry)).href;
  } catch {
    return false;
  }
}

if (isInvokedDirectly()) {
  const port = process.env.FORGE_HARNESS_PORT ?? '50051';
  const defaultAdapter = process.env.FORGE_ADAPTER ?? 'echo';

  const adapters = new Map<string, AgentAdapter>([
    ['echo', new EchoAdapter()],
    ['claude', new ClaudeCodeAdapter()],
    ['goose', new GooseAdapter()],
    ['codex', new CodexAdapter()],
    ['cursor', new CursorAdapter()],
  ]);

  const service = new AgentService(adapters, { defaultAdapter });
  const server = createServer(service);

  server.bindAsync(
    `0.0.0.0:${port}`,
    grpc.ServerCredentials.createInsecure(),
    (err, boundPort) => {
      if (err) {
        console.error(`Failed to bind: ${err.message}`);
        process.exit(1);
      }
      console.log(
        `Forge Harness listening on port ${boundPort} (adapter: ${defaultAdapter})`,
      );
    },
  );
}
