#!/usr/bin/env bash
# lib/cmd_profile.sh - Dispatch handler for `chainbench profile <subcommand>`
# Sourced by the main chainbench dispatcher; $@ contains subcommand arguments.
#
# Subcommands:
#   profile list                  List all available profiles
#   profile show <name>           Print YAML content of a profile
#   profile create <name>         Create a new custom profile from default.yaml

# Guard against double-sourcing
[[ -n "${_CB_CMD_PROFILE_SH_LOADED:-}" ]] && return 0
readonly _CB_CMD_PROFILE_SH_LOADED=1

_CB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${_CB_LIB_DIR}/common.sh"

# ---- Constants ---------------------------------------------------------------

readonly _CB_PROFILE_DIR="${_CB_LIB_DIR}/../profiles"
readonly _CB_PROFILE_CUSTOM_DIR="${_CB_PROFILE_DIR}/custom"
readonly _CB_PROFILE_DEFAULT_FILE="${_CB_PROFILE_DIR}/default.yaml"

# ---- Usage -------------------------------------------------------------------

_cb_profile_usage() {
  cat >&2 <<'EOF'
Usage: chainbench profile <subcommand> [options]

Subcommands:
  list                  List all available profiles (built-in + custom)
  show   <name>         Print the YAML content of a profile
  create <name>         Create a new custom profile based on default.yaml

Examples:
  chainbench profile list
  chainbench profile show minimal
  chainbench profile create my-4node
EOF
}

# ---- Subcommand: list --------------------------------------------------------

_cb_profile_cmd_list() {
  python3 - "$_CB_PROFILE_DIR" "$_CB_PROFILE_CUSTOM_DIR" <<'PYEOF'
import sys, os, re

profiles_dir = sys.argv[1]
custom_dir   = sys.argv[2]

NAME_RE        = re.compile(r'^\s*name\s*:\s*(.+)')
DESC_RE        = re.compile(r'^\s*description\s*:\s*(.+)')
INHERITS_RE    = re.compile(r'^\s*inherits\s*:\s*(.+)')


def parse_yaml_meta(path):
    """Extract name, description, and inherits from the first few lines of YAML."""
    name        = os.path.splitext(os.path.basename(path))[0]
    description = ''
    inherits    = ''

    try:
        with open(path) as fh:
            for line in fh:
                m = NAME_RE.match(line)
                if m:
                    name = m.group(1).strip().strip('"').strip("'")

                m = DESC_RE.match(line)
                if m:
                    description = m.group(1).strip().strip('"').strip("'")

                m = INHERITS_RE.match(line)
                if m:
                    inherits = m.group(1).strip().strip('"').strip("'")
    except OSError:
        pass

    return name, description, inherits


def collect_profiles(directory, source_label):
    """Yield (file_path, name, description, inherits, source_label) for each .yaml."""
    if not os.path.isdir(directory):
        return
    for entry in sorted(os.listdir(directory)):
        if not entry.endswith('.yaml'):
            continue
        path = os.path.join(directory, entry)
        if not os.path.isfile(path):
            continue
        name, desc, inherits = parse_yaml_meta(path)
        yield path, name, desc, inherits, source_label


profiles = list(collect_profiles(profiles_dir, 'built-in'))
profiles += list(collect_profiles(custom_dir, 'custom'))

if not profiles:
    print('No profiles found.')
    sys.exit(0)

# Determine column widths
max_name  = max(len(p[1]) for p in profiles)
max_name  = max(max_name, 4)
max_src   = max(len(p[4]) for p in profiles)
max_src   = max(max_src, 6)
max_inh   = max(len(p[3]) if p[3] else 1 for p in profiles)
max_inh   = max(max_inh, 8)

header = (
    f"{'Name':<{max_name}}  "
    f"{'Source':<{max_src}}  "
    f"{'Inherits':<{max_inh}}  "
    f"Description"
)
print(header)
print('-' * (max_name + max_src + max_inh + 30))

for path, name, desc, inherits, source in profiles:
    inh_label = inherits if inherits else '-'
    print(
        f'{name:<{max_name}}  '
        f'{source:<{max_src}}  '
        f'{inh_label:<{max_inh}}  '
        f'{desc}'
    )
PYEOF
}

# ---- Subcommand: show --------------------------------------------------------

_cb_profile_cmd_show() {
  local profile_name="${1:-}"

  if [[ -z "$profile_name" ]]; then
    log_error "Usage: chainbench profile show <name>"
    return 1
  fi

  local -a candidates=(
    "${_CB_PROFILE_DIR}/${profile_name}.yaml"
    "${_CB_PROFILE_CUSTOM_DIR}/${profile_name}.yaml"
  )

  local path
  for path in "${candidates[@]}"; do
    if [[ -f "$path" ]]; then
      cat "$path"
      return 0
    fi
  done

  log_error "Profile '$profile_name' not found. Searched:"
  for path in "${candidates[@]}"; do
    log_error "  $path"
  done
  return 1
}

# ---- Subcommand: create ------------------------------------------------------

_cb_profile_cmd_create() {
  local profile_name="${1:-}"

  if [[ -z "$profile_name" ]]; then
    log_error "Usage: chainbench profile create <name>"
    return 1
  fi

  # Validate name: only alphanumeric, hyphens, and underscores
  if ! [[ "$profile_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
    log_error "Profile name must contain only alphanumeric characters, hyphens, and underscores"
    return 1
  fi

  if [[ ! -f "$_CB_PROFILE_DEFAULT_FILE" ]]; then
    log_error "Default profile not found: $_CB_PROFILE_DEFAULT_FILE"
    return 1
  fi

  # Ensure the custom directory exists
  if [[ ! -d "$_CB_PROFILE_CUSTOM_DIR" ]]; then
    mkdir -p "$_CB_PROFILE_CUSTOM_DIR" || {
      log_error "Failed to create custom profiles directory: $_CB_PROFILE_CUSTOM_DIR"
      return 1
    }
  fi

  local dest="${_CB_PROFILE_CUSTOM_DIR}/${profile_name}.yaml"

  if [[ -f "$dest" ]]; then
    log_warn "Profile '$profile_name' already exists: $dest"
    return 1
  fi

  cp "$_CB_PROFILE_DEFAULT_FILE" "$dest" || {
    log_error "Failed to copy default profile to: $dest"
    return 1
  }

  log_success "Created custom profile: $dest"
  printf '%s\n' "$dest"
  return 0
}

# ---- Main dispatcher ---------------------------------------------------------

cmd_profile_main() {
  if [[ $# -lt 1 ]]; then
    _cb_profile_usage
    return 1
  fi

  local subcmd="$1"
  shift

  case "$subcmd" in
    list)
      _cb_profile_cmd_list "$@"
      ;;
    show)
      _cb_profile_cmd_show "$@"
      ;;
    create)
      _cb_profile_cmd_create "$@"
      ;;
    --help|-h|help)
      _cb_profile_usage
      return 0
      ;;
    *)
      log_error "Unknown profile subcommand: '$subcmd'"
      _cb_profile_usage
      return 1
      ;;
  esac
}

# ---- Entry point -------------------------------------------------------------

cmd_profile_main "$@"
