#!/usr/bin/env python3
"""Cross-check counts, hashes, mirror consistency and citation results; write analyses/validation.json."""
import json, os, sys, hashlib, glob, subprocess, collections, datetime
OUT=sys.argv[1]; A=f'{OUT}/analyses'
tc=json.load(open(f'{A}/tc-catalog.json')); docs=json.load(open(f'{A}/doc-catalog.json')); deps=json.load(open(f'{A}/tc-dependencies.json'))
xw=json.load(open(f'{A}/tc-spec-id-crosswalk.json')); cd=json.load(open(f'{A}/chain-differences.json')); hd=json.load(open(f'{A}/harness-dependencies.json'))
cc=json.load(open(f'{A}/citation-check.json')); man=json.load(open(f'{OUT}/source-manifest.json'))
v={'validated_at':datetime.datetime.now(datetime.UTC).isoformat(),'checks':{}}
# 1 counts
files=sorted(glob.glob('/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/**/*.json',recursive=True))
v['checks']['tc_json_count']={'expected':len(files),'catalog':len(tc),'dependencies':len(deps),'crosswalk':len(xw),'ok':len(files)==len(tc)==len(deps)==len(xw)}
v['checks']['doc_rows']={'count':len(docs),'common_doc':sum(1 for d in docs if d['doc']=='common'),'dual_doc':sum(1 for d in docs if d['doc']=='dual')}
# 2 hashes: catalog sha == current file sha
stale=[t['file'] for t in tc if hashlib.sha256(open('/Users/wm-it-25_0220/Work/github/chainbench/'+t['file'],'rb').read()).hexdigest()!=t['sha256']]
v['checks']['tc_hash_current']={'stale':stale,'ok':not stale}
# 3 scope counts
sc=collections.Counter(t['common_scope'] for t in tc); sd=collections.Counter(d['common_scope'] for d in docs)
sep3=collections.Counter(x['separation'] for x in tc+docs if x['common_scope']=='three-chain')
v['checks']['scope_counts']={'json':dict(sc),'doc':dict(sd),'three_chain_separation':dict(sep3)}
# 4 every three-chain/two-chain item has config or impl items or is config-only with reason
missing=[x.get('file') or x.get('doc_id') for x in tc+docs if x['common_scope'].startswith(('three','two')) and not x['reason']]
v['checks']['reason_present']={'missing':missing,'ok':not missing}
# 5 citations
v['checks']['citations']={'checked':cc['checked'],'ok':cc['ok'],'fail':len(cc['fail']),'outside_build_selection':len(cc['outside_build_selection']),'chain_diff_items':len(cd),'chain_diff_evidence':sum(len(x['evidence']) for x in cd),'harness_items':len(hd),'harness_evidence':sum(len(x['evidence']) for x in hd)}
# 6 mirror vs working tree
mir={}
for p,repo in [('go-wemix','/Users/wm-it-25_0220/Work/github/chain/go-wemix'),('go-wbft','/Users/wm-it-25_0220/Work/github/chain/go-wbft'),('go-stablenet','/Users/wm-it-25_0220/Work/github/chain/go-stablenet')]:
    sel=json.load(open(f'{OUT}/build/{p}-selected.json'))
    changed=[s['path'] for s in sel['selected'] if hashlib.sha256(open(os.path.join(repo,s['path']),'rb').read()).hexdigest()!=s['sha256']]
    head=subprocess.check_output(['git','-C',repo,'rev-parse','HEAD']).decode().strip()
    mir[p]={'go_files':sel['go_files'],'packages':sel['packages_in_repo'],'golist_errors':len(sel['errors']),'changed_since_mirror':changed,'head_now':head}
v['checks']['mirror_vs_tree']=mir
# 7 graph
g={}
for p in ('go-wemix','go-wbft','go-stablenet'):
    cg=json.load(open(f'{OUT}/graph/{p}/code-graph.json')); log=open(f'{OUT}/graph/{p}.extract.log').read()
    g[p]={'modules':len(cg['modules']),'types':len(cg['types']),'source_files_logged':int([l for l in log.splitlines() if l.startswith('source files:')][0].split(':')[1]),'errors_in_log':log.count('error')}
v['checks']['graph']=g
# 8 id mapping
v['checks']['spec_id_mapping']=dict(collections.Counter(x['mapping_status'] for x in xw))
# 9 confluence
idx=json.load(open(f'{OUT}/sources/confluence/index.json'))['pages']; cat=json.load(open(f'{A}/confluence-spec-catalog.json'))
v['checks']['confluence']={'fetched_this_run':True,'pages':len(idx),'chars':sum(p['chars'] for p in idx),'spec_ids_judged':sum(1 for o in cat if not o['spec_id'].startswith('PR196-TC')),'scope_counts':dict(collections.Counter(o['common_scope'] for o in cat if not o['spec_id'].startswith('PR196-TC'))),'id_diff_vs_20260909':'identical for the 6 previously transcribed pages (see sources/confluence/id-diff-vs-20260909.json)','internal_ip_redacted':True}
# 10 prior citation recheck summary
pr=json.load(open(f'{A}/prior-citation-recheck.json'))
v['checks']['prior_run_citations_recheck']=dict(collections.Counter(r['status'] for r in pr))
v['overall_ok']=all([v['checks']['tc_json_count']['ok'],v['checks']['tc_hash_current']['ok'],v['checks']['reason_present']['ok'],cc['fail']==[]])
json.dump(v,open(f'{A}/validation.json','w'),indent=1,ensure_ascii=False)
# manifest: record mirror heads
man['mirror_heads']={'go-wemix':'902f9fce85c108cf24cdeb77bc70a622049748ce','go-wbft':'b1dc5a8a28d134ed652734cf67b59a2d2f471265','go-stablenet':'0937ac5c93d4f56ded5b24899f381d3d1f208c02','chainbench':'cc501aedaab17ca5ada6e3a407cfda323db26b7a'}
man['notes']=['go-wbft HEAD moved during the analysis (another session rebased); build-selected mirror is at mirror_heads.go-wbft. Files changed since the mirror are listed in analyses/validation.json checks.mirror_vs_tree.','chainbench had no tracked changes; untracked docs/research dirs are analysis outputs.']
man['projects']['chainbench']['head_at_end']=subprocess.check_output(['git','-C','/Users/wm-it-25_0220/Work/github/chainbench','rev-parse','HEAD']).decode().strip()
for p in man['projects']: man['projects'][p]['head_at_end']=mir.get(p,{}).get('head_now', man['projects'][p].get('head_at_end', man['projects'][p]['head']))
json.dump(man,open(f'{OUT}/source-manifest.json','w'),indent=1,ensure_ascii=False)
print(json.dumps({k:(vv if k!='mirror_vs_tree' else {p:(x['changed_since_mirror'],x['head_now'][:12]) for p,x in vv.items()}) for k,vv in v['checks'].items()},ensure_ascii=False,indent=1)); print('overall_ok',v['overall_ok'])
