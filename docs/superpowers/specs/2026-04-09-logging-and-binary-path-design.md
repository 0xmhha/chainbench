# Design: Logging Integration & Binary Path Configuration

**Date:** 2026-04-09
**Status:** Approved, ready for implementation planning
**Scope:** `chainbench` repository (bash CLI + MCP server)

## 1. Background

Code review on `lib/cmd_start.sh`, `lib/common.sh`, and `mcp-server/src/tools/` surfaced two gaps:

1. **Log rotation is declared but not implemented.** `cmd_start.sh:200-213` declares `_logrot_bin`, `_log_max_size`, `_log_max_files` variables, then launches nodes with a plain `nohup ... >> "${node_log}" 2>&1 &`. The `bin/logrot` binary referenced in comments does not exist in the repo. Profile fields `logging.rotation`, `logging.max_size`, `logging.max_files` have no effect. Node logs grow unbounded.
2. **Binary path configuration has no clean runtime override channel.** The only ways to change `chain.binary_path` are editing a git-tracked profile YAML or calling the `chainbench_profile_set` MCP tool (which writes to that same YAML). There is no CLI flag, no env var path (the profile loader overwrites pre-exported env vars), and no machine-local overlay.

The `logrot` binary itself already exists upstream: `go-wemix/cmd/logrot/main.go` is a 3-line wrapper around `github.com/charlanxcc/logrot` with the interface `logrot <file> <size> <count>`. The sibling repo `stable-net/test/local-test/script/run_gstable.sh:92` demonstrates the intended invocation pattern. go-stablenet can build `logrot` from the same source package.

## 2. Goals

1. Restore log rotation so `profiles/*.yaml` `logging.*` settings take effect.
2. Auto-discover `logrot` next to `gstable`, in git-root `build/bin/`, or build it on demand from `cmd/logrot/main.go`.
3. Provide a clear precedence chain for binary path resolution: CLI flag → env var → local overlay → profile YAML → auto-detect.
4. Fix the `_cb_set_var` env-override bug so pre-exported `CHAINBENCH_*` variables are honored by `load_profile`.
5. Add a `chainbench config` CLI subcommand that writes to a machine-local overlay file (`state/local-config.yaml`), git-ignored.
6. Add MCP atomic init: `binary_path?` parameter on `chainbench_init` / `chainbench_start` / `chainbench_restart` / `chainbench_node_start`, plus a new `chainbench_config_set` tool that writes to the overlay.
7. Remove stale `.bak` files from a previous refactor as the first commit.

## 3. Non-Goals

- Writing `logrot` from scratch. The upstream Go package is reused.
- Replacing `chainbench_profile_set`. Team-shared profile edits continue to work through the existing MCP tool; the overlay is an additive layer.
- Adding MCP server tests or a test framework for the TypeScript layer.
- Extending the chain adapter abstraction in `lib/adapters/` or `lib/chain_adapter.sh`.
- Changing log format or `cmd_log.sh` timeline/anomaly/search behavior.
- Adding a `chainbench_config_get` MCP tool. `chainbench_profile_get({ name: "active" })` already returns the effective merged value.

## 4. Architecture

### 4.1 Configuration precedence chain

All `CHAINBENCH_*` variables (notably `BINARY_PATH` and `LOGROT_PATH`) resolve in this order:

```
1. CLI flag                --binary-path / --logrot-path         (one-shot, per-command)
2. Environment variable    CHAINBENCH_BINARY_PATH, ...           (shell session)
3. Local overlay           state/local-config.yaml               (machine-local, git-ignored)
4. Profile YAML            profiles/<name>.yaml + inherits chain (team-shared, git-tracked)
5. Auto-detection          git-root/build/bin → PWD/build/bin
                           → dirname($BINARY)/logrot
                           → go build ./cmd/logrot
                           → $PATH
6. Not found               log_warn + graceful fallback (plain `>>` append for logs)
```

