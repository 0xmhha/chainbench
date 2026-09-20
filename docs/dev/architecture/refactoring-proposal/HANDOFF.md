# Handoff: continue the CLI-first architecture review on another machine

## Start here

This project is in **requirements discussion and refactoring preparation**, not product implementation. Continue with the user in small review units. Do not independently converge on a final package layout or spend the session repeating formal evaluations instead of presenting functional findings.

The user is moving hardware-intensive verification to a more capable machine. No further heavy tests were requested on the original machine. This handoff records the context; it does not certify the destination environment or any unexecuted test.

- Repository: `https://github.com/0xmhha/chainbench`
- Review PR: [#419](https://github.com/0xmhha/chainbench/pull/419), draft
- Branch: `codex/refactoring-proposal-review`
- Initial proposal commit: `c7a959c1`
- CLI review commit: `cb26b1ed`
- Product source revision reviewed: `7f39c5627e0bdaccaa8e207a67767d71579283f5`
- Read [current overview](README.md), then [CLI composition review](cli-composition-review.md), [target/plan draft](target-and-plan.md), and [verification plan](verification.md).
- PR titles and descriptions must be **English**. New handoff/review documents use English; conversation with the user is Korean. Older proposal documents still contain Korean.

Fetch the PR branch in a fresh checkout or worktree on the destination. Record its actual HEAD; do not assume the source baseline and documentation HEAD are the same. Preserve existing local changes. Example in an existing clean clone:

```sh
git fetch origin codex/refactoring-proposal-review
git switch --track origin/codex/refactoring-proposal-review
git status --short
git rev-parse HEAD
```

If the branch already exists, switch to it and inspect divergence before updating. Do not reset or clean a checkout to force these example commands to work.

## User requirements: authoritative discussion context

1. Local chain composition and test execution must be usable from CI.
2. Chain nodes must be deployable to remote servers in an isolated network, with tests against that environment. SSH support alone is not proof of closed-network readiness.
3. The CLI must expose operational functionality and test execution.
4. DSL test definitions must support composition and testing. Clarify the boundary between declarations and pre-provisioned binaries, credentials and server inventory.
5. Existing RPC URLs must allow tests supported by the available capabilities, without composing a chain.
6. MCP must expose sufficient capabilities and descriptions for an LLM to compose nodes/chains and optionally run tests. Composition alone is a valid completed operation. An LLM may use the resulting environment for other tests while developing `go-wemix`, `go-wbft`, or `go-stablenet`.
7. The product must include metrics collection and a dashboard for test status, node status, GUI-driven testing, and debugging. It is not merely a test-result viewer.
8. Test-definition filenames must move from `.json` to `.dsl` to reduce accidental selection of unrelated files. Content syntax, env-file suffixes, discovery, validation, examples and compatibility policy must be discussed; extension alone does not establish semantic validity.
9. Executable roles: `chainbench` for CLI, `chainbench-mcp` for MCP, `chainbenchd` for daemon. Daemon command transport remains undesigned. The current third binary is `chainbench-dashboard`; do not claim `chainbenchd` already exists.

The improvement scope is **chainbench**. The three chain repositories are references for integration contracts, not targets for internal refactoring in this proposal.

## Agreed sequencing and architectural constraints

**CLI functionality and refactoring preparation → MCP functionality → dashboard and daemon.**

The user agreed that CLI/CI execution should work without a daemon. A future daemon can host persistent environment management, metrics collection and dashboard control. Both modes should produce interoperable data that the dashboard can use. Do not decide that every CLI command must call a daemon.

Shared data does not imply shared process-control ownership. Distinguish environment identity, node identity, test-run identity, recorded state, live observations, writer identity, and resource ownership. Storage technology and daemon transport are open. Not all data must live in one file or database.

The original P0–P5 plan is provisional. It underrepresented remote operations, persistent environments and dashboard/daemon needs. The `.dsl` and executable-role requirements amend the earlier blanket promise to preserve all external contracts; provide migration decisions rather than silently removing these requirements.

The frozen Seed and its old evaluation have **not** been updated to certify these later clarifications. Keep historical results as history.

## Latest completed review: standalone CLI composition

See [the detailed graph and source map](cli-composition-review.md).

```text
chainbench chain up
  → chaincmd.newNetUpCmd
  → app.NetUp
  → chainsetup.NetUp / netUpFrom
  → new / place / keys / genesis / config / build / deploy / init / start
  → workspace methods → per-machine resource.Access
  → selected LocalDriver or RemoteDriver/SSH
  → recorded NodeSet returned; no test-engine execution

chainbench run --attach --workspace-dir <dir> <spec>
  → app.AttachRun → testengine.AttachWorkspaceRun
  → readWorkspaceComposed / wiredAttachEngine
  → Engine.Run
```

Confirmed source-level findings:

- `chain up --stage=deploy` stops before init/start. Here `build` assembles launch options; it does not compile the node binary.
- `--stage=start` is the default and requires a binary; a remote binary path refers to the remote host.
- Composition and later tests are separate requests. The composition call does not automatically tear down the network on return. Real-node survival after CLI exit still needs live verification.
- `workspace.json` stores control state locally; target data such as genesis, configs, datadirs and logs may live remotely.
- A composition ID already exists and is derived from workspace location. Do not assume arbitrary relocation preserves identity.
- Mutating steps save partial state even on failure. The composite and individual steps use workspace locking.
- `Composition.Save` directly calls `os.WriteFile`. Safe visibility to concurrent readers is an open concern, not a demonstrated corruption incident.
- `NetworkStatus` reads recorded state; a PID in a file is not a current liveness observation.
- Workspace attach carries more context/capabilities than bare RPC attach. Do not promise process-control, key or metric-dependent tests with only an RPC URL.
- CLI and engine artifact-root defaults differ in how they are supplied. Check effective paths before promising that every test session is stored beside its environment.

Do not replace these existing boundaries with a new abstraction solely because the earlier diagram proposed one.

## Immediate next discussion

The last open question is: **what should successful `chain up` guarantee—deployed inputs, launched processes, RPC readiness, or observed chain progress?**

These are distinct observations. Inspect the existing bootstrap/health paths and discuss the intended default with the user before changing behavior. Standalone composition must not require a test suite, but its completion may include explicit readiness checks if that is the agreed contract.

Next review units:

1. Local CI composition-only → inspect → later attach-test → explicit cleanup.
2. Isolated-network multi-server composition: binary/input availability, placement, SSH and RPC/metrics routes, logs, partial failure and cleanup.
3. DSL composition/testing and `.dsl` discovery/migration.
4. Existing-RPC test eligibility and stored environment/run relationships.
5. Only then consolidate CLI refactoring steps and move to MCP exposure.

Present findings and unresolved decisions after each unit, update this PR, and let the user refine the direction. Product implementation follows an agreed final document revision and explicit implementation scope.

## Verification already performed

### Latest CLI-focused checks

Go 1.25.13, `-count=1 -json -timeout=90s`, outer timeout 120 seconds. Selected existing tests in `cmd/chainbench/chaincmd`, `internal/chainsetup`, and `internal/app`.

- First attempt: 17 test events passed, 5 failed because the sandbox blocked the local allocator lock under the user's chainbench home.
- Retry with lock access: **22 passed**, all three packages passed; no skipped test events.
- [Committed receipt](cli-composition-verification.json) lists test names, package results and raw-log hashes.
- These checks are not full CLI coverage, live remote deployment, three-chain E2E or crash-consistency verification.

### Historical partial baseline

Baseline ID `20260913T101856Z`, product revision above, darwin/arm64 Go 1.25.13:

- 58 packages passed; 1,944 test/subtest PASS, 27 SKIP; 12 packages had no tests.
- CLI build/help/chains/capabilities, WBFT basic scenario and Wemix→WBFT handoff passed for their recorded inputs.
- Original Stablenet run failed: receipt observed, recipient latest balance 0→0; contract step not reached.
- Restored DB observations: nodes 1/3/4 at block 1 with 1 ETH; nodes 2/5 at genesis.
- Instrumented new network: 150 balance observations at 1 ETH and contract result 42 passed.
- One unmodified rerun passed in 42.47 seconds.

**The original Stablenet failure remains open.** Later passes do not prove its cause or resolution. Missing simultaneous receipt/canonical/state/node observations prevent a direct causal conclusion. RPC-error-to-genesis progress misclassification is a code concern, not a proven cause of that failure.

Hardfork, fault, synchronization, governance, full independent Wemix, live skipped cases, race, performance and long-duration coverage remain unverified. SKIP, timeout, interruption and unavailable environments are not PASS.

### Formal evaluation caveats

Original execution `orch_6cf3d0cff337` reported 3/3 execution criteria complete. That was not formal approval. Latest semantic evaluation `job_67642b613320` accepted target/plan and verification-procedure criteria, but rejected the graph-analysis criterion:

- New input documents appeared after the snapshot; original freshness checks failed.
- A global ID/evidence-reference check across all AC1/AC2/AC3 artifacts remains incomplete.
- Separate preserved-snapshot diagnostic passes do not replace those failures.

Mechanical evaluation was unreliable: inherited-environment lint reported three SA5011 findings; direct pinned linter execution passed. A later environment-prefixed configuration caused all five mechanical checks to be skipped. That apparent PASS was rejected and the configuration restored. Do not copy it or disable checks to obtain approval. Exact cause is unresolved. No evaluation job was left running at handoff.

## Destination-machine preparation

Before hardware-intensive testing, record the actual available CPU, RAM, disk, OS/architecture, toolchain, caches and network access. Agree on topology and resource budget; no node-count or performance threshold has been finalized.

- Read destination `AGENTS.md` and current repository instructions.
- Resolve Go from `go.mod` and linter from `.golangci-version`; the reviewed versions were Go 1.25.13 and golangci-lint v2.12.2. Do not copy the original machine's absolute GOROOT/cache paths.
- Obtain the correct `go-wemix`, `go-wbft` and `go-stablenet` checkouts and record revisions/dirty state. Record actual binary SHA-256 and build provenance separately. Rebuild for the destination OS/architecture when required.
- Prepare genesis, keys, topology, server inventory and fixtures. Keep credentials and private keys out of PRs and ordinary logs.
- Inspect each selected test's required environment variables and Skip conditions. E2E uses `-tags e2e`; relevant paths include CHAINBENCH, WBFT_BIN, GSTABLE_BIN and WEMIX_BIN, with additional handoff/hardfork inputs as applicable. Do not invent missing values.
- Check ports, SSH/host identity, filesystem paths and RPC/metrics reachability. In a closed network, explicitly verify that binaries and required assets are available without public downloads.
- Use a new run directory and explicit time/resource limits. Preserve command, working directory, input hashes, elapsed time, terminal status, test events and cleanup results.
- Clean up only resources owned by the current run. Never terminate attached user nodes or delete prior evidence to make a retry pass.
- Host changes create a new baseline; they do not retroactively update the darwin/arm64 historical baseline. Compare equivalent inputs and classify changed conditions.

Follow [verification.md](verification.md) for bounded scenarios and Stablenet observations. Before making a compatibility claim, distinguish recorded source-level paths, mocked/local automated tests and actual live chain results.

## What the PR contains, and what must be transferred separately

The branch contains human-readable analysis, target-plan draft, verification procedures, selected evidence excerpts, the latest CLI call-path review and a compact test receipt. It does **not** contain the complete graphs, raw baselines or Stablenet audit archives. A clone alone cannot reproduce every historical claim.

Original-machine locations (inventory only; these paths will not exist automatically on the destination):

| Material | Original location |
|---|---|
| Main checkout, uncommitted Seed and historical evidence | `/Users/0xtopaz/work/github/0xmhha/chainbench` |
| Frozen Seed | `docs/dev/architecture/chainbench-refactoring-preparation.seed-v1.0.5.frozen.yaml` under main checkout |
| Historical baseline | `docs/dev/architecture/baselines/20260913T101856Z` under main checkout |
| Stablenet audit | `docs/dev/architecture/stablenet-boundary-audit` under main checkout |
| Full analysis execution worktree | `/Users/0xtopaz/.ouroboros/worktrees/chainbench/orch_6cf3d0cff337` |
| Canonical graph artifacts | `docs/dev/architecture/preparation-ac1-v2/artifacts-verified` under execution worktree; inputs/manifests and scripts in the surrounding preparation directories are needed for reproduction |
| Earlier plan/procedures | `preparation-ac2-v1` and `preparation-ac3-retry1` under execution worktree's architecture directory |
| Formal diagnostic logs | `/tmp/chainbench-formal-eval-probe` and `/tmp/chainbench-eval-remediation` |
| Latest raw CLI test logs | `/tmp/chainbench-cli-composition-review-tests.jsonl` and `/tmp/chainbench-cli-composition-review-tests-retry.jsonl` |
| PR editing worktree | `/tmp/chainbench-refactoring-proposal-review` |
| Chain reference repositories | `/Users/0xtopaz/work/github/0xmhha/chain/go-wemix`, `/Users/0xtopaz/work/github/0xmhha/chain/go-wbft`, `/Users/0xtopaz/work/github/0xmhha/chain/go-stablenet` |

Frozen Seed SHA-256: `b0aaf9063b03c9616f95a4a92a7fa0bee4177a874b662f052dcaf50e707cac37`.

If continued review needs full historical reproduction, arrange a deliberate transfer of the corresponding inputs, scripts, manifests and logs; inspect for sensitive data and retain hashes. Those archives have not been uploaded by this handoff. If unavailable, label historical evidence unavailable rather than claiming revalidation. `/tmp` files are temporary and should not be treated as a durable archive. Existing original-checkout changes, including `.gitignore`, were left untouched.

## Suggested opening message for the next assistant

> Continue PR #419 on branch codex/refactoring-proposal-review. Read HANDOFF.md and cli-composition-review.md first. We are collaboratively reviewing CLI functionality before refactoring; MCP follows CLI, then dashboard/daemon. CLI is daemon-independent, but records must be reusable by both modes and dashboard. Do not implement the old P0–P5 plan or run another formal evaluation by default. Start by checking the destination environment and reviewing what standalone chain-up completion guarantees, then propose one bounded local or remote verification scenario with explicit inputs and resource limits. Preserve the original Stablenet failure and distinguish all new evidence from the historical baseline. Update the existing PR as review decisions are made.
