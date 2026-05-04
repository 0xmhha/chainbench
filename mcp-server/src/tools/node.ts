// node.ts — node-scoped lifecycle and raw RPC tools.
//
// Sprint 5c.3 rerouted the three chainbench_node_* tools from runChainbench
// (bash CLI shell-out) to callWire (chainbench-net wire). Task 2 covered
// chainbench_node_rpc; Task 3 covered chainbench_node_stop and
// chainbench_node_start. Sprint 5c.4.1 Task 5 closes the last asymmetry:
// chainbench_node_start's optional binary_path is now forwarded over the
// wire envelope (the Go node.start handler appends `--binary-path <path>`
// to chainbench.sh argv), so the runChainbench fallback branch is gone and
// every node lifecycle call goes through callWire.
//
// The 1-based node index argument is preserved at the MCP surface; the wire
// layer's node_id ('node1', 'node2', ...) is constructed by string
// concatenation. Keeping the LLM-facing shape unchanged means callers see no
// surface change across the reroute.

import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
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
      "Absolute path to the chain binary. When provided, overrides the " +
      "profile's chain.binary_path for this single start. Forwarded to the " +
      "wire layer's node.start handler, which appends `--binary-path <path>` " +
      "to chainbench.sh argv. Must be an absolute path (start with `/`); " +
      "empty / relative paths are rejected at the MCP boundary.",
    ),
}).strict();

type NodeStartArgsT = z.infer<typeof NodeStartArgs>;

export async function _nodeStartHandler(
  args: NodeStartArgsT,
): Promise<FormattedToolResponse> {
  const { node, binary_path } = args;
  // Boundary validation. The Go handler also enforces these checks
  // (defense-in-depth) but rejecting here avoids a wasted wire spawn and
  // surfaces a clearer MCP-side error to the caller.
  if (binary_path !== undefined) {
    if (binary_path.length === 0) {
      return errorResp("binary_path must not be empty");
    }
    if (!binary_path.startsWith("/")) {
      return errorResp("binary_path must be an absolute path");
    }
  }
  const wireArgs: Record<string, unknown> = {
    network: "local",
    node_id: `node${node}`,
  };
  if (binary_path !== undefined) {
    wireArgs.binary_path = binary_path;
  }
  const result = await callWire("node.start", wireArgs);
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
