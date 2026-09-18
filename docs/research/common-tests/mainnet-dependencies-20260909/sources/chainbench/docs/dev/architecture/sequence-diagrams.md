# 시퀀스 다이어그램

> **[이력]** 2026-08-11 시점.
> **현재 상태를 말하지 않는다.** 그때 무엇을 측정·결정했는지의 기록이다.
> 현재 상태는 [[chainbench-worklist]] 와 코드가 정본이다.

> 2026-08-11 코드 실측(keyreg 배선 포함). 자매 문서: [아키텍처](software-architecture.md) · [컴포넌트](component-diagram.md) · [상태](state-diagrams.md)
> 각 다이어그램은 배경 문서의 **알고리즘 1~15** 단계 번호를 주석으로 단다.

---

## 1. 전체 실행 — `chainbench run` (local 모드)

알고리즘 1~15 전 구간.

```mermaid
sequenceDiagram
    autonumber
    actor U as 사용자
    participant CLI as cmd/chainbench run
    participant E as engine.Engine
    participant KS as KeySource
    participant S as session
    participant B as BuildEnv
    participant SUP as supervisor
    participant I as Interpreter
    participant N as 노드 프로세스

    U->>CLI: run spec.json --chain --binary --keys-source
    CLI->>CLI: readSpecFiles
    CLI->>E: NewLocalEngine(LocalConfig{Keys: KeySource})
    CLI->>E: Run(ctx, specs)

    rect rgb(232,244,234)
    note over E,S: 세션 시작 — 키 자료가 먼저 (알고리즘 2·3)
    E->>KS: Ensure(ctx, validators)
    alt preset
        KS->>KS: keys.LoadPreset(dir) + 노드수 검증
    else generate
        KS->>KS: keygen.GeneratePreset (bootnode 로 BLS/PoP)
    end
    KS-->>E: KeySet{Dir, Preset}
    E->>S: NewWithKeys(root, cmd, at, keyreg.Deps)
    S->>S: .chainbench 아래 세션 트리 — keys · environments · tests
    S-->>E: Session — Keys() 는 세션 keys 디렉토리에 루팅된 keyreg
    E->>S: RegisterIdentities(node1..nodeN)
    S->>S: 개인키→주소 재유도, 선언값과 대조 (C2)
    alt 불일치
        S-->>E: error (키 저장 안 함)
        E-->>CLI: 실행 중단
    end
    end

    loop 각 spec (직렬, 알고리즘 14)
        E->>E: testspec.Parse (알고리즘 1)
        alt 미적용 체인 / capability 부족
            E->>S: rec.Status(skip)
        else
            E->>E: fingerprint(spec, resolved)
            E->>S: Environment(fp)
            alt 재사용 없음
                E->>S: NewEnvironment(fp)
                E->>B: BuildEnv(ctx, env, spec)
                B->>B: place.Allocate (배치·포트·용량검증)
                B->>B: GenesisSource (알고리즘 4·5)
                B->>B: AssemblePlan
                B->>B: provision (알고리즘 6, upload-if-absent)
                B->>SUP: BringUp(plan, opts)
                SUP->>N: init --datadir · launch (알고리즘 7·8)
                SUP->>N: LeaderGate → HealthGate (알고리즘 9)
                N-->>SUP: head 전진
                SUP-->>B: NodeSet + procs
                B-->>E: NodeSet, Teardown
                E->>S: PopulateNodeTable · Save
                E->>E: collector.Start (라이브 수집)
            end
            E->>I: RunSpec(spec, env, rec)
            I->>I: preActions (알고리즘 10)
            I->>N: steps — tx / wait / fault (알고리즘 11)
            I->>N: assertions — rpc / log / metric
            I->>I: postActions (알고리즘 12)
            I-->>E: TestStatus
            E->>S: rec.Status (알고리즘 13)
        end
    end

    E->>S: Save() — session.json
    E->>SUP: Teardown (알고리즘 15)
    SUP->>N: SIGTERM → grace → SIGKILL → 검증
    SUP-->>E: 고아 0
    E-->>CLI: sessionRoot
    CLI-->>U: 표 / --json + exit code (0 pass · 1 fail · 2 blocked)
```

**설계상 중요한 순서 2가지**

- 키 자료 materialize 가 **세션 생성보다 먼저**다. 생성이 실패할 수 있는데(bootnode 부재), 시작하지 못할
  실행을 위해 세션 트리를 만들 이유가 없다.
