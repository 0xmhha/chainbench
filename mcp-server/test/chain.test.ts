import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  AccountStateArgs,
  _accountStateHandler,
} from "../src/tools/chain.js";

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

// _accountStateHandler calls callWire() which calls resolveBinary() — there is
// no binaryPath option exposed at the handler boundary. Tests inject the mock
// by setting CHAINBENCH_NET_BIN, MOCK_SCRIPT (mock fixture scenario), and
// CHAINBENCH_DIR (cleared so resolveBinary can't pick a real binary). Saved
// values are restored in afterEach to keep tests isolated.
describe("chainbench_account_state handler", () => {
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

  it("_Happy_DefaultFields", async () => {
    const data = {
      address: "0x" + "a".repeat(40),
      balance: "0x10",
      nonce: "0x1",
      code: "0x",
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _accountStateHandler({
      network: "local",
      address: "0x" + "a".repeat(40),
    });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_Happy_StorageRequiresKey_RejectedAtBoundary", async () => {
    // Cross-field validation lives in the handler (not the zod schema) so
    // it returns a structured INVALID_ARGS response rather than throwing.
    // No wire spawn should occur — MOCK_SCRIPT is left empty so a stray
    // call would surface as a no-result-terminator error rather than pass.
    const out = await _accountStateHandler({
      network: "local",
      address: "0x" + "a".repeat(40),
      fields: ["storage"],
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("storage_key");
  });

  it("_BadAddress_RejectedAtBoundary", () => {
    // Bad-shaped address must throw at zod parse time so MCP SDK rejects
    // the call before the handler runs.
    expect(() =>
      AccountStateArgs.parse({
        network: "local",
        address: "0xnothex",
      }),
    ).toThrow();
  });

  it("_StrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'tag', 'from_block') trip a
    // structured INVALID_ARGS at the zod boundary instead of being silently
    // stripped — fail-loud beats silent strip on an LLM-driven surface.
    expect(() =>
      AccountStateArgs.parse({
        network: "local",
        address: "0x" + "a".repeat(40),
        tag: "latest",
      }),
    ).toThrow();
  });

  it("_RejectsNonArrayFields", () => {
    // fields must be an array of FIELD enum values; a bare string must not
    // be coerced to a single-element array.
    expect(() =>
      AccountStateArgs.parse({
        network: "local",
        address: "0x" + "a".repeat(40),
        fields: "balance",
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
          error: { code: "INVALID_ARGS", message: "unknown network" },
        }),
      },
    ]);
    const out = await _accountStateHandler({
      network: "nope",
      address: "0x" + "a".repeat(40),
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (INVALID_ARGS)");
    expect(out.content[0]?.text).toContain("unknown network");
  });
});
