#!/usr/bin/env python3
"""json_backend.py — the single JSON backend behind lib/json_helpers.sh (P2-1b).

Replaces the previous jq/python3 dual backend. Each subcommand mirrors the
behavior of the corresponding cb_json_* shell function's python path; write/merge
are made atomic (write a sibling temp file, then os.replace) to preserve the
crash-safety the old jq path provided via mktemp+mv.

    json_backend.py read       <file> <dotpath> <default>
    json_backend.py read-stdin <dotpath> <default>          # JSON on stdin
    json_backend.py array-len  <file> <dotpath>
    json_backend.py write      <file> <dotpath> <value>     # atomic
    json_backend.py merge      <file> <override_json>       # atomic
    json_backend.py get-result <json_string>
    json_backend.py has-error  <json_string>                # exit code only
"""
import sys
import json
import os
import tempfile


def _navigate(node, dot_path):
    """Walk a dot path, indexing dicts by key and lists by int. Raises on miss."""
    for part in [p for p in dot_path.split('.') if p]:
        if isinstance(node, dict):
            node = node[part]
        elif isinstance(node, list):
            node = node[int(part)]
        else:
            raise KeyError(part)
    return node


def _emit(node, default):
    """Print a resolved value the way the shell callers expect."""
    if node is None:
        print(default)
    elif isinstance(node, bool):
        print(str(node).lower())
    elif isinstance(node, (dict, list)):
        print(json.dumps(node))
    else:
        print(node)


def _atomic_dump(file_path, data):
    """Write data as pretty JSON (+trailing newline) atomically via os.replace."""
    target_dir = os.path.dirname(os.path.abspath(file_path)) or "."
    fd, tmp = tempfile.mkstemp(dir=target_dir, suffix=".tmp")
    try:
        with os.fdopen(fd, 'w') as fh:
            json.dump(data, fh, indent=2)
            fh.write('\n')
        os.replace(tmp, file_path)
    except Exception:
        try:
            os.unlink(tmp)
        except OSError:
            pass
        raise


def cmd_read(argv):
    file_path, dot_path, default = argv[0], argv[1], argv[2]
    try:
        with open(file_path) as fh:
            data = json.load(fh)
    except Exception:
        print(default)
        sys.exit(1)
    try:
        _emit(_navigate(data, dot_path), default)
    except (KeyError, IndexError, TypeError, ValueError):
        print(default)


def cmd_read_stdin(argv):
    dot_path, default = argv[0], argv[1]
    data = json.loads(sys.stdin.read())
    try:
        _emit(_navigate(data, dot_path), default)
    except (KeyError, IndexError, TypeError, ValueError):
        print(default)


def cmd_array_len(argv):
    # Errors propagate (non-zero exit); the shell wrapper maps that to "0" + rc 1.
    file_path, dot_path = argv[0], argv[1]
    with open(file_path) as fh:
        data = json.load(fh)
    node = _navigate(data, dot_path)
    print(len(node) if isinstance(node, (list, dict)) else 0)


def _cast(v):
    if v == "null" or v == "":
        return None
    if v == "true":
        return True
    if v == "false":
        return False
    try:
        return int(v)
    except ValueError:
        pass
    try:
        return float(v)
    except ValueError:
        pass
    return v


def cmd_write(argv):
    file_path, dot_path, raw_value = argv[0], argv[1], argv[2]
    with open(file_path) as fh:
        data = json.load(fh)

    parts = [p for p in dot_path.split('.') if p]
    node = data
    for p in parts[:-1]:
        if isinstance(node, dict):
            node = node.setdefault(p, {})
        elif isinstance(node, list):
            node = node[int(p)]

    last = parts[-1]
    if isinstance(node, dict):
        node[last] = _cast(raw_value)
    elif isinstance(node, list):
        node[int(last)] = _cast(raw_value)

    _atomic_dump(file_path, data)


def _deep_merge(base, over):
    result = dict(base)
    for k, v in over.items():
        if k in result and isinstance(result[k], dict) and isinstance(v, dict):
            result[k] = _deep_merge(result[k], v)
        else:
            result[k] = v
    return result


def cmd_merge(argv):
    file_path, override = argv[0], json.loads(argv[1])
    with open(file_path) as fh:
        base = json.load(fh)
    _atomic_dump(file_path, _deep_merge(base, override))


def cmd_get_result(argv):
    try:
        d = json.loads(argv[0])
        if 'error' in d and d['error'] is not None:
            msg = (d['error'].get('message', 'unknown error')
                   if isinstance(d['error'], dict) else str(d['error']))
            print(f'RPC error: {msg}', file=sys.stderr)
            sys.exit(1)
        result = d.get('result')
        if result is None:
            pass
        elif isinstance(result, bool):
            print(str(result).lower())
        elif isinstance(result, (dict, list)):
            print(json.dumps(result))
        else:
            print(result)
    except (json.JSONDecodeError, KeyError) as e:
        print(f'JSON parse error: {e}', file=sys.stderr)
        sys.exit(1)


def cmd_has_error(argv):
    d = json.loads(argv[0])
    sys.exit(0 if ('error' in d and d['error'] is not None) else 1)


_DISPATCH = {
    "read": cmd_read,
    "read-stdin": cmd_read_stdin,
    "array-len": cmd_array_len,
    "write": cmd_write,
    "merge": cmd_merge,
    "get-result": cmd_get_result,
    "has-error": cmd_has_error,
}


def main():
    if len(sys.argv) < 2 or sys.argv[1] not in _DISPATCH:
        print(f"json_backend.py: unknown subcommand {sys.argv[1:2]}", file=sys.stderr)
        sys.exit(2)
    _DISPATCH[sys.argv[1]](sys.argv[2:])


if __name__ == "__main__":
    main()
