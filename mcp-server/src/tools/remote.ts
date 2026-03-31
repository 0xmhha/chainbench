import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench } from "../utils/exec.js";

const ALIAS_SCHEMA = z
  .string()
  .regex(/^[a-zA-Z][a-zA-Z0-9-]*$/, "Alias must start with a letter, contain only alphanumeric and dashes")
  .describe("Remote chain alias (e.g. 'eth-mainnet', 'my-testnet')");

const RPC_URL_SCHEMA = z
  .string()
  .regex(/^https?:\/\//, "URL must start with http:// or https://")
  .describe("RPC endpoint URL (e.g. 'https://rpc.ankr.com/eth')");

const CHAIN_TYPE_SCHEMA = z
  .enum(["testnet", "mainnet", "devnet"])
  .optional()
  .default("testnet")
  .describe("Chain type classification (default: testnet)");

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "Done.";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

function validateRpcMethod(method: string): string | null {
  if (!/^[a-zA-Z][a-zA-Z0-9_]*$/.test(method)) {
    return `Invalid RPC method name '${method}'. Must be alphanumeric with underscores (e.g. 'eth_blockNumber').`;
  }
  return null;
}

export function registerRemoteTools(server: McpServer): void {
  // --- Remote add ---
  server.tool(
    "chainbench_remote_add",
    "Register a remote blockchain RPC endpoint for testing and monitoring. Performs a connectivity check on registration.",
    {
      alias: ALIAS_SCHEMA,
      rpc_url: RPC_URL_SCHEMA,
      chain_type: CHAIN_TYPE_SCHEMA,
      ws_url: z
        .string()
        .optional()
        .describe("WebSocket RPC URL (optional, e.g. 'wss://ws.example.com')"),
      auth_header: z
        .string()
        .optional()
        .describe("Authorization header value (SENSITIVE - stored securely, never logged)"),
    },
    async ({ alias, rpc_url, chain_type, ws_url, auth_header }) => {
      let args = `remote add ${alias} ${rpc_url} --type ${chain_type}`;
      if (ws_url) {
        args += ` --ws ${ws_url}`;
      }
      if (auth_header) {
        args += ` --auth-header ${auth_header}`;
      }
      const result = runChainbench(args);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  // --- Remote list ---
  server.tool(
    "chainbench_remote_list",
    "List all registered remote chain RPC endpoints with their status, chain type, and chain ID.",
    {},
    async () => {
      const result = runChainbench("remote list --json");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  // --- Remote remove ---
  server.tool(
    "chainbench_remote_remove",
    "Remove a registered remote chain RPC endpoint by its alias.",
    {
      alias: ALIAS_SCHEMA,
    },
    async ({ alias }) => {
      const result = runChainbench(`remote remove ${alias}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  // --- Remote info ---
  server.tool(
    "chainbench_remote_info",
    "Query comprehensive chain metadata from a remote RPC endpoint: chain ID, block number, gas price, peer count, client version, sync status, and more.",
    {
      alias: ALIAS_SCHEMA,
    },
    async ({ alias }) => {
      const result = runChainbench(`remote info ${alias} --json`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  // --- Remote RPC ---
  server.tool(
    "chainbench_remote_rpc",
    "Send a raw JSON-RPC call to a registered remote chain endpoint. Useful for querying chain state or sending transactions.",
    {
      alias: ALIAS_SCHEMA,
      method: z
        .string()
        .describe("JSON-RPC method name (e.g. 'eth_blockNumber', 'eth_getBalance', 'net_peerCount')"),
      params: z
        .string()
        .optional()
        .describe("JSON array of parameters (e.g. '[\"0xabc...\", \"latest\"]'). Defaults to '[]' if omitted."),
    },
    async ({ alias, method, params }) => {
      const methodError = validateRpcMethod(method);
      if (methodError) {
        return { content: [{ type: "text" as const, text: `Error: ${methodError}` }] };
      }

      if (params !== undefined) {
        try {
          const parsed = JSON.parse(params);
          if (!Array.isArray(parsed)) {
            return {
              content: [{
                type: "text" as const,
                text: "Error: params must be a JSON array (e.g. '[\"0xabc\"]').",
              }],
            };
          }
        } catch {
          return {
            content: [{ type: "text" as const, text: "Error: params is not valid JSON." }],
          };
        }
      }

      const paramsArg = params ? ` '${params}'` : "";
      const result = runChainbench(`node rpc --remote ${alias} ${method}${paramsArg}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );
}
