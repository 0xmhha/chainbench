// chain_tx.ts — signer-required write tools.
//
// Hosts MCP tools that produce signed transactions. The signer alias resolves
// to key material in the host environment (CHAINBENCH_SIGNER_<ALIAS>_KEY or
// CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE + _KEYSTORE_PASSWORD); raw key material
// never crosses the MCP boundary.
//
// Today: chainbench_tx_send (modes legacy | 1559) and
// chainbench_contract_deploy (modes legacy | 1559). Sprint 5c.2 follow-ups
// will land in this file:
//   - Task 5: chainbench_tx_send mode 'set_code'      (EIP-7702)
//   - Task 6: chainbench_tx_send mode 'fee_delegation' (go-stablenet 0x16,
//             dispatched to node.tx_fee_delegation_send via wireCommand)
//
// _buildTxSendWireArgs returns {wireCommand, wireArgs} so Task 6 can
// dispatch fee_delegation to a different wire command without touching the
// handler — every other mode keeps wireCommand: "node.tx_send".

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import {
  formatWireResult,
  type FormattedToolResponse,
} from "../utils/wireResult.js";

const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
const HEX_DATA = /^0x([a-fA-F0-9]{2})*$/;
const SIGNER_ALIAS = /^[A-Za-z][A-Za-z0-9_]*$/;

// errorResp: shared INVALID_ARGS shaping for cross-field handler checks. Keeps
// the structured "Error (INVALID_ARGS): <reason>" text in lock-step with
// formatWireResult's failure path so MCP clients can parse uniformly. Module-
// private; the matching helper in chain_read.ts is intentionally a duplicate
// per plan §0.1 — Sprint 5c.3 P3 will hoist this to utils/mcpResp.ts when the
// third callsite appears.
function errorResp(msg: string): FormattedToolResponse {
  return {
    content: [{ type: "text", text: `Error (INVALID_ARGS): ${msg}` }],
    isError: true,
  };
}

const MODE = z.enum(["legacy", "1559"]);

// DEPLOY_MODE is its own enum because contract creation only has fee-mode
// dimensions: set_code (EIP-7702 authorization list) and fee_delegation
// (go-stablenet 0x16) are tx_send concepts that have no analogue in deploy.
const DEPLOY_MODE = z.enum(["legacy", "1559"]);

export const TxSendArgs = z.object({
  network: z
    .string()
    .min(1)
    .describe("Network name (e.g. 'local', 'sepolia')"),
  node_id: z
    .string()
    .optional()
    .describe("Node ID, default first node"),
  signer: z
    .string()
    .regex(SIGNER_ALIAS)
    .describe(
      "Signer alias (letters/digits/underscore, must start with a letter). " +
        "CHAINBENCH_SIGNER_<ALIAS>_KEY (or CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE " +
        "+ CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE_PASSWORD) must be set in the host " +
        "environment that spawned the MCP server. The MCP layer never receives " +
        "raw key material.",
    ),
  mode: MODE.describe(
    "Transaction fee mode. 'legacy' requires gas_price and rejects " +
      "max_fee_per_gas / max_priority_fee_per_gas. '1559' requires both " +
      "max_fee_per_gas and max_priority_fee_per_gas and rejects gas_price.",
  ),
  to: z
    .string()
    .regex(HEX_ADDRESS)
    .optional()
    .describe(
      "Recipient address (0x-prefixed 40 hex chars). Omit for contract creation " +
        "(use chainbench_contract_deploy in future MCP releases).",
    ),
  value: z
    .string()
    .optional()
    .describe("Value in wei (decimal or 0x-hex)"),
  data: z
    .string()
    .regex(HEX_DATA)
    .optional()
    .describe("Calldata (0x-prefixed hex, even length)"),
  gas: z
    .union([z.string(), z.number()])
    .optional()
    .describe("Gas limit (decimal, 0x-hex, or integer)"),
  nonce: z
    .union([z.string(), z.number()])
    .optional()
    .describe("Nonce (decimal, 0x-hex, or integer); omit for auto-assign"),
  gas_price: z
    .string()
    .optional()
    .describe("Required for mode 'legacy'. Decimal wei or 0x-hex."),
  max_fee_per_gas: z
    .string()
    .optional()
    .describe("Required for mode '1559'. Decimal wei or 0x-hex."),
  max_priority_fee_per_gas: z
    .string()
    .optional()
    .describe("Required for mode '1559'. Decimal wei or 0x-hex."),
}).strict();

type TxSendArgsT = z.infer<typeof TxSendArgs>;

