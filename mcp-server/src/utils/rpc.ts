/**
 * utils/rpc.ts - Direct JSON-RPC client for chainbench MCP server.
 *
 * Reads node ports from state/pids.json and makes HTTP JSON-RPC calls
 * without shelling out. Used by consensus and network MCP tools.
 */
import http from "node:http";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { CHAINBENCH_DIR } from "./exec.js";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface NodeInfo {
  http_port: number;
  status: string;
  p2p_port?: number;
  [key: string]: unknown;
}

interface PidsState {
  nodes: Record<string, NodeInfo>;
}

// ---------------------------------------------------------------------------
// State helpers
// ---------------------------------------------------------------------------

export function loadPidsState(): PidsState {
  const path = resolve(CHAINBENCH_DIR, "state", "pids.json");
  return JSON.parse(readFileSync(path, "utf-8"));
}

export function getRunningNodeIds(): string[] {
  const state = loadPidsState();
  return Object.entries(state.nodes)
    .filter(([, info]) => info.status === "running")
    .map(([id]) => id)
    .sort((a, b) => Number(a) - Number(b));
}

export function getNodePort(nodeId: string): number {
  const state = loadPidsState();
  const node = state.nodes[nodeId];
  if (!node) throw new Error(`Node ${nodeId} not found in pids.json`);
  if (node.status !== "running") throw new Error(`Node ${nodeId} is not running (status: ${node.status})`);
  return node.http_port;
}

// ---------------------------------------------------------------------------
// JSON-RPC call
// ---------------------------------------------------------------------------

export function rpcCall(nodeId: string, method: string, params: unknown[] = []): Promise<unknown> {
  const port = getNodePort(nodeId);
  const body = JSON.stringify({ jsonrpc: "2.0", method, params, id: 1 });

  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        hostname: "127.0.0.1",
        port,
        path: "/",
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Content-Length": Buffer.byteLength(body),
        },
        timeout: 10_000,
      },
      (res) => {
        let data = "";
        res.on("data", (chunk: string) => {
          data += chunk;
        });
        res.on("end", () => {
          try {
            const parsed = JSON.parse(data);
            if (parsed.error) {
              reject(new Error(parsed.error.message || JSON.stringify(parsed.error)));
            } else {
              resolve(parsed.result);
            }
          } catch {
            reject(new Error(`Invalid JSON-RPC response: ${data.substring(0, 200)}`));
          }
        });
      },
    );
    req.on("error", reject);
    req.on("timeout", () => {
      req.destroy();
      reject(new Error(`RPC timeout calling ${method} on node ${nodeId}`));
    });
    req.write(body);
    req.end();
  });
}

// ---------------------------------------------------------------------------
// Convenience: call multiple nodes in parallel
// ---------------------------------------------------------------------------

export async function rpcCallAll(
  method: string,
  params: unknown[] = [],
  nodeIds?: string[],
): Promise<Record<string, unknown>> {
  const ids = nodeIds ?? getRunningNodeIds();
  const entries = await Promise.all(
    ids.map(async (id) => {
      try {
        const result = await rpcCall(id, method, params);
        return [id, result] as const;
      } catch (err) {
        return [id, { error: (err as Error).message }] as const;
      }
    }),
  );
  return Object.fromEntries(entries);
}

// ---------------------------------------------------------------------------
// Hex helpers
// ---------------------------------------------------------------------------

export function toHex(n: number): string {
  return "0x" + n.toString(16);
}

export function fromHex(hex: string): number {
  return parseInt(hex, 16);
}
