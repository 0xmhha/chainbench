// chain_tx.ts — signer-required write tools.
//
// Hosts MCP tools that produce signed transactions. The signer alias resolves
// to key material in the host environment (CHAINBENCH_SIGNER_<ALIAS>_KEY or
// CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE + _KEYSTORE_PASSWORD); raw key material
// never crosses the MCP boundary.
//
// Today: chainbench_tx_send (modes legacy | 1559 | set_code | fee_delegation)
// and chainbench_contract_deploy (modes legacy | 1559).
//
// _buildTxSendWireArgs returns {wireCommand, wireArgs} so the fee_delegation
// mode can dispatch to a different wire command (node.tx_fee_delegation_send)
// without touching the handler — legacy/1559/set_code all stay on
// wireCommand: "node.tx_send" (chainbench-net auto-routes set_code to the
// SetCodeTx envelope when the authorization_list field is present), while
// fee_delegation dispatches to wireCommand: "node.tx_fee_delegation_send"
// (go-stablenet's 0x16 envelope; chain adapter must be on the go-stablenet
// allowlist or chainbench-net surfaces NOT_SUPPORTED).

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import { formatWireResult } from "../utils/wireResult.js";
import { errorResp, type FormattedToolResponse } from "../utils/mcpResp.js";
import {
  HEX_ADDRESS,
  HEX_DATA,
  HEX_HEX,
  SIGNER_ALIAS,
} from "../utils/hex.js";

const MODE = z.enum(["legacy", "1559", "set_code", "fee_delegation"]);

// AuthorizationEntry mirrors the Go-side EIP-7702 authorization tuple
// (chainbench-net wire schema). chain_id and nonce are arbitrary-length
// 0x-hex (HEX_HEX) because they are big.Int values — chain_id may be "0x0"
// to signal "valid on any chain", and a single hex nibble is legal. The
// signer alias resolves on the host via CHAINBENCH_SIGNER_<ALIAS>_KEY just
// like the outer tx signer; raw key material never crosses the MCP boundary.
const AuthorizationEntry = z
  .object({
    chain_id: z
      .string()
      .regex(HEX_HEX)
      .describe("Hex chain ID; 0x0 means valid on any chain"),
    address: z
      .string()
      .regex(HEX_ADDRESS)
      .describe("Delegate target address"),
    nonce: z.string().regex(HEX_HEX).describe("Hex nonce"),
    signer: z
      .string()
      .regex(SIGNER_ALIAS)
      .describe(
        "Authorization signer alias; CHAINBENCH_SIGNER_<ALIAS>_KEY (or " +
          "_KEYSTORE+_KEYSTORE_PASSWORD) must be set in host env",
      ),
  })
  .strict();

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
      "max_fee_per_gas and max_priority_fee_per_gas and rejects gas_price. " +
      "'set_code' (EIP-7702) requires authorization_list plus 1559 fee " +
      "fields. 'fee_delegation' (go-stablenet 0x16) requires fee_payer, " +
      "to, both 1559 fee fields, and explicit gas + nonce (chainbench-net " +
      "does not auto-fill for this tx type); rejects gas_price and " +
      "authorization_list. The chain adapter must support 0x16 " +
      "(go-stablenet allowlist) — chainbench-net surfaces NOT_SUPPORTED " +
      "otherwise.",
  ),
  to: z
    .string()
    .regex(HEX_ADDRESS)
    .optional()
    .describe(
      "Recipient address (0x-prefixed 40 hex chars). Omit for contract " +
        "creation (use chainbench_contract_deploy).",
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
  authorization_list: z
    .array(AuthorizationEntry)
    .optional()
    .describe(
      "EIP-7702 authorization tuples; required (non-empty) for mode " +
        "'set_code'. Rejected by other modes.",
    ),
  fee_payer: z
    .string()
    .regex(SIGNER_ALIAS)
    .optional()
    .describe(
      "Fee payer alias; required for mode 'fee_delegation' (go-stablenet " +
        "0x16). CHAINBENCH_SIGNER_<ALIAS>_KEY (or _KEYSTORE+_KEYSTORE_PASSWORD) " +
        "must be set in host env. Rejected by other modes.",
    ),
}).strict();

type TxSendArgsT = z.infer<typeof TxSendArgs>;

