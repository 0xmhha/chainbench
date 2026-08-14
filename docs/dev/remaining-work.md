# 잔여 작업 리스트 (DSL 이관 · 레거시 은퇴 · 후속 발견)

> 기준일: 2026-08-14 · 브랜치: `refactor/retire-pipeline-setup`
> 근거 문서: 이관 ledger `tests/specs/README.md` · [[legacy-retirement-plan]] (`docs/dev/legacy-retirement-plan.md`) · [[chainbench-worklist]] (`docs/dev/chainbench-worklist.md`)
> 표기: ☐ 미착수 · ◐ 진행 · ☑ 완료

이 문서는 **표현력 블로커가 해소되고 오버레이 케이스까지 라이브 이관이 끝난 시점**의 남은 일감이다.
케이스별 권위 있는 목록·근거는 ledger(`tests/specs/README.md`)에 있고, 여기서는 **블로커 유형별로 묶어** 무엇이 왜 남았는지와 우선순위를 정리한다.

---

## 0. 우선순위 요약

| 순위 | 작업 | 유형 | 블로커 | 상태 |
|------|------|------|--------|------|
| P0 | **overlays/account-extra.json params 배열→문자열 수정** | 버그 | 없음(즉시 가능) | ☐ |
| P1 | **Track 1: 레거시 경로 은퇴** (`test`/MCP/upgrade → engine 재배선 후 `testkit`·`pipeline/testrun` 삭제) | 리팩터 | 소비자 이관 선행 | ☐ (Task #32, DEFERRED) |
| P2 | **거버넌스 상태워드 잔여 케이스 이관** (system-contracts) | 이관 작업 | 없음((G)+(H) 확보) | ☐ |
| P2 | **파괴적 거버넌스 2건 — 격리 넷 이관** | 이관 작업 | 전용 격리 넷 | ☐ |
| P3 | **DSL 표현력 확장이 필요한 케이스군** (아래 §2) | DSL 기능 | 문법/기능 갭 | ☐ |
| P3 | **setup --launch 기본 포트 재검토** | 개선 | 없음 | ☐ |
| P4 | **바이너리/환경 의존 케이스** (P256·MinTip·원격 SSH·업그레이드·wemix4) | 환경 | 외부 바이너리/체인팀 | ☐ |

---

## 1. 즉시 처리 가능 (블로커 없음)

### 1.1 [P0] `overlays/account-extra.json` params 배열→문자열 버그
`internal/chains/stablenet/overlays/account-extra.json` 의
`config.anzeon.systemContracts.govCouncil.params.authorizedAddresses` /
`blacklistedAddresses` 가 **JSON 배열**로 선언돼 있으나, genesis `SystemContract.params` 는
`map[string]string` 이라 gstable `init` 이 `cannot unmarshal array ... of type string` 로 거부한다.
→ **콤마 조인 문자열**로 교정해야 실제 `setup --genesis-overlay` 기동이 성공한다.
(이번 세션 라이브 검증은 스크래치 오버레이 `/tmp/cb-ovl/combined-overlay.json` 에서 문자열로 수정해 통과 — 원본 파일은 미수정.)

> 근거: ledger 결함표 및 이번 세션 setup 기동 실패 로그. 배포 오버레이가 다른 소비자(레거시 testkit setup 경로)에서도 쓰이는지 확인 후 일괄 수정.

### 1.2 [P2] 거버넌스 상태워드 잔여 케이스 이관 (system-contracts)
`(G)` (`receiptLog` + `derive abiCall`) 과 `(H)` (`derive op:"word"` — proposals() 상태 워드[9] 디코드)
프리미티브가 이미 확보됐고 8+건으로 입증됐다. 실행 결과를 side-effect 없이 **상태 워드로만** 확인하는
잔여 케이스들은 **동일 수단으로 이관 가능** — 신규 DSL 코드 불필요, 순수 이관 작업량.

### 1.3 [P2] 파괴적 거버넌스 2건 — 격리 넷 이관
`validator-add-member-executes` · `masterminter-member-add-remove`.
흐름·calldata 는 `(G)+(H)` 로 확보됐으나 **검증자셋·정족수를 변경**해 공유 넷을 오염시키고
이후 spec 의 정족수 가정을 깬다. → 다른 spec 과 분리된 **전용 격리 넷**에서 실행하는 방식 확립 후 이관.

### 1.4 [P3] `setup --launch` 기본 포트 재검토
`internal/engine/localplan.go` 기본값이 p2p 30301 / http 8501 이라 스크래치 넷 기동 시
다른 넷(예: p2p 30301-30305 점유 넷)과 충돌하기 쉽다. 이번엔 `--set ports.base_*` 로 우회했다.
→ 기본 대역을 덜 붐비는 곳으로 옮기거나 충돌 시 자동 회피/명확한 에러를 검토.

---

## 2. DSL 표현력 확장이 필요한 케이스군

가짜로 만들지 않고 갭으로 남긴 것들. 필요한 확장이 생기면 이관한다. (케이스별 상세는 ledger §"이관하지 않은 것과 그 이유".)

| 갭 | 필요 확장 | 막힌 레거시 케이스 |
|----|-----------|---------------------|
| **컨트랙트 자산 + gasUsed/gasLimit read** | 컨트랙트 바이트코드 자산 + receipt `gasUsed`/`gasLimit` | `revert-tx-status-zero` · `out-of-gas-consumes-all` |
| **비동기 제출 + 부정 채굴 기대** | sendTx 비대기 제출 + "채굴되면 안 됨" assertion | `nonce-ordering` · `replacement-tx` · `out-of-order-nonces-mine` · `same-nonce-replacement` |
| **fee-delegation(0x16) 서명 변형** | 손상 이중서명 raw tx 조립(EncodeFeeDelegatedTampered) | `fd-*-sig-invalid` (4) · `fee-delegated-transfer` · `external-fee-delegated-transfer` · `feepayer-insufficient` · `fee-delegated-unfunded-feepayer` |
| **EIP-7702(0x04) set-code** | sendTx authorizationList + authority 서명 | `set-code-delegation` |
| **env 키 주입 / operator funded key** | env 키 바인딩 + 런타임 수취인 생성(일부 `newAccount` 로 이미 가능) | `external-value-transfer` |
| **RPC 오류 코드 구분 / call 부정 기대** | 메서드 존재 프로브(에러코드 구분) · `expectCallError` | `fee-delegate-sign-rpc-present` · `eth-call-revert-returns-error` |
| **토폴로지 파생 산술** | spec 이 자기 토폴로지(검증자수·quorum=ceil 2N/3) 참조 | `validator-set-count` · `prev-seals-quorum` |
| **조건부/에폭 대기** | 에폭 경계 대기 후 그 블록 조회 (`waitFor` 확장) | `epoch-transition-carries-epoch-info` |
| **WS 구독 선행 순서** | 어세션 전 구독 오픈(스텝 순서 표현) | `ws-subscribe-logs` |
| **delayed-boho 크로스포크 조건부 대기** | bohoBlock=N 지연 활성 + fork 전(0x1)/후(latest) 고정 비교 WaitFor | `govminter-code-changes-at-boho` · `p256-inactive-before-boho` · `anzeon-active-before-boho` · `prealloc-preserved-across-boho` |
| **(영구 갭) SDK 클라이언트측 정적 가드** | DSL sendTx 는 노드 직행 → 표현 대상 아님 | `zero-address-transfer-blocked` · `precompile-transfer-blocked` |

---

## 3. 바이너리 / 환경 의존 (외부 확보 선행)

| 항목 | 내용 | 선행 조건 |
|------|------|-----------|
| **P256VERIFY 미탑재** | 현 gstable 빌드가 0x100 에 `"0x"` 반환 → precompile 부재. 라이브 반증으로 이관 보류 | Boho/P256 탑재 바이너리 확보 + 체인팀 확인. 대상: `p256-precompile-active` · `p256-rejects-invalid` · accounts secp256r1 3건(현재 오프라인만) |
| **MinTip 미강제 빌드** | 현 빌드가 tipCap=1wei tx 를 수락·채굴 → 제출거부 단언 무의미 | MinTip 강제 빌드/설정 확보 + 체인팀 확인. 대상: `tipcap-underpriced-rejected` |
| **T5.2 업그레이드 멀티바이너리** | wemix+wbft 핸드오프 | gwemix+etcd 바이너리 (이 환경 라이브 제한) |
| **T5.5 wemix4 이관** | 레거시 스위트 → DSL, 대규모 | — |
| **원격 SSH 라이브 e2e** | RemoteFileSink+RemoteDriver · Collector 원격 tail | SSH 대상 호스트(사용자 환경) |

---

## 4. Track 1 — 레거시 경로 은퇴 (Task #32, DEFERRED)

> 상세: [[legacy-retirement-plan]]

현재 두 실행 경로가 병존한다:
- **신규**: `chainbench run` → `internal/engine` (attach/local) → `internal/testspec` (DSL)
- **레거시**: `chainbench test` → `internal/core/pipeline/testrun` → `internal/testkit` (Go-func anzeon 케이스)

레거시 제거 전 **소비자 이관** 선행이 필수다:
1. ☐ `chainbench test` (cmd) 를 engine 경로로 재배선
2. ☐ MCP `chainbench_test` / 관련 툴을 engine 경로로 재배선
3. ☐ upgrade 소비자(T5.2 계열)의 레거시 의존 제거
4. ☐ 위 3개 완료 후 `internal/core/pipeline/testrun` + `internal/testkit` 삭제

**블로커**: `test`/MCP/upgrade 가 아직 레거시를 사용 → 지금 삭제 금지.
(이미 착수분: 죽은 심볼 `testkit.RunCase` 제거 · 레거시 패키지 signpost · 은퇴 계획 문서화.)

---

## 5. 문서 정합 (낮은 우선순위)

- ledger(`tests/specs/README.md`) system-contracts 잔여 카운트가 본문 진행과 어긋난다(제목 "잔여 21건" vs 커버 문구 "33건"·표 35건). 오버레이 4+1건 이관 반영과 함께 카운트 재집계 필요.
- `id==Name` 유지로 DSL spec 과 레거시 anzeon 케이스의 id 가 겹친다 → 검증은 반드시 `chainbench run`(DSL), `chainbench test` 아님.

---

## 부록: 이번 세션(2026-08-14) 완료분

- ☑ account-extra 4건 (`authorized-extra-bit-synced`·`blacklisted-extra-bit-synced`·`dual-status-extra`·`extra-balance-preserved`) — account-extra 오버레이 넷 라이브 pass
- ☑ `proposal-expiry-transitions` — short-expiry 오버레이 넷 라이브 pass (`waitFor` timestamp 폴링 → expireProposal → 상태워드 Expired(5))
- ☑ 엔진 seam: `chainbench run --cap <name>` — attach 모드가 RPC 로 감지 못하는 오버레이 cap 을 운영자가 단언 → 게이팅 spec 이 skip 아닌 run (`internal/engine/attach.go`·`cmd/chainbench/run.go`·`attach_test.go`)
- 커밋: `abc882c feat(specs): migrate overlay-gated cases to DSL + add run --cap seam`
