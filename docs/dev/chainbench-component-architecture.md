# chainbench 컴포넌트 아키텍처 — High/Middle/Low 계층 · TDD 구현 플랜

> 지위: `chainbench-design.md`(인터페이스)·`chainbench-feature-spec.md`(F1~F16)·`chainbench-refactoring.md`(WP)를
> **컴포넌트 계층(역할 분리)과 구현 순서(TDD·조립)** 관점으로 재정리한 문서. 설계 결정은 앞 4문서와 동일하며,
> 여기서는 "무엇을 어떤 순서로, 어떻게 원자적으로 만들어 안전하게 합칠까"를 확정한다.
> 목적(요구 37): 여러 노드를 서로 다른 역할로 실행해 체인 네트워크를 구성하고 tx/블록을 처리하며 테스트한다.

---

## 0. 사용자 접근에 대한 비판적 검토 (가능성·리스크)

**결론: 접근은 타당하고, 상당 부분이 이미 코드에 구현되어 있어 실현 가능성이 높다.** 다만 **통합 전략 1가지는 수정**을 권한다.

### 0-1. 강하게 동의 (코드리뷰로 확인된 근거)
> **주의:** 아래 "이미 존재"는 **패키지 이름이 아니라 실제 구현을 읽어 확인**한 것이다. 반대로, 이름만으로 "있다"고 보기 쉬운 △/✗ 항목은 §2b에 분리했다.
- **"철학·플로우는 체인 무관, 세부만 체인별"** = 현재 `registry.ChainPlugin`/`ConsensusFamily` 플러그인 패턴 그대로(3체인 실측 등록·`all_test.go`). 정석.
- **"저수준은 atomic + TDD"** — **대부분 실측 확인**: config·genesis(4모드)·portplan·topology·nodeconfig·keys·rpc·driver·remote·accounts 구현+테스트. 단 `procman`은 **로직만 있고 프로덕션 미배선**(§2b) — "테스트 있음 = 동작 배선됨"이 아님.
- **"local/remote를 상위는 일관, 하위에서 상황처리"** — **실측 견고**: `driver.Driver` + `Initializer`/`FileProvisioner` type-assert, 로컬·원격 양쪽 impl·CLI 배선 확인. 이 seam이 사용자 주장의 핵심 근거다.
- **"local도 루프백 ip+port를 가지므로 동일 플로우 가능"** — 정확. **Transport seam**으로 형식화(§4).
- **"remote 제어에 중간 주체가 꼭 필요한가"** — **불필요**가 코드로 증명됨: `remote.go`가 **에이전트 없이 stateless SSH**(`nohup … & echo $!` → PID, `kill <PID>`)로 실행(✓)·정지(**driver 수준 ✓, 단 CLI 미연결 △**, §2b). 별도 데몬 없이 동작(§4-2).
- **단, 관측·정리 계층은 골격만**: 원격 로그수집·bp참여/분기 관측·procman 배선·체인무관 place·용량검증은 **아직 없음**(§2b). 사용자 접근은 유효하나, 이들을 "이미 있음"으로 착각하면 안 된다.

### 0-2. 수정 권고 (비판적 지점)
1. **[중대] "컴포넌트를 각각 완성 후 최종적으로 합친다"(big-bang 통합)는 위험.** 통합 시점에만 드러나는 버그(etcd 타이밍·포트 충돌·handoff 합의전이·genesis 등록 불일치)가 **마지막에 몰린다**. → **walking skeleton + 수직 슬라이스**로 전환: 원자 모듈이 갖춰지는 즉시 **가장 얇은 end-to-end 1개(1체인·local·4노드·tx 1건)** 를 세워 통합을 **앞으로 당기고**, 이후 remote·업그레이드·attach·체인 추가를 **얇은 슬라이스로 계속 붙인다**(§5). 저수준 TDD와 상위 통합-조기화는 병행 가능하며 상충하지 않는다.
2. **[중] 과분해 경계.** 역할 분리는 좋으나 컴포넌트가 지나치게 잘게 쪼개지면 추상화 비용·간접호출이 버그를 부른다. Middle을 **9개 수준**으로 유지(§3), 추상화는 값을 낼 때만 도입.
3. **[중] 용량·배치를 1급 제약으로.** "BFT 블록생성 최소 4노드 / 서버당 3~4노드 / 서버수×가용포트 = 최대치"는 **place가 사전 검증(fail-fast)** 해야 한다(요구: min 4, max 산정). 늦게 터지면 원격 자원 낭비.
4. **[중] attach(rpc-url only) 모드의 관측 강등.** 이미 존재하는 체인을 rpc-url만으로 쓰면 **로그 수집 불가 → collector는 RPC-only로 우아하게 강등**하고 디버깅 충실도 한계를 문서화(§3-M7).
5. **[경] 원격 프로세스 신뢰성.** stateless SSH는 재접속 시 PID 추적이 끊길 수 있음 → **PID+datadir을 session에 영속**(procman 확장)하고, 정지는 `kill $(pidfile)`·정리는 datadir 삭제(별개 기능, design §3.3 S2)로.

