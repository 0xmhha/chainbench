# 단계(stage) 생애 주기 설계

**목적.** 지금 코드는 이미 단계로 나뉘어 돌지만, 그 단계가 **문자열**이고 **성공만
기록**하며 **testengine 에는 없다.** 그래서 실패했을 때 "어느 단계에서 무엇이
잘못됐는지" 를 코드도 사람도 말하지 못한다. 이 문서는 그 셋을 고치는 설계다.

**측정 방법.** 아래 수치는 `go/ast` 로 `internal/chainsetup` 과
`internal/testengine` 의 함수 307개를 파싱해 얻었다(단계 기록 호출, 선행 조건
호출, 호출 관계). 손으로 센 것이 아니다.

---

## 1. 지금 있는 것

### 1.1 단계는 16개다

`markStep` 을 부르는 함수를 AST 로 전수 조사한 결과다.

| 묶음 | 단계 |
|---|---|
| 구성 | `new` `place` `keys` `genesis` `config` `build` `deploy` `init` `start` |
| 운영 | `stop` `restart` `rm` `hardfork` |
| 노드 단위 | `start-node` `stop-node` `swap-node` |

`composeNeeds` 표에 적힌 것은 이 중 6개뿐이다.

### 1.2 선행 조건은 의존 집합이지 전이 그래프가 아니다

```go
var composeNeeds = map[string][]string{
    "config": {"place", "keys"},   // 둘 다 끝나 있어야 한다 (AND)
}
```

`require` 는 목록의 **전부**가 기록돼 있는지 본다. "어느 단계에서 올 수 있는가"
(택일)가 아니라 "무엇이 먼저 끝나 있어야 하는가"(전부)다. 설계에서 이 둘을 섞으면
안 된다. **의존 집합은 그대로 두고 그 위에 상태를 얹는다.**

### 1.3 검사하는 단계가 4개뿐이다

`require` 를 실제로 부르는 함수는 `Genesis` `Config` `LaunchOpts` `Provision`
넷이다. **표에 적힌 `place` 와 `keys` 는 검사하지 않고**, `init` 과 `start` 는 표에도
없고 검사도 없다. 초기화가 실행보다 먼저라는 보장이 **호출 순서에만** 있다.

### 1.4 재사용 판정이 세 곳에 흩어져 있다

- `Workspace.reconcileReuse`(`reuse.go`) — 노드 단위로 재사용 가능한지
- `Workspace.Compare`(`steps_preflight.go`) → `preflight.Check` — 구성 전체가
  원하는 것과 같은지(reuse / rebuild-nodes / rebuild-all / compose)
- `composeWorkspace`(`testengine/suite.go`) — 그 판정을 받아 무엇을 할지

세 곳이 서로 다른 시점에 서로 다른 질문을 한다. 그래서 오늘 `rebuild-all` 이
"다시 만들자" 고 결정해 놓고 데이터 폴더를 안 비우는 일이 생겼다.

### 1.5 이미 `UpStage` 타입이 있다

`type UpStage string` 에 값이 둘(`deploy`, `start`)이고 **"어디까지 갈까"** 를
뜻한다. 지금 설계의 "지금 어디인가" 와는 다른 축이다. 이름이 겹치므로 정리한다.

---

## 2. 무엇이 없나

**하나, 실패가 기록되지 않는다.** `markStep` 은 성공 뒤에만 불린다. 실패한 단계는
파일 어디에도 안 남는다. 시작 시각도 소요 시간도 없다.

**둘, testengine 에 단계가 없다.** 한 덩어리로 흘러가서, 준비 판정 실패가 어느
단계의 실패인지 부를 이름조차 없다.

**셋, 실패 자료 수집이 테스트 기록에 묶여 있다.** `collectFailureData` 는 호출
지점이 하나(`OnFail`)이고 `session.TestRecord` 를 받는다. 테스트가 시작되기 전에
실패하면 붙일 데가 없어 **아무것도 안 남는다.**

---

## 3. 설계

### 3.1 단계를 enum 으로

```go
// Stage is one step of a composition or a test run. It is an integer so the
// order is the type's own: a later stage compares greater, and a record that
// says "failed at Deploy" needs one byte, not a string.
type Stage uint8

const (
    StageNone Stage = iota
    StageNew        // 워크스페이스와 체인 정체성
    StageSurvey     // 재사용 판정 (옛 place 의 앞부분)
    StagePlace      // 노드 표: 역할·머신·경로·포트
    StageKeys
    StageGenesis
    StageConfig
    StageBuild
    StageDeploy
    StageInit
    StageStart
    StageVerify     // 계획대로 떴는가 + 블록이 나오는가
)
```

문자열은 파일에 적을 때만 쓴다(`String()` 과 JSON 변환). 비교와 저장은 정수다.

### 3.2 `place` 이름 검토 — 쪼개기를 제안한다

**지금 `place` 가 하는 일**은 노드 표를 만드는 것이다. 역할별 개수를 정하고, 각
노드를 어느 머신에 놓을지 배정하고, 데이터 디렉터리와 포트를 결정적으로 할당한다.
출력이 그대로 말한다.

