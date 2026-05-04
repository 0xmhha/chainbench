// _harness.ts — reusable real-binary integration test setup.
//
// Extracted from node_rpc.integration.test.ts (Sprint 5c.4.1 Task 0). The
// inline scaffolding (port allocation + Python JSON-RPC mock + temp state
// dir + env snapshot/restore + cleanup) is now parameterised via
// RealBinaryHarnessOptions so subsequent integration tests (Sprint 5c.4+)
// can reuse the same setup without copy-paste.
//
// Two strengthenings vs the inline original (Sprint 5c.3 final-review I1
// + I2):
//   - teardown() awaits SIGTERM exit (1s ceiling) then escalates to
//     SIGKILL when the child neither exited nor was signalled — closes
//     the cleanup-await gap.
//   - Mock stderr is captured to a buffer rather than blackholed; on
//     waitForPort timeout the buffer's first 500 chars are appended to
//     the error message — closes the port-race diagnostics gap.

import { spawn, type ChildProcess } from "node:child_process";
import {
  chmodSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { dirname, join, resolve as resolvePath } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import net from "node:net";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const REPO_ROOT = resolvePath(__dirname, "../../..");
const BINARY = join(REPO_ROOT, "network/bin/chainbench-net");

// Env keys the harness mutates and must restore on teardown.
const ENV_KEYS = [
  "CHAINBENCH_NET_BIN",
  "CHAINBENCH_STATE_DIR",
  "CHAINBENCH_DIR",
  "MOCK_SCRIPT",
] as const;

export interface RealBinaryHarnessOptions {
  /**
   * JSON-RPC method handlers keyed by method name. Each handler returns
   * the value placed in the JSON-RPC response's `result` field. Methods
   * not present in the map produce `{error:{code:-32601,...}}` (method
   * not found). Defaults: eth_chainId -> "0x539", eth_blockNumber ->
   * "0xdeadbeef".
   */
  rpcHandlers?: Record<
    string,
    (req: { id: number | string; params?: unknown[] }) => unknown
  >;
  /** Override pids.json content for the seeded local network. */
  pidsOverride?: object;
  /** Override current-profile.yaml content as raw YAML text. Default seeds a 1-node local network. */
  profileOverride?: string;
  /**
   * Optional fake chainbench.sh script body. When provided the harness
   * writes it to <stateDir>/chainbench.sh + chmod 0755 and exports
   * CHAINBENCH_DIR=<stateDir> so the chainbench-net binary's os/exec
   * calls hit the fake script instead of the real one. Used by the
   * lifecycle integration tests (Sprint 5c.4.1 Task 4) where the wire
   * handler spawns chainbench.sh.
   */
  fakeChainbenchScript?: string;
}

export interface RealBinaryHarness {
  binaryPath: string;
  stateDir: string;
  mockPort: number;
  /** Cleanup handle — must be awaited in afterAll. */
  teardown: () => Promise<void>;
}

export function hasBinary(): boolean {
  return existsSync(BINARY);
}

// Pick an ephemeral port by opening a 0-port listener and reading the OS-
// assigned number. The listener is closed before the caller spawns its
// real server; nothing else should grab the port in the millisecond gap on
// a vitest worker, matching the bash harness pattern.
async function pickFreePort(): Promise<number> {
  return new Promise((resolveP, rejectP) => {
    const srv = net.createServer();
    srv.unref();
    srv.on("error", rejectP);
    srv.listen(0, "127.0.0.1", () => {
      const addr = srv.address();
      if (addr && typeof addr === "object") {
        const port = addr.port;
        srv.close(() => resolveP(port));
      } else {
        srv.close();
        rejectP(new Error("could not pick free port"));
      }
    });
  });
}

// Poll a port until something accepts a TCP connection or we run out of
// retries. Bounded loop + 50ms granularity matches the bash harness; on
// timeout we surface the mock's stderr so port-race failures are
// diagnosable.
async function waitForPort(
  port: number,
  timeoutMs: number,
  stderrSnapshot: () => string,
): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const ok = await new Promise<boolean>((r) => {
      const sock = new net.Socket();
      sock.setTimeout(200);
      sock.once("connect", () => {
        sock.destroy();
        r(true);
      });
      sock.once("error", () => {
        sock.destroy();
        r(false);
      });
      sock.once("timeout", () => {
        sock.destroy();
        r(false);
      });
      sock.connect(port, "127.0.0.1");
    });
    if (ok) return;
    await new Promise((r) => setTimeout(r, 50));
  }
  const tail = stderrSnapshot().slice(0, 500);
  const suffix = tail.length > 0 ? `; mock stderr: ${tail}` : "";
  throw new Error(
    `port ${port} did not accept connections within ${timeoutMs}ms${suffix}`,
  );
}

