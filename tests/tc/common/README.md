# tests/tc/common — 세 체인에서 같은 목적으로 도는 테스트

Confluence [[Common] Test](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2987917348)
가 가려낸 공통 테스트 CT 70개의 자리다. 저장소에 옮겨 둔 사본은
[`docs/tc/common/`](../../../docs/tc/common/) 에 있다.

> **2026-09-28 — 목록을 그대로 믿지 않고 코드에 대조했다.** Confluence 가 공통으로 적은
> 것 중 여덟은 go-stablenet 에만 있는 기능에 기대고 있었고, 스물셋은 반대로 근거 없는
> 제약이 붙어 다른 체인에서 건너뛰고 있었다. 4절이 그 내역이다. Confluence 목록도 같은
> 내용으로 고쳐야 한다.
>
> **2026-09-29 — 빈 CT 를 채웠다.** 목록에는 있는데 공통 케이스가 없던 CT 16개에 케이스 17개를
> 새로 만들어 세 체인(Docker)에서 돌렸다. 이제 70개 CT 가 모두 케이스를 갖는다. 5절과 6절이
> 그 내역이다. 새 케이스는 6절 표에 "(2026-09-29 신규)" 로 표시했다.

## 1. 무엇이 여기 있나

케이스 **99개**다. CT 하나에 케이스가 여럿인 것은 지금 세 체인이 각자의 케이스를 갖고
있기 때문이다 — `CT-NODE-001` 은 `wemix-chain-up`·`wbft-chain-up`·`stablenet-chain-up`
셋이다.

```
tests/tc/common/
├── node/       25건   노드·동기화·네트워크
├── tx/         24건   트랜잭션 전송·거부
├── fee/        12건   수수료·가스 정책
├── contract/   12건   컨트랙트 실행
├── rpc/        17건   조회·구독 API
└── fault/       9건   장애·복구
```

파일명은 `<CT 번호>-<테스트 id>.json` 이다. 앞의 번호가 같은 파일들이 한 CT 를 이룬다.
번호 뒤의 이름은 옮기기 전과 한 글자도 바꾸지 않았다(몇몇 옮긴 케이스는 파일 이름과 `id` 가 다르다) — `id` 로 케이스를 부르는 문서와
검사가 여럿이라, 이름을 같이 바꾸면 무엇이 깨졌는지 옮긴 것과 구별할 수 없다.

CT 하나를 두 영역이 함께 거명한 것이 둘 있다. `chain-id` 는 `CT-NODE-003` 과
`CT-RPC-008`, `remote-chain-info` 는 `CT-NODE-014` 와 `CT-RPC-008` 이다. 앞의 CT 를
따라 `node/` 에 두었다.

## 2. 왜 99개를 CT 수만큼 합치지 않았나

합치는 것이 목표지만 한 번에 하지 않는다. `CT-NODE-001` 의 세 케이스는 같은 것을 보는
듯하지만 기대값이 다르다 — 검증자 수를 확인하는 방법이 체인마다 다르고, WEMIX3.0 은
검증자 조회 자체가 없다. 지금 합치면 그 차이가 코드가 아니라 사람의 기억 속으로 들어간다.

그래서 이번 단계는 자리만 옮긴다. 세 체인의 케이스가 한 디렉터리에서 번호로 묶여 나란히
보이게 된 것이 전부다. 나란히 놓고 봐야 무엇이 같고 무엇이 다른지 셀 수 있다.

## 3. 세 체인 모두에서 게이트를 통과한다

케이스의 `requires` 를 세 체인이 제공하는 capability 집합과 대조한 결과다.

| 체인 | 99건 중 게이트 통과 |
| --- | --- |
| go-stablenet | 98 |
| go-wbft | 98 |
| go-wemix | 98 |

남은 하나는 `fault/004-fault-network-partition` 이다. 체인이 아니라 실행 대상에 `target:remote` 를
요구해, 노드가 원격이나 Docker 서버에 있을 때만 돈다(2026-09-29 기준).

