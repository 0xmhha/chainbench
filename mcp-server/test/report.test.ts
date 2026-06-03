import { describe, it, expect, vi, beforeEach } from "vitest";

// Mock the CLI bridge so we can assert the exact argument string the report
// tool passes to chainbench.sh without spawning a real process. The D1 bug
// was that the tool emitted `--json` while cmd_report.sh only parses
// `--format json` — so the flag was silently ignored and text was returned,
// breaking C4's `summary.failed` loop-back parse.
const { runChainbench } = vi.hoisted(() => ({
  runChainbench: vi.fn(() => ({ stdout: "{}", stderr: "", exitCode: 0 })),
}));
vi.mock("../src/utils/exec.js", () => ({
  runChainbench,
  CHAINBENCH_DIR: "/tmp/chainbench-test",
}));

import { registerTestTools } from "../src/tools/test.js";

type Handler = (args: any) => Promise<{ content: { type: string; text: string }[] }>;

// Minimal McpServer stand-in: captures each registered tool's handler by name
// so a test can invoke it directly.
function captureTools(): Map<string, Handler> {
  const handlers = new Map<string, Handler>();
  const fakeServer = {
    tool: (name: string, _desc: string, _schema: unknown, handler: Handler) => {
      handlers.set(name, handler);
    },
  };
  registerTestTools(fakeServer as any);
  return handlers;
}

describe("chainbench_report format routing (D1)", () => {
  let report: Handler;

  beforeEach(() => {
    runChainbench.mockClear();
    report = captureTools().get("chainbench_report")!;
    expect(report).toBeTypeOf("function");
  });

  it("routes json through the --format flag the CLI actually parses", async () => {
    await report({ format: "json" });
    expect(runChainbench).toHaveBeenCalledWith("report --format json");
  });

  it("passes no format flag for text (the default)", async () => {
    await report({ format: "text" });
    expect(runChainbench).toHaveBeenCalledWith("report");
  });

  it("rejects summary at the boundary (dropped from the allowed set)", async () => {
    const out = await report({ format: "summary" });
    expect(runChainbench).not.toHaveBeenCalled();
    expect(out.content[0]?.text).toContain("Invalid format 'summary'");
  });
});
