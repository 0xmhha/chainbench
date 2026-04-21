# Network Module Rename Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the `hal/` Go module to `network/` (module path, directory, binary `chainbench-hal` → `chainbench-net`, package doc-comments, README, `.gitignore`), update top-level docs (VISION_AND_ROADMAP + historical plan filename), patch `.claude/settings.local.json`, and produce a **single atomic commit** so no intermediate build-broken state enters history.

**Architecture:** Mechanical find-replace rename driven by the design at `docs/superpowers/specs/2026-04-21-network-rename-design.md`. Tasks 1–7 edit the working tree without committing; Task 8 performs full verification and creates the single commit. All Go types (`NetworkController`, `Network`, `Node`, `LocalDriver`, `RemoteDriver`) stay the same — only the module path and the "chainbench-hal" binary-name string change in code.

**Tech Stack:** Go 1.25+, git, bash.

**Reference:** `docs/superpowers/specs/2026-04-21-network-rename-design.md` (mapping, scope, verification checklist).

---

## File Structure

**Renamed (git mv, history preserved):**
- `hal/` → `network/` (entire directory, 27 files)
- `hal/cmd/chainbench-hal/` → `network/cmd/chainbench-net/`
- `docs/superpowers/plans/2026-04-20-hal-foundation.md` → `docs/superpowers/plans/2026-04-20-network-foundation.md`

**Modified (content edit only):**
- `network/go.mod` (module path)
- `network/cmd/chainbench-net/main.go` (`Use:` + printf literal)
- `network/cmd/chainbench-net/main_test.go` (assertion)
- `network/schema/schema.go` (package doc comment)
- `network/schema/event.json` (description field)
- `network/tools.go` (doc comment)
- `network/.gitignore` (anchored ignore pattern line)
- `network/README.md` (all references)
- `network/internal/types/doc.go` (nothing to change unless "hal" appears — verified none)
- `docs/VISION_AND_ROADMAP.md` (63 HAL + ~179 "hal" occurrences, plus §5.15 Android HAL reframe)
- `docs/superpowers/plans/2026-04-20-network-foundation.md` (after rename, internal refs updated)
- `.claude/settings.local.json` (3 Bash allowlist lines)

**Unchanged:**
- Go types (`NetworkController`, `Network`, `Node`, etc.)
- JSON Schema `$id` URIs (no `hal` reference)
- `lib/adapters/*.sh`, `lib/chain_adapter.sh` (separate axis)
- `scripts/inventory/*.sh` (no `hal` refs)
- All other `chainbench.sh`, `install.sh`, `setup.sh`, root `README.md` (verified clean of `hal`/`chainbench-hal`)
- Historic commits / git log
- `docs/superpowers/specs/2026-04-21-network-rename-design.md` (meta doc about this rename; mentions HAL intentionally)

---

## Task 1: Capture pre-rename baseline (read-only)

**Files:** none changed; output saved to `/tmp/rename-baseline/` for comparison.

- [ ] **Step 1.1: Record current SHA and tree state**

```bash
mkdir -p /tmp/rename-baseline
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git rev-parse HEAD > /tmp/rename-baseline/pre-sha
git ls-files hal > /tmp/rename-baseline/hal-files.txt
wc -l /tmp/rename-baseline/hal-files.txt
```
Expected: `27 /tmp/rename-baseline/hal-files.txt`

- [ ] **Step 1.2: Verify build/tests green before starting**

```bash
cd hal && go build ./... && go test ./... && go vet ./... && gofmt -l . && cd ..
```
Expected: all exit 0, `gofmt -l .` prints nothing, tests show 3 packages PASS.

- [ ] **Step 1.3: Snapshot grep counts**

```bash
git grep -win 'hal'             -- '*.go' '*.sh' '*.md' '*.json' '*.mod' '*.sum' ':(exclude)mcp-server/node_modules' | wc -l > /tmp/rename-baseline/hal-count-before
git grep -ln  'chainbench-hal' -- '*.go' '*.sh' '*.md' '*.json' ':(exclude)mcp-server/node_modules' | wc -l > /tmp/rename-baseline/binname-files-before
cat /tmp/rename-baseline/hal-count-before /tmp/rename-baseline/binname-files-before
```
Expected (approximate): `hal-count-before` ≥ 200, `binname-files-before` = 10.

---