**가능성 판정: 실현 가능(High confidence).** 저수준·transport는 이미 존재·검증됨. 신규는 상위 조립(session/engine/testspec/collector)과 몇몇 원자 모듈뿐. 관건은 **조립 순서(TDD + 조기 수직통합)** 이며 본 문서가 그것을 확정한다.

---

## 1. 조직 원리 — 2개의 축

**1차축 = 바운디드 컨텍스트(DDD, §1b·확정)** · **2차축 = 레이어(High/Middle/Low, 아래) — 컨텍스트 내부 구현/조립 순서용.**
초안은 레이어를 1차로 뒀으나(레이어-우선 = 비-DDD), DDD 검토로 **컨텍스트를 1차, 레이어를 2차**로 확정한다.

| 계층(2차축) | 성격 | 검증 방식 |
|------|------|-----------|
| **High** | 추상화·오케스트레이션·변화점 격리 | end-to-end(수직 슬라이스) 라이브 |
| **Middle** | 역할 1개를 완결 담당, 저수준을 조립 | 통합 테스트(실 노드/임시 FS) |
| **Low** | atomic·결정적(가능한 순수함수/단일 syscall) | 단위 테스트(table-driven), 먼저 통과 |

원칙: **의존은 High→Middle→Low 단방향** + **컨텍스트 간은 Context Map(§1b)대로**, 체인별 세부는 **ACL(플러그인)로만** 상위에 들어온다(상위 코드는 체인을 모른다). 소유권 단일화 = **Aggregate Root 규율**(§1b).

---

## 1b. DDD 컨텍스트 맵 (1차 분해 · 확정)

> chainbench는 테스트 하네스이므로 **전략 DDD**(바운디드 컨텍스트·ACL·Core/Supporting·핵심 Aggregate 불변식)만 채택하고, **전술 DDD 풀세트**(도메인 이벤트 전면·CQRS·리포지토리 남발)는 과잉이라 배제한다. 상태=코드 실측(§2b).

| # | 바운디드 컨텍스트 | 분류 | Aggregate(root) · VO | 핵심 불변식 | 구성 컴포넌트(레이어) | 상태 |
|---|-------------------|------|----------------------|-------------|------------------|------|
| C1 | **테스트 오케스트레이션** | **Core** | `TestRun` | 스텝 원자성(부분성공 없음)·status=assertion 파생·preAction 실패⇒BLOCKED | Interpreter(M8)·testspec·assert | ✗ NEW |
| C2 | **네트워크 구성** | **Core** | `Environment` · VO: Fingerprint/Ports/Placement | 포트 무충돌·fingerprint 불변·**genesis 등록신원=실제 키 일치**·BFT min≥4 | Place(M2)·KeyReg(M3)·Genesis(M4)·Provision(M5) | △ |
| C3 | **노드 생명주기** | Supporting | `NodeProcess` | **고아 0**·정지⇒내장 etcd 종료·datadir 삭제=별개 연산 | Supervisor(M6)·procman | △ |
| C4 | **관측·진단** | Supporting | (읽기 모델) | — | Collector(M7)·verify·logs | △ |
| C5 | **세션·아티팩트** | Supporting | `Session`(root) | 경로 단일소유·결정적 레이아웃·env 재사용=fingerprint | Session(M1) | ✗ NEW |
| C6 | **Chain Adapter** | Generic-external · **ACL** | — | **외부 바이너리 quirk를 Core로 누출 금지** | ChainPlugin+Capabilities(H2) | ✓/NEW |
| C7 | **Transport** | Generic-infra · Ports&Adapters | — | 상위는 local/remote 무관 | Driver local/remote(M9) | ✓ |
| C8 | **공유 커널** | Generic | VO 팩토리 | 값 불변 | config·rpc·accounts·node·portplan·nodeconfig | ✓ |

