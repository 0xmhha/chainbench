# 3. 레거시 테스트 → DSL 포팅 감사

- 원본: `~/Work/github/packages/chainbench/tests` (셸 460개 파일, 논리 테스트 254개)
- 대상: `~/Work/github/chainbench` (DSL 스펙 122 + 케이스 29 + Go e2e 40)
- 방법: 양쪽을 AST 로 파싱해 그래프로 만든 뒤(1·2번 문서), 테스트 단위로 대응을 찾고 RPC 메서드·단언·검증 대상을 대조했다.

## 3.1 결론

wemix4 를 제외한 170개 중 판정은 다음과 같다.

| 판정 | 개수 | 뜻 |
|---|---:|---|
| 동등 | 124 | 1:1 대응이 있고 검증 대상이 같다 |
| 통합 | 7 | 여러 원본이 하나의 스펙으로 합쳐졌다 (검증 항목은 유지) |
| 부분 | 21 | 대응은 있으나 원본이 보던 것 일부를 보지 않는다 |
| 누락 | 18 | 대응하는 테스트가 없다 |
| **합계** | **170** | |

wemix4 84개는 `docs/dev/wemix4-port-tracker.md` 가 대응표를 갖고 있다. 그 표의 84행을 실제 산출물과 대조한 결과 75행은 대상(스펙 id · Go 테스트 함수)이 실재했고, 나머지 9행은 파일 경로나 산문으로 적혀 있어 자동 대조가 안 됐을 뿐 실물은 있었다. 다만 `TX-018 → tx-errors` 는 그런 이름의 스펙이 없다. 실제 대응은 `specs/gas-policy/out-of-gas-consumes-all` 이므로 표기를 고쳐야 한다.

## 3.2 누락된 테스트

대응이 없는 것은 18건이었다. 모두 stablenet 계열이다.

> **후속 조치 (감사 이후):** 이 18건은 **전부 DSL 스펙으로 작성해 `tests/tc/` 의 해당
> 자리에 넣었다.** 표현할 문법이 없던 것들은 문법을 함께 만들었다.
> 추가한 스펙 목록과 원본 대비 달라진 점은 `tests/tc/README.md` 4.1 절에 있다.

| # | 원본 | 무엇을 검증하던 테스트인가 | 왜 누락됐나 |
|---:|---|---|---|
| 1 | `post-v1.0.0-change/common-all/05-test-burn-expire-refund` (TC-1-1-03) | 소각 제안이 만료되면 환불 가능 잔액이 생기는지 | `burn-cancel` · `burn-reject` 경로만 포팅됐고 만료 경로가 없다 |
| 2 | `post-v1.0.0-change/common-all/17-test-estimategas-authorizationlist-cost` (TC-4-2-01, TC-4-2-03) | EIP-7702 authorizationList 1건·2건의 estimateGas 증가분 | DSL 의 `estimateGas` 단언에 authorizationList 인자가 없다 |
| 3 | `post-v1.0.0-change/common-all/19-test-upgrade-registry` (TC-5-2-01, TC-5-2-02, TC-5-2-03) | 하드포크 업그레이드 등록 순서·시점 (CollectUpgrades) | 대응 스펙이 없다 |
| 4 | `post-v1.0.0-change/common-all/20-test-v1-params-init` (TC-5-2-04) | v1 시스템 컨트랙트 params 초기값 (스토리지 슬롯 직접 조회) | `eth_getStorageAt` 을 쓰는 스펙이 하나도 없다 |
| 5 | `post-v1.0.0-change/extra-state/03-test-extra-union-merge` (TC-4-5-05,TC-4-5-06) | alloc.Extra 와 GovCouncil params 가 서로 다른 비트를 줄 때 합집합이 중복 없이 병합되는지 | 개별 비트 동기화만 포팅됐고 합집합·중복 제거가 없다 |
| 6 | `post-v1.0.0-change/extra-state/06-test-invalid-extra-reject` (TC-4-5-09) | 미정의 Extra 비트를 가진 genesis 로는 노드가 부팅되지 않아야 함 | 노드 부팅 실패를 단언하는 verb 가 없다 |
| 7 | `post-v1.0.0-change/stand-alone/02-test-genesis-mismatch` (TC-4-1-03) | genesis 가 다른 바이너리로 교체하면 GenesisMismatchError 로 기동 실패해야 함 | 부팅 실패 음성 테스트를 표현할 verb 가 없다 |
| 8 | `post-v1.0.0-change/stand-alone/03-test-unsupported-version` (TC-5-2-06) | 미지원 시스템 컨트랙트 버전이면 BohoBlock 커밋이 실패해야 함 | 같음 |
| 9 | `post-v1.0.0-change/stand-alone/04-test-testnet-genesis-hash` (TC-5-3-01) | 블록 0 해시가 릴리스 고정값과 일치하는지 | 고정 해시를 비교하는 스펙이 없다 |
| 10 | `post-v1.0.0-change/string-handling/01-test-authorized-no-space` (TC-4-3-01) | GovCouncil authorizedAccounts 문자열 파싱(splitAndTrim) 결과 개수 | 6건 모두 대응 스펙이 없다. `authorizedAccountCount` 를 읽는 스펙이 하나도 없다 |
| 11 | `post-v1.0.0-change/string-handling/02-test-authorized-space` (TC-4-3-02) | GovCouncil authorizedAccounts 문자열 파싱(splitAndTrim) 결과 개수 | 6건 모두 대응 스펙이 없다. `authorizedAccountCount` 를 읽는 스펙이 하나도 없다 |
| 12 | `post-v1.0.0-change/string-handling/03-test-authorized-trim` (TC-4-3-03) | GovCouncil authorizedAccounts 문자열 파싱(splitAndTrim) 결과 개수 | 6건 모두 대응 스펙이 없다. `authorizedAccountCount` 를 읽는 스펙이 하나도 없다 |
| 13 | `post-v1.0.0-change/string-handling/04-test-authorized-empty-item` (TC-4-3-04) | GovCouncil authorizedAccounts 문자열 파싱(splitAndTrim) 결과 개수 | 6건 모두 대응 스펙이 없다. `authorizedAccountCount` 를 읽는 스펙이 하나도 없다 |
| 14 | `post-v1.0.0-change/string-handling/05-test-authorized-single` (TC-4-3-05) | GovCouncil authorizedAccounts 문자열 파싱(splitAndTrim) 결과 개수 | 6건 모두 대응 스펙이 없다. `authorizedAccountCount` 를 읽는 스펙이 하나도 없다 |
| 15 | `post-v1.0.0-change/string-handling/06-test-authorized-empty` (TC-4-3-06) | GovCouncil authorizedAccounts 문자열 파싱(splitAndTrim) 결과 개수 | 6건 모두 대응 스펙이 없다. `authorizedAccountCount` 를 읽는 스펙이 하나도 없다 |
| 16 | `regression/blacklist-authorized/05-test-zero-address` (RT-E-05) | 0x0 주소로의 전송이 거부되는지 | 대응 스펙이 없다 |
| 17 | `regression/blacklist-authorized/06-test-precompile-transfer` (RT-E-06) | 프리컴파일·시스템 주소로의 전송이 차단되는지 | 대응 스펙이 없다 |
| 18 | `regression/wbft/05-test-remove-validator` (RT-B-05) | GovValidator 에서 멤버를 제거하면 에폭 경계에서 검증자 집합이 줄어드는지 | `validator-add-member-executes` 로 추가만 포팅했고 제거 방향이 없다 |

