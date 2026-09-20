# 2026-09-09 판정과의 차이 (반대 검토)

같은 파일에 대한 이전 판정(commonality: exact-config / adapter-required / chain-specific)과 이번 판정(공통 범위 + 분리 방식)을 나란히 놓았다. 이번 판정이 정답이라는 뜻이 아니라, 달라진 곳을 드러내 검토하기 위한 표다. 이전 자료는 이전 코드 기준이므로 승계하지 않았다.

| 분석 ID | 파일 | 이전 | 이번 | 차이 이유 |
|---|---|---|---|---|
| TC-002 | basic/02-basic-peers.json | adapter-required | 세 체인 공통 / 설정 분리 | peerCount>=1. 노드 역할 이름(node1..en1)만 프로필로 |
| TC-004 | basic/04-basic-sync.json | adapter-required | 세 체인 공통 / 설정 분리 | sameBlockHash latest |
| TC-034 | go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | adapter-required | StableNet 전용 / 공통 아님 | authorized 계정/헤더 gasTip 정책 |
| TC-048 | go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | adapter-required | 세 체인 공통 / 설정 분리 | genesis 해시 노드 간 일치·parentHash 0 |
| TC-073 | go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | chain-specific | 세 체인 공통 / 설정 분리 + 체인별 기대값 | eth_maxPriorityFeePerGas == tip source. StableNet은 헤더 gasTip과 정확히 같고 다른 체인은 각자의 tip 출처와 비교 |
| TC-085 | go-stablenet/regression/api/20-admin-peers-populated.json | adapter-required | 세 체인 공통 / 설정 분리 | admin_peers (admin namespace 노출 필요). 이미 applicableChains에 wemix 포함 |
| TC-086 | go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | adapter-required | 세 체인 공통 / 설정 분리 | eth_signRawFeeDelegateTransaction 존재. 세 클라이언트 모두 구현(internal/ethapi) |
| TC-089 | go-stablenet/regression/api/24-chain-not-syncing.json | adapter-required | 세 체인 공통 / 설정 분리 | eth_syncing == false |
| TC-109 | go-stablenet/regression/ethereum/17-replacement-tx.json | exact-config | 세 체인 공통 / 설정 분리 + 체인별 기대값 | 동일 nonce 20% 인상 교체. PriceBump 기본값(10%)은 세 클라이언트 동일하나 노드 설정값이므로 프로필로 |
| TC-110 | go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | exact-config | 세 체인 공통 / 설정 분리 + 체인별 기대값 | 동일 nonce 20% 인상 교체. PriceBump 기본값(10%)은 세 클라이언트 동일하나 노드 설정값이므로 프로필로 |
| TC-114 | go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | exact-config | 세 체인 공통 / 설정 분리 + 체인별 기대값 | eth_call revert 오류. 배포 bytecode가 PUSH0(0x5f)을 쓴다. go-wemix EVM은 PUSH0을 실행하지 못하므로 fixture를 체인별로 |
| TC-164 | go-stablenet/topology/01-proxied-pn-routing.json | adapter-required | 세 체인 공통 / 설정 분리 | bp<->pn<->en 계층에서 en 진행. pn 역할이 family마다 구성 가능해야 함 |
| TC-166 | go-stablenet/vocabulary/01-derived-address-and-checksum.json | exact-config | 하네스 전용(체인 무관) / 설정 분리 | 순수 파생(createAddress/contractChecksum). 체인 무관 |
| TC-167 | go-stablenet/vocabulary/02-faucet-funds-account.json | unknown | 세 체인 공통 / 설정 분리 | faucet(node coinbase funder). 공개 RPC에서는 불가 |
| TC-168 | go-stablenet/vocabulary/03-metric-head-block.json | (신규) | 세 체인 공통 / 설정 분리 | 신규 파일 |
| TC-175 | go-wbft/consensus/01-e1-mixed-producers.json | chain-specific | 세 체인 공통 / 별도 구현/처리 | en 노드가 첫 index인 혼합 배치에서 producers == 3. validators 검사는 istanbul_getValidators라 family adapter 필요 |
| TC-177 | go-wbft/network/01-wbft-proxied-routing.json | (신규) | 세 체인 공통 / 설정 분리 | 신규 파일 |
| TC-179 | go-wbft/tx/02-wbft-insufficient-funds-rejected.json | (신규) | 세 체인 공통 / 설정 분리 | 신규 파일 |
| TC-180 | go-wbft/tx/03-wbft-revert-status-zero.json | (신규) | 세 체인 공통 / 설정 분리 | 신규 파일 |
| TC-181 | go-wemix/chain-up/01-wemix-chain-up.json | adapter-required | 세 체인 공통 / 설정 분리 | chainId/blockNumber/peerCount. validators 검사 없음(PoA) |
| TC-182 | go-wemix/chain-up/02-wemix-chain-up-15.json | adapter-required | 세 체인 공통 / 설정 분리 | chainId/blockNumber/peerCount. validators 검사 없음(PoA) |
| TC-187 | go-wemix/tx/02-wemix-insufficient-funds-rejected.json | (신규) | 세 체인 공통 / 설정 분리 | 신규 파일 |
| TC-188 | go-wemix/tx/03-wemix-revert-status-zero.json | (신규) | 세 체인 공통 / 설정 분리 | 신규 파일 |

판정이 달라진 파일 17개(신규 6개 제외).
