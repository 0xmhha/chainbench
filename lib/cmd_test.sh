#!/usr/bin/env bash
# lib/cmd_test.sh - Test runner for chainbench
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Sub-subcommands:
#   test list
#   test run <target>   (target: "all" | category e.g. "basic" | specific e.g. "basic/consensus")

# Guard against double-sourcing
[[ -n "${_CB_CMD_TEST_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_TEST_SH_LOADED=1

_CB_TEST_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_TEST_LIB_DIR}/common.sh"
source "${_CB_TEST_LIB_DIR}/test_meta.sh" 2>/dev/null || true
source "${_CB_TEST_LIB_DIR}/test_scaffold.sh" 2>/dev/null || true
source "${_CB_TEST_LIB_DIR}/network_client.sh" 2>/dev/null || true

# ---- Constants ---------------------------------------------------------------

readonly _CB_TEST_TESTS_DIR="${CHAINBENCH_DIR}/tests"
# Dynamic category discovery — auto-detects test directories
_cb_test_discover_categories() {
  local test_dir="${_CB_TEST_TESTS_DIR}"
  local dir
  for dir in "${test_dir}"/*/; do
    [[ ! -d "$dir" ]] && continue
    local name
    name=$(basename "$dir")
    # Exclude library and unit test directories
    [[ "$name" == "lib" || "$name" == "unit" ]] && continue
    # Include directories containing .sh files (direct or in subdirectories)
    if find "$dir" -name '*.sh' -type f 2>/dev/null | head -1 | grep -q .; then
      printf '%s\n' "$name"
    fi
  done | sort
}

# Build categories array dynamically
mapfile -t _CB_TEST_CATEGORIES < <(_cb_test_discover_categories)

# Color codes
_CB_TEST_GREEN='\033[0;32m'
_CB_TEST_RED='\033[0;31m'
_CB_TEST_CYAN='\033[0;36m'
_CB_TEST_YELLOW='\033[0;33m'
_CB_TEST_BOLD='\033[1m'
_CB_TEST_RESET='\033[0m'

# ---- Usage -------------------------------------------------------------------

_cb_test_usage() {
  cat >&2 <<'EOF'
Usage: chainbench test <subcommand> [target] [--remote <alias>]

Subcommands:
  list              List all available tests with descriptions
  run <target>      Run tests matching target

Target formats:
  all               Run all tests (or only remote/ when --remote is set)
  basic             Run all tests in the basic/ category
  fault             Run all tests in the fault/ category
  stress            Run all tests in the stress/ category
  upgrade           Run all tests in the upgrade/ category
  remote            Run all tests in the remote/ category
  basic/consensus   Run a single specific test

Options:
  --remote <alias>  Run tests against a remote chain (exports CHAINBENCH_REMOTE)

Examples:
  chainbench test list
  chainbench test run all
  chainbench test run basic
  chainbench test run basic/consensus
  chainbench test run remote --remote eth-mainnet
  chainbench test run remote/rpc-health --remote my-testnet
EOF
}

# ---- Capability gating -------------------------------------------------------

# _cb_test_active_capabilities
# Calls chainbench-net network.capabilities and prints the cap list
# (space-separated). Returns 1 if chainbench-net is unreachable.
_cb_test_active_capabilities() {
  if ! type -t cb_net_call >/dev/null 2>&1; then
    return 1
  fi
  local data
  if ! data=$(cb_net_call "network.capabilities" '{}' 2>/dev/null); then
    return 1
  fi
  echo "$data" | jq -r '.capabilities[]?' 2>/dev/null | tr '\n' ' ' | sed 's/ $//'
}

# _cb_test_check_capabilities <test_path>
# Returns 0 if the test can run (no frontmatter, all required caps satisfied,
# or capability resolution is unreachable — best-effort allow).
# Returns 1 with a SKIP diagnostic on stderr when caps are missing.
_cb_test_check_capabilities() {
  local test_path="${1:?_cb_test_check_capabilities: test_path required}"

  # Parse frontmatter; absent or empty → no requirements → run.
  local meta_json required
  meta_json=$(cb_parse_meta "$test_path" 2>/dev/null || echo "{}")
  required=$(echo "$meta_json" | jq -r '.requires_capabilities[]?' 2>/dev/null | tr '\n' ' ')
  required="${required% }"
  [[ -z "$required" ]] && return 0

  # Resolve active network's caps; on failure WARN + permissive allow.
  local active_caps
  if ! active_caps=$(_cb_test_active_capabilities); then
    printf "${_CB_TEST_YELLOW}[WARN]${_CB_TEST_RESET}  cannot resolve network capabilities; running '%s' without gating\n" \
      "$test_path" >&2
    return 0
  fi

  # Compute missing set; bash 3.2 safe (no associative array).
  local missing="" r
  for r in $required; do
    if ! printf ' %s ' "$active_caps" | grep -q " $r "; then
      missing="${missing:+$missing }$r"
    fi
  done

  if [[ -n "$missing" ]]; then
    printf "${_CB_TEST_YELLOW}[SKIP]${_CB_TEST_RESET}  %s — requires capability: %s; active network provides: %s\n" \
      "$test_path" "$missing" "$active_caps" >&2
    return 1
  fi
  return 0
}

# ---- Test discovery ----------------------------------------------------------

# _cb_test_find_scripts <category>
# Prints absolute paths of all .sh test scripts in tests/<category>/.
# Sorted by name.
_cb_test_find_scripts() {
  local category="${1:?_cb_test_find_scripts: category required}"
  local dir="${_CB_TEST_TESTS_DIR}/${category}"

  if [[ ! -d "$dir" ]]; then
    return 0
  fi

  # Discover .sh files recursively, exclude lib/ directories and helper scripts
  find "$dir" -name '*.sh' -type f \
    ! -path '*/lib/*' \
    ! -name 'run-all.sh' \
    | sort
}

# _cb_test_extract_description <script_path>
# Reads the first line matching "# Description: ..." from the script header.
# Prints the description text (after the prefix), or "(no description)" if absent.
_cb_test_extract_description() {
  local script="${1:?_cb_test_extract_description: script required}"
  local desc=""

  while IFS= read -r line; do
    case "$line" in
      "# Description: "*)
        desc="${line#\# Description: }"
        break
        ;;
      "# Description:"*)
        desc="${line#\# Description:}"
        desc="${desc# }"
        break
        ;;
    esac
  done < "$script"

  printf '%s\n' "${desc:-(no description)}"
}

# _cb_test_script_to_name <script_path>
# Converts an absolute script path to a "category/name" identifier.
# e.g. .../tests/basic/consensus.sh -> basic/consensus
_cb_test_script_to_name() {
  local script="${1:?_cb_test_script_to_name: script required}"
  local rel="${script#"${_CB_TEST_TESTS_DIR}/"}"
  printf '%s\n' "${rel%.sh}"
}

# ---- test list ---------------------------------------------------------------

_cb_test_cmd_list() {
  local format="text"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --format) format="${2:-text}"; shift 2 ;;
      --format=*) format="${1#--format=}"; shift ;;
      *) shift ;;
    esac
  done

  if [[ "$format" == "json" ]]; then
    _cb_test_cmd_list_json
    return $?
  fi

  local found=0

  printf '\n'
  printf "${_CB_TEST_BOLD}%-25s  %s${_CB_TEST_RESET}\n" "Test" "Description"
  printf '%-25s  %s\n' "-------------------------" "--------------------------------------------"

  local cat script name desc
  for cat in "${_CB_TEST_CATEGORIES[@]}"; do
    local dir="${_CB_TEST_TESTS_DIR}/${cat}"
    [[ -d "$dir" ]] || continue

    while IFS= read -r script; do
      [[ -z "$script" ]] && continue
      name=$(_cb_test_script_to_name "$script")
      desc=$(_cb_test_extract_description "$script")
      printf '%-25s  %s\n' "$name" "$desc"
      found=$(( found + 1 ))
    done < <(_cb_test_find_scripts "$cat")
  done

  printf '\n'

  if [[ "$found" -eq 0 ]]; then
    printf "${_CB_TEST_YELLOW}[TEST]${_CB_TEST_RESET}  No tests found in %s\n" \
      "${_CB_TEST_TESTS_DIR}" >&2
  else
    printf "${_CB_TEST_CYAN}[TEST]${_CB_TEST_RESET}  %d test(s) available\n" "$found" >&2
  fi

  return 0
}

# _cb_test_cmd_list_json
# Outputs test list as JSON with metadata from frontmatter.
_cb_test_cmd_list_json() {
  local -a all_scripts=()
  local cat script
  for cat in "${_CB_TEST_CATEGORIES[@]}"; do
    while IFS= read -r script; do
      [[ -n "$script" ]] && all_scripts+=("$script")
    done < <(_cb_test_find_scripts "$cat")
  done

  python3 -c "
import json, sys, subprocess, os

chainbench_dir = os.environ.get('CHAINBENCH_DIR', '')
test_meta_sh = os.path.join(chainbench_dir, 'lib', 'test_meta.sh')
tests_dir = os.path.join(chainbench_dir, 'tests')

scripts = sys.argv[1:]
result = {'tests': [], 'total': len(scripts)}

for script in scripts:
    # Derive name from path
    rel = script
    if tests_dir and script.startswith(tests_dir):
        rel = script[len(tests_dir):].lstrip('/')
    name = rel.rsplit('.sh', 1)[0] if rel.endswith('.sh') else rel

    # Parse metadata
    meta = {}
    try:
        r = subprocess.run(
            ['bash', '-c', f'source \"{test_meta_sh}\" 2>/dev/null; cb_parse_meta \"{script}\"'],
            capture_output=True, text=True, timeout=5,
            env={**os.environ}
        )
        if r.stdout.strip():
            meta = json.loads(r.stdout.strip())
    except Exception:
        pass

    # Extract description from header if no meta name
    desc = meta.get('name', '')
    if not desc:
        try:
            with open(script) as f:
                for line in f:
                    if line.startswith('# Description:'):
                        desc = line.split(':', 1)[1].strip()
                        break
        except Exception:
            pass

    entry = {'name': name, 'script': os.path.basename(script)}
    if meta:
        entry['meta'] = meta
    if desc and 'meta' not in entry:
        entry['description'] = desc

    result['tests'].append(entry)

print(json.dumps(result, indent=2, ensure_ascii=False))
" "${all_scripts[@]}"
}

# ---- test run helpers --------------------------------------------------------

# _cb_test_run_single <script_path>
# Executes one test script in a subshell. Returns its exit code.
# Prints per-test pass/fail banner.
_cb_test_run_single() {
  local script="${1:?_cb_test_run_single: script required}"
  local name
  name=$(_cb_test_script_to_name "$script")

  if [[ ! -f "$script" ]]; then
    printf "${_CB_TEST_RED}[SKIP]${_CB_TEST_RESET}  %s — script not found: %s\n" \
      "$name" "$script" >&2
    return 1
  fi

  if [[ ! -x "$script" ]]; then
    printf "${_CB_TEST_YELLOW}[WARN]${_CB_TEST_RESET}  %s — not executable, attempting to run with bash\n" \
      "$name" >&2
    bash "$script"
    return $?
  fi

  "$script"
  return $?
}

# _cb_test_collect_scripts <target>
# Prints the absolute paths of all scripts that match the given target.
# Target: "all" | "basic" | "basic/consensus" | etc.
_cb_test_collect_scripts() {
  local target="${1:?_cb_test_collect_scripts: target required}"

  if [[ "$target" == "all" ]]; then
    local cat
    for cat in "${_CB_TEST_CATEGORIES[@]}"; do
      _cb_test_find_scripts "$cat"
    done
    return 0
  fi

  # Check if target is a known category
  local cat
  for cat in "${_CB_TEST_CATEGORIES[@]}"; do
    if [[ "$target" == "$cat" ]]; then
      _cb_test_find_scripts "$cat"
      return 0
    fi
  done

  # Treat as a specific test path: "category/name"
  local script="${_CB_TEST_TESTS_DIR}/${target}.sh"
  if [[ -f "$script" ]]; then
    printf '%s\n' "$script"
    return 0
  fi

  # Not found
  return 1
}

# ---- dry-run -----------------------------------------------------------------

# _cb_test_dry_run <target> <scripts...>
# Output execution plan as JSON or text without running tests.
_cb_test_dry_run() {
  local target="$1"
  shift
  local -a scripts=("$@")
  local format="${CB_FORMAT:-text}"

  if [[ "$format" == "json" ]]; then
    python3 -c "
import json, sys, subprocess, os

target = sys.argv[1]
scripts = sys.argv[2:]
test_meta_sh = os.environ.get('CHAINBENCH_DIR', '') + '/lib/test_meta.sh'

result = {'target': target, 'scripts': [], 'total_scripts': len(scripts), 'total_estimated_seconds': 0}

for script in scripts:
    name = os.path.basename(script)
    meta = {}
    # Parse meta using cb_parse_meta
    try:
        r = subprocess.run(
            ['bash', '-c', f'source \"{test_meta_sh}\" 2>/dev/null; cb_parse_meta \"{script}\"'],
            capture_output=True, text=True, timeout=5,
            env={**os.environ}
        )
        if r.stdout.strip():
            meta = json.loads(r.stdout.strip())
    except Exception:
        pass

    est = meta.get('estimated_seconds', 0)
    if isinstance(est, (int, float)):
        result['total_estimated_seconds'] += int(est)

    result['scripts'].append({'script': name, 'meta': meta})

print(json.dumps(result, indent=2))
" "$target" "${scripts[@]}"
  else
    printf "Dry-run plan for target: %s\n" "$target" >&2
    printf "Scripts to execute: %d\n\n" "${#scripts[@]}" >&2
    local script
    for script in "${scripts[@]}"; do
      local name meta_json
      name=$(basename "$script")
      meta_json=""
      if type -t cb_parse_meta &>/dev/null; then
        meta_json="$(cb_parse_meta "$script" 2>/dev/null)"
      fi
      printf "  %s" "$name" >&2
      if [[ -n "$meta_json" && "$meta_json" != "{}" ]]; then
        local id
        id="$(echo "$meta_json" | python3 -c "import sys,json; print(json.load(sys.stdin).get('id',''))" 2>/dev/null)"
        [[ -n "$id" ]] && printf " [%s]" "$id" >&2
      fi
      printf "\n" >&2
    done
  fi
}

# ---- test run ----------------------------------------------------------------

_cb_test_cmd_run() {
  local target=""
  local quiet="${CHAINBENCH_QUIET:-0}"
  local remote_alias=""
  local format="${CB_FORMAT:-text}"
  local dry_run=0

  # Parse args: target, --remote, --format, --dry-run flags
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --remote) remote_alias="${2:?--remote requires an alias}"; shift 2 ;;
      --remote=*) remote_alias="${1#--remote=}"; shift ;;
      --format) format="${2:-text}"; shift 2 ;;
      --format=*) format="${1#--format=}"; shift ;;
      --dry-run) dry_run=1; shift ;;
      *) [[ -z "$target" ]] && target="$1" || true; shift ;;
    esac
  done

  # Export format for child test scripts
  export CB_FORMAT="$format"
  target="${target:-all}"

  # When --remote is set, export CHAINBENCH_REMOTE for test scripts
  if [[ -n "$remote_alias" ]]; then
    export CHAINBENCH_REMOTE="$remote_alias"
    log_info "Running tests against remote chain: ${remote_alias}"

    # When target is "all" with --remote, only run remote/ category
    if [[ "$target" == "all" ]]; then
      target="remote"
      log_info "Target 'all' with --remote: running only remote/ tests"
    fi
  fi

  # Collect matching scripts
  local -a scripts=()
  while IFS= read -r s; do
    [[ -n "$s" ]] && scripts+=("$s")
  done < <(_cb_test_collect_scripts "$target" 2>/dev/null)

  if [[ "${#scripts[@]}" -eq 0 ]]; then
    log_error "No tests found for target: '$target'"
    log_error "Run 'chainbench test list' to see available tests."
    return 1
  fi

  # Dry-run: output execution plan without running tests
  if [[ "$dry_run" -eq 1 ]]; then
    _cb_test_dry_run "$target" "${scripts[@]}"
    return $?
  fi

  local total=${#scripts[@]}
  local passed=0
  local failed=0
  local skipped=0
  local -a failed_names=()
  local -a skipped_names=()

  printf '\n'
  printf "${_CB_TEST_CYAN}[TEST]${_CB_TEST_RESET}  Running %d test(s) for target: %s\n" \
    "$total" "$target" >&2
  printf "${_CB_TEST_CYAN}[TEST]${_CB_TEST_RESET}  %s\n" \
    "$(printf '%.0s─' $(seq 1 50))" >&2

  local script name exit_code
  for script in "${scripts[@]}"; do
    name=$(_cb_test_script_to_name "$script")
    printf '\n' >&2

    # Capability gate: skip tests whose required capabilities are not provided
    # by the active network. SKIP diagnostic is emitted by the helper.
    if ! _cb_test_check_capabilities "$script"; then
      skipped=$(( skipped + 1 ))
      skipped_names+=("$name")
      continue
    fi

    # Run test; capture exit code without aborting the runner
    set +e
    _cb_test_run_single "$script"
    exit_code=$?
    set -e

    if [[ "$exit_code" -eq 0 ]]; then
      passed=$(( passed + 1 ))
      printf "${_CB_TEST_GREEN}[PASS]${_CB_TEST_RESET}  %s\n" "$name" >&2
    else
      failed=$(( failed + 1 ))
      failed_names+=("$name")
      printf "${_CB_TEST_RED}[FAIL]${_CB_TEST_RESET}  %s (exit code: %d)\n" \
        "$name" "$exit_code" >&2
    fi
  done

  # Print summary
  printf '\n' >&2
  printf "${_CB_TEST_CYAN}[TEST]${_CB_TEST_RESET}  %s\n" \
    "$(printf '%.0s─' $(seq 1 50))" >&2
  printf "${_CB_TEST_BOLD}[TEST]  Summary: %d/%d passed" "$passed" "$total" >&2

  if [[ "$failed" -gt 0 ]]; then
    printf " | %d failed" "$failed" >&2
  fi
  if [[ "$skipped" -gt 0 ]]; then
    printf " | %d skipped" "$skipped" >&2
  fi
  printf "${_CB_TEST_RESET}\n" >&2

  if [[ "$failed" -gt 0 ]]; then
    printf "${_CB_TEST_RED}[FAIL]${_CB_TEST_RESET}  Failed tests:\n" >&2
    local fn
    for fn in "${failed_names[@]}"; do
      printf "         - %s\n" "$fn" >&2
    done
  fi
  if [[ "$skipped" -gt 0 ]]; then
    printf "${_CB_TEST_YELLOW}[SKIP]${_CB_TEST_RESET}  Skipped tests (capability gating):\n" >&2
    local sn
    for sn in "${skipped_names[@]}"; do
      printf "         - %s\n" "$sn" >&2
    done
  fi
  if [[ "$failed" -eq 0 && "$skipped" -eq 0 ]]; then
    printf "${_CB_TEST_GREEN}[TEST]${_CB_TEST_RESET}  All tests passed.\n" >&2
  fi

  # Output JSON summary when --quiet
  if [[ "$quiet" == "1" ]]; then
    python3 -c "
import json, sys

passed = int(sys.argv[1])
failed = int(sys.argv[2])
total  = int(sys.argv[3])
failed_names_raw = sys.argv[4]
target = sys.argv[5]

failed_names = [n for n in failed_names_raw.split('\x1f') if n]

summary = {
    'target':  target,
    'total':   total,
    'passed':  passed,
    'failed':  failed,
    'status':  'passed' if failed == 0 else 'failed',
    'failed_tests': failed_names,
}
print(json.dumps(summary, indent=2))
" \
      "$passed" "$failed" "$total" \
      "$(printf '%s\x1f' "${failed_names[@]+"${failed_names[@]}"}")" \
      "$target"
  fi

  [[ "$failed" -eq 0 ]]
}

# ---- Subcommand dispatcher ---------------------------------------------------

cmd_test_main() {
  if [[ $# -lt 1 ]]; then
    _cb_test_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    list)
      _cb_test_cmd_list "$@"
      ;;
    run)
      _cb_test_cmd_run "$@"
      ;;
    scaffold)
      # chainbench test scaffold --spec <path> --id <RT-ID> [--target <dir>]
      local spec_path="" scaffold_id="" target_dir="${_CB_TEST_TESTS_DIR}/regression"
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --spec) spec_path="${2:?--spec requires a path}"; shift 2 ;;
          --id)   scaffold_id="${2:?--id requires an RT-ID}"; shift 2 ;;
          --target) target_dir="${2:?--target requires a directory}"; shift 2 ;;
          *) shift ;;
        esac
      done
      if [[ -z "$spec_path" || -z "$scaffold_id" ]]; then
        log_error "Usage: chainbench test scaffold --spec <path> --id <RT-ID> [--target <dir>]"
        return 1
      fi
      _cb_test_scaffold "$spec_path" "$scaffold_id" "$target_dir"
      ;;
    --help|-h|help)
      _cb_test_usage
      return 0
      ;;
    *)
      log_error "Unknown test subcommand: '$subcmd'"
      _cb_test_usage
      return 1
      ;;
  esac
}

# ---- Entry point -------------------------------------------------------------

cmd_test_main "$@"
