/**
 * tools/consensus.ts - WBFT consensus inspection MCP tools.
 *
 * Provides structured access to istanbul_* RPC methods for
 * validator participation, round stability, and seal verification.
 */
import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { rpcCall, rpcCallAll, getRunningNodeIds, toHex, fromHex } from "../utils/rpc.js";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function textResult(text: string) {
  return { content: [{ type: "text" as const, text }] };
}

async function currentBlockNumber(nodeId: string): Promise<number> {
  const hex = (await rpcCall(nodeId, "eth_blockNumber")) as string;
  return fromHex(hex);
}

// ---------------------------------------------------------------------------
// Tool registration
// ---------------------------------------------------------------------------

export function registerConsensusTools(server: McpServer): void {
  // ---- chainbench_consensus_validators ----
  server.tool(
    "chainbench_consensus_validators",
    "Get the list of WBFT validators at a specific block. Returns validator addresses and count. Useful for verifying the active validator set.",
    {
      node: z.number().int().min(1).max(64).optional().default(1)
        .describe("Node to query (1-based index). Default: 1."),
      block: z.number().int().min(0).optional()
        .describe("Block number (decimal). Omit for latest."),
    },
    async ({ node, block }) => {
      try {
        const params = block !== undefined ? [toHex(block)] : [];
        const validators = (await rpcCall(String(node), "istanbul_getValidators", params)) as string[];
        const lines = [
          `Validators at ${block !== undefined ? `block ${block}` : "latest"}:`,
          `Count: ${validators.length}`,
          "",
          ...validators.map((addr, i) => `  ${i + 1}. ${addr}`),
        ];
        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );

  // ---- chainbench_consensus_status ----
  server.tool(
    "chainbench_consensus_status",
    "Get WBFT consensus activity statistics over a block range. Shows validator participation rates, block authoring distribution, round stability, and seal signature counts. Essential for diagnosing consensus health.",
    {
      node: z.number().int().min(1).max(64).optional().default(1)
        .describe("Node to query. Default: 1."),
      blocks: z.number().int().min(1).max(1000).optional().default(100)
        .describe("Number of recent blocks to analyze. Default: 100."),
    },
    async ({ node, blocks }) => {
      try {
        const nodeId = String(node);
        const current = await currentBlockNumber(nodeId);
        const start = Math.max(1, current - blocks + 1);

        const result = (await rpcCall(nodeId, "istanbul_status", [toHex(start), toHex(current)])) as Record<string, unknown>;

        const authorCounts = (result.authorCounts ?? {}) as Record<string, number>;
        const sealerActivity = (result.sealerActivity ?? {}) as Record<string, Record<string, number>>;
        const roundStats = (result.roundStats ?? {}) as Record<string, unknown>;
        const roundDist = (roundStats.roundDistribution ?? {}) as Record<string, number>;

        const totalBlocks = current - start + 1;
        const round0 = roundDist["0"] ?? 0;
        const round0Pct = totalBlocks > 0 ? ((round0 / totalBlocks) * 100).toFixed(1) : "0";

        const lines = [
          `WBFT Consensus Status (blocks ${start}–${current})`,
          "═".repeat(50),
          "",
          `Block range: ${totalBlocks} blocks`,
          `Round-0 finalization: ${round0}/${totalBlocks} (${round0Pct}%)`,
          "",
          "Block Authors:",
          ...Object.entries(authorCounts)
            .sort(([, a], [, b]) => b - a)
            .map(([addr, count]) => `  ${addr}: ${count} blocks (${((count / totalBlocks) * 100).toFixed(1)}%)`),
          "",
          "Committed Seal Participation:",
          ...Object.entries((sealerActivity.committed ?? {}) as Record<string, number>)
            .sort(([, a], [, b]) => b - a)
            .map(([addr, count]) => `  ${addr}: ${count} seals`),
        ];

        if (Object.keys(roundDist).length > 1) {
          lines.push("", "Round Distribution:");
          for (const [round, count] of Object.entries(roundDist).sort(([a], [b]) => Number(a) - Number(b))) {
            lines.push(`  Round ${round}: ${count} blocks`);
          }
        }

        const healthIssues: string[] = [];
        if (Number(round0Pct) < 80) healthIssues.push("Low round-0 rate (< 80%) — possible network or validator issues");
        const activeAuthors = Object.values(authorCounts).filter((v) => v > 0).length;
        if (activeAuthors < 2) healthIssues.push("Only 1 active block author — check validator set");

        if (healthIssues.length > 0) {
          lines.push("", "Health Warnings:", ...healthIssues.map((w) => `  ⚠ ${w}`));
        } else {
          lines.push("", "Health: OK");
        }

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );

  // ---- chainbench_consensus_block_info ----
  server.tool(
    "chainbench_consensus_block_info",
    "Get decoded WBFT extra data from a specific block header. Shows consensus round, prepared/committed seals, RANDAO reveal, gas tip, and epoch info. Useful for inspecting individual block consensus details.",
    {
      node: z.number().int().min(1).max(64).optional().default(1)
        .describe("Node to query. Default: 1."),
      block: z.number().int().min(0)
        .describe("Block number (decimal) to inspect."),
    },
    async ({ node, block }) => {
      try {
        const nodeId = String(node);
        const [extraInfo, signers] = await Promise.all([
          rpcCall(nodeId, "istanbul_getWbftExtraInfo", [toHex(block)]) as Promise<Record<string, unknown>>,
          rpcCall(nodeId, "istanbul_getCommitSignersFromBlock", [toHex(block)]) as Promise<Record<string, unknown>>,
        ]);

        const round = extraInfo.round ?? "0x0";
        const roundNum = typeof round === "string" && round.startsWith("0x") ? fromHex(round) : Number(round);

        const committedSeal = (extraInfo.committedSeal ?? {}) as Record<string, unknown>;
        const sealers = (committedSeal.sealers ?? []) as string[];
        const committers = (signers.committers ?? []) as string[];
        const author = signers.author ?? "unknown";

        const lines = [
          `WBFT Block Info — Block #${block}`,
          "═".repeat(50),
          "",
          `Author (proposer): ${author}`,
          `Consensus round: ${roundNum}`,
          `Committed seal signers: ${sealers.length}`,
          `Commit signers: ${committers.length}`,
        ];

        if (extraInfo.gasTip !== undefined) {
          lines.push(`Gas tip: ${extraInfo.gasTip}`);
        }

        if (extraInfo.epochInfo) {
          const epoch = extraInfo.epochInfo as Record<string, unknown>;
          const candidates = (epoch.candidates ?? []) as unknown[];
          const validators = (epoch.validators ?? []) as unknown[];
          lines.push("", `Epoch boundary: yes (${candidates.length} candidates, ${validators.length} validators)`);
        }

        if (committers.length > 0) {
          lines.push("", "Commit Signers:");
          committers.forEach((addr, i) => lines.push(`  ${i + 1}. ${addr}`));
        }

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );

  // ---- chainbench_consensus_health ----
  server.tool(
    "chainbench_consensus_health",
    "Quick consensus health check across all running nodes. Verifies block production, sync status, validator agreement, and round stability in a single call.",
    {},
    async () => {
      try {
        const nodeIds = getRunningNodeIds();
        if (nodeIds.length === 0) return textResult("No running nodes found.");

        // Block heights
        const heights = await rpcCallAll("eth_blockNumber");
        const blockNums: Record<string, number> = {};
        for (const [id, hex] of Object.entries(heights)) {
          blockNums[id] = typeof hex === "string" ? fromHex(hex) : 0;
        }

        const values = Object.values(blockNums);
        const maxBlock = Math.max(...values);
        const minBlock = Math.min(...values);
        const diff = maxBlock - minBlock;
        const synced = diff <= 2;

        // Validators on node 1
        const validators = (await rpcCall(nodeIds[0], "istanbul_getValidators")) as string[];

        // Recent round stability (last 20 blocks)
        const sampleStart = Math.max(1, maxBlock - 19);
        let round0Count = 0;
        const sampleSize = maxBlock - sampleStart + 1;

        for (let b = sampleStart; b <= maxBlock; b++) {
          try {
            const info = (await rpcCall(nodeIds[0], "istanbul_getWbftExtraInfo", [toHex(b)])) as Record<string, unknown>;
            const r = info.round as string;
            if (r === "0x0" || r === "0x00") round0Count++;
          } catch {
            // skip
          }
        }

        const round0Pct = sampleSize > 0 ? ((round0Count / sampleSize) * 100).toFixed(1) : "0";

        const lines = [
          "Consensus Health Check",
          "═".repeat(50),
          "",
          `Nodes: ${nodeIds.length} running`,
          `Validators: ${validators.length}`,
          `Latest block: ${maxBlock}`,
          `Sync: ${synced ? "OK" : "DEGRADED"} (diff=${diff})`,
          `Round-0 rate: ${round0Count}/${sampleSize} (${round0Pct}%)`,
          "",
          "Node Block Heights:",
          ...Object.entries(blockNums).map(([id, h]) => `  Node ${id}: ${h}${h < maxBlock - 2 ? " ⚠ behind" : ""}`),
        ];

        const ok = synced && Number(round0Pct) >= 80;
        lines.push("", `Overall: ${ok ? "HEALTHY" : "NEEDS ATTENTION"}`);

        return textResult(lines.join("\n"));
      } catch (err) {
        return textResult(`Error: ${(err as Error).message}`);
      }
    },
  );
}