**Context Map(관계):**
- **Engine**(App Service, 도메인 로직 없음) → C1·C2·C3 구동(Customer/Supplier).
- **C2 네트워크 구성** → **C6 ACL**로 genesis 템플릿·NodeComposition(etcd/ncp/staking)·시스템컨트랙트·BLS 획득 + **C7 Transport**로 provision·파일 ship + **C8**(portplan/keys).
- **C3 노드 생명주기** → **C6 ACL**(체인별 launch flags) + **C7 Transport**(exec/kill).
- **C4 관측** → **C7**(로그) + **C8**(RPC).
- **C1~C5** → **C5 Session**에 영속(참조 무결성).
- **C6 ACL** → **외부 시스템**(gwemix·go-wbft·go-stablenet·bootnode·remote servers): 모델 번역·quirk(셸아웃·`deploy-governance` 2-arg 버그·stdout 파싱) 격리.

**DDD가 이 설계에 더하는 4가지(확정):**
1. **Core 집중**: C1·C2가 chainbench의 차별점 → 최대 투자. C7·C8(Generic)은 최소·재사용, C6은 격리.
2. **불변식의 Aggregate 귀속**: 예) "고아 0"은 `NodeProcess`(C3) 불변식 → §2b의 procman 미배선 = "C3가 불변식을 강제 못함"으로 재정의(배선 우선순위 상승).
3. **ACL 승격**: 외부 바이너리의 혼란을 C6 안에 가둬 Core 오염 차단(최대 리스크 감소).
4. **유비쿼터스 언어**: Core(C1·C2)는 도메인어(spec/step/assertion, node/validator·bp/genesis/fork/staker), Generic(C7 Transport 등)은 기술어 허용.

> 시각화: `chainbench-ddd.html`(컨텍스트 맵). 레이어 뷰(`chainbench-components.html`)는 **구현 순서용 2차 뷰**로 병존.

---

## 2. "복잡한 경우의 수" ↔ 담당 컴포넌트 (책임 귀속표)

사용자가 나열한 변주를 **각 컴포넌트가 완결 흡수**해 상위 플로우를 일관되게 유지한다.

| 경우의 수(변주) | 담당(Middle) | 처리 방식 |
|-----------------|--------------|-----------|
| 신규 구축 vs 기존(rpc-url only) | Engine + Collector(M7) | 신규=전체 플로우 / attach=Transport·Provision·Supervisor 생략, Collector는 RPC-only 강등 |
| local vs remote | Transport(M9) | 동일 인터페이스, 하위에서 local exec / ssh exec 분기(type-assert) |
| 바이너리 변경(wemix/wbft/stablenet) | GenesisBuilder(M4)+플러그인(H2) | 바이너리·genesis 템플릿·시스템컨트랙트를 플러그인이 제공 |
| 업그레이드(멀티 바이너리) | Supervisor(M6) | 노드별 바이너리 집합(handoff, type-1) / fork 전 교체(type-2) |
| 노드 수(min4·max=서버×포트) | Place(M2) | 배치·포트 산정 + **용량 사전검증(fail-fast)** |
| 키/계정/노드키(랜덤·기존·원격) | KeyRegistry(M3) | 랜덤생성/기존로드/원격다운로드+업로드, 노드별, genesis 등록 입력 |
| 동일머신(동IP·異port) vs 멀티머신(異IP·동port) | Place(M2)+Transport(M9) | 배치 모드로 결정, IP/port는 **gitignore된 설정파일**(§4-3) |
| config 파일 vs flag(우선순위) | config(Low) | flag>config>default (design [7,8], 이미 구현) |
| 기존 genesis vs 생성/템플릿/툴 | GenesisBuilder(M4) | 4모드(design §3.8), 등록계정·노드정보 정합 강제 |
| 배포 컨트랙트 vs 시스템 컨트랙트 | GenesisBuilder(M4)+플러그인(H2) | 시스템=genesis 등록, 배포=배포 후 등록 스텝(bp 등록 필수) |
| local 배포 vs remote 배포 | Provisioner(M5)+Transport(M9) | datadir·키·genesis·config 물질화를 Transport로 통일(local FS / ssh ship) |
| local 실행 vs remote 실행 | Supervisor(M6)+Transport(M9) | 실행/정지/PID관리를 Transport로 통일(에이전트 없음, §4-2) |
| 결과수집·디버깅 | Collector(M7) | RPC 조회(✓) + 로그(로컬 스캔만, **원격 없음**), attach는 RPC-only |

---

## 2b. 코드 실측 검증 매트릭스 (에이전트 3종 코드리뷰 · 이름이 아니라 구현 확인)

