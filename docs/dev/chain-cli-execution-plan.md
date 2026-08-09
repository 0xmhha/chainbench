# 체인 CLI · DSL 이관 실행 계획 (6-지시 통합)

> 목적: "docs 분석 → 코드 대조 → 미구현 수집 → 체인 CLI 단계별 지원(gstable→gwemix→gwbft) →
> DSL 이관 → 문서화" 6개 지시를 **현재 상태 ↔ 순서화된 다음 단계**로 매핑한다.
> 작성: 2026-08-10 · 기준 커밋: main `2424ccc`. 이 문서는 정본을 **중복하지 않고 참조**한다:
> 진행 정본은 [`chainbench-worklist.md`](chainbench-worklist.md), 부트스트랩 인수인계는
> [`chain-setup/next-automation.md`](chain-setup/next-automation.md), DSL 이관은
> [`../../tests/specs/README.md`](../../tests/specs/README.md).

---

## 1. 문서 인벤토리 (지시 1 — x-bar 관점: HEAD·COMPLEMENT·ADJUNCT)

| 문서 | HEAD (핵심 주제) | COMPLEMENT (필수 내용) | ADJUNCT |
|---|---|---|---|
| `chainbench-design.md` | 아키텍처 정본(구조+인터페이스) | §3 패키지 계약·§4 데이터모델·§6 동시성·§8 마이그레이션 | §9 미해결 질문 |
| `chainbench-component-architecture.md` | High/Middle/Low 계층 계획 | §1b DDD·§2b **코드-사실 검증 매트릭스**·§3 컴포넌트·§5 페이즈 | §0 비평·§6 리스크 |
| `chainbench-feature-spec.md` | 기능 계약 **F1–F16** | 기능별 입력/동작/출력/에러 + AC | 부록: 요구↔F 추적 |
| `chainbench-worklist.md` | **진행 단일 정본** | §1 우선순위·§1b 요약·§2 T0–T6·§3 트리 | — |
| `chain-setup/README.md` | 공통 bring-up 파이프라인 | 12단계·§1b **2-페이즈 계약**·§2 변곡점·§5 결함8 | §4 CLI |
| `chain-setup/next-automation.md` | governance-etcd 자동화 인수인계 | 확정사실·잔여 A/B/C/D(파일:라인) | §4 실행명령 |
| `chain-setup/case-1..4-*.md` | 케이스별 절차 | 절차·재료·자동화 잔여 | — |
| `legacy-retirement-plan.md` | 레거시 test 경로 은퇴 | §1 매핑·§2 순서·§4.4 이관진행(134) | — |
| `tests/specs/README.md` | DSL 이관 현황·규약 | 이관 카운트·이관불가 사유 | — |
| `repro-migration-remaining.md` | repro bash→Go e2e | 완료8·잔여5(각 블로커) | — |
| `wemix4-migration-plan.md`·`wemix4-port-tracker.md` | wemix4 스위트 이관 | 충실도갭·케이스별 트래커 | — |
| `chainbench-audit-2026-08-09.md` | 시점 감사(#204) | 무결성·갭 — **§7이 자기 무효화(worklist가 SSoT)** | — |

---

## 2. 코드-사실 분류 (지시 2 — AST: 패키지→타입→함수) 와 문서 대조

### 2.1 체인 플러그인 (`internal/chains/`)

| chain id | chain_id | 바이너리(매니페스트) | family | bootstrap | 로컬 bring-up |
|---|---|---|---|---|---|
| `stablenet` | 8283 | `gstable` | wbft | static | ✅ 완전 배선 |
| `wbft` | 8284 | `gwbft`(실제 산출물 `gwemix`) | wbft | static | ✅ (`--binary` 절대경로 필수) |
| `wemix` | 8285 | `gwemix` | poa | governance-etcd | ☐ **단독 미배선** |

### 2.2 CLI 표면 (`cmd/chainbench/`)

- **`chain`** (진단·단계실행): `cases`/`steps`/`up`/`status`/`down`. `up` 이 케이스를 단계 단위로 실행하고 OK/FAIL/TODO 로 게이트.
- **`run`** (DSL 실행): `--rpc`(attach)/`--binary`(local). 스텁 없음. exit 2=blocked, 1=fail.
- **`setup`** (static 계획/기동): static-family genesis 만. **governance-etcd 부트스트랩 분기 없음** → wemix `setup` 경로 부재의 근본 원인.
- **`upgrade run`** (핸드오프): **phase-correct·완전 배선** — 프로듀서 단독 governance+etcdInit→메시→await-fork. **동작하는 핸드오프 경로.**

### 2.3 미구현/스텁 (grep 확인)

- 프로덕션 코드에 `panic("not implemented")` **없음**.
- 진짜 미구현: `chainsetup.RunWemix`(3단계 후 TODO, `wemix.go:58`), `setup` 의 governance-etcd 분기, supervisor `Deps.LeaderGate`/`Deps.SwapBinary`(seam만).
- `chain up --case wemix-wbft`(`NewLiveHandoff`)는 **완전 구현이나 옛 단일페이즈** → Partial.

### 2.4 문서-코드 드리프트 (수정 대상)

| # | 드리프트 | 조치 |
|---|---|---|
| 1 | `docs/README.md` 가 feature-spec 을 "F1~F15" 로 표기(실제 F16) | README 수정 |
| 2 | DSL 이관 카운트 문서 간 상이(18 vs ~21, 카테고리 분할 상이) | 정본을 `tests/specs/README.md` 로 단일화·타 문서 링크 |
| 3 | audit(#204) §1–§6 이 후속 PR로 무효(자기 §7이 명시) | audit 상단에 "worklist가 SSoT" 배너(이미 §7 존재) |
| 4 | worklist T5.2 "미착수" ↔ 핸드오프 "검증완료" 의 표현 충돌 | T5.2 를 "`chain up` 핸드오프 2-페이즈화"로 명확화 |

---

## 3. 미구현 기능 · 남은 작업 (지시 3 수집)

정본은 `next-automation.md §3`. 요약:

- **A. 케이스 2 자동화** — `chain up --case wemix-wbft` 를 2-페이즈로(`upgrade run` 이 이미 하는 순서로). A1–A7: `chainsetup/{cases.go,handoff.go,handoff_driver.go}` + `profiles/wemix-upgrade.yaml`(fork 20→100). *비고: `upgrade run` 으로 핸드오프는 이미 동작 — A는 진단 CLI 정합화.*
- **B. 케이스 1 자동화** — `chain up --case wemix`(gwemix 단독). `RunWemix` 10단계 구현 + `setup` 의 governance-etcd 분기 + poa 오케스트레이터 승격. **유일하게 진짜 동작 안 하는 체인.**
- **C. supervisor 잔여(T3.2b)** — `LeaderGate` 프로브를 `admin.wemixInfo.etcd.cluster` 로 배선, `SwapBinary`(type-2) 배선.
- **D. DSL 이관(지시 5)** — 134 중 ~18 완료, **106 잔여**: system-contracts 46·accounts 35·gas-policy 17·hardfork 8. 표현력 블로커 없음(작업량). 이관불가 4건(순서·산술·조건부대기·토폴로지참조).
- **DSL 어휘 갭**(일부 이관의 잠재 블로커): raw-signed tx(`eth_sendRawTransaction`)·txpool 어세션·finality/reorg 어세션·validator-set 어세션·hardfork-control 액션·config-reload 액션 미등록.

---

## 4. 체인 CLI 단계별 지원 현황 (지시 4)

| 순서 | 체인 | 진입점 | 상태 |
|---|---|---|---|
| 1 | **gstable** | `chain up --case stablenet --binary <gstable>` | ✅ 전 단계 자동 + CI 게이트 |
| 2 | **gwemix** | `chain up --case wemix`(단독) / `upgrade run`(핸드오프) | ☐ 단독 미구현 / ✅ 핸드오프는 `upgrade run` |
| 3 | **gwbft** | `chain up --case wbft --binary <go-wbft/.../gwemix>` | ✅ 전 단계 자동(라이브) |

**내 머신 환경**: gstable·gwbft·bootnode·go-wemix 바이너리 모두 빌드됨. etcd 는 gwemix 내장(외부 불필요) → wemix 라이브 검증 가능.

→ 지시 4의 잔여는 **gwemix 단독(B)** 이 핵심. (gstable·gwbft 완료, 핸드오프는 `upgrade run` 으로 동작.)

---

## 5. 실행 순서 (지시 6 — 단계별 지원)

각 페이즈 = 1 브랜치, 작업마다 커밋, 페이즈 완료 시 1 PR(현행 워크플로우). 라이브 검증은 GSTABLE_BIN/GWEMIX_BIN 게이트로 CI green 유지.

### Phase P1 — 문서 정합 + 드리프트 수정 (본 문서 포함)
- 본 실행계획 문서 추가. §2.4 드리프트 4건 수정(README F16, 이관 카운트 단일화).

### Phase P2 — 케이스 2 핸드오프 `chain up` 2-페이즈화 (task A)
- `NewLiveHandoff` 를 `upgrade run` 의 phase-correct 순서로 재배열(프로듀서 격리→governance/etcdInit→verify-etcd→전체기동→메시→await-fork). 검증자 keystore/`--unlock`(static `armSpecs` 그대로). fork 20→100.
- 완료기준: `chain up --case wemix-wbft` 전 단계 OK, 포크 블록 miner 가 검증자.

### Phase P3 — 케이스 1 gwemix 단독 (task B) ★ 지시4 핵심
- `poa` 오케스트레이터 승격(config·키배치·IPC대기·2-페이즈·verify), `RunWemix` 10단계 실구현, `setup` 의 `bootstrap.type=="governance-etcd"` 분기, standalone 프로파일.
- 완료기준: `chain up --case wemix` 전 단계 OK, 블록 지속 전진.

### Phase P4 — supervisor 잔여 (task C, T3.2b)
- `LeaderGate` 실배선(`admin.wemixInfo.etcd.cluster`), `SwapBinary` 배선.

### Phase P5+ — DSL 이관 (지시 5, task D)
- 지시 4 완료 후 착수. 스위트 단위 배치 이관: hardfork(8)·gas-policy(17)·accounts(35)·system-contracts(46).
- 어휘 갭이 막는 케이스는 먼저 DSL 어휘 확장(raw-tx·txpool·validator-set 등) → 그 케이스 이관.
- 각 스위트 이관 시 `tests/specs/README.md` 카운트 갱신, offline `validate` 게이트 + 라이브 `chainbench run` 확인.
- 이관불가 4건은 사유와 함께 유지(대체 커버리지 명시).

### 검증 게이트 (매 푸시 전)
```sh
gofmt -l cmd internal tests            # 빈 출력
go vet ./... && go build ./...
go test -race ./...
golangci-lint run                      # unused 등 vet 미포착 항목
```