**게이트를 통과한다는 것은 "돌 수 있다" 이지 "통과한다" 가 아니다.** 91건이던 때도 이 표는
91/91/91 이었고, 그대로 세 체인에 돌리자 다섯이 깨졌다. 게이트가 막지 못한 것은 케이스가
무엇을 필요로 하는지 말하지 않았기 때문이다. 다섯은 빠졌고, 무엇이 필요한지 이제
`requires` 에 적혀 있다(5절).

케이스가 이름 붙인 `chainPreset` 은 아직 대부분 stablenet 이다. 실행할 때 덮는다.

```sh
chainbench run tests/tc/common/tx/001-value-transfer.json \
  --workspace-dir ~/cbw/x --chain-preset wemix-bp4
```

세 체인이 모두 갖춘 모양은 `bp4`, `bp4-en1`, `bp4-en2-pn1`, `bp7-en7-pn1`, `bp9` 다섯이고 `presets/chain/` 에 세 벌씩 있다. `bp4-en1` 은 동기화 케이스(CT-NODE-004~007, CT-FEE-009)가 쓰려고 2026-09-29 에 wbft·wemix 판을 더했다.

## 4. 공통 판정을 다시 한 내역 (2026-09-28)

세 체인의 노드 소스를 직접 대조했다. 결정적인 것은 셋이다.

| | go-stablenet | go-wbft | go-wemix |
| --- | --- | --- | --- |
| `header.GasTip()` | 있음 | 없음 | 없음 |
| `params.MinBaseFee` | 있음 | 없음 | 없음 |
| `istanbul_getWbftExtraInfo` 응답의 `gasTip` | 있음 | **없음** | 없음 |

세 체인 모두 genesis 가 `londonBlock: 0` 이고 시작 baseFee 가 1 Gwei 로 같다. 거부 문구
`transaction underpriced`·`exceeds block gas limit`·`insufficient funds` 도 셋 다 같다.

### 4.1 공통에서 뺀 여덟 — go-stablenet 전용이다

`gasTip` 또는 `MinBaseFee` 에 기대므로 다른 두 체인에는 물을 대상이 없다.

| 케이스 | CT | 어디로 갔나 |
| --- | --- | --- |
| `feecap-above-min-accepted` | CT-FEE-002 | `go-stablenet/regression/anzeon/08-` |
| `feecap-exact-min-accepted` | CT-FEE-002 | `go-stablenet/regression/anzeon/09-` |
| `basefee-minimum` | CT-FEE-006 | `go-stablenet/regression/anzeon/06-` |
| `gas-price-equals-basefee-plus-tip` | CT-FEE-010 | `go-stablenet/regression/api/07b-` |
| `max-priority-fee-equals-gastip` | CT-FEE-011 | `go-stablenet/regression/api/08-` |
| `legacy-transfer` | CT-TX-002 | `go-stablenet/regression/ethereum/08-` |
| `dynamic-fee-tx` | CT-TX-003 | `go-stablenet/regression/ethereum/09-` |
| `gaslimit-exceeded-rejected` | CT-TX-016 | `go-stablenet/regression/anzeon/11-` |

**셋은 검사하려는 것 자체는 공통이다.** CT-TX-002(Legacy 트랜잭션)·CT-TX-003(EIP-1559)·
CT-TX-016(가스 한도 초과)은 세 체인에서 다 되는 일이고, 공통이 아닌 것은 지금 구현이
gasPrice 를 `gasTip` 으로 구한다는 점이다. 다시 쓰면 살아난다. 나머지 넷(CT-FEE-002·
006·010·011)은 규칙 자체가 go-stablenet 것이라 살릴 수 없다.

### 4.2 제약을 뗀 스물셋 — 근거가 없었다

