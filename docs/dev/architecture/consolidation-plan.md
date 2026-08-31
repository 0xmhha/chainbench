# 통폐합 계획 — 55개 모듈을 관심사 단위로 (사용자 주도, 2026-08-31 확정)

> 배경: 모듈 재편(module-plan)은 파일을 주인에게 옮기는 데는 성공했지만(위반 0),
> 모듈 수는 거의 줄지 않았다(75 → 69, internal 55개). `core` 아래 30개가 평평한
> 형제로 남아 역할이 이름만으로 구분되지 않는다(300줄 미만 24개). 이 문서는
> 사용자가 확정한 목표 구조로의 통폐합 계획이다. 병행 점검:
> `docs/research/chainbench/analyses/`(다른 세션) — 결론이 이 계획과 합치한다.

## 0. 확정된 구조 (2026-08-31)

1. **DSL 문서 하나가 체인 구성과 테스트 코드를 함께 선언한다.** cmd(cli)는 DSL 이
   수행하는 모든 작업을 지원한다 — 같은 core 기능을 두 표면이 1:1 로 부른다.
2. **DSL 의 실행 주체는 chainsetup 과 testengine.** testengine 은 네 부분이다:
   ① chainsetup 수행(테스트가 선언한 체인 구성) ② pre-test hook ③ test 수행
   ④ post-test hook. **interpreter 는 testengine 내부에서 호출된다** — 전달받은
   DSL 파일을 파싱하고, 해석하면서 그 자리에서 작업을 수행한다.
   testengine → chainsetup 의존은 이 구조의 일부다(P6.1 의 게이트를 의도적으로
   뒤집는다).
3. **cmd 에 `net` 이라는 이름을 쓰지 않는다.** 구성 명령 그룹은 `chain`, 단계는
   §3 의 사전을 따른다.
4. **구성 6단계**(사용자 설명 그대로): 노드별 키 → 배치(ip·port 무충돌 약속) →
   enode 사전 생성·정리 → genesis(account·nodekey 반영, 전 노드 공통) →
   노드별 config(노드마다 다를 수 있고 수정 가능해야 한다) →
   command 빌드 → 배포 → 실행(블록 생성).

## 1. 목표 모듈 목록 (internal 55 → 약 20)

| 관심사 | 목표 모듈 | 흡수 | 근거 |
|---|---|---|---|
| 자원 — 배치·서버·저수준 통로 | `resource` | `core/netid`, `core/machine`, `core/remote`, **enode 생성**(`node/enode.go` 에서 이관) | §1h 결정(2026-08-25): netmap(→resource)이 자원 분배·enode 생성·low-level 접근의 유일 통로를 소유 |
| 노드 사실 | `core/node` | `core/topology`, `validatorset` | 선언(topology)과 사실(record)은 같은 대상의 두 면 |
| 프로세스 — 실행·정지·배포 | `core/process` | `core/driver`, `core/launcher`, `core/filestore` | 실행 관심사 하나. 층 경계(실행/정책)는 패키지 내부 파일 경계로 내려간다 — 트레이드는 §4-1 |
| 키 | `core/keyring` | `keyring/operation` 을 본체로 흡수(derive·store 는 하위로 유지) | 암호 파생과 파일 저장은 성격이 달라 하위 유지, use-case 층만 흡수 |
| genesis 빌더 | `core/genesis` | `core/hardfork` | 포크는 genesis 의 시간축 |
| config 빌더 | `core/nodeconfig` | `core/launchopt`, `core/config` | 노드 하나의 설정(파일·argv·해석)이 한 곳 |
| DSL 문법 | `dsl` | `testspec` 개명(문법·파싱·검증·fingerprint 만), `testspec/assert` | interpreter·run·binding 은 testengine 으로 |
| 구성 실행 | `chainsetup` | (단계를 §3 사전으로 개편) | |
| 테스트 실행 | `testengine` | + interpreter(테스트 실행기), 4단계 구조 | §0-2 |
| 테스트 어휘 | `testhelper` | 유지(+잔여 34건의 액션이 이리로) | P8 |
| 실사 / 비교 | `core/inspector` / `core/preflight` | inspector ← `core/health` / preflight 유지 | 실사는 한 눈 |
| 관측 | `core/collector` | `core/obs`, `core/logs` | 이벤트·수집·로그 읽기 |
| 체인 정의 | `chains/*`, `consensus/*`, `core/registry`, `accounts` | 유지. `core/consensus`(23줄)는 registry 로, `core/capability` 는 registry 또는 testengine 으로 | 소품 정리 |
| 표면 | `cmd/*`, `mcp`, `app` | `netcmd` → `chaincmd`(§3), `core/netreg` → mcp 로 흡수, `core/session` 유지(아티팩트 소유) | app 은 MCP 경유층으로 유지(§1h) |
| 은퇴 | — | `testkit`, `core/pipeline/testrun`, `test` cmd, MCP `chainbench_test` | 잔여 34건 이관 완료 시점에 한 번에 |

목표 수: internal 트리 기준 **약 20 + chains/consensus 정의 8** (지금 55).

## 2. 실행 순서 — R 단계 (모듈), 그다음 C 단계 (표면)