```
place: 4 node(s): 3 bp + 1 en + 0 pn; ports: p2p from 31000, http from 8600
```

**그 이름은 그 일에 정확하다.** 배치가 곧 이 단계의 산출물이다. 바꿀 이유가 없다.

바꿔야 할 것은 이름이 아니라 **범위**다. 지금은 "새로 세울지 재사용할지", "서버셋
값이 유효한지", "워크스페이스가 쓸 만한지" 를 판정하는 일이 `place` 앞뒤와 다른
모듈로 흩어져 있다(§1.4). 그 판정을 **`survey` 라는 앞 단계로 모은다.**

`survey` 가 답하는 것은 셋이다. 이 워크스페이스가 유효한가, 서버셋이 요구한 자원을
줄 수 있는가, 이미 구성된 것을 재사용할 수 있는가. 셋 중 하나라도 "아니오" 면 **그
자리에서 끝난다.** 얼리 리턴이 한 자리에 모인다.

이름 후보를 견줬다. `survey`(살펴본다)가 가장 맞다. `check` 는 무엇을 검사하는지
말하지 않고, `plan` 은 이미 `ComposePlan` 이 쓰고 있으며, `assess` 는 결과가
점수처럼 들린다. `survey` 는 "자원과 현황을 조사한다" 는 뜻이 분명하고, 그
산출물이 `place` 의 입력이 된다는 순서도 자연스럽다.

`UpStage` 는 "어디까지 갈까" 라는 다른 축이므로 **`UpTarget` 으로 개명**해 이름
충돌을 없앤다.

### 3.3 기록을 시작과 끝 양쪽에 남긴다

```go
type StageRecord struct {
    Stage     Stage
    StartedAt time.Time
    EndedAt   time.Time
    Result    StageResult // running | done | failed | skipped(reused)
    Detail    string
    Err       string
}
```

단계에 들어갈 때 `running` 으로 적고 나올 때 결과를 채운다. **죽어도 마지막 기록이
`running` 으로 남아 어디서 죽었는지 파일이 말한다.** 오늘 겪은 문제의 절반이 이
한 가지로 사라진다.

`skipped(reused)` 를 둔 이유는 재사용이 실패가 아니기 때문이다. 지금은 재사용하면
단계가 아예 기록되지 않아, 나중에 읽는 사람이 "안 돌았다" 와 "돌 필요가 없었다" 를
구분하지 못한다.

### 3.4 실패를 상태로 다룬다

단계가 실패하면 **그 자리에서 실패 경로로 빠진다.** 실패 경로가 하는 일은 셋이다.
기록에 결과를 적고, **자료를 수집하고**, 정리할 것을 정리한다.

자료 수집을 `session.TestRecord` 에서 떼어 **`StageRecord` 에 붙인다.** 그러면
준비 판정이든 세우기든 테스트든 같은 수집이 돈다. §2 의 셋째가 구조적으로
사라진다.

### 3.5 testengine 도 같은 타입을 쓴다

```go
const (
    StageChainReady Stage = iota + 64 // chainsetup 이 끝난 상태를 확인
    StagePreTest
    StageTest
    StagePostTest
    StageReport
    StageTeardown   // 유지할지 내릴지 결정
)
```

핵심은 **testengine 이 chainsetup 의 끝난 상태만 보고 다음을 정한다**는 것이다.
체인이 이미 구성돼 있으면 `StageChainReady` 에서 바로 다음으로 간다. 지금처럼
판정 문자열을 주고받지 않는다.

테스트 중의 노드 재시작은 **상태를 바꾸지 않는다.** 이미 정해진 함수가 `StageTest`
안에서 도는 동작이다. 모든 동작이 상태를 바꾼다고 보면 단계가 무한히 늘어난다.

---

## 4. 이 설계의 단점

**작업량이 적지 않다.** 단계 기록을 쓰는 자리가 16곳, 읽는 자리가 그만큼 있고,
`chain-record.json` 의 형식이 바뀌므로 `StateFormatVersion` 을 올려야 한다.

**단계가 12개로 는다.** 지금 9개(구성)에서 `survey` 와 `verify` 가 늘었다. 늘어난
만큼 "이 작업은 어느 단계인가" 를 판단할 일이 생긴다. 그래서 §3.5 의 규칙 — 정해진
함수가 도는 것은 상태를 바꾸지 않는다 — 을 문서에 박아 둔다.

**기록이 커진다.** 단계마다 시작·끝·결과를 적으므로 `chain-record.json` 이 지금보다
길어진다. 다만 노드 로그가 이미 실행당 수십 MB 라 상대적으로는 작다.

---

## 5. 순서

1. `Stage` enum 과 `StageRecord` 를 만들고 `markStep` 을 흡수한다.
2. `require` 가 `composeNeeds` 를 enum 으로 읽게 바꾸고, **검사하지 않던 단계
   (`place` `keys` `init` `start`)를 표에 넣는다.**