`family:wbft` 18건과 `engine:anzeon` 5건이다. 18건은 파일 안에서 `wbft` 라는 낱말이
나오는 곳이 `requires` 줄 하나뿐이었고, 쓰는 것은 전부 표준 이더리움 RPC 였다. 5건은
gasPrice 를 1 wei 로 보내거나 블록 가스 한도를 넘겨 거부를 기대하는 것이라, baseFee 가
1 Gwei 에서 시작하는 세 체인 어디서나 성립한다.

이 제약이 붙어 있는 동안 wemix 에서 31건이 **실패가 아니라 조용히 건너뛰고** 있었다
(`internal/testengine/capability.go`: "skipped, not failed").

### 4.3 아직 답을 못 낸 둘

`CT-NODE-016`(`016-block-period-one-second`·`016-stress-block-time`)은 블록 주기가
1초라는 것에 기댄다. go-stablenet 과 go-wbft 는 genesis 템플릿에 `blockPeriodSeconds: 1`
을 적지만 **go-wemix 는 적지 않는다** — 거버넌스 값이고 부트스트랩 때 정해진다. 둘은
게이트를 통과하므로 건너뛰지 않고 **돌아서 실패하거나 운으로 통과한다.** 체인별 기대값을
적을 자리가 필요하고, 그 설계는
[`design-v3/common-tc-01-chain-varying.md`](../../../docs/dev/architecture/design-v3/common-tc-01-chain-varying.md)
§3.2 에 있다.

## 5. 여기 없는 것

없다. 2026-09-28 까지는 CT 12개가 자동 테스트가 없어 옮길 것이 없었고, CT-TX-002·003 은
go-stablenet 전용 케이스만 있었다. 2026-09-29 에 이 14개와 CT-FEE-002 의 "초과" 부분에 케이스를
새로 만들었다.

| CT | 새 케이스 | 전에 있던 것 |
| --- | --- | --- |
| CT-NODE-004 | `node/004-full-sync.json` | bash 스크립트가 있었으나 go-stablenet 전용 Go 테스트로 옮겨졌다 |
| CT-NODE-005 | `node/005-snap-sync.json` | 위와 같다. 이미 블록을 가진 노드를 다시 띄워 실제로는 full 로 동기화했다 |
| CT-NODE-006 | `node/006-downloader-catch-up.json` | 위와 같다 |
| CT-NODE-007 | `node/007-live-block-receive.json` | 위와 같다 |
| CT-NODE-015 | `node/015-block-timestamp-monotonic.json` | 없었다 |
| CT-TX-002 | `tx/002-legacy-value-transfer.json` | go-stablenet 전용 `legacy-transfer` |
| CT-TX-003 | `tx/003-dynamic-fee-transfer.json` | go-stablenet 전용 `dynamic-fee-tx` |
| CT-TX-010 | `tx/010-fee-delegated-access-list.json` | 없었다 |
| CT-TX-011 | `tx/011-keystore-fee-delegate-sign.json` | 없었다(메서드 존재만 CT-TX-009 가 본다) |
| CT-TX-014 | `tx/014-carry-over-and-replace.json` | 없었다 |
| CT-TX-017 | `tx/017-reject-vs-execution-failure.json` | 없었다 |
| CT-FEE-002 (초과) | `fee/002-legacy-gasprice-above-min-accepted.json` · `fee/002-accesslist-gasprice-above-min-accepted.json` · `fee/002-dynamic-feecap-above-min-accepted.json` | 동적 수수료만 go-stablenet 전용으로 있었다 |
| CT-FEE-009 | `fee/009-snap-receipt-gas-price.json` | 없었다 |
| CT-CONTRACT-003 | `contract/003-view-call-leaves-state.json` | 값 읽기만 CT-CONTRACT-001·002 가 부수적으로 했다 |
| CT-RPC-010 | `rpc/010-signed-tx-seen-in-pool.json` | 전송만 `tx/001-value-transfer` 가 했다 |

공통이 아니라 뺀 부분이 둘 있다. CT-FEE-002 의 "최소 가스비와 같으면 받아들여진다" 는 세 체인이
경계를 다르게 정해 부록 B 로, CT-TX-011 의 "대납자 키 서명의 형식 검사" 는 go-wemix 에 없어
부록 A 로 옮겼다.

