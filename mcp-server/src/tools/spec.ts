import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench, shellEscapeArg } from "../utils/exec.js";

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "{}";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

export function registerSpecTools(server: McpServer): void {
  server.tool(
    "chainbench_spec_lookup",
    "Look up a regression test specification by its RT-ID. Returns the test case " +
    "title, priority, type, prerequisites, related TCs, code references, and the " +
    "Gherkin-style scenario excerpt. Use after a test failure to understand what " +
    "the test was supposed to verify and compare against actual results.",
    {
      id: z
        .string()
        .describe("Test case ID (e.g. 'RT-A-2-01', 'RT-B-01', 'RT-F-3-05')"),
      doc: z
        .string()
        .optional()
        .describe("Absolute path to spec document. Omit to use configured path."),
    },
    async ({ id, doc }) => {
      if (!/^RT-[A-Z]/i.test(id)) {
        return {
          content: [{ type: "text" as const, text: "Error: ID must start with 'RT-' (e.g. RT-A-2-01)" }],
        };
      }
      const docArg = doc ? ` ${shellEscapeArg(doc)}` : "";
      const result = runChainbench(`spec lookup ${shellEscapeArg(id)}${docArg}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );
}