## 3.3 부분 포팅 (검증 범위가 줄어든 것)

21건이다. 대응은 있으나 원본이 보던 것 중 일부를 보지 않는다.

| 원본 | 대응 | 줄어든 부분 |
|---|---|---|
| `post-v1.0.0-change/common-all/02-test-v2-code-only-hash` | specs/hardfork/govminter-v2-code | 원본은 바이트코드 해시와 잔액 보존을 함께 봤다. 대응은 코드 존재·형태 위주다 |
| `post-v1.0.0-change/common-all/10-test-govminter-bytecode-hash-state-preserved` | specs/hardfork/govminter-v2-code + prealloc-preserved-across-boho | 원본은 `eth_getStorageAt` 으로 스토리지 보존까지 확인했다. 대응은 잔액·nonce 보존까지다 |
| `post-v1.0.0-change/common-all/18-test-estimategas-no-authorizationlist` | specs/api/estimate-gas (기준선만) | 원본은 authorizationList 없는 경우를 기준선으로 삼아 있는 경우와 비교했다. 대응은 단독 하한만 본다 |
| `post-v1.0.0-change/effectivegasprice/01-test-authorized-account` | specs/gas-policy/authorized-account-gastip-free | 원본은 BP 와 snap-sync EN 두 노드의 값이 같은지를 봤다. 대응은 단일 노드다 |
| `post-v1.0.0-change/effectivegasprice/02-test-regular-account` | specs/gas-policy/effective-gas-price-regular | 같음 (노드 간 일치 미검증) |
| `post-v1.0.0-change/effectivegasprice/03-test-auth-tx-event-last` | specs/system-contracts/authorized-tx-executed-event | 원본은 이벤트가 로그의 *마지막*인지를 봤다. 대응은 이벤트 존재 여부다 |
| `post-v1.0.0-change/extra-state/02-test-extra-contract-to-alloc` | cases/stablenet-account-extra | 원본은 컨트랙트 → alloc 방향 동기화를 별도로 봤다. 대응 케이스는 한 방향만 확인한다 |
| `post-v1.0.0-change/extra-state/07-test-extra-v1-to-v2-delayed` | cases/stablenet-delayed-fork | 원본은 지연 활성화 전후로 계정 상태가 보존되는지를 3개 케이스로 봤다. 대응은 코드 교체 확인 위주다 |
| `post-v1.0.0-change/stand-alone/01-test-signature-compat-sync` | e2e TestE2E_StablenetHardforkSwap | 원본은 교체 전 tx 의 blockNumber·status·from·to 4개 필드 보존과 로그 수집까지 봤다. 대응은 체인 진행·상태 보존 위주다 |
| `regression/api/21-test-sign-raw-fee-delegate` | specs/accounts/fee-delegate-sign-rpc-present | 원본은 `personal_signRawFeeDelegateTransaction` 으로 실제 서명까지 했다. 대응은 RPC 존재 여부만 본다 |
| `regression/ethereum/01-test-genesis-init` | cases/stablenet-chain-up + specs/consensus/chain-id | 원본은 전체 노드의 블록 0 해시가 모두 같은지까지 봤다. 대응은 chainId 와 기동 확인 위주다 |
| `regression/ethereum/10-test-access-list-tx` | specs/accounts/access-list-tx | 원본은 `eth_createAccessList` 로 접근 목록을 만들어 붙였다. 대응 스펙은 접근 목록을 직접 구성한다 |
| `regression/fee-delegation/01-test-fee-delegate-normal` | specs/accounts/fee-delegated-transfer | 원본은 노드의 `personal_*` API 로 서명·언락했다. 대응은 로컬 서명 경로다. 노드 측 personal API 는 더 이상 검증되지 않는다 |
| `regression/fee-delegation/02-test-sender-sig-invalid` | specs/accounts/fd-sender-sig-invalid-rejected | 같음 (personal API 경로 미검증) |
| `regression/fee-delegation/03-test-feepayer-sig-invalid` | specs/accounts/fd-feepayer-sig-invalid-rejected | 같음 |
| `regression/fee-delegation/04-test-feepayer-insufficient` | specs/accounts/feepayer-insufficient-rejected | 같음 |
| `regression/wbft/04-test-add-validator` | specs/system-contracts/validator-add-member-executes | 원본은 추가 후 에폭 경계에서 검증자 집합이 실제로 커지는지까지 봤다. 대응은 멤버 등록 성공까지다 |
| `remote/balance-check` | attach 모드 (chainbench network attach) | attach 모드로 대체됐다. 전용 회귀 케이스는 없다 |
| `remote/chain-info` | attach 모드 | 같음 |
| `remote/rpc-health` | attach 모드 | 같음 |
| `remote/tx-send` | attach 모드 | 같음 |

