#!/usr/bin/env python3
"""Spec-ID-level catalog of every test case ID found on the Confluence Chainbench
folder pages (fetched 2026-09-11), judged against current client code. Each ID is
classified AS SPECIFIED (its own expected result); where a generic subset is
three-chain, that is noted separately. Links each ID to the JSON tests
(tc-spec-id-crosswalk) and scenario-doc rows that reference it, the PR196 144-list
priority, and the 2026-04-13 StableNet execution result."""
import json, re, os, sys, collections
OUT=sys.argv[1]; C=f'{OUT}/sources/confluence'; A=f'{OUT}/analyses'
idx=json.load(open(f'{C}/index.json'))['pages']
pages={p['id']:p for p in idx}
SPEC_PAGES={'2599813145':'Regression Test Case','2599256076':'Regression Test Case with scenario','2614231162':'1st Test Cases (v1.0.0+)','2599289086':'1st Test Scenarios','2636711400':'[WEMIX4.0] Test','2639659344':'[WEMIX 4.0] 테스트 시나리오','2918252564':'[WEMIX3.0] 테스트 시나리오','2874802346':'2nd Change Test Cases (StableNet)','2879193122':'[WEMIX 4.0] 2nd Change Test Cases','2979070094':'[PR196] 집중 테스트 28개 상세 실행 절차','2978054308':'[PR196] 기존 테스트 전체 144개 우선순위 목록'}
FRAG={'RT-F-3','RT-G-1','RT-A-1','RT-A-2','RT-A-3','RT-G-4','RT-G-2','RT-G-3','RT-G-5','RT-F-1','RT-F-2','RT-F-4','RT-F-5'}
# universe: id -> set of page ids it appears on (spec pages only); T-x namespaced per page
uni=collections.defaultdict(set)
for pid,name in SPEC_PAGES.items():
    for i in pages[pid]['ids_found']:
        if i in FRAG: continue
        key = f'{i}@{"stablenet" if pid=="2874802346" else "wemix4"}' if re.fullmatch(r'T-\d',i) else i
        uni[key].add(pid)
# PR196 management ids from the 144 list rows
rows=json.load(open(f'{C}/pr196-144-rows.json'))
prio={}; mgmt={}
for r in rows:
    for t in re.findall(r'(PR196-TC-\d{3}|RT-[A-G]-[\dab-]+|TC-\d-\d-\d{2}|TX-\d{3}|NODE-\d{3}|RPC-\d{3}|BRIOCHE-\d{2})', r['test_id']):
        prio.setdefault(t, r['section'].split(' ')[0])
    if r['test_id'].startswith('PR196-TC') and r['runtime_id']: mgmt[r['test_id']]=r['runtime_id']; uni[r['test_id']].add('2978054308')
# 2026-04-13 results
res={}
for line in open(f'{C}/2611052714-2026-04-13-test.md',encoding='utf-8'):
    m=re.match(r'\|\s*(TC-\d-\d-\d{2})\s*\|.*',line)
    if m:
        cells=[c.strip() for c in line.strip().strip('|').split('|')]
        if len(cells)>=5: res[m.group(1)]=cells[4][:40]
# links to JSON tests / doc rows
xw=json.load(open(f'{A}/tc-spec-id-crosswalk.json')); tc={t['file']:t for t in json.load(open(f'{A}/tc-catalog.json'))}
docs=json.load(open(f'{A}/doc-catalog.json'))
byid=collections.defaultdict(list)
for x in xw:
    for s in x['spec_ids']: byid[s].append({'kind':'json','ref':x['file'].replace('tests/tc/',''),'scope':tc[x['file']]['common_scope'],'separation':tc[x['file']]['separation']})
for d in docs:
    for s in d['spec_ids']:
        for e in ([s] if '~' not in s else [f'TC-1-2-{i:02d}' for i in range(1,7)]): byid[e].append({'kind':'doc','ref':d['doc_id'],'scope':d['common_scope'],'separation':d['separation']})
for m,rid in mgmt.items():
    t=[v for v in tc.values() if v['id']==rid]
    if t: byid[m].append({'kind':'json','ref':t[0]['file'].replace('tests/tc/',''),'scope':t[0]['common_scope'],'separation':t[0]['separation']})