- 신원 등록이 **네트워크 기동보다 먼저**다. 등록신원과 실제 키의 불일치는 기동 후에는
  "설명되지 않는 합의 정지"로만 드러난다.

---

## 2. 환경 구성 상세 — BuildEnv

알고리즘 4~9.

```mermaid
sequenceDiagram
    autonumber
    participant E as engine
    participant P as place.Allocator
    participant G as GenesisSource
    participant PR as provision.Provisioner
    participant FS as FileSink (Local | Remote)
    participant SUP as supervisor
    participant D as driver (Local | Remote)

    E->>P: Allocate(reqs, mode, capacity)
    P->>P: min(BFT≥4) · max(호스트×슬롯 / 포트대역) 사전검증
    alt 용량 초과/미달
        P-->>E: error (원격 자원 낭비 전에 실패)
    end
    P-->>E: []NodePlacement{host, ports, dataPath}

    E->>G: Genesis(plugin, validators)
    G->>G: preset metadata → validators · BLS · extraData · alloc
    G->>G: Family.BuildGenesis(template, params)
    G-->>E: genesis bytes

    E->>E: AssemblePlan(plugin, placements, genesis, dataRoot, caps)

    E->>PR: Provision(genesis, per-node config)
    loop 각 파일
        PR->>FS: Exists(path)?
        alt 이미 있음
            FS-->>PR: true → skip (알고리즘 6)
        else
            PR->>FS: Write / SFTP upload
        end
    end

    E->>SUP: BringUp(plan, Options{LeaderGate, AlignJoinGap, ForkSwaps})
    loop 각 노드
        SUP->>D: InitDatadir(spec, genesis)
        SUP->>D: Launch(spec) → PID
        SUP->>SUP: procman.Track{PID, DataDir, Host}
    end
    opt LeaderGate 요청됨
        SUP->>SUP: JoinGap(N) 에서 데드라인 파생
        SUP->>D: etcd 리더 준비 폴링
        note right of SUP: 요청했는데 미배선이면<br/>조용한 통과가 아니라 오류
    end
    SUP->>SUP: HealthGate — head 전진 폴링
    alt 실패
        SUP->>SUP: Classify(err) → FailureMode
        SUP->>SUP: 재시도 시 RemoveDataDir (stale etcd 정리)
        SUP-->>E: Diagnosis{Mode, Detail, ProducerLog}
    end
    SUP-->>E: NodeSet + Teardown
```

---

## 3. 테스트 해석 — Interpreter

알고리즘 10~13.

```mermaid
sequenceDiagram
    autonumber
    participant I as Interpreter
    participant R as Registry (액션11 · 어세션16)
    participant BIND as 값 바인딩
    participant N as 노드 RPC
    participant C as collector
    participant REC as TestRecord

    I->>I: preActions (idempotent 가드)
    alt preAction 실패
        I->>REC: Status(blocked)
        note right of I: 테스트 미수행 — 실패가 아니라 차단
    end

    loop 각 step
        I->>BIND: $ref / ${ref} 치환 (디스패치 직전, Spec 불변)
        alt 미바인딩 참조
            BIND-->>I: error (조용한 빈 문자열 금지)
        end
        I->>R: Action(name)
        R-->>I: Action
        I->>N: Do(ctx, ActionCtx)
        N-->>I: hash / receipt / value
        opt "save": name
            I->>BIND: bind(name, ActionCtx.Value ?? hash)
        end
        I->>REC: Step(i, StepResult{hash, receipt, 사유})
        alt receipt.status == 0x0 && !expectRevert
            I->>REC: 스텝 실패
        end
    end

    loop 각 assertion
        I->>BIND: $ref 치환
        I->>R: Assertion(name)
        alt source = rpc
            I->>N: eth_* / istanbul_* / wemix_*
        else source = log
            I->>C: WaitLog(node, pattern, timeout)
            C-->>I: LogMatch{file, lines, byteOffset, text}
        end
        I->>REC: Assert(AssertResult + provenance)
    end

    I->>I: postActions (판정과 독립)
    I->>REC: Status(assertions 에서 파생)
```

---

## 4. 원자 스텝 CLI — `chainbench net`

지시 3 의 조합 가능한 표면. 각 스텝은 워크스페이스를 읽고 자기 단계를 수행한 뒤 되쓴다.