// Mode-specific fee field exclusivity. Zod cannot express this cleanly across
// the discriminated union without forcing one of two parallel schemas, so the
// check is a pure function the handler calls before the wire spawn. Returns
// {wireCommand, wireArgs} ready for callWire, or {error} for the boundary
// response. legacy/1559/set_code dispatch to "node.tx_send" (chainbench-net
// auto-routes set_code to the SetCodeTx envelope when authorization_list is
// present); fee_delegation dispatches to "node.tx_fee_delegation_send"
// (go-stablenet 0x16 — chain adapter must accept the type or chainbench-net
// surfaces NOT_SUPPORTED).
export function _buildTxSendWireArgs(
  args: TxSendArgsT,
):
  | {
      wireCommand: "node.tx_send" | "node.tx_fee_delegation_send";
      wireArgs: Record<string, unknown>;
    }
  | { error: string } {
  // Cross-mode field gating. fee_payer is fee_delegation-only (go-stablenet
  // 0x16 envelope); authorization_list is set_code-only (EIP-7702 SetCodeTx).
  // Reject early so the mode-specific branches below can stay focused on
  // fee-field rules.
  if (args.fee_payer !== undefined && args.mode !== "fee_delegation") {
    return {
      error: `mode '${args.mode}' rejects fee_payer (use mode 'fee_delegation' instead)`,
    };
  }
  if (
    args.authorization_list !== undefined &&
    args.mode !== "set_code"
  ) {
    return {
      error: `mode '${args.mode}' rejects authorization_list (use mode 'set_code' instead)`,
    };
  }

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
  } else if (args.mode === "1559") {
    if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
      return {
        error:
          "mode '1559' requires both max_fee_per_gas and max_priority_fee_per_gas",
      };
    }
    if (args.gas_price !== undefined) {
      return { error: "mode '1559' rejects gas_price" };
    }
  } else if (args.mode === "set_code") {
    if (!args.authorization_list || args.authorization_list.length === 0) {
      return {
        error: "mode 'set_code' requires non-empty authorization_list",
      };
    }
    if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
      return {
        error:
          "mode 'set_code' requires both max_fee_per_gas and max_priority_fee_per_gas (1559 envelope)",
      };
    }
    if (args.gas_price !== undefined) {
      return { error: "mode 'set_code' rejects gas_price" };
    }
  } else {
    // mode === "fee_delegation" (go-stablenet 0x16). Stricter than other
    // modes: chainbench-net does NOT auto-fill nonce/gas/fees for this tx
    // type (chain-specific testing intent demands explicit values), to is
    // required (no contract creation), and the set_code combo is rejected.
    // The fee_payer alias is mandatory and resolves on the host the same
    // way the outer signer does.
    if (!args.fee_payer) {
      return { error: "mode 'fee_delegation' requires fee_payer alias" };
    }
    if (!args.to) {
      return {
        error:
          "mode 'fee_delegation' requires to (no contract creation in fee delegation)",
      };
    }
    if (!args.max_fee_per_gas || !args.max_priority_fee_per_gas) {
      return {
        error:
          "mode 'fee_delegation' requires both max_fee_per_gas and max_priority_fee_per_gas",
      };
    }
    if (args.gas === undefined) {
      return {
        error:
          "mode 'fee_delegation' requires gas (chainbench-net does not auto-fill for this tx type)",
      };
    }
    if (args.nonce === undefined) {
      return {
        error:
          "mode 'fee_delegation' requires nonce (chainbench-net does not auto-fill for this tx type)",
      };
    }
    if (args.gas_price !== undefined) {
      return { error: "mode 'fee_delegation' rejects gas_price" };
    }
    // authorization_list rejection is handled by the upstream cross-mode gate
    // (set_code-only); reaching this branch implies args.authorization_list
    // is undefined, so no branch-local check is needed.
  }

  // Wire envelope: legacy/1559/set_code share node.tx_send (chainbench-net
  // auto-detects fee mode from max_fee_per_gas presence and auto-routes to
  // a SetCodeTx (0x4) when authorization_list is present and non-empty).
  // fee_delegation dispatches to node.tx_fee_delegation_send (go-stablenet
  // 0x16). The synthetic 'mode' key is dropped from the wire — only
  // concrete fee fields, auth list, and fee_payer flow through.
  // network/signer are pinned first for stable envelope ordering; remaining
  // optional fields fall through with explicit per-field assignment to
  // match the pattern used by _accountStateHandler.
  const wireCommand: "node.tx_send" | "node.tx_fee_delegation_send" =
    args.mode === "fee_delegation"
      ? "node.tx_fee_delegation_send"
      : "node.tx_send";

  const wireArgs: Record<string, unknown> = {
    network: args.network,
    signer: args.signer,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.fee_payer !== undefined) wireArgs.fee_payer = args.fee_payer;
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
  if (args.authorization_list !== undefined) {
    wireArgs.authorization_list = args.authorization_list;
  }
  return { wireCommand, wireArgs };
}

export async function _txSendHandler(
  args: TxSendArgsT,
): Promise<FormattedToolResponse> {
  const built = _buildTxSendWireArgs(args);
  if ("error" in built) return errorResp(built.error);
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
      "max_priority_fee_per_gas required; gas_price rejected). Mode 'set_code' " +
      "(EIP-7702) builds a SetCodeTx (0x4) — requires non-empty " +
      "authorization_list plus the 1559 fee fields; chainbench-net auto-routes " +
      "to the SetCodeTx envelope when authorization_list is present. Mode " +
      "'fee_delegation' (go-stablenet 0x16) builds a fee-delegated tx — " +
      "requires fee_payer alias, to (no contract creation), both 1559 fee " +
      "fields, and explicit gas + nonce (chainbench-net does not auto-fill " +
      "for this tx type); rejects gas_price and authorization_list. The " +
      "chain adapter must support 0x16 (go-stablenet allowlist) — " +
      "chainbench-net surfaces NOT_SUPPORTED otherwise. The signer (and " +
      "fee_payer) parameters are aliases only — CHAINBENCH_SIGNER_<ALIAS>_KEY " +
      "(or CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE + " +
      "CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE_PASSWORD) must be set in the host " +
      "environment that spawned the MCP server; raw key material never " +
      "crosses the MCP boundary.",
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
}
