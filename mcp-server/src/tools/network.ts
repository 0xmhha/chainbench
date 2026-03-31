/**
 * tools/network.ts - P2P network management and txpool inspection MCP tools.
 *
 * Provides network partition/heal, peer topology, and txpool status
 * across all nodes via admin_* and txpool_* RPC methods.
 */
import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { rpcCall, rpcCallAll, getRunningNodeIds, fromHex } from "../utils/rpc.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function textResult(text: string) {
  return { content: [{ type: "text" as const, text }] };
}

async function getEnode(nodeId: string): Promise<string> {
  const info = (await rpcCall(nodeId, "admin_nodeInfo")) as Record<string, unknown>;
  return (info.enode ?? "") as string;
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

export function registerNetworkTools(server: McpServer): void {
  // ---- chainbench_network_peers ----
  server.tool(
    "chainbench_network_peers",
    "Show peer connectivity for all running nodes or a specific node. Displays peer count, connected peer IDs, and network topology overview.",
    {
      node: z.number().int().min(1).max(64).optional()
        .describe("Specific node to inspect. Omit for all running nodes."),
    },
    async ({ node }) => {
      try {
        const nodeIds = node ? [String(node)] : getRunningNodeIds();
        const lines = ["Network Peer Status", "═".repeat(50), ""];

        for (const nid of nodeIds) {
          const peers = (await rpcCall(nid, "admin_peers")) as Record<string, unknown>[];
          const peerNames = peers.map((p) => {
            const name = (p.name ?? "unknown") as string;
            const id = ((p.id ?? "") as string).substring(0, 16);
            return `${name} (${id}...)`;
          });

          lines.push(`Node ${nid}: ${peers.length} peer(s)`);
          if (peerNames.length > 0) {
            peerNames.forEach((pn) => lines.push(`  - ${pn}`));
          }
          lines.push("");
        }

        // Summary
        const counts = await rpcCallAll("net_peerCount", [], nodeIds);
        const totalPeers = Object.values(counts)
          .filter((v) => typeof v === "string")
          .reduce((sum, hex) => sum + fromHex(hex as string), 0);
        lines.push(`Total connections: ${totalPeers} (counted from all nodes)`);

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );

  // ---- chainbench_network_partition ----
  server.tool(
    "chainbench_network_partition",
    "Create or heal a network partition between two groups of nodes. Creating a partition disconnects all peers between the groups using admin_removePeer. Healing reconnects them using admin_addPeer. Use this to test BFT consensus behavior under network splits.",
    {
      action: z.enum(["create", "heal"])
        .describe("'create' to partition, 'heal' to reconnect."),
      group_a: z.array(z.number().int().min(1).max(64))
        .describe("First group of node indices (e.g., [1, 2])."),
      group_b: z.array(z.number().int().min(1).max(64))
        .describe("Second group of node indices (e.g., [3, 4])."),
    },
    async ({ action, group_a, group_b }) => {
      try {
        const allNodes = [...new Set([...group_a, ...group_b])].map(String);

        // Collect enode URLs
        const enodes: Record<string, string> = {};
        await Promise.all(
          allNodes.map(async (nid) => {
            enodes[nid] = await getEnode(nid);
          }),
        );

        const rpcMethod = action === "create" ? "admin_removePeer" : "admin_addPeer";
        const results: string[] = [];
        let successCount = 0;

        // For each cross-group pair, apply the action
        for (const a of group_a.map(String)) {
          for (const b of group_b.map(String)) {
            // a -> b
            if (enodes[b]) {
              try {
                await rpcCall(a, rpcMethod, [enodes[b]]);
                successCount++;
              } catch {
                results.push(`  Failed: node ${a} -> node ${b}`);
              }
            }
            // b -> a
            if (enodes[a]) {
              try {
                await rpcCall(b, rpcMethod, [enodes[a]]);
                successCount++;
              } catch {
                results.push(`  Failed: node ${b} -> node ${a}`);
              }
            }
          }
        }

        // For 'create', repeat removal to counter auto-reconnect
        if (action === "create") {
          await new Promise((r) => setTimeout(r, 2000));
          for (const a of group_a.map(String)) {
            for (const b of group_b.map(String)) {
              if (enodes[b]) await rpcCall(a, rpcMethod, [enodes[b]]).catch(() => {});
              if (enodes[a]) await rpcCall(b, rpcMethod, [enodes[a]]).catch(() => {});
            }
          }
        }

        const groupAStr = group_a.join(", ");
        const groupBStr = group_b.join(", ");
        const actionVerb = action === "create" ? "Partitioned" : "Reconnected";

        const lines = [
          `${actionVerb}: {${groupAStr}} ${action === "create" ? "||" : "<->"} {${groupBStr}}`,
          `Peer operations: ${successCount} successful`,
        ];

        if (results.length > 0) {
          lines.push("", "Issues:", ...results);
        }

        if (action === "create") {
          const bftThreshold = Math.ceil((2 * (group_a.length + group_b.length)) / 3);
          const aCanConsensus = group_a.length >= bftThreshold;
          const bCanConsensus = group_b.length >= bftThreshold;
          lines.push(
            "",
            `BFT threshold: ${bftThreshold}/${group_a.length + group_b.length}`,
            `Group A (${group_a.length} nodes): ${aCanConsensus ? "CAN produce blocks" : "CANNOT produce blocks"}`,
            `Group B (${group_b.length} nodes): ${bCanConsensus ? "CAN produce blocks" : "CANNOT produce blocks"}`,
          );
        }

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );

  // ---- chainbench_network_topology ----
  server.tool(
    "chainbench_network_topology",
    "Set a specific P2P topology for testing. 'full-mesh' connects all nodes to each other. 'hub-spoke' routes all traffic through a hub node. 'ring' connects nodes in a circular chain. Useful for testing consensus under restricted connectivity.",
    {
      topology: z.enum(["full-mesh", "hub-spoke", "ring"])
        .describe("Target topology pattern."),
      hub: z.number().int().min(1).max(64).optional().default(1)
        .describe("Hub node for hub-spoke topology. Default: 1. Ignored for other topologies."),
    },
    async ({ topology, hub }) => {
      try {
        const nodeIds = getRunningNodeIds();
        if (nodeIds.length < 2) return textResult("Need at least 2 running nodes.");

        // Collect all enodes
        const enodes: Record<string, string> = {};
        await Promise.all(nodeIds.map(async (nid) => { enodes[nid] = await getEnode(nid); }));

        // Step 1: Remove all peer connections
        for (const src of nodeIds) {
          for (const dst of nodeIds) {
            if (src === dst) continue;
            if (enodes[dst]) await rpcCall(src, "admin_removePeer", [enodes[dst]]).catch(() => {});
          }
        }
        await new Promise((r) => setTimeout(r, 2000));

        // Step 2: Build desired connections
        const connections: [string, string][] = [];

        if (topology === "full-mesh") {
          for (let i = 0; i < nodeIds.length; i++) {
            for (let j = i + 1; j < nodeIds.length; j++) {
              connections.push([nodeIds[i], nodeIds[j]]);
            }
          }
        } else if (topology === "hub-spoke") {
          const hubId = String(hub);
          for (const nid of nodeIds) {
            if (nid === hubId) continue;
            connections.push([hubId, nid]);
          }
        } else if (topology === "ring") {
          for (let i = 0; i < nodeIds.length; i++) {
            const next = nodeIds[(i + 1) % nodeIds.length];
            connections.push([nodeIds[i], next]);
          }
        }

        // Apply connections (bidirectional)
        for (const [a, b] of connections) {
          if (enodes[b]) await rpcCall(a, "admin_addPeer", [enodes[b]]).catch(() => {});
          if (enodes[a]) await rpcCall(b, "admin_addPeer", [enodes[a]]).catch(() => {});
        }

        const lines = [
          `Topology set: ${topology}`,
          `Nodes: ${nodeIds.join(", ")}`,
          `Connections: ${connections.length}`,
          "",
          "Links:",
          ...connections.map(([a, b]) => `  ${a} <-> ${b}`),
        ];

        if (topology === "hub-spoke") {
          lines.push("", `Hub: node ${hub} (${nodeIds.length - 1} spokes)`);
        }

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );

  // ---- chainbench_txpool_inspect ----
  server.tool(
    "chainbench_txpool_inspect",
    "Get transaction pool status across all running nodes or a specific node. Shows pending and queued transaction counts. Useful for monitoring transaction propagation and drain behavior.",
    {
      node: z.number().int().min(1).max(64).optional()
        .describe("Specific node to inspect. Omit for all running nodes."),
    },
    async ({ node }) => {
      try {
        const nodeIds = node ? [String(node)] : getRunningNodeIds();
        const lines = ["TxPool Status", "═".repeat(50), ""];

        let totalPending = 0;
        let totalQueued = 0;

        for (const nid of nodeIds) {
          const status = (await rpcCall(nid, "txpool_status")) as Record<string, string>;
          const pending = fromHex(status.pending ?? "0x0");
          const queued = fromHex(status.queued ?? "0x0");
          totalPending += pending;
          totalQueued += queued;
          lines.push(`Node ${nid}: pending=${pending} queued=${queued}`);
        }

        if (nodeIds.length > 1) {
          lines.push("", `Total: pending=${totalPending} queued=${totalQueued}`);
        }

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );
}
