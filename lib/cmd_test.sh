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

# ---- Constants ---------------------------------------------------------------

readonly _CB_TEST_TESTS_DIR="${CHAINBENCH_DIR}/tests"
readonly _CB_TEST_CATEGORIES=(basic fault stress upgrade)

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
Usage: chainbench test <subcommand> [target]

Subcommands:
  list              List all available tests with descriptions
  run <target>      Run tests matching target

Target formats:
  all               Run all tests in all categories
  basic             Run all tests in the basic/ category
  fault             Run all tests in the fault/ category
  stress            Run all tests in the stress/ category
  upgrade           Run all tests in the upgrade/ category
  basic/consensus   Run a single specific test

Examples:
  chainbench test list
  chainbench test run all
  chainbench test run basic
  chainbench test run basic/consensus
  chainbench test run fault/node-crash
EOF
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

  # Use find with sorted output; avoid globbing issues with empty dirs
  find "$dir" -maxdepth 1 -name '*.sh' -type f | sort
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

# ---- test run ----------------------------------------------------------------

_cb_test_cmd_run() {
  local target="${1:-all}"
  local quiet="${CHAINBENCH_QUIET:-0}"

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

  local total=${#scripts[@]}
  local passed=0
  local failed=0
  local -a failed_names=()

  printf '\n'
  printf "${_CB_TEST_CYAN}[TEST]${_CB_TEST_RESET}  Running %d test(s) for target: %s\n" \
    "$total" "$target" >&2
  printf "${_CB_TEST_CYAN}[TEST]${_CB_TEST_RESET}  %s\n" \
    "$(printf '%.0s─' $(seq 1 50))" >&2

  local script name exit_code
  for script in "${scripts[@]}"; do
    name=$(_cb_test_script_to_name "$script")
    printf '\n' >&2

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
    printf " | %d failed${_CB_TEST_RESET}\n" "$failed" >&2
    printf "${_CB_TEST_RED}[FAIL]${_CB_TEST_RESET}  Failed tests:\n" >&2
    local fn
    for fn in "${failed_names[@]}"; do
      printf "         - %s\n" "$fn" >&2
    done
  else
    printf "${_CB_TEST_RESET}\n" >&2
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
      _cb_test_cmd_list
      ;;
    run)
      local target="${1:-all}"
      _cb_test_cmd_run "$target"
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
