import { z } from "zod";
import type { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { runChainbench, CHAINBENCH_DIR } from "../utils/exec.js";
import { readdirSync, statSync } from "fs";
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
    },
    async ({ test }) => {
      const validationError = validateTestName(test);
      if (validationError) {
        return { content: [{ type: "text" as const, text: `Error: ${validationError}` }] };
      }
      const result = runChainbench(`test run ${test}`);
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
}
