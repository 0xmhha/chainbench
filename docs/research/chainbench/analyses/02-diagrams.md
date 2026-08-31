# 구조 다이어그램

## 1. 시스템 컨텍스트

```mermaid
flowchart TB
    Operator[Operator or CI] --> CLI[chainbench CLI]
    Agent[MCP client] --> MCP[chainbench MCP]
    Browser[Dashboard browser] --> Dashboard[Dashboard server]
    CLI --> App[Application use cases]
    MCP --> App
    CLI --> Core[Core modules]
    App --> Setup[chainsetup]
    App --> Engine[testengine]
    Setup --> Nodes[Local or remote chain nodes]
    Engine --> Nodes
    Engine --> Session[Session artifacts]
    Dashboard --> Session
```

**이 그림이 말하는 것**: CLI는 일부 core 모듈을 직접 호출하고 MCP는 app 경유를 목표로 하지만, 현재 두 표면의 경로가 완전히 통일되지는 않았다 (`docs/dev/architecture/architecture-v2.md:38`, `internal/arch/mcp_imports_test.go:17`).

## 2. 현재 모듈 의존과 중복 구성

```mermaid
flowchart LR
    CLI --> App
    CLI --> TestEngine
    CLI --> ChainSetup
    App --> ChainSetup
    App --> TestEngine
    ChainSetup --> Resource
    ChainSetup --> Launcher
    TestEngine --> Resource
    TestEngine --> Launcher
    TestEngine --> TestSpec
    TestSpec --> RPC
    TestSpec --> Session
```

**이 그림이 말하는 것**: 의존 방향보다 중요한 문제는 `ChainSetup`과 `TestEngine`이 모두 resource·launcher를 이용해 환경을 구성한다는 점이다 (`internal/chainsetup/verbs_up.go:85`, `internal/testengine/buildenv.go:71`).

## 3. 테스트 객체 소유권

```mermaid
classDiagram
    Engine --> Session : creates once
    Session *-- Environment : owns by fingerprint
    Session *-- TestRecord : owns by sequence
    Environment *-- Node : runtime view
    Workspace *-- NodeRecord : persisted composition
    ProcessLedger *-- Proc : PID source of truth
```

**이 그림이 말하는 것**: 테스트 결과 소유권은 비교적 명확하지만, workspace의 노드 상태는 topology record와 PID ledger로 나뉜다 (`internal/chainsetup/workspace.go:103`, `internal/core/process/ledger.go:17`).

## 4. 실행 수명주기

```mermaid
stateDiagram-v2
    [*] --> LoadSpecs
    LoadSpecs --> Compose
    Compose --> RunTests
    RunTests --> SaveSession
    SaveSession --> Teardown
    Compose --> PartialFailure
    PartialFailure --> CleanupRisk
    Teardown --> [*]
```

**이 그림이 말하는 것**: 정상 경로는 분명하지만 부분 구성 실패 때 cleanup closure가 반환되지 않고, 취소된 context를 teardown에 다시 쓰는 위험이 있다 (`internal/testengine/buildenv.go:117`, `internal/testengine/engine_impl.go:74`).

## 5. CLI workspace 실행

```mermaid
sequenceDiagram
    participant C as Cobra command
    participant A as app.RunSuite
    participant S as chainsetup
    participant E as testengine
    C->>A: RunSuiteIn
    A->>A: read and parse specs
    A->>S: NetUp
    A->>E: NewAttachEngine and Run
    E->>E: parse specs again
    A->>S: NetStop
```

**이 그림이 말하는 것**: 하나의 workspace 실행에서 스펙이 application과 engine에서 두 번 파싱된다 (`internal/app/workflow.go:113`, `internal/testengine/engine_impl.go:83`).

## 6. 목표 구조

```mermaid
flowchart LR
    CLI[CLI binding and rendering] --> UseCase[Run use case]
    MCP[MCP binding] --> UseCase
    UseCase --> Loader[Spec loader and validator]
    UseCase --> Provisioner[Network provisioner]
    UseCase --> Runner[Test runner]
    Provisioner --> ChainSetup[chainsetup canonical lifecycle]
    Runner --> Interpreter[DSL interpreter]
    Interpreter --> DSL[Pure DSL AST]
    Runner --> Session[Session repository]
```

**이 그림이 말하는 것**: 표면은 하나의 유스케이스를 공유하고, 구성·실행·문법·저장을 각각 하나의 소유자에게 둔다. CLI는 각 하위 계약을 별도 명령이나 테스트 seam으로 검증할 수 있다.

## 7. 리팩토링 진행 방향

```mermaid
flowchart LR
    Keyring[keyring] --> Resource[resource]
    Resource --> Node[node facts]
    Node --> Genesis[genesis]
    Genesis --> NodeConfig[nodeconfig]
    NodeConfig --> Process[process]
    Process --> ChainSetup[chainsetup]
    DSL[dsl] --> TestEngine[testengine]
    Collector[collector] --> TestEngine
    ChainSetup --> TestEngine
    TestEngine --> App[app use cases]
    App --> Surfaces[CLI and MCP]
```

**이 그림이 말하는 것**: 상위 실행 경로를 먼저 합치지 않는다. `keyring` 다음의 작은 모듈부터 계약과 산출물을 확정하고, 상위 모듈은 검증된 결과만 조합한다.
