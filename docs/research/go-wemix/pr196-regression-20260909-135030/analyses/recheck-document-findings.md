# 문서 수행 후보 독립 재검토

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260909/analyses/existing-tc-specs.md)

문서 수행 후보 47행과 Dual 제외 27행 정적 재검토. 코드/노드 테스트 미실행.

기존 후보 분류를 뒤집을 만한 미지원 오탐은 찾지 못했다. 실행 판정을 잘못 만들 수 있는 명세 누락 2건을 아래에 분리했다. 이는 구현 버그 판정이 아니다. 우선순위 재배치 근거는 발견하지 못했다.

## DOC-RECHECK-01 — Sender 서명 오류 테스트의 결정적 입력 조건 누락

원문의 임의 V/R/S 변경은 다른 유효 sender로 복구될 수 있다. R=0 등 구조적으로 무효한 서명으로 고정해야 invalid-sender 분기를 입증한다.
R=0 등 ValidateSignatureValues가 확실히 거부하는 입력을 고정하고 invalid sender 분기 도달을 확인한다. 임의 비트 변경 후 잔액 부족/feePayer 오류를 서명 오류 검증 성공으로 세지 않는다.

근거: sources/common_test_scenarios.md:33, sources/pr-head/core/types/transaction_signing.go:619, sources/pr-head/core/tx_pool.go:688

## DOC-RECHECK-02 — 온체인 nonce oracle의 블록 태그·초기값 조건 누락

원문의 온체인 전송 수 기대값은 latest/확정 블록과 초기 nonce를 명시해야 성립한다. pending은 pool nonce를 반환하므로 구분 필요.
동일 확정 블록 기준 초기 nonce N 대비 포함된 sender 거래 수 k만큼 N+k를 기대한다. pending 조회는 별도 pool nonce 검증으로 분리한다.

근거: sources/pr-head/internal/ethapi/api.go:1779

## 전체 47행 판정

