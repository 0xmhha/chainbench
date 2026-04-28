// chain_read_timeout.test.ts — pinning regression test for the wire-helper
// timeout grace logic in _txWaitHandler.
//
// Sprint 5c.2 Task 3 review caught a critical bug: the handler's caller-
// timeout fallback was 30000ms while chainbench-net's actual server-side
// default (defaultTxWaitMs in handlers_node_tx.go) is 60000ms. With the
// wrong fallback, the wire helper SIGTERMs the spawn at 35s while the Go
// handler is still polling until 60s — defeating the entire grace logic.
//
// This test pins the corrected boundary in place by asserting the wire
// helper's `timeoutMs` option is exactly `caller_timeout + 5000ms` for both
// the default-fallback path (60000 + 5000 = 65000) and a caller-provided
// value (5000 + 5000 = 10000).
//
// File-scoped vi.mock is used (rather than monkey-patching the imported
// module object) because ESM modules are typically frozen, and the mock is
// isolated to this file so it can't leak into the broader chain_read suite.

import { describe, it, expect, vi } from "vitest";

const callWireSpy = vi.fn();

vi.mock("../src/utils/wire.js", async () => {
  const actual =
    await vi.importActual<typeof import("../src/utils/wire.js")>(
      "../src/utils/wire.js",
    );
  return {
    ...actual,
    callWire: (
      cmd: string,
      args: Record<string, unknown>,
      opts?: { timeoutMs?: number },
    ) => {
      callWireSpy(cmd, args, opts);
      return Promise.resolve({
        result: {
          type: "result" as const,
          ok: true as const,
          data: { status: "pending", tx_hash: args.tx_hash },
        },
        events: [],
        progress: [],
        stderr: "",
        exitCode: 0,
      });
    },
  };
});

describe("chainbench_tx_wait handler — wire timeout grace", () => {
  it("_TimeoutGrace_PassedToWireHelper", async () => {
    // Import after vi.mock has been hoisted so the handler picks up the
    // patched callWire.
    const { _txWaitHandler } = await import("../src/tools/chain_read.js");
    const goodHash = "0x" + "a".repeat(64);

    // Default path: caller omits timeout_ms → handler must fall back to
    // 60000ms (chainbench-net defaultTxWaitMs) and add 5000ms grace, so the
    // wire helper sees 65000ms.
    callWireSpy.mockClear();
    await _txWaitHandler({ network: "local", tx_hash: goodHash });
    expect(callWireSpy).toHaveBeenCalledTimes(1);
    expect(callWireSpy.mock.calls[0]?.[2]?.timeoutMs).toBe(65000);

    // Custom path: caller specifies timeout_ms: 5000 → wire timeout is
    // 5000 + 5000 = 10000.
    callWireSpy.mockClear();
    await _txWaitHandler({
      network: "local",
      tx_hash: goodHash,
      timeout_ms: 5000,
    });
    expect(callWireSpy).toHaveBeenCalledTimes(1);
    expect(callWireSpy.mock.calls[0]?.[2]?.timeoutMs).toBe(10000);
  });
});
