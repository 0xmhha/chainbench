import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench } from "../utils/exec.js";
import { formatExecResult } from "../utils/mcpResp.js";

// log output uses "No output." as the blank-stdout fallback.
const formatResult = (result: { stdout: string; stderr: string; exitCode: number }): string =>
  formatExecResult(result, "No output.");

function validatePattern(pattern: string): string | null {
  if (pattern.trim().length === 0) {
    return "Search pattern must not be empty.";
  }
  if (pattern.length > 256) {
    return "Search pattern is too long (max 256 characters).";
  }
  // Reject shell metacharacters that could be used for injection
  if (/[;&|`$\\<>]/.test(pattern)) {
    return `Search pattern contains disallowed characters. Avoid: ; & | \` $ \\ < >`;
  }
  return null;
}

export function registerLogTools(server: McpServer): void {
  server.tool(
    "chainbench_log_search",
    "Search node log files for a pattern. Returns matching log lines. Useful for diagnosing errors, tracking specific events, or finding consensus messages.",
    {
      pattern: z
        .string()
        .describe("Search pattern (grep-compatible regex, e.g. 'ERROR', 'WARN', 'committed block')"),
      node: z
        .number()
        .int()
        .min(1)
        .max(64)
        .optional()
        .describe("1-based node index to search. Omit to search all nodes."),
    },
    async ({ pattern, node }) => {
      const patternError = validatePattern(pattern);
      if (patternError) {
        return { content: [{ type: "text" as const, text: `Error: ${patternError}` }] };
      }

      const nodeArg = node !== undefined ? ` ${node}` : "";
      // Use double-quotes around pattern; exec.ts runs via execSync shell
      const result = runChainbench(`log search "${pattern}"${nodeArg}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_log_timeline",
    "Generate a consensus event timeline from all node logs. Shows block proposals, votes, commits, and view changes ordered by timestamp. Useful for diagnosing consensus issues.",
    {},
    async () => {
      const result = runChainbench("log timeline");
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );
}