> **원칙: 패키지·`_test.go` 존재 ≠ 기능 지원.** 아래는 실제 코드를 읽어 확인한 결과다.
> **✓ 구현확인 · ✓\* 조건부(외부도구/경로의존) · △ 부분·미배선 · ✗ 미구현(설계만).**

| 변주/기능 | 상태 | 근거(file:line) · 격차 |
|-----------|------|------------------------|
| local provision(datadir/config/genesis/키) | ✓ | `driver/local.go:36`, `setup/launch.go:69` |
| local launch + PID | ✓ | `local.go:54` `cmd.Process.Pid` |
| **local stop + 종료검증·leak report** | **△** | stop 경로는 `Kill()`만·검증 없음(`local.go:79`); `procman.StopAll` 검증로직은 **테스트에서만 사용**(프로덕션 미배선) |
| remote provision over SSH(FileProvisioner/Initializer) | ✓ | `remote.go:55,73,87`, assert `launch.go:177,190` |
| remote launch over SSH(nohup+echo $!) | ✓ | `remote.go:104` |
| remote stop by PID | ✓(driver) / **△ CLI미연결** | `remote.go:126`; `stop`/`clean`은 LocalDriver 하드코딩 |
| **"원격 파일 있으면 사용, 없으면 업로드" 조건분기** | **✗** | 조건분기 없음 — setup은 항상 업로드, wemix deploy는 항상 읽기 |
| **procman: {PID,datadir}·원격PID 추적** | **△** | 로컬 `{PID,Label}`만·datadir/원격PID 없음·미배선 |
| attach(rpc-url only) | ✓ | `pipeline/attach/attach.go:25`, `test.go:99` |
| 통합 Driver seam(local/remote 무관) | ✓ | `driver.go:42/58/67`, 양쪽 impl·type-assert |
| SSH 입력(host/port/user/pass) | ✓ / **key_file ✗** | flags+env·wemix `cluster.yaml`+gitignore `credentials`; **key_file 인증 미지원(password only)** |
| 랜덤 키 생성 / 주소파생 / 기존키 로드 | ✓ | `wallet.go:437,461`, `keys.go:77 LoadPreset` |
| BLS/PoP 생성 | ✓\* | **외부 `bootnode` 위임**·stdout 파싱, 바이너리 없으면 실패 |
| 원격 키 다운로드(SSH) | ✓ | `wemix/deploy/keys.go:61 ReadServerKeys` |
| 랜덤키 원격 업로드 | △ | setup `shipIdentities`(✓) / wemix deploy는 의도적 미업로드 |
| genesis 4모드(existing/build/template/upgrade-inherit) | ✓ | `genesis.go:38`, `config.go:70/97/18`, `upgrade/plan.go:138` |
| genesis 계정·신원 등록 + 초기 balance | ✓ | `wbft/genesis.go:53,93`, `deploy/accounts.go:110` |
| 시스템컨트랙트 embed / 배포컨트랙트 등록 | ✓ / ✓(셸아웃) | `wbft/genesis.go:41`; `poa/bootstrap_exec.go:31 DeployGovernance`(gwemix 버그 우회) |
| 포트: 동일머신 스텝(etcd=p2p+1) | ✓ | `portplan.go:27/45` |
| 포트: 멀티머신 동일포트·다른IP | ✓(wemix deploy) | `wemix/deploy/plan.go:42`; **`internal/core/place` 없음·poa 하드코딩** |
| **노드수 min≥4 / max(서버×포트) 검증** | **✗** | topology엔 없음; `wbftQuorumMin=4`는 upgrade handoff 한정 |
| config 우선순위 flag>file>default | ✓ | `config.go:64 Resolve=Merge(Defaults,file,override)` |
| 체인 플러그인 3종(wemix/wbft/stablenet) | ✓ | 3× `init()`+Register, `all_test.go` |
| 업그레이드 멀티바이너리 handoff | ✓ | `upgrade/plan.go:110`, `exec.go:65`; e2e |
| 결과수집 via RPC(height/sync/peers/validators/receipt) | ✓ | `rpc/client.go`, `verify.go:68` |
| **로그수집: 로컬** | **△** | `logs.go:46 Search`는 **파일 스캔**(live tail 아님) |
| **로그수집: 원격(SSH)** | **✗** | 없음 |
| **chainstate: bp참여·reorg/분기 검출** | **✗** | 미구현(height/sync/peers/validators만); `obs`는 인메모리 |
| MCP 도구 | ✓(30) | `mcp/tools.go:33` |

