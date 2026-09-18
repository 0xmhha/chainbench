#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""ct_data.py와 분석 카탈로그로 Confluence 초안 페이지(Markdown)를 만든다."""
import json, re, sys, os, collections
sys.path.insert(0, os.path.dirname(__file__)); from ct_data import CT, AREA_NAME
OUT='/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260911'; D=f'{OUT}/confluence-draft'
cat=json.load(open(f'{OUT}/analyses/confluence-spec-catalog.json')); tc=json.load(open(f'{OUT}/analyses/tc-catalog.json'))
rows=json.load(open(f'{OUT}/sources/confluence/pr196-144-rows.json'))
mgmt={r['runtime_id']:r['test_id'] for r in rows if r['test_id'].startswith('PR196-TC')}
prio=collections.defaultdict(list)
for r in rows:
    if r['runtime_id']: prio[r['runtime_id']].append(r['section'][:2])
    for s in re.findall(r'(RT-[A-G]-[\dab-]+|TC-\d-\d-\d{2}|TX-\d{3}|NODE-\d{3}|RPC-\d{3}|BRIOCHE-\d{2})', r['test_id']): prio[s].append(r['section'][:2])
scope={o['spec_id'].split('@')[0]:o['common_scope'] for o in cat}
res={o['spec_id']:o['stablenet_result_20260413'] for o in cat if o['stablenet_result_20260413']}
SEP={'설정':'설정으로 분리','설정+기대값':'설정으로 분리, 기대값은 체인별 계산','별도 구현':'별도 구현 필요'}
SRC={'RT':'StableNet 회귀','TC':'StableNet 1차 변경','TS':'StableNet 1차 시나리오','TX':'WEMIX4.0','NODE':'WEMIX4.0','RPC':'WEMIX4.0','WBFT':'WEMIX4.0','MINING':'WEMIX3.0','N':'WEMIX3.0 PR196 신규','T':'2차 변경(StableNet·WEMIX4.0)'}
def src(s): return SRC.get(s.split('-')[0],'')
def remark(c):
    parts=[]
    for s in c['spec']:
        tag=' (부분 대응)' if scope.get(s) and scope[s]!='three-chain' else ''
        parts.append(f"{s}{tag}")
    a=[f"{x}{' ('+mgmt[x]+')' if x in mgmt else ''}" for x in c['auto']]
    p=sorted({q for k in c['spec']+c['auto'] for q in prio.get(k,[])})
    out=[]
    if parts: out.append('기존 명세: '+', '.join(parts))
    if a: out.append('자동 테스트: '+', '.join(a))
    if p: out.append('PR196 우선순위: '+'/'.join(p))
    if c['note']: out.append(c['note'])
    return ' / '.join(out)
def pr(c):
    p=sorted({q for k in c['spec']+c['auto'] for q in prio.get(k,[])}); return p[0] if p else '-'
# export json
json.dump([{**c,'remark':remark(c),'pr196_priority':pr(c)} for c in CT],open(f'{D}/ct-list.json','w'),indent=1,ensure_ascii=False)
# ---------- 01 list
n=collections.Counter(c['area'] for c in CT); sepc=collections.Counter(c['sep'] for c in CT)
L=['# 공통 테스트 목록','',
   f'세 체인(WEMIX3.0, WEMIX4.0, StableNet)에서 같은 목적으로 실행할 수 있는 테스트 {len(CT)}개다. 같은 목적의 기존 테스트는 하나로 합쳤고, 합친 원본은 비고에 모두 적었다. 분리 방식은 세 가지다. "설정으로 분리"는 실행 절차를 그대로 두고 체인별 값만 바꾸면 된다. "기대값은 체인별 계산"은 같은 입력에서 나와야 할 정답이 체인마다 달라 정답을 따로 계산해야 한다. "별도 구현 필요"는 실행 도구에 아직 없는 기능이 필요하거나 노드를 직접 다뤄야 한다.','',
   '## 테스트 ID 규칙','',
   '- 형식: `CT-<영역>-<순번 세 자리>` (예: `CT-TX-001`). CT는 세 체인 공통 테스트(Common Test)를 뜻한다.','- 영역: NODE(노드·동기화·네트워크), TX(트랜잭션 전송·거부), FEE(수수료·가스 정책), CONTRACT(컨트랙트 실행), RPC(조회·구독 API), FAULT(장애·복구)','- 순번은 영역마다 001부터 매기고 한 번 부여한 번호는 다시 쓰지 않는다. 테스트를 없애도 번호는 비워 둔다.','- 같은 목적의 기존 테스트가 여러 개면 CT ID는 하나만 만들고 비고에 기존 ID를 전부 적는다.','- 기존 ID의 출처: StableNet 회귀(RT-), StableNet 1차 변경(TC-, TS-), WEMIX4.0(NODE-, TX-, WBFT-, RPC- 세 자리), WEMIX3.0(MINING- 등), 2차 변경(T-), PR196 신규(N-). 자동 테스트는 실행 이름과 PR196 관리 번호(PR196-TC-)를 함께 적는다.','- 비고의 "(부분 대응)"은 기존 명세의 일부만 세 체인 공통이라는 뜻이다. 명세 그대로는 두 체인이나 한 체인에만 해당한다.','',
   '## 영역별 수','','| 영역 | 뜻 | 수 |','|---|---|---:|']
