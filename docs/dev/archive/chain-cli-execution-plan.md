# 체인 CLI · DSL 이관 실행 계획 (6-지시 통합)

> **[대체됨]** 2026-08-10 실행 계획. 순서는 [[chainbench-worklist]] §1g 로 대체.
> **새 작업의 근거로 쓰지 말 것.** 기록으로만 남긴다.

> 목적: "docs 분석 → 코드 대조 → 미구현 수집 → 체인 CLI 단계별 지원(gstable→gwemix→gwbft) →
> DSL 이관 → 문서화" 6개 지시를 **현재 상태 ↔ 순서화된 다음 단계**로 매핑한다.
> 작성: 2026-08-10 · 기준 커밋: main `2424ccc`. 이 문서는 정본을 **중복하지 않고 참조**한다:
> 진행 정본은 [`chainbench-worklist.md`](../chainbench-worklist.md), 부트스트랩 인수인계는
> [`chain-setup/next-automation.md`](../chain-setup/next-automation.md), DSL 이관은
> [`../../tests/specs/README.md`](../../../tests/specs/README.md).

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

**목표 재정의(사용자 확정 2026-08-10)**: 지시 4의 CLI 는 **한 번에 자동 실행되는 오케스트레이터가
아니다.** 체인 구성·테스트 수행의 각 단계를 **독립적으로·수동으로·커스텀 가능하게 실행하고
단계별로 검증**하는 **원자적·조합가능(composable) CLI** 이며, **동일 기능을 MCP 도구로도 노출**하여
LLM 이 여러 행동을 자유롭게 조합(자유도 확대)할 수 있게 한다.

기존 `chain up`(순차 자동)·`setup`(genesis/config/provision 번들)·`upgrade run`(핸드오프 번들)은
단계가 **묶여** 있어 단계별 커스텀·검증이 어렵다. 필요한 것은 각 단계를 **끊어서** 노출하는 표면이다.

### 4.1 원자 명령 표면 (신규 — 공유 워크스페이스 기반)

각 명령은 data-dir 워크스페이스(keys·genesis·config·nodes.json·logs 누적)를 읽고 자기 단계를
수행한 뒤 되쓴다 → 단계 간 이어짐/재실행/커스텀/검증 가능. 각 CLI 명령 = 대응 MCP 도구.

| 단계 | 명령(제안) | 커스텀 flag | 단계 검증 | 재사용 내부 |
|---|---|---|---|---|
| 계정/키 | `chain keys` | `--nodes/--validators/--balance/--password/--preset` | 주소·검증자셋 출력 | `validator set`, `core/keys` |
| 배치 | `chain allocate` | `--mode/--base-p2p/--base-rpc/--hosts` | 포트맵 출력 | `core/place` |
| genesis | `chain genesis` | `--mode(existing\|build\|overlay\|inherit)/--set/--overlay/--template` | genesis 바이트·검증자 치환 | `core/genesis`, `engine.GenesisSource` |
| config | `chain config` | `--sync-mode/--set/--static-nodes` | 노드별 config 출력 | `core/nodeconfig` |
| 수집·배치 | `chain provision` | (로컬/원격) `--remote-host/-user/-port/--key-file` | 물질화 파일·upload-if-absent | `core/filestore`, `driver.RemoteFileSink` |
| init | `chain init` | — | datadir chaindata 확인 | `core/driver` |
| 실행/정지/재실행/삭제 | `chain start`/`stop`/`restart`/`rm` | `--node\|--all/--grace/--data` | PID·head·고아0 | `core/driver`, `core/procman` |
| 로그 | `chain logs` | `--node/--follow/--since` (로컬/원격 tail) | 최근 라인 | `core/logs`, `driver.RemoteLogReader` |
| 부트스트랩(poa) | `chain deploy-governance`/`etcd-init`/`verify-etcd` | `--ipc` | `admin.wemixInfo.etcd.cluster` 비어있지 않음 | `consensus/poa` |
| 테스트 | `chain test` (=`run`) | `--spec/--rpc` | session 판정 | `engine`, 기존 `run` |
| 검증 | `chain status`/`inspect` | `--data-dir` | 각 단계 결과 | 기존 `chain status` |

**설계 원칙**: 원자 명령은 위 내부 로직을 **얇게 감싸/분해**한다(중복 구현 금지). 오케스트레이터
(`chain up`, `upgrade run`)는 **원자 명령의 조합 데모**로 남긴다. MCP 미러는 CLI RunE 와 **동일 코어**를
호출(표면만 둘) — CLI↔MCP 행위 동일.

### 4.2 chain 별 진행

gstable·gwbft(static) 은 원자 명령으로 전 단계 커버가 쉽다(부품 이미 존재). gwemix(governance-etcd)
는 `deploy-governance`/`etcd-init`/`verify-etcd` 원자 명령 + 2-페이즈 조합으로 달성 — 자동
오케스트레이터가 아니라 **사용자/LLM 이 단계를 조합**해 세운다.

---

## 5. 실행 순서 (지시 6 — 단계별 지원, 개정)

각 페이즈 = 1 브랜치, 작업마다 커밋, 페이즈 완료 시 1 PR. 라이브 검증은 GSTABLE_BIN/GWEMIX_BIN 게이트로 CI green 유지.

### P1 — 분석 + 본 실행계획 + 드리프트 수정. (본 PR)

### P2 — 워크스페이스 상태 모델 + static 원자 명령 (gstable 먼저)
- data-dir 워크스페이스 레이아웃 정의·load/save(`internal/chainworkspace` 신규).
- static 경로 원자 명령: `keys`·`allocate`·`genesis`·`config`·`provision`·`init`·`start`·`stop`·`restart`·`rm`·`logs`·`status`. 각 단계 gstable 라이브 검증(GSTABLE_BIN 게이트) + offline 단위테스트.

### P3 — MCP 미러
- P2 원자 명령을 MCP 도구로 노출(핸들러=CLI 코어). LLM 이 단계 조합 가능.

### P4 — governance-etcd 원자 명령 (gwemix)
- `deploy-governance`/`etcd-init`/`verify-etcd` + 2-페이즈 조합. gwemix 단독·핸드오프를 원자 명령 조합으로 검증. supervisor `LeaderGate` 실배선(task C).

### P5 — 원격 배포 원자 명령
- `provision --remote-*`, `start`/`logs` 원격(SSH). RemoteFileSink/RemoteDriver 재사용.

### P6+ — DSL 이관 (지시 5, task D)
- 원자 CLI 로 체인 구성 전 단계 검증 완료 후 착수. 스위트 배치 이관: hardfork(8)·gas-policy(17)·accounts(35)·system-contracts(46).
- 어휘 갭이 막는 케이스는 DSL 어휘 확장(raw-tx·txpool·validator-set 등) 먼저.
- 스위트별 `tests/specs/README.md` 카운트 갱신, offline `validate` 게이트 + 라이브 `chainbench run` 확인. 이관불가 4건은 사유 유지.

### 검증 게이트 (매 푸시 전)
```sh
gofmt -l cmd internal tests            # 빈 출력
go vet ./... && go build ./...
go test -race ./...
golangci-lint run                      # unused 등 vet 미포착 항목
```
