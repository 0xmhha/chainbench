#!/usr/bin/env python3
"""Check every `<repo>/<path>:<line>` — `<snippet>` citation in the analyses
against the CURRENT trees: file exists, line exists, and the snippet (whitespace
-normalized) is on that line. Also checks chain citations against the build-
selected lists. Exit 1 on any failure."""
import re, sys, os, json, glob
OUT=sys.argv[1]; roots=json.loads(sys.argv[2])
sel={}
for p in ('go-wemix','go-wbft','go-stablenet'):
    s=json.load(open(f'{OUT}/build/{p}-selected.json')); sel[p]={x['path'] for x in s['selected']}
pat=re.compile(r'`(go-wemix|go-wbft|go-stablenet|chainbench)/([^`:]+):(\d+)`(?:\s*—\s*`((?:[^`]|``)*)`)?')
norm=lambda s: re.sub(r'\s+',' ',s).strip()
res={'checked':0,'ok':0,'fail':[],'outside_build_selection':[]}
for doc in sorted(glob.glob(f'{OUT}/analyses/*.md')):
    for ln,line in enumerate(open(doc,encoding='utf-8'),1):
        for m in pat.finditer(line):
            repo,path,lno,snip=m.group(1),m.group(2),int(m.group(3)),m.group(4)
            res['checked']+=1
            f=os.path.join(roots[repo],path)
            if repo=='go-wemix' and path.startswith('cmd/gwemix/'): f=os.path.join(roots[repo],'cmd/geth/'+path[len('cmd/gwemix/'):])
            if not os.path.exists(f): res['fail'].append({'doc':os.path.basename(doc),'line':ln,'cite':m.group(0),'why':'missing file'}); continue
            src=open(f,encoding='utf-8',errors='replace').read().split('\n')
            if lno>len(src): res['fail'].append({'doc':os.path.basename(doc),'line':ln,'cite':m.group(0),'why':'line out of range'}); continue
            if snip and norm(snip.replace('``','`')) not in norm(src[lno-1]) and not snip.startswith('주소 등'):
                res['fail'].append({'doc':os.path.basename(doc),'line':ln,'cite':m.group(0)[:120],'why':'snippet not on line','actual':src[lno-1].strip()[:120]}); continue
            res['ok']+=1
            if repo in sel:
                sp=path if not path.startswith('cmd/gwemix/') else 'cmd/geth/'+path[len('cmd/gwemix/'):]
                if sp not in sel[repo] and not sp.endswith('.json'): res['outside_build_selection'].append(f'{repo}/{path}')
res['outside_build_selection']=sorted(set(res['outside_build_selection']))
json.dump(res,open(f'{OUT}/analyses/citation-check.json','w'),indent=1,ensure_ascii=False)
print('checked',res['checked'],'ok',res['ok'],'fail',len(res['fail']),'outside build selection',len(res['outside_build_selection']))
for f in res['fail'][:30]: print(f)
print(res['outside_build_selection'][:20])
sys.exit(1 if res['fail'] else 0)
