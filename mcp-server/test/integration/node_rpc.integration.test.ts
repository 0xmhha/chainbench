// node_rpc.integration.test.ts — first end-to-end test for the rerouted
// chainbench_node_rpc tool. Exercises the full path:
//
//   _nodeRpcHandler (TS)
//     -> callWire (TS spawn helper)
//       -> chainbench-net binary (Go)
//         -> rpc.Client.CallContext (Go)
//           -> Python http.server JSON-RPC mock
//
// Pattern mirrors tests/unit/tests/security-key-boundary.sh: spawn a Python
// JSON-RPC mock on an ephemeral port, then write state/pids.json + state/
// current-profile.yaml so the wire handler's resolveNode finds a "local"
// network whose node1 HTTP endpoint points at the mock.
//
// The suite skips itself when network/bin/chainbench-net is missing so CI
// without a Go toolchain still runs the unit suite. Local dev iteration:
//   cd network && go build -o bin/chainbench-net ./cmd/chainbench-net
//
// Sprint 5c.3 Task 4. Sprint 5c.4+ will add scenarios for the lifecycle
// reroutes against the same scaffolding.

import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { spawn, type ChildProcess } from "node:child_process";
import { existsSync, mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { resolve as resolvePath, dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import net from "node:net";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const REPO_ROOT = resolvePath(__dirname, "../../..");
const BINARY = join(REPO_ROOT, "network/bin/chainbench-net");
const HAS_BINARY = existsSync(BINARY);

// Pick an ephemeral port by opening a 0-port listener and reading the OS-
// assigned number. The listener is closed before the caller spawns its
// real server; nothing else should grab the port in the millisecond gap on
// a vitest worker, so this matches the pattern used by the bash harness
// (python -c 'socket s=...; s.bind(...,0); print(s.getsockname()[1])').
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
// retries. Bounded loop + 100ms granularity matches the bash harness.
async function waitForPort(port: number, timeoutMs: number): Promise<void> {
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
  throw new Error(`port ${port} did not accept connections within ${timeoutMs}ms`);
}

const PYTHON_MOCK = `
import http.server
import json
import sys

port = int(sys.argv[1])

class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try:
            req = json.loads(self.rfile.read(n))
        except Exception:
            req = {}
        rid = req.get('id')
        method = req.get('method')
        if method == 'eth_chainId':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x539'}
        elif method == 'eth_blockNumber':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0xdeadbeef'}
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

describe.skipIf(!HAS_BINARY)("integration: chainbench_node_rpc real binary", () => {
  let mockProc: ChildProcess | undefined;
  let mockPort = 0;
  let stateDir = "";
  // Snapshot env so afterAll can fully restore — vi doesn't auto-revert
  // process.env between tests in a describe block.
  const envBackup: Record<string, string | undefined> = {};

  beforeAll(async () => {
    mockPort = await pickFreePort();

    mockProc = spawn("python3", ["-c", PYTHON_MOCK, String(mockPort)], {
      stdio: ["ignore", "pipe", "pipe"],
    });
    // Drain stdout/stderr so the buffer never blocks the child.
    mockProc.stdout?.on("data", () => {});
    mockProc.stderr?.on("data", () => {});

    await waitForPort(mockPort, 3000);

    stateDir = mkdtempSync(join(tmpdir(), "cb-int-rpc-"));
    mkdirSync(join(stateDir, "networks"), { recursive: true });

    // Seed state/pids.json so resolveNode finds node1 pointing at the mock.
    // Only http_port matters for node.rpc — ws_port / log_file / type are
    // populated to mirror a real chainbench start, but never read.
    const pids = {
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

    // current-profile.yaml — chain_id matches pids.chain_id; chain.type is
    // optional (defaults to "stablenet" downstream) but we set it explicitly
    // so probe-style code paths stay deterministic.
    const profile = `name: integration-test
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
    writeFileSync(
      join(stateDir, "current-profile.yaml"),
      profile,
      "utf-8",
    );

    // Wire-helper env: binary path + state dir. Snapshot prior values so
    // afterAll can restore them and not bleed into other test files in the
    // same vitest worker.
    for (const k of ["CHAINBENCH_NET_BIN", "CHAINBENCH_STATE_DIR"]) {
      envBackup[k] = process.env[k];
    }
    process.env.CHAINBENCH_NET_BIN = BINARY;
    process.env.CHAINBENCH_STATE_DIR = stateDir;
  }, 15000);

  afterAll(() => {
    if (mockProc && mockProc.pid !== undefined) {
      try {
        mockProc.kill("SIGTERM");
      } catch {
        // best-effort
      }
    }
    if (stateDir) {
      rmSync(stateDir, { recursive: true, force: true });
    }
    // Restore env exactly as it was.
    for (const [k, v] of Object.entries(envBackup)) {
      if (v === undefined) delete process.env[k];
      else process.env[k] = v;
    }
  });

  it("eth_blockNumber happy path through real binary + Python mock", async () => {
    const { _nodeRpcHandler } = await import("../../src/tools/node.js");

    const out = await _nodeRpcHandler({
      node: 1,
      method: "eth_blockNumber",
    });

    expect(out.isError).not.toBe(true);
    const text = out.content[0]?.text ?? "";
    // The mock returns 0xdeadbeef; formatWireResult passes the wire data
    // straight through, so the rendered text must contain the hex result.
    expect(text).toContain("0xdeadbeef");
  });
});