새 케이스에 필요해 실행 도구에 더한 것이 넷이다. `sendTx` 의 로컬 키 legacy·접근 목록 전송(`key` +
`gasPrice`), RPC 호출이 정해진 문구로 실패해야 한다는 `rpcError` 검사, 생산하지 않는 노드를 genesis
상태로 되돌리는 `resetNode`(snap 동기화용), 지표 포트가 밖으로 열려 있지 않은 서버에서 그 서버 안에서
지표를 읽는 `metric` 검사다.

### 세 체인에 돌려 보고 되돌린 다섯 (2026-09-28)

처음에 91개를 옮겼다가 다섯을 go-stablenet 영역으로 되돌렸다. 읽어서는 보이지 않고
돌려야 나오는 차이였다. 되돌리면서 무엇이 필요한지 `requires` 에 적었으므로, 이제는 다른
체인에서 실패하지 않고 건너뛴다.

| 케이스 | 되돌아간 곳 | 필요한 것 | 왜 |
| --- | --- | --- | --- |
| `register-contract` | `go-stablenet/vocabulary/03-` | `evm:shanghai` | 배포하는 바이트코드가 PUSH0 를 45번 쓴다. go-wemix 의 EVM 은 London 세대라 그 옵코드가 없어 배포가 가스를 전부 태우고 실패한다 |
| `eth-call-revert-returns-error` | 2026-09-28 에 PUSH0 없는 컨트랙트로 다시 써서 `contract/005-` 로 돌아왔다 | - | 같은 이유였다. PUSH0 9번 |
| `anzeon-basefee-increase` | `go-stablenet/regression/anzeon/03-` | `engine:anzeon` | anzeon 만 상승·하강 문턱 두 개를 쓴다. 표준 EIP-1559 는 목표 하나뿐이라 25% 채우기가 목표 미달이 되어 반대로 내려간다 |
| `anzeon-basefee-stable` | `go-stablenet/regression/anzeon/04-` | `engine:anzeon` | 표준 EIP-1559 에는 유지 구간이 없다 |
| `anzeon-basefee-decrease` | `go-stablenet/regression/anzeon/05-` | `engine:anzeon` | go-wbft 에서 통과했으나 재려던 것과 다른 이유였다 |

마지막 줄이 이 절의 요점이다. **통과했다고 잰 것은 아니다.** 표준 EIP-1559 에서 25%
채우기는 부하가 아니라 목표 미달이라, 부하를 거는 동안에도 기본 수수료가 이미 내려가고
있었다. 셋을 묶어 빼지 않으면 그 하나가 "wbft 에서 도는 공통 테스트" 로 남는다.

`evm:shanghai` 는 이번에 더한 capability 다. 세 체인이 포크를 brioche·croissant·anzeon
이라 부르고 어느 genesis 에도 이더리움 포크 이름이 없어 뽑아낼 데가 없으므로,
`precompile:` 과 같이 매니페스트가 선언한다.

## 6. CT 와 케이스 대응

