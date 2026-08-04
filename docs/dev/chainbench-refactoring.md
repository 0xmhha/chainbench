# chainbench 리팩토링 감사 — 기존 코드 → 목표 설계 매핑

> 목적: 목표 설계(`chainbench-design.md` 인터페이스·데이터모델, `chainbench-feature-spec.md` F1~F15 동작계약)를 향해, **기존 코드를 무엇을 유지/확장/재구성/교체하고 무엇을 신설**할지 코드 감사 기반으로 확정하고, **구현-ready 작업 단위(WP)** 로 분해한다.
> 근거: `chainbench-requirements-review.md`(요구/결정) · `chainbench-design.md`(§3 인터페이스·§4 스키마) · `chainbench-feature-spec.md`(F1~F15 AC).
> 원칙: (1) 견고한 기반 재사용, (2) 인터페이스 우선·하위호환, (3) 점진 이관(기존 pipeline/e2e 공존), (4) 소유권 단일화.
> 규모: 비-테스트 Go **약 13,478 LOC** (pkg + cmd). 인용한 기존 API는 모두 파일:라인 실존 확인(design §3 검증분).

---

## 1. 판정 범례
- **KEEP** 그대로 재사용 · **EXTEND** 계약에 추가 · **REFACTOR** 내부 재구성(동시성/세션인지) · **REPLACE** 재작성(구모델 폐기) · **NEW** 신설 · **CONSOLIDATE** 흩어진 것 통합

---

## 2. 재사용 가능한 견고한 기반 (KEEP / EXTEND)

| 패키지 (LOC) | 현재 역할 | 판정 | 근거·조치 |
|---|---|---|---|
| `core/registry` (284) `ChainPlugin`·`ConsensusFamily` | 체인 플러그인 추상화 | **EXTEND** | 깔끔한 seam. capability 선언(`NodeComposition/SupportedForks/SupportedAssertions/TestCapabilities`)을 Manifest/인터페이스에 추가(D-3, 요구 1·2·3) |
| `core/driver` (341) `Driver`+`Initializer`+`FileProvisioner` | local/remote 노드 실행·파일배치 | **KEEP** | `FileProvisioner`가 이미 remote 파일 upload를 추상화(요구 11). Launch/Provision/Stop 계약 유지 |
| `core/config` (152) `Values`(flat map)+`Merge/Resolve/Flatten` | 설정 값 접근 | **KEEP** | **이미 닷키(`a.b.c`)·레이어 병합 지원** → TestSpec 닷경로 리졸버·[7,8] 우선순위(flag>config>default)에 직결 재사용(§D-2) |
| `accounts` (877) `AccountProvider`·`Wallet` | tx 서명 SDK | **KEEP** | §D-2.2 서명. `AddressForKey`로 키→주소 파생도 존재 |
| `core/genesis` (239) `MergeOverride/BuildGenesis/ExtractConfigSection` | genesis 합성·오버레이 | **KEEP** | 오버레이 규약(새 바이트 반환) 유지(E-3-3) |
| `core/node` (115) `Node/NodeSet/Endpoints/Role` | 노드 모델 | **EXTEND** | env.json node table(D-1)이 이를 파생·확장(binary/buildVersion/sync 필드 추가) |
| `core/procman` (173) | PID 생명주기(mutex) | **EXTEND** | remote PID(ssh) 추적 + **etcd 관측**(RPC/IPC, C-etcd) 추가. 락 규약 유지 |
| `core/obs` (336) `Bus`·`Store` | 이벤트 버스(대시보드) | **KEEP** | collector/대시보드 수집의 전송로(item 34) |
| `core/rpc` (312) | JSON-RPC 클라이언트 | **KEEP** | 검증·endpoint 질의 |
| `pkg/mcp` (1780, 30 tools) | MCP 표면 | **EXTEND** | 결과 응답 연동(F14, 요구 31) |
| `pkg/dashboard` (176) SSE | 대시보드 백엔드 | **EXTEND** | 세션 아티팩트 소비(F15) |
| `consensus/{poa,wbft,upgrade}` (250/152/693) | 합의·핸드오프 | **KEEP/REFACTOR** | 로직 유지, `upgrade/exec.go` 순차 launch만 동시화(E-1) + etcd 리더게이트(C-etcd) |
| `chains/{wbft,wemix,stablenet,...}` | 체인 플러그인 | **EXTEND** | capability 선언 추가(D-3) |

---

## 3. 재구성 대상 (REFACTOR)

