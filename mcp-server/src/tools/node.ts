// node.ts — node-scoped lifecycle and raw RPC tools.
//
// Sprint 5c.3 reroutes the three chainbench_node_* tools from runChainbench
// (bash CLI shell-out) to callWire (chainbench-net wire). Task 2 covered
// chainbench_node_rpc; Task 3 (this commit) covers chainbench_node_stop and
// chainbench_node_start.
//
// The 1-based node index argument is preserved at the MCP surface; the wire
// layer's node_id ('node1', 'node2', ...) is constructed by string
// concatenation. Keeping the LLM-facing shape unchanged means callers see no
// surface change across the reroute.
//
// Known asymmetry: chainbench_node_start exposes an optional binary_path arg
// that the wire layer's node.start handler does not yet accept. When the
// caller supplies binary_path, this handler falls back to runChainbench
// (bash CLI) so the override still works. Sprint 5c.4 will extend the Go
// node.start handler to take binary_path and remove the fallback. Tracked
// in NEXT_WORK §3 P3.

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench, shellEscapeArg } from "../utils/exec.js";
import { callWire } from "../utils/wire.js";
import { formatWireResult } from "../utils/wireResult.js";
import { errorResp, type FormattedToolResponse } from "../utils/mcpResp.js";
import { RPC_METHOD } from "../utils/hex.js";

const NODE_INDEX_SCHEMA = z
  .number()
  .int()
  .min(1)
  .max(64)
  .describe("1-based node index (e.g. 1 for the first node)");

// Exported for unit testing — the MCP SDK consumes the .shape from each
// *Args schema and the matching _*Handler from this file. Keeping them as
// named exports lets node.test.ts exercise the zod parse + handler logic
// directly without spinning up an MCP server stub.

export const NodeStopArgs = z.object({
  node: NODE_INDEX_SCHEMA,
}).strict();

type NodeStopArgsT = z.infer<typeof NodeStopArgs>;

export async function _nodeStopHandler(
  args: NodeStopArgsT,
): Promise<FormattedToolResponse> {
  const result = await callWire("node.stop", {
    network: "local",
    node_id: `node${args.node}`,
  });
  return formatWireResult(result);
}

export const NodeStartArgs = z.object({
  node: NODE_INDEX_SCHEMA,
  binary_path: z
    .string()
    .optional()
    .describe(
      "Absolute path to the chain binary. Overrides the profile's chain.binary_path. " +
      "When provided, the call falls back to the bash CLI path because the wire " +
      "layer's node.start handler does not yet accept this argument.",
    ),
}).strict();

type NodeStartArgsT = z.infer<typeof NodeStartArgs>;

export async function _nodeStartHandler(
  args: NodeStartArgsT,
): Promise<FormattedToolResponse> {
  const { node, binary_path } = args;
  if (binary_path !== undefined) {
    if (binary_path.length === 0) {
      return errorResp("binary_path must not be empty");
    }
    if (!binary_path.startsWith("/")) {
      return errorResp("binary_path must be an absolute path");
    }
    // Wire layer doesn't yet accept binary_path; fall back to bash CLI.
    // TODO Sprint 5c.4: extend node.start wire handler to accept binary_path
    // and drop this branch.
    const bash = runChainbench(
      `node start ${node} --binary-path ${shellEscapeArg(binary_path)}`,
    );
    const text =
      bash.exitCode === 0
        ? bash.stdout || "Done."
        : `Error (exit ${bash.exitCode}): ${bash.stderr || bash.stdout || "unknown error"}`;
    return {
      content: [{ type: "text" as const, text }],
      isError: bash.exitCode !== 0,
    };
  }
  const result = await callWire("node.start", {
    network: "local",
    node_id: `node${node}`,
  });
  return formatWireResult(result);
}

export const NodeRpcArgs = z.object({
  node: NODE_INDEX_SCHEMA,
  method: z
    .string()
    .regex(RPC_METHOD)
    .describe("JSON-RPC method name (e.g. 'eth_blockNumber', 'eth_getBalance', 'net_peerCount')"),
  params: z
    .string()
    .optional()
    .describe("JSON array of parameters (e.g. '[\"0xabc...\", \"latest\"]'). Defaults to '[]' if omitted."),
}).strict();

type NodeRpcArgsT = z.infer<typeof NodeRpcArgs>;

export async function _nodeRpcHandler(
  args: NodeRpcArgsT,
): Promise<FormattedToolResponse> {
  const { node, method, params } = args;
  let parsedParams: unknown[] = [];
  if (params !== undefined) {
    try {
      const parsed = JSON.parse(params);
      if (!Array.isArray(parsed)) {
        return errorResp("params must be a JSON array");
      }
      parsedParams = parsed;
    } catch {
      return errorResp("params is not valid JSON");
    }
  }
  const result = await callWire("node.rpc", {
    network: "local",
    node_id: `node${node}`,
    method,
    params: parsedParams,
  });
  return formatWireResult(result);
}

export function registerNodeTools(server: McpServer): void {
  server.tool(
    "chainbench_node_stop",
    "Stop a specific node by its 1-based index. The node can be restarted later with chainbench_node_start.",
    NodeStopArgs.shape,
    _nodeStopHandler,
  );

  server.tool(
    "chainbench_node_start",
    "Start (or restart) a specific stopped node by its 1-based index. The chain must have been initialized first.",
    NodeStartArgs.shape,
    _nodeStartHandler,
  );

  server.tool(
    "chainbench_node_rpc",
    "Send a JSON-RPC call directly to a specific node and return the response. " +
      "Useful for querying state or sending transactions via raw RPC. The 1-based " +
      "node index is mapped to the wire layer's node_id ('node1', 'node2', ...).",
    NodeRpcArgs.shape,
    _nodeRpcHandler,
  );
}