### NODE — CT 15개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-NODE-001 | 노드 기동과 블록 생성 | `node/001-wemix-chain-up.json` · `node/001-wbft-chain-up.json` · `node/001-stablenet-chain-up.json` |
| CT-NODE-002 | 15노드 구성 기동 | `node/002-wemix-chain-up-15.json` · `node/002-wbft-chain-up-15.json` · `node/002-stablenet-chain-up-15.json` |
| CT-NODE-003 | 제네시스 초기화와 블록 0 해시 일치 | `node/003-genesis-block-hash-consistent.json` · `node/003-chain-id.json` |
| CT-NODE-004 | Full Sync 동기화 | `node/004-full-sync.json` (2026-09-29 신규) |
| CT-NODE-005 | Snap Sync 동기화 | `node/005-snap-sync.json` (2026-09-29 신규) |
| CT-NODE-006 | 누락 블록 일괄 동기화 | `node/006-downloader-catch-up.json` (2026-09-29 신규) |
| CT-NODE-007 | 새 블록 실시간 수신 | `node/007-live-block-receive.json` (2026-09-29 신규) |
| CT-NODE-008 | 피어 연결과 피어 조회 | `node/008-basic-peers.json` · `node/008-admin-peers-populated.json` |
| CT-NODE-009 | 노드 간 최신 블록 해시 일치 | `node/009-basic-consensus.json` · `node/009-basic-sync.json` |
| CT-NODE-010 | 전달 노드 경유 동기화 | `node/010-proxied-pn-routing.json` · `node/010-wbft-proxied-routing.json` |
| CT-NODE-011 | 종단 노드가 먼저 오는 혼합 배치 | `node/011-e1-mixed-producers.json` |
| CT-NODE-013 | 제네시스 불일치 시 기동 거부 | `node/013-genesis-mismatch.json` |
| CT-NODE-014 | 동기화 완료 상태 확인 | `node/014-chain-not-syncing.json` · `node/014-remote-chain-info.json` |
| CT-NODE-015 | 블록 시각 단조 증가 | `node/015-block-timestamp-monotonic.json` (2026-09-29 신규) |
| CT-NODE-016 | 블록 생성 주기 | `node/016-block-period-one-second.json` · `node/016-stress-block-time.json` |

### TX — CT 20개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-TX-001 | 일반 송금 | `tx/001-basic-tx-send.json` · `tx/001-sample-minimal.json` · `tx/001-value-transfer.json` |
| CT-TX-002 | Legacy 트랜잭션 | `tx/002-legacy-value-transfer.json` (2026-09-29 신규) / 공통 아님: `go-stablenet/regression/ethereum/08-legacy-transfer.json` |
| CT-TX-003 | 동적 수수료 트랜잭션 | `tx/003-dynamic-fee-transfer.json` (2026-09-29 신규) / 공통 아님: `go-stablenet/regression/ethereum/09-dynamic-fee-tx.json` |
| CT-TX-004 | 접근 목록 트랜잭션 | `tx/004-access-list-tx.json` |
| CT-TX-005 | 수수료 대납 트랜잭션 | `tx/005-fee-delegated-transfer.json` |
| CT-TX-006 | 대납 트랜잭션의 보낸 이 서명 변조 거부 | `tx/006-fd-sender-sig-invalid-rejected.json` |
| CT-TX-007 | 대납 트랜잭션의 대납자 서명 변조 거부 | `tx/007-fd-feepayer-sig-invalid-rejected.json` |
| CT-TX-008 | 대납자 잔액 부족 거부 | `tx/008-feepayer-insufficient-rejected.json` |
| CT-TX-009 | 대납 서명 API 존재 | `tx/009-fee-delegate-sign-rpc-present.json` |
| CT-TX-010 | 접근 목록을 붙인 대납 트랜잭션 | `tx/010-fee-delegated-access-list.json` (2026-09-29 신규) |
| CT-TX-011 | 노드 키 저장소 경유 대납 서명 | `tx/011-keystore-fee-delegate-sign.json` (2026-09-29 신규) |
| CT-TX-012 | 계정별 nonce 순서 보장 | `tx/012-nonce-ordering.json` |
| CT-TX-013 | 같은 nonce 트랜잭션 교체 | `tx/013-replacement-tx.json` |
| CT-TX-014 | 미포함 트랜잭션의 이월과 교체 | `tx/014-carry-over-and-replace.json` (2026-09-29 신규) |
| CT-TX-015 | 잔액 부족 트랜잭션 거부 | `tx/015-insufficient-funds-rejected.json` · `tx/015-wbft-insufficient-funds-rejected.json` · `tx/015-wemix-insufficient-funds-rejected.json` |
| CT-TX-016 | 블록 가스 한도 초과 트랜잭션 거부 | `tx/016-gas-limit-exceeds-block-rejected.json` / 공통 아님: `go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json` |
| CT-TX-017 | 거부와 실행 실패의 상태 구분 | `tx/017-reject-vs-execution-failure.json` (2026-09-29 신규) |
| CT-TX-018 | 트랜잭션 풀 전파 | `tx/018-basic-txpool-propagation.json` |
| CT-TX-019 | 부하 전송 중 블록 진행 | `tx/019-stress-tx-flood.json` |
| CT-TX-020 | 테스트 계정 자금 지급 | `tx/020-faucet-funds-account.json` |

