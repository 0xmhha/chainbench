import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench, CHAINBENCH_DIR } from "../utils/exec.js";
import { readdirSync, readFileSync, statSync, existsSync } from "fs";
import { resolve } from "path";

function formatResult(result: { stdout: string; stderr: string; exitCode: number }): string {
  if (result.exitCode === 0) {
    return result.stdout || "Done.";
  }
  const detail = result.stderr || result.stdout || "unknown error";
  return `Error (exit ${result.exitCode}): ${detail}`;
}

function discoverTests(): string[] {
  const testsDir = resolve(CHAINBENCH_DIR, "tests");
  const tests: string[] = [];

  const scanDir = (dir: string, prefix: string): void => {
    let entries: string[];
    try {
      entries = readdirSync(dir);
    } catch {
      return;
    }
    for (const entry of entries) {
      if (entry === "lib") continue;
      const fullPath = resolve(dir, entry);
      let stat;
      try {
        stat = statSync(fullPath);
      } catch {
        continue;
      }
      if (stat.isDirectory()) {
        scanDir(fullPath, prefix ? `${prefix}/${entry}` : entry);
      } else if (entry.endsWith(".sh")) {
        const name = entry.replace(/\.sh$/, "");
        tests.push(prefix ? `${prefix}/${name}` : name);
      }
    }
  };

  scanDir(testsDir, "");
  return tests.sort();
}

function validateTestName(test: string): string | null {
  // Allow alphanumeric, dashes, underscores, and single forward-slashes for category/name
  if (!/^[a-zA-Z0-9_\-]+(\/[a-zA-Z0-9_\-]+)*$/.test(test)) {
    return `Invalid test name '${test}'. Use format 'category/test-name' (e.g. 'basic/consensus').`;
  }
  return null;
}

function validateReportFormat(format: string): string | null {
  const allowed = ["text", "json", "summary"];
  if (!allowed.includes(format)) {
    return `Invalid format '${format}'. Allowed values: ${allowed.join(", ")}.`;
  }
  return null;
}

