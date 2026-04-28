// chain_tx.ts — signer-required write tools.
//
// Hosts MCP tools that produce signed transactions. The signer alias resolves
// to key material in the host environment (CHAINBENCH_SIGNER_<ALIAS>_KEY or
// CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE + _KEYSTORE_PASSWORD); raw key material
// never crosses the MCP boundary.
//
// Today: chainbench_tx_send (modes legacy | 1559). Sprint 5c.2 follow-ups
// will land in this file:
//   - Task 4: chainbench_contract_deploy
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

const MODE = z.enum(["legacy", "1559"]);

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
  // Sprint 5c.2 Task 4 will add chainbench_contract_deploy here.
}