| 대상 | 문제 | 조치 |
|---|---|---|
| `pipeline/setup` (`Provision`/`Launch`/`Run`) | 노드 **순차** loop | **동시 provision/launch**(errgroup+semaphore, ctx), **세션 경로 인지**(D-1), placement 주입(D-4). 공개 `BuildPlan/Run` 시그니처는 유지하며 내부만 |
| `pipeline/verify` (`Run`) | 노드별 **순차** 폴링(verify.go:90) | **노드별 goroutine 팬아웃**(index 쓰기). 결과 Report는 **헬스 게이트**로 승격(D-6) |
| `consensus/upgrade/exec.go:163` | `for range specs` 순차 init/provision/launch | 동시화 + 리더(producer) 우선 부트스트랩 게이트 |
| remote (`chains/wemix/deploy` 8파일 + `core/remote` + `driver.RemoteDriver`) | **wemix 전용에 치우침** | 체인-무관 remote 절차로 **일반화·승격**(요구 9·10·11·16): 접근·upload·download를 core에서 공용화, deploy는 wemix 특화만 |
| `core/state` (nodeset.json) | 데이터루트 기준 저장 | **세션 레이아웃 하위로 이동**(D-1), env.json과 통합 |

---

## 4. 교체 대상 (REPLACE) — 테스트 계층

| 대상 (LOC) | 현재 | 교체 |
|---|---|---|
| `pkg/testkit` (341) `Case`·`CaseFunc`·`Register` | **테스트 = Go 함수** 등록 | **TestSpec DSL 해석**으로 교체(§D-2). 단 **`Report/Result/Status`(결과 모델)는 재사용** — 판정 스키마(§D-2.6)로 확장 |
| `pipeline/testrun` (`Run`) | Go-func case 실행 | **정의서 해석 실행기로 재작성**: pre→steps→assert→post + 세션 기록(steps/assert/status.json), provenance, **스텝 결과 기대치(positive/negative, F11)**, 실패시맨틱(§D-2.9) |

> 이관 전략: testkit의 Go-func 경로는 **당분간 병존**(기존 e2e/repro 유지)하고, 신규 정의서 경로를 별도로 세운 뒤, 케이스를 점진 이관. 최종적으로 Go-func 등록은 폐기.
> 주의(F11): tx 스텝은 **선언한 기대(기본 `status 0x1`, negative는 `expectRevert`) 충족 = 원자적 성공**. revert가 실패인지는 기대치가 결정(인프라 실패는 항상 실패) — wemix4의 negative 테스트(GOV-015 등) 이관에 필수.

---

## 5. 신설 대상 (NEW)

| 신규 패키지 | 책임 | 흡수/대체하는 기존 |
|---|---|---|
| `core/session` | `.chainbench/<session>/` 정본 소유·경로 파생·기록(D-1) | 흩어진 로그/state 경로 로직 |
| `testspec` | 정의서 파싱·검증(필수/옵션·닷경로·`,`)·해석(§D-2) | testkit Go-func 등록 |
| `core/place` | 통합 배치·포트(local 결정적/OS할당, remote 동일포트)(D-4) | `portplan`(71)+`topology`(149)+setup 내 배치 로직 **CONSOLIDATE** |
| `core/keyreg` | 키 레지스트리(생성·복사·다운·업, 이름매핑)(§D-2.2) | `core/keys`(136)+`deploy/{credentials,keys}` **CONSOLIDATE** |
| `core/collector` | 라이브 로그 tail + chainstate(블록·bp참여·싱크·피어·분기) 수집(D-1, item 34, §D-2.5·2.7) | `probe`(286)+`obs` 확장 |
| `core/supervisor` | 헬스 게이트·etcd 리더게이트·백오프 복구(C-etcd, D-6) | `runGovHandoff` 재시도(원인은닉) 대체 |

---

## 6. 동시성 리팩토링 지점 (요구 E — 크리티컬)

> 현재 goroutine 5·mutex 6파일뿐(sync 프리미티브 극소수), 노드 기동/검증 순차. **테스트는 항상 직렬(§B-4)**; 동시성은 **한 환경 내 N노드** 처리에만 적용.

1. `setup.Provision/Launch`·`upgrade/exec.Launch`: **errgroup 팬아웃 + `semaphore.Weighted(min(cores-2,N))`**(자원 스파이크·성능 35 방지) + **ctx 취소**(실패시 형제 정리→procman.StopAll, 고아 방지).
2. `verify.Run`: 노드별 팬아웃, **index 슬라이스 쓰기(락 불필요)**.
3. `collector`: 노드별 tail goroutine → **채널→단일 writer**(파일 레이스 제거) + 버퍼·레이트리밋.
4. `upgrade` mesh(admin_addPeer N×N): 팬아웃 + ctx.
5. 레이스 근절: **결정적 index 포트 or OS(`:0`) 할당**(고정포트 이중바인드), 공유 genesis 원본 불변, session 상태 단일 writer/flock.

