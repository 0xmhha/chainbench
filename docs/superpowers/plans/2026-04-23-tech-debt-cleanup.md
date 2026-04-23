# Tech Debt Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Resolve three tech-debt items flagged by prior sprint reviews:
1. 3 pre-existing unit test failures caused by bash-3 incompatibility on macOS
2. `resolveNodeID` re-marshal trick in `handleNodeTailLog` (Sprint 2b.4 M3)
3. Duplicate E2E test setup across `e2e_test.go` + `handlers_test.go` (Sprint 2b.3 M5)

**Architecture:** Small, targeted refactors. No spec needed — each task is ≤50 lines of change. Brief scope notes inline.

**Tech Stack:** bash, Go 1.25. No new dependencies.

---

## Task 1: Bash 4+ detection in `tests/unit/run.sh`

**Problem:** `cmd-spec-lookup.sh`, `cmd-test-dryrun.sh`, `common-parse-overrides.sh` use bash 4+ features (`local -n` nameref, associative arrays). macOS default bash is 3.2, so `tests/unit/run.sh` reports 3/21 failed on macOS.

**Solution:** `run.sh` re-execs with `/opt/homebrew/bin/bash` or `/usr/local/bin/bash` if current bash is < 4.0. If neither available, fail fast with a clear error message.

**Files:**
- Modify: `tests/unit/run.sh`

- [ ] **Step 1.1: Confirm current behavior**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
bash tests/unit/run.sh 2>&1 | tail -5
```
Expected: "Total: 21 | Passed: 18 | Failed: 3" with failing tests listed.

- [ ] **Step 1.2: Read current `run.sh`**

```bash
cat tests/unit/run.sh
```
Note the shebang (`#!/usr/bin/env bash`), the `set -euo pipefail`, and the test iteration loop.

- [ ] **Step 1.3: Insert bash-4+ gate just after `set -euo pipefail`**

Use Edit to add this block. Locate:
```bash
#!/usr/bin/env bash
# tests/unit/run.sh - Unit test runner for chainbench
# Iterates tests/*.sh, executes each in a subshell, and reports pass/fail totals.
set -euo pipefail
```

Append after `set -euo pipefail`:
```bash

# Some unit tests use bash 4+ features (local -n namerefs, associative arrays).
# macOS ships bash 3.2 by default; re-exec with a modern bash if available,
# or fail fast with a clear message.
if (( BASH_VERSINFO[0] < 4 )); then
  for alt in /opt/homebrew/bin/bash /usr/local/bin/bash; do
    if [[ -x "$alt" ]] && "$alt" -c '(( BASH_VERSINFO[0] >= 4 ))' 2>/dev/null; then
      exec "$alt" "$0" "$@"
    fi
  done
  printf 'ERROR: tests/unit/run.sh requires bash 4+ (current: %s)\n' "$BASH_VERSION" >&2
  printf '  macOS: brew install bash\n' >&2
  printf '  Linux: system bash is usually 4+\n' >&2
  exit 1
fi
```

- [ ] **Step 1.4: Run harness — all 21 tests should now PASS**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench && bash tests/unit/run.sh 2>&1 | tail -10
```
Expected: "Total: 21 | Passed: 21 | Failed: 0".

- [ ] **Step 1.5: Verify explicit invocation with bash 3 re-execs cleanly**

```bash
/bin/bash tests/unit/run.sh 2>&1 | tail -10
```
Expected: same "Passed: 21" output (re-exec path taken).

- [ ] **Step 1.6: Lint the shell**

```bash
bash -n tests/unit/run.sh && echo "syntax ok"
```
Expected: `syntax ok`.

- [ ] **Step 1.7: Commit**

```bash
git add tests/unit/run.sh
git commit -m "test: require bash 4+ with re-exec fallback for unit harness"
```

---

## Task 2: Refactor `resolveNodeID` — split string and args variants

**Problem:** `handleNodeTailLog` in `network/cmd/chainbench-net/handlers.go` uses a JSON re-marshal trick to call `resolveNodeID(stateDir, args json.RawMessage)` when it only needs the node ID string. Sprint 2b.4 M3 flagged this as an awkward workaround.

**Solution:** Extract the core resolution into `resolveNodeIDFromString(stateDir string, nodeID string)`. The existing `resolveNodeID(stateDir, args)` stays as a thin args-parsing wrapper that calls the new string form. All three existing handlers (stop/start/restart) continue to use the args-form; `handleNodeTailLog` switches to the string-form.

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`

