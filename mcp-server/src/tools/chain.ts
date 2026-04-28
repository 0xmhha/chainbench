// chain.ts — Sprint 5c high-level tools backed by chainbench-net wire.
//
// 5c.1 lands the first read tool, chainbench_account_state. Future sprints
// extend this file with tx_send (5c.1 Task 4), contract_deploy / contract_call
// / events_get / tx_wait (5c.2), and reroute the existing 38 bash-shelling
// tools through callWire (5c.3). Cross-field validation that zod schemas
// cannot express cleanly (e.g. fields=[storage] requires storage_key) lives
// in the handler so the MCP boundary returns a structured INVALID_ARGS text
// response instead of a thrown exception.

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import type { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
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
});

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

export function registerChainTools(server: McpServer): void {
  server.tool(
    "chainbench_account_state",
    "Read account balance/nonce/code/storage from a network. " +
      "Network can be local or remote (attached). Returns hex-encoded values.",
    AccountStateArgs.shape,
    // Thin SDK adapter: McpServer.tool's callback signature requires
    // CallToolResult, whose content items carry an index signature.
    // FormattedToolResponse is the strict tested shape; this wrapper
    // widens it to the SDK's structural CallToolResult without changing
    // any runtime values.
    async (args) =>
      (await _accountStateHandler(args)) as unknown as CallToolResult,
  );
  // tx_send is added in Task 4.
}
