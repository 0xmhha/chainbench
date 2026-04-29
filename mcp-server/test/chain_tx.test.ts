import { describe, it, expect, beforeAll, beforeEach, afterEach } from "vitest";
import { chmodSync, existsSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import {
  TxSendArgs,
  _buildTxSendWireArgs,
  _txSendHandler,
  ContractDeployArgs,
  _buildContractDeployWireArgs,
} from "../src/tools/chain_tx.js";

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

// _buildTxSendWireArgs is the pure cross-field validator. The mode/fee-field
// exclusivity rule cannot be expressed cleanly in a zod schema (a discriminated
// union on `mode` would force two parallel schemas and lose the shared field
// metadata), so the boundary check lives here and is exercised directly by
// the unit tests below — no spawn, no env setup needed. The handler black-box
// test at the bottom covers the wire passthrough.
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
    // Sprint 5c.2 Task 0 widened the return shape to {wireCommand, wireArgs}.
    // Existing legacy/1559 modes both stay on node.tx_send; Task 6 adds
    // node.tx_fee_delegation_send for fee_delegation mode.
    expect(built.wireCommand).toBe("node.tx_send");
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
    expect(built.wireCommand).toBe("node.tx_send");
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

  it("_1559WithGasPrice_Rejected", () => {
    // Symmetric to _LegacyWithMaxFee_Rejected: mode='1559' must reject
    // gas_price even when both 1559 fee fields are present. Closes the
    // missing branch flagged by the 5c.1 final review (minor #1).
    const built = _buildTxSendWireArgs({
      network: "local",
      signer: "alice",
      mode: "1559",
      max_fee_per_gas: "0x1",
      max_priority_fee_per_gas: "0x1",
      gas_price: "0x1",
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) return;
    expect(built.error).toContain("1559");
    expect(built.error).toContain("gas_price");
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
});

// Sprint 5c.2 Task 5 — mode set_code (EIP-7702). Branch lives in
// _buildTxSendWireArgs alongside legacy/1559; wireCommand stays
// "node.tx_send" because chainbench-net auto-routes to the SetCodeTx
// envelope when the authorization_list field is present and non-empty.
// fee_payer / authorization_list cross-field rejection paths are covered
// here so Task 6 can flip the fee_delegation discriminator without
// re-touching this file's schema-validation expectations.
describe("chainbench_tx_send mode set_code", () => {
  const baseSetCode = {
    network: "local",
    signer: "alice",
    mode: "set_code" as const,
    to: "0x" + "a".repeat(40),
    max_fee_per_gas: "0x59682f00",
    max_priority_fee_per_gas: "0x3b9aca00",
    authorization_list: [
      {
        chain_id: "0x1",
        address: "0x" + "b".repeat(40),
        nonce: "0x0",
        signer: "bob",
      },
    ],
  };

  it("_Happy_SetCode", () => {
    const built = _buildTxSendWireArgs(baseSetCode);
    expect("wireArgs" in built).toBe(true);
    if (!("wireArgs" in built)) throw new Error("expected wireArgs");
    expect(built.wireCommand).toBe("node.tx_send");
    expect(built.wireArgs.authorization_list).toBeDefined();
    expect(built.wireArgs.max_fee_per_gas).toBe("0x59682f00");
  });

  it("_SetCodeWithoutAuthList_Rejected", () => {
    const { authorization_list: _omit, ...partial } = baseSetCode;
    const built = _buildTxSendWireArgs(partial as any);
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("authorization_list");
  });

  it("_SetCodeWithoutMaxFee_Rejected", () => {
    const { max_fee_per_gas: _omit, ...partial } = baseSetCode;
    const built = _buildTxSendWireArgs(partial as any);
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("max_fee_per_gas");
  });

  it("_SetCodeWithGasPrice_Rejected", () => {
    const built = _buildTxSendWireArgs({ ...baseSetCode, gas_price: "0x1" });
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("gas_price");
  });

  it("_AuthListInLegacyMode_Rejected", () => {
    const built = _buildTxSendWireArgs({
      network: "local",
      signer: "alice",
      mode: "legacy",
      gas_price: "0x1",
      authorization_list: baseSetCode.authorization_list,
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("authorization_list");
  });

  it("_BadAuthEntry_RejectedAtBoundary", () => {
    expect(() =>
      TxSendArgs.parse({
        network: "local",
        signer: "alice",
        mode: "set_code",
        max_fee_per_gas: "0x1",
        max_priority_fee_per_gas: "0x1",
        authorization_list: [
          {
            chain_id: "0xnothex", // bad hex
            address: "0x" + "b".repeat(40),
            nonce: "0x0",
            signer: "bob",
          },
        ],
      }),
    ).toThrow();
  });
});

// Sprint 5c.2 Task 6 — mode fee_delegation (go-stablenet 0x16). Unlike
// legacy/1559/set_code (all on node.tx_send), fee_delegation dispatches to
// node.tx_fee_delegation_send via the wireCommand discriminator. Cross-field
// rules are stricter: fee_payer + to + 1559 fee fields + explicit gas +
// explicit nonce are required (chainbench-net does not auto-fill); gas_price
// and authorization_list are rejected.
describe("chainbench_tx_send mode fee_delegation", () => {
  const baseFD = {
    network: "stablenet-mainnet",
    signer: "alice",
    fee_payer: "fpayer-1",
    mode: "fee_delegation" as const,
    to: "0x" + "a".repeat(40),
    max_fee_per_gas: "0x59682f00",
    max_priority_fee_per_gas: "0x3b9aca00",
    gas: 21000,
    nonce: 7,
  };

  it("_Happy_FeeDelegation", () => {
    const built = _buildTxSendWireArgs(baseFD);
    expect("wireArgs" in built).toBe(true);
    if (!("wireArgs" in built)) throw new Error("expected wireArgs");
    expect(built.wireCommand).toBe("node.tx_fee_delegation_send");
    expect(built.wireArgs.fee_payer).toBe("fpayer-1");
    expect(built.wireArgs.signer).toBe("alice");
    expect(built.wireArgs.gas).toBe(21000);
    expect(built.wireArgs.nonce).toBe(7);
  });

  it("_FeeDelegationWithoutFeePayer_Rejected", () => {
    const { fee_payer: _omit, ...partial } = baseFD;
    const built = _buildTxSendWireArgs(partial as any);
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("fee_payer");
  });

  it("_FeeDelegationWithoutTo_Rejected", () => {
    const { to: _omit, ...partial } = baseFD;
    const built = _buildTxSendWireArgs(partial as any);
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("to");
  });

  it("_FeeDelegationWithGasPrice_Rejected", () => {
    const built = _buildTxSendWireArgs({ ...baseFD, gas_price: "0x1" });
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("gas_price");
  });

  it("_FeeDelegationWithoutGasOrNonce_Rejected", () => {
    const { gas: _omitGas, ...partial } = baseFD;
    const built = _buildTxSendWireArgs(partial as any);
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("gas");
  });

  it("_FeeDelegationWithAuthList_Rejected", () => {
    const built = _buildTxSendWireArgs({
      ...baseFD,
      authorization_list: [{
        chain_id: "0x1",
        address: "0x" + "b".repeat(40),
        nonce: "0x0",
        signer: "bob",
      }],
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("authorization_list");
  });

  it("_FeePayerInLegacyMode_Rejected", () => {
    const built = _buildTxSendWireArgs({
      network: "local",
      signer: "alice",
      mode: "legacy",
      gas_price: "0x1",
      fee_payer: "fpayer-1",
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("fee_payer");
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

describe("chainbench_contract_deploy handler", () => {
  const baseLegacy = {
    network: "local",
    signer: "alice",
    mode: "legacy" as const,
    bytecode: "0x6080604052",
    gas_price: "0x1",
  };
  const base1559 = {
    network: "local",
    signer: "alice",
    mode: "1559" as const,
    bytecode: "0x6080604052",
    max_fee_per_gas: "0x59682f00",
    max_priority_fee_per_gas: "0x3b9aca00",
  };

  it("_Happy_LegacyBytecode", () => {
    const built = _buildContractDeployWireArgs(baseLegacy);
    expect("wireArgs" in built).toBe(true);
    if (!("wireArgs" in built)) throw new Error("expected wireArgs");
    expect(built.wireArgs.gas_price).toBe("0x1");
    expect(built.wireArgs.max_fee_per_gas).toBeUndefined();
    expect(built.wireArgs.bytecode).toBe("0x6080604052");
  });

  it("_Happy_1559WithABI", () => {
    const built = _buildContractDeployWireArgs({
      ...base1559,
      abi: '[{"type":"constructor","inputs":[]}]',
      constructor_args: [],
    });
    expect("wireArgs" in built).toBe(true);
    if (!("wireArgs" in built)) throw new Error("expected wireArgs");
    expect(built.wireArgs.max_fee_per_gas).toBe("0x59682f00");
    expect(built.wireArgs.max_priority_fee_per_gas).toBe("0x3b9aca00");
    expect(built.wireArgs.gas_price).toBeUndefined();
    expect(built.wireArgs.abi).toBeDefined();
    expect(built.wireArgs.constructor_args).toEqual([]);
  });

  it("_BadBytecode_RejectedAtBoundary", () => {
    expect(() =>
      ContractDeployArgs.parse({
        ...baseLegacy,
        bytecode: "0xnothex",
      }),
    ).toThrow();
  });

  it("_LegacyWithMaxFee_Rejected", () => {
    const built = _buildContractDeployWireArgs({
      ...baseLegacy,
      max_fee_per_gas: "0x1",
    });
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("legacy");
    expect(built.error).toContain("max_fee");
  });

  it("_1559WithoutMaxFee_Rejected", () => {
    const { max_fee_per_gas: _omit, ...partial } = base1559;
    const built = _buildContractDeployWireArgs(partial as any);
    expect("error" in built).toBe(true);
    if (!("error" in built)) throw new Error("expected error");
    expect(built.error).toContain("1559");
  });

  it("_StrictRejectsUnknownKeys", () => {
    expect(() =>
      ContractDeployArgs.parse({
        ...baseLegacy,
        to: "0x" + "a".repeat(40), // unknown key — deploy creates contracts, no `to`
      }),
    ).toThrow();
  });
});
