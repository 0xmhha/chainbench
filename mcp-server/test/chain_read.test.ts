import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  AccountStateArgs,
  ContractCallArgs,
  EventsGetArgs,
  TxWaitArgs,
  _accountStateHandler,
  _contractCallHandler,
  _eventsGetHandler,
  _txWaitHandler,
} from "../src/tools/chain_read.js";

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

// _contractCallHandler shares the same wire-injection contract: the cross-
// field XOR (calldata vs abi+method+args) is enforced before any spawn so the
// rejected branches don't need MOCK_SCRIPT, and the happy paths set MOCK_SCRIPT
// to a single ok:true result line. Saved env values are restored to keep
// tests isolated from the account_state block above.
describe("chainbench_contract_call handler", () => {
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

  it("_Happy_Calldata", async () => {
    const data = { result_raw: "0x" + "00".repeat(31) + "2a" };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _contractCallHandler({
      network: "local",
      contract_address: "0x" + "b".repeat(40),
      calldata: "0x70a08231",
    });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_Happy_ABI", async () => {
    const data = {
      result_raw: "0x" + "00".repeat(31) + "2a",
      result_decoded: [42],
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _contractCallHandler({
      network: "local",
      contract_address: "0x" + "b".repeat(40),
      abi: '[{"name":"balanceOf","type":"function","inputs":[{"name":"who","type":"address"}],"outputs":[{"type":"uint256"}]}]',
      method: "balanceOf",
      args: ["0x" + "c".repeat(40)],
    });
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("result_decoded");
    expect(out.content[0]?.text).toContain("42");
  });

  it("_BothCalldataAndABI_Rejected", async () => {
    // Cross-field check fires before any spawn — MOCK_SCRIPT intentionally
    // empty so a stray wire call would surface as a no-result-terminator
    // error instead of a silent pass.
    const out = await _contractCallHandler({
      network: "local",
      contract_address: "0x" + "b".repeat(40),
      calldata: "0x70a08231",
      abi: '[{"name":"balanceOf","type":"function","inputs":[],"outputs":[]}]',
      method: "balanceOf",
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("mutually exclusive");
  });

  it("_NeitherCalldataNorABI_Rejected", async () => {
    const out = await _contractCallHandler({
      network: "local",
      contract_address: "0x" + "b".repeat(40),
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("required");
  });

  it("_ABIWithoutMethod_Rejected", async () => {
    const out = await _contractCallHandler({
      network: "local",
      contract_address: "0x" + "b".repeat(40),
      abi: '[{"name":"balanceOf","type":"function","inputs":[],"outputs":[]}]',
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("method");
  });

  it("_BadAddress_RejectedAtBoundary", () => {
    // Bad-shaped contract_address must throw at zod parse time so MCP SDK
    // rejects the call before the handler runs.
    expect(() =>
      ContractCallArgs.parse({
        network: "local",
        contract_address: "0xnothex",
        calldata: "0x",
      }),
    ).toThrow();
  });

  it("_StrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'gas_limit', 'tag') trip a
    // structured INVALID_ARGS at the zod boundary instead of being silently
    // stripped.
    expect(() =>
      ContractCallArgs.parse({
        network: "local",
        contract_address: "0x" + "b".repeat(40),
        calldata: "0x",
        gas_limit: "21000",
      }),
    ).toThrow();
  });
});

// _eventsGetHandler shares the same wire-injection contract as the other
// chain_read tools. The cross-field check (abi <-> event) fires before any
// spawn so the rejected branches don't need MOCK_SCRIPT, and the happy paths
// set MOCK_SCRIPT to a single ok:true result line. Saved env values are
// restored to keep tests isolated from the contract_call block above.
describe("events_get", () => {
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

  it("_Happy_NoFilters", async () => {
    const data = { logs: [] };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _eventsGetHandler({ network: "local" });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_Happy_WithAddressAndTopics", async () => {
    const data = {
      logs: [
        {
          address: "0x" + "b".repeat(40),
          topics: ["0x" + "1".repeat(64), "0x" + "2".repeat(64)],
          data: "0x",
          block_number: "0x10",
          tx_hash: "0x" + "3".repeat(64),
          log_index: "0x0",
        },
      ],
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _eventsGetHandler({
      network: "local",
      address: "0x" + "b".repeat(40),
      topics: ["0x" + "1".repeat(64), ["0x" + "2".repeat(64), "0x" + "4".repeat(64)], null],
    });
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("logs");
    expect(out.content[0]?.text).toContain("0x" + "1".repeat(64));
  });

  it("_Happy_WithABIDecode", async () => {
    const data = {
      logs: [
        {
          address: "0x" + "b".repeat(40),
          topics: ["0x" + "5".repeat(64)],
          data: "0x",
          decoded: {
            event: "Transfer",
            args: { from: "0x" + "a".repeat(40), to: "0x" + "c".repeat(40), value: "0x2a" },
          },
        },
      ],
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _eventsGetHandler({
      network: "local",
      abi: '[{"name":"Transfer","type":"event","inputs":[{"name":"from","type":"address","indexed":true},{"name":"to","type":"address","indexed":true},{"name":"value","type":"uint256"}]}]',
      event: "Transfer",
    });
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("decoded");
    expect(out.content[0]?.text).toContain("Transfer");
  });

  it("_BadTopicHex_RejectedAtBoundary", () => {
    // A topic must be exactly 32 bytes (64 hex chars). Too-short hex must
    // throw at zod parse time rather than reach the wire and surface as
    // an UPSTREAM_ERROR after a wasted spawn.
    expect(() =>
      EventsGetArgs.parse({ network: "local", topics: ["0xabc"] }),
    ).toThrow();
  });

  it("_ABIWithoutEvent_Rejected", async () => {
    // Cross-field check fires before any spawn — MOCK_SCRIPT intentionally
    // empty so a stray wire call would surface as a no-result-terminator
    // error instead of a silent pass.
    const out = await _eventsGetHandler({
      network: "local",
      abi: '[{"name":"Transfer","type":"event","inputs":[]}]',
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("event");
  });

  it("_EventWithoutABI_Rejected", async () => {
    const out = await _eventsGetHandler({
      network: "local",
      event: "Transfer",
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("INVALID_ARGS");
    expect(out.content[0]?.text).toContain("abi");
  });

  it("_StrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'block_hash', 'limit') trip a
    // structured INVALID_ARGS at the zod boundary instead of being silently
    // stripped.
    expect(() =>
      EventsGetArgs.parse({
        network: "local",
        block_hash: "0x" + "1".repeat(64),
      }),
    ).toThrow();
  });
});

// _txWaitHandler shares the same wire-injection contract as the other
// chain_read tools. Both happy paths set MOCK_SCRIPT to a single ok:true
// receipt line; the boundary-rejection cases exercise zod parse() directly
// and never reach the wire. Saved env values are restored to keep tests
// isolated from the events_get block above.
describe("chainbench_tx_wait handler", () => {
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

  it("_Happy_DefaultTimeout", async () => {
    const data = {
      tx_hash: "0x" + "a".repeat(64),
      status: "success",
      block_number: "0x10",
      block_hash: "0x" + "b".repeat(64),
      gas_used: "0x5208",
      logs_count: 0,
      effective_gas_price: "0x3b9aca00",
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _txWaitHandler({
      network: "local",
      tx_hash: "0x" + "a".repeat(64),
    });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_Happy_CustomTimeout", async () => {
    const data = {
      tx_hash: "0x" + "a".repeat(64),
      status: "success",
      block_number: "0x11",
      block_hash: "0x" + "c".repeat(64),
      gas_used: "0x5208",
      logs_count: 1,
      effective_gas_price: "0x3b9aca00",
    };
    process.env.MOCK_SCRIPT = script([
      {
        kind: "stdout",
        line: JSON.stringify({ type: "result", ok: true, data }),
      },
    ]);
    const out = await _txWaitHandler({
      network: "local",
      tx_hash: "0x" + "a".repeat(64),
      timeout_ms: 5000,
    });
    expect(out.isError).toBeFalsy();
    expect(out.content).toHaveLength(1);
    expect(out.content[0]?.text).toContain(JSON.stringify(data, null, 2));
  });

  it("_BadTxHash_RejectedAtBoundary", () => {
    // tx_hash must be exactly 0x + 32-byte (64 hex chars). Too-short hex
    // must throw at zod parse time rather than reach the wire and surface
    // as an INVALID_ARGS after a wasted spawn.
    expect(() =>
      TxWaitArgs.parse({ network: "local", tx_hash: "0xabc" }),
    ).toThrow();
  });

  it("_NegativeTimeout_RejectedAtBoundary", () => {
    // timeout_ms uses z.number().int().positive() — 0 and negatives must
    // throw at zod parse time.
    expect(() =>
      TxWaitArgs.parse({
        network: "local",
        tx_hash: "0x" + "a".repeat(64),
        timeout_ms: -1,
      }),
    ).toThrow();
  });

  it("_StrictRejectsUnknownKeys", () => {
    // .strict() makes hallucinated keys (e.g. 'block_number', 'poll_interval')
    // trip a structured INVALID_ARGS at the zod boundary instead of being
    // silently stripped.
    expect(() =>
      TxWaitArgs.parse({
        network: "local",
        tx_hash: "0x" + "a".repeat(64),
        block_number: "latest",
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
          error: { code: "UPSTREAM_ERROR", message: "rpc dial failed" },
        }),
      },
    ]);
    const out = await _txWaitHandler({
      network: "local",
      tx_hash: "0x" + "a".repeat(64),
    });
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error (UPSTREAM_ERROR)");
    expect(out.content[0]?.text).toContain("rpc dial failed");
  });
});