export function registerTestTools(server: McpServer): void {
  server.tool(
    "chainbench_test_list",
    "List all available built-in tests. Tests are organized by category (e.g. basic/consensus, basic/tx-send). Use chainbench_test_run to execute a specific test.",
    {},
    async () => {
      // Try CLI first; fall back to filesystem discovery
      const result = runChainbench("test list");
      if (result.exitCode === 0 && result.stdout) {
        return { content: [{ type: "text" as const, text: result.stdout }] };
      }

      const tests = discoverTests();
      if (tests.length === 0) {
        return {
          content: [{ type: "text" as const, text: "No tests found in the tests/ directory." }],
        };
      }
      return {
        content: [
          {
            type: "text" as const,
            text: `Available tests:\n${tests.map((t) => `  ${t}`).join("\n")}`,
          },
        ],
      };
    }
  );

  server.tool(
    "chainbench_test_run",
    "Run a specific built-in test by name. The chain must be running before executing tests. Returns the test output including pass/fail results.",
    {
      test: z
        .string()
        .describe("Test name in 'category/name' format (e.g. 'basic/consensus', 'basic/tx-send'). Use chainbench_test_list to see all available tests."),
      format: z
        .enum(["text", "jsonl"])
        .optional()
        .default("text")
        .describe("Output format: 'text' (default) or 'jsonl' (NDJSON event stream for machine parsing)"),
    },
    async ({ test, format }) => {
      const validationError = validateTestName(test);
      if (validationError) {
        return { content: [{ type: "text" as const, text: `Error: ${validationError}` }] };
      }
      const formatFlag = format === "jsonl" ? " --format jsonl" : "";
      const result = runChainbench(`test run ${test}${formatFlag}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_test_run_remote",
    "Run tests against a registered remote chain. When target is 'all' or 'remote', only remote-compatible tests are executed. Requires a remote alias registered via chainbench_remote_add.",
    {
      test: z
        .string()
        .optional()
        .default("remote")
        .describe("Test name or category to run (default: 'remote' for all remote tests). Use 'remote/rpc-health' for a specific test."),
      alias: z
        .string()
        .regex(/^[a-zA-Z][a-zA-Z0-9-]*$/, "Alias must start with a letter, contain only alphanumeric and dashes")
        .describe("Remote chain alias to run tests against (e.g. 'eth-mainnet')"),
    },
    async ({ test, alias }) => {
      const validationError = validateTestName(test);
      if (validationError) {
        return { content: [{ type: "text" as const, text: `Error: ${validationError}` }] };
      }
      const result = runChainbench(`test run ${test} --remote ${alias}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_report",
    "Retrieve the test results report. Shows a summary of all tests that have been run, their pass/fail status, and timing information.",
    {
      format: z
        .string()
        .optional()
        .default("text")
        .describe("Output format: 'text' (default), 'json', or 'summary'"),
    },
    async ({ format }) => {
      const validationError = validateReportFormat(format);
      if (validationError) {
        return { content: [{ type: "text" as const, text: `Error: ${validationError}` }] };
      }

      const formatFlag = format === "json" ? " --json" : format === "summary" ? " --summary" : "";
      const result = runChainbench(`report${formatFlag}`);
      return { content: [{ type: "text" as const, text: formatResult(result) }] };
    }
  );

  server.tool(
    "chainbench_failure_context",
    "Retrieve the most recent failure context snapshot. " +
    "When a test fails, chainbench auto-captures per-node block height, peer count, " +
    "syncing status, recent blocks, and log tails into state/failures/. " +
    "Call this after a test failure to get full diagnostic data in one request.",
    {},
    async () => {
      const failuresDir = resolve(CHAINBENCH_DIR, "state", "failures");
      if (!existsSync(failuresDir)) {
        return { content: [{ type: "text" as const, text: "No failure context available. No tests have failed yet." }] };
      }

      // Find the most recent failure directory
      let dirs: string[];
      try {
        dirs = readdirSync(failuresDir)
          .map(d => resolve(failuresDir, d))
          .filter(d => { try { return statSync(d).isDirectory(); } catch { return false; } })
          .sort()
          .reverse();
      } catch {
        return { content: [{ type: "text" as const, text: "Error reading failure contexts." }] };
      }

      if (dirs.length === 0) {
        return { content: [{ type: "text" as const, text: "No failure context available." }] };
      }

      const latestDir = dirs[0];
      const contextFile = resolve(latestDir, "context.json");
      if (!existsSync(contextFile)) {
        return { content: [{ type: "text" as const, text: `Failure directory exists but context.json is missing: ${latestDir}` }] };
      }

      try {
        const contextJson = readFileSync(contextFile, "utf-8");
        return { content: [{ type: "text" as const, text: contextJson }] };
      } catch (e: any) {
        return { content: [{ type: "text" as const, text: `Error reading context.json: ${e.message}` }] };
      }
    }
  );

  server.tool(
    "chainbench_state_compact",
    "Return a compact JSON snapshot of the current chain state (< 300 bytes). " +
    "Includes running status, profile name, per-node block height and peer count, " +
    "consensus health, and last test result. Efficient for LLM context — call every turn.",
    {},
    async () => {
      const result = runChainbench("status --json --compact");
      if (result.exitCode === 0) {
        return { content: [{ type: "text" as const, text: result.stdout || "{}" }] };
      }
      // Fallback: construct minimal state from pids.json
      const pidsFile = resolve(CHAINBENCH_DIR, "state", "pids.json");
      if (!existsSync(pidsFile)) {
        return { content: [{ type: "text" as const, text: JSON.stringify({ running: false }) }] };
      }
      try {
        const pids = JSON.parse(readFileSync(pidsFile, "utf-8"));
        const compact: Record<string, unknown> = {
          running: true,
          profile: pids.profile || "unknown",
          nodes: Object.fromEntries(
            Object.entries(pids.nodes || {}).map(([k, v]: [string, any]) => [
              k,
              { block: "?", peers: "?", role: v.type || "?" },
            ])
          ),
        };
        return { content: [{ type: "text" as const, text: JSON.stringify(compact) }] };
      } catch {
        return { content: [{ type: "text" as const, text: JSON.stringify({ running: false, error: "pids.json parse failed" }) }] };
      }
    }
  );
}
