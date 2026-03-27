import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench } from "../utils/exec.js";

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "Done.";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

export function registerLifecycleTools(server: McpServer): void {
  server.tool(
    "chainbench_init",
    "Initialize a local blockchain from a profile. Generates genesis.json, TOML configs, and initializes node data directories. Must be run before starting the chain.",
    {
      profile: z
        .string()
        .default("default")
        .describe("Profile name to use from the profiles/ directory (e.g. 'default', 'minimal', 'large')"),
    },
    async ({ profile }) => {
      // Validate profile name: alphanumeric, dashes, underscores, forward-slash only
      if (!/^[a-zA-Z0-9_\-/]+$/.test(profile)) {
        return {
          content: [
            {
              type: "text" as const,
              text: `Error: invalid profile name '${profile}'. Only alphanumeric characters, dashes, underscores, and slashes are allowed.`,
            },
          ],
        };
      }
      const result = runChainbench(`init --profile ${profile} --quiet`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_start",
    "Start all chain nodes. The chain must have been initialized with chainbench_init first.",
    {},
    async () => {
      const result = runChainbench("start --quiet");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_stop",
    "Stop all running chain nodes gracefully.",
    {},
    async () => {
      const result = runChainbench("stop --quiet");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_restart",
    "Restart the chain: stop all nodes, clean data, re-initialize with the same profile, and start fresh.",
    {},
    async () => {
      const result = runChainbench("restart --quiet");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_status",
    "Return current node status as JSON. Includes per-node block height, peer count, running state, and overall consensus health.",
    {},
    async () => {
      const result = runChainbench("status --json");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );
}
