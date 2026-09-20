#!/usr/bin/env python3
"""Parse the two scenario Markdown tables into records."""
import re, json, sys, hashlib, os
OUT=sys.argv[1]
def parse(path, kind):
    rows=[]; section=None
    lines=open(path,encoding='utf-8').read().split('\n')
    for ln,line in enumerate(lines,1):
        if line.startswith('## ') or line.startswith('### '):
            section=line.strip('# ').strip()
        if line.startswith('|') and not re.match(r'^\|\s*-', line) and '카테고리' not in line:
            cells=[c.strip() for c in line.strip().strip('|').split('|')]
            if len(cells)<6: continue
            cat, ids, title, purpose, flow, expected = cells[:6]
            id_list=re.findall(r'`([A-Z]+-[A-Z0-9-]+(?:~\d+)?)`', ids)
            scripts=re.findall(r'\(`([^`]+\.sh)`\)', ids)
            chains=re.findall(r'\*\*(StableNet|WEMIX4|WEMIX3\.0)\*\*', ids)
            no_script=[m for m in re.findall(r'\*\*(StableNet|WEMIX4|WEMIX3\.0)\*\*: \(([^)]*)\)', ids)]
            rows.append({'doc':kind,'source_file':os.path.relpath(path,OUT),'source_line':ln,'section':section,'category':cat.strip('*'),
                         'spec_ids':id_list,'scripts':scripts,'chains_named':chains,'no_script_notes':no_script,
                         'title':re.sub(r'`','',title),'purpose':purpose,'flow':flow,'expected':expected})
    return rows
rows=parse(os.path.join(OUT,'sources/common_test_scenarios.md'),'common')+parse(os.path.join(OUT,'sources/dual_chain_test_scenarios.md'),'dual')
for i,r in enumerate(rows,1): r['doc_id']=f"DOC-{'C' if r['doc']=='common' else 'D'}-{i if r['doc']=='common' else i-sum(1 for x in rows if x['doc']=='common'):03d}"
json.dump(rows,open(os.path.join(OUT,'analyses','scenario-doc-rows.json'),'w'),indent=1,ensure_ascii=False)
from collections import Counter
print(len(rows), Counter(r['doc'] for r in rows), Counter(r['section'] for r in rows))
for r in rows: print(r['doc_id'], r['section'][:18], r['spec_ids'], r['title'][:40])