**결론(정직한 재평가):** 노드 구성·실행의 **핵심 골격은 실제로 견고**(provision/launch·local/remote seam·genesis 4모드·키·업그레이드·플러그인·RPC수집·attach 모두 ✓). 그러나 제가 초안에서 "✓"로 뭉뚱그린 것 중 **다음은 신규/보강 작업으로 반드시 계상**해야 한다:
1. **procman 프로덕션 배선 + 종료검증 + 원격 PID + datadir**(현재 stop은 검증 없는 Kill).
2. **체인무관 `place` 통합**(로컬 portplan + 원격 deploy를 하나로) + **용량검증(min≥4·max)**.
3. **"원격 파일 존재 시 재사용, 없으면 업로드" 조건분기**.
4. **원격 로그수집 + live tail**(현재 로컬 스캔만) + **bp참여·분기검출**.
5. **랜덤키 원격 업로드 경로 통일**(setup·deploy 이원화 해소), **key_file SSH 인증**, **BLS 외부 bootnode 의존 명시**.

이 5개가 "이름만 보고 있다고 착각하기 쉬운" 실제 공백이며, TDD 플랜(§5)에서 신규 phase로 다룬다.

---

## 3. 컴포넌트 카탈로그 (High / Middle / Low)

### High level (추상화·오케스트레이션)
- **H1. Engine** — 체인 무관 오케스트레이터. 균일 플로우: `spec.Parse → config.Resolve → Fingerprint → env 재사용 or (Place→KeyReg→Genesis→Provision→Supervise→Collect) → Interpreter.Run → 결과기록 → Teardown`. 체인별 분기 없음.
- **H2. ChainPlugin + Capabilities** — **유일한 변화점**. 체인별: genesis 템플릿, NodeComposition(etcd/ncp/staking 초기화 훅), 시스템컨트랙트 등록, SupportedForks/Assertions(design §3.7). 상위는 이 인터페이스로만 체인차를 접한다.
- **H3. Surfaces** — CLI(`cmd/chainbench`)·MCP·dashboard. 엔진 호출 + 세션 아티팩트 소비의 **얇은 어댑터**(로직 없음).

### Middle level (역할 완결 컴포넌트) — [ ]안은 실측 격차(§2b)
- **M1. Session**[NEW] — `.chainbench/<session>/` 정본·경로 파생·env 재사용·기록. (design §3.1) · **미존재**
- **M2. Place**[NEW·CONSOLIDATE] — host/port/배치. **실측: `internal/core/place` 없음.** 로컬=`portplan`(스텝, ✓), 원격 동일포트/다른IP=`wemix/deploy`(✓, **단 poa 하드코딩·체인특화**). → **두 구현을 체인무관 core로 통합** + **용량 검증(min≥4·max=서버×포트)은 현재 없음 → 신규**. (design §3.4)
- **M3. KeyRegistry**[NEW·CONSOLIDATE keys+deploy/keys] — 노드별 키(랜덤 ✓/기존 ✓/원격다운로드 ✓). **BLS/PoP는 외부 `bootnode` 위임**(✓\*). **랜덤키 원격 업로드는 경로의존(setup ✓ / wemix deploy는 미업로드) → 통일 필요.** (design §3.5)
- **M4. GenesisBuilder**[EXTEND genesis] — 4모드(✓) + 등록계정·노드정합(✓) + 시스템컨트랙트 embed(✓). **배포컨트랙트 등록은 `gwemix deploy-governance` 셸아웃**(체인특화 → 플러그인 뒤로). (design §3.8)
- **M5. Provisioner**[REFACTOR setup] — datadir+키+genesis+config **물질화**를 Transport로 통일(로컬 ✓/원격 ✓). **격차: "원격에 파일 있으면 사용, 없으면 업로드"의 조건분기 없음**(현재 항상-업로드 or 항상-읽기) → **신규**.
- **M6. Supervisor**[NEW] — N노드 기동(✓)·헬스게이트(블록생성/etcd 리더 ✓ 일부)·teardown·프로세스 제어. **격차: `procman` 종료검증이 프로덕션 미배선(stop은 Kill만)·로컬PID만·원격 stop CLI 미연결 → 배선+검증+원격PID+datadir 필요.** (design §3.3)
- **M7. Collector**[NEW·probe+obs] — RPC 수집(height/sync/peers/validators ✓). **격차: 원격 로그수집 없음(로컬도 스캔이지 tail 아님)·bp참여/reorg 관측 없음·`obs`는 인메모리 스캐폴딩** → **원격 tail·bp참여·분기검출 신규**. attach=RPC-only 강등. (design §3.6)
- **M8. Interpreter**[REPLACE testkit] — pre→steps→assert→post, atomic 스텝, provenance. **미존재**(현재 Go-func 테스트). (design §3.2)
- **M9. Transport**[KEEP·EXTEND driver] — **통합 seam 실측 견고**: 단일 `Driver`(Provision/Launch/Stop) + `Initializer`/`FileProvisioner` type-assert, 로컬·원격 양쪽 구현·CLI 배선(✓). local(loopback)·remote(ssh nohup+PID). **stateless(에이전트 없음, 검증됨)**. 격차: **stop 종료검증·key_file 인증 미지원**.

