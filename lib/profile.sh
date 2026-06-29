#!/usr/bin/env bash
# lib/profile.sh - YAML profile parser and environment variable exporter
# Source this file: source lib/profile.sh
#
# Primary entry point: load_profile <profile_name>
# After a successful call all CHAINBENCH_* variables are exported in the
# current shell and CHAINBENCH_PROFILE_JSON points to a temporary JSON file
# containing the fully merged profile.

# Guard against double-sourcing
[[ -n "${_CHAINBENCH_PROFILE_SH_LOADED:-}" ]] && return 0
readonly _CHAINBENCH_PROFILE_SH_LOADED=1

# Source common utilities if not already loaded
_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${_CB_LIB_DIR}/common.sh"
# Cross-layer default constants (SSOT-X1, generated from network/schema/defaults.json).
# shellcheck source=lib/defaults.generated.sh
source "${_CB_LIB_DIR}/defaults.generated.sh"

# ---- Internal constants ------------------------------------------------------

readonly _CB_PROFILES_DIR="${_CB_LIB_DIR}/../profiles"
readonly _CB_SCRIPTS_DIR="${_CB_LIB_DIR}/../scripts"

# ---- Python helpers ----------------------------------------------------------
# The YAML->JSON merge and JSON field extraction live in standalone scripts
# (scripts/merge_profile.py, scripts/extract_json.py) so this file stays shell.
# The argv contracts are unchanged from the previously-inlined heredocs (P2-1).

# _cb_python_merge_yaml <profile_path>
# Outputs a fully merged JSON document to stdout (uses PyYAML when available,
# else a built-in subset parser). Args passed positionally to merge_profile.py.
_cb_python_merge_yaml() {
  local profile_path="${1:?_cb_python_merge_yaml: profile path required}"
  local chainbench_dir="${CHAINBENCH_DIR:-}"

  python3 "${_CB_SCRIPTS_DIR}/merge_profile.py" "$profile_path" "$_CB_PROFILES_DIR" "$chainbench_dir"
}


# ---- Profile location --------------------------------------------------------

# _cb_find_profile_path <name>
# Prints the path to the YAML file. Returns 1 if not found.
_cb_find_profile_path() {
  local name="${1:?_cb_find_profile_path: profile name required}"

  local candidates=(
    "${_CB_PROFILES_DIR}/${name}.yaml"
    "${_CB_PROFILES_DIR}/custom/${name}.yaml"
  )

  local p
  for p in "${candidates[@]}"; do
    if [[ -f "$p" ]]; then
      printf '%s\n' "$p"
      return 0
    fi
  done

  log_error "Profile '$name' not found. Searched:"
  for p in "${candidates[@]}"; do
    log_error "  $p"
  done
  return 1
}

# ---- JSON field extraction ---------------------------------------------------

# _cb_jq_get <json_file> <jq_filter> [default_value]
# Extracts a value from a JSON file via scripts/extract_json.py (avoids jq).
# Argv contract unchanged from the previously-inlined heredoc (P2-1).
_cb_jq_get() {
  local json_file="${1:?_cb_jq_get: json file required}"
  local filter="${2:?_cb_jq_get: jq filter required}"
  local default="${3:-}"

  python3 "${_CB_SCRIPTS_DIR}/extract_json.py" "$json_file" "$filter" "$default"
}

# ---- Validation --------------------------------------------------------------

# _cb_validate_profile_json <json_file>
# Checks that mandatory fields are present. Returns 1 on failure.
_cb_validate_profile_json() {
  local json_file="${1:?_cb_validate_profile_json: json file required}"
  local errors=0

  local required_fields=(
    ".chain.binary"
    ".nodes.validators"
    ".ports.base_p2p"
    ".ports.base_http"
  )

  local field
  for field in "${required_fields[@]}"; do
    local val
    val="$(_cb_jq_get "$json_file" "$field")"
    if [[ -z "$val" ]]; then
      log_error "Profile validation: required field '$field' is missing or empty"
      (( errors++ ))
    fi
  done

  [[ $errors -eq 0 ]]
}

# ---- Export ------------------------------------------------------------------

