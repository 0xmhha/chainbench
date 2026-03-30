import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench } from "../utils/exec.js";

const PROJECT_ROOT_SCHEMA = z
  .string()
  .optional()
  .describe(
    "Absolute path to the blockchain project root (e.g., go-stablenet repo). " +
    "Used to auto-detect the binary at <project_root>/build/bin/. " +
    "Falls back to CHAINBENCH_PROJECT_ROOT env var, then $PATH lookup.",
  );

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "Done.";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

function validateProjectRoot(projectRoot: string | undefined): string | null {
  if (projectRoot && !projectRoot.startsWith("/")) {
    return "project_root must be an absolute path.";
  }
  return null;
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
      project_root: PROJECT_ROOT_SCHEMA,
    },
    async ({ profile, project_root }) => {
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
      const rootError = validateProjectRoot(project_root);
      if (rootError) {
        return { content: [{ type: "text" as const, text: `Error: ${rootError}` }] };
      }
      const result = runChainbench(`init --profile ${profile} --quiet`, {
        cwd: project_root,
      });
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_start",
    "Start all chain nodes. The chain must have been initialized with chainbench_init first.",
    {
      project_root: PROJECT_ROOT_SCHEMA,
    },
    async ({ project_root }) => {
      const rootError = validateProjectRoot(project_root);
      if (rootError) {
        return { content: [{ type: "text" as const, text: `Error: ${rootError}` }] };
      }
      const result = runChainbench("start --quiet", { cwd: project_root });
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
    {
      project_root: PROJECT_ROOT_SCHEMA,
    },
    async ({ project_root }) => {
      const rootError = validateProjectRoot(project_root);
      if (rootError) {
        return { content: [{ type: "text" as const, text: `Error: ${rootError}` }] };
      }
      const result = runChainbench("restart --quiet", { cwd: project_root });
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