- [ ] **Step 2.1: Read current `handlers.go` to identify `resolveNodeID` boundaries**

```bash
grep -n 'resolveNodeID\|newHandleNodeTailLog' network/cmd/chainbench-net/handlers.go
```
Note line numbers of the function definition and its call sites.

- [ ] **Step 2.2: Refactor `resolveNodeID` into two functions**

Replace the current `resolveNodeID` function body (approximately 25 lines) with TWO functions:

```go
// resolveNodeIDFromString looks up nodeID in pids.json and returns the
// (nodeID, nodeNum) pair. Returns APIError sentinels:
//   INVALID_ARGS   — empty or malformed node_id, node not found
//   UPSTREAM_ERROR — state.LoadActive failure
func resolveNodeIDFromString(stateDir, nodeID string) (string, string, error) {
	if nodeID == "" {
		return "", "", NewInvalidArgs("args.node_id is required")
	}
	if !strings.HasPrefix(nodeID, "node") {
		return "", "", NewInvalidArgs(fmt.Sprintf(`node_id must start with "node" prefix (got %q)`, nodeID))
	}
	num := strings.TrimPrefix(nodeID, "node")
	if num == "" {
		return "", "", NewInvalidArgs("node_id missing numeric suffix")
	}
	net, lerr := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "local"})
	if lerr != nil {
		return "", "", NewUpstream("failed to load active state", lerr)
	}
	for _, n := range net.Nodes {
		if n.Id == nodeID {
			return nodeID, num, nil
		}
	}
	return "", "", NewInvalidArgs(fmt.Sprintf("node_id %q not found in active network", nodeID))
}

// resolveNodeID parses args into {node_id} then delegates to
// resolveNodeIDFromString. Used by handlers that receive the full wire
// envelope args.
func resolveNodeID(stateDir string, args json.RawMessage) (string, string, error) {
	var req struct {
		NodeID string `json:"node_id"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &req); err != nil {
			return "", "", NewInvalidArgs(fmt.Sprintf("args: %v", err))
		}
	}
	return resolveNodeIDFromString(stateDir, req.NodeID)
}
```

- [ ] **Step 2.3: Simplify `handleNodeTailLog` to call the string form directly**

Locate the body of `newHandleNodeTailLog`. Find the block that looks like:
```go
		nidPayload, _ := json.Marshal(map[string]any{"node_id": req.NodeID})
		nodeID, _, rerr := resolveNodeID(stateDir, nidPayload)
		if rerr != nil {
			return nil, rerr
		}
```

Replace with:
```go
		nodeID, _, rerr := resolveNodeIDFromString(stateDir, req.NodeID)
		if rerr != nil {
			return nil, rerr
		}
```

- [ ] **Step 2.4: Run existing tests — all should pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/...
```
Expected: all tests pass (this is a pure refactor with no behavior change).

- [ ] **Step 2.5: Build + vet + fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go vet ./... && gofmt -l cmd/chainbench-net/
```
Expected: all clean.

- [ ] **Step 2.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go
git commit -m "network: split resolveNodeID into args and string variants"
```

---

## Task 3: Consolidate E2E test setup helper

**Problem:** `e2e_test.go` has a repeated state + chainbenchDir setup pattern across 4 tests (NetworkLoad, NodeStop, NodeStart, NodeRestart, NodeTailLog — each ~15-25 lines of setup). Sprint 2b.3 M5 flagged this as ~40+ lines of duplication.