# _cb_export_profile_vars <json_file> <profile_name>
# Reads the merged JSON and exports CHAINBENCH_* environment variables.
_cb_export_profile_vars() {
  local json_file="${1:?_cb_export_profile_vars: json file required}"
  local profile_name="${2:?_cb_export_profile_vars: profile name required}"

  # Helper: read a field, fall back to default, then export as CHAINBENCH_VAR.
  # Env-first guard: if the variable is already set and non-empty, preserve it.
  # This allows CLI flags and env vars to take precedence over profile values.
  # Opt-out: set CHAINBENCH_PROFILE_ENV_OVERRIDE=0 to always use profile values.
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

  # Profile metadata
  export CHAINBENCH_PROFILE_NAME="$profile_name"
  export CHAINBENCH_PROFILE_JSON="$json_file"

  # chain.*
  _cb_set_var CHAINBENCH_BINARY            ".chain.binary"       "gstable"
  _cb_set_var CHAINBENCH_BINARY_PATH       ".chain.binary_path"  ""
  _cb_set_var CHAINBENCH_NETWORK_ID        ".chain.network_id"   ""
  _cb_set_var CHAINBENCH_CHAIN_ID          ".chain.chain_id"     ""
  _cb_set_var CHAINBENCH_CHAIN_TYPE        ".chain.type"          "stablenet"
  _cb_set_var CHAINBENCH_LOGROT_PATH      ".chain.logrot_path"   ""

  # data.*
  _cb_set_var CHAINBENCH_DATA_DIR          ".data.directory"     "data"

  # genesis.*
  _cb_set_var CHAINBENCH_GENESIS_TEMPLATE  ".genesis.template"   ""

  # nodes.*
  _cb_set_var CHAINBENCH_VALIDATORS        ".nodes.validators"   "1"
  _cb_set_var CHAINBENCH_ENDPOINTS         ".nodes.endpoints"    "0"
  _cb_set_var CHAINBENCH_VERBOSITY         ".nodes.verbosity"    "3"
  _cb_set_var CHAINBENCH_EN_VERBOSITY      ".nodes.en_verbosity" "3"
  _cb_set_var CHAINBENCH_GCMODE            ".nodes.gcmode"       "full"
  _cb_set_var CHAINBENCH_CACHE             ".nodes.cache"        "1024"
  _cb_set_var CHAINBENCH_EXTRA_FLAGS       ".nodes.extra_flags"  ""

  # Computed total node count
  export CHAINBENCH_TOTAL_NODES=$(( CHAINBENCH_VALIDATORS + CHAINBENCH_ENDPOINTS ))

  # keys.*
  _cb_set_var CHAINBENCH_KEYS_MODE         ".keys.mode"          "static"
  _cb_set_var CHAINBENCH_KEYS_SOURCE       ".keys.source"        "keys/preset"

  # ports.*
  _cb_set_var CHAINBENCH_BASE_P2P          ".ports.base_p2p"     "${CB_DEFAULT_BASE_P2P}"
  _cb_set_var CHAINBENCH_BASE_HTTP         ".ports.base_http"    "${CB_DEFAULT_BASE_HTTP}"
  _cb_set_var CHAINBENCH_BASE_WS           ".ports.base_ws"      "${CB_DEFAULT_BASE_WS}"
  _cb_set_var CHAINBENCH_BASE_AUTH         ".ports.base_auth"    "${CB_DEFAULT_BASE_AUTH}"
  _cb_set_var CHAINBENCH_BASE_METRICS      ".ports.base_metrics" "${CB_DEFAULT_BASE_METRICS}"

  # logging.*
  _cb_set_var CHAINBENCH_LOG_ROTATION      ".logging.rotation"   "true"
  _cb_set_var CHAINBENCH_LOG_MAX_SIZE      ".logging.max_size"   "10M"
  _cb_set_var CHAINBENCH_LOG_MAX_FILES     ".logging.max_files"  "5"
  _cb_set_var CHAINBENCH_LOG_DIR           ".logging.directory"  "data/logs"

  # tests.*
  _cb_set_var CHAINBENCH_AUTO_RUN_TESTS    ".tests.auto_run"     ""

  unset -f _cb_set_var
}

# ---- Temporary file management -----------------------------------------------

# _cb_cleanup_profile_json
# Removes the temporary JSON file if it was created by this session.
_cb_cleanup_profile_json() {
  if [[ -n "${CHAINBENCH_PROFILE_JSON:-}" && -f "${CHAINBENCH_PROFILE_JSON}" ]]; then
    rm -f "${CHAINBENCH_PROFILE_JSON}"
    unset CHAINBENCH_PROFILE_JSON
  fi
}

# Register cleanup unless caller opts out
if [[ "${CHAINBENCH_NO_CLEANUP:-0}" != "1" ]]; then
  trap _cb_cleanup_profile_json EXIT
fi

# ---- Public API --------------------------------------------------------------

# load_profile <profile_name>
# Main entry point. Resolves the profile (with inheritance), validates it,
# exports all CHAINBENCH_* variables, and sets CHAINBENCH_PROFILE_JSON.
load_profile() {
  local profile_name="${1:?load_profile: profile name required}"

  require_cmd python3 "python3 is required to parse YAML profiles" || return 1

  # Locate the YAML source file
  local profile_path
  profile_path="$(_cb_find_profile_path "$profile_name")" || return 1

  log_info "Loading profile '$profile_name' from $profile_path"

  # Create a temporary file to hold the merged JSON
  local json_tmp
  json_tmp="$(mktemp /tmp/chainbench-profile-XXXXXX)" || {
    log_error "Failed to create temporary file for profile JSON"
    return 1
  }

  # Parse YAML -> JSON (handles inheritance internally)
  local python_output
  if ! python_output="$(_cb_python_merge_yaml "$profile_path")"; then
    log_error "Failed to parse profile '$profile_name'"
    rm -f "$json_tmp"
    return 1
  fi

  # Validate that we got non-empty JSON
  if [[ -z "$python_output" ]]; then
    log_error "Profile parser produced no output for '$profile_name'"
    rm -f "$json_tmp"
    return 1
  fi

  printf '%s\n' "$python_output" > "$json_tmp"

  # Validate required fields
  if ! _cb_validate_profile_json "$json_tmp"; then
    rm -f "$json_tmp"
    return 1
  fi

  # Export environment variables
  _cb_export_profile_vars "$json_tmp" "$profile_name"

  log_success "Profile '$profile_name' loaded: ${CHAINBENCH_VALIDATORS}v + ${CHAINBENCH_ENDPOINTS}en (total ${CHAINBENCH_TOTAL_NODES} nodes)"
  return 0
}