## Task 2: Directory + cmd subdir rename (git mv)

**Files:**
- `hal/` → `network/` (git rename, all 27 files)
- `hal/cmd/chainbench-hal/` → `network/cmd/chainbench-net/` (git rename after previous)

- [ ] **Step 2.1: Rename top-level module directory**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git mv hal network
git status -s | head -15
```
Expected: each of the 27 files shows `R  hal/... -> network/...`. No untracked/unstaged surprises.

- [ ] **Step 2.2: Rename cmd binary subdirectory**

```bash
git mv network/cmd/chainbench-hal network/cmd/chainbench-net
git status -s | grep chainbench | head -5
```
Expected:
```
R  hal/cmd/chainbench-hal/main.go -> network/cmd/chainbench-net/main.go
R  hal/cmd/chainbench-hal/main_test.go -> network/cmd/chainbench-net/main_test.go
```

- [ ] **Step 2.3: Verify tree structure**

```bash
ls network/ && ls network/cmd/
```
Expected: `cmd/ internal/ schema/ ...` and `chainbench-net` (no `chainbench-hal`).

---

## Task 3: Update Go module path + verify resolution

**Files:**
- Modify: `network/go.mod`

- [ ] **Step 3.1: Update module line**

Edit `/Users/wm-it-22-00661/Work/github/tools/chainbench/network/go.mod` line 1.

Before:
```
module github.com/0xmhha/chainbench/hal
```

After:
```
module github.com/0xmhha/chainbench/network
```

Use Edit tool with exact old_string `module github.com/0xmhha/chainbench/hal` → new_string `module github.com/0xmhha/chainbench/network`.

- [ ] **Step 3.2: Run `go mod tidy` (expected no-op for deps; module path normalized)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go mod tidy
```
Expected: exit 0, `go.sum` unchanged (no external deps reference our module path).

- [ ] **Step 3.3: Confirm no Go source imports the old path**

```bash
grep -rn 'github.com/0xmhha/chainbench/hal' network/ 2>&1
```
Expected: no matches (empty output, exit 1).

- [ ] **Step 3.4: Attempt build — expect it to still succeed**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./...
```
Expected: exit 0. (No Go source files import `chainbench/hal` — our Go code is self-contained per package.)

---

## Task 4: Update `chainbench-hal` literals in code

**Files (all under `network/`):**
- Modify: `network/cmd/chainbench-net/main.go`
- Modify: `network/cmd/chainbench-net/main_test.go`
- Modify: `network/schema/schema.go`
- Modify: `network/schema/event.json`
- Modify: `network/tools.go`
- Modify: `network/.gitignore`

- [ ] **Step 4.1: Update `main.go` — cobra `Use:` and version printf**

Edit `network/cmd/chainbench-net/main.go`.

Replace (with Edit tool, exact match):
```go
		Use:           "chainbench-hal",
```
with:
```go
		Use:           "chainbench-net",
```

Then replace:
```go
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "chainbench-hal %s\n", version)
```
with:
```go
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "chainbench-net %s\n", version)
```

- [ ] **Step 4.2: Update `main_test.go` — test assertion**

Edit `network/cmd/chainbench-net/main_test.go`.

Replace:
```go
	if !strings.HasPrefix(out, "chainbench-hal ") {
		t.Fatalf("want prefix %q, got %q", "chainbench-hal ", out)
	}
```
with:
```go
	if !strings.HasPrefix(out, "chainbench-net ") {
		t.Fatalf("want prefix %q, got %q", "chainbench-net ", out)
	}
```

- [ ] **Step 4.3: Update `schema.go` doc comment**

Edit `network/schema/schema.go`. Replace:
```go
// Package schema embeds the JSON Schemas that define the chainbench-hal
```
with:
```go
// Package schema embeds the JSON Schemas that define the chainbench-net
```

- [ ] **Step 4.4: Update `event.json` description**

Edit `network/schema/event.json`. Replace:
```json
  "description": "One JSON object per line on chainbench-hal stdout. A stream always terminates with exactly one type=result message.",
```
with:
```json
  "description": "One JSON object per line on chainbench-net stdout. A stream always terminates with exactly one type=result message.",