### Low level (atomic·TDD 먼저)
> 상태(코드 실측, §2b): **✓ 구현확인** · **✓\* 구현되나 조건부**(외부도구/경로의존) · **△ 부분/미배선** · **NEW 신규**
| 모듈 | 상태 | 책임(atomic) · 실측 주석 |
|------|------|--------------|
| config resolve/merge/flatten | ✓ | 우선순위 병합(flag>config>default) — `config.go:64` |
| genesis build/merge/override/extract | ✓ | 순수 바이트 변환(원본 불변) — 4모드 전부 실측 |
| portplan alloc | ✓ | 결정적 포트(etcd=p2p+1) — `portplan.go:27`. **로컬 스텝 전용**(원격 동일포트는 wemix/deploy 별도) |
| topology parse/validate | ✓ | 역할·sync·개수 검증(roleAliases bp/en). **min≥4·max 용량검증 없음** |
| nodeconfig generate | ✓ | TOML 생성 |
| keygen(keypair/addr) | ✓ | 랜덤(crypto/rand)·주소파생 |
| keygen(BLS/PoP) | ✓\* | **외부 `bootnode` 바이너리에 위임**(stdout regex 파싱; 바이너리 없으면 실패). 네이티브 Go BLS 없음 |
| accounts sign/derive | ✓ | tx 서명·주소 — `wallet.go:437,461` |
| rpc call | ✓ | 단건 JSON-RPC — `rpc/client.go` |
| transport prim(local exec/file/kill) | ✓ | 단일 실행/복사/시그널. **stop 후 종료검증 없음**(Kill만) |
| transport prim(ssh exec/sftp/kill) | ✓ | nohup+PID·kill·SFTP(stateless) — `remote.go:104`. **key_file 인증 미지원(password only)** |
| procman(PID track, stop-verify) | △ | 종료검증·leak-report **로직은 있으나 프로덕션 미배선(테스트 전용)**·**로컬 PID만**(datadir·원격PID 없음) → **배선+datadir+원격PID 필요** |
| **testspec parse/validate/fingerprint** | **NEW** | 순수 파싱·6요소 해시 |
| **assert funcs(typed)** | **NEW** | wei/addr/hex/bool 비교 |
| **session path derive** | **NEW** | 결정적 경로·env-id 축약 |
| log tail prim | △→NEW | **현재는 로컬 파일 스캔**(`logs.Search`, `logs.go:46` — 글롭+파싱, **live tail 아님**). **원격 로그수집 없음**. → live-tail(로컬/원격 SSH) 신규 |

> 관찰: 신규/보강 원자 모듈 = testspec·assert·session-path(순수 신규) + **log tail**(스캔→live-tail·원격 확장) + **procman 배선**(△). 나머지는 EXTEND. 저수준 TDD 부담은 작지만, **"이미 있다"고 착각하기 쉬운 △항목(procman 배선·원격 로그·용량검증·upload-if-absent)** 을 신규 작업으로 반드시 계상해야 한다(§2b).

---

## 4. Local/Remote 통합 (Transport seam) — 핵심 결정

### 4-1. 통합 인터페이스
상위(Provisioner/Supervisor/Collector)는 **local/remote를 모른다.** 하나의 seam을 통해 동작:
```go
// (기존 driver.Driver + capability를 Transport로 명명·형식화)
type Transport interface {
    Exec(ctx, cmd) (stdout string, pid int, err error) // nohup 실행 시 pid 반환
    PutFile(ctx, path, content, mode) error            // local write / remote SFTP(ship)
    GetFile(ctx, path) ([]byte, error)                 // 기존 파일 읽기(원격 다운로드 포함)
    Kill(ctx, pid) error                               // local signal / ssh kill
    TailLog(ctx, path) (<-chan string, error)          // local tail / ssh tail
    Endpoint(port int) string                          // local=127.0.0.1:port / remote=serverIP:port
}
```
- **local = 루프백 IP + 스텝 포트**, **remote = 서버 IP + 동일 포트** → `Endpoint()`가 흡수. 상위 플로우 동일.
- 구현체: `LocalTransport`(exec.Command/os/signal), `RemoteTransport`(SSH/SFTP). 선택은 배치 모드(place)로 결정.

