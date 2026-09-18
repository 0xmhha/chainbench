#!/usr/bin/env python3
"""Fresh commonality judgment for every tests/tc JSON case (2026-09-11), made from
the current step content (scripts/step-signatures.txt) and current client code.
Scope values: three-chain | two-chain:wbft+stablenet | two-chain:wemix+wbft |
stablenet-only | wbft-only | wemix-only | transition-only | harness-only.
Separation: config-only (shared steps, env/profile values differ) |
config+oracle (shared steps, expected value or input needs a per-chain policy
source) | adapter (a step or precondition needs per-chain implementation) |
not-common."""
import json, sys, os, fnmatch
OUT = sys.argv[1]
recs = json.load(open(os.path.join(OUT, 'analyses', 'tc-dependencies.json')))

# (pattern, family, scope, separation, reason, config_items, impl_items)
R = []
def rule(pat, family, scope, sep, reason, cfg=(), impl=()):
    R.append((pat, family, scope, sep, reason, list(cfg), list(impl)))

CFG_ACC = 'from/to 주소 literal을 계정 label(역할)로 치환'
CFG_TOPO = 'topology·timeout·블록 수를 프로필 값으로'
CFG_FEE = 'gas/fee 입력을 프로필 값으로'
IMPL_SIGNER = '공개 RPC에서는 node-signed 경로 대신 로컬 서명 sender 필요 (H-05)'
IMPL_CONS = '합의 family별 정상·장애 판정 oracle (PoA/etcd vs WBFT quorum) (CD-B-01)'
IMPL_FEESRC = 'istanbul_getWbftExtraInfo.gasTip 입력을 체인별 fee source로 대체 (eth_maxPriorityFeePerGas 또는 profile) (CD-A-06)'
IMPL_FEEORACLE = '수수료 기대값(baseFee 변화·하한·effectiveGasPrice 공식)을 체인별 FeePolicy oracle로 (CD-A-06)'
IMPL_PROC = '소유한 프로세스(compose/workspace)에서만 실행, attach 대상은 SKIP 사유 기록 (H-07)'

# ---- shared directories
rule('basic/01-*', 'sync-lifecycle', 'three-chain', 'config+oracle', '블록 진행과 latest 해시 일치. requires consensus의 의미가 family마다 다르다', [CFG_TOPO], [IMPL_CONS])
rule('basic/02-*', 'rpc-basics', 'three-chain', 'config-only', 'peerCount>=1. 노드 역할 이름(node1..en1)만 프로필로', [CFG_TOPO])
rule('basic/03-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_blockNumber 응답', [CFG_TOPO])
rule('basic/04-*', 'sync-lifecycle', 'three-chain', 'config-only', 'sameBlockHash latest', [CFG_TOPO])
rule('basic/05-*', 'tx-basics', 'three-chain', 'config-only', 'node-signed 송금. from 주소 literal(preset node1 계정)', [CFG_ACC], [IMPL_SIGNER])
rule('basic/06-*', 'txpool', 'three-chain', 'config-only', 'node1 송금 후 node2에서 잔액 관측(전파)', [CFG_ACC], [IMPL_SIGNER])
rule('basic/07-*', 'consensus-wbft', 'two-chain:wbft+stablenet', 'config-only', 'istanbul_getValidators / getWbftExtraInfo. go-wemix에는 istanbul namespace가 없다', [CFG_TOPO])
rule('fault/01-*', 'fault-topology', 'three-chain', 'adapter', 'admin_removePeer 분할 후 회복. 프로세스·peer 제어 필요', [CFG_TOPO], [IMPL_PROC, IMPL_CONS])
rule('fault/02-*', 'fault-topology', 'three-chain', 'adapter', '1/4 정지 후 진행. 허용 장애 수는 family별', [CFG_TOPO], [IMPL_PROC, IMPL_CONS])
rule('fault/03-*', 'fault-topology', 'three-chain', 'adapter', '정지·재시작 후 동기화', [CFG_TOPO], [IMPL_PROC, IMPL_CONS])
rule('fault/04-*', 'fault-topology', 'three-chain', 'adapter', 'hub-spoke 분할 + 송금. from literal', [CFG_TOPO, CFG_ACC], [IMPL_PROC, IMPL_CONS, IMPL_SIGNER])
rule('fault/05-*', 'fault-topology', 'three-chain', 'adapter', '2/4 정지 시 blockHalt 기대. WBFT quorum 3/4와 PoA/etcd 다수결은 다른 규칙이므로 halt 기대값은 family oracle', [CFG_TOPO], [IMPL_PROC, IMPL_CONS])
rule('fault/06-*', 'fault-topology', 'three-chain', 'adapter', 'leader 정지 후 pending 소진. from literal, txpool_status 필요', [CFG_TOPO, CFG_ACC], [IMPL_PROC, IMPL_CONS, IMPL_SIGNER])
rule('remote/*', 'remote-live', 'three-chain', 'config-only', 'attach 대상의 읽기 RPC만 사용. applicableChains에 wemix 추가만 필요', ['RPC endpoint 프로필', 'applicableChains'])
rule('samples/01-*', 'tx-basics', 'three-chain', 'config-only', 'node1→node2 송금. $transferGas는 hooks.pre에서 채움', [CFG_FEE], [IMPL_SIGNER])
rule('samples/02-*', 'fault-topology', 'three-chain', 'adapter', 'en1 정지·재시작. genesis overlay bohoBlock은 stablenet 전용 키이므로 env 분리', [CFG_TOPO, 'genesis overlay를 체인 env로'], [IMPL_PROC])
rule('stress/01-*', 'stress', 'three-chain', 'config+oracle', '15블록이 60초 이내. 블록 주기가 체인·설정별로 다르다', ['blockPeriod·maxSeconds 프로필'], ['블록 주기 기대값을 profile oracle로'])
rule('stress/02-*', 'stress', 'three-chain', 'adapter', 'load(gas-burn 계약 생성)로 fillPercent 30%. from literal, node-signed', [CFG_ACC, 'fillPercent·blocks 프로필'], [IMPL_SIGNER, 'load 생성기의 계약 fixture가 각 체인 EVM revision에서 유효한지 확인'])

