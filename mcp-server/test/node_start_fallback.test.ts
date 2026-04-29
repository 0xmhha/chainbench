// node_start_fallback.test.ts — pinning test for the chainbench_node_start
// binary_path fallback path.
//
// Sprint 5c.3 Task 3 reroutes node.stop and node.start through callWire, but
// node.start has a known asymmetry: the wire layer's node.start handler does
// not yet accept binary_path. When the caller supplies binary_path the
// handler falls back to runChainbench (bash CLI). This test verifies the
// fallback fires with the expected shell-quoted argument and surfaces the
// process result correctly.
//
// File-scoped vi.mock is used (rather than monkey-patching the imported
// module object) because ESM modules are typically frozen, and the mock is
// isolated to this file so it can't leak into the broader node.test suite.
// Pattern mirrors chain_read_timeout.test.ts.

import { describe, it, expect, vi } from "vitest";

const runChainbenchSpy = vi.fn();

vi.mock("../src/utils/exec.js", async () => {
  const actual =
    await vi.importActual<typeof import("../src/utils/exec.js")>(
      "../src/utils/exec.js",
    );
  return {
    ...actual,
    runChainbench: (args: string, options?: { cwd?: string }) => {
      return runChainbenchSpy(args, options);
    },
  };
});

describe("chainbench_node_start handler — binary_path fallback", () => {
  it("_NodeStart_WithBinaryPath_FallsBackToBash", async () => {
    // Import after vi.mock has been hoisted so the handler picks up the
    // patched runChainbench.
    const { _nodeStartHandler } = await import("../src/tools/node.js");

    runChainbenchSpy.mockClear();
    runChainbenchSpy.mockReturnValue({
      stdout: "Started node 1.",
      stderr: "",
      exitCode: 0,
    });

    const out = await _nodeStartHandler({
      node: 1,
      binary_path: "/opt/chains/wemix/bin/wemix",
    });

    // Bash fallback fired exactly once with the expected, shell-escaped arg.
    expect(runChainbenchSpy).toHaveBeenCalledTimes(1);
    const cmd = runChainbenchSpy.mock.calls[0]?.[0] as string;
    expect(cmd).toContain("node start 1");
    expect(cmd).toContain("--binary-path");
    // Single-quoted POSIX shell escape preserves the absolute path verbatim.
    expect(cmd).toContain("'/opt/chains/wemix/bin/wemix'");

    // Handler surfaces the bash result as a non-error response.
    expect(out.isError).toBeFalsy();
    expect(out.content[0]?.text).toContain("Started node 1.");
  });

  it("_NodeStart_WithBinaryPath_BashFailure_PropagatesIsError", async () => {
    const { _nodeStartHandler } = await import("../src/tools/node.js");

    runChainbenchSpy.mockClear();
    runChainbenchSpy.mockReturnValue({
      stdout: "",
      stderr: "binary not found",
      exitCode: 2,
    });

    const out = await _nodeStartHandler({
      node: 1,
      binary_path: "/missing/binary",
    });

    expect(runChainbenchSpy).toHaveBeenCalledTimes(1);
    expect(out.isError).toBe(true);
    expect(out.content[0]?.text).toContain("Error");
    expect(out.content[0]?.text).toContain("binary not found");
  });
});