### 4-2. 원격 제어 = stateless SSH (에이전트 없음) — 권고
- **근거**: `remote.go`가 이미 `nohup <cmd> & echo $!`로 PID 확보 + `kill <PID>` over SSH로 동작. 별도 데몬 배포·유지 불필요 → 운영 복잡도·장애면 최소.
- **신뢰성 보강**: PID+datadir을 **session에 영속**(재접속·재시작에도 제어 가능). 정지=`kill`, 정리=datadir 삭제(별개). 프로세스 확인=`kill -0`/`ps`.
- **에이전트 도입은 보류**: 서브초 스트리밍 제어·대량 팬아웃 지연이 문제될 때만 재검토(현재 요구엔 불필요). SSH 왕복 지연은 노드 수 규모(수십)에서 허용.

### 4-3. 민감정보(IP/port/SSH) 관리
- 사용할 **서버 IP 목록·포트·SSH(id/pw/port)** 는 **정의서에 넣지 않고** `remote-server-config.yaml`(런타임 로드)로 관리. **repository 미저장(gitignore)**, `remote-server-config.sample.yaml`만 추적(design §7·L6b, 이미 반영·gitignore 처리 완료).

---

## 5. TDD 구현 플랜 (원자 검증 → 조기 수직통합 → 확장)

> 철학: **저수준은 TDD로 원자 검증**, **통합은 뒤로 미루지 않고 walking skeleton으로 앞당김**, 이후 **수직 슬라이스**로 확장. big-bang 통합 배제(§0-2-1).

> **실측 5대 공백(§2b) → phase 배치**(이름만으로 "있다" 착각 금지): ① procman 배선+검증+원격PID+datadir → **P1·P2·P3**. ② 체인무관 place 통합 + 용량검증(min≥4·max) → **P1**. ③ 원격 파일 upload-if-absent 조건분기 → **P2·P3(Provisioner)**. ④ 원격 로그수집·live tail·bp참여·분기검출 → **P3(Collector)**. ⑤ 랜덤키 원격 업로드 통일·key_file 인증·BLS 외부의존 명시 → **P1·P2**.

### Phase 0 — 인터페이스 동결
Transport·Place.Allocator·KeyRegistry·GenesisBuilder·Provisioner·Supervisor·Collector·Session·Interpreter·Capabilities 확정(design §3과 1:1). 게이트: 컴파일되는 인터페이스 + feature-spec AC 매핑.

### Phase 1 — Low(atomic) TDD
- **NEW 순수 3종**(testspec parse+fingerprint / assert funcs / session-path): table-driven 단위테스트 **먼저**(RED→GREEN).
- **place 통합(NEW)**: portplan(로컬 스텝) + wemix/deploy(원격 동일포트)를 **체인무관 Allocator**로 병합 + **용량검증(min≥4·max=서버×포트) 신규**. 단위테스트.
- **procman EXTEND**: `{PID,datadir}` + 원격 PID + `Alive` 검증 — 단, **핵심은 프로덕션 stop 경로에 배선**(현재 미배선, §2b). 
- **keyreg**: 랜덤/기존/원격다운로드 통합, **BLS는 외부 bootnode 위임을 인터페이스로 캡슐화**(부재 시 명확 오류).
- 게이트: 단위 100% + 동시 모듈 `-race` 클린.