// Build the Python JSON-RPC mock script. The handler dispatch table is
// generated from the user-supplied rpcHandlers (defaulted) and embedded
// as a JSON literal so the mock subprocess needs no extra arguments
// beyond the port.
function buildMockScript(
  handlers: Record<
    string,
    (req: { id: number | string; params?: unknown[] }) => unknown
  >,
): string {
  // Build a static JSON map: method -> result (eagerly evaluated). We
  // call each handler with a synthetic empty request; the mock applies
  // the captured `id` server-side. This keeps the Python child code
  // dependency-free while still letting tests parameterise responses.
  const table: Record<string, unknown> = {};
  for (const [method, fn] of Object.entries(handlers)) {
    table[method] = fn({ id: 0 });
  }
  const tableJson = JSON.stringify(table);
  return `
import http.server
import json
import sys

port = int(sys.argv[1])
TABLE = json.loads(${JSON.stringify(tableJson)})

class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try:
            req = json.loads(self.rfile.read(n))
        except Exception:
            req = {}
        rid = req.get('id')
        method = req.get('method')
        if method in TABLE:
            body = {'jsonrpc': '2.0', 'id': rid, 'result': TABLE[method]}
        else:
            body = {'jsonrpc': '2.0', 'id': rid,
                    'error': {'code': -32601, 'message': 'method not found'}}
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *args, **kwargs):
        pass

http.server.HTTPServer(('127.0.0.1', port), H).serve_forever()
`.trim();
}

const DEFAULT_HANDLERS: Record<
  string,
  (req: { id: number | string; params?: unknown[] }) => unknown
> = {
  eth_chainId: () => "0x539",
  eth_blockNumber: () => "0xdeadbeef",
};

// SIGTERM → race(exited, 1s) → SIGKILL fallback shared by the setup-failure
// catch path and the main teardown(). Safe to call when mockProc is null/
// undefined or already dead; all kill() calls swallow EPERM/ESRCH (Sprint
// 5c.4.1 Task 0 review I2 fix — previously the catch path only fired
// SIGTERM and fell through, leaking a stuck mock past partial-setup
// failures).
async function killMockGracefully(
  mockProc: ChildProcess | null | undefined,
): Promise<void> {
  if (!mockProc || mockProc.pid === undefined) return;
  const exited = new Promise<void>((r) => {
    mockProc.once("exit", () => r());
  });
  try {
    mockProc.kill("SIGTERM");
  } catch {
    // best-effort: process may already be dead
  }
  await Promise.race([
    exited,
    new Promise<void>((r) => setTimeout(r, 1000)),
  ]);
  if (mockProc.exitCode === null && mockProc.signalCode === null) {
    try {
      mockProc.kill("SIGKILL");
    } catch {
      // best-effort
    }
  }
}