### FEE — CT 7개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-FEE-001 | 최소 팁 미달 거부 | `fee/001-tip-below-min-rejected.json` |
| CT-FEE-002 | 최소 가스비 경계값 | `fee/002-legacy-gasprice-below-min-rejected.json` · `fee/002-accesslist-gasprice-below-min-rejected.json` · `fee/002-feecap-below-min-rejected.json` · `fee/002-legacy-gasprice-above-min-accepted.json` (2026-09-29 신규) · `fee/002-accesslist-gasprice-above-min-accepted.json` (2026-09-29 신규) · `fee/002-dynamic-feecap-above-min-accepted.json` (2026-09-29 신규) / 공통 아님: `go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json` · `go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json` |
| CT-FEE-007 | 실제 적용 가스 가격 기록 | `fee/007-effective-gas-price.json` |
| CT-FEE-008 | 실제 적용 가스 가격 노드 간 일치 | `fee/008-effective-gas-price-regular-bp-en.json` |
| CT-FEE-009 | 스냅 동기화 노드의 영수증 가스 가격 보존 | `fee/009-snap-receipt-gas-price.json` (2026-09-29 신규) |
| CT-FEE-010 | 권장 가스 가격 조회 | `fee/010-gas-price-positive.json` / 공통 아님: `go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json` |
| CT-FEE-012 | 수수료 이력 조회 | `fee/012-fee-history-well-formed.json` |

### CONTRACT — CT 7개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-CONTRACT-001 | 컨트랙트 배포 | `contract/001-contract-roundtrip.json` · `contract/001-wemix-tx-and-contract.json` · `contract/001-wbft-tx-and-contract.json` |
| CT-CONTRACT-002 | 상태 변경 함수 호출 | `contract/002-storage-write-and-read.json` |
| CT-CONTRACT-003 | 조회 함수 호출 | `contract/003-view-call-leaves-state.json` (2026-09-29 신규) |
| CT-CONTRACT-004 | 가스 추정 | `contract/004-estimate-gas.json` |
| CT-CONTRACT-005 | 조회 호출의 되돌림 오류 | `contract/005-eth-call-revert-returns-error.json` |
| CT-CONTRACT-006 | 되돌림 트랜잭션의 실패 상태 | `contract/006-revert-tx-status-zero.json` · `contract/006-wbft-revert-status-zero.json` · `contract/006-wemix-revert-status-zero.json` · `contract/006-negative-tx-revert.json` |
| CT-CONTRACT-007 | 가스 소진 트랜잭션 | `contract/007-out-of-gas-consumes-all.json` |

