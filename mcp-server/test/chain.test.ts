import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  AccountStateArgs,
  TxSendArgs,
  _accountStateHandler,
  _buildTxSendWireArgs,
  _txSendHandler,
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

// _buildTxSendWireArgs is the pure cross-field validator. The mode/fee-field
// exclusivity rule cannot be expressed cleanly in a zod schema (a discriminated
// union on `mode` would force two parallel schemas and lose the shared field
// metadata), so the boundary check lives here and is exercised directly by
// the first four tests below — no spawn, no env setup needed. The handler
// black-box test at the bottom covers the wire passthrough.
describe("chainbench_tx_send arg builder", () => {
  const baseLegacy = {
    network: "local",
    signer: "alice",
    mode: "legacy" as const,
    to: "0x" + "b".repeat(40),
    value: "0x10",
    gas_price: "0x3b9aca00",
  };

  const base1559 = {
    network: "local",
    signer: "alice",
    mode: "1559" as const,
    to: "0x" + "b".repeat(40),
    value: "0x10",
    max_fee_per_gas: "0x3b9aca00",
    max_priority_fee_per_gas: "0x3b9aca00",
  };

  it("_Happy_Legacy", () => {
    const built = _buildTxSendWireArgs(baseLegacy);
    expect("wireArgs" in built).toBe(true);
    if (!("wireArgs" in built)) return;
    expect(built.wireArgs.gas_price).toBe("0x3b9aca00");
    expect(built.wireArgs.max_fee_per_gas).toBeUndefined();
    expect(built.wireArgs.max_priority_fee_per_gas).toBeUndefined();
    // The synthetic 'mode' key never reaches the wire envelope.
    expect(built.wireArgs.mode).toBeUndefined();
    expect(built.wireArgs.network).toBe("local");
    expect(built.wireArgs.signer).toBe("alice");
    expect(built.wireArgs.to).toBe("0x" + "b".repeat(40));
    expect(built.wireArgs.value).toBe("0x10");
  });

  it("_Happy_1559", () => {
    const built = _buildTxSendWireArgs(base1559);
    expect("wireArgs" in built).toBe(true);
    if (!("wireArgs" in built)) return;
    expect(built.wireArgs.max_fee_per_gas).toBe("0x3b9aca00");
    expect(built.wireArgs.max_priority_fee_per_gas).toBe("0x3b9aca00");
    expect(built.wireArgs.gas_price).toBeUndefined();
    expect(built.wireArgs.mode).toBeUndefined();
    expect(built.wireArgs.network).toBe("local");
    expect(built.wireArgs.signer).toBe("alice");
  });

  it("_LegacyWithMaxFee_Rejected", () => {
    const built = _buildTxSendWireArgs({
      ...baseLegacy,
      max_fee_per_gas: "0x3b9aca00",
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) return;
    expect(built.error).toContain("legacy");
    expect(built.error).toContain("max_fee_per_gas");
  });

  it("_1559WithoutMaxFee_Rejected", () => {
    // max_fee_per_gas present but max_priority_fee_per_gas missing.
    const built = _buildTxSendWireArgs({
      network: "local",
      signer: "alice",
      mode: "1559",
      to: "0x" + "b".repeat(40),
      max_fee_per_gas: "0x3b9aca00",
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) return;
    expect(built.error).toContain("1559");
    expect(built.error).toContain("max_priority_fee_per_gas");
  });

  it("_BadSignerAlias_RejectedAtBoundary", () => {
    // Signer alias must start with a letter — '1bad' fails the regex at
    // zod parse time so the MCP SDK rejects the call before the handler.
    expect(() =>
      TxSendArgs.parse({
        network: "local",
        signer: "1bad",
        mode: "legacy",
        gas_price: "0x1",
      }),
    ).toThrow();
  });

  it("_OddLengthData_RejectedAtBoundary", () => {
    // EVM calldata is byte-aligned, so the HEX_DATA regex must reject
    // odd-length payloads ('0xa') at zod parse time. Empty ('0x') and
    // even-length ('0x1234') payloads continue to pass.
    expect(() =>
      TxSendArgs.parse({
        network: "local",
        signer: "alice",
        mode: "legacy",
        gas_price: "0x1",
        data: "0xa",
      }),
    ).toThrow();
  });

  it("_LegacyWithMaxPriorityFee_Rejected", () => {
    // Covers the second operand of the legacy-mode '||' rejection clause:
    // max_priority_fee_per_gas alone (without max_fee_per_gas) must still
    // trip the boundary check.
    const built = _buildTxSendWireArgs({
      network: "local",
      signer: "alice",
      mode: "legacy",
      gas_price: "0x1",
      max_priority_fee_per_gas: "0x1",
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) return;
    expect(built.error).toContain("max_priority_fee_per_gas");
  });
});

// Black-box handler test: drives the full path through _buildTxSendWireArgs +
// callWire + formatWireResult against the mock binary. The build-arg unit
// tests above cover the cross-field rejection branches; this one verifies
// the wire-failure passthrough surfaces isError:true.
describe("chainbench_tx_send handler", () => {
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

  it("_WireFailure_PassedThrough", async () => {
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({
          type: "result",
          ok: false,
          error: { code: "UPSTREAM_ERROR", message: "nonce too low" },
        }),
      },
    ]);
    const out = await _txSendHandler({
      network: "local",
      signer: "alice",
      mode: "legacy",
      to: "0x" + "b".repeat(40),
      value: "0x10",
      gas_price: "0x3b9aca00",
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (UPSTREAM_ERROR)");
    expect(out.content[0]?.text).toContain("nonce too low");
  });
});