| ID | 중요도 | 판정 | 근거 | 검토 |
|---|---|---|---|---|
| RT-A-2-04 / TX-010 | P0 | 유지 | sources/pr-head/miner/worker.go:1146 | 역순 제출 후 (blockNumber, transactionIndex)로 sender nonce 순서를 대조하는 명세는 유효. pool 수락만으로 채굴 순서를 판정하면 안 된다. |
| RT-A-1-02 / NODE-003 | P0 | 유지 | sources/pr-head/eth/handler.go:150 | 지원. 동일 높이의 hash/stateRoot 비교 및 실제 full 경로 관측 필요. |
| RT-A-1-06 | P0 | 유지 | sources/pr-head/eth/handler.go:282 | 높이 gap만으로 경로 확정하지 않는 조건 적절. |
| RT-A-1-07 | P0 | 유지 | sources/pr-head/eth/handler.go:288 | 실제 fetcher 진입 관측 조건 적절. |
| RT-A-2-01 / TX-006 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1879 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-02 / TX-003 | P1 | 유지 | sources/pr-head/core/tx_pool.go:655 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-03 / TX-007 | P1 | 유지 | sources/pr-head/core/tx_pool.go:651 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-D-01 / TX-004 | P1 | 유지 | sources/pr-head/core/state_transition.go:198 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-D-03 / TX-014 | P1 | 명세 보완 필요 | sources/pr-head/core/types/transaction_signing.go:619 | 원문의 임의 V/R/S 변경은 다른 유효 sender로 복구될 수 있다. R=0 등 구조적으로 무효한 서명으로 고정해야 invalid-sender 분기를 입증한다. |
| RT-D-04 / TX-015 | P1 | 유지 | sources/pr-head/core/types/transaction_signing.go:204 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-D-05 / TX-016 | P1 | 유지 | sources/pr-head/core/tx_pool.go:713 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-06 / TX-011 | P1 | 유지 | sources/pr-head/core/tx_pool.go:720 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-07 / TX-012 | P1 | 유지 | sources/pr-head/core/tx_pool.go:673 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-08 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1879 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-09 / TX-013 | P1 | 유지 | sources/pr-head/core/tx_list.go:285 | 실제 PriceBump와 두 fee cap 인상을 명시하여 원문 10% 고정 오류를 피했다. 원문은 queued 교체 시나리오이며 pending 교체 전체 커버리지로 확대 해석하지 않는다. |
| RT-A-3-01 / TX-005 | P1 | 유지 | sources/pr-head/core/vm/evm.go:499 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-3-02 | P1 | 유지 | sources/pr-head/core/vm/evm.go:168 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-3-06 / TX-017 | P1 | 유지 | sources/pr-head/core/vm/evm.go:236 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-3-07 / TX-018 | P1 | 유지 | sources/pr-head/core/vm/evm.go:237 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-4-01 / RPC-001 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:738 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-4-03 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1992 | 마이닝 전 txpool 관찰이 필요하므로 이식 시 채굴 제어 또는 nonce gap 등 관측 창 확보. |
| RT-G-1-01 / RPC-002 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:950 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-1-02 / RPC-002 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:970 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-1-04 / RPC-007 | P1 | 유지 | sources/pr-head/internal/ethapi/api.go:1838 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-1-05 / RPC-015 | P1 | 명세 보완 필요 | sources/pr-head/internal/ethapi/api.go:1779 | 원문의 온체인 전송 수 기대값은 latest/확정 블록과 초기 nonce를 명시해야 성립한다. pending은 pool nonce를 반환하므로 구분 필요. |
| RT-A-1-04 / NODE-005 | P1 | 유지 | sources/pr-head/core/blockchain.go:220 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-2-05a | P1 | 유지 | sources/pr-head/core/tx_pool.go:693 | 이미 로컬 DropUnderPriced/EffectiveGasTip과 원격 GasTipCap을 구분했으므로 오탐 아님. |
| RT-A-1-03 / NODE-004 | P1 | 유지 | sources/pr-head/eth/handler.go:368 | 조건부 및 fallback 제외 문구 적절. |
| RT-A-3-03 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1206 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-3-04 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1218 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-3-05 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1170 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-4-02 / RPC-012 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:843 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-4-04 / RPC-014 | P2 | 유지 | sources/pr-head/eth/filters/api.go:327 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-4-06 / RPC-020 | P2 | 유지 | sources/pr-head/eth/filters/api.go:208 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-4-07 / RPC-021 | P2 | 유지 | sources/pr-head/eth/filters/api.go:238 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-1-03 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1798 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-2-01 / RPC-016 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:92 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-2-02 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:104 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-2-03 / RPC-017 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:119 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-4-02 / RPC-018 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:241 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-4-03 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:191 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-1-01 | P2 | 유지 | sources/pr-head/core/genesis.go:239 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-A-1-05 | P2 | 유지 | sources/pr-head/p2p/server.go:584 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RPC-008 | P2 | 유지 | sources/pr-head/eth/api.go:748 | Brioche config·wemixapi.Info 조건 적절. P2 간접 회귀 분류 유지. |
| TC-4-2-02 | P2 | 유지 | sources/pr-head/internal/ethapi/api.go:1218 | 원문 자체가 authorization 없는 일반 전송 baseline이다. 조건부 일반 estimateGas 이식은 정당하며 EIP-7702 지원으로 세지 않는 문구도 적절. |
| RT-A-4-05 / RPC-013 | P3 | 유지 | sources/pr-head/internal/ethapi/api.go:729 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |
| RT-G-5-01 | P3 | 유지 | sources/pr-head/internal/ethapi/api.go:2402 | 지원 코드와 기존 이식 조건에 모순을 발견하지 못함. |

## 제외 항목 대조

- RT-B-01 / WBFT-002 (분석 DOC-D-002)..023: 제외 유지: WBFT 고유 oracle을 PoA로 그대로 적용할 수 없음 (sources/pr-head/eth/ethconfig/config.go:217)
- RT-A-2-10 / TX-008 (분석 DOC-D-024)..025: 제외 유지: Type 0x4 decoder 및 authorization gas 지원 없음 (sources/pr-head/core/types/transaction.go:193)
- TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-009 (분석 DOC-D-027)..029: 제외 유지: 0x100 P256 precompile 미등록. 빈 반환은 지원 증거가 아님 (sources/pr-head/core/vm/contracts.go:84)
