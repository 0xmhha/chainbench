# 모니터링 이슈 재검토와 수정 계획 (2026-09-10)

검토 기준 HEAD: `2cc82692` (main, PR #370 반영)
대상 문서: `docs/research/chainbench/analyses/11-monitoring-open-issues.md` (MON-001~016)
경로 표기: 저장소 루트(`/Users/wm-it-25_0220/Work/github/chainbench`) 기준 상대 경로.

이 문서는 모니터링 세션이 제기한 16건을 **현재 코드에서 다시 판정한 결과**와, 그로부터
나온 수정 계획이다. 모니터링 문서 자체는 다른 세션이 관리하므로 여기서 고치지 않는다.
제거·정정이 필요한 항목은 5절에 근거와 함께 적는다.

## 1. 검토 방법과 한계

각 항목을 입력 조건 → 호출 → 저장 → 실제 사용 → 출력까지 코드로 추적했다. 문서에 적힌
줄 번호나 함수 이름을 그대로 믿지 않고 현재 코드에서 다시 찾았다.

- 실행한 테스트(읽기 전용): `go test ./internal/resource/ -run TestParseWorkspaceConfig`,
  `go test ./internal/testengine/ -run TestCompositionOf_WorkspaceConfigDataRoot`. 둘 다 통과.
- 저장소 **밖** 스크래치에서 Go 동작만 확인한 것 3건: nil map 대입, `os.WriteFile`의 기존
  파일 권한, RLP/`big.Int` 오버플로. 저장소에 파일을 만들지 않았다.
- docker·실제 원격 서버·실제 개인 키는 열지 않았다. 따라서 아래 판정은 **정적 추적과
  국소 재현**이며, 라이브 재현이 필요한 항목은 6절 검증 계획에 따로 적는다.
- 기준 시점의 전체 테스트(`./internal/... ./cmd/...`)는 통과 상태였다. 즉 아래 결함들은
  모두 **기존 테스트가 잡지 못하는** 것들이다.

## 2. 판정 요약

유효 14건, 해결·검증 완료 2건. 오탐은 없었다.

| MON | 판정 | 핵심 근거 | 우선순위 |
|---|---|---|---|
| 001 | 유효 | `internal/core/node/record.go` Key 영속 + `internal/chainsetup/verbs_up.go` Request 2중 저장 | 높음 |
| 002 | 유효 | `internal/core/keyring/store/declared.go` 조기 반환으로 검증 미도달 | 높음 |
| 003 | 유효 | `internal/dsl/spec_v2.go` nil map 대입 | 보통 |
| 004 | 해결·검증완료 | 두 sample 모두 tracked, 테스트 통과 확인 | — |
| 005 | 유효 | `env/docker/gen-env.sh`가 `internal/resource/serverset.go`가 거부하는 파일을 생성 | 높음 |
| 006 | 해결·검증완료 | `internal/testengine/compose.go` 충돌 거부, 테스트 통과 확인 | — |
| 007 | 유효(확대) | detail 오보 + **capability 오보**까지 | 높음 |
| 008 | 유효 | `internal/chainsetup/record.go` 평면 Layout + 오류 무시 | 보통 |
| 009 | 유효 | 판정 전에 실행 경로 파일을 덮어씀 | 높음 |
| 010 | 유효(확대) | 서버가 하나여도 구성 간 충돌, 이후 오정지 가능 | 높음 |
| 011 | 유효 | `internal/core/filestore/filestore.go` 기존 파일 권한 미변경 | 높음 |
| 012 | 유효 | `internal/consensus/wbft/extradata_decode.go` 음수 길이 → 슬라이스 panic | 보통 |
| 013 | 유효 | 샘플 안내가 실제 소비 범위를 넘어섬 | 보통 |
| 014 | 유효(확대) | `internal/consensus/poa/validators.go` 부호·범위 미검사 + RPC 폭주 | 높음 |
| 015 | 유효(확대) | `--json` 혼입 + **종료 코드 구분 소실** | 보통 |
| 016 | 유효 | 과거 해시 비교 + 관측 누락을 일치로 처리 | 높음 |

## 3. 문서보다 심각하거나 범위가 넓은 항목

### MON-009 — 구성 격리가 보호해 주지 않는다

`compositionId`는 워크스페이스 디렉터리의 해시이고(`internal/chainsetup/new.go`) 한 번
정해지면 바뀌지 않는다. reuse-if-matching은 정의상 **같은 워크스페이스**를 다시 up 하므로
격리 경로가 이전 실행과 동일하다. 따라서 genesis·config 쓰기는 실행 중인 노드가 쓰던
바로 그 파일을 덮어쓴다. 격리는 서로 다른 워크스페이스를 갈라줄 뿐, 같은 워크스페이스의
연속 실행을 갈라주지 않는다.

거부 경로에 복원이 없고, 각 단계가 성공·실패 모두에서 상태를 저장하므로 이전 해시도
남지 않는다. 프로세스는 계속 돌지만 디스크의 신원이 어긋나며, 이후 `restart`·`init`이
거부된 내용으로 부팅한다.

기존 테스트 `TestReconcileReuse_GenesisChangeRefusesAndTouchesNothing`은 이름과 달리
메모리의 PID 필드만 확인한다. 파일에 대해서는 아무것도 증명하지 않는다.

### MON-007 — 기록만이 아니라 capability도 틀린다

`internal/chainsetup/steps_compose.go`의 `networkCapabilities` 호출은 existing 분기에서도
실행된다. `env.hardforks`로 들어온 `<fork>Block=N`이 실제 genesis 파일과 무관하게
`delayed-<fork>` capability로 광고되고, capability로 게이트된 테스트가 그 fork가 없는
체인에서 실행된다. 원 문서에 없던 파급이다.

### MON-010 — 서버가 하나여도 발생한다

`path.Base(datadir)`가 버리는 것은 서버만이 아니라 `compositionId`다. 한 서버에 두 구성이
있으면 둘 다 basename이 `node1`이 되어 충돌한다. 잘못 연결된 PID는 이후 재작업에서
`stopByPID`로 **다른 구성의 노드를 정지**시킬 수 있다.

## 4. 문서에 없던 추가 발견

새 항목 후보(MON-017 이후 번호는 모니터링 세션이 부여).

- **N1. 파일 스토어 mode 계약 불일치**: 원격 쓰기는 `chmod`를 명시 실행하는데 로컬
  `filestore.Local.Write`는 생성 시에만 mode를 적용한다. 같은 인터페이스의 두 구현이
  다르게 동작한다. MON-011의 실제 원인이며, `internal/chainsetup/steps_compose.go`와
  `internal/consensus/upgrade/handoff.go`의 비밀 쓰기도 같은 원인에 걸린다.
- **N2. 실행 기록에 노드 config가 없다**: `recordRun`은 manifest·launch-commands·genesis만
  모은다. genesis는 잘못된 경로에서 읽고, config는 아예 수집하지 않는다.
- **N3. 순차 실행에서 종료 코드 구분이 사라진다**: 단일 실행은 실패 1 / blocked 2를
  구분하지만 순차는 일반 오류를 반환한다.
- **N4. data-root 충돌 규칙이 2중 구현**: `internal/testengine/compose.go`와
  `internal/app/workspaceconfig.go`에 메시지가 다른 두 구현이 있다. 공통 소유자로
  `internal/resource`가 가능하다(순환 없음 확인).
- **N5. `env/docker/README.md`가 생성되지 않는 파일을 안내**: `server-set-wemix.yaml`은
  `gen-env.sh`가 만들지 않는다.

**최종 report 범위**: 현재는 정의서마다 세션 1개와 `report.json` 1개가 생성된다. 명령
전체를 묶는 통합 report 파일은 없고, CLI가 정의서별 집계를 출력할 뿐이다. 이는 결함이
아니라 미구현 후속이며, 필요하면 별도 항목으로 세운다.

## 5. 모니터링 문서에서 제거·정정할 항목

**제거(해결·검증 완료)**

- **MON-004**: `workspace-config.sample.yaml`과 `server-set.sample.yaml` 모두 HEAD에
  tracked. `TestParseWorkspaceConfig_Sample` 실행 통과. 샘플에 실제 사이트 값 없음.
- **MON-006**: `internal/testengine/compose.go`에 충돌 거부가 있고 회귀 테스트 통과.
  단계형 CLI·MCP도 `internal/app/workspaceconfig.go`로 같은 규칙을 적용한다.
  다만 규칙이 두 곳에 있는 것은 N4로 따로 남긴다.

**정정(내용 확대)**: MON-007, MON-009, MON-010, MON-011(심볼릭 링크·`DownloadDir`),
MON-014(uint256 최대값·RPC 폭주), MON-015(종료 코드).

## 6. 수정 계획

작은 모듈 → 조합 모듈 → 표면 순서. 각 Phase는 앞 Phase에 의존한다.

### Phase A — 프리미티브·합의 패밀리 (서로 독립, 병렬 가능)

| 작업 | 범위 | 완료 기준 |
|---|---|---|
| A1 (MON-012) | `internal/consensus/wbft/extradata_decode.go` | 과대 길이·잘린 헤더·최대값에서 panic 없이 오류. 파서 fuzz 추가 |
| A2 (MON-014) | `internal/consensus/poa/validators.go` | uint256 ABI 형식 검증과 상한. 음수·범위 초과·과대 멤버 수를 할당 전에 거부 |
| A3 (MON-003) | `internal/dsl/spec_v2.go` | null·배열·스칼라 env를 오류로. 파일 기반 사전 검증 경로 포함 |
| A4 (MON-011·N1) | `internal/core/filestore/filestore.go` | 로컬 Write가 mode를 강제(생성·기존 동일). 심볼릭 링크 안전 처리. `DownloadTo`는 이 계약을 사용 |

A4를 `DownloadTo`가 아니라 스토어에서 고치는 이유: 같은 원인이 걸린 비밀 쓰기 여러 곳을
한 번에 없애고 원격 구현과 계약을 맞추기 위해서다. 표면·호출부마다 중복 수정하지 않는다.

### Phase B — 키와 비밀 (사용자 결정 필요)

- **B1 (MON-001)**: 인라인 키가 `State.Nodes[].Key`와 `State.Request` 두 곳에 남는다.
  결정에 따라 인라인 거부 또는 저장 시 참조화.
- **B2 (MON-002)**: 기존 링 재사용은 유지하고, 명시 키와 기존 신원이 다르면 자원을
  바꾸기 전에 거부. 단계 detail이 "declared"라고 보고하는 것도 실제와 맞춘다.

### Phase C — 구성 오케스트레이션 (`internal/chainsetup`)

- **C1 (MON-009)**: 후보 입력 생성과 실행 경로 반영을 분리. 거부 시 파일·해시·PID 보존.
- **C2 (MON-010)**: 발견 결과에 서버와 전체 datadir 보존, 요청 노드와 위치 대조,
  같은 label의 복수 후보를 묵시적으로 선택하지 않음.
- **C3 (MON-008·N2)**: `recordRun`이 `state.GenesisPath`와 올바른 머신을 사용. 노드 config
  수집 추가. 수집 실패를 성공으로 숨기지 않음.
- **C4 (MON-007)**: existing과 변경 옵션(chain id·hardfork·overlay) 충돌을 공통 core에서
  거부. capability 산출도 같은 규칙을 따름.
- **C5 (MON-016)**: 결정에 따라 현재 파일 기준 관측 또는 범위 명확화. 관측 누락을 일치로
  처리하지 않음.

C1과 C5는 "대상의 현재 파일을 읽어 해시한다"는 같은 기능이 필요하다. **공용 헬퍼 하나를
만들어 둘이 재사용한다.**

### Phase D — 표면·환경

- **D1 (MON-015·N3)**: `--json`이면 stdout 전체가 기계 판독 형식, 진행 설명은 stderr.
  순차 실행도 종료 코드 계약을 따름.
- **D2 (MON-005·N5)**: `gen-env.sh`가 새 server-set 계약을 따르고, 필요한 workspace-config를
  생성하거나 준비 절차를 명시. README 정정.
- **D3 (MON-013)**: 샘플 안내를 실제 범위로 정정(또는 결정에 따라 기능 구현).
- **D4 (N4)**: data-root 충돌 규칙을 `internal/resource`로 모아 두 구현 제거.

## 7. 검증 계획

- **단위·회귀**: 각 이슈의 제거 조건을 그대로 테스트로 만든다. 특히 C1은 **실제 `up`
  경로**에서 거부 전후의 genesis/config 바이트·기록 해시·PID를 비교한다(현재 테스트가
  메모리 필드만 보는 결함을 메운다).
- **Docker**: C1(거부 후 보존), C2(한 서버 두 구성 충돌), C5(승인 후 서버 파일만 변경),
  D1(stdout 전체 파싱), D2(새 체크아웃 → 생성기 → 실행).
- **실제 원격 서버**: 별도 승인 없이 수행하지 않는다.
- 각 Phase 종료 시 `go build`·`go vet`·`gofmt`·`go test`·`golangci-lint`와 arch·feature
  래칫을 통과시킨다.

## 8. 확정된 결정 (2026-09-10 승인)

1. **MON-001 — 인라인 개인 키를 거부한다.** 노드 키는 로컬 파일 경로로만 받는다.
   `srv://` 거부는 그대로 두고, `0x`-hex 원문은 오류로 만든다. 상태·요청 기록 어디에도
   원문이 들어올 경로 자체를 없앤다. 인라인 형식을 쓰던 기존 테스트는 키 파일로 옮긴다.
2. **MON-016 — 현재 파일을 읽어 검사한다.** 검사 시점에 대상에서 genesis와 노드 config를
   실제로 읽어 해시한다. 파일 없음·읽기 실패는 일치가 아니라 오류다. 관측 누락을 일치로
   처리하던 문제도 함께 고친다. 검사는 baseline과 실행 환경을 바꾸지 않는다.
3. **MON-009 — 판정을 쓰기 앞으로 옮긴다.** 후보를 만들어 해시로 판정하고, 통과한 뒤에만
   대상에 쓴다. 후보·반영의 완전 분리(부분 재사용 품질)는 별도 후속으로 남긴다.
4. **MON-013 — 안내를 실제 소비 범위로 낮춘다.** `binaryAliases`와 객체형 참조의 구현은
   별도 후속으로 남긴다. 주석 예시가 실제로는 파싱되지 않는다는 점도 명시한다.
5. **MON-005 — 생성기가 workspace-config까지 만든다.** server-set에서 `dataRoot`를 빼고
   같은 값을 담은 workspace-config를 함께 생성해 "손으로 쓰는 파일 0"을 유지한다.
   README의 없는 파일(`server-set-wemix.yaml`) 안내도 함께 고친다.
6. **PR은 하나로 묶는다.** 내부 순서는 6절 그대로 A → B → C → D.

### 이 결정에서 파생된 후속(이번 범위 밖)

- 후보·반영 완전 분리(부분 재사용에서 승인된 노드만 정지·반영·재기동)
- `binaryAliases`와 `{server,ref}`·`serverIndex`·`localPath` 객체 참조의 실제 소비
- 여러 정의서 실행의 통합 report
