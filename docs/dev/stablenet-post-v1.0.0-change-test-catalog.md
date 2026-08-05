# go-stablenet post-v1.0.0-change 테스트 카탈로그 (포팅 참고용)

> 출처: 삭제된 로컬 브랜치 `feat/post-v1.0.0-change-migration`(커밋 `9908be2`, 2026-07-20, bash 시절)의
> `tests/regression/post-v1.0.0-change/` 51개 bash 회귀 테스트. **코드(bash)는 Go 재작성으로 폐기**됐으나,
> 향후 go-stablenet 테스트를 Go로 포팅할 때 "어떤 케이스가 있는지"의 **카탈로그**로 보존한다.
> (wemix4 포팅과는 별개 스위트. wemix4는 `wemix4-port-tracker.md` 참조.)
> 각 항목은 원본 `---chainbench-meta---` 헤더의 id·name·tags·profile.

## a-stand-alone

| id | name | tags | profile |
|----|------|------|---------|
| TC-3-1-04 | 기존 secp256k1 서명 데이터의 신규 바이너리 동기화 호환성 | secp256k1, signature, compatibility, sync, upgrade | — |
| TC-4-1-03 | v2→v3 binary 교체 시 GenesisMismatchError 검증 (validator 정보 변형) | hardfork, config, genesis, mismatch, validator | — |
| TC-5-2-06 | 미지원 시스템 컨트랙트 버전 — BohoBlock 커밋 실패 검증 | hardfork, boho, govminter, error-handling, unsupported-version | — |

## b-extra-state

| id | name | tags | profile |
|----|------|------|---------|
| TC-4-5-01,TC-4-5-02 | Account Extra alloc bits reflected in AccountManager (authorized + blacklisted) | hardfork, boho, account-extra | hardfork-account-extra |
| TC-4-5-03,TC-4-5-04 | GovCouncil params synced into AccountManager (contract → alloc direction) | hardfork, boho, account-extra | hardfork-account-extra |
| TC-4-5-05,TC-4-5-06 | Union merge: alloc.Extra AND GovCouncil params produce no duplicates | hardfork, boho, account-extra | hardfork-account-extra |
| TC-4-5-07 | TEST_ACC_F dual status: authorized AND blacklisted simultaneously | hardfork, boho, account-extra | hardfork-account-extra |
| TC-4-5-08 | alloc.Extra sync must not alter account balances | hardfork, boho, account-extra | hardfork-account-extra |
| TC-4-5-09 | 미정의 Extra 비트 baked-in 바이너리는 부팅 실패해야 함 | hardfork, boho, account-extra, negative | — |
| TC-4-5-10,TC-4-5-11,TC-4-5-12 | Delayed Boho activation preserves account states (decodePrealloc restoration) | hardfork, boho, account-extra, delayed | hardfork-boho-delayed |

## c-build-verify

| id | name | tags | profile |
|----|------|------|---------|
| TC-5-1-01 | Full binary compilation succeeds | hardfork, build, compilation | — |
| TC-5-1-02 | Unit tests pass (make test-short) | hardfork, build, test | — |
| TC-5-1-03 | gstable version output is valid | hardfork, build, version | — |

## d-egp

| id | name | tags | profile |
|----|------|------|---------|
| TC-4-6-01 | EffectiveGasPrice for authorized account | hardfork, boho, effectiveGasPrice, authorized | — |
| TC-4-6-02 | EffectiveGasPrice for regular (non-authorized) account | hardfork, boho, effectiveGasPrice, regular | — |
| TC-4-6-03 | EffectiveGasPrice consistent across BP and EN nodes | hardfork, boho, effectiveGasPrice, consistency | — |
| TC-4-6-04 | AuthorizedTxExecuted is last log in authorized tx receipt | hardfork, boho, authorized, event | — |

## e-string-handling

| id | name | tags | profile |
|----|------|------|---------|
| TC-4-3-01 | GovCouncil authorizedAccounts splitAndTrim — 공백 없음 0xaaa,0xbbb,0xccc → 3 | hardfork, config, splitAndTrim, gov, authorizedAccounts | — |
| TC-4-3-02 | GovCouncil authorizedAccounts splitAndTrim — 공백 포함 0xaaa, 0xbbb, 0xccc → 3 | hardfork, config, splitAndTrim, gov, authorizedAccounts | — |
| TC-4-3-03 | GovCouncil authorizedAccounts splitAndTrim — 앞뒤 공백  0xaaa , 0xbbb  → 2 | hardfork, config, splitAndTrim, gov, authorizedAccounts | — |
| TC-4-3-04 | GovCouncil authorizedAccounts splitAndTrim — 빈 항목 0xaaa,,0xbbb → 2 | hardfork, config, splitAndTrim, gov, authorizedAccounts | — |
| TC-4-3-05 | GovCouncil authorizedAccounts splitAndTrim — 단일 항목 0xaaa → 1 | hardfork, config, splitAndTrim, gov, authorizedAccounts | — |
| TC-4-3-06 | GovCouncil authorizedAccounts splitAndTrim — 빈 문자열 → 0 | hardfork, config, splitAndTrim, gov, authorizedAccounts | — |

