// node.ts — node-scoped lifecycle and raw RPC tools.
//
// Sprint 5c.3 Task 2 reroutes chainbench_node_rpc from runChainbench (bash CLI
// shell-out) to callWire (chainbench-net wire). chainbench_node_stop and
// chainbench_node_start remain on the bash path until Task 3.
//
// The 1-based node index argument is preserved at the MCP surface; the wire
// layer's node_id ('node1', 'node2', ...) is constructed by string
// concatenation. Keeping the LLM-facing shape unchanged means callers see no
// surface change across the reroute.

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

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "Done.";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

// Exported for unit testing — the MCP SDK consumes the .shape from
// _NodeRpcArgs and the handler from _nodeRpcHandler. Keeping them as named
// exports lets node.test.ts exercise the zod parse + handler logic directly
// without spinning up an MCP server stub.
export const _NodeRpcArgs = z.object({
  node: NODE_INDEX_SCHEMA,
  method: z
    .string()
    .regex(RPC_METHOD)
    .describe("JSON-RPC method name (e.g. 'eth_blockNumber', 'eth_getBalance', 'net_peerCount')"),
  params: z
    .string()
    .optional()
    .describe("JSON array of parameters (e.g. '[\"0xabc...\", \"latest\"]'). Defaults to '[]' if omitted."),
});

type NodeRpcArgsT = z.infer<typeof _NodeRpcArgs>;

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
    {
      node: NODE_INDEX_SCHEMA,
    },
    async ({ node }) => {
      const result = runChainbench(`node stop ${node}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_node_start",
    "Start (or restart) a specific stopped node by its 1-based index. The chain must have been initialized first.",
    {
      node: NODE_INDEX_SCHEMA,
      binary_path: z
        .string()
        .optional()
        .describe("Absolute path to the chain binary. Overrides profile's chain.binary_path for this invocation."),
    },
    async ({ node, binary_path }) => {
      if (binary_path !== undefined) {
        if (binary_path.length === 0) {
          return { content: [{ type: "text" as const, text: "Error: binary_path must not be empty." }] };
        }
        if (!binary_path.startsWith("/")) {
          return { content: [{ type: "text" as const, text: "Error: binary_path must be an absolute path." }] };
        }
      }
      const bpArg = binary_path ? ` --binary-path ${shellEscapeArg(binary_path)}` : "";
      const result = runChainbench(`node start ${node}${bpArg}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_node_rpc",
    "Send a JSON-RPC call directly to a specific node and return the response. " +
      "Useful for querying state or sending transactions via raw RPC. The 1-based " +
      "node index is mapped to the wire layer's node_id ('node1', 'node2', ...).",
    _NodeRpcArgs.shape,
    _nodeRpcHandler,
  );
}