```

- [ ] **Step 4.5: Update `tools.go` doc comment**

Edit `network/tools.go`. Replace:
```go
// in go.mod but not compiled into the chainbench-hal binary.
```
with:
```go
// in go.mod but not compiled into the chainbench-net binary.
```

- [ ] **Step 4.6: Update `.gitignore` binary pattern**

Edit `network/.gitignore`. Replace:
```
/chainbench-hal
```
with:
```
/chainbench-net
```

- [ ] **Step 4.7: Verify no `chainbench-hal` remains inside `network/`**

```bash
grep -rn 'chainbench-hal' /Users/wm-it-22-00661/Work/github/tools/chainbench/network 2>&1
```
Expected: no matches (empty output, exit 1).

- [ ] **Step 4.8: Verify build + tests now green**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go test ./... && go vet ./... && gofmt -l .
```
Expected:
- `go build`: exit 0
- `go test`: 3 packages PASS (`cmd/chainbench-net`, `internal/types`, `schema`)
- `go vet`: exit 0
- `gofmt -l .`: empty output

---

## Task 5: Update `network/README.md`

**Files:**
- Modify: `network/README.md`

- [ ] **Step 5.1: Replace all `hal` and `chainbench-hal` occurrences**

Open `network/README.md` and apply the following edits (use Edit tool with `replace_all: true` where safe, else multiple Edit calls).

Replace `chainbench-hal` with `chainbench-net` (replace_all).

Then the resulting file should look like (verify byte-for-byte):
```markdown
# chainbench-net

Network abstraction layer for chainbench. Provides a uniform command/event
interface over local, remote, and (future) ssh-remote chain nodes. Invoked as a
subprocess by the chainbench CLI and MCP server.

See `docs/VISION_AND_ROADMAP.md` §5.15–5.17 for the design.

## Prerequisites

- Go 1.25+ (required by the `go-jsonschema` code generator dependency)

## Build

    go build -o bin/chainbench-net ./cmd/chainbench-net

## Develop

    go generate ./...
    go test ./...

## Tools

Development-only tool dependencies are pinned via `tools.go` under the `tools`
build tag. Normal builds exclude that file. To validate the tool pin and its
transitive graph use:

    go build -tags tools ./...
    go list  -tags tools ./...
```

- [ ] **Step 5.2: Confirm no `hal` residual**

```bash
grep -in 'hal' network/README.md
```
Expected: no output (exit 1). The word "hal" does not appear in other contexts.

---

## Task 6: Update top-level docs (VISION_AND_ROADMAP + plan rename)

**Files:**
- Rename: `docs/superpowers/plans/2026-04-20-hal-foundation.md` → `docs/superpowers/plans/2026-04-20-network-foundation.md`
- Modify: `docs/VISION_AND_ROADMAP.md` (large edits: 63 HAL + many path references + §5.15 reframe)
- Modify: `docs/superpowers/plans/2026-04-20-network-foundation.md` (post-rename content update)

- [ ] **Step 6.1: Rename the historical plan file**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git mv docs/superpowers/plans/2026-04-20-hal-foundation.md \
       docs/superpowers/plans/2026-04-20-network-foundation.md
git status -s | grep foundation
```
Expected: `R  docs/superpowers/plans/2026-04-20-hal-foundation.md -> docs/superpowers/plans/2026-04-20-network-foundation.md`.

- [ ] **Step 6.2: Bulk replace `chainbench-hal` → `chainbench-net` and path references in VISION_AND_ROADMAP**

Apply these replacements to `docs/VISION_AND_ROADMAP.md` **in this exact order** (each Edit with `replace_all: true`). Order matters because longer, more specific patterns must be processed before shorter ones that would otherwise consume their substrings.

1. `hal/cmd/chainbench-hal` → `network/cmd/chainbench-net` (longest compound path first)
2. `mcp-server/src/hal/` → `network/` (collapse roadmap's old mention of HAL-inside-MCP placeholder — no longer applicable)
3. `hal/schema/` → `network/schema/`
4. `hal/internal/types/` → `network/internal/types/`
5. `github.com/0xmhha/chainbench/hal` → `github.com/0xmhha/chainbench/network`
6. `chainbench-hal` → `chainbench-net` (now safe — compound patterns already handled)
7. `` `hal/` `` → `` `network/` `` (backtick-wrapped bare dir name; use literal backticks in `old_string` / `new_string`)
8. `/hal/` → `/network/` (safety net for any remaining slash-delimited fragments)

- [ ] **Step 6.3: Replace conceptual term "HAL" with "Network Abstraction"**

In `docs/VISION_AND_ROADMAP.md`, these are the conceptual occurrences to change (use Edit, not replace_all, because Android HAL references in §5.15 need preservation):

Section heading changes:
- `### 5.15 HAL 아키텍처 상세 (Q4 확장)` → `### 5.15 Network Abstraction 아키텍처 상세 (Q4 확장)`
- `### 5.17 Go HAL 구현 상세 (S1~S8 확정 기반)` → `### 5.17 Go Network 모듈 구현 상세 (S1~S8 확정 기반)`
- `#### 5.17.3 Spawn-per-call 프로토콜 (S2 상세)` → unchanged (no HAL reference)

