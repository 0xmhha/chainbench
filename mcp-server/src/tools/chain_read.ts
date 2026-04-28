// chain_read.ts — read-only high-level tools (no signer required).
//
// Hosts MCP tools that read chain state without spending gas or producing
// transactions. Today: chainbench_account_state, chainbench_contract_call.
// Sprint 5c.2 follow-ups (Tasks 2-3) will land in this file:
//   - chainbench_events_get     (eth_getLogs wrapper, optional event decoding)
//   - chainbench_tx_wait        (receipt polling)
//
// Cross-field validation that zod schemas cannot express cleanly (e.g.
// fields=['storage'] requires storage_key, or contract_call's calldata XOR
// abi+method+args) lives in the handler so the MCP boundary returns a
// structured INVALID_ARGS text response instead of a thrown exception.

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { callWire } from "../utils/wire.js";
import {
  formatWireResult,
  type FormattedToolResponse,
} from "../utils/wireResult.js";

const HEX_ADDRESS = /^0x[a-fA-F0-9]{40}$/;
const HEX_STORAGE_KEY = /^0x[a-fA-F0-9]{1,64}$/;
// HEX_DATA: 0x-prefixed even-length hex (matches chain_tx.ts deliberately —
// per plan §0.1 the duplication is intentional until a 5c.3 utils extraction).
const HEX_DATA = /^0x([a-fA-F0-9]{2})*$/;

const FIELD = z.enum(["balance", "nonce", "code", "storage"]);

// errorResp: shared INVALID_ARGS shaping for cross-field handler checks. Keeps
// the structured "Error (INVALID_ARGS): <reason>" text in lock-step with
// formatWireResult's failure path so MCP clients can parse uniformly. Module-
// private; the matching helper in chain_tx.ts is intentionally a duplicate
// per plan §0.1 (utils extraction is a 5c.3 P3 candidate).
function errorResp(msg: string): FormattedToolResponse {
  return {
    content: [{ type: "text", text: `Error (INVALID_ARGS): ${msg}` }],
    isError: true,
  };
}

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

export const ContractCallArgs = z.object({
  network: z
    .string()
    .min(1)
    .describe("Network name (e.g. 'local', 'sepolia')"),
  node_id: z
    .string()
    .optional()
    .describe("Node ID, default first node"),
  contract_address: z
    .string()
    .regex(HEX_ADDRESS)
    .describe("Target contract address (0x-prefixed 40 hex chars)"),
  calldata: z
    .string()
    .regex(HEX_DATA)
    .optional()
    .describe(
      "Raw, already-encoded calldata (0x-prefixed, even-length hex including " +
        "the 4-byte selector and ABI-encoded args). Mutually exclusive with " +
        "abi/method/args.",
    ),
  abi: z
    .string()
    .optional()
    .describe(
      "JSON-encoded contract ABI string. Server-side encoding mode — pair " +
        "with method (required) and args (optional). Mutually exclusive with " +
        "calldata.",
    ),
  method: z
    .string()
    .optional()
    .describe("Method name to call. Required when abi is provided."),
  args: z
    .array(z.unknown())
    .optional()
    .describe(
      "Arguments for the method, ordered by ABI input list. Defaults to " +
        "empty when omitted. Tuple / nested-array / fixed-bytesN(N!=32) " +
        "inputs require raw calldata fallback.",
    ),
  block_number: z
    .union([z.string(), z.number()])
    .optional()
    .describe("'latest', 'earliest', '0x10', or integer block number"),
  from: z
    .string()
    .regex(HEX_ADDRESS)
    .optional()
    .describe("Optional caller address (0x-prefixed 40 hex chars)"),
}).strict();

type ContractCallArgsT = z.infer<typeof ContractCallArgs>;

export async function _contractCallHandler(
  args: ContractCallArgsT,
): Promise<FormattedToolResponse> {
  // Cross-field validation: calldata XOR (abi + method + args). Zod cannot
  // express the XOR + dependency without a discriminated union that would
  // sacrifice the .strict() unknown-key rejection, so the check lives here.
  if (args.calldata !== undefined && args.abi !== undefined) {
    return errorResp("calldata and abi are mutually exclusive");
  }
  if (args.calldata === undefined && args.abi === undefined) {
    return errorResp(
      "either calldata or (abi + method + args) is required",
    );
  }
  if (args.abi !== undefined && args.method === undefined) {
    return errorResp("abi requires method");
  }

  const wireArgs: Record<string, unknown> = {
    network: args.network,
    contract_address: args.contract_address,
  };
  if (args.node_id !== undefined) wireArgs.node_id = args.node_id;
  if (args.calldata !== undefined) wireArgs.calldata = args.calldata;
  if (args.abi !== undefined) wireArgs.abi = args.abi;
  if (args.method !== undefined) wireArgs.method = args.method;
  if (args.args !== undefined) wireArgs.args = args.args;
  if (args.block_number !== undefined) wireArgs.block_number = args.block_number;
  if (args.from !== undefined) wireArgs.from = args.from;

  const result = await callWire("node.contract_call", wireArgs);
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
  server.tool(
    "chainbench_contract_call",
    "Read-only contract call (eth_call). Either provide raw `calldata` " +
      "(0x-prefixed hex, even-length, including 4-byte selector + ABI-encoded " +
      "args) or provide `abi` + `method` + `args` for server-side encoding. " +
      "Returns `result_raw` always; `result_decoded` is populated only when " +
      "ABI mode is used. Tuple / nested-array / fixed-bytesN(N!=32) inputs " +
      "require raw calldata fallback.",
    ContractCallArgs.shape,
    _contractCallHandler,
  );
  // Sprint 5c.2 Tasks 2-3 will add events_get, tx_wait here.
}