## f-genesis-recovery

| id | name | tags | profile |
|----|------|------|---------|
| TC-4-4-01 | Simultaneous Anzeon+Boho activation at block 0 | hardfork, boho, anzeon, simultaneous | — |
| TC-4-4-02 | Boho activates at block N while Anzeon already active | hardfork, boho, anzeon, delayed | hardfork-boho-delayed |
| TC-4-4-03 | GovMinter v2 bytecode injected at genesis (BohoBlock=0) | hardfork, boho, genesis, injection | — |
| TC-4-4-04 | GovMinter bytecode changes at BohoBlock (delayed activation) | hardfork, boho, delayed, injection | hardfork-boho-delayed |
| TC-5-2-05 | GovMinter v2 업그레이드가 기대 bytecode hash로 코드만 교체하고 balance 를 보존하는지 검증 | hardfork, boho, govminter, upgrade, code-only, balance, bytecode-hash | — |
| TC-5-3-01 | testnet embedded genesis block 0 hash consistency | hardfork, genesis, decodePrealloc, testnet, hash, automation | — |

## g-default-v1_0_2

| id | name | tags | profile |
|----|------|------|---------|
| TC-1-1-01, TC-1-1-10 | 소각 제안 취소 → refundableBalance 이동 및 BurnDepositRefunded 이벤트 검증 | hardfork, boho, govminter, burn, refund, cancel, events | — |
| TC-1-1-02 | proposeBurn rejection creates refundable balance | hardfork, boho, govminter, burn, reject, refund | — |
| TC-1-1-03 | 소각 제안 만료(expire) → refundableBalance 이동 → claimBurnRefund 정상 출금 검증 | hardfork, boho, govminter, burn, expire, refund | — |
| TC-1-1-04 | Executed burn has no refundable balance | hardfork, boho, govminter, burn, execute | — |
| TC-1-1-05, TC-1-1-09 | claimBurnRefund 정상 출금 및 BurnRefundClaimed 이벤트 검증 | hardfork, boho, govminter, burn, claim, refund, events | — |
| TC-1-1-06 | claimBurnRefund with zero balance reverts | hardfork, boho, govminter, burn, revert | — |
| TC-1-1-07 | Second claimBurnRefund reverts (double claim prevention) | hardfork, boho, govminter, burn, revert | — |
| TC-1-1-11, TC-1-1-12 | Boho 하드포크 전후 GovMinter 바이트코드 hash 교체 및 v1 상태 보존 검증 | hardfork, boho, govminter, bytecode, bytecode-hash, storage, upgrade | — |
| TC-1-2-01, TC-1-2-02, TC-1-2-03, TC-1-2-04, TC-1-2-05, TC-1-2-06 | secp256r1 (P-256) precompile 통합 검증 — TC-1-2-01~06 | hardfork, boho, precompile, secp256r1, p256, eip-7951 | — |
| TC-1-3-01 | DynamicFeeTx with gasFeeCap below minimum is rejected | hardfork, boho, gas, feecap, rejection | — |
| TC-1-3-02 | DynamicFeeTx with exact minimum gasFeeCap is accepted | hardfork, boho, gas, feecap, boundary | — |
| TC-1-3-03 | DynamicFeeTx with gasFeeCap above minimum is accepted | hardfork, boho, gas, feecap | — |
| TC-1-3-04 | Legacy TX with gasPrice below minimum is rejected | hardfork, boho, gas, legacy, rejection | — |
| TC-1-3-05 | AccessListTx with gasPrice below minimum is rejected | hardfork, boho, gas, accesslist, rejection | — |
| TC-1-3-06 | DynamicFeeTx gasTipCap 최소값 미만 거부 검증 | hardfork, anzeon, gas, dynamicfee, tipcap, rejection | — |
| TC-1-3-01 | Minimum gas fee enforcement verification | hardfork, boho, gas, fee, basefee | — |
| TC-4-1-01 | Boho hardfork chain config verification | hardfork, boho, config, chain | — |
| TC-4-1-02 | Boho 하드포크 값의 체인 설정 반영 및 런타임 활성화 검증 | hardfork, boho, config, chain, govminter, activation | — |
| TC-4-2-01, TC-4-2-03 | EIP-7702 AuthorizationList 1건/2건 estimateGas 비용 반영 검증 | hardfork, eip7702, estimateGas, authlist, gas | — |
| TC-4-2-02 | AuthorizationList 없는 일반 전송 estimateGas baseline 검증 | hardfork, eip7702, estimateGas, dynamicfee, authlist, baseline | — |
| TC-5-2-01, TC-5-2-02, TC-5-2-03 | 하드포크 업그레이드 등록 순서·시점 검증 (CollectUpgrades + SetConfigFromChainConfig) | hardfork, boho, anzeon, upgrade, registry, govminter | — |
| TC-5-2-04 | v1 시스템 컨트랙트 Params 초기화 값 검증 | hardfork, boho, anzeon, system-contract, params, storage, govvalidator | — |

