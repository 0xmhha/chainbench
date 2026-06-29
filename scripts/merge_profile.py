#!/usr/bin/env python3
"""merge_profile.py — resolve a chainbench YAML profile to a merged JSON document.

Extracted verbatim from lib/profile.sh's `_cb_python_merge_yaml` (P2-1). The
contract is unchanged so the shell wrapper is a drop-in:

    python3 merge_profile.py <profile_path> <profiles_root> [chainbench_dir]

  argv[1] PROFILE_PATH   absolute path to the YAML profile to load
  argv[2] PROFILES_ROOT  profiles directory (used to resolve `inherits:` parents)
  argv[3] CHAINBENCH_DIR optional; if set, merges state/local-config.yaml overlay

Output: a single minified JSON object on stdout. On any error: `ERROR: <msg>` on
stderr and exit 1. Uses PyYAML when importable, else a built-in subset parser
(zero third-party dependency in the fallback path).
"""
import sys
import json
import os
import re

PROFILE_PATH  = sys.argv[1]
PROFILES_ROOT = sys.argv[2]
CHAINBENCH_DIR = sys.argv[3] if len(sys.argv) > 3 else ""


# --------------------------------------------------------------------------- #
# YAML loading                                                                 #
# --------------------------------------------------------------------------- #

def _simple_yaml_parse(text):
    """
    Minimal YAML parser for the subset used by chainbench profiles.

    Handles:
      - Indented block mappings (nested dicts)
      - Sequences introduced by '- value' lines
      - Quoted / unquoted scalar values
      - Inline comments (# ...)
      - null / true / false / integer / float scalars
      - Blank lines and pure-comment lines are ignored
    """
    lines = text.splitlines()
    # Annotate each non-empty, non-comment line with its indent level
    tokens = []
    for raw in lines:
        stripped = raw.rstrip()
        if not stripped or stripped.lstrip().startswith('#'):
            continue
        indent = len(stripped) - len(stripped.lstrip())
        content = stripped.lstrip()
        # Strip inline comment (outside quotes)
        content = _strip_inline_comment(content)
        tokens.append((indent, content))

    result, _ = _parse_mapping(tokens, 0, -1)
    return result


def _strip_inline_comment(s):
    """Remove trailing # comment that is not inside quotes."""
    in_single = False
    in_double = False
    for i, ch in enumerate(s):
        if ch == "'" and not in_double:
            in_single = not in_single
        elif ch == '"' and not in_single:
            in_double = not in_double
        elif ch == '#' and not in_single and not in_double:
            return s[:i].rstrip()
    return s


def _cast_scalar(raw):
    """Convert a scalar string to a Python native type."""
    raw = raw.strip()
    # Quoted strings
    if (raw.startswith('"') and raw.endswith('"')) or \
       (raw.startswith("'") and raw.endswith("'")):
        return raw[1:-1]
    lower = raw.lower()
    if lower in ('null', '~', ''):
        return None
    if lower == 'true':
        return True
    if lower == 'false':
        return False
    # Integer
    try:
        return int(raw)
    except ValueError:
        pass
    # Float
    try:
        return float(raw)
    except ValueError:
        pass
    return raw


def _parse_mapping(tokens, start, parent_indent):
    """
    Parse a YAML mapping (dict) starting at tokens[start].
    Returns (dict, next_index).
    Stops when a token with indent <= parent_indent is encountered.
    """
    result = {}
    i = start
    while i < len(tokens):
        indent, content = tokens[i]
        if indent <= parent_indent:
            break
        # Sequence item - should not appear at mapping level, skip
        if content.startswith('- ') or content == '-':
            break
        # Key: value  or  Key:
        if ':' not in content:
            i += 1
            continue
        colon_pos = content.index(':')
        key = content[:colon_pos].strip()
        rest = content[colon_pos + 1:].strip()

        if rest:
            # Inline value on same line
            result[key] = _cast_scalar(rest)
            i += 1
        else:
            # Value continues on next lines
            i += 1
            if i < len(tokens):
                next_indent, next_content = tokens[i]
                if next_indent > indent:
                    if next_content.startswith('- ') or next_content == '-':
                        # Sequence
                        seq, i = _parse_sequence(tokens, i, indent)
                        result[key] = seq
                    else:
                        # Nested mapping
                        sub, i = _parse_mapping(tokens, i, indent)
                        result[key] = sub
                else:
                    result[key] = None
            else:
                result[key] = None
    return result, i


def _parse_sequence(tokens, start, parent_indent):
    """
    Parse a YAML sequence (list) starting at tokens[start].
    Returns (list, next_index).
    """
    result = []
    i = start
    while i < len(tokens):
        indent, content = tokens[i]
        if indent <= parent_indent:
            break
        if content.startswith('- '):
            item_str = content[2:].strip()
            result.append(_cast_scalar(item_str))
            i += 1
        elif content == '-':
            result.append(None)
            i += 1
        else:
            break
    return result, i


def load_yaml_file(path):
    try:
        import yaml
        with open(path) as fh:
            return yaml.safe_load(fh) or {}
    except ImportError:
        with open(path) as fh:
            return _simple_yaml_parse(fh.read()) or {}


# --------------------------------------------------------------------------- #
# Deep merge                                                                   #
# --------------------------------------------------------------------------- #

def deep_merge(base, override):
    """Recursively merge override into base; override wins on conflicts."""
    result = dict(base)
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = value
    return result


# --------------------------------------------------------------------------- #
# Inheritance resolution                                                       #
# --------------------------------------------------------------------------- #

MAX_INHERIT_DEPTH = 10

def find_profile(name, profiles_root):
    """Search profiles/<name>.yaml then profiles/custom/<name>.yaml."""
    candidates = [
        os.path.join(profiles_root, f"{name}.yaml"),
        os.path.join(profiles_root, "custom", f"{name}.yaml"),
    ]
    for c in candidates:
        if os.path.isfile(c):
            return c
    return None


def load_with_inheritance(path, profiles_root, depth=0):
    if depth > MAX_INHERIT_DEPTH:
        raise RuntimeError(f"Inheritance depth exceeded ({MAX_INHERIT_DEPTH}), "
                           "possible circular reference")
    data = load_yaml_file(path)
    parent_name = data.get('inherits')
    if not parent_name:
        return data

    parent_path = find_profile(str(parent_name), profiles_root)
    if parent_path is None:
        raise FileNotFoundError(
            f"Parent profile '{parent_name}' not found in {profiles_root}"
        )

    parent_data = load_with_inheritance(parent_path, profiles_root, depth + 1)
    # Remove meta-only field before merging
    child = {k: v for k, v in data.items() if k != 'inherits'}
    return deep_merge(parent_data, child)


# --------------------------------------------------------------------------- #
# Main                                                                         #
# --------------------------------------------------------------------------- #

try:
    merged = load_with_inheritance(PROFILE_PATH, PROFILES_ROOT)

    # Merge local overlay if present (machine-local, git-ignored)
    if CHAINBENCH_DIR:
        overlay_path = os.path.join(CHAINBENCH_DIR, "state", "local-config.yaml")
        if os.path.isfile(overlay_path):
            overlay = load_yaml_file(overlay_path)
            if overlay:
                if 'inherits' in overlay:
                    print("WARN: local-config.yaml: 'inherits' field ignored in overlays",
                          file=sys.stderr)
                    overlay = {k: v for k, v in overlay.items() if k != 'inherits'}
                merged = deep_merge(merged, overlay)

    print(json.dumps(merged, ensure_ascii=False))
except Exception as exc:
    print(f"ERROR: {exc}", file=sys.stderr)
    sys.exit(1)