for a,name in AREA_NAME.items(): L.append(f'| {a} | {name} | {n[a]} |')
L+=['',f"분리 방식별: 설정으로 분리 {sepc['설정']}개, 기대값은 체인별 계산 {sepc['설정+기대값']}개, 별도 구현 필요 {sepc['별도 구현']}개.",'']
for a,name in AREA_NAME.items():
    L+=[f'## {a} ({name})','','| ID | 테스트 | 목적 | 기대 결과 | 분리 방식 | 우선순위 | 비고 |','|---|---|---|---|---|---|---|']
    for c in CT:
        if c['area']==a: L.append(f"| {c['id']} | {c['name']} | {c['purpose']} | {c['expected']} | {SEP[c['sep']]} | {pr(c)} | {remark(c)} |")
    L.append('')
open(f'{D}/01-common-test-list.md','w').write('\n'.join(L)+'\n')
# ---------- detail pages per area
DEP_KO={'계정':'테스트 계정','RPC':'RPC 주소·API 노출','컨트랙트':'컨트랙트','ChainID':'체인 ID','포크':'하드포크 활성','수수료':'수수료 값','토폴로지':'노드 구성·시간','바이너리':'노드 프로그램','프로세스':'노드 직접 제어'}
for i,(a,name) in enumerate(AREA_NAME.items()):
    L=[f'# 상세 실행 절차 {a} ({name})','',f'이 페이지는 {name} 영역 공통 테스트 {n[a]}개의 준비, 절차, 기대 결과, 체인별 차이를 적는다. 목록과 ID 규칙은 [공통 테스트 목록] 페이지에 있다. 세 체인의 값이 다른 곳은 "체인별 차이"에 적었고, 그 값은 실행 설정(프로필)에 둔다.','']
    for c in CT:
        if c['area']!=a: continue
        L+=[f"## {c['id']} · {c['name']}",'',f"- **목적**: {c['purpose']}",f"- **의존 요소**: {', '.join(DEP_KO[d] for d in c['deps'])}",f"- **분리 방식**: {SEP[c['sep']]}",'- **절차**:']
        L+=[f'    {k+1}. {s}' for k,s in enumerate(c['steps'])]
        L+=[f"- **기대 결과**: {c['expected']}"]
        if c['diff']: L.append(f"- **체인별 차이**: {c['diff']}")
        L.append(f"- **비고**: {remark(c) or '-'}")
        L.append('')
    open(f'{D}/1{i}-detail-{a}.md','w').write('\n'.join(L)+'\n')
# ---------- appendix: two-chain and excluded (from spec catalog + json catalog)
FAM_KO={
 'consensus-wbft':'WBFT 합의 정보(서명, 에폭, 라운드)를 검사한다. WEMIX3.0 합의에는 없다',
 'consensus-rpc-istanbul':'합의 정보 조회 API를 쓴다. WEMIX3.0에는 이 API가 없다',
 'modern-evm':'계정 코드 위임(EIP-7702) 또는 P256 서명 검증을 쓴다. WEMIX3.0은 지원하지 않고, 두 체인도 켜는 방법이 다르다',
 'reward':'Brioche 블록 보상을 검사한다. StableNet에는 블록 보상이 없다',
 'fee-policy':'기본 수수료 상한을 검사한다. WEMIX4.0에는 상한이 없다',
 'txpool':'트랜잭션 풀의 누적 잔액 검사를 겨냥한다. WEMIX3.0은 건별 검사만 확인됐다',
 'rpc-basics':'블록 조회 결과에 합의 정보가 포함되는지까지 본다',
 'stablenet-post-v1':'StableNet 1.0.0 이후 변경(발행 컨트랙트 개편, 하드포크 동시 적용, 계정 상태 저장)을 검사한다',
 'stablenet-system-contract':'StableNet 시스템 컨트랙트(코인 어댑터, 검증자, 발행, 위원회)를 검사한다',
 'stablenet-account-policy':'StableNet 계정 차단·인증 정책을 검사한다',
 'stablenet-anzeon-fee':'StableNet 헤더 팁 강제 규칙을 검사한다',
 'sync-lifecycle':'StableNet 내장 테스트넷 제네시스로 초기화한다',
 'wbft-governance':'WEMIX4.0 거버넌스 컨트랙트(스테이킹, NCP, 보상)를 검사한다',
 'wemix-etcd':'WEMIX3.0의 etcd 작업·토큰 키 복구를 검사한다',
 'wemix-mining':'WEMIX3.0 블록 생산 규칙(따라잡기, 중복 생산 제한)을 검사한다',
 'wemix-governance':'WEMIX3.0 거버넌스 멤버 변경과 노드 연동을 검사한다',
 'wemix-rpc':'WEMIX3.0 전용 조회 API와 헤더 필드를 검사한다',
 'wemix-pr196':'WEMIX3.0 PR196 수정의 크기 제한값(블록 8 MiB, 메시지 10 MiB)을 검사한다',
 'transition':'WEMIX3.0 데이터로 WEMIX4.0을 띄우는 전환 시나리오다',
 'unit-benchmark':'노드를 띄우지 않는 성능 측정이다','unit-test':'노드를 띄우지 않는 단위 테스트다','build':'빌드와 실행 파일 확인이다',
 'harness-vocabulary':'실행 도구 자체의 계산 기능 검사다',
 'fee-observation':'수수료 값 관측이며 헤더 팁 규칙 부분이 StableNet 전용이다',
}
def why(o):
    sid=o['spec_id'].split('@')[0]
    if sid in ('N-001','N-002','N-003','N-004','N-005','N-006','N-007','N-016'): return 'WEMIX3.0 PR196 신규 테스트 가운데 집중 목록에서 뺀 항목이다'
    if sid=='RT-A-2-05b': return 'StableNet의 최대 수수료 하한(최소 기본 수수료와 최소 팁의 합) 검사다. 다른 두 체인의 풀에는 이 검사가 없다'
    return FAM_KO.get(o['family'], o['reason'])