## 3.4 통합된 것 (검증 항목은 유지)

| 원본 | 통합 대상 |
|---|---|
| `regression/ethereum/13-test-feecap-underpriced` | specs/accounts/dynamic-fee-below-basefee-rejected |
| `regression/ethereum/20-test-contract-call` | specs/accounts/contract-roundtrip |
| `regression/ethereum/21-test-eth-call-view` | specs/accounts/contract-roundtrip |
| `regression/system-contracts/21-test-address-blacklisted-event` | specs/system-contracts/blacklist-proposal-executes (이벤트 단언 포함) |
| `regression/system-contracts/24-test-authorized-account-removed-event` | specs/system-contracts/unauthorize-proposal-executes (이벤트 단언 포함) |
| `regression/wbft/11-test-prev-committed-seal` | specs/consensus/prev-seals-quorum |
| `regression/wbft/12-test-prev-prepared-seal` | specs/consensus/prev-seals-quorum |

## 3.5 전체 대응표 (wemix4 제외)

| # | 카테고리 | 원본 | ID | 대응 | 판정 |
|---:|---|---|---|---|---|
| 1 | `basic` | `consensus` | — | cases/basic-consensus | 동등 |
| 2 | `basic` | `peers` | — | cases/basic-peers | 동등 |
| 3 | `basic` | `rpc-health` | — | cases/basic-rpc-health | 동등 |
| 4 | `basic` | `sync` | — | cases/basic-sync | 동등 |
| 5 | `basic` | `tx-send` | — | cases/basic-tx-send | 동등 |
| 6 | `basic` | `txpool-propagation` | — | cases/basic-txpool-propagation | 동등 |
| 7 | `basic` | `wbft-consensus` | — | cases/basic-wbft-consensus | 동등 |
| 8 | `fault` | `network-partition` | — | cases/fault-network-partition | 동등 |
| 9 | `fault` | `node-crash` | — | cases/fault-node-crash | 동등 |
| 10 | `fault` | `node-recover` | — | cases/fault-node-recover | 동등 |
| 11 | `fault` | `p2p-topology` | — | cases/fault-p2p-topology | 동등 |
| 12 | `fault` | `two-down` | — | cases/fault-two-down | 동등 |
| 13 | `fault` | `txpool-leader-change` | — | cases/fault-txpool-leader-change | 동등 |
| 14 | `post-v1.0.0-change/common-all` | `01-test-inject-contracts-delayed` | TC-4-4-04 | cases/stablenet-delayed-fork | 동등 |
| 15 | `post-v1.0.0-change/common-all` | `02-test-v2-code-only-hash` | TC-5-2-05 | specs/hardfork/govminter-v2-code | 부분 |
| 16 | `post-v1.0.0-change/common-all` | `03-test-burn-cancel-refund` | TC-1-1-01, TC-1-1-10 | specs/system-contracts/burn-cancel-refundable | 동등 |
| 17 | `post-v1.0.0-change/common-all` | `04-test-burn-reject-refund` | TC-1-1-02 | specs/system-contracts/burn-reject-refundable | 동등 |
| 18 | `post-v1.0.0-change/common-all` | `05-test-burn-expire-refund` | TC-1-1-03 | — 없음 (만료 경로 미포팅) | 누락 |
| 19 | `post-v1.0.0-change/common-all` | `06-test-burn-execute-no-refund` | TC-1-1-04 | specs/system-contracts/burn-execute-no-refundable | 동등 |
| 20 | `post-v1.0.0-change/common-all` | `07-test-claim-refund-success` | TC-1-1-05, TC-1-1-09 | specs/system-contracts/claim-burn-refund-succeeds | 동등 |
| 21 | `post-v1.0.0-change/common-all` | `08-test-claim-refund-zero-revert` | TC-1-1-06 | specs/system-contracts/claim-zero-refund-reverts | 동등 |
| 22 | `post-v1.0.0-change/common-all` | `09-test-claim-refund-double-revert` | TC-1-1-07 | specs/system-contracts/claim-burn-refund-double-reverts | 동등 |
| 23 | `post-v1.0.0-change/common-all` | `10-test-govminter-bytecode-hash-state-preserved` | TC-1-1-11, TC-1-1-12 | specs/hardfork/govminter-v2-code + prealloc-preserved-across-boho | 부분 |
| 24 | `post-v1.0.0-change/common-all` | `11-test-all-secp256r1-precompile` | TC-1-2-01, TC-1-2-02, TC-1-2-03, TC-1-2-04, TC-1-2-05, TC-1-2-06 | specs/accounts/secp256r1-precompile-{valid,invalid,short-input} | 동등 |
| 25 | `post-v1.0.0-change/common-all` | `12-test-legacy-gasprice-below-min-revert` | TC-1-3-04 | specs/gas-policy/legacy-gasprice-below-min-rejected | 동등 |
| 26 | `post-v1.0.0-change/common-all` | `13-test-accesslist-gasprice-below-min-revert` | TC-1-3-05 | specs/gas-policy/accesslist-gasprice-below-min-rejected | 동등 |
| 27 | `post-v1.0.0-change/common-all` | `14-test-dynamic-fee-tipcap-below-min-revert` | TC-1-3-06 | specs/gas-policy/feecap-below-min-rejected | 동등 |
| 28 | `post-v1.0.0-change/common-all` | `15-test-chain-config-boho` | TC-4-1-01 | specs/hardfork/boho-chain-config-active | 동등 |
| 29 | `post-v1.0.0-change/common-all` | `16-test-boho-chain-config-activation` | TC-4-1-02 | specs/hardfork/anzeon-active-before-boho | 동등 |
| 30 | `post-v1.0.0-change/common-all` | `17-test-estimategas-authorizationlist-cost` | TC-4-2-01, TC-4-2-03 | — 없음 (authorizationList estimateGas 미포팅) | 누락 |
| 31 | `post-v1.0.0-change/common-all` | `18-test-estimategas-no-authorizationlist` | TC-4-2-02 | specs/api/estimate-gas (기준선만) | 부분 |
| 32 | `post-v1.0.0-change/common-all` | `19-test-upgrade-registry` | TC-5-2-01, TC-5-2-02, TC-5-2-03 | — 없음 (CollectUpgrades 등록 순서 미포팅) | 누락 |
| 33 | `post-v1.0.0-change/common-all` | `20-test-v1-params-init` | TC-5-2-04 | — 없음 (eth_getStorageAt 슬롯 검증 미포팅) | 누락 |
| 34 | `post-v1.0.0-change/effectivegasprice` | `01-test-authorized-account` | TC-4-6-01 | specs/gas-policy/authorized-account-gastip-free | 부분 |
| 35 | `post-v1.0.0-change/effectivegasprice` | `02-test-regular-account` | TC-4-6-02 | specs/gas-policy/effective-gas-price-regular | 부분 |
| 36 | `post-v1.0.0-change/effectivegasprice` | `03-test-auth-tx-event-last` | TC-4-6-04 | specs/system-contracts/authorized-tx-executed-event | 부분 |
| 37 | `post-v1.0.0-change/extra-state` | `01-test-extra-alloc-to-contract` | TC-4-5-01,TC-4-5-02 | specs/system-contracts/authorized-extra-bit-synced, blacklisted-extra-bit-synced | 동등 |
| 38 | `post-v1.0.0-change/extra-state` | `02-test-extra-contract-to-alloc` | TC-4-5-03,TC-4-5-04 | cases/stablenet-account-extra | 부분 |
| 39 | `post-v1.0.0-change/extra-state` | `03-test-extra-union-merge` | TC-4-5-05,TC-4-5-06 | — 없음 (합집합 중복 제거 미검증) | 누락 |
| 40 | `post-v1.0.0-change/extra-state` | `04-test-extra-dual-status` | TC-4-5-07 | specs/system-contracts/dual-status-extra | 동등 |
| 41 | `post-v1.0.0-change/extra-state` | `05-test-extra-balance-preserved` | TC-4-5-08 | specs/system-contracts/extra-balance-preserved | 동등 |
| 42 | `post-v1.0.0-change/extra-state` | `06-test-invalid-extra-reject` | TC-4-5-09 | — 없음 (부팅 실패 음성 테스트 미포팅) | 누락 |
| 43 | `post-v1.0.0-change/extra-state` | `07-test-extra-v1-to-v2-delayed` | TC-4-5-10,TC-4-5-11,TC-4-5-12 | cases/stablenet-delayed-fork | 부분 |
| 44 | `post-v1.0.0-change/stand-alone` | `01-test-signature-compat-sync` | TC-3-1-04 | e2e TestE2E_StablenetHardforkSwap | 부분 |
| 45 | `post-v1.0.0-change/stand-alone` | `02-test-genesis-mismatch` | TC-4-1-03 | — 없음 (GenesisMismatch 음성 테스트 미포팅) | 누락 |
| 46 | `post-v1.0.0-change/stand-alone` | `03-test-unsupported-version` | TC-5-2-06 | — 없음 (미지원 버전 커밋 실패 미포팅) | 누락 |
| 47 | `post-v1.0.0-change/stand-alone` | `04-test-testnet-genesis-hash` | TC-5-3-01 | — 없음 (genesis hash 고정값 검증 미포팅) | 누락 |
| 48 | `post-v1.0.0-change/string-handling` | `01-test-authorized-no-space` | TC-4-3-01 | — 없음 | 누락 |
| 49 | `post-v1.0.0-change/string-handling` | `02-test-authorized-space` | TC-4-3-02 | — 없음 | 누락 |
| 50 | `post-v1.0.0-change/string-handling` | `03-test-authorized-trim` | TC-4-3-03 | — 없음 | 누락 |
| 51 | `post-v1.0.0-change/string-handling` | `04-test-authorized-empty-item` | TC-4-3-04 | — 없음 | 누락 |
| 52 | `post-v1.0.0-change/string-handling` | `05-test-authorized-single` | TC-4-3-05 | — 없음 | 누락 |
| 53 | `post-v1.0.0-change/string-handling` | `06-test-authorized-empty` | TC-4-3-06 | — 없음 | 누락 |
| 54 | `regression/anzeon` | `01-test-regular-account-gastip-forced` | RT-C-01 | specs/gas-policy/regular-account-gastip-forced | 동등 |
| 55 | `regression/anzeon` | `02-test-authorized-account-gastip-free` | RT-C-02 | specs/gas-policy/authorized-account-gastip-free | 동등 |
| 56 | `regression/anzeon` | `03-test-basefee-increase` | RT-C-03 | cases/anzeon-basefee-increase | 동등 |
| 57 | `regression/anzeon` | `04-test-basefee-stable` | RT-C-04 | cases/anzeon-basefee-stable | 동등 |
| 58 | `regression/anzeon` | `05-test-basefee-decrease` | RT-C-05 | cases/anzeon-basefee-decrease | 동등 |
| 59 | `regression/anzeon` | `06-test-min-basefee` | RT-C-06 | specs/gas-policy/basefee-minimum | 동등 |
| 60 | `regression/anzeon` | `07-test-max-basefee` | RT-C-07 | specs/gas-policy/basefee-maximum | 동등 |
| 61 | `regression/api` | `01-test-get-block-by-number` | RT-G-1-01 | specs/api/block-transactions-field | 동등 |
| 62 | `regression/api` | `02-test-get-block-by-hash` | RT-G-1-02 | specs/api/block-by-hash-consistency | 동등 |
| 63 | `regression/api` | `03-test-get-tx-by-hash` | RT-G-1-03 | specs/accounts/transaction-by-hash-fields | 동등 |
| 64 | `regression/api` | `04-test-get-tx-receipt` | RT-G-1-04 | specs/accounts/transaction-receipt-fields | 동등 |
| 65 | `regression/api` | `05-test-get-tx-count` | RT-G-1-05 | specs/accounts/transaction-count-increments | 동등 |
| 66 | `regression/api` | `06-test-get-code-system` | RT-G-1-06 | specs/system-contracts/system-contracts-deployed | 동등 |
| 67 | `regression/api` | `07-test-gas-price` | RT-G-2-01 | specs/api/gas-price-positive + gas-policy/gas-price-equals-basefee-plus-tip | 동등 |
| 68 | `regression/api` | `08-test-max-priority-fee` | RT-G-2-02 | specs/gas-policy/max-priority-fee-equals-gastip | 동등 |
| 69 | `regression/api` | `09-test-fee-history` | RT-G-2-03 | specs/api/fee-history-well-formed | 동등 |
| 70 | `regression/api` | `10-test-estimate-system-call` | RT-G-2-04 | specs/gas-policy/estimate-gas-token-transfer | 동등 |
| 71 | `regression/api` | `11-test-node-address` | RT-G-3-01 | specs/consensus/node-address-returned | 동등 |
| 72 | `regression/api` | `12-test-get-validators` | RT-G-3-02 | specs/consensus/validator-set-nonempty, validator-set-count | 동등 |
| 73 | `regression/api` | `13-test-get-commit-signers` | RT-G-3-03 | specs/consensus/commit-signers-quorum | 동등 |
| 74 | `regression/api` | `14-test-get-wbft-extra` | RT-G-3-04 | specs/consensus/wbft-extra-info-fields | 동등 |
| 75 | `regression/api` | `15-test-istanbul-status` | RT-G-3-05 | specs/consensus/istanbul-status-fields | 동등 |
| 76 | `regression/api` | `16-test-is-validator` | RT-G-3-06 | specs/consensus/is-validator-flags | 동등 |
| 77 | `regression/api` | `17-test-net-peer-count` | RT-G-4-01 | cases/basic-peers | 동등 |
| 78 | `regression/api` | `18-test-txpool-status` | RT-G-4-02 | specs/api/txpool-status | 동등 |
| 79 | `regression/api` | `19-test-txpool-content` | RT-G-4-03 | specs/api/txpool-content-well-formed | 동등 |
| 80 | `regression/api` | `20-test-admin-peers` | RT-G-4-04 | specs/network/admin-peers-populated | 동등 |
| 81 | `regression/api` | `21-test-sign-raw-fee-delegate` | RT-G-5-01 | specs/accounts/fee-delegate-sign-rpc-present | 부분 |
| 82 | `regression/api` | `22-test-total-supply` | RT-G-5-02 | specs/system-contracts/token-total-supply-readable | 동등 |
| 83 | `regression/api` | `23-test-allowance` | RT-G-5-03 | specs/system-contracts/token-approve-sets-allowance | 동등 |
| 84 | `regression/blacklist-authorized` | `01-test-sender-blacklisted` | RT-E-01 | specs/system-contracts/sender-blacklisted-rejected | 동등 |
| 85 | `regression/blacklist-authorized` | `02-test-recipient-blacklisted` | RT-E-02 | specs/system-contracts/recipient-blacklisted-rejected | 동등 |
| 86 | `regression/blacklist-authorized` | `03-test-feepayer-blacklisted` | RT-E-03 | specs/system-contracts/feepayer-blacklisted-rejected | 동등 |
| 87 | `regression/blacklist-authorized` | `04-test-unblacklist` | RT-E-04 | specs/system-contracts/address-unblacklisted-event | 동등 |
| 88 | `regression/blacklist-authorized` | `05-test-zero-address` | RT-E-05 | — 없음 | 누락 |
| 89 | `regression/blacklist-authorized` | `06-test-precompile-transfer` | RT-E-06 | — 없음 | 누락 |
| 90 | `regression/blacklist-authorized` | `07-test-is-blacklisted` | RT-E-07 | specs/system-contracts/account-blacklist-readable | 동등 |
| 91 | `regression/blacklist-authorized` | `08-test-is-authorized` | RT-E-08 | specs/system-contracts/account-authorization-readable | 동등 |
| 92 | `regression/blacklist-authorized` | `09-test-authorized-tx-executed` | RT-E-09 | specs/system-contracts/authorized-tx-executed-event | 동등 |
| 93 | `regression/ethereum` | `01-test-genesis-init` | RT-A-1-01 | cases/stablenet-chain-up + specs/consensus/chain-id | 부분 |
| 94 | `regression/ethereum` | `02-test-full-sync` | RT-A-1-02 | e2e TestE2E_StablenetSyncGap | 동등 |
| 95 | `regression/ethereum` | `03-test-snap-sync` | RT-A-1-03 | e2e TestE2E_StablenetSyncGap / TestE2E_WbftSnapSync | 동등 |
| 96 | `regression/ethereum` | `04-test-node-restart` | RT-A-1-04 | cases/fault-node-recover + e2e ConsensusLifecycle | 동등 |
| 97 | `regression/ethereum` | `05-test-p2p-peers` | RT-A-1-05 | specs/network/admin-peers-populated + cases/basic-peers | 동등 |
| 98 | `regression/ethereum` | `06-test-downloader-path` | RT-A-1-06 | e2e TestE2E_StablenetSyncGap | 동등 |
| 99 | `regression/ethereum` | `07-test-block-fetcher-path` | RT-A-1-07 | e2e TestE2E_StablenetBlockPropagation | 동등 |
| 100 | `regression/ethereum` | `08-test-legacy-tx` | RT-A-2-01 | specs/accounts/legacy-transfer | 동등 |
| 101 | `regression/ethereum` | `09-test-dynamic-fee-tx` | RT-A-2-02 | specs/accounts/dynamic-fee-tx | 동등 |
| 102 | `regression/ethereum` | `10-test-access-list-tx` | RT-A-2-03 | specs/accounts/access-list-tx | 부분 |
| 103 | `regression/ethereum` | `11-test-nonce-ordering` | RT-A-2-04 | specs/accounts/nonce-ordering, out-of-order-nonces-mine | 동등 |
| 104 | `regression/ethereum` | `12-test-tipcap-underpriced` | RT-A-2-05a | specs/accounts/dynamic-fee-below-basefee-rejected | 동등 |
| 105 | `regression/ethereum` | `13-test-feecap-underpriced` | RT-A-2-05b | specs/accounts/dynamic-fee-below-basefee-rejected | 통합 |
| 106 | `regression/ethereum` | `14-test-insufficient-funds` | RT-A-2-06 | specs/accounts/insufficient-funds-rejected | 동등 |
| 107 | `regression/ethereum` | `15-test-gaslimit-exceeded` | RT-A-2-07 | specs/accounts/gas-limit-exceeds-block-rejected | 동등 |
| 108 | `regression/ethereum` | `16-test-effective-gas-price` | RT-A-2-08 | specs/accounts/effective-gas-price | 동등 |
| 109 | `regression/ethereum` | `17-test-replacement-tx` | RT-A-2-09 | specs/accounts/replacement-tx, same-nonce-replacement | 동등 |
| 110 | `regression/ethereum` | `18-test-setcode-tx` | RT-A-2-10 | specs/accounts/set-code-delegation | 동등 |
| 111 | `regression/ethereum` | `19-test-contract-deploy` | RT-A-3-01 | specs/accounts/contract-roundtrip | 동등 |
| 112 | `regression/ethereum` | `20-test-contract-call` | RT-A-3-02 | specs/accounts/contract-roundtrip | 통합 |
| 113 | `regression/ethereum` | `21-test-eth-call-view` | RT-A-3-03 | specs/accounts/contract-roundtrip | 통합 |
| 114 | `regression/ethereum` | `22-test-estimate-gas` | RT-A-3-04 | specs/api/estimate-gas | 동등 |
| 115 | `regression/ethereum` | `23-test-eth-call-revert` | RT-A-3-05 | specs/accounts/eth-call-revert-returns-error | 동등 |
| 116 | `regression/ethereum` | `24-test-revert-tx` | RT-A-3-06 | specs/gas-policy/revert-tx-status-zero | 동등 |
| 117 | `regression/ethereum` | `25-test-out-of-gas` | RT-A-3-07 | specs/gas-policy/out-of-gas-consumes-all | 동등 |
| 118 | `regression/ethereum` | `26-test-eth-block-number` | RT-A-4-01 | cases/basic-rpc-health | 동등 |
| 119 | `regression/ethereum` | `27-test-eth-get-balance` | RT-A-4-02 | specs/accounts/genesis-balance, value-transfer | 동등 |
| 120 | `regression/ethereum` | `28-test-send-raw-tx` | RT-A-4-03 | specs/accounts/value-transfer | 동등 |
| 121 | `regression/ethereum` | `29-test-eth-get-logs` | RT-A-4-04 | specs/api/logs-query-well-formed | 동등 |
| 122 | `regression/ethereum` | `30-test-eth-chain-id` | RT-A-4-05 | specs/consensus/chain-id | 동등 |
| 123 | `regression/ethereum` | `31-test-ws-subscribe-heads` | RT-A-4-06 | specs/api/ws-subscribe-new-heads | 동등 |
| 124 | `regression/ethereum` | `32-test-ws-subscribe-logs` | RT-A-4-07 | specs/api/ws-subscribe-logs | 동등 |
| 125 | `regression/fee-delegation` | `01-test-fee-delegate-normal` | RT-D-01 | specs/accounts/fee-delegated-transfer | 부분 |
| 126 | `regression/fee-delegation` | `02-test-sender-sig-invalid` | RT-D-03 | specs/accounts/fd-sender-sig-invalid-rejected | 부분 |
| 127 | `regression/fee-delegation` | `03-test-feepayer-sig-invalid` | RT-D-04 | specs/accounts/fd-feepayer-sig-invalid-rejected | 부분 |
| 128 | `regression/fee-delegation` | `04-test-feepayer-insufficient` | RT-D-05 | specs/accounts/feepayer-insufficient-rejected | 부분 |
| 129 | `regression/system-contracts` | `01-test-native-transfer` | RT-F-1-01 | specs/system-contracts/native-coin-adapter-code + token-transfer-emits-event | 동등 |
| 130 | `regression/system-contracts` | `02-test-balance-of` | RT-F-1-02 | specs/system-contracts/token-balance-readable | 동등 |
| 131 | `regression/system-contracts` | `03-test-approve-transferfrom` | RT-F-1-03 | specs/system-contracts/token-approve-sets-allowance, token-transfer-from-moves-balance | 동등 |
| 132 | `regression/system-contracts` | `04-test-mint-transfer-event` | RT-F-1-04 | specs/system-contracts/mint-transfer-event | 동등 |
| 133 | `regression/system-contracts` | `05-test-burn-transfer-event` | RT-F-1-05 | specs/system-contracts/burn-transfer-event | 동등 |
| 134 | `regression/system-contracts` | `06-test-mint-proposal` | RT-F-2-01 | specs/system-contracts/mint-proposal-executes | 동등 |
| 135 | `regression/system-contracts` | `07-test-burn-proposal` | RT-F-2-02 | specs/system-contracts/burn-proposal-executes | 동등 |
| 136 | `regression/system-contracts` | `08-test-quorum-deficient` | RT-F-2-03 | specs/system-contracts/quorum-deficient-stays-voting | 동등 |
| 137 | `regression/system-contracts` | `09-test-validator-metadata` | RT-F-3-04 | specs/system-contracts/validator-metadata-readable | 동등 |
| 138 | `regression/system-contracts` | `10-test-propose-gastip` | RT-F-3-05 | specs/system-contracts/gastip-governance-updates-header | 동등 |
| 139 | `regression/system-contracts` | `11-test-proposal-expiry` | RT-F-3-06 | specs/system-contracts/proposal-expiry-transitions + e2e ProposalExpiry | 동등 |
| 140 | `regression/system-contracts` | `12-test-add-minter` | RT-F-4-01 | specs/system-contracts/configure-minter-proposal-executes | 동등 |
| 141 | `regression/system-contracts` | `13-test-remove-minter` | RT-F-4-02 | specs/system-contracts/remove-minter-executes | 동등 |
| 142 | `regression/system-contracts` | `14-test-masterminter-self-member` | RT-F-4-03 | specs/system-contracts/masterminter-member-add-remove | 동등 |
| 143 | `regression/system-contracts` | `15-test-non-member-rejected` | RT-F-4-04 | specs/system-contracts/non-member-configure-minter-rejected | 동등 |
| 144 | `regression/system-contracts` | `16-test-blacklist` | RT-F-5-01 | specs/system-contracts/blacklist-proposal-executes | 동등 |
| 145 | `regression/system-contracts` | `17-test-unblacklist` | — | specs/system-contracts/address-unblacklisted-event | 동등 |
| 146 | `regression/system-contracts` | `18-test-authorize` | RT-F-5-03 | specs/system-contracts/authorize-proposal-executes | 동등 |
| 147 | `regression/system-contracts` | `19-test-unauthorize` | — | specs/system-contracts/unauthorize-proposal-executes | 동등 |
| 148 | `regression/system-contracts` | `20-test-direct-blacklist-rejected` | RT-F-5-05 | specs/system-contracts/direct-blacklist-call-rejected | 동등 |
| 149 | `regression/system-contracts` | `21-test-address-blacklisted-event` | RT-F-5-06 | specs/system-contracts/blacklist-proposal-executes (이벤트 단언 포함) | 통합 |
| 150 | `regression/system-contracts` | `22-test-address-unblacklisted-event` | RT-F-5-07 | specs/system-contracts/address-unblacklisted-event | 동등 |
| 151 | `regression/system-contracts` | `23-test-authorized-account-added-event` | RT-F-5-08 | specs/system-contracts/authorized-account-added-event | 동등 |
| 152 | `regression/system-contracts` | `24-test-authorized-account-removed-event` | RT-F-5-09 | specs/system-contracts/unauthorize-proposal-executes (이벤트 단언 포함) | 통합 |
| 153 | `regression/wbft` | `01-test-block-period` | RT-B-01 | specs/consensus/block-period-one-second | 동등 |
| 154 | `regression/wbft` | `02-test-wbft-extra-seal` | RT-B-02 | specs/consensus/wbft-seals-quorum | 동등 |
| 155 | `regression/wbft` | `03-test-epoch-transition` | RT-B-03 | specs/consensus/epoch-transition-carries-epoch-info | 동등 |
| 156 | `regression/wbft` | `04-test-add-validator` | RT-B-04 | specs/system-contracts/validator-add-member-executes | 부분 |
| 157 | `regression/wbft` | `05-test-remove-validator` | RT-B-05 | — (add 만 있고 remove 없음) | 누락 |
| 158 | `regression/wbft` | `06-test-gastip-header-sync` | RT-B-06 | specs/system-contracts/gastip-governance-updates-header | 동등 |
| 159 | `regression/wbft` | `07-test-istanbul-get-validators` | RT-B-07 | specs/consensus/validator-set-nonempty | 동등 |
| 160 | `regression/wbft` | `08-test-quorum-deficient` | RT-B-08 | e2e TestE2E_WbftFaultHalt / WbftQuorum* | 동등 |
| 161 | `regression/wbft` | `09-test-round-change` | RT-B-09 | e2e TestE2E_WbftViewChange | 동등 |
| 162 | `regression/wbft` | `10-test-post-round-change` | RT-B-10 | e2e TestE2E_WbftRoundRobinProposer | 동등 |
| 163 | `regression/wbft` | `11-test-prev-committed-seal` | RT-B-11 | specs/consensus/prev-seals-quorum | 통합 |
| 164 | `regression/wbft` | `12-test-prev-prepared-seal` | RT-B-12 | specs/consensus/prev-seals-quorum | 통합 |
| 165 | `remote` | `balance-check` | — | attach 모드 (chainbench network attach) | 부분 |
| 166 | `remote` | `chain-info` | — | attach 모드 | 부분 |
| 167 | `remote` | `rpc-health` | — | attach 모드 | 부분 |
| 168 | `remote` | `tx-send` | — | attach 모드 | 부분 |
| 169 | `stress` | `block-time` | — | cases/stress-block-time | 동등 |
| 170 | `stress` | `tx-flood` | — | cases/stress-tx-flood | 동등 |