```mermaid
sequenceDiagram
    autonumber
    actor U as 사용자 / LLM
    participant CLI as chainbench net STEP
    participant W as Workspace (→session, T7.7)
    participant CORE as core/*
    participant T as Target (local | SSH)

    U->>CLI: net new --chain --target
    CLI->>W: Open / 생성 · TargetSpec 해석
    W-->>U: 워크스페이스 경로

    U->>CLI: net keys --source random|preset
    CLI->>W: 상태 로드
    CLI->>CORE: KeySource.Ensure
    CLI->>W: markStep("keys") · Save
    W-->>U: 주소 · 검증자셋 출력

    U->>CLI: net allocate --mode os
    CLI->>CORE: place.Allocate (+용량검증)
    CLI->>W: 포트맵 저장
    W-->>U: 포트맵 출력

    U->>CLI: net genesis --mode template --set --overlay
    CLI->>CORE: genesis.Build
    W-->>U: genesis 해시 · validator 치환 결과

    U->>CLI: net launchopts
    CLI->>CORE: launchopt.Builder.Build (T7.3)
    CLI-->>U: 조립된 argv 출력 (실행 없이 확인)

    U->>CLI: net provision && net init && net start
    CLI->>T: 물질화 (upload-if-absent) · init · launch
    CLI->>W: PID 기록
    W-->>U: PID · head

    U->>CLI: net run case.json
    CLI->>CORE: engine.RunSpec
    W-->>U: session 판정

    U->>CLI: net stop --grace --rm-data
    CLI->>T: SIGTERM → grace → SIGKILL → 검증
    CLI-->>U: 고아 0 확인
```

**규율**: 각 스텝은 idempotent(재실행 안전)이고, 선행조건 미충족 시 조용히 성공하지 않고
"무엇을 먼저 실행하라"를 말한다. CLI 스텝 1개 = MCP 도구 1개 = `app` 유스케이스 1개.

---

## 5. 원격 실행 — stateless SSH

```mermaid
sequenceDiagram
    autonumber
    participant UP as Provisioner / Supervisor / Collector
    participant D as driver.RemoteDriver
    participant CFG as server-set.yaml
    participant H as 원격 호스트

    note over UP: 상위는 local/remote 를 모른다
    D->>CFG: 자격증명 로드 (gitignore · spec 에 없음)
    CFG-->>D: user · sshPort · keyPath|password · hosts
    D->>H: SSH 연결 (key 우선, 0600 강제)

    UP->>D: FileSink.Exists(path)
    D->>H: test -f
    alt 있음
        H-->>D: 0 → 재사용
    else 없음
        UP->>D: ProvisionFile(base64)
        D->>H: SFTP write (키 0600)
    end

    UP->>D: InitDatadir
    D->>H: 노드바이너리 init --datadir DIR genesis.json

    UP->>D: Launch(spec)
    D->>H: nohup 노드바이너리 ARGS 백그라운드 실행 후 PID 출력
    H-->>D: PID
    D-->>UP: handle{PID, Host}

    UP->>D: LogReader.ReadFrom(path, offset)
    D->>H: tail -c +N (1-based 바이트 오프셋)
    note right of D: tail -n 은 줄 단위라<br/>정확한 바이트 위치를 잃는다

    UP->>D: Stop(PID)
    D->>H: kill PID
    D->>H: kill -0 PID (종료 검증)
```

---

## 6. 실패 경로 — 진단과 분류

```mermaid
sequenceDiagram
    autonumber
    participant E as engine
    participant SUP as supervisor
    participant PM as procman
    participant REC as session

    E->>SUP: BringUp
    SUP->>SUP: launch / gate 실패
    SUP->>SUP: Classify(err)
    alt etcd 조인/stale
        SUP->>PM: Teardown{RemoveDataDir: true}
        SUP->>SUP: 백오프 후 재시도
    else quorum / fork / rpc
        SUP->>SUP: FailureMode 기록
    else 매칭 없음
        SUP->>SUP: UnknownFailure
        note right of SUP: 그럴듯한 오분류는<br/>읽는 사람을 엉뚱한 로그로 보낸다
    end
    SUP-->>E: Diagnosis{OK:false, Mode, Detail, ProducerLog}
    E->>REC: rec.Status(blocked) + 사유 기록
    E->>E: 다음 spec 으로 계속 (직렬)
    note over E: blocked 는 fail 보다 심각 → exit code 2
```