**Solution:** Extract `setupE2EDirs(t)` to the top of `e2e_test.go` that returns `(stateDir, chainbenchDir string)`. The tail-log test has a slightly different shape (writes an extra log file + rewrites pids.json), so it takes an inline variant.

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 3.1: Read current `e2e_test.go`**

```bash
cat network/cmd/chainbench-net/e2e_test.go | head -60
```
Identify the 4 setup blocks.

- [ ] **Step 3.2: Insert `setupE2EDirs` helper near the top of the file**

After the `import` block, before the first test function, add:

```go
// setupE2EDirs builds two tempdirs: stateDir populated with pids.json +
// current-profile.yaml fixtures, and chainbenchDir with the stub script copied
// in as chainbench.sh (so drivers that exec "${dir}/chainbench.sh" find it).
//
// Tests that need to write custom log files or mutate fixtures should do so
// after calling this helper. Tests that don't use the stub (network.load,
// tail_log) can pass a nil chainbenchDir override by ignoring the second
// return value.
func setupE2EDirs(t *testing.T) (stateDir, chainbenchDir string) {
	t.Helper()
	stateDir = t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stateDir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	chainbenchDir = t.TempDir()
	stub, err := os.ReadFile(filepath.Join("testdata", "chainbench-stub.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chainbenchDir, "chainbench.sh"), stub, 0o755); err != nil {
		t.Fatal(err)
	}
	return stateDir, chainbenchDir
}
```

- [ ] **Step 3.3: Replace each test's setup block with a call to the helper**

For `TestE2E_NetworkLoad_ViaRootCommand`, `TestE2E_NodeStop_ViaRootCommand`, `TestE2E_NodeStart_ViaRootCommand`, `TestE2E_NodeRestart_ViaRootCommand_EventOrder`:

Replace the block (approximately):
```go
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// ... and chainbenchDir section ...
```

with:
```go
	stateDir, chainbenchDir := setupE2EDirs(t)
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
	t.Setenv("CHAINBENCH_DIR", chainbenchDir)
```

(Note: for `TestE2E_NetworkLoad_ViaRootCommand`, the original only needs stateDir — the chainbenchDir assignment is harmless but unused. Acceptable.)

For `TestE2E_NodeTailLog_ViaRootCommand`, the setup differs (rewrites pids.json + writes a log file), so leave as-is OR do a more surgical extract. **Do NOT modify the tail_log test in this refactor** — its setup shape is distinct enough that forcing it into the helper would hurt readability.

**CRITICAL**: Preserve each test's specific assertions exactly — this is a pure setup refactor. Do not rewrite any `assert*` / `if`/`switch` logic.

- [ ] **Step 3.4: Run all E2E tests**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/... -run TestE2E
```
Expected: all 5 E2E tests PASS.

- [ ] **Step 3.5: Full cmd package + module**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./... && go vet ./... && gofmt -l .
```
Expected: all clean.

- [ ] **Step 3.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "test: consolidate E2E setup via setupE2EDirs helper"
```

---

## Final verification

- [ ] **Commit list**

```bash
git log --oneline c3992c6..HEAD
```
Expected 3 commits:
1. `test: require bash 4+ with re-exec fallback for unit harness`
2. `network: split resolveNodeID into args and string variants`
3. `test: consolidate E2E setup via setupE2EDirs helper`

- [ ] **Full test suites**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
bash tests/unit/run.sh 2>&1 | tail -5           # expect 21/21 pass
cd network && go test -race ./... && go vet ./... && gofmt -l .
```

---

## Out of scope (explicit)

- Sprint 2b.3 M3 (`APIError.Details` structured error fields) — defer until 2nd multi-phase command is needed
- Sprint 2b.3 M4 (`resolveNodeID` hardcoded `Name: "local"`) — defer until 2nd network name is needed (Sprint 3 remote driver era)
- Sprint 2c M3 (`_cb_net_parse_result` jq 3-call optimization) — defer; current volumes fine
- Sprint 2c M4 (jq version gate) — defer until newer jq features used
- Table-driven refactor of `TestHandleNode{Stop,Start,Restart}_*` — out of scope (readable enough today)