### RPC — CT 15개 모두 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-RPC-001 | 최신 블록 번호 조회 | `rpc/001-basic-rpc-health.json` · `rpc/001-remote-rpc-health.json` |
| CT-RPC-002 | 블록 번호로 블록 조회 | `rpc/002-block-transactions-field.json` |
| CT-RPC-003 | 블록 해시로 블록 조회 | `rpc/003-block-by-hash-consistency.json` |
| CT-RPC-004 | 해시로 트랜잭션 조회 | `rpc/004-transaction-by-hash-fields.json` |
| CT-RPC-005 | 영수증 조회 | `rpc/005-transaction-receipt-fields.json` |
| CT-RPC-006 | 계정 nonce 조회 | `rpc/006-transaction-count-increments.json` |
| CT-RPC-007 | 잔액 조회 | `rpc/007-genesis-balance.json` · `rpc/007-remote-balance-check.json` |
| CT-RPC-008 | 체인 ID 조회 | `node/003-chain-id.json` · `node/014-remote-chain-info.json` |
| CT-RPC-009 | 이벤트 로그 조회 | `rpc/009-logs-query-well-formed.json` · `rpc/009-contract-event-emitted.json` |
| CT-RPC-010 | 서명된 트랜잭션 전송과 풀 조회 | `rpc/010-signed-tx-seen-in-pool.json` (2026-09-29 신규) |
| CT-RPC-011 | 트랜잭션 풀 건수 조회 | `rpc/011-txpool-status.json` |
| CT-RPC-012 | 트랜잭션 풀 내용 조회 | `rpc/012-txpool-content-well-formed.json` |
| CT-RPC-013 | 새 블록 구독 | `rpc/013-ws-subscribe-new-heads.json` |
| CT-RPC-014 | 이벤트 로그 구독 | `rpc/014-ws-subscribe-logs.json` |
| CT-RPC-015 | 노드 지표 조회 | `rpc/015-metric-head-block.json` |

### FAULT — CT 6개 중 6개가 공통 케이스를 갖고 있다

| CT | 무엇을 보나 | 공통에 있는 케이스 |
| --- | --- | --- |
| CT-FAULT-001 | 생성 노드 1대 중단과 재시작 | `fault/001-fault-node-crash.json` · `fault/001-wemix-node-crash.json` · `fault/001-wbft-node-crash.json` · `fault/001-sample-lifecycle.json` |
| CT-FAULT-002 | 중단 노드 복구 후 동기화 | `fault/002-fault-node-recover.json` |
| CT-FAULT-003 | 생성 노드 2대 중단 시 합의 중단과 복구 | `fault/003-fault-two-down.json` |
| CT-FAULT-004 | 네트워크 분리와 복구 | `fault/004-fault-network-partition.json` |
| CT-FAULT-005 | 허브형 연결에서 합의와 전파 | `fault/005-fault-p2p-topology.json` |
| CT-FAULT-006 | 생산 노드 중단 시 대기 트랜잭션 처리 | `fault/006-fault-txpool-leader-change.json` |

## 7. 실패했거나 목적을 검증하지 못하던 케이스 (2026-09-29 에 정리)

세 체인(go-stablenet 740526d0, go-wbft, go-wemix)에 하나씩 차례로 돌린 결과다. 케이스가 문서의
기대 결과를 실제로 비교하도록 고친 뒤(`ceb79098`) 실패가 셋 남았고, 셋 다 같은 날 정리했다.
둘은 재는 방법을 바꿔 통과했고 하나는 공통에서 나갔다. 7.2 는 통과하면서 재려던 것을 재지
못하던 넷이고, 같은 날 함께 고쳤다.

### 7.1 실패하던 셋 — 2026-09-29 에 정리했다

**`fault/004-fault-network-partition`** — 세 체인 통과로 바뀌었다. 피어를 끊는 것으로는 갈라지지 않았다. `admin_addPeer` 로 들어간 상대는 static peer 가 되어 devp2p 가 계속 다시 걸고, go-wemix 는 30초마다 거버넌스를 읽어 멤버에게 직접 거는 고리가 하나 더 있다. 측정에서 선언 4에 실제 피어 8이었고 다리를 멈춰도 갈라지지 않았다. 그래서 노드에게 부탁하지 않고 노드가 도는 기계에 규칙을 넣어 패킷을 떨어뜨린다(`partition` 의 `method: "firewall"`). 도구가 없으면 규칙을 하나도 쓰기 전에 실패하며 무엇을 설치할지 알려 준다. 기계에 셸이 필요하므로 `requires` 에 `target:remote` 를 적고, 로컬 실행에서는 건너뛴다. 도커 함대에서 세 체인 모두 통과(2026-09-29, 175~177초).

