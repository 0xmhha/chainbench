// chain.ts — Sprint 5c high-level tools backed by chainbench-net wire.
//
// 5c.1 lands the first two high-level tools: chainbench_account_state (read)
// and chainbench_tx_send (write, mode: legacy | 1559). Future sprints extend
// this file with contract_deploy / contract_call / events_get / tx_wait and
// the remaining tx_send modes (set_code, fee_delegation) in 5c.2, then
// reroute the existing 38 bash-shelling tools through callWire in 5c.3.
// Cross-field validation that zod schemas cannot express cleanly (e.g.
// fields=[storage] requires storage_key, or mode/fee-field exclusivity for
// tx_send) lives in the handler so the MCP boundary returns a structured
// INVALID_ARGS text response instead of a thrown exception.

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import {
  formatWireResult,
  type FormattedToolResponse,
} from "../utils/wireResult.js";

const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
const HEX_STORAGE_KEY = /^0x[a-fA-F0-9]{1,64}$/;
const HEX_DATA = /^0x([a-fA-F0-9]{2})*$/;
const SIGNER_ALIAS = /^[A-Za-z][A-Za-z0-9_]*$/;

const FIELD = z.enum(["balance", "nonce", "code", "storage"]);
const MODE = z.enum(["legacy", "1559"]);

export const AccountStateArgs = z.object({
  network: z
    .string()
    .min(1)
    .describe("Network name (e.g. 'local', 'sepolia')"),
  node_id: z
    .string()
    .optional()
    .describe("Node ID, default first node"),
  address: z
    .string()
    .regex(HEX_ADDRESS)
    .describe("Account address (0x-prefixed 40 hex chars)"),
  fields: z
    .array(FIELD)
    .optional()
    .describe(
      "Default ['balance','nonce','code']. 'storage' requires storage_key.",
    ),
  storage_key: z
    .string()
    .regex(HEX_STORAGE_KEY)
    .optional()
    .describe("Storage slot, required if fields includes 'storage'"),
  block_number: z
    .union([z.string(), z.number()])
    .optional()
    .describe("'latest', 'earliest', '0x10', or integer block number"),
}).strict();

type AccountStateArgsT = z.infer<typeof AccountStateArgs>;

export async function _accountStateHandler(
  args: AccountStateArgsT,
): Promise<FormattedToolResponse> {
  // Cross-field validation: fields=['storage'] requires storage_key. Zod
  // schemas can't express this cleanly without forcing the storage_key
  // field unconditionally, so the check lives here. Returns a structured
  // INVALID_ARGS response (matching chainbench-net error shape) rather
  // than throwing — MCP clients see isError:true and a parseable text.
  if (args.fields?.includes("storage") && !args.storage_key) {
    return {
      content: [
        {
          type: "text",
          text: "Error (INVALID_ARGS): fields includes 'storage' but storage_key is missing",
        },
      ],
      isError: true,
    };
  }

  const wireArgs: Record<string, unknown> = {
    network: args.network,
    address: args.address,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  // When fields is undefined, omit it from the wire envelope so chainbench-net
  // applies its own default (['balance','nonce','code']) — the MCP layer
  // stays oblivious to that default.
  if (args.fields !== undefined) wireArgs.fields = args.fields;
  if (args.storage_key !== undefined) wireArgs.storage_key = args.storage_key;
  if (args.block_number !== undefined) wireArgs.block_number = args.block_number;

  const result = await callWire("node.account_state", wireArgs);
  return formatWireResult(result);
}

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
// either {wireArgs} ready for callWire or {error} for the boundary response.
export function _buildTxSendWireArgs(
  args: TxSendArgsT,
):
  | { wireArgs: Record<string, unknown> }
  | { error: string } {
  if (args.mode === "legacy") {
    if (!args.gas_price) {
      return { error: "mode 'legacy' requires gas_price" };
    }
    if (args.max_fee_per_gas !== undefined || args.max_priority_fee_per_gas !== undefined) {
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
  // the presence of max_fee_per_gas, so we drop the synthetic 'mode' key and
  // forward only the concrete fee fields. network/signer are pinned first
  // for stable envelope ordering; everything else falls through unchanged.
  const wireArgs: Record<string, unknown> = {
    network: args.network,
    signer: args.signer,
  };
  for (const [k, v] of Object.entries(args)) {
    if (k === "mode" || k === "network" || k === "signer") continue;
    if (v === undefined) continue;
    wireArgs[k] = v;
  }
  return { wireArgs };
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
  const result = await callWire("node.tx_send", built.wireArgs);
  return formatWireResult(result);
}

export function registerChainTools(server: McpServer): void {
  server.tool(
    "chainbench_account_state",
    "Read account balance/nonce/code/storage from a network. " +
      "Network can be local or remote (attached). Returns hex-encoded values.",
    AccountStateArgs.shape,
    _accountStateHandler,
  );
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
}