The current code collapses #2 and #3 into #4 because `_cb_set_var` unconditionally `export`s, overwriting pre-exported env vars. The fix is env-first: if a `CHAINBENCH_*` variable is already set and non-empty, `_cb_set_var` returns early. Users can opt out with `CHAINBENCH_PROFILE_ENV_OVERRIDE=0`.

### 4.2 Profile merge layers

```
profiles/<name>.yaml + inherits chain
        │ load_with_inheritance()
        ▼
merged profile (in memory)
        │ deep_merge(state/local-config.yaml) if present     <-- NEW
        ▼
final merged JSON (written to CHAINBENCH_PROFILE_JSON)
        │ _cb_export_profile_vars() with env-first guard
        ▼
exported CHAINBENCH_* environment variables
```

Overlay merging happens inside the Python block of `_cb_python_merge_yaml` in `lib/profile.sh`. `CHAINBENCH_DIR` is passed as a new argument to the Python script.

### 4.3 Logrot discovery (hybrid + auto-build)

```
resolve_logrot(binary_path, explicit_logrot_path):
  1. If explicit_logrot_path set and executable     → use it
  2. If dirname(binary_path)/logrot executable      → use it
  3. If <git-root>/build/bin/logrot executable      → use it
  4. If <git-root>/cmd/logrot/main.go exists
       AND `go` command available
       → run `go build -o <git-root>/build/bin/logrot ./cmd/logrot`
         logging to state/logrot-build.log
       → on success, use the newly built binary
  5. If logrot in $PATH                             → use it (with warning)
  6. Otherwise                                       → return empty string
                                                      + log_warn
```

Step 6 is non-fatal. `cmd_start.sh` falls back to plain `>>` append when the return is empty. `logging.rotation: false` in the profile short-circuits discovery entirely.

### 4.4 Node launch pipeline

**Current (buggy):**

```bash
nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
# _logrot_bin declared but never used
```

**Replacement:**

```bash
if [[ -n "${LOGROT_BIN}" ]]; then
  nohup "${launch_cmd[@]}" \
    > >("${LOGROT_BIN}" "${node_log}" "${CHAINBENCH_LOG_MAX_SIZE}" "${CHAINBENCH_LOG_MAX_FILES}") \
    2>&1 &
  local node_pid=$!
else
  nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
  local node_pid=$!
fi
disown "${node_pid}" 2>/dev/null || true
```

Process substitution is used instead of a pipe so `$!` captures the `gstable` PID, not `logrot`. PID tracking in `pids.json` is unchanged. The `pkill -f "logrot.*node.*log"` line in `cmd_stop.sh` becomes a functioning cleanup instead of a no-op.

### 4.5 Component boundaries

```
chainbench.sh
  └─ sources lib/cmd_<subcommand>.sh

lib/common.sh
  ├─ resolve_binary()                   (existing)
  ├─ resolve_logrot()                   (NEW)
  ├─ _cb_build_logrot_from_source()     (NEW)
  └─ _cb_parse_runtime_overrides()      (NEW)

lib/profile.sh
  ├─ load_profile()                     (existing, interface unchanged)
  ├─ _cb_set_var()                      (MODIFIED, env-first)
  └─ _cb_python_merge_yaml()            (MODIFIED, overlay merge)

lib/cmd_init.sh      (MODIFIED: runtime-override parser at top)
lib/cmd_start.sh     (MODIFIED: parser + logrot + launch pipeline)
lib/cmd_restart.sh   (MODIFIED: pass overrides to init/start)
lib/cmd_node.sh      (MODIFIED: parser inside node start)
lib/cmd_stop.sh      (MODIFIED: logrot cleanup message)
lib/cmd_config.sh    (NEW, ~250-350 lines)
    └─ get / set / unset / list, reads/writes state/local-config.yaml

mcp-server/src/utils/exec.ts            (MODIFIED: shellEscapeArg)
mcp-server/src/tools/lifecycle.ts       (MODIFIED: +binary_path)
mcp-server/src/tools/node.ts            (MODIFIED: +binary_path)
mcp-server/src/tools/config.ts          (NEW: chainbench_config_set)
mcp-server/src/tools/schema.ts          (MODIFIED: doc sections)
mcp-server/src/index.ts                 (MODIFIED: register config tools)

profiles/default.yaml                   (MODIFIED: +chain.logrot_path)
.gitignore                              (MODIFIED: +overlay + build log)
README.md                               (MODIFIED: CLI ref, schema, MCP, troubleshooting)
setup.sh                                (MODIFIED: next-steps guidance)
```