**`fault/005-fault-p2p-topology`** — 세 체인 통과로 바뀌었다. 끊는 대신 처음부터 허브형으로 세운다. 구성에 pn 이 있으면 연결이 proxied 로 잡혀 en 은 pn 하고만 이어진다. 허브인 pn1 을 멈춰 en 이 정말 그 길 하나였는지 보이고, 다시 띄워 따라잡는 것을 본다. 로컬과 도커 양쪽에서 통과.

**`node/012-signature-compat-across-swap`** — 공통에서 뺐다. 바꿔 낄 두 번째 빌드가 체인마다 필요해 세 체인이면 여섯 개다. 사유는 부록 B 에 적었다.

### 7.2 목적을 검증하지 못하던 넷 — 2026-09-29 에 정리했다

**`node/010-wbft-proxied-routing`** — `blockAdvance` 가 `onEach` 의 첫 노드만 읽고 있었다. 여기서 첫 노드는 bp1 이고, 이 케이스가 확인해야 하는 것은 en1 이 전달 노드를 거쳐 블록을 받는가다. 전달이 끊겨 en1 이 멈춰 있어도 통과했다. `blockAdvance` 와 `blockHalt` 가 적힌 노드를 모두 보도록 고쳤고, 실패하면 움직이지 않은 노드의 이름을 말한다. `onEach` 를 쓰는 다른 케이스도 함께 고쳐졌다.

**`rpc/009-contract-event-emitted`** — 필터에 토픽을 넣고 돌아온 로그의 topic0 이 그 토픽과 같은지 봤다. 필터가 그것만 골라 주니 늘 참이고, 실제로는 로그가 하나라도 있는지만 확인됐다. 셋으로 나눴다. 주소와 토픽으로 거르면 로그가 하나 나오는가, 토픽 없이 주소로만 거른 로그의 topic0 이 기대값과 같은가, 다른 토픽으로 거르면 하나도 안 나오는가.

**`node/016-block-period-one-second`** — 블록 시각은 초 단위 정수라, 주기가 1초 언저리면 연속 두 블록의 차이가 0이 되기도 1이 되기도 한다. 표본이 둘뿐이어서 결과가 동전 던지기였다. 20블록 평균으로 바꾸고 경계를 밀리초로 받게 했다(`minMillis`/`maxMillis`). 정수 초로는 하한을 0 아니면 1로밖에 못 적는데 0은 아무것도 말하지 않는 값이다.

**`node/016-stress-block-time`** — 같은 이유로 상한만 있었다. 밀리초 경계가 생겨 60블록에 [0.5, 2]초를 잡는다. 이제 블록이 쏟아지는 쪽 고장도 잡는다.

세 체인에서 돌려 확인했다(2026-09-29). go-wemix 의 `016-block-period` 는 평균 0.8초 언저리로 안정적으로 통과한다.

남은 한계를 적어 둔다. `016-block-period` 의 구간이 [0.6, 1.4]초로 넓다. 체인마다 주기가 다른데 `isPerChain` 은 기댓값 하나만 바꾸므로 경계 두 개를 체인별로 줄 수 없고, 세 체인이 설정 주기를 같은 방법으로 RPC 로 내놓지 않는다. 좁히려면 그 값을 읽어 견주어야 한다. 지금 구간도 멈춘 체인과 두 배 빠른 체인은 잡는다.

### 7.3 도구와 문서

- `chainbench validate` 는 `newAccount` 에 `saveKey` 가 없는 것을 잡지 못한다. 실행해야 드러난다.
- `docs/tc/common/06-detailed-procedures/06-fault.md` 의 CT-FAULT-004 절차는 아직 "4대를 2대씩, 피어 연결을 끊는다" 이다. 케이스는 생산 노드 9대를 4-1-4 로 선언하고 다리 노드를 멈추는 방식으로 바꿨다. 8대가 살아 있어 정족수(6)를 넘으므로, 멈춘다면 노드가 빠져서가 아니라 갈라져서다. wemix 처리를 정한 뒤 문서를 함께 고친다.

