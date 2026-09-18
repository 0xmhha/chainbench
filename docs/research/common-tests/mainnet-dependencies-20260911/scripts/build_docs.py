#!/usr/bin/env python3
"""Render the data-driven analysis documents from the JSON catalogs."""
import json, os, sys, collections
OUT=sys.argv[1]; A=os.path.join(OUT,'analyses')
tc=json.load(open(f'{A}/tc-catalog.json')); docs=json.load(open(f'{A}/doc-catalog.json'))
deps=json.load(open(f'{A}/tc-dependencies.json')); xw={x['file']:x for x in json.load(open(f'{A}/tc-spec-id-crosswalk.json'))}
prior=json.load(open('/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260909/analyses/all-tests.json'))
ph={p['source_file'].replace('sources/chainbench/',''):p for p in prior if p['source_kind']=='json_case'}
pd={p['source_line']:p for p in prior if p['source_kind']=='document' and 'common_' in p['source_file']}
pdd={p['source_line']:p for p in prior if p['source_kind']=='document' and 'dual_' in p['source_file']}
SCOPE_KO={'three-chain':'세 체인 공통','two-chain:wbft+stablenet':'2체인(WEMIX4.0+StableNet)','two-chain:wemix+wbft':'2체인(WEMIX3.0+WEMIX4.0)','two-chain:wemix+stablenet':'2체인(WEMIX3.0+StableNet)','stablenet-only':'StableNet 전용','wbft-only':'WEMIX4.0 전용','wemix-only':'WEMIX3.0 전용','transition-only':'전환 전용','harness-only':'하네스 전용(체인 무관)'}
SEP_KO={'config-only':'설정 분리','config+oracle':'설정 분리 + 체인별 기대값','adapter':'별도 구현/처리','not-common':'공통 아님'}
def tcid(t): return f"TC-{tc.index(t)+1:03d}"
def link(f): return f"[{f.replace('tests/tc/','')}](../../../../../{f})"
# ---------- all-tests.md
L=['# 전체 테스트 목록 (2026-09-11 현재 파일 기준)','',
   f'JSON 케이스 {len(tc)}개와 명세 문서 행 {len(docs)}개, 총 {len(tc)+len(docs)}개를 현재 내용으로 다시 읽고 판정했다. 판정 값의 뜻은 [README](README.md)에 있다. 명세 ID 대응은 [ID 대응표](existing-tc-specs.md)를 본다.','',
   '## JSON 케이스 (tests/tc)','','| 분석 ID | 파일 | 실행 ID | 명세 ID | 작성 체인 | applicableChains | requires | family | 공통 범위 | 분리 방식 | 판정 이유 |','|---|---|---|---|---|---|---|---|---|---|---|']
for t in tc:
    L.append(f"| {tcid(t)} | {link(t['file'])} | {t['id']} | {xw[t['file']]['display_id']} | {t['source_chain']} | {t['applicableChains'] or '-'} | {','.join(t['requires'])} | {t['family']} | {SCOPE_KO[t['common_scope']]} | {SEP_KO[t['separation']]} | {t['reason']} |")
L+=['','## 명세 문서 행','','| 분석 ID | 문서 | 절 | 명세 ID | 제목 | family | 공통 범위 | 분리 방식 | 판정 이유 |','|---|---|---|---|---|---|---|---|---|']
for d in docs:
    L.append(f"| {d['doc_id']} | [{d['source_file'].split('/')[-1]} {d['source_line']}행](../{d['source_file']}) | {d['section'][:28]} | {', '.join(d['spec_ids']) or '-'} | {d['title']} | {d['family']} | {SCOPE_KO[d['common_scope']]} | {SEP_KO[d['separation']]} | {d['reason']} |")
open(f'{A}/all-tests.md','w').write('\n'.join(L)+'\n')
# ---------- common-candidates.md
def sec(title, items, kind):
    out=[f'## {title}','',f'{len(items)}개.','']
    if not items: return out+['없음','']
    out+=['| 분석 ID | 대상 | 명세 ID | family | 분리 방식 | 설정으로 분리할 항목 | 별도 구현·처리 |','|---|---|---|---|---|---|---|']
    for x in items:
        if kind=='tc': out.append(f"| {tcid(x)} | {link(x['file'])} | {xw[x['file']]['display_id']} | {x['family']} | {SEP_KO[x['separation']]} | {'; '.join(x['configuration_items']) or '-'} | {'; '.join(x['implementation_items']) or '-'} |")
        else: out.append(f"| {x['doc_id']} | {x['title']} | {', '.join(x['spec_ids']) or '-'} | {x['family']} | {SEP_KO[x['separation']]} | {'; '.join(x['configuration_items']) or '-'} | {'; '.join(x['implementation_items']) or '-'} |")
    return out+['']