## 5. Detailed Component Changes

### 5.1 `lib/common.sh`

Three new functions. Existing `resolve_binary`, `get_node_port`, `is_truthy`, `require_cmd`, color/logging helpers are untouched.

**`resolve_logrot(binary_path, explicit_logrot_path)`** — implements §4.3. Prints an absolute path to stdout or an empty string. Operational log messages go to stderr via `log_info` / `log_warn`.

**`_cb_build_logrot_from_source(git_root)`** — internal helper called by `resolve_logrot` step 4. Checks for `<git_root>/cmd/logrot/main.go`, verifies `go` is available, runs `go build -o build/bin/logrot ./cmd/logrot` from `git_root`, redirects build output to `state/logrot-build.log`. Returns the built path on success, empty on failure.

**`_cb_parse_runtime_overrides(remaining_array_ref, ...args)`** — shared CLI flag parser invoked from each command that accepts overrides. Uses bash nameref. Consumes `--binary-path`, `--binary-path=`, `--logrot-path`, `--logrot-path=`; exports the corresponding `CHAINBENCH_*` env vars; appends unknown flags to the `remaining` array for the command to handle.

Empty values and non-absolute paths fail fast with `log_error` and exit 1.

### 5.2 `lib/profile.sh`

**`_cb_set_var` env-first guard:**

```bash
_cb_set_var() {
  local var_name="$1"
  local field="$2"
  local default="${3:-}"

  if [[ "${CHAINBENCH_PROFILE_ENV_OVERRIDE:-1}" == "1" ]] \
      && [[ -n "${!var_name+x}" ]] \
      && [[ -n "${!var_name}" ]]; then
    return 0
  fi

  local value
  value="$(_cb_jq_get "$json_file" "$field" "$default")"
  export "${var_name}=${value}"
}
```

The `-n "${!var_name+x}"` test checks "is set", `-n "${!var_name}"` checks "not empty". A user who wants to reset an override can `export CHAINBENCH_BINARY_PATH=""`, which falls through to the profile value.

**Overlay merging in `_cb_python_merge_yaml`:**

```python
merged = load_with_inheritance(PROFILE_PATH, PROFILES_ROOT)

overlay_path = os.path.join(CHAINBENCH_DIR, "state", "local-config.yaml")
if os.path.isfile(overlay_path):
    overlay = load_yaml_file(overlay_path)
    if overlay:
        # Drop 'inherits' from overlays with a warning; overlays are pure override layers.
        if 'inherits' in overlay:
            print("WARN: local-config.yaml: 'inherits' field ignored", file=sys.stderr)
            overlay = {k: v for k, v in overlay.items() if k != 'inherits'}
        merged = deep_merge(merged, overlay)

print(json.dumps(merged, ensure_ascii=False))
```

`CHAINBENCH_DIR` is passed as a third argument to the Python block (after `profile_path` and `_CB_PROFILES_DIR`).

**New variable export:**

```bash
_cb_set_var CHAINBENCH_LOGROT_PATH ".chain.logrot_path" ""
```

Added to the `chain.*` block inside `_cb_export_profile_vars`.

### 5.3 `lib/cmd_init.sh`, `lib/cmd_start.sh`, `lib/cmd_restart.sh`, `lib/cmd_node.sh`

Each gets a parser block at the top (immediately after sourcing dependencies, before `load_profile`):

```bash
_CB_INIT_REMAINING=()
_cb_parse_runtime_overrides _CB_INIT_REMAINING "$@"
set -- "${_CB_INIT_REMAINING[@]}"
```