// Mode-specific fee field exclusivity. Zod cannot express this cleanly across
// the discriminated union without forcing one of two parallel schemas, so the
// check is a pure function the handler calls before the wire spawn. Returns
// {wireCommand, wireArgs} ready for callWire, or {error} for the boundary
// response. wireCommand defaults to "node.tx_send"; Sprint 5c.2 Task 6 will
// switch fee_delegation mode to "node.tx_fee_delegation_send".
export function _buildTxSendWireArgs(
  args: TxSendArgsT,
):
  | {
      wireCommand: "node.tx_send" | "node.tx_fee_delegation_send";
      wireArgs: Record<string, unknown>;
    }
  | { error: string } {
  if (args.mode === "legacy") {
    if (!args.gas_price) {
      return { error: "mode 'legacy' requires gas_price" };
    }
    if (
      args.max_fee_per_gas !== undefined ||
      args.max_priority_fee_per_gas !== undefined
    ) {
      return {
        error:
          "mode 'legacy' rejects max_fee_per_gas / max_priority_fee_per_gas",
      };
    }
  } else {
    // mode === "1559"
    if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
      return {
        error:
          "mode '1559' requires both max_fee_per_gas and max_priority_fee_per_gas",
      };
    }
    if (args.gas_price !== undefined) {
      return { error: "mode '1559' rejects gas_price" };
    }
  }

  // Wire envelope: chainbench-net's node.tx_send auto-detects fee mode from
  // the presence of max_fee_per_gas, so the synthetic 'mode' key is dropped
  // and only concrete fee fields are forwarded. network/signer are pinned
  // first for stable envelope ordering; remaining optional fields fall
  // through with explicit per-field assignment to match the pattern used by
  // _accountStateHandler.
  const wireArgs: Record<string, unknown> = {
    network: args.network,
    signer: args.signer,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.to !== undefined) wireArgs.to = args.to;
  if (args.value !== undefined) wireArgs.value = args.value;
  if (args.data !== undefined) wireArgs.data = args.data;
  if (args.gas !== undefined) wireArgs.gas = args.gas;
  if (args.nonce !== undefined) wireArgs.nonce = args.nonce;
  if (args.gas_price !== undefined) wireArgs.gas_price = args.gas_price;
  if (args.max_fee_per_gas !== undefined) {
    wireArgs.max_fee_per_gas = args.max_fee_per_gas;
  }
  if (args.max_priority_fee_per_gas !== undefined) {
    wireArgs.max_priority_fee_per_gas = args.max_priority_fee_per_gas;
  }
  return { wireCommand: "node.tx_send", wireArgs };
}

export async function _txSendHandler(
  args: TxSendArgsT,
): Promise<FormattedToolResponse> {
  const built = _buildTxSendWireArgs(args);
  if ("error" in built) {
    return {
      content: [
        { type: "text", text: `Error (INVALID_ARGS): ${built.error}` },
      ],
      isError: true,
    };
  }
  const result = await callWire(built.wireCommand, built.wireArgs);
  return formatWireResult(result);
}

export const ContractDeployArgs = z.object({
  network: z
    .string()
    .min(1)
    .describe("Network alias from networks/<name>.json or 'local'."),
  node_id: z
    .string()
    .optional()
    .describe("Specific node ID; defaults to first node."),
  signer: z
    .string()
    .regex(SIGNER_ALIAS)
    .describe(
      "Signer alias; CHAINBENCH_SIGNER_<ALIAS>_KEY (or _KEYSTORE+_KEYSTORE_PASSWORD) " +
        "must be set in host env.",
    ),
  mode: DEPLOY_MODE.describe(
    "'legacy' uses gas_price; '1559' uses max_fee_per_gas + max_priority_fee_per_gas.",
  ),
  bytecode: z
    .string()
    .regex(HEX_DATA)
    .describe("Contract bytecode (0x-prefixed even-length hex)."),
  abi: z
    .string()
    .optional()
    .describe(
      "Optional JSON ABI string. When provided with constructor_args, server " +
        "encodes constructor.",
    ),
  constructor_args: z
    .array(z.unknown())
    .optional()
    .describe("Constructor arguments matched to ABI's constructor inputs."),
  value: z
    .string()
    .optional()
    .describe("Wei to send (decimal or 0x-hex)."),
  gas: z
    .union([z.string(), z.number()])
    .optional()
    .describe("Gas limit; auto-estimated if omitted."),
  nonce: z
    .union([z.string(), z.number()])
    .optional()
    .describe("Tx nonce; auto-fetched if omitted."),
  gas_price: z
    .string()
    .optional()
    .describe("Required for mode 'legacy'."),
  max_fee_per_gas: z
    .string()
    .optional()
    .describe("Required for mode '1559'."),
  max_priority_fee_per_gas: z
    .string()
    .optional()
    .describe("Required for mode '1559'."),
}).strict();

type ContractDeployArgsT = z.infer<typeof ContractDeployArgs>;