### Phase 2 — Transport TDD
- Local/Remote Transport를 driver 위에 형식화. remote: PID 반환·`kill -0` **종료검증**(현재 Kill만) + **key_file 인증** 추가(현재 password only).
- **upload-if-absent 조건분기 신규**: `test -f` 존재확인 → 있으면 재사용/없으면 업로드(현재 항상-업로드 or 항상-읽기, §2b).
- 게이트: 단위(모의) + **통합 1건**(로컬 더미 프로세스 기동/**검증 종료**·0-고아; ssh는 테스트호스트 or mock).

### Phase 3 — Middle 통합 TDD
- **Provisioner**: datadir+키+genesis+config 물질화(local·remote 동일 경로) + upload-if-absent 반영.
- **Supervisor**: **실제 4노드 로컬 wbft** 기동 → 헬스게이트(블록생성·etcd 리더) → teardown 후 **고아 0 + datadir 삭제**. **procman이 실제 stop 경로에 배선됐는지**를 leak=0으로 검증.
- **Collector**: 로컬 **live tail**(스캔→tail 승격, 누락 0) + **원격 SSH tail 신규** + RPC 스냅샷 + **bp참여·분기(reorg)검출 신규**. attach=RPC-only 경로.
- 게이트: 각 컴포넌트 통합테스트 라이브 통과.

### Phase 4 — **Walking Skeleton (조기 수직통합)** ★
가장 얇은 end-to-end: **Engine이 Place→KeyReg→Genesis→Provision→Supervise→Collect→Interpreter를 조립**해 **1체인(wbft)·local·4노드·tx 1건 spec**을 수행 → 세션 아티팩트 생성·검증. **여기서 통합 리스크(포트·etcd·genesis정합)를 조기 노출·해소**. 라이브 검증.

### Phase 5 — 수직 슬라이스로 확장
각 슬라이스 = 얇은 end-to-end 추가, **매번 통합 유지**:
1. **remote**(Transport 교체만) — 동일 spec이 원격에서 동작.
2. **업그레이드 멀티바이너리**(go-wemix+go-wbft handoff, Supervisor type-1).
3. **attach 모드**(rpc-url only, Collector RPC-only).
4. **체인 추가**(stablenet) — **C6 ACL(플러그인)만 추가, Core(C1·C2) 무변경으로 검증** → ACL 격리·추상화 증명.
5. **wemix4 스위트 이관** — DSL 정의서로 케이스 포팅(기존 e2e 병존).

### Phase 6 — 표면·마감
MCP/dashboard가 세션 아티팩트 소비(F14·F15), CLI 정리. 성능·백프레셔(O7)·`-race` 게이트(O6) 최종 확인.

### 5-x. 기존 WP(refactoring §7.1)와의 관계
본 플랜은 WP를 대체하지 않고 **조립 순서를 부여**한다: WP1(session/M1)·WP2(place/M2)·WP3(testspec/M8+Low)·WP4(supervisor/M6)·WP5(keyreg/M3+collector/M7)·WP6(capabilities/H2+표면/H3). **Phase 4 walking skeleton은 WP1~5를 관통하는 얇은 수직통합**으로, 수평 WP에 없던 "조기 통합" 축을 추가한다.

---

## 6. 리스크 · 완화

| 리스크 | 완화 |
|--------|------|
| **"이름만 존재"를 구현으로 오인**(procman 미배선·원격로그·place·용량검증·upload-if-absent) | **§2b 실측 매트릭스를 착수 전 기준으로 사용**; △/✗ 항목은 신규 작업으로 계상(§5 phase 배치) |
| **테스트 존재 = 배선됨 오인** | procman처럼 로직만 있고 미배선일 수 있음 → 통합테스트(Phase 3)에서 실제 stop 경로 leak=0 검증 |
| big-bang 통합에 버그 집중 | walking skeleton(Phase 4) + 수직 슬라이스로 통합 상시화 |
| 과분해로 추상화 비용↑ | Middle 9개로 제한, 추상화는 값 낼 때만 |
| BLS/원격배포의 외부바이너리(bootnode/gwemix) 의존 | 플러그인 뒤로 캡슐화·부재 시 명확 오류; 버그 우회(2-arg deploy-governance)를 문서화 |
| 원격 PID 추적 유실 | PID+datadir session 영속, kill/`kill -0`/datadir삭제 분리 |
| 용량 초과(서버×포트) 늦은 발견 | Place 사전 검증(min4·max, fail-fast) |
| attach 디버깅 충실도 한계 | Collector RPC-only 강등 명시, 한계 문서화 |
| 체인 추가 시 상위 오염 | 변화점을 Capabilities 뒤로만, Phase 5-4에서 엔진 무변경 검증 |

---

## 7. 산출물
- 본 문서(`chainbench-component-architecture.md`) = **DDD 컨텍스트 맵(1차, §1b)** + 레이어/컴포넌트(2차, §3) + 코드 실측(§2b) + TDD 조립 순서(§5).
- 다이어그램: `chainbench-ddd.html`(컨텍스트 맵 — Core/Supporting/Generic/ACL·Aggregate) · `chainbench-components.html`(레이어 뷰 — 구현 순서).
- 앞 4문서와 정합: 요구/결정(requirements-review) · 인터페이스/데이터(design) · 동작계약 F1~F16(feature-spec) · 코드매핑/WP(refactoring).
- **구현 착수**: Phase 0(인터페이스 동결) → Phase 1(Low TDD) → … → Phase 4(walking skeleton = Core C1·C2·C3 조립 라이브) 게이트 통과 시 확장.
