#!/usr/bin/env python3
"""extract_json.py — read a value from a JSON file by dot-path.

Extracted verbatim from lib/profile.sh's `_cb_jq_get` (P2-1). Avoids a jq
dependency. Contract unchanged:

    python3 extract_json.py <json_file> <dot_filter> [default]

  argv[1] json_file   path to the JSON document
  argv[2] filter_expr dot path, e.g. ".nodes.validators" (leading dot optional)
  argv[3] default     printed when the path is missing or resolves to null

Output rules: booleans as "true"/"false"; lists space-joined (for bash arrays);
scalars as-is. Missing path / type error → prints the default. Always exits 0.
"""
import sys
import json

json_file = sys.argv[1]
filter_expr = sys.argv[2]   # e.g. ".nodes.validators"
default_val = sys.argv[3] if len(sys.argv) > 3 else ""

with open(json_file) as fh:
    data = json.load(fh)

# Navigate the dot-separated path (skip leading dot)
parts = [p for p in filter_expr.lstrip('.').split('.') if p]
node = data
try:
    for part in parts:
        node = node[part]
    if node is None:
        print(default_val)
    elif isinstance(node, bool):
        print(str(node).lower())
    elif isinstance(node, list):
        # Space-separated for bash arrays
        print(' '.join(str(x) for x in node))
    else:
        print(node)
except (KeyError, TypeError):
    print(default_val)
