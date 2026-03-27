import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench } from "../utils/exec.js";

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

function validateRpcMethod(method: string): string | null {
  // Allow only valid JSON-RPC method names: namespace_methodName
  if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(method)) {
    return `Invalid RPC method name '${method}'. Must be alphanumeric with underscores (e.g. 'eth_blockNumber').`;
  }
  return null;
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
    },
    async ({ node }) => {
      const result = runChainbench(`node start ${node}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_node_rpc",
    "Send a JSON-RPC call directly to a specific node and return the response. Useful for querying state or sending transactions to individual nodes.",
    {
      node: NODE_INDEX_SCHEMA,
      method: z
        .string()
        .describe("JSON-RPC method name (e.g. 'eth_blockNumber', 'eth_getBalance', 'net_peerCount')"),
      params: z
        .string()
        .optional()
        .describe("JSON array of parameters (e.g. '[\"0xabc...\", \"latest\"]'). Defaults to '[]' if omitted."),
    },
    async ({ node, method, params }) => {
      const methodError = validateRpcMethod(method);
      if (methodError) {
        return { content: [{ type: "text" as const, text: `Error: ${methodError}` }] };
      }

      if (params !== undefined) {
        try {
          const parsed = JSON.parse(params);
          if (!Array.isArray(parsed)) {
            return {
              content: [
                {
                  type: "text" as const,
                  text: "Error: params must be a JSON array (e.g. '[\"0xabc\"]').",
                },
              ],
            };
          }
        } catch {
          return {
            content: [{ type: "text" as const, text: "Error: params is not valid JSON." }],
          };
        }
      }

      const paramsArg = params ? ` '${params}'` : "";
      const result = runChainbench(`node rpc ${node} ${method}${paramsArg}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );
}