c3=[t for t in tc if t['common_scope']=='three-chain']; d3=[d for d in docs if d['common_scope']=='three-chain']
c2=[t for t in tc if t['common_scope'].startswith('two-chain')]; d2=[d for d in docs if d['common_scope'].startswith('two-chain')]
cx=[t for t in tc if t['common_scope'] not in ('three-chain',) and not t['common_scope'].startswith('two-chain')]; dx=[d for d in docs if d['common_scope']=='stablenet-only']
cnt=collections.Counter(t['separation'] for t in c3+d3)
L=['# 공통 수행 후보 (2026-09-11)','',
   f'세 체인 공통 후보 {len(c3)+len(d3)}개(JSON {len(c3)}, 문서 {len(d3)}). 이 중 설정만 분리하면 되는 것 {cnt["config-only"]}개, 설정 분리에 체인별 기대값 oracle이 더 필요한 것 {cnt["config+oracle"]}개, 별도 구현이나 소유 환경이 필요한 것 {cnt["adapter"]}개. 2체인 후보 {len(c2)+len(d2)}개, 공통 제외 {len(cx)+len(dx)}개.','',
   '후보 수는 원본 항목 수다. 같은 기능의 문서 행과 JSON, 그리고 이미 세 체인으로 이식된 사본(insufficient-funds, revert-status-zero)이 각각 세어진다. 공개 RPC에서 바로 실행 가능하다는 뜻이 아니다. 별도 구현 항목의 괄호 ID(CD-A, CD-B, H)는 [클라이언트 차이](chain-differences.md)와 [하네스 분석](harness-dependencies.md)의 항목이다.','']
L+=sec('1. 세 체인 공통 후보 (JSON)',c3,'tc')+sec('2. 세 체인 공통 후보 (명세 문서)',d3,'doc')+sec('3. 2체인 후보 (JSON)',c2,'tc')+sec('4. 2체인 후보 (명세 문서)',d2,'doc')
L+=['## 5. 공통 제외','',f'JSON {len(cx)}개, 문서 {len(dx)}개.','','| 분석 ID | 대상 | 범위 | 이유 |','|---|---|---|---|']
for t in cx: L.append(f"| {tcid(t)} | {link(t['file'])} | {SCOPE_KO[t['common_scope']]} | {t['reason']} |")
for d in dx: L.append(f"| {d['doc_id']} | {d['title']} | {SCOPE_KO[d['common_scope']]} | {d['reason']} |")
open(f'{A}/common-candidates.md','w').write('\n'.join(L)+'\n')
# ---------- dependency-index.md
CAT_KO={'accounts':'테스트 계정','rpc':'RPC Endpoint·namespace·capability','contracts':'컨트랙트 주소·calldata','chain_id':'Chain ID·client family','forks':'포크·합의 전용 RPC','fees':'수수료·가스 입력','topology':'토폴로지·시간·프로세스 제어','binary':'바이너리'}
L=['# 메인넷 의존 필드 색인 (tests/tc, 2026-09-11)','',
   '`scripts/extract_tc_dependencies.py`가 현재 195개 JSON에서 뽑은 필드다. 값은 테스트 fixture이므로 그대로 적었다. 개인 키처럼 보이는 값은 없었다(있으면 redacted 표기). 각 항목의 전체 목록은 [tc-dependencies.json](tc-dependencies.json)에 있다.','',
   '## 요약','','| 분류 | 필드 수 | 파일 수 | 값 종류 |','|---|---:|---:|---|']
for c in CAT_KO:
    rows=[(r['file'],x) for r in deps for x in r['dependencies'][c]]
    files=len({f for f,_ in rows}); kinds=collections.Counter(x['value_kind'] for _,x in rows)
    L.append(f"| {CAT_KO[c]} | {len(rows)} | {files} | {', '.join(f'{k} {v}' for k,v in kinds.most_common())} |")
L+=['','## 하드코딩된 주소 literal','','| 주소 | 파일 수 | 의미 | 처리 |','|---|---:|---|---|']
addr=collections.Counter(); meaning={}
for r in deps:
    for c in ('accounts','contracts'):
        for x in r['dependencies'][c]:
            if x['value_kind']=='literal-address':
                addr[x['value']]+=0; meaning.setdefault(x['value'],x['meaning'])
    for a in set(x['value'] for c in ('accounts','contracts') for x in r['dependencies'][c] if x['value_kind']=='literal-address'): addr[a]+=1
KNOWN={'0xc17d493883eaa3b4cceb0f214b273392d562f9d8':'preset 키셋 node1 계정(node-signed sender). 계정 label로 치환',
       '0x70997970C51812dc3A010C7d01b50e0d17dc79C8':'수신용 고정 주소. label 또는 newAccount binding으로',
       '0x1111111111111111111111111111111111111111':'EIP-7702 delegate 대상·createAddress 입력 상수. 그대로 둔다',
       '0x0000000000000000000000000000000000000000':'zero address. 그대로 둔다',
       '0x5400d8b543eaf6738c7b44799623bea88fd0f5ee':'GovValidator 추가 대상 validator 주소(StableNet 전용)'}