export async function setupRealBinaryHarness(
  opts: RealBinaryHarnessOptions = {},
): Promise<RealBinaryHarness> {
  if (!hasBinary()) {
    throw new Error(
      `chainbench-net binary missing at ${BINARY}; build it before invoking the harness or guard the describe block with describe.skipIf(!hasBinary())`,
    );
  }

  // Snapshot env before any mutation so teardown can fully restore.
  const envBackup: Record<string, string | undefined> = {};
  for (const k of ENV_KEYS) {
    envBackup[k] = process.env[k];
  }

  const mockPort = await pickFreePort();
  const handlers = opts.rpcHandlers ?? DEFAULT_HANDLERS;
  const mockScript = buildMockScript(handlers);

  let stderrBuf = "";
  const mockProc: ChildProcess = spawn(
    "python3",
    ["-c", mockScript, String(mockPort)],
    { stdio: ["ignore", "pipe", "pipe"] },
  );
  // Drain stdout so the buffer never blocks the child; capture stderr
  // so waitForPort timeouts can include diagnostic context (Sprint 5c.3
  // I2 fix).
  mockProc.stdout?.on("data", () => {});
  mockProc.stderr?.on("data", (b: Buffer) => {
    stderrBuf += b.toString("utf-8");
  });

  const stateDir = mkdtempSync(join(tmpdir(), "cb-int-harness-"));

  // Wrap the rest in try/catch so a setup failure tears down the mock
  // and temp dir before propagating — vitest's beforeAll-failure cleanup
  // does not run afterAll on the same describe block.
  try {
    await waitForPort(mockPort, 3000, () => stderrBuf);

    mkdirSync(join(stateDir, "networks"), { recursive: true });

    const pids = opts.pidsOverride ?? {
      chain_id: "1337",
      profile: "integration-test",
      started_at: new Date().toISOString(),
      nodes: {
        "1": {
          pid: 0,
          type: "validator",
          p2p_port: 0,
          http_port: mockPort,
          ws_port: mockPort + 1,
          auth_port: 0,
          metrics_port: 0,
          status: "running",
          log_file: join(stateDir, "node1.log"),
          binary: "",
          datadir: "",
        },
      },
    };
    writeFileSync(
      join(stateDir, "pids.json"),
      JSON.stringify(pids, null, 2),
      "utf-8",
    );

    const profileText =
      opts.profileOverride !== undefined
        ? opts.profileOverride
        : `name: integration-test
chain:
  type: ethereum
  chain_id: 1337
  network_id: 1337
nodes:
  validators: 1
  endpoints: 0
ports:
  base_p2p: 0
  base_http: ${mockPort}
  base_ws: ${mockPort + 1}
`;
    writeFileSync(join(stateDir, "current-profile.yaml"), profileText, "utf-8");

    if (opts.fakeChainbenchScript !== undefined) {
      const scriptPath = join(stateDir, "chainbench.sh");
      writeFileSync(scriptPath, opts.fakeChainbenchScript, "utf-8");
      chmodSync(scriptPath, 0o755);
      process.env.CHAINBENCH_DIR = stateDir;
    }

    process.env.CHAINBENCH_NET_BIN = BINARY;
    process.env.CHAINBENCH_STATE_DIR = stateDir;
  } catch (err) {
    // Partial setup: kill the mock + clean state dir so the failure
    // message reaches vitest with no resource leak. Uses the same
    // SIGTERM → race → SIGKILL helper as teardown() to avoid leaking a
    // stuck mock when setup fails after spawn (e.g. waitForPort timeout
    // because the mock bound the wrong interface or the Python import
    // hung).
    await killMockGracefully(mockProc);
    rmSync(stateDir, { recursive: true, force: true });
    for (const [k, v] of Object.entries(envBackup)) {
      if (v === undefined) delete process.env[k];
      else process.env[k] = v;
    }
    throw err;
  }

  const teardown = async (): Promise<void> => {
    await killMockGracefully(mockProc);
    rmSync(stateDir, { recursive: true, force: true });
    for (const [k, v] of Object.entries(envBackup)) {
      if (v === undefined) delete process.env[k];
      else process.env[k] = v;
    }
  };

  return {
    binaryPath: BINARY,
    stateDir,
    mockPort,
    teardown,
  };
}
