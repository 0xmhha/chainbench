# chainbench 구현 작업 트래커 — 작업 리스트 · 우선순위 · 폴더 트리 예상도

> **[정본]** **작업 순서·상태의 단일 출처.**
> 이 문서는 *무엇을 만들어야 하는가* 를 정한다. 설계 제안이 여기와 어긋나면 **제안을 고친다.**

> 근거: `chainbench-component-architecture.md` (2026-09-24 삭제)(§2b 실측·§3 컴포넌트·§5 Phase·§1b DDD) · `chainbench-design.md`(§3 인터페이스) ·
> `chainbench-feature-spec.md`(F1~F16 AC) · `chainbench-refactoring.md` (2026-09-24 삭제)(WP1~6).
> 원칙: **Low는 TDD 먼저 → walking skeleton으로 조기 통합 → 수직 슬라이스 확장**(big-bang 금지). 코드는 [[go-code-quality-guidelines]] 준수.
> 상태 표기: 본문은 ☐ 미착수 · ◐ 진행 · ☑ 완료. **열린 작업은 본문이 아니라 아래 §0 을 본다.**

---

> **경로 안내 (2026-09-29).** 열린 작업은 §0 뿐이고 그 아래는 이력이다. 이력 문단의
> 경로·명령은 **그때의 트리**를 그대로 적은 것이라 지금 없는 것이 많다 — 옮겨진 것은
> `internal/engine`→`internal/chainsetup`+`internal/testengine`,
> `internal/testspec`→`internal/dsl`, `internal/testkit`→`internal/testsupport`,
> `internal/netmap`→`internal/core/node`+`internal/resource`,
> `internal/keyring`→`internal/core/keyring`,
> `internal/core/launchopt`→`internal/core/nodeconfig`,
> `internal/core/capability`→`internal/core/registry`,
> `internal/consensus/upgrade`→`internal/preset`+`internal/core/hardfork`,
> `tests/cases/`·`tests/specs/`→`tests/tc/` 다. 없어진 것은 git 이력에 있다.
> **§0 의 경로는 현재 트리를 가리켜야 한다.**

## 0. 열린 작업 — 정본 목록 (2026-09-12)

> **여기만 보면 된다.** 아래 본문 2,900여 줄은 **이력**이고, 본문의 `☐`/`◐` 는 *그 시점의*
> 기록이다. 새 작업은 이 목록에 한 줄을 더하고 근거를 본문에 적는다.
>
> **왜 이 절이 생겼나.** 이 문서는 마커를 **두 어휘**로 써 왔다 — 머리말이 선언한
> `☐ ◐ ☑` 와, 나중 절들이 쓰는 GitHub 식 `- [ ]`/`- [x]`. 2026-09-12 에 세 차례에 걸쳐
> "미완료 0건" 으로 보고했는데 **틀렸다**: `- [ ]` 와 `◐` 만 세고 `☐` 11개를 빼놓았다.
> 게다가 마커가 **산문·셀 중간·셀 머리** 세 자리에 섞여 나와서 기계로 셀 수 없다 —
> "마지막 셀이 마커로 시작" 규칙을 짜 보니 S2 를 놓치고(줄바꿈된 표 행) 다른 행을
> 잘못 셌다. 그래서 **세는 대신 한곳에 적는다.**
>
> 한계는 정직하게 적는다: 이 목록은 **본문에만 조용히 추가된 항목을 잡지 못한다.**
> 잡는 것은 목록 자체의 표류다 — `arch.TestWorklistOpenWorkIsListed` 가 개수를 못 박고,
> 각 줄이 가리키는 절이 실제로 있는지 확인한다.

- [x] **Web UI 전체 구현과 14개 인수 조건** — **완료 (2026-10-12).** 근거: [전체 작업 목록](web-ui-worklist.json), [완료 기준과 단계 기록](web-ui-worklist.md), [누락 목록](web-ui-gaps.json).
      문서 대조에서 나온 누락 48개를 모두 닫고 CLI 전환 PR #448을 main `662fff77`로 병합했다.
      소스 지문 `3887afccc004727a051e913ce61d937e37a7274cd1036bc3b0372bb3d15a70bc`에서 열네 기준이 capture와 독립 재현을 모두 통과했다.
      후속 변경의 최종 인수는 §0a에서 별도로 추적한다.

- [ ] **수직 슬라이스 확장** — 근거: §1 표 `10+` 행. 남은 조각은 그 행이 적고 있다.
- [x] **b-5 `setup` → `net up` 전환 — 현행 재평가 완료 (2026-10-14)** — 근거: §1f 표 `b-5` 행과 §0a 12번.
      현행 구성 표면은 `chain up`이고 과거 두 명령은 별칭까지 없다. 실제 세 체인 공개 CLI 구성·구동·종료·제거와
      설정·기록 보존을 확인했다. [증거](setup-transition-evidence.json). 메인넷 전체와 최종 Web 인수는 별도 잔여다.
- [x] **S2. MCP 레지스트리 소비 — 완료 (2026-10-13)** — 근거: §1g 표 `S2` 행과 [실제 호출 증거](mcp-registry-evidence.json).
      표준 app 직접 호출 60곳을 typed binding으로 전환하고 공통 문자열 스키마 36곳을 레지스트리에서 가져온다.
      복합 RPC 배열 두 곳과 플러그인·capability 카탈로그의 함수 핸들러는 보존했다. 입력·출력·스키마·층 경계 계약,
      전체 make check와 관련 race, 세 체인 각각 4노드의 실제 MCP 구성·조회·정지·제거가 통과했다.
- [x] **N2~N6. blueprint 흡수가 중간 상태다** — **완료 (2026-09-20). 항목의 전제가 틀렸다.**
      §1m 의 `◐` 는 "`chainsetup` 이 `blueprint.X` 를 부르는가" 로 판정한 것인데, **그건 N3·N4·N6 어느 것의
      게이트도 아니다.** §1g 가 다섯 전부를 2026-09-07 라이브 게이트와 함께 `☑` 로 적고 있고, 셋을 코드로
      다시 확인했다: N4 는 `chainsetup` 직접 파일 쓰기 **0곳**, N6 은 `verbs_steps.go:149` 가 혼용을 거부
      (병존은 **설계된 이관 기간 동작**이지 미완성이 아니다), N5 는 `chaincmd/blueprint.go` 의 두 플래그.
      `placements()` 의 세 갈래도 N6 이 의도한 병존이다. §1m 의 `◐` 를 `☑` 로 고쳤다. 실제 잔여는 아래 한 줄.
- [x] **N3 서버 배치·패밀리 BLS — 완료 (2026-10-13)** — 근거: §1g 표 `N3` 행과 [실제 CLI 증거](n3-allocation-evidence.json).
      blueprint의 노드별 `server`를 슬롯 요청에 전달하고 명시 배치를 먼저 예약한다. 서버 이름·주소 중복,
      미등록 서버·지정 서버 용량 초과·잘못된 역할은 점유 없이 거절한다. 원격 참조와 로컬 경로의 계약을 보존했다.
      생성·선언·혼합 키 모두 `ValidatorsCarryBLS()`에 따라 파생하며 주소와 기존 keyPreset은 보존한다.
      전체 make check·관련 race·세 체인 4노드 CLI 생성·구동·정지·제거를 통과했다.

- [x] **2단계·3단계 (netmap 트랙) — 현행 재평가 완료 (2026-10-14)** — 근거: §1m과 §0a 13번.
      서버 배치·패밀리별 BLS 잔여, 현행 소유자·역할 계약, A8 재측정과 A7b 판단을 확인했다.
      전체 코드 AST 그래프 capture·독립 재현에서 누락과 끊어진 간선이 0이다. [증거](all-code-graph-evidence.json).
      2026-10-16 후속 고정 소스도 [별도 갱신](all-code-graph-refresh-evidence.json)으로 확인했다.
      스윕/표준입력 경계 보강 고정소스2,075파일·내장AST44·동적입력31도 [별도 측정](all-code-graph-sweep-round-refresh-evidence.json)으로 확인했다.
      후속 준비 코드와 설치 셸도2,081파일·누락0으로 [갱신](all-code-graph-web-preflight-refresh-evidence.json)했다.
      원문 TC 충분성과 외부 W3·W4는 이 완료에 포함하지 않는다.
- [x] **통폐합 — 닫을지 판단** — **완료 (2026-09-20). 닫았다.**
      `architecture/consolidation-plan.md` (2026-09-24 삭제) 을 `[이력]` 으로 내렸다.
      진단("작은 형제 24개")이 7개로 해소됐고, 남은 3건은 각각 import 순환·코드가 명시적으로 반대·층 역행이다.
      **방향 뒤집기(큰 덩어리)는 이 문서를 이어받지 않고 백지에서 새로 설계한다** — 이 계획은 "조각이 너무 많다"
      는 진단 위에 서 있어 이어받으면 반대 방향으로 끌린다. 제약 셋은 문서가 아니라 코드가 지킨다
      (`filestore/doc.go` + `arch.TestCommentsDoNotContradictTheCode`, `arch.TestNoUpwardDependency`, Go 의 순환 금지).
- [x] **요구 19개의 충분성 — 검토·보강 완료 (2026-10-13)** — 근거: §1s와
      [19개 행동·증거·한계 검토](requirement-sufficiency-review.json). 연결된 이름만으로 충분하다고 판단하지 않고
      각각의 실제 assertion과 생산 경로를 대조했다. R11 배포 provenance, R14 직렬 수행, R15 공개 계획 동등성,
      R19 같은 초·동시 세션 격리를 RED→GREEN으로 보강했다. 주석·문자열을 테스트로 세던 연결 검사는 Go AST로 바꿨다.
      관련 전체 race와 세 체인 실제 배포·세션 영속·종료 데이터 회수를 통과했다.
      이 검토는 아래 메인넷·공통 TC·수직 슬라이스와 최종 Web 인수를 닫는 근거가 아니다.
- [ ] **메인넷 설정 트랙 — 라이브 잔여 셋** — 근거: [`architecture/mainnet-config-worklist.md`](architecture/mainnet-config-worklist.md).
      결과는 그 문서 §11.4.1(L1)·§11.4.2(L2). 오프라인 잔여는 2026-09-18 에 0 이 됐고, L1 전체 회귀와 L2 경합 전수는 2026-09-20 에
      닫혔다(§11.4.1·§11.4.2). `restart-at-boho` 도 2026-09-20 에 닫혔다 — pre/post 두 바이너리를
      golang 컨테이너 안에서 네이티브로 빌드해 쌍으로 돌렸다. 남은 것은 **HEAD 바이너리로 L1 재실행**
      (위 실행은 2026-08-10 빌드로 돌았다. `PlaceBinary` 는 절대 경로를 *대상 위의* 경로로 읽으므로
      로컬 빌드가 함대에 올라가지 않았다) · 실제 망 대응 R2~R6. 후속 Linux L1은 소스e967의44통과·1실패·1운영자 중단과 소스908의30통과·1BLOCKED·1실패인 별도 부분 배치다. 합산해 전체 인수로 세지 않는다.
      StableNet의 previous prepared seal 거절은 [S1 인계](chain-handover-2026-09-12.md#7-s1--stablenet의-previous-prepared-seal-거절-2026-10-15)에 기록했으며 원인은 미확정·미해결이다.
- [ ] **공통 TC 가 요구하는 도구 기능 둘이 어느 항목도 아니다** — 근거: [`architecture/mainnet-config-worklist.md` §8](architecture/mainnet-config-worklist.md) 의 대응표(2026-09-20 신설)와
      [`docs/tc/common/03-separate-implementation-items.md`](../tc/common/03-separate-implementation-items.md) §2.
      여덟 중 여섯은 R2~R6 이 덮지만 **동기화 경로 관찰**(어느 경로로 블록을 받았는지)과
      **준비물 공유**(한 케이스 파일이 저장한 값을 다른 파일이 이름으로 읽기)는 R 에도 V 에도 P 에도 없다.
      P4 는 preset→케이스 방향이라 케이스 파일 사이를 잇지 못한다. 회귀 실행 묶음 A·C 가 준비물 공유를 전제한다.
      **R 묶음을 다 끝내도 공통 TC 는 열리지 않는다.**

- [ ] **go-wemix 코어: 거버넌스가 반만 지어진 구간에서 full sync 가 영구히 멈춘다** — 근거: §1s G6,
      인계는 [`chain-handover-2026-09-12.md`](chain-handover-2026-09-12.md) **W3**.
      **W1 과 같은 블록의 이웃한 줄이다**(`admin.go:997-1003`) — 함께 고치는 것이 맞다.
      **chainbench 가 고칠 수 없다 — 체인팀 인계 건이다.** 거버넌스는 블록 4~13에 걸쳐 지어지고
      (proxy 가 imp 주소를, registry 등록이 proxy 주소를 필요로 하는 의존 사슬), 그 중간 블록의
      상태를 쥔 채 다음 블록을 검증한 노드는 "registry 는 있는데 주소록이 비었다" 를 본다.
      `wemix/admin.go:1000` 이 그 revert 를 "아직 준비 안 됨" 으로 인정하지 않아 `unauthorized block`
      으로 판정하고, 상태를 버릴 수 없으므로 **영구 고장**이 된다. 15노드 poa 에서 재현 2/2,
      걸리는 노드는 매번 다르다(배치 경계 경합). full sync 만 취약하다.
      **이 항목이 열려 있는 한 `02-wemix-chain-up-15` 는 BLOCKED 로 남는다.**
- [x] **죽은 명령을 부르는 e2e** — **해소 (2026-09-24). 둘이 아니라 스무 개였다.**
      이 항목은 `upgrade_data_migration_e2e_test.go` 와 `upgrade_gov_ncp_lifecycle_e2e_test.go`
      둘이라고 적었다. 바이너리를 걸고 돌리니 **7개 파일 20개 함수**가 전부
      `chainbench upgrade run` 에서 죽었다 — 파일을 셌지 함수를 세지 않았고, 돌려 보지
      않았다. `invocationDebt` 도 같은 정도로 적게 잡고 있었고, 지금은 비어 있다.
      스무 개 전부 리팩토링이 남긴 경로(`chainbench run` + `wemix-to-wbft` chain-preset)로
      옮겼고 21건이 통과한다. 상세는 §1s G4 · G4-a.

- [x] **feature 레지스트리 미등록 0건 — 완료 (2026-10-13)** — 근거: `internal/feature/coverage_test.go`와
      [실제 등록 호출 증거](feature-registry-evidence.json).
      현재 app usecase 90개 모두 등록되었고 미등록은 0개다(2026-10-13 AST 함수 연결 측정).
      새 등록의 typed 호출·키링 공개/비밀 분리·네트워크 기록·RPC 계약과 CLI/MCP 관련 race가 통과했다.
      같은 등록을 통해 세 체인 각각 4노드를 실제 구성·구동·조회하고 정지·제거하는 직렬 race 증명을 통과했다.
      MCP의 입력 스키마와 실행 경로를 레지스트리 소비로 바꾸는 후속 작업은 S2에 남는다.

- [x] **`R` 접두사가 두 계획에서 충돌한다** — **해소 (2026-09-20). 개명하지 않고 한쪽을 닫아서 풀었다.**
      근거: §1n 표 `R1` 행(원격 트랙)과 `architecture/consolidation-plan.md` (2026-09-24 삭제) §R1(소형 모듈 흡수)이
      같은 `R1`~`R5/R6` 으로 다른 것을 가리켰다. 통폐합 계획이 `[이력]` 이 되면서 **살아 있는 `R` 은 원격 트랙 하나뿐**이다.
      새 리팩토링은 백지에서 번호 체계를 새로 잡으므로 재충돌하지 않는다.
- [x] **`architecture/target-architecture.md` (2026-09-24 삭제) 가 [현행 설계]인데 낡았다** — **완료 (2026-09-20).** [이력]로 내렸다. 내린 근거는 실측이다: 그 문서가 L1 로 그리는 모듈 이름 일곱
      (`driver`·`netmap`·`place`·`portplan`·`target`·`serverset`·`peering`)이 **전부 `internal/` 에 없고**,
      지금 쓰는 `internal/resource` 는 **0회** 나온다. 흡수된 자리를 문서 머리에 적었고, 인덱스 행도 바꿨다.
      같이 고친 둘: `refactoring-proposal/` (2026-09-24 삭제) 6종이 무등급·미등재였고(**[제안]** 등급을 새로 정의해 등재),
      `consolidation-plan.md` (2026-09-24 삭제) 의 인덱스 행이 문서 스스로 "낡았다"고 적는 것과 어긋나 있었다.

- [x] ~~**T. 테스트 영역(0x8000)의 상태**~~ — **완료 (2026-09-22).** 근거: [`architecture/design-v3/state-machine-05-test-failures.md`](architecture/design-v3/state-machine-05-test-failures.md).
      실패 102곳을 재고(T1), 여섯 블록에 실패 스물을 붙여 전이표를 쓰고(T2), 실행이 자기 상태를
      말하게 하고(T3), 붙기 경로까지 같은 단계를 지난다는 것을 확인했다(T4). **전 케이스 라이브로
      209/209 통과**(71분, 실패·blocked·skip 0건) — stablenet 181 · wbft 15 · wemix 13, 하드포크
      세 건과 포크 전 빌드를 요구하는 `01-boho-crossed-by-restart` 포함. 스위프 도구는
      `scripts/tcsweep.sh` 에 남겼다.
- [x] **선언 모델 — 닫힘 (2026-09-24 표시 정정).** 제목은 "남은 여섯" 이라고 적고 본문은 "이것으로
      닫힌다" 로 끝나 있었다. 설계 문서의 표에서 P-1~P-8 이 전부 취소선이고 D-a·D-b·D-c 도
      끝났다 — 일이 아니라 **표시가 남아 있었다**. 근거: [`architecture/design-v3/declaration-model-2026-09-22.md` §5](architecture/design-v3/declaration-model-2026-09-22.md).
      결함 셋(D-a·D-b·D-c)과 P-1·P-1b·P-2·P-3·P-8 은 끝났다. **P-5 는 할 일이 없었다** —
      `suite run` 에는 이미 `--server-set` 이 있고(`resourcecmd.ServerFlags` 를 빌려 쓴다),
      MCP 도 마찬가지다. **P-4 도 끝났다** — `internal/core/origin` 에 rung 일곱을 둔 어휘 하나를
      만들고 기록 둘이 그것을 쓴다. `nodeconfig.Layer` 는 "누가 요청했는가" 라는 다른 질문에
      답하므로 합치지 않았고, 그 이유는 설계 문서 §P-4 에 있다. **P-6 도 끝났다** — 검사를 쓰면서 결함 둘이 나왔고(한 노브를
      여러 노드에 주면 망이 겹친다 / 스코프가 좁으면 CLI 를 이긴다) 둘 다 고쳤다. **P-7 도 끝났다** — README 를 다시 썼고,
      `direction.md` 의 "정본" 표에서 없어진 갈래를 지웠고, #419 설계 기록에 뒤에 무엇이
      바뀌었는지 가리키는 줄을 달았다. **선언 모델은 이것으로 닫힌다.**

**체인팀 몫 (여기서 할 일 없음)**: R6 잔여(go-wemix boot-etcd) · W1 `verifyBlockSig` 패닉 ·
B1 `istanbul_getWbftExtraInfo` 블록 태그. 정본은
[`chain-handover-2026-09-12.md`](chain-handover-2026-09-12.md).

**일부러 비워 둔 자리 (요구가 생기면 연다)**: 네트워크 공통 건강 게이트의 `WantChainID`(케이스별 선언은 아래 R1 실행 관문에서 검사) ·
stablenet 고아 핸들러 방향(registry 에 lister 없음) · `process/stop.go` 의 취소 의미(의도적).


> **이어서 할 일은 §1n "다음 사람에게" 부터 읽는다** (2026-09-08). 남은 것이 무엇이고 무엇이
> 필요한지, 그리고 라이브를 쫓을 때 이번에 비싸게 배운 측정 함정 여섯이 거기 있다.

---

## 0a. 후속 작업 실행 목록 (2026-10-13)

사용자는 아래 14개를 권장 순서로 끝까지 수행하도록 승인했다. §0의 제품 잔여 10건과
추가 운영·검증 작업 4건을 함께 추적한다. 모든 단계는 의미 있는 RED→GREEN, 관련 회귀,
필요한 실제 증명을 통과한 파일만 커밋한다. push·PR·merge는 별도 사용자 승인 뒤 수행한다.
미추적 인수인계 문서는 스냅숏이며 아래 상태를 덮어쓰지 않는다.

| 순서 | 작업 | 완료 기준 | 상태 |
|---|---|---|---|
| 1 | 증명 보존 정리·WAL 압축 도구 | 참조·활성·링크·최근 파일 보존; 조회 실패 거절; macOS 압축 bytes·권한·소유자·mtime·xattr·ACL 보존; 56개 receipt 해시 대조 | 진행 — 작업 Go 빌드 캐시의 보존 계약24개·전체 검사와 실제17,132개/약3.86GiB 회수·남긴20,121파일 독립 대조를 보강했다([증거](go-cache-maintenance-evidence.json)). 과거 receipt 미발견분과 전체 인수는 여전히 남는다. Linux 보존 전송을27계약/6개 RED·CI와 실제 파일6회/거절5회, 세 체인 실제 시도15보관/735항목의 독립 대조로 보강했다([증거](linux-retention-transport-evidence.json)). 기능 결과는2통과·WEMIX EN 합의 확인1BLOCKED이며 전체 인수는 아니다. 격리 Linux 경로 선택14개 RED·10계약/관련/전체 검사와 종료6노드364항목의 읽기 전용 capture·독립 대조를 보강했다([증거](isolated-linux-retention-evidence.json)). 기존 보존 계약·압축 검증 완료; 새 JSON/Markdown/인계 참조 보강·47계약·실제 보존/압축·호환 명령 dry run 완료. 과거 56 receipt는 28 일치·26 없음·2 다른 해시이므로 자료 전수 확인/후속 인수는 미완료. 추가 복사본·Git·314압축 파일 검색에서29/56의 동일 bytes 위치를 확인했으나27개는 범위 내 미발견이며 원본을 덮어쓰지 않았다([복구 검색](historical-receipt-recovery-evidence.json)). [새 증거](storage-document-evidence.json). 기존 근거: `9812165f`, 보존 계약 17개·Python 74개·`storage_probe.py` 실제 실증 |
| 2 | 문서·현재 상태 정합성 | Web JSON/Markdown 상태 일치; 과거 단계와 후속 HEAD 검증 구분; feature 수치의 AST 래칫 대조 | 완료 — `worklist_status_test.go` RED→GREEN; 출시 14/14·누락 48/48·미등록 59건; 후속 공개 명령 안내도202→216 갱신·14누락 해소·3실패 대조/재생성·CI 누락 RED→직접 실행/4계약·216셸 구문/209단일 스위프 정의의 독립 대조와 CI 래칫 보강([증거](case-command-docs-evidence.json)) |
| 3 | suite 실패 테스트 3건 보강 | 존재하고 유효한 DSL·키 입력으로 실제 setup 실패까지 도달; JSON/비JSON 동일 code 2와 구체 원인 | 완료 — `setup_failure_test.go` 세 경로 RED→GREEN; suite 전체 race와 실제 CLI 단일·다중·비JSON 실행 모두 code 2·바이너리 셋업 원인 확인 |
| 4 | dashboard 임시 폴더 정리 경합 | 재현 또는 수명 주기 근거 확보; 작업 종료 대기 뒤 폴더 정리; 반복·race 통과 | 완료 — 원래 테스트 1,000회 race 미재현; 지연 fixture에서 최종 commit 전 TempDir 정리를 RED로 재현; terminal 기록 대기 후 두 테스트 100회·dashboard 전체 race 통과 |
| 5 | feature usecase 등록 | 잔여 app usecase 전부를 의미 있는 레지스트리 호출·타입 계약으로 연결; 미등록 0 래칫과 CLI/MCP 회귀 | 완료 — 59건 미등록 RED→90/90 함수 연결 GREEN; typed·키링·네트워크·RPC 계약, 관련 전체 race와 세 체인 native 호출 통과; `feature-registry-evidence.json` |
| 6 | S2 MCP 레지스트리 소비 | 실제 중복·설명 차이 측정; 공통 입력은 레지스트리 소비; 특수 입력 보존; 스키마·실행 계약 회귀 | 완료 — 표준 호출 60곳·공통 문자열 스키마 36곳 전환; 복합 배열·카탈로그 보존; 계약·make check·race·세 체인 실제 MCP 실행 통과; `mcp-registry-evidence.json` |
| 7 | N3 서버 배치·BLS | blueprint 노드별 server 반영; 중복·미등록 배치 거절; BLS 필요 패밀리만 파생; 관련 실제 체인 회귀 | 완료 — 배치·실패 원자성·로컬 경로·3종 키 원천 RED→GREEN; make check·race·세 체인 CLI 실제 구동과 정리 통과; `n3-allocation-evidence.json` |
| 8 | 요구 19개 충분성 | R1–R19 각각의 동작을 증명하는 테스트 대조; R14·R15·R19 등 부족분 RED→GREEN 보강 | 완료 — 19개 assertion·생산 경로·검증 한계 대조; R11·R14·R15·R19 RED→GREEN; Go AST 증거 연결; 관련 전체 race·세 체인 실제 배포·CLI/MCP 실행·root report와 종료 DB 회수 통과. 전체 live·upgrade·최종 Web 인수는 별도 잔여; `requirement-sufficiency-review.json` |
| 9 | 메인넷 설정 live | HEAD 소스 출처가 확인된 실제 바이너리로 L1 회귀; 실제 망 R2–R6 전수 결과·제약 기록 | 진행 — 기본 round의 체인 중복과 attach 결과 경로 누락을14RED→16계약·기존19계약·DSL race/전체 검사로 고쳤다. 고정34b 소스의 RPC016 세 체인+attach4PASS/86단언·47동작, 준비 host 포함5호출·20보관/1,035항목·624로컬파일·DB/규칙0을 독립 확인했다. 첫0실행 선택 오류와 자료도 보존했고 전체358/최종Web은 남긴다([증거](sweep-family-output-evidence.json)). 소스4e0의79PASS/279NOTRUN을 정상 경계에서 중단하고353보관/20,414항목·10,736로컬파일·DB/규칙0을 독립 확인했다. 실제 체인은StableNet72/WBFT1/WEMIX6이며53번의StableNet 표시는 실제WBFT와 달라 의도한 체인 전수를 거절한다. attach1건의 기본 세션 경로 이탈11파일도 원본을 유지한 보관 bytes/권한/소유자/mtime 대조로 보존했다. 당시 남긴 계획·출력 경로 결함은 위 스윕 보강으로 고쳤으며 전체 L1 재실행은 남긴다([이전 발견](linux-l1-round-boundary-evidence.json)). 현행96cc 소스에서16PASS/342NOTRUN과74보관/4,456항목·DB/규칙0을 독립 확인했다. 뒤쪽 서버 충돌의 지연 거절은8RED→GREEN·관련/전체 검사·세 체인 정상3/거절3의 실제 증명으로 보강했다([검증](partition-placement-evidence.json)). 전체 인수는 남긴다([증거](linux-l1-placement-boundary-evidence.json)). firewall 복구 오류7RED→GREEN·관련 전체 race/전체 검사와 세 체인 실제 복구 성공을 확인했다. 원래 케이스2PASS/1FAIL 뒤 승인 판정의 P2P 범위3PASS를 확인했다. 기록된 etcd 합의 포트도4RED→GREEN·회귀/전체 검사·동일 소스 재검증3PASS와80교차 TCP/8내부 연결의 차단·복구로 보강했다. 첫2PASS/1BLOCKED와 캐시 생성 예외도 보존했으며 전체 인수로 세지 않는다([증거](partition-consensus-port-evidence.json)). 현재 고정 소스216파일/358계획에서 기본·계약·장애14PASS/344NOTRUN을 확인하고 방화벽 복구 오류 전파 검토를 위해 정상 경계에서 중단했다.63보관/3,788항목·로컬1,950파일·15대 종료/DB0과 남은 partition규칙0을 독립 확인했으며 전체L1은 미완료다([증거](linux-l1-current-boundary-evidence.json)). 중단 표식을 케이스 정지/보존 뒤 확인하도록5개 RED→19계약·전체 검사로 보강했다. 실제358계획에서1PASS/357NOTRUN·다음attach 미시작·DB0을 독립 확인했으며,2건 선택 인자 누락/원 검증기 실패도 보존했다([증거](sweep-boundary-stop-evidence.json)). 모든 기록 노드의 RPC/블록 누락을 실패로 처리하고 합의 오류를 고정 분류/숫자 코드로 보존했다.16개 RED·관련 전체 race·전체 검사와 같은 고정 소스의 세 체인5노드9단언/독립 감사를 통과했다. 앞선2차단/1실패와 간헐 시간 제한은 보존·미해결이다([증거](all-node-probe-integrity-evidence.json)). 후속 WEMIX CT-NODE-009의 별도 반복은 기능 소스 변경 없이1건/3단언·5노드 확인이 통과했다. RPC55개·5실행 바이너리·135보관 항목·독립 감사와 종료DB0을 확인했지만 앞선 ENunknown 원인은미확정이며 해결로 표시하지 않는다([진단](wemix-en-capability-diagnostic-evidence.json)). 이전 실제 CT-NODE-009는 StableNet/WBFT 통과·WEMIX EN 합의 확인 unknown으로BLOCKED였다. 노드5개 모두높이51이었고 W3의 unauthorized block은 관찰하지 않았다. 원인은미확정이며 보존/회수15개만 완료다([증거](linux-retention-transport-evidence.json)). 현행216파일·358회 계획의 격리 시작은1건/2단언 통과·두 번째 attach host 중단 복구이며 전체는 미완료다([파일·종료·독립 보존](linux-l1-isolated-pilot-evidence.json)). 외부 저장소의 로컬 HEAD 세 체인 native 재빌드·출처 기록; R2 BP4/EN1 CLI/MCP 인증 읽기와 구조화 UI 통과; P2/P3 초기값·R1 선언 ID 관문 완료. R3 합의 여러 노드 조회·PUSH0·P256 실제 확인, P4 선언 사유 기록을 검증했다. R4 정수 제어 18필드와 실제 CLI/MCP 실패 144건·Web 검증도 통과했다. 종료 DB 제거 통과. R6 CLI/MCP 다중 preset 실행·출처 기록도 검증했다. R3 RPC·WS·시스템 컨트랙트의 실제 존재 확인은 네 native 프로필·24실행·320노드 관측으로 통과했다. 컨트랙트 조회의 노드별 체인 주소표·미등록 체인 거절·과거 기록 fallback도 실제 전환4노드3프로필로 보강했다([증거](recorded-contract-capability-evidence.json)). Linux15대 fleet·세 체인 원래15노드 시작·과거→현재 포크/서명 교체2건은 완료했다. 후속L1은 소스e967의46행과 소스908의32행인 별도 부분 배치다([두 번째 배치](linux-l1-second-cohort-evidence.json)). 공식 StableNet4d17eea0·WBFTd535dd3d의 별도 Mac/Linux 빌드·출처를 확인했다([입력](upstream-pinned-build-evidence.json)). 새 두 입력의 원래15노드 시작·정지·독립 보존은 통과했다([실제 검증](upstream-fleet-startup-evidence.json)). StableNet Mac의 ProcessExtraSeal nil패닉도 추가 관찰했으며 원인·빈도·해결은 미확정이다([기록](legacy-extra-state-evidence.json)). 배포 delegate 케이스까지 더한 현행 계획은216파일·358회(미실행)이며 준비 확인의 실제 시간 상한을 보강했다. 전체 L1·R3 다른 동작·P4 충분성·R4 포크 제어 실증/전수 회귀·서명 변조의 typed 제출 거부는17대조·공개6실행·노드6관측으로 보강했다([증거](tamper-submission-rejection-evidence.json)). R5 다른 영향 케이스·전체 출처 충분성은 남음; 상세 증거는 아래 단계 기록 |
| 10 | 공통 TC 도구 둘 | 블록 수신 경로 관측의 출처와 의미 검증; 케이스 사이 이름 기반 준비물 공유·격리·실제 실행 | 진행 — 준비물 공유 완료. metric 출처·실패 거절·대체 경로·구조화 UI와 세 체인 관측 가능한 14개 실제 케이스 완료. WEMIX CT-NODE-007은 외부 ETH/LES 계량기 중복 등록으로 미완료이며 15개 전체 통과로 표시하지 않는다; [준비물 증거](prepared-sharing-evidence.json), [동기화 증거](sync-path-evidence.json), [W4 인계](chain-handover-2026-09-12.md#6-w4--ethles-fetcher-계량기-이름-충돌-2026-10-14) |
| 11 | 수직 슬라이스 재평가 | upgrade·wemix4 이관·legacy 잔여를 실제 소비자와 회귀로 확인; 제품 잔여 구현 또는 근거 있는 해소 | 진행 — 공개 서명 교체의 주 노드·두 EN 영수증 송수신자도4개 RED·23대조, 새 Linux 고정 소스1건/42단언·독립 대조·정상 종료/DB 회수로 보강했다([증거](legacy-signature-receipt-evidence.json)). 원문 서명 호환의 두 EN·블록/영수증 보존·현재 높이±2·완전 로그를21로그/19케이스 대조와 Linux1.0.1→고정1.1.0의 실제1건/36단언·독립대조·종료 DB 회수로 보강했다([증거](legacy-signature-compatibility-evidence.json)). 공개 제네시스의 모든 노드 해시·부모·ID를 세 체인 실제·독립 검증으로 보강했다([증거](legacy-genesis-evidence.json)). 종료·전환·재접속 회귀 보강과 기존 고정 소스 20개 전체 race/DB 회수 완료. GOV-018 staking 재개를 실제 RED→GREEN으로 보강해 receipt·금액·정확한 이벤트를 확인했고 관련 실제 3개 회귀도 통과했다. GOV-019의 목록·후속 에폭 복귀·정확한 이벤트도 실제 RED→GREEN으로 보강했다. NODE-002의 정확한 블록·트랜잭션·영수증 이관과 포크 이후 동기화도 실제 race로 보강했다. TX-017과 기존 RT-A-3-07의 실제 쓰기 뒤 롤백·가스·수수료도 세 체인에서 보강했다. GOV-019 미달 금액 채굴 거절과 GOV-020 즉시 이벤트·GOV-021 축약 지연 기준은 사용자 승인 뒤 정확한 실제 단언으로 보강했다. TX-018의 유한한 estimateGas - 1 입력은 실제 대조했지만 WBFT·StableNet에서 성공해 원문 기대와 충돌했고 사용자가 추정 상한과 실제 부족한 가스를 구분하도록 승인했다. 승인 기준의 공통 DSL 케이스와 세 체인 실제 회귀를 완료했다. WBFT-010의 실제 서명/mix 연쇄와 WBFT-002 건강한 개별 간격도 보강했다. TX-010 실제 영수증 순서와 TX-013 queued 교체 전후·기존 hash null도 세 체인에서 보강했다([증거](nonce-order-replacement-evidence.json)). 승인 GOV-004/011의30초·70초 실제 출금과 원문 설정의 별도 배포 조회도6개 실제 race/독립 대조로 보강했다([증거](approved-unbonding-evidence.json)). 원문 검토85 scoped-confirmed·1충돌·1병합이며37개 WEMIX 전체 native race/종료 보존 대조를 완료했다([증거](wemix37-regression-evidence.json)). 원문 전체·공개 DSL·legacy 소비자 재평가는 남음. 캐시된 Git 커밋에서 legacy셸460개를 복구해 bytes/모드·외부셸AST를 확인했다. 원문02/10의 코드 해시·경계 상태·4슬롯·v2조회는14개 RED→GREEN과 새StableNet의2공개케이스·14실제단언·독립대조로 보강했다([증거](legacy-code-state-evidence.json)). extra02/07b의 양방향·여섯 상태·경계 잔액도18개 RED와22개 대조, 두 실제구성·36단언으로 보강했다([증거](legacy-extra-state-evidence.json)). 첫 실제시도의 StableNet ProcessExtraSeal 패닉/BLOCKED는 보존했고 미해결이다. 개인API 원문4건도 공개DSL·전용키·실제4건과기존6회귀로 보강했다([증거](legacy-personal-delegation-evidence.json)). 원문basic/sync의 두 높이 차이 검사는14개 잘못된 통과RED→18대조와 준비된BP4·EN1의 세체인15노드·39단언·독립대조로 보강했다. 초기RPC오류는 보존하고 실제검증자 준비 뒤 동일공개케이스를 읽기전용으로 실행했다([증거](legacy-sync-spread-evidence.json)). 전체legacy 시나리오 동등성은 미확인([입력 복구](legacy-source-recovery-evidence.json)); [종료·전환](handoff-runtime-evidence.json), [재접속](handoff-reattach-evidence.json), [20개 회귀](handoff-regression-evidence.json), [GOV-018](gov-staking-resume-evidence.json), [GOV-019](gov-staking-reactivation-evidence.json), [NODE-002](data-migration-evidence.json), [실패 거래](transaction-failure-evidence.json), [거버넌스 거절·이벤트](governance-rejection-events-evidence.json) |
| 12 | b-5 재평가 | 현행 `chain up`과 기존 `setup→net up` 전제 대조; 죽은 명령 도입 없이 경로·회귀·상태 정정 | 완료 — 실제 root 등록·구성 입력과 과거 상태를 RED→GREEN으로 대조. CLI/MCP/Web/DSL의 현행 구성 소비 경로 확인; 전체 검사·관련 race·외부 저장소 로컬 HEAD 바이너리 세 체인의 공개 CLI 구성·블록 전진·종료·노드 데이터 제거 통과. 설정·기록 126파일 보존; [증거](setup-transition-evidence.json) |
| 13 | netmap 2/3 재평가 | A8 재측정; A7b 현행 판단과 실제 코드 근거; Go·Python을 포함한 모든 코드 언어의 AST 인벤토리·분류 그래프와 누락 0 검증; 열린 단계와 역사 기록 일치 | 완료 — WAL감사경로·명령시간제한 보강 뒤 고정cca2 코드2,093파일·138,658노드/147,999간선·누락/끊어진간선0·두그래프 일치와구문703키/766발생·특정누락1,406거절·과거70그래프보존을확인했다([현재 측정](all-code-graph-wal-audit-evidence.json)). 이전 측정: 종료triecache·실제검증자 준비 보강 뒤 고정1125 코드2,093파일·138,516노드/147,855간선·두그래프 일치·누락/끊어진간선0을 재확인했다. Go/Python/공개DSL 구문714키·755발생과 특정누락1,428거절·과거66그래프 보존을 확인했다([최근 측정](all-code-graph-legacy-sync-evidence.json)). 고정7e13 코드2,093파일·누락/끊어진간선0·두 그래프 일치, 새 Go/DSL 구문 누락172거절과 과거62그래프 보존을 확인했다([최근 측정](all-code-graph-legacy-sync-evidence.json)). 새 높이 차이 native 증명과 전체 인수는 별도 잔여다. 완료 캐시·CI 보강까지 고정3541 소스2,091파일·누락/끊어진간선0·두 그래프 일치, 특정누락386거절과 과거50그래프 보존을 확인했다([최근 측정](all-code-graph-completed-cache-evidence.json)). Web 인자 보강 뒤 고정0b8f 소스2,091파일·누락/끊어진간선0·두 그래프 일치와 특정누락144거절·과거46그래프 보존을 확인했다([최근 측정](all-code-graph-web05-arguments-evidence.json)). 포크 전 합의 보강까지 고정 b492 소스2,090파일/137,815노드/147,121간선·누락/끊어진간선0을 두 번 재현했다. Go테스트800·내장AST41/동적25·일반변조12/특정누락90거절·과거42그래프 보존과 A8 공개비메서드1,891/도달불가0을 확인했다([최근 측정](all-code-graph-upgrade-consensus-evidence.json)). native 선택·제품 VCS·독립 Git 코드까지 고정 b08b 소스2,087파일/137,696노드/146,976간선·누락/끊어진간선0을 재현했다. Go테스트799·내장AST41/동적25·일반변조12/특정누락178거절·과거38그래프 보존([최근 측정](all-code-graph-final-preflight-evidence.json)). 고정 설치셸의 구문 누락1RED→3Go계약·4선택계약·전체 검사, 실제5SSH회귀/15대보존과 독립 검증을 통과했다. aadd 소스2,081파일/137,178노드/146,413간선·누락/끊어진간선0을 재현했다. Bash63·내장AST41/동적25·일반변조12/특정누락104거절·과거34그래프보존([현재 측정](all-code-graph-web-preflight-refresh-evidence.json)). 표준입력 뒤 인자 경계7RED→21계약·전체 검사와 스윕 보강 고정소스2,075파일/136,638노드/145,824간선·누락/끊어진 간선0을 확인했다. 내장AST44·동적입력31, 일반변조12·선언/스윕본문 누락116·SSH본문 누락4거절과 과거26그래프 보존을 확인했다([현재 측정](all-code-graph-sweep-round-refresh-evidence.json)). 그룹 배치 검사 뒤2,074파일·136,270노드/145,437간선·누락/끊어진 간선0,15계약·일반변조12/특정누락28거절과 과거22그래프 보존을 확인했다([이전 배치 측정](all-code-graph-placement-refresh-evidence.json)). 방화벽 보강 뒤2,073파일·136,230노드/145,394간선·누락/끊어진 간선0, 파서15계약·일반변조12/특정누락22거절과 과거18그래프 보존을 확인했다([이전 방화벽 측정](all-code-graph-firewall-refresh-evidence.json)). 중단 경계 뒤2,070파일·136,003노드/145,142간선·두 그래프 일치/누락0, 추가 함수1/호출2/회귀8과 특정누락3종·일반변조12거절·과거그래프14보존도 확인했다([이전 경계 측정](all-code-graph-sweep-refresh-evidence.json)). 비교/관측 보강 뒤 새 고정 소스의2,070파일·Go 테스트793파일·135,880노드/145,019간선, 두 그래프 일치·누락/끊어진 간선0과 변조12거절·과거 그래프10보존을 확인했다([이전 관측 측정](all-code-graph-probe-refresh-evidence.json)). 명시한 구문·재평가 범위 유지. 후속 표준입력 보강은15계약/7개 RED·전체 검사와2,069파일·14언어/형식·내장 AST38개·미해석36개, 두 capture의 해시 일치와독립 거절12개를 확인했다. Docker 캡처·삭제·중첩 삭제3프로그램 포함을 별도 확인했고 누락/끊어진 간선0이다([증거](all-code-graph-stdin-evidence.json)). 기존 N3/A7b 판단과 [이전 고정 소스](all-code-graph-evidence.json)를 보존했다. 2026-10-16 후속 고정 소스는 14언어/형식 2,060파일·Go 테스트790파일·내장 AST36개, 미해석 동적 입력33개를 명시하고 누락/끊어진 간선0을 확인했다. A8 비메서드 공개심볼1,890개/도달 불가0. 10계약·관련 race·전체 검사, 두 별도 capture의 해시 일치와 별도 감사10실패 대조 통과. 전체 실행 커버리지나 최종 Web 인수는 아님; [갱신 증거](all-code-graph-refresh-evidence.json) |
| 14 | go-wemix W3 | 최신 외부 소스·바이너리 상태와 실제 재현 확인; 외부 체인 수정 없이 재현 자료·인계 갱신; 해결 없으면 열린 상태 보존 | 재현·인계 완료, 결함 미해결 — 원격 HEAD `4538a9f5`를 별도 빌드해 실제 verifier 거절 2회와 연결된 full-sync downloader 거절을 race로 확인했다. 관측·로그 검증 RED→GREEN; 준비 오류도 보존. 종료 DB 0개·746파일 내용/권한 보존. 과거 15노드 경합·재시작 복구·최종 Web 인수는 이번 범위 아님; [증거](wemix-w3-evidence.json). 외부 수정 전 제품 항목은 열림 |

**2026-10-18 WAL 감사 경로와 시간 제한 보강:**
압축입력과감사경로가같거나정규화별칭/하드링크인경우복사전에거절한다. 파일을연뒤에도원본·압축임시파일·
다른파일디스크립터·늦게바뀐심볼릭링크를대조해교체전에거절한다. 기존감사접두부와원본파일은유지한다.
초기7실패·커밋기준3디스크립터실패·늦은링크1실패를확인하고정리27/수명주기32/캐시48검사를통과했다.
프로세스·열린파일·메타데이터명령은30초,압축은60초로제한하며시간초과시원본과감사를보존한다.
소유한41MiB합성WAL모양파일에실제macOS압축을적용해42,627,072allocated bytes를줄였고내용/권한/
소유자/그룹/mtime/xattr/ACL·기존감사기록과10변조거절을독립확인했다. 실제체인WAL회수나전체스토리지증가
해결로확대하지않는다. 초기준비실패와첫통합검증도보존했다. 같은실행코드의전체AST갱신·두그래프일치·703키/766발생·누락1,406거절과과거70그래프보존을확인했다. 전체make check도통과했다. 반복스토리지증가정책·과거receipt·최종live/Web인수는별도잔여다.
[계약·실제합성파일압축·독립대조](all-code-graph-wal-audit-evidence.json).

**2026-10-18 준비·triecache 보강 뒤 전체 AST 재측정:**
고정1125 소스2,093파일·14언어/형식·138,516노드/147,855간선을 두 번 파싱해 일치를 확인했다.
소유2,537/제외444·누락/끊어진간선0, Go테스트803·내장AST41/미해석동적입력25다.
Go/Python 함수·호출과공개DSL27구문의714키·755발생을별도AST로대조하고각그래프의누락1,428건을거절했다.
일반변조12거절·과거66그래프의내용/권한/소유자/mtime보존·A8공개비메서드1,891/도달불가0도확인했다.
검사중저장소파일을수정하지않았고,게시메타데이터와정본상태는모든측정종료뒤갱신했다.
아래세체인동기화증명은별도고정소스의동일실행코드범위이며전체동적호출/legacy/L1/Web인수는아니다.
[전체구문·독립대조·누락거절](all-code-graph-legacy-sync-evidence.json).

**2026-10-18 legacy 동기화의 준비 조건·종료 triecache 회수:**
공개CT-NODE-009는 각 체크포인트에서5노드 높이를 한 번씩 읽고최솟값 대비차이≤2를 모두 확인한다.
동일과거해시를 가진 지연노드14개 잘못된통과를 RED로 확인했고18대조를 통과했다.
첫 실제시험의StableNet/WBFT PASS·WEMIX 초기RPC BLOCKED는 보존했다. 준비조건 단위검사와관련race 뒤,
세체인의실제5노드검증자 집합을 제한시간 안에 확인하고 준비된workspace에읽기전용으로붙여
동일공개케이스를실행했다.15노드·39단언과별도18변조거절·증명보존·종료DB0을확인했다.
WEMIX는초기RPC오류16건을보존하고17번째관찰라운드에서전노드준비를확인했으며최종높이는57이었다.
W3해결·cold-start시간·전체legacy/L1/Web인수는아니다. 종료triecache누락도2RED→수명주기32검사와
정리23/Go캐시48회귀로보강했다. 기존실패시험의5캐시2,945,024allocated bytes를회수하고563증명파일의
내용/권한/소유자/mtime/xattr/ACL과기존감사기록을보존했다. 전체스토리지증가해결은별도잔여다.
[실제증명·초기실패·회수와독립대조](legacy-sync-spread-evidence.json).

**2026-10-18 legacy 높이 차이 보강 뒤 전체 AST 재측정:**
고정7e13 작업소스의2,093파일·14언어/형식·138,389노드/147,725간선을 두 번 파싱해 같은 지문을 확인했다.
소유2,536/제외443·누락/끊어진간선0, Go테스트803·내장AST41/미해석 동적입력25다.
파서21계약·일반변조12거절과 새 Go 함수/호출·공개DSL27구문의86키/112발생에 대한 누락172거절을 확인했다.
과거62그래프의 bytes/권한/소유자/mtime과 A8 공개비메서드1,891/도달불가0을 보존했다.
같은 과거 해시를 가진 지연 노드의 잘못된 통과14건을 실제 공개케이스 실행으로 확인하고18대조를 통과시켰지만,
이 측정 당시 세 체인 실제 높이 차이 증명은 대기 중이었다. 위 준비 조건 보강 뒤 별도 실제증명을 완료했으며 전체legacy/L1/race/e2e/Web 인수는 남아 있다.
[소스 동결·전체 구문·독립 감사·누락 거절](all-code-graph-legacy-sync-evidence.json).

**2026-10-18 완료 캐시·CI 보강 뒤 전체 AST 재측정:**
고정3541 작업소스의2,091파일·14언어/형식·138,254노드/147,569간선을 두 번 파싱해 같은 지문을 확인했다.
소유2,532/제외441·누락/끊어진간선0, Go테스트801·내장AST41/미해석 동적입력25다.
파서21계약·일반변조12거절과 Python/Make의193키·203구문 발생에 대한 누락386거절을 확인했다.
같은 줄의 중복 호출도 Python AST의 중복 수로 대조했고 기존 그래프는 거절·새 선언 검사는 통과했다.
과거50그래프의 bytes/권한/소유자/mtime과 A8 공개비메서드1,891/도달불가0을 보존했다.
문서 검사는 갱신 전13번 보류를 거절했으며 테스트를 약하게 바꾸지 않고 실제 측정을 끝낸 뒤 완료로 돌렸다.
게시 메타데이터를 제외한 현재 코드의 구문 측정이며 전체 동적 호출·런타임 커버리지나 후속Web14·전체L1/race/e2e는 아니다.
[고정소스·원본 보존·독립 감사·구문 누락 거절](all-code-graph-completed-cache-evidence.json).

**2026-10-18 완료 소스의 Go 캐시 보강과 저장 공간 경계:**
고정8689 소스의 WEB01 capture·독립 재현과 WEB02 capture는 통과했다. 다음 재현은20GiB 미만에서
시작하지 않았고,54항목 원본과 메타데이터를 보존했다. 호환 정리 명령은 과거 증명3파일 부재로 거절됐으며
성공으로 세거나 참조를 제거해 우회하지 않았다. 기존24계약 뒤 오래된 캐시1,104개/446,230,528bytes를 회수했다.
종료된 소유 런타임의 같은 native복사90개도 APFS 공유 복사로 전환해 파일내용/권한/소유자/mtime/xattr/ACL을
확인했지만 즉시 여유 증가가3,047,424bytes뿐이었다. 논리중복5.3GiB를 회수로 세지 않는다. inode/ctime 변경과
실제 폴더mtime의 독립 비교 부재도 기록했다. 기존 WEB01/02·별도 WEB01 검증은 유지됐다.
최근 캐시의 완료 소스 연관을 명시적으로 검증하는 opt-in을18RED→48계약으로 보강했다.
기본8시간 보존·활성/열린파일·미참조·4GiB상한/1GiB최소보존은 유지하며 종료 기록·소스 경로·메타데이터를
삭제 직전에 다시 확인한다. 가짜 실패 재현의 실제 디스크/Git 의존도 분리하고, CI누락1RED 뒤
정리23/수명주기28검사를 기본 목록에 추가했다. 모의19GiB에서도28검사가 통과하며 실제 저용량 거절 검사는 유지한다.
최종 전체make check 뒤 실제2,011캐시/1,015,865,344bytes(완료 소스1,958·기존오래된53)를 회수했다.
별도 감사에서 남긴17,857파일과14소스 사본/파일메타데이터를 대조하고13변조를 거절했다.
두 캐시 회수의 할당bytes 합계는1,462,095,872이며 다른 볼륨 여유 변화는 그 결과로 과장하지 않는다.
증명 사이에 새 옵션을 적용해 완료 소스 캐시의 누적을 줄인다. 현재 소스의 전체AST와 Web 인수는 다시 갱신한다.
과거receipt27·전체L1/race/e2e와1·9·10·11·14번은 열려 있다. Kevin Mac의22번 포트 접근도 거절돼 혼합 빌드 출처는 미확인이다.
[정책·RED/GREEN·실제 회수·감사·보존과 한계](completed-go-cache-evidence.json).

**2026-10-18 Web 인자 보강 뒤 전체 AST 재측정:**
고정0b8f 소스의2,091코드파일·14언어/형식·137,906노드/147,219간선을 두 번 수집해 같은 지문을 확인했다.
소유2,530/제외439·누락/끊어진간선0, Go테스트801·내장AST41/미해석 동적입력25이며
A8 공개비메서드1,891개 중 도달불가0이다. 파서21계약·일반변조12거절과 선언/호출72개의
특정누락144거절, 기존 그래프 거절→새 그래프 선언 검사 통과를 확인했다. 변경6케이스의 파싱/지문도 같다.
과거46그래프의 bytes/권한/소유자/mtime을 보존했다. 이는 구문 측정이며 전체 동적 호출·실행 커버리지나
후속Web14·전체L1/race/e2e를 닫지 않는다. 게시 메타데이터는 코드 분류에서 명시적으로 제외한다.
[고정소스·두 그래프·독립 감사·누락 거절](all-code-graph-web05-arguments-evidence.json).

**2026-10-18 Web 인자15개 실행 누락과 키 원본 보존 보강:**
07e/c5f2 소스의 WEB01~04 capture·독립 재현은 통과했지만 WEB05는 실제 작업 성공 뒤에도
438인자 중423개만 실행돼 미완료였다. 원본377항목을 보존하고, 현행 계약을 읽는 CI 검사에서
동일9등록/15누락을 RED로 재현했다. 대납 배포·raw 변조/원본 성공·정규식 구간·완전 로그·metric 대안을 보강했다.
주소 기준표의 추가21위치도 RED와 차이 검토 뒤 갱신했다. 첫 실제시도는 새 배포 대납자의 수수료 예약
10.14ETH가 잔액10ETH를 넘어 실패했다. 예산100ETH와 명시 수수료 상한을 적용했다.
같은 실행에서 코퍼스 주소 단위 fixture가 키20파일을 같은 bytes로 다시 써 mtime을 바꿨다.
별도20RED 뒤 Register의 대상을 임시 저장소로 바꿨으며 원본 시간을 되돌리지 않았다.
최종 전체 make check·관련5패키지 race와 고정c569 실제 WEB05 capture가 통과했다.
실제14케이스·438/438 편집/실행·238코퍼스 import, 독립 자료 감사23변조 거절과 원본 메타데이터 불변·종료DB/프로세스0이다.
처음 감사의 단언 전용 steps 파일 가정과 동작 expect 인자 이중 계수 오류도 보존·정정했다.
통과 원본240항목·실패 원본28항목의 내용/권한/소유자/mtime/xattr/ACL도 보존했다.
이는 WEB05 한 기준이며 독립 브라우저 재현·후속 고정 소스의 전체Web14·전체L1/race/e2e를 완료하지 않는다.
실제 accessList wire 값의 별도 조회와 live 로그 구간 증명도 이번 감사 범위가 아니다.
1·9·10·11·14번은 계속 열려 있고 AST는 다음 단계에서 갱신한다.
[RED/GREEN·실제 실행·감사·실패와 보존](web05-argument-coverage-evidence.json).

**2026-10-18 포크 합의 보강 뒤 전체 AST 재측정:**
고정 b492 소스에서2,090코드파일·14언어/형식·137,815노드/147,121간선을 두 번 수집해 같은 지문을 확인했다.
소유2,527경로/제외437·누락/끊어진간선0, Go테스트800·내장AST41/동적입력25이며
A8 공개비메서드1,891개 중 도달불가0이다. 파서21계약·일반변조12거절과 새 선언/메서드/호출45개의
특정누락90거절을 확인했고 과거42그래프의 bytes/권한/소유자/mtime도 같다.
첫 보조 감사는 같은 줄의 서로 다른 anonymous.ID 호출 두 개를 하나로 가정해 실패했다.
원본 감사와 그래프를 보존하고 Go AST의 시작/끝 위치까지 대조하도록 정정했다. 두 Go AST 입력은 지문도 같다.
이는 구문 구조의 측정이며 모든 동적 호출·실행 커버리지·전체 L1·후속 Web 인수를 완료하지 않는다.
[고정 소스·독립 감사·특정 누락 거절·최초 감사 오류](all-code-graph-upgrade-consensus-evidence.json).

**2026-10-18 실제 Web 포크 전 합의 관문 보강:**
고정 e9e/ef35 소스의 WEB-01~04는 capture와 별도 Git/소스 사본의 독립 재현이 모두 통과했다.
WEB-05는 첫 여섯 케이스 뒤 포크 전 합의 관문에서 BLOCKED였으므로 전체 인수로 세지 않는다.
네 WBFT 후속 바이너리는 아직 PoA 블록을 읽지만 관문은 istanbul 조회를 요구했고, 모두 -32000이었다.
선언된 concurrent 포크·실제 높이와 후속 체인의 명시적 조회 계약으로 활성 합의를 선택하도록 보강했다.
WBFT의 포크 전 정보 메서드는 기존 WEMIX의 admin_wemixInfo가 아닌 eth_wemixInfo이며,
오류 뒤 다른 합의로 재시도하거나 지원 계약이 없는 조합을 통과시키지 않는다.
최초8RED·방언/계약 누락4RED 뒤14시나리오·관련7패키지 전체 race·전체 make check를 통과했다.
새 고정 작업 소스5dfa의 실제 Web 포크 구간에서 네 후속 노드는 높이23의 PoA 검증자1명을 확인했고,
높이27의 이체가 포크120 뒤 높이122·WBFT 검증자4명·잔액 보존으로 이어졌다. 실제3단언·독립22변조 거절과
종료 프로세스/DB0을 확인했다. 첫 감사의 saveKey 비공개화 비교 오류와 정정도 보존했다.
이는 포크 구간 수정의 증명이며 전체 WEB-05·현재 후속 소스의 WEB14·전체 L1/e2e를 완료하지 않는다.
e9e 한정 Web4/14를 후속 소스에 재사용하지 않으며 AST도 다시 측정한다. 1·9·10·11·14번은 계속 열려 있다.
[이전 고정 소스의 부분 인수](web-frozen-partial-evidence.json),
[RED/GREEN·실제 포크·독립 감사](upgrade-consensus-phase-evidence.json).

**2026-10-18 기존 증명 원본 보존과 추가 캐시 회수:**
기존 인수504항목·WEB-02 이후 개발 증명1,290항목·새 부분 인수168항목은 비활성을 확인하고
같은 디스크의 별도 위치로 원본을 이동했다. 내용·종류·권한·소유자·mtime·xattr·ACL이 같다.
tar 복원의 그룹 변경과 macOS provenance 추가는 실패로 남기고, 그 복원본을 보존 성공으로 세지 않았다.
옛 WEB-02 영수증3개는 첫 인수 원본에 남아 있다. 개발 폴더 보존은 WEB-02 재실행 뒤의 스냅숏이며,
그 전에 덮인 보조 개발 파일까지 불변이라고 주장하지 않는다. 과거 receipt27개 미발견은 여전히 남는다.
보존 계약24개 뒤 오래된 작업 Go 캐시15,511파일·3,939,880,960할당bytes를 회수했고,
같은 실행에서 남긴11,019파일의 bytes/권한/소유자/mtime을 재대조했다. 소스·증명은 삭제하지 않았다.
이후 정상 빌드의 캐시 접근/재생성까지 불변이라는 뜻은 아니다. 당시 여유 공간은25GiB였다.
[보존 원본·실패한 복원·캐시 감사](web-proof-preservation-cache-evidence.json).

**2026-10-18 인수 준비 코드 전체 AST 재측정:**
native 선택·제품 Go VCS 검증·현재 CLI 캡처·독립 Git 경계까지 고정
b08b674912c4beaccd7221e4783a37853912693ee501d230c843dce3284a4858에서 전체 구문 그래프를 두 번 수집했다.
2,087코드파일·14언어/형식(Go1,406/Python131/Go테스트799)·137,696노드/146,976간선이며 지문이 같다.
소유2,520경로/제외433·누락/끊어진간선0, 내장AST41·미해석 동적입력25, A8 공개비메서드1,890/도달불가0이다.
파서21계약·일반변조12거절과 새 선언·연결·설치셸 함수89개의 누락178거절을 확인했다.
첫 보조 감사는 문자열 속 Go 프로그램의 main을 실제 선언으로 잘못 센 실패이며 보존했다.
보조 검사도 정규식 대신 Go AST 선언으로 정정한 뒤 통과했고 과거38그래프의 bytes/권한/소유자/mtime도 같다.
이는 모든 동적 호출·실행 커버리지·전체 L1·최종 Web 인수를 주장하지 않는다.
[고정 소스·첫 감사 오류·정정·독립 그래프·누락 거절](all-code-graph-final-preflight-evidence.json).

**2026-10-18 독립 복사본의 전용 Git 경계:**
Git 없는 복사본에서 추적 파일 검사가 실행되지 않는 2개 RED를 확인했다. 전용 refs/config/index와
읽기용 원본 object alternates를 준비하고 원본 작업 내용·미추적 파일을 보존하도록 보강했다.
분리 인덱스의 원본 공유 파일 mtime이 조회로 갱신되는 추가 실패도 확인해, 필요한 공유 base 조회는
복사본에서만 수행한다. 원본 시간을 복원해 실패를 숨기지 않는다. 상속 Git 경로 override와 private hook도 차단한다.
6계약·기존 runtime6/사전검사7/native선택15/Web14의11회귀·전체 make check가 통과했다.
고정 d8b6aadee6d47483837f783b8156543fdb843011ddc86c3e31318c379cd1b56f의 실제 전체 복사본에서
추적 경로2,476개와 Git 의존 Go 문서 검사2개 PASS/skip0을 확인했다. 원본 Git2,180파일·소스·미추적 입력과
증거를 독립 대조했고 변조12개를 거절했다. 종료 후 재생성 가능한 의존성 복사본 약49MiB만 회수했다.
이것은 실제 Web/browser 독립 재현이나 전체 race 통과를 대신하지 않는다. 후속 AST도 갱신했다([측정](all-code-graph-final-preflight-evidence.json)). 전체 실제 인수는 남긴다.
[추가 실패·경계 계약·원본 보존·실제 검사·독립 감사](independent-git-view-evidence.json).

**2026-10-18 제품 바이너리 출처 검증과 현재 CLI 캡처 갱신:**
Go VCS 출처가 유효한 native 형식의 unit fixture도 매니페스트 검증·업로드에서 거절되던 2개 RED를 확인했다.
두 제품 경로가 같은 출처 추출을 사용하도록 보강했다. 기존 유효한 버전 stamp는 유지하고, stamp가 없으면
Go 빌드 정보의 정확한 Git 커밋·소스 미변경을 요구한다. 모호한 stamp·변경/누락된 출처는 계속 거절한다.
검사 프로그램은 노드 실행을 구현하지 않은 unit fixture이며 실제 체인 증명과 구분한다.
관련 core/app/dashboard/MCP 네 패키지 전체 race와 전체 make check가 통과했다. 문서 섹션 합계 누락도 정정했다.
첫 실제 세 바이너리의 옵션 표면 검사는 세 문서 커밋이 오래돼 stale로 실패했으며 실패를 보존했다.
같은 고정 바이너리로 CLI 표면 80/78/58개·서로 다른 플래그 179/177/195개를 다시 캡처했다.
기존 숨김·거절 플래그 설명은 누락/잘못된 설명 0개로 대조했다. 코드 줄 번호와 RPC/metrics 분석은
과거 source snapshot으로 명시했으며 현재 코드 분석을 완료했다고 바꾸지 않았다. 원본 캡처도 보존했다.
갱신 후 고정 4e5de322903e576ed2f86c424821877d2d92df50a2e2048949715fe7d92bc6e4에서
세 native 명령/옵션 표면 race가 PASS했고, 같은 소스의 실제 API 업로드 3개·커밋/체크섬·재시작 보존도 통과했다.
독립 payload/출처 검증과 변조 11개 거절, 종료 DB 0개를 확인했다. 전체 L1·전체 race·Web14 인수는 아니다.
독립 복사본의 Git 경계와 후속 소스 AST·전체 최신 receipt 회귀는 남긴다.
[RED/GREEN·첫 stale 실패·실제 제품·CLI 표면·독립 증거](native-product-source-identity-evidence.json).

**2026-10-18 Web 실제 바이너리 선택 경로 일치·VCS 출처 확인:**
Web04/공유 증명이선택manifest를 무시하던5RED와 유효한Go VCS 메타데이터의 미지원1RED를 확인했다.
Web01과 같은선택경로로 묶고15계약·사전검사7/Web14의11회귀·전체make check를 통과했다.
첫 실제시도의WEMIX 거절은 보존했다. 내장버전에는Git Commit 줄이 없지만Go 빌드 정보의4538a9f5와
소스미변경·패밀리표시를 확인했으며 외부소스/바이너리를 수정하거나 다시빌드하지 않았다.
새고정c087d6c6ee5f9f8da5c45caef8df6c4bba15107f2e3063f7dfd69f0942aa98bb에서
세핀바이너리의 실제version/help·선택/복사와 복사전거절3건, 외부소스6,508항목 보존·독립변조14거절을 확인했다.
관측 native 명령은version/help뿐이며 노드/DB생성0이다. 실제Web 자산업로드와ManifestBinary.Verify는
아직이전Git Commit 줄만요구하는것으로 소스대조됐으므로 별도RED→GREEN·실제제품증명이 남는다.
독립복사본의Git 부재로 tracked-tree검사가skip될수있는경로도 재현·보강해야 한다.
이후수정의AST갱신·전체L1·Python최신receipt·전체race·Web14 인수는 남긴다.
[첫실패·선택계약·외부원형·실제준비·잔여경계](native-fixture-selection-evidence.json).

**2026-10-18 준비 코드 갱신·설치 본문의 독립 셸 AST 포함:**
첫 구조 측정2,080파일은 재현됐지만Go wrapper의 고정 설치본문이 Bash로 파싱되지 않아 이번 전체 언어 기준을 닫지 않았다.
독립 셸 입력 계약1RED를 확인하고 같은본문의`install.sh`를Go embed로 포함하도록 고쳤다.
Go3계약race·선택4계약(10경우)·전체make check·실제SSH5회귀와15대 원본 자료 보존/독립 변조11거절이 통과했다.
고정aaddd83b8a8b37195c417c8ca7021b7dbbc08128929026f0aeb7da1e3a94ef3e에서
2,081파일·14언어/형식·137,178노드/146,413간선, 소유2,510경로/제외429·누락/끊어진간선0을 확인했다.
Go테스트798파일·Bash63파일이며 설치본문의 원래문자열 해시15947023도 별도Bash AST로 확인했다.
내장AST41·동적입력25는 보수적 구문 분석 범위다. 모든 wrapper·동적 호출 해석을 주장하지 않는다.
두 전체그래프/A8 지문 일치·공개 비메서드1,890/도달불가0, 일반변조12·새 선언/호출/셸 함수 누락104거절,
과거34그래프 bytes/권한/소유자/mtime 보존을 확인했다. 전체L1·Web 인수는 여전히 미완료다.
[첫 누락·실제 회귀·소스 고정·AST·독립 감사](all-code-graph-web-preflight-refresh-evidence.json).

**2026-10-18 Web 사전 빌드 소스 고정·일반 검사 환경 격리:**
기존SPA 재작성·native 게이트 상속·전체race 캐시 재사용을 기존 실행 경로의 확장4RED로 확인했다.
공유 사전 검사7계약·Web14의11계약·기존runtime6계약과 전체make check를 통과했다.
WEB01/03/04/05/10과 공유 인수의SPA 빌드는 임시 복사본에서 검토된 embedded bytes를 재현해야 하며,
다르면 저장소를 고치는 대신 거절한다. 일반Go 검사는Git 정보가 있는 원 checkout에서 실행하고,
native/Docker/e2e 게이트10개를 제거한다. 실제 native 증명의 자식 환경은 이 경계 밖에서 유지한다.
필수Web14 전체race는`-count=1`로 캐시 재사용을 금지하고 기존 직렬·3시간 상한을 유지한다.
실제SPA3파일 재현·Go 로컬맵4검사·실제 자식 환경 격리와2,458소스 파일의 bytes/권한/소유자/mtime 보존,
임시복사본0잔여를 독립 확인했고 변조12개를 거절했다. 전체race·전체Python·L1·Web14 인수는 아니다.
[소스 고정·RED/GREEN·실제 검사·독립 증거](web-prerequisite-isolation-evidence.json).

**2026-10-18 Docker 회귀 준비 대상·기존 자료 보존 보강:**
고정 컨테이너·사용자를 무시하던10개 RED를 확인해4계약(10경우)·전체make check와 관련3패키지race를 통과했다.
실제 서버 설정/주소 매핑 로더로server1을 해석하고 compose 출처·실행 ID·주소·SSH 포트를 변경 전에 대조한다.
기존 준비 파일은 bytes/유형/권한/소유자가 다르면 덮어쓰지 않고 거절하며 없는 파일만 설치한다.
첫 실제 시도의5회귀 통과·준비/반복은 관찰했지만 마지막 주변 자료 보존 판정이 실패해 전체 통과로 세지 않는다.
그 첫 비교의 메모리상 원본이 저장되지 않아 차이는 재구성할 수 없으며 원인 해결도 주장하지 않는다.
새 baseline을 먼저 저장한 진단은15대 전후 일치였고, 별도 재시도에서 소유 sentinel 거절·원본 복구,
준비·반복·5개 실제SSH race,15대 원본/후속 자료의 완전 일치와 독립 변조11거절을 확인했다.
종료DB/임시stage0이다. 전체L1·Web14 인수와 후속 소스AST 갱신은 남긴다.
[범위·첫 실패·재시도·독립 증거](docker-regression-preparation-evidence.json).

**2026-10-18 최종 Web 실행 전 검사 보강 — 최종 인수는 미완료:**
W4 명시 허용값 누락과 기본10분 제한을6개 RED로 확인하고11계약·전체make check를 통과했다.
직렬 전체race는 명시3시간 상한을 사용하고 W4는 정확한`1`을 요구한다. 일반 검사에는 native/W4 환경을 전달하지 않는다.
출처가 확인된 실제 바이너리·소유 Docker 설정 경로에서 누락5/잘못된값3을 준비 이전에 거절했다.
자식 실행0·출력폴더0과 소스/입력 보존, 독립변조7거절을 확인했으며 실제 블록체인/race 성공을 주장하지 않는다.
전체Python149검사는9실패·2오류다. 이전 생산모듈의145검사에서도 동일11건이어서 새 회귀로 분류하지 않지만,
과거 receipt/실행파일 부족을 완화하거나 전체 GREEN으로 표시하지 않는다. 새 capture와 전체 재검사는 남긴다.
[검사 범위·실패·독립 증거](web-final-preflight-evidence.json).

**2026-10-18 스윕 갱신과 표준입력 프로그램 인자 경계 — 전체 AST 재측정:**
`python3 - "$arg"`의 고정 본문을 동적으로 분류하고 `-` 뒤의 `-c` 데이터도 옵션으로 오해하던7개 RED를 확인했다.
표준입력 표시 뒤는 프로그램 인자로 구분해21파서 계약·전체make check를 통과했다. 표시 앞의 알 수 없는 옵션,
스크립트 파일의 stdin 데이터와 명시한 명령의 우선순위는 계속 구분한다. 분석 과정에서 본문을 실행하지 않는다.
소스369a135548272f467ecb920c15c3a7a3c69187f485bf841c64c1c79a5b222963에서
2,075파일·14언어/형식(Go1,402·Go테스트797)·136,638노드/145,824간선을 두 번 수집해 지문 일치를 확인했다.
소유2,500경로 중 제외425를 명시했고 누락/끊어진 간선0, 내장AST44·미해석 동적 입력31이다.
스윕의 고정Python4본문/5함수와 SSH준비2본문이 새로 포함됐다. 일반 변조12개와 선언54개/스윕본문4개의 누락116개,
SSH본문 누락4개를 거절했고 소스/고정 도구/현재 그래프와 과거26그래프의 bytes/메타데이터 보존을 확인했다.
A8 비메서드 공개심볼1,890개/도달 불가0이며 메서드는 이 규칙의 측정 대상이 아니다.
전체 동적 평가/실행 커버리지·전체L1·다른 잔여·최종Web 인수를 뜻하지 않는다.
[파서 RED/GREEN·네 그래프·독립 감사와 누락 대조](all-code-graph-sweep-round-refresh-evidence.json).

**2026-10-18 공통 round의 실제 체인과 attach 결과 경로 보강:**
기본StableNet round가 WBFT/WEMIX 원본을 그대로 실행하던 문제와 attach의 결과 경로/no-skips 누락을
14개 단위 RED로 확인하고16계약 GREEN, 기존 종료/보존19계약·관련DSL3패키지 race·전체make check를 통과했다.
공통 round는 모두 같은 형태의 대상 프리셋을 명시하며 파일 이름 대신 상속을 포함한 실제 chain 선언을 확인한다.
대상 누락/오류/순환/다른 체인은 해당 실행 전에 거절한다. attach에는 관측 결과 경로와 엄격 판정만 전달한다.
소스34b94a01eb91e60184113ce05d1d12c7dd8b981b34456de66a4f43acb4d789aa,
CLI210a77d7dca5c6c23f460fa0f68fd4d1a0c4b67b5d7565186ed6cef0830ca23f에서
공개RPC-016의 StableNet/WEMIX/WBFT와 소유한 망의 attach가4PASS·86단언/47동작으로 통과했다.
준비 host를 포함한5개 실제 호출과4세션의 정의서/단언/상태/체인/실제plan/결과 경로를 독립 대조했다.
체인/계획 변조8종과 호출/세션 변조32종을 거절했고 attach 세션은 지정한 소유 runtime 안에 저장됐다.
20보관/1,035항목·로컬624파일·경계60회·새 프로세스/FD30회·DB/분리규칙0과 소스/고정 입력 보존도 독립 확인했다.
첫 임시 선택기의 정규식 오류는0실행·0DB·41파일 보존으로 남겼고 기능RED로 세지 않는다.
새 이름/환경에서 실제 선택2파일을 사전 대조한 뒤 같은 소스/CLI로 재검증했다. 이전 실패 자료는 덮어쓰지 않았다.
전체216파일/358계획의 읽기 전용 대조에서는 공통213행의 해석된 체인 불일치0을 확인했지만 전체 실제 실행은 아니다.
이번 스크립트/회귀의 AST 갱신·전체L1·다른R3/P4/R4/R5·legacy·최종Web은 별도 잔여다.
[RED/GREEN·선택 오류 보존·실제4회·호출/세션/보존 독립 대조](sweep-family-output-evidence.json).

**2026-10-18 현행 L1 79건과 실제 체인 대조 — 전체 인수 거절:**
소스4e0f3eb3572c742f306afef989463d9fb23265dae858f25c0b2da89d8f7a11fb·HEAD66cade7d에서
216파일/358계획 중79PASS·279NOTRUN, 종료130으로 정상 경계 중단했다.
실행 중 저장소/고정 도구는 바뀌지 않았고353보관/20,414항목·로컬10,736파일·경계1,185회·새 프로세스/FD30회,
남은DB/분리규칙0과 공유 키/바이너리 보존을 독립 확인했다. 이 감사의 변형 누락0은 계획의 문자열 기준이다.
추가80개 실제run 호출과79개 세션/정의서/판정/환경/망 기록을 대조한 실제 체인은 StableNet72/WBFT1/WEMIX6이다.
공통RPC-016은 원래wbft-bp4-en1인데 기본StableNet round가 override를 생략해53번을 WBFT로 실행했다.
79기능PASS는 유지하지만 의도한 세 체인 전수로 세지 않는다. 앞서73건을 StableNet71+기본2로 설명한 중간 집계도 정정한다.
attach1건은 artifact-root/no-skips를 전달하지 않아 기본 사용자 세션 경로로 결과를 썼다.
해당11파일은 원본을 지우지 않고 별도PAX 보관의 bytes/권한/uid/gid/mtimeNS와 대조했으며 원본 불변을 확인했다.
추가 판독기의 첫 chain필드 가정 오류는 옛attach 환경에 해당 필드가 없어서 난 것이고 기능RED로 세지 않는다.
소유한host 망 기록과 실제chain show의 dial RPC를 attach 엔드포인트와 정확히 대조한 뒤 감사를 마쳤다.
계획 생성/실제 체인 검증·attach 결과 경로 전달의 RED→GREEN·새 고정 소스 실제 증명·전체L1·최종Web은 남는다.
[원시 결과·독립 보존·실제 호출 대조·경로 이탈 보존](linux-l1-round-boundary-evidence.json).

**2026-10-18 그룹 배치 검사 뒤 전체 코드 AST 갱신:**
소스f5ccbdcab71d6c364c8cf7608321facebe5c771e94ebb07d17dd3b39ed0637ff에서
2,074파일·14언어/형식(Go1,402·Go테스트797)·136,270노드/145,437간선을 두 번 수집해 지문 일치를 확인했다.
파일 누락/끊어진 간선0, 내장AST38개·동적 입력37개, A8 비메서드 공개심볼1,890개/도달 불가0이다.
파서15계약·일반변조12개와 방화벽3함수/회귀11함수의 특정누락28개를 거절했고, 과거22그래프의 bytes/메타데이터도 보존했다.
전체 실행/동적 프로그램 커버리지·전체L1·최종Web 인수는 별도 잔여다.
[네 그래프·현재 소스·독립 감사](all-code-graph-placement-refresh-evidence.json).

**2026-10-18 방화벽 쓰기 전 전체 그룹 배치 검사:**
뒤쪽 peer/self, 세 번째 그룹, 중복 노드의 공유 서버를8~24개 명령 뒤 거절하던8개 단위 RED를 확인했다.
모든 그룹의 기록된 호스트를 먼저 검사해 첫 쓰기 전에 거절한다. 같은 그룹 내부 공유는 허용하며 cut/restore의10시나리오 GREEN,
관련5패키지 전체 race·전체 make check를 통과했다. 소스bc0c59c39f26ad5aee76b3fe2c828612acc2f1ae6baca89085d6088fb88f1d64,
CLI425bd53c5edb0f146f7131e10c5c0349f3a68640405e7e48a7f365082d99a312에서 세 체인 각각4노드/3호스트를 실제 실행했다.
node1/node4는 같은 호스트의 다른 포트다. 같은 그룹에 둔3개는 분리·정지·복구·고정 높이 해시가 통과하고,
다른 그룹에 둔3개는 정확한 같은 호스트 사유로 거절되며 종료 전에도 방화벽 규칙이0이다.
원시 CLI/케이스는3PASS·의도한3FAIL이고6개 목적을 만족했다. 잘못된 정의를 정상 케이스 PASS로 바꾸지 않았다.
독립 감사는18보관/1,132항목·로컬783파일·경계90회·새 프로세스/FD30회,
종료 전후 방화벽180회·남은DB/규칙0·소스/고정 입력/공유 키/바이너리 보존을 확인했다.
호스트/종료코드/단언/관측 변조24개를 거절했고6개 실제 argv/체인/케이스 지문도 고정 계획과 대조했다.
처음 임시 설정 옆localmap 누락으로 genesis 단계에서 멈춘 자료(195항목/115파일·분리 미도달)와,
사람용 stdout에서 상세 오류를 찾던 도구가 실제 거절3개를 미완료로 분류한 배치(1,132항목/783파일)를 따로 보존했다.
도구를steps.json의 오류 판독으로 고친 뒤 같은 제품 소스/CLI의 새6개 배치를 통과했다. 잘못된 판정 플래그는 덮어쓰지 않았다.
이 둘은 기능 RED로 세지 않는다. AST 갱신·공개9노드 케이스의 새 소스 전수 회귀·전체L1·다른 잔여·최종Web은 남는다.
[단위 RED/GREEN·세 실제 배치·원시 실패 보존·독립 감사](partition-placement-evidence.json).

**2026-10-18 현행 L1 16건과 배치 거절 지연 — 범위 구분:**
소스96cc59d069e3ede3c28dd380f07e5a1c2ca921164634978515d428f995a3d206·HEAD53eed866,
CLIbd2a0c2b6f9ad8b5c42f2aff506c4fd7f7a048df2854bf98bb3de514ef245df7에서216파일/358계획을 고정했다.
공통71파일의 세 체인 변형과 단일 체인145파일의 누락/중복/오래된 지문을 사전 검사했고5실패 대조·격리10·보존37계약을 통과했다.
기본2·계약8·장애6이 실제 통과했고,9노드 방화벽 복구와 허브 연결·txpool 리더 교체도 포함한다.
전체 배치 중 별도 private Go overlay 재현에서 뒤쪽의 교차 그룹 공유 서버를 거절하기 전에8개 방화벽 명령을 실행함을 확인했다.
이 재현은 fake host 명령 캡처를 쓴 단위 RED이며 실제 망의 잘못된 분리로 세지 않는다. 저장소·고정 CLI는 변경하지 않았다.
현재 케이스의 정상 종료·보존을 기다려16PASS/342NOTRUN·종료130으로 멈췄다. 배치 전체 완료나 결함 해결이 아니다.
독립 감사는74보관/4,456항목·로컬2,274파일·경계240회·새 프로세스/FD30회,
소스/고정 입력/공유 키/바이너리 보존·남은DB/규칙0과216/358계획의 누락0을 확인했다.
사전 전체 배치 검증의 RED→GREEN·실제 증명·다음 고정 소스 전체L1·최종Web은 남는다.
[원본 결과·보존 감사·private 단위 재현](linux-l1-placement-boundary-evidence.json).

**2026-10-18 방화벽·판정 회귀 뒤 전체 코드 AST 갱신:**
소스f2deabf362c3d1a60f2bffe7471bc04b27537e87b2d182c198b6562fbb612fb6에서
2,073코드 파일·14언어/형식(Go1,401·Go테스트796)·136,230노드/145,394간선을 두 번 생성해 지문 일치를 확인했다.
파일 누락/끊어진 간선0, 내장AST38개·동적 입력37개를 명시했으며 과거18개 그래프의 bytes/메타데이터를 보존했다.
A8 비메서드 공개심볼1,890개/도달 불가0이다. 파서15계약과 독립 일반변조12개,
fwRun/fwRules 및 새 회귀9함수를 빼는22개 실패 대조도 통과했다. 파서 변경이나 새로운 기능 RED를 주장하지 않는다.
전체 실행/동적 프로그램 커버리지·전체L1·외부 결함 해결·최종Web 인수는 별도 잔여다.
[소스·네 그래프·독립 감사·회귀 포함](all-code-graph-firewall-refresh-evidence.json).

**2026-10-18 기록된 etcd 합의 포트까지 분리·복구:**
P2P TCP/UDP에 더해 기록된 etcd peer TCP의 네 방향 흐름을 차단·복구하도록 보강했다.
임의의 P2P+1 추정은 하지 않고0인 미사용 포트, localhost client, RPC/WS/auth/metric은 차단 대상에서 뺀다.
양쪽/한쪽/미사용 포트와 복구의4개 RED→5시나리오 GREEN, 기존 방화벽 회귀·관련5패키지 전체 race·전체 make check를 통과했다.
첫 실제 배치는2PASS/1BLOCKED였다. WEMIX join-node8이 node9 거버넌스 구성원을2분 내 찾지 못해 분리에 도달하지 못했다.
같은 소스6d6f0b984bb1615c2bbd70abaadacfa44c323be1d159d0043ecd0870962c99f3·같은 CLI bytes에서
관측을9노드 시작 완료 뒤로 미룬 새 배치가3PASS했다. 시작 실패의 원인·관측 부하와의 인과·외부 결함 해결은 미확정이다.
WEMIX 소수4대는136→136, 과반은238→265였다. 세 체인 모두 복구 뒤9대의 고정 높이 해시가 같다.
실제 TCP 관측262회에서 교차 그룹80연결(P2P/etcd 각40방향)의 정상→차단→복구와 내부8대조 연결 유지를 확인했다.
독립 감사는27보관/1,323항목·로컬656파일·경계45회·새 프로세스/FD30회·소스/고정 입력 보존,
남은DB/partition규칙0을 확인했고 판정 변조24개를 거절했다. 첫 배치의1,260항목/629파일과 BLOCKED도 따로 보존했다.
상태 확인 중 방지 환경을 빠뜨려 무시 대상Python cache1개가 생성된 예외를 기록했다. 변경 없는 소스의 컴파일 결과와
정확히 같음을 확인해 bytes/권한/소유자/mtime을 별도 보존하고 모든 감사 뒤 저장소 캐시를 제거했다.
소스·문서·고정 입력은 불변이었으나 무시 대상까지 모든 저장소 파일이 불변이었다고 주장하지 않는다.
이 단계는 해당 케이스와 기록된 합의 포트 범위이며 AST 갱신·전체L1·다른 TC/외부 결함·최종Web은 남는다.
[RED/GREEN·두 실제 배치·TCP 원시 관측·예외](partition-consensus-port-evidence.json).

**2026-10-18 승인된 4/5 판정과 고정 높이 복구 — P2P 범위:**
사용자 승인에 따라 WEMIX 소수4대 정지·과반 생산 지속, 다른 BFT 체인 과반 정지를 구분했다.
복구 뒤9대 모두 기준 높이 도달·syncing=false를 기다리고 같은 고정 높이 해시를 비교한다.
의미 있는20개 RED(19개 관측 누락/판정과1개 낮은 높이 비교)를 확인하고30시나리오·race10회,
관련5패키지 전체 race·전체 make check를 통과했다. 잘못된 첫 높이 fixture는 RED 근거에서 제외했다.
고정 소스9d8763afa816272b7d1994bc835c374c5b541000c5e3e7cf356536c6d9a89fc3의 세 체인 실제 결과는3PASS다.
단언24개·동작75개와 독립 실패 대조24개,27보관/1,323항목·로컬655파일,
경계45회·새 프로세스/FD30회·남은DB/규칙0·소스/고정 입력 보존을 확인했다.
**이 결과는 P2P 포트 분리 범위다.** 후속 외부 소스 읽기에서 WEMIX의 etcd peer가 별도 TCP 포트임을 확인했다.
실제 기록은P2P30301/etcd30302/localhost client30303인데 기존 방화벽은P2P만 차단한다.
따라서 완전한 합의망 단절을 증명한 것으로 세지 않으며 기록된 etcd peer의 차단·복구 실증을 다음 필수 단계로 남긴다.
AST 갱신·전체L1·최종Web도 남는다. [판정·복구 증거와 격리 한계](partition-quorum-evidence.json).

**2026-10-18 firewall 복구 오류 전파와 WEMIX 판정 충돌:**
삭제 실패1/2/7과 조회 오류2/3/4/126을 성공으로 반환하던7개 실제 셸 RED를 재현했다.
없는 규칙의 정상 종료와 중복8규칙16회 제거를 보존하면서 오류 상태를 호출자에게 전달하도록 고쳤다.
관련5패키지 전체 race·전체 make check가 통과했다. 첫 sandbox bind 거절과 주석/줄수 래칫 실패는 별도로 보존했다.
고정 소스c0af3e18b12dad2b9998194fd6d226187c69d835e92d3a3debcd0d3fb0301974의
원래 CT-FAULT-004 세 체인 결과는2PASS/1FAIL이다. 복구 동작은 세 체인 모두 성공했고 복구 뒤9노드 해시가 같다.
WEMIX의 유일한 실패는 변경한 복구 동작 앞의 과반 쪽 blockHalt다. 소수 쪽은30초+0, 과반 쪽은+26이었다.
27보관/1,323항목·로컬666파일, 경계45회·새 프로세스/FD30회·소스/보조 입력/공유 키/바이너리 보존,
남은DB/partition규칙0을 독립 확인했다. 첫 기동의 etcd 실패 문장만으로 W2 재발을 주장하지 않는다.
사용자는4/5분리를 유지하고 WEMIX 소수 정지·과반 생산 지속, 복구 뒤 전 노드 동기화·동일 해시를 검증하도록 승인했다.
이 기준의 DSL/문서/회귀·실제 재검증과 AST 갱신은 아직 남는다. 전체L1/최종Web은 미완료다.
[RED/GREEN·세 체인 복구 범위·실패 보존](firewall-healing-evidence.json).

**2026-10-17 현행 L1 첫14건과 복구 오류 전파 검토:**
소스797d5b256fbd35b83141a2b94b67a0891f9a044d473be3afdf8a7fa82b8175cd와 HEAD ae4c1c8d,
CLI9dae6c04492334ca255ddaee188a048019b0dc20aca1d717f86cacbf6f39565e를 고정했다.
전체 make check·사전66계약 뒤 기본2·계약8·장애4의14건이 통과했다.9노드 실제 firewall 단절/복구도 포함한다.
fwRun의 복구 셸이 삭제 실패 뒤 break/true로 성공하는 경로를 검토 중 발견해, 실행 중인 케이스를 강제 종료하지 않고
정상 정지/보존 뒤 중단했다. 결과는14PASS/344NOTRUN·종료130이며 결함 수정/전체L1로 세지 않는다.
독립 감사는63보관/3,788항목·로컬1,950파일, 경계별15대210회 확인·새 프로세스/FD30회,
공유 키/바이너리/소스/고정 보조 입력 보존·남은DB0을 확인했다. 후속 읽기 전용 관측에서15대의 partition규칙도0이다.
원본/헬퍼/저장소는 실제 실행과 독립 감사 중 바뀌지 않았다. 방화벽 오류의 RED→GREEN과 실제 재증명,
전체358회·외부 결함·원문/legacy·최종Web14는 남는다. [실제 결과·독립 보존](linux-l1-current-boundary-evidence.json).

**2026-10-16 중단 경계 뒤 전체 AST 재측정:**
커밋35fbfa4f 뒤 고정 소스8250f219fa48cf72f2d48ee77652235c707b325722944139e1307650f8d98398에서
코드2,070파일·14언어/형식·Go1,398(테스트793),136,003노드·145,142간선과 누락/끊어진 간선0을 확인했다.
두 전체 그래프와 두 A8 결과의 bytes 해시가 각각 일치하고 과거 그래프14개도 보존했다.
새 중단 함수1개·호출2곳·회귀 선언8개를 두 그래프에 대조해 특정 누락3종을 각각 거절했다.
파서15계약과 일반 변조12거절, 내장AST38·미해석 동적 입력36·A8 비메서드 공개심볼1,890/도달 불가0을 확인했다.
구문 범위의 측정이며 실행 커버리지/전체L1/Web 인수는 아니다. [분류·그래프·독립 감사](all-code-graph-sweep-refresh-evidence.json).

**2026-10-16 L1의 안전한 케이스 경계 중단:**
TCSWEEP_STOP_FILE 표식이 있으면 현재 케이스의 정상 정지와 설정된 보존 훅이 끝난 뒤 다음 케이스 전에 멈춘다.
남은 횟수는 NOTRUN으로 합산하고 종료130/STOPPED를 기록한다. 정지/보존 실패는 기존 종료2를 유지하며,
마지막 케이스까지 끝났으면 정상 완료 결과를 유지한다. 표식 내용은 읽거나 쓰거나 지우지 않는다.
기존 코드의 무시/다음 실행을5개 RED로 확인해19계약(다음 attach 호스트 회귀 포함)과 전체 검사를 통과했다.
실제 파일럿에서 운영자가2건 제한 인자를 빠뜨려 전체216파일/358회 계획이 선택됐다. 원래2건 계획/검증기는 실패로 보존한다.
원본을 바꾸지 않은 별도 감사는 실제358계획과1PASS/357NOTRUN·종료130·다음attach 호스트 미시작을 대조했다.
첫 StableNet4BP/1EN의2단언,5보관/300항목·로컬156파일·새 프로세스/FD30회·공유 키/바이너리/입력 보존·DB0을 확인했고
잘못된 결과6개를 거절했다. 이1건을 전체 L1이나 원래2건 선택 파일럿 성공으로 세지 않는다.
고정 소스는 ce0f7cad865963de708456b2c96f34a3f49bcb80390e2ccc49e3290c42813377이며 실행·감사 중 저장소/고정 헬퍼 수정은 없었다.
셸 변경 뒤 최종 AST 재측정과 전체 L1/Web 인수는 남는다. [구현·선택 실수·실제 범위·독립 감사](sweep-boundary-stop-evidence.json).

**2026-10-16 비교·관측 보강 후 전체 AST 재측정:**
커밋1d8f7c97 뒤 고정 소스 f0c94ef520242634cc1dd7e88238a10ba95b88e6966fec2bc5e36c759be46619에서
소유 코드2,070파일·14언어/형식·Go1,398파일(테스트793)을 파싱했다.135,880노드·145,019간선이며 누락/끊어진 간선0이다.
내장 AST38개·미해석 동적 입력36개를 그대로 명시했다. 두 전체 그래프와 두 A8 결과의 bytes 해시가 각각 일치한다.
A8 비메서드 공개심볼1,890개/도달 불가0이며, 방법론상 메서드의 도달성은 이 수치 밖이다.
파서15계약과 별도 감사의 변조12거절·중첩 Docker/삭제3프로그램·과거 그래프10개 보존을 확인했다.
파서 구현 변경이나 새 기능 RED→GREEN을 주장하지 않는 재측정이며, 실행 커버리지/전체L1/최종Web 인수가 아니다.
[소스·분류·그래프·독립 감사](all-code-graph-probe-refresh-evidence.json).

**2026-10-16 전체 기록 노드 비교·합의 오류 관측 보강:**
기본 sameBlockHash 비교에서 RPC가 없는 기록 노드를 제외하거나 블록이 없는 노드를 건너뛰어 성공하던 경로를 막았다.
명시한 on/onEach 선택은 유지하며, 블록 누락은 Actual에 null로 남긴다. 남은 호출이 없는 allTargets는 제거했다.
합의 관측에는 고정 실패 분류와 숫자 RPC 코드(0 포함)만 더하고, 상위 오류 메시지/URL·기존 판정/사유·시간 제한·요구는 바꾸지 않았다.
기능 RED16개 뒤 집중26개·관련 전체 race1,891 PASS 이벤트/40 gated SKIP·전체 검사를 통과했다.
앞선 race의 재시작 준비2테스트 시간 제한 실패는 보존했다. 코드 수정 없이 분리10회와 전체 재실행이 통과했으며 원인/해결은 미확정이다.
고정 소스 c8075c12986e7b42262bf581dfdd9fd0a0da76b32ff536c1a2d47c09b6d9a91e의 첫 실제 배치는
StableNet 준비 BLOCKED·WBFT 높이21 진행 FAIL·WEMIX EN BLOCKED였다. WBFT의 두 해시 비교는5노드 모두 일치했다.
StableNet/WBFT의4BP seal 거절과 WEMIX EN의 rpc_error/-32000을 독립 대조7개로 보존했다. 정확한 실패 RPC 메서드/원인은 미확정이다.
소스·CLI·바이너리·조건을 바꾸지 않은 새 격리 배치에서는 세 체인 각3단언이 통과했다.
독립 감사는6개 해시 비교의5노드·15실행 바이너리·45원문 RPC와 변조12개 거절,15보관/735항목·로컬384파일,
새 프로세스/FD30회·공유 키/프리셋45파일 보존·종료DB0을 확인했다. 첫 실패 배치의15보관/735항목·353파일도 별도로 보존했다.
반복 성공을 seal 결함·과거 EN unknown 해결이나 전체 L1/Web 인수로 세지 않는다. 최종 AST/전체 인수는 남는다.
[계약·실제 성공/실패·독립 감사](all-node-probe-integrity-evidence.json).

**2026-10-16 WEMIX EN 합의 조회 반복 진단 — 앞선 원인 미확정:**
고정 소스 `2527c6471f65780dd058a707fbe380c518a9489dba2e64dadc49d3999fbd72bd`와 같은 Linux WEMIX 바이너리에서
CT-NODE-009를 다시 실행해1건/3단언이 통과했다. 기능 Go·공개 케이스·프리셋은 이전 시도 이후 수정하지 않았다.
현재5노드의 합의 probe는 모두 supported이고, 본문은 높이68·70에서 모두 같은 해시와 주 노드의68→70 전진을 확인했다.
별도 RPC55개는 이후 관측 시점에5노드 모두 높이132·admin 정보 객체·같은 governance 코드와4명의 구성원을 확인했다.
EN의 self=null·etcd={}는 이때 정상 조회와 함께 관찰됐다. 상시 기능 부재나 Info()의 빈 문자열을 원인으로 확정하지 않는다.
독립 감사는 RPC 변조4개와 단언 변조4개를 거절했고5실행 바이너리·5보관/135항목·로컬125파일·공유 키/프리셋45파일,
새 프로세스/FD10회와 종료DB0을 확인했다. 이전 실패 시점의 RPC 원문·오류 범주가 없어 그 원인은 여전히 미확정이다.
기능 관문·시간 제한·요구·외부 소스를 바꾸어 통과시킨 것이 아니며, 이 반복 성공을 결함 수정 RED→GREEN이나 전체 인수로 세지 않는다.
이전 차단 자료도 보존한다. [실행·원문·독립 감사](wemix-en-capability-diagnostic-evidence.json).

**2026-10-16 Docker 표준입력·중첩 셸 AST 보강:**
Python의 정적 UTF-8 `input=`과 Docker exec의 명시적 stdin 전달, 제한된 xargs 명령을 구문 그래프에 포함한다.
초기5개 실패와 동적 Docker 옵션·`sh -s -c` 우선순위의 추가 실패2개를 확인해15개 계약으로 보강했다.
스크립트 파일/명시적 -c에 전달되는 일반 데이터는 코드로 오인하지 않으며, 미지원 옵션·동적 입력은 남긴다.
고정 소스 `f6163e12d7413149e434b124138c31f8ddf2129ee9f14935caf5978ef41c61e7`에서
2,069파일·14언어/형식·Go 테스트792파일·내장 AST38개·미해석 동적 입력36개를 확인했다.
두 그래프는 SHA `0741c968a0a823d36c6d83b11967247510580080754bdd7f23a09dd16454b909`로 일치했다.
독립 감사는 소스 집합/해시·간선·중첩 입력 출처와 새 보존 전송의3개 셸 프로그램을 대조하고12개 잘못된 자료를 거절했다.
A8의 기존 비메서드 공개심볼은1,890개·도달 불가0이며, 과거 그래프6개도 그대로 보존했다.
구문·명시한 어휘 범위의 갱신이며 전체 동적 호출·실행 커버리지·최종 Web/L1 인수는 아니다. [근거](all-code-graph-stdin-evidence.json).

**2026-10-16 작업 Go 캐시의 선택 정리:**
8시간 이상 사용/변경되지 않은 정규 캐시 파일만 최대4GiB 회수하고 최소1GiB는 남기는 도구를 추가했다.
최근/미래 시각·참조 해시·알 수 없는 이름·링크·실행 디렉터리·제어 파일은 보존한다.
Go 도구나 캐시 FD가 보이거나 조회가 실패하면 거절하고, 두 최신 조회와 삭제 직전 파일/디렉터리 신원 검사를 수행한다.
기본값은 조회다. 새 보고서 경로를 정해 `python3 -B scripts/prune_go_cache.py --report /private/tmp/chainbench-cache-review.json`로 확인하고,
회수할 때는 별도의 새 보고서 경로와 `--delete`를 쓴다. 보고서를 덮어쓰지 않는다. 각 큰 빌드 묶음의 전후·실제 증명 시작 전 여유를 확인하고,
20GiB 미만이면 기존 승인 정리 규칙을 먼저 적용한다. 실행 중에는 정리하지 않고 새 빌드도 함께 시작하지 않는다.
초기 반복 문서 검색은 삭제 전 중단해 실패 자료를 보존했다. 해시 식별자를 한 번 추출하도록 보강한 후,
고정 소스 `a17a517391d03a8c1bbe10b5e447a03adb05e44eeae9741978bf7ec95364c2cd`에서 실제 열린 파일의 거절과
17,132개/4,148,563,968바이트 회수를 확인했다. 독립 감사는 남은 일반20,121파일의 해시·메타데이터와 전체 경로 집합을 대조했다.
여유는 약21.1→25.0GiB였다. 캐시는 재생성 가능하므로 이후 빌드에서 다시 늘 수 있고, 주기별 예산과 기존 종료 DB 회수를 함께 유지한다.
이 단계는 모듈 소스·증명·사용자 앱·swap을 정리하지 않았고 전체 L1/Web 인수를 뜻하지 않는다. [계약·실제 회수·독립 감사](go-cache-maintenance-evidence.json).

**2026-10-16 Linux 보존 전송 보강 — 기능 결과와 종료 검증 분리:**
정상/변조 패키지·보호 경로·키 별칭·노드 그룹의27계약을 추가했다. 실제6개 RED와 관련/전체 검사,
CI 직접 연결을 통과했다. 종료 서버마다 metadata/보관/전후 관측을 한 전송으로 묶고,
두 번의 최신 프로세스/FD·두 번의 소유자 확인·기록된 전체 노드 선택과 파일 재검증은 유지한다.
고정 소스 `dc0a43aa6bc2985d53ab7c8fc5f71476fb1314b601a03184361b5048a0fa0eec`에서
실제 파일6회 회수와5회 거절·보호 파일 보존을 검증하고, 별도 감사61항목·60파일을 확인했다.
첫 native는 관측 도구를1번에만 설치하고 모든 서버에 있다고 요구한 준비물 목록 오류로 중단했다.
거절 기록과 기존 회수 경로의 보관을 보존했고, 목록을 서버별로 고친 새 시도는세 체인 모두 종료/회수됐다.
새15보관·735항목·로컬374파일·독립 프로세스/FD30회와 공유 키/프리셋45파일 보존을 대조했다.
**기능 결과는 StableNet/WBFT 각1건/3단언 통과, WEMIX1건 BLOCKED다.** WEMIX는5노드가높이51까지
진행했지만 EN의 합의 기능 확인이 unknown이었고 DSL 본문은 실행하지 않았다. W3 unauthorized block은0건이다.
원인을 W3로 단정하거나 기능3개 통과로 세지 않는다. 종료 검증은 실패/차단 시에도 자료를 보존·회수하는 범위다.
실행 중 여유가 일시적으로20GiB 아래로 내려갔고, 종료 후 약21GiB였다. 작업 Go 빌드 캐시는 약7.5GiB로,
이후 사용 중·현재 입력을 보존하며 오래된 재생성 가능 캐시만 정리할 필요가 있다.
전체 L1·원문·외부 결함·최종 Web은 여전히 미완료다. [RED/GREEN·실제 파일·체인 실행·감사](linux-retention-transport-evidence.json).

**2026-10-16 격리 L1 첫 실행과 중단 복구 — 전체358회 미완료:**
고정 소스 `31db8357515ec717a36451239fb2146dc9c221253609cea947b47e0c0fea17c8`에서
216파일·358회 모두 다른 키/원격 루트로 연결하고10개 입력 계약을 RED→GREEN으로 확인했다.
실제 첫 공개 케이스1건/2단언과5노드 종료가 통과했다. 반복되는 원격 확인 호출을 보강하려고 스케줄러를 닫았으나,
두 번째 attach host는 이미 시작돼 있었다. 명령·부모 확인만으로 케이스 경계를 보장하지 못한 운영자 오류다.
두 번째 공간을 별도 정상 종료·보존·회수했으며 이 케이스를 완료나 PASS로 세지 않는다. 원래 종료-15 요약은 덮어쓰지 않았다.
별도 감사는 두 공간600항목·로컬232파일·프로세스/FD30회·공유 키 보존·종료 DB4종0개를 확인했다.
소스와 기존 helper 입력은 변하지 않았다. 이 실행은 전체 L1·원문·외부 결함·최종 Web을 닫지 않는다.
이후 실행기는 종료/회수 뒤 명시적인 경계 검사를 두고 원격 전송 호출을 묶어 검증한다.
[부분 실행·중단·정상 종료 복구·독립 보존 대조](linux-l1-isolated-pilot-evidence.json).

**2026-10-16 Linux 실행별 보존 경로 격리 — 새 L1 실행과 구분:**
`selected_roots`에 명시적 `data_root`를 받아 실행별 키·설정·로그·노드 데이터만 선택한다.
기존 공유 루트는 기본값으로 유지하며 다른 실행·경로 별칭·루트 이탈14개 입력을 RED로 확인하고10개 계약과 관련/전체 검사를 통과했다.
고정 소스 `47b45dcc8ed317763405e99b407d1462e9f0f15010c5e736ffe90a964ee4b134`에서
앞서 종료한6노드의364항목·원래 보관 파일·공유 키를 읽기 전용 capture와 별도 감사로 대조했다.
프로세스/FD24회가 비활성을 확인했고 로컬151파일을 포함한229개 입력은 이번 관찰 동안 변하지 않았다.
정의된 DB4종은 기존 회수 후0개다. 새 체인을 실행하거나 DB를 삭제하지 않았다.
첫 사전 대조는 과거 프리셋20파일의 mtime 차이를 감지해 중단했다. 내용·권한·소유자는 같았으며,
그 실패 로그와 과거 증명을 보존하고 현재 기준을 새로 고정했다. 과거 시각이 계속 보존됐다고 주장하지 않는다.
전체 L1·legacy·최종 Web은 여전히 미완료다. [RED/GREEN·읽기 전용 대조](isolated-linux-retention-evidence.json).

**2026-10-16 공개 명령 안내216개 정합성 — 실제 체인 실행과 구분:**
공개216케이스 중 명령 안내가202개만 싣고 있던14누락을 발견했다. 실제 파일 집합·표시 수·생성기 결과를 검사하는
세 계약을 RED로 확인한 뒤 기존 생성기로 목록을216개로 다시 만들었다. 새 `case-docs` 검사는 `make check`에 포함한다.
CI가 `make check`를 부르지 않아 이 검사도 빠지는 것을 추가 RED로 확인했고, test job에 직접 연결해4계약을 통과했다.
별도 고정 소스 `3600dfe51211f1cc3d88dd80672a6e6d7e481b846cb410f92b184b2acd0507ca`에서
셸216블록/AST5,574노드를 파싱하고 CLI216개가 각 파일을 정확히 한 번씩 가리키는지 대조했다.
CI 연결까지 포함한 후속 고정 소스 `db592f179cc0b2deb7229b48342b4bee8e6675a70d047b234500585ed5f911c8`도 같은 결과로 재확인했다.
안내된 스위프209개도 실제 grep 선택 결과가 정의 파일 하나인지 확인했다. 필수 입력 자리표시는 구문 분석용으로만
치환했고 명령을 실행하지 않았다. 따라서 전체358회 L1·체인 실행 성공·최종 Web으로 세지 않는다.
[실제 누락·RED/GREEN·별도 구문/파일 대조](case-command-docs-evidence.json).

**2026-10-16 공개 서명 교체 영수증 송수신자 보강 — 전체 legacy/L1는 미완료:**
앞선36단언 고정 소스는 독립 RPC에서 영수증 송수신자를 대조했지만, 공개 DSL은 그 필드를 직접 판정하지 않았다.
거래 자체의 송수신자는 맞고 영수증만 다른 주 노드/EN 네 상태의 거짓 통과를 RED로 확인했다.
기존 공개 판정에 주 노드와 두 EN의 영수증 from/to 여섯 단언을 추가해42단언·23대조(정상2/오류21)를 통과했다.
관련4패키지 전체 race와 전체 검사가 통과했고, 새 고정 소스
`51fe8ef22cf5bcbe15b251b4f3b80e4765483ef941485acf2bbed010622accc2`의
격리 Linux BP4/EN2에서1.0.1 bf17→고정1.1.0 4d17의 실제1건/42단언·독립 RPC18관측을 확인했다.
서명 원본 바이트·Keccak·거래/블록/영수증·현재 높이·로그와 변조 거절8개를 별도 대조했다.
이번 실행은 정상 종료0·6서버 회수·정의된 DB4종0개이며, 원격364항목·로컬151파일과
기존 공유 키·프리셋45파일을 유지했다. 회수12회·독립12회 프로세스/FD 관측도 통과했다.
이전36단언 실행과 종료 복구 기록은 덮어쓰지 않았다. 원문15노드·모든 버전 쌍·전체254/87/L1·최종 Web은 여전히 별도다.
[새 RED/GREEN·고정 소스·정상 종료·독립 대조](legacy-signature-receipt-evidence.json).

후속 문서 점검에서 공개216케이스 중 사용 안내 명령14개 누락도 확인했다. 문서의202개 표시는 현재 수치가 아니며,
후속 [명령 안내 보강](case-command-docs-evidence.json)에서216개로 재생성하고 재발 방지 검사를 추가했다. [누락 목록](/private/tmp/chainbench-case-command-drift-20261016.json).

**2026-10-16 원문 서명 교체 두 EN의 현행 보강 — 전체 legacy/L1는 미완료:**
원문의 셸 두 개를13b5e12c Git blob과 다시 대조했다. 허용 높이 차이는2블록이며, 기존 공개 케이스는
EN 하나의 이전 거래 필드·버전만 비교하고 읽은 로그를 판정하지 않았다. 원래14개의 거짓 통과와
3블록 지연·빈 로그·잘린 줄 경계도 RED로 확인했다. 공개 케이스19대조(정상2·오류17)와 로그 입력21대조를 통과했다.
`readNodeLog.requireComplete`는 빈 캡처와 크기 한도 도달을 거절한다. `derive.regexpCount`는
이전 로그 접두와 줄 경계를 지킨 새 구간을 검사하며, 회전·잘림·형식 오류를 성공으로 바꾸지 않는다.
어휘·인자·편집 계약·작성 안내에 반영했고 관련4패키지 전체 race와 전체 검사가 통과했다.
고정 소스 `0fc117d001be52e65f4dd2946a49dcfb6e7559f94d620a02566fd0a87de0271c`의
소유 Linux BP4/EN2에서1.0.1(bf17c960) 네 BP를 유지하고 두 EN을1.1.0(4d17eea0)으로 교체했다.
공개1건/36단언·실제 실행 파일6개 해시·별도 RPC18관측을 확인했다. 같은 서명 원본 바이트·Keccak 해시,
거래/블록/영수증·현재 높이·동기화 상태·새 로그를 독립 대조했다. 별도 secp256k1 복구 구현은 아니다.
이번 자료는 원문15노드의 EN8/11 전체 환경과 구분한다. 전용 키·원격 데이터 경로를 사용했고 기존 공유 키를 덮어쓰지 않았다.
초기 종료 도구는 지원하지 않는 `--json` 때문에 거절돼 원래 실패 기록을 남겼다. 같은 소스의 올바른 종료를 완료한 뒤
6서버 보존·미사용 확인·정의된 종료 DB4종0개와 원격364항목/로컬153파일 보존을 독립 검증했다.
회수12회·독립12회 프로세스/FD 관측, 기존 로컬 프리셋45파일과 원격 공유 키 불변도 확인했다.
한 실제 실행의 종료를 마무리한 것이며 두 번째 실행이나 원래 실패 자체를 성공으로 바꾼 것이 아니다.
원문 전체254개·87개·L1·최종 Web과 S1/ProcessExtraSeal/W3/W4는 열려 있다.
[원문·RED/GREEN·고정 소스·최초 실패·종료 복구·독립 대조](legacy-signature-compatibility-evidence.json).

**2026-10-16 전체 코드 AST 갱신·실행 문자열 파싱 — 전체 실행 인수는 미완료:**
이전 그래프의 1,922파일은 과거 고정 소스의 결과다. 후속 HEAD의 새136파일·변경53파일을 먼저 대조해
과거 그래프를 현재 커버리지로 사용할 수 없음을 확인했다. 실행 문자열을 놓치던7대조를 RED로 확인했고,
Go os/exec·Python subprocess·셸의 정적 프로그램을 각 언어로 파싱하도록 보강했다.
변수 가림·재할당, shell=True 인자 의미, 빈 셸 인자, heredoc 데이터와 코드 구분, 제외 바이너리와
내장 코드 출처도 실패 대조로 보강했다. 10개 계약·관련 race·전체 검사가 통과했다.
고정 소스 `80c435d629c56c2e622d041b771cbf0ea358631cabfaeab5f9c560463dcd274f`에서
14언어/형식 2,060파일·Go 테스트790파일·내장 AST36개를 파싱했다. 구조 노드134,579개·간선143,659개,
제외401개·누락/끊어진 간선0개다. 동적 프로그램 입력33개는 해결된 코드로 표시하지 않았다.
별도 프로세스의 두 그래프 SHA256은 `909cdd7cc89225c78131aa1bdccf8856131de287618d6418bf9295e92749d07e`로 같았다.
별도 감사는 파일 집합·내용 해시·간선·내장 코드 출처와10개 변조 거절을 검증했다.
기존 두 그래프의 내용·권한·소유자·ns 수정 시각과 파서 입력24개의 불변도 확인했다.
A8은 비메서드 공개심볼1,890개(내부454·외부1,171·테스트176·시그니처80·상수집합9), 도달 불가0이다.
메서드·런타임 도달성·모든 동적 프로그램·전체 L1/legacy/87개·최종 Web은 이번 증명 범위가 아니다.
실제 노드나 DB를 만들지 않았으며 새 그래프 두 개는 각53,783,260바이트다.
[고정 소스·실패 대조·파서 출처·별도 감사](all-code-graph-refresh-evidence.json).

**2026-10-16 원문17 배포 delegate 가스 추정 보강 — 전체 legacy는 미완료:**
원문 Git blob의 Simple7702Delegate를 solc-js 0.8.28로 별도 컴파일해 공개17b에 배포한다.
생성 코드 479바이트·실행 코드 451바이트와 AST 44노드/43포함간선을 확인했다. 같은 컴파일러 재실행의 코드·ABI·메타데이터·AST도 일치했다.
비용만 검사하는 유효 초안의 누락 6개를 RED로 확인했고, 배포 코드 해시·초기 getter·원래 같은 주소/금액 생략 조건을 더해 12대조를 통과했다.
주소 기준선은 새 배포 from 참조 1개만 997→998로 추가했으며 기존 997개는 그대로다.
관련 2패키지 전체 race와 전체 검사가 통과했다. 컴파일 fixture도 기존 고정 소스 지문에 포함되는 텍스트로 보관한다.
고정 소스 `15d361b79087ee65f34cf927fe362c2bf6065955e10088acb62dc7d48649f3b3`의
StableNet4d17eea0·WBFTd535dd3d에서 새17b와 기존 EOA·1wei 케이스를 직렬 통과했다(공개 4건/42단언·실제 RPC 16관측).
모두 21000/46382/71865였고, 실제 생성 거래·영수증·CREATE 주소·코드와 원래 비용 범위를 독립 대조했다.
컴파일러 파일 95개/링크 2개와 프리셋 키 45개의 불변, 증명 860파일의 내용·권한·소유자·ns 수정 시각과 종료 DB/trie 0개를 확인했다.
이번 단계가 만든 미사용 npm 다운로드 캐시 39파일/7,893,232바이트만 두 차례 사용 여부를 확인해 제거했고 컴파일러 입력은 보존했다.
현재 계획은 216파일·358회이며 전체 실행은 아니다. 원문 EN 토폴로지·Foundry 캐시 바이트·배포 fee 표현과 현재 fixture를 구분한다.
전체 254개 동등성·L1·87개·최종 Web은 열려 있다. [원문·컴파일·실제·독립 증거](legacy-deployed-authorization-evidence.json).

**2026-10-16 RPC 응답 명확성과 메서드 존재 보강 — R3/R5 전체는 미완료:**
non-null result와 error의 동시 반환, 오류 필드 누락, 결과 없는 응답, 후행 JSON의 core RED 20개를 확인했다.
변조 액션 1개, HTTP·깨진 응답 등의 존재 오탐 5개, rpcError 모순 1개, 존재 검사 프로토콜 3개도 RED→GREEN으로 확인했다.
명확한 노드 오류만 typed로 보존하며 결과를 버리는 호출도 검사한다. 기존 result:null+완전한 error 호환과 인자 오류 probe는 유지한다.
관련 6패키지 전체 race와 최종 ready-check가 통과했다. 중간 전체 검사의 추가 3개 RED도 보존했다.
고정 소스 `9375d652e85e6ca5c997ec64344dddf465dc50c2dcc446939d06d18809d08e3c`에서
세 체인의 공개 메서드 존재 3건과 StableNet personal 4건을 직렬로 통과했다(공개 7건/49단언·별도 12노드 probe).
12노드 모두 -32000/missing FeePayer였으며, cache64 시작 16개와 전용 키 경로를 확인했다.
두 묶음의 독립 대조와 마지막 합동 대조로 814파일의 내용·권한·소유자·ns 수정 시각, 프리셋 키 45개의 불변과 종료 DB 0개를 확인했다.
trie cache 2,256,903바이트는 보존했다. 메서드 등록 smoke와 정상 서명 행동을 구분하며 원문 EN 환경 전체는 아니다.
rpcError/methodPresent는 현재 on만 지원하고 onEach는 계약에 없으므로 새 미완료 기능으로 세지 않는다.
ID·버전·중복 키를 전부 검증하거나 WS를 변경한 것은 아니며 전체 R3/R5/L1/legacy/87/최종 Web은 열려 있다.
[RED·고정 실제·독립·합동 보존 증거](rpc-response-contract-evidence.json).

**2026-10-16 rpcError 제출 오류 분류 보강 — R5 전체는 미완료:**
두 거래 제출 메서드의 rpcError가 JSON 파싱·요청·인자·내부 오류를 기대 문구만으로 통과시키는8개 RED를 확인했다.
공통 checkSubmitRejected를 소비하며 표준 프로토콜5코드를 거절하고 일반 서명API의 인자 오류 검사는 유지한다.
첫 수정에서 메서드 부재 문자열 보호가 빠질 수 있는2개 추가 RED도 확인해 기존 보호를 유지했다.
집중17대조·관련4패키지 전체 race·최종 전체 검사가 통과했다. 공개 케이스 입력은 바꾸지 않았다.
고정소스 `6c45176304d440719979d470447ce21cc9afc28c1992a806400ce17665beae0c`의
StableNet4d17eea0 BP4에서 기존 personal 공개4건/46단언을 재실행했고 직접 변조 거절·충전 후 같은 서명 바이트
성공·정확 송금/수수료·3블록 비포함과 독립 대조를 확인했다.254파일과8개 전용 import 키를 보존했고 DB/trie는0이다.
두 메서드의 프로토콜 오류는 단위 주입이며 실제 증명은raw제출 경로다. 잔액부족 단언의 숫자 오류코드는 별도 기록하지 않는다.
onEach는 현재 계약에 없으므로 새 미완료 기능으로 세지 않는다. 모순 result/error 응답은 위의 후속 RPC 응답 단계에서 다뤘으며, 다른 R5 조합·전체 L1/87/legacy/Web은 열려 있다.
[RED·실제·독립 증거와 한계](rpcerror-submission-classification-evidence.json).

**2026-10-16 원문18 EOA·1wei 가스 기준선 보강 — 이 단계에서는 원문17 배포 fixture 제외:**
기존 공개17은 같은 주소·금액 생략으로 추정했고 수신자의 코드 없는 EOA 조건이 없었다.
서로 다른 새 수신자·value0x1·eth_getCode0x를 더하고 기존0/1/2 authorization의9개 비용 검사는 유지했다.
단위11대조에서 조건4개 거짓 통과와 잘못된 요청 형태1개를 RED로 확인한 뒤 GREEN으로 만들었다.
기록기의gzip 누락도 별도 RED→GREEN으로 고쳤다. 관련2패키지 전체 race·전체 검사를 통과했다.
고정소스 `135b660ec2c6c0b46c67102ceafb4a9a13c8290c133caee0b1e47235e8a4f9c3`의
StableNet4d17eea0·WBFTd535dd3d BP4에서 공개2실행/20단언과8개 실제 RPC 관측을 확인했다.
두 체인 모두21000/46382/71865이며 독립 계산·원본gzip 대조·18개 변조 대조가 통과했다.
성공430파일과 첫 기록누락 실패215파일 내용·권한·소유자·ns수정시각을 보존했고 종료 DB와 trie cache는0이다.
첫 공개케이스PASS/10단언은 중계기록0개로 증명 전체FAIL이었으며 성공으로 합산하지 않는다.
원문18의EN1→EN2 대신BP coinbase→새EOA를 쓰고, 원문17의배포된Simple7702Delegate 대신 코드없는delegate를
쓰는 차이도 남겼다. 원문17 배포 부분은 위의 후속17b에서 별도 검증했으며, 전체254개 동등성·L1·87개·최종Web은 열려 있다.
[원문·실패·재시도·독립 증거](legacy-authorization-estimate-evidence.json).

**2026-10-16 공개 제네시스 전체 노드 검증 보강 — 전체 legacy는 미완료:**
누락 노드·빈 해시·짧은 해시·비주 노드의 잘못된 부모 해시 등5개 거짓 통과를 실제 실행기에서 RED로 확인했다.
공개 CT-NODE-003은 각 노드의32바이트 해시와0인 부모 해시를 onEach all로 검사한다.
sameBlockHash의 일반 누락 처리 의미는 바꾸지 않았다.10개 대조·관련2패키지 전체 race·전체 검사를 통과했다.
소스 `6018386f8f5266ef04644f88642f1d793e2bfbf182c8b2f17519309e907a637e`의
고정 StableNet4d17eea0·WBFTd535dd3d·WEMIX4538a9f5에서 공개3실행/15단언과별도12노드 조회를 확인했다.
독립 관측 오류24개를 거절했고563파일 내용·권한·소유자·ns수정시각,45개 기존키 파일 불변을 확인했다.
종료 DB는0개이고 trie cache2,356,490바이트는 보존했다. 첫 검증기의config.toml 이름 가정 오류와 원본을 유지한 채,
실제 --config 경로 및 보관본을 확인하는 별도 검증기가 통과했다.
원문은 누락 노드를 건너뛰며 private1010/testnet8283을 썼으므로 현행 BP4/체인별 ID와 구분한다.
[원문·RED→GREEN·실제·독립 증거](legacy-genesis-evidence.json). 전체L1·254개 동등성·87개·최종Web은 미완료다.

**2026-10-16 개인 계정 API 대납 원문4건 보강 — 전체 legacy는 미완료:**
원문 import/unlock→personal_signTransaction→personal_signRawFeeDelegateTransaction 경로를
공개01b–04b 네 케이스로 추가했다. 기존 공통 로컬 서명 케이스는 유지한다.
`sendRawTampered raw`는 키·수신자·금액 기반 생성 입력과 배타적이며 원시0x16의
Sender.S 또는 FeePayer.R만 바꾼다. 비어 있지 않은 data/access-list와 중첩 RLP 길이도
SDK 예상 바이트·서명 검증으로 대조했다. 원본 버퍼는 바꾸지 않는다.
raw경로·입력 진단11개 RED와 키 저장소 격리4개 RED를 GREEN으로 만들었다.
관련5패키지 전체 race·추가 accounts race·프런트71개·최종 전체 검사가 통과했다.
fuzz 두 실행은256,491/312,462회이며 완전성 증명은 아니다. 새 네 케이스의 주소 참조48개만949→997로 추가했다.

소스 `5381fcaeee26d7de39805ebe96618016b507a27deb16b601589bafebcc7d193e`의
Mac 고정 입력 StableNet4d17eea0·WBFTd535dd3d·WEMIX4538a9f5에서
새 개인API4건/46단언과 기존 공통6건을 직렬 통과했다. 16개 실제 cache64 노드 시작을 확인했다.
정확한 송금·수수료, 노드 서명의0x02→0x16 보존, 자금 충전 후 같은 서명 바이트의 채굴을 확인했다.
반복 제출한 두 변조 입력은 정확한 typed 오류·3블록 뒤 거래/영수증 null·자금/nonce 불변을 대조했다.
독립4회 프로세스/열린 파일 조회와785파일의 집합·bytes·권한·소유자·ns mtime을 확인했다.
종료DB3종0개, 작은 triecache2,335,098bytes는 보존했다. 프리셋45파일은 전후 불변이고
개인API import 키8개는 전용 workspace 안에 있다.

첫 시도는 공용 keystore 경로를 발견해 운영자가 중단했다. 그 사이 생성된 테스트 키2개는
이번 송금 영수증 주소와 대조해 전용 위치로 옮겼고 기존 암호화 키5개의 Git bytes는 유지했다.
중단213파일·이전 구현 스냅숏을 보존했다. 다음 격리 시도는3통과·1사유 문구 불일치였으며260파일을
보존한 뒤 정확한 feePayer 오류 접두부로 고쳤다. 실패·중단 자료를 최종 통과로 세지 않는다.
원문 EN 역할·대략적 변조 위치와 현재 BP4/EOA·구조적 변조는 구분한다.
CLI 분석의 잘못된 --personal 표기도 실제 --rpc.enabledeprecatedpersonal로만 정정했다.
공개 파일은215개·전체 계획은357회/매핑 누락0으로 재측정했지만 전체 실행은 아직 하지 않았다.
다른 legacy·원문87·Linux/L1/R3/P4/R4/R5·외부 결함·최종AST/WEB14는 남는다.
[원문·입력·세 시도·독립 증거](legacy-personal-delegation-evidence.json).

**2026-10-16 서명 변조의 제출 거부 판정 — R5 전체는 미완료:**
`sendRawTampered`가 HTTP·연결·응답 형식·RPC 규약 오류와 메서드명만의 사유 일치를
노드 서명 거부로 통과시키는11개 RED를 확인했다. 기존 제출 판정기를 재사용해 hash 부재·typed 노드 오류·
규약 오류 제외·노드 메시지의 사유를 요구하고, 통과한 거부만 값으로 저장한다.
새17개 대조와 기존 제출 회귀·helper/engine/accounts 전체 race·최종 전체 검사가 통과했다.
검사 중 드러난 CLI/MCP 초 경계 오판은 별도3989c2ff에서 고정 시각 RED와30회 실제 구성 race로 고쳤고,
생산 코드 감소4줄과 TX-014 검토의 대상 해시/설명도 실제 변경과 대조했다.

소스 `24dee8cb4fc4a5e7821c5188804d6ff5c43342366fc37a0e739fd8baef86cafc`에서
새 StableNet4d17eea0·WBFTd535dd3d·WEMIX4538a9f5의 Mac BP4/cache64를 직렬 실행했다.
기존 공개 케이스6개와 추가 직접 노드 거부6건이 통과했다. 공개 케이스는 자금을 넣은 계정이며,
추가 관측은 새 무잔액 계정의 정확한 오류·원시 해시·즉시 거래/영수증 null을 확인한 범위다.
실행·보조 입력 불변과 새 프로세스/열린 파일 조회2회·531파일의 bytes/권한/소유자/ns mtime·파일 집합을
독립 대조했다. 종료DB3종0개, 작은 triecache2,321,254bytes는 보존했다.
첫 대조기의 saveKey 마스킹 가정 오류는 로그를 남기고 두 필드만 허용·변형30건 거절로 바로잡았다.
개인 계정 API 원문4개, 지연 미채굴/암호학적 서명 독립검사, 전체R5/L1/legacy/87·최종AST/WEB14는
이번 결과에 포함하지 않는다. [실행·독립·검사 증거](tamper-submission-rejection-evidence.json).

**2026-10-16 legacy extra 양방향·포크 경계 보강 — 전체 legacy는 미완료:**
과거02의 GovCouncil params→AccountManager 방향과07의 세 계정·여섯 상태를 현재 공개02/07b와 대조했다.
기존 케이스가 원문 방향, 반대 비트, dual, 과거 블록의 잔액 오류를 놓치는18개 RED를 확인했다.
정상2·오류20의22개 대조를 GREEN으로 만들고, 실제 RPC extra와 VM 플래그를 비교하도록 보강했다.
기존 alloc 방향 검사는 유지했다.07b는 Boho6/v2를 명시하고 블록5/6의 세 계정·여섯 상태와 잔액을 고정한다.
요구 태그·단위 fixture 제공 능력의 누락을 관련 회귀에서 보완했고, 두 케이스 주소 위치만910→949로 갱신했다.
관련 engine/helper 전체 race와 최종 전체 검사가 통과했다.

소스 `e66aaef7736af0d580cfa0aaa7c540efa85bc267d351609bebbc46250ff1508d`의 첫 실제 시도는
StableNet4d17eea0 Mac node4의 ProcessExtraSeal 패닉으로 준비 단계 BLOCKED였으며 본문과 다음 구성은 실행되지 않았다.
실패 증거·DB회수·두 번의 새 프로세스/열린 파일 조회를 독립 확인했다. 같은 소스·입력의 새 workspace에서
두 구성을 직렬 실행해 두 공개 케이스·36단언·11상태 관측을 확인했다. 시작 패닉 해결로 세지 않는다.
재현 시도 뒤에도 첫 자료가 불변이고, 성공한394파일의 bytes·권한·소유자·ns mtime·파일 집합과 종료DB0을 대조했다.
검증기의 원문 별칭/실행 주소 가정 오류는 실패 로그를 보존하고 실제 매니페스트 대조·네 음성 대상으로 바로잡았다.
원문 기본 Boho100·주소/토폴로지와 현재6·로컬EOA/BP4는 구분한다. Merkle proof 암호 검증은 이번 범위가 아니다.
전체254/87·Linux·메인넷·외부패닉해결·최종AST/WEB14는 남는다.
[원문·RED/GREEN·두 시도·독립 증거](legacy-extra-state-evidence.json).

**2026-10-16 legacy GovMinter 코드·상태 보존 보강 — 전체 legacy는 미완료:**
복구한 원문의 TC-5-2-05·TC-1-1-11/12와 현재 공개02/10을 대조했다. 잘못된 v1/v2 해시,
경계의 빈 코드·잔액 변경, 스토리지 네 슬롯의 개별 변경, v2 조회 오류를 기존 케이스가 통과시키는14개 RED를 확인했다.
원문 고정 Keccak 해시와 블록9/10을 직접 비교하고 네 슬롯·v2 refundableBalance 조회를 보강했다.
기존 prealloc 잔액·nonce 검사는 유지했다. 코드 교체가 선언되지 않던10번도 기존 Boho10/v2 프리셋을 쓰도록 고쳤다.
정상2·오류14의16개 단위 대조와 engine/helper 전체 race가 통과했다. 의도한 두 케이스의 주소 위치만
900→910으로 검토·갱신했고 최종 cache64 포함 전체 검사도 통과했다.

소스 `414cc74dcfa6c3f4bc3d91daca8bf923918433cd6fe68b60da85c362f220ea3d`의 실제
StableNet4d17eea0 Mac BP4·Boho10에서 두 공개 케이스/14단언을 race로 확인했다. 코드19124/19810bytes의
로컬 Keccak·RPC 해시·원문 고정값이 같고, 블록9/10 연결과 네 슬롯의 양수 값 보존·v2 조회를 독립 대조했다.
GovMinter 잔액은 전후0이며, 자금이 있는 계약 잔액을 보존했다는 주장은 아니다. 원문 기본 높이100과 현재10은 구분한다.
소스·보조 입력 불변, 실제 cache64 네 노드, 새 프로세스·열린 파일 조회와 DB0·206파일 속성 보존을 확인했다.
이는 두 원문 명제의 현행 로컬 보강이며 전체254/87·Linux·메인넷·S1해결·최종AST/WEB14 인수는 별도 잔여다.
[원문·RED/GREEN·실제·독립 증거](legacy-code-state-evidence.json).

**2026-10-16 새 StableNet·WBFT 원래 15노드 시작 — 전체 L1은 미완료:**
StableNet 4d17eea0·WBFT d535dd3d를 소유 서버 15대에 배치하기 전 새 프로세스·열린 파일과
원래 5개 바이너리 해시를 확인했다. 원래 호스트 bytes와 서버별 속성을 보존하고 두 파일씩 원자적으로 바꿨다.
기존 사용자 컨테이너는 변경하지 않았다. 소스
`de214fec706dde04ecc2ce717cfe3f63f2be0c23d79b7c2a9795cfd5c15fa0de`에서
원래 CT-NODE-002의 BP7·EN7·PN1을 두 체인에서 직렬 실행해 각각 pass1·다른 판정0을 확인했다.
체인 ID·블록 진행·검증자7·각15노드 peer를 대조했고, 실행 중 데이터 회수 거부를 확인했다.
종료 뒤 30개 압축·2,023엔트리·538개 로컬 파일과 15대의 새 프로세스·열린 파일·바이너리·DB0을 독립 대조했다.
소스와 보조 입력의 전후 지문도 같다. Mac 실제 실행이나 WEMIX 재실행·S1 해결·전체 L1·최종 Web 인수로 확대하지 않는다.
[배치·실제 실행·독립 보존 증거](upstream-fleet-startup-evidence.json).

**2026-10-16 새 체인 입력과 두 번째 L1 부분 배치 — 전체 인수는 미완료:**
소스 `908f853668e0eb1161b54f1f596b9914f992369b8274b845c300cc947a6ae58c`를 유지한
세 직렬 구간의 실제 32행은 30 PASS·1 BLOCKED·1 FAIL이다. 이전 행을 다시 실행하거나 판정을 바꾸지 않고
이어갔으며, 321행은 미실행이다. 32 workspace를 보존·회수한 뒤 압축 157개·10,523엔트리·로컬 파일 2,336개와
15대의 새 프로세스·열린 파일·바이너리·DB 조회를 독립 대조했다. CT-NODE-003 준비 제한과
CT-NODE-005의 600초 대기 실패는 원래 판정으로 남겼다. 후자는 reader 시작 전 부모 119의 seal 거절이다.
이는 S1 원인 확정이나 해결이 아니다. 공식 dev 변경을 확인해 케이스 경계에서 신호 없이 배치를 끝냈다.
[32행과 독립 증거](linux-l1-second-cohort-evidence.json).

공식 StableNet `4d17eea018c8dd579409db30bfd27de79c4f4121`과
WBFT `d535dd3d25110977ce19503d8fcbf7e9a5430562`를 별도 깨끗한 복사본에서 Mac·Linux ARM64로
빌드했다. 2,233/2,219개 Git 엔트리·원본/빌더 복사본·VCS 수정 없음·바이너리 SHA를 대조했다.
기존 외부 작업 트리는 수정하지 않았다. WEMIX는 확인한 4538a9f5를 유지한다.
이번 캠페인은 시작 시점의 이 커밋에 고정하며, 빌드 성공을 실제 구동·전체 L1·S1 해결로 세지 않는다.
빌드가 끝난 소유 Docker 캐시 1,032,376,320 bytes와 새로 받은 미참조 빌더 이미지를 회수했고 소스·바이너리·증거는 보존했다.
[소스와 네 빌드 출처](upstream-pinned-build-evidence.json).

legacy 원문은 다른 로컬 packages 저장소의 캐시된 `origin/feat/private-chain` 커밋 13b5e12c에서
셸 460개·1,548,814 bytes를 추출했다. 원본 HEAD 불변과 Git blob·모드를 독립 대조했다.
외부 셸 구문 그래프 460파일·16,747노드·31,392간선은 누락과 끊어진 간선이 0이다. 한 파서의 유효한 Bash 배열 glob
제한은 원본 실패를 보존하고 다른 고정 파서로 처리했다. 원본 소유자·mtime 복구나 과거 스냅숏 동일성,
내장 프로그램 파싱·동적 바인딩·실행 커버리지·254개 논리 테스트 동등성은 주장하지 않는다.
[분석 입력 복구와 한계](legacy-source-recovery-evidence.json). 9·11번과 최종 AST·WEB-14는 계속 열린 상태다.

**2026-10-16 R3/P4 노드별 컨트랙트 주소표 — 전체 능력 검증은 미완료:**
합의 관문과 달리 컨트랙트 관문은 실행 전체의 주소표를 썼다. 기존 코드에서 기록된WBFT후속 노드를
WEMIX주소표로BLOCKED, 미등록 체인을StableNet주소표로잘못통과, 체인 출처누락의3개 RED를 확인했다.
초기 포트bind권한 거절은 별도 환경 실패이며 제품RED로 세지 않는다. 각 노드의 기록된 체인을 선택하고
없는 옛기록만 실행체인으로fallback한다. 미등록 체인의 컨트랙트는다른 주소표를 조회하지않고unknown/BLOCKED다.
새/기존 관문race·engine/setup/MCP전체race·make check·e2e vet가 통과했다. 고정소스
`4f9e693adb2707b25c0040acb62d49ba506f74ba0b315e04c0df5575971bf5da`의 실제WEMIX→WBFT전환에서
기록된4후속노드와미등록체인·옛기록3프로필을47.331초native race로 확인했다.
두 양성은pass, 음성은정확한BLOCKED이며 Go실패/skip은0이다. 노드별 주소/코드SHA/길이와
원본spec/관측/상태를 독립대조했고 실제PID/체인/endpoint불변,새참조없음·DB3유형0·258파일속성보존을 확인했다.
작은triecache390776bytes는 보존했다. 계약ABI/버전/권한·혼합wbft-stablenet·전체L1·전체R3/P4·
최종AST/WEB14는 남는다. 앞37개회귀는 앞고정소스의범위를유지한다.
[회귀·실제·독립 증거](recorded-contract-capability-evidence.json).

**2026-10-16 과거 receipt 복구 검색 — 전체 보존 복구는 미완료:**
원래 경로의56개는 여전히28일치·26없음·2해시불일치다. 로컬 증거3영역의168후보 파일,
314압축 파일 및WEB08의Git2ref를 읽기 전용으로 대조해29/56의 동일 bytes 위치를 찾았다.
추가1개는WEB08 evidence이며3복사본과main662fff77의Git blob이 원래 기대SHA와 같다.
나머지27개는 이 범위에서 찾지 못했고 다른 머신/백업의 전역 손실을 주장하지 않는다.
압축 파일에는receipt가0개였고314개 전후SHA·mode·uid/gid·ns mtime·크기를 대조했다.
독립 재해시와보존23테스트가 통과했다. 원본을 덮어쓰거나 복구한 파일은0개이며,
복사본의 현재 속성을 원본 속성의 복구로 세지 않는다. 작업1과 새최종Web14는 계속 열림이다.
[원본 경로 감사·동일 복사본·검색 한계](historical-receipt-recovery-evidence.json).

**2026-10-16 현재37개 WEMIX 전체 회귀 — 수직 슬라이스 전체 완료는 아님:**
컴파일한 `^TestWemix` 목록37개를 소스
`d716e13b8061ef0d19a0f1a78a1ec6b61ed2a95ab26348da656236d8efff2122`에서
직렬 native race로 실행했다.3492.607초,37pass/0fail/0skip이며 소스·보조 입력 전후 지문이 같다.
독립 대조는 원본의37개 판정과 예상 ID 일치, GOV004/011의 실제 거래·Keccak 이벤트·원금/가스/nonce,
37runtime의 두 차례 새 프로세스/열린 파일 조회와 DB3유형0,8937파일의 bytes/mode/uid/gid/ns mtime와
파일 집합 일치를 확인했다.37개를 다시 실행한 독립 재현으로 세지는 않는다.
작은 triecache는 보존 자료로 남겼다. W3 진단 PASS는 재현 성공이며 외부 결함은 미해결이다.
옛20개·최근6개 증거는 원래 소스/범위를 유지한다. 전체87·공개DSL·legacy·전체L1·메인넷/R3/P4/R4/R5·
최종AST/WEB14와 외부S1/W3/W4는 이번 결과로 닫지 않는다. [회귀·독립 대조 증거](wemix37-regression-evidence.json).

**2026-10-16 승인 GOV-004·GOV-011 실제 출금 보강 — 전체87개는 미완료:**
사용자가 현행v1의 조기 `withdraw(0)` 성공/무출금과30초·70초 축약 기간을 승인했다.
원문의 revert 기대와72시간·7일 값은 지우지 않고 승인한 차이·범위를 덧붙였다.
실제 동일 흐름의 전액 unstake·정확한 제거 이벤트/INACTIVE, 조기 원금 미출금·자격/포인터 보존·정확한 가스,
만기 원금/이벤트/슬롯·포인터/nonce 장부를 검증했다. 별도 위임자는 delegate/undelegate 입력·양측 이벤트·금액·
staking power와30초 만기까지 검증했다. 원문259200/604800 설정 fixture의 getter와 raw storage도 대조했지만,
실제 메인넷 파라미터 조회나72시간·7일 대기를 주장하지 않는다.

증가만 보는 모델은15개 음성 대조에서 RED였다. 첫 실제 배치는5pass/1fail이며 새 검증기가 변경 없는
계약 사후잔액의 diff-mode 생략을 잘못 거절했다. 원본과6runtime을 보존한 뒤 실제 trace/고정 소스의 의미를
대조하고 생략 양성1개·누락/변경 음성4개를 RED→GREEN으로 보강했다. 호출자/입금 잔액 누락과 실제 계약 원금 감소는
계속 거절한다. 관련 race·e2e vet·전체 검사를 통과한 새 고정 소스
`ecb87e6302b83a0a36e714318b0864a74232f14821345695d4ec637b48be5da7`에서6개 실제 race를 다시 실행해
529.896초,6pass/0fail/0skip을 확인했다. 원본 거래/Keccak 이벤트/정확한 원금·가스의 독립 대조,
소스·보조 입력 전후 일치,6runtime의 새 활성 참조 없음·DB3유형0 및1368파일 bytes/mode/uid/gid/mtime 보존까지 통과했다.
약2.35MB triecache는 증명 자료로 보존했고 회수됐다고 세지 않았다. [실제·독립 증거](approved-unbonding-evidence.json).
원문 검토는85 scoped-confirmed·1충돌(TX-019)·1병합이며 전체 수락은 아니다.
새로 컴파일해 센 TestWemix 함수는37개다. 이번6개를37개 전체 회귀나 옛20개로 세지 않는다.
legacy 원본·메인넷/R3/P4/R4/R5·전체L1·외부S1/W3/W4·최종Web14는 여전히 남는다.

**2026-10-15 준비 확인의 전체 대기 시간 제한 보강 — 전체 L1은 미완료:**
353회 계획을 고정 소스 `e967b7e1687a83d94cce5c716e16d265bc5c374c4f6342dd6ef60ac8f63ac0de`에서
직렬 실행한 부분 결과는44통과·1실패·1운영자 중단이다.307행은 미실행이며 중단1행도 실제 판정을 다시 받아야 한다.
기존28행과 이후1행·17행을 원본 로그/아카이브와 대조해 가져왔으며 재실행한 것처럼 세지 않았다.
모든46 workspace의 자료를 보존하고 종료 DB를 회수했으며15서버 전수 확인도 완료했다.
보조 재개 스크립트의 이전 pause 표시 경로 오류와 최종 보고서의 잔여28 상수도 원본·수정 기록을 분리했다.
[부분 배치 증거](linux-l1-cohort-evidence.json).

StableNet4생산자가 부모107/다음108 및 부모2/다음3에서 previous prepared seal을 거절했다.
첫 snap 케이스는600초 대기 실패이며, 두 번째 이벤트 로그 케이스는 준비 확인73회/480초 뒤
신선한 PID·부모·UID·명령을 대조한 소유 CLI에 한 번의 정상 중단 신호를 보냈다. 실제 종료130이므로
스위프의 원시BLOCKED를 자연 발생한 제품 실패나 시간 초과로 확대하지 않는다. 외부 S1은 미해결이다.

준비 확인은 휴식만 예산에서 빼고 관찰 시간을 제외하고 있었다. 관찰·복구·휴식에 같은 deadline을
전달하고 늦은 READY를 거절했다. 부모 취소/부모 deadline은 오류로, 자체 deadline은 명시적 비준비 판정으로 구분한다.
원래 소스 overlay에서 관찰·재시작·늦은 READY·휴식4개 RED, 부모 오류 구분과 수정본 전체 nodemonitor race GREEN,
관련5패키지 race와 make check가 통과했다. 고정 소스
`ef85c60f513251e405ffd7b7cf473d81efbc1de7c7bf3a0ca3f399c2f7256995`의 실제 Linux4BP 정상 케이스가 통과했다.
같은 workspace에서4노드 채굴을 실제로 멈춰 RPC 응답/높이 정지를 확인한 뒤8초 제한이 종료2와 정확한 사유로 끝났다.
CLI 전체12.020초는 preflight 등을 포함하며 gate 자체8초와 구분한다. 소스·보조 입력 전후 같음,
4아카이브 원본/bytes·권한·uid/gid·ns mtime·링크 독립 대조와15서버 새 참조/DB0까지 통과했다.
첫 임시 case의 상대 preset 해석 오류는 네이티브 시작 전 fixture 실패로 별도 보존했다.
[시간 제한 증거](node-readiness-deadline-evidence.json). 새 변경 뒤 전체L1·R3/P4/R4/R5·원문87 전체·최종 Web14는 남는다. 승인 GOV-004/011 보강은 위 후속 기록에서 별도로 완료했다.

**2026-10-15 L1에서 발견한 attach 주소 변환 수정:**211파일·353회 계획의 첫 배치는
4행(3pass/1blocked) 뒤 중단했다. 원래 basic08의 host 체인은 정상 구동됐지만 스위프의 localmap
정규식이 블록형 YAML을 읽지 못해172.31.42.11:8601로 직접 접속했고, 실제 관측은
rpc_probe_unavailable이었다. 신선한 PID/부모/명령을 확인한 소유 스케줄러만 중단했다.
마지막 통과 케이스는 노드 종료 뒤 회수 훅이 완료되지 않아 멱등 종료·별도 신선한 보존/회수를
수행했고, 소스 수정 전15서버의 활성 참조/DB0을 확인했다.349행은 실행하지 않았고 전체L1은 미완료다.

기존 Go localmap 계약을 쓰는 선택적 `chain show --dial-rpc`/MCP `dialRPC` 조회를 추가했다.
원래 배치 Host/HTTP는 보존하고 rpcURL을 추가하며, 일반 배치 조회는 localmap에 의존하지 않는다.
스위프는 이 결과를 소비하고 YAML 정규식 해석을 제거했다. CLI/MCP의 부재·잘못된 bool과 실제
블록 YAML 스위프 실패를 RED로 재현했고, 흐름/블록·IPv6·invalid map·opt-in·읽기 계약 및
기존 회귀를 GREEN으로 보강했다. App/CLI/MCP/feature 전체 race와 전체 검사가 통과했다.
줄 수 문서의 실제 변화를 측정 도구로 반영했다. 고정 소스
`19274685939e5750cf6ad825877946f9c4760650811134331cc70e634a388c34`와 보조 입력 지문이
전후 같은 실제 basic08 재실행에서 CLI/stdio MCP가 같은127.0.0.1:46001 주소를 반환했다.
MCP 기본 조회·잘못된 bool 거절 및3개 제어 파일 무변경도 확인했다. 실제 attach2단언과
host setup3단언을 통과했고 pass1/fail0/blocked0/skip0, 종료·DB 회수까지80.053초다.
독립 원본·아카이브·5서버 조회에서 DB0,92파일을 보존했다. [실제 수정·대조·중단 기록](dial-address-query-evidence.json).
전체353회는 수정된 소스에서 다시 실행해야 하며 R3/P4/R4/R5·legacy·W3/W4·최종 Web14는 잔여다.

**2026-10-15 원래 Linux 포크·서명 교체2건 실제 통과:** 고정 소스
`f812907e751ac9f46497f3b1d14c4bf31accff7e2ae6b0107343c775421a260e`에서
원래 케이스를 수정하지 않고 직렬 실행했다. Boho 이전ad0122af0으로4BP를 시작하고 높이27에
현재0937ac5c로 재실행해, Boho200 이후 높이202와 컨트랙트 코드 변경을 확인했다(271초).
다음은 Boho 지원1.0.1 bf17c9607의4BP/1EN에서 실제 raw 거래를 보낸 뒤 EN을1.1.0으로 바꿔
clientVersion 변경·같은 거래의 blockNumber/status/from/to를 확인했다(50초).8단언·pass2/fail0/
blocked0/skip0이며 초기 배치를 포함437.316초다. 정확한 정방향 binary-for 바인딩과 기존
CLI/engine race 계약을 통과했다. 첫 배치 준비는 증명기가 런타임 GOPATH까지 전체 버전 문자열로
비교해 중단됐고, 같은 ELF/안정적인6개 신원 필드를 검증하도록 RED→GREEN으로 보강했다.
실패 원본·임시 ELF의 내용/권한/참조 부재를 보존하고 그 중복 사본 하나만 회수한 뒤 새 런타임에서
전체를 재실행했다. ELF5종의15서버 해시·원래3컨테이너 보존과8단언·거래/영수증·아카이브9개를
독립 대조했다. 각 실행은 다음 실행 전에 사전/사후 파일 명세 대조·종료 DB 회수를 마쳤고,
최종 신선한9개 서버 조회에서도 활성 참조/DB0이다.131파일과 별도 정리 명세를 보존했다.
[실제 전환·독립 대조](linux-upgrade-cases-evidence.json). raw RLP의 독립 암호학적 서명 복원까지
검증한 것은 아니다. 끝난 전용 빌드 캐시835,538,014bytes와 새 빌더 이미지 참조를 회수했고,
다른 사용자 자원·필요한 호스트 Go 캐시는 유지했다. 전체L1/R3/P4/R4/R5·legacy·W3/W4·최종 Web14는 잔여다.

**2026-10-15 원래 버전 교체 케이스의 Linux 입력 빌드:** Boho 이전 문서 지정
`ad0122af042d49022052a9783b3086d40d308db8`와 Boho 지원1.0.1
`bf17c9607a6b893ea85e6301e1cdb02d55b218de`의2133/2138파일을 Git 객체와 대조한 별도
압축 입력으로 준비했다. 원본 소스는 수정하지 않았고 사용자 미추적 파일도 보존했다.
고정 소스 `bfe3545456a4857113d32f1f1a4f8273ffca9ff21fac3c770491c449851db2f0`에서
두 입력을 실제 읽기 전용 Linux 볼륨에 복사해 전수 해시·실제 빌드·ELF 실행/버전·호스트 복사 해시를
각각113.232초/22.805초로 확인했다. 아카이브에는 .git이 없어 자동 Go VCS 대신 원래 저장소의
공식 internal/version 링커 변수를 사용했으며 그 차이를 기록했다. 독립 디코더의 옵션 출력 기대 오류와
첫 아카이브 권한 기대 오류는 보존했다. 독립 소스/출력 재대조 후 입력·ELF·21증거파일을 남기고
종료된 빌드 컨테이너2개와 소유 소스 볼륨2개를 회수했다. [출처·독립 대조](linux-legacy-build-evidence.json).
원래 포크/버전 교체의 실제 체인 수행·전체L1은 아직 미완료다. 후속 호환 케이스와 혼합 구현 입력
확인에 필요할 수 있는 이번 전용 빌드 캐시/이미지만 유지한다. 혼합5개용 wbft-stablenet 입력은
현재 세 소스/바이너리에서 확인되지 않아 사용자에게 위치를 질문했다. 다른 바이너리로 대체하지 않는다.

**2026-10-15 Linux 보존의 범위 제한·독립 파일시스템 대조:** 전역 dataRoot 대신
현재 composition의 node/runtime/logs와 공유 keys만 보관해 과거 실행의 중복 아카이브 증가를 막았다.
전역 루트/인덱스 자체 대조 모델은 범위와9개 메타데이터 변조 음성 대조에서 실패했다.
새 디코더가 `.`을 잘못 허용한 실제 실패 테스트도 보강해 거절하도록 수정했다.7개 보존 계약,
기존27개 runtime-storage와9개 스위프 회귀·전체 검사를 통과했다. GNU find의 사전 bytes/SHA·mode·
uid/gid·나노초 mtime·타입·링크·파일 크기를 PAX 아카이브 및 아카이브 직후 파일시스템과 대조해야
회수를 허용한다. 원래 basic07의5노드를 실제 Linux에서101.752초로 실행·종료·회수했고,
별도 사후 조회에서도 보존 파일 내용·권한 일치와 DB/캐시0개를 확인했다. DB 제거로 변경되는
부모 디렉터리 mtime만 사후 대조에서 구분했다. 고정 소스
`677ca507d64834568ccf83055a7d76143219dfa3edd21ed6d24ab2ac387aba27`가 전후 같았다.
아카이브5개75,641bytes·증거67파일 보존. [증거](linux-retention-evidence.json).
ACL/xattr 보존까지 검증한 것은 아니며 전체L1·외부 결함·최종 Web14 인수는 잔여다.

**2026-10-15 전량 스위프의 증거 보존·회수 관문:** 기존 스위프는 통과 workspace와
원격 로그를 제거하고 실패 DB는 남겼다. 실행 파일 경로인 `TCSWEEP_RETAIN_HOOK`을 지정하면
성공/실패 양쪽에서 종료 뒤 보존·회수를 호출하고 workspace를 남긴다. 종료·회수가 실패하면
다음 케이스를 실행하지 않고 code2로 중단하며 기존 workspace도 덮어쓰지 않는다.
고정 CLI 경로도 `TCSWEEP_BIN`으로 고른다. 기존 모델에서7개 실패를 RED로 확인했고,
수정 후9개 회귀(기존 훅 없는 성공/실패 정책 포함)와 전체 검사가 통과했다. 검사는 make check에
연결했다. 원래 basic07의5노드 케이스를 실제 Linux에서58.011초에 통과하고 설정·결과·로그57파일과
서버별 PAX 아카이브를 보존했다. 종료 후 프로세스·열린 파일·기록된 dataDir의 DB/캐시0개를
독립 대조했다. 고정 소스 `9a8520cb1f1729f4cce01bf5f5d25c10dbab77ac856d4fc3b1ef3cddc4b7519b`가
전후 같았다. [증거](sweep-retention-evidence.json). 훅은 자체적으로 소유·비활성·보존을 검증해야 하며
실제 전체 L1이나 실패 케이스 전수 인수는 이번 하위 증명에 포함하지 않는다.

**2026-10-15 세 체인 Linux15노드 시작·종료·DB 회수:** 고정 소스
`9a6466a731d181f8ae807eea740b15ea467f78dbeafa1801af5af74df4624015`에서
원래 CT-NODE-002를 StableNet/WBFT/WEMIX 순으로 직렬 실행했다. 각각111.369/115.401/278.418초에
BP7·EN7·PN1의15서버 배치, 체인 ID, 블록 전진, 검증자7개와15노드의 실제 peer 연결을 통과했다.
4GiB Docker VM에서 cache64만 명시했다. argv만 보던 모델이 놓친 cwd/fd/삭제된 fd/불완전·실패 조회를
RED→GREEN으로 보강하고 실제 실행 중 참조61/61/32건으로 회수 거부를 확인했다. 종료 뒤 각 서버의
두 차례 완전한 프로세스·열린 파일 조회, PAX 메타데이터 아카이브45개와 인덱스를 보존하고 DB/캐시를
회수했다. 독립 디코더는 보관 bytes·mode·uid/gid·PAX mtime·링크를 인덱스와 대조했다.
이는 아카이브/인덱스 일치이며 별도 사전 파일시스템 메타데이터 기준선까지 증명한 것은 아니다.
증명기 첫 대조는 onEach의15개 결과가 한 rpcCall 레코드에 묶인 실제 형식을18레코드로 잘못 예상해
실패했다. 실제4레코드와15개 노드 값을 엄격히 대조하도록 고쳤으며 체인 실행 결과는 변경하지 않았다.
마지막 독립15서버 조회에서 세 실행의 기록된 dataDir 아래 DB/캐시0개, 기존3컨테이너 상태 보존을
확인했고673파일을 보존했다. [증거](linux-chain-startup-evidence.json).
이번 WEMIX 시작 성공은 경합 결함 W3나 계량 결함 W4의 해결이 아니다. 전체L1/R3/P4/R4/R5,
원문·legacy·최종 Web14는 미완료다.

**2026-10-15 별도 Linux15대 가상 fleet 준비·실제 SSH/sudo 검증:** 기존3개 컨테이너를
재생성하거나 변경하지 않고 별도 네트워크·이름·150개 loopback 포트로15대 가상 서버를 세웠다.
개수만 확인하던 모델이 놓친10개 신원/해시/플랫폼/전송/경로/sudo/상태/디스크 음성 대조를
RED→GREEN으로 보강했다(기존 missing-host 거절 포함11개 음성 subtest). 공개 CLI의 실제 content
address와4개 잘못된 형식도 별도 RED→GREEN으로 확인했다. 첫 실제 관측은15대의 SSH/sudo/버전
확인 후 증명기의 sha256 접두사 비교 누락으로 실패했고 원본을 보존했다. 엄격한 prefix/payload/path
검증으로 고친 뒤15대 전체를 다시 관측해61.597초로 통과했다. 고정 소스
`31ccf3be59b1affd6c50e1244c634ed850f400e9cef40465c5892664215a78a5`와 입력 설정은
전후 같았다. 각 가상 서버는 지속되는 별도 machine-id/SSH host key를 가지며, 실제 CLI의 SSH 신원·
세 ELF의 hash/실행 가능·실행과 root600 파일의 실제 sudo 다운로드를 확인했다. 잘못된 비밀번호와
sudo 비허용은 실제 거절됐고 파일도 생성되지 않았다. Docker 원본·CLI 원본 응답·기존3개 상태를
독립 대조했다. [증거](linux-fleet-readiness-evidence.json).15대의 물리 서버나 블록체인 실행을
검증한 것으로 확대하지 않는다. DB/블록체인 프로세스는 아직 시작하지 않았고200파일을 보존했다.
fleet는 후속 직렬 체인 검증에 필요해 유지 중이다. 전체L1/R3/P4/R4/R5·원문·legacy·W3/W4·최종 Web14는 잔여다.

**2026-10-15 나머지 Linux 바이너리·소유 빌드 캐시 회수:** 직전 빌드 보호 계약을 다시
통과한 뒤 코드 변경 없이 WBFT7af50e45·WEMIX4538a9f5를 실제 Linux ARM64로 직렬 빌드했다.
고정 소스 `541e48f37e852503d7e868358da59cbee7f1f7768043fecd93fa067bd7b3547c`가
전후 같았고 WBFT2215개 파일, WEMIX2050개 파일·symlink1개를 읽기 전용 Linux bind 안에서도
전수 대조했다. WEMIX 원본은 수정하지 않고 공유 경로의 입력 사본2056항목을 대조했으며,
미초기화 gitlink5개를 같은 상태로 보존했다(초기 복사 준비 오류도 보존). 새 ELF의 서버1대
복사 해시·실행·기존 파일 tar 보존·원래 정지 상태 복귀를 각각33.387초/77.066초 실제 검증과
독립 대조로 확인했다. 앞선 StableNet을 합쳐 세 바이너리는 준비됐지만 fleet15나 전체L1은 아니다.
목적이 끝난 전용 Docker 빌드 캐시1,531,834,368bytes와 이번에 받은 Go1.25/1.19 빌더 이미지에
새 참조가 없는지 확인해 회수했다. 다른 adk-gomod 볼륨·기존 Go1.26.8 이미지·필요한 호스트 Go
캐시는 유지했다. 현재 디스크 여유는 실측36.6GiB이며 증명·바이너리·이전 파일은 보존했다.
[두 체인 출처·독립 대조·회수 기록](linux-remaining-build-evidence.json).
native 마감 당시 검증 루트3개의 DB0·16파일 보존(이동 명세는 별도 추가). 후속 전체 포맷 검사가
생성 입력 사본까지 읽어 실패해, 외부 코드를 포맷하지 않고 프로젝트 밖 보존 폴더로 이동했다.
2428항목의 내용·권한·UID/GID·mtime·inode를 재대조했고 기존 증명 경로는 링크로 유지했다.
Git 조회로 바뀐 디렉터리 mtime는 보존 명세의 값으로 복원해 재확인했다. 전체 포맷 검사는 다시 통과했다.
9번의15대 배치·전체L1/R3/P4/R4/R5, 10·11·14번과 최종 Web14는 잔여다.

**2026-10-15 Linux 빌드의 오래된 ELF 오배포 수정·실제 검증:** Docker 출력이 호스트에
보이지 않아도 기존 ELF를 복사하던 결함을 RED로 재현했다. 별도 빌드 임시 디렉터리에서
새 출력만 검증한 뒤 교체하도록 고쳤다. 실패 종료 코드(ELF 일부가 남는 경우 포함), 새 ELF 형식,
호스트 가시성을 모두 검사하고 실패 시 이전 파일을 보존한다. Bash EXIT 시 local 소멸과
command substitution의 errexit 예외도 실패 테스트로 확인해 처리했다. 빌드별 임시 폴더를 회수하고
작은 VM의 GOMAXPROCS/GOFLAGS도 컨테이너로 전달한다. 양성+실패4종·관련 resource/setup 전체
race·전체 make check가 통과했다. 소스 지문
`2f3d76ad80e4da5a2a9194b1f132eef3c3f028a89126c617b1f08de36d7710e0`을 고정해
실제 StableNet0937 소스2241파일을 Linux 읽기 전용 bind 안에서도 전수 해시 대조한 뒤 빌드했다.
새 ARM64 ELF를 소유 서버1대에 복사하고 해시·실행을 확인했다. 기존 서버 바이너리는 tar의
내용·권한·소유자·mtime와 함께 보존했고 서버는 원래 정지 상태로 돌렸다. 첫 실제 빌드/복사는
성공했지만 증명 스크립트가 tracked 변경과 Go vcs.modified를 같게 봐 출처 검사에 실패했다.
사용자 미추적 설정을 보존하고 두 상태를 구분한 뒤 동일 소유 캐시로 실제 재실행14.315초와
독립 대조를 통과했다. [증거](linux-fresh-build-evidence.json). 이번 단계는 StableNet1개 Linux
빌드·서버1대 범위이며 WBFT/WEMIX Linux·fleet15·전체L1·메인넷·최종 Web14는 미완료다.

**2026-10-15 W4 원격 확인 소스4538 재현·인계 보강:** 과거 로컬1350376 진단과 구분해
원격에서 확인했던4538a9f5의 깨끗한 소스·바이너리로 다시 재현했다(현재 원격 HEAD 재조회는 아님).
0 관찰만으로 잘못 판정하던17개 exporter/진행/헤더/피어/출처/초기화/해결 표기 음성 대조를
RED→GREEN으로 보강하고 engine/helper/collector race·정적 검사·전체 make check를 통과했다.
고정 소스 `c1318dd65c180a914e38345a7251ec73f3043b247221e80408085dfed0e827d1`에서
실제 진단 race142.493초/skip0·독립 대조가 통과했다. 공개 CT-NODE-007은 정확한 수신 metric
단언 하나로 pass0/fail1이며, 추가20블록 EN71→91/peer4와 BP hash/stateRoot 일치에도 두 원본
exporter의 broadcast/announce가0이다. 실제 LES→ETH init 순서와 소스4파일의 AST 중복 등록도
대조했다. 종료 DB0·129파일 보존. [증거](wemix-w4-latest-evidence.json),
[상세 인계](chain-handover-2026-09-12.md#6-1-w4--원격에서-확인했던-4538a9f5-바이너리-재확인-2026-10-15).
외부 소스 수정 없음·issueResolved=false·strictCaseAccepted=false이며 작업10과 W4는 미완료다.
이는 전체87·Linux·메인넷·최종 Web14 인수를 대신하지 않는다.

**2026-10-15 NODE-003/004 원문 cold full·snap 포크 replay 보강:** 높이/hash만 확인하는
모델이 놓친 거래 종류·raw/inclusion·cold 시작·진행·pivot·정확한 상태/영수증 32개 음성 대조를
RED로 확인하고 양성+32개를 GREEN으로 보강했다. 관련 CLI 전체 e2e race(100.762초), tagged vet,
전체 make check를 통과한 고정 소스 `0eb6dae8ea0f77f1fb0be6f21432e6c4c926141395949602d7716406db36a4b8`에서
실제 native race가 281.425초/skip0으로 통과했다. WEMIX WPOA→WBFT Croissant100 전후에
Legacy·Dynamic·FeeDelegation 전송·Dynamic 배포·쓰기 각각5종(총10건)을 채굴했다.
소스210에서 cold full을 시작해223까지, full 종료 뒤 소스224에서 cold snap을 시작해237까지 동기화했다.
비교 높이222/236의 hash/stateRoot, fork100 hash, 실제 snap pivot169, 같은 높이의10거래/영수증과
정확한 sender/payer 수수료·수취21ETH·nonce10/0·코드·저장값42/84를 독립 RLP/Keccak으로 재대조했다.
종료 후 새 프로세스 전수 조회로 미사용을 확인하고 DB0·309파일의 내용/권한/소유자/mtime를 보존했다.
[증거](crossfork-sync-evidence.json). 검토 상태는 confirmed_scope81→83·partial3→1이며,
GOV-011·원문 충돌·legacy·Linux·메인넷·latest 코드 원인·W3/W4·최종 Web14는 아직 잔여다.
이번 결과를 전체87 완료나 외부 결함 해결로 표시하지 않는다.

**2026-10-15 RPC-008 정확한 첫 반감·실제 proposer 원장 보강:** 양수/감소만 확인하는
판정이 설정·반감 비율·태그/채굴 높이·부모·proposer·실제 잔액·수수료/분배 격리의23개
오류를 허용하는 RED와 이를 거절하는 GREEN을 확인했다. 정적 검사가 지적한 조건식도
단순화했고 실행·구성·관측 race와 최종 실행 엔진 race·전체 검사를 통과했다.

첫 실제 시도는 준비 케이스 후 보상 RPC가 HTTP에 없어 -32601로 실패했다. 실패 자료를
보존하고 기존 보상 fixture와 같은 노드별 http.api/wemix 선언을 명시한 뒤 검사·실제 증명을
재통과했다. 실제 WBFT4/Croissant0/Brioche0에서 WBFT/Brioche 기본 보상을 모두2ETH로
명시하고 firstHalving10/period10/rate50/times4/finish1000을 실제 genesis와 대조했다.
공개 케이스 입력5/15와 경계9/10의 실제 채굴 블록에서 RPC2/2/1/1ETH와 proposer의 정확한
parent→mined 잔액 증가가 일치했다. 거래0·gas0·등록staker0·beneficiary없음을 확인했으며
독립 수치/헤더/신원/설정 대조도 통과했다. Brioche 활성화 경계를 넘은 증명은 아니다.

최종 실제 검증(55.53초, skip0)의 고정 소스
`ce82637ec29cc3b8728a56068d69824fd11c8f724a192ef1452cb4d0bbab4289` 전후가 같다.
[정확한 반감·잔액 증거](brioche-reward-evidence.json)는 실패/성공 파일398개의 내용·권한·
소유자·mtime을 보존했고 활성 프로세스·종료 DB는0개다. 원문87개는 확인 범위81·부분3·
충돌2·병합1이다. Brioche 활성화 이전/이후·단계 이관·WEMIX 레거시 보상 분배·공개 JSON
전체 실행·전체87·Linux·메인넷 live·legacy·latest 코드 원인·최종 Web 인수는 별도 잔여다.

**2026-10-15 GOV-009·010 원문 threshold5·target7·NCP 순위 전환 보강:** 기존 플래그
거절을 유지한 판정이 profile·신원·등록/NCP 수·고정 높이·금액·선정 집합/BLS·권한 투표의
31가지 오류를 허용하는 RED와 이를 거절하는 GREEN을 확인했다. CLI 전체 race·정적 검사·
전체 검사도 통과했다.

생성 BLS 키의 실제 WEMIX1+WBFT8, Croissant20/epoch10/UseNCPtrue/target7/threshold5에서
원문 순서대로 검증했다. NCP staker1/4명과 nonNCP H 등록 후4명(총등록5)의 epoch30/40/50은
stabilizingtrue·seed8을 유지했고, E/F/G까지 등록한 NCP staker7명의60은 false·target7이었다.
초기 NCP council은 계속7명이며 이 수와 활성 NCP staker 수를 혼동하지 않았다. 원문은
1/4/4/7 순서이므로 정확히5명인 별도 중간 에폭을 검증했다고 하지 않는다.

A40M·B35M·C30M·D25M·E20M·F15M·G10M·H10M WMX를 실제 등록하고 H를35M 더해45M로
만들었다. 최고액 H는 nonNCP인70에도 제외돼 A~G7명이 유지됐다. 정확한 과반수 NCP 추가
투표의 마지막78번 직후 첫80번 에폭에서는 H가 포함되고 최하위 G가 제외돼7명이 됐다.
여섯 상태의 등록·NCP·정확한 금액·계약 목록, raw14건의 hash/nonce/calldata·등록/증액/투표
이벤트, RLP·BLS/index·새 집합의 다음 블록 쿼럼6/5와 실제 노드 신원을 독립 대조했다.

실제 검증(93.226초, skip0)의 고정 소스
`a5b9a9e40cbf913b7030484274c4d53f4b4044aecb9871e3b246e2734fcca8fb` 전후가 같다.
[선정·안정화 증거](ncp-selection-evidence.json)는 파일457개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0개다. 원문87개는 확인 범위80·부분4·충돌2·병합1이다.
기존 공개 JSON 전체 실행·전체87·Linux·메인넷 live·legacy·latest 코드 관측 원인·최종 Web
인수는 별도 잔여이며 임의 threshold/churn 전수로 확대하지 않는다.

**2026-10-15 GOV-007·008 NCP 제거/자기 탈퇴와 첫 다음 에폭 제외 보강:** 멤버 수·
제외 여부만 보고 기존 실패 영수증 거절을 유지하는 모델이 경로별26개 오류를 통과시키는
RED를 확인했다. 이후 더 강한 판정도 늦은 에폭을 허용하는 두 RED를 추가로 확인했고,
첫 다음 에폭만 허용해 양성2개·음성54개를 GREEN으로 만들었다. 초기 문법 실패는 의미 있는
RED로 세지 않았다. CLI 전체 race·정적 검사·전체 검사를 재통과했다. 잘못된 기존 NCP
가결 설명도 실제 계약의 floor(n/2)+1로 고쳤으며 동작 코드는 바꾸지 않았다.

실제 WEMIX→WBFT Croissant20/UseNCPtrue에서 NCP7명과 등록된 네 스테이커로 검증했다.
43번 제안과44~47번 찬성표에서 3표까지 미가결, 4표째 정확한 type2/NewProposal/Vote/
NCPRemoved/Finalized(true)와 council7→6을 확인했다. 6명 상태의 자기 탈퇴54번은 한 거래에
NewProposal/NCPRemoved/Finalized만 있고 투표가 없으며 council6→5가 됐다. 각각 최종 거래
직전/영수증 높이의 전체 stakerInfo·BLS·minimum1e25·활성·운영자 매핑이 정확히 유지됐다.
첫 다음 epoch50/60의 후보·BLS에서 대상 스테이커만 제외됐고, API51/61와 실제 헤더는
밸리데이터4→3→2·쿼럼3/3/2로 계속 생산했다. 독립 raw Keccak/nonce/calldata·영수증/
canonical 포함·이벤트 ABI·RLP/index/BLS/bitmap·첫 에폭 공식 대조도 통과했다.

첫 실제 관측도 통과했지만 첫 에폭 판정을 보강한 뒤 최종 소스로 다시 실행했다.
최종 검증(108.132초, skip0)의 소스
`5c83e59a2962a09119a2778d16cbdae35dbf0c1f7601a63b95e884b644b0bf89` 전후가 같다.
[제거·탈퇴 증거](ncp-removal-evidence.json)는 두 실행 파일458개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0개다. 합의4노드·독립 NCP 운영자·target4/threshold4의
로컬 fixture이며 7노드 합의·원문 threshold5 안정화·순위·공개 JSON 전체 실행은 별도다.
원문87개는 확인 범위78·부분6·충돌2·병합1이며, 전체87·Linux·메인넷 live·legacy·latest
코드 관측 원인 조사·최종 Web 인수는 별도 잔여다.

**2026-10-15 GOV-006 7명 NCP 과반수 투표 보강:** 최종 멤버 수·후보 포함만 보는
판정의 RED를 확인했다. 기존 실패 영수증 거절도 포함한 모델로 다시 대조한 결과 추가
누락은23가지이고, 기존 실패 status 거절1가지는 새 누락으로 세지 않았다. 양성·총24개
음성 조건, CLI 전체 race·정적 검사·전체 검사가 통과했다.

실제 WEMIX→WBFT Croissant20 이후 서로 다른 NCP 운영자7명에서 제안26번·투표27~30번을
확인했다. 정확한 type1/NewProposal·Vote와 원문 입력을 대조했고, 3표까지 Voting/accepts3·
목록7·후보false를 유지했다. 4표째의 같은 성공 거래에 Vote/NCPAdded/ProposalFinalized(true)가
정확히 있으며 Accepted/accepts4·목록8·후보true·제안ID 증가가 즉시 반영됐다. 각 거래의
영수증 높이 고정 상태, raw Keccak·nonce·calldata·canonical 포함·이벤트 ABI의 독립 대조도
통과했다. 합의 노드는4개이고 NCP7 운영자는 별도 계정이다. 7노드 합의·staker 등록·제거·
자기 탈퇴·순위·안정화와 기존 공개 JSON 전체 재실행은 이 증명의 범위가 아니다.

실제 검증(78.341초, skip0)의 고정 소스
`1f8b424c3157726b27c2a491b6daa46bcccbd4e1374d7cb03435418922793d5e` 전후가 같다.
[7명 과반수 증거](ncp-majority-evidence.json)는 파일229개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0개다. 원문87개는 확인 범위76·부분8·충돌2·병합1이다.
전체87·Linux·메인넷 live·legacy·latest 코드 관측 원인 조사·최종 Web 인수는 별도 잔여다.

**2026-10-15 WBFT-005·GOV-005 원문 등록/에폭 변경 보강:** epoch 존재만 확인하는
판정이 잘못된 높이·등록·실제 stake·집합·BLS/index 등의21가지 오류를 통과시키는 RED와,
이를 거절하는 GREEN을 확인했다. 컴파일만 실패한 overlay는 의미 있는 RED로 세지 않았다.
관련 CLI 전체 race·정적 검사·전체 검사가 통과했다. 첫 실제 시도는 빈 키링 암호 때문에
WEMIX 배포가 입력을 요구해 준비 단계에서 실패했고, 다음 시도는 고정 tip1wei 거래의
전 노드 pending 관측에 실패했다. 두 실패 자료를 보존했고 명시 암호·노드 제안 수수료로
fixture를 고친 뒤 관련 회귀·정적 검사와 실제 race를 다시 통과했다.

생성 키의 WEMIX1+WBFT5노드, Croissant100/epoch10에서102 높이를 멈춰 첫 신규 GovStaking
등록을 정확히103에 채굴했다. 나머지는104·105·106에 등록됐고 네 거래의 원문 minimum1e25,
등록 전102의0·등록 후 영수증 높이의 totalStaked1e25, input/receipt/event/BLS/PoP/proxy가
정확했다. 109의 초기 seed5 집합이110에서 등록된 네 staker/validator/BLS로 갱신되고
stabilizing=false이며,111 RPC가 그 집합을 사용했다. 일반109/111의 epochInfo는 명시 null이다.
독립 RLP의9필드·epoch 내용·raw Keccak·이벤트 ABI·이전5/다음4의 쿼럼까지 일치했다.
등록 대상은 초기 seed였으므로 seed 외부 노드 편입으로 확대하지 않는다. target4/threshold4/
useNCPfalse와2초 주기를 명시한 로컬 fixture이며 기존 공개 JSON 전체 재실행은 아니다.

실제 검증(139.086초, skip0)의 고정 소스
`e37a6c3fae80bf3b7eb7d984b766bd2dc4fd79f604b6e373063ea6d43df06a98` 전후가 같다.
[등록·에폭 증거](registered-epoch-evidence.json)는 세 시도 파일704개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0개다. 원문87개는 확인 범위75·부분9·충돌2·병합1이다.
전체87·Linux·메인넷 live·legacy·latest 코드 관측 원인 조사·최종 Web 인수는 별도 잔여다.

**2026-10-15 WBFT-009 이전 블록 서명자 집합 보강:** count>=3만 확인하는 판정이
잘못된 부모 조회·블록 연결·밸리데이터 수·쿼럼·중복/외부 주소·집계서명 형식의14가지 오류를
허용하는 RED→GREEN을 확인했다. 기존 쿼럼 장애 경계와 관련 실행 엔진·노드 관측·체인 구성
전체 race·정적 검사·전체 검사도 통과했다.

생성한 BLS 키를 쓰는 실제 WBFT Croissant0 5·7노드에서 각각 부모13~15/자식14~16 세 쌍을
조회했다. 부모 높이의 실제 밸리데이터는 생성된 live 신원과 일치했고 두 이전 서명자 집합은
모두 그 부분집합으로 중복 없이 쿼럼4/5를 충족했다. 독립 RLP에서 child.prevRound는
parent.round와 같고 이전 bitmap·96byte signature와 API 주소 대응이 정확했다. 부모 현재
seal의 모든 비트도 자식 이전 seal에 남아 있었다. 7노드의 부모 집계5명이 다음 헤더에서
7명으로 병합된 실물을 확인했다. 따라서 늦은 seal 병합을 지원하는 체인 계약대로 집계서명
바이트의 무조건적 동일성은 요구하지 않았다. 일정한 seed 집합이며 집합 변경 경계는 별도다.

실제 두 구성 검증(75.043초, skip0)의 고정 소스
`e118fbd6b5a18eaefb02b6bb754f0695b42ce09a44127ddae3fa98da26b1ebe4` 전후가 같다.
[이전 서명자 증거](previous-sealers-evidence.json)는 파일634개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0개다. 기존 공개 JSON 전체를 실행한 것은 아니다.
원문87개는 확인 범위73·부분11·충돌2·병합1이며, 전체87·Linux·메인넷 live·legacy·latest 코드
관측 원인 조사·최종 Web 인수는 별도 잔여다.

**2026-10-15 RPC-020·021 실제 WebSocket 내용·개수·지연 보강:** 개수만 확인하는
판정이 acknowledgement·관측 범위·헤더 연결·필터·영수증·수신 지연 등의 추가20가지 오류를
허용하는 RED→GREEN을 확인했다. 기존 빈 로그 거절은 유지했고 양성·21개 음성 조건과
구독 프로토콜·거래 생애주기 회귀가 통과했다. 첫 실제 시도는 latest 배포 코드 조회가0x여서
WS를 시작하기 전에 중단됐다. 정확한 원인은 미확정이며 생산 결함 해결로 표시하지 않는다.
영수증의 다음 canonical 블록을 기다린 뒤 영수증 블록에서 코드를 읽도록 관측을 고정했다.
고정 태그·블록 해시·주소·성공 영수증·비어 있지 않은 코드의 양성/5음성 조건을 추가했고,
실패 직전 배포 행과 raw도 보존하도록 보강했다. 미사용 import로 실패한 컴파일도 고쳐
관련 CLI 전체 race·정적 검사·전체 검사를 다시 통과했다.

실제 WEMIX→WBFT Croissant20 이후 WS node2에서44번 높이 뒤 구독을 열고45~49의 다섯
생성 블록과 다섯 newHeads를 정확히 대조했다. number/hash/parentHash와 부모 연결이 고정
HTTP canonical 블록과 같았다. 공개 LOG1 runtime을 실제 두 주소에 배포했고, address+topic
필터가 다른 주소의 동일 topic 성공 로그를 제외하면서 대상 영수증과 정확히 같은 로그를
한 건 받았다. 대상 SDK 호출 전부터 수신까지의 상한291.625ms가 실제 BlockPeriod1초보다
짧았다. 이 상한은 서명·큐 대기·합의도 포함하며 초 단위 header.timestamp를 빼서 지연을
추정하지 않았다. 모든 환경의 통지 SLA로 일반화하지 않는다.

실제 증명(67.657초, skip0)의 고정 소스
`1d826d73661a39a1c85fd210e72ab116dc4fc35e8065d648b276c77c93f587b8` 전후가 같다.
[WS 내용·지연 증거](websocket-witness-evidence.json)는 첫 실패 시도까지 두 폴더의 파일460개
내용·권한·소유자·mtime을 보존했고 활성 프로세스·종료 DB는0개다. 첫 실패의 배포 raw/영수증은
그때의 검사 전에 보존하지 못했다는 한계도 기록했다. 기존 공개 JSON 전체를 실행한 것은
아니며 runtime만 재사용했다. 원문87개는 확인 범위72·부분12·충돌2·병합1이다.
전체87·latest 코드 관측 원인 조사·Linux·메인넷 live·legacy·최종 Web 인수는 별도 잔여다.

**2026-10-15 TX-001 같은 거래의 전체 생애주기 보강:** 채굴 status만 확인하는 판정이
잘못된 pending·전파·생산 제어·블록 포함·확정·잔액/수수료 등의 추가20가지 오류를 허용하는
RED를 확인했다. 기존 실패 status 거절은 유지했으며 양성·21개 음성 조건으로 GREEN을
확인했다. 기존 raw 거절과 관련 CLI·계정 전체 race·정적 검사·전체 검사도 통과했다.

체인 분석의 MinerAPI 연결과 실제 WBFT 소스를 대조해 miner_start의 인자 없는 서명을
사용했다. 실제 WEMIX→WBFT Croissant20 후 소유한 네 후속 생산자의 채굴을 일시 중지했다.
높이25가 유지되는 동안 같은 type0x02·1WMX·gas21000 raw 거래가 네 노드에서 아직 채굴되지
않았고 각 pending.0에 정확히 있었다. 생산을 재개한 첫26번 블록의 영수증 인덱스에 한 번만
포함됐으며, 29번까지 진행한 뒤 네 노드의 canonical 블록·stateRoot·성공 영수증이 일치했다.
수신액은 정확히1WMX이고 sender는 value와2100006402270000wei의 가스비만 부담했다.
nonce는0→1이었다. 독립 raw RLP/Keccak 해독과 실제 baseFee+cappedTip 계산도 통과했다.

실제 증명(77.027초, skip0)의 고정 소스
`f308916db726874b31bf9495d939466ea16ce89a890b3fce5bceda17666fbd90` 전후가 같다.
[거래 생애주기 증거](transaction-lifecycle-evidence.json)는 파일229개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0개다. 신규 지갑의 개인키를 증거에 남기지 않았다.
명시적인 생산 일시 중지 제어를 사용한 새 native 대응 테스트이며 프록시 토폴로지의 기존
공개 JSON을 재실행하거나 바꾼 것은 아니다. 원문87개는 확인 범위70·부분14·충돌2·병합1이다.
전체87·Linux·메인넷 live·legacy·최종 Web 인수는 별도 잔여다.

**2026-10-15 TX-012·014 실제 raw 거절·해시 부재 보강:** 어떤 오류 메시지든 허용하는
판정이 잘못된 노드 오류·raw 해시·조회 출처·null 응답·블록 진행·상태 변경 등의 19가지 오류를
허용하는 RED→GREEN을 확인했다. 조회를 수행하지 않은 것을 null로 착각하지 않도록 네 개의
실제 응답과 method·phase·해시를 요구했다. 처음 native 파일을 컴파일할 때 존재하지 않는
GasPrice 도우미를 사용한 오류도 고쳐 eth_gasPrice 호출로 재검사했다. 관련 CLI·계정·DSL 전체
race, 기존 입력 가림·정책 판정 회귀, 정적 검사·전체 검사가 통과했다.

실제 WEMIX→WBFT Croissant20 이후 높이24의 gasLimit104488313을 읽고104488314로 서명한
legacy raw는 정확한 exceeds block gas limit 오류로 거절됐다. 유효한 type0x16 이중 서명에서
Sender.S만 xor0xff로 바꾼 raw는 정확한 fee delegation: invalid feePayer 오류로 거절됐다.
두 건 모두 실제 raw Keccak 해시로 거래·영수증을 즉시와3블록 뒤 조회해 정확한 null을
확인했고, 새로 할당한 비생산자 계정의 nonce·잔액은 그대로였다. Sender.S만 정상으로 복원한
대조군은 같은 nonce로 status1 채굴됐다. 독립 RLP 비교에서 나머지 필드와 payer 서명은 같았다.
현행 서명 결합 오류를 원문의 ErrInvalidSig 명칭과 동일하다고 쓰지 않는다.

실제 증명(79.031초, skip0)의 고정 소스
`d6c6e6266d7a5f980684e1c6c0f3bff29a91328a0574be91fbea258d7e225647` 전후가 같다.
[raw 거절 증거](raw-refusal-evidence.json)는 파일229개의 내용·권한·소유자·mtime을 보존했고
활성 프로세스·종료 DB는 0개다. 기존 공개 JSON을 이번 실행에서 raw 제출·hash null 검사로
전환하거나 재실행한 것은 아니다. 새 native 대응 테스트의 범위를 확인했으며 원문87개는
확인 범위69·부분15·충돌2·병합1이다. 전체87·Linux·메인넷 live·legacy·최종 Web 인수는 잔여다.

**2026-10-15 RPC-022 원문 에폭110 보강:** EpochInfo 존재만 확인하는 판정이 잘못된
원문 좌표·조회 높이·헤더 연결·빈 멤버·다른 주소·다른 BLS 키·인덱스·null 형식 등의
15가지 오류를 허용하는 RED→GREEN을 확인했다. 기존 에폭 모양·포크 헤더·거버넌스 원문
경계 회귀도 통과했다.

실제 WEMIX→WBFT Croissant100·EpochLength10에서 109·110·111을 고정 조회했다. 두 일반
블록의 epochInfo는 명시적 null이고 110에는 정확한 네 seed 스테이커·밸리데이터와 프리셋
48byte BLS 키가 들어 있었다. 독립 canonical RLP 해독으로 원시 헤더 세 개(510·822·510byte),
실제 부모 연결·epoch payload/null·주소·diligence·인덱스·BLS·commit quorum과 RPC 응답을
대조했다. threshold5 미만 네 seed 멤버의 안정화 유지이며 신규 등록이나 집합 변경을
검증했다고 주장하지 않는다. WBFT-005·GOV-005는 별도 잔여다.

Tagged CLI 전체 race·정적 검사·전체 검사와 원문 거버넌스·경계 검증 및 기존 포크20 안정화
실제 회귀(220.294초, skip0)가 통과했다. 고정 소스
`6bcfd5c73a7bf9fa32c4c9e413bb9a5eea83f37fbd54862da3eb007142a50465` 전후가 같다.
[원문 에폭 증거](original-epoch-evidence.json)는 두 폴더 파일456개의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는 0개다. 원문87개는 확인 범위67·부분17·충돌2·병합1이며
전체 인수로 표시하지 않는다. Linux·메인넷 live·legacy·최종 Web 전수 인수는 별도 잔여다.

**2026-10-15 이관 후 공개 거래·P256 다섯 케이스 보강:** 합계만 확인하는 판정이
누락·신원·순서·프리셋·숨은 상태 등의 8가지 오류를 허용하는 RED를 확인했고, 정확한 다섯
케이스와 9개 음성 조건으로 보강했다. 첫 실제 시도는 다섯 공개 케이스가 모두 성공했지만
저장된 명세를 추가 대조하는 검사가 기존 key/saveKey 가림 처리를 예상하지 못해 실패했다.
가림 처리는 유지하고, 실행 전 원문 동작 일치와 저장 후 예상 가림 결과를 구분했다.
양성 대조와 입력 키 참조·대상·저장된 동작·신원 변경 5개 음성 대조가 통과했다.

새 고정 소스에서 WEMIX→WBFT Croissant20 후속 노드 네 개에 원문 동작을 그대로 실행해
TX-004·009·011·015·020의 단언 14개·동작 28개를 확인했다. 실제 거래·영수증 6개를 독립
대조했으며 type0x16 송금에서 sender는 1ETH만, payer는 정확히 2,100,000,298,557,000wei의
수수료만 차감됐다. 잔액 부족과 변조 FeePayer 서명은 원문 동작이 제출 거절의 필수 사유를
강제한다. 별도의 raw 오류 응답 파일을 보존했다고 주장하지 않는다. 공개 P256 유효160byte는
정확한 word1, 짧은159byte와 빈 입력은 각각 0x였다. 네 후속 노드의 실제 유효/무효 지원
관문도 통과했다. 은퇴한 WEMIX 노드를 제외한 범위이며 TX-019 충돌은 별도 잔여다.

Tagged 관련 전체 race·정적 검사·전체 검사와 실제 공개 케이스 및 기존 등록 회귀
(127.351초, skip0)가 통과했다. 고정 소스
`08b5e725619848e3b0b89a6113cfa72f810f3cd5c32bcd5aa17ee4bf8631bfed` 전후가 같다.
[이관 후 공개 케이스 증거](postfork-public-evidence.json)는 첫 실패 시도까지 4개 폴더의
파일 1,012개 내용·권한·소유자·mtime을 보존했고 활성 프로세스·종료 DB는 0개다.
87개 상태는 확인 범위66·부분18·충돌2·병합1이며 전체 인수로 표시하지 않는다.
Linux·메인넷 live·legacy·최종 Web 전수 인수는 별도 잔여다.

**2026-10-15 NODE-005 재시작 주기·서명 재참여 보강:** 연속 링크만 확인하는
판정이 평균 간격, 실제 신원, 조회 블록, commit quorum 등의 11가지 오류를 허용하는
RED→GREEN을 확인했다. 기존 짧은 창·잘못된 링크 거절은 유지했고, 13개 음성 조건과
기존 재시작 준비·view-change 판정 회귀를 통과했다.

실제 WBFT Croissant0·BP4·BlockPeriod1에서 공개 CT-FAULT-001의 원문 동작을 그대로 실행했다.
네 단언·여덟 동작이 정지 중 진행, catch-up, hash·stateRoot 정합성을 검증했다. 실제 node3은
동일 바이너리·명령·데이터 경로에서 PID42830→42924와 revision 증가로 재시작됐고 원래 키
주소가 같았다. 고정 높이의 resync hash/root가 일치했다. 이후 26~31번의 여섯 헤더에서
다섯 timestamp 간격 모두 1초·Round0이며, 조회한 header와 정확히 묶인 commit quorum에 원래
노드가 참여했다. 재시작 endpoint도 마지막 31번 hash를 확인했다. 후보는 모두 보존했고 첫
여섯 후보가 회복 창을 충족했다. 즉시 항상 첫 창이 통과한다고 일반화하지 않는다.

실행 엔진 전체 race·정적 검사·전체 검사와 실제 회귀(35.455초, skip 0)가 통과했다.
고정 소스 `76ba5d35f1ef513eafbf5f6712a720359477b2ae0fcf6ee9806178502d631673`
전후가 같다. [재시작 회복 증거](restart-participation-evidence.json)는 파일 199개의 내용·권한·
소유자·mtime을 보존했고 활성 프로세스와 종료 DB는 0개다. 기본 세션의 단언 파일명은
assert.json이며 별도 native fixture의 assertions.json과 구분했다. 단계적 이관·원문 87개 전체·
Linux·메인넷 live·legacy·최종 Web 인수는 별도 잔여다.

**2026-10-15 NODE-006 실제 NCP 공백 파싱 보강:** 초기화 성공만 확인하는 판정이
잘못된 활성화·블록 진행·조회 시점·수·membership·목록·입력의 12가지 오류를 허용하는
RED→GREEN을 확인했다. 목록 순서에 의존하지 않는 집합 대조도 통과했다. 실제 이관은
Croissant20·useNCP=true이며, 공백이 포함된 일곱 주소 CSV가 생성 genesis와 다섯 nodegenesis에
그대로 보존됐다. 높이 25에서 ncpCount7, 정확한 ncpList 집합, 일곱 trim 주소의 isNCP word1과
0 주소의 word0을 독립 대조했다. 실제 체인은 이후 높이 27까지 진행했다.

Tagged CLI 전체 race·e2e vet·전체 검사와 실제 이관 검증(73.467초, skip 0)이 통과했다.
고정 소스 `40556e48fd021f690ce311ecd2aa3cb295e13f166e9c6416a5bc5c8a0f0f518c`
전후가 같다. [NCP 공백 실제 증거](ncp-whitespace-evidence.json)는 파일 228개의 내용·권한·
소유자·mtime을 보존했고 활성 프로세스와 종료 DB는 0개다. 초기화만 확인한 과거 단계와
이번 RPC·블록 진행 증명을 구분한다. 원문 87개 전체·Linux·메인넷 live·legacy·공개 DSL·
최종 Web 인수는 별도 잔여다.

**2026-10-15 NODE-007 초기화 거절 원인 보강:** 기존의 어떤 실패든 통과시키는
판정이 다른 genesis 오류, 빈 출력, 실행 실패, signal/timeout, 잘못된 인자, 불완전한 사유의
6가지 오류를 허용하는 RED→GREEN을 확인했다. 정상 종료한 노드 프로세스의 비정상 코드와
정확한 `govNCP is configured but no initial NCPs provided` 사유를 함께 확인한다.

검증된 WBFT 바이너리에서 useNCP=true·빈 목록은 init exit1/정확한 사유였고, 7개 공백 패턴
주소는 init exit0이었다. Tagged 관련 race·e2e vet·최종 전체 검사와 실제 두 초기화
(6.448초, skip 0)가 통과했다. 첫 전체 검사의 새 주석 규율 실패도 고쳐 재검사했고 로그를
보존했다. 고정 소스 `9f65ea8a600b16a904e06f881023b454d0f73f8676fa5aeb301dbde3b4ba3490`
전후가 같다. [NCP 초기화 증거](ncp-init-evidence.json)는 실제 명령·바이너리 SHA·genesis·
종료 코드·전체 출력의 파일 14개 내용·권한·소유자·mtime을 보존했다. 공통 소유권·공간 검사와
종료 후 DB 회수를 사용하며, 활성 프로세스와 종료 DB는 0개다. NODE-006의 실제 ncpCount,
isNCP, 블록 생산과 원문 87개 전체·Linux·메인넷 live·legacy·최종 Web는 별도 잔여다.

**2026-10-15 이관 이후 활성화 높이 준비 판정 보강(§0a 9·11):** 이전 실제 자료에서
네트워크 높이 21·후속 node4 높이 19인데 일반 2블록 허용치 때문에 READY였고, 이어진 합의
probe는 unknown이어서 case가 blocked였다. 같은 조건을 단위 테스트에서 재현했다. 기존
판정은 기다리지 않았고 영원히 활성화되지 않는 노드도 즉시 OK로 처리하는 RED였다.

노드 모니터에 절대 최소 높이를 연결했다. 일반 허용치는 유지하고, handed-over 후속 노드가
선언된 포크 높이에 도달해야 통과한다. 포크 전 주차, restart 방식의 활성화 전 교체,
역할을 끝낸 이전 producer, 활성화 이후 관측 오차, 치명적 오류 우선순위와 대기 예산을
보존하는 회귀를 통과했다. 실행 엔진·노드 모니터·체인 구성 전체 race·e2e vet·최종 전체 검사와
실제 이관 20/100 회귀(225.259초, skip 0)가 통과했다. 두 실행 모두 최소 높이 때문에 기다린
실제 로그가 있고, case probe에서 후속 노드 네 개는 각각 모두 23/103이며 합의 기능도
supported였다. 재시도는 없었다. 인위적 실제 노드 지연을 주입한 것은 아니며, 단위 gate가
정확한 21/19 혼합 상태의 대기·시간 제한을 결정적으로 검증한다.

고정 소스 `9feb35e4777ca9137eb1c06010b86689e5f127174d0d140a0ee8c1486db61a7a`
전후가 같다. [활성화 준비 증거](fork-readiness-evidence.json)는 이전 실패 자료를 참조하고,
새 2개 실행 파일 456개의 내용·권한·소유자·mtime을 보존했다. 활성 프로세스와 종료 DB는 0개다.
첫 전체 검사는 구조 문서의 줄 수가 16줄 뒤처진 것을 잡았고, 기존 측정 도구로 수치를 갱신한 뒤
통과했다. 이로써 포크 전 높이가 ready로 통과하던 구체적인 후속 잔여를 보강했다.
그 밖의 모든 RPC 지연·W3/W4·원문 87개 전체·Linux·메인넷 live·최종 Web 인수 해결은 아니다.

**2026-10-15 RPC-009 등록 전후 읽기 보강:** 양수만 확인하는 판정이 잘못된 설정,
minimum, total 증가량, 블록 조회, 등록 거래의 10가지 오류를 허용하는 RED→GREEN을 확인했다.
실제 등록 영수증의 부모·채굴 블록을 고정했다. minimum getter는 생성 설정 1e25와 같고,
totalStaking은 0→1e25로 정확히 증가했다. 기존 등록 input/value/receipt/event와 Rewardee의
새 배포도 독립 대조했다. Tagged CLI 전체 race·e2e vet·최종 전체 검사와 실제 등록 검증
(118.358초, skip 0)이 통과했다.

첫 전체 검사는 충분성 검토 테스트가 records[0]을 항상 partial로 가정해 실패했다.
NODE-001의 현행 상태를 유지하고 open/scoped 상태로 검사 대상을 선택하도록 고쳤다.
목록을 뒤집거나 미완료 항목을 앞에 두어도 16가지 오류를 거절하는 회귀를 추가했고,
검토 테스트 6개가 통과했다. 최초 실패 로그를 보존했다. 고정 소스
`4b122ec9d8d08689bcce4e52f5e49a392bec9248af623004de7a79a6d9c4f45b` 전후가 같다.
[등록 전후 읽기 증거](staking-read-evidence.json)는 실패한 준비 시도를 포함한 2개 실행의
파일 454개 내용·권한·소유자·mtime을 보존했다. 활성 프로세스와 종료 DB는 0개다.

**새 준비 상태 후속 잔여(§0a 9·11):** 첫 이관 준비에서 node1~3은 높이 21의 합의 기능이
supported였지만, node4는 높이 19에서 unknown/rpc_probe_unavailable이어서 case가 blocked였다.
일반적인 ready와 실제 포크 이후 합의 준비 상태를 구분해 후속 검토한다. 같은 소스의 재시도
성공을 간헐적 문제 해결로 표시하지 않는다. 원문 87개 전체·Linux·메인넷 live·legacy·공개 DSL·
최종 Web 인수는 별도 잔여다.

**2026-10-15 NODE-001 원문 헤더 경계 보강:** presence만 확인하는 판정이 잘못된
높이/부모/Extra/포크전노드거절/epoch/API-to-wire12조건을 허용하는 RED→GREEN을 확인했다.
실제 원문Croissant100 이관에서99의31bytes WPOA clientvanity와 정확한-32000/
block is not a wbft block 거절을 확인했다. 100의622bytes Extra는Croissantv1의9필드RLP이며
Randao65·committed signature96·bitmap3명·epoch4validator/BLS48·staker/diligence/stabilizing/
round/vanity를 독립 RLP해석으로 API와 대조했다. 실제99hash→100parent linkage도 일치했다.

Tagged CLI 전체 race·e2e vet·전체 검사와 실제 원문 경계/기존GOV001·002배포·슬롯 회귀
(147.048초, skip0)가 통과했다. 고정 소스
`ab44f6042b2750b6f665e364e5de21ed977eb59f94d3ad7dffc2ba4eb8c0ab7d` 전후가 같다.
[헤더 경계 증거](fork-header-evidence.json)는228파일의내용·권한·소유자·mtime을 보존했고
활성 프로세스·종료 DB는0이다. 원문87개 전체·Linux·메인넷live·legacy·공개DSL·최종Web는
별도 잔여다. 이 정상 이관 성공을 과거W3 partialRegistry reader 결함의 해결로 확대하지 않는다.

**2026-10-15 GOV-001·002 원문 배포 경계·설정 보강:** presence/positive만 확인하는
판정이 잘못된 높이/배포전code/누락계약/코드/설정/슬롯/getter16조건을 허용하는 RED→GREEN과
원문100대신축약20을 선언하던 입력 RED→GREEN을 확인했다. 실제 WEMIX→WBFT 이관은
원문Croissant100이며 높이99에서 네 주소code0x, 높이100에서 GovConfig282/GovStaking20556/
GovRewardeeImp2261/GovNCP5501bytes를 확인했다. 100의슬롯0~3·getter는 min1e25/max1e26/
staker604800/delegator604800과 정확히 같고 생성genesis와다섯 nodegenesis도 독립 대조했다.

Tagged CLI 전체 race·e2e vet·전체 검사와 실제 원문 배포/기존fork20 auto출금 회귀
(217.536초, skip0)가 통과했다. 경계에서 후속 노드의99수신 지연이 관찰됐으나 같은 시도에서
99수신→100합의→102이상 진행했다. 로그는 보존하며 이 성공을 과거W3결함 해결로 확대하지 않는다.
고정 소스 `2ed58bd224b15d2aae7f79f16e9aa0b8e98bcf92ea9d9a276c0e8723c0cacba1`
전후가 같다. [원문 배포·설정 증거](governance-fork-evidence.json)는2실행/456파일 내용·권한·
소유자·mtime을 보존했고 활성 프로세스·종료 DB는0이다. 604800초는 실제 설정값 조회이며
7일출금 대기는 아님. 원문87개 전체·Linux·메인넷live·legacy·공개DSL·최종Web는 별도 잔여다.

**2026-10-15 GOV-004 원문 auto mode 출금 대조:** 원문 withdraw(0)을 실제 전송해
만기13초 전 status1/no events/no payout·slot/pointers 불변과 정확gas를 확인했다.
실제 부모/채굴 블록의 계약잔액은 모두 원금2e25와 같다. 만기1초 후 같은 입력은 동일slot의
Withdrawn index0/amount2e25·원금−gas·contract debit·slot0/pointers1로 출금됐다.
원문의 조기 withdrawal time not reached revert 기대는 실제 현행v1 auto mode와 충돌하므로
GOV-004는 conflict·기준 결정 대기로 남긴다. 현행관찰 통과를 원문 요구 완료로 세지 않는다.

Success만 확인하는 판정의15조건과 블록별 잔액 누락/차이2조건 RED→GREEN을 확인했다.
Tagged CLI 전체 race·e2e vet·전체 검사와 실제 auto/explicit/expiry3개 회귀(279.707초, skip0)가
통과했다. 최초실행은 diffMode가 변동없는 계약을 생략한 것을 판정이 잘못 가정해 실패했고
로그/고정 소스/종료자료를 그대로 보존했다. 부모/채굴 잔액 직접 조회로 보강한 뒤 재검증했다.
고정 소스 `c5fc3a6881fd886f3366350ce96639000249b209fa696b016d8400fd7fd4ac48`
전후가 같다. [자동 출금 증거](auto-withdrawal-evidence.json)는 실패 포함6실행/1368파일의
내용·권한·소유자·mtime을 보존했고 활성 프로세스·종료 DB는0이다. 메인넷7일·원문87개 전체·
Linux·legacy·공개 DSL·최종 Web 인수는 별도 잔여다.

**2026-10-15 GOV-004 명시 count 출금 보강:** success만 확인하는 판정이
잘못된 시각/설정/금액/slot/pointers/조기 거절/출금13조건을 허용하는 RED→GREEN을 확인했다.
비검증자 operator의 전액2minimum 해제 뒤 동일 credential을 15초 chain-time fixture로 검증했다.
만기13초 전 withdraw(1)은 status0과 정확한 insufficient withdrawable credentials로 채굴됐고
slot/pointers가 그대로였다. 만기1초 후 같은 index의 exact Withdrawn·원금2e25−gas·contract debit·
slot0/pointers1을 독립 대조했다. Tagged CLI 전체 race·e2e vet·전체 검사와 실제 출금/expiry
회귀(184.238초, skip0)가 통과했다.

고정 소스 `418eda0aa6a50ece40b62bdc49611ca8bd1324c0cb89818e2813f844095fe457`
전후가 같다. [명시 count 출금 증거](unstake-withdrawal-evidence.json)는456파일 내용·권한·
소유자·mtime을 보존했고 활성 프로세스·종료 DB는0이다. 원문의 withdraw(0)은 현행 계약의
별도 auto mode이므로 이번 withdraw(1) 거절을 원문 입력의 거절로 표시하지 않는다.
withdraw(0)의 실제 만기 전/후 결과와 원문 사유 차이는 별도 잔여다. 메인넷7일·원문87개 전체·
Linux·legacy·공개 DSL·최종 Web 인수도 별도 잔여이며 GOV-004는 partial을 유지한다.

**2026-10-15 GOV-016 전액 unstake 입력·이벤트 보강:** 잔액0만 확인하는 판정이
잘못된 입력/금액/활성 상태/이벤트/receipt13조건을 허용하는 RED→GREEN을 확인했다.
기존 이관 테스트는 원문대로 minimum2배를 등록·unstake한다. 실제2e25 input/value0/status1,
Unstaked amount2e25·StakerRemoved(emitter/subject/unique/data), active true→false,
stake2minimum→0을 독립 대조했다. Tagged CLI 전체 race·e2e vet·전체 검사와 실제
전액 해제/기존 미달 거절 회귀(148.711초, skip0)가 통과했다.

고정 소스 `4442968ac09277d1423c6284c84a09bfe4d990b247a37de8bb69bdd7291e53fa`
전후가 같다. [전액 해제 증거](full-unstake-evidence.json)는456파일 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. GOV-004의 activefalse·두 이벤트 누락 부분도 보강했지만
조기 withdraw의 정확한 채굴 거절/상태불변과 만료후 동일 credential 출금 index/amount/ledger는
남는다. 원문87개 전체·Linux·메인넷7일·legacy·공개 DSL·최종 Web 인수는 별도 잔여다.

**2026-10-15 GOV-003 등록 자동배포 보강:** active만 확인하는 판정이 잘못된
거래/금액/이벤트/BLS/코드/구현/mapping14조건을 허용하는 RED→GREEN과 기존 배포 주소를
새 배포로 오인하는1조건 RED→GREEN을 확인했다. 기존 최소금액 등록 흐름에 정확한 거래
input/value/receipt·StakerRegistered ABI·BLS48/PoP96·active/operator mapping을 연결하고
등록 부모블록code0x/등록블록code155bytes·implementation slot0x1002/구현코드를 직접 대조했다.

Tagged CLI 전체 race·e2e vet·전체 검사와 실제 이관 등록/기존 보상 ledger 회귀
(136.171초, skip0)가 통과했다. 실제 최소1e25 입력·이벤트의 emitter/subject/operator/rewardee/
feeRecipient/feeRate0/amount/dynamic BLS offset/padding을 독립 계산했다. 고정 소스
`bb0b8b0846e4162ec45eb86a5e0395a83f12a59d9ba5fe32b0937d6b9dc4bb2d` 전후가 같다.
[등록 배포 증거](staker-registration-evidence.json)는456파일 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 원문87개 전체·Linux·메인넷·legacy·공개 DSL·
최종 Web 인수는 별도 잔여다.

**2026-10-15 공개 core 케이스 최신 실증:** 목적에 맞는 기존 단언을 미구현으로 세지
않고 실제 publiccase를 fresh 실행했다. aggregate pass7만 확인하는 판정이 누락/중복/다른ID/
seq/preset/skip/fail8조건을 허용하는 RED→GREEN을 확인했다. 관련 실행 엔진 전체 race·전체
검사와 세 체인의 BP4 EN1 native race(216.117초, skip0), 정확21case·93단언이 통과했다.
블록 진행·nonce1증가·새계정의정확잔액/수수료·양수가격/base+tip·세 계약 배포/저장42/view비변경
공개case를 실행하고 개별 명세/ID/seq/preset/상태·거래/영수증·실제 code/view/slot0을 독립 대조했다.
RPC-001/012/015/016·TX-005의 fresh core 의미를 확인했다. TX-005의 London호환 손코드
fixture와 원문 ABIselector 차이는 유지하며 원문절차까지 동일하다고 표시하지 않는다.

고정 소스 `7c227fe682284523844460f8b6af2bd0998bb237f6d4a89e0ff914492f0e6452`
전후가 같다. [공개 core 증거](public-core-evidence.json)는767파일 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 기존 publiccase bytes를 바꾸지 않았다. WBFT는
Croissant0 직접 구성이며 원문87개 전체·Linux·메인넷·legacy·단계적 이관·최종 Web 인수는
별도 잔여다. 과거 W3 진단을 이번 BP4 EN1 성공으로 해결됐다고 바꾸지 않는다.

**2026-10-15 RPC-007 영수증 출처 보강:** 기존 status/price/logs 판정이 잘못된
receipt/transaction block hash·번호·missing hash·0/invalid gas7조건을 허용하는 RED→GREEN을
확인했다. 공개 케이스는 양수 gas와 고정한 번호의 실제 블록/거래 hash·번호 대조를 추가했다.
관련 전체 race·최종 전체 검사와 세 native race(192.844초, skip0)·공개 단언24개가 통과했다.
실제 거래·영수증·두 방식의 전체 블록 조회·포함 거래가 일치하고 gasUsed는 양수/실제 한도
이내이다. 생성된WBFT croissantBlock0 이후 높이를 직접 읽어 대조했고 두 회귀 패밀리는
실제 londonBlock0 설정을 대조했다. 존재하지 않는 StableNet anzeonBlock을 가정하지 않는다.

고정 소스 `94ecae8d478547ff897e61caa78d1b99913726e6a524a33b62b45d6e523c2daf`
전후가 같다. [영수증 출처 증거](receipt-origin-evidence.json)는501파일 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 기존 주소 corpus bytes는 바뀌지 않았다. WBFT는
Croissant0 직접 구성이며 단계적 이관·원문87개 전체·Linux·메인넷·legacy·최종 Web 인수는
별도 잔여다.

**2026-10-15 TX-007 배포된 slot0 접근 목록 보강:** 기존 BALANCE/eth_createAccessList
회귀를 유지하면서 원문의 contractAddr/slot0 입력을 추가했다. 잘못된 주소·슬롯·배열·거래/영수증
타입·저장값7조건 RED→GREEN을 확인했다. 최초 actual은 로컬 sendTx 생성의 수신자 제한으로
세 체인 모두 실패했다. 전용 deployContract로 바꾼 다음 시도는 WEMIX만 통과했고 StableNet은
SDK의 기본 수수료 상한13.52e18 upfront·WBFT는 기본 access-list 시뮬레이션5e18 예산으로
준비가 실패했다. 이를 제품 결함/전체 통과로 세지 않고 두 시도 모두 accepted=false로 보존했다.

전용 배포 계약과 bounded 준비 입력 RED→GREEN, 이전 상수 예산을 세 quote에서 실행한
RED→GREEN을 확인했다. 현재 노드 quote×500000+1e18로 준비 금액을 계산하고 배포의
실제 gasPrice·시뮬레이션gas100000/quote hex를 선언한다. 최종 관련 전체 race·전체 검사와
세 체인 native race(164.546초, skip0), 공개 단언39개가 통과했다. 원문 type1의 단일 생성주소/
slot0 접근 목록·영수증status/type·7바이트 SSTORE 런타임·실제 채굴 높이의 slot0=42를 확인했고,
원문RLP·Keccak 거래hash·ECDSA 복원sender를 독립 대조했다.

고정 소스 `37107141ed39e0bc1db1e0a4aa7e242fe873c119060aa5c40dd0fe630a5ed957`
전후가 같다. [배포 slot0 증거](access-list-evidence.json)는 실패 포함9실행의1533파일 내용·
권한·소유자·mtime을 보존했고 활성 프로세스·종료 DB는0이다. 주소 corpus는 의도한 새 참조3곳만
갱신했다. WBFT는 Croissant0 직접 구성이고 단계적 이관·원문87개 전체·Linux·메인넷·legacy·
최종 Web 인수는 별도 잔여다.

**2026-10-15 RPC-003/004/010/011/013 합의 조회 연결 보강:** presence 판정 모델이 잘못된
노드/역할·신원·제네시스ID·에폭·쿼럼·연결19조건을 허용하는 RED→GREEN을 확인했다.
초기 import 준비 오류는 별도 보존하고 제품 RED로 세지 않았다. 최초 실제 실행은 공개4케이스가
통과했으나 서명자 API를 배열로 decode한다고 잘못 가정해 전체 증명에 실패했다. 실제 객체의
Number/Hash/Author까지 헤더에 연결하는 추가4조건 RED→GREEN을 확인했고, 준비 중 구조체
필드 누락 오류도 별도 보존했다. 관련 실행 엔진 전체 race·최종 전체 검사가 통과했다.

수정된 actual native race(34.255초, skip0)에서 공개4케이스·단언9개와 BP4/EN1의
공개키→Keccak 노드ID/주소·metadata·관리API 신원 연결을 독립 대조했다. 높이21 BP4=true,
EN=false이며 실제 생성 genesis.chainId8284와 모든5노드 RPC가 같다. epoch10의9/10/11·
19/20/21에서 tagged validators=target4의 BP주소집합, BlockSigners 객체=실제헤더,
Committers>=3·중복 없음·검증자 부분집합을 확인했다.

고정 소스 `b602471a8445dec67ab5d13ec24549b44ca1ceaadcc31c89dfe2c5ad6609eb05`
전후가 같다. [신원·에폭 조회 증거](identity-epoch-evidence.json)는 최초 실패도 accepted=false로
남기고540파일 내용·권한·소유자·mtime을 보존했으며 두 실행의 활성 프로세스·종료 DB는0이다.
단언만 있는 is-validator case는 동작0개라 steps.json을 만들지 않는 실제 기록 계약을 확인했다.
원문87개 전체·Linux·메인넷·legacy·단계적 이관·최종 Web 인수는 별도 잔여다.

**2026-10-15 GOV-013 승인 기준의 운영자 청구 보강:** 사용자는 원문 operator 프리뷰0과
현행 v1의 스테이커 지분 프리뷰를 구분하는 기준을 승인했다. 원문 절차는 보존하고 해당 절에
승인한 차이/대조 기준을 덧붙였다. 기존 잔액 증가 판정 모델이 잘못된 프리뷰·등록 mapping·
위임 상태·이벤트·gas·출금·nonce·input14조건을 허용하는 RED를 확인하고 GREEN으로 보강했다.
등록 운영자는 비검증자, 위임자는 없으며 실제 claim 부모 블록0x25에서 두 preview와
등록 mapping·self/total stake를 조회했다. operator 보상0·스테이커 보상12e18, fee0이다.

정확 Claimed/RewardPaid와 FeePaid 부재·Rewardee12e18 출금·운영자12e18−실제 가스비
14402303986268594Wei 수령·nonce를 거래 단위 diff로 독립 대조했다. tagged CLI 전체 race·
e2e vet·전체 검사·실제 WEMIX→WBFT 운영자/위임자 회귀(174.296초, skip0)가 통과했다.
고정 소스 `b103570970f1a33cd0b5676100ab76c37641c3d7b6a4777446eb9f420d489a90`
전후가 같고 [운영자 청구 증거](operator-claim-evidence.json)는456파일의 내용·권한·소유자·
mtime을 보존했으며 활성 프로세스·종료 DB는0이다. 원문87개 전체·Linux·메인넷·legacy·
공개 DSL·최종 Web 인수는 별도 잔여다.

**2026-10-15 RPC-014 반환 로그 대조 보강:** 필터·개수·topic만 확인하던 공개 케이스가
잘못된 주소/높이/블록 hash/거래 hash/removed/data6종을 허용하는 RED를 확인했다.
영수증과 직접 대조하도록 보강한 정상1/음성6종 GREEN·관련 전체 race·전체 검사와
세 체인 native race(157.925초, skip0), 공개 단언30개가 통과했다. LOG1 배포·실행 영수증과
조회 로그의 주소·번호·블록/거래 hash·topic·data·removed·인덱스를 독립 대조했고,
기존 topic 양성/음성 필터와 다른 topic0건 조건을 유지했다.

고정 소스 `7ee7a7f780ebc1497169e5626ababe372f4cd81e164afd714edb5df941863954`
전후가 같다. [반환 로그 증거](event-log-evidence.json)는501파일 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 주소 corpus의 bytes 변경은 없다. WBFT Croissant0
직접 구성이며 단계적 이관·원문87개 전체·Linux·메인넷·legacy·최종 Web 인수는 별도 잔여다.

**2026-10-15 RPC-006 50블록 통계 보강:** 필드 존재 판정 모델이 잘못된 범위·합계·
미등록/중복 서명자·쿼럼 미달·헤더16조건을 허용하는 RED를 확인하고 정확한 판정으로 GREEN을
확인했다. 관련 실행 엔진 전체 race·전체 검사와 실제 native race(55.044초, skip0)가 통과했다.
현재 WBFT Croissant0 BP4에서 원문1..50 입력을 조회하고 50개 블록의 author·round와
Extra의 네 seal 종류별 주소를 재합산했다. author50·round50·currentCommitted150(각3이상),
prepared150·prevPrepared161·prevCommitted161·total622가 RPC 통계의 각 주소별 값과 같다.
기존 공개 presence 단언4개도 유지해 실제 실행했다. 합계 oracle는 native 테스트에 있다.

고정 소스 `a3596a2c83640f1b87b90d6eb1f33bc1633c1e1c585713359c12fb9afaae1c31`
전후가 같으며 [합의 통계 증거](consensus-statistics-evidence.json)는196파일 내용·권한·
소유자·mtime을 보존했고 활성 프로세스·종료 DB는0이다. 단계적 이관·원문87개 전체·Linux·
메인넷·legacy·최종 Web 인수는 별도 잔여다.

**2026-10-15 RPC-002 전체 블록 조회 보강:** 기존 latest 두 조회의 경합·false 입력과
extraData/miner/거래 객체 누락을 공개 케이스에서 보강했다. 정상 입력의 protocol 및 잘못된
응답7종을 기존 단언이 놓치는 RED를 확인했고, 정상1/음성9종 GREEN·관련 전체 race·전체
검사를 통과했다. 실제 거래의 채굴 번호를 고정하고 두 조회 모두 true로 호출하며
hash·number·extraData·miner·transactions와 전체 응답 일치를 독립 대조했다.

세 체인 native race(154.715초, skip0), 공개 단언30개가 통과했다. 고정 소스
`41a53f442dfad8b376a3737bca885a1a3b7fc4f1ccb56ddf412d836912de4bd4` 전후가 같다.
[전체 블록 조회 증거](block-lookup-evidence.json)는501파일의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 주소 corpus 갱신은 기존 faucet/새 수신자2곳뿐이다.
WBFT는 Croissant0 직접 구성이다. 단계적 이관·원문87개 전체·Linux·메인넷·legacy·최종 Web
인수 완료로 확대하지 않는다.

**2026-10-15 GOV-023 개별 credential 만료 보강:** 머신 시계 sleep·큰 잔액 증가로
처리 항목을 추정하는 조건을 실제 체인 시각·슬롯 상태·포인터·정확한 이벤트/출금 ledger로
바꿨다. 잘못된 시간/처리/포인터9조건과 threshold/이벤트/슬롯/입력/가스비6조건의 RED→GREEN을
확인했다. 위임 setup을 먼저 마친 뒤 긴 unstake/짧은 undelegate credential을 만들고,
쿼리한15/5초 기간과 실제 생성 블록 timestamp·requestTime·withdrawableTime을 연결한다.

관련 tagged CLI 전체 race·전체 검사와 실제 native race(95.220초, skip0)가 통과했다.
첫 출금 timestamp1792021917은 짧은 만료1792021916 이후·긴 만료1792021924 이전이며
slot1만 삭제·slot0 그대로·withdrawalIndex0을 확인했다. 두 번째1792021925는 긴 만료 이후
slot0 삭제·withdrawalIndex2가 됐다. 각 Withdrawn 인덱스1/0·amount1e25·caller 수령−실제가스/
contract 출금·nonce를 독립 대조했다. 고정 소스
`123c42867acb9a6733077c0d9fbe4e5c3c457fbdb859997830de08da26875180` 전후가 같다.
[개별 만료 증거](credential-expiry-evidence.json)는228파일의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 원문은 block 기반 표현을 쓰지만 현행 v1은 timestamp
초 단위이므로 차이를 명시하며 원문 block-number 기대까지 받아들인 것으로 표시하지 않는다.
원문87개 전체·Linux·메인넷 기간·legacy·공개 DSL·최종 Web 인수는 별도 잔여다.

**2026-10-15 GOV-012 보상 누적·전환 옵션 보강:** 단순 잔액 증가 baseline이 짧은 구간·
잘못된 합계/RPC·헤더 연결·대상 검증자·다른 거래·등록 계정·등록 목록을 통과시키는 RED→GREEN을
확인했다. 원문의 RPC 호출은 세 실제 시도에서 실패했다. 최초 구성은 명시 Brioche 객체와 HTTP
namespace가 없었고, 추가 선언/값을 실행 명령과 실제 설정에서 확인해도 전환 뒤 모듈이 사라졌다.
전환 재실행의 직접 Argv 생성이 기록된 launch override를 건너뛰는 것을 확인했다.
지원되는 옵션의 누락/잘못된 옵션 무시 RED→GREEN을 보강하고, 공통 검증·조합 경로로 연결했다.
현재 역할·노드·호출 시 설정의 우선순위를 유지하고, EN→BP 승격 뒤에도 필요한 RPC는 node1~4
범위로 선언했다. 옵션 없는 기존 handoff 유지도 검증했다. 첫 단위 준비의 unsupported verbosity와
패키지 줄 수 문서 불일치도 보존하고 지원되는 maxpeers/측정 문서 갱신으로 바로잡았다.

관련 chainsetup 전체·nodeconfig·tagged CLI 전체 race·전체 검사와 실제 native race(91.380초,
skip0)가 통과했다. 실제 전환 후4개 BP의 기록 args와 live wemix 모듈, 등록 이벤트의 rewardee·
단일 등록 stake·15개 연결된 무거래/적격 블록의 원문 RPC 합계15e18 wei와 고정 높이 잔액 차이가
일치하는 것을 독립 대조했다. 명시 보상은 기본값과 같은1e18이고 별도 수혜자 배분은 없다.
고정 소스 `9fddc3a76b17168e914f2a51e3501796a328b10c31bd4a6c7ed199766282ed5b` 전후가 같다.
[보상 누적 증거](block-reward-evidence.json)는 세 실패 포함4실행912파일의 내용·권한·소유자·
mtime을 보존했고 활성 프로세스·종료 DB는0이다. R4의 이 옵션 재실행 범위도 보강했지만 전체
R4/Linux/mainnet 인수가 아니다. 원문87개 전체·legacy·공개 DSL·최종 Web 인수는 별도 잔여다.
동일 높이의 GOV-013 원문 operator 프리뷰0과 staker 지분 프리뷰17e18은 별도 입력 충돌로
사용자 결정 대기이며 이 보상 누적 완료에 포함하지 않는다. 외부 소스는 수정하지 않았다.

**2026-10-15 GOV-014 정상 위임자 청구 보강:** 잔액 증가만 보는 baseline이
누락/중복 이벤트·잘못된 amount/restake/fee·계정별 잔액 차이·프리뷰 시점·fee 비율 등11조건을
통과시키는 RED를 확인했다. 비검증자 운영자/위임자/feeRecipient, 동일 stake·feeRate250/
precision10000, 위임 뒤10블록 보상 누적과 정확한 청구 ledger를 GREEN으로 보강했다.
청구 거래 index0의 parent 상태에서 previewReward를 다시 읽어 실행 직전의 총액/수수료를 묶고,
Claimed·FeePaid의 emitter/indexed/data와 거래 단위 계정/가스비/nonce를 전수 대조한다.
Claimed.amount는 계약 수수료 차감 뒤의 값이며 거래 가스비는 별도다.

관련 tagged CLI 전체 race·전체 검사와 실제 위임자 청구 native race(89.377초, skip0)가
통과했다. 프리뷰 총액6e18·수수료1.5e17 wei, Claimed5.85e18, rewardee6e18 출금,
feeRecipient1.5e17 수령, 위임자5.85e18−gas13,515,606,368,415,564 증가를 독립 재계산했다.
고정 소스 `65016892622acd06dccfad84a2b7ddf083d4dc1169be18b39511c41a65480382` 전후가 같다.
[정상 청구 ledger 증거](claim-ledger-evidence.json)는228파일의 내용·권한·소유자·mtime을
보존했고 활성 프로세스·종료 DB는0이다. 원문87개 전체·Linux·메인넷·legacy·공개 DSL·최종
Web 인수와 다른 비율/반올림 구성 전수는 별도 잔여이며 외부 소스는 수정하지 않았다.

**2026-10-15 GOV-022 타 스테이커 청구 차단 보강:** receipt 실패만 보는 baseline이
누락된 상태 추적·보상 계정/타 운영자 변경·호출자 보상 수령·잘못된 nonce/사유·보상 미누적·보호
계정 누락을 통과시키는 RED를 확인했다. 두 스테이커 등록·각 보상 누적, OperatorA→B와 무관한
계정C→A의 실제 채굴 실패·정확한 사유·거래 단위 prestate diff를 GREEN으로 보강했다.
블록 보상 유입은 전역 latest 잔액에 섞이므로 실패 거래 자체의 변경을 직접 검사한다.
두 rewardee·OperatorB·GovStaking은 거래 diff에서 변경되지 않고 호출자는 정확한 가스비만
차감되며 nonce가1 증가한다. 기존 증가만 보는 조건과 임의 Execute 오류 통과는 제거했다.

관련 tagged CLI 전체 race·전체 검사와 무권한/정당한 운영자 청구 두 native race(199.182초,
skip0)가 통과했다. 두 보상 계정의3e18/1e18 wei와 두 호출의 정확한 ABI·status0·revert·상태/
가스비를 독립 대조했다. 최초 준비 실패는 보존했고 재시도 뒤에 검증했으며 준비 경합/W3 해결로
표시하지 않는다. 고정 소스 `ce94ceeb27d61eb34231b74851a7f8013ff2c08f93dd57c841bf3ca5ad829b55`
전후가 같다. [청구 차단 증거](claim-guard-evidence.json)는3실행682파일의 내용·권한·소유자·
mtime을 보존했고 활성 프로세스·종료 DB는0이다. 정당한 청구 성공 회귀는 GOV-013의 정확한
금액/이벤트 전체 완료를 의미하지 않는다. 원문87개 전체·Linux·메인넷·legacy·공개 DSL·최종 Web
인수는 별도 잔여이며 외부 소스는 수정하지 않았다.

**2026-10-15 GOV-015·017 채굴 거절 보강:** 원문의2minimum 예치/minimum+1 언스테이킹과
useNCP=true 긴급 모드 stake·unstake·delegate 세 동작을 명시 가스로 제출하도록 보강했다.
추정 오류·불완전/중복 동작·잘못된 사유/입력/상태를 통과시키는 RED→GREEN을 확인했다.
최초 native는 언스테이킹만 통과했고 긴급 모드에서 정확한 채굴 revert 뒤 가스비 등식이 실패했다.
해당 실패 블록의 proposer가 송신자인 것을 로그로 확인했다. 비검증자 운영자 계정을 별도 자금으로
구성하고 실제 검증자 목록 밖임을 확인하도록 보강해, 가스비·환불의 정확한 등식을 유지했다.
최초 긴급 모드 준비도 consensus rpc_probe_unavailable로 한 번 blocked됐으며 재시도/실패를 보존한다.

보강 후 관련 tagged CLI 전체 race·전체 검사와 실제 두 native race(160.708초, skip0)가 통과했다.
총 네 채굴 실패의 정확한 입력·status0·부분 가스·ABI revert 사유·stake/delegation/active 불변·
환불·가스비, 긴급 모드 해제 투표와 GOV-018 재개 거래의 정확한 이벤트도 독립 대조했다.
고정 소스 `fdc8a5b856e73cd87f31964a863eaec380a9ae3b2fae68b1ead9e3ff0bac552d` 전후가 같다.
[거버넌스 차단 증거](governance-guards-evidence.json)는 최초 실패 포함5실행의1,138파일 내용·
권한·소유자·mtime을 보존했고 활성 프로세스·종료 DB는0이다. W3 또는 이전 경합 해결로
표시하지 않는다. 단일 NCP 로컬 범위이며 원문87개 전체·다중 council/Linux·메인넷·legacy·공개 DSL·
최종 Web 인수는 별도 잔여다. 외부 체인 소스는 수정하지 않았다.

**2026-10-15 RPC-023 검증자 제외·복귀 보강:** static 역할/flag만 보는 baseline이
다른 주소·접속점·과거/미적용 에폭·RPC와 실제 목록 불일치 등 여덟 조건을 통과시키는 RED를
확인했다. 동일 노드·접속점·적용 후 높이, 기본/특정 높이 isValidator와 실제 getValidators
집합을 연결하는 GREEN을 보강했다. 기존 GOV-019 계약 목록·다음 에폭·최소 미달 채굴 거절·
정확한 이벤트는 유지했다. RPC null은 false로 해석하지 않는다.
관련 tagged CLI 전체 race·전체 검사와 실제 등록/전체 언스테이킹/재활성화 native race
(111.224초, skip0)가 통과했다. 같은 주소와 접속점에서 에폭40/높이41 true(3검증자),
에폭50/높이51 false(2검증자), 에폭60/높이62 true(3검증자)를 독립 대조했다.
고정 소스 `e9a5a2fbc436daa2e68369a0c25a2e5ad5fbb665b28e925dee42066f9f8a1a39` 전후가 같고
[검증자 전환 증거](validator-transition-evidence.json)는228파일의 내용·권한·소유자·mtime을
보존했다. 활성 프로세스·종료 DB는0이며 외부 소스는 수정하지 않았다. 원문87개 전체·Linux·
메인넷·legacy·공개 DSL 시나리오·최종 Web 인수는 별도 잔여다.

**2026-10-15 RPC-005 일반 블록 epochInfo 보강:** committedSeal 존재만 보는 baseline이
누락된 epochInfo·일반 블록의 객체·경계의 null·잘못된 PrevRound/Randao/서명을 통과시키는 RED를
확인했다. 명시적 null과 경계 검증자 정보의 정상 대조군·잘못된 응답 거절을 GREEN으로 보강했다.
관련 tagged CLI 전체 race·전체 검사와 실제 WEMIX→WBFT 후11헤더 native race(48.370초,
skip0)가 통과했다. 실제 genesis epochLength10과24~34의 일반10블록 null·경계30블록 객체를
확인했다. 기존 현재 epoch 키/BLS 검증을 유지하고 별도 외부 체인 RLP·types·blst 검증기로
RPC의 reveal·PrevRound·서명자·서명이 원본 헤더와 같은지 대조했다. 변형 RPC 네 필드도 거절했다.
고정 소스 `49e064e32bc3f0dce14ae0cba4ba555b80a408b2e3b4b45b321fbbaef1746da2` 전후가 같고
[일반/경계 증거](extra-epoch-evidence.json)는228파일의 내용·권한·소유자·mtime을 보존했다.
활성 프로세스·종료 DB는0이며 외부 소스는 수정하지 않았다. 원문87개 전체·Linux·메인넷·legacy·
공개 DSL 암호 단언·최종 Web 인수와 WBFT-001 합의 단계 로그 순서는 별도 잔여다.

**2026-10-15 TX-002·016 승인 기준 보강:** 사용자가 조회 시점/대상 블록의 BaseFee 구분과
현행 대납자 제출 거부·동일 바이트 자금 충전 대조군 기준을 승인했다. 원문에 조정과 과거 기대 차이를
명시하고 기존 진단 자료는 그대로 보존했다. 대상 블록 없이 snapshot만으로 판단하거나 잘못된 오류
코드/사유를 통과시키는 RED를 확인했다. 연속 다섯 실제 블록의 BaseFee·거래 미포함·영수증 부재·
계정 상태 불변과 정확한 대납자 사유를 GREEN으로 보강했다.

관련 tagged CLI 전체 race·전체 검사·태그 vet와 실제 전환 후 거래/거버넌스 두 native race
(166.489초, skip0)가 통과했다. cap1 거래는 풀에서 받았지만 대상 BaseFee가 각각
203718·178254·155973·136477·119418이라 채굴되지 않았다. 대납자 잔액0은 정확한 `-32000`
사유로 거절됐고, 충전 뒤 같은 서명 바이트가 성공하며 수수료2,100,000,754,068,000을 확인했다.
독립 RLP·Keccak·계정/수수료 재계산도 통과했고 고정 소스
`7020d246fec1b9fa80cb2526244de50efcd2fef70b7540f7f3a08480bff43b88` 전후가 같다.
[승인 기준 증거](tx-policy-approved-evidence.json)는456파일의 내용·권한·소유자·mtime을 보존하고
활성 프로세스·종료 DB는0이다. 이전 진단의 “결정 대기”는 당시 기록으로 유지하며 현재 승인 여부는
이 기록이 정한다. 원문87개 전체·Linux·메인넷·legacy·최종 Web 인수와 TX-019 결정은 남아 있다.

**2026-10-15 WBFT-007·008·011~013 정족수 경계 보강:** 단순 진행·복구 baseline이
부족한 서명자·종료 노드 서명·잘못된 정족수/생존 수·짧거나 전진한 중단 구간·잘못된 상태 집계를
통과시키는 RED를 확인했다. 다섯 정상 구성과 열한 잘못된 조건의 GREEN을 보강했다.
실제 7/1·7/3·3/1·6/1·6/2 노드/장애 구성을 직렬 실행해 건강한 세 블록의 서명자와
`istanbul_status` 집계를 대조했다. 진행 가능한 두 구성은 장애 중 열 블록을 검사했고,
6/1 구성은 매번 정확히 다섯 생존 서명자를 확인했다. 부족한 세 구성은 종료 후1초 안정화 뒤
모든 생존 노드의 높이가8초 동안 유지됐다. 원래 신원으로 재시작·동기화하고 세 복구 헤더도 확인했다.

관련 실행 엔진 전체 race·전체 검사와 다섯 실제 native race(254.555초, skip0)가 통과했다.
고정 소스 `9a7259f6311d00d3685d5d26a7df5f5a2b319e76a39804ef958852103e9a0a49` 전후가 같고,
보존된 RPC 자료의 독립 대조도 통과했다. [정족수 증거](quorum-boundaries-evidence.json)는
1,548파일의 내용·권한·소유자·mtime을 보존했으며 활성 프로세스·종료 DB는0이다.
현재 Croissant0 로컬 범위이며 원문 단계적 이관·Linux·메인넷·legacy·공개 DSL·최종 Web 인수는
별도 잔여다. 이전 간헐 실패의 원인을 해결했다고 표시하지 않으며 §0a 11번 전체는 계속 진행이다.

**2026-10-15 WBFT-003·004 실제 라운드 전환·재시작 보강:** 단순 회복·3개 연결 baseline은
실제 대상·라운드·PrevRound·지연·새 proposer·PID·길이·높이·메시지 조건 누락을 통과시켰다.
RED→GREEN으로 실제 헤더와 검증자 순서에서 다음 round-zero proposer를 선택하고 원래 관리 노드를
종료·재시작하도록 보강했다. 최초 native는 round1·PrevRound1·재시작까지 진행했으나 RPC 준비 전에
신원을 조회해 실패했다. 이 실패를 보존하고 준비 경합 RED→GREEN과 잘못된 신원·기한 거절도
보강했다. 헤더 시간 차이2초는 실제 생성 지연이 아니므로 실제 commit 로그의 시간을 별도로
검증한다. 원래 헤더 연결·라운드·PrevRound 조건을 줄이지 않았다.

관련 실행 엔진 전체 race·전체 검사와 실제 BP4 native race(44.289초, skip 0)가 통과했다.
실제 14번 블록의 round1·다른 proposer, 다음 블록 PrevRound1, 생존3노드의 해당 메시지,
원래 신원/PID 재시작·동기화와 이후5개 연결을 독립 대조했다. genesis blockPeriod2·requestTimeout3를
확인했고 commit 로그 지연은3,055ms, 헤더 시간 차이는2초로 구분했다. 고정 소스
`439f9e782b076bfbbfc39e860946ff6c0211c7d270a9d4bdc62531837dbc4250` 전후가 같다.
검토 상태 변경으로 특정 ID를 하드코딩한 self-merge 변형이 실제 자기 참조가 아니게 된 RED도
확인했다. 선택한 merged 행 자신의 ID로 바꿔 순환 거절과 실제87개 검토 검증 GREEN을 유지했다.
[라운드 전환 증거](view-change-evidence.json)는 성공200파일과 최초 실패198파일의 내용·권한을
보존했고 활성 프로세스·종료 DB는0이다. 현재 Croissant0 로컬 범위이며 원문의 단계적 이관 환경·
Linux·메인넷·legacy·공개 DSL 시나리오·최종 Web 인수는 별도 잔여다. §0a 11번 전체는 계속 진행이다.

**2026-10-15 WBFT-006 RoundRobin 50블록 보강:** 서로 다른 proposer 존재만 보는 baseline은
짧은 구간·낯선 proposer·연속 중복·불균등·높이/parent·상태 집계/범위 불일치 여덟 조건을
통과시켰다. 이 RED를 확인하고 실제 5검증자의 정확한 101~150 구간·각10회·연속 중복 없음·
모든 parent 연결과 `istanbul_status`의 author 집계/범위를 대조하도록 GREEN을 보강했다.
최초 native는 chainPreset 누락으로 기동 전에 실패했고 준비 오류 RED→GREEN으로 BP5 파싱도
확인했다. 실패를 성공으로 바꾸어 표기하지 않는다.

관련 실행 엔진 전체 race·전체 검사와 실제 WBFT BP5 native race(189.628초, skip 0)가 통과했다.
genesis의 proposerPolicy0·targetValidators5, 50헤더와 RPC 집계에서 5개 주소 모두 각10회인 것을
독립 대조했다. 고정 소스 `17cabc2579784001569bba02f1ddefc601724bf0fde39c911ef4ebf3c2d27a86`
전후가 같고 준비 실패를 포함한 247파일의 내용·권한을 보존했으며 활성 프로세스·종료 DB는 0이다.
[RoundRobin 증거](round-robin-evidence.json)는 Croissant0으로 시작한 현재 로컬 환경 범위다.
원문의 단계적 이관 환경·Linux·메인넷·legacy·최종 Web 인수는 별도 잔여이며 §0a 11번 전체는
계속 진행이다. 기존 4검증자/16블록 테스트는 더 좁은 과거 참조로 유지한다.

**2026-10-15 WBFT-001·RPC-005 현재 commit 집계 서명 보강:** 존재·길이만 보는 baseline은
잘못된 서명·bitmap·quorum·헤더·round·키·무한대 점·중복 키 아홉 조건을 통과시켰다.
이 합성 RED를 확인하고 현재 epoch 순서의 키·bitmap·quorum과 원본 헤더의 commit 메시지를
연결하는 kilic BLS pairing으로 GREEN을 보강했다. 합성 WBFT 9필드·StableNet 10필드 extra,
잘못된 헤더·무한대 공개키도 회귀했다. 과거 공개 사례의 native RED를 실행한 것으로 표시하지 않는다.

실제 WEMIX→WBFT 포크 이후 연속 11헤더를 epoch20·30의 키와 현재 검증자 순서로 검증했다.
수정하지 않은 외부 WBFT의 RLP·types 해시·blst를 사용하는 별도 임시 검증기로 헤더 번호·해시·
quorum·서명을 독립 대조하고 변경 메시지 거절도 확인했다. 관련 전체 tagged CLI race·전체 검사와
새 집계 서명/기존 헤더 무결성 두 native race(165.578초, skip 0)가 통과했다.
고정 소스 `793b2e0a2363c7b8a62400ad51955437fd7d560581173a8131e3344865ee70ca` 전후가 같고
456파일의 내용·권한을 보존했으며 활성 프로세스·종료 DB는 0이다.
[집계 서명 증거](wbft-aggregate-seal-evidence.json)는 현재 commit 암호 검증 범위다.
합의 단계 log 순서·비경계 RPC의 epochInfo null·공개 DSL 암호 단언·Linux·메인넷·legacy·
최종 Web 인수는 별도 잔여다. 현재 컴파일된 `TestWemix`는 24개이며 이번 두 실행을 24개 전체
통과로 확대하지 않는다. §0a 11번 전체는 계속 진행이며 외부 소스는 수정하지 않았다.

**2026-10-15 RPC-018 EN 격리·풀 상세 보강:** 기존 합의 중단 사례는 다른 hash의 같은 nonce·가스를
통과시켰다. 두 hash 변형 RED→GREEN을 보강하고 실제 EN(node5)만 BP4에서 peer 격리하는 사례를 별도로
추가했다. 최초 세 체인과 후속 진단은 복구·격리·전파 관문에서 실패했고 실패 자료를 유지했다.
실제 `admin_nodeInfo`의 NAT 외부 주소, EN 자금 동기화 누락, 늦은 연결, 로컬 풀에서만 허용되는
고정 수수료를 대조했다. peer 변경 false 거절·기록된 host/port 복구(attach 광고 보존)·교차 그룹 peer
연속 부재 확인과 잔존 연결 재제거·확인 불가 응답 거절의 각 RED→GREEN을 보강했다.
새 사례는 EN 잔액 동기화 뒤 실제 추천 가격·팁으로 충분한 cap을 구성한다. EN peer0·블록 정지와
BP 전진, nonce0 pending·nonce2 queued의 정확한 hash·필드·미채굴 영수증, 재연결 뒤 nonce1을
채우고 세 거래의 성공 영수증을 검사한다. 기존 합의 중단 사례는 별도 회귀로 유지한다.

관련 전체 race·전체 검사와 최신 고정 소스의 세 체인 6사례·147단언 native race(261.555초,
skip 0)가 통과했다. 독립 hash·nonce·블록·영수증 대조도 통과했고 지문
`bc919d1a958d5cb3f313e6c085e43d4fdbbc6a4fb7c04afd0248140471e38f39` 전후가 같다.
이전 실패·진단을 포함한 17실행 3,814파일의 내용·권한을 보존했으며 활성 프로세스·종료 DB는 0이다.
[풀 격리 증거](pool-isolation-evidence.json)의 과거 실패는 실패로 유지했다. 이 결과는 BP4·EN1 로컬
범위이며 원문의 `en_2` 대신 node5를 썼다. §0a 11번 전체·이관 환경·Linux·메인넷·legacy·최종
Web 인수는 계속 미완료다. 외부 체인 소스는 수정하지 않았다.

**2026-10-15 RPC-019 peer 구조 보강:** 기존 공개 사례는 ID 포함·개수만 확인해 마지막 peer의
빈 주소·누락 network·누락/빈 protocols를 통과시켰다. 네 조건의 RED를 확인하고 BP4·EN1 고정 fixture의
정확한 네 peer 전수 필드를 검사하도록 GREEN으로 보강했다. 기존 E2E도 RPC 오류를 0개로 숨기는
RED를 확인하고 typed 오류 코드·사유를 반환하도록 고쳤다. 정상 빈 목록과 오류의 의미를 구분한다.
관련 전체 race·전체 검사·E2E 오류 회귀·세 체인 native race(167.636초, skip 0)가 통과했다.
노드별 peerCount는 한 recorder 단언으로 합산돼 실제 기록은 총 54단언이다. 단일 응답의 네 peer
필드와 각 상대 노드의 직접 조회 ID도 독립 대조했다. 고정 소스
`cb17d43508cd1e2f43d103e0520a25ce84caef93aaea14f4db5673f0685457db` 전후가 같고,
608파일의 내용·권한을 보존했으며 활성 프로세스·종료 DB는 0이다.
[peer 증거](peer-fields-evidence.json)는 로컬 fixture 범위다. §0a 11번 전체·원문 이관 환경·
Linux·메인넷·legacy·최종 Web 인수는 계속 미완료다.

**2026-10-15 RPC-017 수수료 이력 보강:** 기존 NotNil 검사는 짧은 배열·0 기본 수수료·범위 밖
사용률·빠진 percentile을 통과시켰다. 정상 대조군과 잘못된 응답 일곱 가지의 RED를 확인하고
원문 `10/latest/[25,75]` 입력, 11개 기본 수수료 양수, 10개 사용률 0..1,
10개 보상 행·각 두 percentile을 공개 DSL의 모든 인덱스에서 검사하도록 GREEN으로 보강했다.
관련 전체 race·전체 검사와 세 체인의 실제 135단언·native race(157.281초, skip 0)가 통과했다.
각 단일 RPC 응답도 독립 전수 대조했고 고정 소스
`e387b3038ceb8cda59ab9e62cf198b60fa8a974a474d7beaa2fd09c2755d460c` 전후가 같았다.
증명 501파일의 내용·권한을 보존하고 활성 프로세스·종료 DB는 0이다.
[수수료 이력 증거](fee-history-evidence.json)는 로컬 fixture 범위이며 §0a 11번 전체·Linux·메인넷·
legacy·최종 Web 인수는 미완료 상태를 유지한다.

**2026-10-15 TX-003·006 수수료 필드 보강:** 기존 공개 DSL은 동적 가격을 무조건 base+tip으로
계산하고 실제 signed cap/tip·legacy gasPrice·receipt type을 직접 묶지 않았다. cap 제한 정상값이
실패하고 잘못된 가격·필드가 통과하는 RED를 확인한 뒤, 정확한 정수 `min`과 조회·단언의 schema·format
소비를 보강했다. 원문 입력인 관측 base+tip으로 cap을 구성하고 실제 채굴 블록의
base+min(tip,cap−base)를 검사한다. Legacy는 선언한 노드 추천 가격·실제 거래·영수증을 대조한다.

관련 race·전체 검사와 세 체인 공개 사례 6개·48단언의 native race(169.408초, skip 0)가 통과했다.
고정 지문 `3e147f8c5447b1cf949b332ec22b9b029b3b205903fb1c0e0cfdbb2efca0d38a` 전후가 같고,
독립 계산·활성 부재 확인 후 증명 528파일의 내용·권한을 보존하고 종료 DB는 0개다.
[수수료 증거](fee-price-evidence.json)는 통제된 로컬 입력의 범위이며 임의 가격 정책의 가족 간 동등성,
이관·Linux·메인넷·87개 전체·최종 Web 인수의 완료로 확대하지 않는다. §0a 11번은 계속 진행이다.

**2026-10-15 TX-008 위임 실행 보강:** 기존 공개 케이스는 고정된 빈 주소를 위임 대상으로
삼고 코드 표시만 확인했다. 실제 실행이 없거나 잘못된/양쪽 저장소가 바뀌어도 통과하는 네 조건의 RED를
정상 대조군과 함께 확인했다. 실제 London 호환 저장소 계약을 배포해 새 authority에 위임하고,
스폰서가 authority에 쓰기·조회한 값을 직접 저장소와 대조하도록 GREEN으로 바꿨다.
원본 계약의 저장소가 그대로 0인지도 검사한다. 정확한 23바이트 marker는 실제 배포 주소와
정수 연산으로 구성한다. EN이 없던 프리셋도 WBFT BP4+EN1로 바로잡았다.

공개 케이스와 독립 RPC 대조는 실제 WBFT(Croissant/Applepie0)에서 13단언·native race·skip 0·
28.503초 PASS였다. 실제 type4의 authorization target, marker, authority view/slot0 42와
원본 계약 view/slot0 0을 확인했다. 고정 소스
`01b125456dd4e4899b6876a4f6da69d2b133a5d60979cd8cb4fd6aa76bc0a027`의 전후 지문이 같았고
활성 프로세스·DB 0, 240파일의 내용·권한·소유자·mtime을 보존했다.
전체 make check와 실행·DSL·계정 세 패키지 전체 race도 통과했다.
주소 golden의 880→888은 이 단일 케이스의 주소 위치 이동·추가만 포함한다.

87개 대조에서 TX-008을 명시한 범위 증명으로 갱신했다. 다른 체인·이관·Linux/mainnet·전체 87개·
최종 Web 인수 완료는 아니다. 나머지 부분 70·범위 증명 12·충돌 3·통합 2이며 전체 완료는 false다.
근거: [실제 위임 코드 실행·저장소 분리 증거](setcode-execution-evidence.json).

**2026-10-15 TX-002·016 실제 정책 진단:** 새 진단 분류에서 준비·추정·HTTP 실패, 잘못된
서명 바이트/해시, 답 없는 요청과 error+hash 혼합을 실제 노드 제출 거부로 세지 않도록 RED→GREEN을 검증했다.
실제 Croissant 전환 뒤 최신 BaseFee 70,557에서 cap 70,556의 유효 type2를 제출했더니 받아졌고,
다음 26번 블록 BaseFee가 61,738로 내려가 status 1로 채굴됐다. 실제 적용 가격은 70,556으로
`baseFee + min(tipCap, cap-baseFee)`의 cap 제한 분기와 정확한 잔액·nonce·송금 영향을 독립 대조했다.
최신 블록의 fee와 대상 채굴 블록의 fee는 같은 값이라고 가정할 수 없다.

미충전 FeePayer의 유효한 이중 서명 type16은 eth_sendRawTransaction의 code -32000,
정확한 `fee delegation: insufficient feePayer's funds for gas * price`로 제출 거부됐다.
다섯 블록 뒤에도 원본 서명 bytes의 해시 조회와 영수증은 null이고 계정 상태는 그대로였다.
FeePayer를 충전한 뒤 같은 서명 bytes가 status 1/type16으로 채굴됐으며 송신자는 value 1,
대납자는 정확한 gasUsed×effectiveGasPrice 2,100,000,153,174,000만 부담했다.
서명·요청/응답·raw RLP 인자·SDK 래퍼를 통하지 않은 표준 Keccak 재계산도 대조했다.

진단과 관련 거버넌스 읽기 두 native는 고정 소스
`31e98b9bd57449f035815a631f65dac9493a0856499b6a52703cbd6fd3c0408c`에서 161.174초·race PASS·skip 0,
전후 지문 일치를 확인했다. 두 환경의 활성 프로세스/DB는 0이며 456파일 내용·권한·소유자·mtime을 보존했다.
전체 make check·tagged vet·CLI tagged 전체 race(105.072초)와 최종 분류 단위 race도 통과했다.
현재 컴파일된 TestWemix 분모는 새 진단을 포함한 23개다. 앞 절의 22개는 그 검토 당시 값이다.

이것은 요구 충돌의 실제 진단이며 TX-002·016 원문 통과나 결함 해결이 아니다.
기준 변경 질문은 남겼지만 답은 아직 없으므로 원문을 수정하지 않았고 두 조건은 conflict로 유지한다.
TX-019 결정·원문 87개 잔여·legacy·Linux/mainnet·외부 W3/W4·최종 Web 인수도 계속 남는다.
근거: [원본 서명 입력·노드 정책·정상 대조군 증거](tx-policy-boundary-evidence.json).

**2026-10-15 원문 87개 충분성 대조 고정:** 삭제된 tracker의 84개나 테스트 이름의 존재를
분모로 쓰지 않고 원문 87개 ID 전부와 공통 70개를 포함한 현행 TC 입력·단언을 대조했다.
[조건별 정본 대조](wemix4-tc-sufficiency-review.json)는 부분 70·명시 범위 증명 11·충돌 3·
미검증 1·통합 2로 나누고 각 원문 section, 소스, JSON 단언 위치와 증거의 해시·잔여 조건을 보존한다.
범위 증명도 전체 원문·메인넷 완료가 아니며, WBFT-004/GOV-024 통합은 부모 조건의 미완료를 상속한다.

`make tc-review`는 87개 누락/중복/알 수 없는 ID, 숨긴 잔여, 오래된 원문/소스/증거,
변경된 실제 JSON 단언, 근거 없는 전체 완료, 잘못된/순환 통합을 거절한다.
Go 함수와 메서드는 build tag를 포함한 실제 AST 선언으로 확인하며 주석/문자열을 함수로 세지 않는다.
희망적인 전체/부분 완료 14조건 RED→GREEN과 AST 가짜 선언·잘못된 문서 형태 RED→GREEN을 확인했고
전체 make check와 CI의 `make tc-review`에 연결했다. 소스 해시가 바뀌면 실제 명제를 재검토해야 하며,
해시만 갱신해 조건 검토를 대신하지 않는다.

TX-002 BaseFee−1 제출/미포함 의미와 TX-016 대납 잔액 부족의 제출 거부/채굴 실패 기대는
실제 입력 관측과 결정이 남는다. TX-019의 정확한 P256 실패 반환값은 사용자 결정 대기다.
legacy 원본 두 경로도 없으므로 행동 동치를 확인했다고 하지 않는다.
현재 컴파일된 TestWemix 분모는 22개이며 옛 20개 고정 스크립트는 재사용하지 않는다.
이 대조 고정은 조건 누락 방지이며, 열린 87개 조건의 구현·전체 실제 실행이나 최종 Web 인수 완료가 아니다.
외부 Ouroboros QA 전달은 자동 승인 검토에서 거절돼 별도 승인 대기다. 로컬 artifact/process 검토만 반영했다.
전체 make check·기존 AST 인벤토리 회귀 4개와 고정 소스 CLI capture/독립 재현의 동일 결과를 확인했다.
검증 지문은 `aad84c9e139196b926d1a8be3120ab12574224c5d18f01a1ec7ebc1c09a3abff`이며 증거 게시 이전 소스다.
근거: [대조 검사 RED→GREEN·독립 재현](tc-sufficiency-guards-evidence.json).

**2026-10-15 TX-010·013 및 안전 회수 보강:** 공개 케이스가 블록 순서 역전·같은 블록의
인덱스 역전/중복을 통과시키고 queued 교체 전후를 조회하지 않는 RED를 확인했다.
정상 같은/다른 블록 대조군과 잘못된 대기열 해시를 실제 DSL 인터프리터로 검증해 GREEN으로 바꿨다.
TX-010은 역순 제출 2·1의 queued 해시, 성공/실제 nonce, 영수증의 blockNumber 비감소와
동일 블록 인덱스 엄격 증가를 확인한다. TX-013은 nonce 2의 기존/새 queued 해시와 정확한 건수,
nonce 0·1 격차 채우기, 기존 hash 조회/영수증 null과 교체 거래 성공을 확인한다.

첫 세 체인 시도는 케이스·거래 대조가 통과했지만 WEMIX 종료 DB 회수의 활동 감지로 최종 FAIL이었다.
당시 활동 주체는 기록되지 않았으므로 특정 프로세스의 경합이라고 단정하지 않는다.
새 조회에서 프로세스/열린 파일 없음과 보존 파일 117개의 내용·권한·소유자·mtime을 확인한 뒤
남은 DB만 회수했다. 안전 검사 자체는 유지하고 활동 감지에만 기존 종료 기한 내 새 조회로 재시도한다.
조회 불가·소유권·링크 오류는 즉시 실패한다. 회수 중 활동 RED→GREEN·관련 Python 63개와
실제 열린 파일의 활동 감지/대기 및 시간 초과 시 삭제 거절 두 실행을 확인했다.

새 고정 소스 `9b0104fd3418eb787b1152b8034e56b17c4ac20847b0690d1848bbbb38aa14e7`에서
세 체인 두 케이스씩 6개·race·skip 0·209.971초 PASS, 실행 전후 지문 일치를 확인했다.
첫 실패 시도도 보존하며 6개 환경의 DB/활성 프로세스는 0, 1,056파일 내용·권한·소유자·mtime을 대조했다.
전체 make check와 Go 관련 두 패키지 race는 통과했다. 주소 golden의 878→880은 두 케이스의
주소 위치 이동과 격차 거래·계정 조회 추가만 포함한다.

전체 Python 108개 중 과거 Web 증명 검증 11개(9실패·2오류)는 변경/누락된 artifact·runtime 불일치로
통과하지 못했다. WEB-04의 옛 dashboard 파일도 없다. 이는 §0a 1번 과거 자료 대조와
후속 고정 소스 Web 인수의 잔여이며, 관련 회귀 통과나 이번 두 TC 증명으로 닫지 않는다.
원문 87개 전수·legacy·Linux/mainnet·W3/W4·최종 Web 인수는 계속 남는다.
근거: [nonce 순서·대기열 교체 및 안전 회수 증거](nonce-order-replacement-evidence.json).

**2026-10-15 WBFT-010·002 보강:** 기존 Randao 케이스는 reveal과 mix의 존재만 확인했고,
공통 평균 간격 케이스도 원문의 개별 1초 조건을 대신할 수 없었다. 변조된 서명·proposer·mix·
높이·parent·간격·round 8조건을 단위 RED→GREEN으로 거절하고 정상 암호학적 연쇄,
잘못된 chain ID·짧은 창도 검증했다. 원문의 XOR 표기는 현행 프로토콜의 32바이트
Keccak256(reveal)로 명확히 했으며 65바이트 reveal 자체와 XOR했다고 주장하지 않는다.

명시적 blockPeriodSeconds/requestTimeoutSeconds 1의 실제 WEMIX→WBFT 포크에서 24번 anchor와
다음 연속 헤더 10개를 관측했다. Croissant-v1 ECDSA 복구 주소 10개가 실제 proposer와 같았고,
mix 전이 10개와 높이·parent 연결, round 0 및 개별 timestamp 차이 10개 모두 1초였다.
새 검증과 관련 거버넌스 읽기 두 native 테스트는 고정 소스
`d7b1f6c7be8c6b96526492d7ac3c36a4960e2f9ffc4906950761d76c42d30061`에서 157.613초·race PASS·Go skip 0이었다.
두 환경의 활성 프로세스·종료 DB는 0이며 456파일의 내용·권한·소유자·mtime을 대조했다.
전체 make check·tagged vet·관련 tagged race도 통과했다.
건강한 round-zero 창의 충분성이다. view-change 시간·원문 87개 전수·legacy·Linux/mainnet·
최종 Web 인수는 별도 잔여다. 컴파일된 전체 native 분모에는 이 새 테스트도 포함해야 한다.
근거: [실제 서명·mix·개별 간격 증거](wbft-header-integrity-evidence.json).

**2026-10-15 승인 거버넌스 기준 보강:** 사용자 승인에 따라 GOV-019 미달 금액을 현행 v1의
실제 채굴 거절로 검증한다. SDK 추정 거절에 머물지 않도록 명시 가스로 제출하고 status 0·실제 입력/value·
정확한 Error(string) revert trace·value 환불과 수수료·비활성/금액 0/목록 불변을 대조했다.
정상 재활성화의 정확한 Staked 이벤트와 후속 에폭 복귀는 유지했다.

GOV-020은 즉시 요율 변경과 정확한 Requested emitter·staker·old/new rate, Executed 부재·위임량 0·
지연 요청 없음·실제 거래 입력을 검증한다. 기존 원문의 이벤트 기대 차이는 승인 기록에 남겼다.
GOV-021은 명시적 30초 fixture에서 정확한 요청·적용 이벤트, 지연 전 eth_call 및 실제 채굴 거절의
정확한 사유·수수료·상태 불변, 체인 timestamp의 지연 경계와 적용 후 요청 제거를 확인했다.
사용자는 축약 fixture를 승인했으며 604,800초 메인넷 경과 검증으로 확대하지 않는다.

빠진 채굴 연결 11조건과 topic만 맞는 잘못된 이벤트 7조건을 단위 RED→GREEN으로 거절했고,
관련 tagged race와 vet를 통과했다. 최종 거버넌스 세 건과 긴급 모드 실제 회귀 네 건은 고정 소스
`b05ede48bf65c0fac68a912c4e855b55d4f5a10fc13a873474f6ce690b54a287`에서 417.070초·race PASS·Go skip 0이었다.
이전 준비 관문 차단과 중단 실행도 보존한다. 중단 실행은 한 테스트의 통과 뒤 끊겼으므로 최종 인수가 아니다.
재실행은 소스 지문을 먼저 저장했다. 전체 13환경은 활성 프로세스·종료 DB 0이며 2,758파일의
해시·권한·소유자·mtime을 다시 대조했다. 최종 전체 make check·CLI 전체 tagged race·tagged vet도 통과했다.
87개 충분성·legacy·메인넷·최종 Web 인수는 계속 남는다.
근거: [거버넌스 실제 거절·이벤트·중단 보존](governance-rejection-events-evidence.json).

**2026-10-15 TX-018 유한한 추정 경계 진단:** 잘못된 입력·거래 hash·블록·영수증·상태를
받아들이던 검증을 단위 RED→GREEN으로 보강했다. 원문 기대를 실제 WBFT에 적용한 RED는
추정치 31,513·한도 31,512·실제 소비 31,160·status 1·저장소 42→1을 기록했다.
임시 파일의 프리셋 조회 실패는 준비 오류로 별도 보존하며 제품 RED로 세지 않는다.

같은 초기 상태의 독립 계약 세 개에서 추정 한도, 추정치−1, 실제 성공 소비량−1을 비교했다.
StableNet·WBFT는 추정치−1이 성공했고, WEMIX는 추정치와 실제 소비가 31,160으로 같아
추정치−1이 실패했다. 세 체인의 실제 소비량−1 입력은 모두 status 0·gasUsed 31,159·
저장소 42 유지·완료된 SSTORE 뒤 실패를 확인했다. 실제 calldata·가스 한도·영수증·정확한
수수료 차감과 후속 블록 전진도 대조했다. 진단 9개 거래와 기존 실패 거래 6개 회귀는 같은
고정 소스 `aa700fb41042eb559cfb91a41d88ea57993a6514c8a066bcb03e7927ef109b77`에서
직렬 race로 통과했고 Go skip은 0이었다. 8개 시도의 활성 프로세스·종료 DB는 0이며
1,230파일의 내용·권한·소유자·mtime을 다시 대조했다.

이것은 유한한 입력에 대한 진단이며 TX-018 원문 인수 완료가 아니다. 진단 뒤 사용자는 추정 상한과 실제 부족한 가스를 구분하는 완료 기준을 승인했다.
원문 변경 전의 위 진단 기록은 보존하고, 승인 기록과 공통 DSL 케이스를 추가했다.
아래 승인 기준의 새 케이스 실제 회귀를 별도로 완료했다. GOV-019의 미달 거래 채굴 검증·87개 충분성·legacy·Linux/mainnet·최종 Web 인수도 남는다.
근거: [추정 경계·실제 영수증·보존 자료](estimate-boundary-evidence.json).

**TX-018 승인 기준 완료:** 사용자 승인 뒤 공통 `CT-CONTRACT-008-finite-gas-boundary`를 추가했다.
두 독립 초기 상태에서 추정 한도의 성공·실제 소비 상한과 성공 소비량−1의 실제 채굴 실패·전량 소비·롤백을
DSL로 실행한다. 새 호출자의 정확한 수수료와 실제 쓰기 뒤 실패를 확인한다. 기존 무한 반복 케이스는 보존했다.
주소 golden은 새 케이스의 16위치만 추가했고 기존 862위치는 그대로다.

세 체인의 공통 DSL 케이스 각 16단언·실제 쓰기 거래 6건과 estimate−1 진단 거래 9건,
기존 실패 거래 6건 회귀가 고정 소스 `bc491358c03f64bb50a6fa583875764f24de838976c4737a684356240a8da7e7`에서
직렬 race로 통과했다. 준비 실패·원문 기대 RED와 이전 진단도 보존했고 전체 14시도의 활성 프로세스·종료 DB는 0,
2,283파일의 해시·권한·소유자·mtime 대조가 통과했다. 관련 두 패키지 전체 race와 최종 전체 make check도 통과했다.
이 완료는 승인된 TX-018의 공통 local native 범위이며 Linux/mainnet·87개 충분성·최종 Web 인수의 완료가 아니다.
GOV-019도 현행 거절 계약에 맞추도록 승인했지만 미달 거래 채굴·사유·상태 보강은 다음 단계다.
근거: [승인 케이스·진단·실제 회귀 증거](estimate-boundary-evidence.json).

**2026-10-13 저장 공간 보강 완료:** `runtime_storage.py`가 새 테스트 폴더를 소유권 표시와 20 GiB 사전 검사로 만들고,
성공한 실행 뒤 인용되지 않은 종료 노드 DB와 독립 재현의 의존성 복사본만 자동 제거한다.
과거·사용자 폴더, 활성 데이터, 설정·로그·바이너리·소스·스냅샷·세션·증명은 보존한다.
보존 계약 31개, 실제 128 MiB 반복 3회, WEB-08 capture·독립 재현, 세 체인과 SSH 개별 실행을 확인했다.
원본 소스와 receipt 불변성도 통과했다. 근거는 [정리 증거](runtime-storage-evidence.json)이며 최종 14개 인수와는 별개다.

**실패 경로의 저장 공간 보강 완료:** 성공 뒤에만 도달하던 회수를 종료 `finally`로 옮겼다.
부분 JSON/JSONL과 직접 인용한 DB는 보존하고, 종료를 기다린 뒤 소유권이 확인된 임시 DB와 재현 의존성만 회수한다.
capture 실패와 독립 재현 실패의 누적을 RED→GREEN으로 확인했다. 보존·경로 계약 45개,
실제 128 MiB 실패 반복 3회(각 종료 후 20 KiB), 실제 WBFT 4프로세스 시작 후 의도한 브라우저 실패,
세 체인 성공 작업과 SSH 회귀가 통과했다. 갑작스러운 OS 종료는 `finally`를 실행하지 못하므로
활성·참조 확인을 거치는 기존 보수적 유지보수 경로에 남긴다. [실패 정리 증거](runtime-storage-failure-evidence.json).
전체 Python 인수 검사의 과거 receipt 불일치와 아래 최종 인수는 아직 남아 있다.

**2026-10-14 브라우저 의존성 누적 보강 완료:** 재현 폴더의 `source/tests/webui/node_modules`가
종료 정리에서 빠진 것을 RED로 재현하고 회수 대상에 추가했다. 보존·경로 계약 47개와 실제 Playwright
복사·의도한 실패·회수 반복 3회가 통과했고, 각 종료 후 잔여는 16 KiB였다.
현재 머신에서는 증명으로 확인한 세 과거 폴더의 의존성 복사본 5곳과 비참조 lightchaindata 5곳을 제거했다.
나머지 7,809개 파일의 내용·권한·소유자·mtime은 동일했다. 직접 참조한 chaindata 5곳과 출처가
확인되지 않은 과거 의존성 복사본 두 곳은 보존했다. 전체 디스크 여유는 45 GiB다.
APFS 공유 블록 때문에 삭제 경로의 할당량을 실제 확보 용량으로 해석하지 않는다.
근거: [브라우저 의존성 정리 증거](browser-dependency-storage-evidence.json). 최종 인수는 계속 미완료다.

**2026-10-14 handoff 종료·전환 회귀 보강:** 이전 E2E는 종료 실패를 로그로만 남긴 뒤 전체 폴더를 삭제했다.
이제 기록을 통한 생산 종료 경로를 사용하고, 종료를 입증할 수 없으면 삭제와 재시도를 거절한다.
소유권·20 GiB 사전 검사·실제 프로세스와 열린 파일 조회를 공유 정리 모듈에 맡겨 종료 후 비참조 DB만 회수한다.
입력·설정·로그·세션은 보존한다. 실제 열린 파일로 거절을 확인한 뒤 64 MiB를 회수했고 내용·권한·mtime은 유지됐다.
그다음 실제 전환에서 capability 관문이 WBFT 노드에도 WEMIX API를 조회하고 이전 PID/역할을 사용하던 회귀를 발견했다.
노드별 기록된 체인을 전달하고, 전환 결과의 표를 갱신하며, 전환 후 fork-1 대기를 생산으로 오인하지 않게 고쳤다.
RED→GREEN·전체 make check·관련 네 패키지 race·tagged 명령 AST/종료 계약이 통과했다.
같은 고정 소스에서 실제 거버넌스 조회와 데이터 인식 이관이 각 한 번에 PASS·Go skip 0건으로 통과했다.
실패한 네 시도와 성공한 두 전환의 종료 DB는 모두 0건이며 소스 SHA 17개가 일치한다.
이관은 producer/reader 높이 19와 기존 블록 상태 읽기까지다. 이관 노드의 이후 생산이나 87개 wemix4 시나리오,
governance E2E 전체, Linux L1, 최종 Web 인수를 완료한 증거가 아니다. §0a 11번은 계속 진행 상태다.
근거: [handoff 증거](handoff-runtime-evidence.json).

**같은 날 workspace 재접속 보강:** 포크 조회가 저장된 전환 완료 상태를 전달하지 않아 재접속 시
fork-1 대기를 다시 허용하던 경로를 두 RED로 확인했다. 기존 `ForkStanding`의 기록 읽기를 재사용해
모든 successor의 역할/바이너리 전환 상태를 전달하고 준비 관문을 복원한다. 기록 상태는 실제 생산의 증거가
아니므로 전환한 망은 계속 RPC 건강·생산 확인을 거친다. 전환 전·부분·완료 상태 계약,
관련 두 패키지 전체 race·전체 make check·tagged 명령/종료 계약이 통과했다.
실제 전환 뒤 CLI `run --attach --no-node-control`로 다시 읽어 PASS 1·skip 0,
WBFT 4·WEMIX 1의 합의 출처와 다섯 노드의 PID·역할·체인 불변성을 확인했다. 종료 DB는 0건이다.
소스 SHA 6개가 일치한다. 전체 governance E2E와 11번의 다른 재평가는 계속 남는다.
근거: [재접속 증거](handoff-reattach-evidence.json).

**현행 WEMIX E2E 전체 회귀:** `fa83c3c7`의 고정 소스에서 컴파일된 목록의 20개 native 테스트를
직렬 race로 실행해 20 PASS·FAIL/Go skip 0건, 1,715.233초를 확인했다. 전체 소스 해시는 실행 전후
`0127073c4399a3b7d457a56c41cccd2d11e1a6f1cdea3853fb3ac74eb2fc6ef2`로 같다.
검증자 선정과 NCP 추가는 최초 RPC capability 확인이 차단돼 각각 두 번째 구성에서 통과했다.
이 초기 차단 자료도 보존했으며 전체 시도 22곳은 프로세스·종료 DB 0건이다.
과거 “21 PASS”의 다른 한 건은 모의 RPC를 사용하는 `TestStablenetGovernance_DSL_E2E`이므로 native 분모에 넣지 않는다.
전체 make check에서 그 기존 모의 검사도 통과했지만 이것은 native 21건의 증거가 아니다.

**이관 충분성은 별도 잔여다.** 삭제된 과거 tracker의 ID는 84개로, 현재 명세 87개 중 WBFT-004와
GOV-018/019는 독립 행이 없다. WBFT-004는 병합 항목이고 GOV-018/019는 기존 함수의 실제 단언을 대조해야 한다.
당시 Go AST로 일곱 파일의 함수 67개·native 테스트 20개·실패 조건 150곳을 읽었으며,
GOV-018의 staking 재개, GOV-019의 목록·후속 에폭 후보 복귀가 당시 두 테스트에 없었다. 아래 후속 보강과 구분한다.
GOV-019의 최소 금액 미달 원문과 현행 계약 거절 조건도 다르다. NODE-002도 당시에는 앞서 적은 데이터 인식 범위만 검증했다. 아래의 후속 정확성·포크 이후 동기화 보강과 구분한다.
따라서 기존 테스트 20개 전체 통과를 원문 87개 충분성이나 11번 완료로 바꾸지 않는다.
근거: [20개 실제 회귀와 초기 차단·보존 목록](handoff-regression-evidence.json).

**GOV-018 누락 보강:** 위 AST 대조에서 발견한 staking 재개 누락을 실제 체인에서 RED로 확인했다.
긴급 모드 플래그가 해제돼도 추가 staking을 수행하지 않아 금액이 1e25에 머물렀고, 새 단언은 2e25를 요구하며 실패했다.
이제 해제 후 staking을 실행하고 receipt의 성공·transaction hash, 실제 금액 증가,
`Staked(address,uint256)`의 GovStaking emitter·정확한 staker·금액을 모두 확인한다.
원본 receipt도 보존한다. 전체 make check·tagged race와 실제 재활성화·긴급 모드·지연 수수료 변경
3개 직렬 race 회귀가 PASS·Go skip 0으로 통과했다. 재활성화의 초기 준비 차단 한 번은 그대로 기록했고,
RED와 GREEN의 시도 5곳은 모두 프로세스·종료 DB 0건이다.
이것은 GOV-018의 범위 보강이다. 앞의 20개 전수 회귀는 그 당시 소스의 증거이며, 이 변경 뒤에는 관련 3개를 재검증했다.
GOV-019와 원문 87개 충분성·legacy 재평가·최종 Web 인수는 계속 남는다.
근거: [실제 RED→GREEN·receipt·종료 증거](gov-staking-resume-evidence.json).

**GOV-019 누락 보강:** 주소 존재만 확인하면 안정화 단계의 이전 검증자 목록으로 통과할 수 있었다.
그 첫 관찰을 RED로 쓰지 않고, 안정화 종료를 요구한 단언으로 실제 RED를 확인했다.
기존 fixture는 45초 뒤에도 stabilizing=true·후보/검증자 4개였다. 이제 permissioned operator 세 명과
두 보조 staker를 등록해 대상의 비활성 상태에서도 생산이 이어지도록 한다.
실제 에폭 40→50→60에서 계약 목록과 후보·검증자가 정확한 주소로 3→2→3을 보였고,
각 에폭은 상태 변경 뒤의 새 경계이며 stabilizing=false였다. 재활성화 receipt의 성공·hash와
Staked emitter·indexed staker·금액도 모두 확인했다.

초기 보강 fixture는 genesis overlay를 한 단계 더 감싸 자금 배정에 실패했다.
그 실패도 보존했으며, 수정 후 같은 고정 소스에서 재활성화와 관련 세 회귀가 모두 PASS·Go skip 0이었다.
전체 make check·최종 tagged race를 통과했고, 이번 단계의 모든 시도 11곳은 활성 프로세스·종료 DB 0건이다.
보존 파일은 경로·해시·권한·소유자·mtime을 inventory로 기록하고 다시 대조했다.

원문의 최소 금액 미달 "revert 없이 단순 stake만 추가"는 현행 v1 계약과 다르다.
실제 eth_call은 error 3과 정확한 minimum-staking revert 사유를 반환했으며 비활성/금액 0 상태를 유지했다.
이것은 현행 계약의 거절 확인이며 미달 금액 거래의 채굴 성공이나 원문 충족을 주장하지 않는다.
그 차이와 원문 87개 충분성·legacy 재평가·최종 Web 인수는 계속 미완료다.
근거: [실제 RED→GREEN·에폭·receipt·보존 inventory](gov-staking-reactivation-evidence.json).

**NODE-002 정확성·후속 동기화 보강 (2026-10-14):** 과거 블록의 값이 비어 있지 않은지만 보던
검사를 정확한 전후 비교로 바꿨다. 블록 hash/state root/number, 채굴된 트랜잭션 10필드와
영수증 13필드를 이관 전·이관 직후·포크 이후 동기화 뒤에 대조한다. 다른 블록·입력·가스 값과
누락 트랜잭션을 허용하던 비교를 RED→GREEN으로 확인했다. 기존 이관 목적지의 디렉터리·파일·링크도
삭제하지 않고 거절하도록 세 보존 계약을 보강했다.

실제 원격 HEAD go-wemix `4538a9f5`와 검증한 go-wbft로 기존 datadir의 상대 bridge를 열었다.
최초 오프라인 높이는 정확히 19이고, 과거 7번 블록의 트랜잭션·성공 영수증이 그대로 남았다.
생산자만 기록 기반으로 종료하고 네 successor는 계속 구동했다. 이관 reader가 실제 successor에
연결해 포크 20을 넘어 높이 36까지 단조 증가했고, 그 블록 hash/state root/number도 successor와 같았다.
reader가 직접 봉인하는 validator가 되었다고 주장하지 않는다.

관련 W3 진단도 같은 고정 소스에서 직렬 race로 통과했다. 첫 구성의 RPC capability 차단은 보존했고
두 번째 구성에서 실제 거절을 재현했다. 두 실행의 소스 지문은 모두
`f010645827e80be7d3c26639c032650beae1e7a350e3151c6f40bbedf3263a3b`이며 실행 중 바뀌지 않았다.
세 runtime의 종료 DB와 활성 프로세스는 0개이고, 738개 보존 파일의 내용·권한·소유자·mtime을 대조했다.
원문 87개 전수 충분성·legacy·메인넷·W3/W4와 최종 Web 인수는 계속 남는다.
근거: [이관·동기화·관련 회귀 증거](data-migration-evidence.json).

**TX-017·가스 소진 롤백 보강 (2026-10-14):** 기존 CT-CONTRACT-006/007은 DSL 단언을
통과했지만 실패 전에 저장소를 쓰지 않아 상태 롤백을 증명하지 못했다. 실제 WBFT trace의
SSTORE 완료를 요구해 두 케이스 모두 RED를 확인했다. 이제 슬롯 0을 42로 초기화하고,
조회 분기와 SSTORE(1) 후 REVERT/무한 반복 분기를 가진 London 호환 fixture를 배포한다.
정확한 runtime·실패 영수증·eth_call/저장소의 전후 42 유지·독립 호출자의 정확한 수수료 차감과
후속 블록 전진을 확인한다. Native 관찰에서는 실제 채굴 거래의 hash/blockHash/calldata/가스 한도도
조회하며, trace에 오류 없이 완료된 SSTORE와 그 뒤 명령이 있어야 한다.

최종 고정 소스 `92ccd8b5ad56fcaf7f0d0482daef31006971e3c4a99b49a9f6b2e46aec12d331`에서
StableNet·WBFT·WEMIX 두 케이스씩 총 6 PASS·skip 0건을 직렬 race로 확인했다.
REVERT 한도는 실제 거래의 100000, 가스 소진은 50000이며, 각각 일부/전량 가스를 소비했다.
처음 두 RED와 보강 중간 GREEN도 보존했다. 총 8개 runtime은 활성 프로세스와 종료 DB가 0개다.
원본 증명·설정·로그 1466파일의 내용·권한·소유자·mtime을 인벤토리로 보존하고 다시 대조했다.

**TX-018 전체는 미완료다.** 이번 무한 반복 fixture는 원문의 `estimateGas - 1` 절차가 아니다.
WBFT `internal/ethapi/api.go`의 상향 오차 0.015와 gasestimator의 종료 조건도 읽었다.
따라서 추정치에서 1을 뺀 값이 반드시 부족한 가스라는 전제를 성립한 것으로 취급하지 않는다.
실제 estimate-minus-one 거래를 보냈거나 원문의 입력을 충족했다고 주장하지 않는다.
원문 87개 충분성·legacy·메인넷·W3/W4와 최종 Web 인수도 계속 남는다.
근거: [실제 RED→GREEN·채굴 거래·롤백·보존 증거](transaction-failure-evidence.json).

**13번 전체 언어 AST·netmap 재평가:** Go 전용 도구가 빌드 태그 테스트의 선언과 비Go 코드를 빠뜨리고,
잘못된 비Go 코드도 성공으로 처리하는 RED를 확인했다. 새 도구는 Go 표준 파서·Python ast·Acorn/Svelte/PostCSS·
mvdan 셸 AST·고정 Tree-sitter 문법으로 실제 소스 전체를 읽는다. Go 테스트/빌드 태그·생성 SPA·go.txt fixture도 포함한다.
Make recipe·workflow run·Docker 명령·HTML script/style·알려진 인터프리터 heredoc은 따로 파싱하며,
치환을 실행하지 않고 심볼과 변환 원문 위치/해시로 보존한다. 파싱 실패·알 수 없는 파일·링크는 실패한다.
문서·키/퍼즈 데이터·생성 증명 메타데이터와 외부 의존성/무시된 산출물의 범위를 명시했다.

고정 소스에서 14개 언어/형식의 1,922파일, Go 테스트 670파일, 내장 AST 21개를 파싱했다.
그래프는 노드 120,391개·간선 128,233개이며 파싱 누락·중복 ID·끊어진 간선은 0개였다.
독립 실행도 그래프 SHA256 284088d4756433ca73ee58fa1931dd322e1a20d1d6dc65583a05754f1ad2473a로 동일했고,
실행 전후 전체 소스 해시도 같았다. 모든 파싱 파일의 해시를 실제 원문과 다시 대조했다.
초기 파서 비정상 종료의 정확한 원인은 확정하지 않았다. 호환 버전 고정과 파서 수명 보강 뒤 전체 실행과 재현을 통과했다.
사용하지 않는 새 파서 라이브러리 약 347 MiB는 정리했고 필요한 분석 도구만 남겼다.

A8의 공개 비메서드 심볼은 1,888개였다: 외부 1,168·테스트 174·공개 시그니처 81·상수 집합 9·내부 소비 456,
도달 불가 0. 이 규칙은 메서드를 재는 것으로 오인하지 않는다. A7b의 과거 명령/입력 표는 RED→GREEN으로
현행 hardfork와 UpgradeV2/ChainCrossFork로 정정했다. 별도 폐기된 스택을 복원하거나 이름 충돌로 통합을 결정하지 않는다.
netmap 문서의 현재 소유자는 resource·core/node·core/blueprint이며, 역할은 bp/en/pn만 허용한다.
N3 잔여는 §0a 7번에서 실제로 해소했고 b-5의 후속 공개 CLI 세 체인 회귀도 통과했다.
전체 make check·관련 전체 race·AST 계약을 통과했다. 새 Go 분석 패키지의 문서 누락도 전체 검사에서 잡아 정정했다.
이 그래프는 구조·선언·import·호출을 분류한 구문 그래프이며 타입 해석·전체 동적 호출·실행 커버리지를 주장하지 않는다.
근거: [도구·고정 소스·독립 그래프·실패 및 회귀](all-code-graph-evidence.json), [사용 방법](../../scripts/inventory/README.md).

**최종 공통 게이트: 미완료.** 위 변경 뒤 고정 소스에서 Web 14개 capture·독립 재현을 직렬 실행하며
WEB-10과 WEB-14, 전체 필수 race·live 검사를 포함한다. 실행 중에는 문서도 수정하지 않는다.
실행 전 검사만 보강했으며 전체Python의 기존 receipt 실패11건도 새 capture와 함께 재검증한다([범위](web-final-preflight-evidence.json)).
Docker 준비 대상·기존 파일 보존을 보강했고([증거](docker-regression-preparation-evidence.json)), 후속 AST도 설치 셸을 포함해 갱신했다([갱신](all-code-graph-web-preflight-refresh-evidence.json)). 전체 실제 인수는 남긴다.
사전SPA 빌드 격리·일반 검사 게이트 제거·필수race 캐시 금지도 보강했지만 실제 전체race 성공으로 세지 않는다([증거](web-prerequisite-isolation-evidence.json)).
선택 manifest의 공유와 Go VCS 준비 검증을 보강했고([증거](native-fixture-selection-evidence.json)), 제품 업로드·매니페스트 검증과 현재 CLI 캡처도 실제 검증했다([증거](native-product-source-identity-evidence.json)). 독립 Git 경계도 실제 추적 검사·원본 보존으로 보강했다([증거](independent-git-view-evidence.json)). 후속 AST와 전체 인수는 남긴다.
외부 체인 W3와 같은 미해결 조건은 재현이나 인계만으로 구현 완료로 바꾸지 않는다.

**9번 추가 진행:** [접속 이름 증거](attach-client-identity-evidence.json)는 논리 바이너리와
빌드 대상의 정확한 이름 비교·세 체인 실제 접속과 합의 조회를 확인했다.
[인증 서명 증거](authenticated-signing-evidence.json)는 SDK 지갑 14종의 인증 전송 계약,
실제 CLI/MCP 로컬 서명 송금 두 건과 종료 DB 제거를 확인했다.
기존 세 체인 배포 회귀도 통과했지만 선언 계정의 local deploy를 증명한 것은 아니다.
**2026-10-14 선언 계정 자산 경로 보강:** [자산 서명 증거](declared-asset-signing-evidence.json)는
faucet·deployContract·registerContract·load의 선언 keyFile 계정 로컬 서명을 확인했다.
인증 RPC 계약과 반복 load의 명시 nonce 증가를 RED→GREEN으로 검증했고, make check·관련 6패키지 race가 통과했다.
실제 stablenet BP4/EN1에서 CLI와 MCP 각각 여섯 서명(총 12건)을 제출하고 세션 PASS·배포 코드·종료 DB 0건을 확인했다.
다른 체인의 로컬 자산 실증과 고급 입력은 위 단일 체인 증거의 완료 범위가 아니다.
**같은 날 legacy·접근 목록 보강:** [전체 필드 서명 증거](specified-legacy-signing-evidence.json)는
legacy의 데이터·가스·nonce·gasPrice와 비어 있지 않은 EIP-2930 접근 목록 보존을 확인했다.
faucet·배포·등록·반복 load의 legacy 서명도 SDK 독립 구성 바이트와 대조했다.
RED→GREEN·전체 make check·관련 6패키지 race와 세 체인 BP4/EN1의 실제 CLI/MCP 실행이 통과했다.
총 72건을 로컬 서명했고, 온체인 거래 형식·데이터·가스·금액·storage key와 종료 DB 0건을 확인했다.
**배포 override 보강:** [배포 입력 증거](deployment-overrides-evidence.json)는 가스를 생략해도
명시한 nonce·수수료를 추정 요청과 SDK 서명에 보존하고, 명시 key 배포의 인자 소비 검증을 실제 코드와 맞췄다.
RED→GREEN·전체 검사·관련 race·세 체인 실제 CLI/MCP 72건이 통과했으며,
배포 12건의 추정 가스·nonce·코드·legacy 가격을 온체인 거래와 대조했다. 성공·실패 종료 DB는 0건이다.
관측 프록시의 gzip 처리 오류는 별도 RED→GREEN으로 고쳤으며 원본 전송 바이트·헤더·길이를 보존한다.
**동적 접근 목록 보강:** [동적 목록 증거](dynamic-access-list-evidence.json)는 EIP-1559의 주소·storage key
목록과 데이터·가스·nonce·수수료 보존을 RED→GREEN으로 확인했다. 가스 지정·생략 두 경로의 서명 바이트를
독립 SDK 구성과 대조했다. 빈 목록·잘못된 항목·기존 legacy 회귀, 전체 검사·관련 race가 통과했다.
세 체인 CLI/MCP 78건 중 동적 목록 송금 6건의 온체인 항목과 종료 DB 0건을 확인했다.
**목록 편집·참조 보강:** [편집 증거](access-list-editor-evidence.json)는 공통 스키마의 주소·storage key
필드를 실제 브라우저에서 클릭 추가하고 저장된 주소 참조를 검증·저장·내보내는 과정을 확인했다.
계정/선언 컨트랙트 이름을 실제 주소로 풀되 원본 선언은 바꾸지 않는다. 필요한 키 배열과 알 수 없는 필드도 검사한다.
RED→GREEN·UI 모델 19개·전체 검사·관련 race, 세 체인의 CLI/MCP 78건과 이름 참조 목록 송금 12건이 통과했다.
브라우저와 native 증명의 종료 DB는 0건이다. 브라우저 자산의 기존 출처 기록은 Linux HEAD L1 증거로 대체하지 않는다.
위 목록 편집 단계에서는 대납 고급 입력이 남아 있었다. 해당 단계의 native 목록 송금은 가스를 명시했다.
준비·추정·전송 오류를 실제 제출 거부와 구분하는 검증도 R5에서 확인해야 한다.
native faucet은 동적 수수료 경로다.
R3 나머지, R4 잔여, R1/R5/R6와 전체 Linux L1은 계속 남는다.
**대납 송금 입력 보강:** [대납 입력 증거](fee-delegation-inputs-evidence.json)는 선언한 `feePayer` 계정과
송신 계정의 SDK 이중 서명에 gas·data·nonce·수수료 상한·접근 목록을 보존한다. gas 명시·생략 경로를
독립 SDK 구성 바이트와 RED→GREEN으로 대조했다. 전체 검사·관련 6패키지 race와 세 체인의 실제 CLI/MCP가 통과했다.
Applepie를 block 0에서 활성화한 genesis를 검사하고 90건의 로컬 서명, 대납 거래 12건과 gas 추정 6건을 확인했다.
기본 StableNet genesis에서 Applepie 미활성으로 거절된 실행도 별도 자료로 보존했다. 종료 DB와 남은 테스트 노드는 0건이다.
대납 계약 생성, 새 대납 계정 필드의 실제 브라우저 검증, 다른 gas 생략 경로와 잘못된 숫자 입력은 남는다.
준비·추정·HTTP 오류와 실제 제출 거부를 구분하는 R5, 전체 Linux L1 및 최종 Web 인수도 남는다.
**제출 거부 판정 보강:** [제출 거부 증거](submission-rejection-evidence.json)는 준비·추정·nonce 조회·로컬 guard·
연결·HTTP·RPC 프로토콜 오류가 거부 판정을 잘못 통과하는 문제를 RED→GREEN으로 고쳤다.
반환 hash가 없고 제출 메서드의 typed 노드 오류일 때만 사유를 노드 메시지와 비교한다.
`keptOut`의 제출 오류 분기도 같은 판정을 사용한다. 전체 검사·관련 7패키지 race와 세 체인 실제 CLI/MCP가 통과했다.
102건의 raw 제출 중 90건은 정상 거래이고 12건은 실제 nonce·intrinsic gas 거부다. 노드의 HTTP 200·메서드·code·사유·hash 부재를 기록했다.
종료 DB와 잔여 테스트 노드는 0건이다. `keptOut` receipt 조회 실패를 무시하는 폴링과 SDK 로컬 guard에서 끝나는 blacklist 케이스는 남는다.
P2/P3 초기값과 R5/L1의 영향 케이스 전수 검증도 아직 남는다.
**keptOut receipt 관측 보강:** [관측 증거](keptout-visibility-evidence.json)는 receipt 조회가 RPC/HTTP 오류인데
블록 진행만 보고 성공하던 문제를 RED→GREEN으로 고쳤다. 조회 실패는 관측 불가 오류로 반환한다.
로컬 HTTP 오류·미존재·채굴됨의 네 상태, 전체 검사·관련 7패키지 race가 통과했다.
세 체인 CLI/MCP에서 미래 nonce 거래 6건이 실제 접수되고 두 블록 동안 receipt 없이 pending으로 남는 정상 경로를 확인했다.
전체 108건 제출에는 정상 거래 90건·pending 6건·실제 제출 거부 12건이 포함된다. 종료 DB와 잔여 테스트 노드는 0건이다.
RPC/HTTP 오류는 로컬 fault 테스트이며 native 연결 오류를 주입한 증명으로 세지 않는다. 두 블록 관측은 영구 미포함의 증거가 아니다.
Blacklist의 SDK 로컬 guard 케이스, P2/P3·R5/L1 전수와 다른 제품 잔여는 계속 남는다.
**gas 생략 calldata 입력 보강:** [calldata 증거](calldata-overrides-evidence.json)는 access list 없이 data를 보낼 때
nonce·수수료가 SDK Execute 경로에서 사라지던 문제와 legacy 추정 요청에서 nonce·gasPrice가 빠지던 문제를 RED→GREEN으로 고쳤다.
동적 nonce/수수료 조합 세 가지와 legacy를 추정 요청·독립 SDK 서명 바이트와 대조했다. 전체 검사·관련 7패키지 race가 통과했다.
세 체인 CLI/MCP 120건 제출 중 새 calldata 거래 12건의 gas·nonce·수수료·data를 실제 추정 요청과 온체인 거래로 대조했다.
기존 대납·접근 목록·거부·pending 경로도 통과했고 종료 DB와 잔여 테스트 노드는 0건이다.
실제 실행은 동적 옵션을 함께 지정한 경로와 legacy이며, nonce-only/fee-only는 SDK 테스트 범위다.
대납 계약 생성·대납 계정 UI 검증·잘못된 숫자 입력, blacklist 실제 제출과 P2/P3·전체 L1은 남는다.
**blacklist 실제 제출·MCP 프리셋 보강:** [실제 거부/프리셋 증거](blacklist-preset-parity-evidence.json)는
원래 Sender 케이스가 SDK guard에서 끝나 raw 제출 0건으로 실패하는 RED를 기록했다. 명시 gas를 추가해 실제 노드로 제출한다.
Sender·Recipient·FeePayer 원본 세 정의서의 CLI/MCP 여섯 실행과 실제 blacklist 거부 6건이 통과했다.
노드 서명·raw 서명 메서드, HTTP 200·code -32000·사유·hash 부재를 관측하고 종료 DB 0건을 확인했다.
이 실제 실행에서 발견한 MCP `chainPreset.extends` 미해석도 고쳤다. `baseDir`와 cwd/상대 문맥에서
CLI와 같은 DSL 검색·병합을 app 경계를 통해 재사용한다. run은 해석 뒤 실행하고 validate는 입력별 결과를 보존한다.
named preset·override·누락·상대 경로의 RED→GREEN, 전체 검사·관련 7패키지 race가 통과했다.
프리셋 초기값 P2/P3, 다른 영향 케이스의 전체 R5/L1, 대납 계약 생성·숫자 검증·대납 UI 등은 남는다.
9번 전체 상태는 **진행**을 유지한다.


**트랜잭션 숫자 입력 보강:** [숫자 입력 증거](transaction-number-evidence.json)는 gas·nonce·수수료·value·amount의
소수·음수·null·잘못된 문자열·JSON 안전 범위 초과를 RPC와 지갑 호출 전에 거절한다. 큰 정수는 decimal/hex 문자열로
정확히 보존하고 gas·nonce의 64비트 초과도 거절한다. RED→GREEN·전체 make check·관련 7패키지 race·UI 모델과
실제 브라우저의 검증·저장·내보내기가 통과했다. 세 체인 CLI/MCP의 잘못된 입력 84건은 fail 1·다른 판정 0으로
영속되며 추가 거래를 제출하지 않았다. 기대된 노드 거부를 포함한 제출 120건의 회귀도 통과했고 종료 DB와 테스트 노드는 0건이다.
load의 blocks·fillPercent 등 제어 숫자, 대납 계약 생성·대납 필드 브라우저 검증과 9번 나머지·최종 Web 인수는 남는다.

**load 제어 숫자 보강:** [제어 입력 증거](load-control-evidence.json)는 `blocks`·`fillPercent`의 소수·0·잘못된 값·
범위 초과를 연결·서명 전에 거절한다. 생략한 blocks의 기본값은 유지하고 정수 변환은 64비트 문자열도 정확히 보존한다.
공통 Web 계약도 숫자와 decimal/hex 문자열의 양수·백분율 범위를 검사한다. RED→GREEN·전체 make check·관련 7패키지
race·CLI suite 추가 race·UI 모델·실제 브라우저 저장/내보내기와 세 체인 직렬 native race가 통과했다.
실패 144건 중 새 load 제어 실패 60건은 추가 제출 없이 fail 1·다른 판정 0으로 영속됐고 기대된 노드 거부를 포함한 제출 120건도 회귀했다.
성공·실패 실행과 브라우저의 종료 DB는 0건이다. gas와 fillPercent를 함께 넣은 첫 증명 입력의 충돌은 별도 실패 자료로
보존하고 바로잡은 뒤 전부 다시 검증했다. 이전·이번 브라우저 증명은 향후 capture가 덮어쓰지 않는 소유 런타임 폴더에
내용·권한·소유자·mtime 동일성을 확인해 복사했다. 다른 제어 필드의 잘못된 명시 값 처리와 R4·9번 잔여는 남는다.

**대납 계약 생성·계정 선택 보강:** [계약 생성 증거](delegated-creation-evidence.json)는 선언 대납 계정과 키 입력을
`deployContract`의 이중 서명에 연결하고 init code·value·nonce·수수료·접근 목록과 가스 지정/추정을 보존한다.
별도 override가 없어도 계약 생성용 전체 거래를 사용한다. 충돌·빈 입력·대납 수수료 쌍 누락·노드 서명 조합은
실행 전에 거절한다. SDK 독립 바이트 대조, 추정 실패의 제출 차단, RED→GREEN·전체 make check·관련 7패키지 race와
새 임베드 Web 번들이 통과했다. 실제 브라우저는 선언 계정을 클릭 선택하고 충돌 필드 추가를 막으며, 잘못된 import를
보존한 채 검증하고 키 바인딩을 저장/내보냈다. 세 체인 CLI/MCP에서 가스 지정·추정·기본값 대납 계약 18건을 실제로
생성해 거래·수수료 대납자 서명·코드·잔액·영수증과 대조했다(Applepie block 0 명시). 기대된 노드 거부 12건을 포함한
제출 138건과 잘못된 숫자 입력 144건도 회귀했다. 성공·실패 브라우저와 native 실행의 종료 DB는 0건이며 증명은
내용·권한·소유 정보를 보존한 별도 런타임 경로에 둔다. 키 입력·일반 생성의 접근 목록은 SDK 계약 증거로 구분한다.
다른 제어 입력·R3/R4 잔여·P2/P3·R1/R6·전체 Linux L1·10–14번과 최종 Web 인수는 계속 남는다.

**P2/P3 초기값 완료:** [프리셋 초기값 증거](preset-values-evidence.json)는 `values`의 JSON 타입과 기존 깊은 병합·
null 삭제·배열 교체를 유지한다. 초기값은 첫 사전 hook부터 참조할 수 있고 오프라인 참조 검사도 인식한다.
실행마다 깊게 복사하며 이후 save 결과의 덮어쓰기는 해당 실행에만 남는다. 변경한 초기값은 케이스 의미 지문을 바꾼다.
구조화 Web 편집은 불리언·중첩 null·초기 참조·import/export를 보존하고, 공유 문서에 초기 개인키를 넣어 서명하는
우회는 기존 키 보호 규칙으로 거절한다. RED→GREEN·전체 make check·관련 7패키지 race·UI 모델·실제 브라우저와
세 체인 직렬 native race가 통과했다. CLI/MCP의 두 케이스 실행을 각각 수행해 초기값 송금 12건과 첫 hook의 배열
소비·케이스 격리를 확인했다. 전체 제출 150건(기대된 노드 거부 12건 포함)과 숫자 오류 144건도 회귀했다. 종료 DB는
0건이고 증명은 별도 보존 경로에 두었다. R1 체인 ID 실행 관문은 케이스 assertion과 구분해 미완료로 둔다.
R3/R4 잔여·P4/R6·나머지 R5/전체 Linux L1·10–14번과 최종 Web 인수는 계속 남는다.

**R1 선언 체인 ID 관문 완료:** [체인 ID 증거](chain-identity-evidence.json)는 preset의 `values.chainId`와
기록된 모든 실행 노드의 인증된 `eth_chainId`를 사전 hook보다 먼저 비교한다. 명시 0도 검사하며, 불일치는
FAIL, 조회 불가·사용 불가 응답은 BLOCKED다. 여러 노드에서 불일치와 조회 불가가 함께 나타나면 FAIL로 판정한다.
선언이 없으면 기존 동작을 유지하며 관측한 값으로 기대값을 만들지 않는다. 구조화 Web의 정수·문자열 입력,
소수 거절과 내보내기도 확인했다. RED→GREEN·전체 make check·관련 race·브라우저·세 체인 직렬 native race가
통과했다. CLI/MCP의 18판정에서 90개 노드 관측과 실패/차단 시 거래 미제출을 확인했고, 전체 제출 156건
(기대된 거부 12건 포함)·숫자 오류 144건을 회귀했다. 증명별 소스 SHA 92개가 일치하고 종료 DB는 0건이다.
첫 native fixture의 BLOCKED 종료 코드 기대값 오류는 수정 후 세 체인 모두 재실행했으며 실패 자료도 보존했다.
별도 fresh compose 인수와 Linux HEAD L1을 통과한 것으로 해석하지 않는다. 9번 나머지와 10–14번은 계속 남는다.

**P4 선언 능력 사유 보강:** [능력 판정 증거](capability-reason-evidence.json)는 기존 `requires`와 preset의
`attach.provides`/`genesis.provides` 계약을 유지하며, 부족한 능력 목록 또는 대상 체인 불일치를 구체적으로
기록한다. 예상된 SKIP과 예상하지 못한 SKIP의 FAIL 정책을 유지하고 `capability-gate.json`에 선언 입력과
판정을 보존한다. RED→GREEN·고정 소스 전체 make check·관련 race와 세 체인의 실제 CLI/MCP가 통과했다.
18판정은 의도한 PASS 6·SKIP 6·정책 FAIL 6이며 Go 테스트 실패·skip은 0건이다. 긍정 사례는 제공 선언을
소비해 실제 인증된 블록과 선언 계정 잔액을 읽었다. 기존 156제출·12기대 거부·144숫자 오류·18체인 ID 판정도
회귀했고, 증명별 소스 SHA 96개가 일치하며 종료 DB는 0건이다. 실제 기능 지원의 검증은 R3에 남는다.
P4 전체 사전 요구의 충분성, 9번 나머지, 10–14번과 최종 Web 인수를 이 부분 증거로 완료하지 않는다.

**R3 합의 API 여러 노드 확인:** [노드별 합의 증거](consensus-nodes-evidence.json)는 첫 노드뿐 아니라
기록된 모든 실행 노드를 각 엔드포인트의 인증으로 확인한다. 노드별 지원·미지원·조회 불가와 실제 출처를
보존하고, 조회 불가는 미지원보다 우선해 BLOCKED로 처리한다. 확인 가능한 노드 중 미지원이 있으면 기존
SKIP·예상하지 못한 SKIP의 FAIL 정책을 적용한다. 기존 첫 노드 필드는 유지하며 각 호출에는 5초 제한을 둔다.
RED→GREEN·전체 make check·관련 네 패키지 race·세 체인 직렬 native race가 통과했다. CLI/MCP 12실행에서
60노드 관측(지원 54·실제 접속 불가 6)을 확인했고 native 차단 사례는 step 파일을 만들지 않았다.
인증된 단위 fixture에서 미지원·조회 불가 때 PreSpec과 사전 hook·테스트가 실행되지 않는 것도 확인했다.
실제 method 미지원과 미지원/조회 불가 혼합은 인증된 단위 fixture로 검증했으며 native 응답은 위조하지 않았다.
기존 156제출·12기대 거부·144숫자 오류·18체인 ID 판정도 회귀했고 증명별 소스 SHA 98개가 일치한다.
종료 DB는 0건이다. R3의 다른 RPC·VM·거래·계정 상태·포크 검증과 P4 충분성, 9번 나머지·10–14번은 남는다.

**R3 PUSH0 실제 실행 확인:** [VM 증거](vm-capability-evidence.json)는 기존 `evm:shanghai` 요구의 opcode
적격성을 모든 실행 노드에서 읽기 전용 `eth_call`로 확인한다. 정해진 제한 가스·바이트코드의 정확한 반환값을
확인하며 거래는 제출하지 않는다. 실제 PUSH0 미지원 오류는 SKIP이고, 조회용 API 부재·잘못된 인자·무관한
실행 오류·사용 불가 반환·HTTP 실패는 BLOCKED다. 체인 ID·합의 관측은 별도 파일로 유지한다.
세 경계 RED→GREEN·최종 전체 make check·관련 네 패키지 race·세 체인 직렬 native race가 통과했다.
CLI/MCP 6실행·30노드에서 StableNet/WBFT의 지원 20관측과 WEMIX의 실제 미지원 10관측을 확인했다.
의도한 VM 판정은 PASS 4·SKIP 2이며 Go 테스트 실패·skip은 0건이다. VM 거래 제출은 0건이며 기존
156제출·12기대 거부·144숫자 오류와 체인 ID·합의 회귀도 통과했다. 소스 SHA 100개씩이 일치하고 종료 DB와
관련 프로세스는 0건이다. 모든 Shanghai 규칙이나 다른 VM·precompile·계정·거래·포크까지 완료한 증거는 아니다.
9번 나머지·P4 충분성·10–14번과 최종 Web 인수는 계속 남는다.

**R3 P256 동작·실제 SKIP 정책 보강:** [P256 증거](p256-capability-evidence.json)는 기존 corpus의 유효 서명을
표준 암호 검증으로 독립 대조하고, 같은 공개키의 무효 서명도 모든 노드에 읽기 전용 호출한다. 유효·무효 입력을
같이 승인하는 상수 반환, 사용 불가 응답과 조회 실패는 BLOCKED다. 두 입력이 모두 비어 있으면 미지원 SKIP이다.
정적 제공 태그만으로 `skipsOn`을 낡았다고 판단하던 결함도 합의·VM·P256 6사례 RED→GREEN으로 수정했다.
실제 미지원이면 예상 SKIP을 유지하고 실제 지원이면 낡은 선언을 hook/본문 전에 FAIL로 처리한다.
최종 전체 make check·관련 네 패키지 race·네 구성 직렬 native race가 통과했다. StableNet 기본/Boho 활성,
WBFT, WEMIX에서 CLI/MCP 8실행·40관측(지원 20·미지원 20)을 확인했다. 같은 StableNet 바이너리의 fork
설정에 따라 결과가 바뀌며 비활성 망의 명시 `skipsOn`도 올바르게 처리된다. P256 거래 제출은 0건이고
기존 208제출·16기대 거부·192숫자 오류·24체인 ID 판정과 합의/VM 회귀도 통과했다. 소스 SHA 104개씩이
일치하고 종료 DB·관련 프로세스는 0건이다. 모든 precompile의 입력 경계나 정확한 가스 비용까지 증명한 것은
아니다. R3 다른 조회·계정·거래·포크, P4 충분성, 9번 나머지·10–14번과 최종 Web 인수는 계속 남는다.

**정수 제어 입력 18필드 보강:** [증거](unsigned-control-evidence.json)는 waitBlock·sendTx·load·노드 로그·블록 관찰·WS·리더의
소수·잘못된 타입·범위 초과를 소비자 IO 전에 거절하고, 0과 생략·decimal/hex·64비트 값을 구분한다.
런타임 범위와 Web 스키마는 같은 규칙을 읽는다. keptOut의 블록 덧셈 overflow와 잘못된 WS count의 구독 정리도
RED→GREEN으로 보강했다. 전체 make check·관련 race·브라우저 저장/내보내기와 네 프로필 직렬 native race가 통과했다.
StableNet 기본·Boho 활성·WBFT·WEMIX의 CLI/MCP에서 제어 실패 144건은 fail 1·다른 판정 0이며 제출은 추가되지 않았다.
기존 숫자 실패 192건·거래 제출 208건도 회귀했다. 각 프로필의 소스 114개 해시를 현재 파일과 대조했으며 종료 DB는 0건이다.
증명 입력의 expect 누락·단언 오류 기록 위치·WS 불필요 인자는 실패 자료를 보존하고 사전 계약 검사로 보강했다.
이 증명 입력 실패는 생산 코드의 RED나 통과 자료로 세지 않는다. 대기 시간·기본값·R4 나머지, R3/P4/R5/R6·Linux L1·
10–14번과 후속 소스의 Web 최종 인수는 계속 남는다.

**대기 시간 입력·기본값 보강:** [증거](duration-control-evidence.json)는 timeout·pollInterval·within 22필드의
0·음수·빈 문자열·잘못된 타입·범위 초과를 소비자 IO 또는 제어 전에 거절한다. 생략한 기존 기본값과
소수 단위·복합 시간·microsecond·양수 int64 nanosecond를 보존한다. blockHalt·crossFork의 기존 기본값도
Web에 표시한다. 잘못된 wsCollected 시간은 열린 구독을 닫고 JSON 가능한 관측을 남긴다.
실제 결함 RED→GREEN·전체 make check·관련 다섯 패키지 race·브라우저 저장/내보내기와 네 프로필 native race가 통과했다.
attach 가능한 21필드의 CLI/MCP 실패 168건은 추가 제출 없이 fail 1·다른 판정 0으로 기록됐다.
기존 정수 제어 실패 144건·숫자 입력 실패 192건·거래 제출 208건도 회귀했다. 소스 120개씩 해시가 일치하며
종료 노드 DB·프로세스는 0건이고 여유 공간은 약 38 GiB다. crossFork는 제어 callback 차단의 단위 증거이며
실제 fork handover 증거가 아니다. 업그레이드가 없는 attach 입력은 사전 검사에서 거절되므로 라이브 fixture에서 제외했다.
이 한계와 실제 포크 제어·R4 전수·R3/P4/R5/R6·Linux L1·10–14번·후속 Web 최종 인수는 열린 상태로 보존한다.

**R6 여러 preset 실행·출처 기록 완료:** [증거](preset-matrix-evidence.json)는 같은 v2 참조/extends 케이스 묶음을
CLI `--chain-presets` 반복과 MCP `chainPresets` 배열로 직렬 실행한다. 같은 `test.run-matrix` 등록 usecase를 사용하며,
선택 대상마다 workspace/session을 분리하고 결과·session/status/report에 preset ID를 보존한다. 모든 대상 사전 검사,
대상 실패 뒤 계속 실행, 취소·소유 망 정리 실패 시 후속 기동 차단, 실행 없는 구성 계획을 검증했다.
출처 ID를 실행 의미와 분리해 v1 이관 동등성을 보존했으며 MCP chainId의 정확한 전달도 RED→GREEN으로 보강했다.
전체 make check·관련 여덟 패키지 race·추가 계획/취소 계약·실제 브라우저 회귀와 세 체인 native 직렬 증명이 통과했다.
CLI는 6 PASS, MCP는 의도한 WBFT ID 불일치 2 FAIL을 보존하고 다음 WEMIX까지 4 PASS를 기록했다.
소스 89개 해시가 일치하고 종료 노드 DB·프로세스는 0건이다. CLI DB는 MCP 전에 회수하며 증명 폴더는 약 9.9 MiB다.
91개 app usecase의 등록 누락도 0건이다. 이 단계는 Web 다중 대상 기동 폼을 추가하거나 전체 공통 TC의
바이너리/버전/fork/서명/skip 출처 충분성을 완료한 근거가 아니다. Linux L1·R3/P4/R4/R5·10–14번과
후속 소스의 Web 최종 인수는 계속 남는다.

**R3 RPC·WS·시스템 컨트랙트 확인 완료:** [관측 증거](network-capability-evidence.json)는 모든 기록된 노드의
`eth_blockNumber`, 기록된 WS 주소의 인증된 `eth_subscribe(newHeads)`, 매니페스트 시스템 컨트랙트의 코드를 확인한다.
바이너리 내장 accountManager는 코드가 없으므로 `isBlacklisted(address)`의 boolean 응답으로 확인한다.
지원·미지원·조회 불가를 구분하며 조회 불가는 다른 미지원 판정보다 우선해 BLOCKED로 남긴다.
관측에는 노드·역할·메서드·주소·코드 해시를 보존하고 URL·자격증명·원문 오류를 쓰지 않는다.
WS 구독은 응답 대기 중 취소·빈 ID 거절·typed RPC 오류·명시 종료와 알림 버퍼 종료를 보강했다.
RED→GREEN·전체 make check·관련 여섯 패키지 race·실제 브라우저와 네 native 프로필의 직렬 race 증명이 통과했다.
새 캠페인 24실행은 16 PASS와 의도한 8 BLOCKED를 기록했고, 노드 관측 320건은 supported 312·unknown 8이다.
숫자 입력 실패 192건·정수 제어 실패 144건·시간 입력 실패 168건과 기존 raw 제출 208건도 회귀했다.
각 증명의 소스 123개 해시가 일치하며 종료 DB·관련 프로세스는 0건이다. 설정·로그·세션·증명은 보존했다.
RPC 전체 메서드, WS 이벤트 완전성, 계약 ABI/버전/권한/포크 동작을 확인한 근거는 아니다.
R3 나머지·P4 충분성·Linux L1·9번 잔여·10–14번과 후속 Web 최종 인수는 계속 남는다.

**WS 응답·알림 격리와 편집기 경합 수정:** [증거](ws-protocol-evidence.json)는 요청 ID·JSON-RPC 버전이 다른
구독 응답을 거절하고, 다른 구독의 알림을 소비자에게 넘기지 않는다. 다섯 응답 오류와 알림 혼선을 RED→GREEN으로 확인했다.
브라우저 회귀에서 발견한 가져오기/편집 경합도 응답을 의도적으로 지연해 RED로 재현했다. 비동기 처리 중에는 선언 편집과
새 케이스·파일 선택을 비활성화하고 완료 후 편집·저장·내보내기를 확인했다. 첫 지연 테스트의 route 정리 오류는 제품 RED로 세지 않는다.
관련 여섯 패키지 race·프론트 28개 테스트·SPA 빌드·전체 make check·실제 브라우저·네 native 프로필 직렬 race가 통과했다.
각 native 소스 해시 125개가 일치하며 종료 노드 DB는 0건이다. 이 단계는 R3 전체나 후속 Web 최종 인수를 닫지 않는다.
정리 도구의 기본 증명 검색 범위가 새 docs/dev JSON/Markdown을 빠뜨리는 누락은 다음 보강 단계에서 별도로 검증한다.

**저장 공간 정리의 새 문서 참조 보강:** [증거](storage-document-evidence.json)는 docs의 JSON/JSONL/Markdown과
루트 인수인계 Markdown을 기본 보존 참조로 읽는다. 참조 누락·읽기 실패를 RED로 확인했고, receipt가 없거나 해시가 다르면
완료로 세지 않는 감사 계약까지 47개 보존 테스트와 전체 make check가 통과했다. 실제 전용 폴더에서 새 문서 참조 보존,
미참조 자료 삭제, 열린 WAL 보존, 내용·권한·소유자·mtime·xattr·ACL을 유지한 투명 압축을 확인했다.
기존 `/private/tmp/cbui-cleanup.py`는 원본을 백업하고 테스트된 fail-closed 도구를 호출하도록 교체했다.
실제 runtime 182개 계획은 177 보존·5 삭제 후보이며 운영 자료 삭제는 하지 않았다.
과거 publication audit의 56 receipt는 28 일치·26 경로 없음·WEB-08 2개 다른 해시다. 둘의 수정 시각은 이번 작업 전인
2026-10-13이다. 원인을 추정하거나 과거 audit를 고치지 않았다. 실제 감사는 이 한계를 기록하고 code 2를 반환한다.
보존 코드와 새 참조 실증은 완료됐지만 1번 전체 자료 확인과 후속 소스의 고정 인수는 열린 상태다.

**10번 준비물 공유 완료:** [증거](prepared-sharing-evidence.json)는 케이스가 `prepared.exports`로 공개한 값을
뒤 케이스가 `prepared.imports`의 `{case, name}`으로 가져오게 한다. 한 실행·동일 환경에서만 공유하고 결과는 깊이 복사한다.
PASS 뒤에만 모든 공개 값을 원자적으로 게시한다. 실패·blocked·skip한 공급자와 누락 자료는 소비 케이스를 훅/환경 생성 전에
BLOCKED로 남긴다. 사적인 byte buffer·opaque handle·기존 scrub 대상 비밀 필드와 saveKey 공개는 거절한다.
개별 작성은 선언·변수만 검증하며 실행 목록은 공급자 누락·순서 역전·미선언 이름·중복을 사전 검사한다.
별도 정의 실행 모드는 의존 케이스를 부분 실행하기 전에 `--suite` / MCP `suite=true`를 안내한다.
v1 이관·v2 스키마·구조화 UI의 저장된 공급자/공개 이름 선택·import/export를 연결했다. 스키마의 중첩 Go 타입도 래칫에 등록했다.
RED→GREEN·프론트 31개·SPA 빌드·전체 make check·관련 여섯 패키지 전체 race가 통과했다.
실제 브라우저에서 한 번 배포한 계약을 다음 케이스가 읽고 호출해 2 PASS를 확인했다. 네 native 프로필의 CLI/MCP
16캠페인·32판정은 16 PASS·의도한 8 FAIL/8 BLOCKED다. 캠페인마다 공급자만 2거래를 제출하며 소비자는 재제출하지 않는다.
각 증명의 소스 해시 134개가 일치하고 종료 DB·관련 프로세스는 0건이다. 브라우저의 선언/배치 혼동은 실제 RED로 고쳤고,
잘못된 증명 선택자와 step JSON 필드명은 fixture 오류로 구분해 보존했다. 10번의 동기화 관찰·9번 나머지·11–14번·
과거 receipt 미확인과 최종 Web 인수는 계속 남는다.

**10번 동기화 관찰 부분 보강:** [증거](sync-path-evidence.json)는 원격 머신 조회 실패를 다른 HTTP 프로세스의 값으로
대체하던 경로, HTTP 실패 본문·무한대·직접 HTTP 클라이언트의 크기 제한을 넘은 응답의 거짓 성공을 RED→GREEN으로 막았다.
metric 표본마다 노드·역할·현재 PID·조회 시각·출처·값·관측 상태를 기록한다. 선택 항목 `alternatives`는 같은 노드의
선언한 모든 지표가 유한값으로 관측되고 하나 이상이 비교를 만족해야 한다. 구조화 UI의 클릭 작성·중복 거절·저장·내보내기,
프론트 31개·전체 make check·관련 6개 패키지 race가 통과했다. full/snap/누락 복구/실시간/receipt 가격 케이스는
블록 해시·state root·잔액·receipt 내용도 비교하며 수신량만으로 상태 반영을 판정하지 않는다.
stablenet·WBFT 각 5개와 WEMIX 4개가 최종 동결 소스로 직렬 race 증명을 통과했다. 표본 16개·소스 해시 70건 일치,
종료 DB 0개다. 추가한 RPC 비교의 십진 높이 오류는 `format: hex`로 수정했고 fixture 오류로 따로 기록했다.
WEMIX 실시간 케이스는 같은 높이의 hash/stateRoot가 맞아도 두 fetcher 지표가 0이어서 엄격한 검사가 실패한다.
원본 Prometheus·expvar, 실제 바이너리 초기화 추적과 소스를 대조해 LES가 먼저 같은 이름을 등록하고 ETH 등록 오류가
무시되는 외부 결함을 확인했다. 외부 코드를 고치거나 조건을 낮추지 않았다. **15개 전체와 10번 전체는 미완료**다.
기존 W3·Linux fleet/L1·9번 나머지·11–14번·과거 receipt 확인·최종 Web 14개 인수도 계속 열린 상태다.

## 1. 개발 우선순위 (walking skeleton 최단 도달 → 심화)

크리티컬 패스 = 스켈레톤에 필요한 각 컴포넌트의 **얇은 버전**부터.

| 순위 | 작업 | 이유 | 상태 |
|------|------|------|------|
| 0 | **`pkg/` → `internal/` 마이그레이션** | 앱은 internal이 정석(외부 importer 0, 컴파일러 강제 캡슐화); 신규 패키지 전 정리 | ☑ |
| 1 | **P0 인터페이스 동결** | 전 작업 언블록·병렬 TDD 개방 | ☑ |
| 2 | **Session(M1)** | 모든 기록의 정본, 기반 | ☑ |
| 3 | **Place(M2)+용량검증** | 배치·포트(고정포트 충돌 제거) | ☑ |
| 4 | **procman 배선+확장(M6 핵심)** | 최우선 안전 갭(검증 없는 Kill→고아 위험) | ☑ |
| 5 | **testspec Parse+Fingerprint** | spec 실행·env 재사용 | ☑ |
| 6 | **keyreg(M3) + assert funcs** | 신원·검증 | ☑ |
| 7 | **Provisioner(M5) → Supervisor(M6)** | 물질화·기동·teardown | ☑ |
| 8 | **Collector RPC-min + Interpreter-min** | 스켈레톤 최소 관측·실행 | ☑ |
| 9 | **★ Engine walking skeleton** | 첫 통합 증명(1체인·local·4노드·tx1) — **실 gstable 라이브 e2e 통과(RunSpec 실행 수직)** | ☑ |
| 10+ | 수직 슬라이스 확장: remote☑ · attach☑ · 표면(capability/CLI/MCP/-race/백프레셔)☑ · Collector 심화☑ · stablenet☑ · DSL 어휘(바인딩·fault·자산·logs·gas·ws)☑ · read-기반 스위트 이관☑ · 업그레이드(T5.2)☐ · wemix4 이관(T5.5)☐ · 레거시 제거(소비자 이관 후)☐ | 수직 슬라이스 확장 | ◐ |

---

## 1b. 진행 현황 요약 (2026-08-09 기준)

**walking skeleton 완성 · 재설계 엔진(H1)이 CLI·MCP 양쪽에서 도달 가능 · 실행 수직 전체가 CI(mock/attach)+라이브(gstable) 커버.**

**2026-08-09 x-bar 정렬 검토 후속 (이 브랜치):** 문서 정합(T6.5·T6.5b·audit 스냅샷 강등) → **DSL 보충어 어휘 확장**(스텝 값 바인딩 `save`/`$ref`+`read` · fault 액션 5종 · 자산/컨트랙트 액션 3종 · `logs` 이벤트 어세션 · tx fee-cap/nonce 인자 · `gasPrice`/범용 `rpcCall`/`wsSubscribe`) → **supervisor 선언 논항 방출**(T3.2b) → **원격 SSH tail**(`LogReader` boundary). 액션 2개·어세션 12개 → **액션 11개·어세션 16개**. 라이브 증명: `TestEngine_Live_NewVocabulary`(실 gstable 4노드). 부수적으로 **라이브에서만 드러난 결함 2건 수정** — `procman` 이 loopback 호스트를 원격으로 오판(단일 노드 정지 불가), 실패 스텝의 사유가 아티팩트에 기록되지 않음.

**재조정 검토 (#225 머지 후):** 병합 상태 green 확인(`go build/vet ./...`·`golangci-lint`·`go test ./...` 52 pkg 전부 통과). 진행 재정리: **T6.5·T6.5b 완료**(문서 pkg→internal 정리·§2b/§3/§5 실측 갱신), **T4.2 인벤토리 현행화**(액션 11·어세션 16). **표현력 블로커는 전부 해소** — 이제 남은 건 **작업량·환경 의존** 뿐: (1) 케이스별 스위트 이관(anzeon 거버넌스·wbft·api·repro — 소스 있음, 라이브만 바이너리), (2) **T5.2 업그레이드 멀티바이너리**(gwemix+etcd 필요), (3) **T5.5 wemix4 이관**(대규모), (4) **레거시 제거**(testkit/testrun — `test`/MCP/upgrade 소비자 이관 선행), (5) **실 SSH/바이너리 라이브 검증**(사용자 환경). 표현력이 아닌 이 항목들이 다음 우선순위 결정 대상.

**완료 (PR):**
- Phase 4 walking skeleton — Engine 오케스트레이션·빌트인 tx/rpc·RunSpec·BuildEnv(AssemblePlan/GenesisSource/orchestration)·최상위 `NewLocalEngine`. **실 gstable 4노드 라이브 e2e**(BuildEnv 기동 + RunSpec 실행 + capstone `Engine.Run`) 통과. (#188~#197)
- launcher 물질화 `provision.Provisioner` 경유(B) (#198)
- SSH key_file 인증 + `kill -0` 종료검증(procman) (D) (#199)
- remote 슬라이스: `RemoteFileSink` + 원격 가능 launcher(Initializer 라우팅) (C) (#200)
- attach 모드: `NewAttachEngine`(RPC-only, mock RPC 로 CI e2e) (#201)
- Phase 6 표면: capability 게이팅(spec `requires`)·`-race` 게이트·CLI `chainbench run` (#202)
- Phase 6 표면: MCP `chainbench_run`·공유 `ReadSessionSummary`·obs 백프레셔(O7) 테스트 (#203, 리뷰중)

**남은 작업:**
- ~~**T6.3 dashboard**~~ ☑ 엔진이 obs 이벤트 emit(`Deps.Emit`/`Bus`) → `chainbench run --dashboard <url>` 가 `dashboard.Forward` 로 chainbench-dashboard 에 스트리밍(라이브). + ☑ **완료 세션 디스크 조회**(F15 AC3): `session.List`/`SessionFilePath`/`ChainstatePaths` + dashboard `/api/sessions`·`/api/sessions/{id}`(session.json)·`/api/sessions/{id}/chainstate`(chainstate.jsonl), `chainbench-dashboard -artifact-root` 로 활성화. attach+mock RPC end-to-end + httptest 로 CI 커버.
- **T5.2 업그레이드 멀티바이너리**(wemix+wbft 핸드오프) — gwemix+etcd 바이너리 필요, 이 환경 라이브 검증 제한.
- ☑ **T5.4 stablenet**(ACL 플러그인·Core 무변경 검증) — stablenet 거버넌스 시나리오가 DSL 엔진에서 실행됨을 CI 검증(예제 `stablenet-governance-read.json` + govbind `proposals(uint256)` calldata·mock RPC e2e `call` 어세션 pass). core 무변경. 실 gstable 라이브(RunSpec/BuildEnv)는 기존 게이트 테스트로 커버.
- **T5.5 wemix4 이관**(레거시 스위트 → DSL) — 대규모.
- **Collector 심화(T3.3)** — ☑ 로컬 live tail(스캔→tail·부분줄 안전)·bp참여 집계(head producer·window prune)·fork/reorg 검출(높이별 hash 불일치)·**엔진 배선**(local/attach 이 `Bus` 설정 시 collection 실행→chainstate·로그를 obs 로 미러, teardown 시 정지)·**chainstate 세션 영속화**(`chainstate/chainstate.jsonl` — F10/F15 jsonl+obs 미러). ☐ 원격 SSH tail(사용자 SSH 환경 필요, T5.1 계열).
- **실 원격 SSH 호스트 라이브 e2e**(RemoteFileSink+RemoteDriver) — SSH 대상 필요(사용자 환경).
- [x] **레거시 경로 정리 — 완료 (2026-09-12 실측).** 이 항목이 "소비자 이관 전 제거 금지" 로
  끝나 있었는데, **제거가 이미 끝났다**: `internal/testkit`·`internal/testrun`·`internal/testspec`
  세 디렉토리가 모두 없고, `RunCase` 는 프로덕션에서 참조되지 않으며, 은퇴 계획 문서는
  `docs/dev/archive/legacy-retirement-plan.md` (2026-09-24 삭제) 로 보관됐다.
  재확인: `for p in internal/testkit internal/testrun internal/testspec; do [ -d $p ] && echo "$p EXISTS"; done`
  (아무것도 출력되지 않아야 한다) · `grep -rn "RunCase" internal/ cmd/ | grep -v _test`
  **(원래 기록)** 착수: 죽은 심볼 제거(`testkit.RunCase`)·레거시 패키지 signpost(testkit·testrun→engine+testspec)·**은퇴 계획 문서**(`legacy-retirement-plan` (2026-09-24 삭제) = `docs/dev/archive/legacy-retirement-plan.md`: 매핑·순서·블로커·DSL 표현력 갭). **suite 이관 착수**: `tests/` Go-func 케이스는 repo 에 있어 DSL 포팅 가능(라이브만 바이너리 필요) — `onEach` 다중노드 어세션 수정 + 신규 빌트인(`blockAdvance` 헤드 전진·`sameBlockHash` 노드간 no-fork, `rpc.BlockByNumber` 기반) 추가로 **`tests/network` 3케이스 전부 DSL 포팅 완료**(`network-peers.json`·`network-health.json`). **`tests/anzeon` 착수**: read-shape 계열(adapter-code `codeAt NotEqual`, balanceOf/isAuthorized `call Regexp`)→`stablenet-system-contracts.json`, base fee 경계(`baseFee` 신규 빌트인, `rpc.BlockByNumber` baseFeePerGas)→`stablenet-gas-policy.json`, 하드포크 아티팩트(P-256 precompile `call`·GovMinter `codeAt`·chainId/blockNumber)→`stablenet-hardfork.json`, 가스 추정(`estimateGas`=`rpc.EstimateGas`)→`stablenet-estimate-gas.json`, token-metadata(`call`+`Contains` 심볼)→`stablenet-token-metadata.json` 포팅. **read-기반 anzeon 이관 완료**(시스템컨트랙트·getter·base fee·하드포크·estimate-gas·token-metadata). **잔여 anzeon 6범주(교차-call 비교·거버넌스 다단계·gasTip 조합·fee-cap tx·명시 nonce·WS)의 표현력 블로커는 2026-08-09 전부 해소** — `save`/`$ref` 바인딩+`read`(T4.2a)·`logs`(eth_getLogs)·`gasPrice`+범용 `rpcCall`·tx fee-cap/nonce 인자·`wsSubscribe`. 예제: `stablenet-token-invariants`·`governance-event-flow`·`gas-policy-derived`·`stablenet-fee-boundary`·`stablenet-nonce-ordering`·`ws-subscribe-heads`. **라이브 증명**: `TestEngine_Live_NewVocabulary`(실 gstable 4노드, GSTABLE_BIN). 남은 건 표현력이 아니라 **케이스별 이관 작업량**(`legacy-retirement-plan` §4.3). `test`/MCP/upgrade 가 아직 레거시 사용 → 소비자 이관 전 제거 금지.

---

## 1c. 재계획 (2026-08-11) — 배경요구 재대조 후 잔여 작업

> 근거: [[dsl-v2-proposal]] · `chain-binary-flag-graph` (2026-09-24 삭제) · `archive/structure-and-atomic-cli-proposal`
> (2026-08-11 검토 3종 — 셋째는 제안분이 구현되어 `archive/README` (2026-09-24 삭제) 로 이동했다).
> 이 절은 §2 의 Phase 목록을 대체하지 않고, **배경 요구(체인 구성 5요소 · 실행옵션 · 3-검증원)** 대조에서
> 새로 드러난 갭과 그 순서를 얹는다. §2 의 미완 항목(T5.2·T5.5·레거시 제거)은 그대로 유효하다.

### 완료

- ☑ **T7.1 keyreg 프로덕션 배선** — `keyreg.New` 의 프로덕션 호출 지점이 0 이던 문제(배경 1.4·1.5 / 알고리즘 2·3 미구현)를 해소.
  - `session.NewWithKeys(baseDir, cmd, at, keyreg.Deps)` 신설 — **레지스트리 경로를 session 이 소유**(design §3.1). `New(..., nil)` 은 테스트용으로 병존.
  - `keyreg.Literal` 소스 신설 — 호출자가 이미 들고 있는 키 자료(preset 노드 신원)가 재-읽기 없이 레지스트리로 들어오는 경로.
  - `keyreg.EnsureOpts.ExpectAddress` 신설 — 등록 시 **개인키에서 주소를 재유도해 선언값과 대조**. C2 불변식("genesis 등록신원 = 실제 키 일치")을 등록 시점에 강제한다. 불일치 시 키는 저장되지 않는다(디스크·메모리 모두).
  - `engine.KeySource` boundary — `PresetKeySource`(기존 세트 사용) / `GeneratedKeySource`(`keygen.GeneratePreset` 로 신규 생성, idempotent 재사용). `Dir()` 이 구성 시점에 알려지므로 launcher·genesis source 배선은 그대로.
  - `engine.RegisterIdentities` — 세션 레지스트리에 `node1..nodeN` 등록. `sess.Keys()` 의 **첫 실소비자**.
  - `Deps.NewSession` 이 ctx 를 받는다 — 키 생성이 외부 `bootnode` 프로세스를 부르므로 취소·timeout 이 전파돼야 한다.
  - CLI: `run --keys-source preset|generate --bootnode <path>`.
  - 검증: keyreg 단위 3건(Literal·ExpectAddress 4케이스·거부 시 미저장) · session 2건 · engine 6건 · **e2e 2건**(바이너리 없이 `Run` → `<session>/keys/node<i>/address` 생성 확인, 키셋보다 큰 토폴로지 거부) · CLI 6케이스. `go test -race ./...` green.
  - ~~한계: `keygen` 이 extraData 를 0-placeholder 로 씀~~ → **T7.2 에서 해소**: `keygen.WBFTExtraData` 가 생성 검증자셋에서 extra-data RLP 를 계산한다.

### 잔여 (우선순위 순)

| # | 작업 | 왜 이 순서인가 | 상태 |
|---|---|---|---|
| **T7.2** | **wbft extraData RLP 산출** — `keygen.WBFTExtraData`: WBFTExtra(10필드) RLP 를 자체 최소 인코더로 계산(geth 의존성 없음). 게이트: 배포된 preset 의 extra-data 를 자기 메타데이터에서 바이트 동일 재현. GasTip=InitialGasTip·Diligence=DefaultDiligence 는 체인 소스 대조 확인 | T7.1 이 연 "랜덤 키셋" 경로의 유일한 잔여 블로커였음 | ☑ |
| **T7.3** | **`internal/core/launchopt`** — Dialect 2장(geth114 / geth110-wemix) + 관심사 모듈 10 + Builder(cross-module 검증) | 배경 2·알고리즘 7 미충족. 현재 launch args 가 5곳 분산 | ☑ |
| **T7.4** | **launchopt 전환** — `armSpecs`(engine)·`upgrade.LaunchArgs`·`ExtraArgs` 클로저 2곳(chainsetup·cmd)을 Builder 로 흡수 + CLI `--chain-id/--network-id/--launch-opt` 커스텀 심. 게이트는 flag-pair 동등성(레거시 argv 가 관심사를 2회 교차 배치해 바이트 동일은 구조적으로 불가 — `architecture/code-graph.md` §4). 잔여: legacy stack A 의 `nodeconfig.LaunchArgs` 호출 2곳은 T7.11 에서 스택과 함께 이관 | 5곳 → 1곳(레거시 스택 제외) | ☑ |
| **T7.5** | **`internal/app` 유스케이스 층** — 유스케이스 1개=함수 1개, cobra·MCP 타입 무지. ChainNew/ChainStatus + net 스텝 전체가 이 층 경유 | fan-out 축소는 소비자 이관에 비례(레거시 소비자는 T7.11 잔여) | ☑ |
| **T7.6** | **`net` 원자 스텝** — keys/allocate/genesis/config/launchopts/provision/init/start/stop/restart/rm/logs/health. 각 스텝 = app 함수 1 + CLI 서브커맨드 1 + MCP 도구 1. keys 는 engine.KeySource, argv 는 engine.NodeLaunchArgs(단일 조립 지점) 재사용 | 로컬 타깃 완성; 원격 rm/logs 는 명시적 미지원 오류 | ☑ |
| **T7.7** | **`netcompose.Workspace` → `core/session` 흡수** — `session.Composition`(장수명 환경 모드)이 디렉토리·manifest·스텝 스탬프를 소유, Workspace 는 도메인 상태만 | 잔여 저장소 `core/state` 는 T7.11 에서 스택과 함께 | ☑ |
| **T7.8** | **DSL v2** — env/case 분리, do/expect 통일 문장형(v1 은 같은 시퀀스로 desugar — 실행 경로 1개), strict 파싱, keys/launch/genesis(set·overlay) 선언, schema/v2.schema.json 정본, `migrate-spec`(라운드트립 게이트), hooks.onFail | 미배선 선언은 이름 붙여 거부: genesis existing/build/inherit·role-scoped launch·override hook(G5) | ☑ |
| **T7.9** | **metric 검증원** — portplan 이 metrics 포트(HTTP+3, rpcStep≥4) 할당, collector.ScrapeMetrics(Prometheus 텍스트), `expect:"metric"` 어세션(기본 GreaterOrEqual). metrics 포트 없는 노드는 명시적 실패 | 3-검증원(log·rpc·metric) 완성 | ☑ |
| **T7.10** | **단일 경로 문법** — `netcompose.ParseTarget`: `/local/path` · `user@host:/path` · `ssh://user@host:port/path`. `net new --target` + MCP `target` 인자; 레거시 4-플래그는 유지하되 혼용 거부 | setup 명령의 4-플래그는 T7.11 에서 스택과 함께 | ☑ |
| **T7.11** | **레거시 스택 A 제거** — 진행: `core/probe`→`core/collector`(Detect) · `Plan`→`core/driver` · `pipeline/verify`→**`core/health`**(app.VerifyNetwork 경유) · `pipeline/attach`→**`core/node.AttachedSet`** 흡수 완료. pipeline 3/5 소멸(verify·attach 제거, Plan 이전). **표면 이관 완료**(§1d) · **패키지 이동 완료**(§1e: `pipeline/setup`→`core/bringup`) · **netcompose 대체 진행 중**(§1f: b-1~b-4 완료, b-5·b-6 은 라이브 검증 선행). **잔여**: `pipeline/testrun`+`testkit`(cmd test + mcp, **케이스 이관 91건 선행**) | 표면은 app 1곳으로 수렴 — 남은 건 라이브 검증 후 전환·삭제, 그리고 케이스 이관(작업량) | ☑ **완료 확인 2026-09-07.** 대상 셋이 전부 없다 — `core/probe`·`core/driver`·`internal/pipeline` 어느 것도 트리에 없고, 흡수처(`core/collector`·`core/health`·`core/node`)만 남았다. 게이트로 재고 닫는다 |
| **T7.12** | **`overlays/account-extra.json` params 형식 교정** — `internal/chains/stablenet/overlays/account-extra.json` 의 `govCouncil.params.authorizedAddresses`·`blacklistedAddresses` 가 JSON 배열인데, genesis 의 `SystemContract.params` 는 `map[string]string` 이라 `gstable init` 이 `cannot unmarshal array ... of type string` 으로 거부한다. 콤마로 이어붙인 문자열로 고쳐야 `setup --genesis-overlay` 기동이 성공한다. 다른 소비자(레거시 testkit setup 경로)도 이 오버레이를 쓰는지 확인 후 일괄 교정. **게이트**: 이 오버레이로 `tests/repro/stablenet-account-extra.sh` 가 스킵 없이 통과 | 라이브 검증 때 스크래치 사본으로만 우회했고 원본은 그대로다 — 오버레이 경로가 실제로는 깨져 있다 (`remaining-work` (2026-09-24 삭제) §1.1) | ☑ **완료 확인 2026-09-07.** 오버레이에 `params` 키가 아예 없고, **왜 넣으면 안 되는지**가 파일 주석에 남아 있다(base 템플릿의 `govCouncil.params` 는 평평한 string-map 이라 배열을 넣으면 `gstable init` 이 거부한다). 이 오버레이를 쓰는 스펙 4건 전부 `validate` 통과 |
| — | **T5.2 업그레이드 멀티바이너리** · **T5.5 wemix4 이관** · **실 SSH 라이브 e2e** | §2 기존 항목, 환경 의존 | ☑ 셋 다 닫혔다 (T5.2 §2 에 ☑ · T5.5 2026-09-12 확인 · SSH e2e 2026-09-11 docker 15대) |

---

## 1d. T7.11a — 레거시 스택 A 표면 이관 (2026-08-18 완료)

> 목표: 레거시 패키지를 **삭제하는 것**이 아니라, 삭제를 기계적으로 만드는 것.
> 착수 전 실측: `pipeline/setup` 소비자 7파일 · `core/state` 소비자 9파일이 각자
> `state.Load → setup.X → state.Save` 를 반복하고 있었다. 이 중복이 삭제 불가의 실제 원인이었다.

| # | 작업 | 결과 |
|---|---|---|
| **T7.11a-1** | 네트워크 수명주기 유스케이스 | `driver.StopNode/RelaunchNode/StopNodeSet` 흡수(`Plan`→driver 와 동일 근거) · `app.NetworkStatus/NetworkStop/NodeStop/NodeStart/NetworkRemove` + `app.GCSessions` 신설 · `Deps.Driver` boundary(프로세스 없이 테스트, 원격 라우팅) · cmd status/stop/node/clean + stop MCP 도구 이관 |
| **T7.11a-2** | 네트워크 기동 유스케이스 | `app.NetworkPlan/NetworkProvision/NetworkLaunch`(단일 `NetworkSpecIn`) + `app.ResolveChain` 신설 · cmd setup 258→175줄(플래그 바인딩·렌더링만) · start MCP 도구가 같은 함수 경유 |
| **T7.11b** | 체인 업그레이드 | `app.HardforkPlan/HardforkExecute` 신설(계획/실행 분리) · cmd hardfork 이관 |
| **T7.11c** | MCP 잔여 표면 | mcp status·setup_plan·resolveNodeSet + cmd test 의 노드셋 해석을 app 경유로 |

**소비자 수렴**: `pipeline/setup` 7파일 → **app + 라이브테스트 1건**, `core/state` 9파일 → **app 전용**.
`cmd/chainbench` 는 두 레거시 패키지를 더 이상 import 하지 않는다.

**이관이 드러낸 결함 2건**(둘 다 테스트가 없어 보이지 않던 것):
- start MCP 도구가 `nodespecs.json` 을 저장하지 않아, MCP 로 띄운 네트워크에서는 `node start` 가 동작하지 않았다 → `NetworkLaunch` 경유로 해소.
- `--endpoints 0` 이 "미지정"과 구분되지 않았다(둘 다 0) → `*int` 로 교정하고 테스트 추가.

**동작 변경 1건(의도)**: 토폴로지 파일의 `chain` 이 `--chain` 을 조용히 덮어쓰던 것을,
사용자가 명시한 경우에는 덮어쓰지 않도록 바꿨다(`ChainExplicit`).

**검증**: `go build/vet ./...` · `go test ./...` · 영향 패키지 `-race` 전부 통과. 신규 단위 테스트 27건.

---

## 1e. (a) 패키지 이동 — 완료

`core/pipeline/setup` → **`core/bringup`**(패키지명 `setup`→`bringup`), 죽은 `Plan = driver.Plan` 별칭 제거,
에러 접두사 정리, `core/state` package doc 에 단일 소유자·수명 명시. **순수 이동 — 동작·시그니처 변경 없음.**
`core/pipeline/` 에는 `testrun` 만 남았고, 그것은 레거시 스택 B 와 함께 사라진다.

이동한 이유: 3-phase pipeline 프레이밍은 이미 소멸했는데(verify→`core/health`, attach→`node.AttachedSet`,
`Plan`→`core/driver`) 이름만 남아 존재하지 않는 phase 를 암시했고, `cmd setup`·`internal/chainsetup` 과 3중 충돌했다.

---

## 1f. (b) netcompose 가 레거시 setup 스택을 대체 — 진행 중

착수 전 실측한 두 스택의 격차. netcompose 는 launch argv(`engine.NodeLaunchArgs`)·포트 할당(`place`)·
키 소스(`engine.KeySource`)를 이미 엔진과 공유하고 있었으나, **네트워크를 기술하는 방법**에 구멍이 있었다.

| # | 작업 | 내용 | 상태 |
|---|---|---|---|
| **b-1** | 구성 패리티(결함) | **syncMode 미렌더 수정** — `Config` 이 `nodeconfig.Params.SyncMode` 를 채우지 않아 모든 노드가 `full` 이었다(steps 로 구성한 snap-sync 테스트가 조용히 full sync 를 돌고 있었음). `--endpoint-syncmode` 신설, validator 는 항상 full. + genesis `--set` 오버라이드·`--overlay` 딥머지(양쪽 fork 순서 재검증) + capabilities 파생(manifest+ws+`delayed-<fork>`+overlay) | ☑ |
| **b-2** | 토폴로지·외부 매니페스트 | `allocate --topology`(노드별 role/sync_mode/bootnode; validator 수는 **요청값이 아니라 해석된 배치**에서 셈 — genesis 가 이 값으로 검증자셋을 만든다) · `new --manifest/--genesis-template`(워크스페이스에 기록 → 이후 모든 스텝이 같은 플러그인 해석) · 체인 해석을 `chains/external.ResolveChain` 1곳으로 | ☑ |
| **b-3** | NodeSet 브릿지 | `Workspace.NodeSet()`/`RPCHost()` + `app.NetworkStatus`/`NetworkStop` 이 **디렉토리의 상태 매니페스트로 스택을 판별**해 양쪽을 읽는다 → `status`/`stop`·MCP 도구가 워크스페이스에서 그대로 동작. 부수 수정: health 스텝이 target 무관하게 `127.0.0.1` 을 찌르던 원격 버그 | ☑ |
| **b-4** | `net up` 매크로 | 9개 스텝을 순서대로 실행하는 유스케이스 1개 + CLI. `--stage provision\|start`. 실패 시에도 성공한 스텝을 출력(워크스페이스는 그 지점부터 손으로 재개 가능). **`--stage=provision` 로 end-to-end 실증**(genesis·config 3개·argv·노드 테이블) | ☑ |
| **b-5** | 과거 `setup` → `net up` 전제 대조 | 현행 root에 두 명령과 별칭이 없고 `chain up`이 구성 표면이다. 아래 2026-08 기록은 당시 전제이며 현행 두 경로의 중복을 뜻하지 않는다. 새 명령이나 스택을 복원하지 않는다 | ☑ **현행 경로 확인 2026-10-14** — 실제 세 체인 공개 CLI 회귀까지 완료; §0a 12번과 [증거](setup-transition-evidence.json) |
| **b-6** | `core/bringup`·`core/state` 삭제 | ☑ **완료** — #241 병합 시 `engine.LocalSetup`·`session.SaveLocalNodeSet` 로 수렴하며 두 패키지가 소멸했다(§1f-x) | ☑ |

### 2026-10-14 현행 경로 재평가

b-5가 전제한 root의 setup/net 명령은 별칭까지 없다. 실제 Cobra 트리의 chain up과
workspace-dir·stage·blueprint·chain-preset·server-set 입력을 확인하고, 그 사실과 어긋난 표를 RED→GREEN으로 정정했다.
CLI는 app.ChainUp→verb.ChainUp을 사용하고, MCP는 feature의 chain.up을 통해 같은 경로를 사용한다.
Web 구성과 매니페스트 셋업은 chain up 하위 프로세스를 호출하며, DSL 구성은
composeWorkspace→ChainUpComparing을 통해 같은 구성 코어를 사용한다.
app/net.go라는 파일명이나 과거 주석은 별도의 폐기된 실행 스택이 있다는 근거가 아니다.

전체 make check와 cmd·chaincmd·chainsetup/verb의 관련 전체 race를 통과했다.
외부 HEAD 출처·바이너리 해시를 확인한 stablenet/wbft/wemix에서 공개 CLI를 직렬 race로 실행했다.
각각 생성한 키로 네 노드를 구성하고 9개 이상 단계 출력, 블록 전진·peer 연결과 stop/rm 성공을 확인했다.
세 체인 PASS·Go skip 0, 소스 해시는 실행 전후 동일했다. 종료 후 프로세스·노드 DB·기록의 노드는 0건이고,
설정·로그·실행 기록 126파일은 해시·권한·소유자·mtime을 보존 목록으로 대조했다.
chain rm은 데이터 평면만 제거하므로 작업공간의 제어 기록은 남는다.

이것으로 b-5의 현행 경로 재평가는 완료다. 이후의 2026-08 절은 당시 분석과 절차를 남긴 역사 기록이다.
Linux fleet·전체 mainnet fidelity·wemix4 충분성·W3/W4·최종 Web 인수는 이 결과의 범위에 포함하지 않는다.
근거: [RED→GREEN·현행 소비 경로·세 체인 실제 CLI](setup-transition-evidence.json).

### 1f-x. 병합에서 달라진 것 (2026-08-21)

이 절이 계획한 것과 **실제로 병합된 것이 다르다.** 브랜치가 열려 있는 동안 main 에
#239·#240 이 들어와 `core/pipeline/setup` 과 `core/state` 를 먼저 지웠고, 그 자리를
`engine.LocalSetup`·`engine.BuildLocalPlan`·`session.SaveLocalNodeSet` 이 채웠다.

이 브랜치는 같은 결론에 다른 경로로 도달해 있었으므로(§1e 는 `pipeline/setup` 을
`core/bringup` 으로 **옮겼을 뿐**이고, 실측 파일 유사도 80~99%), 병합 시 셋 중 둘은
main 쪽을 채택했다.

| 쟁점 | 채택 | 근거 |
|---|---|---|
| nodeset 영속 | `session.SaveLocalNodeSet` (main) | [[layers]] 가 `core/state` 를 ❌ 레거시로, `core/session` 을 ✅ 소유자로 판정 |
| plan/provision/launch | `engine.LocalSetup` (main) | `core/bringup` 이 genesis 를 `os.WriteFile` 로 직접 써 파일 통로를 우회 |
| `StopNode`·`RelaunchNode`·`StopNodeSet` | **`core/driver` (이 브랜치)** | `driver.Driver`·`node.NodeSet` 만 쓴다. `engine` 에 두면 노드 하나 멈추는 데 내부 24패키지 import(vs 2) |

조용히 사라질 뻔한 것 하나를 옮겼다: `bringup.Run` 은 setup 진행 이벤트를 발행했으나
`engine.LocalSetup` 은 하지 않았다. 끝날 때까지 아무것도 보고하지 않는 기동은 멈춘 것과
구분되지 않으므로 `LocalSetup` 에 `Bus` 를 더했다.

### 라이브 검증 결과 (2026-08-18, 실제 바이너리)

| 체인 | `net up` | 블록 전진 | `run` api 9건 | 고아 |
|---|---|---|---|---|
| stablenet | 성공 | 97 → 110 → 122 | 9/9 | 0 |
| wbft | 성공 | 24 → 36 → 48 | 9/9 | 0 |
| **wemix** | **실패 — 블록 0** | 정지 | — | 0 |

wemix 실패의 원인은 `net up` 이 아니라 **`poa.BuildGenesis` 가 템플릿 치환을 하기 때문**이다 —
`alloc:{}`·`minerNodeId:"0x0"` 인 죽은 genesis 가 나오고 노드가 ethash 로 돈다. 같은 4노드를
**실제 절차대로 손으로** 띄우면 정상이다(거버넌스 컨트랙트 5개 · etcd 4멤버 · 블록 238→268 ·
4노드 전부 sealing, 20블록에서 5/6/5/4). 즉 **F4(`GenesisArtifacts`)·F5(poa 액션 배선)가 이 결함의
수정분**이고, b-5 의 wemix 몫은 그것을 기다린다.

wbft 계열은 게이트를 통과했으므로 b-5 를 stablenet/wbft 한정으로 먼저 끊을 수 있다.

### b-5 를 지금 하지 않은 이유

`--stage=start` 는 실제 프로세스를 띄우므로 **체인 바이너리 없이는 검증할 수 없다.** 그리고 전환은
관측 가능한 변화를 동반한다 — **포트 대역이 달라진다**(bringup: `ports.base_*` = http 8501·p2p 30301 /
netcompose: `place` 할당 = http 8600·p2p 31000, step 10). 기존 절차 문서·라이브 스크립트가 이 번호에
의존한다면 함께 갱신해야 한다.

**검증 절차(바이너리 보유 환경)**:
```sh
chainbench net up --data-dir /tmp/n1 --chain stablenet --binary $GSTABLE_BIN \
  --keys presets/keys --validators 4
chainbench net health --data-dir /tmp/n1     # 블록 전진 확인
chainbench run --data-dir /tmp/n1 tests/specs/api/*.json
chainbench net stop --data-dir /tmp/n1       # 고아 0 확인
```
이것이 통과하면 b-5(내부 교체)는 작은 변경이고, b-6(삭제)이 뒤따른다.

---

> **여전히 미배선인 선언 1건**: `testspec.Deps.Keys` 는 타입으로만 존재하고 소비자가 없다.
> 현재 `sendTx` 는 노드측 unlocked 계정으로 서명하므로(`eth_sendTransaction`) 로컬 서명키를 쓰지 않는다.
> **소비자가 생길 때(로컬 서명 tx 액션) 배선한다** — 소비자 없이 값을 넣으면 T3.2b 가 고친 "선언만 하고 방출 안 함"을 되풀이하게 된다.

---

## 1g. 아키텍처 정합 · 패밀리 기동 (사전 작업 완료 · K0 부터 착수)

> 설계 근거: [[layers]](architecture/layers.md) · [[module-responsibilities]](architecture/module-responsibilities.md) ·
> [[family-bringup-design]](family-bringup-design.md).
> **작업 상태는 이 절에서만 관리한다** — 설계 문서는 *무엇을 왜*, 이 절은 *언제 어디까지*.

세 갈래가 있고 서로 독립이다. A(규칙) → B(파서) → F(패밀리 기동) 순으로 착수하되,
B 는 F 와 병행 가능하다.

### 완료 — 착수 전 사전 작업 (2026-08-18~19)

코드 갈래에 들어가기 전에 **가정을 사실로 바꾸는 일**만 먼저 했다. 아래 셋이 없으면
K0·S0 가 추측 위에 서게 된다.

| # | 작업 | 무엇이 확정됐나 | 상태 |
|---|---|---|---|
| **P-1** | BLS 파생을 순수 Go 로 실증 | `kilic/bls12-381` + stdlib `crypto/hkdf` 로 **preset node1~5 의 BLS 공개키·PoP 이 바이트 동일**하게 재현됨, `CGO_ENABLED=0` 에서. → **K1(`--bootnode` 제거)이 가능하다는 것이 증명됨.** 함정 둘: PoP 의 DST(`..._POP_`) 누락 시 형식은 멀쩡한데 검증 실패 · `blst_keygen` v4 는 salt 를 루프 **이전에** 한 번 해시 | ☑ |
| **P-2** | `netcompose/target.go` → `core/target` 이동 | L4 에 있던 타깃 해석이 L1 로 내려감(`driver`/`provision`/`remote` 만 import). S 계열의 선행 조건 | ☑ |
| **P-3** | **문서 통치 구조** — 전 문서에 등급(정본/현행 설계/이력/대체됨) 표기 + `docs/README.md` 에 권위 순서 + 대체된 2건 `dev/archive/` (2026-09-24 삭제) 이동 | 문서끼리 어긋날 때 **무엇이 이기는지**가 정해짐. 이걸 한 이유는 아래 참조 | ☑ |

**P-3 을 한 이유**는 낡은 문서가 아니라 **등급이 없는 문서**였다. 23개 `dev/` 문서 중 자기 지위를
밝힌 것이 4개뿐이어서, 정본인 [[chainbench-requirements-review]] §D-2.8 을 제안 문서로 오인해
건너뛰었고 그 결과 하드포크 분류를 반대로 잡았다(A7b 가 그 정정분이다). 문서를 지웠다면
**정답이 든 문서가 사라졌을 것**이므로, 삭제가 아니라 등급 표기로 해결했다.

부수 정리: 저장소의 키가 전부 테스트 픽스처임을 루트 `README.md`·`tests/README.md` 에 명시하고
(스캐너 검출 32건이 정상 상태임과 **진짜 유출을 가려내는 기준**을 함께 적음), 파일명 오탐용
`.precommit-allow` 를 도입했다(면제는 파일명 검사에만, 내용 스캔은 그대로).

### A — 규칙을 코드로 (설계가 썩지 않게)

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **A1** | 레이어 검사 테스트 (`internal/arch`) — [[layers]] §3 표를 **파싱**한다(코드에 복제하지 않음) | ☑ 상향 의존 **0건** · 미배치 패키지 거부 · 유령 항목 거부 · 네 실패 경로를 실제로 확인 |
| **A2** | 상태 쓰기 허용목록 테스트 (`internal/arch`) — [[layers]] §5 표가 정본 | ☑ ❌ 4곳(app·chainsetup·wemix/deploy·upgrade)은 예외로 명시 · 신규 위반 차단 · **stale 예외도 차단** · `os.Create` 를 세면서 미기재였던 `engine` 을 찾아냄 |
| **A3** | `app` 의 `topology.yaml` 쓰기를 `filestore.Store` 경유로 | ☑ **정리가 아니라 결함 수정이었다** — 원격 프로비전이 genesis·config 를 조작자 머신에 쓰고 있었다(신원만 원격으로 갔다). `Deps.Files` boundary 추가 · 드라이버가 파일을 보낼 수 있으면 그것이 기본 저장소 · 회귀 테스트 3건 |
| **A4** | `chains/wemix/deploy` 를 `FileStore` 경유로 | ☑ `pullKeystores` 가 읽기·쓰기 양쪽 모두 store 경유. 직접 파일 쓰기 0건 |
| **A4b** | `chainsetup`·`consensus/upgrade` 를 `FileStore` 경유로 | ☑ **완료 2026-08-23.** 실측은 8+2 가 아니라 **11+2 = 13곳**. `Options`/`HandoffOptions`/`upgrade.LaunchOptions` 에 `Files` boundary(nil=로컬). **직접 쓰기 0건**이 되어 [[layers]] §5 의 ❌ 가 사라졌고, **A2 가 stale 예외를 잡아** 표에서 지우게 했다(가드가 의도대로 동작). `os.MkdirAll` 대부분이 함께 사라진 게 핵심 — `Write` 가 부모를 만들므로 디렉토리를 미리 만들던 코드는 **경로를 아는 코드**였고 그게 위반의 실체였다. 라이브: `chain up --case wemix` 15/15 · 엔진 게이트 통과 | ☑ |
| **A5** | ~~`core/bringup`·`core/state`·`testkit`·`core/pipeline/testrun`~~ **전부 소멸 2026-09-06** | — | ☑ 앞 둘은 #241 에서(`engine.LocalSetup`·`session.SaveLocalNodeSet` 로 수렴). 뒤 둘도 이제 없다 — 케이스 이관(P6.4)과 R 트랙 재편이 가져갔다 |
| **A6** | ~~`netreg`·`obs` 를 `session` 으로 흡수~~ **완료 2026-09-06 확인** | 컨트롤 플레인 단일화 | ☑ R1 에서 끝났다. `internal/netreg` 도 `core/obs` 도 없고, 네트워크 레지스트리는 `core/session/netreg.go`(`SaveNetwork`·`ListNetworks`), 이벤트 버스는 `core/collector/bus.go` 다 |
| **A7b** | `hardfork`와 선언형 upgrade의 현행 관계 재평가 (2026-10-14) | 입력·효과·실제 소비 경로를 대조 | ☒ **표면을 억지로 통합하지 않는 결정 유지.** 과거 `upgrade run`과 별도 upgrade.Plan 스택은 없다. 현재는 선언형 `UpgradeV2`가 concurrent/restart를 공통으로 표현하고 chainsetup이 구성·전환을 수행한다. 아래는 현행 입력 대조이며 예전 11개/8개 추가 필드 수치는 현재 근거로 쓰지 않는다. |

| 사례 | 현행 표면과 선언 | 입력·효과 |
|---|---|---|
| 이미 구성된 망의 실행 파일 교체 | root `hardfork` → app/verb HardforkPlan·HardforkExecute → core/hardfork.SwapPlan | HardforkPlanIn 네 필드: DataDir·ToChain·ToBinary·Block. 기존 상태에서 from-chain과 노드·설정·데이터 경로를 읽어 계획하고, 같은 노드 데이터를 유지해 재실행한다. CLI의 dry-run은 별도 실행 정책이다 |
| 테스트 망 구성 중 선언한 포크 | `run`의 chainPreset/env에 `UpgradeV2` → composeWorkspace·ChainCrossFork | Fork·At·From·To·Style·Carry를 선언한다. deprecated Template은 입력하면 거절한다. concurrent는 별도 역할의 두 빌드를 함께 구성하고, restart는 같은 노드를 재실행한다. 각 선언은 체인·배치·genesis와 함께 해석한다 |
| genesis만으로 활성화하는 포크 | genesis hardforks 또는 `--set bohoBlock=10` | 실행 파일 교체 계획이 아니라 config의 활성화 블록이다. 추가 SwapPlan이나 두 번째 포크 선언으로 중복 표현하지 않는다 |

이 구분은 이름 충돌이나 삭제된 구현의 추가 필드 수에 기대지 않는다.
이미 존재하는 망의 명령과 망 구성 시의 선언은 상태 전제와 부수 효과가 다르며,
concurrent/restart는 현행 선언에서 이미 공통화되어 있다. 새 범용 선언이나 폐기된 명령을 도입하지 않는다.
실제 root 등록·입력과 문서의 불일치를 RED→GREEN으로 검사하고, 관련 코어·선언·전환 회귀를 확인한다.

| **A8** | **외부 소비자 0 인 공개 심볼 정리** | 원인별로 갈라 세고, 진짜 도달 불가만 없앤다 | ☑ **완료 2026-09-07 — 세 번 보고한 숫자가 모두 틀린 것을 말하고 있었다.** 230 → 896 → 928 은 "이만큼 죽었다"로 읽혔지만, 실제로는 "다른 패키지가 셀렉터로 이름을 대지 않는다"였고 거기엔 원인이 다섯 있다. ① **메서드**(928 중 424): 수신자가 값이라 이 방식으로는 **아예 측정 불가**다. ② **공개 시그니처에 든 타입**: 호출자가 값만 받고 타입 이름을 안 적는다(`app` 의 반환 타입들). ③ **형제가 쓰이는 상수 세트의 한 멤버**: govbind 의 제안 상태 일곱은 컨트랙트 getter 가 돌려주는 값의 대응표라, 아직 아무도 안 쓴 것을 지우면 표가 반쪽이 된다. ④ **자기 패키지만 쓰는 것**(277). 여기서 제안된 처방인 "대부분 unexport"는 **틀렸다** — `nodeconfig` 의 `Key*` 111개는 knob 의 공개 어휘이고 일부는 밖에서도 쓴다. ⑤ **정말 아무도 안 부르는 것: 1,226개 중 3개**. 지웠다: 컨텍스트 없는 `store.Import` 래퍼(A7 의 겹침도 함께 사라졌다) · 구현체도 소비자도 없이 `Driver` 를 다시 설명하던 `process.Transport` · 표면이 자기 목록을 안 들게 하려고 만들었는데 그런 표면이 없던 `remote.EnvNames`. 계산은 `internal/arch/reach.go` 한 곳에 있고 도구(`code-graph -dead`)와 테스트가 그것을 함께 쓴다(#349 의 교훈). 테스트가 지키는 것은 개수가 아니라 **불변식**이다 — 공개인데 아무도 이름을 대지 않는 것이 0. 개수에 예산을 걸면 knob 하나 추가할 때마다 실패하고, 옳은 일을 벌하는 래칫은 사람들이 끄는 법을 배운다. **변이가 내 결함을 잡았다**: 첫 판은 자기 선언이 자기 이름을 한 번 쓴다는 것을 안 빼서 `nowhere` 가 구조적으로 0 이었고, 안 부르는 함수를 넣어도 통과했다 |
| **A7** | **이름 겹침 검출 테스트** — 공개 패키지 레벨 선언이 두 패키지 이상에 같은 이름으로 있으면 보고한다 | ☑ **완료 2026-09-07.** `internal/arch/naming.go` 가 세고 `naming_test.go` 가 래칫을 건다. **메서드는 세지 않는다** — 수신자가 이름공간이라 `Stop`·`Save` 같은 것 577개가 목록을 채우고 `RoleValidator` 를 덮었다. 규칙으로 설명되는 것 101개(app 파사드는 **참조까지 확인한다**. 그래야 뜻이 다른 `app.Network` 대 `keyring.Network` 를 파사드로 오인하지 않는다), 관용 19개, 빚 33개이며 관용과 빚은 둘 다 줄기만 하고 유령 항목도 거부한다. 변이 둘(새 겹침 추가 · 참조 검사 제거)로 확인했다. **첫 수확이 `RoleValidator`×3** 이라 그 자리에서 지웠다: `dsl.RoleValidator`→`BinaryAfter`(포크 뒤 바이너리), `validatorset.RoleValidator`→`AccountValidator`(계정의 기능). 08-25 이 지목했던 `Runner`×3 과 `Ports` 는 빚 목록에 이유와 함께 남았다 | ☑ |

**이름은 각 항목이 자기 범위에서 함께 고친다**([[layers]] §5b). 개명만 하는 커밋은 리뷰가 어렵고
동작 변경과 섞이면 더 어렵다. 규칙: **한 개념 = 한 이름, 다른 개념 = 다른 이름, 식별자는 명명된 타입.**

**A1·A2 를 먼저 하는 이유**: 이후 모든 작업(B·F 포함)이 규칙 위반을 자동으로 잡힌다.
설계를 지키는 일을 사람의 주의력에 맡기지 않는다.

### B — DSL 파서를 모듈로

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **B1** | **`testspec` 분할** — 문법과 런타임을 가른다 | `validate` 는 다이얼도 쓰기도 하지 않는다 · 파서 fuzz | ☑ **완료 2026-09-08 — 게이트를 다시 썼다.** 분할은 끝났고(#311·#326), fuzz 도 끝났다(#357, 550만 실행에 v1 문법 결함 셋 발견).\n\n**옛 게이트 "`validate` 가 rpc/session 을 링크하지 않는다"는 통과할 수 없고, 이제 잴 값어치도 없다.** U 트랙이 모든 표면을 `app` 으로 모았고 `app` 의 fanOut 이 22 라, 한 바이너리에서 어느 명령을 켜도 전부 링크된다. 바이너리를 쪼개면 문자는 만족하지만 그 통합을 되돌린다.\n\n**실측이 진단도 뒤집었다**: `validate.go` 는 rpc·session 을 **한 번도 참조하지 않는다**(쓰는 것은 dsl·dsl/interp·testhelper·core/node·core/registry). 링크는 둘에서 온다 — ① run 경로와 같은 패키지에 산다, ② `validate` 가 이름 해석에 부르는 `testhelper.Registry()` 가 **이름 목록이 아니라 액션 구현 자체**라 rpc·session·keyring·accounts 를 끌고 온다. 그래서 파일을 별도 패키지로 옮겨도 ②가 남는다.\n\n**게이트는 행동으로 바꿨다** — 지킬 것은 링크 크기가 아니라 "아무것도 없을 때 돌리는 명령"이라는 계약이다. `TestValidate_TouchesNothing` 은 커밋된 스펙 122개를 빈 작업 디렉터리에서 검증하고 **아무 파일도 안 생기는지**와 25ms 안에 끝나는지를 본다(다이얼했다면 시한까지 기다렸을 것이다). `TestValidate_ReachesForNothingLive` 는 `validate.go` 가 rpc·session·process·filestore·remote·chainsetup 을 import 하지 않는지 본다 — 라이브 검사가 들어오면 여기서 먼저 걸린다. 변이 둘로 확인.\n\n**어휘에서 이름을 떼어내는 안(②의 근본 해결)은 하지 않는다.** 액션 45개의 등록 방식을 바꾸는 일인데 얻는 것이 오프라인 검증의 링크 크기뿐이다. 문법만 쓰는 다른 소비자가 생기면 그때 근거가 선다 |
| **B2** | `Spec.Fingerprint()` 가 `string` 반환 (현재 `session.Fingerprint`) | **구문이 L3 에 묶인 유일한 이유**가 이 타입 하나다 — B1 의 선행 조건 | ☑ **완료 2026-09-07 (#359) — 다만 이 행의 전제가 틀렸다.** "구문이 L3 에 묶인 유일한 이유가 이 타입 하나"가 아니었다: `dsl` 자체는 fanOut 0 이고, 묶인 것은 `dsl/interp` 이며 그것이 `session` 을 쓰는 곳은 세 파일이다(`fingerprint.go` 1개 · `interpreter.go` 4개 · `run.go` 9개). `Fingerprint` 를 `string` 으로 되돌려도 첫 파일 하나만 풀린다. 실제로 한 일은 **요구를 좁힌 것**이다 — interp 는 `Environment` 11개 중 4개, `TestRecord` 12개 중 4개만 쓰므로 `NodeTable`·`Recorder` 로 선언했다. 완전 분리(결과 타입 이관)는 서로 맞아야 하는 구조체를 둘 만들어 아티팩트에서 증거가 조용히 빠지는 길이라 **하지 않기로 했고 근거를 남겼다** |

현재 `testspec` 이 `collector`·`session`·`rpc`·`keyreg`·`accounts` 를 import 해서
**순수해야 할 파서가 L3 로 끌려 올라가 있다**([[module-responsibilities]] §3).

**이름 주의**: 인터프리터를 `dsl/engine` 으로 두면 `internal/engine`(테스트벤치 엔진)과 겹친다.
**`engine` 은 하나뿐이어야 한다** — 주도하는 쪽이다. 인터프리터는 `dsl/interp`.
이 구조(엔진이 파싱→환경준비→인터프리터 순차실행→기록을 주도)는 **이미 코드에 있다**
(`engine.Run` + `engine/wire.go`의 `NewRunSpec`).

**레지스트리는 import 가 아니라 주입이다.** 엔진(L3)이 기능 레지스트리(L5)를 import 하면 상향이지만,
`interpreter.go` 가 이미 `Registry`·`Action`·`Assertion`·`Deps` 를 **스스로 정의**하고 L5 가 구현체를
넘긴다 — 값 전달이라 층을 거스르지 않는다. 코드 변경이 아니라 문서 문구 문제였다.

### F — 패밀리별 기동 (wemix 를 Go 로)

| # | 작업 | 게이트 | 리스크 | 상태 |
|---|---|---|---|---|
| **F1** | `PortReservation` — 패밀리별 포트 대역. `serverset` 전역 `p2pStep>=2` 제거 | ☑ **완료 2026-08-22.** poa {3,3}·wbft {2,3} · poa 는 step 2 를 거부 · wemix 노드가 **etcd client(p2p+2)까지 예약**(실측: workspace `etcdClient: 31002`) · stablenet 포트 불변 | 낮음 | ☑ |
| **F2** | `--networkid` 방출 | ☑ **완료 2026-08-22.** 다이얼렉트가 아니라 **매니페스트 사실**이라 모든 체인에 방출(argv 실측 `--networkid 8283`) · `--network-id` 오버라이드는 상위 레이어라 그대로 이김 · stablenet 4노드 라이브 블록 전진 | 낮음 | ☑ |
| **F3** | `BringUpPhases` + `Deps.Launch(nodes)` + `Deps.Action` | ☑ **완료 2026-08-22.** wbft 는 1페이즈(노드 목록 없음 = 전체) — stablenet 4노드 라이브 불변 · poa 는 boot(deploy-governance·etcd-init·verify-etcd) → rest · **미배선 액션은 오류**(LeaderGate·SwapBinary 와 동일 계약) · 액션 실패 시 다음 페이즈 미기동 | **중** | ☑ |
| **F4** | `GenesisArtifacts` + `WemixGenesisSource`(`poa.PrepareTemplate`→`GenerateGenesis`) | ☑ **완료 2026-08-22.** wbft `Extra` nil · poa config `Validate()` 통과 · `poa.Family.BuildGenesis` 는 이제 **거부**(치환 템플릿을 genesis 로 돌려주던 자리) · **라이브(실 gwemix)**: alloc 4계정·extraData 실값·chainId 8285(매니페스트)·`gwemix init` 수용 | 중 | ☑ |
| **F5** | poa 액션 배선 + 유스케이스 수렴(3곳→1곳) | ☑ **라이브 통과 2026-08-22** — wemix 4노드가 **일반 엔진 경로**로 뜬다: 바이너리 생성 genesis → boot 페이즈(프로듀서 단독) → deploy-governance(컨트랙트 5종) → etcd-init → **verify-etcd 가 클러스터 확인** → rest 페이즈 → 블록 전진. `TestWemix_Live_BringUp`(GWEMIX_BIN 게이트). **스텝 경로도 페이즈를 태운다**(2026-08-23): `net up`/`net start` 가 패밀리 페이즈 순서로 기동하고 사이에 부트스트랩 액션을 실행한다 — wemix 가 엔진·스텝 양쪽에서 뜬다. **F5b 완료 — 다중 프로듀서 sealing 로테이션**(2026-08-23): `net up --validators 4` 46초, 클러스터가 넷을 다 담고(`node1..node4`) **최근 25블록 봉인자 4명(8/6/6/5)**. 조인은 rest 페이즈의 `etcd-join` 액션이 수행한다.

가설(순서)은 **틀렸다**. go-wemix 소스가 말하는 실제 계약은 둘이다. ① **`admin.etcdJoin(name)` 의 인자는 조인할 노드가 아니라 물어볼 상대**다 — 조이너가 eth 와이어(`EtcdAddMemberMsg 0x14`)로 피어에게 요청하면 피어가 클러스터 문자열로 답하고 조이너가 그걸로 자기 서버를 띄운다(`wemix/etcdutil.go:1289`, `eth/protocols/eth/wemix_handlers.go:77`). 자기 이름을 넘기면 **에러 없이 아무 일도 안 일어난다** — 앞선 세션이 여기서 막혔다. ② 조이너는 거버넌스 멤버 목록을 **체인에서 읽어** 알게 되므로, 방금 뜬 노드는 아무도 몰라 `not found` 로 거절한다. 그래서 `admin.wemixInfo.nodes` 에 상대가 보일 때까지 기다린 뒤 조인한다. 또 조인이 **null 을 반환하고도 클러스터에 안 들어가는 경우**(4대 중 1대)가 실측돼, 반환값이 아니라 **클러스터를 증거로** 재시도한다. | 중 | ☑ |
| **F6** | `chainsetup/wemix.go` 의 `NotImplemented` 제거 | ☑ **라이브 통과 2026-08-23** — `chain up --case wemix --validators 4` 가 **15스텝 전부 OK**(deploy-governance 12.3s · etcd-join 37.6s · head 22), 25블록 봉인자 4명. wemix 케이스는 `Supported`. 절차 자체도 정정했다: **선언된 순서가 틀려 있었다**(genesis 가 allocate 앞 — 거버넌스 멤버는 배치에서 나오는 ip/port 를 담는다), 그리고 `launch-rest`·`etcd-join` 이 아예 빠져 있었다. 러너는 절차 사본을 갖지 않고 netmap·WemixGenesisSource·패밀리 페이즈·WemixBootstrap 을 그대로 조립한다. 스텝 id 와 패밀리 액션 이름이 어긋나면 테스트가 막는다. | 낮음 | ☑ |

**F1 은 버그 수정이다** — 현재 `p2pStep>=2` 는 wemix 에서 틀렸다(etcd 가 p2p+1 peer·p2p+2 client 둘을 쓴다).

**F3 이 유일한 실질 리스크다.** `Deps.Launch` 시그니처 변경이 engine·netcompose·chainsetup 에
파급된다. `nodes=nil → 전체` 규약이면 이관은 기계적이고, wbft 계열은 argv·순서가 바이트 동일하게
유지되므로 stablenet/wbft 회귀를 §1f b-5 절차로 즉시 확인할 수 있다.

### K — keyring (첫 착수 대상)

> 근거: [[keyring-design]](keyring-design.md).
> **키는 세 체인이 동일하다**(실증: 같은 nodekey → 세 체인 동일한 address·pubkey, BLS 도 Go 파생이
> `bootnode` 출력과 바이트 동일). 그래서 여기부터 정리하면 위쪽이 단순해진다.
> 실측 문제: 키 관심사가 **5패키지 1,236줄**에 흩어져 있고(읽기 `keys`/쓰기 `keygen`/저장 `keymat`/
> 런타임 `keyreg`), preset 이 **신원·네트워크 결정·파생 산출물 셋을 섞어** 담아 preset 을 전제로 만든다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **K0** | `core/keyring` 신설 — nodekey 생성 · 신원 파생(주소·devp2p 공개키·BLS·PoP, 전부 in-process) | 배포 preset 의 node1..5 를 nodekey 만으로 **바이트 동일 재현**(골든) · `CGO_ENABLED=0` · fuzz | ☑ |
| **K1** | `--bootnode` 제거 — BLS 를 자체 파생 | **`PATH` 를 비운 채 4노드 키셋 생성 성공** · 세 계층(keygen·engine·CLI)에 각각 게이트 테스트 | ☑ |
| **K2** | `keygen.WBFTExtraData` → `consensus/wbft.ExtraData` | 골든 유지 · **`BuildGenesis` 가 비어 있으면 파생** · `Take` 의 stale extraData 결함 수정 | ☑ |
| **K3** | `core/keys`·`keygen`·`keymat`·`core/keyreg` 흡수 | **4패키지 → 1**(1,565줄 → `core/keyring` 1,497줄) · **신원 타입 5 → 1** · `Nodekey` → `PrivateKey`(역할이 아니라 실체로 명명) | ☑ |
| **K4** | `keyring` 명령 — new/add/list/show/import/export | `--keyring` 출처 표기(플래그/env/기본) · `--with-bls` 선택 · `export` 는 `--yes` 필수 · `list --verify` · `add` 는 검증자 승격 안 함 · 기존 3개 그룹 유지(deprecated) | ☑ |
| **K5** | preset 분해 — 신원과 네트워크 결정을 타입으로 분리 | `Preset{Nodes, Network}` · `NetworkFor(n)` 이 선언 유무를 흡수 · **`keyring new --validators 0` = 신원만** · **라이브: 신원만 있는 링으로 stablenet 4노드 블록 생성 + api 9/9** · 기존 preset 읽기 호환 | ☑ |
| **K6** | `provision.FileSink` → `FileStore` (읽기 추가) | **자체 SSH 파일 I/O 9곳 → 0** · 와이어 형식 정의 1곳 · `keyring.FileSource` 가 로컬·원격 겸용 | ☑ |
| **K8** | **표면 통일** — 유스케이스를 `internal/app` 으로, CLI·MCP 는 노출 수단으로 | CLI 로 만든 링을 MCP 가 읽음(실증) · MCP 도구 5개(`new`/`add`/`list`/`show`/`import`) · **`export` 는 의도적 부재**(비밀이 에이전트 기록에 남지 않도록, 부재를 테스트로 고정) · `GenerateOpts.Validators` 를 `*int` 로 바꿔 "미설정"과 "없음"을 타입으로 구분 | ☑ |
| **K9** | **원격 링** (사용자 결정 2026-08-25) — `--keyring-dir` 이 target 문법(`srv://…`)을 수용, 링의 생성·조회·가져오기가 **파일 인터페이스 경유로 서버 위에서** 동작(`GenerateAt/ExtendAt/ImportAt/LoadPresetAt`, 로컬 래퍼 유지로 기존 18개 호출처 무변경). `--server-set`·`--docker` 는 전 동사 공통 플래그로 승격. **수정된 선재 결함**: 원격형 경로를 로컬 폴더 *이름*으로 취급해 운영자 머신에 조용히 생성하던 오배치 | 라이브(docker): 서버 위 생성·add 비승격·`list --verify` 원격 통과·중복 생성 원격 존재 검사로 거부·`--docker` 부재 시 실주소 timeout · 게이트 스위트 `Live_…CreatesARingOnAServer` 상설화 | ☑ |
| **K10** | **링 통째 가져오기 + 무결성 게이트** (사용자 결정 2026-08-25) — `import --from-ring <target>` 이 원격/로컬 링 전체를 한 명령으로 복제: 라벨·순번·validator 선언(BLS 목록·alloc 포함) 그대로, 항목마다 키에서 재파생해 원본 인덱스와 대조(`Entry.Verify`) 후 하나라도 다르면 전체 거부, 목적지에 링이 있으면 거부, 비밀번호는 원본 유지 또는 `--password` 재암호화. 단건 import 는 `--expect-address` 로 같은 대조를 호출자가 걸 수 있다. MCP `keyring_import` 에 `fromRing`/`expectAddress` 동일 노출 | 단위(복제가 선언 보존·변조 원본 거부·목적지 점유 거부) + CLI(선언 동반·플래그 혼용 거부·주소 불일치 거부) + 라이브 `Live_…ClonesARingFromAServer`(서버 링 → 로컬, 치환 보고·양측 주소 일치·로컬 재검증) | ☑ |
| **K7** | `--from` 단일 경로 문법 + `srv://<인벤토리이름>/path` | **네 표기가 한 코드로**(로컬·srv·host:path·ssh://) · **명령줄에 IP 없음** · 플래그 4개 → 1개(구 플래그는 deprecated 유지) | ☑ |

**의존성 추가**(K0 에서 완료): `github.com/kilic/bls12-381 v0.1.0` — **순수 Go** BLS12-381.
당초 적었던 `supranational/blst` 는 **CGO 라 `CGO_ENABLED=0` 빌드를 깬다**. kilic 은 go-wbft 의
`go.sum` 과 **모듈 해시가 동일**하다. `decred/…/secp256k1/v4` 는 이미 간접 의존성이었고 직접으로 승격.

**K6 이 keyring 을 넘어선다**: `FileSink` 에 읽기가 없어서 `keymat` 이 자체 SSH 읽기를 따로 만들었다 —
추상화가 한쪽 방향만 있으면 반대 방향은 옆에 새로 생긴다. 넓히면 청사진 읽기·genesis 확인·
산출물 검증이 전부 같은 통로를 쓴다.

**DST 주의**(K0): PoP 서명의 DST 를 빼면 **형식은 멀쩡한데 검증 실패하는 PoP** 이 나온다.
실제로 첫 파생 시도에서 그렇게 됐다 — 골든 테스트로 고정한다.

### N — 네트워크 청사진 (구성 정보를 하나의 선언으로)

> 근거: [[network-blueprint-design]](network-blueprint-design.md).
> 실측 문제: 구성 정보가 **4조각**(`topology.yaml`·`serverset`·`presets/keys`·`poa.Config`)으로 흩어져
> 어느 것도 전체를 말하지 못한다. **preset 이 선택이 아니라 전제**이고(`keys.LoadPreset` 필수),
> 바이너리 경로가 **20개 파일**에 분산돼 있다. 노드별 nodekey·계정·포트·서버를 지정할 수단이 없다.

> **원칙: raw 가 먼저, preset 은 나중.** 손으로 쓴 값만으로 네트워크가 서는 것을 먼저 만들고,
> 그 위에 preset 을 **생성기**로 얹는다. 반대로 하면 preset 이 다시 전제가 된다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **N0** | 역할을 `bp·en·pn` 3종으로 정리 — **NM1 로 흡수**([[netmap-design]] §4) | `validator→bp`·`endpoint→en` 이관 · `node.RoleEN`/`RoleEndpoint` 중복 해소 · 기존 토폴로지 호환 | ☑ **NM6 완료 2026-09-07.** 조립·핸드오프·attach 가 모두 `bp`/`en` 을 기록하고 `LegacySpelling` 은 없다. 옛 철자는 **읽는 자리에서 접는다** — `Node.UnmarshalJSON` 과 `Record.UnmarshalJSON` 둘 다에 붙였는데, 워크스페이스가 표를 두 벌(`Record` 는 workspace.json, `Node` 는 단계 사이 전달) 들고 있어 한쪽만 접으면 `chain status` 는 "validator", 나머지는 "bp" 를 말하게 된다. 선택자는 집합 대신 `node.Is` 로 접어 코드가 직접 넘긴 표까지 맞춘다. 알아채지 못한 채 굳을 뻔한 것을 테스트 다섯 개가 잡았다(라이브 재검증은 아직) |
| **N0b** | **피어링 그래프를 역할에서 파생** — 현재는 풀메시 고정. `bp ↔ pn ↔ en`(en 은 bp 를 직접 모른다) | ☑ **NM2 완료** — mesh 골든 동일 · proxied 는 bp/en 의 목록에 pn 만 · poa+pn 은 `SupportsRole` 로 거부. 조립 4곳의 **전환**은 NM3 | ☑ |
| **N1** | `blueprint` 선언 스키마 + 파서 (L1 순수) | 부분 청사진 round-trip · 미지 필드 거부 · fuzz | ☑ **완료 2026-09-07.** `internal/core/blueprint` — `Parse`·`Marshal`·`Validate`. **해석은 하지 않는다**: 빠진 값을 채우는 일은 N2 몫이고, 둘을 갈라 두어야 부분 선언이 왕복한다. 미지 필드는 `KnownFields` 로 거부하는데, 선언에서는 오타가 증상을 남기지 않고 다른 곳에서 온 값으로 조용히 대체되기 때문이다. 검증은 **문서가 혼자 판단할 수 있는 것만** 한다(이름 중복·역할·참조가 선언된 노드를 가리키는지·모순되는 두 지정). 체인 등록 여부나 파일 존재는 확인하지 않는다 — 그것을 여기서 하면 같은 질문이 두 곳에 생긴다. **퍼저가 진짜 결함을 찾았다**: `nodes: []` 와 `nodes` 생략이 서로 다른 값이 되어 생성기 출력이 자기 입력과 같지 않았다(`normalize` 로 접었고 그 입력을 코퍼스에 남겼다). 금액은 텍스트로 든다 — `1500000000000000000000000` 을 숫자로 읽으면 아래 자리가 사라지고 제네시스가 다른 잔액을 만든다. 포트는 `node.Endpoints` 를 그대로 쓴다(NM7 이 끝낸 "포트 표현 3벌"이 선언 형식에서 다시 시작되지 않게). 왕복 9종 · 미지 필드 6종 · 문서 내부 규칙 22종 · fuzz 326만 실행, 변이 셋으로 확인 |
| **N2** | `Resolve` — 출처 사슬 + `Sources` 기록 | 같은 청사진 → 항상 같은 `ResolvedNetwork`(결정성) | ☑ **완료 2026-09-07.** `Resolve(bp, Inputs) ResolvedNetwork`. 파일도 소켓도 열지 않고 배치·키셋·체인 사실을 **주입받는다** — 그래야 같은 입력이 같은 스냅샷을 낸다. **포트는 필드 단위로 병합한다**: `ports: {p2p: 8589}` 는 p2p 만 고정하고 나머지는 배치가 정한다. 통째로 골랐다면 포트 하나를 적는 일이 나머지 여섯을 버리는 일이 됐을 것이다. **검사를 두 곳으로 갈랐다** — 문서 혼자 판단할 수 있는 것은 `Validate`, 노드 표가 있어야 아는 것은 `Resolve`. 이 구분이 결함 둘을 드러냈다: ① 이름을 안 붙인 문서에서 `validators: [bp1]` 을 `Validate` 가 거부했다(역할 라벨은 파생되므로 정당한 참조다), ② 바이너리 오버라이드가 **선언된** 이름으로만 맞춰 이름 없는 노드는 영영 못 바꾸면서 아무 말도 안 했다. 이제 확정된 이름으로 맞추고, 가리키는 노드가 없으면 거절한다. 검증자 목록은 노드 순서로 낸다 — 제네시스가 목록으로 기록하므로 순서가 흔들리면 문서 하나에서 제네시스 둘이 나온다. 결정성 20회 반복 · 출처 9종 · 거절 5종, 변이 셋으로 확인 |
| **N3** | **raw 경로 완성** — 노드별 nodekey·계정·포트·서버·바이너리 오버라이드 선언 | **preset 없이** 손으로 쓴 청사진만으로 3체인 4노드 기동(라이브) | ☑ **완료 2026-09-07 — 라이브 게이트 통과.** `chain up --blueprint network.yaml` 로 stablenet·wbft·wemix 각 4노드(bp 3 + en 1)가 preset 디렉터리 없이 서서 블록을 만들었다(높이 15/15/59, 노드끼리 일치). 청사진이 키를 들면 `store.DeclaredKeys` 가 그 링을 재료화하고, 조립은 `KeySource` 경계를 그대로 쓰므로 새 경로를 배울 것이 없다. **첫 시도는 조용히 실패했다** — 모든 스텝이 성공을 보고했는데 봉인 노드가 전부 `Failed to read password file` 로 죽었다. 링 기록 절차를 일부만 되풀이해 공용 `password` 파일을 빠뜨린 탓이고, 이 트랙이 없애려는 갈라짐을 내가 만든 셈이다. `ImportRing`(이미 있던 단일 writer)을 쓰게 고쳤고 덤으로 키가 색인이 주장하는 신원을 재파생하는지까지 검증한다. **못 지키는 선언은 이름을 대고 거절한다**: 노드별 `server:` 는 아직 배정기가 못 받으므로 무시하지 않고 거부한다. 남은 빚 둘: 노드별 서버 배치, 그리고 BLS 를 패밀리에 묻지 않고 늘 파생하는 것(생성 경로도 같다) |
| **N4** | `Materialize` — 노드별 산출물 묶음 → Sink | 로컬/원격 분기 없음 | ☑ **완료 2026-09-07 — 재구조화는 A3·A4b 가 이미 끝냈고, 남은 일은 그것을 못 박는 것이었다.** 실측: `chainsetup` 에 직접 파일 쓰기 **0곳**, 18곳이 전부 `Files` 경계를 지난다. `IsRemote()` 를 묻는 자리는 10곳인데 **9곳은 "대상이 어디냐"는 사실**이지 두 번째 코드 경로가 아니다(원격 노드의 키는 대상 데이터 루트 밑에 있고, 로컬은 키 세트 그 자체다 — 조작자 쪽 경로를 원격 config 에 박은 것이 노드가 못 보는 기계에서 nodekey 를 찾게 만든 원인이었다). `internal/arch` 에 래칫을 두어 **각 자리가 왜 사실인지를 목록에 적게** 했다. 줄기만 하고, 사라진 항목도 거부한다. 변이 둘로 확인. **진짜 결함은 하나**: `rm` 이 원격 데이터 플레인을 거부한다. `filestore.Store` 에 삭제가 없어서(확인·읽기·쓰기·체크섬뿐) 경계를 통과할 수 없다. 파괴적 원격 작업이라 검증할 기계가 있는 **R6 과 함께** 간다. 그때까지는 절반만 지우거나 지웠다고 거짓말하지 않고 소리 내어 거절한다 |
| **N5** | preset 을 청사진 **생성기**로 — `chain blueprint --from-preset` | preset 산출 청사진이 N3 경로와 동일 결과 | ☑ **완료 2026-09-07.** 뒤집힘이 성립한다: `preset → (내부 조립) → 네트워크` 가 `preset → 청사진 → 네트워크` 가 됐고, 가운데를 열어 보고 고칠 수 있다. **키는 경로로 가리키고 문서에 박지 않는다** — 청사진은 읽고 나누고 커밋하라고 만든 것이라, 비밀키를 넣는 생성기는 기본값을 위험하게 만든다. 테스트가 문서에 키가 섞였는지 본다. 동일성은 저장소 preset 으로 확인했다: 찍은 문서를 다시 읽어 해석하면 주소·devp2p 키·BLS 키·검증자 순서가 preset 이 든 것과 전부 같다. **라이브**: 찍은 문서로 wbft 5노드(bp 4 + en 1)가 서서 높이 15, 워크스페이스의 검증자 넷이 preset metadata 와 순서까지 일치. `chain blueprint` 는 명령 표면의 네 번째 갈래(저작)로 등록했다 — 대상을 건드리지 않으므로 `stop`·`rm` 과 같은 칸에 두면 도는 체인에 뭔가 한다고 말하는 셈이다. 변이 둘(키 인라인 · 노드 순서 뒤집기)로 확인 |
| **N6** | `topology.yaml` 흡수 (이관 기간 병존, 혼용 거부) | 두 형식을 다 읽되 한 조립에 섞으면 거부 | ☑ **완료 2026-09-07.** 청사진은 토폴로지의 진부분집합이라(토폴로지가 말하는 것 전부 + 키·포트·서버·계정·alloc·거버넌스) `chain blueprint --from-topology` 로 손 번역 없이 넓은 형식으로 옮긴다. 그래서 흡수이지 손실 있는 이관이 아니고, 플래그 데이 없이 둘을 병존시킬 수 있다. **혼용은 거부한다** — 한 조립에 청사진과 토폴로지를 같이 주면 둘 중 한 저자는 자기 것이 아닌 네트워크를 읽게 된다. **옮길 자리가 없는 것은 버리지 않고 보고한다**: bootnode 표시와 토폴로지식 바이너리 이름(청사진은 경로를 직접 쓴다). 조용히 잃으면 온 파일과 다른 네트워크를 만드는 문서가 되고, 그게 이 트랙이 없애려는 실패다. 레거시 철자는 변환에서 정본으로 접힌다(NM6) |
| **N7** | **`core/netmap`** — ☑ **완료 2026-08-22** (NM1·NM1c·NM1b·NM2·NM3·NM4·NM5). 배치의 단일 소유자가 섰다: 자원 풀·결정적 할당·양방향 대장·라벨·피어링·경로. `place` 할당기 소멸 | 실측(착수 전): 배치 타입 8개 · 포트 표현 3벌(etcd 소실) · 역할 어휘 4벌 · static-nodes 조립 4벌 전부 풀메시 · `"node%d"` 라벨 파생 32곳. **종료 시**: 포트 1벌(etcd 보존) · 역할 폴딩 1곳 · 피어링 1곳(mesh/proxied) · 경로 파생 1곳 | ☑ |
| **N8** | ~~`serverset` 가용 자원 풀~~ → **이미 있음**(`slots`·포트 밴드, 2026-08-18). 명시적 범위 풀은 근거 생기면 | — | ☑ |
| **N9** | **해석 순서 강제** | 선행 단계가 안 돌았으면 명시적으로 거부한다 | ☑ **완료 2026-09-07.** `composeNeeds` 한 곳에 선언하고 모든 단계가 `require` 를 거친다. **적어 보니 두 군데가 틀려 있었다**: 순서는 keyring→netmap 이 아니라 `place → keys` 다(`keys` 가 노드 수를 배치에서 받는다), 그리고 `genesis` 는 키셋을 요구하지 않는다(템플릿을 치환하는 외부 체인은 키를 안 읽고, 요구하게 했더니 되던 조립이 깨졌다). 선언을 훑는 표 테스트 + 순환·도달 검사 + 유스케이스 경유 1건, 변이 세 개로 확인 |
| **N10** | **계정 라벨** — 라벨 ↔ 주소·개인키. **faucet 은 예약 라벨** | ☑ **완료 2026-09-05.** 실측이 문제를 말한다: 스펙 94파일에 주소 505회, 그중 **키에서 파생돼 키셋이 바뀌면 틀어지는 것이 143회**(`node1` 하나가 93회). 라벨은 주소만이 아니라 **서명 방법**까지 답한다 — 노드 계정은 그 노드가, 하네스만 가진 계정은 여기서 서명해 raw 로 보낸다. `0x` 접두사면 주소, 아니면 라벨. 미지의 라벨은 **아는 라벨을 나열하며 실패**(zero address 로 흘러가지 않는다). 주소 자리(`address`·`from`·`to`·`deployer`·`funder`)는 전부 라벨을 받는다. **2차**: `accounts:` 선언으로 dev 계정을 **genesis 밖에서** 만들고 faucet 으로 채운다 — 계정 추가가 genesis 수정을 뜻하지 않게 된 것이 이 단계의 요점. 잔액 없이 선언하면 0으로 남는다(가스 부족 경로 테스트용). 라이브: 선언한 dev1 이 **자기 키로 서명**하고 dev2 잔액이 정확히 1 wei(0에서 시작해 그 한 번만 받음). 잔여: 스펙 143곳 이관(기계적) | ☑ |
| **N11** | **다중 config** — 노드별 지정 | 일부 노드만 다른 config 로 재기동한다 | ☑ **완료 2026-09-08.** 게이트는 `swapNode` 가 이미 만족했고(`binary` 없이 `config` 만 줘도 액션·유스케이스·워크스페이스 세 층이 받는다), 빠져 있던 통합 테스트를 채웠다. 걸림돌이던 "키셋을 갖춘 기동된 망"은 실제로 조립해서 만들었다 — preset 키·배치·제네시스·config·argv 기록까지 진짜로 하고 드라이버만 스텁이다. 검증: node2 의 config 가 `SyncMode = "snap"` 으로 다시 렌더되고, **node1 은 그대로이며**, 오버라이드가 `node2` 범위에만 기록돼 나중 재기동이 그것을 쓴다. `restartNode` 에 `config:` 를 또 붙이지 않는다 — 한 동작에 철자가 둘이 된다. 변이 둘(config 만은 거부 · 오버라이드를 전 노드 범위로)로 확인. 부수 확인: config 만 바꾸는 스왑도 **바이너리 기록은 필요하다**(재기동에 실행 파일이 있어야 한다) |
| **N12** | **deploy skip 을 내용 해시로** | 같은 경로에 **다른 내용**이면 skip 하지 않음 | ☑ **2026-09-06.** deploy 가 genesis 와 config 를 **존재 여부만** 보고 "reused, not rewritten" 이라고 보고했다. 누가 편집한 genesis 도, 이전 구성이 남긴 config 도 똑같이 "있음"이라 노드가 그걸로 뜬다.

genesis·config 를 쓸 때 해시를 `State.LaunchInputs` 에 기록하고, deploy 가 대상의 내용과 대조한다. 다르면 파일 이름과 두 해시를 대고 어느 스텝을 다시 돌릴지 말한다. 되쓰지 않고 거절하는 이유는 deploy 의 일이 만드는 것이 아니라 확인하는 것이기 때문이다.

**같은 스텝 안에서 신원 파일은 이미 제대로 하고 있었다** — `shipIdentities` 가 `Checksum` 과 `filestore.Hash` 로 내용을 견준다. genesis·config 만 빠져 있었다.

테스트 3건(편집된 genesis 거절 · 교체된 config 거절 · 자기가 쓴 것은 통과). 변이 둘로 확인했다: 비교를 끄거나 해시 기록을 빼면 편집된 genesis 가 "reused" 로 통과한다 |
| **N13** | **skip 사유 기록** — `TestRecord.Status(s)` 에 사유가 없어 "왜 skip 됐는지"가 아티팩트에 안 남는다 | ☑ **F5 와 함께 완료 2026-08-22** — `TestRecord.Reason(why)` + `statusDoc.Reason`. 적용 4곳: 파싱 실패·미적용(`does not apply to this target (chain or required capabilities)`)·blocked 2종. 이유 없는 blocked 하나가 `chain.binary` 누락을 찾는 데 한 세션을 썼다 | ☑ |
| **N14** | ~~**`capability` 이름 충돌 정리**~~ **대상 소멸 2026-09-06** | — | ☒ `engine/capability` 도 `core/capability` 도 없다. 이름이 겹치는 패키지는 `wbft` 하나뿐이고(체인 플러그인과 합의 패밀리) 그건 별개다. 충돌은 R 트랙의 모듈 재편 과정에서 사라졌다 |

**N7~N14 는 2026-08-19 요구 재도출분**([[network-blueprint-design]] §6). 세 체인을 실제로 구성한
기록에서 다시 뽑았고, 다섯 요구(로컬 포트·원격 IP풀·라벨 지정·enode 순서·계정 라벨)가 전부
**같은 것 하나**를 가리켜 `netmap` 으로 모았다.

**키 파생 주의**(N3): wbft 계열은 nodekey 하나에서 계정·BLS 가 파생되므로 `bootnode` 바이너리가
필수다. poa 는 계정이 nodekey 와 독립이라 `account:` 를 따로 선언해야 한다.
**BLS 는 선언 필드가 아니다** — 선언하면 nodekey 와 어긋날 수 있고, 그 불일치는 합의에서 터진다.

**`pn` 주의**(N0b): 세 체인에 proxy 모드 플래그가 **없다**(실측). `pn` 은 argv 가 아니라
**static-nodes 그래프**로 표현된다 — `bp ↔ pn ↔ en`, en 은 bp 를 직접 알지 못한다.
현재 구현은 풀메시라 pn 을 두어도 효과가 없다.
**poa(wemix)는 pn 을 쓰지 않는다** — etcd 가 그 자리다. 선언하면 조용히 무시하지 말고 **오류**로
거부한다(`Family.SupportsRole`).

**N 과 F 의 관계**: F4(`GenesisArtifacts`)·F3(`BringUpPhases`)는 `ResolvedNetwork` 를 입력으로 받는다.
**N1·N2 를 먼저** 해야 F 가 조각을 다시 모으지 않는다.

### S — 표면 통일 (CLI/MCP/DSL 을 한 레지스트리로)

> 근거: [[surface-unification-design]](surface-unification-design.md).
> **2026-09-05: 표면 경로 부분은 §1l 의 U 트랙이 가져갔다.** S5·S6 은 폐기했다. 아래 줄수는
> 08-18 시점 기록이며, 지금은 CLI 가 네 패키지로 나뉘어 합계 4,176줄이다(`cmd/chainbench`
> 1,981 · `keyringcmd` 1,119 · `chaincmd` 828 · `resourcecmd` 248, 비테스트 기준). 코드가 옮겨
> 갔을 뿐 줄지 않았다.
> 실측 문제: `cmd/chainbench` 4,569줄 중 **21파일이 `app` 을 우회**해 L1~L4 를 직접 부른다.
> MCP 도구 46개 중 **34개가 JSON 스키마 손작성**. DSL 은 또 다른 레지스트리를 갖는다 —
> **기능 목록이 세 벌**이고 실제로 갈라졌다(`faucet`·`verify` 는 DSL 에 없다).

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **S0** | **`internal/feature`**(별도 패키지, `Deps` 소유) 레지스트리 골격 · 입력 태그→cobra 플래그/JSON 스키마 바인딩 · **`ReadOnly` 속성**(선언식 — 상태 불변 + 출력에 비밀 없음; `keyring export` 는 자격 없음) | 기존 동작 무변경 · 미등록 기능 카운트 테스트 · ReadOnly 선언이 스키마에 노출 | ☑ **완료 2026-09-08.** `internal/feature`(L5): `Registration`·`Register[In,Out]`·`Stage`·`ReadOnly`, 그리고 **입력 struct 태그 하나가 두 바인딩을 만든다** — `Flags` 가 cobra 플래그를, `Schema` 가 MCP JSON 스키마를.

**게이트 셋 다 실측으로 닫았다.** ① *기존 동작 무변경*: `ChainGenesisIn` 에 태그를 달았고(태그는 무해하다), 파생한 플래그 4개가 `chain genesis` 가 손으로 선언한 것과 이름·타입·도움말까지 일치한다. 그래서 S1 의 이관은 표면을 바꾸는 일이 아니라 **손으로 쓴 절반을 지우는 일**이 된다. ② *미등록 기능 카운트*: app 의 유스케이스 80개 중 0개 등록. 큰 숫자이고 그게 계획이다 — 래칫이 막는 것은 이 숫자가 **오르는 것**이다. ③ *ReadOnly 가 스키마에 노출*: `readOnlyHint` 로 나간다.

**이름 둘을 A7 이 커밋 전에 잡았다**: 설계가 `Descriptor` 라 불렀는데 그 이름이 이미 두 뜻(`core/registry` 의 체인 플러그인, `app` 의 재수출)으로 있어 `Registration` 으로, `All` 은 `core/registry` 것과 겹쳐 `Registered` 로 바꿨다.

**명령은 생성하지 않는다**(§3.4). 이름·계층·도움말 문구는 사람이 정하고, 태그에서 만드는 것은 플래그 바인딩뿐이다. 변이 셋으로 확인 |
| **S1** | ① Compose 이관 — `net.*` 9스텝 등록(이미 `app` 경유라 등록만) | `net up` 3체인 회귀 | ☑ **완료 2026-09-08.** compose 스테이지 **13개** 등록(9스텝 + 조회 넷). 미등록 카운트 80 → **67**, 래칫이 내려간 것을 알아채고 상수를 낮추라고 실패로 말했다. **태그 대조가 갈라짐 하나를 잡았다**: `chain.keys --keys-source` 가 태그엔 기본값 없음, 명령엔 `preset` 이었다. 명령을 파생으로 바꿀 때 몰랐으면 `chain keys` 의 기본 동작이 조용히 바뀐다. 그래서 **기본값 태그를 이번에 만들었다** — 영값에서 유추할 수 없다(`--validators` 는 4가 기본, `--endpoints` 는 0이 기본인데 유추하면 검증자 0인 네트워크가 된다). 파생 플래그 24개가 이름·타입·도움말·기본값까지 일치한다. 대조는 **한쪽 방향**이다 — 태그가 파생하는 것은 명령에 다 있어야 하고 명령은 더 가져도 된다(`--server` 류는 공유 바인더 몫이라 어느 한 입력의 것이 아니다). **라이브 3체인 회귀 통과**: 등록 전 커밋을 별도 worktree 에 빌드해 같은 절차로 나란히 돌렸다(stablenet 20/24 · wemix 58/58 · wbft 24/0). wbft 만 갈려서 before↔after 를 번갈아 4판씩 더 돌렸더니 **before 도 4판 중 1판이 0** 이고 after 는 2판이 정상 — 등록과 무관한 wbft 자체의 불안정이고, 이번 주에 규명한 "검증자가 늦게 합류하면 망이 안 선다"와 같은 자리다(`chain up` 은 아직 `health.Participants` 를 안 쓴다). **측정 실수 셋을 기록해 둔다**: 12초 단일 표본이 head 0 을 회귀처럼 보이게 했고(wbft 는 1에 머물다 가속한다), `pkill -f 'gstable|gwemix'` 가 그 문자열을 본문에 담은 러너 자신을 죽였고, 도는 스크립트를 편집해 bash 가 아직 안 읽은 줄이 어긋났으며 첫 러너가 살아 있는 채 둘째를 띄워 포트 8600 을 두고 다퉜다 — 같은 바이너리·같은 체인이 24 와 0 을 동시에 낸 것이 그 증거다 |
| **S2** | MCP `net_*` 를 레지스트리 소비로 전환 | 손작성 스키마 감소분 측정 | ◐ **조회 전용 목록은 완료(2026-09-08), 스키마 감축은 근거가 약하다.** §4.4 규칙 3 을 구현했다 — 도구가 `ReadOnly` 로 선언하고 `tools/list` 가 MCP 자신의 철자인 `annotations.readOnlyHint` 로 실어 나른다(이 서버는 2024-11-05 를 선언하고 그 필드는 2025-03-26 에 생겼으므로, 모르는 클라이언트는 무시하고 아는 쪽은 정답을 얻는다 — chainbench 만 아는 이름을 짓는 것보다 낫다). 54개 중 **28개**가 선언했다.

**CLI 와 MCP 가 어긋나지 않도록 짝을 못박았다.** 공유 레지스트리가 아직 없어 같은 사실을 두 번 선언하는데(cobra 주석 / 구조체 필드), 그게 바로 갈라지는 모양이라 18쌍을 테스트가 붙들고 있다. 한쪽만 표시를 떼면 실패한다(변이 확인).

**손작성 스키마 감축은 지금 하지 않는다.** 실측: 되풀이되는 속성은 `rpc` 16 · `workspaceDir` 11 · `chain` 11 이고 전부 한 줄짜리 선언이다. **설명이 갈라졌는지 세어 보니 갈라지지 않았다** — `rpc` 15곳이 같은 표현이고 다른 둘은 실제로 다른 인자다(배열 허용, attach 용). 즉 이 감축은 교정이 아니라 예방이고, 54개 도구를 기계적으로 훑는 diff 를 그 근거로 사기에는 약하다. 갈라짐이 실제로 생기면 그때가 착수 시점이고, 그 신호는 이 줄에 적힌 숫자를 다시 세면 나온다 |
| **S3** | ② Test 이관 — `tx`·`faucet`·`contract`·`verify` | CLI/MCP 동시 노출 | ☑ **U4 가 했다(2026-09-05).** 넷 다 `app.TxSend`·`Faucet`·`ContractDeploy`·`VerifyNetwork` 를 CLI 와 MCP 가 함께 부른다(실측 확인 2026-09-06). **DSL 은 뺀다** — 액션은 표면이 아니라 L3 어휘이고 app 을 부르면 import 순환이다(§1l U7 참조) |
| **S4** | ③ Report 이관 — `status`·`report`·`logs` | 등록 + 태그가 배포된 표면과 일치 | ☑ **완료 2026-09-08.** 넷 등록(report.session·report.log·report.timeline·network.status), 모두 조회 전용. 미등록 카운트 67 → **63**.\n\n**등록하려니 세 함수의 모양이 맞지 않았다** — `LogSearch(_ Deps, dir string, in)` 처럼 디렉터리를 별도 인자로 받았는데, 그건 사실 입력이다. `(ctx, Deps, In)` 으로 바꾸면서 필드를 두 번 선언하지 않으려고 **바인더가 임베디드 구조체를 읽게** 했다. 그래서 `app.LogSearchIn` 은 `Dir` + `collector.SearchOpts` 이고, 검색 조건은 여전히 한 곳에만 선언돼 있다.\n\n**표면 래칫이 곧바로 잡았다**: 호출부를 고치다 `reportcmd/log` 가 `collector` 를 직접 참조하게 됐다. app 이 `LogSearchFilter` 로 어휘를 내주게 고쳤고, 그 이름은 A7 이 다시 잡았다 — `LogFilter` 는 `core/rpc` 에서 eth_getLogs 필터를 뜻한다. 파생 플래그 7개가 명령과 일치하고, 변이로 확인했다 |
| **S5** | ~~`cmd/` 규칙 위반 파일 정리~~ **폐기 2026-09-05** | 게이트가 "`cmd/` 가 `app` 만 import" 였는데, 지키려던 성질은 import 목록이 아니라 표면 사이의 동등성이다. 줄수 목표(~1,800)는 **달성하지 못했다**: CLI 네 패키지 합계가 4,569 → 4,176 으로 거의 그대로고, 줄어든 것처럼 보였던 것은 코드가 `chaincmd`·`keyringcmd`·`resourcecmd` 로 옮겨 갔기 때문이다. 폐기 사유는 목표 달성이 아니라 규칙 교체다. 남은 실체는 §1l 의 U 트랙이 가져간다 | ☒ |
| **S6** | ~~`cmd` import 화이트리스트 테스트~~ **폐기 2026-09-05** | 대리 지표 대신 기능별 동등성 테스트로 대체한다(U0·U2~U7). 라체트는 "app 을 거치지 않는 항목 수"로 세운다 | ☒ |
| **S7** | **`query` 조회 투영** — `ReadOnly` 기능들을 최상위 `query <명사> <동사>` 로 **자동 생성**(손 트리 금지, [[surface-unification-design]] §4.4, 확정 2026-08-25). 정본 철자는 명사 그룹, `query` 는 같은 등록의 두 번째 렌더링. MCP 는 같은 속성으로 조회 전용 도구 목록을 얻는다 | `query keyring list` == `keyring list` (같은 등록 실증) · 비-ReadOnly 기능이 query 에 나타나면 테스트 실패 | ☑ **완료 2026-09-07.** `query` 는 손으로 만들지 않고 **명령 트리에서 생성한다**. 각 명령이 `surface.ReadOnly` 로 스스로 선언하고(추론이 아니다 — `node rpc` 는 메서드가 인자라 코드로 알 수 없고, `keyring show` 는 안전한데 `keyring export` 는 비밀을 찍으니 둘 다 "찍기만 한다"로는 갈리지 않는다), 투영이 그것을 읽는다. 19개가 선언했고 `query` 가 정확히 그것만 담는다.

**같은 등록임을 세 가지로 확인한다**: 설명이 같고, `RunE` 가 같은 함수 값이며, **플래그 변수가 공유된다**(투영에 `--json` 을 세우면 정본 명령이 그것을 본다). 셋째가 잡는 위험은 "생성자를 다시 불러 만든 두 번째 등록"이다 — 도움말은 멀쩡해 보이지만 변수가 따로라 갈라진다.

**주석 하나를 실측으로 고쳤다**: "복사로는 변수 공유를 얻을 수 없다"고 적었는데 틀렸다. `pflag.Flag` 의 `Value` 가 인터페이스라 구조체를 복사해도 변수는 공유된다. 탐침으로 확인하고 주석을 사실대로 바꿨다.

변이 셋으로 확인: 쓰기 명령(`chain genesis`)에 선언을 붙이면 잡히고, 플래그를 새로 선언하면 "두 플래그 집합"으로 걸리고, 투영에서 하나를 빼면 선언과 안 맞는다고 실패한다 |
| **S8** | **netmap 조회의 자기 그룹 독립** (사용자 결정 2026-08-25) — 배치 조회를 `net` 에서 분리해 모듈 이름 그대로의 최상위 그룹 `netmap`(show·pool·plan)으로. 동기: 조회가 조합 그룹에 묶여 있으면 netmap 모듈만 고쳤을 때 그 부분만 테스트할 수 없다. `plan` 신설 = 할당기를 질문으로 실행(인벤토리+형태→배치표, 워크스페이스 없음·무기록) → 배치 변경을 조합 없이 검증하는 통로. MCP 도 동일 이동(`chainbench_netmap_show/pool/plan`). `net` 은 상태를 바꾸는 조합 단계만 소유 | CLI 테스트 7건(plan 결정성·무기록·producer 0 거부·인벤토리 반영·자격 비노출, show 양방향·워크스페이스 요구, pool used 집계) — netmap 만 대상으로 실행 가능 | ☑ |
| **S10** | **용도별 포트 대역 + 방화벽 재현** (사용자 요구 2026-08-26) — ① 서버 세트 `ports` 에 `ws`/`auth`/`metrics` 대역 선언 추가(선언 시 rpc 파생 꺼짐, `portplan.PlanBands`). 로더의 p2pStep 최소는 1로(패밀리 요구는 allocate 의 portplan 이 검사 — wemix 만 넓은 예약, wbft 는 관성이던 예약 2→1 정직화). ② docker 서버들: `firewall.sh` 가 선언된 허용 목록만 열고 기본 DROP(목록은 그 파일이 가진다). ③ sshd 10022 + 서버 세트 `ssh.port: 10022`, gen-env 가 그 대역을 서버 세트에 기록 | 단위: 용도별 대역 로드·검증·tight-step 판정. 라이브: 허용/차단 포트 프로브(DROP·refused 구분), 새 대역으로 5대 분산 기동→블록 24→고아 0, keyring·driver·워크플로 라이브 스위트 10022 경유 통과 | ☑ |
| **S11** | **machine.Kind 제거 — 구분은 저장하지 않고 파생** (사용자 결정 2026-08-26) — Kind(local/remote/server)는 축이 지리처럼 읽히지만 실체는 "지정 방식"이었고, 저장된 kind 는 주소와 어긋날 수 있었다. Spec 은 필드만 갖고(서버명·호스트·경로), 지역성은 파생한다: `Server` 있음 → 세트 항목, 루프백/빈 호스트(로그인 미지정) → 이 머신, 그 외 → 직접 지정(환경 인증). 루프백+User/Port 명시는 의도적 SSH dial(터널·퍼블리시 포트). 표시 문구는 `Spec.Describe()` 한 곳으로 — net status·MCP·실행 기록의 자체 분기 소멸, 래칫 3항목 축소 | 단위: 파생 규칙·Validate·Describe. 라이브: fleet 기동·status `server server1:…` 표기·workspace.json 에 kind 없음·실행 기록 where 일치·keyring 스위트 통과 | ☑ |
| **S12** | **호스트키 정책을 파일로** (사용자 정책 2026-08-26: 환경변수 셋업 없이 config 로드) — `CHAINBENCH_SSH_KNOWN_HOSTS`·`CHAINBENCH_SSH_INSECURE_HOST_KEY` 제거. 서버 세트 `ssh:` 에 `known_hosts_file`/`insecure_host_key`(정확히 하나) 선언, 경로는 다른 시크릿 파일과 같은 규칙(세트 파일 기준 상대·`~` 확장). `remote.HostKeyPolicy` 가 데이터, `Spec.ResolveWithPolicy` 가 전달, 직접 표기(user@host)는 세트의 set-level 정책을 적용. docker 생성 세트는 `insecure_host_key: true` 를 스스로 선언 | 단위: 정책 콜백·모순 선언 거부·파일 경로 해석. 라이브: **환경변수 0개**로 keyring 원격 생성·검증, 5대 fleet 기동·정지(고아 0), keyring·driver·sudo 라이브 스위트 통과 | ☑ |
| **S9** | **서버 세트 단일화** (사용자 결정 2026-08-25) — ① 용어: 서버 목록 파일을 "서버 세트"로 통일(파일 `server-set.yaml`, 플래그 `--server-set`, 코드 `serverset` 패키지와 일치; "inventory" 는 Ansible 미경험자에게 낯설고, "server config" 는 노드 config 와 충돌). ② 자격 단일화: 이름 붙은 서버의 SSH 자격은 **서버 세트 파일이 유일한 출처** — 환경변수 참조 제거(남은 export 가 접속을 조용히 바꾸는 사고 차단). 비밀을 파일 밖에 두려면 `ssh.password_file`/`key_passphrase_file` 로 한 줄짜리 0600 파일 참조(시크릿 매니저가 내려주는 형태). `CHAINBENCH_REMOTE_*` 는 참조할 파일이 없는 직접 표기(`user@host:/path`) 전용으로 축소 | 단위: 환경변수를 켜도 파일 값이 이기는 것·password_file 판독(개행 절사)·password/password_file 동시 지정 거부·빈 시크릿 파일 거부. 라이브: docker 서버들에서 keyring 전체 스위트 재통과 | ☑ |

**`internal/feature` 를 별도 패키지로 두는 이유**: `app/feature` 로 하면 `Invoke(ctx, Deps, in)` 의
`Deps` 가 `app` 에 있어 `app → app/feature → app` **참조 순환**이 된다. 순환은 발생해서는 안 되며,
발생했다는 것 자체가 설계 미흡의 증거다. `feature` 가 `Deps` 를 소유하고 `app` 이 그것을 import 한다.

**F 계열과의 순서**: 독립이지만 `net start` 를 둘 다 건드린다 — **F3(페이즈 구조)을 먼저** 하고
S1 에서 등록해야 두 번 등록하지 않는다.

### D — 대시보드 디버깅 지원 (metric 시각화부터)

> 근거: [[dashboard-metrics-design]](dashboard-metrics-design.md). 결정(2026-08-24):
> **Prometheus·Grafana 서버 없이 자체 동작** — 노드의 `/debug/metrics/prometheus` 를
> 대시보드가 직접 긁는다. 파서(collector)·대상 목록(netmap)·SSE 배관·SPA 골격은 재사용.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **D1** | metric 인포그래픽 — 주기 스크레이프 루프 + 메모리 링버퍼 + SPA 차트(블록 높이·피어·txpool 노드별 비교) | 실행 중인 네트워크에서 노드별 차트가 실데이터로 갱신 · 외부 프로세스 의존 0 · 레퍼런스는 설계 문서 §4 (statsviz·Gatus·expfmt 등) | 미착수 |
| D2 | 완료 세션 화면 — 있는 `/api/sessions`·chainstate API 에 화면 소비자를 만든다 | 완료 세션의 판정·체인 상태 이력이 브라우저에서 열람 | 미착수 |
| D3 | 로그 연계 — 차트·이벤트 시점에서 해당 노드 로그로 점프 | collector tail 배관 재사용 | 미착수 |

### A(환경) — 지금 이 환경에서 무엇이 돌고 있는가

> 근거: 13개 체인 구성 요구 중 ⑩⑪⑫⑬ 는 한 문제다 — **아무도 환경의 현재 상태를 모른다.**
> 실측(2026-08-24~25): `os/signal` import 0건 · 파일 락 0건 · `preflight` 는 선언 정합성만
> 검사하고 살아있는 포트는 안 본다 · `procman` 은 자기가 띄운 것만 안다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **A1** | 인터럽트 핸들러 + 노드 수명 분리 + 실패 경로에서도 기록 | ☑ **완료 2026-08-25.** `net up --chain wemix` 를 30초에 인터럽트 → 종료 130 · 노드 4개 · **PID 4개 기록** · `net stop` 이 0으로 정리. 드러난 결함 2건: 노드를 `exec.CommandContext(요청 ctx)` 로 띄워 CLI 호출에 종속(취소되자 4개 중 3개만 남음) · 워크스페이스를 성공 시에만 저장해 실패한 스텝이 PID까지 버림 | ☑ |
| **A2** | 워크스페이스 락 — 실행 단위, 4상태(free/live/stale/foreign) | ☑ **완료 2026-08-25.** 46초 wemix 기동 중 t=20s 동시 실행이 pid·host·시각·명령줄과 함께 거부됨. **락은 호출이 아니라 실행 단위** — `net up` 이 9스텝을 부르며 각각 또 잡으므로 재진입 허용, 가장 바깥만 해제(첫 구현은 안쪽 release 가 바깥 락을 지워 동시 실행이 그대로 들어왔다) | ☑ |
| **A3** | 기동 전 포트 점유 조회 (`core/occupancy`) | ☑ **완료 2026-08-25.** `init` 직전(타깃에 쓰기 전) 전 노드 포트를 조회해 거부하고, **우리 것인지 남의 것인지** 분류한다. **dial 로는 못 잡는다**: 노드가 와일드카드 소켓에 바인드하면 루프백 dial 이 거부된다(lsof `*:8600` 점유 중 dial 실패 실측). bind 도 한 형태로는 부족 — 실측: 8600(와일드카드 점유)은 `127.0.0.1` bind 성공/`:` bind 실패, 8603(루프백 점유)은 정반대. **둘 다 성공해야 비어 있는 것**으로 판정하니 20개 리슨 소켓을 20개로 정확히 잡는다 | ☑ |
| A4 | 원격/fleet 점유 조회 — 모든 서버 대상(요구 ⑩의 "모든 서버") | **해소 확인 (2026-09-12 실측)**: `scanPorts`(`steps_lifecycle.go:715`)가 주소를 호스트별로 묶고 **각 호스트를 그 호스트 자신의 머신이 프로브**한다("the probe lies from anywhere else"). 비고가 이미 R5 해소를 적고도 상태가 ☐ 였다 | ☑ |

### H — 약속된 위치 (요구 ⑦)

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **H1** | **`core/home`(L0) 신설 — `~/.chainbench` 의 소유자** | ☑ **완료 2026-09-05.** "기본 위치" 가 세 답이었다: 키 세트 `keys/default`, 세션 `chainbench-out` (**둘 다 cwd 상대**), 구성만 `~/.chainbench/<stamp>`. 실측: 다른 디렉토리에서 `keyring list` 가 **자기가 만든 링을 못 보고**, `run` 이 세션을 그 자리에 흩뿌렸다. 셋을 한 답으로 모았고 우선순위(명시 > `CHAINBENCH_KEYRING` > 약속된 위치)는 그대로. 라이브: `/tmp` 에서 만든 링을 다른 곳에서 조회, 경로 없이 구성하면 `~/.chainbench/<stamp>/chainsetup` 에 모인다. 아키텍처 가드가 신규 패키지를 잡아 §3 에 배치될 때까지 막았다 | ☑ |

### G — genesis 정리 (리팩토링 순서 ③)

> 순서: keyring ☑ → netmap ☑ → **genesis** → config. 실측(2026-08-25): genesis 를
> 만드는 진입점이 흩어져 있고, 그 결과 경로마다 기능이 달랐다.

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **G1** | **조립 지점 단일화** — `engine.BuildGenesis`(소스 선택 + 커스터마이즈)를 모든 경로가 부른다 | ☑ **완료 2026-08-25.** 소스 선택이 **5곳 → 1곳**(엔진·스텝 두 경로가 `poa.FamilyID` 로 분기 + 세 곳이 하드코딩). 커스터마이즈를 소스 **밖으로** 빼서 `netcompose.customizeGenesis`(=`genesis.BuildNetwork` 의 옵션 처리를 줄 단위로 재구현한 사본) 소멸. 라이브: stablenet 4노드 · `chain up --case wemix` 15/15 | ☑ |
| **G1a** | 닫힌 격차 — **wemix + overlay** | ☑ genesis overlay 가 스텝 경로에선 반영되고 엔진 경로에선 **말없이 버려지던** 분기. 커스터마이즈가 소스 안(한 패밀리만 쓰는 곳)에 있었던 탓. 이제 패밀리가 만든 base 위에 동일하게 적용된다 | ☑ |
| **G2** | 핸드오프 경로를 패밀리 선언으로 · `chains/wemix/deploy` 폐기 | ◐ **2026-09-05.** **`remote` 명령군과 `deploy` 패키지(995줄) 폐기** — 네 하위명령 모두 core 경유 대체가 있다(`chain up --server`·poa 페이즈 액션·`upgrade`·`keyring import --from srv://`). **핸드오프가 패밀리 순서를 따른다**: 전원 기동 후 부트스트랩하던 것을 `BringUpPhases` 로 물어 프로듀서 단독 → 거버넌스·etcd → 나머지 → 메시 순으로. 거버넌스 배포 전 `WaitProducing` 도 컴포지션 경로와 맞췄다. **간헐 실패의 원인을 잡았다(2026-09-05).** 앞서 신원 대조 문제로 적었는데, 측정해 보니 아니었다. 실패한 판에서 프로듀서에게 직접 물으니 `self.name="producer"`, `self.addr=0xf959…` 로 자신을 제대로 찾고 있었다. `self.miner=false` 는 원인이 아니라 클러스터가 비어 있어서 생긴 결과였다. 진짜 원인은 두 가지다. 첫째, 노드는 거버넌스 계약을 읽고 나서야 자기가 어느 멤버인지 알고, 그 전에는 `admin.etcdInit()` 이 `ErrNotRunning` 으로 거절한다. 둘째, 그 거절을 콘솔이 글자로 찍을 뿐 프로세스는 성공으로 끝나는데 `poa.EtcdInit` 이 출력을 안 읽었다. 형제 함수 `EtcdJoin` 은 F5b 때 같은 대가를 치르고 이미 읽고 있었다. 그래서 클러스터가 만들어지지 않았는데도 아무 데서도 오류가 나지 않았다. 고친 것은 `poa.WaitSelf`(자신을 알아볼 때까지 대기, 두 경로 모두 배선)와 `EtcdInit` 의 출력 검사다. **덤으로 요구사항 ⑩의 미배선을 찾았다** — 컴포지션 경로는 A3 이후 포트를 검사하는데 핸드오프는 안 했다. 남은 노드가 프로듀서 p2p 포트를 잡고 있자 "IPC 가 30초 안에 안 생겼다"로만 보고하고 포트도 범인도 말하지 않았다(3연속 실패). `Handoff.checkVacant` 로 배선하고, 어느 노드를 띄우는지 판정하는 규칙을 `selected` 하나로 합쳤다(검사와 기동이 어긋나면 안 되므로). **확인**: 최종 트리로 `TestUpgradeRunE2E` 연속 통과 | ☑ |
| G3 | genesis 산출물의 by-product(`Extra`) 배치 규칙 정리 — 지금은 소비자마다 따로 쓴다 | **해소 확인 (2026-09-12 실측)**: 규칙이 하나이고 쓰는 곳도 하나다 — `steps_compose.go:768` 이 `filepath.Dir(genesis)` 옆에 `Extra` 를 놓고 launch input 으로 기록하며, 모든 머신에서 같은 file store 를 지난다. "소비자마다 따로 쓴다" 는 더 이상 맞지 않는다 | ☑ |

### R — 로컬 docker 를 원격 서버처럼 (원격 경로 검증)

> 근거: [[docker-remote-design]](docker-remote-design.md). 실 원격 서버 없이 Rancher 의
> ubuntu 컨테이너를 가상 서버로 쓴다. 인벤토리는 실주소를 유지하고, 하네스가 접속하는
> 최하위 4곳(dialSSH + RPC 조립 3곳)에서만 `AddrMap` 이 loopback 퍼블리시 포트로 치환.
> 선례: `~/Work/github/packages/wemix-bp-test` 의 LocalMap (동작 검증 완료).

| # | 작업 | 게이트 | 상태 |
|---|---|---|---|
| **R1** | `AddrMap` boundary — **`--docker` 옵션이 전원**(파일 존재는 활성화 아님, §3.2a) + 매핑 파일(gitignore) 로드 + 접속 경계 주입 + 적용 보고 | ☑ **완료 2026-08-24.** `remote.AddrMap` 을 `target.resolveOver`(SSH 두 형태의 단일 수렴점)와 netcompose 의 `resolveTarget`/헬스 프로브에 주입. 옵션 없으면 파일 있어도 항등(라이브: 실주소 다이얼 후 timeout 실증) · 옵션+파일 부재는 오류(패키지·CLI 회귀) · `net` 은 `State.Docker` 영속 · 적용 내역 보고("docker: dialing 172.30.0.11:22 as 127.0.0.1:2201") · CLI/MCP 동일 옵션 | ☑ |
| R2 | docker 가상 서버 생성 스크립트 — compose + 인벤토리 v2 + localmap 자동 생성 | ☑ **완료 2026-08-24** (`env/docker/gen-env.sh`). 15대 기동, 생성된 인벤토리를 `net pool --fleet` 이 15×1=15 로 읽음. 퍼블리시 포트는 127.0.0.1 바인딩. server15 는 pn 예정(역할은 할당이 정하므로 서버 계층 구분 없음). **같은 날 접근 모델 교체(DR-b 해소)**: 실서버가 id+password 이고 sudo 가 그 비밀번호를 요구한다는 운영 사실에 맞춰, 키 로그인을 없애고 사용자 `chainbench`+비밀번호(+비밀번호 요구 sudo)로 재구성. `remote.ExecWithInput` + `driver.SSHSudoRunner`(sudo -S -k, 비밀번호는 stdin) 신설 — NM-e 의 "운반만 되던 sudo" 가 소비 가능해짐. 라이브: password 로그인·sudo whoami=root·root 전용 쓰기·keyring 원격 2종 전부 통과 | ☑ |
| R3 | keyring 원격 경로 라이브 — `import --from srv://` | ☑ **완료 2026-08-24.** server1 의 nodekey 를 `srv://server1/...` + `--docker` 로 가져와 주소·공개키 파생, 번역 보고 출력. 서버로의 쓰기 방향은 R4 의 provision 이 담당(keyring 에 서버 쓰기 verb 없음 — 의도). **후속(같은 날): 검증을 상설 테스트로** — 게이트된 라이브 스위트(`Live_Keyring*`, `CHAINBENCH_DOCKER_SERVERS=<build>` + docker 만 있으면 한 명령) 가 raw key·암호화 keystore(+password) 원격 가져오기와 주소 왕복 동일성을 검증. 이 스위트가 **실결함 1건을 즉시 잡음**: `user@host:path` 형태는 포트 미지정(0)이 매핑 뒤에 22 로 기본화되어 변환표를 지나침 → 매핑 전 기본화(`mapCredentials`, docker 불필요 단위 회귀 동반). **니모닉 가져오기를 CLI/MCP 에 노출**(core 에만 있던 갭): 골든 벡터(dev mnemonic → 0xf39F…) + 출처 배타성 테스트. 실측: `Ring.Install` 은 소비자 0 (배송은 provision 소관 — 정리 후보) | ☑ |
| R4 | 원격 조립·기동 라이브 (단일 서버) | ☑ **완료 2026-08-24.** `net up --server server1 --docker` 로 stablenet 4검증자가 **docker 서버 위에서** 15스텝 완주, 블록 16→24 전진(매핑 포트로 프로브), stop 후 고아 0. **P2 실증**: genesis·static-nodes·workspace 전부 실주소(172.30.0.11), loopback 0건(metrics 자기 바인드 기본값 제외). **원격 경로의 선재 결함 4건을 이 과정에서 발견·수정**: ① init 이 타깃 경로를 로컬 `os.ReadFile` 로 읽음 → Files boundary 경유 ② netcompose 에 신원 배송 부재 + config/argv 가 로컬 키 경로를 타깃에 구움 → `keysBase()` + provision 의 `shipIdentities`(engine 방식 이식) ③ **원격 launch 셸 문법** — `mkdir && nohup CMD &` 는 리스트 전체가 백그라운드 서브셸이 되어 세션 파이프를 문 채 노드를 기다림(노드가 즉사할 때만 우연히 통과) → `|| exit 1; nohup … &` + 문법 회귀 테스트 ④ 헬스 프로브가 fleet 에서 노드별 주소 대신 target 주소를 물음 → 노드 기록 주소로 | ☑ |
| R5 | **fleet 다중 호스트 기동** — 노드별 머신 해석 완성: allocate 가 노드마다 서버 세트 항목명을 기록하고(`NodeState.Server`), genesis·config·provision(신원 배송)·init·start·stop·restart·logs·사전 점검(포트 프로브·바이너리 검사)이 전부 **그 노드의 머신**으로 간다(`machineFor`/`eachMachine`, 명령당 머신별 1회 dial 캐시) | ☑ 라이브(2026-08-26): 5대 서버에 4 검증자+1 endpoint 분산 기동 → 서로 다른 머신끼리 합의해 블록 26 봉인 → 대장에 5머신 5기록 → stop → 5대 전부 고아 0. 단위: fleet allocate 가 서버명·호스트 분산을 기록 | ☑ |

| **R6** | **poa(wemix) 원격 실행 + 직렬 브링업** — poa 는 genesis 를 바이너리 실행으로 만들고 기동 뒤 거버넌스·etcd 부트스트랩도 바이너리를 실행하는데, 둘 다 하네스 로컬 `os/exec` 였다. `resource.Access`(Files+Driver.Commander)로 genesis 생성과 부트스트랩을 **타깃에서** 실행(노드별 머신 해석). 부트 노드를 최상위 producer 로 하고, 나머지 producer 를 내림차순으로 **한 대씩 시작+join**(직렬)하여 joiner 의 fork 경합(late-join 노드가 mid-reorg 구간을 "unauthorized block"으로 거부)을 제거 | ☑ 라이브(2026-09-02): 15대 docker 서버셋에 14 bp + 1 en 웹믹스가 원격에서 genesis 생성·거버넌스·etcd 부트스트랩 완주, etcd 14멤버 형성, sealer 로테이션 확인. `run` 에 `--docker` 추가. 단, **아래 잔여 버그로 재현 신뢰성은 불완전** | ◐ |

**R6 잔여 버그 (2026-09-02, 추후 디버깅) — go-wemix etcd 붕괴.** 직렬 브링업으로 fork 경합은
사라졌으나, 14 bp 규모의 느린 브리지에서 **부트 노드가 형성한 etcd 클러스터가 가끔 붕괴**한다.
`etcdInit` 으로 형성(VerifyEtcd 통과)된 뒤, 노드의 배경 etcd 관리가 거버넌스 멤버 목록(14개)을
보고 그 "클러스터"에 join 하려다 실패하며 형성된 클러스터를 잃는다:

```
etcd join failed name=node14 error="not found"
etcd failed to start: cannot fetch cluster info from peer urls
```

증상: 부트 노드 `admin.wemixInfo.etcd.cluster = undefined`, 부트가 거버넌스 블록에서 정지,
후속 join 이 전부 "not found". 재현 2회 중 1회 완전 성공(14멤버·로테이션), 1회 부트 etcd 붕괴.
chainbench 오케스트레이션(genesis·부트스트랩·직렬 순서)은 정상이고, **go-wemix 바이너리의 etcd
형성·안정성** 영역이다. 조사 방향: (a) 부트 노드가 거버넌스 멤버 전체를 아는 상태로 단일 클러스터를
안정 유지하는 조건(초기 클러스터를 자기 자신만으로 고정, 나머지는 add-member 순서), (b) etcd 붕괴
감지 후 재형성(`etcdInit` 재발행)을 브링업에 넣을지 — 단 형성 중 재발행이 오히려 방해한 선례 있음.
키·genesis 는 원인이 아님을 실측 확인(genesis 해시 전 노드 동일, 배포된 키 경로를 command 가 사용).

> **정정·인계 (2026-09-12).** 위 단락의 "join 하려다 형성된 클러스터를 잃는다" 는 소스와 맞지
> 않는다. join 경로는 `etcdIsRunning()` 가드 뒤에 있어 돌고 있는 etcd 를 내릴 수 없고,
> `"not found"` 는 join 을 할 **피어를 못 찾은** 것이다. 현재 판독과 재현·조사 방향은
> [`chain-handover-2026-09-12.md`](chain-handover-2026-09-12.md) §2 가 이긴다.

### 확정된 결정 (2026-08-22)

| # | 결정 | 근거 |
|---|---|---|
| D1 | **노드는 이름 둘을 갖는다** — 신원 `node7`(=`Index`, 저장·경로·keyring), 별칭 `en2`(역할 내 서수, 정의서 주소지정). **개명 없음** | 영속 식별자는 이미 라벨이 아니라 `Index int` 라 표기·주소지정 문제였다. spec 은 여러 토폴로지에서 도는데 `node7` 은 넷마다 다른 노드를 가리킨다 ([[netmap-design]] §2.5a) |
| D2 | **`--peering proxied` 는 필수** | 메인넷이 bp–pn–en 이고 **트랜잭션이 en 을 거쳐 전파**된다. mesh 만으로는 실제 전파 경로를 태우지 못한다 |
| D3 | **인벤토리 v2 단독** (v1 호환 없음) | 실제 파일은 gitignore 라 배포본이 확인되지 않는다 — 지금이 전환 비용 최저. 호환 로더는 다섯 번째 폴딩표가 된다 |

**NM5 완료 (2026-08-22) — netmap 본 트랙 종료.** 잔여는 NM6(철자 방출 전환, 2026-08-24 에
NM3 에서 분리 — N0 행과 [[netmap-design]] §4) 하나다. 라벨이 파생값에서 **데이터**가 됐다:
워크스페이스가 `label` 을 저장하고(구 워크스페이스는 index 폴백), `netmap.Layout` 이
datadir·config·log 경로를 그 라벨에서 파생한다 — `fmt.Sprintf("node%d")` 로 흩어져 있던 6곳이
한 함수가 됐다. `Request.Label` 로 운영자가 지은 이름이 보존되고, `net map --addr 127.0.0.1:31021`
로 **로그 한 줄의 주소를 노드로 되짚는다**(파생 etcd 포트 포함).

**`place` 의 할당기를 삭제했다** — `Allocator`·`NodePlacement`·`Mode`·`Capacity` 모두 소비자 0.
`NodeReq`(역할·sync·binary)만 남고, 그것은 배치가 아니라 기동 입력이다. N7 이 겨눴던
"`NodePlacement.Name` 이 버려진다"는 **필드 자체를 지워서** 닫혔다 — 5곳에서 4가지 철자로
만들어지고 아무도 읽지 않던 값이었다. **N7 완료.**

**NM4 완료 (2026-08-22)** — 조회 표면. `net map` 은 네 방향으로 답한다(노드 번호·라벨(신원 또는
별칭)·호스트·포트) — **포트로 노드를 되찾는 것**이 원래 동기였고, 이제 파생 etcd 포트로도 된다.
`net pool` 은 "왜 15개가 거부됐는가"를 명령 하나로 답한다(호스트×슬롯=용량, 사용/여유, 출처).
유스케이스는 `app` 에 하나씩, CLI·MCP 는 바인딩만(K8 선례). **자격증명은 어느 쪽에도 없고**,
`NetworkPoolOut` 에 그런 필드가 없다는 것을 리플렉션 테스트로 고정했다 — 유출은 조용하기 때문이다.

**NM4 를 하다 같은 결함을 한 번 더 잡았다**: `net map` 이 etcd 를 `-` 로 찍길래 보니
워크스페이스→netmap 변환이 또 그 필드를 빠뜨리고 있었다(손복사 4곳 중 하나). 필드별 복사를
없애고 `NodeState` 가 `node.Endpoints` 를 **임베드**하도록 바꿨다 — JSON 키는 그대로 인라인되어
영속 형식이 변하지 않으면서, 복사가 사라지니 빠뜨릴 필드도 없다.

**NM3 부분 완료 (2026-08-22)** — static-nodes 조립이 세 벌에서 한 곳(`netmap.Peering`)으로
모였다. engine·netcompose 가 이를 경유하고 `--peering` 이 CLI·MCP 에 노출된다. 착수 전
**전수 조사**로 역할 철자를 하나만 비교하던 9곳을 `netmap.Is` 로 접었다(NM3 part 1) — 그 목록에
genesis 검증자 수·`--unlock` 무장·BFT 최소치가 들어 있어, 정규 철자를 방출하는 순간 전부
오동작했을 자리다.

**라이브가 설계를 정정했다**: 초안의 proxied(= bp 가 pn 만 다이얼)로 5노드를 띄우자 **블록이
0에서 멈췄다.** 모든 bp 가 `ROUND-CHANGE` 를 자기 것만 세며 반복했고(`currentRoundChanges.count=1`),
pn 로그의 WBFT 라인은 2줄뿐이었다 — **pn 은 검증자가 아니라 합의 트래픽을 중계하지 않는다.**
확정형은 **bp↔bp 직결 + bp↔pn + pn↔en**(en 은 여전히 bp 를 모른다). 정정 후 재검증:
stablenet mesh 4노드 api 9/9 · wbft mesh 4노드 블록 54 · stablenet proxied 5노드 블록 전진 +
피어 4 + api 9/9, 세 경우 모두 고아 0.
**할당 경로도 전환됐다**: engine·netcompose·chainsetup 이 `netmap.Assign` 을 쓰고
`place.Allocator` 의 프로덕션 호출은 **0**이다. 전환하며 실측한 두 가지: `LocalOSAssigned` 는
소비자가 없었고(격자로 흡수되지 않는 별개 전략), `MinValidators` 는 전 호출지에서 1이었다 —
"프로듀서 최소 하나"는 `Assign` 이 직접 거부한다. 라이브 재확인: 포트 동일(8600·31000 대역),
api 9/9, 고아 0.
**포트 표현도 하나가 됐다** — 3벌 → 1벌. 설계는 `node.Endpoints` 를 `netmap.Ports` 로 *대체*
한다고 했으나 그 방향은 **상향 의존**이다(`node` 는 L0, `netmap` 은 L1). 어휘를 아래에 두는 것이
유일한 무순환 해법이라, `node.Endpoints` 가 `Etcd` 를 갖고 `portplan.Ports`·`netmap.Ports` 가
그 별칭이 됐다. 결과: **etcd 포트가 런타임·워크스페이스까지 살아남는다**(`"etcd": 31001` 실측).
그 포트는 `p2pStep>=2` 규칙이 존재하는 이유인데, 규칙이 지키는 값을 정작 아무도 되읽을 수
없던 상태였다. **NM3 완료** — 단, 원래 범위에 있던 철자 방출 전환은 여기서 하지 않았고,
**NM6 으로 분리**해 남겼다(2026-08-24 검토에서 확정. N0 행이 추적한다).

**NM2 완료 (2026-08-22)** — 피어링이 역할에서 파생된다. `mesh` 는 현행과 바이트 동일(골든:
`armSpecs` 가 렌더한 config 의 enode 목록 == `netmap.Mesh`, **self 항목 포함까지**; self 를 뺀
변형으로 실패를 확인), `proxied` 는 bp↔pn↔en 이라 **en 의 목록에 bp 가 없다**. pn 없는 proxied 와
poa+pn 은 거부한다 — `ConsensusFamily.SupportsRole` 신설(구현 2곳뿐이라 값싼 boundary).
**동시에 고친 잠복 결함**: 두 패밀리의 `StartFlags` 가 `--mine` 을 `RoleValidator` 철자에만
걸고 있었다. NM3 이 정규 철자를 방출하면 **프로듀서가 --mine 없이 떠서 체인이 멈춘다** —
NM1c 가 셀렉터에서 찾은 것과 같은 부류이며, 이번엔 블록 생성 자체를 좌우한다. 이제 두 철자
모두 `netmap.NormalizeRole` 로 접는다. N0b 가 닫혔다.

**NM1b 완료 (2026-08-22)** — `netmap.Pool`/`Assign` 이 자원 격자(hosts × slots)를 결정적으로
할당하고, 인벤토리가 v2(pool) 단독이 됐다. `place` 의 두 결정적 모드는 이 격자의 특수해임을
**등가 테스트로 고정**했다(라벨·호스트·전 포트 비교) — 없으면 NM3 이 리팩터를 자처하며 모든
넷의 포트를 옮긴다. v2 는 호스트별 개별 설정을 잃었다(풀은 균질); 근거가 생기면 `hosts[]`
오버라이드로 얹는다. `--peering`·`net map`·`net pool` 은 NM2·NM4.

**NM1c 완료 (2026-08-22)** — D1 의 코드 반영이자 NM3 의 선행 조건. 셀렉터의 역할 폴딩표가
`netmap.NormalizeRole` 을 경유하고, 신원 라벨(`node7`)도 셀렉터로 해석된다.
**고친 결함**: `session.rolesForToken` 이 `"bp"` 를 `RoleValidator` 에만 매핑해, 정규 역할
`bp` 를 가진 노드가 자기 셀렉터에 매칭되지 않았다. 두 철자가 섞인 넷에서는 **`bp1` 이 두 번째
노드로 조용히 해석**된다(옛 표로 실측). NM3 이 정규 철자를 방출하기 시작하면 터질 자리였다.

### 결정이 필요한 열린 질문

[[family-bringup-design]] §9 참조. 요약: (1) `Phase.Actions` 를 문자열로 둘지 타입 상수로 둘지,
(2) 부트노드 선정 기준(현 `poa.BootRole` 은 4검증자면 전부 참이라 기준이 못 된다 → 토폴로지
`bootnode: true` 권장), (3) wemix `config.json` 의 `env` 정책값을 어디서 받을지.

---

## 1h. 아키텍처 v2 — 모듈 재편 (사용자 결정 2026-08-25)

> 결정 요지. **CLI 는 core 를 직접 호출**한다(기능 동작 확인 용도). **MCP 는 app 을
> 경유**하며, app 은 워크플로 층이다(DSL 파싱 → chainsetup → testengine → 수집 → 레포트).
> **netmap** 이 서버 정보 관리·자원 분배(ip·port)·enode 생성(공개키는 입력)·low level
> 접근 wrapper(유일 통로)를 소유한다. low level(machine·process·FileStore·driver·SSH)은
> 필요 정보를 전부 파라미터로 받는 순수 기능으로 남는다. 경계마다 **소비자 측 작은
> interface** 로만 노출한다. 상세 설계는 V0.1 에서 architecture 문서로 기록한다.

**모듈 네이밍 규칙** (새 모듈·개명 시 이 표로 판정):

1. 소문자 한 덩어리 — 하이픈·언더스코어·대문자 금지, 폴더명 = 패키지명
2. 소유하는 명사로 짓는다 — 동작이 역할이면 "대상+동작명사" 합성 최대 2단어 (`chainsetup`)
3. 덤핑 단어 금지 — util·common·helpers·shared·misc·base·manager (관리자는 타입으로: `process.Manager`)
4. stutter 금지 — `machine.Spec` ○ / `machine.Machine` ✗ (현 `target.Target` 이 위반 사례)
5. 단수형 — 집합이 주제일 때만 집합 명사 (`serverset` ○, 현 `accounts` 위반)
6. 계층은 경로가 말한다 — 이름에 core·app 접두어 금지
7. 약어는 업계 표준만 — rpc·ssh·mcp·dsl ○ / `netreg` ✗

**빌드 안전 순서 원칙**: 모든 이동은 신설(추가만) → 소비자 전환 → 구코드 제거의
세 단계로 하고, 태스크 하나 = PR 하나 = 빌드·테스트·lint 통과 상태다. 개명은 한 PR
안에서 원자적으로 끝낸다(별칭 잔류 금지). 선행 열이 비면 착수 가능.

| # | 작업 | 선행 | 게이트 | 상태 |
|---|---|---|---|---|
| **V0.1** | 아키텍처 v2 결정 기록 — 레이어 그림·모듈 책임·CLI/MCP 비대칭·네이밍 규칙을 architecture 문서로 | — | 문서 등급 표기(현행 설계) + `docs/README.md` 권위 순서 반영 | ☑ |
| **V0.2** | AST 전수 측정 — app·netcompose·engine·target·driver 함수별 이동표(현 위치 → 목표 칸) | — | 이동표가 V1~V6 각 태스크의 대상 파일을 명시 | ☑ (8패키지 541심볼, `v2-move-map` (2026-09-24 삭제)) |
| **V1.1** | `core/target` → `core/machine` 개명 — `machine.Spec`/`machine.Access`(stutter 해소), 소비자 일괄 전환 | V0.2 | 한 PR 원자 개명 · 전 소비자 컴파일 · 기존 테스트 무변경 통과 | ☑ |
| **V1.2** | 무분기 감사 — machine 소비자의 local/remote 분기 전수 검사, 분기는 machine 내부로 | V1.1 | 제거: app keyring 로컬 지름길(해석기로 단일화)·netcompose 구조 검증(`Spec.Validate` 신설로 이동). 잔여 분기는 래칫 테스트가 유예 목록으로 고정(V2.2·V2.3·V5 에서 소멸, 축소만 허용) — `internal/arch` TestMachineConsumersDoNotBranchOnKind | ☑ |
| **V2.1** | netmap 접근 wrapper 신설 — 서버 이름 → 능력 손잡이(FileStore·Driver), `--docker` 치환·치환 보고·자격 결합 내장. 추가만, 기존 코드 무변경 | V1.1 | 단위: 치환·보고·자격이 wrapper 한 곳에서 재현 | ☑ (`netmap.Opener`) |
| **V2.2** | keyring 소비 전환 — `RingRef.open` 의 개별 배선을 wrapper 호출로 교체 | V2.1 | keyring 라이브 스위트(원격 링·복제) 통과 | ☑ (keyflags 분기도 소멸, 래칫 축소) |
| **V2.3** | netcompose 소비 전환 — `resolveTarget` 을 wrapper 로 교체(서버 세트 nil 전달 결함 구조적 해소) | V2.1 | 원격 net 단계가 env 변수 없이 서버 세트만으로 동작(라이브) | ☑ (health 프로브도 `Opener.AddrMap` 경유) |
| **V2.4** | serverset 흡수 — netmap 이 서버 정보 관리를 소유(패키지 이동 또는 내부화) | V2.2, V2.3 | 외부에서 serverset 직접 import 0건 — `internal/netmap/internal/serverset` 로 컴파일러가 강제. `ResolveServer` 도 app 에서 모듈로 이동(app 은 별칭만) | ☑ |
| **V2.5** | enode 생성 이관 — netcompose 의 enode·static-nodes 조합을 netmap 으로(공개키는 입력 파라미터) | V2.3 | 기존 enode 골든 값 바이트 동일 (`netmap.Enode`·`PeerList`, 골든 테스트 이동) | ☑ |
| **V3.1** | `keyring/store` 분리 — 링 저장·읽기(레이아웃·metadata·암호화 파일)를 하위 패키지로, 키 역학은 keyring 에 잔류 | V0.2 | 한 PR 내 소비자 전환 · 전 테스트 통과 | ☑ (17개 소비 파일 전환, 저장 테스트 동반 이동) |
| **V3.2** | 링 위치 해석 이동 — `--keyring-dir` 우선순위(플래그>env>기본)를 store 로 | V3.1 | 위치 보고(`keyring: <dir> (<source>)`) 동작 유지 | ☑ (`store.Locate`) |
| **V3.3** | app keyring 슬림화 + CLI 직접 호출 — keyringcmd 가 core(store·netmap wrapper)를 직접 호출, app 은 MCP 용 얇은 함수만 | V2.2, V3.2 | CLI 전 명령 동작 동일(테스트) · app keyring 은 호출만 | ☑ (동사 조립은 `internal/keyring` 모듈로 — CLI 직접 호출, app 은 별칭+위임 27줄) |
| **V4.1** | driver 조회 보강 — 머신에서 바이너리/pid 실행 여부·포트 사용·명령 수행(결과 회수) 원시 기능 채움 | V1.1 | 단위 + docker 라이브(양면 프로브 유지) | ☑ (`ProcessInspector`·`Commander` 양쪽 드라이버. 원격 pid 는 `/proc` — 비특권 로그인에서 kill -0 의 EPERM 이 부재로 읽히는 함정 회피) |
| **V4.2** | `core/process` 신설 — `process.Manager` 실행 대장: 어떤 머신·어떤 바이너리·어떤 명령·pid | V4.1 | 단위: 기록·조회·이중 기동 감지 | ☑ (기존 procman 흡수·개명. 영속 `Ledger` 신설: Record 가 이중 기동을 두 pid 명시로 거부, 재열람 왕복 고정) |
| **V4.3** | pid 기록 전환 — netcompose 워크스페이스의 pid 관리를 process 대장으로 | V4.2 | start→stop→고아 0 라이브 재검증 | ☑ (`process.json` 이 정본, NodeState.PID 는 열람 뷰로 동기화. 라이브: 4노드 기동→블록 26→stop→서버 고아 0·대장 0건) |
| **V5.1** | chainsetup 수렴 — netcompose 의 순차 진행을 chainsetup 으로 이동, 단계 내용은 기능 모듈로 분리. 역할 정의: 체인을 구성해 블록 생성 상태까지 | V2.3, V4.3 | 기존 net up 전 단계 라이브 통과 · netcompose 잔여 코드 0 | ☑ (패키지 해체 완료 — 레거시 사례 러너는 T7.11 은퇴까지 동거, Step→CaseStep. 단계 내용의 기능 모듈 분리는 이미 경계에 있는 것 유지) |
| **V5.2** | 실행 기록 폴더 — 지정 폴더 아래 실행마다 폴더: 체인 id·입력 사본·배치표·genesis·실행 명령. **서버 세트 ssh 절 제외**(테스트로 고정) | V5.1 | 기록에서 자격증명 grep 0건 테스트 | ☑ (`runs/<stamp>/` — manifest·genesis 회수·launch-commands. 카나리 비밀번호 테스트로 값 비노출 고정; "password" 단어는 argv 의 파일 경로로 정당) |
| **V5.3** | 사전 점검 배선 — 구성 전 process 대장으로 기동 중 노드 검사, 케이스별 함수 분리·조립(전체 셋업·부분 재시작·점검만) | V5.1 | 이미 도는 노드 위 재구성 거부 라이브 | ☑ (포트 충돌은 init 의 기존 점검, 포트가 비어도 같은 바이너리가 돌면 start 가 pid 지목 거부 — `checkUnmanaged`+`Preflight`(점검만 진입점). 라이브 양쪽 재현) |
| **V5.4** | CLI `netcmd` 추출 — net 그룹 6파일을 패키지로, chainsetup 직접 호출 (serverFlags 중복 해소 포함) | V5.1 | keyringcmd 패턴 준수 · 도움말 무손실 | ☑ (선행으로 net·network·hardfork 동사 23개를 chainsetup 모듈로 이동(app 은 위임만), 테스트 동반 이동. 서버 선택 플래그는 `cmd/chainbench/internal/serverflag` 하나로 — netcmd·netmapcmd·run 공용) |
| **V6.1** | engine → `testengine` — 구성 책임 제거, "구성된 체인 위에서 테스트만 일관 수행" 으로 축소·개명 | V5.1 | 기존 테스트 스위트 결과 동일 | ☑ (구성 파일 10개(빌드환경·genesis·keysource·launcher·plan·nodecontrol·wemix 계열)와 테스트가 chainsetup 으로, 러너는 setup_bridge 한 파일로 위탁 — 의존 방향 chainsetup→engine 이 testengine→chainsetup 으로 역전, 순환 0) |
| **V6.2** | app 워크플로 — DSL 파싱 → chainsetup → testengine → 수집 → 레포트를 app 이 한 흐름으로 제공 | V6.1 | e2e: DSL 입력 하나로 셋업+테스트+레포트 산출 | ☑ (`app.RunSuite`: testspec.ReadFiles(신설, CLI 도 공용) → ChainUp → 봉인 대기(WaitBlocks) → attach 실행 → 수집 → 자동 해체. 라이브 e2e: DSL 1건 → server1 4노드 → 1 pass → 고아 0) |
| **V6.3** | MCP 전환 — MCP 도구가 app 워크플로·얇은 app 함수만 경유(CLI 는 core 직접 유지) | V6.2 | run 도구가 app 경유로 전환(`AttachRun`/`SessionSummary`). 잔여 직결 import 14종은 래칫 테스트가 축소 전용 목록으로 고정(각 항목이 소멸 후속을 명시) — `internal/arch` TestMCPGoesThroughApp | ☑ (전면 0건은 후속 축소로) |
| **V7** | 기회 개명 백로그 | 해당 트랙 | 네이밍 규칙 표 판정 통과 | ☑ **완료 2026-09-07 — 둘의 판정이 갈렸다.** `netreg`(규칙 7)은 **모듈이 이미 없다** — R1 이 `core/session` 으로 흡수했고, 남은 것은 파일 이름 셋이었다. 규칙 표는 모듈을 판정하지만 약어는 읽는 사람이 마주치는 모든 이름에 해당하므로 `networks.go` 로 바꿨다(소비자 0, 기계적).

`accounts`(규칙 5)는 **개명하지 않는다.** 이 패키지는 상류 SDK `github.com/0xmhha/accounts` 위의 경계이고, 이름이 그 SDK 를 가리켜서 찾기 쉽다. 규칙 5 의 목적은 복수형이 "실은 하나"라는 사실을 감추는 것을 막는 일인데, 여기서 주제는 계정 하나도 계정 집합도 아니라 **SDK 경계**다. 단수 `account` 로 바꾸면 그 대응이 끊기고(읽는 사람이 "accounts SDK 를 어떻게 쓰나"를 찾을 곳이 사라진다) 소비자 27곳이 바뀐다. 규칙이 막으려는 해악이 없는 자리에서 규칙의 글자만 맞추는 거래다 |

각 태스크 마무리마다: 해당 경계에 소비자 측 interface 수립 · 네이밍 규칙 판정 ·
빌드·테스트·lint·(원격이면) docker 라이브 게이트.

## 1i. 모듈 재편 — 관심사 단위 (사용자 결정 2026-08-27)

> 설계·근거·이동표는 `module-plan` (2026-09-24 삭제). 실측은
> `go run ./scripts/inventory/code-graph -symbols .` 이며, **단계마다 재측정으로 열고
> 재측정으로 닫는다**(같은 문서 §1).
>
> 부서 셋으로 나눈다. **자원(`resource`)이 정하고, 노드(`node`)가 기록하고,
> 프로세스(`process`)가 돌린다.** 그 위에 빌더 셋(genesis · nodeconfig · dsl),
> 그 위에 표면(cmd · mcp), 그 위에 오케스트레이션(chainsetup · testengine).
>
> **표는 실행 순서다**(위에서 아래로). P5(표면)가 P2 앞에 오는 것은 2026-08-28
> 결정 — P1 이 `netmap` 모듈을 없앤 뒤에도 표면이 그 이름을 부르고 있어, 이름과
> 경계를 먼저 맞춘다. P5 는 기록 구조를 건드리지 않는다(그건 P2 의 것).
>
> 이름은 **부서가 소유한 대상 명사**로 짓는다. 은유(`netmap`)·약어(`netreg`·
> `portplan`)·산출물(`place`)·동작(`alloc`)은 간판이 되지 못한다. `alloc` 은 이
> 저장소에서 이미 genesis 계정 배분을 뜻하므로 쓸 수 없다(실측 13건).

| # | 작업 | 선행 | 게이트 | 상태 |
|---|---|---|---|---|
| **P1.1** | **어휘·지도·경로·enode → `core/node`** — label(35)·role(36)·map(114)·peering(159)·layout(38)·enode(32) 약 380줄 이동. `Ports` 별칭 제거하고 `node.Endpoints` 로 통일 | — | `node` out-edge **0 유지**(L0) · 어휘만 쓰던 6곳(poa·wbft·nodeconfig·session·topology·testspec)의 `core/netmap` import 소멸 · `core/netmap` 에 pool.go 만 잔류 · 기존 테스트 무변경 통과 | ☑ **완료 2026-08-27** — `node` out-edge 0 · 어휘 6곳 import 소멸(엣지 268→260) · `core/netmap` 은 2파일 191줄만 잔류 · `NodeLabel`→`node.Label` stutter 해소 · 61패키지 테스트 통과 |
| **P1.2** | **`serverset` 승격 + `Opener` 합류 → `internal/resource`** — `netmap/internal/serverset`(1,028)와 `netmap` 표면(229)을 한 패키지로. 봉인 목적은 wrapper 가 같은 패키지에 들어오면서 유지. **서버 쪽 `Placement` 삭제**: `Source`·`DataRoot`·`Remote` 세 필드가 전부 같은 반환값(`Pool.Source`·`Target.DataRoot`·`Spec.IsRemote()`)에 이미 있는 사본이라, `ResolveServer` 가 `Pool` 과 `machine.Spec` 을 따로 돌려준다. **`fleet` 낱말 제거**(제품 용어는 server set 하나): `--fleet`→`--all-servers` · MCP `"fleet"`→`"all_servers"` · `ServerRef.Fleet`→`.All` · `fleetTarget`→`setTarget` · `CHAINBENCH_DOCKER_SERVERS`→`CHAINBENCH_DOCKER_SERVERS` · `ServersBuildDir`→`ServersBuildDir` · 주석의 은유까지. **`Config.Fleet()` 삭제 → `Config.Pool()` 로 통합**(둘이 같은 일을 하고 `Pool()` 은 프로덕션 호출자 0). 슬롯 나눗셈(`slots/len(hosts)`)은 제거 — **착수 후 실측 결과 도달 불가능한 죽은 산술**이었다(v2 는 풀 하나에 slots 하나를 선언하고 `expand()` 가 모든 호스트에 복사하므로 같은 값 N개를 N으로 나눈 값이다). 호스트별 슬롯은 형식 변경이라 별도 결정으로 분리 | P1.1 | `serverset → core/netmap` 엣지 소멸(동일 패키지) · 서버 쪽 `Placement` 심볼 0 · **`fleet` 문자열 0**(주석·테스트·스크립트 포함) · `--server-set` 과 `--docker` 라이브 경로 동작 동일 · keyring 원격 스위트(링 생성·복제) 통과 | ☑ **완료 2026-08-27** — `internal/netmap` 소멸(패키지 75→74, 엣지 260→258) · 서버측 `Placement` 심볼 0(`ResolveServerOut{Pool, Target, HasTarget}`) · `fleet` 문자열 0(코드·스크립트·샘플) · `resource` 5파일 1,176줄, 소비자 6 · 계층 위반 0 · 61패키지 통과 |
| **P1.3** | **풀·배정·포트밴드 → `resource`, `place` 흡수** — pool(145)+portplan(184) 이동, `place.NodeReq` → `node.LaunchReq`(착수 후 정정: 기동 요청이라 배정 요청과 다르다), `Ports` 별칭 2개 제거하고 `node.Endpoints` 로 통일 | P1.2 | 패키지 **4개 소멸**(`core/netmap`·`netmap`·`core/portplan`·`core/place`) · `resource → node` 단방향 · 계층 위반 0·  `net allocate`·`netmap plan` 산출 바이트 동일 | ☑ **완료 2026-08-28** — 패키지 4개 소멸(75→71, 엣지 268→246) · `resource → node`·machine·remote 만 · `node` out-edge 0 · `Bands` 중복 2→1(`plan()` 소멸) · `Reservation` 은 노드 사실이라 `core/node` 로(패밀리·registry 가 resource 를 import 하지 않음) · 골든 동일 · 61패키지·-race·lint 0 |
| **P1.4** | **슬롯 상한 검사** (결정 2026-08-28) — `Pool.Validate()` 가 선언한 슬롯 수만큼 `PlanBands` 를 실제로 돌려 `ValidatePorts` 로 충돌 검사. "가용 포트 수를 넘는 slots" 를 선언 시점에 거부(지금은 밴드에 끝이 없어 `slots: 1000` 도 통과, `pool.go` 주석은 하지 않는 검사를 한다고 적혀 있음). 형식 변경 없음. *(옛 P1.4? "호스트별 슬롯" 은 폐기 — 밴드가 세트 공통이라 포트가 허용하는 노드 수는 모든 호스트에서 같다. 개념이 성립하지 않음, 사용자 확인 2026-08-28)* | P1.3 | 초과 slots 선언이 `Validate` 에서 거부됨 · 기존 유효 세트는 전부 통과 | ☑ **완료 2026-08-28** — `Pool.Validate()` 가 선언 슬롯 전부를 `PlanBands`→`ValidatePorts` 로 검사. p2p 8500/10 + rpc 8600 에서 slots 11 이 "11 slot(s) exceed … port 8600" 으로 거부, 10 은 통과(테스트) |
| **P1.5** | **`resource.Inventory` 신설** (결정 2026-08-28) — 서버 세트 소유 모듈이 가용/할당을 관리: 메모리 인스턴스(`Open`·`Adopt`·`Take`·`Release`·`Usage`·`Full`), ip·port 가 필요한 곳은 여기서 할당받는다. **메모리가 정본**(파일 영속·복구는 최종 항목 F1). **반납 = `rm`**(`stop` 은 pid 만 제거, 자원은 노드 것으로 유지 — pid 는 재기동 시 갈린다). `Full` 오류는 누가 쥐고 있는지까지 출력. `net pool` 의 `Used` 가 워크스페이스 하나만 세던 것이 이것으로 고쳐짐 | P1.4 | `Usage` 가 같은 세트의 모든 네트워크를 계수 · `Take` 초과 시 `ErrFull` 에 보유자 목록 · 단위 테스트 | ☑ **완료 2026-08-28** — `resource.Inventory`(`NewInventory`·`Adopt`·`Take`·`Release`·`Usage`·`Full`·`ErrFull`) 메모리 인스턴스. `Adopt` 은 워크스페이스 기록(`chainsetup.Allocations`)에서 파생, host+p2p 포트로 슬롯 역산, 세트 밖·밴드 밖 기록은 무시, 중복 claim 은 먼저 것 유지. `app.NetworkPool` 이 명명된 워크스페이스 + `~/.chainbench` 아래 전부(`chainsetup.Discover`)를 채택해 `Used/Free/ByNetwork` 답변, `resource pool` 이 보유자 출력. 기본 워크스페이스 경로 소유를 `chainsetup.DefaultWorkspaceDir` 로 이동. **라이브 확인: 같은 세트의 두 조립이 같은 포트를 받아 인벤토리가 첫 것만 셈 → P2.x 의 근거** |
| **P5** | **표면 재정리 (P2 앞으로 당김, 결정 2026-08-28)** — ① `netmapcmd` 해체: `netmap` 모듈이 P1 에서 사라졌는데 표면이 그 이름을 부르고 있다. `netmap pool`(자원)·`plan`(배정 미리보기)은 resource 의 그룹으로, `show`(배치 조회)는 node 쪽으로, keyring 방식(모듈=그룹=MCP 묶음) 적용. 구체 모양은 착수 시 제안. ② **플래그 분리**(사용자 결정 2026-08-28): `--workspace-dir` = 셋업 정보를 생성하는 경로(+ 기본 경로 규칙, `~/.chainbench/<날짜시간>/<테스트명>/chainsetup` 방향) / `--data-dir` = 노드가 블록 데이터를 쌓는 디렉터리(geth 계열 `--datadir` 과 같은 뜻으로 통일). ③ CLI 는 모듈 직접·MCP 는 app 경유, 래칫 14종 축소. **P2 와의 선긋기: 이름과 경계만, 기록 구조는 건드리지 않는다** | P1.5 | 표면에서 `netmap` 문자열 0(재편으로 정한 이름만) · `--data-dir` 이 워크스페이스를 뜻하는 곳 0 · 래칫 14→한 자릿수 | ☑ **완료 2026-08-28** — `netmapcmd` 소멸: `resource pool·plan`(신설 `resourcecmd`) / `net show`(netcmd 로) / MCP `chainbench_resource_pool`·`_resource_plan`·`_net_show` · 공용 렌더러 `internal/mapview`(plan 과 show 가 같은 표) · 현행 경로 9곳 `--data-dir`→`--workspace-dir`(MCP 인자 `dataDir`→`workspaceDir`), 레거시 14곳은 T7.11 은퇴까지 유지 · `net new/up` 기본 경로 `~/.chainbench/<타임스탬프>/chainsetup`(생략 시 첫 줄에 경로 보고) · 표면에서 `netmap` 문자열 0 · 61패키지·race·lint 0 |
| **P2** | 노드 사실 레코드 — "노드 하나" 타입 10 → 3, 경로 계산 4곳 → 1 | P5 | 심볼 인벤토리로 계수 확인. `workspace.json` 의 생성 주체·이유와 워크스페이스 삼중 정의(chainsetup.Workspace·session.Composition·Environment)의 소유 확정 포함 | ☑ **완료 2026-08-28** — `node.Record` 신설(옛 `chainsetup.NodeState`, JSON 계약 유지) · 노드 타입 10→3(Record·driver.NodeSpec·node.Node), 나머지는 정확한 이름으로(`collector.Sample`·`chainsetup.Probe`·`topology.Entry`) · `node.Layout` 에 Nodekey/Keystore/StaticNodes/IPC 경로 추가, 데이터플레인 손조립 0 · 워크스페이스 삼중 정의 해소: `workspace.json` 의 주체는 `session.Composition`, `chainsetup.Workspace` 는 그 위의 도메인 상태, `session.Environment` 는 다른 수명(아티팩트) · `node` out-edge 0 유지 · 61패키지·race·lint 0 |
| **P2.x** | **배정이 Inventory 를 소비** — `Assign` 이 `Inventory` 에서 빈 슬롯을 받아 시작. 같은 세트로 올린 두 번째 네트워크가 첫 네트워크의 포트를 다시 받지 않게 됨(지금은 둘 다 슬롯 1부터 시작해 충돌) | P2 | 같은 세트 위 2개 네트워크 동시 기동 라이브 | ☑ **완료 2026-08-28** — `Inventory.Assign(reqs, network)` 이 빈 슬롯을 `Take` 해 배치(`place`). `chainsetup.Inventory(pool, self, named…)` 한 곳에서 인벤토리를 조립(기본 루트 전부 + 명명, 자기 자신 제외)하고 allocate·plan·pool 이 같은 claim 집합을 읽는다. `resource.Assign(pool)` 은 빈 인벤토리 위의 계획으로 남아 골든 바이트 동일. **라이브**: 같은 세트 두 조립이 31000/31010 과 31020/31030 을 받고, 반쯤 찬 세트의 `resource plan` 이 31040 부터 시작 |
| **P3** | 프로세스 — 실행은 `driver`, 정책은 `process`. 기동 진입점 8 → **3**(driver 실행 · `launcher` 기동 정책 · `process` 종료 정책 — 사용자 결정 2026-08-28: 옛 `supervisor` 는 sudo 역할로 읽혀 `launcher` 로). **`occupancy` → `inspector` 개명·확장**(사용자 결정 2026-08-28): ip 가용·port 가용·경로 유효성(data root·datadir·genesis·keystore·nodekey·config·log·binary)을 요청 시에만 실사해 사실만 답한다. 타입은 stutter 회피(`inspector.Report` 등), `driver.ProcessInspector` 어휘도 이때 정리 | P2 | 진입점 계수 · `chainsetup` 714줄 감소 | ☑ **P3 완료 2026-08-28** · P3.1 — `core/launcher` = `supervisor`+`chainsetup.LocalLauncher`(→`Direct`)+`driver/lifecycle.go`, `supervisor` 낱말 0, `driver.NodeOf` 로 Node 조립 4→1(hardfork 의 loopback 리터럴 소멸), `workspace.go` 의 etcd 포트 누락 복사 수정, `chainsetup` 6,593→6,256. **P3.2 완료 2026-08-28** — `NodeController`→`launcher.Controller`(pid 맵 삭제: arming 만 기억, pid 는 `session.Environment` 노드표 한 곳 — `NodeControl.Stop/Start` 가 갱신된 노드를 돌려주고 fault 액션이 `Env.UpdateNode` 로 써넣음), `chainsetup` 6,256→6,123. `Workspace.Start/Stop/Restart` 는 노드별 머신을 거쳐 driver 를 부르고 `Record.PID` 에 적으므로 그대로(조립 모드의 한 기록). `LocalSetup` 은 레거시 `setup --launch`/MCP `_start` 경로라 T7.11 과 함께. **P3.3 완료 2026-08-28** — `core/occupancy`→`core/inspector`: `Ports`(옛 Scan) · `Paths`(file seam 으로 타깃에서 존재 확인) · `Hosts`(도달) 세 질문, 요청 시에만, 사실만. `net start` 가 기동 전 binary·genesis·datadir·config 존재를 각 노드의 머신에서 확인해 빠진 것을 이름으로 보고. `occupancy` 문자열 0 |
| **P4** | 빌더 셋 — genesis 생성 지점 5 → 1, config 렌더 2 → 1, dsl 파서가 액션을 import 하지 않음 | P2, P3 | 계수 + import 방향 | ☑ **P4 완료 2026-08-28** · P4.1 genesis 완료 2026-08-28** — `core/genesis` 가 소스 선택(`SourceFor`, 패밀리 id 분기 0: `SourceProvider` 타입 단언)·`Compose`·프리셋 소스 소유, wemix 소스는 `consensus/poa.GenesisSource`(Family 가 capability 구현), 호출자 5곳 전부 `Compose`/주입 `Source` 경유, `chainsetup` 직접 파일 쓰기 0(layers §5 에서 제외), 5,828줄. **P4.2 config 완료 2026-08-28** — `nodeconfig.Spec` 단일 입력, `TOML`/`Argv` 두 렌더러, Spec 조립은 `launcher.NodeConfig` 한 곳, compose 의 config·launchopts·start 가 `peerPlan` 으로 같은 입력을 모음, argv 조립 3→1(`upgrade.LaunchArgs`·deploy 평평한 `LaunchArgs` 도 `Argv` 경유), `driverSpec` 의 `SyncMode` 누락 수정. `launchopt` 는 소유로 편입(호출자는 `nodeconfig.Argv` 뿐), 디렉터리 유지. **P4.3 dsl 완료 2026-08-28** — `testspec`(문법·해석기 1,432줄)과 `internal/testhelper`(액션·어세션·리더 2,129줄) 분리. `Registry` 에 `Reader` 추가로 문법이 액션 파일을 부르던 고리(`readerFor`) 제거, `NewRegistry()` 빈 레지스트리 + `testhelper.Register`. `testspec→testhelper` import 0 |
| **P4.x** | **`preflight` 재정의** (결정 2026-08-28) — 계획 자기모순 검사에서 **현재 vs 목표 비교**로: 타깃의 현 체인 구성을 분석하고, 정상 동작 여부를 inspector 로 확인하고, 다음 테스트가 요구하는 구성과 비교해 "그대로 사용 / N번 서버만 재구성 / 전체 재구성" 을 답한다(연속 테스트의 재구성 비용 제거 — 설정 파일 비교만으로는 부족: 파일이 같아도 노드가 비정상일 수 있다). 기존 계획 검사는 각 빌더로(포트→resource, genesis 포크→genesis 빌더, netid→config 빌더) | P4, P3 | 동일 구성 연속 테스트에서 재구성 스킵 라이브 · 부분 변경 시 해당 노드만 재구성 | ☑ **완료 2026-08-28** — `core/preflight` = `Have`/`Want`/`Compare`/`Check`/`Decision`(reuse·rebuild-nodes·rebuild-all·compose, 이유 포함), 의존 `node` 뿐 · `chainsetup.Workspace.Have/Compare` + liveness(pid 는 노드의 머신, RPC head 는 노드 주소) · `app.RunSuite` 가 ChainUp 전에 물어 reuse 는 건너뛰고 rebuild-nodes 는 `ChainRestart` 만 · 옛 계획 검사는 `upgrade.NetworkPlan.validate` 로(빌더 함수 호출) · 표 테스트 9 + liveness 3 |
| **P6** | `chainsetup`·`testengine` — 남는 것은 순서뿐. 6,593 → 2,000줄 이하 | P5 | `setup_bridge.go` 소멸 · `testengine→chainsetup` 엣지 소멸 | ☒ **닫는다 2026-09-08 — 줄 수 목표는 근거를 잃었고, 남은 게이트는 좁히기로 달성되지 않는다.**\n\n**실측**: `chainsetup` 4,652줄 · `testengine` 2,349줄, 합 7,001. 그중 **주석이 1,547줄(22%)** 이고 공백 451, 실제 코드는 5,003 이다. 함수·메서드가 213개라 평균 20줄 남짓 — 비대한 함수가 몰린 구조가 아니다. 가장 큰 것이 `netUpFrom` 133 · `RunSuite` 132 · `Run` 109 · `Allocate` 99 다.\n\n**목표 2,000 은 P5 시점의 6,593 에서 잡은 것인데, 그 뒤 이 두 패키지가 하는 일이 늘었다**(청사진 N1~N6, N9 순서 선언, NM6, 원격 경로). 줄이 는 것은 기능이 는 것이지 부풀어서가 아니다. 그리고 줄 수를 목표로 삼으면 **가장 먼저 지워지는 것이 그 22%의 주석**인데, 이 세션에서 판단을 바꾼 것이 바로 그 주석들이었다(`derive quorum` 이 노드 표를 안 읽는 이유, `genesis` 가 키셋을 요구하지 않는 이유).\n\n**게이트 둘 중 하나는 충족**이다 — `setup_bridge.go` 는 없다. 남은 `testengine→chainsetup` 엣지는 비테스트 4파일이고, **층 위반이 아니다**(둘 다 L4). 노드 수명(start/stop/restart/swap)을 인터페이스로 좁혀 보면 **4파일 중 1개**(`nodegate.go`)만 import 가 사라지고 `suite.go` 에는 `ChainUp`·`Open`·`WantOf`·`NetworkStatus`·`ChainStop`·`ChainEndpoints` 등 11개가 남는다 — **패키지 엣지는 그대로다.** 인터페이스가 12개 메서드라면 그것은 요구를 좁힌 것이 아니라 chainsetup 을 다른 이름으로 부르는 것이다.\n\n**엣지를 실제로 없애는 길은 하나뿐이다**: 조립을 testengine 밖으로 내보내 엔진이 망을 *받게* 하는 것. 그것은 좁히기가 아니라 재설계이고, 근거가 생기면 그때 별도 항목으로 세운다. 지금은 열어 둘 이유가 없다 |
| **P7** | DSL 케이스 4종 — go-wemix · wemix→wbft · wbft 단독 · stablenet | P6 | 러너에 `if chain ==` 0건 | ☑ **완료 2026-08-28** — 당시 `tests/cases/env/` 선언 4개 + 케이스 4개(지금은 `tests/tc/`, env 는 정의서 인라인) · 문법 `env.upgrade`(schema·strict·lowering, `binaries` 는 producer/validator 역할) · 실행기 `app.RunSuite` 가 선언의 모양으로 조립기 선택(`upgrade` → `upgrade.Handoff`, 아니면 `ChainUp`) · 표면 `run --workspace-dir` · `validate` 가 env 를 풀고 env 파일을 선언으로 검증 · 러너 `if chain ==` 0건 · **라이브 4/4 완결 2026-08-31**(gstable·go-wbft·go-wemix 빌드; 핸드오프는 후계 검증자의 블록 21 봉인까지) — 그 과정에서 gwemix 0.10.x 의 `etcd.members` 모양 변화가 깨뜨린 verify 파싱을 `EtcdState` 필드 제거로 해소 |
| **P8** | `test-helper` — 액션 1,541줄 + testkit + tests 공통부 취합 | P7 | 파서가 액션을 모르고 액션이 문법을 모른다 | ☑ **완료 2026-09-07 (#357).** 남은 것은 문법 갭이 아니라 **낡은 기록**이었다 — 막혔다던 19건 중 15건에 이미 스펙이 있었고 122개 전부 `validate` 를 통과한다. 문서를 고치고 `TestSpecDoc_BlockedCasesHaveNoSpec` 이 그 주장을 검사하게 했다. 진짜 남은 8건은 이유가 유효하다(SDK 가드 2 · 조작자 공급 키 2 · 다른 빌드가 필요한 4) |
| **F1(최종)** | **파일 영속·복구 시스템** (사용자 결정 2026-08-28: 모든 작업의 맨 마지막) — chainbench 프로세스 장애로 중단됐을 때 재실행하여 이전 진행 상황을 복구하고 서버 상태를 재확인. `Inventory` 등 메모리 정본의 파일 저장이 이때 들어온다. 그 전까지는 기존 기록에서 `Adopt` 으로 파생(사본 금지 원칙) | P8 | **설계안 2026-08-28**: `docs/dev/architecture/f1-recovery.md` — 요청 기록(`workspace.json.request`) · `net resume`(잠금 인수 → 생사 대조 → 첫 미완 단계부터 → 재확인) · 세트 잠금(인벤토리 파일 없음) · 주인 없는 프로세스 입양. §4 는 제안대로 결정 → ☑ **완료 2026-08-28**: `State.Request` 기록 · `net resume`(reconcile → 첫 미완 단계부터 → 죽은 노드 재기동) · `session.AcquireLock` + 세트 잠금(`~/.chainbench/<set>.lock`) · 우리 argv 프로세스 입양 · 단위 6건 + gstable 라이브(kill -9 → resume) | ☑ |

### 1j. 사용자 주도 통폐합 (2026-08-31 ~ 2026-09-20 — **닫혔다**)

> **완료로 닫혔다 (2026-09-20).** 정본이던 `docs/dev/architecture/consolidation-plan.md` (2026-09-24 삭제) 은
> `[이력]` 이고, 같은 날 그 선행 계획 `module-plan.md` (2026-09-24 삭제) 도 닫혔다. 진단("작은 형제 24개")이
> 7개로 해소됐고 남은 3건은 각각 import 순환·코드가 명시적으로 반대·층 역행이다.
> **다음 리팩토링은 이 절도 그 문서들도 이어받지 않는다** — 백지에서 다시 설계한다.
> 아래는 그때의 기록이고, 인용하는 수치는 전부 2026-09-12 이전이다.

§1i 의 표는 F1 까지 전부 끝났다. 이후는 **사용자가 주도**한다. 확정된 방향:
모듈을 관심사 단위로 통폐합(internal 55 → 약 20, R1~R5), 그다음 표면 재정리
(`net` 폐기 → `chain` 그룹, 구성 6단계 사전, CLI↔DSL 1:1). testengine 은
① chainsetup 수행 ② pre hook ③ test(interpreter 내장) ④ post hook 의 4단계가
되고, testengine → chainsetup 의존은 의도적으로 부활한다(P6.1 게이트 대체).
아래의 예전 후보 목록은 이 계획 안으로 흡수됐다(34건 이관 = R5, verbs 질문 =
표면 재정리에서 함께).

**R1 완료 (2026-08-31, PR #325 — 9 relocation, internal 46).** 소형 흡수 8건 계획 중,
측정된 층 그래프와 대조해 arch-안전한 것만 실행했다: topology→node · launchopt·config→nodeconfig ·
netid→resource · consensus·capability→registry · obs·logs→collector · netreg→session. 상세·근거는
정본(`consolidation-plan.md` (2026-09-24 삭제) §R1 실행 결과). **~~다음은 R2(DSL 분리).~~ 2026-09-12 정정: R2·R4·R5 는 이미 끝났다** — R2 는 배치가 계획과 다르지만 게이트를 만족한다(`dsl` fanOut 0). 남은 것은 R3 의 3분의 1 이고, 그 3건에 반대 근거가 있다. `consolidation-plan.md` §5 를 본다.

**R1 에서 갈라져 나온 후속 작업 (별도 트랙):**

- ☑ **validatorset 홈 결정 — 완료 (2026-09-12).** 정본은 아래 §1s D 의 `validatorset 홈 결정` 항목이다(패밀리가 자기 지식을 소유하도록 `registry.RingAccountReader` 를 더했다). 이 줄은 §1j 의 원본 표기였고 닫힌 것을 미착수로 보이고 있었다.
  **(원래 기록)** `core/node`(L0)로 넣으려던 계획은 층 위반(validatorset 이
  `chains/all`·`registry` import). 지금은 제자리(독립 L3)에 둔다. 로스터 계산(키+registry→검증자)의
  올바른 소유 모듈을 후속에서 정한다. 소비자는 `cmd` 하나뿐.
- ☑ **health 를 inspector 조합 레이어로 재배선 — 철회 (2026-09-12).** 공통 기반이 없어 재배선의 값이 없다고 판정하고 래칫(`arch.TestInspectorStaysOnTheEnvironmentSide`)으로 경계를 고정했다. 상세는 §1s G-D. **(원래 기록)** health→inspector 흡수는 하지 않기로 결정.
  방향: `inspector` 는 atomic 실사 프리미티브(L1, "판단 없음")로 두고, `health`(블록 전진 *판정*)는
  그 atomic 들을 **조합하는 inspector 위 레이어**로 제공한다(현재 health 는 obs/rpc 를 직접 쓴다 →
  inspector 프리미티브를 쓰도록 재배선). atomic ↔ 조합의 층 분리.
- ☑ **hardfork 는 통폐합 대상 아님**(결정으로 닫힘) — `hardfork`(바이너리 swap)와 `consensus/upgrade`(합의-패밀리
  handoff)는 의도적으로 다른 모델이라 별개로 유지. R1 표의 `hardfork→genesis` 항목은 폐기.

이전 기록 (2026-08-31 이전 후보):

1. **레거시 34건 이관 + 스택 은퇴** — `tests/specs/README.md` 잔여 표의 문법 갭을
   묶음별로(자산·부정기대·fee-delegation 0x16·EIP-7702·비동기 제출·토폴로지 파생
   quorum·delayed-boho 크로스오버) 확장하고, 끝나면 `testkit`·`testrun`·`test` 명령·
   MCP `chainbench_test` 를 한 번에 은퇴(T7.11 완결). 이제 네 갈래가 전부 라이브로
   돌므로 이관분의 라이브 확인도 곧바로 된다.
2. **열린 질문 7** — `verbs_*.go`(1,384줄)의 자리. `app` 이전이면 `chainsetup` 이
   2,000줄 아래로 가지만 "CLI 는 core 직접" 원칙과 충돌 — 사용자 결정.
3. **열린 질문 2·5** — `session`(1,594줄) 경계 재측정, `netreg`(161줄, 소비자 mcp
   하나) 개명 또는 흡수. 소품.


## 1k. 체인 실행·테스트 증적 완결 (사용자 확인 2026-09-02)

> 제품 방향은 [[chainbench-system-direction]](chainbench-system-direction.md), 현재 코드와의 차이와
> 완료 조건은 `refactoring-follow-up-handoff-2026-09-02` (2026-09-24 삭제)를
> 따른다. 이 절만 작업 순서와 상태를 소유한다. 기존 완료 기능을 다시 만들지 말고 각 항목의 첫 단계에서
> 코드·테스트·CLI를 재측정해 남은 차이만 구현한다.

| # | 작업 | 선행 | 핵심 게이트 | 상태 |
|---|---|---|---|---|
| **E0** | **현행 재측정과 계약 고정** — key/resource/node/genesis/config/process/monitor/test/report의 producer·consumer, CLI·DSL·MCP 입력, local/remote/Docker 경로를 AST와 테스트로 재확인 | — | 중복 owner·직접 import·기존 완료 기능 목록, 수정 대상과 비대상 파일 확정 | ☑ |
| **E0A** | **session artifact 계약 선행** — `environments/<env-id>` 정본 + `tests/<NNN>_<name>`별 env-ref·사용자료 snapshot/reference + root report의 2축 schema와 owner 고정 | E0 | testengine=verdict, session=영속, report builder=집계 · secret 정책 · 원자 write/readback | ☑ |
| **E1** | **resolved producer → genesis** — validator count 기반 첫 N개 재선택을 제거하고 실제 BP identity/address/BLS를 사용 | E0A | `EN,BP,PN,BP` topology genesis readback 일치 · 누락/중복 no-write · 결정성 | ☑ |
| **E2** | **자료 재사용 무결성** — asset별 owner를 유지하며 local/SSH/Docker의 binary·genesis·config·key·contract를 checksum으로 비교하고 결과 기록 | E0A | 같은 내용 재전송 0 · 같은 이름/다른 내용 오재사용 0 · secret 출력 0 | ☑ |
| **E3** | **config·command 증적** — `nodeconfig` Command Builder, `config-<test-purpose>` fixture, override 우선순위, config readback, 노드별 argv·binary·checksum 기록 | E0A | node override 격리 · config/argv 동일 사실 · 변경 전후 revision 보존 | ☑ |
| **E4** | **process와 개별 node control 정합** — Direct/Launcher/chainsetup start 중복을 측정하고 PID·실행 command·start/stop/restart·binary/config 교체를 하나의 ledger에 연결 | E2, E3 | 개별/전체 제어 · 실제 PID/command 일치 · 부분 실패 cleanup · orphan 0 · 세 환경 동등 | ☑ |
| **E5** | **DSL syntax + semantic/capability 사전 검사** — schema/parser drift, selector/wait timeout, PN 제약과 chain·binary별 role/flag/RPC/metric/action/assertion/upgrade 지원을 모든 write 전에 검증 | E0A | unsupported는 no-write · applicableChains는 SKIP · CLI/MCP 판정 동등 | ☑ |
| **E6** | **`nodemonitor` 실행 허가** — inspector/preflight/health/collector를 복제하지 않고 READY/WAITABLE/RESTARTABLE/FATAL로 조합, `MaxNodeMonitorTimeout` 적용 | E1~E5 | 재사용 전·각 테스트 전 gate · 제한 재시작 · 파괴적 자동 복구 0 · 판정 증적 | ☑ **stage a**(`internal/nodemonitor` L4): `Verdict`·`Classify`(순수, worst-first)·`Gate`(관측→분류→제한복구 루프, `Observer`/`Restarter`/`Clock`/`EvidenceSink` 주입). `process.FailureMode` 재사용, 파괴 상태(chainId 불일치·fork·EtcdStale·QuorumLost)=FATAL 자동복구 0. **stage b**(testengine 배선): `health.Run`+recorded pid→Facts(순수 `factsFromReport`), `chainsetup.ChainRestart` 재시작, `composeWorkspace`가 compose/reuse 후 1회·`PreSpec` 훅이 각 테스트 전 게이트. 관측·재시작은 재구현 없이 기존 함수 재사용(사용자 결정). 고아 `process.BringUp` 상한재시도는 모양이 달라 미재사용→1k-Z2 이월. **라이브 검증(GSTABLE_BIN) 이월** — CI는 순수 매핑·게이트·엔진루프 단위테스트로 green | ☑ |
| **E7** | **동적 테스트와 contract** — 노드별 제어, partition/heal, binary/config 교체, 동적 `save/$ref` 주소와 deployer+nonce 결정적 주소 | E4, E5, E6 | per-node mixed binary · contract tx/receipt/address/checksum · 상태별 verdict | ☑ 기존(fault·partition·save/$ref·deploy·call)은 재구현 없이 조합. **신규**: ① `createAddress` 리더 — `accounts.CreateAddress`(SDK `tx.CreateAddress` 래핑, canonical 벡터 검증), nonce 생략 시 현재 nonce → deployContract 예상=실제 assert. ② `swapNode` action — 노드별 binary/config 교체(`chainsetup.NodeSwap`, 같은 datadir/genesis, **E4 recordSwap로 revision 보존**, config는 `writeNodeConfig` 공유·config-`<purpose>` fixture provenance). `interp.NodeSwapper` 선택 인터페이스(코어 미변경). ③ `contractChecksum` 리더 — bytecode/runtime code sha256(`filestore.Hash` 재사용). 기대 revert=PASS 등 verdict는 기존(`checkTxOutcome`). **swapNode config launch·라이브 시나리오는 사용자 환경 이월**, CI는 매핑·플러밍·revision·config render 단위테스트로 green | ☑ |
| **E8** | **최종 증적 집계와 report** — E0A schema에 append-only logs, chainstate JSONL, assertion provenance, 테스트별 result를 채우고 root report 생성 | E0A, E3, E6, E7 | 실패 자료 수집 · remote reconnect · 전체 종료 후 report · secret 원문 0 | ☑ 기존(session schema·`AssertResult.Provenance`·report.Build/Summary·chainstate_sink)은 확장. **E8-1** `session.Scrub` — 증적 write seam(`writeJSON`+`Spec`)에서 키·비밀번호 마스킹, hash/주소는 보존. workspace.json(기능)은 제외. **E8-2** FAIL/BLOCKED 시 엔진 `OnFail` 훅이 `collectFailureData`로 health.Run(RPC/peer/block)·process 레저(pid/command)·노드 로그 tail을 테스트별 `observations/`에 수집(`TestRecord.Observation`, 스크럽 적용). **E8-3** report가 observations/*를 링크, RunSuite가 bus로 chainstate.jsonl을 headless 경로에서도 기록. **E8-4** `collector.ReconnectingLogReader`(백오프 재시도, 항상 적용) + `resource.Access.Runner` 노출 + `chainsetup.NetworkRunner` + RunSuite가 remote면 `RemoteLogReader` 주입. **원격 SSH read·chainstate 라이브는 사용자 환경 이월**, CI는 스크럽·OnFail·tail·reconnect·report 링크 단위테스트로 green | ☑ |
| **E9** | **표면·환경 동등성** — CLI 단계, DSL 자동 구성, MCP 도구와 local/remote/Docker simulation을 같은 시나리오로 검증 | E1~E8 | 의미·기본값·오류·결과 동등 · 일반 mixed-binary와 `consensus/upgrade` handoff 구분 | ☑ **E9-1** MCP↔CLI GAP 해소: `chainbench_run`이 rpc 없으면 compose(`app.RunSuite`, `RunSuiteIn.SpecContent`로 인라인 스펙)·있으면 attach. `chainbench_hardfork`→`app.HardforkPlan/Execute`(execute 기본 false). `chainbench_upgrade`→신설 `app.UpgradeRun`(handoff orchestration을 `upgrade.Handoff.Run`으로 추출, CLI `upgrade run`과 공유). **E9-2** 구조적 parity: 두 표면이 미설정 시 zero-value를 넘겨 `compositionOf`가 정본 기본값 단일 소스(validators 4·presets/keys 고정), run 모드 선택 검증. mixed-binary vs handoff 구분·컴포즈 기본값/선택/가드는 기존 `compose_internal_test`가 커버. 세 표면은 arch(`TestMCPGoesThroughApp`)가 app 경유로 고정. **행위적 CLI-vs-MCP diff는 cmd=main이라 불가**, local/remote/Docker 실제 실행은 라이브 이월(환경 분기 단일 지점 arch 고정) | ☑ |

E1·E2·E3·E5는 E0A에서 저장 계약과 수정 파일을 분리한 뒤 병렬 진행할 수 있다. E4는 process와 chainsetup,
E6는 inspector/preflight/health/collector, E7은 DSL/testhelper/upgrade, E8은 session/collector/report
영역과 충돌하므로 해당 owner 작업과 동시에 수정하지 않는다.

각 작업은 코드 변경 전에 입력·출력 타입, owner, 허용 import, 외부 상태 변경 시점과 rollback을 검토받는다.
구현 후에는 단위·거부·readback 테스트, CLI JSON 예시, local 검증과 가능한 remote/Docker 라이브 결과,
남은 wrapper·alias·직접 import를 보고한다.

### 1k-Z. 레거시 네이밍 정리 (추후, E-series 이후 · 비차단) — ☑ 완료

**완료(2026-09-03):** R1에서 흡수된 파일이 달고 있던 leftover `// Package <옛>` doc 주석
6곳(netreg→session, probe/logs→collector, config/launchopt→nodeconfig, event.go의 중복
`Package collector`)을 일반 섹션 주석으로 바꿔 패키지마다 정본 doc 하나만 남겼다. 폐기된
옛-패키지 에러 접두 정리: netreg `state:`→`session:`, run store `obs:`→`collector:`. 살아있는
하위 개념 접두 `logs:`(로그 검색)·`launchopt:`(런치 옵션)는 유지(폐기 패키지명이 아니라 실제
개념). 테스트가 이 문자열을 검사하지 않고 errors.Is 무영향 → 동작·공개 계약 변화 0. 남은 옛-이름
언급은 의도적 provenance 주석("Formerly the standalone X package…")뿐. build·test·arch·lint 통과.

아래는 착수 전 기록:

모듈 통폐합(R1) 과정에서 흡수된 파일·식별자가 **옛 이름을 그대로 달고 있다.** 동작에는
문제 없으나 "어디를 봐야 하는가"를 흐린다. E-series를 막지 않는 **순수 개명 리팩토링**으로
뒤에 한 번에 정리한다(파일·심볼 rename + 주석의 "옛 core/X" 표기 정돈, 동작 변화 0).

정리 대상(측정으로 확정):

- `netreg.go` — `core/netreg` 가 session 으로 합쳐졌는데 파일명·에러
  접두("state: …")·주석이 옛 `core/state` 어휘를 유지(E0A 에서 관찰). session 어휘로 통일.
- `core/collector` 안의 "옛 `core/obs`"·"옛 `core/logs`", `core/nodeconfig` 안의 "옛
  `core/launchopt`"·"옛 `core/config`" 등 §3 표가 나열한 흡수 이력 표기 — 코드 심볼은 이미
  새 위치이나 파일/주석이 옛 이름을 인용하는 곳을 정돈.
- 판정: **파일 이동/개명만으로 완료로 보지 않는다** — 에러 접두·doc 주석·테스트 이름까지
  새 owner 어휘로 맞춘 뒤 완료.

게이트: `go build`·`go test`·arch 테스트 통과, 동작·공개 계약 변화 0, 남은 옛-이름 인용 목록 보고.

### 1k-Z2. E4 에서 미룬 항목 (추후, 비차단)

E4(launch+record 통일 · 스왑 revision 보존)에서 근거를 대고 미룬 둘.

- **핸드오프 레저 기록** — `consensus/upgrade` 핸드오프는 workspace 가 없어(compose.go
  핸드오프 경로) 영속 레저(process.json)를 읽는 소비자가 없다. teardown 은 레저가 아니라
  반환 NodeSet 으로 `StopNodeSet` 을 부른다. `LaunchAndRecord` 는 nil 레저를 받아 이미
  통과한다. **핸드오프가 workspace 를 얻는 작업과 함께** LaunchOptions 에 선택적 Ledger 를
  넣고 teardown 에서 Clear 한다. 그 전까지는 SIGKILL 엣지의 orphan 복구만을 위해 배선하지
  않는다.
- ☑ **죽은 process 오케스트레이션 은퇴 (완료)** — `Launcher`(bringup.go: impl/NewLauncher/
  BringUp/gate/Teardown/Deps/Result)·`Controller`(controller.go)·`Manager`(process.go)·
  `Direct`(direct.go)·`PlanOf`(launcher_plan.go)·`gate.go`(JoinGap/JoinWindow/Classify)·
  `StopNode`/`RelaunchNode`·`Proc.IsRemote` 를 그 테스트와 함께 제거(~3200줄). 생존 프리미티브만
  남김: `FailureMode`(→failuremode.go, nodemonitor 소비)·`NodeConfig`(→nodeconfig.go, chainsetup
  소비)·`StopNodeSet`(핸드오프 teardown)·`Proc`/`Alive`(레저·inspector)·`Plan`(데이터 캐리어).
  no-branch-on-kind 래칫에서 process 항목 제거. build·vet·test·lint 통과, 동작 변화 0.

## 1l. 표면 단일 진입 (사용자 결정 2026-09-05)

> **결정**: CLI·MCP·DSL 은 모두 `app` 을 통해 기능에 닿는다. 세 표면이 같은 층을 지나야
> 같은 경험을 준다. `app` 이 커지더라도 규칙이 하나인 편이 낫다는 판단이다.
> 이 결정은 [[architecture-v2]](architecture/architecture-v2.md) §2 의 비대칭 규칙("CLI 는 core
> 직접, MCP 는 app 경유", 2026-08-25)을 대체한다. S 절의 **S5·S6 은 폐기**한다. 게이트가
> "cmd 는 app 만 import" 였는데, 지키려던 성질은 import 목록이 아니라 표면 사이의 동등성이다.
>
> **근거가 된 실측(2026-09-05, AST 전수)**: 세 표면의 등록 지점 **157개**(CLI 58 · MCP 54 ·
> DSL 45 = 액션 18 + 어서션 27)를 파싱해 실제로 닿는 패키지를 따라갔다. app 을 지나지 않고
> 그 아래에 닿는 항목이 **109개**다(CLI 40 · MCP 24 · DSL 45). app 을 한 번도 부르지 않는
> 항목만 세면 102개다.
>
> 표면 사이 비교는 이름이 겹쳐 자동으로 묶으면 틀린다(`new` 는 `chain` 밑에도 `keyring`
> 밑에도 있고, `run` 은 스위트 실행과 `upgrade run` 둘 다이다). 그래서 **손으로 확인한 32쌍**만
> 쓴다. 그중 **22쌍이 서로 다른 경로로 간다.** 양쪽이 모두 app 을 지나는 것은 `chain show`·
> `hardfork`·`resource plan`·`resource pool` **4쌍뿐이고**, 나머지 6쌍은 양쪽이 똑같이 app 을
> 건너뛴다(경로는 같지만 구현은 각자다).
>
> 갈라진 예를 들면, `report` 는 CLI 가 `core/report`+`core/session` 을 직접 부르는데 MCP 는 app
> 으로 간다. `build`·`config`·`health`·`logs`·`resume`·`status`·`new` 는 CLI 가 `chainsetup` 을
> 직접 부르고 MCP `chain_*` 는 전부 app 이다. keyring 다섯 동사는 CLI 가
> `core/keyring/operation` 을 부르고 MCP 는 app 을 부르는데, app 이 그 모듈을 얇게 감싼 것이라
> 구현은 같고 경로만 다르다. `send`·`deploy`·`faucet` 은 CLI·MCP·DSL 세 곳이 각각 조립한다.
>
> **선례**: Helm 은 `pkg/action` 에 동사마다 액션을 하나씩 두고 CLI 와 SDK 가 같은 것을 부른다
> (`pkg/action/doc.go`: "Actions approximately match the command line invocations that the Helm
> client uses"). CLI 는 `cmd/helm` 이 아니라 `pkg/cmd` 에 있고, `cmd/helm/helm.go` 는 50줄짜리
> `package main` 으로 `helmcmd.NewRootCmd` 만 부른다. U1 이 따르려는 모양이 이것이다.
>
> 반대 방향의 증거도 같은 저장소에 있다. 차트 의존성을 갱신하는 `downloader.Manager` 는
> `pkg/cmd` 의 다섯 파일(`install`·`upgrade`·`package`·`dependency_update`·`dependency_build`)에
> 나오고 **`pkg/action` 에는 한 번도 나오지 않는다.** 옵션 `DependencyUpdate` 는 액션이
> 선언하는데 그 동작은 CLI 가 구현한다. `action.Install` 을 직접 쓰는 소비자가 그 옵션을 켜도
> 아무 일도 일어나지 않는다는 뜻이다. **표면이 층 위에 앉는 것을 허용하면 갈라진다.**
>
> Docker 는 CLI 가 엔진에 닿는 통로를 API 클라이언트 하나로 두었다(`cli/command/**` 의 각
> 동사가 `dockerCLI.Client()` 로 시작한다). 데몬이 별도 프로세스라 우회할 방법 자체가 없다.
>
> **규칙**: 기능 하나에 `app` 진입점 하나를 둔다. `core` 모듈은 기구를 갖고, `app` 은 동사를
> 갖고, 표면은 바인딩과 렌더링만 한다. 표면은 `app` 을 건너뛰지 않는다.
> **게이트는 import 규칙이 아니라 동등성 테스트다.** 같은 입력에 두 표면이 같은 결과를 내는
> 것을 기능마다 증명한다. 그러려면 CLI 가 import 가능해야 하므로 U1 이 앞에 온다.

| # | 작업 | 선행 | 핵심 게이트 | 상태 |
|---|---|---|---|---|
| **U0** | **규칙 확정과 측정 고정** | — | 도구가 157 항목·우회 109 를 재현 · 라체트가 표면별 상한을 고정 · 동등성 테스트 본보기 1건 · 문서 사이 모순 0 | ☑ **2026-09-05.** 문서 쪽은 #349 에서 끝냈다(`architecture-v2` §2 개정, S5·S6 폐기, 표면 통일 설계에 대체 표시). 코드 쪽은 이번에 붙였다. 인벤토리 걷기를 `internal/arch/surface.go` 로 옮겨 **도구와 테스트가 같은 코드를 센다**(두 벌이면 숫자가 갈라진다). `TestSurfacesReachThroughApp` 이 표면별 예산(CLI 40 · MCP 24 · DSL 18 · DSLa 27)을 **양방향으로** 잡는다. 넘으면 빚이 늘었다고 막고, 밑돌면 작업하고도 천장을 안 내렸다고 막는다. 실제로 41 로 올리고 39 로 내려 둘 다 실패하는 것을 확인했다. **동등성 테스트 본보기**는 `cmd/chainbench/resourcecmd/parity_test.go` 의 `TestParity_ResourcePool` 이다. `resource pool` 은 CLI 와 MCP 가 이미 둘 다 `app.NetworkPool` 을 부르는 네 쌍 중 하나라 지금 통과한다. 렌더링이 아니라 **답**을 비교한다(양쪽 JSON 을 값으로 디코드). MCP 쪽에 `res.Slots++` 를 심어 실제로 어긋남을 잡는 것을 확인하고 되돌렸다 |
| **U1** | **CLI 를 import 가능한 패키지로** (U0 의 본보기가 `resourcecmd` 에서 돌아가는 것은 그 패키지가 이미 import 가능하기 때문이다. `package main` 에 남은 32개는 같은 테스트를 쓸 수 없다.) — `package main` 안에 cobra 명령 생성자가 **32개** 있어 그 명령들은 테스트에서 부를 수 없다. 표면 패키지로 옮기고 `package main` 은 배선만 남긴다(Helm 이 `cmd/helm` 을 `pkg/cmd` 로 옮긴 것과 같은 이유). 동등성 테스트가 가능해지는 전제다 | U0 | `package main` 의 명령 생성자 32 → 0 · 옮긴 그룹마다 표면 테스트 1건 이상 · E9 가 "cmd=main 이라 불가"로 이월한 CLI-vs-MCP diff 가 가능해짐 | ☑ **32 → 1 (2026-09-05).** `package main` 에는 이제 루트 하나만 남았다(149줄, `main.go`·`root.go`·`interrupt.go`). Helm 이 자기 `helm.go` 를 50줄 배선으로 두고 명령을 `pkg/cmd` 에 옮긴 것과 같은 모양이다. 표면 패키지는 여덟 개가 늘었다. `txcmd`(tx·contract) · `accountcmd`(account·faucet) · `catalogcmd`(chains·capabilities) · `nodecmd` · `lifecyclecmd`(status·stop·clean·verify·consensus) · `suitecmd`(run·validate·migrate-spec) · `reportcmd`(report·log) · `upgradecmd`(upgrade·genesis·run·hardfork). 그룹이 곧 패키지라는 기존 관례(`keyringcmd`·`chaincmd`·`resourcecmd`)를 그대로 따랐다.

**이동은 순수하다**: 라체트 109 불변, 명령 트리 불변.

**공유 심볼 둘을 정리했다.** `exitError` 는 `suitecmd` 가 만들고 `main` 이 읽으므로 표면과 프로세스 사이의 계약이다. `cmd/chainbench/exitcode` 패키지로 뺐다. `obsBus` 는 전역 `dashboardURL` 을 읽고 있었는데, `--dashboard` 는 루트의 persistent 플래그라 하위 명령이 상속한다. 명령에서 직접 읽게 바꿔 표면 패키지에서 공유 가변 상태를 없앴다.

**앞서는 쓸 수 없던 테스트 30건**이 붙었다. 그게 이 항목의 요점이다. 동등성 4건(`account state`·`contract call`·`chains`·`node rpc`)은 표면이 **무엇을 인쇄했는지가 아니라 체인에 무엇을 물었는지**를 비교한다. CLI 는 사람에게, MCP 는 프로그램에게 쓰므로 글자를 맞추면 서식을 고정하게 된다. 가짜 노드가 RPC 호출을 기록하고 두 목록을 견준다. 변이를 심어 실제로 잡는 것을 확인했다. 나머지는 거절 경로와 종료 코드 계약이다.

**e2e 도 실제 마운트 지점을 거치게 바꿨다.** 잎사귀 생성자를 직접 부르면 그 명령이 루트에 제대로 붙어 있는지는 시험하지 못한다. `newRootCmd()` 에 `upgrade run` 을 붙여 부른다.

덤으로 `resolve.go` 의 `remoteDriver` 를 걷어냈다. `remote` 명령군 폐기(#346) 이후 자기 테스트 말고 부르는 데가 없었다. |
| **U2** | **조합 계열 이관** — `chaincmd` 의 명령 19개가 `chainsetup` 을 직접 불렀다 | U1 | 기능별 CLI==MCP 동등성 테스트 · 라체트 감소 · `net up` 3체인 회귀 | ☑ **2026-09-05. CLI 40 → 31.** `chaincmd` 가 `chainsetup`·`resource`·`core/node` 를 직접 부르던 것을 전부 `app` 경유로 돌렸다. app 이 이미 대부분을 얇게 감싸고 있어서(`ChainStatus` 는 `chainsetupmod.ChainStatus` 로 넘기는 한 줄) 빠진 진입점만 채웠다. `ChainEnodes` · `DefaultWorkspaceDir` · `State` · `NodeSet` · `TargetSpec`/`ParseTarget`.

**기본 워크스페이스를 app 으로 올린 것이 이 항목의 알맹이다.** 두 표면이 각자 계산하면 한쪽이 다른 쪽이 못 찾는 곳에 구성한다. `app.DefaultWorkspaceDir` 하나만 쓰게 했다.

**동등성 테스트 3건.** 인쇄한 글자가 아니라 **디스크에 남은 워크스페이스**를 비교한다. 구성은 디스크에 대한 부수효과이고 그 상태가 곧 답이다. 워크스페이스 경로와 시각만 정규화하고 나머지는 그대로 견준다.

**여기서 제 테스트 결함을 둘 잡았다.** 처음에는 디렉터리를 훑어 "JSON 파일"을 집었는데 알파벳순으로 `process.json`(양쪽 다 `{"procs": []}`)이 걸려서, 빈 문서 둘을 비교하며 통과하고 있었다. `workspace.json` 을 이름으로 지정하고 가드를 실질화했다. status 쪽은 "stablenet" 부분 문자열을 찾았는데 그 낱말이 step 의 `detail` 안에도 있어서, 도구가 chain 필드를 아예 안 내보내도 통과했다. 도구의 JSON 을 파싱해 구조로 견주게 고쳤다. 셋 다 변이를 심어 실제로 잡는 것을 확인했다.

`internal/app/net.go` 머리말의 "CLI 는 모듈을 직접 부르고 여기를 지나지 않는다" 도 고쳤다. U0 에서 뒤집힌 규칙의 잔재였다 |
| **U3** | **keyring 계열 이관** | U1 | 동등성 테스트 · keyring 라이브 스위트 통과 · `operation` 단위 테스트는 app 을 지나지 않음 | ☑ **2026-09-05. CLI 31 → 24.** 표면이 `core/keyring/operation`·`keyring`·`store`·`derive`·`resource` 를 직접 부르던 것을 app 경유로 돌렸다.

**중복 구현을 하나 지운 것이 알맹이다.** `keyflags.go` 가 플래그로 `keyring.Source` 를 조립하고 있었는데, `operation.ImportIn.source` 에 같은 읽기가 이미 있었다. "정확히 하나의 출처"나 "니모닉 없는 hd 옵션"이 무슨 뜻인지를 두 곳이 각자 정하고 있었다는 뜻이다. 모듈 쪽을 `operation.KeyRef`+`ResolveKey` 로 승격하고 표면은 **설명만** 하게 했다. `SaveKey`·`GenerateKey` 도 같은 이유로 모듈로 올렸다. 비밀번호는 값이 아니라 씨앗 함수로 넘긴다. `--password-once` 는 물어보는데, 쓰지도 않을 비밀번호를 미리 묻는 건 사용자를 대하는 좋은 방식이 아니다.

**단위 테스트가 진짜 결함을 잡았다.** CLI 의 `--hd-coin-type` 기본값이 60 이라, 모듈의 "니모닉 없이 hd 옵션을 줬다" 거절에 항상 걸렸다. 플래그 기본값은 사용자가 지정한 것이 아니므로 0 으로 바꾸고 뜻은 도움말에 적었다.

**표면 사이 기능 격차도 찾았다.** MCP `keyring_import` 에 `privateKey` 가 없었다(CLI 에는 `--private-key` 가 있다). app 이 이미 그 필드를 갖고 있어 스키마 한 줄로 닫았다.

동등성 3건(list·show·import). 셋 다 변이로 확인했다 |
| **U4** | **테스트 동사 계열 신설** | U1 | CLI·MCP 동등성 · `mcpImportAllowed` 에서 `accounts`·`core/rpc` 제거 | ☑ **2026-09-05. CLI 24 → 17, MCP 24 → 18.** `send`·`deploy`·`call`·`faucet`·`wait`·`state` 의 유스케이스가 어느 모듈에도 없이 **CLI·MCP·DSL 세 곳에 각각** 쓰여 있었다. `internal/app/chainops.go` 로 모아 CLI 와 MCP 를 그리로 돌렸다. `HexBytes`·`Wei` 도 여기 있다. 한쪽이 `0x` 접두사를 요구하고 다른 쪽이 안 하는 차이는 아무도 테스트할 생각을 못 한다.

`accountcmd/provider.go` 는 소멸했다(`app.ChainRef.provider` 가 같은 일을 한다). MCP 의 `openWalletFromArgs`·`hexBytes`·`weiArg` 도 소멸했다. **라체트에서 `internal/accounts` 가 빠졌다** — "계정 동사 이관과 함께 사라진다"고 예고돼 있던 항목이고, 실제로 그렇게 됐다.

동등성 4건인데 **서명된 원본 트랜잭션을 바이트로 비교**한다. nonce·gas·value·calldata·서명을 한 번에 덮으므로, 여기서 일치하면 두 표면이 `--value` 나 `--data` 를 다르게 읽고 있을 수 없다. 넷 다 변이로 확인했다.

**잔여 없음(정정 2026-09-06).** "DSL 도 옮긴다"고 적었는데 U7 에서 불가로 판정됐다 — `testhelper` 는 L3 이고 app 은 L5 이며, import 순환이다. DSL 액션은 표면이 아니라 언어의 어휘다 |
| **U5** | **실행·보고 계열 이관** | U2 | 동등성 테스트 · `run.go` 에서 흐름 조립 소멸 · 라이브 스위트 회귀 | ☑ **2026-09-05. CLI 17 → 12, MCP 18 → 16.** `run`·`report`·`validate`·`log`·`verify` 를 app 경유로.

**`app.Report` 가 산문을 돌려주고 있었다.** MCP 는 그걸 그대로 냈고 CLI 는 같은 읽기를 다시 해서 표로 그렸다. 렌더링하는 층은 표면이 우회할 수밖에 없는 층이다. 이제 보고서를 돌려주고 표면이 각자 그린다.

**`VerifyNetworkIn` 주석에 "노드 집합 해석은 표면의 몫"이라고 적혀 있었고, 실제로 두 표면이 각자 했다.** 워크스페이스가 어느 엔드포인트를 뜻하는지, 붙인 집합을 뭐라 부르는지를 두 번 정하고 있었다. `app.ResolveNodes` 하나로 합쳤다.

**레이어 검사가 제 실수를 잡았다.** 대시보드 이벤트 배선을 app 에 넣었더니 `L5 app → L6 dashboard` 로 걸렸다. 표면이 다른 표면을 부르는 것은 우회가 아니므로, 배선은 `dashboard.Stream` 으로 표면 층에 두고 라체트도 L6 끼리의 호출은 세지 않게 고쳤다. 안 그러면 옳은 일을 하고 숫자가 나빠진다.

동등성 2건(report·log). 둘 다 변이로 확인했다.

**잔여**: `verify` 는 `dashboard` 만 부른다(같은 층이라 규칙 위반이 아니다) |
| **U6** | **조회 계열 이관** | U2 | 동등성 테스트 · ReadOnly 선언이 세 표면에 동일 노출 | ☑ **2026-09-05. CLI 12 → 0, MCP 15 → 0.** **두 표면 112개 등록이 전부 app 을 지난다.**

CLI 쪽은 `chains`·`capabilities`·`consensus`·`node rpc`·`migrate-spec`·`validator`·`upgrade` 를, MCP 쪽은 `network_*`·`remote_rpc`·`consensus_*`·`txpool`·`node_rpc`·`log`·`chain_new` 을 옮겼다. `app.Chains`·`Capabilities`·`Validators`·`NodeCall`·`ReadNode`·`MigrateSpec`·`DeriveIdentity`·`ValidatorSetOf`·`GenerateSet`·`ResolveBinary`·`UpgradeGenesis`·`AttachNetwork`·`Network(s)`·`DetachNetwork`·`DetectNetwork`·`PeersOf`·`CallOnNode`·`PeersOfNode`·`NodeAt`·`RegisterCapability` 를 신설했다.

**중복 둘이 죽었다.** `resolveBinary` 가 `upgradecmd` 와 app 에 각각 있어서 한쪽이 받는 이름을 다른 쪽이 거절할 수 있었다. 그리고 validator 키에서 무엇을 파생할지(`WithBLS` 인지 `AccountOnly` 인지)를 표면이 정하고 있었는데, 그건 합의 패밀리의 규칙이다.

**저장된 망의 노드에 닿는 법도 하나로 모았다.** 붙여 둔 노드는 SSH 터널이나 docker 매핑 뒤에 있을 수 있고, 거기 닿는 길은 노드 기록의 auth 서술자다. `CallOnNode`·`PeersOfNode` 는 URL 이 아니라 노드를 받는다.

**제가 낸 회귀를 테스트가 잡았다.** `ReadNode` 를 만들면서 네 읽기를 전부 필수로 했는데, 원래는 head 만 필수고 나머지는 최선 노력이었다. net 네임스페이스를 끈 노드가 오류가 돼 버렸다. 계약을 되돌리고 왜 그런지 적었다 |
| **U7** | ~~**DSL 흡수**~~ **전제가 틀렸다 — 폐기 2026-09-05** | U4 | — | ☒ **DSL 액션은 표면이 아니다.** 계획할 때 "액션 18개와 어세션 27개가 app 을 지나지 않는다"고 셌는데, `testhelper` 는 [[layers]] §3 에서 **L3 도메인 서비스**다. `dsl/interp` 의 `Action`/`Assertion` 계약을 구현하는 쪽이고, app 은 L5 다. 아래층이 위층을 부를 수 없다.

**컴파일러로 확인했다.** `testhelper` 에 `app` import 를 넣어 보니 레이어 위반일 뿐 아니라 **import 순환**이다: `app → testengine → … → testhelper`.

**중복의 성격도 달랐다.** U3·U4 에서 찾은 것은 같은 읽기가 두 벌 있는 것이었다. DSL 의 `sendTx` 와 `faucet` 은 라벨을 주소로 풀고, 노드 서명과 로컬 서명을 가르고, 수수료 인자를 적용한다. `app.TxSend`/`Faucet` 보다 하는 일이 많고 메커니즘이 다르다. 같은 구현의 복사본이 아니라 언어의 어휘다.

DSL 에게 표면은 `run` 이고, CLI 와 MCP 두 철자 모두 이미 app 을 지난다.

**라체트도 고쳤다.** 45개를 빚으로 세면 구조상 0 에 닿을 수 없고, 내려갈 수 없는 천장은 다음 사람에게 무시하는 법을 가르친다. 이제 표면만 센다(0/112). 어휘는 세지 않고 보고만 한다 |
| **U8** | **규칙 대칭 마감** | U2~U6 | 우회 항목 0 · 문서와 테스트가 한 규칙만 말함 | ☑ **2026-09-05.** `mcpImportAllowed` 의 마지막 여섯 항목(`core/rpc`·`core/collector`·`resource`·`core/node`·`core/session`·`core/remote`)이 각자 예고한 이관과 함께 사라져 목록이 비었다. 목록은 줄기만 하고 사라진 항목이 남아 있으면 테스트가 실패하므로, **빈 map 은 규칙이 안 걸린 것이 아니라 지켜지고 있다는 뜻이다.** 비대칭 라체트는 이것으로 폐기했고, 표면 규칙은 `TestSurfacesReachThroughApp` 하나가 말한다 |

**측정 기준선(2026-09-05 착수 시점)**: 등록 항목 157 = CLI 58 + MCP 54 + DSL 45(액션 18 + 어서션 27).
app 아래에 닿는 항목 109 = CLI 40 + MCP 24 + DSL 45.
손으로 확인한 32쌍 중 경로 불일치 22, 양쪽 다 app 인 것 4.

**U0~U8 이후(같은 날)**: 표면 등록 **0 / 112** 가 app 을 우회한다(CLI 0 · MCP 0).
DSL 어휘 45개는 L3 이라 세지 않는다 — U7 행에 왜 그런지 적었다.
동등성 테스트 17건이 붙었고 전부 변이로 확인했다.
`go run ./scripts/inventory/surface-graph .` 으로 재현한다.

### 라이브 스위트에서 남은 관찰 (2026-09-05)

`tests/e2e` 를 되살린 뒤 실제 바이너리로 돌린 결과다. 17개 중 **16 통과**, 1 skip, 그리고 간헐 실패 1건.

**`TestE2E_WbftQuorum6of6Halts2` 의 간헐 실패를 쫓다가 별개의 결함을 찾았다.**

실패한 판의 로그를 보니 재기동한 노드에 `Unclean shutdown detected` 와 `Switch sync mode from full sync to snap sync: snap sync incomplete` 가 찍혀 있었다. 노드는 높이 0 에 갇히고 피어 하나만 붙은 채 자기 ROUND-CHANGE 만 듣고 있었다.

**여기서 두 번 틀렸다.** "불결한 종료가 원인"이라고 했는데 통과한 판에도 똑같이 있었다. "snap sync 전환이 원인"으로 고쳤는데 그것도 통과한 판에 있었다. 실패한 판 하나만 보고 원인을 말하면 안 된다는 것을 배웠다. **통과한 판에도 그 표시가 있는지 먼저 봐야 한다.**

그 과정에서 **`process.Stop` 이 문서와 다르게 동작하는 것**을 찾았다. 바로 위 주석과 `doc.go` 가 "SIGTERM, grace, SIGKILL" 이라고 말하는데 코드는 `proc.Kill()`, 즉 유예 없는 SIGKILL 이었다. 데이터베이스를 가진 프로세스에게는 닫을 기회를 줘야 한다. 원격 드라이버는 또 달라서, 같은 `node stop` 이 로컬에서는 즉사이고 원격에서는 신호만 던지고 반환이었다.

고친 뒤 10판: **전부 통과, `unclean=0`, `snap=0`, 재기동 노드가 매번 블록을 받아 따라잡았다**(수정 전 4판은 `unclean` 4/4, `snap` 2/4). 개입으로 인과 사슬의 앞 고리를 끊은 것은 확인했지만, **이 수정이 그 실패를 고쳤다고는 주장하지 않는다** — 수정 전에도 4판 연속 통과한 적이 있어 실패율을 정확히 모른다.

**`TestE2E_StablenetProposalExpiry` 는 되살렸다(2026-09-06).** 이미 구성된 워크스페이스에 capability 로 게이트된 스펙을 돌릴 방법이 없었다. `chainbench test` 는 폐기됐고, `run --workspace-dir` 는 스펙 선언대로 **다시 구성**하고, `run --rpc` 는 붙기는 하지만 워크스페이스를 안 읽어서 게이트가 늘 닫혀 있었다.

`run --workspace-dir <dir> --attach` 를 만들었다. 워크스페이스가 엔드포인트와 **그 구성이 광고한 capability** 를 함께 준다. 둘을 한 번에 읽는 것이 핵심이다. 엔드포인트만 넘기면 게이트가 대조할 것이 없어 스펙이 skip 된다.

MCP `chainbench_run` 에도 `attach` 인자로 같이 넣었다. 한쪽에만 있으면 U 트랙이 없앤 격차가 다시 생긴다.

`AttachRunIn.Caps` 는 원래부터 있었는데 아무도 채우지 않았다. 배선만 없던 것이지 설계가 없던 것은 아니다.

**변이로 확인했다**: `in.Caps` 를 비우면 정확히 예전 증상(skip)으로 돌아간다.

**다만 시나리오는 아직 통과하지 않는다(정정 2026-09-06).** 처음에 1판만 돌리고 "통과"라고 적었는데 성급했다. 3판씩 두 번 재니 **3판 중 2판 실패**다. 고쳐진 것은 **게이트**이고, 실패는 그 안쪽의 별개 문제다 — `sendTx` 가 en1 에 보낸 트랜잭션의 영수증이 시간 초과한다(`context deadline exceeded`). N12 적용 전후로 실패 비율이 같아(각 2/3) N12 탓도 아니다.

skip 이 fail 로 바뀐 것은 후퇴가 아니다. skip 은 아무것도 검증하지 않던 상태이고, fail 은 시나리오가 실제로 무언가를 재고 있다는 뜻이다. 원인은 미상이며 wbft 정족수 회복 건과는 별개다.

### 노드 주소 체계와 라우팅 (2026-09-06)

**두 체계가 같은 집합 위에 겹쳐 있다.** `node1` 은 **순서**(`Index`)로, `en1` 은 **역할**로 정해진다. 역할 어휘는 `bp`(생산자)·`en`(RPC 엔드포인트)·`pn`(중계 계층)이고, 레거시 철자 `validator`/`endpoint` 는 `NormalizeRole` 이 접는다. `parseSelector` → `rolesForToken` → `NormalizeRole` 사슬은 정연하다. `pn` 은 topology 파일로만 생기고 카운트 경로(`--validators/--endpoints`)로는 안 생기는데, `peering.go` 가 그 경우를 막으므로 일관된다.

**어긋난 곳은 배정과 사용 사이였다.** 두 규칙이 각각 옳은데 아무도 둘을 맞춰 보지 않았다.

- `process.NodeConfig` 는 **`bp` 에만** `--unlock` 을 건다. `en`·`pn` 은 어떤 계정으로도 서명하지 못한다.
- `ResolveAccount` 는 **`node<N>` 에 키를 주지 않는다.** 그 노드가 서명한다는 뜻이다.

합치면 **`from: nodeN` 은 `on` 이 같은 노드일 때만 성립한다.** 그런데 `on` 과 `from` 은 독립 선택자이고, `malformedSelectors` 는 `on` 의 문법만 본다.

v1 스펙 45개가 `on: enN, from: nodeN` 으로 쓰여 있었고, 접속 표가 역할을 다 버려 `en1` 이 우연히 node1 로 풀리는 동안만 굴러갔다. 고친 것은 셋이다. ① 서명이 필요한 100곳의 `on` 을 송신 노드로 맞췄다(로컬 서명해 raw 로 보내는 30곳은 어느 노드든 받으므로 `en1` 을 유지). ② `misdirectedSends` 가 오프라인에서 거절한다. ③ 워크스페이스에 붙을 때 역할을 보존해 `en1` 이 실제로 엔드포인트를 가리킨다.

**검증 경로가 둘로 갈라져 있던 것도 찾았다.** `Precheck` 와 `validateRaw` 가 각자 검사해서, `Precheck` 에만 넣은 검사를 `validate` 명령이 못 잡았다. 지금은 둘 다 지난다.

**측정**: `proposal-expiry` 가 3판 중 1판 통과에서 **5판 중 4판 통과**로.

### 미해결 — 체인이 제때 처리하지 않는다 (2026-09-06)

둘 다 원인 미상이고 모양이 닮았다. 함께 보는 편이 나을 수 있다.

**`TestE2E_WbftQuorum6of6Halts2`** — 6노드 중 2개를 멈춰 정족수를 깨는 것은 매번 되고, 되살린 뒤 2분 안에 생산이 재개되지 않는다(5판 중 2판). 쫓다가 `process.Stop` 이 SIGKILL 이던 별개 결함을 찾아 고쳤고 그 뒤 10판 통과했지만, 수정 전에도 4판 연속 통과한 적이 있어 **인과는 증명하지 못했다.**

**`TestE2E_StablenetProposalExpiry`** — 올바른 노드(`node1`)에 보낸 트랜잭션의 영수증이 안 온다(5판 중 1판). 라우팅을 고친 뒤 빈도가 크게 떨어졌으므로 상당 부분은 스펙 문제였고, 남은 것이 원래부터 있던 별개 문제인지 같은 뿌리인지는 모른다.

**쫓을 때의 교훈**: 실패한 판 하나만 보고 원인을 말하면 안 된다. 이 두 건을 쫓으며 가설 세 개가 무너졌는데, 셋 다 **통과한 판에도 같은 표시가 있었다.** 통과·실패를 나란히 놓고 갈리는 것을 찾아야 한다.

### U 트랙에서 배운 것 (2026-09-05)

계획을 세울 때는 몰랐고 하면서 알게 된 것들이다. 다음에 같은 종류의 일을 할 때 쓰라고 적는다.

**중복은 두 종류였고 대응이 다르다.** CLI 와 MCP 사이의 것은 같은 읽기를 두 벌 쓴 것이라 하나로 합치면 됐다(U3 의 키 참조, U4 의 온체인 동사). DSL 과의 것은 그렇지 않았다. 이름이 같고 의도가 같아도 메커니즘이 다르면 합칠 대상이 아니다. 세기 전에 열어 봐야 한다.

**레이어가 계획을 이긴다.** U7 은 "DSL 도 app 을 지나게 한다"였는데, `testhelper` 가 L3 이라 불가능했다. 문서를 읽고 안 것이 아니라 import 를 넣고 컴파일러에게 물어서 알았다. 계획이 레이어를 건드릴 때는 그렇게 확인하는 편이 빠르다.

**지표가 못 내려가면 지표를 의심한다.** DSL 45개를 빚으로 세는 한 0 에 닿을 수 없었다. 내려갈 수 없는 천장은 다음 사람에게 무시하는 법을 가르친다.

**통과하는 테스트는 변이로 확인해야 한다.** 17건 중 두 건이 처음에는 아무것도 증명하지 못했다. 하나는 디렉터리를 훑다 빈 `process.json` 두 개를 비교하고 있었고, 하나는 step 의 detail 안에 우연히 들어 있는 낱말을 찾고 있었다. 둘 다 초록색이었다.

**컴파일되는 것과 도는 것은 다르다.** `tests/e2e` 는 8월 말부터 죽어 있었는데 CI 의 e2e 태그 vet 은 못 잡았다. 명령 이름이 문자열이기 때문이다. 테스트가 **참조하는 것**의 썩음은 컴파일러가 잡지만, 테스트가 **요구하는 것**의 썩음은 실제로 돌려 봐야 안다.

**라이브는 한 번에 하나만 돌린다.** 전체 스위트가 도는 중에 다른 실행을 시작했다가 그쪽 `pkill` 이 앞 실행의 노드를 죽였다.

## 1m. 남은 작업 — 순서 (2026-09-07 그래프 재분석)

> 2026-09-06 에 게이트로 전수를 세어 두었다. 하루 뒤 AST 그래프(패키지 64개, 간선 202개, 위반 0)로
> 각 항목이 어느 패키지 어느 파일에 떨어지는지 다시 재고 순서를 붙였다.
> **세는 일과 순서를 정하는 일은 달랐다.** 셀 때 "작고 독립적"으로 분류한 것 중 둘이 그렇지 않았고,
> 하나는 지금 코드에서 통과할 수 없는 게이트였다.

### 재측정이 뒤집은 전제 셋

**B2 는 적힌 만큼 풀지 못한다.** "구문이 L3 에 묶인 유일한 이유가 이 타입 하나"라고 적었는데,
`internal/dsl/interp` 는 세 파일에서 `session` 을 쓴다.

| 파일 | 쓰는 이름 |
|---|---|
| `fingerprint.go` | `Fingerprint` 하나만 쓴다 |
| `interpreter.go` | `AssertResult`·`Environment`·`TestRecord`·`TestStatus` 를 쓴다 |
| `run.go` | 위에 더해 `PostResult`·`StatusBlocked/Fail/Pass`·`StepResult` 까지 아홉 개를 쓴다 |

`Fingerprint` 가 `string` 을 반환하게 바꾸면 `fingerprint.go` 하나가 풀리고 패키지 의존은 그대로다.
덧붙이면 문법 패키지 `internal/dsl` 자체는 fanOut 이 0 이라 이미 깨끗하다. 묶여 있는 것은 구문이 아니라
해석기이고, 그것을 떼는 일은 `run.go` 의 아홉 이름을 옮기는 큰 일이다. **B2 는 작은 항목이 아니다.**

**B1-a 의 게이트는 U 트랙이 통과 불가능하게 만들었다.** `suitecmd` 가 지금 참조하는 내부 패키지는
`app`·`dashboard`·`core/home`·`chains/all` 뿐이고 `validate.go` 는 `app.Validate` 한 줄만 부른다.
그런데 `app` 의 fanOut 이 22 라서 링크가 `suitecmd → app → testengine → core/rpc·core/session` 으로
전이된다. 바이너리가 하나인 이상 어느 명령을 켜도 전부 링크된다. 즉 "링크하지 않게"는 바이너리를
쪼개지 않는 한 통과할 수 없다. **게이트를 호출 수준으로 다시 쓰는 결정이 착수보다 먼저다.**

**A7 은 `wbft` 하나가 아니다.** `RoleValidator` 라는 같은 이름이 뜻이 다른 세 곳에 있다.

| 어디 | 값 | 무엇을 가리키나 |
|---|---|---|
| `core/node/node.go:45` | `"validator"` | 블록을 만드는 **노드 역할**의 옛 표기다 |
| `dsl/spec_v2.go:95` | `"validator"` | 포크 뒤를 이어받는 **바이너리**다 |
| `validatorset/validatorset.go:20` | `"validator"` | **계정**의 기능이다 |

2026-09-06 에 고친 `node1` 대 `en1` 혼동과 같은 뿌리다. 이름 겹침 검출을 먼저 붙이면 N0/NM6 이
무엇을 지워야 하는지가 목록으로 나온다. **A7 은 N0/NM6 의 선행 조건이다.**

### 그래프가 찾아낸 합류점

`internal/chainsetup/steps_compose.go` 한 파일(841줄, 이 저장소에서 가장 큰 비테스트 파일)에
네 항목이 겹친다.

- **N9** 가 강제하려는 해석 순서가 이 파일의 함수 순서 그 자체다.
  `Keys`(64행) → `Allocate`(193행) → `Genesis`(314행) → `Config`(396행) → `Provision`(520행).
- **N0/NM6** 의 레거시 역할 방출이 `placements()` 안 167·171 행이다.
- **N1~N6** 청사진이 건드릴 `placements`(144행)·`Netmap`(672행)·`peerPlan`(794행)이 여기 있다.
- **P6** 이 줄이려는 6,781줄 가운데 가장 큰 단일 파일이 이것이고 `steps_lifecycle.go`(918줄)가 다음이다.

따로 착수하면 같은 파일을 네 번 연다. 그리고 **P6 은 독립 작업이 아니라 위 셋의 결과다.** 줄 수를
목표로 삼아 따로 착수하면 U 트랙 때처럼 코드가 옮겨 다니기만 하고 줄지 않는다(그때 4,176 대 4,569 로
확인했다). 같은 이유로 **A8 은 N1~N6 뒤에 와야 한다.** 청사진을 바꾸면 죽은 심볼 목록이 다시 그려지므로
지금 세어 둔 896건은 착수 근거가 아니라 착수 뒤에 다시 잴 대상이다.

### 순서

**0단계 ☑ 2026-09-07.** B1-a 게이트를 다시 쓰고, S 트랙을 남은 값어치로 줄여 적고, A8 을 2단계 뒤로 옮겼다.
이 절이 그 결과다.

**1단계 ☑ 2026-09-07. A7 이름 겹침 검출.** 규칙으로 설명되는 것 101개, 관용 19개, 빚 33개.
첫 수확이 `RoleValidator`×3 이었고, 그것이 2단계의 입구가 되었다.

**2단계 ☑ 현행 재평가 2026-10-14.** N1~N6은 현행 blueprint·resource·node·chainsetup으로 소비된다. §0a 7번에서 노드별 서버 배치와 가족별 BLS 잔여도 닫았다. 아래 표의 과거 설명을 현행 미완료로 해석하지 않는다.

| 무엇 | 상태 |
|---|---|
| **N0/NM6** 역할 방출을 정본으로 | ☑ 방출은 `bp`/`en` 이고 `LegacySpelling` 은 없다. 옛 철자는 `Node`·`Record` 두 경계에서 읽을 때 접는다 |
| **N9** 해석 순서 강제 | ☑ `composeNeeds` 에 선언하고 모든 단계가 `require` 를 거친다 |

**라이브 재검증 2026-09-07: 19/19 통과**(763초, 실패도 스킵도 없음). 역할 방출이 바뀌면 argv 와 지속 상태가
달라지는데, 두 체인에서 조립·기동·합의·하드포크 스왑·정족수 붕괴와 회복·스냅 싱크가 모두 지나갔다.
다만 **불안정 두 건의 상태는 이 한 판으로 알 수 없다** — `WbftQuorum6of6Halts2`(원래 5판 중 2판 실패)와
`StablenetProposalExpiry`(5판 중 1판 실패)가 통과했지만 한 판 통과는 예상 범위 안이다.
| **N1** 선언 스키마와 파서 | ☑ `internal/core/blueprint`. `steps_compose.go` 를 건드리지 않는 새 패키지라 앞의 둘과 파일이 겹치지 않았다 |
| **N2~N6** 해석·raw 경로·물질화·preset·topology 흡수 | ☑ **정정 2026-09-20 — 이 칸의 `◐` 가 틀렸다.** 판정에 쓴 잣대가 "`chainsetup` 이 `blueprint.X` 를 부르는가" 였는데 **그건 N3·N4·N6 어느 것의 게이트도 아니다.** 각 게이트는 §1g 표에 있고 다섯 전부 2026-09-07 에 라이브로 통과했다. 셋을 코드로 다시 확인: N4(로컬/원격 분기 없음) — `chainsetup` 직접 파일 쓰기 **0곳**; N6(혼용 거부) — `verbs_steps.go:149`, 그리고 **두 형식 병존은 "이관 기간 병존" 이라고 게이트가 명시한 설계 동작**이다; N5 — `chaincmd/blueprint.go` 의 `--from-preset`/`--from-topology` 가 서로 배타. `placements()`(`steps_place.go:72`)의 세 갈래는 미완성이 아니라 그 병존이다. **잘못 판정한 방식이 문제였다** — 흡수 여부를 호출 관계로 물었고, 게이트는 동작으로 묻는다. 실제 잔여는 §1g `N3` 행이 이름을 대고 적은 빚 둘(노드별 `server:` 배치, BLS 항상 파생)뿐이다 |

**3단계 ☑ 2026-10-14.** A8 재측정과 A7b 현행 대조를 수행했다. 모든 언어 AST 인벤토리·그래프의 고정 소스 capture/독립 재현과 관련 전체 검사가 통과했다. [증거](all-code-graph-evidence.json). 이 완료는 원문 87개 충분성·전체 mainnet·외부 W3/W4·최종 Web 인수를 뜻하지 않는다.

**병행할 수 있는 것** — 위와 파일이 겹치지 않는다.

| # | 무엇 | 지금 상태 | 게이트 |
|---|---|---|---|
| **N11** | 다중 config | ☑ **완료 2026-09-08.** 게이트는 `swapNode` 가 이미 만족했고, 빠져 있던 통합 테스트를 채웠다(진짜로 조립하고 드라이버만 스텁) | 일부 노드만 다른 config 로 재기동한다 |
| **V7** | 기회 개명 백로그 | ☑ `netreg` 는 모듈이 이미 없어 파일 이름만 `networks.go` 로 바꿨고, `accounts` 는 상류 SDK 를 가리키는 이름이라 두기로 했다 | 네이밍 규칙 표를 통과한다 |
| **B1**-b | 파서 fuzz | ☑ **완료 2026-09-07 (#357).** `FuzzParse`·`FuzzMigrateV1`·`FuzzInlineEnv` 셋. 씨앗은 저장소의 스펙 122개 — 무작위 바이트는 거부 경로만 훑고, 진짜 스펙이라야 변이가 수용 경로에 닿는다 | 죽지 않는 것에 더해, **통과한 것은 해석기가 쓸 수 있어야 한다**(id·chain·schemaVersion). **fuzz 가 v1 문법의 결함을 찾았다** — 아래 |
| **P8** | 미이관 테스트 케이스 | 문법 갭을 메우거나 이관하지 않을 이유를 적는다 | ☑ **완료 2026-09-07 — 갭이 아니라 기록이 문제였다.** `tests/specs/README.md` 가 갭에 막혔다고 적은 19건 중 **15건에 이미 스펙이 있고 전부 검증을 통과한다**. 없다고 적힌 프리미티브가 그동안 다 생겼기 때문이다: 로그를 유발하기 전에 구독을 여는 `wsOpen`, 손상된 이중서명을 조립하는 `sendRawTampered`, EIP-7702 의 `sendSetCode`, 로컬 키를 만드는 `newAccount`, "오류는 나도 되지만 not-found 는 아니다"를 묻는 `methodPresent`, revert 를 기대하는 `callError`, ceil(2n/3) 의 `derive op:"quorum"`. **불완전한 문서보다 나쁜 상태였다** — 그걸 보고 계획하면 끝난 일을 다시 하고, 커버리지를 감사하면 실제보다 얇다고 믿는다. 문서를 실제 상태로 고치고 `TestSpecDoc_BlockedCasesHaveNoSpec` 이 그 주장을 검사하게 했다(layers.md 처럼 **문서를 파싱하고 복제하지 않는다**). 변이 둘로 확인. **진짜 남은 8건**: SDK 클라이언트 가드 2건(표현 대상이 아니다) · 조작자 공급 키가 필요한 2건(`newAccount` 는 키를 만들 뿐 받지 못한다) · 바이너리가 기능을 안 담은 P256 3건과 genesis 빌더가 처음부터 최종 코드를 굽는 govminter 1건(라이브 반증, 다른 빌드가 필요하다). 기준선도 바뀌었다 — 레거시 등록부 `testkit.Cases()` 가 2026-09-06 에 사라져(A5) 134/56 같은 숫자는 다시 뽑을 수 없다. 기준은 커밋된 스펙 **122개**이고 전부 `validate` 를 통과한다 |
| **B2** | 해석기를 `session` 에서 떼기 | 요구를 좁힌다. **완전 분리는 하지 않는다** | ☑ **완료 2026-09-07 — 재보고 후 범위를 바꿨다.** 실측하니 결합이 타입 목록보다 훨씬 좁았다: interp 는 `Environment` 11개 중 **4개**(`Nodes`·`Resolve`·`ResolveEach`·`UpdateNode`), `TestRecord` 12개 중 **4개**(`Step`·`Assert`·`PostAction`·`Status`)만 쓴다. 그래서 `interp.NodeTable` 과 `interp.Recorder` 로 요구를 선언했다(`operation.Opener` 와 같은 수법). 세션 쪽은 구조적으로 만족하므로 엔진은 바뀐 것이 없다.

**완전 분리는 안 하기로 했고 그 이유를 남긴다.** `StepResult` 형제들을 interp 로 옮기면 서로 맞아야 하는 구조체가 둘이 되고, 한쪽에 필드를 더하고 다른 쪽을 잊으면 **아티팩트에서 증거가 조용히 빠진다.** 그건 올해 내내 없애 온 실패다(포트 표현 3벌, 링 writer 2벌, 낡은 이관 기록). import 하나 줄이자고 되풀이되는 위험을 사는 것은 나쁜 거래다. 게다가 애초의 동기였던 링크 분리는 U 트랙 이후 성립하지 않는다 — 바이너리가 하나라 `app` 을 통해 전부 링크된다.

**좁히기가 능력이 됐다는 증거**: 해석기가 세션 없이 돈다. 전에는 interp·testhelper 의 테스트가 임시 디렉터리에 진짜 세션을 세웠는데, 11개 구현보다 그게 쌌기 때문이다. 이제 4개면 된다. 변이 둘로 확인 |

**S 트랙 잔여** — U 트랙이 "기능 목록이 세 벌"이라는 08-18 진단을 이미 무너뜨렸다. CLI 와 MCP 는 한
진입점을 쓴다. 남은 값어치는 **MCP 손작성 스키마 감축**과 **`query` 투영(S7)** 둘뿐이다.
S0·S1·S2·S4 는 여기서 닫는다.

### 이 기기에서 못 하는 것 — 로컬 docker 가 필요하다

> **정정 2026-09-07.** 이 절은 "다른 머신이 필요하다"고 적혀 있었는데 **근거 없이 쓴 것이고 사실과 반대다.**
> R 트랙의 전제 자체가 "실 원격 서버 없이 컨테이너를 가상 서버로 쓴다"이고([[docker-remote-design]]),
> R1~R5 는 전부 끝났으며 **R6 도 15대 docker 서버셋에서 라이브로 완주했다**(2026-09-02).
> 필요한 것은 다른 기계가 아니라 이 기계 위의 docker 데몬이고, 지금 그것이 떠 있지 않다.
> docker 는 `env/docker/gen-env.sh` 가 만들고 산출물(인벤토리·localmap)은 gitignore 대상이다.

| # | 무엇 | 남은 것 |
|---|---|---|
| **R6** 잔여 | poa 원격 브링업 | **chainbench 결함이 아니다.** `etcdInit` 이 형성한 클러스터를 노드의 배경 etcd 관리가 거버넌스 멤버 14개를 보고 다시 join 하려다 잃는다. 재현 2회 중 1회. go-wemix 바이너리 쪽이다 |
| **G2** 잔여 | 핸드오프의 원격 경로 | R6 과 같은 docker 에서 본다 |
| `chain rm` 원격 | N4 가 남긴 유일한 진짜 분기 | `filestore.Store` 에 삭제가 없다. 파괴적 원격 작업이라 검증할 docker 가 있을 때 함께 |

### 미해결 조사

| 무엇 | 실측 | 아는 것 |
|---|---|---|
| **`TestE2E_WbftQuorum6of6Halts2`** | **2026-09-07: 18판 무실패** | **재현되지 않는다. 고쳤다는 말이 아니다.** 실제 테스트 10판 + 손 재현 8판. "head 2 에 멈춤"이라는 기록에서 **이른 정지가 방아쇠**라는 가설을 세워 head 2·4·20 에서 시험했는데 여덟 판 전부 3초 만에 회복했고 여섯 노드가 피어 5를 보고했다 — 가설은 무너졌고 대신할 원인은 없다. 그 사이 들어간 변경(`process.Stop`·NM6)이 이것과 연결된다는 것도 보인 적이 없다. 재현되면 이제 진단이 남는다 |
| ~~**`TestE2E_StablenetProposalExpiry`**~~ | ☑ **규명·수정 2026-09-07 (#358)** | 실패 4판과 통과 5판이 **`node1 sealed=0`** 하나로 갈렸다. chainbench 의 결함은 준비 판정이었다 — `detectProducing` 이 주 노드 하나에게만 물어서, 4검증자 BFT 망이 셋만 봉인해도 "생산 중"이었다. `health.Participants` 로 봉인자를 세고 하네스가 **모든 생산자의 참여**를 기다린다. 검사 없이 16판 중 5실패 → 검사와 함께 16판 중 0실패, 빼면 5판 안에 복귀 |
| **`G2` 잔여** | — | 원격 경로라 docker 와 함께 본다 |

### 이 순서의 단점

2단계가 한 덩어리라 PR 이 커지고, 중간에 멈추면 `steps_compose.go` 가 절반만 바뀐 채 남는다.
쪼개려면 N0/NM6 만 먼저 잘라낼 수 있는데 그러면 N9·N1~N6 때 같은 파일을 한 번 더 연다.
그리고 0단계는 결과물이 문서뿐이라 진척이 눈에 보이지 않는다.

**쫓을 때의 교훈**: 실패한 판 하나만 보고 원인을 말하면 안 된다. 위 두 건에서 가설 셋이 무너졌는데
**셋 다 통과한 판에도 같은 표시가 있었다.** 통과와 실패를 나란히 놓고 갈리는 것을 찾는 단계가 원인과
우연을 가른다.

## 1n. 다음 사람에게 (2026-09-08)

> 이 기기(docker 데몬 미기동)에서 할 수 있는 일은 끝났다. 남은 것은 셋인데 **하나만 실제 작업**이고
> 둘은 착수 근거가 아직 없다.

### 지금 트리 상태

`main` + PR #361(커밋 12개). 53개 패키지 통과, `gofmt`·`go vet`·`golangci-lint`·`go build -tags e2e` 통과.
라이브 스위트는 2026-09-07 에 19/19 통과했다.

이번에 닫힌 것: S0·S1·S2(조회 목록)·S4·S7 · N0/NM6 · N9 · N1~N6 · A7 · A7b · A8 · B1 · B1-b · N11 · P6 · P8 · V7.

### 남은 것 1 — 로컬 docker 가 필요하다 (유일한 실제 작업)

**다른 기계가 아니라 docker 데몬이 필요하다.** R 트랙의 전제가 "실 원격 서버 없이 컨테이너를 가상
서버로 쓴다"이고 R1~R5 는 끝났으며 R6 도 15대 서버셋에서 라이브 완주했다([[docker-remote-design]]).

착수 순서:

1. docker 를 띄우고 `env/docker/gen-env.sh` 로 15대 docker 를 만든다. 산출물(인벤토리·localmap)은
   gitignore 대상이라 저장소에 없다.
2. **R6 잔여** — `etcdInit` 이 형성한 클러스터를 노드의 배경 etcd 관리가 거버넌스 멤버 14개를 보고
   다시 join 하려다 잃는다(재현 2회 중 1회). **chainbench 오케스트레이션은 정상이고 go-wemix
   바이너리 쪽이다.** 체인팀과 볼 일이지 여기서 고칠 것이 아닐 수 있다.
3. **G2 잔여** — 핸드오프의 원격 경로. 같은 docker 에서 본다.
4. **`chain rm` 원격** — `filestore.Store` 에 삭제가 없어(확인·읽기·쓰기·체크섬뿐) 경계를 통과하지
   못한다. 파괴적 원격 작업이므로 검증할 docker 가 있을 때 인터페이스에 `Remove` 를 더한다. 지금은
   절반만 지우거나 지웠다고 거짓말하지 않고 소리 내어 거절한다.

### 남은 것 2 — 근거가 생기면 (지금은 하지 않는다)

| 무엇 | 무엇이 근거가 되나 |
|---|---|
| **S2 스키마 감축** | 지금 되풀이는 `rpc` 16 · `workspaceDir` 11 · `chain` 11 인데 **설명이 갈라지지 않았다**(16곳 중 15곳이 같은 표현). 예방이지 교정이 아니다. 저 숫자를 다시 세어 갈라짐이 보이면 그때다 |
| **어휘에서 이름 떼기** | `validate` 가 이름 해석에 부르는 레지스트리가 액션 구현 자체라 rpc·session 을 끌고 온다. 액션 45개의 등록 방식을 바꾸는 일이고, 문법만 쓰는 **다른 소비자**가 생기면 근거가 선다 |
| **testengine→chainsetup 엣지** | 좁히기로는 안 된다(4파일 중 1개만 사라진다, P6 참조). 조립을 엔진 밖으로 내보내 엔진이 망을 *받게* 하는 재설계라야 하고, 그럴 이유가 생기면 별도 항목으로 세운다 |
| **나머지 기능 등록** | `internal/feature` 에 **24개**가 등록됐고(2026-09-11 재측정, 문서의 17은 낡았다) 래칫이 63을 천장으로 잡고 있다. 표면을 파생 플래그로 바꾸는 일은 `TestComposeFeatures_TagsMatchTheCommands` 가 지켜 주므로 **손으로 쓴 절반을 지우는 일**이 된다 |

### 남은 것 3 — 재현되면 본다

`TestE2E_WbftQuorum6of6Halts2` 는 2026-09-07 에 18판 무실패였고 **재현되지 않는다**(고쳤다는 뜻이
아니다). 이제 대기가 포기할 때 모든 노드의 head 와 피어 수를 남기므로, 다시 나오면 그것부터 본다.

### 이번에 비싸게 배운 것 — 측정을 먼저 의심하라

라이브를 쫓을 때 **틀린 답을 그럴듯하게 만든 것이 코드가 아니라 측정이었다.**

1. **한 번만 재고 판단하지 않는다.** 12초 단일 표본이 head 0 을 회귀처럼 보이게 했다. wbft 는 1에
   머물다 가속하므로 25초까지 재면 24다.
2. **`pkill -f` 는 자기 자신을 맞춘다.** 패턴이 스크립트 본문에 들어 있으면 러너가 죽는다.
   `pkill -x <실행파일>` 로 정확히 맞춘다.
3. **도는 스크립트를 편집하지 않는다.** bash 는 바이트 오프셋으로 이어 읽으므로 아직 안 읽은 줄이
   어긋난다. 복사본을 만들어 돌린다.
4. **라이브를 둘 이상 동시에 돌리지 않는다.** 포트 8600 을 두고 다투면 같은 바이너리·같은 체인이
   24와 0을 함께 낸다. 새로 띄우기 전에 `pgrep` 으로 0을 확인한다.
5. **통과와 실패를 나란히 놓는다.** 실패한 판만 보면 가설이 선다. 제안 만료 건은 통과 5판과 실패
   4판을 견줘 `node1 sealed=0` 하나로 갈렸다.
6. **`golangci-lint` 를 검증 절차에 넣는다.** `gofmt`·`vet`·`build`·`test` 만 돌리다 CI 에서 `unused`
   에 걸렸다. 로컬에 있는데 안 돌린 것이다.


## 1o. 배선 감사 — 반영 대기 (WA 트랙, 2026-09-08)

제품의 두 목적이 실제로 배선됐는지 코드 전체를 AST 그래프로 훑어 확인했다. 6개 계층
(DSL 문법·인터프리터 / compose(DSL→체인) / run·record·report / MCP 표면 / CLI 표면·parity /
tests/tc 코퍼스)을 병렬로 감사했다. 두 목적은 이렇다.

1. tests/tc 의 DSL 을 파싱해 체인을 구성하고, 지정 테스트를 수행하고, 결과 리포트를 쓴다.
2. 체인을 구성한 뒤 동적으로 원하는 테스트를 수행한다.

둘 다 MCP·CLI 양쪽에서 되어야 한다. 도달성 요약:

| | CLI | MCP |
|---|---|---|
| 목적1 spec→구성→실행→리포트 | 됨(`chainbench run`) | 구멍: 세션 경로 미출력→리포트 회수 불가, spec 경로 실행 불가, 실패를 tool error 로 안 냄 |
| 목적2 구성 후 동적 실행 | 구멍: 단일 테스트 목록·실행 없음, 네트워크 attach 없음 | 구멍: 한 번에 구성+유지 안 됨, 노드 stop/start 없음, run 은 항상 teardown |

happy path 는 CLI 에서만 온전하다. 아래는 심각도 순 작업리스트다. 모두 정적 분석이며 라이브
실행으로 확인한 것은 없다(특히 WA1 은 이름·grep 증거다).

> **상태(2026-09-12 재측정).** 이 목록은 PR #363 으로 반영을 마쳤고, **이번에 체크박스를
> 실제로 바꿨다.**
>
> 2026-09-10 에도 "반영을 마쳤으니 미완으로 읽지 말라"는 주석이 바로 여기 붙어 있었다.
> 그런데도 그 뒤로 **WA14·WA19·WA24 를 열린 항목으로 착각해 각각 코드 발굴을 다시 했다.**
> 교훈은 분명하다 — **체크박스 23개 위에 붙인 산문은 읽히지 않는다.** 상태는 상태를 나타내는
> 자리에 적어야 하고, 그 자리는 박스다.
>
> 그래서 이번에는 (1) 확인한 항목의 박스를 `[x]` 로 바꾸고, (2) 각 항목에 **30초 안에 다시
> 확인할 수 있는 명령**을 붙였다. 같은 발굴을 여섯 번째로 하지 않기 위한 것이다.
> 개별 확인하지 않은 항목은 **`◐`** 로 뒀다 — 임의로 완료 표시하지 않는다. `◐` 는
> "미완"이 아니라 **"이번 재측정에서 개별 확인하지 않았다"** 는 뜻이고, 각 항목의 `증거:`
> 줄이 어디를 보면 되는지 이미 가리키고 있다. §1p 가 남은 것을 따로 적고 있으므로 `◐` 가
> 곧 할 일 목록은 아니다.
>
> **오늘 확인해 완료로 바꾼 것 (13건)**: WA2·WA3·WA4·WA5·WA6·WA7·WA8·WA13·WA14·WA16·WA19·WA26
> (+ WA24 는 별도 항목으로 재측정). 그중 셋은 "조용히 거짓 통과" 부류라 실제로 돌려서 확인했다 —
> WA8 은 `do` 의 `expect` 오타가 거부되는지, WA13 은 오타 난 `source` 가 `validate` 에서
> `UNRESOLVED` 로 걸리는지(`read`·`waitFor` 둘 다), WA7 은 실패한 실행이 tool error 로 나가는지.
>
> **남은 것은 §1p 에 따로 적혀 있다** — MCP 플러그인 재배포(D, 코드 아님).

### A. 정확성·목적 차단 (먼저 반영)

- [x] **WA1** [치명] 라이브 MCP 플러그인이 이 소스로 빌드된 게 아니다. **해소 (2026-09-12)** —
  "이 소스로 빌드된 게 아니다" 가 문자 그대로 맞았다. PATH 의 `chainbench-mcp` 가 **다른
  저장소**(`auto-coding/chainbench`)의 2026-07-26 빌드를 가리키는 심볼릭 링크였다. 이 저장소로
  다시 빌드해 링크를 돌리고 `tools/list` 로 73개를 확인했다. 상세는 §1s D.
  **(원래 진단)** 라이브는 `net_*`·`chainbench_test`·`test_list`·`setup_plan` 이름인데 이 브랜치는 `chain_*` 로 등록하고 test/test_list/setup_plan 이 없다. 증거: `internal/mcp/tools.go:16-76`. 방향: 라이브 바이너리를 이 소스로 재빌드하거나 이름 매핑을 맞춘다.
- [x] **WA2** [목적1·2 차단] 단일 테스트 목록·실행 기능이 CLI·MCP 양쪽에 없다. testengine 에 RunOne/ListTests 카탈로그 API 가 없다. 증거: `internal/testengine/` 전역 grep 무결과, `cmd/chainbench/` 에 test 그룹 없음. 방향: testengine 에 목록·단건 실행 API 를 만들고 CLI·MCP 에 붙인다. **확인(2026-09-12): 완료.** `chainbench test --help` 가 뜨고 `chainbench_test_list` 가 MCP 에 등록돼 있다
- [x] **WA3** [목적2 차단·MCP] 한 번에 구성+유지가 MCP 로 안 된다. `ChainUp` 이 CLI 전용(`cmd/chainbench/chaincmd/up.go:49`), `chainbench_run` 은 항상 teardown 하고 `KeepUp` 을 MCP 로 노출 안 한다(`internal/mcp/run_tool.go:108-115`, `internal/testengine/suite.go:75-76`). run 이 `Server`·`Docker`·`WaitBlocks` 도 드롭한다. 방향: ChainUp 을 MCP 로 노출하거나 run 에 KeepUp·Server·Docker 를 배선한다. **확인(2026-09-12): 완료.** `grep -c KeepUp internal/mcp/run_tool.go` > 0
- [x] **WA4** [목적2·MCP] 노드 단위 stop/start(장애 주입)가 MCP 에 없다. `app.NodeStop`/`NodeStart` 가 CLI 전용(`internal/app/net.go:147,152`, `cmd/chainbench/nodecmd/node.go:32,55`). 방향: MCP 도구를 더한다. **확인(2026-09-12): 완료.** `grep -l chainbench_node_start internal/mcp/*.go`
- [x] **WA5** [목적2·CLI] 네트워크 레지스트리(attach/detach/list/info/peers/topology, remote_rpc)가 CLI 에 없다. MCP 전용(`internal/mcp/network_tools.go`, `remote_tools.go`). 방향: CLI `network`·`remote` 그룹을 더한다. **확인(2026-09-12): 완료.** `chainbench network --help` 가 뜬다
- [x] **WA6** [목적1·MCP] MCP run 이 세션 경로를 출력 안 해 리포트를 회수 못 한다(dataDir 없으면 임시 dir). spec 을 경로로 못 넘겨 env 참조 해석이 우회된다(`ReadSpecFiles` CLI 전용). 증거: `internal/mcp/run_tool.go:104,141-153`, `internal/app/workflow.go:115-120`. 방향: run 출력에 세션 root 를 싣고, spec 경로 입력을 받는다. **확인(2026-09-12): 완료.** `grep -n SessionRoot internal/mcp/run_tool.go`
- [x] **WA7** [목적1·MCP] MCP run 이 테스트 실패를 tool error 로 안 낸다(항상 nil). CLI 는 exit 1/2 를 낸다. 증거: `internal/mcp/run_tool.go:141-153`, `cmd/chainbench/suitecmd/run.go:237-245`. 방향: `Summary.Failed()` 이면 tool error 를 낸다. **확인(2026-09-12): 완료.** `grep -n "Summary.Fail > 0" internal/mcp/run_tool.go` — 실패·blocked 면 tool error 를 낸다
- [x] **WA8** [정확성] `do` 의 `expect`(revert/reject/fail) 오타가 검증 안 돼, 오타 나면 "성공해야 함"으로 조용히 떨어진다. 부정 케이스가 거짓 통과한다. 증거: `internal/testhelper/builtins.go:432,446`, `fault.go:63`. 방향: expect 어휘를 오프라인 검증에 넣는다. **확인(2026-09-12): 완료.** `grep -n expectAdjuncts internal/dsl/spec_v2.go` — 어휘 밖 값은 거부된다
- [x] **WA9** [정확성] node-table `role:pn` 이 아직 mesh 로 배선된다. proxied 자동 선택이 count-form 에만 있다(`internal/testengine/compose.go:175-177`). node-table 형식은 `inlineTopologyOf` 경로라 `proxies=0` 으로 mesh 가 된다. en 이 bp 를 직접 본다. 방향: node-table 에 pn 이 있으면 proxied 를 선택하고, `Peering.Validate` 가 mesh+pn 을 거절하게 한다.
  - **해소 확인 (2026-09-12 실측).** node-table 경로도 덮인다 —
    `compose.go:237` 이 `proxies > 0 || topologyHasProxy(inlineTopo)` 로 peering 을 proxied 로
    만들고, `topologyHasProxy`(`compose.go:359-372`)가 node-table 의 `RolePN` 을 본다. 즉 pn 을
    **개수로 쓰든 역할로 쓰든 같은 계층**이 된다.
    재확인: `grep -n "topologyHasProxy" internal/testengine/compose.go`
- [x] **WA10** [정확성·증적] attach 경로(`app.AttachRun`)가 readiness 게이트·실패증적·Precheck 를 배선 안 한다. MCP `run --attach` 로 돌면 E6·E8 이 무력화되고 assertion 오류 메시지가 사라진다. 증거: `internal/app/workflow.go:104-108,112` vs `internal/testengine/suite.go:261,330-338`. 방향: AttachConfig 에 PreSpec·OnFail·Control·LogReader·Precheck 를 배선한다.
  - **해소 확인 (2026-09-12 실측).** compose 와 attach 가 **같은 꼬리를 공유한다** —
    `readWorkspaceComposed` 가 `suite.go:430`(attach)와 `505`(compose) 양쪽에서 불리고,
    그 안에서 엔드포인트·능력·노드표를 읽고 `gateReady` 로 게이트한다. 함수 주석이 WA10 을
    인용한다. 재확인: `grep -n "readWorkspaceComposed" internal/testengine/suite.go`
- [x] **WA11** [증적] artifacts.json 매니페스트를 쓰는 프로덕션 코드가 없다. `Recorder` 인터페이스에 Artifacts 통로가 없어 리포트 추적성 필드가 항상 빈다. 증거: `internal/dsl/interp/requirements.go:46-55`, `internal/core/session/record_impl.go:62-67`, `internal/core/report/report.go:17-24`. 방향: Recorder 에 Artifacts 를 노출하고 compose 산출물을 기록한다.
  - **해소 확인 (2026-09-12 실측).** 프로덕션 writer 가 있다 —
    `engine_impl.go:85-90` 의 `recordArtifacts` 가 `rec.Artifacts` 를 부르고, 그 입력
    `Deps.Artifacts` 는 **compose**(`suite.go:372`, `composedArtifacts(net)`)와
    **attach**(`attach.go:205`) 양쪽에서 채워진다. 즉 추적성 필드는 더 이상 항상 비지 않는다.
    재확인: `grep -rn "\.Artifacts(\|Artifacts:" internal/testengine/ | grep -v _test`
      — 호출부 1개와 채우는 곳 2개(compose·attach)가 함께 나와야 한다.
- [x] **WA12** [증적·경미] pre-action 실패·FAIL 이 status.json 에 reason 을 안 남긴다. 증거: `internal/dsl/interp/run.go:28-33,50,61-72`. 방향: `rec.Reason(...)` 를 부른다.
  - **해소 확인 (2026-09-12 실측).** `rec.Reason` 이 세 자리에 있다 —
    pre-action 실패(`interp/run.go:40`), 스텝 실패(`:64`), 단정 실패(`:81`). 라이브 실행의
    `status.json` 에서 `"reason": "2 assertion(s) failed"` 와
    `"step 3 (sendTx) failed: …"` 를 직접 확인했다.
    재확인: `grep -n "rec.Reason" internal/dsl/interp/run.go`

### B. 문법 — 선언됐으나 죽은 배선

- [x] **WA13** [문법] `waitFor` 의 source 가 오프라인 미검증(read 는 검증). 오타가 라이브에서야 실패한다(스펙 20건 사용). 증거: `internal/dsl/interp/resolve.go:41`, `internal/testhelper/read.go:234-237`. 방향: waitFor source 도 오프라인 검증에 넣는다. **확인(2026-09-12): 완료.** 오타 source 를 담은 스펙에 `chainbench validate` → `UNRESOLVED: source:<오타>`. read·waitFor 둘 다 걸린다
- [x] **WA14** [문법] 케이스 상위 `on`(DefaultOn)이 라우팅에 미적용. 스텝에 `on` 이 없으면 조용히 nodes[0] 로 간다. 증거: `internal/dsl/spec_v2.go:309`, `internal/dsl/interp/run.go:205-224`. 방향: resolveOn 이 DefaultOn 을 기본값으로 읽게 한다. **확인(2026-09-12): 완료.** `grep -n applyDefaultOn internal/dsl/interp/run.go` + `TestRun_DefaultOnRoutesStatements`. 라이브 스펙은 `tests/tc/go-wemix/vocabulary/01-*`
- [x] **WA15** [문법] 상위 `timeouts` 맵이 아무 데서도 안 읽힌다. 증거: `internal/dsl/spec.go:36`, 소비처는 `migrate.go:91` 뿐. 방향: 인터프리터가 액션 타임아웃 기본값으로 소비하게 한다.
  - **완료 (2026-09-12). 그리고 제가 어제 적은 판단이 틀렸다.**
    "런타임 기본값으로는 안 쓰인다" 고 썼는데 **소비자가 있다** — `interp/run.go:30` 이
    `caseTimeout(s.Timeouts)` 으로 **케이스 전체를 context 데드라인으로 묶는다**
    (`run.go:253-265`, 단위 테스트 `run_test.go:102`). 제가 `spec_v2.go` 와 `testhelper` 만
    grep 하고 `interp` 를 안 봤다. **두 선택지(소비 / 철회)는 애초에 필요하지 않았다.**
  - **남아 있던 진짜 느슨함은 키였다.** 값은 duration 으로 검사하는데 **이름은 검사하지
    않았다** — `timeouts.step` 은 유효한 duration 으로 통과한 뒤 **아무것도 묶지 않는다.**
    예산을 정한 것처럼 보이는데 예산이 없다. 읽히는 것은 `case`(별칭 `test`) 하나다.
  - **한 것**: 어휘 밖 이름을 파싱 시점에 거부하고(`timeoutKeys`), 스키마에도
    `propertyNames.enum` 으로 같은 두 이름을 못 박고, **둘이 어긋나면 실패하는 테스트**를
    더했다(§WA17 의 이음매와 같은 방식 — 목록을 복사하지 않고 `timeoutKeys` 와 직접 비교).
  - 변이 3건: 키 검사 제거(이 변경 직전 상태) · 파서 어휘만 넓히기(스키마와 불일치) ·
    스키마 제약 제거. 셋 다 실패 확인. 값 검사를 이름 검사가 가리지 않는지도 테스트로 고정했다.
  - **절반 해소 (2026-09-12 실측). 남은 것은 판단이다.**
    "아무 데서도 안 읽힌다" 는 더 이상 맞지 않다 — `spec_v2.go:380-384` 가 각 값을
    `time.ParseDuration` 으로 검증하고 파싱 불가하면 **파싱 시점에 거부**한다. 그래서 오타가
    조용히 무시되는 절반은 사라졌다.
    - **여전히 런타임 기본값으로는 쓰이지 않는다.** 값이 검증만 되고 아무 액션의 타임아웃도
      정하지 않는다. 이대로 두면 "선언은 받는데 효과가 없는 어휘" 이며, 이 저장소가
      `WA24`(죽은 능력)에서 걷어내던 바로 그 모양이다.
    - **선택지 둘.** (가) 인터프리터가 액션 타임아웃 기본값으로 소비한다 — 스펙이 반복해서
      쓰는 `timeout` 을 한 곳에 모을 수 있지만, 액션별 기본값과의 우선순위를 정해야 하고
      기존 스펙의 동작이 바뀔 수 있다. (나) 어휘에서 걷는다 — 값이 없으면 싸고 정직하지만,
      스펙 작성자가 반복을 계속 진다.
    - 재확인: `grep -n "Timeouts" internal/dsl/spec_v2.go internal/testhelper/*.go | grep -v _test`
- [x] **WA16** [문법] `InDelta` 비교자가 DSL 에서 도달 불가(디스패치 맵에 없음). 증거: `internal/dsl/assert/assert.go:97,202-219`. 방향: funcs 맵에 등록하거나 인터프리터에서 호출한다. **확인(2026-09-12): 완료.** `grep -rn InDelta internal/testhelper/` — testhelper 비교자 경유로 도달한다
- [x] **WA17 — 완료 (2026-09-12). 드리프트 주장은 낡았고, 진짜 구멍은 다른 자리였다.**
  - **낡은 절반**: "`onEach` 스키마는 string, 코드는 array" 는 더 이상 맞지 않다. 스키마의
    `doStatement.onEach`·`expectStatement.onEach` 는 **array of string** 이고 코드는
    `args["onEach"].([]any)` 로 읽는다 — 일치한다. 그리고 `envSpec`·`caseSpec` 의 필드 이름은
    Go 구조체의 json 태그와 **양방향으로 정확히 같다**(실측: 차집합 양쪽 다 공집합).
    이미 지키는 테스트가 있었다 — `TestSchemaV2MatchesParsedFields`·`…ParsedTypes`·
    `…StatementOnEachIsArray`.
  - **진짜 구멍**: `doStatement.expect` 가 **제약 없는 string** 이었다. 파서는 네 값
    (`receipt`·`revert`·`reject`·`fail`, `dsl.expectAdjuncts`)만 받고 그 밖의 값을 거부하는데,
    스키마는 아무 문자열이나 받았다. **WA8 이 파서에서 막은 구멍이 문서에는 그대로 남아
    있었다** — 이 파일로 검증하는 쪽(에디터·외부 도구·`$schema` 참조)은 오타를 유효하다고 말한다.
    오타가 통과하면 기본값 "성공해야 한다" 로 흘러 **부정 케이스가 조용히 긍정이 된다.**
  - **한 것**: 스키마에 `enum` 을 더해 파서와 맞췄고, 그 둘이 어긋나면 실패하는 테스트를 뒀다
    (`internal/dsl/schemasync_test.go`). 목록을 복사하지 않고 `expectAdjuncts` 와 직접 비교한다 —
    복사본은 다시 어긋난다. 더 느슨해질 수 있는 다른 두 축(`additionalProperties:false`,
    `required` 집합)도 같은 테스트가 잡는다.
  - **중복은 만들지 않았다.** 필드 이름·타입·`onEach` 는 이미 지키는 테스트가 있어서 다시 쓰지
    않았고, 그 테스트들이 덮지 않는 것만 더했다. 같은 검사를 두 번 구현하면 둘이 어긋난다.
  - **검증을 스키마로 돌리지는 않았다.** JSON-schema 의존성을 들여 파서의 집행을 복제하는
    값이 없다. 임베드 주석이 말하는 역할("파서·문서·외부 도구가 공유하는 필드 수준 정본")은
    그대로 두고, **그 말이 참이 되도록 이음매를 테스트로 고정**했다.
  - 변이 4건: enum 제거(이 변경 직전 상태) · enum 이 파서와 불일치 · `additionalProperties` 완화 ·
    `required` 에서 한 항목 제거. 각각 실패 확인.
  - 작성자 문서에도 네 값과 `reject` 의 `reason` 규율을 적었다(`docs/guide/dsl-authoring.md`) —
    같은 사실이 세 곳(파서·스키마·가이드)에 필요하고, 가이드가 비어 있었다.
  **(원래 진단)** 임베드 `SchemaV2` 가 검증에 미사용이고 파서와 이미 드리프트(`onEach` 스키마는 string, 코드는 array). 증거: `internal/dsl/spec_v2.go:18`, `internal/dsl/schema/v2.schema.json:290,331` vs `run.go:212`. 방향: 스키마를 실제 검증에 쓰거나 파서와 일치시킨다.
- [x] **WA18** [문법·경미] `onEach` 가 do-스텝 스키마에 있으나 액션 라우팅이 소비 안 한다(어서션만 소비). 증거: `v2.schema.json:290`, `builtins.go:543`. 방향: do-스텝 onEach 를 소비하거나 스키마에서 뺀다.
  - **해소 확인 (2026-09-12 실측).** do-스텝도 소비한다 — `interp/run.go:152-191` 이
    `onEach` 를 팬아웃해 선택된 노드마다 액션을 한 번씩 돌리고(각 호출은 `on` 한 개로
    바뀌어 `selectorTarget` 가 그 노드로 라우팅한다), 주석이 WA18 을 인용한다.
    재확인: `grep -n "onEach" internal/dsl/interp/run.go`

### C. compose — DSL→체인 배선 누락

- [x] **WA19** [compose] `env.target` 이 배치를 안 하고 fingerprint 만 바꾼다(오배선). 증거: `spec_v2.go:308`, `compose.go:158-170`(Target 미설정), `verbs_up.go:45`. 방향: compose 가 Target 을 ChainUpIn 에 배선하거나, env.target 을 문법에서 뺀다. **확인(2026-09-12): 완료.** `grep -n "up.Target = tgt" internal/testengine/compose.go` + `TestCompositionOf_EnvTargetPlaces`
- [x] **WA20** [compose] upgrade(handoff) env 가 같은 env 의 hardforks/topology/launch/config 를 무시한다. 증거: `compose.go:100-116`. 방향: 핸드오프 경로가 이 필드들을 이어받게 한다.
  - **해소 확인 (2026-09-12 실측) — 다르게 해결됐다.** 무시하는 것이 아니라 **소리 내어
    거절한다**: `compose.go:147-149` 가 핸드오프 env 에 hardforks·topology·launch·config 가
    있으면 오류로 나간다("a handoff composes from its profile and template; …"). 선언의 절반을
    조용히 버리는 것보다 거절이 낫다는 판단이며, 이 항목이 요구한 것을 만족한다.
    재확인: `grep -n "a handoff composes from its profile" internal/testengine/compose.go`
- [x] **WA21** [compose·역방향] DSL 로 표현 못 하는 chainsetup 기능: manifest/template(`verbs_up.go:42-43`), blueprint(`verbs_up.go:59`), keys-validator-subset(`verbs_steps.go:89-90`), chainID/networkID 고정, deploy-only stage(`verbs_up.go:27`). 방향: 필요한 것을 EnvV2 문법에 더한다.
  - **완료 (2026-09-12 실측). 5개 중 3개는 이미 표현 가능하고, 1개는 다른 경로로 되고,
    1개는 철회한다.**
    - `manifest`·`genesisTemplate` → **EnvV2 에 있다.**
    - `blueprint` → **EnvV2 에 있다.**
    - keys-validator-subset → **`keys.validators` 로 있다**(`KeySourceV2` 의 필드는
      `source`·`ref`·`validators`·`bootnode`).
    - **chainID/networkID 고정 → genesis 오버레이로 된다.** `genesis.set` / `genesis.overlay`
      가 `spec.Chain.GenesisOverlay` 로 그대로 실려 genesis config 에 적용되고,
      `config.chainId` 는 그 config 안에 있다. 같은 경로로 `config.applepieBlock: 0` 이
      구성된 genesis 에 들어가는 것을 F6 작업에서 실측했다. `--chain-id`/`--network-id` 는
      **실행 오버라이드**로 남는다 — 선언이 아니라 그 실행에 대한 지시라 자리가 맞다.
    - **deploy-only stage → 철회한다.** DSL 케이스는 **기동되지 않은 망에 단정할 수 없다** —
      RPC 가 없으면 읽을 것이 없다. 구성 산출물만 보는 케이스를 위해 문법을 넓히는 것은
      소비자 없는 표현력이고, 이 저장소가 두 번 되돌린 모양이다(`Transport` 타입, 객체형 참조).
      필요해지면 요구와 함께 다시 연다.
    - 재확인: `python3 -c "import re;src=open('internal/dsl/spec_v2.go').read();print(re.findall(r'json:\"([^\",]+)',re.search(r'type EnvV2 struct \{(.*?)\n\}',src,re.S).group(1)))"`
- [x] **WA22** [compose·게이팅] `env.capabilities` 가 케이스에 `requires` 가 있으면 통째로 버려진다. 증거: `spec_v2.go:312-314`. 방향: 두 목록을 병합한다.
  - **해소 확인 (2026-09-12 실측).** 버리지 않고 **합집합**이다 —
    `spec_v2.go:398-412` 가 케이스의 `requires` 를 먼저 넣고 env 의 `capabilities` 중
    중복 아닌 것을 덧붙인다. 주석이 그 의도를 적고 있다.
    재확인: `grep -n -A14 "The env.s capabilities and the case" internal/dsl/spec_v2.go`

### D. 커버리지·문서

- [x] **WA23** [커버리지] compose→run→report 전 과정 테스트가 전부 live-gated 라 CI 가 건너뛴다. 증거: `internal/testengine/*_live_test.go`. 방향: 바이너리 없이 도는 CI 통합 테스트를 하나 만든다.
  - **완료 (2026-09-12). 양쪽 절반을 다른 방법으로 덮었다.**
    - **compose 쪽**은 이미 있었다 — `runsuite_gate_test.go` 가 **기동 전 게이트의 거절
      15개**를 바이너리 없이 태운다. 그 거절들이 되돌릴 수 없는 것을 막는 자리다.
    - **run→report 쪽을 이번에 더했다** — `runtoreport_internal_test.go`. `NewAttachEngine`
      은 바이너리도 preset 도 필요 없고 **도달 가능한 RPC 만** 필요하므로, `httptest` 로
      **JSON-RPC 서버를 세워** 스펙을 판정과 리포트까지 태운다.
    - **스텁은 하네스가 아니라 체인이다. 이 구분이 이 테스트의 존재 이유다.** 하네스를
      스텁하면 RPC 에 아무도 답하지 않아 게이트와 단정이 **코드와 무관한 이유로** 통과·실패한다 —
      이 작업 앞부분의 스텁 라이프사이클 테스트가 통과하면서 아무것도 증명하지 못한 그 자리다.
      루프백에 진짜 서버를 세우면 인터프리터·단정·레코더가 **자기 일을 실제로** 한다.
    - 단정하는 것: 통과 스펙이 `session.json` 에 verdict 와 `assert.json` 에 **실제 수치**를
      남긴다 · 실패 스펙이 `status.json` 에 **이유**를 남긴다(WA12 가 말한 그것) · 체인이
      답하지 못한 메서드는 **패닉도 조용한 통과도 아니고 기록된 판정**이 된다.
    - **계약 하나를 잘못 알고 있었다.** 처음에 "실패 스펙이면 `Run` 이 에러를 돌려야 한다" 고
      단정했는데 아니다 — **실패한 테스트는 결과이고 엔진 에러가 아니다.** 엔진은 기록하고
      표면이 판단한다(CLI 는 exit code, MCP 는 tool error — WA7). 그 경계를 별도 테스트로
      못 박았다. 둘을 합치면 **실패한 스위트와 망가진 하네스를 구별할 수 없다.**
    - 변이 3건: 이유 기록 제거 · 실패 단정이 verdict 를 못 뒤집게 하기 · `assert.json` 기록
      제거. 셋 다 실패 확인. **첫 M2 는 메시지의 숫자만 바꿔 통과했다** — verdict 자체를
      건드리도록 다시 했다.
  - **절반 해소 (2026-09-12 실측).** "전부 live-gated" 는 더 이상 맞지 않다 —
    `RunSuite` 를 구동하는 테스트 6개 중 `runsuite_gate_test.go` 는 **라이브 게이트가 없다**.
    남은 5개(`suite_live_test.go`·`vocabulary_live_test.go`·`accountlabel_live_test.go`·
    `devaccount_live_test.go`·`wemix_live_test.go`)는 여전히 바이너리를 요구한다.
    - 남은 것은 **compose→run→report 전 과정**을 바이너리 없이 한 번 지나가는 테스트다.
      스텁으로 하면 이 세션이 이미 밟은 함정이 있다 — 스텁은 RPC 에 답하지 않으므로 게이트가
      엉뚱한 이유로 통과·실패하고, 테스트가 통과하면서 아무것도 증명하지 않는다.
      그래서 무엇을 스텁하고 무엇을 진짜로 둘지가 이 작업의 실체다.
    - 재확인: `grep -rln "RunSuite(" internal/testengine/*_test.go`
  - **재측정 (2026-09-12).** 방향을 짐작하지 말고 **재고 골랐다**: live 테스트는 환경변수 없이는
    skip 되므로 평소 `go test -cover` 가 곧 CI 커버리지다. 전체 64.6%.
    `TestEngine_GeneratesReport`(run→report)와 `TestAttachEngine_RunAgainstMockRPC`(attach)는
    이미 CI 에서 돈다 — 진짜 공백은 그쪽이 아니었다.
  - **측정이 지목한 것: `internal/core/keyring/operation` 이 0.0%, 함수 29개, 테스트 파일 없음.**
    **누가 개인키를 받는지 결정하는 바로 그 패키지**다(L2 에서 방금 고친 곳). 인덱스가 모든 키를
    담은 채 `keyring list` 에 읽히던 것이 아무에게도 걸리지 않은 이유가 여기 있었다.
  - **단정을 "출력" 이 아니라 "무엇을 읽었는가" 로 썼다.** 읽은 경로를 기록하는 store 를 끼워
    `list` 가 `node<N>/nodekey` 를 **한 번도 열지 않는지**를 본다. 출력만 검사하면 모든 키를
    읽고 나서 버리는 구현도 통과한다 — 그건 공개의 양을 재는 검사가 아니다.
    `--verify` 는 반대로 **반드시 읽어야** 한다(키에서 재파생하므로). `Show` 는 양쪽 모두
    아니어야 하고, `Export` 만 낸다.
  - 5건, 변이 3건으로 확인: `list` 가 다시 키를 읽게 하면 잡히고(오늘 고친 그 결함),
    `--verify` 를 키 없이 돌리면 잡히고, `Show` 가 개인키를 실으면 잡힌다.
    패키지 커버리지 0.0% → 34.4%.
  - **남은 공백 — 절반 채웠다 (2026-09-12).** compose 를 **끝까지** 지나는 테스트는 바이너리가
    필요해 여전히 없다. 대신 **그 앞의 게이트를 덮었다**: `internal/testengine` 에도 같은 판정
    밀도 순위를 적용하니 `RunSuite` 가 **57점**(분기 27·거부 15)으로 압도적 1위였고, 그 거부
    중 여섯 개는 **아무것도 기동하기 전에** 발동하므로 CI 에서 돈다.
    - 덮은 것: 스펙 없음·워크스페이스 없음·**파싱 실패한 스펙의 번호를 이름으로 대는지**·
      서로 다른 체인 거부·**서로 다른 구성 거부**(바이너리·토폴로지·하드포크 각각)·
      unresolved 참조 거부·경로 입력 형태.
    - **가장 값진 것은 `sameComposition` 이다.** 스펙들이 체인은 같은데 **다른 망**을 요구하면,
      첫 번째로 구성하고 둘 다 돌리는 것은 두 번째 스펙을 **자기가 묘사하지 않은 망**에 대고
      시험하는 것이다 — 틀린 이유로 통과한다. 세 요소를 각각 다르게 해 거부를 확인했다.
    - **게이트가 "다 거부하는" 안전한 종류가 아님도 고정했다**: 한 망을 묘사하는 두 스펙은
      게이트를 지나야 하고, 그 뒤 바이너리가 없어 실패해야 한다.
    - 변이 2건. **첫 시도는 제가 엉뚱한 필드를 건드려 안 잡혔다** — `env.binaries.default` 는
      `Chain.Binary`(단수)로 낮춰지므로 `Binaries`(복수)만 키에서 빼도 단수 쪽이 여전히 잡는다.
      키에서 바이너리를 **전부** 빼야 재현된다. 변이가 실패하지 않으면 코드가 옳은 게 아니라
      변이가 틀린 경우도 있다는 기록.
    - `internal/testengine` 51.1% → 59.8%.
  - **이어서: `cmd/chainbench/lifecyclecmd`(25.9%) — 파괴적 명령의 파서에서 결함 1건.**
    - **`clean --older-than` 의 `parseAge` 가 0% 였다.** `time.ParseDuration` 에 일·주 단위가
      없어 손으로 쓴 파서이고, 그 앞에 **디렉터리를 지우는 연산**이 있다. 위험이 비대칭이다 —
      값을 작게 읽으면 파일이 남고, 잘못 읽으면 작업물이 사라진다.
    - **결함: 0 이하의 나이를 통과시켰다.** `parseAge("-7d")` 가 -168h 를 돌려주고,
      `GCSessions` 는 `OlderThan <= 0` 을 **"나이 정책 없음"** 으로 읽는다. 그래서
      `--older-than -7d --keep-last 5` 는 운영자가 준 나이 조건을 **조용히 버리고** 날짜와
      무관하게 최근 5개만 남긴다 — 의도보다 많이 지운다. 값이 아직 운영자의 말인 자리에서
      거부하게 했다. 변이로 확인.
    - **함께: `verify` 의 agreement 출력 3분기를 덮었다.** 이 세션에서 제가 더한 코드인데 표면
      쪽이 비어 있었다. mock JSON-RPC 두 노드로 **운영자가 읽는 줄**을 검사한다 — 합의/분기/
      **검사 불가**. 세 번째가 핵심이다: 비교를 못 한 것이 합의로 읽히면 안 된다. 변이(검사 불가
      분기를 죽이기)로 확인.
    - `cmd/chainbench/lifecyclecmd` 27.9% → 44.9%.
  - **이어서: `internal/core/registry`(36.3%) — 인자 디코더에서 비대칭 1건.**
    - **`Arg*` 다섯 개가 0% 였다.** 모든 capability·MCP 도구 호출이 여기를 지난다. 실패 모양은
      조용하다 — 디코드되지 않는 값은 **기본값**이 되므로, 호출자가 준 인자가 쓰이지도 거부되지도
      않고 도구가 다른 일을 한다. 이 디코더들은 **거부할 수 없다**(def 반환이 계약이고, 거부는
      스키마의 일). 그래서 **무엇을 이해하는지가 곧 동작 전부**다.
    - **결함: `ArgInt` 은 숫자 문자열을 받는데(`"5"`→5) `ArgBool` 은 `"true"` 를 기본값으로
      떨어뜨렸다.** 같은 파일 안의 비대칭이고, `"all": "true"` 를 보낸 호출자는 플래그가 거부된
      것이 아니라 **무시된** 결과를 받는다. 거부할 수 없는 디코더에서 선택지는 "무시"와 "이해"
      둘뿐이고, 오해를 낳지 않는 쪽은 후자다. 문자열 `true`/`false`(대소문자·공백 무관)를 받게 했다.
    - **숫자는 받지 않는다.** 1/0 을 참거짓으로 읽는 것은 다른 언어의 관습이고 JSON 의 것이
      아니다. 빈 문자열이나 2 가 어느 쪽인지는 **읽는 것이 아니라 발명하는 것**이다. 변이로
      확인했다(숫자를 받게 하면 잡힌다).
    - 다섯 디코더 전부 표로 덮었다. `ArgStrings` 의 "비-문자열 원소는 건너뛴다"(패닉 대신)와
      `ArgBigInt` 의 "십진수만, 아니면 nil"(호출자가 '주지 않았음'을 알 수 있게)도 성질로 고정.
    - 변이 3건. 그중 둘은 **첫 시도가 적용되지 않았다** — 들여쓰기가 달라 치환이 빗나갔고,
      빌드 실패/통과로 위장됐다. 변이는 적용됐는지부터 확인해야 한다.
    - `internal/core/registry` 36.3% → 68.8%.
  - **이어서 (2026-09-12): 커버리지 숫자만으로는 어디를 칠지 못 정한다.** `internal/app` 이
    11.4% 인데 함수 135개 중 118개가 0% 다. 배선 계층이라 **그 대부분은 통과시키기만 하는
    함수**이고, 숫자를 따라가면 값 없는 곳에 테스트를 쓰게 된다.
    - **그래서 판정 밀도로 줄을 세웠다**: 0% 함수마다 본문의 분기(`if`/`switch`/`for`) 수와
      거부(`fmt.Errorf`/`errors.New`) 수를 세어 `분기 + 2×거부` 로 정렬했다. 거부에 가중치를
      두는 이유는, **틀렸을 때 실패하지 않고 잘못된 일을 하는 자리**가 거부가 많은 자리이기
      때문이다. 상위: `UpgradeRun`(23)·`AttachRun`(16)·`TxWait`(15)·`uploadDests`(13)·
      `transferSource`(11).
    - **`transfer.go` 의 목적지 결정을 골랐다** — 순수 함수라 I/O 없이 표로 덮이고,
      **파일이 남의 기계 어디에 쓰이는지**를 정한다. 업로드가 잘못된 경로로 풀리면 실패하지
      않고 **쓴다**.
    - 결정표 2개(각 6~7 케이스) + 성질 2개를 붙였다. 성질 쪽이 더 값지다: 업로드는
      **base name 만** 서버에 닿으므로 로컬 디렉터리 구조가 재현되지 않고 `../../../etc/passwd`
      가 `passwd` 로 납작해진다. 다운로드의 `--name` 은 용도 폴더를 벗어날 수 없다.
    - 변이 2건(`--remote` 가 여러 파일을 받게 하기·`Base` 대신 전체 경로 쓰기)으로 확인.
      `internal/app` 11.4% → 21.4%.
  - **이어서: `AttachRun`(순위 2위, 16점).** 먼저 **WA10 이 이미 고쳐져 있음을 확인**했다 —
    주석이 WA10 을 직접 인용하고, `Precheck` 이 돌며, DataDir 이 있으면
    `AttachWorkspaceRun`(readiness 게이트·실패증적·fault control·매니페스트 배선)로 간다.
    - **그래서 붙인 것은 그 수정을 구조적으로 고정하는 테스트다.** 두 분기를 가르는 것은
      **출력이 없는 라우팅 결정**이라, 누가 둘을 합쳐도 아무것도 실패하지 않고 증적 수집만
      조용히 멈춘다 — WA10 이 말한 실패가 정확히 그것이다. 그래서 **실패가 어느 쪽에서
      나왔는지**로 라우팅을 고정한다(워크스페이스 경로는 자기 접두사를 붙인다).
    - 4건 + 변이 2건: 워크스페이스 분기를 접으면 잡히고, `Precheck` 을 빼면 잡힌다.
      `internal/app` 21.4% → 22.9%.
    - **테스트를 쓰다 배운 것**: v2 케이스는 `env` 가 없으면 **파싱되지 않고**, 파싱되지 않은
      스펙은 설계상 precheck 에서 조용히 빠진다. 처음 쓴 스펙에 `env` 가 없어 precheck 을
      통과해 버렸다 — 의도된 간격이지만, 스펙 한 줄이 빠지면 검사 자체가 건너뛰어진다는 뜻이라
      테스트에 그 이유를 적어 두었다.
  - **이어서: `TxWait`(순위 3위, 15점).** 결함은 없었지만 **테스트가 하나를 찾아냈다.**
    - **이 함수의 실패 모양은 크래시가 아니다.** 대기 중(null) 영수증을 답으로 받으면
      `Status` 가 빈 **0 값 영수증**이 돌아가고, `Succeeded()` 가 false 라 호출자는
      **"트랜잭션 실패"** 로 읽는다 — 사실은 아직 안 들어간 것인데. 막는 곳이 두 군데다
      (클라이언트가 `null`→`nil`, `TxWait` 이 nil→계속 대기). 따로 바뀔 수 있어 호출자 쪽에서
      성질로 고정했다. 변이로 확인: 클라이언트의 정규화를 빼면 `{Status: BlockNumber: GasUsed:}`
      가 **오류 없이** 돌아온다.
    - **테스트가 찾은 것 — 대기가 자기가 말한 한도를 넘었다.** `Timeout: 1ms` 로 준 테스트가
      **1초** 걸렸다. 폴링 수면이 남은 예산과 무관하게 고정 1초였기 때문이다. 타임아웃을
      **단정으로** 쓰는 하네스에서("X 안에 들어와야 한다") 실제 한도가 최대 1초 더 길다는 것은
      단정이 말보다 약하다는 뜻이다. 수면을 남은 예산으로 깎았다. 변이(고정 1초 복원)로 확인.
    - 6건. `internal/app` 22.9% → 25.6%.
  - **이어서: `ContractDeploy`(순위 4위, 13점) — 공유 디코더에서 결함 2건.** 주소 계산은
    provider 에 위임하므로 그쪽엔 결함이 없었고, 결정은 입력 검증에 몰려 있었다. 그 검증이
    **있을 수 없는 입력 둘을 통과시키고 있었다.** 둘 다 크래시도 거부도 아닌 모양이다.
    - **`Wei("-5")` 가 오류 없이 -5 를 돌려줬다.** `SetString` 은 부호를 기꺼이 받는다. 그래서
      음수 금액이 노드까지 가서, **운영자가 타이핑한 액수가 아니라 인코딩에 대한 메시지**로
      거절됐다. 숫자가 아직 운영자의 말인 자리에서 거부한다. 이 함수의 doc 이 이미 "다른 진법으로
      조용히 읽히면 요청한 것과 다른 액수가 움직인다"고 적어 두었는데, **음수도 같은 부류**다.
    - **`--bytecode 0x` 가 아무것도 배포하지 않고 주소를 보고했다.** `in.Bytecode == ""` 검사는
      `"0x"` 를 통과시키고, 그것은 **0 바이트**로 디코드된다. 코드 없는 생성은 성공하고 주소를
      남긴다 — 끝난 것처럼 보이고 아무것도 배포하지 않은 배포다. 변이로 확인하면 증상이 그대로
      드러난다: 거부를 없애면 오류가 **연결 오류**로 바뀐다(= 닿는 노드였다면 성공했다).
    - `HexBytes("0x")` 자체는 그대로 뒀다 — 빈 calldata 는 정당하다. **바이트가 필요한 호출자가
      그렇게 말해야 한다**는 쪽으로 고쳤다.
    - 결정표 2개 + 케이스 2개. `internal/app` 25.6% → 28.2%.
  - **이어서: `NetworkPlan`(순위 5위, 12점).** 결함은 없었다 — 대신 **사용자가 말로 요구한 배치
    순서가 운영자가 쓰는 층에서 한 번도 단정된 적이 없었다**는 것을 채웠다.
    - 고정한 성질: 요청한 모양 그대로 나오는지(총수·역할별 수·모든 항목에 호스트와 포트가
      있는지), **모든 서버에 한 칸씩 주고 나서야 첫 서버로 되돌아오며 그때는 포트 대역이
      달라지는지**(= 15대에 30노드를 얹는 근거), validator 0 이하 거부, 미등록 체인 거부,
      **용량 초과를 조용히 잘라내지 않고 계산을 담은 메시지로 거부하는지**.
    - 변이 2건: `Inventory.free()` 의 순회를 호스트 우선으로 뒤집으면 `six nodes landed on
      1 host(s)` 로 잡히고, 용량 초과를 조용히 자르면 `total=1` 로 잡힌다.
    - **테스트 픽스처가 규격을 가르쳐 줬다**: p2p·rpc 만 선언한 서버 세트는 거부된다
      (`rpcStep is 1, want >= 3` — http/ws/auth 를 rpc 에서 파생하므로). ws·auth 대역을
      선언하면 step 1 이 정당해진다. `env/docker` 가 그렇게 쓰는 이유가 이것이다.
    - `internal/app` 28.2% → 31.7%.
  - **이어서: `UpgradeGenesis`(순위 6위, 13점) — 순위표의 마지막 실질 항목.** 결함은 없었고,
    **이 세션에서 실측으로 알아낸 성질이 아직 아무데도 단정돼 있지 않았다**는 것을 채웠다.
    - **병합이 무엇을 하는지가 프로파일 주석에 틀리게 적혀 있었던 그 자리다.** 주석은
      `extra_data` 가 validator 의 RLP 인코딩이라고 했지만, 병합은 후속 제네시스의 extraData 를
      **가져오지 않는다** — `config.<fork>` 섹션과 `<fork>Block` 만 들어올리고 나머지는 전부
      from-chain 의 것이다(extraData·alloc 포함). 그것을 성질로 고정했다: base 의 config 키 +
      `{croissant, croissantBlock}` **정확히 그만큼**, extraData 는 그대로, alloc 은 그대로.
    - **변이가 그 틀린 주석이 주장했던 동작을 그대로 재현한다.** 병합의 base 를 후속 제네시스로
      바꾸면 실패 출력에 **validator RLP extraData** 가 찍히고 `berlinBlock`·`pangyoBlock` 이
      섞여 들어온다. 즉 주석이 묘사한 것은 실제로 존재할 수 있었던 다른 구현이었다.
    - 나머지 성질: fork 섹션이 프로파일의 validator 집합을 담고 **BLS 키 수가 일치**하는지
      (어긋나면 제네시스는 통과하고 서명에서 실패한다), 노드 투영이 **프로듀서 먼저**이고
      network id 가 하나이며 포트가 0 이 아닌지. 변이(프로듀서를 뒤로) 로 확인.
    - 거부 3건(없는 프로파일·없는 from-genesis·미등록 to-chain)도 덮었다.
    - `internal/app` 31.7% → 34.3%. **판정 밀도 순위표의 상위 6개를 다 지났다** — 남은 것은
      라이브가 필요한 `UpgradeRun`(23) 하나다.
- [x] **WA24** [죽은 능력] — **재측정 (2026-09-12).** 목록의 9개 중 **대부분은 이미 해소됐고**, 남은 것은 성격이 다르다. 스펙을 JSON 으로 파싱해 `do`/`expect`/`source` 실사용을 세었다(설명 문구의 단어가 아니라).
  - **완료 (2026-09-12 실측). `hooks.onFail` 은 실행까지 테스트돼 있다.**
    `interp/run.go:63`(do 실패)과 `:84`(단정 실패) 두 경로가 `runRecorded(s.OnFailActions…)`
    를 부르고, `interp/run_test.go` 의 두 테스트가 **훅이 돌았는지를 단정한다**
    (`onFailRan`, :308 과 :347 — "onFail diagnostics must run after a do failure").
  - 즉 "통과하는 tc 케이스로는 도달할 수 없다" 는 맞지만, 그것은 **인터프리터 레벨에서
    덮인 이유**이고 열린 공백이 아니었다. 스위트에 일부러 실패하는 케이스를 두지 않아도 된다.
    재확인: `grep -n "onFailRan" internal/dsl/interp/run_test.go`
  - **이미 스펙이 있다 (5개)**: `faucet`·`registerContract`·`metric`·`createAddress`·`contractChecksum` — 전부 `tests/tc/go-stablenet/vocabulary/` 아래에 있다.
  - **`defaultOn` — 이번에 채웠다.** `tests/tc/go-wemix/vocabulary/01-default-on-routes-every-step`. 인터프리터 단위 테스트(`TestRun_DefaultOnRoutesStatements`)는 있었지만 **`chainbench run` 을 지나는 라이브 스펙이 없었다**. 단정을 구별되게 골랐다 — `admin_wemixInfo.self.name` 은 **노드마다 답이 다른 유일한 값**이라, 기본 타깃이 무시돼 `nodes[0]` 로 떨어지면 `node1` 이 나와 실패한다. `blockNumber`·`peerCount` 로 썼으면 어느 쪽이든 통과해 아무것도 증명하지 못했을 것이고, **조용히 무시되는 기본값이 살아남는 방식이 정확히 그것이다.** 변이(케이스 상위 `on` 제거)로 확인: `expected node3 actual node1`.
  - **훅도 같은 실행에서 돌았다.** `hooks.pre`·`hooks.post` 를 얹었고 세션 기록의 `postaction.json` 에 `{"Name":"waitBlock","OK":true}` 로 남는다. 첫 단정의 provenance 에 `"on": "node3"` 이 찍히는 것이 기본 타깃이 적용된 증거다.
  - **`boot` 는 죽은 어휘가 아니다.** `core/node/node.go:50` 이 **RoleBP 의 레거시 철자**라고 적어 두었다 — 별칭이지 쓰이지 않는 능력이 아니다. 목록에서 성격이 잘못 분류돼 있었다.
  - **남은 것 둘, 각각 이유가 다르다**: `hooks.onFail` 은 **실패할 때만** 돈다 — 통과하는 tc 케이스로는 도달할 수 없고(스위트에 일부러 실패하는 케이스를 둘 수 없다), 낮춤(lowering)은 `spec_v2_test.go` 가 이미 고정한다. 실행까지 보려면 인터프리터 레벨 테스트가 맞다. `placement`(= `env.target`)는 **WA19 와 같은 항목**이다 — 파싱은 되는데 배치를 하지 않고 fingerprint 만 바꾼다. 어휘 문제가 아니라 오배선이라 WA19 로 남긴다.
- [x] **WA14** [문법] 케이스 상위 `on`(DefaultOn) 라우팅 — **이미 고쳐져 있다 (2026-09-12 확인).** `interp/run.go:57` 이 매 statement 마다 `applyDefaultOn` 을 부르고, 명시적 `on`/`onEach` 가 있으면 비켜난다. 인터프리터 테스트(`TestRun_DefaultOnRoutesStatements`)가 WA14 를 지목해 고정하고 있다. 이번에 라이브 스펙까지 붙였다(위 WA24 참고).
- [x] **WA25** [커버리지] go-wemix(5건)·go-wbft(6건) 얕음. **해소 — 아래 §1p 의 `WA25. go-wemix·go-wbft 커버리지` 항목이 정본이다**(각각 9건·15건, 2026-09-11). 이 줄은 §1o 의 원본 표시이며 닫힌 것을 부분으로 보이고 있었다. **proxied 라우팅 스펙은 생겼다 (2026-09-11)** — `tests/tc/common/node/010-wbft-proxied-routing.json`. peer 수가 그 그래프의 서명이다: en1=1 · pn1=3 · bp1=bp2=2, 라이브 실측이 정확히 일치했다. **판별력도 확인**했다 — 같은 docker 에서 pn 없는 4노드 mesh 를 올리면 en1 이 3 을 보므로 `en1 == 1` 은 mesh 를 실제로 구분한다(공허한 검사가 아니다). 남은 것은 두 체인의 tx·fault·거버넌스 케이스 확충이다. 참고: **poa(go-wemix) 는 pn 을 거부**하므로(패밀리에 프록시 계층이 없다) proxied 라우팅 검증은 wbft 계열에만 성립한다.
- [x] **WA26** [문서] SPECS.md 가 없어진 `internal/testspec` 를 7곳 참조(드리프트). 증거: `tests/tc/SPECS.md:123,136,155,159,336,380`. 방향: `internal/testhelper`/`internal/testengine` 로 갱신한다. (2026-09-08 추가된 `docs/guide/dsl-authoring.md` 로 일부 해소 가능.) **확인(2026-09-12): 완료.** `grep -c internal/testspec tests/tc/SPECS.md` == 0


## 1p. WA 트랙 이후 — fleet 검증 결과와 남은 작업 (2026-09-08)

> **이 절은 이력이다 (2026-09-12).** 여기 적힌 항목은 전부 닫혔고, 표시도 맞췄다.
> **열린 작업은 §1s 에만 있다** — 같은 항목을 두 곳에 두면 한 곳이 낡고, 그 낡은 쪽이
> 읽히는 일이 이 문서에서 여섯 번 있었다(T5.2·T6.6·T5.5·WA14·WA19·그리고 이 절의 L1·L2).

WA 트랙은 PR #363 으로 main 에 머지됐다(WA1~WA26 중 코드/문법/표면 항목 반영). 이어
docker(15대, stablenet/wbft/wemix)에서 커버리지 스펙을 라이브로 돌려 다음을 확인했다.

**라이브 통과 확인**: createAddress·contractChecksum, faucet(amount 를 십진 wei 로 수정),
proxied pn 라우팅(keys preset 로 변경), registerContract, go-wbft tx·fault, go-wemix fault.
전부 preset 키만 쓴다 — 생성 키를 저장소에 남기지 않는다.

### 남은 작업

- [x] **C. metric 수집 인프라 — 완료 (2026-09-12 실측).** 머리글이 "라이브 검증만 남았다" 로
  남아 있었지만 본문이 이미 라이브 통과를 적고 있고(15대 docker, `chain_head_block=3`),
  라이브가 더 찾은 둘도 고쳐져 있다: `env/docker/gen-env.sh` 가 metrics 포트를
  퍼블리시하고(`METRICS_PUB_BASE`, 기본 16060) localmap 에 `6060: 1606N` 이 들어가며,
  경로는 `collector.MetricsURLOn` 이 붙인다(주소만 해석하던 `HTTPEndpoint` 와 분리). 되살린
  스펙도 자리에 있다(`tests/tc/common/rpc/CT-RPC-015-metric-head-block.json`).
  재확인: `grep -n "METRICS_PUB_BASE" env/docker/gen-env.sh` ·
  `grep -n "func MetricsURLOn" internal/core/collector/metrics.go`
  **(원래 기록)** 코드 경로는 열렸다 (2026-09-11), 라이브 검증만 남았다. 진단이 셋 중 하나 틀렸다 — ②수집 경로는 이미
  있었다(`collector.ScrapeMetrics`). 실제로 막던 것과 처리:
  - **바인드 주소**: `metricsHost` 기본값이 `127.0.0.1` 이라 노드 자기 기계에서만 닿았다.
    HTTP 와 같은 `0.0.0.0` 으로 맞췄다. 좁히려면 `metricsHost` 노브를 노드별로 준다.
  - **주소 번역**: 어서션이 `Host`+`Ports.Metrics` 로 직접 조립해 dial 했다. `node.Node` 에
    `MetricsURL` 을 더해 `RPCURL` 과 **같은 opener 를 지나게** 했다 — 번역은 이제 구성이
    기록하고 어서션은 읽기만 한다.
  - **config 없는 기동**: `Argv` 에 Metrics 모듈이 없어 핸드오프 재기동 노드는 metrics 가
    아예 없었다(같은 망이 포크 전에는 답하고 후에는 답하지 않았다). 포트가 배정된 노드면
    커맨드라인으로도 말하게 했다.
  - 제거했던 `03-metric-head-block` 스펙을 **되살렸고 라이브로 통과시켰다**(2026-09-11,
    15대 docker, `chain_head_block=3`).
  - **라이브가 유닛 검증이 놓친 것 둘을 더 찾았다.** ① `env/docker/gen-env.sh` 가 metrics
    포트 6060 을 퍼블리시하지도 localmap 에 넣지도 않았다 — `AddrMap` 은 매핑 없는 포트를
    **그대로 두므로** 호스트는 loopback 으로 바뀌고 포트는 6060 으로 남아 아무것도 답하지
    않았다. ② 기록된 `MetricsURL` 을 `HTTPEndpoint`(주소만 해석)로 만들어
    **`/debug/metrics/prometheus` 경로가 빠졌다** — 살아 있는 metrics 서버가 루트에서 404 를
    냈다. 유닛 테스트는 둘 다 잡지 못했다(전자는 docker 정의, 후자는 경로를 검사하지 않음).
- [x] **B — 부정 경로**: `go-stablenet/tx/01-negative-tx-revert` (revert 하는 런타임 배포
  후 `expect:revert`) fleet 검증 완료.
- [x] **B 잔여 — 거버넌스(체인 특화, fleet)** — **완료 (2026-09-12).** go-wbft 쓰기는 `tests/tc/go-wbft/governance/02-*`(PR #391), go-wemix(poa) 쓰기는 `tests/tc/go-wemix/governance/01-*`(PR #392). 아래 진단이 맞았다 — 옮겨 쓸 수 없었고, 각 체인의 컨트랙트로 따로 썼다. **(원래 진단)** go-wbft·go-wemix 거버넌스는 stablenet 을
  그대로 옮길 수 없다 — stablenet 의 GovValidator 흐름(0x…1001, proposeAddMember)을 go-wbft 로
  적응해 fleet 에서 돌리니 **2단계 proposeAddMember 가 revert** 했다(node1 이 gwbft 의 gov
  멤버가 아니거나 시스템 컨트랙트 셋업이 다름). 각 체인의 실제 거버넌스(컨트랙트 주소·선택자·
  멤버·정족수)를 확인한 뒤 작성해야 한다. go-wemix(poa)는 etcd/거버넌스 배포 경로라 더 다르다.
  registerContract 도 실제 메서드/ABI 를 쓰는 케이스는 여기 포함(현재는 임의 호출 컨트랙트로 최소 검증).
  negative-tx 의 reject(제출 거부) 변형도 남아 있다(신뢰할 수 있는 재현 방법 확정 필요).
- [x] **D. WA1 — 라이브 MCP 플러그인 — 해소 (2026-09-12).** 원인은 낡은 빌드가 아니라
  **심볼릭 링크가 다른 저장소를 가리키는 것**이었다. 아래 §1s D 에 실측과 조치가 있다.
  **(원래 진단, 틀렸다)** 배포본이 `net_*` 이름의 stale 빌드다. 저장소는
  `chain_*` 로 개명됐으니 재빌드·재배포만 하면 맞는다. 코드가 아니라 배포 작업이다.
- [x] **E. WA21 — DSL 문법 확장(D4 로 미룬 것)**. env 에 `blueprint`(레이아웃+키 한 문서)와
  `keys.nodekeys.validators`(생성 키 중 N개만 validator)를 더해 compose 로 배선했다(단위 테스트).
  **deploy-only stage 는 넣지 않았다** — 테스트 케이스는 어서션을 돌리므로 항상 망을 기동해야
  한다(UpStart). 배포만 하는 compose 는 테스트 env 의미가 없다. blueprint/validator-subset 를
  실제로 쓰는 라이브 스펙은 다른 커버리지처럼 fleet 에서 작성·검증하면 된다.
- [x] **WA10 잔여 없음** — attach 경로 게이트·증적(E6·E8)까지 반영됨(PR #363). 라우팅을 구조적으로 고정한 테스트는 PR #400.

기존 §1n 의 R6(go-wemix etcd collapse — 체인팀), G2(핸드오프 원격), 원격 `chain rm`
(filestore Remove 부재)은 WA 와 무관하게 그대로 남아 있다.


## 1q. workspace-config 트랙 이후 — 남은 두 항목 (2026-09-10)

> **이 절은 이력이다 (2026-09-12).** 두 항목(L1·L2) 모두 닫혔다. 둘 다 **원래 진단이
> 빗나갔던** 경우라 진단을 남겨 뒀다 — L1 은 attempt 축이 아니라 원격의 로그 덮어쓰기였고,
> L2 는 compose 의 다운로드가 아니라 링 인덱스가 개인키를 담은 것이었다. 상세는 §1s.

workspace-config(W1~W6, PR #369)와 그 후속(런타임 validator 검사·정의서 순차 실행·
승인 기준·다이얼 주소 레이어 정리, PR #370)은 반영됐다. 인계 문서
`docs/research/chainbench/analyses/10-prepared-inputs-server-ref-handoff.md` 에서 나온
항목 중 **둘은 아직 코드가 없고, 지금까지 이 작업 리스트에 적혀 있지 않았다.** 연구
문서에만 남아 있으면 사라지므로 여기 옮긴다.

- [x] **L1 — 완료 (2026-09-12, PR #389).** 진단이 옮겨졌다: attempt 축이 없는 것이 아니라 **원격 launch 가 로그를 덮어쓰는 것**(로컬은 `O_APPEND`, 원격은 `>`)이 문제였다. 원격도 덧붙이고 launch 마다 마커를 쓴다 — 경로 축을 만들지 않고 파일 안에서 시도를 가른다. 상세는 §1s. **(원래 진단)** 지금 노드 로그는 노드당 한 파일
  (`logs/<compId>/<label>.log`)이라 **재기동하면 이전 실행의 로그가 덮인다**. 실패한
  시도의 증적이 다음 시도로 지워지는 것이 문제다 — 특히 reuse-if-matching 이 한 노드를
  여러 번 재작업할 때. 인계 문서와 `workspace-config.sample.yaml` 의 주석은
  `logs/<compId>/<nodeId>/<attemptId>.log` 를 예고하지만 `core/node/layout.go` 에
  attemptId 개념이 없다. **필요한 것**: layout 에 attempt 축 추가, 기동마다 증가시키는
  소유자 결정(런 원장이 유력), 실패 수집(`collectFailureData`)이 해당 시도의 로그를
  고르도록 배선. **게이트**: 한 노드를 두 번 재기동한 뒤 두 시도의 로그가 모두 남아 있고,
  실패한 테스트의 관측 폴더가 그 시도의 것을 담는다.

- [x] **L2 — 완료 (2026-09-12, PR #394·#397).** 진단이 옮겨졌다: 문제는 compose 의 다운로드가 아니라 **링 인덱스가 모든 개인키를 담고 있어 인덱스 한 번 읽기가 링 전체를 공개**한 것이었다. 인덱스에서 키를 뺐고, 키가 필요한 세 호출자만 노드별 파일을 읽는다. 상세는 §1s. **(원래 진단)** 인계 문서 §7 은 "준비된 키의 검증은
  가능한 한 대상에서 수행하고 공개 신원만 반환한다"고 정했다. 현재 srv:// keyring 은
  **키 묶음 전체를 로컬로 내려받아** 쓴다(`materializeKeyring`, PR #370). 이는 사용자가
  명시적으로 승인한 경로이고 런타임 서명에 로컬 경로가 필요해서 정당하지만, §7 이 말한
  "대상에서 검증" 은 아직 없다. **필요한 것**: 대상 서버에서 키를 열어 주소·공개키만
  돌려주는 경로(다운로드 없이), 그리고 그 결과로 genesis validator 대조. 다운로드가 필요한
  경우(서명)와 검증만 필요한 경우(대조)를 구분하는 것이 요점이다. **게이트**: 원격 키 묶음을
  로컬에 내려받지 않고 공개 신원 목록을 얻어 genesis 와 대조한다.

두 항목 모두 지금 동작을 막지 않는다 — L1 은 증적 보존, L2 는 키 취급 범위를 좁히는
일이다. 착수 전에 인계 문서 §7·§8 을 함께 읽을 것.


## 1r. 모니터링 이슈 재검토와 수정 계획 (2026-09-10)

모니터링 세션이 제기한 16건(MON-001~016,
`docs/research/chainbench/analyses/11-monitoring-open-issues.md` (2026-09-24 삭제))을 HEAD `2cc82692` 기준으로
다시 판정했다. 판정 근거·추가 발견·검증 계획은 정본
`monitoring-issue-review-2026-09-10.md` (2026-09-24 삭제)에 있다.

결과: **유효 14건, 해결·검증 완료 2건(MON-004·MON-006), 오탐 0건.** 기준 시점의 전체
테스트는 통과 상태였으므로, 남은 14건은 모두 기존 테스트가 잡지 못하는 결함이다.
문서에 없던 추가 발견 5건(N1~N5)도 정본에 적었다.

작업은 작은 모듈에서 상위 조합, 표면 순서로 진행한다. 각 항목의 완료 기준은 정본의
6절, 검증 방법은 7절을 따른다.

- ☑ **MR-A. 프리미티브·합의 패밀리** (`c6b8b091`)
  - A1 MON-012 `internal/consensus/wbft` RLP 길이 범위 검사 + fuzz
  - A2 MON-014 `internal/consensus/poa` 멤버 수 uint256 검증·상한
  - A3 MON-003 `internal/dsl` null·비객체 env 거부
  - A4 MON-011·N1 `internal/core/filestore` 로컬 Write가 mode 강제(심볼릭 링크 포함).
    비밀 쓰기 여러 곳의 공통 원인이므로 호출부마다 고치지 않고 스토어에서 고친다.
- ☑ **MR-B. 키와 비밀** (`2246f14e`) — 결정 ①② 승인 반영
  - B1 MON-001 인라인 개인 키가 `State.Nodes[].Key`·`State.Request`에 남지 않게
  - B2 MON-002 명시 키와 기존 keyring 신원 불일치를 자원 변경 전에 거부(재사용 자체는 유지)
- ☑ **MR-C. 구성 오케스트레이션** (`internal/chainsetup`) — C1 `e6641505`, C2~C4 `72e7ca63`, C5 `d370597d`
  - C1 MON-009 후보 생성과 실행 경로 반영 분리 — 거부 시 파일·해시·PID 보존
  - C2 MON-010 발견 결과에 서버·전체 datadir 보존(구성 간 label 충돌 제거)
  - C3 MON-008·N2 `recordRun`이 실제 GenesisPath 사용 + 노드 config 수집
  - C4 MON-007 existing genesis와 변경 옵션 충돌 거부(capability 산출 포함)
  - C5 MON-016 baseline 관측 시점·누락 처리 정정
  - C1·C5는 "대상의 현재 파일을 읽어 해시" 기능을 공유하므로 공용 헬퍼 하나로 만든다.
- ☑ **MR-D. 표면·환경**
  - ☑ D1 MON-015·N3 `--json` stdout 계약과 순차 실행 종료 코드 (`edb05eb8`)
  - ☑ D2 MON-005·N5 `env/docker/gen-env.sh`와 README를 새 server-set 계약에 맞춤 (`d210ac93`).
    생성기가 workspace-config 와 server-set-wemix 까지 찍고, 생성물을 실제 파서로 읽는
    테스트를 붙였다. 로컬 `build/` 는 손으로 고쳐 놓은 상태였고 생성기만 옛 형식을
    쓰고 있었다 — 새 체크아웃에서만 깨지는 모양이었다.
  - ☑ D3 MON-013 샘플 안내를 실제 소비 범위와 일치 (`21d47abe`).
    `binaryAliases` 는 파싱만 되고 읽는 곳이 없으며, 객체형 참조는 아예 로드에
    실패한다. 두 경우를 갈라 적고 테스트로 고정했다.
  - ☑ D4 N4 data-root 충돌 규칙을 `internal/resource` 한 곳으로 (`35735508`)

- ☑ **Docker 라이브 검증** (컨테이너 15대) — 7절 다섯 시나리오 모두 통과. 검증 중
  포트 충돌 메시지가 다른 워크스페이스의 노드를 자기 것으로 부르는 결함을 새로 찾아
  같이 고쳤다(`9d0109e3`). 상세는 정본 9절.

6절 결정 6건은 승인 완료(정본 8절). 남은 것은 PR 하나.


## 1s. 남은 작업 한눈에 (2026-09-10 작성 · 2026-09-11 재측정 · **2026-09-24 갱신**)

§1n 부터 §1r 까지 트랙마다 흩어져 있던 미완 항목을 한 곳에 모았다. 각 항목의 근거와
배경은 원래 절에 그대로 두고, 여기서는 **무엇이 남았고 왜 남았는지**만 적는다.

### 지금 열린 것 (2026-09-24)

아래 G 절이 정본이다. 9월 12일과 9월 22일 사이에 두 트랙이 돌았고, 이 문서는 그동안
따라오지 못했다 — 9월 11일에 멈춰 있었다. **이 절을 읽는 사람이 먼저 볼 것은 G 다.**

| 갈래 | 상태 |
|---|---|
| G. HSM 리팩토링 (2026-09-21~) | **검증 끝. DSL 208/209 PASS · e2e 계층 통과 · 회귀 0건** |
| G6. 15노드 poa endpoint 영구 합류 실패 | 열림 · **원인 확정 — go-wemix 코어 결함, 체인팀 인계 대기** |
| ~~G7. `TestRemoteDriver_E2E` 의 fixture 가 저장소에 없다~~ | **닫힘 (2026-09-24) — 테스트가 자기 바이너리를 들고 간다** |
| ~~G3. `preset.KeysDir` 의 기준이 문서와 다르다~~ | **닫힘 (2026-09-24) — 주석을 사실에 맞추고 실패가 기준을 말한다** |
| ~~G5. 죽은 `.env.json` 필터 넷~~ | **닫힘 (2026-09-24)** |
| ~~G4. `upgrade run` e2e 20건~~ | **닫힘 (2026-09-24) — 21 PASS · 0 FAIL** |
| ~~G4-a. overlay 가 fork 섹션에서 버려짐~~ | **닫힘 (2026-09-24) — 순서 원복** |
| B~F 절의 잔여 | 9월 12일 기록 그대로. 다시 재지 않았다 |

**G6 만 이 브랜치 밖의 일이다** — go-wemix 의 블록 검증 안이다. G4·G4-a·G7 은 전부 앞선
리팩토링이 남긴 것이고(#419가 순서를 뒤집고 명령을 지웠으며, #130이 sshd 하네스를 지웠다)
여기서 되돌렸다.

### 재측정 (2026-09-11, PR #383 머지 후)

`main` 기준으로 이 절의 열린 항목을 하나씩 코드에 대조했다. 세 가지가 달라졌다.

| 항목 | 적혀 있던 것 | 실제 (2026-09-11) |
|---|---|---|
| WA25 커버리지 | go-wemix 5건·go-wbft 6건, proxied 라우팅 스펙 없음 | **8건·11건**, `go-wbft/network/01-wbft-proxied-routing.json` 으로 **pn/proxied 는 해소** |
| D. MCP 재배포 | 라이브가 `net_*` 이름의 낡은 빌드 | 라이브도 `chainbench_*` 이지만 **다른(더 오래된) 집합**이다 — 진단 문구를 아래에서 고쳤다 |
| `binaryAliases` | 리졸버는 있고 호출자가 테스트 둘뿐 | 변화 없음 (`BinaryPath` 호출부는 여전히 `workspaceconfig_test.go` 2곳뿐) |

나머지 열린 항목(B 거버넌스 케이스, L1, L2, 후보·반영 분리, 통합 report, R6, validatorset
홈, health 재배선)은 **그대로 열려 있다**. 확인 방법은 각 항목에 적었다.

**착수 순서 제안.** 갈래끼리 서로 막지 않으므로 순서는 강제가 아니지만, 지금 시점에서
값이 큰 순서는 이렇다.

1. ~~**R7 (epoch 재결정 경로)**~~ — **완료 (2026-09-11).** 라이브로 재결정을 일으켜
   관측하고 상설 e2e 로 고정했다. 상세는 §1s E 의 R7 항목.
2. **D. MCP 플러그인 재배포** — 코드가 아니라 배포다. 저장소 58개 도구 중 라이브가 못
   보는 것이 절반이라, 이걸 두면 MCP 표면 작업의 검증이 전부 겉돈다.
3. **WA25 잔여 + B 거버넌스 케이스** — 커버리지. docker 가 서 있는 동안 해야 싸다.
4. **L1 / L2 / 후보·반영 분리 / 통합 report** — 증적·운영 품질. 지금 동작을 막지 않는다.
5. **R6 / validatorset 홈 / health 재배선** — 각각 체인팀 몫, 설계 결정, 구조 정리.

### 묶음 (PR 단위, 2026-09-11)

항목 하나에 PR 하나는 비효율이다. 아래는 **같은 코드 면을 건드려서 함께 검증하는 편이 싼**
단위로 묶은 것이다. 묶음 안에서는 커밋을 나누되 PR 은 하나로 낸다.

| 묶음 | 담긴 항목 | 왜 한 묶음인가 | 전제 |
|---|---|---|---|
| **G-A. 커버리지** — ☑ 완료 (2026-09-12) | WA25 ☑ · B 잔여(체인별 거버넌스) ☑(2026-09-12: 실제 ABI `registerContract` + reject 이유 못 박기까지) · `istanbul_getWbftExtraInfo` 태그 관측 ☑(가이드에 기록) | 전부 `tests/tc` 스펙과 라이브 실행이다. 같은 docker 를 한 번 세워 한꺼번에 돌린다 | docker 15대 |
| **G-B. 증적** — ☑ 완료 (2026-09-11) | L1 ☑(원격 로그 truncate 수정) · 통합 report ☑(`report --all`) · 후보·반영 완전 분리 ☑(표적 정지를 증명) | 셋 다 "실행이 무엇을 남기는가" 다 | 없음 |
| **G-C. 대상 경계 잔여** — ☑ 완료 (2026-09-12) | L2 ☑(전송까지 제거) · `binaryAliases` ☑(객체형 참조는 철회) · T3.3·T5.1 실 SSH 라이브 e2e ☑ · T2.1 ☑ 철회 | 전부 "로컬 가정이 남은 대상 경계" 라는 한 가지 경향이다. 이 트랙이 지금까지 찾은 결함이 전부 그 모양이었다 | docker 15대 |
| **G-D. 구조 정리** — ☑ 완료 (2026-09-12) | validatorset ☑(패밀리가 자기 지식을 소유) · health→inspector ☑ 철회(공통 기반 없음, 래칫으로 고정) | 둘 다 소유 모듈 결정이고 소비자가 적다 | 없음 |
| **G-E. 이관** — ☑ 완료였음 (2026-09-12 확인) | T5.5 wemix4 이관 — 표시만 낡았고 47건 전부 ported | 실제로 남은 이관 작업은 없다 | — |

**여기서 할 수 없는 것 — 둘 다 틀렸다 (2026-09-12).** `D. 라이브 MCP 플러그인` 은 배포가
아니라 **심볼릭 링크가 다른 저장소를 가리키는 것**이었고 여기서 고쳤다(아래 D 항목).
`R6. go-wemix boot-etcd collapse` 는 체인팀 몫이 맞지만 "넘겼다" 로 끝낼 일이 아니었다 —
소스를 읽으니 범위를 한 질문으로 줄일 수 있었다([[chain-handover-2026-09-12]] §2).
**남은 교훈**: "여기서 할 수 없다" 는 분류는 원인을 짚은 뒤에만 쓸 수 있다. 두 항목 모두
원인을 짚지 않은 채로 그 칸에 들어가 있었고, 짚어 보니 둘 다 할 일이 있었다.

### A. 지금 진행 중

- ~~**모니터링 트랙(§1r)**~~ — **머지 완료 (`9b6b0930`).** 재검토를 두 번 거쳤다.
  1차에서 **MON-001 이 미해결이었음을 확인하고 고쳤다**(인라인 키가 상태 파일·stderr·
  `--json` stdout 세 곳으로 샜다). 2차에서 **MON-017 을 새로 확인해 고쳤다** — 노드별
  바이너리 이름이 다르면 실행 중인 노드를 발견하지 못했고, Docker 에서 전후를 대조해
  확인했다. **MON-015 의 단일 실행 분기**도 복수 실행과 답이 달라 맞췄다. MON-007·010 은
  동작은 맞았고 근거를 넓혔다. 3차에서 **MON-018** — 그 키 노출 회귀 테스트가 문법 오류로
  끝나 검사에 닿은 적이 없었음을 확인하고 바로잡았다(프로덕션 결함은 없었다). 상세와 검증
  수준 구분은 정본의 재검토 절들.

### B. fleet 커버리지와 관측 (§1p)

라이브 docker 가 있어야 진행되는 갈래다. **docker 는 2026-09-11 에 섰다** — 15대 docker,
세 체인 패밀리 15노드 스모크가 모두 통과했다(stablenet · wbft · wemix(poa)). 체인
바이너리는 세 저장소를 `golang:1.23-bookworm` 컨테이너에서 linux/arm64 로 빌드했다
(호스트가 macOS 라 `CGO_ENABLED=0` 크로스컴파일은 blst 에서 실패한다).

- [x] **C. metric 수집 인프라** — **라이브 검증 완료 (2026-09-11).** 15대 docker 에서
  `03-metric-head-block` 통과(`chain_head_block=3`). 상세는 §1p C. 라이브가 코드 검증이
  놓친 결함 둘을 더 찾았다 — `gen-env.sh` 가 metrics 포트를 퍼블리시·매핑하지 않았고,
  기록된 `MetricsURL` 에 Prometheus 경로가 빠져 있었다(둘 다 수정).
- [x] **B 잔여 — 체인별 거버넌스 케이스 — 완료 (2026-09-12).** 쓰기 2건에 이어 남아 있던 둘(실제 ABI `registerContract`, reject 변형)도 닫았다. 상세는 이 항목 끝의 "남은 둘 다 완료".
  **(원래 기록)** go-wbft 쓰기 완료 (2026-09-12).
  `governance/02-wbft-governance-register-staker` — 읽기 케이스(`governance/01`)가 제네시스가
  무엇을 들고 있는지를 고정한다면, 이것은 **거버넌스 상태를 바꾸고 체인이 동의하는지**를 본다.
  - **stablenet 흐름을 옮기지 않고 go-wbft 자신의 컨트랙트로 썼다** — 이 항목이 경고하던
    그대로다. `GovStaking.registerStaker` 는 `msg.sender != _staker` 와 operator 1인 1스테이커를
    강제하므로, 자기 키를 자기 노드에서 서명할 수 있는 node1 이 node2 의 신원을 등록한다.
    BLS 공개키·PoP 는 `presets/keys` 픽스처에서 가져온다.
  - 단정: 쓰기 **전** `isStaker(node2) == false`, 쓰기 **후** `true`, 그리고 **다른 노드(node3)에서도**
    `true`. 라이브 통과. **변이**로 `sendTx` 를 빼면 정확히 **2건**(node1·node3 의 사후 단정)이
    실패한다 — 상태를 바꾼 것이 그 쓰기였고, 두 노드가 같은 답을 한다는 뜻이다.
  - **go-wemix(poa) 쓰기도 완료 (2026-09-12).**
    `go-wemix/governance/01-wemix-governance-staking-deposit` — 형태가 정말 다르다:
    **wbft 는 거버넌스 컨트랙트가 제네시스 고정 주소(0x1000~)에 있지만, wemix 는 etcd/거버넌스
    부트스트랩 중에 배포하므로 주소가 실행마다 다르다.** 그래서 스펙이 쓰기 전에 체인에게
    **어디에 staking 이 있는지 물어야** 한다 — `read` 의 `rpcCall` source 로
    `admin_wemixInfo` 의 `staking` 을 뽑아 `$staking` 으로 쓴다.
    - `Staking.deposit()` 은 payable 이고 `nonReentrant`·`notRevoked` 만 걸려 있다(투표 없음).
      wbft 의 `registerStaker` 에 해당하는 자리다.
    - **사전 단정이 쓰기만큼 값어치 있다**: 프로듀서의 스테이크는 부트스트랩에서 **전부
      잠긴다**. 그래서 `availableBalanceOf` 가 0 에서 시작하고, 그것을 움직일 수 있는 것은
      이 입금뿐이다. 0 → 정확히 1e18, 그리고 **다른 노드(node3)에서도** 같은 답.
    - 변이(`sendTx` 제거)로 정확히 2건이 실패한다.
    - **공허한 단정 하나를 빼면서 배운 것**: 처음에 `balanceOf(node1) > 1 ether` 를 넣었는데,
      부트스트랩 스테이크가 이미 ~1.5e24 라 **입금과 무관하게 항상 통과**했다. 변이가 3건이
      아니라 2건만 실패하는 것으로 드러났다 — 증거처럼 보이지만 증거가 아닌 검사다. 뺐다.
  - **남은 둘 다 완료 (2026-09-12).** 둘 다 "이름이 말하는 것을 검사하지 않고 있었다" 였다.
    - **`registerContract` 실제 ABI** — `go-stablenet/vocabulary/03-register-contract` 를 고쳤다.
      전에는 **아무 호출이나 받고 상수를 돌려주는 스텁**을 배포해 "revert 하지 않았다" 만
      단정했다. 4바이트 아무거나로 통과하므로 **액션을 태웠을 뿐 등록을 증명하지 않았다.**
      이제 `register(address)`·`registered(address)`·`count()` 를 가진 NodeRegistry 를 배포하고
      **되읽는다**: 전 0 · 후 1 · **다른 노드(node3)도 같은 답** · `count == 1` ·
      같은 주소 재등록은 컨트랙트 자신의 `require` 로 revert. solc 0.8.28 로 컴파일한 바이트코드를
      인라인해 저장소에 컴파일러 의존을 만들지 않았고, 소스와 선택자 3개를 설명에 적었다.
      변이(등록 스텝 제거)로 사후 단정이 0 을 보며 실패.
    - **negative-tx 의 reject 변형은 이미 있었다 — 문제는 다른 것이었다.** 실측: reject 스펙이
      go-stablenet 16건(158건 중), go-wbft·go-wemix 각 1건이다. 즉 "변형이 없다" 는 낡은 기록이다.
      실제 결함은 **`expect: reject` 에 `reason` 이 없으면 어떤 거절이든 통과한다**는 것이었다.
      세 건(`go-stablenet/regression/ethereum/14`, `go-wbft/tx/02`, `go-wemix/tx/02`)이
      "잔액 부족으로 거절" 을 주장하면서 **아무 이유의 거절이나 받고** 있었다. `reason` 을
      `insufficient funds` 로 못 박았다.
      - **추측하지 않고 탐침했다**: 일부러 틀린 `reason` 을 넣으니 스텝이 실패하며 실제 메시지를
        찍었다 — `insufficient funds for gas * price + value: balance 10000000000000000000000000,
        tx cost 10000002419600000000000000`. 같은 상수가 세 포크의 `core/error.go` 에 동일하게
        선언돼 있고, 세 스펙이 각자의 바이너리에서 통과하는 것이 그 부분 문자열이 거기서도
        맞는다는 증거다.
- [x] **F5 — 완료 (2026-09-12). 그리고 이것이 하네스 결함 하나를 덮고 있었다.**
  10건에 탐침을 돌려 실제 문구를 읽었다. **8건은 이름과 맞아** 못 박았다 —
  `transaction underpriced`(수수료 하한 4건), `exceeds block gas limit`(2건),
  `insufficient feePayer's funds`(fee-delegation 2건). `tests/tc` 에 `reason` 없는
  `expect: reject` 는 **0건**이고, 10건이 한 배치에서 함께 통과한다.
  - **system-contracts 2건은 두 결함이 상쇄되어 초록이었다.** 호출자에게 **10 ether** 를
    주고 **15.04 ether** 짜리 tx 를 보내므로 노드가 *잔액 부족*으로 거절했고, 이 스펙들이
    존재하는 이유인 **접근 제어에는 닿은 적이 없다.** 제대로 펀딩하면 노드가 두 tx 를
    **받아들인다** — 거부는 제출이 아니라 **실행** 시점이므로 `expect: reject` 자체가 틀린
    종류였다. 둘 다 `expect: revert` 로 바꿨다.
    - 변이로 revert 의 원인이 접근 제어임을 확인: `configureMinter` 를 **거버넌스 멤버**로
      부르면 성공하고, blacklist 호출 대상을 평범한 주소로 바꾸면 성공한다.
    - 멤버 변이의 첫 시도는 하네스가 `sends from a node account through a different node`
      로 거부했다 — 프리체크가 제대로 동작한 것이다(노드 계정은 키를 가진 노드만 서명한다).
  - **fee-delegation 2건의 첫 탐침 결과는 제 측정 오류였다.** `transaction type not supported:
    type 22 rejected, pool not yet in Applepie` 는 스펙 잘못이 아니라 **아래 F6** 이었다 —
    두 스펙은 Applepie 오버레이를 선언하는데 그것 없이 구성된 네트워크에서 돌고 있었다.
    고친 하네스에서 다시 재면 이름대로 fee payer 잔액을 말한다.
- [x] **F6. 스펙이 선언하지 않은 체인에서 돌고 있었다 — 해소 (2026-09-12).** [치명]
  F5 의 탐침이 찾아냈다. **`expect: reject` 의 공허함이 하네스 결함을 덮고 있었다** —
  선언한 포크가 없는 체인에서 tx 가 "지원하지 않는 타입" 으로 거절되었고, 그것이 스펙이
  주장하는 "fee payer 잔액 부족" 의 증거로 통과했다.
  - **최소 재현**: 오버레이 없는 스펙 1건 → 오버레이 있는 스펙 1건을 **한 워크스페이스**에서
    연달아 돌린다. 두 번째가 첫 번째의 체인을 본다. 단독 실행과 답이 다르다.
  - **결함 3개, 전부 같은 모양이다 — 장치는 있는데 한쪽이 채워지지 않았다.**
    1. **preflight 가 genesis 를 비교한 적이 없다.** `Have` 는 genesis 바이트 해시를 채우고
       `Want` 는 비워 두는데, 비교는 **어느 한쪽이 비면 건너뛴다** — 즉 작성된 날부터 무력했다.
       이제 양쪽이 **선언된 genesis**(chain id·오버레이·dot-path set·template·manifest·
       기성 genesis)를 같은 방식으로 다이제스트한다. 빌드된 바이트가 아니라 선언인 이유는
       비교가 genesis 스텝보다 먼저 일어나기 때문이고, 바이트 쪽은 launch-input 해시가 본다.
       `Have` 가 같은 언어를 쓸 수 있는 것은 **요청이 기록돼 있기 때문**(F1)이다.
    2. **오버레이 파일 이름이 하나로 고정이었다.** 여러 스펙이 워크스페이스를 공유하므로
       두 번째 오버레이가 첫 번째를 덮어썼고, 앞 환경의 **기록된 요청이 뒤 환경의 바이트를
       가리켰다** — 구성의 자기 기록을 믿을 수 없는 유일한 자리였다. 이름에 내용 해시를 붙였다.
    3. **가장 나쁜 것: rebuild 가 rebuild 를 하지 않았다.** 비교를 고치니 판정이
       `rebuild-all: genesis differs` 로 정확히 나왔고 **아무 일도 일어나지 않았다** —
       init·start 는 **pid 가 기록된 노드를 건너뛰므로**, 두 번째 구성이 genesis 를 디스크에
       다시 쓰고 노드는 전부 옛 체인을 계속 서비스했다. 실측(고치기 전): 워크스페이스 genesis 는
       `applepieBlock: 0` 인데 **네 노드 전부 `Applepie: #<nil>`, 기동 1회**. 이제
       rebuild-all 앞에서 네트워크를 멈추고, 같은 실행이 **노드당 기동 2회**, 두 번째가
       `Applepie: #0` 을 보고한다.
  - 단위 테스트: 다이제스트(무엇이 갈라져야 하고 무엇은 아닌지·set 의 순서·길이 접두사),
    preflight 판정(다르면 rebuild·같으면 여전히 reuse·**어느 한쪽이 비면 일부러 rebuild 하지
    않는다** — 옛 워크스페이스를 살리는 의도된 맹점), 오버레이 파일명. 각각 변이로 실패 확인.
    **오버레이 변이는 첫 시도가 빌드만 깨뜨려서 다시 했다** — 그건 아무것도 증명하지 않는다.
  - **stop-before-rebuild 는 스텁이 아니라 라이브로 증명했다.** 이 작업 앞부분에서 스텁
    라이프사이클 테스트가 **통과하면서 아무것도 증명하지 못한** 선례가 있다(스텁은 RPC 에
    답하지 않으므로 모든 노드가 엉뚱한 이유로 redo 로 분류됐다). 여기서는 노드 로그가 증거다.
  - **남은 맹점(의도적)**: 요청이 기록되기 전에 구성된 워크스페이스는 다이제스트가 없어
    비교를 건너뛴다. 그것을 rebuild 로 만들면 옛 빌드로 구성한 모든 네트워크가 무너진다.
    테스트가 그 의도를 못 박고 있다.

- [x] **WA25. go-wemix·go-wbft 커버리지 — 해소 (2026-09-11).** 각각 **9건·15건**이 됐고
  (시작은 5건·6건), 비어 있던 축을 전부 채웠다: pn/proxied 라우팅, 정족수 경계(4노드·15노드),
  제네시스 거버넌스(wbft·poa 각각), 15노드 엣지 제출. 아래는 그 과정의 기록이다.
  **(원래 진단)** **부분 해소 (2026-09-11 재측정).** 각각
  **8건·11건**으로 늘었고(go-stablenet 은 156건), 없다고 적혀 있던 **pn/proxied 라우팅
  스펙은 생겼다** — `tests/tc/common/node/010-wbft-proxied-routing.json`(bp 2·pn 1·en 1,
  네 노드의 peerCount 를 각각 단정한 뒤 blockAdvance). 남은 것은 **깊이**다: 두 체인 모두
  fault 가 노드 크래시 1건뿐이고, 15노드 스펙은 chain-up 뿐이라 규모에서의 tx·합의 거동을
  보는 스펙이 없다. 확인: `find tests/tc/go-wemix tests/tc/go-wbft -name '*.json' | wc -l`.
  - **진행 (2026-09-11, G-A).** wbft 쪽 깊이 3건을 라이브로 채웠다 — 정족수 경계
    (`consensus/02`), 제네시스의 거버넌스 컨트랙트(`governance/01`), **15노드 proxied 에서
    엣지 제출**(`tx/04`). 셋 다 변이로 실패 가능함을 확인했다.
  - **`tx/04` 를 만들며 라이브가 가르쳐 준 것 둘** (둘 다 `docs/guide/dsl-authoring.md` 에
    적었다):
    1. **키를 갖지 않은 노드에서는 보낼 수 없다.** `from` 에 노드 계정을 적으면 그 노드가
       `eth_sendTransaction` 으로 서명하므로, en1 에 faucet 으로 보내달라고 하면
       `unknown account` 로 실패한다. 메시지가 키스토어 이야기라 토폴로지 문제로 오해하기 쉽다.
       엣지 제출은 `sendTx` 의 `key` 인자(로컬 서명 → `eth_sendRawTransaction`)로 한다.
    2. **프록시 뒤의 노드는 뒤처진다.** bp1 이 영수증을 돌려준 직후에도 en1 에는 그 블록이
       없어 `balance 0` 으로 거절된다. 다른 노드에서 만든 상태를 근거로 보내려면 그 상태가
       보내려는 노드에 도착했는지 `waitFor` 로 먼저 기다려야 한다. 이 기다림 자체가 단정이다.
  - **go-wemix(poa) 깊이도 채웠다 (2026-09-11).** `consensus/01-wemix-etcd-and-governance` —
    `admin_wemixInfo` 로 **etcd 클러스터가 형성됐는지**와 **거버넌스 멤버가 전부 생산 중인지**를
    본다. 블록이 오르는 것은 둘 다 보여주지 못한다: 거버넌스에 등록됐지만 etcd 에 합류하지 못한
    멤버는 `down` 으로 뜨고 체인은 나머지로 계속 움직이며, 핸드오프 작업이 만난 것은 **앞 단계가
    전부 성공을 보고하는 동안 비어 있던 클러스터**였다.
    - **두 지표는 서로 다른 사실이다 (실측).** node3 을 세우면 `miners` 는 `node3/down` 으로
      바뀌지만 `etcd.members` 는 **그대로 4** 다 — 전자는 생존, 후자는 등록이다. 하나만 단정하면
      실패의 절반을 놓친다. 그래서 둘 다 단정한다.
    - 변이로 실패 확인(`etcd.members` 기대값을 5로). **주의**: 노드를 죽여서 변이하려 하면
      readiness 게이트가 `network not ready to test` 로 단정 전에 막는다 — 의도된 동작이고,
      fault 주입은 attach 가 아니라 스펙 안의 `stopNode` 로 해야 한다.
  - **15노드 fault 도 채웠다 (2026-09-11).** `fault/02-wbft-quorum-at-15-nodes` — validator 13
    이면 정족수는 9 다. **4대를 세워 정확히 9 를 남기면 계속 생산**하고, **5대째를 세워 8 이
    되면 멈춰야** 하며, 되살리면 재개해야 한다. 한 스펙 안에서 두 절반이 서로를 증명한다 —
    9 로 계속 도는 것이 8 에서 멈춘 것을 우연이 아니게 만든다. 변이(5대째를 세우지 않음)로
    실패 확인.
- [x] **D. 라이브 MCP 플러그인 — 해소 (2026-09-12). 원인은 배포가 아니라 심볼릭 링크였다.**
  플러그인은 `command: chainbench-mcp` 로 PATH 에서 찾는데
  (`~/.claude/plugins/cache/stablenet-expert/core-dev/0.1.0/.mcp.json`), 그 항목이
  **다른 저장소**를 가리키고 있었다:
  `~/.local/bin/chainbench-mcp` → `~/work/github/0xmhha/auto-coding/chainbench/bin/chainbench-mcp`
  (링크 2026-06-09, 바이너리 2026-07-26). `auto-coding/chainbench` 는 별개 git 저장소이고
  마지막 커밋이 `b1471fa`(PR #170) 이며 작업이 2026-08-04 에 멈췄다 — 이 저장소는 PR #409 다.
  - **"낡은 빌드" 가 아니라 "다른 저장소의 빌드" 였다.** 도구 집합이 절반만 겹치고 라이브에만
    옛 이름 3개(`chainbench_setup_plan`·`chainbench_start`·`chainbench_test`)가 있는 것이
    그것으로 전부 설명된다. 이 저장소에는 그 세 이름이 없다.
  - **조치**: 이 저장소의 `cmd/chainbench-mcp` 를 빌드해 링크를 돌렸다. `tools/list` 를
    stdio 로 직접 물어 **73개**를 확인했다(라이브는 30개였다). 옛 대상 경로는 되돌릴 수 있게
    기록해 두었다. `auto-coding/chainbench` 는 5주 넘게 멈춰 있어 그쪽을 깨뜨릴 위험이 낮다고
    판단했다.
  - **다음 세션에서 적용된다.** MCP 서버는 세션 시작에 연결되므로 링크를 돌린 세션의 도구
    목록은 바뀌지 않는다. 확인은 새 세션에서 `chainbench_chain_up` 이 보이는지로 한다.
  - **`CHAINBENCH_DIR` 는 이 저장소에서 읽는 곳이 없다**(플러그인 `.mcp.json` 은 여전히
    넘긴다). 옛 빌드의 잔재이고, 지금은 무해하다.
  **(원래 진단, 2026-09-11 — 원인을 못 짚었다)** "라이브가 `net_*` 이름" 은 이제 맞지 않는다 — 라이브도 `chainbench_*` 를
  쓰지만 **더 오래된 다른 집합**이다. 저장소는 도구 58개를 등록하는데
  (`grep -rh 'Name:\s*"chainbench_' internal/mcp/*.go`), 라이브에는 `chainbench_chain_*`
  20여 개와 `keyring_*`·`resource_*`·`upgrade`·`hardfork`·`validate`·`run`·`node_start/stop`
  이 **아예 없고**, 반대로 라이브에만 있는 `chainbench_start`·`chainbench_test`·
  `chainbench_setup_plan` 은 저장소에 없는 옛 이름이다. **코드 작업이 아니라 배포 작업이며**,
  두기만 하면 MCP 표면을 건드리는 작업의 라이브 검증이 전부 겉돈다.
  - CLI 쪽은 해소됐다(2026-09-11 확인): `test`·`network`·`node`·`resource`·`validate`·
    `report` 그룹이 모두 `chainbench --help` 에 있다. WA2·WA5 가 가리키던 공백이다.

- [x] **관측 (2026-09-11). `istanbul_getWbftExtraInfo` 는 `"latest"` 같은 블록 태그를 받지
  못한다.** 라이브에서 `"latest"` 로 부르면 `block is not a wbft block` 로 실패하고, 같은
  체인에 16진수 블록 번호를 주면 정상 응답한다. 원인은 go-wbft 쪽이다 —
  `consensus/wbft/backend/api.go:418` 이 `big.NewInt(int64(number))` 로 태그의 음수 표현을
  그대로 블록 번호로 쓰고 `IsCroissant` 가 거짓이 되어 나간다. **오류 메시지가 원인을
  가린다**(태그를 못 푼 것인데 "wbft 블록이 아니다" 라고 한다). chainbench 쪽 영향은
  스펙 작성이다: 다른 `istanbul_*` 는 `"latest"` 를 받으므로(`istanbul_getValidators` 를
  쓰는 기존 케이스들이 그렇다) 같은 습관으로 쓰면 실패한다. 이 메서드를 쓰는 스펙은 epoch
  경계 번호를 명시해야 한다. **체인팀 몫이지만, 스펙 작성자가 먼저 밟는다.**
  - **인계 완료 (2026-09-12).** `docs/dev/chain-handover-2026-09-12.md` §3 (B1) 으로 옮겼다.
    거기서 하나 더 확인했다: **고칠 모양이 같은 파일 안에 이미 있다.** 285줄 위
    `GetValidators` 는 포인터로 받고 `rpc.LatestBlockNumber` 를 먼저 판정하며,
    `GetCommitSignersFromBlock`·`calculateBlockRange`·`IsValidator` 도 같다.
    `GetWbftExtraInfo` 만 값으로 받고 판정을 생략한다.
  - **정정.** 위 단락의 "스펙은 epoch 경계 번호를 명시해야 한다" 는 필요 이상으로 좁다.
    저장소에는 이미 일반 우회가 있다 — DSL 의 `"@latest"` 센티넬
    (`internal/testhelper/derived.go:235`)이 호출 시점에 `eth_blockNumber` 결과로 치환하고,
    `waitFor` 안에서는 폴링마다 재치환한다. 이 메서드를 쓰는 기존 스펙들이 전부 그것을 쓴다.
    **그 센티넬이 존재하는 이유가 이 결함이다.**

- [x] **관측 (2026-09-11). go-wemix 바이너리가 `verifyBlockSig` 에서 패닉한다.** 15+15
  핸드오프 중 **15대 중 한 대**가 동기화하다 죽는다 —
  `panic: runtime error: invalid memory address or nil pointer dereference` →
  `math/big.(*Int).Sign` ← `wemix.verifyBlockSig` (`wemix/admin.go:1002`) ←
  `consensus/ethash.(*Ethash).verifyHeader`. 3회 연속 재현했고 **매번 다른 노드**가 죽는다
  (node4 → node5 → node5). 결과는 `nodes not ready for mesh: endpoint … not ready within 30s`.
  - **chainbench 결함이 아니다.** 오늘 오전에 세 번 성공하던 **명시 경로 호출**
    (`--from-binary /data/chainbench/bin/gwemix`)을 그대로 다시 돌려도 같은 패닉으로 실패한다.
    즉 호출 형태와 무관하고, 이 기기의 상태가 달라지면서 go-wemix 쪽 경합에 걸리기 시작했다.
  - 눈에 띄는 점: 죽는 노드는 **ethash 검증 경로**를 타면서 wemix 블록 서명을 검증하고 있다.
    거버넌스를 아직 싣기 전의 창으로 보인다. **체인팀 몫**이고 R6 과 같은 성격이되 증상은 다르다.
  - **원인을 소스에서 확정하고 인계했다 (2026-09-12).** `docs/dev/chain-handover-2026-09-12.md`
    §1 (W1). **단락 평가의 좌우가 뒤집혀 있다.** `admin.go:1001` 의 가드는
    `err != nil || count.Sign() == 0` 이라 `err != nil` 에서 단락해 `count` 를 읽지 않는데,
    1002 줄의 반환식은 `err == ErrNotInitialized || count.Sign() == 0` 로 순서가 바뀌어,
    `err` 이 **다른 에러**이면 nil `count` 를 읽는다. 생성된 바인딩이 에러 경로에서 nil 을
    돌려준다(`wemix/bind/gen_gov_abi.go:2129` 의 `*new(*big.Int)`). 그 높이에 거버넌스 코드가
    없으면 언팩이 `abi: attempting to unmarshall an empty string …` 로 실패하는데, 이것이
    nil 도 `ErrNotInitialized` 도 아닌 바로 그 값이다. 관측된 `addr=0x10` 은 nil `*big.Int`
    의 `abs` 슬라이스 길이 필드 오프셋(16)과 같다.

### C. 키 취급과 증적 (§1q)

둘 다 지금 동작을 막지 않는다. 하나는 증적 보존, 하나는 키가 로컬에 내려오는 범위를
좁히는 일이다.

- [x] **L1. 재기동이 이전 시도의 로그를 지우던 것 — 해소 (2026-09-11).** "attempt 축이
  없다"로 적혀 있었지만, 측정하니 **축이 없는 것이 문제가 아니라 원격이 로그를 덮어쓰는 것이
  문제**였다.
  - **두 드라이버가 달랐다.** 로컬은 늘 `O_APPEND` 로 열었고(`local.go`), 원격 launch 명령은
    `> log` 였다(`remote.go`). 그래서 **원격/docker 에서 노드를 재기동하면 죽은 이유가
    지워졌다** — 그리고 재기동은 fault 스펙(`restartNode`)과 reuse-if-matching 이 타는 바로 그
    경로다. 대상에 따라 실행이 남기는 것이 달랐다는 뜻이기도 하다.
  - **겹쳐 있던 두 번째 손해**: collector 는 바이트 오프셋으로 tail 한다. truncate 뒤에는
    오프셋이 짧아진 파일의 끝을 넘어가 있어, 새 시도가 옛 길이를 넘길 때까지 **아무것도 읽지
    못한다**. 그 사이 건너뛴 줄이 새 시도의 startup — 재기동이 왜 그렇게 됐는지 말하는 부분이다.
  - **고친 것**: 원격도 `>>` 로 덧붙이고, launch 마다 `=== chainbench: launch node<N>
    <binary> ===` 마커를 같은 append 로 먼저 쓴다. 한 파일에 여러 시도가 쌓이므로 경계가 필요하고,
    마커가 **바이너리 이름을 담는** 이유는 재기동이 다른 빌드일 수 있기 때문이다(하드포크 스왑이
    그것이다). 즉 별도 경로 축을 만들지 않고 파일 안에서 시도를 가른다 — 모든 `LogPath` 소비자
    (collector tail·`chain logs`·`node.Record`)를 건드리지 않는 쪽을 골랐다.
  - 단위 3건 + **라이브 1건**(실 docker 에서 두 번 launch → 첫 시도가 남아 있고 순서가 맞고
    마커가 2개). 변이 2건으로 실패 확인 — `>>` 를 `>` 로 되돌리면 라이브가
    `the restart erased the previous attempt` 로 잡고, 마커를 없애면 `found 0` 으로 잡는다.
  - **기존 테스트 둘을 고쳐야 했다.** `"|| exit 1; nohup"` 의 **인접**을 단정하고 있었는데,
    지키려던 성질은 "mkdir 이 `&&` 가 아니라 `|| exit 1;` 로 갈라지고 nohup 이 그 뒤에 온다"는
    것이다. 인접을 박으면 그 틈에 올바른 것을 더할 때마다 깨진다. 성질로 바꿨다.
- [x] **L2. 대상에서 키 검증(공개 신원만 반환) — 완료 (2026-09-12).** 가르는 일(2026-09-11)에
  이어 **전송 자체를 없앴다**. 재확인: `grep -n "func LoadPresetWithKeys" internal/core/keyring/preset.go`
  가 둘을 보이고, 인덱스 읽기(`LoadPresetAt`)는 신원만 돌려준다. 아래 본문의 `LoadPublicPreset`
  은 그 중간 단계의 이름이고, 지금은 `LoadPreset`(신원) / `LoadPresetWithKeys`(키) 로 갈렸다.
  **(원래 표기)** 가르는 일은 했다 (2026-09-11). 전송을 없애는 일이 남았다.
  - **먼저 진단이 정확하지 않았다.** "묶음 전체를 내려받는다"는 것은 compose 경로
    (`chainsetup.materializeKeyring`)의 이야기이고, 그쪽은 **키를 각 노드에 배포해야 하므로
    정당하다**. 읽기 경로(`keyring list`/`show`)는 이미 다운로드 없이 대상에서 읽고 있었다.
    실제 문제는 다른 자리였다: **링 인덱스는 파일 하나이고 그 안에 모든 노드의 `nodekey` 가
    있다.** 그래서 `LoadPresetAt` 한 번이 원격이면 **링 전체의 개인키가 운영자 프로세스로
    넘어온다** — `list` 가 `export` 와 같은 만큼 공개하고 있었고, 어디에도 그렇게 적혀 있지
    않았다.
  - **가른 것.** `Preset.PublicOnly()` 와 `store.LoadPublicPreset(At)` 을 더해, 읽기가 돌아오는
    순간 개인키를 버린다. 신원만 필요한 호출부는 전부 그쪽으로 옮겼다 — `operation.List`
    (단 `--verify` 는 아니다), `operation.Show`, `chainsetup/genesis_verify`,
    `chainsetup/baseline`. 키를 요구하는 곳은 **두 곳만 남았고 그렇게 말한다**:
    `--verify`(개인키에서 신원을 재파생하므로 없이는 불가)와 `Export`(의도적 공개).
    `LoadPresetAt` 의 doc 이 이제 **원격 읽기는 개인키를 옮긴다**고 명시한다.
  - **public-only 항목은 검증을 거부한다.** 없으면 0 키로 파생해 *진짜 같은* 주소 불일치를
    보고하고(실측: `key derives address 0x3f17f196… but the file records 0xc17d4938…`), 손상된
    키자료처럼 읽힌다. 변이로 확인했다.
  - **라이브 (docker).** 원격 링을 만들고 `keyring list` 가 신원만 내보내는 것과
    `list --verify` 가 여전히 통과하는 것을 확인했다.
  - **전송 자체도 없앴다 (2026-09-12, 사용자 결정).** 제가 내놓은 두 선택지(대상에서 파생 /
    포맷 분리)보다 나은 셋째가 링을 읽어 보니 있었다 — **생성기가 `node<N>/nodekey` 를 이미
    항상 쓴다.** 인덱스의 개인키는 그 **중복**이었다. 그래서 인덱스에서 뺐다.
    - **인덱스 읽기는 이제 구조적으로 공개다.** `LoadPresetAt` 은 신원만 돌려준다. 키가 필요한
      호출자는 `LoadPresetWithKeysAt` 을 부르고 **N번 읽기**를 대가로 낸다 — 원격이면 N 왕복이다.
      그 값을 치르는 곳은 셋뿐이다: `--verify`(키에서 신원을 재파생), `export`(의도적 공개),
      `import-ring`(신원을 복사하므로 신원을 만드는 것이 필요하다). 구성 경로는 원래부터
      노드별 파일에서 배포하고 있었다.
    - **옛 링은 그대로 동작한다.** 키를 요청했는데 노드별 파일이 없으면 **그때만** 인덱스의
      레거시 필드로 폴백한다. 신원 읽기는 그 필드를 절대 보지 않으므로, 옛 링이 살아 있으면서도
      모든 읽기가 다시 공개가 되지는 않는다. 인덱스만 남긴 링으로 `--verify` 통과를 실측했다.
    - **라이브가 잡고 스위트는 못 잡은 결함이 하나 있었다.** `testengine.ringFor` 가 스펙이
      **서명에 쓰는** 링을 만드는데 신원만 읽고 있어, 모든 실행이 시작에서
      `register identity node1: invalid private key: all zero` 로 죽었다. 이 패키지의 기존
      테스트는 ringFor 에 **링이 없는 디렉터리**만 넘겨(남의 망에 attach 하는 경우) 조기 반환
      경로만 덮고 있었다. 진짜 링을 주는 회귀 테스트를 붙였고, 변이로 그 오류 메시지를 그대로
      재현했다.
    - 변이 2건(인덱스에 키를 다시 쓰기·신원 읽기가 키를 싣기)으로 확인. 라이브: 새 링 생성 →
      인덱스 필드가 `address`·`index`·`publicKey` 뿐, `list`·`list --verify`·`export` 정상,
      wbft 체인 기동과 서명하는 tx 케이스 통과.
    - `Preset.PublicOnly` 는 호출자가 없어져 제거했다 — 인덱스가 키를 담지 않으니 방어할 값이
      애초에 생기지 않는다.

### D. 모니터링 트랙에서 파생된 후속 (§1r 8절)

이번 PR 의 범위 밖으로 명시하고 미룬 것들이다.

- [x] **후보·반영 완전 분리 — 완료 (2026-09-11).** 코드는 이미 옳았고, **증명이 비어
  있었다.** 구멍을 정확히 말하면: 이 패키지의 드라이버 스텁은 전부 `Stop` 에서 nil 을 돌려주고
  **아무것도 기록하지 않는다.** 그래서 "drift 하나에 망 전체를 정지시키는" 구현도 기존 테스트를
  모두 통과한다 — 되돌릴 수 없는 그 한 행위가 어디에도 관측되지 않았다.
  - 기존 부분 케이스(`TestReconcileReuse_StoppedDriftedNodeClearedFromLedger`)는 drift 한 노드에
    **PID 0**(한 번도 안 돈 노드)을 주므로 정지 경로에 아예 닿지 않는다.
  - `Stop` 을 기록하는 드라이버로 세 가지를 고정했다: **둘 다 살아서 응답하는데 config 만 다를
    때 정지되는 것은 drift 한 쪽 pid 하나뿐**, 전부 일치하면 **정지 0건**(재사용의 존재 이유),
    거부되면 **정지 0건**(정지는 되돌릴 수 없고, 거부는 망이 그대로여야 하는 경우다).
  - 변이 2건으로 실패 확인 — 재사용 분기를 무력화하면 `stopped [111 222]` 로 잡히고, 거부 후
    조기 반환을 없애면 `a refusal stopped [111]` 로 잡힌다.
  - **부수로 배운 것**: 같은 성질을 `ChainUp` 끝단에서 스텁으로 재보려 했더니 통과했는데,
    **아무것도 증명하지 못하는 통과**였다 — 스텁은 RPC 로 응답하지 않으므로 두 노드가 다
    `not answering` 으로 redo 되고(`0 node(s) reused, 2 redone`), pid 는 스텁이 같은 값을
    돌려줘서 우연히 같았다. 그 테스트는 버렸다. 판정에 `alive` 가 필요한 성질은 응답하는 노드가
    있어야 하므로 reconcile 수준에서 보는 것이 맞다.
- [x] **`binaryAliases` 와 객체형 참조 — 닫혔다 (2026-09-12).** 별칭은 배선됐고 실사용
  호출부가 있었다(당시 `app/upgrade.go:249` → `resource.WorkspaceConfig.BinaryPath`; 재확인:
  `grep -rn "BinaryPath(" internal/ cmd/ | grep -v _test.go`). 객체형 참조는 사용자 결정으로
  철회했다. 양쪽이 닫혔으므로 `◐` 가 아니다.
  **(원래 표기)** `binaryAliases` 는 배선했다 (2026-09-11). 대상에서 **bare 바이너리 이름**이 workspace-config 를 거쳐 풀린다 —
  `dataRoot` + `paths.binaries` 에서 찾고 별칭이 있으면 적용한다(`app.resolveBinaryOn` →
  `WorkspaceConfig.BinaryPath`). 이전에는 대상에서 bare 이름을 **거부**했고, 그래서
  `upgrade run --all-servers` 를 돌릴 때마다 config 가 이미 아는 경로를
  `--from-binary /data/chainbench/bin/gwemix` 로 손으로 적어야 했다. 즉 별칭에 소비자가
  없던 것과 그 papercut 은 **같은 결함의 양면**이었다. 단위 테스트 5건 + 변이 2건.
  라이브: bare 이름이 `/data/chainbench/bin/gwbft` 로 풀려 노드가 떴다(아래 관측 참고).
  **객체형 참조는 철회했다 (2026-09-12, 사용자 결정).** `{server,ref}`·`{serverIndex,ref}`·
  `{localPath}` 은 구현하지 않고 문서에서 약속을 걷었다.
  - **이유는 표현력이 아니라 값이다.** `resource.InputRef` 는 네 형식을 이미 구현·테스트한
    채로 있지만, 그 값을 preset 에서 끝까지 나르려면 `KeysDir`·`GenesisExisting` 을 문자열로
    쓰는 **32개 파일 105곳**을 구조체로 바꿔야 한다(측정값). 그 표현력을 요구하는 호출자가
    하나도 없다 — 소비자 없는 구조를 넓게 배선하는 것은 이 저장소가 이미 두 번 되돌린
    모양이다(`Transport` 타입, health→inspector 재배선).
  - **포기한 것은 `serverIndex` 하나.** 서버를 이름이 아니라 **순번**으로 고르는 것은 문자열로
    표현할 수 없다. 환경마다 서버 이름이 다른 곳에 같은 preset 을 쓰려면 필요해질 수 있고,
    그때 요구와 함께 다시 연다. `InputRef` 는 그대로 두었으니 배선만 하면 된다.
  - **샘플에서 스케치를 걷은 것이 핵심이다.** "미구현" 딱지가 붙은 주석 예시는 기능의 부재보다
    나쁘다 — preset 의 genesis/keyring 자리에 맵을 적으면 폴백이 아니라 **파일 전체가 거부**되므로,
    주석을 푼 운영자는 기능을 얻는 게 아니라 설정을 잃는다.
  - 가드 테스트를 목적에 맞게 바꿨다(`TestWorkspaceConfig_SampleSketchesNothingThatCannotBeUsed`):
    파서가 여전히 거부하는지 **그리고** 샘플이 그것을 **복사 가능한 형태로 보여주지 않는지**를
    함께 본다. 단어가 아니라 YAML 모양(콜론 붙은 키)을 보므로, 철회 사실을 산문으로 적는 것은
    막지 않는다. 변이 2건으로 양쪽 절반이 각각 실패함을 확인했다.
  - **(원래 진단)** 전자는 **리졸버까지는 생겼다** —
  `WorkspaceConfig.BinaryPath`(`resource/workspaceconfig.go:327`)가 별칭을 적용한다. 다만
  **호출자가 테스트 둘뿐이라 프로덕션 경로에는 아직 배선되지 않았다** — PR #383 머지 후
  재확인해도 `.BinaryPath(` 호출부는 `workspaceconfig_test.go` 2곳뿐이다. 후자
  (`{server,ref}`·`serverIndex`·`localPath`)는 아직 파싱조차 안 된다. 샘플과 안내
  문서가 "미구현" 이라고 적어 두었고, 구현하면 `TestWorkspaceConfig_SampleCommentsMatchWhatParses`
  가 실패하며 그 주석을 걷으라고 알린다.
- [x] **여러 정의서 실행의 통합 report — 완료 (2026-09-11).** `report --all` 이 디렉터리 아래
  **모든 세션**을 하나로 합친다. 이전에는 세션 여러 개를 담은 루트를 줘도 **가장 최근 것 하나**만
  읽었고, 그건 "이 묶음이 통과했나"의 답이 아니다. 정의서 파일마다 한 번 실행하는 것이 이
  하네스의 보통 사용법이라, 묶음 판정을 눈으로 조립해야 했다.
  - **합치는 일은 렌더링이 아니라 core 에 있다** (`report.Combine`). 합계를 표면마다 더하면
    두 표면이 다르게 더하게 된다.
  - **행마다 실행 이름을 붙인다.** `seq` 는 한 실행 안에서만 유일해서, 실행 4개면 `seq 1` 이
    4개다. 실측 출력이 그 이유를 그대로 보여준다 — 같은 케이스의 pass/fail 쌍 4행이 세션
    열 없이는 구별되지 않는다. `TestReport.Session` 을 더했고(단일 실행에서는 비어 있다 —
    그때는 report 자신이 세션을 말한다), 이미 붙어 있는 값은 덮지 않는다(합친 것을 다시 합쳐도
    원래 실행을 가리킨다).
  - **통합 report 의 세션 id 는 존재하지 않는 이름**(`(combined)`)이다. 실제 id 를 가져가면
    증적 경로가 한 실행을 가리켜, 따라간 사람이 **다른 실행의 증적**을 보게 된다.
  - **두 표면 모두**에 넣었다(CLI `--all`, MCP `all`). MCP 스키마는 손으로 쓰는 것이라 자동으로
    따라오지 않는다 — 넣지 않으면 두 표면이 다른 질문에 답한다. CLI 는 `core/report` 를 직접
    import 할 수 없어(`arch.TestSurfacesReachThroughApp` 가 잡았다) `app.CombinedSession`·
    `app.ReportSessions` 로 중계한다.
  - 단위 7건 + 변이 3건(합계를 덮어쓰기·세션 미부착·실제 id 사용). 실제 세션 4개로 end-to-end
    확인: `4 session(s) combined pass=2 fail=2`.

### D2. 코드 건강도 검토에서 나온 것 — **다섯 항목 완료 (2026-09-11)**

**추가 정리 (2026-09-12).** 남아 있던 넷을 다시 재고 둘을 처리했다. 하나는 **제 측정이
틀렸고**, 하나는 판단해서 손대지 않기로 했다.

- [x] **package doc 공백 — 해소. 다만 14개가 아니라 9개였다.** 제가 `^// Package ` 만 세어서
  `main` 패키지 5개(`cmd/chainbench`·`chainbench-dashboard`·`chainbench-mcp`,
  `scripts/inventory/` 둘)를 공백으로 잡았는데, **main 패키지의 올바른 형식은 `// Command X`**
  이고 다섯 개 전부 이미 갖고 있었다. 실제 공백은 `cmd/chainbench/*cmd` 9개였다.
  - **그 9개는 채울 값이 있었다** — 명령 묶음의 경계가 자명하지 않다. 왜 `clean`·`verify` 가
    `lifecyclecmd` 이고 `show`·`status` 는 `chaincmd` 인지, `faucet` 이 왜 `txcmd` 가 아니라
    `accountcmd` 인지 같은 것이다. 각 doc 이 **주체가 무엇인가**로 그 경계를 적는다
    (composition 이냐 running network 냐, transaction 이냐 account 냐).
  - 재확인: `for d in $(go list ./... | sed 's|github.com/0xmhha/chainbench/||'); do [ -d "$d" ] && { grep -lqE "^// (Package|Command) " $d/*.go 2>/dev/null || echo $d; }; done`
    — 아무것도 출력되지 않아야 한다. **`// Command` 를 함께 받는 것이 핵심이다.**
- [x] **`internal/chains/stablenet` 테스트 없음 — 해소. 그리고 없던 이음매를 찾았다.**
  이 패키지는 `caps.jsonl` 로 능력 10개를 **선언**하고 `init()` 에서 핸들러를 **따로** 등록한다.
  `registry.All`/`For` 는 **노출된 것만**(핸들러나 flat tool 이 있는 것) 돌려주므로, jsonl 에
  항목을 더하고 핸들러를 안 쓰면 **아무 데서도 오류가 아니다** — 파일은 있다고 말하는데
  호출자는 없다고 듣는다. 지금은 10 = 10 으로 맞지만 붙잡는 것이 없었다.
  - **반대 방향은 쓸 수 없다는 것을 변이로 배웠다.** 처음에 "선언 없는 핸들러" 도 단정했는데,
    `For()` 가 **카탈로그를 순회**하므로 jsonl 한 줄을 지우면 양쪽에서 동시에 사라져 **테스트가
    통과했다.** 제거하려던 바로 그 모양(공허한 단정)을 제가 만든 것이라 지우고, **왜 쓸 수
    없는지**를 파일에 적었다(핸들러 맵은 private 이고 lister 가 없다). 그 방향의 심각도도 낮다 —
    고아 핸들러는 도달 불가일 뿐 틀리지 않는다. 그래서 core 에 lister 를 더하지 않았다.
- ☑ **폴링 루프 — 결함 하나를 고치고, 제 측정을 정정했다 (2026-09-12).**
  - **"10벌이 같은 모양" 은 과장이었다.** 열어 보니 루프들이 서로 다르다: 반환형이
    `error` 와 `(T, error)` 로 갈리고, 시도 중 에러를 **즉시 반환하는 것**(`executor.go:122`
    의 `files.Exists`)과 **모아 두는 것**(`executor.go:310` 의 `last error`,
    `info.go:80` 의 `last`/`lastErr`)이 섞이고, 데드라인 검사 위치도 다르다
    (`handoff.go:746` 은 루프 머리에서 본다). 하나의 헬퍼로 흡수하려면 제네릭과 노브 여럿이
    필요하고, **그러면 원래 루프보다 읽기 어려워진다.**
  - **정말 같은 것은 안쪽의 "취소를 존중하는 대기" 하나다** — `select { ctx.Done() /
    time.After(interval) }` 가 9곳에서 글자까지 같다. **그리고 10번째가 버그였다.**
  - **`upgrade.WaitEndpointsReady` 가 취소를 무시하고 있었다 (고쳤다).** 그 자리만
    `time.Sleep` 이었다 — context 가 끼어들 수 없는 유일한 대기 형태다. 호출자가 포기한 뒤에도
    죽은 엔드포인트를 **남은 예산 전부**(핸드오프 경로에서 30초) 계속 찔렀고, 그래서 실행의
    취소가 그만큼 늦게 보고됐다. 같은 패키지의 다른 폴링 루프는 전부 `ctx.Done()` 을 보고
    있었고 이 하나만 아니었다.
    - 단위 테스트 3건: 취소가 **즉시** `context.Canceled` 로 돌아온다(예산은 5분으로 두어,
      존중하지 않으면 통과가 아니라 **타임아웃으로 실패**한다) · 취소가 없으면 여전히 자기
      데드라인으로 실패하고 **엔드포인트 이름을 말한다** · 빈 항목은 기다릴 것이 아니라 건너뛴다.
    - 변이(`time.Sleep` 으로 되돌리기)로 확인: 테스트가 "ignored cancellation" 으로 실패하고
      11초를 쓴다(고친 뒤에는 0.9초).
  - **공통 대기도 통합했다 (2026-09-12).** 처음에 "비용의 모양 때문에 따로 연다" 고 적었는데,
    그 비용이 실제로는 표 한 줄이었다. `internal/core/wait` 를 만들고 `Sleep(ctx, d)` 하나를 뒀다.
    - **11곳을 바꿨다** — `upgrade`(mesh·handoff) · `poa`(info·executor 2곳·bootstrap_exec 5곳) ·
      `core/collector`. 각 자리는 `if err := wait.Sleep(ctx, poll); err != nil { return …, err }`
      한 줄이 되고, 반환값이 다른 것(`""`·`last`·`LogMatch{}`)은 그대로 유지된다.
    - **12번째는 바꾸지 않았다.** `core/process/stop.go:58` 은 취소 시 `ctx.Err()` 가 아니라
      `gone(pid)` 를 돌려준다 — "포기했으니 프로세스가 사라졌는지 답한다" 는 **다른 의미**이고,
      통합한다고 같게 만들면 안 된다.
    - **왜 이름 하나가 값인가**: 올바른 형태가 `select` 4줄이고 틀린 형태가 `time.Sleep` 한 줄이면,
      **틀린 쪽이 더 짧다.** 11곳에 손으로 쓰여 있었다는 것은 짧은 쪽을 실수로 쓸 기회가 11번
      있었다는 뜻이고, 실제로 한 번 그랬다. 이제 올바른 쪽이 한 줄이다.
    - **배치 래칫이 제 역할을 했다.** 새 패키지를 넣자마자 `arch.TestEveryPackageIsPlaced` 가
      "`core/wait` 가 `layers.md` §3 에 없다" 로 실패했다. L0(내부 import 0)에 한 줄을 적어 해소.
    - 단위 테스트 5건 + 변이 4건: 기다리지 않는 구현 · 위쪽 `ctx.Err()` 제거 · `d<=0` 을
      취소보다 먼저 보기 · 취소를 `nil` 로 삼키기. 넷 다 실패 확인.
      **변이가 하나 알려 준 것**: 위쪽 `ctx.Err()` 검사를 지워도 `d<=0` 케이스만 깨진다 —
      양수 d 에서는 `select` 가 이미 기다리지 않고 취소를 보고하기 때문이다. 그래서 그 검사의
      진짜 용도는 **0 간격**이며, 주석을 그렇게 고쳤다(처음에 더 넓게 적었다).
      **변이 M1 은 한 번 실패했다** — 본문을 `time.Sleep(d)` 로 바꾸니 한 시간짜리 테스트가
      go test 기본 타임아웃까지 매달렸다. `-timeout 20s` 로 다시 돌려 판정했다.
  - 재확인: `grep -rn "case <-time.After" internal/consensus/ internal/core/ | grep -v _test | wc -l` ·
    `grep -rn "time\.Sleep(" internal/consensus/ | grep -v _test` (비어야 한다 — 호출 형태로 찾는다. `time.Sleep` 만 찾으면 이 수정을 설명하는 주석 문장에 걸린다)
  **(원래 진단, 과장)** `internal/consensus/` 의
  `time.Now()` 20곳은 **전부 같은 모양**이다: `deadline := time.Now().Add(timeout)` → 시도 →
  `time.Now().After(deadline)` 면 **자기만의 오류 메시지**로 반환 → `select ctx.Done() /
  time.After(interval)`. 결정에 쓰이는 시계는 **하나도 없다** — 전부 기다림의 한계다.
  - **그래서 "시계를 주입하라" 가 이 항목의 답이 아니다.** 20곳에 `Clock` 을 꿰는 것은
    재현성을 얻지 못하고 변경 면만 넓힌다. 실제 결함은 **중복**이고, 이 저장소가 `shellQuote`
    4벌·`Arg*` 2벌에서 한 번씩 처리한 그 모양이다.
  - **다른 것은 오류 메시지뿐이고 그것은 값이 있다** — "cluster is still empty 30s after init"
    처럼 각 자리가 무엇을 기다렸는지 말한다. 그래서 헬퍼는 메시지를 클로저로 받아야 하고,
    **메시지를 하나로 합치면 안 된다.**
  - **별도 항목으로 둔다.** 브링업 경로 10곳을 건드리는 변경이라 라이브 확인이 필요하고,
    `netUpFrom` 분해를 따로 두기로 한 것과 같은 이유다 — 다른 변경에 얹으면 검토 범위가 흐려진다.
  - 재확인: `grep -rn "time.Now()" internal/consensus/ | grep -v _test | wc -l` (20) ·
    `grep -rn "deadline := time.Now().Add" internal/consensus/ | grep -v _test | wc -l` (10)
- [x] **가장 긴 함수 `lowerCase`(195줄) — 손대지 않기로 판단했다 (2026-09-12).**
  `internal/dsl/spec_v2.go:357`. 읽어 보니 **엉킨 로직이 아니라 문법의 크기**다 — id·env 검증
  뒤에 선언을 하나씩 접어 넣는 **순차 fold** 이고(timeouts·capabilities·manifest·binaries·
  upgrade·genesis·keys/launch·hooks·statements), 각 블록에 관심사를 적은 주석이 있다.
  - 9개 헬퍼로 쪼개면 전부 같은 `spec` 값을 만지므로 **결합은 그대로이고 간접만 늘어난다.**
    그리고 순서가 규칙을 읽히게 하는 자리가 있다 — "capabilities 는 requires 와 합집합, 케이스
    순서 먼저" 는 앞뒤 맥락에서만 뜻이 통한다.
  - 길이만 보고 다시 열지 않도록 판단을 적어 둔다. 코드 건강도 검토도 `netUpFrom` 을 같은
    이유로 따로 두라고 했다.


AST 로 다시 측정했다. 구조는 깨끗하다 — 층 위반 0, 래칫 통과, 린터 0건. 중복과 문서
두 갈래만 남았고, 중복은 **표면·상위 계층이 프리미티브를 각자 다시 만든** 한 가지
경향이었다. 근거와 위치는 정본
`architecture/code-health-review-2026-09-10.md` (2026-09-24 삭제).

- [x] `shellQuote` 4곳 통합 → `remote.ShellQuote` 하나. 셸을 실제로 태워 왕복시키는
      테스트를 붙였다(따옴표·개행·`$(id)`·`;id;` 18케이스). 네 사본 중 어느 것도
      테스트가 없었다.
- [x] `ArgString`/`ArgInt` 사본 제거. **단순 치환이 불가능했다** — `internal/mcp` 는
      `core/*` 를 직접 import 할 수 없고(`arch.TestMCPGoesThroughApp`, shrink-only),
      그래서 그 사본은 **규칙이 만들어낸 결과**였다. `app` 이 중계하도록 고쳤다
      (`app/args.go`). 구현은 `core/registry` 한 곳, 짧은 이름은 183 호출부에 그대로.
      `ArgStrings`/`ArgBool` 도 같이 올렸다.
- [x] `deps(cmd)` → `surface.Deps`. 실제로는 **12곳이었고 세 갈래로 갈라져 있었다** —
      7곳 Logf 만, 4곳 +Env, 1곳 +Command. 즉 어떤 명령 트리로 들어왔느냐에 따라
      워크스페이스 잠금이 `(command not recorded)` 로 남거나 실행을 이름으로 남겼다.
      합집합으로 통일했다(Env 는 app 의 nil 폴백과 동일, Command 는 잠금이 늘 가져야
      할 출처).
- [x] package doc — `internal` 미문서 **8개를 0개로**. 그 과정에서 두 건은 이름이
      틀린 문서였다: `core/keyring/operation` 이 "Package keyring", `core/filestore` 가
      "Package provision" 으로 시작했다(이관 후 남은 잔재, godoc 에 틀린 이름이 뜬다).
- [x] `netUpFrom` 191줄 → **95줄**. `planUp`(38, 검증·기본값)과 `upSteps`(69, 스텝
      테이블)로 갈랐다. 잠금 획득과 실행 루프만 본문에 남는다.

**이 작업이 새로 찾은 것**

- [x] ~~**`core/keyring/derive` 에 테스트가 하나도 없다.**~~ **해소 (2026-09-11).**
      `presets/keys` 을 정답 벡터로 삼아 5노드의 주소·devp2p 공개키·BLS 공개키·PoP 를 전부
      재파생해 바이트 단위로 대조한다. **테스트가 실패할 수 있음을 뮤테이션으로 확인**했다 —
      version-3 salt(주석이 지목한 그 미묘한 지점)로 바꾸면 preset 대조가 깨진다.
      `Derive` 의 doc 이 이미 "픽스처와 바이트 단위로 대조된다"고 적어 두었는데 그 대조가
      실제로는 없었다. 원래 진단: BLS 파생이 blst 의 `blst_keygen` 과 어긋나면 *형태는
      맞지만 wbft 노드가 거부하는* 키가 나온다 — 합의 문제처럼 보이는 키 문제다.

### E. 오래 남아 있는 잔여 (§1n 및 그 이전)

- [x] **원격 `chain rm`** — **완료·라이브 검증 (2026-09-11).** `filestore.Store` 에
  `Remove` 를 더하고 `Workspace.Rm` 이 target 의 store 를 지나게 했다. 그래서 이제 원격이
  분기가 아니라 같은 경로다 — `internal/arch` 의 target-분기 허용 목록에서 `Rm` 항목이
  **사라졌다**(래칫이 줄었다). 파괴적 연산이라 가드를 두 겹으로 뒀다:
  `filestore.CheckRemovable`(빈 경로·상대 경로·`/`·1세그먼트·glob·`$`/`~` 거부, 두 Store
  구현 모두가 적용)과 `filestore.CheckWithin`(호출자가 아는 target dataRoot 안으로 구속,
  root 자신도 거부). 라이브: stablenet 3노드 → stop → rm → **수동 정리 없이** wbft 3노드가
  같은 서버에 올라가 블록 8까지 진행. 자기 구성만 지우고 다른 구성과 `bin/` 은 남는다.
  `env/docker/README.md` 의 수동 정리 절차를 이 명령으로 교체했다.
  - 남은 흠(경미): 삭제 후 `runtime/<id>/`·`configs/` **빈 디렉터리가 남는다.** 기록된
    경로만 지우기 때문이고(그게 안전한 쪽이다 — 구성 id 가 비면 `runtime/` 전체가 대상이
    될 수 있다), 아무것도 막지 않는다.
- [x] **G2. 핸드오프 원격 — 완료·라이브 검증 (2026-09-11).** 원격 대상에서 핸드오프가
  끝까지 돈다: `handoff confirmed: head 21; block 21 sealed by 0x8eb79036… (successor)`.
  `upgrade run --server <name> --server-set … --workspace-config … --docker`.
  - **포트는 이제 server set 에서 나온다.** 이전에는 profile 의 `base_rpc: 40010` 을 써서
    docker 가 퍼블리시하지 않는 포트로 dial 했다. 서버를 지정하면 그 서버의 slot 을 받는다
    (5노드 → slot 1~5, http 8601~8605, p2p step 3 으로 30301/30304/…). profile 의
    `ports:` 는 단일 호스트 로컬 실행의 폴백으로 남는다.
  - **라이브가 찾은 결함 셋** — 전부 "대상 인식 경계가 이미 있는데 로컬 가정이 남아 있던" 자리:
    1. `upgrade/launch.go` 가 HTTP 를 **`127.0.0.1` 에 하드코딩**했다. 컨테이너 안에서는 그것이
       컨테이너 자신의 loopback 이라 퍼블리시 포트가 아무것도 못 만난다 — 노드는 돌고 있는데
       바깥에서 "not ready" 로 보인다. 대상에 배치된 노드는 `0.0.0.0` 에 바인드한다.
    2. readiness·mesh dial 이 노드 **자기 주소**로 나갔다. `NodeSpec.RPCURL`(opener 가 번역)을
       더해 `process.NodeOf` 가 그것을 우선한다 — `MetricsURL` 과 같은 형태다.
    3. `env/docker/firewall.sh` 가 포트 대역을 **4 slot 폭으로 하드코딩**했다(8601:8604).
       `gen-env.sh` 의 `SLOTS` 를 올려도 방화벽이 안 따라와, slot 5 노드가 돌면서 DROP 됐다.
       이제 두 파일이 같은 knob 에서 파생된다.
  - `resource.ValidatePortsPerHost` 를 더했다. 포트는 **머신 안에서만** 경쟁하므로, 서버 15대에
    퍼진 망은 node1·node2 가 다른 호스트에 같은 포트를 갖는 것이 정상이다. 기존
    `ValidatePorts` 는 그것을 충돌로 봤다.
  - **서버 여러 대에 걸친 핸드오프도 완료 (2026-09-11).** `LaunchOptions.Machine` /
    `HandoffInputs.Machine` 이 노드별로 file store 와 driver 를 푼다 —
    `chainsetup.machineFor` 와 같은 seam 이다. 그 전에는 드라이버가 자기 transport 의 호스트로만
    접속하고 `NodeSpec.Host` 를 **읽지 않아**, 실측상 5노드를 5서버에 배치해도 datadir 5개와
    살아남은 프로세스 1개가 전부 server1 에 몰렸다(나머지는 그 서버의 포트에서 충돌). 라이브
    확인: server1 에 wemix producer 1대, server2~5 에 wbft validator 각 1대,
    `handoff confirmed: head 21`.
    - genesis 는 **머신마다 한 번** 쓴다. 중복 제거 키는 store 가 아니라 **호스트**다 — store 는
      비교 불가능한 인터페이스 값이라(원격 것은 func 를 담은 struct) map 키로 쓰면 panic 한다.
      실제로 그렇게 만들었다가 `hash of unhashable type: process.RemoteFileStore` 로 터졌다.
    - enode 는 각 노드 **자기 호스트**로 만든다. 한 호스트를 쓰면 전 노드가 첫 서버 주소를
      광고해 피어링이 성립하지 않는다.
  - **핸드오버 검증을 강화했다 (2026-09-11).** `AwaitFork` 가 블록 `fork+1` **하나**를 읽고
    `miner != producer` 만 보고 있었다. 두 구멍이 있었다 — producer 가 아닌 **제3의 주소**가
    봉인해도 통과하고, successor 가 한 블록만 봉인하고 producer 가 다시 가져가도 통과했다.
    이제 `fork+1`~`fork+10` 전부를 읽어 **선언된 successor validator 집합 안에서** 봉인됐는지
    확인하고, 몇 명이 교대했는지 보고한다. 라이브: `blocks 21-30 all sealed by the successor
    set, across 4 of 4 validator(s)`. 직접 확인한 봉인자 — 블록 18·19 는 producer
    `0xf9593d…`, 21~30 은 wbft validator 3명 이상이 교대. DSL 경로도 같은 함수를 타므로
    (`testengine/compose.go:645`) 자동 적용된다. 실패 가능함을 단위 테스트 5건으로 고정했다.
  - 로컬 경로 회귀 없음: 로컬 핸드오프도 `handoff confirmed: head 22` 로 완주.
- [x] **G3. 15+15 핸드오프 — 완료·라이브 검증 (2026-09-11).** docker 15대에
  **wemix 노드 15대 + wbft 노드 15대**, 서버 1대당 각 1대씩. 할당기는 이미 요청한 대로
  동작한다 — slot-major 로 서버 1~15 에 slot 1 을 주고, 다시 서버 1 로 돌아와 **포트만
  바꿔** slot 2 를 준다(plan 1~15 = p2p 30301/http 8601, plan 16~30 = p2p 30304/http 8602).
  제네시스는 **1개 파일**이다: wemix 제네시스를 base 로 `config.croissant` 섹션만 병합한
  것으로, 15대 전 서버에서 md5 동일(`340b27d7…`), croissant.init 에 validator 15명과 BLS
  키 15개. `presets/chain/wemix-upgrade-15.yaml` 신설(골든 1+4 프로파일은 손대지 않았다).
  - **확인한 결과** — `handoff confirmed: head 50; blocks 21-50 all sealed by the
    successor set, across 15 of 15 validator(s)`. RPC 로 따로 읽은 것: 블록 19 는 boot
    wemix producer `0xfa119c06…`, 블록 20 부터는 wbft validator, **producer 자신의 head 는
    19 에서 멈춘다**(설계대로). 컨테이너마다 프로세스는 정확히 `gwemix` 1개 + `gwbft` 1개.
  - **관측 창을 집합 크기에 맞췄다.** 10블록 고정이면 아무리 많아도 최대 10명만 보인다 —
    15명 세트가 "10 of 15" 로 보고됐고 그보다 더 보고할 수가 없었다. 창은 이제 `2n`
    (최소 10)이고, **DISTINCT 봉인자가 wbft 자신의 정족수 `floor(2n/3)+1` 이상**이어야
    한다. "모든 블록이 집합 소속"은 한 명이 전부 봉인해도 성립하므로 집합이 생산한다는
    주장이 못 된다. 전원(n명) 요구는 라운드 체인지 한 번에 실패하므로 쓰지 않는다.
  - **`--all-servers` 가 단독으로 동작하지 않았다.** CLI 가드가 `--server` 만 알아서
    `--data-dir is required` 로 막혔고, 통과시켜도 app 안에서 `a data dir is required` 로
    죽었다. base target 은 이제 **배치가 답한다** — placement 의 첫 노드가 앉은 머신이
    workspace-config 의 data root 를 소유하는 서버다(`firstPlacedServer`). `--server` 를
    나란히 적을 필요가 없어졌다.
  - **DSL 쪽 단정도 같은 강도로 올렸다.** `miner NotEqual <producer>` 는 **세상의 다른 모든
    주소**를 통과시킨다. `assert.In`(멤버십, 집합이 `is` 쪽·주소 대소문자 무시·빈 집합은
    거부)을 더하고 `tests/tc/go-wemix/hardfork/01-croissant-successors-take-over.json`(그때 `handoff/01-*.json`)
이 선언된 successor 집합을 쓰게 했다.
  - **profile 의 `extra_data` 는 핸드오프에서 무효다(실측).** 병합은 `config.croissant`
    섹션만 들어올리므로 wbft 제네시스의 top-level extraData 는 버려지고, 병합 파일은 wemix
    템플릿의 것을 유지한다(ASCII `"chainbench handoff…"` 149바이트). validator 집합은
    `croissant.init` 으로 들어간다. 프로파일 주석을 사실에 맞게 고쳤다.
  - **`targetValidators` 고정값 해결 — 완료·라이브 검증 (2026-09-11).** 제네시스 템플릿의
    `targetValidators`·`epochLength` 를 플레이스홀더로 바꿔 **validator 수에서 파생**시켰다
    (`__TARGET_VALIDATORS__` = n, `__EPOCH_LENGTH__` = max(10, n)). 둘을 함께 파생시키는
    이유는 go-wbft 가 `epochLength >= targetValidators` 를 강제하기 때문에 하나만 올리면
    제네시스가 거부되기 때문이다. 값이 한 곳에서만 결정되므로 wbft 단독 체인 경로와 핸드오프
    경로가 갈라지지 않는다.
    - 라이브 3건. **15+15 핸드오프**: 병합 제네시스 `targetValidators: 15, epochLength: 15`,
      `handoff confirmed: head 50; … across 15 of 15 validator(s)`, epoch 경계가 10블록에서
      15블록 간격(블록 20·35·50)으로 옮겨진 것까지 확인. **wbft 4노드**(골든 경로, 로컬):
      `targetValidators: 4, epochLength: 10`, `wbft-chain-up pass`. **wbft 15노드**(docker,
      bp 13): `targetValidators: 13, epochLength: 13`, `wbft-chain-up-15 pass`.
    - 테스트 5건. 파생 자체(집합 크기 일치·`epochLength >= targetValidators`·작은 집합은
      바닥 10 유지)와 **출하 템플릿이 실제로 플레이스홀더를 들고 있는지**를 따로 고정했다 —
      후자가 없으면 파생이 맞아도 도달하지 않는다(이 브랜치 전체가 그 실패 양식이었다).
      변이 3건으로 실패 가능함을 증명했다.
    - 남은 것은 **R7** 로 분리했다: `stabilizingStakersThreshold` 는 5 고정 그대로이고,
      그래서 epoch 재결정 경로 자체가 아직 한 번도 실행된 적이 없다.
  - **(위 문제의 진단 기록)**: `internal/chains/wbft/genesis.json` 의 `targetValidators` 가 집합 크기와
    무관하게 **1 로 고정**이고, 병합 결과에도 1 로 남는다. 출처는 go-wbft 상류의 플레이스홀더
    (`params/config.go` 의 `TargetValidators: newUint64(1), // TODO: define validators`)이며,
    chainbench 템플릿은 최초 커밋(#16)부터 그 값을 그대로 들고 있다.
    - **처음에 "useNCP 가 꺼져 있어 읽히지 않는다"고 적었는데 그건 틀렸다.** `UseNCP` 는
      staker 를 어디서 읽을지만 고른다(`engine.GetStakers`). epoch 경계의
      `decideValidators(…, TargetValidators)` 는 NCP 와 무관하게 돈다.
    - **실제 이유는 체인이 안정화 단계를 벗어나지 않기 때문이다.** genesis epoch 는
      `Stabilizing = true` 로 시작하고(`consensus/wbft/config.go:264`), 안정화 중에는
      `newEpoch.Validators = latestEpochInfo.Validators` 로 직행해 `decideValidators` 를
      아예 부르지 않는다. 벗어나는 조건은 GovStaking 컨트랙트의 staker 수가
      `stabilizingStakersThreshold`(템플릿 5) 이상이 되는 것인데, 핸드오프에서는 producer 가
      **wemix** 거버넌스에 스테이킹했을 뿐 wbft GovStaking 에는 아무도 스테이킹하지 않는다.
      라이브 실측: 블록 30·100·2000 모두 `stabilizing: true`, stakers 15 / validators 15.
    - **벗어나는 순간 validator 가 1명으로 줄어든다.** `decideValidators` 는 stake 내림차순
      정렬 후 `indices[:targetValidators]` 로 자른다(`engine.go:1287`). 따라서 이것은 무해가
      아니라 **아직 도달하지 않은** 경로다.
    - **고칠 때 같이 봐야 하는 제약**: `params/config_wbft.go:130` 이
      `epochLength >= targetValidators` 를 강제한다. 템플릿의 `epochLength` 는 10 이므로
      `targetValidators: 15` 만 바꾸면 제네시스 검증에서 거부된다. 둘을 함께 올려야 한다.
- [x] **R7. epoch 재결정 경로 — 완료·라이브 검증 (2026-09-11).** "한 번도 실행된 적 없다"는
  진단이 맞았고, 이제 실행시켜 보고 상설 테스트로 고정했다.
  - **라이브 (docker 15+15).** wbft `GovStaking` 에 문턱(5) 이상인 **6명을 등록**했다 —
    `registerStaker(amount, staker, feeRecipient, 0, blsPK, blsSig)`, operator 는 producer
    계정 6개(컨트랙트가 `msg.sender != _staker` 와 operator 1인 1스테이커를 강제하므로
    스테이커 수만큼 별도 operator 가 필요하다), staker 는 **실행 중인 wbft 노드** 6개의 주소
    (실행 중이 아닌 주소를 넣으면 경계에서 생산 주체가 사라져 체인이 선다). 등록은 블록 158
    에 들어갔고, **다음 경계인 블록 170 에서 `stabilizing: false`, stakers 6, validators 6**
    으로 바뀌었다. 그 뒤 블록 171~244(74블록)의 봉인자는 **정확히 그 6명**이고 고르게
    교대했다. 제네시스의 15명은 생산에서 빠졌다 — 재결정된 집합은 기존 집합을 확장하는 것이
    아니라 **대체한다**.
  - **이것이 G3 의 파생을 검증한다.** 옛 고정값 `targetValidators: 1` 이었다면
    `decideValidators` 가 `indices[:1]` 로 잘라 **validator 1명**이 됐을 상황이다.
  - **상설 테스트**: `cmd/chainbench/upgrade_gov_epoch_e2e_test.go`
    (`TestWemixGovernanceEpochDecidesTheValidatorSetE2E`, `-tags e2e`). 골든 1+4 프로파일에
    `stabilizingStakersThreshold` 를 4 로 낮추는 genesis overlay 를 씌우고, alloc 으로 펀딩한
    **operator 4개**(preset 키가 아닌 고정 dev 키)가 선언된 validator 4명을 스테이커로 등록한다.
    경계에서 `stabilizing:false` · **decided validators == 스테이킹한 4명**(수만이 아니라 주소
    집합까지) · 그 뒤 한 epoch 동안 실제로 봉인하며 2명 이상이 교대하는 것까지 단정한다.
    라이브 결과: `epoch 40 decided 4 validators from 4 stakers; blocks 41-50 sealed by 4 of them`.
  - **실패 가능함을 변이로 증명했다.** 템플릿을 옛 `targetValidators: 1` 로 되돌리고 같은
    테스트를 돌리면 `block 40 decided 1 validator(s), want 4` 로 실패한다. 즉 이 테스트는
    G3 의 회귀를 실제로 잡는다.
  - **부수로 확인된 것**: `GovNCP.inspectOperation` 은 `!emergencyMode` 만 보므로 스테이커
    등록에 NCP 투표가 필요하지 않다. 기존 `TestWemixGovernanceStabilizingE2E` 주석이
    "NCP 7개가 필요해서 above-threshold 분기는 포팅하지 않았다"고 적어 둔 것은 과한 전제였다 —
    필요한 것은 **operator 계정 수**였고, alloc overlay 로 만들면 된다.

- **(원래 진단, 보존) R7 이 무엇이었나.** wbft 는 epoch 경계마다
  validator 집합을 다시 정하는데(`decideValidators`), chainbench 가 만드는 체인은 그 경계에
  **도달하지 않는다.** genesis epoch 가 `Stabilizing = true` 로 시작하고, 안정화 중에는
  `newEpoch.Validators = latestEpochInfo.Validators` 로 직행해 재결정을 건너뛴다. 벗어나는
  조건은 wbft `GovStaking` 의 staker 수가 `stabilizingStakersThreshold`(템플릿 5) 이상이
  되는 것인데, 핸드오프에서 producer 는 **wemix** 거버넌스에 스테이킹하고 wbft 쪽에는
  아무도 하지 않는다. 라이브 실측(15+15, `istanbul_getWbftExtraInfo`): 관측한 모든 epoch
  경계에서 `stabilizing: true`.
  - **왜 남겨두면 안 되는가.** `targetValidators` 를 집합 크기에서 파생시킨 것(G3)은 제네시스가
    **수용되는지**까지만 라이브로 확인했다. 그 값이 실제로 쓰이는 곳은 재결정 경로 하나뿐이라,
    파생이 의도대로 동작하는지는 아직 관측되지 않았다. 지금 상태에서 회귀가 생기면 안정화를
    벗어나는 첫 체인에서야 드러난다.
  - **기존 커버리지의 경계.** `TestWemixGovernanceStabilizingE2E`(`cmd/chainbench/
    upgrade_gov_staking_e2e_test.go`)가 **below-threshold 분기만** 검증한다. 테스트 주석이
    이유를 적어 뒀다 — `stabilizing -> false` 로 넘기려면 staker 를 문턱까지 올려야 하고,
    그건 useNCP 기반 선출과 거버넌스 NCP 7개, 즉 최소 핸드오프 preset 보다 많은 펀딩된
    계정을 요구한다.
  - **하려면 무엇이 필요한가.** (1) wbft `GovStaking` 에 문턱 이상(5명)이 스테이킹하도록 계정을
    펀딩·등록, (2) epoch 경계를 하나 넘기고 `istanbul_getWbftExtraInfo` 로 `stabilizing: false`
    확인, (3) 그 시점의 validator 수가 **집합 크기 그대로**(`targetValidators` 로 잘리지 않음)
    인지 단정. 15+15 프로파일은 producer 15명이 이미 펀딩돼 있어(handoffBalance) 계정 수는
    문제가 아니다 — wbft 쪽 `stake()` 를 태우는 절차가 없는 것이 문제다.
  - **판단이 필요한 선택지.** 문턱을 낮춰(예: 1) 재결정을 빨리 관측할 수도 있지만, 그러면
    체인이 epoch마다 집합을 다시 정하므로 다른 케이스들의 안정성 가정이 바뀐다. 문턱은 집합
    크기가 아니라 운영 파라미터라 G3 의 파생에 넣지 않았다.
- [x] **R6. go-wemix boot-etcd collapse.** 키·genesis 문제가 아님을 확인하고 넘겼다.
  **체인팀 몫이다. 인계 완료 (2026-09-12)** — `docs/dev/chain-handover-2026-09-12.md` §2 (W2).
  - **인계하면서 이전 판독을 고쳤다.** "join 하려다 형성된 클러스터를 잃는다" 는 **틀렸다** —
    join 경로는 `etcdIsRunning()` 으로 가드되어 돌고 있는 로컬 etcd 를 내릴 수 없고,
    `etcd join failed … "not found"` 는 `etcdAutoJoin` 이 **etcd 를 돌린다고 광고하는 피어를
    하나도 못 찾은** 결과다(원인이 아니라 결과).
  - **범위가 한 질문으로 줄었다.** `etcdInfo()` 는 `ma.etcd == nil` 일 때만 `.cluster` 가
    `undefined` 로 보이고(멤버 조회 실패는 빈 문자열이다), **생산 코드는 `ma.etcd` 를 다시
    nil 로 만들지 않는다**(`wemix/` 전체에서 그 대입은 테스트 한 곳뿐). 그러면 `VerifyEtcd`
    가 통과한 프로세스와 `undefined` 를 돌려준 프로세스는 같을 수 없다 — **부트가 재시작했다는
    뜻이고, 그 죽음이 W1 일 수 있다**(미확인: W2 는 2026-09-02, W1 패닉 관측은 2026-09-11).
- [x] **validatorset 홈 결정 — 완료 (2026-09-12).** "어느 모듈로 옮길까"가 아니라 **무엇을
  잘못 소유하고 있나**가 답이었다. `validatorset.Load` 가 **합의 패밀리를 문자열로 분기**하고
  있었다(`switch family { case "wbft": … case "poa": … }`). 이 저장소가 다른 곳에서 래칫으로
  금지한 바로 그 패턴이다 — 대상에게 묻는 대신 대상을 보고 분기한다.
  - **패밀리 지식은 패밀리가 소유한다.** `registry.RingAccountReader` 를 더했다 — 이미 있는
    `GenesisValidatorReader`·`RuntimeValidatorReader` 와 같은 선택적 capability 형태다.
    wbft 는 validator(+BLS 유무)와 거버넌스 council 을, poa 는 **빈 목록과 이유**를 돌려준다
    (제네시스에 validator 가 들어갈 자리 자체가 없다 — 그냥 비워 두면 *키가 빠진 키셋*처럼
    보인다). `validatorset` 은 조합만 남았고, 패키지 자체는 제자리(domain)에 둔다 — code-graph
    가 이미 `chains`·`consensus`·`accounts` 와 같은 domain 으로 분류하고 있었다.
  - **default 가 구조적이 됐다.** 전에는 switch 가 모르는 패밀리를 "unknown" 으로 보고했는데,
    그건 체인이 아니라 **이 패키지**에 대한 사실이었다. 이제 "아직 말하지 않았다"로 바뀐다.
  - **역할 어휘를 한 곳으로.** `AccountValidator`·`AccountGovernance`·`AccountNode` 를
    `registry` 로 옮겼다(그 이름들을 담는 타입 곁). `validatorset` 에 중계를 두려다
    `arch.TestNamesDoNotCollide` 에 걸렸다 — 중계도 두 번째 선언이고, 래칫이 옳았다.
  - 새 커버리지 래칫: **등록된 모든 패밀리가 이 capability 를 구현해야 한다**
    (`TestRoster_EveryRegisteredFamilyAnswers`). 구현을 잊으면 로스터가 조용히 노드 신원만
    보여주는데, 운영자는 그것을 읽고 믿는다. 변이 2건으로 실패 확인. CLI 출력은 세 체인 모두
    이전과 동일하다(동작 보존 리팩터).
- [x] **health 를 inspector 조합으로 재배선 — 철회 (2026-09-12).** 두 패키지를 읽으니 **조합할
  공통 기반이 없다.** `inspector` 는 실행 **전** 대상 환경을 TCP·파일스토어로 묻는다(포트가
  잡혀 있나·경로가 없나·호스트가 닿나). `health` 는 **체인**이 생산 중인지를 RPC 로 묻고
  판정한다. health 를 inspector 위에 올리려면 inspector 에 "이 노드들을 RPC 로 샘플링하라"는
  네 번째 질문을 줘야 하는데, 그건 아무도 요구하지 않고 inspector 자신의 doc 이 배제한다.
  - **결정을 산문이 아니라 제약으로 남겼다**: `arch.TestInspectorStaysOnTheEnvironmentSide`
    가 `core/inspector` 의 `core/rpc` import 를 거부한다. 재배선의 첫걸음이 이유와 함께 실패한다.
    변이로 확인.
  - **겹치는 것이 없다는 말은 아니다.** 체인 생존을 샘플링하는 곳이 셋이다 — `health`(verify 의
    판정), `collector`(대시보드 스트림), 엔진의 launch gate. health 의 doc 이 gate 를 분리해
    두어야 하는 이유를 이미 적어 두었다(하나는 전 노드 보고, 하나는 한 노드로 기동을 막는다).
    **health 와 collector 가 샘플러를 공유해야 하는가** — **답: 공유하지 않는다 (2026-09-12).**
    두 샘플의 교집합은 height·peers 둘뿐이고, health 는 chainId·syncing 을, collector 는
    headHash·headMiner 를 따로 필요로 한다. 모양도 다르다 — collector 는 주기적으로 폴링하며
    **이력을 누적**하고(fork 검출·BP 참여 집계가 창을 요구한다), health 는 요청 시 **두 번
    샘플링해 판정**한다. 합치면 한쪽 로직을 다른 쪽 위에 다시 구현하게 된다. health 의 doc 이
    launch gate 를 분리해 둔 이유와 같다.
    - **대신 진짜 결함이 나왔다.** 물으려던 질문("두 샘플러가 무엇을 아는가")을 나란히 놓고
      보니, **collector 는 분기를 검출하는데 health 는 보지 않았다.** `health.Report` 는
      `Producing` 과 노드별 상태뿐이었고, `Producing` 은 **primary 노드의 높이가 오르는지**다 —
      **갈라진 망은 갈라진 양쪽이 모두 생산하므로 그 조건을 만족한다.** 즉 `chainbench verify`
      가 분기된 망을 healthy 로 보고했다. DSL 에는 `sameBlockHash` 어서션이 이미 있었으니,
      능력이 없던 게 아니라 **판정에 배선되지 않았다.**
    - 고쳤다: `health.Agreement` 를 더해 **응답하는 모든 노드가 가진 가장 높은 블록**에서
      해시를 비교한다. 머리(head)가 아니라 공통 높이인 이유는, 한 블록 뒤처진 노드가
      **정상적으로** 다른 머리 해시를 갖기 때문이다 — 머리로 비교하면 지연을 분기로 오판한다.
    - **검사를 못 했을 때 "합의함" 으로 보고하지 않는다.** `Checked` 와 `Agreed` 를 분리했다.
      노드가 하나뿐이거나, Prober 가 해시를 못 읽거나, 한 노드가 응답을 거부하면 `Checked=false`
      와 이유를 낸다. 이번 세션에서 반복된 공허한 참(빈 successor 집합·빈 `In` 집합)과 같은 자리다.
    - **라이브 증명**: 서로 독립인 두 wbft 망의 노드를 섞어 물었다 —
      `producing: true` / `agreement: NO — block 77 has 2 different hashes — 0x7dd9cf5d3a…: [1];
      0xa874c0a900…: [2]`. 첫 줄이 예전 `verify` 의 답 전부였다. 정상 망에서는
      `agreement: yes, at block 79`.
    - 단위 6건(분기·정상·지연은 분기 아님·검사 불가 3종) + 변이 2건(머리로 비교하기·검사 불가를
      합의로 처리하기). CLI·MCP 두 표면 모두 `producing` 옆에 나란히 보고한다.
- [x] **Phase 2~6 의 부분 완료 항목 — 닫혔다 (2026-09-12).** 이 항목의 본문이 이미 T2.1·T3.3·T5.1·T5.5 를 전부 해소로 적고 있는데 머리글만 부분으로 남아 있었다. 아래 각 절의 `◐` 도 같은 이유로 ☑ 로 바꿨다. **(원래 기록, 2026-09-11 재확인)** — **T2.1** 은 남은 것이 없다 — Transport 타입은
  이미 쓰였다가 "Driver 가 이미 그것이고 구현체가 없다"는 이유로 삭제됐다(철회, 각 절 참고). **T3.3·T5.1** 에 남아 있던 "실 SSH 호스트 대상 라이브
  e2e" 는 **완료됐다 (2026-09-11)** — `env/docker` 의 15대가 SSH 로 닿는 원격이고, 원격 로그
  tail 을 실 sshd 에 대고 검증했다(각 절 참고). **T5.5**(wemix4 이관)도 **완료였다 (2026-09-12 확인)** — 포트 트래커의 케이스 표는
  47건이 전부 `ported` 이고 `covered`·`deferred` 가 하나도 없으며, 마지막 항목이
  "This was the last open wemix4 item — the suite is now fully ported/covered" 라고 적어
  두었다. 이름이 적힌 테스트 6건이 저장소에 실재하는 것도 확인했다(`TestE2E_WbftSnapSync`,
  `TestE2E_WbftGenesisNCPWhitespace`, `TestE2E_WbftGenesisEmptyNCP`,
  `TestWemixDataMigrationE2E`, `TestWemixGovernanceCredentialExpiryE2E`,
  `TestWemixGovernanceFeeChangeDelayedE2E`). **T5.2·T6.6 에 이어 세 번째로, 끝난 일이 두
  기록 사이에서 미완으로 보이던 경우다.** 트래커의 "Remaining work" 머리말이 GOV-023 을
  미해결로 적고 케이스 표는 ported 로 적는 모순도 함께 고쳤다.

### F. 거짓 보증 — 통과해도 아무것도 막지 못하는 검사 (신규 2026-09-12)

`feat/invariant-test-catalog` 브랜치(2026-06-16, PR 없이 남아 있었다)를 읽다가 나왔다. 그
브랜치는 불변식 5개를 기존 테스트에 묶으면서 **binding 등급을 정직하게** 매겼다 — `good` 은
"그 불변식의 *구별되는* 주장을 단정한다", `partial` 은 "메커니즘은 단정하지만 구별되는 주장은
아니다 → 초록불이 **거짓 보증**을 준다". 그 `partial` 세 건을 오늘 다시 측정했다.

한 건은 그 사이에 닫혔다. 둘은 그대로 열려 있고, **둘 다 이 저장소가 반복해서 만난 모양**이다:
증거처럼 보이지만 증거가 아닌 검사.

- [x] **F1 — 완료 (2026-09-12).** `regression/anzeon/12-basefee-redistributed-not-burned`.
  설계대로 갔고, **부등식이 아니라 등식으로** 끝났다 — 변이 2번이 정확한 수를 보여 준 덕이다.
  - **실측**: 팁 0(인가 계정) 전송 1건에서 validator 4명이 **각각 105000000000000000 wei**,
    송신자는 **420000000000000001 wei** 감소(1 wei 는 옮긴 value). 즉 4등분의 합이 낸 것과
    정확히 같다. 그래서 "늘어난다" 대신 **공급 보존을 등식으로** 단정한다:
    `paid(=sbefore-safter-1) == credited(=d1+d2+d3+d4)`.
  - 단정 8개: 네 증분 각각 > 0(봉인 안 한 셋도 받는다 — 팁은 coinbase 한 곳으로만 간다) ·
    세 쌍이 서로 같음(Diligence 동일) · **보존 등식 1개**.
  - **변이 3건, 각각 다른 절반이 깨진다.** (1) 전송 제거 → 네 증분이 **전부 0** 이 되어 성장
    단정 4개가 실패한다. 이것이 "다른 크레딧 경로가 없다"(고정 블록 보상 없음)를 **외부에서**
    확인해 준다. (2) 한 읽기를 validator 대신 송신자로 → 그 증분이 음수가 되어 성장·동일성이
    함께 실패한다. (3) **credited 를 4개 중 3개 합으로** → 보존 등식만 실패하고(420 vs 315)
    나머지 7개는 통과한다. **한 몫을 소각한 모양이며, 기존 6건이 못 잡는 것이 바로 이것이다.**
  **(원래 진단)**
  go-stablenet 의 baseFee 는 EIP-1559 와 달리 소각되지 않고 validator 에게 재분배된다
  (WKRC 공급 보존). 그런데 `tests/tc/go-stablenet/regression/anzeon/` 의 basefee 스펙 6건
  (`03-increase`·`04-stable`·`05-decrease`·`06-minimum`·`07-maximum`, `09-feecap-exact-min`)은
  **공식만** 본다 — 사용률에 따라 baseFee 가 오르내리는지.
  - **그래서 baseFee 를 소각하도록 바꿔도 6건 전부 초록이다.** 공급 보존이라는 구별되는 주장을
    단정하는 검사가 0건이다. 재측정(2026-09-12): baseFee 를 언급하는 스펙 중 coinbase·validator
    잔액을 단정하는 것은 없다. 확인: `for f in $(grep -rln basefee tests/tc/go-stablenet); do grep -li "coinbase\|balance" $f; done`
  - **재분배 경로를 소스에서 확인했다** (go-stablenet `0937ac5c9`,
    `consensus/wbft/engine/engine.go:972` `distributeBaseFee`): `BaseFee × GasUsed` 를 그 epoch
    validator 들에게 **`Diligence` 비례로 나누고** 나머지(dust)만 `header.Coinbase` 로 보낸다.
    `GasUsed == 0` 이면 건너뛴다. 즉 **블록을 봉인하지 않은 validator 도 받는다** — 이것이
    구별되는 주장이다. 상류에 단위 테스트(`engine_test.go:1026` `TestBaseFeeDistribution`)가
    있고, 없는 것은 **종단 검증**이다.
  - **주의 — 순진한 단정은 그 자체로 공허하다.** "validator 잔액이 늘어난다" 만 보면 안 된다:
    그 validator 가 블록을 봉인하면 `effectiveTip` 을 coinbase 로 받으므로(`core/state_transition.go:576`)
    **baseFee 를 소각해도 잔액이 오른다.** 통과하지만 아무것도 막지 못하는, 이 항목이 지적하는
    바로 그 모양이 된다.
  - **그래서 팁을 0 으로 만들어야 한다.** anzeon 에는 이미 그 장치가 있다 — 인가된 계정은
    gastip 을 면제받는다(`regression/anzeon/02-authorized-account-gastip-free`, 컨트랙트
    `0x…1004` 의 2-of-N 승인 흐름이 DSL 로 이미 쓰여 있다). **인가 계정이 `maxPriorityFeePerGas: 0`
    으로 부하를 보내면** coinbase 의 팁 수입이 0 이 되고, 그때 남는 잔액 증가는 재분배뿐이다.
  - **설계**: (1) 02 의 인가 흐름을 재사용해 새 계정을 인가·펀딩, (2) validator 2·3 의 잔액을
    읽어 저장(1번은 제외한다 — 기존 부하 스펙의 송신자이고 수수료를 내는 쪽이라 혼입된다),
    (3) 팁 0 으로 여러 건 전송, (4) **두 validator 모두** 잔액 증가를 단정. 소각이면 둘 다
    실패한다. 변이는 전송 스텝을 빼서 두 단정이 함께 실패하는 것으로 확인한다.
  - 주소: validator 1 `0xc17d4938…`(= 기존 부하 송신자), 2 `0x2493a84a…`, 3 `0x8c4a10b9…`
    (`presets/keys/metadata.json`). 네 validator 의 `Diligence` 는 extraData 에서 동일하므로
    지분도 같다.
- [x] **F2 — 완료 (2026-09-12).** `regression/api/12c-validator-equal-power`. 가중치를 돌려주는
  RPC 가 없으므로 **동등성의 관측 가능한 결과 둘**을 단정하고, **무엇을 단정하지 않는지도
  스펙에 적었다**.
  - **턴**: 네 validator 각자가 최신 블록의 제안자로 등장할 때까지 기다린다 — 슬롯이 없는
    validator 는 영원히 나타나지 않고 대기가 타임아웃으로 실패한다.
  - **표**: `prevCommittedSeal.sealers` 가 **정족수가 아니라 집합 전체**여야 한다. 30블록 연속
    4/4 로 실측했다. 이것이 "표 1개씩 동등하게 센다" 의 외부 관측 형태다.
  - 변이 2건: (1) 네 번째 대기를 validator 아닌 주소로 → 타임아웃 실패. (2) 완성된 seal 대신
    **진행 중 seal**(`committedSeal`)을 읽으면 집합 크기 4 에 대해 **3** 이 나와 실패한다 —
    "정족수로 충분했다" 와 "모든 표가 세어졌다" 의 차이가 정확히 이것이다.
  - **단정하지 않는 것**: 한 사이클 안의 턴 수 동등성. DSL 이 **런타임에 알게 된 높이 기준
    상대 블록을 주소지정하지 못한다**(`eth_getBlockByNumber` 는 16진수를 요구하고 `derive` 는
    10진수를 낸다). 고정 높이로 대신하는 것은 아래 F4 때문에 위험하다.
- [x] **F4 — 완료 (2026-09-12). 제가 적은 기제는 틀렸고, 진짜 원인은 더 컸다.**
  - **정정: "로테이션이 동기화된 노드를 건너뛴다" 는 틀렸다.** 노드 로그가 그 반대를 말한다 —
    `next.valSet` 은 항상 4명 전체이고 빠진 노드의 차례에 `next.proposer` 가 그 노드로
    **정확히 배정된다.** 그 다음에 `TIMER CHANGING ROUND` 가 뜨고 round 1 의 제안자가 다음
    validator 로 넘어간다. **로테이션은 한 번도 틀리지 않았다** — 그 노드의 차례가 매번
    라운드 체인지 타임아웃으로 날아간 것이다. 건강한 실행은 **블록 1부터 주기 4** 로 깨끗하게
    돈다(대조군).
  - **진짜 원인은 게이트가 자기 조건을 채우지 않은 것이다.** health 보고를 facts 로 옮기는
    자리가 `Wanted` 만 세우고 `WantPeers`·`WantChainID`·`WantParticipate` 를 0 으로 남겼다.
    그래서 분류기의 **세 조건이 실행 경로에서 발화할 수 없었다** — 작성되고, 분류기 단위
    테스트까지 있고, 호출자에서 닿지 않는다. **want 쪽이 채워지지 않은 비교**와 같은 모양이며,
    이번 작업에서 이 계열 코드에서 **세 번째**로 만난 것이다(F6 의 preflight, F6 의 오버레이 경로).
  - **왜 아무 조건도 못 잡았나.** 그 노드는 `syncing` 이 아니었다(피어가 없으면 동기화할
    상대가 없다). 그리고 `advancing` 은 **네트워크에 대한 사실**이라 생산 중인 세 노드가
    그 노드 몫까지 참으로 만든다. 그래서 `READY` 가 나왔다.
  - **더한 조건 둘.** (1) 다중 노드 네트워크에서 **피어 최소 1개**. 하한을 n-1 이 아니라 1 로
    둔 이유: proxied 토폴로지는 bp 에게 의도적으로 pn 하나만 준다 — n-1 이면 **선언한 대로
    연결된 네트워크를 거부**한다. (2) **네트워크 머리보다 크게 뒤처지지 않을 것.** 전에는
    facts 에 네트워크 머리가 아예 없었다(`health.Report` 는 네트워크 전체에 대한 `Producing`
    하나만 든다). 허용 오차 2블록 — 노드를 하나씩 읽는 동안 체인이 계속 움직이므로.
  - **일부러 채우지 않은 둘, 이유를 코드에 적었다.** `WantParticipate` 는 이 관측자에
    `Participating` 출처가 없다. `WantChainID` 는 불일치가 **FATAL** 이므로 기대값이 권위
    있어야 하는데, 관측에서 가져오면 순환이고(전부 틀린 genesis 면 자기와 일치한다),
    워크스페이스는 체인의 **이름**을 기록하며 요청의 숫자 id 는 매니페스트가 줄 때 0 이다.
    먼저 답한 노드에서 채우면 **치명 판정이 순회 순서에 달린다.**
  - 단위 테스트: 두 조건과 이를 먹이는 매핑(머리는 첫 노드가 아니라 **최대**·1노드 네트워크는
    피어를 원하지 않음·뒤처짐이 죽은 프로세스를 앞지르지 않음). **변이 6건**, 각각 빌드를
    살린 채 실패 확인.
  - **라이브로 보이지 못한 것**: 새 거절이 실제로 발화하는 장면. 따라잡기 자체가 30블록에
    ~300ms 라 "뒤처졌고 동시에 피어가 있는" 창이 너무 짧고, 실제 정지는 **피어가 없던 30초**
    창이었는데 그 경합을 요청대로 재현하지 못했다. **진단은 라이브, 거절은 단위 테스트**다.

  [권장] F2 를 쓰다 실측으로 나왔다. 4노드 stablenet 에서 게이트는 **블록 2 전에 4대 전부
  READY**(alive·rpc·advancing)로 보고했지만, **네 번째 validator 는 블록 31 에 처음 제안**했다.
  블록 1~30 은 셋이서 주기 3 으로 돌았고, 블록 32 부터 깨끗한 주기 4 가 됐다(블록 40 이후
  `miner(n) == miner(n+4)` 위반 0건).
  - **왜 문제인가**: "advancing" 은 **따라잡고 있다**는 뜻이고 **제안 차례를 받는다**는 뜻이
    아니다. 로테이션 균등성·제안자 신원·epoch 경계를 **고정 높이로** 보는 스펙은 이 창에서
    조용히 다른 것을 측정한다. 지금 저장소에 그런 스펙이 있는지는 확인하지 않았다.
  - **선택지**: (a) 게이트에 "모든 validator 가 최소 1회 제안" 을 조건으로 더한다 — 정확하지만
    브링업이 최대 한 사이클+싱크만큼 길어진다. (b) readiness 를 그대로 두고 **스펙이 기다리게**
    한다(F2 가 택한 방식) — 싸지만 스펙마다 반복된다. (c) 사실을 문서에만 적는다 — 가장 싸고,
    다음 사람이 같은 함정을 밟는다.
  - 재확인: 4노드를 세우고 `eth_getBlockByNumber` 로 블록 1~40 의 `miner` 를 세어 본다.
  **(원래 진단)**
  불변식은 "모든 활성 validator 는 동등한 투표 권한을 가진다" 인데, `istanbul_getValidators`
  를 쓰는 스펙들(`go-stablenet/regression/api/12-validator-set-nonempty`·`12b-validator-set-count`,
  `go-wbft/chain-up/01`·`02`, `basic/07`)은 **집합 멤버십과 개수**만 단정한다. 가중치·표=1 을
  보는 것은 0건이다. 확인: `grep -rln "votingPower\|weight" tests/tc` 무결과.
  - **어떻게 볼지가 F1 보다 어렵다.** 권한을 직접 읽는 RPC 가 있으면 그것을 단정하고, 없으면
    **프로포저 로테이션의 균등성**이 대리 지표다 — epoch 하나에서 각 validator 가 제안한 횟수가
    같은 범위에 드는지. 후자는 대리 지표이므로 그렇게 적어야 한다.
  - F1 보다 우선도가 낮은 이유: 동등 권한이 깨지는 경로가 지금 코드에 없다(제네시스 목록이
    곧 집합이고 가중치 필드가 없다). **지금은 회귀 방지**이고, 거버넌스로 가중치가 들어오면
    그때 차단이 된다.
- [x] **F3. WBFT 정족수 공식의 수치 직접 검증 — 이미 닫혔다 (2026-09-11).** 카탈로그가
  "임계 동작만 보고 `ceil(2N/3)` 수치를 직접 단정하지 않는다" 고 지적한 것은 그 시점에
  맞았다. `fault/02-wbft-quorum-at-15-nodes` 가 그것을 닫았다 — validator 13 에서 정족수 9 를
  **수치로** 붙잡고, 9 면 계속 돌고 8 이면 멈추는 양쪽을 한 스펙에서 증명한다.

**카탈로그 파일 자체는 가져오지 않았다.** `catalog/invariant-tests.yaml` 은 (1) 이 저장소에
없는 외부 경로(`study/docs/research/ai-knowledge-data/...`)를 불변식 ID 의 출처로 적고,
(2) 테스트 `ref` 를 옛 레이아웃(`regression/d-fee-delegation/...`)으로 적는다 — 지금은
`tests/tc/...` 다. 즉 **파일을 살리려면 두 축을 다시 매핑**해야 하고, 그 값은 위 두 항목을
직접 고치는 것보다 작다. 브랜치는 남겨 두었다(`origin/feat/invariant-test-catalog`).

### 이번에 바로잡은 표시

- §1o 의 WA1~WA26 체크박스는 PR #363 으로 반영을 마쳤는데 미완으로 남아 있었다. 절
  머리에 상태와 표본 확인 근거를 적었다.
- `hardfork 는 통폐합 대상 아님` 은 할 일이 아니라 내린 결정이라 ☑ 로 바꿨다.

**2026-09-12 — 그 걸음을 기계가 대신하게 했다.** 앞 항목에서 "나머지 18개는 한눈에 맞춰 본
것이지 검증이 아니다, 각각에 증거를 붙이는 일은 따로 열어야 한다" 고 적었다. 그 일을 했다.

- **§12 표에 `증거` 열을 더했다.** 요구 19개 각각이 **그것을 붙잡는 테스트 하나**를 이름으로
  댄다. 이름은 추측하지 않고 테스트 코퍼스에서 찾아 붙였다 — 예: R03 은
  `TestProvision_ReusesIdenticalOverwritesDifferent`(같으면 재사용, 다르면 덮어쓴다 —
  "동일성 검사" 가 바로 그것), R08 은 `TestBinaryFor_ResolvesPerNodeOverFallback`,
  R17 은 `TestCheck_ADifferentGenesisIsADifferentChain`.
- **`arch.TestEveryRequirementNamesEvidence` / `…NamedTestExists` 가 그 표를 지킨다.**
  기계가 붙잡을 수 있는 둘만 붙잡는다: **모든 요구가 증거를 댄다**, 그리고 **댄 이름이 실제로
  존재한다**. 이름이 바뀌거나 지워지면 표가 답한 척하면서 아무것도 답하지 않게 되는데, 그것이
  이 세션이 계속 고쳐 온 낡은 기록과 **정확히 같은 모양**이다.
- **일부러 하지 않는 것**: 그 테스트가 그 요구를 *증명하는지* 는 판정하지 않는다. 기계가 할 수
  없고, 하는 척하면 이 패키지가 없애려는 바로 그 공허한 검사가 된다. 표는 **연결**을 보증하고
  **충분성**은 사람이 읽는다.
- 변이 3건: 증거 칸 비우기 · 이름을 없는 테스트로 바꾸기 · 열 자체를 빼기. 셋 다 실패 확인.
- 19개 이름을 실제로 한 번에 돌려 전부 통과하는 것도 확인했다
  (`go test ./... -run "^(Test…|Test…)$"`, 13개 패키지).

**2026-09-12 — 요구 추적성(R01~R19)을 처음 걸어 봤고, R10 에서 결함을 찾았다.** 워크리스트가
비었으므로 다음은 도출해야 했다. 두 차례 전 "커버리지만 하고 있는 것 아닌가" 지적 이후 제가
"옳은 축" 이라고 적고도 **한 번도 걸어 보지 않은 것**이 정본(`chainbench-system-direction.md`)
§12 의 요구 19개다. 어느 것도 검증하는 테스트나 문서가 없었다(`grep -rln "R0[1-9]" docs/ internal/`
→ 그 문서 하나).

- **먼저 증거가 가장 약한 것을 골랐다.** R01~R09·R11~R19 는 이 세션의 작업이 직접 증거를
  남겼거나(R02 docker 원격·R13 `report --all`·R14 한 명령 10스펙·R17 preflight·R18 nodemonitor·
  R19 세션 디렉토리) 기존 기능으로 바로 확인된다. **R10 만 제가 설명할 수 없었다.**
- **R10 은 두 조각이 빠져 있었다.** 요구는 "원본 fixture, override, 최종 config, **적용 노드·시각**과
  checksum 을 보존한다" 인데, `ConfigProvenance` 에 **시각이 없었고**, `setConfigProvenance` 가
  같은 노드의 항목을 **교체**했다. 즉 `swapNode` 가 mid-test 로 config 를 바꾸면 **구성 당시의
  기록이 사라진다.** 코드 주석이 각 기록을 "a new revision" 이라 부르면서 하나만 남기고 있었다 —
  이름은 리비전인데 역사가 없었다.
- **고친 것**: `addConfigProvenance` 가 **append** 하고, 각 리비전이 주입된 시계로 찍힌 `at`
  (RFC3339, UTC)을 든다. 명시된 `at` 은 덮어쓰지 않는다(재구성된 역사가 순서를 잃지 않게).
  구성 단계는 리스트를 비우고 다시 쌓으므로 재구성이 낡은 기록을 남기지 않는다.
- 단위 테스트 3건 + 변이 3건(교체로 되돌리기 = 이 변경 직전 상태 · 시각 제거 · 명시 시각 무시).
  셋 다 실패 확인.
- **남은 18개는 "한눈에 맞춰 봤다" 수준이고 검증이 아니다.** 각각에 증거 명령을 붙이는 일은
  따로 열어야 한다 — 지금 적으면 걸어 보지 않은 것을 걸어 봤다고 적는 것이 된다.

**2026-09-12 — 라이브 없이 보낸 것을 라이브로 검증했다 (11/11).** PR #414 가 `wait.Sleep`
으로 **11곳**을 바꿨는데, 그중 **8곳이 poa 브링업 경로**다. 단위 테스트만 돌리고 보냈고, 이
저장소의 기록이 반복해서 말하는 것이 바로 그 위험이다 — 라이브가 단위 테스트가 못 잡는 부류를
잡는다(핸드오프 결함 6건, 원격 로그 잘림, `ringFor`). 그래서 돌렸다.

| 검증 | 덮은 사이트 | 결과 |
|---|---|---|
| **15노드 wemix 브링업** (docker 15대, 13 bp + 2 en, `tests/tc/common/node/002-wemix-chain-up-15.json`, 그때 `tests/tc/go-wemix/chain-up/02`) | poa 8곳 — `WaitForIPC`·`WaitForIPCOn`·`WaitSelf`·`VerifyEtcd`·`WaitForMember`·`joinOne`·`WaitProducing`·`WaitEtcdCluster` | **pass.** 직렬 내림차순 브링업 완주, 13 bp 전부 etcd 형성, 블록 생산 |
| **golden 5노드 핸드오프** (로컬, `presets/chain/wemix-upgrade.yaml`) | upgrade 2곳 — `WaitEndpointsReady`·`AwaitFork` | **pass.** `handoff confirmed: head 30; blocks 21-30 all sealed by the successor set, across 4 of 4 validator(s)` |
| 단위 테스트 | `collector.WaitLog` 1곳 | 타임아웃 경로(`"nonexistent"`, 100ms)가 그 루프를 실제로 돈다. **프로덕션 호출자는 없다** — 인터페이스 계약이다 |

- **8곳이 어디서 쓰이는지 먼저 확인하고 실행을 골랐다.** 각 사이트의 함수를 AST 가 아니라
  `awk` 로 거슬러 올라가 이름을 뽑고, 그 함수들이 poa 브링업이 지나가는 자리임을 확인한 뒤
  15노드 브링업을 선택했다. 핸드오프는 2곳만 추가로 덮으므로 **5노드 golden** 으로 충분했다
  (15+15 는 30노드 키셋이 필요하고 W1 패닉 위험이 있다).
- **golden 실행은 W1 을 밟지 않았다.** 로컬 5노드다. W1 은 15노드 docker 에서만 관측됐다.
- 재확인: `ls ~/.chainbench/ws-live15/sessions/*/session.json` · `grep "handoff confirmed" /tmp/golden.log`
  (세션은 재실행하면 새로 생긴다 — 위 인용이 그 실행의 출력이다)

**2026-09-12 — 브랜치 정리와, 남기고 싶었던 것.** 원격에 PR 없는 브랜치가 둘 있었다.

- `chore/mark-test-only-signing-key` — **삭제했다.** 커밋 `6f054014` 의 내용(테스트 키 3곳의
  `betterleaks:allow` 주석)이 main 에 그대로 있다. 확인은 `git diff origin/main...<branch>`
  (세 점 — 두 점은 main 의 진전을 브랜치의 삭제로 보여줘서 349파일 diff 가 나온다)와
  `git grep betterleaks:allow origin/main` 으로 했다. 되살릴 필요가 생기면 `6f054014` 다.
- `feat/invariant-test-catalog` — **삭제했다 (2026-09-12, 사용자 승인).** 되살릴 필요가
  생기면 `31764df51fb8faf508dd6f62ce0031d8a3637d33` (2026-06-16, `catalog/invariant-tests.yaml`
  78줄 한 파일)이다. 지우기 전에 남길 것을 먼저 남겼다 — 이 브랜치의 **발견**은
  F1·F2·F3 으로 흡수했지만 **방법**은 어디에도 없었다. 그 방법이 이 세션에서 값을 증명한
  것이라 [`../guide/dsl-authoring.md`](../guide/dsl-authoring.md) §8 로 옮겼다 — "통과하는데
  아무것도 막지 못하는 케이스를 쓰지 않는 법".
  - 옮긴 것: **구별되는 주장 vs 메커니즘**을 가르는 질문("이 단정을 통과시키면서 불변식을
    깨는 구현이 있는가"), 이 세션의 실례 4건(baseFee 공식·validator 멤버십·`reason` 없는
    reject·팁으로도 오르는 잔액), **변이의 함정 둘**(적용됐는지 확인 / 실패 건수를 센다),
    그리고 **쓸 수 없는 것은 쓸 수 없다고 적는다**.
  - 그래서 **버려도 잃는 것이 없었다** — `catalog/invariant-tests.yaml` 은 이 저장소에 없는
    외부 경로를 불변식 ID 출처로 적고 테스트 참조가 옛 레이아웃이라, 되살리려면 두 축을 다시
    매핑해야 한다. 값은 파일이 아니라 **등급 매기는 방법**이었고 그것은 가이드에 있다.
  - **원격 브랜치는 이제 `main` 뿐이다** (작업 브랜치 제외). 확인:
    `git ls-remote --heads origin`

**2026-09-12 — "한쪽이 채워지지 않음" 을 래칫으로 고정했다.** 이번 주에 같은 결함을 세 번
만났다. 세 번 모두 diff 에서는 버그처럼 보이지 않았다 — 필드도, 비교도, 조건도 전부 작성돼
있고 테스트까지 있는데, **값을 넣는 쪽이 없어서 배포된 날부터 무력**이었다.

| 결함 | 죽어 있던 것 | 결과 |
|---|---|---|
| `preflight.Want.GenesisDeclared` | genesis 비교(양쪽 중 하나만 비면 건너뛴다) | 오버레이만 다른 두 환경이 한 네트워크를 공유, 두 번째 테스트가 첫 번째의 체인에서 돌았다 |
| `nodemonitor.Facts.WantPeers` | 피어 없는 노드를 거부하는 조건 | validator 4대 중 1대가 높이 0에 있는데 게이트가 ready 라고 했다 |
| `nodemonitor.Facts.Forked` | 분기를 FATAL 로 보는 조건 | **이번에 찾았다** — health 가 답을 갖고 있고 분류기가 조건을 갖고 있는데 아무도 옮기지 않았다. 포크된 망이 ready 로 통과한다 |

`arch.TestEveryFieldAProducerOwnsIsFilledOrExplained` 가 이 부류를 잡는다. **grep 으로는 판정
할 수 없다** — CLI 플래그를 리플렉션으로 바인딩하는 요청 타입은 소스에 대입이 아예 없으므로
"한 번도 대입되지 않았다" 가 참이면서 아무 뜻도 아니다(실측: 순진하게 세면 100건, 태그를 걸러도
61건, 그중 `UpgradeRunIn.AllServers` 는 실제로 동작한다). 그래서 규칙을 **이름 붙인 (타입,
생산자) 쌍**으로 좁혔다 — 손으로 쓴 결정 구조체를 한 함수가 만들어 다른 함수가 읽는 자리,
생산자가 유일한 writer 인 경우다.
- 예외는 **이유와 함께** 적는다. `WantChainID` 는 권위 있는 출처가 없고(관측에서 가져오면
  순환), `WantParticipate`·`Participating` 은 이 관측자에 출처가 없고, `Failure` 는 프로세스
  점검이 분류하는 것이며, `Want.Nodes` 는 개수만 아는 요청이 노드별 사실을 발명할 수 없다.
- 래칫이 **양방향**이다: 채워지지도 적히지도 않은 필드도 실패하고, **적혀 있는데 실제로는
  채워지는** 필드도 실패한다(낡은 예외는 다음 사람에게 목록을 건너뛰라고 가르친다).
- 변이 4건: `GenesisDeclared` 대입 제거(PR #412 직전 상태) · `Forked` 대입 제거(이 커밋 직전
  상태) · 낡은 예외 · 없는 필드 이름. 넷 다 실패 확인.
- **목록은 늘고, 예외는 줄기만 한다.** 새 결정 구조체가 생기면 쌍을 더한다.

**2026-09-12 — `◐` 전수 상태 점검 (19 → 5).** 체크박스는 0건이었는데 부분 표시가 19건
남아 있었다. 하나씩 실측했더니 **14건이 낡은 기록**이었다. 부분 표시는 체크박스보다 더 잘
썩는다 — "일부는 됐다" 는 다시 재도 틀리지 않으므로 아무도 고치지 않는다.

| 항목 | 적혀 있던 것 | 실측 |
|---|---|---|
| WA9 | node-table `role:pn` 이 mesh 로 배선된다 | `topologyHasProxy` 가 node-table 경로를 덮는다 |
| WA10 | attach 가 게이트·증적·Precheck 를 잃는다 | `readWorkspaceComposed` 를 compose·attach 가 공유한다 |
| WA11 | artifacts.json 프로덕션 writer 가 없다 | `recordArtifacts` 가 있고 양쪽 경로가 채운다 |
| WA12 | FAIL 이 reason 을 안 남긴다 | `rec.Reason` 이 세 자리에 있고 라이브 `status.json` 으로 확인 |
| WA18 | `onEach` 를 do-스텝이 소비 안 한다 | `interp/run.go` 가 팬아웃한다 |
| WA20 | 핸드오프 env 가 선언 절반을 무시한다 | 무시가 아니라 **거절**한다 |
| WA22 | `env.capabilities` 가 버려진다 | **합집합**이다 |
| WA25 | 커버리지 얕음 | 아래 §1p 항목이 정본이고 닫혔다 |
| 레거시 경로 정리 | "소비자 이관 전 제거 금지" | 세 패키지가 이미 **없다** |
| Phase 2~6 + T2.1·T3.3·T5.1 | 부분 완료 | 본문이 이미 전부 해소로 적고 있었다 |
| C. metric 수집 | "라이브 검증만 남았다" | 본문이 이미 라이브 통과를 적고, 파생 둘도 고쳐져 있다 |

**절반만 해소된 둘은 절반이라고 적었다.** WA15 는 파싱 시점 검증이 생겼고 런타임 기본값은
아직 없다(판단 필요). WA23 은 live 게이트 없는 실행 테스트가 하나 생겼고 다섯은 남았다.
**진짜 열린 셋**은 WA17(임베드 스키마가 검증에 미사용)·WA21(DSL 표현력 갭 목록 재측정)·
WA24(`hooks.onFail` 은 통과하는 스위트로 도달 불가)다.

**2026-09-12 — 내가 만든 잘못된 측정 하나.** "원격 브랜치 27개가 스테일하다" 고 정리했는데
틀렸다. 원격에는 처음부터 **브랜치가 3개뿐**이었다(`main` + PR 없는 2개). 내가 본 27개는
**로컬 클론의 낡은 remote-tracking 참조**였고, GitHub 가 머지 때 head 브랜치를 자동 삭제하고
있었다. 확인은 `git branch -r`(로컬 캐시)이 아니라 `git ls-remote --heads origin`(원격 진실)
으로 해야 하고, 정리는 `git push --delete`(없는 것을 지우려다 실패한다)가 아니라
`git remote prune origin` 이다. 같은 실수를 막는 문장: **`git branch -r` 은 원격을 보여주지
않는다. 마지막으로 fetch 했을 때의 원격을 보여준다.**


### G. HSM 리팩토링 — 검증 완료 (2026-09-21 착수 · 2026-09-24 전량 검증)

브랜치 `refactor/hsm-state-machine`. `internal/core/lifecycle` 의 표를 걸어 다니던 머신을
계층형 상태 머신으로 바꿨다. 설계는 `docs/research/chainbench/analyses/14`, 작업 prompt 는
`15`, 들어온 구현의 검토는 `16`, 수정 prompt 는 `17`, 두 머신의 state diagram 은 `18` 이다.
14~17 은 `5a206871` 에서 지웠다(`git show 5a206871^:<경로>`). 남은 것은 `18` 뿐이다.

**원칙(사용자 지시).** 길을 다 만들고, 동작을 확인하고, 그다음에 기존 것을 지운다. 그래서
삭제는 마지막 세 커밋에 모였다.

**한 일.** 커밋 31개. `internal/core/statemachine` 을 새로 세우고, chainsetup 에 닿는 경로
일곱을 하나씩 머신 위로 옮기고(`chain up`, 독립 step 아홉, resume, reuse-if-matching,
compose-comparing, 조작 여섯, run), 그다음 옛 머신과 표를 지웠다. 마지막 커밋에서 크로싱의
세 순간을 상태로 만들고 `StepOut.Passed` 와 그것만 채우던 `lifecycle.Status` 17개를 지웠다.

**검증된 것.**

| 무엇 | 결과 |
|---|---|
| 유닛 전량 · `go vet` · `golangci-lint` · `gofmt` | 통과 |
| DSL 라이브 go-wbft 16건 + go-wemix 13건 | 29건 전부 PASS |
| DSL 라이브 go-stablenet 부분 (47/209 에서 중단) | 47건 전부 PASS |
| DSL 라이브 hardfork 4건 (크로싱 두 모양 전부) | 4건 전부 PASS |

**전량 검증 (2026-09-23, docker 15대, HEAD `08035919`).** 로컬에 체인 바이너리가 없는
기기라 스위프를 `env/docker` 15대 위에서 돌렸다 — `TCSWEEP_FLAGS` 로 `--server-set` ·
`--workspace-config` · `--docker` · `--all-servers` 를 넘긴다(env/docker/README.md).

| 무엇 | 결과 |
|---|---|
| 유닛 전량 · `gofmt -l` · `go vet` · `golangci-lint` | 통과, 0 issues |
| `go vet -tags e2e ./...` (e2e 컴파일) | 통과 |
| docker live 테스트 17건 (chainsetup 2 · resource 3 · process 7 · keyringcmd 4 · app 1) | 17건 전부 PASS |
| **209건 DSL 스위프** | **208 PASS · 0 FAIL · 1 BLOCKED** |
| **핸드오버 e2e 21건** (2026-09-24) | **21 PASS · 0 FAIL** |

스위프는 두 번에 나눠 돌았다. go-wemix 13건은 poa 가 노드마다 p2p 옆 3연속 포트를 잡으므로
`server-set-wemix.yaml`(p2p step 3) + `--node-monitor-timeout 5m` 이 따로 필요하다. 한 번의
스위프로 두 서버 세트를 다룰 수 없다.

- [x] **G1. 전량 검증 — DSL 스위프는 끝났다.** 209건 중 208건 PASS, **FAIL 0건**.
  15노드 구성(`wbft-chain-up-15`, `stablenet-chain-up-15`)과 정족수 계열
  (`wbft-quorum-at-15-nodes` 129s, `wbft-quorum-halt-and-recover` 98s)이 전부 통과했다 —
  HSM 리팩토링이 건드린 면이 바로 이쪽이다. 남은 1건은 아래 G6.
- [x] **G1b. e2e 태그 계층도 돌았다 (2026-09-23 최초 · 2026-09-24 재측정).** 이 계층은 **호스트에 darwin/arm64 체인
  바이너리**를 요구한다 — docker 는 대신하지 못한다(그 테스트들은 체인을 로컬 프로세스로
  띄우지 SSH 로 띄우지 않는다). 바이너리는 이 기기에 있다:
  `~/work/github/wemade/{go-wemix,go-wbft}/build/bin/gwemix` 와 `go-stablenet/build/bin/gstable`.

  게이트는 `GSTABLE_BIN` · `WBFT_BIN` · `CHAINBENCH_E2E_FROM_BIN` ·
  `CHAINBENCH_E2E_TO_BIN` 넷이다. `CHAINBENCH_E2E_TEMPLATE` 은 더 필요 없다 — genesis
  템플릿은 chain-preset 이 소유한다(G4).

  | 무엇 | 결과 |
  |---|---|
  | `tests/e2e` (실제 체인을 띄우는 계층) | 통과, 854초 (캐시 끄고 잰 값) |
  | 핸드오버 e2e 21건 | **21 PASS · 0 FAIL** (G4 에서 살려냄) |
  | `internal/testengine` · `internal/app` · 그 외 60여 패키지 | 통과 |

  처음 돌렸을 때는 21건이 실패했고 **전부 발판이 없어진 테스트**였다. 그 21건은 G4 와 G7
  이고, 어느 것도 이 브랜치가 만든 것이 아니다. G4 는 닫혔고 G7 만 남았다.

  **주의: `go test` 캐시.** 두 번째 실행에서 35개 패키지가 `(cached)` 로 통과했고 그중
  `tests/e2e` 가 있었다. 캐시는 재실행이 아니다 — 이 계층을 다시 잴 때는 `-count=1` 을 준다.
- [x] **G2. PR.** #425. G1 보다 먼저 냈다(사용자 의도). 이 절이 그 PR 의 검증 기록이다.
- [ ] **G6. 15노드 poa 에서 endpoint 하나가 영구히 합류하지 못한다 — go-wemix 결함, 원인
  확정 (2026-09-23 관측 · 2026-09-24 재현·근본원인 측정).**
  `go-wemix/chain-up/02-wemix-chain-up-15` 만 BLOCKED 된다 —
  `1 node(s) still not ready after 5m0s`. 케이스 자신의 주장은 약하고(블록 2 이상, 피어 0
  아님), 실패하는 곳은 스텝이 아니라 그 앞의 readiness 게이트다. 스텝은 실행되지 않는다.

  **chainbench 의 문제가 아니다.** 15대를 같은 방식으로 띄웠고, 갈라지는 지점은 go-wemix 의
  블록 검증 안이다. 기동 순서로도 피할 수 없다 — 아래 "왜 순서로 못 피하나" 참조.

  ### 무엇이 일어나나

  재현된다. **2/2. 다만 걸리는 노드는 매번 다르다** — 2026-09-23 node13, 2026-09-24 node9.
  증적은 `~/.chainbench/20260923-100943-97772/` 와 `~/.chainbench/20260924-013034-76103/`
  (노드 15대 로그 전부 + health.json + processes.json).

  걸린 노드는 블록 N 에서 멈추고 블록 N+1 을 `unauthorized block` 으로 거절한다. 피어를
  끊고 다시 붙기를 반복하며 영원히 나아가지 못한다.

  ### 왜 한 노드만 — 데이터 문제가 아니다

  geth 는 배치의 헤더를 **블록을 넣기 전에 한꺼번에·동시에** 검증한다
  (`core/blockchain.go:1446` → `consensus/ethash/consensus.go:146`). 그래서 블록 N+1 의
  검증이 일어나는 순간의 로컬 head 가 노드마다 다르다.

  | 2026-09-24 실측 | 2차 배치 | 블록 6 검증 시 head | 블록 5 상태 | 판정 |
  |---|---|---|---|---|
  | node8·10·11·12·13·14 | 블록 4~67 (64개) | 3 | **없음** | 통과 |
  | **node9** | **블록 4~5 (2개)** | **5** | **있음** | **거절** |

  배치 크기는 다운로더가 그 순간 받아 둔 만큼이라 **경합**이다. 둘 다 큐에는 69개를 알고
  있었다(`blockTasks=69`).

  블록 5의 데이터는 멀쩡하다 — `health.json` 이 "all 15 node(s) report the same hash at
  block 5" 라고 적고, node9 의 registry 코드와 `magic()` 도 정상이다.

  ### 왜 거절하나 — 거버넌스가 블록 4~13에 걸쳐 지어진다

  chainbench 는 `gwemix wemix deploy-governance` 를 **한 번** 부른다. 나누는 것은 go-wemix
  안의 의존 사슬이다: proxy 는 imp 의 주소를 생성자 인자로 받고, registry 등록은 proxy
  주소를, 초기화는 registry 를 필요로 한다. `WaitMined()` 가 다섯 번 걸린다
  (`wemix/bind/structs.go` `DeployGovContracts`·`ExecuteInitialize`).

  | 블록 | 트랜잭션 | 무엇 |
  |---|---|---|
  | 4 | 생성 6 | registry + 구현(imp) 5 |
  | 6 | 생성 5 | proxy 5 |
  | 9 | 호출 9 | registry 도메인 등록 + 초기화 |
  | 12·13 | 호출 5 | 멤버·스테이킹 마무리 |

  블록 5에서 검사가 보는 것(node9 콘솔로 직접 재현):

      registry code  = 3250 hex chars
      magic()        = "Wemix Registry"                  ← registry 로 인식된다
      getContractAddress("GovernanceContract")
                     → execution reverted: address should be non-zero

  **주소록은 있고 자기소개도 하는데 내용이 비어 있다.**

  ### 결함 — 준비 안 된 상태의 표현이 셋인데 둘만 인정한다

  `wemix/admin.go:992 verifyBlockSig` 는 ① registry 에서 Gov 주소 찾기 → ② 멤버 수 →
  ③ enode 조회 순으로 간다. 블록 5에서는 **①에서 죽으므로 ②·③에 닿지 못한다.**

  | 노드가 보는 것 | 에러 | 판정 |
  |---|---|---|
  | 상태가 아예 없다 | `missing trie node` → `ethereum.NotFound` | 통과 (`admin.go:1000`) |
  | 멤버가 0명 | — | 통과 (`admin.go:1002`) |
  | **registry 는 있는데 주소록이 비었다** | **`revert: address should be non-zero`** | **거절** |

      return err == wemixminer.ErrNotInitialized || errors.Is(err, ethereum.NotFound)

  revert 는 둘 중 어느 것도 아니라 `false` → `consensus.ErrUnauthorized` 다.

  ### 왜 영구적인가

  관대한 경로는 **상태가 없을 때만** 열린다. 블록 5를 넣은 노드는 그 상태를 버릴 수 없고,
  geth 는 실패한 import 에서 head 를 되감지 않는다. 그래서 재시도 입력이 매번 같다 —
  **재기동해도 head 5, peers 0** (실측). 피어를 바꿔도 소용없다: 거절은 피어가 준 데이터가
  아니라 자기 상태 때문이고, 어느 피어가 줘도 블록 6은 같다. 앞으로 가는 유일한 문이 블록
  6인데 막혀 있으니 검사가 정상 동작하는 블록 9 이후에 영원히 닿지 못한다.

  ### 왜 기동 순서로 못 피하나

  `BootstrapPlan()` 은 이미 `deploy-governance → init-etcd → start-nodes` 다. 실측 타임라인:
  블록 9(거버넌스 완료) 01:31:20 → etcd 01:31:24 → 나머지 14대 01:31:26 이후, endpoint 는
  01:32:55(체인은 이미 블록 39). **모든 노드가 거버넌스 완료 뒤에 뜬다.** 그래도 실패하는
  것은, 늦게 뜬 노드도 **블록 1부터 재생**해야 하므로 어차피 그 구간을 지나기 때문이다.
  검증은 기동 시점이 아니라 **재생 시점**에 일어난다.

  ### full sync 만 취약하다

  | | full sync | snap sync |
  |---|---|---|
  | seal 검증 대상 | 모든 블록 (`blockchain.go:1444`) | 100개당 무작위 1개 (`headerchain.go:332`, `fsHeaderCheckFrequency=100`) |
  | 초기 구간 state 보유 | 직접 실행해 만든다 → 있다 | 만들지 않는다 (pivot 에서 받음) |
  | 취약 | **예** | 아니오 (두 이유가 독립적으로 막는다) |

  chainbench 기본값은 `full` 이다(`nodeconfig.go:165`), 그래서 15대 전원이 취약한 조건에
  있었다. `--syncmode snap` 은 증상 회피일 뿐이고, full sync 를 요구하는 검증 환경에는
  쓸 수 없다.

  ### 메인넷 영향

  | 상황 | 위험 |
  |---|---|
  | 운영 중 노드가 새 블록을 하나씩 받음 | **없음** — 항상 부모 상태가 있고 거버넌스는 완성돼 있다 |
  | 신규 구성 직후 노드 합류 | **높음** — 재현 2/2 |
  | **메인넷을 genesis 부터 full sync** | **가능** — 그 체인 초기에도 같은 미완성 구간이 있다면, 언제 띄우든 재생하며 지난다 |

  ### 체인팀에 넘길 것

  1. `wemix/admin.go:1000` 의 통과 조건에 **revert 를 포함**한다. "거버넌스를 아직 읽을 수
     없다" 는 상태가 없을 때든 주소록이 비었을 때든 같은 뜻인데 지금은 전자만 통과한다.
  2. `NewGovContracts`(`wemix/bind/structs.go`)가 `CallOpts` 에 높이를 넣지 않아 주소록을
     **검증 대상 높이가 아니라 latest 기준**으로 푼다. 이번에는 node9 의 latest 가 마침 5라
     결과가 같았지만 일반적으로 다른 답이 나올 수 있는 자리다.

- [x] **G4. `upgrade run` 을 부르던 e2e 20건을 리팩토링이 남긴 경로로 옮겼다 (2026-09-24).**
  `upgrade run` 은 main 의 `6ed4b37c` (#419, 2026-09-20)가 지웠다. 그 리팩토링은 핸드오버를
  전용 composer 에서 **일반 composition 경로**로 옮겼고, 명령을 몰던 e2e 20건은 따라 고치지
  않았다. 각 시도가 0.00초에 끝났으므로 체인이 한 번도 뜬 적이 없다.

  이 항목은 2026-09-22 에 "테스트 함수는 둘" 이라고 적었다. **틀렸다** — 바이너리를 걸고
  돌리니 **7개 파일 20개 함수**였다. 파일을 셌지 함수를 세지 않았고, 돌려 보지 않았다.
  `invocationDebt` 도 같은 정도로 적게 잡고 있었다.

  **어떻게 옮겼나.** 명령이 플래그로 받던 것을 chain-preset 이 선언한다.

  | 이전 | 지금 |
  |---|---|
  | `upgrade run --profile wemix-upgrade.yaml --from-binary --to-binary --template` | `chainbench run <생성 케이스> --keep-up` |
  | 프로파일 YAML | `presets/chain/wemix-to-wbft.json` (두 바이너리 · croissant · 블록 20) |
  | stdout 정규식으로 RPC·PID 파싱 | 워크스페이스 `chain-record.json` |
  | `--genesis-overlay` 플래그 | chain-preset 의 `genesis.overlay` 선언 |

  **진단이 원인을 엉뚱한 곳으로 돌리고 있었다.** 재시도 고리가 `cmd.Execute()` 의 에러를
  버려서, "unknown command" 가 `flaky producer/etcd bootstrap` 으로 15일간 보고됐다. 로그만
  보면 go-wemix 가 불안정한 것처럼 읽혔다. 이제 그 에러를 기록한다.

  **막혔던 것 넷과 그 해결.**
  1. 임시 디렉터리의 케이스가 `presets/chain` 을 못 찾는다 → 커밋된 preset 을 읽어 인라인.
  2. preset 의 `keys` 가 CWD 기준이라 풀리지 않는다 → `--keys` 로 절대 경로.
  3. 운영자 계정에 genesis 잔고가 없다(13건이 `insufficient funds`) → 옛 프로파일이 주던
     자금을 테스트가 선언한다.
  4. 마이그레이션 테스트가 node1 을 producer 로 가정 → 이 preset 은 node5 가 producer 다.
     어느 노드인지는 선언이 정하므로 **기록이 답하게** 했다.

  **마지막 5건은 리팩토링이 뒤집은 순서였다.** 상세는 아래 §G4-a.

  `invocationDebt` 는 비었다 — 래칫이 항목을 지우라고 요구했고, 그것이 그 장부의 쓰임이다.

- [x] **G4-a. genesis overlay 가 fork 섹션 안의 선언을 조용히 버리고 있었다 (2026-09-24).**
  G4 의 마지막 5건이 여기서 막혀 있었다. overlay 가 `config.croissant` 아래에 선언한 것은
  **전부 무시됐다** — 받아들여진 뒤 폐기됐고, 충돌을 알리는 것은 아무것도 없었다.

  | overlay 가 요구 | genesis 결과 |
  |---|---|
  | `stabilizingStakersThreshold: 2` | 5 (템플릿 기본값) |
  | `govNCP.params.ncps: 주소 하나` | 네 멤버 전부 |

  순서 때문이다. `genesis.Customize`(`internal/core/genesis/genesis.go:111`)가 base 에
  overlay 를 병합한 **뒤에**, `steps_genesis.go` 가 링에서 만든 fork 섹션으로
  `SetConfigSection` 을 부르고 그 함수는 `config.croissant` 를 **통째로 교체**한다
  (`internal/core/genesis/config.go:37`).

  **이것은 리팩토링이 뒤집은 순서다.** #419가 흡수한 핸드오버는 정확히 그 이유로 fork 를
  구성한 **뒤에** overlay 를 적용했다 — `upgrade.Handoff.Run` 이
  `ComposePlan` → `ApplyOverlay` 순서였다. 그 일은 `7f39c562` 의
  `internal/consensus/upgrade/handoff.go` 에 있었고 지금은 `internal/chainsetup/steps_genesis.go`
  가 한다. 옮겨 오면서 순서가 뒤집혔고, 그 파일에서 fork 섹션 뒤에 overlay 를 다시 병합해
  되돌렸다.

  **왜 테스트가 이것을 필요로 했나.** 옛 프로파일은 검증자 집합과 거버넌스 의회를 **분리해**
  선언했다 — `validators.addresses` 4명, `validators.members` 1명. chain-preset 에는 의회를
  선언할 자리가 없어서 키셋의 `systemContractMembers`(4명)가 강제되고, 제안 정족수가 3이
  된다. 단독 NCP 를 전제한 GOV-006/007/008/017 은 그 위에서 성립하지 않는다. overlay 가
  효력을 되찾자 테스트가 자기 의회를 선언할 수 있게 됐다.

  **남은 설계 질문(비차단).** chain-preset 이 의회를 선언할 수 있어야 하는가. 지금은
  overlay 로 우회하고 있고, 옛 프로파일에는 그 필드가 있었다.

  **찾은 방법을 적어 둔다.** 리팩토링 이전 트리를 worktree 로 받아(`7f39c562`) AST 로 파싱하고,
  `upgrade run` 의 실행 절차를 복원해 지금 경로와 대조했다. 그 대조 없이는 순서 역전이
  보이지 않았다 — 그 전까지는 이것을 새로 설계할 문제로 다루고 있었고, 정답(순서 원복)을
  가장 위험한 안으로 잘못 평가했다.

- [x] **G7. `TestRemoteDriver_E2E` 가 자기 노드 바이너리를 들고 간다 (2026-09-24 해소).**
  `internal/core/process/remote_e2e_test.go` 는 원격에 `/usr/local/bin/fakenode` 가 있다고
  가정했고, 자기 주석은 그것을 `tests/remote/sshd/run.sh` 가 띄운다고 적었다. **그 디렉터리는
  `704f9e1a` (#130, 레거시 bash 스위트 폐기, 2026-07-29)가 지웠다.** 그 뒤로 어디서 돌려도
  `exit 127: no such file` 이었고, SSH 변수가 없으면 건너뛰는 게이트 뒤라 아무도 실패를 보지
  못했다. 두 달이다. `invocationDebt` 가 이걸 못 잡는 것은 그 장부가 명령만 보고 fixture 는
  보지 않기 때문이다.

  **없어진 컨테이너가 준 것은 둘이었다** — sshd 와 가짜 노드 바이너리. sshd 는 `env/docker`
  가 이미 준다. 빠진 것은 바이너리 하나뿐이었다.

  그래서 **테스트가 직접 쓴다**(`standInNode`). `init` 은 0으로 끝나고 나머지는
  `sleep 3600` 인 네 줄 스크립트면 driver 가 요구하는 것 — init 성공, Stop 이 죽일 수 있는
  살아 있는 프로세스 — 을 둘 다 만족한다. 이미지도 root 도 필요 없고, 로그인할 수 있는
  호스트면 어디서든 돈다. 진짜 체인 바이너리를 쓰면 빌드가 전제로 하나 더 붙을 뿐이다.

  `env/docker` 의 server1(sshd 2201)로 검증했다: **PASS 2.3초.** 실행법은 파일 머리말에.
- [x] **G3. `preset.KeysDir` 은 저장소 루트 기준이 아니다 — 주석이 틀렸다 (2026-09-24 해소).**
  `internal/preset/doc.go` 는 "relative to the repository root" 라고 적었다. 그렇지 않다.
  호출부 아홉은 **전부 플래그·설정의 기본값**이고, 상대 경로는 프로세스 작업 디렉터리
  기준으로 풀린다. 그리고 chainbench 는 `install.sh` 로 설치되는 바이너리라 **저장소 루트가
  없는 환경이 정상**이다 — "루트를 찾아 풀자"는 선택지는 애초에 성립하지 않는다.

  이 항목은 "다른 데서 실행하면 **조용히** 없는 디렉터리를 가리킨다" 고도 적었다. 그것도
  틀렸다. `/tmp` 에서 돌려 보면 조용하지 않다:
  `keyring: read keys: open presets/keys/metadata.json: no such file or directory`.
  경로는 말한다. 말하지 않던 것은 **그 경로가 무엇을 기준으로 풀렸는지**다.

  그래서 둘을 고쳤다. 주석을 사실에 맞췄고, 상대 경로일 때만 실패에 기준을 덧붙인다 —
  `("presets/keys" is relative, resolved against the working directory /tmp)`.
  절대 경로면 아무것도 붙이지 않는다.

- [x] **G5. 죽은 `.env.json` 접미사 필터 넷 — 제거 (2026-09-24).** 네 곳 모두 `tests/tc` 를
  훑는데, 그 트리에는 `kind: case` 파일 209개뿐이고 `*.env.json` 은 **한 개도 없다**.
  거르는 것이 0건이었다. 라이브 테스트가 쓰던 이름이 `<id>.json` 으로 바뀐 뒤 필터만
  남은 것이고, 해롭지는 않아도 다음 사람에게 있지도 않은 규칙을 가르친다.

  `cmd/chainbench/validate_test.go` 에서는 그 필터가 함수 하나의 전부였다(`specFilesUnder`).
  `jsonFilesUnder` 와 같아졌으므로 접고 호출부 둘을 옮겼다. 남은 함수의 주석이 **걸러낼 것이
  없다는 사실과 그 이름이 무엇이었는지**를 적는다. 나머지 셋은 제외 절만 뺐다
  (`internal/testengine/corpus_gate_test.go` 1곳, `internal/testhelper/corpus_address_test.go` 2곳).


## 2. 전체 작업 리스트 (Phase · Task)

### Phase 0 — 레이아웃 정리 + 인터페이스 동결
- ☑ **T0.0 `pkg/` → `internal/` 마이그레이션** 기계적 rename: 디렉토리 이동 + import 경로 `github.com/0xmhha/chainbench/pkg/…` → `…/internal/…`(cmd/ 포함 전 소스). **게이트**: `go build ./...` + `go test ./...` 통과, 동작 변화 0. (외부 importer 없음 → 파괴 없음.) 공개 API가 필요하면 그 slice만 `pkg/`에 남김.
- ☑ **T0.1** Transport·Allocator(+Capacity)·KeyRegistry(+BLSDeriver)·GenesisBuilder·Provisioner·Supervisor·Collector·Session·Interpreter·Capabilities를 **컴파일되는 Go stub**으로(design §3과 1:1, `internal/` 하위). enum은 typed const(FailureMode/TestStatus/Mode/Source). **게이트**: 컴파일 + 각 인터페이스↔F(AC) 매핑 표.

### Phase 1 — Low atomic (TDD 먼저, 병렬 가능)
- ☑ **T1.1 testspec** Parse+필수검증(schemaVersion 등)·**Fingerprint(canonical·키정렬→결정성)**·`Get(dotPath)`·`,`파서. 순수. — F3·F7
- ☑ **T1.2 assert funcs** 타입인지 비교(Equal/Len/EqualHashAt/EqualCI/InDelta …). 순수. — F6
- ☑ **T1.3 session-path** 결정적 경로·env-id(`env-`+12hex)·레이아웃. — F1
- ☑ **T1.4 place** portplan(로컬 스텝/OS)+원격 동일포트 **통합 Allocator** + **용량검증(min≥4·max=서버×포트)**. — F12
- ☑ **T1.5 procman EXTEND** `{PID,datadir}`·원격PID·`Alive`. (stop 경로 배선은 T3.2) + **`StopOne`**(단일 노드 정지 — fault 스텝용, `StopAll` 은 네트워크 전체를 내려 쿼럼 테스트에 못 씀) + **loopback=local 판정 수정**(라이브 검증에서 발견: 로컬 런처가 `Host:"127.0.0.1"` 을 기록해 `IsRemote()` 가 참이 되어 시그널 대상에서 제외되고 있었음 — teardown 이 동작한 건 `Teardown` 이 빈 Host 로 **중복 track** 하던 우연 덕분) — F13
- ☑ **T1.6 keyreg** 랜덤/기존/원격다운로드 통합 + **BLSDeriver boundary**(외부 bootnode 캡슐화, 부재 시 오류). — F2
- **게이트**: 단위 100% + 동시 모듈 `-race`.

### Phase 2 — Transport
- ☑ **T2.1** Local/Remote Transport를 driver 위에 형식화 + **종료검증(`kill -0`)** + **key_file 인증**. **[D안]** ☑ **key_file 인증**: `remote.Credentials` 에 `PrivateKey`/`Passphrase` 추가, `authMethods`(key 우선+password, 최소 1개 필수, 키자료 미노출), `LoadPrivateKey`(0600 강제·insecure perm 거부). deploy `credentials.go` 가 `key_file`(+ `CHAINBENCH_REMOTE_KEY_FILE`/`_PASSPHRASE` env) 를 `remote.Credentials.PrivateKey` 로 로드 — "future phase 예약" 게이트 제거. 단위검증(authMethods 4케이스·키 미노출·perm 거부·For key_file). ☑ **종료검증(`kill -0`)**: 이미 `procman.Alive`(signal 0)+`StopAll`(SIGTERM→wait→SIGKILL→poll→leak 보고)로 구현, 엔진 teardown 이 이를 경유(라이브 고아0 확인). **남은 것: 없음 — 철회 (2026-09-11).** "driver 위 Transport 타입 형식화" 는 **이미 해봤고
되돌렸다.** `internal/arch/reach_test.go` 의 기록: 도달 불가 심볼 정리에서 삭제된 셋 중 하나가
"a Transport interface that described what Driver already is and had no implementer" 였다. 즉
`Driver` + capability 인터페이스들(`Initializer`·`Commander`·`ProcessInspector`·
`FileProvisioner`·`CmdlineInspector`·`PortProber`)이 그 역할을 이미 하고 있고, 그 위에 타입을
하나 더 두면 구현체 없는 껍데기가 된다. 두 기록이 이어지지 않아 여기에 미완으로 남아 있었을
뿐이다. **다시 쓰지 않는다.**
- ☑ **T2.2 upload-if-absent**(로컬·FileSink; 원격 SSH sink는 remote 슬라이스) `test -f` 존재확인 → 재사용/업로드(현재 항상-업로드 or 항상-읽기).
- **게이트**: 단위(모의) + 통합 1건(로컬 더미 프로세스 검증-종료·고아0).

### Phase 3 — Middle 통합(라이브)
- ☑ **T3.1 Provisioner** datadir+키+genesis+config 물질화(local·remote 동일경로)+upload-if-absent.
- ☑ **T3.2 Supervisor** 오케스트레이션·teardown·procman배선 단위완료 + **실4노드 라이브 헬스게이트(블록전진 gate)로 Phase4 BuildEnv e2e 에서 검증**(고아0). etcd 리더 게이트는 wemix 계열(gwemix/etcd 필요) 후속.
- ☑ **T3.3 Collector** ☑ RPC 스냅샷(height·peers)·WaitLog poll·**로컬 live tail**(offset 증분·스캔→tail·부분줄 미방출·`Deps.OnLine` obs 미러 boundary)·**bp참여 집계**(head producer 샘플→`BPParticipation`, `Deps.BPWindow` 로 bounded prune)·**fork/reorg 검출**(높이별 first-seen hash, 노드 간/샘플 간 불일치→`Forked`). 단위+`-race` 검증. ☑ **엔진 배선**(`withCollection` 이 local/attach 의 BuildEnv 를 래핑 — `Bus` 설정 시 env 별 collector 실행, RPC probe(`rpc.Client.HeadBlock`)로 샘플, chainstate 스냅샷·tail 로그를 obs 로 미러, teardown 시 정지; attach+mock RPC 로 chainstate 이벤트 e2e 검증)·**chainstate 세션 영속화**(collection 이 스냅샷을 `chainstate/chainstate.jsonl` 로 기록 — F10/F15 jsonl+obs 미러, 완료 세션 재생용; best-effort). ☑ **원격 SSH tail**: tail 루프가 **`collector.LogReader` boundary**(`ReadFrom(ctx,path,offset)`)을 거치도록 리팩터 → 로컬은 `LocalLogReader`(파일), 원격은 `driver.RemoteLogReader`(SSH `tail -c +N`, **1-based 바이트 오프셋** — `tail -n` 은 줄 단위라 collector 가 추적하는 정확한 바이트 위치를 잃어 줄 중복/분할이 남). 파일 부재는 오류 아님(첫 줄 쓰기 전 tail 시작 허용), 전송 실패만 오류. 단위검증 5건. **실 SSH 호스트 라이브 e2e — 완료 (2026-09-11)**: `internal/core/process/remote_log_live_test.go`
(`Live_RemoteLog*`, `CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build`). 가짜 runner 로는 명령의
**철자**만 고정할 수 있고 그 명령이 **무엇을 하는지**는 보이지 않는다 — 그리고 오프셋 산술이
바로 증분 tail 이 바이트를 중복시키거나 잃는 자리다. 실 sshd 에 대고 네 가지를 확인했다:
offset 0 이 파일 전체, 직전 읽기가 끝난 오프셋에서 읽으면 **덧붙은 부분만**, 두 읽기를 이으면
파일과 정확히 같음, EOF 오프셋은 빈 결과. 줄 중간 오프셋(바이트 단위이지 줄 단위가 아님)과
파일 부재(오류 아님), `collector.LogReader` 로 어댑터 없이 쓰이는 것도 같이 고정했다.
**변이로 실패 확인**: `tail -c +N` 을 0-based 로 착각하면(`offset+1` → `offset`) 읽기 경계마다
`\n` 이 하나 중복돼 실패한다 — 주석이 경고하던 바로 그 결함이다. attach=RPC-only(로컬 로그 없음→tail no-op).
- ☑ **T3.4 Session 저장·재사용** `session.json`·env fingerprint 재사용(엔진 오케스트레이션이 fingerprint 로 env 재사용)·records(spec/steps/assert/status). 엔진 단위·라이브 e2e·attach e2e 로 검증(summary.pass 판정).
- ☑ **T3.2b supervisor 선언 논항 방출** (x-bar 정렬 검토 2026-08-09) 선언만 되어 있던 `Options` 3개를 실제로 읽는다.
  - **`LeaderGate`**: `Deps.LeaderGate(ctx, ns, window)` boundary 신설 — **HealthGate 보다 먼저** 실행(클러스터에 리더가 없으면 노드는 healthy 일 수 없다). "리더 준비"의 판정은 체인특화(go-wemix 내장 etcd)라 주입식이고, **언제 돌릴지·얼마나 기다릴지·실패를 어떻게 분류할지는 supervisor 가 소유**. **요청했는데 미배선이면 조용히 통과가 아니라 오류**(F13 AC-1).
  - **`AlignJoinGap`**: `JoinGap(N)`(C-etcd 표: ≤11→7s·≤23→11s·≤41→17s·else 23s)·`JoinWindow(N)=(N+1)*gap` 신설 — 리더 게이트 데드라인을 **클러스터 크기에서 파생**. 고정 타임아웃은 *아직 자기 조인 슬롯이 오지 않은* 노드를 조인 실패로 오판한다(L7).
  - **`ForkSwaps`**: `Deps.SwapBinary` boundary 으로 type-2 스왑 수행. **선언했는데 미배선이면 오류** — 조용히 건너뛰면 체인이 잘못된 바이너리로 포크를 넘어 훨씬 덜 명확한 곳에서 실패한다(F9 AC-3).
  - **`FailureMode` 분류**: `Classify(err)` 신설(etcd stale/join·quorum·fork·rpc 시그니처) — **launch 실패도 분류**(이전엔 전부 `RPCUnready`), HealthGate 가 Mode 를 안 채우면 에러 텍스트에서 파생, **Gate 가 스스로 분류했으면 보존**. 매칭 없으면 `UnknownFailure` — 그럴듯한 오분류는 읽는 사람을 엉뚱한 로그로 보내므로 정직한 "모름"이 낫다(F13 AC-2).
  - **enum 0 값 교정**: `EtcdJoinFailed` 가 iota 0 이라 **빈 `Diagnosis` 가 "EtcdJoinFailed" 로 읽히던** 함정 제거 → `UnknownFailure` 를 0 으로.
  - 재시도 시 `RemoveDataDir:true` 로 stale etcd 상태 정리(F13 AC-3)는 기존 동작 유지·주석 명시. 단위검증 12건.
  - **잔여**: 실제 etcd 리더 게이트 구현체(wemix 계열)와 `SwapBinary` 구현체 배선은 **T5.2**(gwemix 라이브 필요).
- **게이트**: 각 컴포넌트 통합테스트 라이브.

### Phase 4 — Walking Skeleton ★
- ☑ **T4.1 Engine** DI 오케스트레이션(Parse→Applicable skip→fingerprint 기반 env 재사용/BuildEnv→RunSpec→session 저장→종료 시 Teardown) 구현·단위검증 완료. 실제 Place→KeyReg→Genesis→Provision→Supervise→Collect 배선의 **1체인 wbft·local·4노드·tx1** live e2e 는 사용자 환경(체인 바이너리 필요)으로 이월.
- ☑ **T4.2 빌트인 tx/rpc + 어휘 확장(#225)** `NewRegistry(true)` 시드 — **액션 11**: `sendTx`(fee-cap/nonce 인자 포함)·`waitBlock`·`read`(임의 RPC-read 소스 저장)·`deployContract`·`registerContract`·`faucet`·`stopNode`·`startNode`·`restartNode`·`partition`·`healPartition`. **어세션 16**: `chainId`·`blockNumber`·`peerCount`·`balanceAt`·`codeAt`·`nonceAt`·`call`·`txStatus`(F11)·`blockAdvance`·`sameBlockHash`·`baseFee`·`estimateGas`·`gasPrice`·`rpcCall`(체인 어휘를 spec 문자열로, core 무지 유지)·`logs`(eth_getLogs select)·`wsSubscribe`. **스텝 값 바인딩**(`save`/`$ref`/`${..}`). mock RPC 단위검증 + 라이브 `TestEngine_Live_NewVocabulary`(실 gstable 4노드). → 컨트랙트 배포·이벤트·fault·WS·교차-call 전부 표현 가능.
- ☑ **T4.2b 노드 생명주기·fault 액션** `stopNode`/`startNode`/`restartNode` 는 **`testspec.NodeControl` boundary**(`Deps.Nodes`) 경유 — 로컬 엔진은 `engine.NodeController`(런처 앞단에서 노드별 arming·PID 기억, supervisor 와 **procman 공유**해 중도 재기동 노드도 teardown 에 포함) 를 주입하고, **attach 는 nil** 이라 액션이 "이 실행은 노드 프로세스를 소유하지 않는다"고 명확히 실패한다. `procman.StopOne`(SIGTERM→grace→SIGKILL→검증) 신규 — `StopAll` 은 네트워크 전체를 내려 쿼럼 테스트에 못 씀. `partition`/`healPartition` 은 `admin_removePeer`/`admin_addPeer`(+`admin_nodeInfo` 로 enode 획득) 로 **그룹 경계를 넘는 링크를 양방향 절단**. 예제 `fault-node-restart.json`·`fault-partition-fork.json`(`requires:["process"]` → attach 에서 skip). 단위검증(선택·미배선 오류·8회 removePeer·풀메시 heal·재기동 arming 재사용).
- ☑ **T4.2c 자산·컨트랙트 액션** `faucet`(수신자에 wei 공급 — 런타임 생성 서명키는 genesis alloc 에 없어 첫 tx 가스도 못 냄; `from` 미지정 시 **대상 노드의 unlocked coinbase** 를 자금원으로 사용)·`deployContract`(`to` 없는 tx → 영수증의 `contractAddress` 를 `ac.Value` 로 바인딩 — **주소가 영수증에만 있으므로 `sendTx` 로는 표현 불가**, 이것이 별도 액션인 이유)·`registerContract`(배포 주소로의 등록 호출 — `to` 필수·revert 는 항상 실패인 **의도-명시형 sendTx**). 예제 `contract-deploy-and-register.json`(faucet→deploy→`$contract` 로 register→codeAt/txStatus/balanceAt 검증). 단위검증(coinbase 자금원·value hex·필수인자·주소 없는 영수증 오류·배포엔 `to` 미포함).
- ☑ **T4.3 RunSpec 배선** `engine.NewRunSpec(testspec.Deps)` — 인터프리터를 `Deps.RunSpec` 에 바인딩하는 조립 boundary. spec→interpreter→빌트인 어세션→RPC→기록 상태 **실행 수직**을 mock RPC 로 통합검증(pass/fail). BuildEnv 배선(Place→KeyReg→Genesis→Provision→Supervise; 레거시 `pipeline/setup.Plan` 매핑 필요)은 후속.
- ☑ **T4.3b tx 스텝 시맨틱(F11)** sendTx 가 영수증 status 를 검사 — 기본 revert(0x0)=스텝 실패, `expectRevert:true`(또는 `expect:"revert"`)=원자적 negative(성공하면 스텝 실패). 스텝 provenance(tx hash·receipt)를 `ActionCtx` 출력→`StepResult.Hash/Receipt` 로 기록(미사용 필드 활용). mock RPC 단위검증(revert 실패·expectRevert 양방향·provenance 기록·미지 스텝 기록후 실패). → wemix4 negative(GOV 권한거부 등) 이관 기반.
- ☑ **T4.4 BuildEnv 배선 [A안]** `engine.AssemblePlan(plugin, []PlacedNode, genesis, dataRoot, caps)` — place 할당 포트로 `setup.Plan` 을 조립하는 순수함수(레거시 `setup.BuildPlan` 의 `node.Offset` 고정포트 경로 대체). LocalOSAssigned·remote 모드가 이 함수 변경 없이 합성됨. 실 wbft 플러그인으로 단위검증(할당포트 반영·바이너리 폴백·datadir 기본값). **후속 슬라이스**: (T4.4b) keyreg→genesis 피딩(검증자 주소·ExtraData RLP), (T4.4c) provision+supervisor.BringUp 오케스트레이션(launch boundary 주입).
- ☑ **T4.4b GenesisSource boundary** `engine.GenesisSource`(plugin·검증자수→genesis 바이트) + `PresetGenesisSource`(preset `metadata.json` 기반). **핵심 발견**: wbft 계열 ExtraData(RLP 검증자셋)는 코드가 계산하지 않고 preset 에 baked — 패밀리는 템플릿 치환만 함. 따라서 랜덤 keyreg 키만으로는 유효 genesis 불가 → preset 이 검증자셋·ExtraData 소스, keyreg 는 노드 신원/계정 키. 실 wbft 패밀리+최소 템플릿으로 단위검증(치환·Take 검증자 제한·preset 부재 에러). BuildEnv 조립(T4.4c)이 이 boundary 을 호출.
- ☑ **T4.4c BuildEnv 오케스트레이션** `engine.NewBuildEnv(BuildDeps)` — allocate(place)→genesis(GenesisSource)→AssemblePlan→provision(주입 boundary)→`supervisor.BringUp`(launch/health boundary 주입)→NodeSet+Teardown 반환. `Deps.BuildEnv` production wiring 완성. fake allocator/genesis/provision + 실 supervisor(fake launch/health)로 파이프라인 스레딩 단위검증(4노드 조립·genesis 검증자수·plan 포트·teardown·에러전파). **남은 것**: (1) 실 `Provision` 파일생성(config.toml·keystore·nodekey; 레거시 `setup.provision` 참조), (2) 실 4노드 라이브 e2e(바이너리 필요, 사용자 환경). → **코드 수준 walking skeleton 조립 완료**(BuildEnv+RunSpec 모두 `Deps` 배선).
- ☑ **T4.4d GenesisSource 라이브검증** `TestPresetGenesisSource_Live_GstableInit`(GSTABLE_BIN 게이트) — `PresetGenesisSource`+실 stablenet 플러그인으로 `presets/keys` 에서 genesis 생성 → **실 gstable v1.1.0 `init` 통과**("Successfully wrote genesis state", `<datadir>/gstable/chaindata` 생성). 이월했던 "랜덤키만으로 유효 genesis 불가·preset 필요" 가정을 실 바이너리로 확정. CI 는 바이너리 부재로 SKIP(green 유지). **실 4노드 라이브 기동(Provision+driver Launch+HealthGate)은 T4.4e 후속**.
- ☑ **T4.4e RunSpec 라이브 e2e (walking skeleton 증명)** `TestRunSpec_Live_Stablenet`(GSTABLE_BIN 게이트) — 실 gstable 로 4노드 stablenet 기동(`setup.Launch` fixture) → `engine.NewRunSpec`(인터프리터+빌트인) 으로 spec 실행: **sendTx(노드서명+영수증)·chainId==8283·blockNumber≥1 어세션 전부 실 RPC 대상 → status pass** → teardown 고아0. 로컬 39.7s 통과. **발견**: (1) geth IPC 유닉스소켓 경로 <104자 제약 → dataRoot 는 `/tmp/cblXXX`(t.TempDir 불가), (2) wbft 블록생성 웜업 ~35s → 헬스게이트 넉넉히 폴링. → **DSL 이 실 체인에서 실행·검증됨(실행 수직 라이브 완성)**. 남은 것: BuildEnv 자체의 실 launcher(place 포트 기동)는 T4.4f 선택.
- ☑ **T4.4f BuildEnv 실 launcher (기동 수직 라이브)** `engine.LocalLauncher`(preset 기반 arming: config 렌더·신원설치·datadir init·기동) + `armSpecs`(순수, 단위검증: validator --unlock/--nodekey, static-node enode 가 plan p2p 포트 사용, endpoint 는 unlock 없음). `TestBuildEnv_Live_Stablenet`(GSTABLE_BIN 게이트): 실 allocator+PresetGenesisSource+LocalLauncher+블록전진 헬스게이트로 `NewBuildEnv` → **실 4노드 stablenet 을 allocator 할당 포트(node1 :8600)로 기동·헬스통과·teardown 고아0**. 로컬 통과. → **Engine 기동 수직(BuildEnv)도 실 체인 라이브 증명. RunSpec(#195)+BuildEnv 로 Engine.Run 전 구간 라이브 커버.** (짧은 session root 로 IPC 소켓 경로 <104자 유지.)
- ☑ **T4.5 Engine 최상위 배선 (capstone)** `engine.NewLocalEngine(LocalConfig)` — allocator·PresetGenesisSource·LocalLauncher·`NewBlockAdvanceGate`·인터프리터·session 을 하나의 `engine.Deps` 로 조립하는 실행 가능한 진입점(CLI/MCP 가 호출). `NewBlockAdvanceGate`(head≥target 폴링, 로컬 non-etcd 라이브니스), `applicableTo`/`validatorReqs` 헬퍼. 단위검증(applicable 매칭·구성검증·미지체인 에러). **`TestEngine_Live_FullRun`(GSTABLE_BIN 게이트) capstone**: `NewLocalEngine`→`Engine.Run([spec])` 한 번으로 **실 gstable 4노드 기동→spec 실행(chainId/blockNumber)→teardown→session.json 저장, summary.pass=1** 검증. 로컬 4.5s 통과·고아0. → **walking skeleton 을 실행 가능한 단일 진입점으로 종료(Engine.Run 전체가 실 체인에서 동작).**
- ☑ **T4.6 launcher 파일 물질화를 provision.Provisioner 경유 [B안]** `LocalLauncher` 가 genesis·per-node config 를 `provision.Provisioner`(`FileSink`) 로 물질화 — 기존 ad-hoc `os.WriteFile(genesis)`+`driver.Provision(config)` 제거. **upload-if-absent**(기존 파일 재사용) + **원격 `RemoteFileSink` 로 교체할 boundary**(슬라이스 C 대비) 확보. 순수 `materialize` 헬퍼로 분리해 recording FileSink 로 단위검증(genesis+config 기록·재사용 스킵). 리팩터 후 실 gstable 라이브 2종(BuildEnv/FullRun) 재통과·고아0. 프로덕션 engine 은 이제 레거시 `setup.Provision/Launch` 미호출(`setup.Plan` 타입만 사용).

### Phase 5 — 수직 슬라이스 (매번 통합 유지)
- ☑ **T5.1 remote [C안]** ☑ `driver.RemoteFileSink`(`provision.FileSink` 구현: SSH `test -f` 존재확인 + `ProvisionFile` base64 전송) — B의 `LocalLauncher.Sink` boundary 에 그대로 주입 가능(upload-if-absent). ☑ launcher init 을 `driver.Initializer` capability 경유로 라우팅(local/remote 드라이버 공통) → `LocalLauncher{Driver:RemoteDriver, Sink:RemoteFileSink}` 로 **원격 기동 가능**. 단위검증: RemoteFileSink(exist/absent/transport err·base64 write·`provision.FileSink` 만족), launcher 전체 합성(materialize→init(Initializer)→launch, fake driver/sink). 로컬 live 2종 재통과(init 라우팅 변경 후·고아0). **남은 것**: 없음 — 원격 로그 tail 의 실 SSH 라이브 e2e 는 T3.3 항목에 적은 대로 완료(2026-09-11).
원격 기동 자체는 핸드오프·`chain rm`·metric 이 이미 docker 에서 라이브로 돌았다. · ☑ **T5.2 업그레이드 멀티바이너리**(wemix+wbft) — `internal/consensus/upgrade` 가 두 바이너리를 동시에 기동한다. 라이브(그때 `tests/tc/go-wemix/handoff/01-*`, 지금 `hardfork/01-croissant-successors-take-over.json`): 골든 1+4 와 **docker 15+15** 모두 `handoff confirmed`(2026-09-11). DSL 쪽도 `env.binaries.{producer,validator}` 로 표현된다(`tests/tc/go-wemix/handoff/01-*.json`) · ☑ **T5.3 attach** `engine.NewAttachEngine(AttachConfig{Chain,RPCURLs,ArtifactRoot})` + `NewAttachBuildEnv`(attach.Build 로 RPC 엔드포인트에서 NodeSet 구성, **기동/teardown 없음** — attach 는 노드를 만들지 않음). 바이너리·preset 불필요 → **mock RPC 로 Engine.Run 전체 e2e 가 CI 에서 실행**(chainId/blockNumber 어세션 pass·미적용 spec skip·구성검증). walking skeleton 실행수직을 바이너리 없이 CI 커버하는 첫 통합. · ☑ **T5.4 stablenet**(ACL 플러그인·Core 무변경) — 거버넌스 read 시나리오를 DSL 로 표현·엔진 실행 CI 검증(예제 spec + govbind calldata mock RPC e2e). · ☐ **T5.5 wemix4 이관**(DSL).

### Phase 6 — 표면·마감
- ☑ **T6.7 spec 오프라인 검증 + 예제** `chainbench validate [spec…]` — 실행 없이 (1) 파싱 검증(OK/INVALID), (2) **이름 해결**(`testspec.Unresolved`: 스텝 액션·어세션 이름을 빌트인 registry 와 대조 → 미등록 시 `UNRESOLVED: action:…/assert:…`, 오타를 런타임 전 포착), 무효/미해결 시 exit 1. `--chain` 은 manifest capability·applicableChains 대조로 실행/스킵(OK·SKIP(chain not applicable)·SKIP(needs caps))을 정보 표시(파싱/해결 오류만 실패). `examples/specs/*.json`(RPC-read·tx/waitBlock 스텝·expectRevert negative·call/txStatus) + CI 가드 테스트(`validate --chain stablenet` 전부 OK)로 DSL 문서-파서 드리프트 방지. → 이관 작성자 빠른 피드백.
- ☑ **T6.6 F16 세션 운영·보안 규약** (O2·O3·O4·O5 전부 완료 — 미완 표시가 하나도 없는데 ◐ 로 남아 있었다) ☑ **schemaVersion 거부(O2)**: `testspec.validate` 가 미지원 버전 명시 거부(현재 지원 `"1"`). ☑ **CI exit code(O5)**: `chainbench run` 이 세션 판정을 exit code 로 매핑 — 전건 pass=0·fail=1·blocked/인프라=2(`exitError`+`main` 배선). ☑ **세션 GC(O4)**: `chainbench clean --artifact-root --older-than <Nd|Nw|dur>/--keep-last N` — 완료 세션(session.json 보유)만 대상 → 실행중 세션 보존, `session.SessionDir`/`List` 재사용. ☑ **키파일 0600(O3)**: keyreg 가 이미 0600/0700 생성(기존). 단위검증(버전 거부·exit 0/1/2·older-than/keep-last/정책필수·실행중 보존).
- ☑ **T6.5 문서 경로 드리프트 정리** T0.0(`pkg/`→`internal/`) 완료 후에도 문서가 삭제된 `pkg/` 경로를 참조 → 후속 작업 착수 시 혼선. 실제 대상은 모두 `internal/` 하위에 존재(경로만 stale). **최초 스코프는 2문서 6건이었으나 x-bar 정렬 검토(2026-08-09)에서 실제 표면이 7문서 20건임이 드러나 확대**: `repro-migration-remaining.md`(3) · `wemix4-migration-plan.md`(5) · `wemix4-port-tracker.md`(4: `pkg/core/pipeline/testrun`·`pkg/core/procman`×2·`pkg/core/topology`) · **`topology.md`(1: `pkg/core/topology` — 사용자 대상 참조 문서라 최우선)** · `chainbench-requirements-review.md`(2: `pkg/mcp`·`pkg/consensus/poa`) · `chainbench-refactoring.md`(3: `pkg/mcp`·`pkg/dashboard`·`pkg/testkit`) · `chainbench-component-architecture.md`(2: `pkg/core/place`) 전부 `internal/…` 로 정정. `worklist`/`audit` 의 `pkg/` 언급은 마이그레이션 자체를 서술하므로 유지. **경로가 아니라 판정이 낡은 건은 T6.5b**(§2b 실측 매트릭스).
- ☑ **T6.5b component-architecture §2b 실측 매트릭스 갱신** §2b 판정이 낡았던 것(`place` 없음·`procman` 미배선·용량검증·upload-if-absent·로컬 tail·bp참여/reorg 없음)을 **현재(#225) 열로 전부 해소 표기**(#225). 후속 정합(이 검토): §3 C-테이블에 남아있던 stale 델타(C1 "어휘 △"·C3 "게이트 △"·C4 "원격 tail ✗")를 #225 반영으로 갱신 + §2b 컬럼 헤더 `#224`→`#225`.
- ☑ **T6.1 Capabilities** spec `requires: [cap...]` 필드 + 엔진 capability 게이팅: `satisfies`/`applicableWithCaps`(체인 매칭 ∧ 필요 capability ⊆ 타깃 제공). NewLocalEngine 은 `localCapabilities`(manifest.Capabilities + "ws"), NewAttachEngine 은 `["rpc"]` 를 제공집합으로. 미충족 spec 은 skip(fail 아님). 단위(satisfies 4케이스·applicableWithCaps) + attach mock RPC e2e(ws 요구 spec → rpc-only attach 에서 skip). `requires` 는 env 를 바꾸지 않으므로 fingerprint 미포함. · ☑ **T6.2 MCP 결과연동**(F14) `chainbench_run` MCP 도구 — attach 모드로 DSL spec 실행 후 session 판정 반환(CLI `run` 의 MCP 대응). `engine.ReadSessionSummary`(session.json 단일 리더)를 CLI·MCP 가 공유(중복 제거). attach+mock RPC 로 CI 테스트(pass=1·인자 검증). · ☑ **T6.3 dashboard**(F15) — 엔진 오케스트레이션이 obs 이벤트 emit(`Deps.Emit`/`Network`, nil-safe no-op): run started·building environment·environment reused·running spec·spec `<status>`·run complete. `NewLocalEngine`/`NewAttachEngine` 가 `Bus` 옵션으로 배선, `chainbench run --dashboard <url>` 가 `dashboard.Forward` 로 chainbench-dashboard `/api/events` 에 스트리밍(종료 시 flush). 엔진 emit 단위 + attach mock RPC → Forward → dashboard.Server end-to-end CI 테스트. obs.Bus 백프레셔(bounded buffer·drop-on-full)로 관측이 실행을 막지 않음. ☑ **완료 세션 디스크 조회**(F15 AC3): `session.List`/`SessionFilePath`/`ChainstatePaths`(레이아웃 소유자) + dashboard `/api/sessions`·`/api/sessions/{id}`(session.json 판정)·`/api/sessions/{id}/chainstate`(chainstate.jsonl → JSON 배열), id 는 단일 세그먼트 검증(traversal 방지), `chainbench-dashboard -artifact-root` 로 활성화(httptest 검증). · ◐ **T6.4** 백프레셔(O7)·`-race` 게이트(O6)·CLI 정리. ☑ **`-race` 게이트**: CI test 잡을 `go test -race ./...` 로(경합 CI 상시 검출; 풀런 ~71s 로 기존과 동등 — 느린 `verify` 테스트가 CPU 아닌 대기 바운드). ☑ **백프레셔(O7)** obs.Bus 는 bounded buffer(256)·drop-on-full(non-blocking select)·`Dropped` 카운터로 이미 구현 — 회귀 테스트 추가(느린 구독자에서 blocking 없이 drop·정확한 카운트 검증). · ☑ **CLI 엔진 배선**: `chainbench run [spec.json…]` 명령 — `--rpc`(attach: `NewAttachEngine`) 또는 `--binary`(local: `NewLocalEngine`) 로 DSL spec 을 엔진에 실행, `session.json` 요약 출력·실패 시 non-zero exit. attach+mock RPC 로 CI 테스트(pass=1·실패 non-zero·모드 검증). → 재설계 엔진이 CLI 에서 도달 가능. ☑ **기계 판독 출력**: `run --json`(세션 판정: session 경로+tests+pass/fail/blocked/skip)·`validate --json`(per-spec spec/id/ok/result 배열) — exit code(F16-O5)와 함께 무인 CI 게이팅 완성. 단위검증(JSON 파싱·필드).

---

## 3. 폴더 트리 예상도 (구현 후 · `internal/` 레이아웃)

> **레이아웃 결정**: chainbench는 애플리케이션이고 `pkg/`가 자기 `cmd/`에만 쓰임(외부 importer 0) → **`internal/`이 정석**(컴파일러 강제 캡슐화). `pkg/`는 관례일 뿐이라 폐기. 진짜 공개 Go API가 생기면 그 slice만 `pkg/`로 승격.
> `[NEW]` 신규 · `[EXT]` 확장 · `[KEEP]` 유지·재사용 · `[REF]` 재구성 · `[REPL]` 교체 · `[→x]` x로 흡수/이동. C#=DDD 컨텍스트(§1b).

```
cmd/                          # 바이너리 진입점(유일하게 internal/ 밖에서 internal/ import)
  chainbench/          [EXT]   · engine 호출
  chainbench-mcp/      [KEEP]
  chainbench-dashboard/         [KEEP]
internal/                     # 전 구현 패키지(외부 import 컴파일러 차단)
  engine/              [NEW]   H1 · 오케스트레이터(Parse→…→Teardown 조립)
  testspec/            [NEW]   C1 · DSL Parse·Fingerprint·Interpreter
    assert/            [NEW]   C1 · 타입인지 검증 함수
  accounts/            [KEEP]  C8 · tx 서명(외부 SDK github.com/0xmhha/accounts 래핑)
  core/
    session/           [NEW]   C5 · .chainbench 정본·env 재사용·기록
    place/             [NEW]   C2 · 배치·포트 통합 + 용량검증(portplan/topology 내부 재사용)
    keyreg/            [NEW]   C2 · 키 레지스트리 + BLSDeriver boundary
    collector/         [NEW]   C4 · live tail·원격tail·chainstate·bp참여·분기
    supervisor/        [NEW]   C3 · 기동·헬스게이트·teardown·복구
    driver/            [EXT]   C7 · Transport boundary(+kill검증·key_file·upload-if-absent)
    procman/           [EXT]   C3 · {PID,datadir}·원격PID·stop 경로 배선
    genesis/           [EXT]   C2 · GenesisBuilder 4모드
    registry/          [EXT]   C6 · ChainPlugin + Capabilities
    capability/        [EXT]   C6
    obs/               [EXT]   C4 · collector 전송로
    logs/              [EXT]   C4 · 스캔→live tail
    probe/             [→collector] C4
    keys/              [→keyreg]    C2
    state/             [→session]   C5
    config/ node/ rpc/ nodeconfig/ portplan/ topology/ remote/ hardfork/ preflight/ netid/ consensus/  [KEEP]  C8/C2/C7
    pipeline/
      setup/           [REF]   C2 · Provisioner(동시화·Transport 통일)
      verify/          [REF]   C4 · 동시화
      attach/          [KEEP]  · rpc-url only(RPC 강등)
      testrun/         [REPL]  → engine+testspec
  consensus/
    poa/ wbft/ upgrade/    [KEEP]  · 합의·핸드오프
  chains/
    wemix/ wbft/ stablenet/ ...  [EXT]  C6 · plugin + Capabilities + BLSDeriver impl
      wemix/deploy/       [REF]  · 원격 공통절차 core 승격, wemix 특화만 잔류
  mcp/                 [EXT]   H3 · 결과연동
  dashboard/           [EXT]   H3 · 세션 소비
  testkit/             [REPL]  · Go-func→DSL (Report 결과모델은 재사용)
루트:
  server-set.sample.yaml  [KEEP]  · 실파일은 gitignore(SSH 자격증명)
  .chainbench/<session>/            (런타임 산출·gitignore) · session이 소유
```

**T0.0 마이그레이션**: 위 `pkg/*` 전체를 `internal/*`로 이동 + import 경로 rename(cmd/ 포함). 구조는 보존(디렉토리 그대로, 경로 prefix만 pkg→internal).
**신규 디렉토리(7)**: `internal/engine`, `internal/testspec`(+`/assert`), `internal/core/{session,place,keyreg,collector,supervisor}`.
**흡수/이동**: `core/keys`→keyreg, `core/probe`→collector, `core/state`→session.
**교체**: `pipeline/testrun`·`testkit`(Go-func) → engine+testspec DSL(결과모델 재사용).

---

## 4. 진행 규칙
- **작업 순서·상태의 단일 출처는 이 문서다.** 설계 문서([[layers]]·[[module-responsibilities]]·[[family-bringup-design]]·[[dsl-v2-proposal]] 등)는 *무엇을 왜* 만 담고, 순서표를 복제하지 않는다 — 복제하면 갈라진다(2026-08-18 실측: 착수 순서가 3문서에 중복돼 있었다).
- **TDD**: Low는 RED→GREEN 단위테스트 먼저. 동시성은 `-race`. 통합은 라이브(실 노드).
- **Go 품질**: [[go-code-quality-guidelines]](const화·DI·typed enum·ctx 전파·docs.go 등) 준수.
- **PR 단위**: Task 1개 ≈ PR 1개. 커밋은 [[commit-message-rules]]·[[pr-body-no-emoji-no-claude]]·[[git-add-explicit]].
- **하위호환 병존**: 신규 경로를 별도로 세우고 기존 `setup/upgrade/test`는 유지 → 회귀 없이 점진 이관.
- **게이트**: 각 Task는 비-e2e 통과 + 해당 F의 AC(대표 1건 라이브) 충족 후 다음.
