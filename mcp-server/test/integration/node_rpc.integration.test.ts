// node_rpc.integration.test.ts — first end-to-end test for the rerouted
// chainbench_node_rpc tool. Exercises the full path:
//
//   _nodeRpcHandler (TS)
//     -> callWire (TS spawn helper)
//       -> chainbench-net binary (Go)
//         -> rpc.Client.CallContext (Go)
//           -> Python http.server JSON-RPC mock
//
// The scaffolding (port allocation + Python JSON-RPC mock + state-dir
// seeding + env snapshot/restore + cleanup) lives in `_harness.ts`
// (extracted in Sprint 5c.4.1 Task 0). The test body is unchanged from
// Sprint 5c.3 Task 4 — same mock, same expected response — and serves
// as a regression check on the harness extraction.
//
// The suite skips itself when network/bin/chainbench-net is missing so
// CI without a Go toolchain still runs the unit suite. Local dev:
//   cd network && go build -o bin/chainbench-net ./cmd/chainbench-net

import { describe, it, expect, beforeAll, afterAll } from "vitest";
import {
  setupRealBinaryHarness,
  hasBinary,
  type RealBinaryHarness,
} from "./_harness.js";

describe.skipIf(!hasBinary())(
  "integration: chainbench_node_rpc real binary",
  () => {
    let harness: RealBinaryHarness | undefined;

    beforeAll(async () => {
      harness = await setupRealBinaryHarness({
        rpcHandlers: {
          eth_chainId: () => "0x539",
          eth_blockNumber: () => "0xdeadbeef",
        },
      });
    }, 15000);

    afterAll(async () => {
      if (harness) await harness.teardown();
    });

    it("eth_blockNumber happy path through real binary + Python mock", async () => {
      const { _nodeRpcHandler } = await import("../../src/tools/node.js");

      const out = await _nodeRpcHandler({
        node: 1,
        method: "eth_blockNumber",
      });

      expect(out.isError).not.toBe(true);
      const text = out.content[0]?.text ?? "";
      // The mock returns 0xdeadbeef; formatWireResult passes the wire data
      // straight through, so the rendered text must contain the hex result.
      expect(text).toContain("0xdeadbeef");
    });
  },
);