Name the array per-command (`_CB_INIT_REMAINING`, `_CB_START_REMAINING`, etc.) to avoid cross-contamination when files are sourced.

For `cmd_node.sh`, the parser runs **inside** `_cb_node_cmd_start` after `$1` (the node number) has been consumed, so the remaining arguments are override flags only.

`cmd_restart.sh` calls `cmd_init.sh` and `cmd_start.sh` internally. The env vars exported by the top-level parser propagate naturally through the sourced commands. No per-stage re-parsing is needed.

### 5.4 `lib/cmd_start.sh` logrot integration

After `resolve_binary` returns, before the launch loop:

```bash
LOGROT_BIN=""
if is_truthy "${CHAINBENCH_LOG_ROTATION:-true}"; then
  LOGROT_BIN="$(resolve_logrot "${BINARY}" "${CHAINBENCH_LOGROT_PATH:-}")" || LOGROT_BIN=""
fi

if [[ -z "${LOGROT_BIN}" ]] && is_truthy "${CHAINBENCH_LOG_ROTATION:-true}"; then
  log_warn "logrot not available — logs will grow unbounded. Set chain.logrot_path or build <git-root>/cmd/logrot."
fi
```

Inside `_start_launch_node`, delete the dead declarations at lines 200-213 (`_logrot_bin`, `_log_max_size`, `_log_max_files`) and replace the `nohup` invocation with the conditional block from §4.4.

### 5.5 `lib/cmd_config.sh` (new)

Four subcommands:

- `chainbench config list` — prints the full content of `state/local-config.yaml`, or `(empty)` if the file is missing.
- `chainbench config get <field>` — dot-notation lookup. Searches the overlay first. Exits 1 with `(not found)` if the field is absent.
- `chainbench config set <field> <value>` — `value` is parsed as JSON first; on parse failure it is treated as a string. The overlay is loaded, modified with a nested-set helper, and written back atomically (temp file + rename).
- `chainbench config unset <field>` — removes a field. Empty parent dicts cascade-clean.

Field validation: `^[a-zA-Z0-9_][a-zA-Z0-9_.]*$` (no `..`, no leading dot, no path traversal). Python `yaml.safe_load` blocks tag injection.

Implementation style matches `lib/profile.sh`'s existing inline Python pattern. YAML serialization follows the `jsonToYaml` helper in `mcp-server/src/tools/schema.ts:181`.

### 5.6 `mcp-server/src/utils/exec.ts`

New helper:

```typescript
export function shellEscapeArg(arg: string): string {
  // POSIX single-quote escape, embedded quotes via '\''
  return `'${arg.replace(/'/g, "'\\''")}'`;
}
```

`runChainbench` continues to take a single command string. All callers that include user-controlled values in `args` must route them through `shellEscapeArg`.

### 5.7 `mcp-server/src/tools/lifecycle.ts` and `node.ts`

Add `binary_path?: string` to `chainbench_init`, `chainbench_start`, `chainbench_restart`, `chainbench_node_start`. Validation helper:

```typescript
function validateBinaryPath(binary_path: string | undefined): string | null {
  if (binary_path === undefined) return null;
  if (!binary_path.startsWith("/")) return "binary_path must be an absolute path.";
  if (binary_path.length === 0)     return "binary_path must not be empty.";
  return null;
}
```

When `binary_path` is present, append `--binary-path ${shellEscapeArg(binary_path)}` to the `runChainbench` args string.

### 5.8 `mcp-server/src/tools/config.ts` (new)

Single tool:

```typescript
server.tool(
  "chainbench_config_set",
  "Write a field to the machine-local overlay (state/local-config.yaml). " +
  "Persistent but git-ignored. Use for machine-specific paths like chain.binary_path. " +
  "Different from chainbench_profile_set which edits the git-tracked profile YAML.",
  {
    field: z.string().describe("Dot-notation path (e.g., 'chain.binary_path')"),
    value: z.string().describe("Value. JSON-parsed if valid JSON, else string."),
  },
  async ({ field, value }) => {
    if (!/^[a-zA-Z0-9_][a-zA-Z0-9_.]*$/.test(field)) {
      return errorResponse("Invalid field path");
    }
    const result = runChainbench(
      `config set ${shellEscapeArg(field)} ${shellEscapeArg(value)}`
    );
    return { content: [{ type: "text" as const, text: formatResult(result) }] };
  }
);
```

Registered via `registerConfigTools(server)` in `mcp-server/src/index.ts`.

### 5.9 Profile & documentation updates

- `profiles/default.yaml` — add `logrot_path: ""` to the `chain:` block with a Korean comment matching the existing style.
- `.gitignore` — add `state/local-config.yaml` and `state/logrot-build.log`.
- `README.md` — add a `chainbench config` section to the CLI reference, document `chain.logrot_path` in the profile schema, add `chainbench_config_set` to the MCP tools table, add a logrot troubleshooting entry, update the Quick Start note on line 73 to recommend `chainbench config set chain.binary_path`.
- `setup.sh` — change the "next steps" block to recommend `chainbench config set` over editing `profiles/default.yaml`, and add a line suggesting `bash tests/unit/run.sh`.
- `mcp-server/src/tools/schema.ts` — update `SECTION_DOCS.chain` to mention `chain.logrot_path`, and update `SECTION_DOCS.logging` to reflect that logrot integration is now live.

## 6. Data Flow Walkthroughs

### 6.1 One-shot CLI override

```
$ chainbench init --profile default --binary-path /opt/gstable-rc1/build/bin/gstable