# ---- go-stablenet/regression
rule('go-stablenet/regression/api/01-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_getBlockByNumber 필드 형태')
rule('go-stablenet/regression/api/02-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_getBlockByHash 일치')
rule('go-stablenet/regression/api/03-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_getTransactionByHash 필드. from 기대값이 node1 주소 literal', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/regression/api/04-*', 'rpc-basics', 'three-chain', 'config-only', 'receipt 필드', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/regression/api/05-*', 'rpc-basics', 'three-chain', 'config-only', 'nonce 증가. params 안 주소 literal', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/regression/api/06-*', 'stablenet-system-contract', 'stablenet-only', 'not-common', '0x1000~0x1004 코드 존재. 다른 체인엔 해당 주소에 시스템 계약이 없다')
rule('go-stablenet/regression/api/07-gas-price-positive*', 'fee-observation', 'three-chain', 'config-only', 'eth_gasPrice > 0')
rule('go-stablenet/regression/api/07b-*', 'fee-observation', 'three-chain', 'config+oracle', 'eth_gasPrice == baseFee + tip. 공식은 세 클라이언트 같고(SuggestGasTipCap + baseFee) tip 출처만 다르다(StableNet 헤더 gasTip, WEMIX3.0 governance, WBFT oracle)', ['tip source 프로필'], [IMPL_FEESRC])
rule('go-stablenet/regression/api/08-*', 'fee-observation', 'three-chain', 'config+oracle', 'eth_maxPriorityFeePerGas == tip source. StableNet은 헤더 gasTip과 정확히 같고 다른 체인은 각자의 tip 출처와 비교', ['tip source 프로필'], [IMPL_FEESRC])
rule('go-stablenet/regression/api/09-*', 'fee-observation', 'three-chain', 'config-only', 'eth_feeHistory 형태')
rule('go-stablenet/regression/api/10-*', 'stablenet-system-contract', 'stablenet-only', 'adapter', 'estimateGas 대상이 NativeCoinAdapter(0x1000). 일반 계약 fixture로 바꾸면 three-chain', ['대상 계약을 배포 fixture binding으로'], ['일반 ERC20 fixture 배포 후 estimateGas (CD-B-04)'])
rule('go-stablenet/regression/api/1[123456]*', 'consensus-rpc-istanbul', 'two-chain:wbft+stablenet', 'config-only', 'istanbul_* 조회. validators 수 4는 topology 값', [CFG_TOPO])
rule('go-stablenet/regression/api/18-*', 'txpool', 'three-chain', 'config-only', 'txpool_status 형태 (txpool namespace 노출 필요)', ['rpcModules 프로필'])
rule('go-stablenet/regression/api/19-*', 'txpool', 'three-chain', 'config-only', 'txpool_content 형태', ['rpcModules 프로필'])
rule('go-stablenet/regression/api/20-*', 'rpc-basics', 'three-chain', 'config-only', 'admin_peers (admin namespace 노출 필요). 이미 applicableChains에 wemix 포함', ['rpcModules 프로필'])
rule('go-stablenet/regression/api/21-*', 'fee-delegation', 'three-chain', 'config-only', 'eth_signRawFeeDelegateTransaction 존재. 세 클라이언트 모두 구현(internal/ethapi)', [CFG_ACC])
rule('go-stablenet/regression/api/2[23]-*', 'stablenet-system-contract', 'stablenet-only', 'not-common', 'NativeCoinAdapter(0x1000) ERC20 조회/승인')
rule('go-stablenet/regression/api/24-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_syncing == false')
rule('go-stablenet/regression/anzeon/0[12]-*', 'stablenet-anzeon-fee', 'stablenet-only', 'not-common', '일반/authorized 계정 tip 강제 정책 (0x1004 GovCouncil)')
rule('go-stablenet/regression/anzeon/0[345]-*', 'fee-policy', 'three-chain', 'config+oracle', '부하에 따른 baseFee 증가/유지/감소. 임계값·target·변화율이 체인별(WEMIX governance, WBFT EIP-1559 50%, StableNet Anzeon threshold)', ['fillPercent·blocks 프로필', CFG_ACC], [IMPL_FEEORACLE, IMPL_SIGNER])
rule('go-stablenet/regression/anzeon/06-*', 'fee-policy', 'three-chain', 'config+oracle', 'baseFee 하한. 값(20 Gwei)은 StableNet MinBaseFee. WEMIX3는 governance, WBFT는 0 하한이라 프로필 하한 비교로만 공통', ['minBaseFee 프로필'], [IMPL_FEEORACLE])
rule('go-stablenet/regression/anzeon/07-*', 'fee-policy', 'two-chain:wemix+stablenet', 'config+oracle', 'baseFee 상한. StableNet MaxBaseFee 상수와 WEMIX3.0 governance maxBaseFee는 있고 WBFT EIP-1559에는 상한이 없다', ['maxBaseFee 프로필(StableNet 상수 / WEMIX3.0 governance 값)'], [IMPL_FEEORACLE])
rule('go-stablenet/regression/anzeon/0[89]-*', 'fee-policy', 'three-chain', 'config+oracle', 'feeCap >= baseFee+tip 수락. tip 입력을 istanbul gasTip에서 읽는다', [CFG_FEE], [IMPL_FEESRC])
rule('go-stablenet/regression/anzeon/11-*', 'invalid-transaction', 'three-chain', 'config+oracle', '블록 gasLimit 초과 거부. fee 입력이 istanbul gasTip', [CFG_FEE], [IMPL_FEESRC])
rule('go-stablenet/regression/blacklist-authorized/*', 'stablenet-account-policy', 'stablenet-only', 'not-common', 'blacklist/authorized 계정 상태(GovCouncil 0x1004)와 zero/precompile 주소 송금 거부는 StableNet 전용 검증 규칙')
rule('go-stablenet/regression/ethereum/01-*', 'sync-lifecycle', 'three-chain', 'config+oracle', 'chain up 검사에 istanbul_getValidators 포함. validators 검사는 family adapter로', [CFG_TOPO], ['validators 검사를 consensus family capability로 치환 (H-04)'])
rule('go-stablenet/regression/ethereum/33-*', 'sync-lifecycle', 'three-chain', 'config+oracle', '15노드 chain up. 위와 같음', [CFG_TOPO], ['validators 검사를 consensus family capability로 치환 (H-04)'])
rule('go-stablenet/regression/ethereum/0[89]-*', 'transaction-types', 'three-chain', 'config+oracle', 'type 0x0/0x2 송금. gasPrice/feeCap 입력을 istanbul gasTip으로 계산', [CFG_FEE, CFG_ACC], [IMPL_FEESRC])
rule('go-stablenet/regression/ethereum/10-*', 'transaction-types', 'three-chain', 'config-only', 'type 0x1 + eth_createAccessList. 세 클라이언트 모두 구현', [CFG_FEE, CFG_ACC])
rule('go-stablenet/regression/ethereum/11*', 'nonce-replacement', 'three-chain', 'config-only', '로컬 키로 nonce 역순 제출. 고정 maxFee/tip literal', [CFG_FEE])
rule('go-stablenet/regression/ethereum/12-*', 'fee-policy', 'three-chain', 'config+oracle', 'tip=1 거부(expect reject). 거부 기준이 다르다: WEMIX3 pool.gasPrice, WBFT baseFee/pricelimit, StableNet MinTip', [CFG_FEE], [IMPL_FEEORACLE])
rule('go-stablenet/regression/ethereum/14-*', 'invalid-transaction', 'three-chain', 'config-only', '잔액 초과 송금 거부(expect reject). 이미 wbft/wemix로 이식됨')
rule('go-stablenet/regression/ethereum/15-*', 'invalid-transaction', 'three-chain', 'config-only', '블록 gasLimit 초과 거부(expect reject)')
rule('go-stablenet/regression/ethereum/16-*', 'fee-observation', 'three-chain', 'config-only', 'effectiveGasPrice > 0, gasUsed >= 21000')
rule('go-stablenet/regression/ethereum/17*', 'nonce-replacement', 'three-chain', 'config+oracle', '동일 nonce 20% 인상 교체. PriceBump 기본값(10%)은 세 클라이언트 동일하나 노드 설정값이므로 프로필로', ['priceBump 프로필', CFG_FEE])
rule('go-stablenet/regression/ethereum/18-*', 'modern-evm', 'two-chain:wbft+stablenet', 'config+oracle', 'EIP-7702 type 0x4. go-wemix는 타입 미지원. WBFT Croissant / StableNet Anzeon 활성 조건', ['fork 활성 프로필'], ['fork gate preflight (CD-A-04)'])
rule('go-stablenet/regression/ethereum/19-*', 'evm-contract', 'three-chain', 'config-only', 'deployContract(node-signed) + codeAt + call. fixture는 PUSH0 미사용', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/regression/ethereum/22-*', 'evm-contract', 'three-chain', 'config-only', 'estimateGas to precompile 0x01 >= 21000')
rule('go-stablenet/regression/ethereum/23-*', 'evm-contract', 'three-chain', 'config+oracle', 'eth_call revert 오류. 배포 bytecode가 PUSH0(0x5f)을 쓴다. go-wemix EVM은 PUSH0을 실행하지 못하므로 fixture를 체인별로', ['EVM revision별 fixture bytecode'], ['fixture 컴파일 타깃을 공통 최소 revision(pre-Shanghai)으로 맞추거나 체인별 fixture (CD-B-05)'])
rule('go-stablenet/regression/ethereum/24-*', 'evm-contract', 'three-chain', 'config-only', 'revert tx status 0x0. 이미 wbft/wemix로 이식됨')
rule('go-stablenet/regression/ethereum/25-*', 'evm-contract', 'three-chain', 'config-only', 'OOG gasUsed == gasLimit')
rule('go-stablenet/regression/ethereum/27-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_getBalance(node1) > 0')
rule('go-stablenet/regression/ethereum/27b-*', 'tx-basics', 'three-chain', 'config-only', '1 ETH 송금 잔액 차이. 수신 주소 literal', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/regression/ethereum/29-*', 'logs-subscription', 'three-chain', 'config-only', 'eth_getLogs 형태')
rule('go-stablenet/regression/ethereum/30-*', 'rpc-basics', 'three-chain', 'config-only', 'eth_chainId > 0. 실제 값 비교는 프로필 expectedChainId로', ['expectedChainId 프로필'])
rule('go-stablenet/regression/ethereum/3[12]-*', 'logs-subscription', 'three-chain', 'config-only', 'WS newHeads/logs 구독. ws capability 필요. fixture는 PUSH0 미사용', ['WS endpoint 프로필'], [IMPL_SIGNER])
rule('go-stablenet/regression/ethereum/36-*', 'logs-subscription', 'three-chain', 'config-only', '이벤트 로그 topic0 조회', [], [IMPL_SIGNER])
rule('go-stablenet/regression/fee-delegation/*', 'fee-delegation', 'three-chain', 'config+oracle', 'type 0x16 대납/서명 변조/잔액 부족. 세 클라이언트 모두 타입 지원. env genesis overlay(applepieBlock 0)는 fork gate. fee payer 잔액·blacklist 정책은 체인별', ['fork 활성 프로필', CFG_ACC], ['fee payer 비용·거부 사유 oracle (CD-A-03)'])
rule('go-stablenet/regression/system-contracts/*', 'stablenet-system-contract', 'stablenet-only', 'not-common', 'NativeCoinAdapter/GovMinter/GovMasterMinter/GovCouncil/GovValidator 거버넌스 시나리오')
rule('go-stablenet/regression/wbft/01-*', 'consensus-wbft', 'three-chain', 'config+oracle', '연속 블록 timestamp 차 == 1초. 값은 blockPeriod 프로필로 두면 PoA에도 적용 가능', ['blockPeriod 프로필'], ['블록 주기 기대값 profile oracle'])
rule('go-stablenet/regression/wbft/0[23]-*', 'consensus-wbft', 'two-chain:wbft+stablenet', 'config-only', 'WBFTExtra seal/epochInfo. epoch 길이(0x8c=140)는 genesis 값', ['epoch 프로필'])
rule('go-stablenet/regression/wbft/04*', 'stablenet-system-contract', 'stablenet-only', 'not-common', 'GovValidator(0x1001) 멤버 추가. WBFT4.0 거버넌스 계약(GovStaking 등)과 ABI·의미가 다르다')
rule('go-stablenet/regression/wbft/05-*', 'stablenet-system-contract', 'stablenet-only', 'not-common', 'GovValidator 멤버 제거')
rule('go-stablenet/regression/wbft/11-*', 'consensus-wbft', 'two-chain:wbft+stablenet', 'config-only', 'prevCommittedSeal/prevPreparedSeal >= quorum(ceil(2n/3))', [CFG_TOPO])
rule('go-stablenet/regression/wbft/13-*', 'consensus-wbft', 'two-chain:wbft+stablenet', 'config-only', 'randaoReveal·mixHash 존재')
rule('go-stablenet/regression/wbft/14-*', 'stablenet-anzeon-fee', 'stablenet-only', 'not-common', 'WBFTExtra.gasTip 필드는 StableNet 전용')
rule('go-stablenet/topology/01-*', 'fault-topology', 'three-chain', 'config-only', 'bp<->pn<->en 계층에서 en 진행. pn 역할이 family마다 구성 가능해야 함', [CFG_TOPO], ['pn 역할을 poa family가 지원하는지 확인 (H-04)'])
rule('go-stablenet/tx/01-*', 'evm-contract', 'three-chain', 'config-only', 'revert 계약 배포·호출. from literal, node-signed', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/vocabulary/01-*', 'harness-vocabulary', 'harness-only', 'config-only', '순수 파생(createAddress/contractChecksum). 체인 무관')
rule('go-stablenet/vocabulary/02-*', 'harness-vocabulary', 'three-chain', 'config-only', 'faucet(node coinbase funder). 공개 RPC에서는 불가', [], [IMPL_SIGNER])
rule('go-stablenet/vocabulary/03-metric*', 'harness-vocabulary', 'three-chain', 'config-only', 'Prometheus chain_head_block. --metrics 플래그는 세 클라이언트 공통(geth 계열)', ['launch metrics 프로필'])
rule('go-stablenet/vocabulary/03-register*', 'harness-vocabulary', 'three-chain', 'config-only', 'deployContract+registerContract(node-signed)', [CFG_ACC], [IMPL_SIGNER])

# ---- go-stablenet/post-v1.0.0-change
rule('go-stablenet/post-v1.0.0-change/common-all/1[234]-*', 'fee-policy', 'three-chain', 'config+oracle', '최소 가스비 미만 거부(expect reject). 하한 기준이 체인별(StableNet MinBaseFee/MinTip, WEMIX3 pool.gasPrice, WBFT pricelimit)', [CFG_FEE], [IMPL_FEEORACLE])
rule('go-stablenet/post-v1.0.0-change/common-all/17-*', 'modern-evm', 'two-chain:wbft+stablenet', 'config+oracle', 'EIP-7702 authorizationList estimateGas 비용. go-wemix 미지원', ['fork 활성 프로필'], ['fork gate preflight (CD-A-04)'])
rule('go-stablenet/post-v1.0.0-change/common-all/*', 'stablenet-post-v1', 'stablenet-only', 'not-common', 'Boho/Anzeon 포크, GovMinter v2, burn refund, delayed fork 등 StableNet v1.0.0 이후 변경 검증')
rule('go-stablenet/post-v1.0.0-change/effectivegasprice/02b-*', 'fee-observation', 'three-chain', 'config-only', 'BP/EN receipt의 effectiveGasPrice 동일. 수신 주소 literal', [CFG_ACC], [IMPL_SIGNER])
rule('go-stablenet/post-v1.0.0-change/effectivegasprice/*', 'stablenet-anzeon-fee', 'stablenet-only', 'not-common', 'authorized 계정/헤더 gasTip 정책')
rule('go-stablenet/post-v1.0.0-change/extra-state/*', 'stablenet-post-v1', 'stablenet-only', 'not-common', 'StableNet account extra-state 비트')
rule('go-stablenet/post-v1.0.0-change/stand-alone/01b-*', 'sync-lifecycle', 'three-chain', 'adapter', '바이너리 교체(swapNode) 전후 서명 호환. 목적은 공통이나 교체 바이너리·버전 문자열·로그 문구가 체인별 (PR196 144 목록에 WEMIX3.0 후보로 포함됨)', ['swap 대상 바이너리 프로필'], [IMPL_PROC, '버전 문자열·로그 기대값을 체인별로'])
rule('go-stablenet/post-v1.0.0-change/stand-alone/02-*', 'sync-lifecycle', 'three-chain', 'adapter', 'genesis 불일치 시 기동 거부. 목적은 공통이나 오류 로그 문구가 체인별 (PR196 144 목록에 WEMIX3.0 후보로 포함됨)', ['swap 대상 genesis overlay 프로필'], [IMPL_PROC, '기동 실패 로그 기대값을 체인별로'])
rule('go-stablenet/post-v1.0.0-change/stand-alone/04-*', 'sync-lifecycle', 'three-chain', 'config-only', 'genesis 해시 노드 간 일치·parentHash 0')
rule('go-stablenet/post-v1.0.0-change/stand-alone/*', 'stablenet-post-v1', 'stablenet-only', 'not-common', 'gstable 버전 교체(swapNode)·genesis 불일치·미지원 버전 로그. 바이너리 버전 문자열이 StableNet 전용')
rule('go-stablenet/post-v1.0.0-change/string-handling/*', 'stablenet-post-v1', 'stablenet-only', 'not-common', 'genesis authorizedAccounts 문자열 파싱')

# ---- go-wbft
rule('go-wbft/accounts/*', 'modern-evm', 'two-chain:wbft+stablenet', 'config+oracle', 'P256VERIFY(0x100). go-wemix 미지원. WBFT Croissant / StableNet Boho 활성 조건', ['fork 활성 프로필'], ['fork gate preflight (CD-A-05)'])
rule('go-wbft/chain-up/*', 'sync-lifecycle', 'three-chain', 'config+oracle', 'chain up + istanbul_getValidators', [CFG_TOPO], ['validators 검사를 consensus family capability로 치환 (H-04)'])
rule('go-wbft/consensus/01-*', 'sync-lifecycle', 'three-chain', 'adapter', 'en 노드가 첫 index인 혼합 배치에서 producers == 3. validators 검사는 istanbul_getValidators라 family adapter 필요', [CFG_TOPO], ['validators 검사를 consensus family capability로 치환 (H-04)'])
rule('go-wbft/fault/01-*', 'fault-topology', 'three-chain', 'adapter', '1/4 정지 후 진행·재시작', [CFG_TOPO], [IMPL_PROC, IMPL_CONS])
rule('go-wbft/network/01-*', 'fault-topology', 'three-chain', 'config-only', 'proxied peer graph의 peerCount 형태', [CFG_TOPO], ['pn 역할을 poa family가 지원하는지 확인 (H-04)'])
rule('go-wbft/tx/01-*', 'evm-contract', 'three-chain', 'config-only', '송금+배포+호출. from literal, genesis alloc overlay로 자금', [CFG_ACC, 'alloc overlay를 env.accounts로'], [IMPL_SIGNER])
rule('go-wbft/tx/02-*', 'invalid-transaction', 'three-chain', 'config-only', '잔액 초과 거부 (stablenet/14 이식본)')
rule('go-wbft/tx/03-*', 'evm-contract', 'three-chain', 'config-only', 'revert status 0x0 (stablenet/24 이식본)')

# ---- go-wemix
rule('go-wemix/chain-up/*', 'sync-lifecycle', 'three-chain', 'config-only', 'chainId/blockNumber/peerCount. validators 검사 없음(PoA)', [CFG_TOPO])
rule('go-wemix/fault/01-*', 'fault-topology', 'three-chain', 'adapter', '1/4 정지 후 진행·재시작 (PoA)', [CFG_TOPO], [IMPL_PROC, IMPL_CONS])
rule('go-wemix/handoff/01-*', 'transition', 'transition-only', 'not-common', 'go-wemix → go-wbft 혼합 바이너리 handoff. 특정 전환 시나리오')
rule('go-wemix/rpc/01-*', 'reward', 'two-chain:wemix+wbft', 'config+oracle', 'wemix_getBriocheBlockReward halving. StableNet에 없음. genesis brioche overlay', ['brioche fork 프로필'], ['보상 기대값 oracle (CD-B-06)'])
rule('go-wemix/tx/01-*', 'evm-contract', 'three-chain', 'config-only', '송금+배포+호출. from literal, alloc overlay', [CFG_ACC, 'alloc overlay를 env.accounts로'], [IMPL_SIGNER])
rule('go-wemix/tx/02-*', 'invalid-transaction', 'three-chain', 'config-only', '잔액 초과 거부 (stablenet/14 이식본)')
rule('go-wemix/tx/03-*', 'evm-contract', 'three-chain', 'config-only', 'revert status 0x0 (stablenet/24 이식본)')

out = []
unmatched = []
for r in recs:
    f = r['file'].replace('tests/tc/', '')
    hit = None
    for pat, fam, scope, sep, reason, cfg, impl in R:
        if fnmatch.fnmatch(f, pat) or fnmatch.fnmatch(f, pat + '.json'):
            hit = (fam, scope, sep, reason, cfg, impl); break
    if not hit:
        unmatched.append(f); continue
    fam, scope, sep, reason, cfg, impl = hit
    fl = r['flags']
    cfg = list(cfg); impl = list(impl)
    # dependency-driven additions (mechanical, from the extractor)
    if fl['hardcoded_address_literals'] and CFG_ACC not in cfg and scope != 'stablenet-only':
        cfg.append('주소 literal(%d개)을 label/binding으로' % len(fl['hardcoded_address_literals']))
    if r['env'].get('genesis') and 'genesis overlay를 체인 env로' not in cfg and scope not in ('stablenet-only',):
        cfg.append('genesis overlay/set을 체인 env로 분리')
    if fl['uses_ws'] and 'WS endpoint 프로필' not in cfg: cfg.append('WS endpoint 프로필')
    if r['requires'] and 'requires 유지' not in cfg: cfg.append('requires=' + ','.join(r['requires']))
    out.append({'file': r['file'], 'id': r['id'], 'sha256': r['sha256'], 'source_chain': r['env'].get('chain'),
                'applicableChains': r['applicableChains'], 'requires': r['requires'], 'family': fam, 'common_scope': scope,
                'separation': sep, 'reason': reason, 'configuration_items': cfg, 'implementation_items': impl,
                'dependency_counts': r['dependency_counts'], 'flags': fl, 'methods': r['methods']})
json.dump(out, open(os.path.join(OUT, 'analyses', 'tc-catalog.json'), 'w'), indent=1, ensure_ascii=False)
from collections import Counter
print('classified', len(out), 'unmatched', unmatched)
print(Counter(o['common_scope'] for o in out))
print(Counter((o['common_scope'], o['separation']) for o in out))
print(Counter(o['family'] for o in out).most_common())