Common phrase replacements (use Edit, one at a time to preserve context):
- `HAL boundary` → `Network Abstraction boundary`
- `HAL Interface` → `Network Abstraction Interface`
- `HAL 인터페이스` → `Network Abstraction 인터페이스`
- `HAL 패키지` → `Network 모듈`
- `HAL이` → `Network Abstraction이`
- `HAL의` → `Network Abstraction의`
- `HAL 내부` → `Network Abstraction 내부`
- `HAL 호스트` → `Network Abstraction 호스트`
- `HAL 프로세스` → `Network 프로세스`
- `HAL spawn` → `네트워크 바이너리 spawn`
- `HAL 경유` → `네트워크 바이너리 경유`
- `HAL 바이너리` → `네트워크 바이너리`

- [ ] **Step 6.4: Reframe §5.15 Android HAL metaphor opening**

Locate the §5.15 subsection that currently reads (approximate — verify surrounding context):

```markdown
**메타포**: Android HAL — 상위(앱/프레임워크)는 하드웨어를 모르고, HAL 인터페이스로만 통신. 하위(벤더 드라이버)는 하드웨어별로 분리.
```

Replace with:

```markdown
**영감 (Inspiration)**: Android HAL의 "상위는 하위 구현을 모르고 command-in / event-out 인터페이스로만 통신" 패턴에서 영감을 받음. 단, 본 프로젝트는 **하드웨어가 아닌 체인 네트워크**를 추상화하므로 `Hardware Abstraction Layer` 명칭 대신 **`Network Abstraction`** 으로 명명한다.
```

- [ ] **Step 6.5: Verify VISION_AND_ROADMAP residual HAL count**

```bash
grep -cwi 'hal' docs/VISION_AND_ROADMAP.md
```
Expected: ≤ 3 (the intentional "Android HAL" reference in §5.15 and possibly `Hardware Abstraction Layer` phrase).

- [ ] **Step 6.6: Update renamed plan file content**

In `docs/superpowers/plans/2026-04-20-network-foundation.md` (now at the new filename), apply:

1. Title change: `# HAL Foundation Implementation Plan` → `# Network Foundation Implementation Plan`
2. `chainbench-hal` → `chainbench-net` (replace_all)
3. `hal/` → `network/` (replace_all, BUT first visually scan for any standalone word "hal" that should remain — none expected in the finished plan)
4. `github.com/0xmhha/chainbench/hal` → `github.com/0xmhha/chainbench/network` (replace_all)
5. `2026-04-20-hal-foundation.md` → `2026-04-20-network-foundation.md` (replace_all, so internal self-references line up)
6. Title/header HAL mentions → Network Abstraction (manual edits, similar to §5.15 approach)

- [ ] **Step 6.7: Verify renamed plan has no stray HAL path refs**

```bash
grep -cn 'chainbench-hal\|/hal/\|chainbench/hal' docs/superpowers/plans/2026-04-20-network-foundation.md
```
Expected: 0.

---

## Task 7: Update `.claude/settings.local.json`

**Files:**
- Modify: `.claude/settings.local.json`

- [ ] **Step 7.1: Patch three Bash allowlist entries**

Edit `.claude/settings.local.json`. Replace (exact match):
```json
      "Bash(/tmp/chainbench-hal version *)",
```
with:
```json
      "Bash(/tmp/chainbench-net version *)",
```

Replace:
```json
      "Bash(/tmp/chainbench-hal-final version *)",
```
with:
```json
      "Bash(/tmp/chainbench-net-final version *)",
```