chainbench.sh       : CHAINBENCH_PROFILE=default; subcommand=init
cmd_init.sh         : _cb_parse_runtime_overrides
                      → export CHAINBENCH_BINARY_PATH=/opt/gstable-rc1/build/bin/gstable
load_profile        : _cb_python_merge_yaml → merged JSON
                      → _cb_set_var sees CHAINBENCH_BINARY_PATH already set → preserves /opt/...
resolve_binary      : returns /opt/gstable-rc1/build/bin/gstable
cmd_init.sh         : persists to state/current-profile-merged.json
                      (existing lines 22-29 behavior)
```

### 6.2 Persistent machine-local config

```
$ chainbench config set chain.binary_path /opt/gstable/build/bin/gstable
$ chainbench config set chain.logrot_path /opt/gstable/build/bin/logrot
$ chainbench init

cmd_config.sh set   : creates state/local-config.yaml with nested chain block
load_profile        : Python block deep-merges state/local-config.yaml on top of profile
                      → merged chain.binary_path = /opt/gstable/build/bin/gstable
_cb_set_var         : no env var set → exports merged value
resolve_binary      : /opt/gstable/build/bin/gstable
```

### 6.3 Logrot auto-build

Preconditions: `/opt/gstable/build/bin/gstable` exists, `/opt/gstable/build/bin/logrot` does not, `/opt/gstable/cmd/logrot/main.go` exists, `go` is on `$PATH`.

```
resolve_logrot      :
  step 1 (profile chain.logrot_path)       → empty, skip
  step 2 (dirname(BINARY)/logrot)          → not found, skip
  step 3 (git-root/build/bin/logrot)       → not found, skip
  step 4 (git-root/cmd/logrot/main.go)     → found
    _cb_build_logrot_from_source
      → cd /opt/gstable && go build -o build/bin/logrot ./cmd/logrot
      → stdout+stderr → state/logrot-build.log
      → success, returns /opt/gstable/build/bin/logrot
  log_info "Built logrot from source: /opt/gstable/build/bin/logrot"
_start_launch_node  :
  nohup gstable ... > >(logrot node1.log 10M 5) 2>&1 &
  node_pid=$!   # gstable PID, not logrot
```

### 6.4 MCP atomic init

```
User: "Initialize the chain with the gstable binary at /opt/gstable-rc1/build/bin/gstable"

LLM → chainbench_init({
  profile: "default",
  binary_path: "/opt/gstable-rc1/build/bin/gstable"
})

