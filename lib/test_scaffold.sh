#!/usr/bin/env bash
# lib/test_scaffold.sh - Generate test script skeleton from spec document
#
# Usage: source lib/test_scaffold.sh
#        _cb_test_scaffold <spec_doc> <rt_id> <output_dir>

[[ -n "${_CB_TEST_SCAFFOLD_LOADED:-}" ]] && return 0
readonly _CB_TEST_SCAFFOLD_LOADED=1

# _cb_test_scaffold <spec_doc> <rt_id> <output_dir>
# Parses the spec document for the given RT-ID, extracts scenario steps,
# and generates a test script skeleton with frontmatter and TODO markers.
_cb_test_scaffold() {
  local spec_doc="${1:?scaffold: spec document path required}"
  local rt_id="${2:?scaffold: RT-ID required}"
  local output_dir="${3:?scaffold: output directory required}"

  if [[ ! -f "$spec_doc" ]]; then
    echo "ERROR: spec document not found: $spec_doc" >&2
    return 1
  fi

  mkdir -p "$output_dir"

  python3 - "$spec_doc" "$rt_id" "$output_dir" <<'PYEOF'
import sys, re, os, json

spec_doc = sys.argv[1]
rt_id = sys.argv[2]
output_dir = sys.argv[3]

with open(spec_doc, encoding='utf-8') as f:
    content = f.read()

# Find the section
pattern = re.compile(
    r'^####\s+' + re.escape(rt_id) + r'\b\s*—?\s*([^\n]*)\n',
    re.MULTILINE
)
match = pattern.search(content)
if not match:
    print(f"ERROR: TC '{rt_id}' not found in spec", file=sys.stderr)
    sys.exit(1)

title = match.group(1).strip()

# Extract section body
end_pattern = re.compile(r'^(?:####\s|---\s*$)', re.MULTILINE)
end_match = end_pattern.search(content, match.end())
body = content[match.end():end_match.start() if end_match else len(content)].strip()

# Extract fields from table
fields = {}
for line in body.split('\n'):
    m = re.match(r'\|\s*\*\*(\w[\w\s]*)\*\*\s*\|\s*(.+?)\s*\|', line)
    if m:
        fields[m.group(1).strip()] = m.group(2).strip()

# Extract scenario steps (lines starting with - **Given/When/Then/And**)
scenario_lines = []
for line in body.split('\n'):
    m = re.match(r'-\s*\*\*(Given|When|Then|And)\*\*\s*(.*)', line)
    if m:
        scenario_lines.append((m.group(1), m.group(2).strip()))

# Derive filename from RT-ID
# RT-A-2-01 → a2-01, RT-B-01 → b-01
parts = rt_id.replace('RT-', '').split('-')
if len(parts) >= 4:
    # RT-A-2-01 → a2-01
    fname = f"{parts[0].lower()}{parts[1]}-{parts[2]}-{'-'.join(parts[3:])}"
elif len(parts) >= 3:
    fname = f"{parts[0].lower()}{parts[1]}-{parts[2]}"
elif len(parts) >= 2:
    fname = f"{parts[0].lower()}-{parts[1]}"
else:
    fname = parts[0].lower()

# Slugify title for filename
title_slug = re.sub(r'[^a-zA-Z0-9]+', '-', title.lower()).strip('-')[:30]
output_file = os.path.join(output_dir, f"{fname}-{title_slug}.sh")

# Infer category from ID section letter
section = parts[0].upper() if parts else 'X'
category_map = {
    'A': 'regression/a-ethereum',
    'B': 'regression/b-wbft',
    'C': 'regression/c-anzeon',
    'D': 'regression/d-fee-delegation',
    'E': 'regression/e-blacklist-authorized',
    'F': 'regression/f-system-contracts',
    'G': 'regression/g-api',
}
category = category_map.get(section, f'regression/{section.lower()}-unknown')

# Infer tags
tags = []
if section == 'A':
    sub = parts[1] if len(parts) > 1 else ''
    if sub == '1': tags.append('sync')
    elif sub == '2': tags.append('tx')
    elif sub == '3': tags.append('contract')
    elif sub == '4': tags.append('rpc')
elif section == 'B': tags.append('wbft')
elif section == 'C': tags.append('anzeon')
elif section == 'D': tags.append('fee-delegation')
elif section == 'E': tags.append('blacklist')
elif section == 'F': tags.append('governance')
elif section == 'G': tags.append('rpc')

prereqs = fields.get('선행 TC', 'none')
deps = []
if prereqs and prereqs.lower() != 'none' and prereqs != '—':
    deps = [p.strip() for p in re.split(r'[,;]', prereqs) if p.strip().startswith('RT-')]

tags_str = '[' + ', '.join(tags) + ']'
deps_str = '[' + ', '.join(deps) + ']'

# Build script
lines = []
lines.append('#!/usr/bin/env bash')
lines.append(f'# ---chainbench-meta---')
lines.append(f'# id: {rt_id}')
lines.append(f'# name: {title}')
lines.append(f'# category: {category}')
lines.append(f'# tags: {tags_str}')
lines.append(f'# estimated_seconds: 30')
lines.append(f'# preconditions:')
lines.append(f'#   chain_running: true')
lines.append(f'#   python_packages: [eth-account, requests, eth-utils]')
lines.append(f'# depends_on: {deps_str}')
lines.append(f'# ---end-meta---')
lines.append(f'# {rt_id} — {title}')
lines.append('set -euo pipefail')
lines.append('')
lines.append('source "$(dirname "$0")/../lib/common.sh"')
lines.append('')
test_name = f'{category}/{fname}-{title_slug}'.replace('//', '/')
lines.append(f'test_start "{test_name}"')
lines.append('check_env || { test_result; exit 1; }')
lines.append('')

# Generate scenario sections
current_phase = None
for step_type, step_text in scenario_lines:
    if step_type in ('Given', 'When'):
        if current_phase != step_type:
            lines.append(f'# {step_type}')
            current_phase = step_type
        lines.append(f'# TODO: {step_text}')
        lines.append('')
    elif step_type == 'Then':
        if current_phase != 'Then':
            lines.append('# Then')
            current_phase = 'Then'
        lines.append(f'# TODO: assert — {step_text}')
        lines.append('')
    elif step_type == 'And':
        lines.append(f'# TODO: {step_text}')
        lines.append('')

if not scenario_lines:
    lines.append('# TODO: implement test scenario')
    lines.append('')

lines.append('test_result')
lines.append('')

with open(output_file, 'w') as f:
    f.write('\n'.join(lines))

os.chmod(output_file, 0o755)
print(output_file)
PYEOF
}