Replace:
```json
      "Bash(rm /tmp/chainbench-hal-final)"
```
with:
```json
      "Bash(rm /tmp/chainbench-net-final)"
```

- [ ] **Step 7.2: Verify JSON still parses**

```bash
python3 -c "import json; json.load(open('.claude/settings.local.json'))" && echo OK
```
Expected: `OK`.

---

## Task 8: Final verification + single atomic commit

**Files:** no new edits; runs verification and creates the commit.

- [ ] **Step 8.1: Full Go verification**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build ./...
go test ./...
go vet ./...
gofmt -l .
go build -tags tools ./...
```
Expected:
- `go build`: exit 0
- `go test`: 3 packages PASS
- `go vet`: exit 0
- `gofmt -l .`: empty
- `go build -tags tools`: exit 0

- [ ] **Step 8.2: Binary smoke test**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
go build -o /tmp/chainbench-net-verify ./network/cmd/chainbench-net
/tmp/chainbench-net-verify version
rm /tmp/chainbench-net-verify
```
Expected stdout: `chainbench-net 0.0.0-dev`.

- [ ] **Step 8.3: Inventory scripts still run clean**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
scripts/inventory/list-adapter-functions.sh | head -3
scripts/inventory/scan-binary-hardcoding.sh | wc -l
```
Expected: header line + rows; scan returns non-empty count (unchanged — these tools touch `lib/`, not `hal`/`network`).

- [ ] **Step 8.4: Residual `hal` check (CRITICAL)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git grep -win 'hal' -- '*.go' '*.sh' '*.md' '*.json' '*.mod' '*.sum' ':(exclude)mcp-server/node_modules' ':(exclude)docs/superpowers/specs/2026-04-21-network-rename-design.md' ':(exclude)docs/superpowers/plans/2026-04-21-network-rename.md'
```
Expected matches (acceptable):
- `docs/VISION_AND_ROADMAP.md` §5.15: "Android HAL" intentional reference (1–3 hits)
- `docs/VISION_AND_ROADMAP.md` §5.15: "Hardware Abstraction Layer" full phrase
- `docs/superpowers/plans/2026-04-20-network-foundation.md`: none (must be 0)
- Anything else: **FAIL — investigate and fix before commit**

If unexpected matches appear, stop and edit them out. Do not proceed until only the acceptable 1–3 Android HAL refs remain.

- [ ] **Step 8.5: Residual `chainbench-hal` check (CRITICAL)**

```bash
git grep -ln 'chainbench-hal' -- ':(exclude)mcp-server/node_modules' ':(exclude)docs/superpowers/specs/2026-04-21-network-rename-design.md' ':(exclude)docs/superpowers/plans/2026-04-21-network-rename.md'
```
Expected: empty output (no files). If any file appears, edit it and re-verify.

- [ ] **Step 8.6: Residual `/hal/` path check (CRITICAL)**

```bash
git grep -ln '/hal/\|chainbench/hal' -- ':(exclude)mcp-server/node_modules' ':(exclude)docs/superpowers/specs/2026-04-21-network-rename-design.md' ':(exclude)docs/superpowers/plans/2026-04-21-network-rename.md'
```
Expected: empty output. Fix any residuals before commit.

- [ ] **Step 8.7: Review staged changes**

```bash
git status
git diff --stat --cached; git diff --stat
```
Expected: around 30+ files changed, mix of `R` (renames) and `M` (modifications). No untracked files unless intentional.

- [ ] **Step 8.8: Stage everything and create the single commit**

```bash
git add -A
git status -s | head -20
git commit -m "refactor: rename HAL module to network"
git rev-parse HEAD
git show --stat HEAD | head -40
```
Expected: one commit created; `git show --stat` shows the rename + modifications.

- [ ] **Step 8.9: Post-commit sanity**

```bash
cd network && go test ./... 2>&1 | tail -5
cd ..
git log --oneline -3
```
Expected: 3 packages PASS. Most recent commit: `refactor: rename HAL module to network`.

---

## Rollback procedure (if anything goes sideways before Step 8.8)

All edits up to Step 8.7 are in the working tree only. To abandon:
```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git restore --source=HEAD --staged --worktree .
git clean -fd network docs/superpowers/plans
```

After Step 8.8, rollback is `git revert HEAD` or `git reset --hard HEAD~1` (destructive — only with explicit user confirmation).
