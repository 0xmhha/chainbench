import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench, shellEscapeArg } from "../utils/exec.js";

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "Done.";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

export function registerConfigTools(server: McpServer): void {
  server.tool(
    "chainbench_config_set",
    "Write a field to the machine-local overlay (state/local-config.yaml). " +
    "Persistent but git-ignored. Use for machine-specific paths like chain.binary_path. " +
    "Different from chainbench_profile_set which edits the git-tracked profile YAML.",
    {
      field: z.string().describe("Dot-notation path (e.g., 'chain.binary_path', 'chain.logrot_path')"),
      value: z.string().describe("Value. JSON-parsed if valid JSON, else string."),
    },
    async ({ field, value }) => {
      if (!/^[a-zA-Z0-9_][a-zA-Z0-9_.]*$/.test(field)) {
        return { content: [{ type: "text" as const, text: "Error: invalid field path." }] };
      }
      const result = runChainbench(
        `config set ${shellEscapeArg(field)} ${shellEscapeArg(value)}`
      );
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_config_get",
    "Read a field from the machine-local overlay (state/local-config.yaml).",
    {
      field: z.string().describe("Dot-notation path (e.g., 'chain.binary_path')"),
    },
    async ({ field }) => {
      if (!/^[a-zA-Z0-9_][a-zA-Z0-9_.]*$/.test(field)) {
        return { content: [{ type: "text" as const, text: "Error: invalid field path." }] };
      }
      const result = runChainbench(`config get ${shellEscapeArg(field)}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_config_list",
    "List all fields in the machine-local overlay (state/local-config.yaml).",
    {},
    async () => {
      const result = runChainbench("config list");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );
}