3. 실패 경로를 만들고 자료 수집을 `StageRecord` 에 붙인다.
4. 재사용 판정 세 곳을 `survey` 로 모은다.
5. testengine 에 단계를 넣는다.

1~3 이 오늘 겪은 문제를 직접 겨냥한다. 4~5 는 그 위에 쌓는다.

---

## 6. 결과물이 쌓이는 자리 (2026-09-17 확정)

### 6.1 구조

```
<루트, 기본 ~/.chainbench, 사용자가 덮을 수 있음>/
  <세션 id>/
    <실행 시각 YYYYMMDD-HHMMSS>/
      001_<테스트 id>/      ← 그 테스트의 결과와 증거 (실패 원인이 망 구축이어도 여기)
      002_<테스트 id>/
      report.<확장자>       ← 테스트 폴더들과 같은 층
```

체인 구성만 지시한 실행은 테스트가 없으므로, 테스트 폴더 대신 **무엇을 구성했는지**
로 폴더를 만든다.

### 6.2 세션 id

**실행을 주도하는 프로세스**가 세션이다. MCP 는 연결이 오래 살아 있으니 한 세션
아래 여러 실행이 쌓이고, CLI 는 한 번 돌고 끝나니 한 세션에 한 실행이다. 둘을
같은 규칙으로 설명할 수 있다.

값은 `<프로세스 시작 시각 UTC 초>-<pid>` (예: `20260917-060312-41210`).
**결정론적이다** — 한 프로세스 안에서 몇 번을 물어도 같은 값이라 실행 중간에
폴더가 갈라지지 않는다. **겹치지 않는다** — 같은 초에 같은 pid 가 둘일 수 없다.
`os.Getpid()` 는 표준 라이브러리 한 줄이고 이 저장소가 이미 `session/lock.go` 에서
쓴다. 폴더 이름의 pid 로 **어느 프로세스가 만들었는지 추적**할 수 있는 것이 덤이다.

환경 변수는 쓰지 않는다 — 한 머신의 모든 실행에 같은 값이 걸려 구분이 안 된다.

### 6.3 폴더를 만드는 시점

**체인을 구성하거나 테스트를 실행하는 경로에서만**, 그리고 **실제로 필요할 때**
만든다. 키 생성이나 genesis 검증까지 폴더를 만들면 금세 쌓이고, 미리 만들면
아무것도 하지 않고 끝난 실행이 빈 폴더를 남긴다.

### 6.4 망 구축 실패도 그 테스트의 실패다

`run` 에 정의서 다섯 개를 주면 **다섯 번의 독립된 실행**이다. 각각이 망을 세우고
각각이 실패할 수 있다. 그러므로 망 구축 실패의 증거도 **그 테스트의 폴더 아래**
남는다. 무엇을 하려다 실패했는지가 그 자리에서 읽혀야 한다.

로그가 커도 테스트마다 모은다. 한 벌만 두고 가리키게 하면 **나중에 어느 테스트의
실패인지 되짚기 어렵다.**

### 6.5 구현된 모습 (2026-09-17)

셋 다 들어갔다.

```
~/.chainbench/sessions/          ← 기본 루트, --artifact-root 로 덮는다
  20260917-073226-57155/         ← 세션: 프로세스 시작 시각 + pid
    UTC-20260917-073226/         ← 실행
      tests/001_basic-consensus/
        status.json              ← blocked, 이유는 망 구축 실패 문장 그대로
        observations/            ← 그 실패의 증거
      report.json                ← 테스트 폴더들과 같은 층
```

**순서를 뒤집었다.** 세션은 이제 망을 세우기 **전에** 열리고, 엔진은 그것을 받아
쓴다. 그래서 망이 안 뜨면 그 테스트의 폴더가 만들어지고, 판정이 `blocked` 로
적히고, 증거가 그 아래 `observations/` 로 간다. 워크스페이스의 `failures/` 는
없앴다.

**루트를 워크스페이스 밖으로 옮겼다.** 워크스페이스는 임시 공간이라 디스크를
비우려고, 또는 깨끗하게 다시 구성하려고 지운다. 그때마다 판정과 로그가 같이
사라졌다. 체인이 있는 곳과 그 체인을 시험한 기록이 남는 곳은 수명이 다르다.

**세션 층은 `session.New` 한 곳에서 붙인다.** 트리가 어디에 놓이는지는 원래 그
함수가 소유하므로, 구성 경로·attach 경로·워크스페이스 attach 경로가 저절로 같은
규칙을 따른다. `session.List` 는 한 단계 더 내려가고, 찾은 id 는 프로세스
디렉터리를 포함한 상대 경로다(`IDFor`). 옛 평면 배치도 계속 읽힌다.

### 6.6 남은 이름 빚

코드의 `session.Session` 은 **실행 하나**를 뜻한다. 디스크에서 세션은 그 위층이다.
이름이 어긋나 있다. 고치려면 `session.json` 과 그 170여 군데를 건드려야 하므로
따로 잡는다.
