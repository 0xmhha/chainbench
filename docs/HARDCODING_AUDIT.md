# Chain Binary Hardcoding Audit

> **Status:** baseline for the adapter-extraction work in `docs/VISION_AND_ROADMAP.md` §5.12 M4.
> Regenerate with:
>
>     scripts/inventory/scan-binary-hardcoding.sh

## What this is

Every site in `lib/cmd_*.sh` that names the current chain binary (`gstable`)
explicitly, rather than going through the adapter. Each of these has to be
refactored before any second chain (wbft, wemix, ethereum) can be supported.

## Findings

```
lib/cmd_init.sh:183:# ---- Run gstable init -------------------------------------------------------
lib/cmd_init.sh:190:  log_info "  node${_ni}: gstable init"
lib/cmd_init.sh:192:    log_error "gstable init failed for node${_ni}"
lib/cmd_node.sh:231:# Reconstructs the gstable launch arguments from pids.json for the given node key.
lib/cmd_node.sh:234:# strips NUL bytes. gstable arguments never contain newlines, so \n is safe.
lib/cmd_node.sh:262:binary   = node.get("binary",    "gstable")
lib/cmd_node.sh:356:  # (handles: short name like "gstable", relative path, or missing binary in pids.json).
lib/cmd_start.sh:220:  # Launch node — gstable writes directly to file, PID captured for health checks
lib/cmd_stop.sh:14:_BINARY_NAME="${CHAINBENCH_BINARY:-gstable}"
```

Total: 9 hits across 4 files (`cmd_init.sh`, `cmd_node.sh`, `cmd_start.sh`,
`cmd_stop.sh`). `docs/VISION_AND_ROADMAP.md` §3 cites "7곳"; the finer-grained
line-level scan above counts every literal `gstable` occurrence. The extra two
hits are adjacent lines inside the same log-cosmetic block (`cmd_init.sh:190,192`
belong to classification row #2; `cmd_node.sh:234` belongs to row #5). The
classification below groups them into the 7 conceptual categories.

## Classification

| # | Site | Category | Proposed replacement |
|---|------|----------|----------------------|
| 1 | `cmd_init.sh:113` — `_CHAIN_TYPE="${CHAINBENCH_CHAIN_TYPE:-stablenet}"` | Default selection | Keep default (profile override already works) |
| 2 | `cmd_init.sh:183–192` — "Run gstable init" comment + error message | Log cosmetics | Reference `${_BINARY_NAME}` from adapter |
| 3 | `cmd_start.sh:220` — launch comment | Log cosmetics | Reference adapter binary name |
| 4 | `cmd_stop.sh:14` — `_BINARY_NAME="${CHAINBENCH_BINARY:-gstable}"` | Default name for `pkill` | Fetch from active network's adapter (`adapter_binary_name`) |
| 5 | `cmd_node.sh:231–234` — doc comments | Log cosmetics | Update after adapter name wired |
| 6 | `cmd_node.sh:262` — `binary = node.get("binary","gstable")` | Runtime fallback | Read from `state/pids.json` written by adapter-aware start |
| 7 | `cmd_node.sh:356` — doc comment | Log cosmetics | Update with adapter name |

## Exit criteria for this audit

This document is considered closed when:

1. `scripts/inventory/scan-binary-hardcoding.sh` returns zero lines
2. Every replacement in the table above has been implemented in a commit
3. `chainbench init --profile <non-stablenet>` succeeds end-to-end with a non-stub adapter