lifecycle.ts        :
  validateBinaryPath("/opt/...")  → ok
  args = `init --profile default --quiet --binary-path '/opt/gstable-rc1/build/bin/gstable'`
  runChainbench(args, { cwd: project_root })
```

The remaining flow is identical to §6.1.

### 6.5 Fallback when logrot is entirely unavailable

Preconditions: no logrot anywhere, no source, no `go` command.

```
resolve_logrot      : steps 1-5 all miss → log_warn + return ""
cmd_start.sh        : LOGROT_BIN="" → log_warn "logrot not available ..."
_start_launch_node  :
  nohup gstable ... >> node1.log 2>&1 &      # plain append, unchanged behavior
node_pid            : $! as before
```

Chain starts normally; the user sees a single warning line during startup.

### 6.6 Precedence resolution table

| CLI flag | env var | overlay | profile | Final |
|----------|---------|---------|---------|-------|
| `/a`     | -       | -       | `/d`    | `/a`  |
| -        | `/b`    | -       | `/d`    | `/b`  |
| -        | -       | `/c`    | `/d`    | `/c`  |
| -        | -       | -       | `/d`    | `/d`  |
| `/a`     | `/b`    | `/c`    | `/d`    | `/a`  |
| -        | `/b`    | `/c`    | -       | `/b`  |
| `/a`     | -       | -       | -       | `/a`  |
| -        | -       | -       | `""`    | auto  |

This table maps directly to unit test cases in `tests/unit/tests/common-resolve-binary.sh` and `profile-env-override.sh`.

## 7. Error Handling

### 7.1 Runtime override parsing

| Failure | Action | Message |
|---|---|---|
| Flag missing value (`--binary-path` at end of args) | exit 1 | `"--binary-path requires a value (absolute path)"` |
| Non-absolute value | exit 1 | `"--binary-path must be an absolute path (got: 'gstable')"` |
| Value path does not exist | No immediate error; caught at `resolve_binary` | `resolve_binary` warns and falls back |
| Unknown flag | Passed through to the command | (command handles) |

### 7.2 Overlay merging

| Failure | Action | Message |
|---|---|---|
| File missing | no-op | none |
| Empty file | no-op | none |
| Malformed YAML | `load_profile` fails, exit 1 | `"ERROR: local-config.yaml parse failed: <detail>"` |
| `inherits` field in overlay | Stripped and warned | `"local-config.yaml: 'inherits' field ignored in overlays"` |
| Type mismatch (e.g. `validators: "four"`) | Propagates to `_cb_validate_profile_json` | Existing validation path |

### 7.3 `_cb_set_var` env-first edge cases

| State | Current | After fix |
|---|---|---|
| env empty, profile has value | profile value | **unchanged** |
| env set (non-empty), profile has value | profile overrides env (bug) | **env preserved** |
| env set but empty string | profile overrides | profile overrides (explicit reset UX) |
| `CHAINBENCH_PROFILE_ENV_OVERRIDE=0`, env set, profile has value | profile overrides | profile overrides (opt-out) |

### 7.4 Logrot discovery

| Step | Failure mode | Action |
|---|---|---|
| 1 profile path | not executable | warn, proceed |
| 2-3 neighbor/git-root | missing | silent, proceed |
| 4 auto-build | `cmd/logrot/main.go` absent | silent, proceed |
| 4 auto-build | `go` unavailable | `log_info`, proceed |
| 4 auto-build | `go build` fails | `log_warn "logrot build failed, see state/logrot-build.log"`, proceed |
| 5 `$PATH` | miss | silent, proceed |
| Final | all miss | single `log_warn`, return empty, `cmd_start.sh` falls back to plain `>>` |

`logging.rotation: false` short-circuits the entire discovery chain. A chain start with rotation disabled never invokes `resolve_logrot`.

### 7.5 `cmd_config.sh`

| Failure | Action |
|---|---|
| Missing subcommand args | Usage + exit 1 |
| Invalid field path (empty, double dot, leading dot) | `log_error` + exit 1 |
| YAML-unsafe values | `yaml.safe_load` rejects dangerous tags |
| `state/` not writable | `log_error` + exit 1 |
| `unset` of non-existent field | `log_warn` + exit 0 |
| `get` of non-existent field | `(not found)` + exit 1 |
| Interrupted write | temp file + atomic `rename` preserves the original |

### 7.6 MCP

| Failure | Action |
|---|---|
| `binary_path` not absolute | Error response, no CLI invocation |
| Shell metacharacters in any forwarded value | `shellEscapeArg` escapes before concatenation |
| Invalid `chainbench_config_set` field | Rejected by client-side regex before calling CLI |

### 7.7 Process substitution

| Scenario | Behavior |
|---|---|
| `logrot` crashes immediately | gstable may receive SIGPIPE. Mitigation strategy deferred to implementation (consider `trap '' PIPE` or a lightweight supervisor). |
| gstable exits normally | logrot receives EOF and exits. No zombies. |
| Stale logrot from an earlier run | `cmd_stop.sh:55` `pkill -f "logrot.*node.*log"` provides the safety net. |

## 8. Testing

### 8.1 Directory layout

```
tests/
├── regression/              (existing, untouched)
└── unit/                    (new)
    ├── run.sh
    ├── lib/assert.sh
    ├── fixtures/
    │   ├── mock-gstable
    │   ├── mock-go/go
    │   └── profiles/
    └── tests/
        ├── smoke-meta.sh
        ├── common-resolve-binary.sh
        ├── common-resolve-logrot.sh
        ├── common-parse-overrides.sh
        ├── profile-env-override.sh
        ├── profile-overlay-merge.sh
        ├── cmd-config.sh
        └── smoke-logrot-integration.sh