---

## 7. 마이그레이션 전략 (틀어짐 방지)

- **인터페이스 우선**: session/place/keyreg/collector/supervisor의 **인터페이스를 먼저 확정**(feature-spec의 AC와 1:1), 구현은 그 뒤.
- **하위호환 병존**: 신규 경로를 별도 커맨드/플래그로 세우고, 기존 `setup/upgrade/test` 경로는 유지 → 회귀 위험 없이 케이스 점진 이관.
- **검증 게이트**: 각 WP마다 기존 비-e2e 스위트 통과 + 해당 F의 AC 충족(대표 e2e 1건 라이브) 후 다음 WP.

### 7.1 작업 단위(WP) — 구현-ready 분해 (로드맵 §F 순서)

| WP | 범위(패키지·판정) | design 앵커 | feature-spec | 완료 게이트(AC) |
|----|-------------------|-------------|--------------|-----------------|
| **WP1** | `session` [NEW] + `state` 이동 [REFACTOR] | §3.1, §4.1·4.2 | F1 | F1-1..4: 커맨드=세션1, 2축(env/tests) 결정적 경로 |
| **WP2** | `place` [NEW·CONSOLIDATE portplan+topology] | §3.4 | F12 | F12-1: back-to-back 포트 이중바인드 0 |
| **WP3** | `testspec`+`testrun` [REPLACE] · 결과모델 `testkit.Report` 재사용 | §3.2, §4.3·4.4 | F3·F4·F5·F6·F11 | F3-1/F11-2·3: 파싱검증·**negative 스텝 시맨틱** |
| **WP4** | `supervisor` [NEW] + `setup/verify/upgrade` 동시화 [REFACTOR] + `procman` etcd관측 [EXTEND] | §3.3, §6, C-etcd | F13 | F13-2·4: Mode 정확·고아 0 |
| **WP5** | `keyreg` [NEW·CONSOLIDATE keys+deploy] + `collector` [NEW] + fingerprint 재사용 | §3.5·3.6, §D-2.4 | F2·F7·F8·F10 | F7-1·2: 재사용/재구성 판정, F10-1: 로그 누락 0 |
| **WP6** | `registry.Capabilities` [EXTEND] + `mcp`·`dashboard` 연동 [EXTEND] | §3.7 | F9·F14·F15 | F14-1·2, F15: 결과·상태 소비 |

- **인터페이스 우선**: 각 WP는 design §3 인터페이스를 먼저 고정(feature-spec AC와 1:1) 후 구현.
- **하위호환 병존**: 신규 경로를 별도 커맨드/플래그로, 기존 `setup/upgrade/test` 유지 → 회귀 없이 케이스 점진 이관.

---

## 8. 리스크

| 리스크 | 완화 |
|---|---|
| 테스트 모델 교체(REPLACE)의 광범위 영향 | 병존 이관, 결과 모델(testkit.Report) 재사용으로 표면 최소화 |
| 동시 기동이 etcd 타이밍 악화 | supervisor의 **리더 우선 부트스트랩 + gap 정렬**(C-etcd)이 오히려 개선; semaphore로 상한 |
| remote 일반화 범위 과다 | wemix deploy 특화는 남기고 **공통 절차만** core 승격 |
| 세션 레이아웃 변경이 대시보드/MCP에 파급 | session 경로 API를 단일 소유 → 소비자는 API만 사용 |

---

## 9. 설계 문서 세트 · 구현 착수 조건
설계 단계 산출물(모두 `docs/dev/`):

| 문서 | 역할 | 상태 |
|------|------|------|
| `chainbench-requirements-review.md` | 요구 37·사양검토·격차·etcd실체·동시성 | ✅ 정합 |
| `chainbench-design.md` | 구조·인터페이스(§3)·데이터모델(§4)·동시성(§6) | ✅ 검토·정정 |
| `chainbench-feature-spec.md` | F1~F15 동작계약·AC | ✅ 정합 |
| `chainbench-refactoring.md` (본 문서) | 기존→목표 매핑·**WP 분해** | ✅ 정련 |

**구현 착수 조건**: 위 4종 확정 + 각 WP의 design 인터페이스 고정. 구현은 WP1(session)부터 §7.1 순서로, 각 WP는 비-e2e 통과 + 해당 F의 AC 라이브 확인을 게이트로 진행.
