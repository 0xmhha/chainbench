#!/usr/bin/env python3
"""Fresh judgment of the 74 scenario-document rows against current client code."""
import json, os, sys
OUT=sys.argv[1]
rows=json.load(open(os.path.join(OUT,'analyses','scenario-doc-rows.json')))
# per doc_id override: (family, scope, separation, reason, config_items, implementation_items)
FEEORACLE='수수료 기대값을 체인별 FeePolicy oracle로 (CD-A-06)'
SIGNER='로컬 서명 sender로 raw tx 전송 (H-05)'
J={}
def j(ids, fam, scope, sep, reason, cfg=(), impl=()):
    for i in ids: J[i]=(fam,scope,sep,reason,list(cfg),list(impl))
j(['DOC-C-001'],'transaction-types','three-chain','config+oracle','Legacy tx. effectiveGasPrice==gasPrice는 표준, StableNet 비인증 계정은 헤더 GasTip 강제라 기대식이 다르다',['fee 입력 프로필'],[FEEORACLE,SIGNER])
j(['DOC-C-002'],'transaction-types','three-chain','config+oracle','Dynamic fee tx. effectiveGasPrice 공식이 StableNet 일반 계정에서 다르다',['fee 입력 프로필'],[FEEORACLE,SIGNER])
j(['DOC-C-003'],'transaction-types','three-chain','config-only','Access list tx. 세 클라이언트 모두 type 0x1과 eth_createAccessList 지원',['fee 입력 프로필'],[SIGNER])
j(['DOC-C-004','DOC-C-005','DOC-C-006','DOC-C-007'],'fee-delegation','three-chain','config+oracle','type 0x16. 세 클라이언트 구현. fee payer 잔액 검사·오류 문자열·fork gate(WBFT Croissant 분기, StableNet Anzeon blacklist)가 다르다',['sender/feePayer 계정 프로필','fork 활성 프로필'],['fee payer 정책·거부 사유 oracle (CD-A-03)',SIGNER])
j(['DOC-C-008'],'nonce-replacement','three-chain','config-only','nonce 순서. 표준 txpool 동작',['fee 입력 프로필'],[SIGNER])
j(['DOC-C-009'],'fee-policy','three-chain','config+oracle','underpriced 거부. WEMIX3 pool.gasPrice, WBFT pricelimit/baseFee, StableNet MinTip으로 기준이 다르다',['minTip/pricelimit 프로필'],[FEEORACLE])
j(['DOC-C-010','DOC-C-011'],'invalid-transaction','three-chain','config+oracle','잔액 부족·gasLimit 초과 거부. 동작은 공통, 오류 문자열은 클라이언트 버전별로 다를 수 있어 문자열 완전일치 대신 거부 여부로 판정',[],['오류 문자열 비교를 거부 여부 판정으로 통일 (CD-A-08)'])
j(['DOC-C-012'],'fee-observation','three-chain','config-only','BP/EN effectiveGasPrice 일치·baseFee 이상',['노드 역할(BP/EN) 프로필'],[SIGNER])
j(['DOC-C-013'],'nonce-replacement','three-chain','config+oracle','replacement 10%+ 인상. PriceBump는 노드 설정값',['priceBump 프로필'],[SIGNER])
j(['DOC-C-014','DOC-C-015','DOC-C-016','DOC-C-017','DOC-C-018','DOC-C-019','DOC-C-020'],'evm-contract','three-chain','config+oracle','배포/호출/eth_call/estimateGas/revert/OOG. fixture bytecode를 세 체인 공통 EVM revision으로 컴파일해야 한다',['fixture bytecode·ABI 프로필'],['EVM revision별 fixture 검증 (CD-B-05)',SIGNER])
j(['DOC-C-021','DOC-C-022','DOC-C-025','DOC-C-028','DOC-C-029','DOC-C-030','DOC-C-031','DOC-C-032','DOC-C-035'],'rpc-basics','three-chain','config-only','표준 eth_* 조회',['expectedChainId·알려진 계정 프로필'])
j(['DOC-C-023'],'rpc-basics','three-chain','config-only','eth_sendRawTransaction + txpool 조회',['txpool namespace 노출'],[SIGNER])
j(['DOC-C-024'],'logs-subscription','three-chain','config-only','eth_getLogs',[],[SIGNER])
j(['DOC-C-026','DOC-C-027'],'logs-subscription','three-chain','config-only','WS 구독. 세 클라이언트 filters API 구현',['WS endpoint 프로필'],[SIGNER])
j(['DOC-C-033','DOC-C-034'],'fee-observation','three-chain','config+oracle','eth_gasPrice / eth_maxPriorityFeePerGas. StableNet은 헤더 GasTip 고정식, 나머지는 oracle 추정',[],[FEEORACLE])
j(['DOC-C-036','DOC-C-037'],'txpool','three-chain','config-only','txpool_status/content',['txpool namespace 노출'],[SIGNER])
j(['DOC-C-038'],'fee-delegation','three-chain','config-only','eth_signRawFeeDelegateTransaction 존재. 세 클라이언트 internal/ethapi에 구현')
j(['DOC-C-039'],'sync-lifecycle','three-chain','config+oracle','genesis 초기화·block 0 해시. WEMIX3 genesis는 바이너리가 governance 설정에서 생성, WBFT/StableNet은 template',['genesis 원본·expectedGenesisHash 프로필'],['family별 genesis 생성 경로 (H-04)'])
j(['DOC-C-040','DOC-C-041'],'sync-lifecycle','three-chain','config+oracle','full/snap sync. 세 클라이언트 모두 downloader에 SnapSync가 있으나 기본 syncmode가 다르다(go-wemix snap, go-wbft/go-stablenet full). WBFT 계열 snap sync 가능 여부는 실행 확인 필요',['syncmode launch 프로필'],['동기화 노드 추가·진행 관측 구현 (H-04)'])
j(['DOC-C-042'],'fault-topology','three-chain','adapter','노드 재기동',[],['프로세스 제어 (H-07)'])
j(['DOC-C-043'],'rpc-basics','three-chain','config-only','bootnodes/peers',['bootnode·정적 peer 프로필'])
j(['DOC-C-044','DOC-C-045'],'sync-lifecycle','three-chain','adapter','downloader/fetcher 경로. 경로 증명에는 노드 로그·metrics 관측이 필요',[],['경로 관측기 구현 (readNodeLog/metric) (H-08)'])
j(['DOC-D-001'],'reward','two-chain:wemix+wbft','config+oracle','wemix_getBriocheBlockReward. go-wemix eth/api.go, go-wbft eth/api_wemix.go 구현. StableNet 없음',['brioche fork 프로필'],['보상 기대값 oracle (CD-B-06)'])
j(['DOC-D-002'],'consensus-wbft','two-chain:wbft+stablenet','config-only','블록 주기 1초. blockPeriod 프로필로 두면 WEMIX3에도 확장 가능',['blockPeriod 프로필'])
j(['DOC-D-003','DOC-D-004','DOC-D-009','DOC-D-010','DOC-D-011'],'consensus-wbft','two-chain:wbft+stablenet','config-only','WBFTExtra seal/epoch/prevSeal/randao',['epoch·topology 프로필'])
j(['DOC-D-005'],'consensus-wbft','two-chain:wbft+stablenet','config+oracle','WBFTExtra 필드 반영. GasTip 필드는 StableNet 전용이므로 WEMIX4는 다른 필드로 판정',[],['체인별 헤더 필드 oracle (CD-B-03)'])
j(['DOC-D-006','DOC-D-007','DOC-D-008'],'consensus-wbft','two-chain:wbft+stablenet','adapter','round change·proposer 순환. 제안자 정지에 프로세스 제어 필요',[],['프로세스 제어 (H-07)'])
j(['DOC-D-012'],'consensus-wbft','stablenet-only','not-common','쿼럼 미달 블록 유도는 문서상 StableNet 전용 관점')
j(['DOC-D-013','DOC-D-014','DOC-D-015','DOC-D-016','DOC-D-017'],'consensus-wbft','two-chain:wbft+stablenet','adapter','검증자 장애 수별 합의 지속/중단. 프로세스 제어·quorum 계산',['validator 수 프로필'],['프로세스 제어 (H-07)','quorum oracle (CD-B-01)'])
j(['DOC-D-018','DOC-D-019','DOC-D-020','DOC-D-021','DOC-D-022','DOC-D-023'],'consensus-rpc-istanbul','two-chain:wbft+stablenet','config-only','istanbul_* RPC')
j(['DOC-D-024','DOC-D-025','DOC-D-026'],'modern-evm','two-chain:wbft+stablenet','config+oracle','EIP-7702. go-wemix 미지원. WBFT Croissant/StableNet Anzeon gate',['fork 활성 프로필'],['fork gate preflight (CD-A-04)'])
j(['DOC-D-027','DOC-D-028','DOC-D-029'],'modern-evm','two-chain:wbft+stablenet','config+oracle','P256VERIFY. go-wemix 미지원. WBFT Croissant/StableNet Boho gate',['fork 활성 프로필'],['fork gate preflight (CD-A-05)'])
out=[]
for r in rows:
    fam,scope,sep,reason,cfg,impl=J[r['doc_id']]
    out.append({**r,'family':fam,'common_scope':scope,'separation':sep,'reason':reason,'configuration_items':cfg,'implementation_items':impl})
json.dump(out,open(os.path.join(OUT,'analyses','doc-catalog.json'),'w'),indent=1,ensure_ascii=False)
from collections import Counter
print(len(out),Counter(o['common_scope'] for o in out),Counter((o['common_scope'],o['separation']) for o in out))
