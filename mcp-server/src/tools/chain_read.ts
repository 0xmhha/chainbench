// chain_read.ts — read-only high-level tools (no signer required).
//
// Hosts MCP tools that read chain state without spending gas or producing
// transactions. Today: chainbench_account_state. Sprint 5c.2 follow-ups
// (Tasks 1-3) will land in this file:
//   - chainbench_contract_call  (eth_call wrapper, optional ABI decoding)
//   - chainbench_events_get     (eth_getLogs wrapper, optional event decoding)
//   - chainbench_tx_wait        (receipt polling)
//
// Cross-field validation that zod schemas cannot express cleanly (e.g.
// fields=['storage'] requires storage_key) lives in the handler so the MCP
// boundary returns a structured INVALID_ARGS text response instead of a
// thrown exception.

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import {
  formatWireResult,
  type FormattedToolResponse,
} from "../utils/wireResult.js";

const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
const HEX_STORAGE_KEY = /^0x[a-fA-F0-9]{1,64}$/;

const FIELD = z.enum(["balance", "nonce", "code", "storage"]);

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

export function registerChainReadTools(server: McpServer): void {
  server.tool(
    "chainbench_account_state",
    "Read account balance/nonce/code/storage from a network. " +
      "Network can be local or remote (attached). Returns hex-encoded values.",
    AccountStateArgs.shape,
    _accountStateHandler,
  );
  // Sprint 5c.2 Tasks 1-3 will add contract_call, events_get, tx_wait here.
}
