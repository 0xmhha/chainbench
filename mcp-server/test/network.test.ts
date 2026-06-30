import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  NetworkCapabilitiesArgs,
  _networkCapabilitiesHandler,
  NetworkAttachArgs,
  _networkAttachHandler,
} from "../src/tools/network.js";

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

// _networkCapabilitiesHandler calls callWire() which calls resolveBinary() —
// there is no binaryPath option exposed at the handler boundary. Tests inject
// the mock by setting CHAINBENCH_NET_BIN, MOCK_SCRIPT (mock fixture scenario),
// and CHAINBENCH_DIR (cleared so resolveBinary can't pick a real binary).
// Saved values are restored in afterEach to keep tests isolated.
describe("chainbench_network_capabilities handler", () => {
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

  it("_Happy_Local", async () => {
    // Local provider declares all 6 capabilities. The handler should
    // surface the JSON-pretty-printed wire data verbatim, so every cap
    // name appears in the response text.
    const data = {
      network: "local",
      capabilities: ["admin", "fs", "network-topology", "process", "rpc", "ws"],
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _networkCapabilitiesHandler({ network: "local" });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    const text = out.content[0]?.text ?? "";
    expect(text).toContain(JSON.stringify(data, null, 2));
    for (const cap of [
      "admin",
      "fs",
      "network-topology",
      "process",
      "rpc",
      "ws",
    ]) {
      expect(text).toContain(cap);
    }
  });

  it("_Happy_Remote", async () => {
    // Remote-attached networks declare only rpc + ws (no process control,
    // no filesystem access). The reduced cap set must be reflected in the
    // response so LLM callers can gate process-bound scenarios.
    const data = {
      network: "sepolia",
      capabilities: ["rpc", "ws"],
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _networkCapabilitiesHandler({ network: "sepolia" });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    const text = out.content[0]?.text ?? "";
    expect(text).toContain(JSON.stringify(data, null, 2));
    expect(text).toContain("sepolia");
    expect(text).toContain("rpc");
    expect(text).toContain("ws");
    // Negative checks: caps a remote network must NOT advertise.
    expect(text).not.toContain("process");
    expect(text).not.toContain("admin");
  });

  it("_Happy_Hybrid", async () => {
    // A hybrid network (mixed local + remote nodes) advertises only the
    // intersection of its providers' capabilities — the conservative lower
    // bound. The local∩remote intersection is computed on the wire side
    // (Go inferNetworkCapabilities); MCP passes the result through verbatim,
    // so the surface looks identical to a pure-remote network: rpc + ws only.
    const data = {
      network: "hybrid-example",
      capabilities: ["rpc", "ws"],
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _networkCapabilitiesHandler({ network: "hybrid-example" });
    expect(out.isError).toBeFalsy();
    const text = out.content[0]?.text ?? "";
    expect(text).toContain(JSON.stringify(data, null, 2));
    expect(text).toContain("hybrid-example");
    expect(text).toContain("rpc");
    expect(text).toContain("ws");
    // process/fs/admin are dropped by the intersection — a hybrid network
    // cannot guarantee them on every node, so they must not surface.
    expect(text).not.toContain("process");
    expect(text).not.toContain("admin");
  });

  it("_StrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'extra') trip a structured
    // INVALID_ARGS at the zod boundary instead of being silently stripped —
    // fail-loud beats silent strip on an LLM-driven surface.
    expect(() =>
      NetworkCapabilitiesArgs.parse({ network: "foo", extra: "bar" }),
    ).toThrow();
  });

  it("_WireFailure_PassedThrough", async () => {
    // Wire-level failures (e.g. unknown network) must surface as
    // isError:true with the structured error code/message preserved, so the
    // MCP client sees the same error envelope the wire emitted.
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "INVALID_ARGS", message: "unknown network" },
        }),
      },
    ]);
    const out = await _networkCapabilitiesHandler({ network: "nope" });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (INVALID_ARGS)");
    expect(out.content[0]?.text).toContain("unknown network");
  });
});

describe("chainbench_network_attach handler", () => {
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

  it("_Attach_Remote_Happy", async () => {
    const data = {
      name: "sepolia",
      chain_type: "ethereum",
      chain_id: 11155111,
      nodes: [{ id: "node1", provider: "remote", http: "https://rpc.example" }],
      rpc_url: "https://rpc.example",
      created: true,
    };
    process.env.MOCK_SCRIPT = script([
      { kind: "stdout", line: JSON.stringify({ type: "result", ok: true, data }) },
    ]);
    const out = await _networkAttachHandler({
      name: "sepolia",
      rpc_url: "https://rpc.example",
    });
    expect(out.isError).toBeFalsy();
    const text = out.content[0]?.text ?? "";
    expect(text).toContain("sepolia");
    expect(text).toContain("11155111");
  });

  it("_Attach_SSHRemote_SchemaAccepts", () => {
    // The ssh-remote shape (provider + ssh-password auth + provider_meta) must
    // pass zod so an LLM can construct it; .strict() still rejects typos.
    expect(() =>
      NetworkAttachArgs.parse({
        name: "sshnet",
        rpc_url: "http://127.0.0.1:8545",
        provider: "ssh-remote",
        auth: { type: "ssh-password", user: "deploy", host: "10.0.0.1", env: "SSH_PW" },
        provider_meta: { log_file: "/var/log/node.log", stop_cmd: "systemctl stop x" },
      }),
    ).not.toThrow();
  });

  it("_StrictRejectsUnknownKeys", () => {
    expect(() =>
      NetworkAttachArgs.parse({ name: "x", rpc_url: "u", bogus: 1 }),
    ).toThrow();
    // nested auth is also strict
    expect(() =>
      NetworkAttachArgs.parse({
        name: "x",
        rpc_url: "u",
        auth: { type: "api-key", env: "K", typo: true },
      }),
    ).toThrow();
  });

  it("_WireFailure_PassedThrough", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "INVALID_ARGS", message: "args.provider must be" },
        }),
      },
    ]);
    const out = await _networkAttachHandler({
      name: "x",
      rpc_url: "u",
      provider: "remote",
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (INVALID_ARGS)");
  });
});