// Mode-specific fee-field exclusivity for contract creation. Mirrors the
// tx_send build-args validator: legacy requires gas_price and rejects
// max_fee_*; 1559 requires both max_fee_per_gas + max_priority_fee_per_gas
// and rejects gas_price. Returns wireArgs ready for callWire's
// node.contract_deploy command, or {error} for the boundary response.
export function _buildContractDeployWireArgs(
  args: ContractDeployArgsT,
):
  | { wireArgs: Record<string, unknown> }
  | { error: string } {
  if (args.mode === "legacy") {
    if (!args.gas_price) {
      return { error: "mode 'legacy' requires gas_price" };
    }
    if (
      args.max_fee_per_gas !== undefined ||
      args.max_priority_fee_per_gas !== undefined
    ) {
      return {
        error:
          "mode 'legacy' rejects max_fee_per_gas / max_priority_fee_per_gas",
      };
    }
  } else {
    // mode === "1559"
    if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
      return {
        error:
          "mode '1559' requires both max_fee_per_gas and max_priority_fee_per_gas",
      };
    }
    if (args.gas_price !== undefined) {
      return { error: "mode '1559' rejects gas_price" };
    }
  }

  // Wire envelope: chainbench-net's node.contract_deploy auto-detects fee
  // mode from presence of max_fee_per_gas, so the synthetic 'mode' key is
  // dropped. network/signer/bytecode are pinned first for stable envelope
  // ordering; remaining optional fields fall through with explicit per-field
  // assignment matching _buildTxSendWireArgs.
  const wireArgs: Record<string, unknown> = {
    network: args.network,
    signer: args.signer,
    bytecode: args.bytecode,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.abi !== undefined) wireArgs.abi = args.abi;
  if (args.constructor_args !== undefined) {
    wireArgs.constructor_args = args.constructor_args;
  }
  if (args.value !== undefined) wireArgs.value = args.value;
  if (args.gas !== undefined) wireArgs.gas = args.gas;
  if (args.nonce !== undefined) wireArgs.nonce = args.nonce;
  if (args.gas_price !== undefined) wireArgs.gas_price = args.gas_price;
  if (args.max_fee_per_gas !== undefined) {
    wireArgs.max_fee_per_gas = args.max_fee_per_gas;
  }
  if (args.max_priority_fee_per_gas !== undefined) {
    wireArgs.max_priority_fee_per_gas = args.max_priority_fee_per_gas;
  }
  return { wireArgs };
}

export async function _contractDeployHandler(
  args: ContractDeployArgsT,
): Promise<FormattedToolResponse> {
  const built = _buildContractDeployWireArgs(args);
  if ("error" in built) return errorResp(built.error);
  const result = await callWire("node.contract_deploy", built.wireArgs);
  return formatWireResult(result);
}

export function registerChainTxTools(server: McpServer): void {
  server.tool(
    "chainbench_tx_send",
    "Send a signed transaction. Mode 'legacy' uses pre-EIP-1559 gas pricing " +
      "(gas_price required; max_fee_per_gas / max_priority_fee_per_gas rejected). " +
      "Mode '1559' uses EIP-1559 dynamic-fee fields (max_fee_per_gas + " +
      "max_priority_fee_per_gas required; gas_price rejected). The signer " +
      "parameter is an alias only — CHAINBENCH_SIGNER_<ALIAS>_KEY (or " +
      "CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE + CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE_PASSWORD) " +
      "must be set in the host environment that spawned the MCP server; raw key " +
      "material never crosses the MCP boundary. Modes 'set_code' (EIP-7702) and " +
      "'fee_delegation' (go-stablenet 0x16) are NOT yet exposed — use the wire " +
      "protocol directly until a future MCP release adds them.",
    TxSendArgs.shape,
    _txSendHandler,
  );
  server.tool(
    "chainbench_contract_deploy",
    "Deploy a contract by signing a contract-creation transaction (to: nil). " +
      "Mode 'legacy' uses pre-EIP-1559 gas pricing (gas_price required). " +
      "Mode '1559' uses dynamic fee fields (max_fee_per_gas + " +
      "max_priority_fee_per_gas required). When abi + constructor_args " +
      "provided, server encodes constructor and appends to bytecode. Returns " +
      "{tx_hash, contract_address}; contract_address is locally computed " +
      "(CREATE address from sender + nonce) — confirm actual deployment via " +
      "chainbench_tx_wait.",
    ContractDeployArgs.shape,
    _contractDeployHandler,
  );
  // Sprint 5c.2 Tasks 5-6 remaining: chainbench_tx_send mode 'set_code'
  // (EIP-7702) and mode 'fee_delegation' (go-stablenet 0x16 dispatched via
  // node.tx_fee_delegation_send wireCommand).
}