def gen(o): return (o['generic_subset'] or '').replace('three-chain:','세 체인 공통인 부분:').replace('profile','프로필')
SC={'two-chain:wbft+stablenet':'WEMIX4.0 + StableNet','two-chain:wemix+wbft':'WEMIX3.0 + WEMIX4.0','two-chain:wemix+stablenet':'WEMIX3.0 + StableNet'}
real=[o for o in cat if not o['spec_id'].startswith('PR196')]
L=['# 부록 A. 두 체인에서만 가능한 테스트','','세 체인 중 두 체인에서만 같은 목적으로 실행할 수 있는 기존 테스트다. 새 ID는 부여하지 않고 기존 ID로 적는다. 대부분 WEMIX4.0과 StableNet이 같은 합의 방식(WBFT)을 쓰기 때문에 생기는 묶음이다.','']
for k,label in SC.items():
    items=[o for o in real if o['common_scope']==k]
    if not items: continue
    L+=[f'## {label} ({len(items)}개)','','| 기존 ID | 출처 | 내용 | 비고 |','|---|---|---|---|']
    for o in items:
        L.append(f"| {o['spec_id'].replace('@stablenet',' (StableNet)').replace('@wemix4',' (WEMIX4.0)')} | {', '.join([t for t in o['page_titles'] if 'PR196' not in t] or o['page_titles'])} | {why(o)} | {gen(o)} |")
    L.append('')
jl=[t for t in tc if t['common_scope'].startswith('two-chain')]
L+=['## 두 체인용 자동 테스트','','| 실행 ID | 파일 | 범위 | 내용 |','|---|---|---|---|']
for t in jl: L.append(f"| {t['id']} | {t['file'].replace('tests/tc/','')} | {SC[t['common_scope']]} | {FAM_KO.get(t['family'], t['reason'])} |")
open(f'{D}/30-appendix-two-chain.md','w').write('\n'.join(L)+'\n')
EX={'stablenet-only':'StableNet 전용','wbft-only':'WEMIX4.0 전용','wemix-only':'WEMIX3.0 전용','transition-only':'WEMIX3.0에서 WEMIX4.0으로 넘어가는 전환 시나리오','excluded-unit-or-build':'노드를 띄우는 테스트가 아님(단위 테스트·빌드 확인)','harness-only':'실행 도구 자체 검사(체인 무관)'}
L=['# 부록 B. 공통에서 제외한 테스트와 이유','','한 체인에만 있는 기능이거나 노드 실행 테스트가 아닌 항목이다. ID는 기존 것을 그대로 적는다.','']
for k,label in EX.items():
    items=[o for o in real if o['common_scope']==k]
    if items:
        L+=[f'## {label} ({len(items)}개)','','| 기존 ID | 출처 | 이유 |','|---|---|---|']
        for o in items: L.append(f"| {o['spec_id'].replace('@stablenet',' (StableNet)').replace('@wemix4',' (WEMIX4.0)')} | {', '.join([t for t in o['page_titles'] if 'PR196' not in t] or o['page_titles'])} | {why(o)} |")
        L.append('')
jl=[t for t in tc if t['common_scope'] in EX]
L+=['## 제외한 자동 테스트','','| 실행 ID | 파일 | 구분 | 이유 |','|---|---|---|---|']
for t in jl: L.append(f"| {t['id']} | {t['file'].replace('tests/tc/','')} | {EX[t['common_scope']]} | {FAM_KO.get(t['family'], t['reason'])} |")
open(f'{D}/31-appendix-excluded.md','w').write('\n'.join(L)+'\n')
print('rendered', len(CT))