각 단계의 공통 게이트: `go build/vet/test ./...` · `-race`(손댄 패키지) · lint 0 ·
`internal/arch` 통과 · 그래프 재생성(패키지 수 감소 확인) · 실행 경로를 건드린
단계는 stablenet 라이브 케이스(`run --workspace-dir tests/cases/stablenet/…`) 1회.

### R1. 소형 흡수 (독립적, 낱개 커밋·PR 1~2개)

기계적 이동 + 개명. 서로 독립이라 순서 무관.

| 이동 | 줄 | 비고 |
|---|---|---|
| `core/topology`, `validatorset` → `core/node` | 149+85 | 선언 타입은 declaration.go 로 |
| `core/hardfork` → `core/genesis` | 128 | |
| `core/launchopt`, `core/config` → `core/nodeconfig` | 935+152 | argv 조립·설정 해석이 한 지붕 |
| `core/health` → `core/inspector` | 187 | |
| `core/obs`, `core/logs` → `core/collector` | +207 | obs.Bus 는 collector 의 이벤트 면 |
| `core/netid` → `resource` | 45 | |
| `core/netreg` → `mcp` | 161 | 소비자가 mcp 뿐 |
| `core/consensus` → `core/registry`, `core/capability` → 소비자 조사 후 registry/testengine | 23+222 | |

### R2. DSL 분리 — 문법과 실행기

- `testspec` → **`dsl`** 로 개명, 문법·파싱·검증·fingerprint·Registry 계약만 남긴다.
- interpreter(run.go·binding·resolve 실행부) → **`testengine`** 으로 이관.
- 게이트: `dsl` 의 out-edge 에 실행 인프라(rpc·session·collector) 0.

### R3. 프로세스·자원 관심사

- `core/driver` + `core/launcher` + `core/filestore` → `core/process`.
- `core/machine` + `core/remote` → `resource` (low-level 접근의 유일 통로).
- enode 생성(`node/enode.go`) → `resource` (공개키는 입력; §1h 결정 이행).

### R4. 구성 경로 단일화 + testengine 4단계

- `testengine.Run`: DSL 파싱 → chainsetup 수행(env 선언·preflight 재사용 판단)
  → pre hook → test(interpreter) → post hook.
- `testengine.NewBuildEnv`(자체 조립)와 `NewLocalEngine` 의 빌드 경로 삭제 —
  구성 소유자는 chainsetup 하나.
- `app.RunSuite` 는 MCP 경유용 위임으로 축소.

### R5. 은퇴 (독립 트랙과 연동)

- 레거시 34건 이관(문법 갭 묶음별) 완료 → `testkit`·`pipeline/testrun`·`test`
  cmd·MCP `chainbench_test` 를 한 번에 제거.

### C 단계 — 표면 재정리 (제안 1, R4 이후)

- `net` 그룹 폐기 → **`chain`** 그룹: `keys → place → enode → genesis → config →
  build → deploy → run` (+`up`, `stop/status/resume`). 단계 사전과 1:1.
- 키가 배치보다 먼저. enode 는 조회 가능한 산출물. **config 는 노드 단위**
  (`chain config --node N --set k=v`; DSL 은 `config.all` + `config.node<N>`).
- CLI ↔ DSL 1:1 대응표를 문서·테스트로 고정(사전에 없는 명령·어휘 금지).
- 완료 후 케이스 4갈래 라이브 재검증.

## 3. 단계 사전 (C 단계의 정본)

| # | 단계 | 산출물 | DSL(env) | CLI |
|---|---|---|---|---|
| ① | keys | 노드별 nodekey·keystore·BLS | `nodes`·`keys` | `chain keys` |
| ② | place | 노드별 host·port 표 (무충돌) | `topology`·`servers` | `chain place` |
| ③ | enode | 노드별 enode 목록 | ①②에서 파생 | `chain enode` |
| ④ | genesis | genesis.json (전 노드 공통) | `genesis` | `chain genesis` |
| ⑤ | config | 노드별 config (노드 단위 수정) | `config.all`·`config.node<N>` | `chain config` |
| ⑥ | build | 노드별 실행 command | `binaries`·`launch` | `chain build` |
| ⑦ | deploy | 머신 위의 파일들 | `servers` | `chain deploy` |
| ⑧ | run | 살아 있는 체인 | — | `chain run` |

## 4. 트레이드와 열린 결정

1. **driver/launcher 합병**: 실행(L1)과 정책(L3)의 경계가 패키지 사이에서 파일
   사이로 내려간다. 컴파일러 강제가 줄어드는 대신 모듈 수가 준다 — 사용자 방향
   (관심사 통합)을 따른다. 진입점 3개(실행·기동 정책·핸드오프)는 타입으로 유지.
2. **keyring 하위(derive·store)**: 성격이 달라(암호/파일) 하위 패키지로 유지하는
   안을 기본으로 한다. 더 합치고 싶으면 R1 에서 결정.
3. **session**: 아티팩트·컴포지션·잠금의 소유자로 유지. 재측정은 R4 뒤.
4. P6.1 의 게이트("testengine → chainsetup 0")는 §0-2 로 **대체**된다. arch 테스트를
   새 구조(테스트엔진이 구성을 소유)로 고쳐 쓴다.