for a,n in addr.most_common():
    m=meaning[a]; how=KNOWN.get(a) or ('StableNet 시스템 계약. 체인 adapter' if m.startswith('SYSTEM') else ('수신용 sink 주소(C0FFEE..). 프로필 recipient 또는 그대로' if 'C0FFEE' in a.upper() else '검토'))
    L.append(f"| `{a}` | {n} | {m} | {how} |")
L+=['','## 분류별 필드 목록','']
for c in CAT_KO:
    L+=[f'### {CAT_KO[c]}','','| 파일 | 필드 | 값 종류 | 값 | 의미 |','|---|---|---|---|---|']
    for r in deps:
        for x in r['dependencies'][c]:
            if x['field'].startswith('env.topology') and c=='topology' and x['field'].count('.')>2: continue
            L.append(f"| {r['file'].replace('tests/tc/','')} | `{x['field']}` | {x['value_kind']} | `{str(x['value'])[:60]}` | {x['meaning']} |")
    L.append('')
open(f'{A}/dependency-index.md','w').write('\n'.join(L)+'\n')
# ---------- existing-tc-specs.md
L=['# 기존 명세 ID 대응표 (2026-09-11)','',
   '실행 ID(JSON id)와 Confluence 명세 ID의 대응이다. 파일 해시가 2026-09-09 보존본과 같은 187개와 description만 바뀐 2개는 그때의 대응을 출발점으로 쓰고, 신규 6개는 이번에 대응했다. 2026-09-11에 Confluence Chainbench 폴더 페이지 29개를 직접 읽어(`../sources/confluence/`) 대응된 명세 ID 전부가 실제 페이지에 있는지 확인했다(상태 `confluence-verified`, 부분 대응은 `confluence-verified-partial`). 대응은 명세 전체를 구현했다는 뜻이 아니며, `[부분 대응]`은 명세 흐름의 일부만 검증한다는 뜻이다. 명세 ID 단위의 판정은 [Confluence 명세 ID 판정](confluence-spec-catalog.md)에 있다.','',
   '| 분석 ID | 파일 | 실행 ID | 명세 ID | 상태 | 근거 |','|---|---|---|---|---|---|']
for t in tc:
    x=xw[t['file']]; L.append(f"| {tcid(t)} | {t['file'].replace('tests/tc/','')} | {t['id']} | {x['display_id']} | {x['mapping_status']} | {x['basis']} |")
cnt=collections.Counter(xw[t['file']]['mapping_status'] for t in tc)
L+=['',f"상태별: {', '.join(f'{k} {v}건' for k,v in cnt.items())}. unconfirmed 43건은 명세 ID가 없는 하네스·기본 케이스이며 PR196 144 목록이 그중 37건에 PR196-TC 관리 ID를 부여했다."]
open(f'{A}/existing-tc-specs.md','w').write('\n'.join(L)+'\n')
# ---------- counter-review vs prior classification
L=['# 2026-09-09 판정과의 차이 (반대 검토)','',
   '같은 파일에 대한 이전 판정(commonality: exact-config / adapter-required / chain-specific)과 이번 판정(공통 범위 + 분리 방식)을 나란히 놓았다. 이번 판정이 정답이라는 뜻이 아니라, 달라진 곳을 드러내 검토하기 위한 표다. 이전 자료는 이전 코드 기준이므로 승계하지 않았다.','',
   '| 분석 ID | 파일 | 이전 | 이번 | 차이 이유 |','|---|---|---|---|---|']
MAP={'exact-config':'three-chain/config-only','adapter-required':'three-chain/config+oracle|adapter','chain-specific':'not three-chain'}
diffs=0
for t in tc:
    p=ph.get(t['file'])
    if not p: L.append(f"| {tcid(t)} | {t['file'].replace('tests/tc/','')} | (신규) | {SCOPE_KO[t['common_scope']]} / {SEP_KO[t['separation']]} | 신규 파일 |"); continue
    prev=p['commonality']; now=t['common_scope']; sep=t['separation']
    agree = (prev=='exact-config' and now=='three-chain' and sep=='config-only') or (prev=='adapter-required' and now=='three-chain' and sep in ('config+oracle','adapter')) or (prev=='chain-specific' and now!='three-chain')
    if not agree:
        diffs+=1; L.append(f"| {tcid(t)} | {t['file'].replace('tests/tc/','')} | {prev} | {SCOPE_KO[now]} / {SEP_KO[sep]} | {t['reason']} |")
L+=['',f'판정이 달라진 파일 {diffs}개(신규 6개 제외).']
open(f'{A}/counter-review.md','w').write('\n'.join(L)+'\n')
print('docs built; diffs', diffs)