TH='three-chain'; W4S='two-chain:wbft+stablenet'; W3W4='two-chain:wemix+wbft'; W3S='two-chain:wemix+stablenet'; SN='stablenet-only'; W4='wbft-only'; W3='wemix-only'; TR='transition-only'; EX='excluded-unit-or-build'
CO='config-only'; CQ='config+oracle'; AD='adapter'; NC='not-common'
R=[]  # (regex, scope, sep, family, reason, generic_subset)
def r(pat,scope,sep,fam,reason,generic=None): R.append((re.compile('^'+pat+'$'),scope,sep,fam,reason,generic))
# ---- StableNet regression RT-*
r('RT-A-1-01',TH,CQ,'sync-lifecycle','genesis 초기화·블록 0 해시 동일. WEMIX3.0 genesis는 바이너리 생성, WBFT 두 체인은 템플릿 (CD-B-07)')
r('RT-A-1-01-A',SN,NC,'sync-lifecycle','--testnet 내장 genesis(8283, 고정 hash)는 StableNet 전용')
r('RT-A-1-01-B',TH,CQ,'sync-lifecycle','사용자 genesis.json init. 필수 필드(anzeon 섹션 등)가 체인별 (CD-B-07)')
r('RT-A-1-0[23]',TH,CQ,'sync-lifecycle','Full/Snap sync. 기본 syncmode가 다르고 WEMIX3.0은 etcd syncCheck 개입 (CD-B-08, CD-B-09)')
r('RT-A-1-04',TH,AD,'fault-topology','노드 재시작. 프로세스 제어 필요 (H-07). 기대 "1초 간격"은 WBFT 값')
r('RT-A-1-05',TH,CO,'rpc-basics','PN bootnode 경유 피어 연결. bootnode 프로필')
r('RT-A-1-0[67]',TH,AD,'sync-lifecycle','downloader/fetcher 경로. 경로 증명에 로그·metrics 계측 필요 (H-08). 전제 "Anzeon 활성"은 StableNet 표기')
r('RT-A-2-0[12]',TH,CQ,'transaction-types','type 0x0/0x2. 명세의 effectiveGasPrice 공식은 표준식이나 StableNet 비인가 계정은 헤더 GasTip으로 tip이 바뀐다 (CD-A-06)')
r('RT-A-2-03',TH,CO,'transaction-types','type 0x1. 세 클라이언트 공통')
r('RT-A-2-04',TH,CO,'nonce-replacement','nonce 순서')
r('RT-A-2-05a',TH,CQ,'fee-policy','tip 하한 미달 거부. 하한 출처가 체인별(pool.gasPrice / miner.gasprice / MinTip). 메시지 prefix "gas tip cap"은 go-wbft/go-stablenet 형식 (CD-A-06, CD-A-08)')
r('RT-A-2-05b',SN,NC,'fee-policy','feeCap < MinBaseFee+MinTip 거부는 Anzeon 전용 검사 (CD-A-06)')
r('RT-A-2-0[67]',TH,CQ,'invalid-transaction','잔액 부족·gasLimit 초과 거부. sentinel은 같고 감싼 문맥이 다르다 (CD-A-08)')
r('RT-A-2-08',TH,CO,'fee-observation','receipt.effectiveGasPrice 존재')
r('RT-A-2-09',TH,CQ,'nonce-replacement','replacement. priceBump 프로필')
r('RT-A-2-10',W4S,CQ,'modern-evm','EIP-7702. go-wemix 미지원. gate가 Croissant 높이 vs anzeon 설정 존재 (CD-A-04)')
r('RT-A-3-0[1-7]',TH,CQ,'evm-contract','배포/호출/eth_call/estimateGas/revert/OOG. fixture EVM revision(PUSH0) 공통화 필요 (CD-B-05)')
r('RT-A-4-0[1-4]',TH,CO,'rpc-basics','표준 eth_* 조회')
r('RT-A-4-05',TH,CO,'rpc-basics','eth_chainId == genesis chainId. expectedChainId 프로필 (CD-A-01)')
r('RT-A-4-0[67]',TH,CO,'logs-subscription','WS 구독. ws endpoint 프로필')
r('RT-B-01',W4S,CO,'consensus-wbft','1초 주기(명세 그대로). blockPeriod 프로필로 두면 세 체인 관측 가능','three-chain: 블록 주기 profile 값 비교')
r('RT-B-0[237]',W4S,CO,'consensus-wbft','WBFTExtra seal/epoch/validators (istanbul_*)')
r('RT-B-0[45]',SN,NC,'stablenet-system-contract','GovValidator(0x1001) 제안·승인. WEMIX4.0 GovStaking과 다름 (CD-B-04)')
r('RT-B-06',SN,NC,'stablenet-anzeon-fee','WBFTExtra.GasTip 필드는 StableNet 전용 (CD-B-03)')
r('RT-B-08',W4S,AD,'consensus-wbft','쿼럼 미달 블록 거부. 조작 블록 주입 도구 필요(하네스에 없음)')
r('RT-B-(09|10)',W4S,AD,'consensus-wbft','round change. 제안자 정지에 프로세스 제어 (H-07)')
r('RT-B-1[12]',W4S,CO,'consensus-wbft','PrevCommittedSeal/PrevPreparedSeal ≥ quorum')
r('RT-C-0[12]',SN,NC,'stablenet-anzeon-fee','비인가/인가 계정 tip 강제는 StableNet 정책 (CD-A-06, CD-A-07)')
r('RT-C-0[345]',TH,CQ,'fee-policy','baseFee 증가/유지/감소. 임계값·target이 세 공식 (CD-A-06)')
r('RT-C-06',TH,CQ,'fee-policy','baseFee 하한. StableNet 상수 / WEMIX3.0 governance clamp 1 / WBFT 0 → 프로필 하한 비교')
r('RT-C-07',W3S,CQ,'fee-policy','baseFee 상한. WBFT EIP-1559에는 상한이 없다 (CD-A-06)')
r('RT-D-0[1345]',TH,CQ,'fee-delegation','type 0x16 대납·서명 변조·잔액 부족. 세 클라이언트 구현. 대납자 잔액 기준(feeCap/gasPrice)과 거부 시점(txpool/실행)이 다르다 (CD-A-03)')
r('RT-E-0[1-9]',SN,NC,'stablenet-account-policy','blacklist/authorized/zero·precompile 전송 차단 (CD-A-07)')
r('RT-F-.*',SN,NC,'stablenet-system-contract','NativeCoinAdapter/GovMinter/GovValidator/GovMasterMinter/GovCouncil (CD-B-04). RT-F-3-01~03은 취소선(RT-B-04/05 중복)')
r('RT-G-1-0[1-5]',TH,CO,'rpc-basics','eth 블록/tx 조회')
r('RT-G-1-06',SN,NC,'stablenet-system-contract','0x1000 코드 조회. 시스템 계약 주소는 체인별','three-chain: 배포한 일반 계약의 eth_getCode')
r('RT-G-2-0[12]',TH,CQ,'fee-observation','eth_gasPrice = tip + baseFee, eth_maxPriorityFeePerGas = tip. 식은 같고 tip 출처만 다르다 (CD-A-06)')
r('RT-G-2-03',TH,CQ,'fee-observation','feeHistory 형태는 공통. "MinBaseFee 이상" 조건은 StableNet')
r('RT-G-2-04',SN,AD,'stablenet-system-contract','NativeCoinAdapter.transfer estimateGas. 일반 계약 fixture면 three-chain(RT-A-3-04)','three-chain: 일반 계약 estimateGas')
r('RT-G-3-0[1-6]',W4S,CO,'consensus-rpc-istanbul','istanbul_* (CD-B-02)')
r('RT-G-4-0[1-4]',TH,CO,'rpc-basics','net/txpool/admin 조회. namespace 노출 프로필')
r('RT-G-5-01',TH,CO,'fee-delegation','eth_signRawFeeDelegateTransaction 세 클라이언트 구현 (CD-B-02)')
r('RT-G-5-0[23]',SN,NC,'stablenet-system-contract','NativeCoinAdapter totalSupply/allowance')
# ---- 1st Test Cases TC-*
r(r'TC-1-1-\d{2}',SN,NC,'stablenet-post-v1','GovMinter v2 burn refund / Boho 업그레이드')
r('TC-1-2-0[1-6]',W4S,CQ,'modern-evm','P256(0x100). go-wemix 없음. WBFT Croissant / StableNet Boho gate (CD-A-05). TC-1-2-02(Boho 이전 미존재)는 StableNet gate 전용')
r('TC-1-3-0[1-6]',TH,CQ,'fee-policy','최소 가스비 하한. 기준값이 체인별 (CD-A-06)')
r('TC-3-1-0[123]',EX,NC,'unit-benchmark','Go 벤치마크. 노드 실행 테스트가 아님(명세도 코드 기반 불가로 표시)')
r('TC-3-1-04',TH,AD,'sync-lifecycle','바이너리 교체 전후 서명 호환. swapNode 바이너리·로그 문구 체인별')
r('TC-4-1-0[12]',SN,NC,'stablenet-post-v1','Anzeon 설정으로 WBFT 엔진 초기화·Boho 반영')
r('TC-4-1-03',TH,AD,'sync-lifecycle','저장 genesis와 불일치 시 기동 거부. 로그 문구 체인별')
r('TC-4-2-0[123]',W4S,CQ,'modern-evm','7702 authorizationList estimateGas (CD-A-04)')
r('TC-4-3-0[1-6]',SN,NC,'stablenet-post-v1','GovCouncil authorized 주소 문자열 파싱 (WEMIX4.0의 NODE-006/007이 대응 개념)')
r('TC-4-4-0[1-4]',SN,NC,'stablenet-post-v1','Anzeon+Boho 동일 블록 적용')
r(r'TC-4-5-\d{2}',SN,NC,'stablenet-post-v1','alloc.Extra·GovCouncil 동기화')
r('TC-4-6-0[14]',SN,NC,'stablenet-anzeon-fee','인증 계정 egp 재계산·AuthorizedTxExecuted 로그')
r('TC-4-6-02',SN,NC,'stablenet-anzeon-fee','headerGasTip으로 egp 재계산(명세 그대로)','three-chain: BP/EN receipt effectiveGasPrice 동일 (JSON 02b)')
r('TC-4-6-03',TH,CQ,'fee-observation','egp가 있으면 덮어쓰지 않음. snap sync 노드 필요')
r('TC-5-1-0[123]',EX,NC,'build','빌드·CI·runtime.Version 확인')
r('TC-5-2-0[1-6]',SN,NC,'stablenet-post-v1','CollectUpgrades/시스템 계약 버전 레지스트리')
r('TC-5-3-01',TH,CO,'sync-lifecycle','genesis 블록 해시 노드 간 일치')
# ---- 1st Test Scenarios TS-* (map to TC groups)
r(r'TS-1-1-\d{2}',SN,NC,'stablenet-post-v1','TS는 TC-1-1 시나리오')
r('TS-1-2',W4S,CQ,'modern-evm','TC-1-2 시나리오')
r('TS-1-3-0[12]',TH,CQ,'fee-policy','TC-1-3 시나리오')
r('TS-2-[1-4]',EX,NC,'unit-test','취약점 패치 단위 테스트(명세가 통합 테스트 제외)')
r('TS-4-1-01',SN,NC,'stablenet-post-v1','TC-4-1-01/02 시나리오')
r('TS-4-1-02',TH,AD,'sync-lifecycle','TC-4-1-03 시나리오')
r('TS-4-2',W4S,CQ,'modern-evm','TC-4-2 시나리오')
r('TS-4-3-01',SN,NC,'stablenet-post-v1','TC-4-3 시나리오')
r('TS-4-4',SN,NC,'stablenet-post-v1','TC-4-4 시나리오')
r('TS-4-5-0[123]',SN,NC,'stablenet-post-v1','TC-4-5 시나리오')
r('TS-4-6',TH,CQ,'fee-observation','TC-4-6 시나리오(snap sync egp). 인증 계정 부분은 StableNet')
r('TS-5-1-01',EX,NC,'build','TC-5-1 시나리오')
r('TS-5-2-0[123]',SN,NC,'stablenet-post-v1','TC-5-2 시나리오')
r('TS-5-3-01',TH,CO,'sync-lifecycle','TC-5-3 시나리오')
# ---- WEMIX4.0
r('NODE-00[12]',TR,NC,'transition','go-wemix 데이터로 go-wbft 기동·하드포크 전후 비교. 전환 시나리오')
r('NODE-00[34]',TH,CQ,'sync-lifecycle','Full/Snap sync (CD-B-08/09)')
r('NODE-005',TH,AD,'fault-topology','1/7 장애 후 복구. 허용 장애 수는 family별 (CD-B-01, H-07)')
r('NODE-00[67]',W4,NC,'wbft-governance','NCP 주소 파싱. StableNet TC-4-3이 대응 개념')
r('TX-001',TH,CO,'tx-basics','일반 송금')
r('TX-002',TH,CQ,'fee-policy','baseFee 미달 거부. 기준이 체인별 (CD-A-06)')
r('TX-00[36]',TH,CQ,'transaction-types','0x2/0x0. effectiveGasPrice 식 (CD-A-06)')
r('TX-004',TH,CQ,'fee-delegation','0x16 (CD-A-03)')
r('TX-005',TH,CQ,'evm-contract','배포·상태 변경·view·revert·OOG. fixture revision (CD-B-05)')
r('TX-007',TH,CO,'transaction-types','0x1')
r('TX-008',W4S,CQ,'modern-evm','7702 (CD-A-04)')
r('TX-(009|019|020)',W4S,CQ,'modern-evm','P256 (CD-A-05)')
r('TX-010',TH,CO,'nonce-replacement','nonce 순서')
r('TX-01[12]',TH,CQ,'invalid-transaction','거부 sentinel (CD-A-08)')
r('TX-013',TH,CQ,'nonce-replacement','queued 교체. priceBump')
r('TX-01[456]',TH,CQ,'fee-delegation','대납 서명 변조·잔액 부족 (CD-A-03)')
r('TX-01[78]',TH,CQ,'evm-contract','revert/OOG. fixture revision')
r('WBFT-00[1259]',W4S,CO,'consensus-wbft','Finalize/주기/epoch/prevSeal')
r('WBFT-004',W4S,CO,'consensus-wbft','WBFT-003에 통합')
r('WBFT-00[3678]',W4S,AD,'consensus-wbft','view change·proposer 순환·장애 수. 프로세스 제어 (H-07)')
r('WBFT-010',W4S,CO,'consensus-wbft','RandaoReveal/MixDigest')
r('WBFT-01[123]',W4S,AD,'consensus-wbft','quorum 계산 장애 시험. 프로세스 제어')
r(r'GOV-0\d{2}',W4,NC,'wbft-governance','GovConfig/GovStaking/GovNCP/GovRewardee (CD-B-04)')
r('RPC-001',TH,CO,'rpc-basics','eth_blockNumber')
r('RPC-002',W4S,CO,'rpc-basics','블록 조회 "WBFTExtra 포함"(명세 그대로)','three-chain: 번호/해시 조회 일치')
r('RPC-00[3456]',W4S,CO,'consensus-rpc-istanbul','istanbul_*')
r('RPC-007',TH,CO,'rpc-basics','receipt')
r('RPC-008',W3W4,CQ,'reward','wemix_getBriocheBlockReward. Brioche 설정 필요 (CD-B-06)')
r('RPC-009',W4,NC,'wbft-governance','거버넌스 계약 eth_call')
r('RPC-01[01]',W4S,CO,'consensus-rpc-istanbul','istanbul_nodeAddress/isValidator')
r('RPC-01[2345]',TH,CO,'rpc-basics','eth_getBalance/chainId/getLogs/getTransactionCount')
r('RPC-016',TH,CQ,'fee-observation','eth_gasPrice ≥ baseFee. tip 출처 체인별')
r('RPC-017',TH,CO,'fee-observation','eth_feeHistory')
r('RPC-01[89]',TH,CO,'rpc-basics','txpool_*/admin_peers. namespace 노출')
r('RPC-02[01]',TH,CO,'logs-subscription','WS 구독')
r('RPC-022',W4S,CQ,'consensus-wbft','epochInfo 존재. Stakers 키는 WEMIX4.0, StableNet은 candidates (CD-B-03)')
r('RPC-023',W4,NC,'wbft-governance','언스테이킹 후 검증자 제외')
# ---- WEMIX3.0
r('ETCD-0[1-5]',W3,NC,'wemix-etcd','etcd work/token 키 (CD-B-01)')
r('MINING-01',TH,CO,'consensus-wbft','타임스탬프 단조 증가. 관측은 세 체인 공통')
r('MINING-0[23]',W3,NC,'wemix-mining','time-it catch-up, miner limit 창은 PoA 규칙')
r('BRIOCHE-0[12]',W3W4,CQ,'reward','wemix_briocheConfig/halvingSchedule/getBriocheBlockReward. 두 클라이언트 모두 Brioche 설정 시 등록 (CD-B-06)')
r('BRIOCHE-03',W3W4,AD,'reward','실제 보상 배분. WEMIX3.0 governance 배분 vs WEMIX4.0 beneficiary (CD-B-06)')
r('GOV-01',W3,NC,'wemix-governance','거버넌스 멤버 변경 시 BP·피어 갱신')
r('RPC-01',W3,NC,'wemix-rpc','admin_wemixInfo')
r('RPC-02',W3,NC,'wemix-rpc','rewards/fees/minerNodeSig 헤더 필드')
# ---- 2nd change (both pages)
r('T-1@.*',W4S,CQ,'txpool','txpool 누적 잔액 검사. go-wbft/go-stablenet은 pending 합산 검사, go-wemix core/tx_pool.go는 단건 검사만 확인됨 (CD-A-03)')
r('T-2@.*',TH,CQ,'fee-delegation','AccessList 포함 0x16. go-wemix 경로는 실행 확인 필요')
r('T-3@.*',TH,CQ,'fee-delegation','eth_sendTransaction/eth_signTransaction 0x16 서명. keystore 지원은 클라이언트별 확인 필요')
r('T-4@.*',W4S,CO,'consensus-wbft','vanityData raw hex')
r('T-5@.*',W4S,CO,'consensus-rpc-istanbul','istanbul_status 안정성')
# ---- PR196 new tests
r('N-008',TH,CQ,'nonce-replacement','미포함 tx 이월·교체. 크기 한도 유도는 WEMIX3.0 PR196 값')
r('N-009',TH,CQ,'invalid-transaction','거부 vs 실행 실패 상태. 공통 관측')
r('N-01[0-5]',W3,NC,'wemix-pr196','블록 크기(8 MiB)·메시지 크기(10 MiB)·생성 종료 조건은 PR196 hotfix 값. 다른 두 클라이언트의 대응 제한값은 확인하지 않음')
r('N-00[1-7]',W3,NC,'wemix-pr196','PR196 신규 테스트 중 집중 목록 제외분(N-005 혼합 노드 구성 등). 상세 절차 페이지에 없어 내용 미열람')
r('N-016',W3,NC,'wemix-pr196','PR196 신규 테스트. 144 목록·집중 목록 모두 제외, 내용 미열람')
r(r'PR196-TC-\d{3}',None,None,'pr196-management-id','JSON 테스트 관리 ID. 판정은 연결된 JSON 케이스를 따른다')
out=[]
for key in sorted(uni, key=lambda k:(re.sub(r'\d+','',k), [int(x) for x in re.findall(r'\d+',k)])):
    base=key.split('@')[0]
    hit=None
    for pat,scope,sep,fam,reason,generic in R:
        if pat.match(key) or pat.match(base): hit=(scope,sep,fam,reason,generic); break
    if not hit: print('UNMATCHED',key); continue
    scope,sep,fam,reason,generic=hit
    links=byid.get(base,[])+([] if base==key else byid.get(key,[]))
    if key.startswith('PR196-TC') and links: scope,sep=links[0]['scope'],links[0]['separation']
    out.append({'spec_id':key,'pages':sorted(uni[key]),'page_titles':[SPEC_PAGES[p] for p in sorted(uni[key])],'common_scope':scope,'separation':sep,'family':fam,'reason':reason,'generic_subset':generic,'pr196_priority':prio.get(base),'stablenet_result_20260413':res.get(base),'linked':links,'runtime_id':mgmt.get(key)})
json.dump(out,open(f'{A}/confluence-spec-catalog.json','w'),indent=1,ensure_ascii=False)
c=collections.Counter(o['common_scope'] for o in out)
print(len(out),c); print(collections.Counter((o['common_scope'],o['separation']) for o in out if o['common_scope']=='three-chain'))
print('linked to json/doc:',sum(1 for o in out if o['linked']),'unlinked three-chain:',[o['spec_id'] for o in out if o['common_scope']=='three-chain' and not o['linked']])
