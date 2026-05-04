// lifecycle.integration.test.ts — first end-to-end test for the rerouted
// chainbench_status tool (Sprint 5c.4.1 Task 4). Exercises the full path:
//
//   _statusHandler (TS)
//     -> callWire (TS spawn helper)
//       -> chainbench-net binary (Go: network.status handler)
//         -> os/exec spawn of chainbench.sh status --json
//
// We stub the bash chainbench.sh with a fake script written into the temp
// state dir (via the harness's `fakeChainbenchScript` option). The Go
// handler honours CHAINBENCH_DIR which the harness sets, so the spawn
// hits the fake script instead of the real one — keeping the test
// hermetic.
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

describe.skipIf(!hasBinary())("integration: chainbench_status", () => {
  let harness: RealBinaryHarness | undefined;

  beforeAll(async () => {
    // Fake chainbench.sh that emits a known JSON status when invoked
    // with `status --json`. Any other invocation exits non-zero so the
    // Go handler classifies the failure as UPSTREAM_ERROR if the test
    // ever drifts into an unexpected path.
    const fakeScript = `#!/bin/bash
if [[ "$1" == "status" && "$2" == "--json" ]]; then
  cat <<'EOF'
{"nodes":[{"id":"node1","running":true,"block_height":42}],"healthy":true}
EOF
  exit 0
fi
exit 1
`;
    harness = await setupRealBinaryHarness({
      fakeChainbenchScript: fakeScript,
    });
  }, 15000);

  afterAll(async () => {
    if (harness) await harness.teardown();
  });

  it("end-to-end status through chainbench-net", async () => {
    const { _statusHandler } = await import("../../src/tools/lifecycle.js");
    const out = await _statusHandler({});
    const text = out.content[0]?.text ?? "";
    expect(text).toContain("healthy");
    expect(text).toContain("block_height");
    expect(out.isError).toBeFalsy();
  });
});