```

`run.sh` iterates `tests/*.sh`, executes each in a subshell, and reports pass/fail totals. Exit code is non-zero if any test failed.

### 8.2 Assertion helper

`tests/unit/lib/assert.sh` exports `assert_eq`, `assert_neq`, `assert_empty`, `assert_nonempty`, `assert_file_exists`, `assert_contains`, `assert_exit_code`. Zero external dependencies. Failure prints a short reason and exits 1.

### 8.3 Case coverage per file

**`common-resolve-binary.sh`** (regression guard for existing behavior):
1. explicit path valid and executable
2. explicit path not executable → auto-detect fallback
3. git-root/build/bin/$bin
4. PWD/build/bin/$bin
5. $PATH hit with warning
6. complete miss → exit 1

**`common-resolve-logrot.sh`**:
1. explicit `chain.logrot_path`
2. `dirname($BINARY)/logrot`
3. `git-root/build/bin/logrot`
4. build from source (mock `go` succeeds)
5. source present but `go` unavailable → next step
6. build fails → warn, next step
7. `$PATH` hit
8. all miss → empty + warn

The mock `go` binary in `fixtures/mock-go/go` inspects its args; when it sees `build -o <out> ./cmd/logrot` it writes a trivial executable to `<out>`. Tests prepend `fixtures/mock-go` to `PATH` before calling `resolve_logrot`.

**`common-parse-overrides.sh`**:
1. `--binary-path /x init` → env + remaining
2. `--binary-path=/x init` → same
3. `init --binary-path /x` → same
4. `--logrot-path /y --binary-path /x` → both exported
5. Unknown flag passes through
6. Missing value → exit 2

**`profile-env-override.sh`**:
1. env set, profile has value → env wins
2. env empty string, profile has value → profile wins (reset UX)
3. env unset → profile wins
4. `CHAINBENCH_PROFILE_ENV_OVERRIDE=0` + env set → profile wins (opt-out)

**`profile-overlay-merge.sh`**:
1. no overlay file → profile only
2. overlay overrides a leaf field
3. overlay adds a new field
4. malformed overlay → `load_profile` exit 1
5. overlay with `inherits` → ignored + warn
6. deep merge preserves sibling fields

**`cmd-config.sh`**:
1. `set` creates file
2. `get` returns the value
3. Second `set` preserves earlier fields
4. `unset` removes the targeted field
5. Cascade-clean empty parent dicts
6. `list` prints the whole overlay
7. Invalid field (`..`) → exit 1
8. `get` of missing field → exit 1
9. JSON value `[1,2]` → stored as array
10. Atomic write (temp + rename verified indirectly via file existence check)

**`smoke-logrot-integration.sh`** (mock-based):
- Instead of running `gstable`, use a shell loop (`while echo "block $i"; do ((i++)); sleep 0.01; done`) as the data source.
- Pipe through `resolve_logrot`'s result with `CHAINBENCH_LOG_MAX_SIZE=1K`.
- Assert that `node1.log.1` appears within 3 seconds.

### 8.4 Runtime targets

- `bash tests/unit/run.sh` < 10 seconds total
- Individual unit tests < 2 seconds each
- `smoke-logrot-integration.sh` < 5 seconds

### 8.5 Regression tests

`tests/regression/` scripts under active modification by the user. This spec does not add to or modify them. Manual smoke verification: `chainbench init && chainbench start && chainbench status` after each phase to confirm nothing broke.

### 8.6 CI

No GitHub Actions workflow exists. Adding one is out of scope. `setup.sh` will print an instruction to run `bash tests/unit/run.sh` manually.

## 9. Implementation Phases

Nine commits, each independently verifiable.

| Phase | Commit message | Scope |
|---|---|---|
| 0 | `chore: remove stale .bak files from previous refactor` | Delete 3 `.bak` files |
| 1 | `test: add unit test runner scaffolding` | `tests/unit/run.sh`, `lib/assert.sh`, meta smoke test |
| 2 | `test(common): pin down resolve_binary behavior with unit tests` | `common-resolve-binary.sh` + mock-gstable fixture |
| 3 | `feat(cli): add --binary-path / --logrot-path runtime overrides` | `_cb_parse_runtime_overrides` + call sites in init/start/restart/node + parser tests |
| 4 | `fix(profile): respect env vars and merge local overlay during profile load` | `_cb_set_var` env-first, overlay merge in Python block, `.gitignore`, two test files |
| 5 | `feat(cli): add 'chainbench config' subcommand for local overlay` | `lib/cmd_config.sh` + tests |
| 6 | `feat(logging): integrate logrot with hybrid discovery and auto-build` | `resolve_logrot`, `_cb_build_logrot_from_source`, `cmd_start.sh` launch pipeline, `profiles/default.yaml`, `cmd_stop.sh` message, mock-go fixture, logrot tests |
| 7 | `feat(mcp): add binary_path override to lifecycle tools and config_set tool` | `shellEscapeArg`, `lifecycle.ts`, `node.ts`, `config.ts`, `schema.ts`, `index.ts` |
| 8 | `docs: document logrot integration and chainbench config command` | `README.md`, `setup.sh`, remaining comments |

Phase dependencies are strictly linear. Emergency rollback points:

- **After Phase 3:** runtime override channel works end-to-end (no overlay, no logrot), users can point at arbitrary binaries via CLI.
- **After Phase 6:** log rotation is fully restored. Main value delivered. MCP and docs still pending but the CLI is complete.

## 10. Open Questions (deferred to implementation)

- **SIGPIPE handling:** should `cmd_start.sh` set `trap '' PIPE` before launching, or wrap the process-substitution subshell? To be decided when wiring up Phase 6.
- **Rotated file glob in `cmd_log.sh`:** once `node1.log.1`, `node1.log.2`, etc. can exist, should `log search` include them? Current scope says no (non-goal), but a one-liner extension in `_cb_log_all_log_files` may be worthwhile. Decide during Phase 6 review.
- **`cmd_restart.sh` override propagation:** current `cmd_restart.sh` structure should be inspected during Phase 3. If it re-execs child commands via `bash chainbench.sh`, env vars propagate automatically. If it sources them, propagation already works. Verify rather than guess.

These are implementation-time decisions, not design decisions, and do not block spec approval.
