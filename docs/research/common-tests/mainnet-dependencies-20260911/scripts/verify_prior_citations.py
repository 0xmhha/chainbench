#!/usr/bin/env python3
"""Re-check every `sources/<project>/<path>:<line>` citation of the 2026-09-09 run
against the CURRENT working trees. Reports: same-line match, moved (snippet found
at another line), or missing (snippet not found)."""
import re, sys, json, os
roots = {'go-wemix': '/Users/wm-it-25_0220/Work/github/chain/go-wemix',
         'go-wbft': '/Users/wm-it-25_0220/Work/github/chain/go-wbft',
         'go-stablenet': '/Users/wm-it-25_0220/Work/github/chain/go-stablenet',
         'chainbench': '/Users/wm-it-25_0220/Work/github/chainbench'}
pat = re.compile(r'`sources/([^/`]+)/([^`:]+):(\d+)`(?:\s*—\s*`([^`]*)`)?')
rows = []
for doc in sys.argv[1:]:
    for ln, line in enumerate(open(doc), 1):
        for m in pat.finditer(line):
            proj, path, lno, snip = m.group(1), m.group(2), int(m.group(3)), m.group(4)
            f = os.path.join(roots[proj], path)
            status, newline = 'missing-file', None
            if os.path.exists(f):
                src = open(f, errors='replace').read().split('\n')
                if snip and snip not in ('주소 등 literal 필드 존재(값 생략)',):
                    hits = [i+1 for i, l in enumerate(src) if snip.strip() in l]
                    if lno in hits: status, newline = 'same', lno
                    elif hits: status, newline = 'moved', hits[0]
                    else: status = 'snippet-missing'
                else:
                    status, newline = ('line-exists' if lno <= len(src) else 'line-out-of-range'), lno
            rows.append({'doc': os.path.basename(doc), 'doc_line': ln, 'project': proj, 'path': path, 'line': lno, 'snippet': snip, 'status': status, 'current_line': newline})
json.dump(rows, open(os.path.join(os.environ['OUT'], 'analyses', 'prior-citation-recheck.json'), 'w'), indent=1, ensure_ascii=False)
from collections import Counter
print(Counter((r['project'], r['status']) for r in rows))
for r in rows:
    if r['status'] not in ('same', 'line-exists'):
        print(r['doc'], r['project'], r['path'], r['line'], '->', r['status'], r['current_line'], (r['snippet'] or '')[:60])
