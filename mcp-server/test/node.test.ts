import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { NodeRpcArgs, _nodeRpcHandler } from "../src/tools/node.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const MOCK_BIN = resolve(__dirname, "fixtures/mock-chainbench-net.mjs");

type Step =
  | { kind: "stdout"; line: string; delayMs?: number }
  | { kind: "stderr"; line: string; delayMs?: number }
  | { kind: "exit"; code?: number; delayMs?: number };

function script(steps: Step[]): string {
  return Buffer.from(JSON.stringify(steps), "utf-8").toString("base64");
}

beforeAll(() => {
  // Ensure the mock fixture is executable on a fresh checkout.
  if (existsSync(MOCK_BIN)) {
    chmodSync(MOCK_BIN, 0o755);
  }
});

// _nodeRpcHandler calls callWire() which calls resolveBinary() — there is no
// binaryPath option exposed at the handler boundary. Tests inject the mock by
// setting CHAINBENCH_NET_BIN, MOCK_SCRIPT (mock fixture scenario), and
// CHAINBENCH_DIR (cleared so resolveBinary can't pick a real binary). Saved
// values are restored in afterEach to keep tests isolated.
describe("chainbench_node_rpc handler", () => {
  let savedBin: string | undefined;
  let savedScript: string | undefined;
  let savedDir: string | undefined;

  beforeEach(() => {
    savedBin = process.env.CHAINBENCH_NET_BIN;
    savedScript = process.env.MOCK_SCRIPT;
    savedDir = process.env.CHAINBENCH_DIR;
    process.env.CHAINBENCH_NET_BIN = MOCK_BIN;
    delete process.env.CHAINBENCH_DIR;
  });

  afterEach(() => {
    if (savedBin === undefined) delete process.env.CHAINBENCH_NET_BIN;
    else process.env.CHAINBENCH_NET_BIN = savedBin;
    if (savedScript === undefined) delete process.env.MOCK_SCRIPT;
    else process.env.MOCK_SCRIPT = savedScript;
    if (savedDir === undefined) delete process.env.CHAINBENCH_DIR;
    else process.env.CHAINBENCH_DIR = savedDir;
  });

  it("_Happy_NodeRpc", async () => {
    const data = { result: "0x10" };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _nodeRpcHandler({
      node: 1,
      method: "eth_blockNumber",
    });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain("0x10");
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_BadMethod_RejectedAtBoundary", () => {
    // Invalid method names (e.g. containing whitespace) must throw at zod
    // parse time so the MCP SDK rejects the call before the handler runs —
    // no wire spawn occurs.
    expect(() =>
      NodeRpcArgs.parse({ node: 1, method: "eth blockNumber" }),
    ).toThrow();
  });

  it("_StrictRejectsUnknownKeys", () => {
    expect(() =>
      NodeRpcArgs.parse({
        node: 1,
        method: "eth_blockNumber",
        tag: "latest", // unknown key — strict() should reject
      } as any),
    ).toThrow();
  });

  it("_BadParams_NotArray_ReturnsIsError", async () => {
    // params parses to a non-array JSON value — handler returns errorResp
    // before any spawn. MOCK_SCRIPT intentionally empty so a stray wire call
    // would surface as a no-result-terminator error rather than pass.
    const out = await _nodeRpcHandler({
      node: 1,
      method: "eth_blockNumber",
      params: '{"x":1}',
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("JSON array");
  });

  it("_BadParams_NotJSON_ReturnsIsError", async () => {
    // params is malformed JSON — handler returns errorResp before any spawn.
    const out = await _nodeRpcHandler({
      node: 1,
      method: "eth_blockNumber",
      params: "not-json",
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("valid JSON");
  });

  it("_WireFailure_PassedThrough", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "UPSTREAM_ERROR", message: "rpc dial failed" },
        }),
      },
    ]);
    const out = await _nodeRpcHandler({
      node: 1,
      method: "eth_blockNumber",
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (UPSTREAM_ERROR)");
    expect(out.content[0]?.text).toContain("rpc dial failed");
  });
});
