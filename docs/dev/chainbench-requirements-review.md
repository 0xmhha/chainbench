# chainbench 요구사항(37) 검토 · 격차 분석 · 동시성/안전성 설계

> 관점: 블록체인 · Golang · 테스트 아키텍처 전문가의 비판적 검토.
> 기준: 현재 코드베이스(아키텍처 다이어그램의 6-layer 구조) + 사용자 제시 참고사항 37개.
> 방법: (A) 37개 요구사항 문서화 → (B) 요구사항 자체의 오기입/모순 검토 → (C) 현재 코드 격차(문제점) →
> (D) 유연성 위한 설계 개선 → (E) 고루틴 동시성 + 뮤텍스/세마포어/락 안전성 → (F) 우선순위 로드맵.

---

## A. 37개 요구사항 — 주제별 문서화

| # | 주제군 | 요구사항 요약 |
|---|--------|--------------|
| 1–3 | **멀티체인 · 체인별 차이** | 멀티체인 구조 / 체인별 노드 구성법 상이 / 체인별 지원기능 상이 → 일부 테스트는 특정 체인만 적용, 체인별 설정 필요 |
| 4–6 | **노드 역할 · 저장방식 · live 유사성** | bp(생성·검증)/en(싱크) 구성 / full·archive 저장방식 / live 유사 위해 bp·en·full·archive 혼합 |
| 7–8 | **설정 소스 · 우선순위** | genesis 필수 + config 선택(없으면 default 코드) / flag 파라미터 선택(없으면 코드값) |
| 9–11,16 | **실행 위치 · remote** | local/remote 구동 / remote 접근 절차 / remote로 노드 구성정보 upload / remote 1서버=1노드는 동일포트·다른IP |
| 12–13 | **노드 관리 · 검증** | 프로세스 포함 start/stop 관리 + 상태확인 / RPC로 의도한 구성 확인 |
| 14,33,34,35 | **아티팩트 · 디버깅 수집** | `.chainbench/<날짜시간+체인>/<체인>/logs/<노드>.log` 로그 수집 / 테스트별 요청·응답·결과 기록 / 블록·bp참여·싱크높이·피어·분기 수집(대시보드) / local·remote 모두, 성능영향 없이 안정 수집 |
| 15,16 | **포트 정책** | 동일서버 다중노드 → 포트 충돌 회피 / 1서버=1노드 → 동일포트+다른IP |
| 17 | **키·노드정보 · genesis 등록** | bp 식별정보 genesis 기입 / 신규키 생성시 계정·노드정보 genesis 등록 / 기존키는 읽기 / remote는 다운로드하여 `.chainbench/<test>/<chain>/<node>/`에 저장 |
| 18–25 | **체인 종류 · 합의 · 업그레이드** | wemix(3.0)/wbft(4.0)/stablenet / wemix=etcd·wpoa·ncp / wbft=staking validator / stablenet=wbft합의+stable coin+별도정책 / wemix→wbft = 하드포크 업그레이드(블록번호 트리거) |
| 26 | **하드포크 일반화** | 하드포크명 변수에 블록번호 설정 → 해당 블록 도달시 하드포크 수행 |
| 27,28 | **테스트 생명주기 · 재사용** | 사전액션→테스트→사후액션(없으면 skip) / 이미 구성된 체인 재사용(환경 기록 비교 후 사전액션 skip) |
| 29,30,32 | **테스트 정의서(DSL)** | JSON key/value 정의서(multiple은 `,`, 중첩키 `a.b.c`), 필수·옵션 지시 / tx 서명키·파라미터 확장설계 / 검증: rpc·구현함수·log 추출 → 기대값 비교 |
| 31 | **MCP** | 전 기능 MCP 제공 + 결과 응답 전송 |
| 36 | **바이너리 · config 셋업** | 바이너리 지정 + 빌드버전 메타 기록 / config 파일로 셋업, 노드시작 옵션으로 config 지정 |
| 37 | **목적(리팩터링)** | shell→golang 전면 이관, 정의서+config 기반, 확장성·가독성·안정성·복구·명확한 결과추출로 런타임 문제 조기발견·디버깅 |

---

## B. 요구사항 자체의 검토 — 오기입 · 모순 · 모호성

> 사양 자체를 비판적으로 읽었을 때 바로잡거나 명확화가 필요한 지점.

1. **[23 · 정정됨 — 내 이전 판단이 오류] 요구사항 23이 옳다. go-wbft 블록생성 주체 = validator(staking), NCP 아님.**
   재검증 결과: wemix4 전반에서 **블록생성 주체는 "validator"** 로 불리고(TargetValidators, 밸리데이터 세트), **"NCP"는 governance 컨트랙트 맥락**(`GovNCP`, `isNCP()`, `newProposalToAddNCP`, `UseNCP`)에서만 등장한다. go-wbft 설정 **기본값은 `UseNCP=false`**(`chain/go-wbft/params/config_wbft.go:277`)이며, 이때 validator = `Stakers(govStaking)` = staking 등록 노드 전체 중 상위. 즉 **네이티브 wbft = 순수 staking-validator, NCP 미사용.**
   wemix4 테스트 genesis가 `UseNCP=true`인 것은 **wemix→wbft 업그레이드 시나리오에서 wemix의 NCP governance 승계를 검증**하기 위한 *테스트-특정 설정*일 뿐, wbft 네이티브 모델이 아니다. → 내 이전 "[23 중대]" 지적(=permissioned/NCP)은 철회. **요구사항 23은 정확하다.**

2. **[14 vs 17 vs 33 vs 34 · 중대] `.chainbench` 디렉터리 레이아웃이 항목마다 다르게 서술되어 단일 정본이 없다.**
   - 14: `.chainbench/<날짜시간+체인>/<체인>/logs/<노드>.log` — 세션명에 체인명, 다시 하위에 체인 폴더로 **중복**.
   - 17/33/34: "테스트 폴더 하위 …" 로 시작하나 세션·체인·테스트의 중첩 순서가 항목마다 불일치.
   → **하나의 정본 스키마**로 통합 필요(§D-1 제안).

3. **[7,8 · 확정] 우선순위 = flag > config > default (geth-family 표준). 사용자 확인 완료.**
   chainbench는 `nodeconfig.Generate`로 TOML을 쓰고(`ConfigPath`), `LaunchArgs`로 CLI flag를 함께 넘긴다. geth 계열은 config 파일을 로드한 뒤 **CLI flag가 override** → 실효 **CLI flag > config file > code default**. (handoff는 config 파일 없이 flag-only.) 사용자 확인으로 이 동작이 **의도된 정본**임이 확정됨(별도 커스텀 resolution 불필요).

4. **[28 · 정정됨] 병렬 없음 — 모든 테스트는 직렬. 같은 체인 구성은 연속 실행으로 재구성 생략.**
   내 이전 "재사용=직렬 / 격리=병렬" 프레이밍 철회. 정확히는: **한 테스트 수행 중 다음 테스트는 대기(항상 직렬)**. 각 테스트는 자신이 필요한 **환경(체인 구성)을 정의**하고, 러너는 현재 구성이 그 정의와 일치하면 setup(사전액션)을 **skip**하고 연속 수행한다(재구성 비용 절감). 즉 "테스트 정의의 환경 명세 + 현재 구성 fingerprint 비교 → skip/재구성"이 핵심(§D-5).

5. **[5 · 경] "각 블록의 단계 상태" 는 모호.** archive 노드 = **모든 과거 world-state(블록별 state trie 전체) 보존** → 임의 과거 블록 상태조회 가능. "단계 상태"보다 "블록별 전체 상태 이력"이 정확.

6. **[20 · 경] "go-wbft가 go-wemix 블록을 전달받아 싱크" 는 표현 주의.** 별도 체인이 wemix 블록을 수신하는 게 아니라, **동일 체인**을 devp2p로 싱크하며 go-wbft 바이너리가 pre-fork(wpoa) 블록을 검증하는 것. 포크 이후 생성 주체만 전환.

7. **[16 + 19 · 경] remote 동일포트 모델은 etcd/p2p peer URL(ip:port) 도달성에 의존.** 1서버=1노드·동일포트는 OK지만, wemix etcd 클러스터 형성은 각 노드의 **도달 가능한 peer IP**를 전제로 한다(방화벽/바인드 주소 주의).

8. **[3 vs 29 · 경] 체인별 테스트 적용성** 은 정의서의 **1급 필드**(예: `applicableChains`)로 두고 러너가 미적용 체인은 skip 처리해야 한다. 현 사양엔 "일부는 안 됨"만 있고 표현 수단이 없다.

9. **[35 · 경] "성능영향 없이 수집" 은 구현 제약을 강제한다.** 노드 프로세스를 블로킹하지 않는 **out-of-process 로그 tail/stream + 비동기·버퍼·레이트리밋** 이 유일한 해법(§E).

**총평:** 37개는 방향으로서 일관되며, (1)~(4)는 후속 Q&A에서 **모두 확정**되었다 — [23] validator(NCP 아님) 정정, [14/17/33/34] §D-1 정본 스키마, [7,8] flag>config>default 확정, [28] **항상 직렬**·같은 구성 연속 재사용. 남은 미결 의사결정 없음.

---

## C. 현재 코드 격차 — 우리의 문제점

| 요구군 | 현재 상태 | 판정 | 핵심 문제 |
|--------|-----------|------|-----------|
| 1,2,3,18,24 멀티체인 플러그인 | `registry` + `chains/{stablenet,wbft,wemix}`, `Family()` | 🟢 구조 있음 | 체인별 **노드구성법(2)**·**테스트 적용성(3)** 을 선언·강제하는 매트릭스 없음 |
| 4,5,6 역할·저장·혼합 | `topology`가 role + `full/snap/archive` 검증 | 🟡 부분 | live 유사 **혼합 프리셋/강제(6)** 없음; 현 handoff는 1P+4V(EN 없음)로 비-live |
| 7,8,36 설정소스·우선순위·바이너리메타 | genesis 템플릿 + `profiles` + flag | 🟡 부분 | **우선순위 미정**, 통합 `--config` 없음, **바이너리 빌드버전 메타 기록 없음(36)** |
| 9,10,11,16 remote | `core/remote`(ssh/auth), `driver.RemoteDriver`, `chains/wemix/deploy` | 🟡 부분 | wemix 전용에 치우침; **모든 체인 일반화**·노드구성 upload 표준화 미흡 |
| 12 노드관리 | `procman`(PID추적·StopAll 검증 로직 존재) | 🟡 부분 | **StopAll 검증·leak-report 로직은 있으나 프로덕션 stop 경로에 미배선**(실제 stop은 검증 없는 `Kill()`, 테스트 전용)·**로컬 PID만**; remote 프로세스 관리·검증 표준화 미흡 (컴포넌트 §2b) |
| 13 RPC검증 | `pipeline/verify` + `core/rpc` | 🟢 | 노드별 폴링 **순차**(verify.go:90) |
| 14,33,34,35 아티팩트·디버깅 | 로그→`<dataRoot>/logs/`, `nodeset.json`, `obs`+`dashboard`(SSE) | 🔴 미흡 | **`.chainbench` 세션 레이아웃 부재**; 테스트별 요청/응답/결과 기록 없음; 블록·bp참여·싱크·분기 수집 **깊이 부족**; remote 무영향 수집 미검증 |
| 15,16 포트 | `portplan`(스텝) | 🟡 부분 | **handoff는 고정포트(30010 step10)** → 연속/병렬 충돌(이 세션 실측 문제); local스텝·remote동일포트 **단일 배치기 부재** |
| 17 키·genesis·다운로드 | `keys generate`(신규 등록), `deploy/credentials·keys`(다운로드) | 🟡 부분 | `.chainbench/<test>/<chain>/<node>/` 표준 위치 미준수 |
| 25,26 하드포크 | `core/hardfork`, `consensus/upgrade` | 🟢 | (일반 하드포크 블록번호 주입은 overlay/override로 가능) |
| 27 사전/사후 액션 | 없음 | 🔴 없음 | **pre→test→post 생명주기 프레임워크 부재** |
| 28 재사용·fingerprint | `attach` + `nodeset.json`/`state.json` | 🔴 미흡 | **환경 fingerprint 비교→skip** 없음. e2e 14개 GOV 테스트가 **각자 handoff 부팅**(비효율·비-stateful) |
| 29,30,32 테스트 DSL | `testkit.Case`(Go 함수) + `Report`(JSON) | 🔴 없음 | **JSON 정의서 DSL·해석기 부재**. 테스트=Go 코드 → shell→go 이관의 핵심 미구현 |
| 31 MCP | `pkg/mcp` 30 tools | 🟢 | 결과 응답 경로는 dashboard/report와 연결 필요 |
| 37 목적(안정성·복구) | 재시도(runGovHandoff) | 🔴 미흡 | 재시도가 **실패원인 은닉**, 헬스기반 복구 없음. etcd flaky를 "인식·복구"로 다루지 못함 |

**요약된 3대 문제**
1. **테스트 계층의 선언성 부재** — 테스트가 Go 코드라 shell 스위트의 확장성/가독성 목표(37)를 못 채운다. **JSON 정의서 + 해석기**(29,30,32)가 없다.
2. **런타임 아티팩트/상태의 정본 부재** — `.chainbench` 세션 레이아웃·환경 fingerprint·디버깅 수집(14,17,28,33,34,35)이 흩어져 있어, 재사용·디버깅·대시보드가 모두 반쪽이다.
3. **오케스트레이션의 순차성·불안정성** — 노드 기동/검증이 순차(exec.go:163), 포트 고정, 재시도가 원인을 숨김 → 대규모(7+노드)·연속 실행에서 느리고 취약(15,16,27,35,37).

---

## C-etcd. "etcd flaky"의 실체 — go-wemix 내장 etcd 코드 분석 (요구 3)

> "flaky"는 무작위가 아니다. `chain/go-wemix/wemix/etcdutil.go` 를 분석하면 **결정적 타이밍/스케줄 문제**임이 드러난다.
> **출처(S4):** 아래 go-wemix/go-wbft 내부 분석은 **외부 go-wemix 체크아웃**(예: `packages/` 트리) 기준이며, chainbench repo에는 vendoring되어 있지 않다(라인번호는 그 체크아웃 기준). chainbench repo에서 직접 확인되는 것은 **`admin.etcdInit()` 호출부**(`pkg/consensus/poa/bootstrap_exec.go:44`)와 포트 예약(`portplan`)뿐이다.

**메커니즘**
- **etcd peer 포트 = `P2P+1`**(wemix 바이너리가 파생). chainbench는 이를 `portplan.Ports.Etcd`로 예약하며, RPC 밴드(http/ws/auth)와 **분리된 밴드**라 충돌하지 않는다(p2pStep≥2·rpcStep≥3). 클러스터 토큰 = `etcdClusterName` 고정. (client 포트는 go-wemix 내부이며 chainbench가 별도 예약하지 않는다 — peer(+1)만이 충돌-임계 포트.)
- **부트스트랩** `etcdInit()`(=chainbench의 `admin.etcdInit()`): `newConfig(true)` → `ClusterStateFlagNew + ForceNewCluster` → 새 단일노드 클러스터의 **리더**가 되고, 이후 그 노드의 `MiningPeers`에 `"*"` 표식이 붙는다.
- **조인** `etcdAutoJoin()`(노드가 주기적으로 자동 수행): 아직 조인 안 한 miner들(`tobes`)을 모아 개수로 `gap`을 정한다 — **sz≤11 → gap=7s, ≤23 → 11s, ≤41 → 17s, 그 외 23s**. 자기 index `ix`에 대해 **시간 슬롯** `t = (ct/tt)*tt + sz + ix*gap ± gap/4` 을 계산하고 **그 슬롯까지 `time.Sleep`**. 슬롯에서 `"*"` 보유 + `up` 상태인 리더를 찾고 — **있으면 join, 없으면 즉시 `"etcd join failed: not found"` (호출 내 재시도 없음)**.

**왜 handoff에서 실패하나 (두 모드)**
1. **첫 기동 타이밍**: producer(단일 etcd 노드)가 `etcdInit`로 리더가 되기 *전에* 내부 `etcdAutoJoin`이 자기 슬롯에 도달해 "리더 없음"으로 `join failed`. 다음 슬롯까지 **최대 `tt = sz*gap`초(sz=11이면 77초)** 를 허비 → 이것이 "handoff not observed within 100s" 와 재시도 수분 소요의 원인. (무작위가 아니라 **슬롯-스케줄 낭비**.)
2. **재기동 시 stale etcd**: 체인을 종료 후 **같은 datadir로 재기동**하면 etcd가 `Existing` 상태로 죽은 peer에 접속 시도 → `"cannot fetch cluster info from peer urls"`. (요구 3의 "종료후 재실행 관리 부재"가 정확히 이 지점.)

**chainbench가 해야 할 것 (프로세스/타이밍 관리)**
- **리더-우선 확정**: producer의 `etcdInit` 호출 후 **etcd 준비(`etcdIsReady`/리더 `"*"`)를 폴링 확인**하고 나서 다음 단계 진행. "호출하고 기대"가 아니라 **확인 게이트**.
- **시작 간격의 체계적 설정**: 조인이 필요한 구성에서는 **리더 부트스트랩 완료 시점 직후**에 조인 슬롯이 오도록 노드 시작 시각을 `gap`(7/11/17/23s)에 맞춰 정렬 → 슬롯 낭비(`tt`초) 제거. (요구 3의 "연결이 바로 되도록 시작 간격을 체계적으로")
- **재기동 시 datadir 정리(S2 정정)**: etcd는 **노드 프로세스 내장**이라 프로세스를 종료하면 **함께 종료**된다("살아있는 etcd 정리"는 불필요). 문제는 **같은 datadir의 낡은 클러스터 상태**뿐이므로, **재-셋업 전 datadir 삭제**로 `Existing` 조인 루프를 차단한다. **노드 종료(kill PID)와 datadir 삭제는 별개 기능**이며, 재-셋업 = StopAll + RemoveDataDir. 정리 안 하면 데이터가 계속 쌓여 디스크 관리가 안 된다(요구 12·D-1 세션 정리).
- **procman 확장(datadir 추적)**: 현재 procman은 `{PID, Label}`만 추적(`procman.go:22`)하고 **datadir는 추적하지 않는다**. 정확한 종료+정리를 위해 노드별 **`{PID, datadir}`** 를 추적하도록 확장해야 한다. 또한 etcd는 별도 PID가 없으므로 **상태는 RPC/IPC(`etcdIsReady`, `getMiners`의 `"*"`)로 관측**해 lifecycle에 포함(요구 12·3). (chainbench에는 `cmd/chainbench/clean.go`가 `os.RemoveAll(dataDir)`로 루트 삭제를 이미 제공 — 노드별 추적과 연계 필요.)

> 요약: 이전 세션의 "flaky"는 **go-wemix etcd의 슬롯-스케줄 조인 + stale-datadir 재조인** 두 가지이며, chainbench는 (i)리더 준비 확인 게이트, (ii)슬롯(`gap`)에 맞춘 시작 정렬, (iii)재기동 시 **datadir 삭제**로 "바로 연결"을 만들 수 있다(내장 etcd는 프로세스 종료로 함께 죽으므로 별도 정리 불필요). 내가 넣었던 "port-free 폴링"은 이 실체와 무관했다.

---

## D. 유연성을 위한 설계 개선 제안

> **이하 D-1·D-2는 이번 설계 검토(전 세션 Q&A)의 확정본이다.** 세부 결정은 각주 §D-2.x 에 정리.

### D-1. `session` 패키지 — `.chainbench` 아티팩트 정본 (14,17,28,33,34,35) · **확정 폴더 구조**
**세션 = 테스트 수행 커맨드 1회.** 한 커맨드가 여러 테스트를 (직렬로) 수행하며, 각 테스트가 **서로 다른 체인/환경**을 쓸 수 있으므로, 세션 아래를 **`environments/`(공유 체인 인스턴스) + `tests/`(테스트별 기록) 2축**으로 분리한다. 체인 아티팩트(로그·키·chainstate)는 환경 소유, 판정·결과는 테스트 소유, 테스트는 자신이 쓴 환경을 **참조(env-ref)** 한다(연속 재사용 시 중복 없음).
```
.chainbench/
  <session>/                                   # = 테스트 수행 커맨드 1회
    session.json                               # 커맨드·시각·테스트 목록/순서·요약결과
    keys/<name>/{private,address,bls?,pop?}    # (a) 키 레지스트리: op1/bp1/acctA→키. 랜덤생성 or 복사 or remote다운로드; remote 실행 시 약속경로로 업로드
    environments/
      <env-id>/                                # env-id = "env-"+fingerprint[:12] (§D-2.4·L5: 선언값 해시 앞 12hex)
        env.json                               # fingerprint(설정만) + node table + dataPath
        genesis.json  config/                  # 사용된 genesis·config 스냅샷(36 기록)
        nodes/<node>/{nodekey,address,bls,keystore}   # 노드 신원(17) (keys/ 레지스트리 참조)
        logs/<node>.log                        # tail 누적(14, §D-2.7)
        chainstate/{blocks,bp-participation,sync,peers,forks}  # 크로스노드 수집(34, §D-2.5)
    tests/
      <NNN>_<test-id>/                         # 세션→테스트, 순번 정렬
        spec.json  env-ref→environments/<env-id>
        steps.json                             # tx 입력·해시·논스·가스·영수증
        assert.json                            # 검증 출처 provenance(rpc/func/log) — §D-2.6
        status.json                            # pass/fail/blocked·시간
        postaction.json                        # (i) postAction 결과 (판정과 독립)
```
`env.json` 의 **node table** (endpoint 해석의 근거):
```jsonc
"dataPath": "/data/.../<env>",   // (g) 노드 로그가 쌓이는 실제 경로
"nodes": [
  {"name":"bp1","role":"bp","sync":"archive","binary":"go-wbft","buildVersion":"...","rpc":"http://IP:PORT","ws":"..."}, // 도메인 용어 bp/en (§3.9; 코드 enum은 내부 validator/endpoint)
  {"name":"en1","role":"en","sync":"full",   "binary":"go-wbft","buildVersion":"...","rpc":"http://IP:PORT2","ws":"..."}
]   // local=포트 상이 / remote=동일포트+다른IP (15,16)
```

### D-2. 선언적 `TestSpec`(JSON DSL) + **atomic 해석기** (29,30,32,37 · 요구 1)
shell의 느슨한 명령 나열을 **트랜잭션적 스텝**(성공/실패 명확, 부분성공 없음)으로 치환. 스키마(확정):
```jsonc
{
  "id": "GOV-005", "applicableChains": "wbft",              // (3) 미적용 체인은 SKIP
  "chain":   { "name":"wbft", "binary":"go-wbft",           // (f) 단일=전노드 동일 / 혼합은 아래
               "binaries": {"producer":"go-wemix","default":"go-wbft"},  // 또는 "profile":"wemix-upgrade"
               "config":"...", "genesisOverlay":{...} },
  "topology":{ "bp":7, "en":5, "sync":{"bp1":"archive","default":"full"}, "bootnode":15 }, // (4,5,6)
  "hardforks": { "croissant":100, "brioche":50 },           // (25,26) 값 있으면 fork 존재
  "placement": "local|remote", "remote": {"cluster":"cluster.yaml"}, // (9-11,16)
  "defaultOn": "bp:any",                                    // endpoint 폴백 (§D-2.1)
  "preActions":  [ {"ensureChain":true}, {"ensureStaker":{"name":"A"}} ],  // (27,28) idempotent 가드
  "steps":       [ {"tx":{"on":"bp1","signer":"op1","call":"registerStaker(...)","args":[...],
                          "gas":"auto|<n>|null","waitFor":"receipt"}} ],   // (30,a,b)
  "assertions":  [ {"on":"bp1","source":"rpc","method":"istanbul_getValidators","assert":"Len","expected":7},
                   {"on":"en1","source":"rpc","method":"istanbul_isValidator","assert":"False"},
                   {"onEach":["bp1","en1"],"source":"rpc","method":"eth_getBlockByNumber",
                    "assert":"EqualHashAt","at":"<h>"},                   // (34) 무분기
                   {"on":"bp1","source":"log","waitLog":true,"match":"block reward .*","assert":"NotNil"} ],
  "postActions": [ {"unstake":{...}} ]
}
```

**§D-2.1 Endpoint/target (어디서 테스트하나)** — 스텝의 `on`=tx 제출 노드, 검증의 `on`=조회 노드, `onEach`=다중노드 비교. 셀렉터: 이름/역할+index/`bp:any`·`en:any`. 미지정 시 `defaultOn`. 해석기가 셀렉터→`env.json.nodes[].rpc` 로 해석.

**§D-2.2 키/논스/가스** — (a) 모든 키는 세션 `keys/`에 이름매핑. 랜덤생성·복사·remote 다운로드/업로드 일원화. 논스: `eth_getTransactionCount` 순차. 가스 `gas`: `"auto"`(RPC 추정)/`<숫자>`(static)/`null`(미설정 검증용).

**§D-2.3 wait 프리미티브** — `waitReceipt·waitBlocks·waitEpoch·waitSeconds·waitFork`(독립 스텝 또는 `waitFor` modifier). **모든 wait에 timeout 필수**, 초과 시 hang 없이 FAIL/ERROR + 아티팩트(37). 테스트 전체 timeout 별도.

**§D-2.4 fingerprint(선언값) ↔ preAction(런타임)** — fingerprint = **테스트 정의서에 이미 적는 값에서 파생**: `sha256(binaries-set + genesis + config + topology + hardforks + placement)`(placement=local↔remote 포함, O1). 체인을 건드리기 전 **정의서 필드 비교만으로 재사용 판정**(동일=재사용, 상이=재구성). **폴더명 env-id = `"env-"+앞 12hex`**(전체 해시는 env.json에만, 경로초과 방지·L5). 런타임 상태("staker A 등록됨")는 fingerprint가 **아니라** preAction의 idempotent 가드(RPC 확인 후 있으면 skip). 둘을 섞지 않음.

**§D-2.5 크로스노드 검증** (4,34) — `istanbul_getCommitSignersFromBlock`로 **bp 참여**, en `eth_blockNumber`로 **싱크 추종**, `onEach`+`EqualHashAt`로 **무분기(전 노드 동일 hash)**.

**§D-2.6 assert 함수 + provenance** (32) — 유닛테스트 네이밍: `Equal·NotEqual·NotNil·Nil·True·False·Contains·Greater(OrEqual)·Less(OrEqual)·Len·Regexp·InDelta·ElementsMatch·EqualCI·EqualWith`. wei/address/hex/bool **타입 인지 비교**. `assert.json`에 **출처 provenance**: rpc(method·params·raw)/func(name·args·return)/log(**logFile·lines·byteOffset·extracted**).

**§D-2.7 data path + 라이브 로그** (g) — 노드 로그가 쌓이는 `dataPath`를 설정. 수집기는 **append-only tail**(일회성 복사 아님)로 누락 방지. 로그검증은 `waitLog`(패턴 나올 때까지 tail, timeout)로 완결성/가독성 보장. remote는 SSH tail. out-of-process·무영향(35).

**§D-2.8 바이너리 셋 + 하드포크 2 type** (f,h,36) — 환경은 **`node→(binary,buildVersion)` 집합**. 쉬운 설정: `binary` 단일=전노드, 혼합만 `binaries`/`profile`. 하드포크 2종: **① 체인 업그레이드**(go-wemix→go-wbft, 서로 다른 바이너리 동시=handoff) / **② 동일체인 하드포크**(fork 블록 전에 fork-aware 바이너리로 교체, pre≠post 바이너리). config의 `hardforks` 값 유무로 존재 판정, 내용은 기록용 수집.

**§D-2.9 pre/post 실패** (i,27) — preAction 실패 → **테스트 미수행(BLOCKED)**. postAction 실패 → **어떤 테스트인지 기록**하되 **판정 집계는 assertion 기준 독립**.

- **닷 경로 리졸버**(`a.b.c`) + multiple `,` 파서(29). **필수/옵션 검증기** + JSON 스키마로 신규 테스트 진입장벽 하강(37).

### D-3. Chain capability 매트릭스 (1,2,3)
각 플러그인이 `NodeComposition()`·`SupportedForks()`·`SupportedAssertions()`·`TestCapabilities()`를 선언. 해석기가 정의서의 `applicableChains`/기능요구와 대조해 **미적용은 SKIP**(현 `capability` 확장).

### D-4. 통합 Placement/Port 배치기 (15,16)
`place.Allocate(nodes, mode, cap)` (design §3.4, `cap Capacity`로 **용량 사전검증**):
- `local`: 노드 index 기반 **결정적 포트 스텝** 또는 OS 할당(`:0` 바인드 후 회수) → 고정포트 충돌(현 handoff 문제) 제거.
- `remote-1-per-host`: **동일 포트 + 서버별 IP**.
- **용량 검증(fail-fast)**: validators ≥ 4(BFT min) 미달·노드수 > max(local 포트대역 / remote Σ서버슬롯) 초과 → 배치 이전 오류.
portplan·topology·remote 를 하나의 배치 결과(NodePlacement)로 수렴.

### D-5. 환경 fingerprint + 재사용 컨트롤러 (28)
fingerprint 정의는 **§D-2.4 확정본**: `sha256(binaries-set + genesis + config + topology + hardforks + placement)` — **테스트 정의서 선언값에서 파생**(런타임 상태 아님). 다음 테스트의 선언값 fingerprint를 현재 `environments/<env-id>`와 비교해 **일치 시 setup skip(재사용)**, 불일치 시 새 env-id로 재구성(폴더명은 12hex 축약·L5). 런타임 상태는 preAction(idempotent RPC 가드)이 담당. (모든 테스트 직렬 — §B-4.)

### D-6. 헬스 기반 복구 수퍼바이저 (27,35,37)
현 "재시도+원인은닉"을 대체: launch 후 **진단 게이트**가 실제 실패모드(etcd join 실패 vs fork 미도달)를 분류하고, (a)백오프 재기동 또는 (b)명확한 아티팩트로 실패. 실패원인·producer 로그를 세션에 보존(이번 세션에서 시작한 진단 로깅의 정식화).

---

## E. 고루틴 동시성 · 뮤텍스/세마포어/락 안전성

> 현재: 노드 기동/검증이 **순차**(exec.go:163, verify.go:90), 비-테스트 프로덕션 goroutine **4곳**(local.go:70, subscribe.go:56·57, dashboard/client.go:23)·mutex **6파일**뿐(sync 프리미티브 극소수). 대규모·연속 실행의 병목이자 취약점.

### E-1. 동시화할 곳 (goroutine + errgroup + 세마포어)
| 대상 | 현재 | 개선 | 안전장치 |
|------|------|------|----------|
| 노드 init/provision/launch | 순차 loop | `errgroup` 팬아웃 | **`semaphore.Weighted(max(1, min(cores-2,N)))`** 로 동시 기동 상한(1~2코어 언더플로우 클램프, S1) → etcd/gwemix 자원 스파이크·성능영향(35) 방지 |
| 노드별 헬스/RPC 폴링(verify) | 순차 | 노드별 goroutine | 결과는 **index 기반 슬라이스 쓰기**(락 불필요) |
| handoff mesh(admin_addPeer, N×N) | 순차 | 팬아웃 | ctx 취소 전파 |
| 로그/메트릭 수집(local tail·remote stream) | 미흡 | 노드별 수집 goroutine | **채널→단일 writer** 로 직렬화(파일 쓰기 레이스 제거) + 버퍼·레이트리밋(35) |

> **주의(§B-4):** 테스트 실행 자체는 **항상 직렬**(병렬 없음). 위 동시성은 모두 **한 환경(체인)을 세울 때 그 안의 N개 노드**를 동시에 다루는 것이지, 여러 테스트를 동시에 돌리는 것이 아니다.

### E-2. 필요한 동기화 프리미티브
- **`sync.Mutex`**: procman PID 맵(존재, 유지), 세션 결과맵, `env.json`/`session.json` 쓰기.
- **`sync.RWMutex`(승격 검토)**: 다독 대상. **주의: obs.Bus는 현재 `sync.Mutex`**(bus.go:17)이며, capability 셋도 mutex — 다독이 지배적일 때만 RWMutex로 승격.
- **`semaphore.Weighted`(x/sync)**: 노드 기동 동시성 상한 `max(1, min(cores-2,N))`(35 성능·S1 클램프) — 버퍼드 채널로 대체 가능.
- **`errgroup.Group`**: 팬아웃 + **최초 에러 시 ctx 취소** → 형제 노드 정리(고아 방지, procman 연계).
- **`sync.Once`**: registry/embed 1회 초기화.
- **`context.Context`**: 전 goroutine에 전파 → 실패/타임아웃 시 일괄 취소·teardown.
- **파일 락(flock) 또는 단일 writer**: `.chainbench` 세션 상태를 복수 러너가 만질 때.

### E-3. 반드시 피해야 할 레이스 (비판적 점검)
1. **동일 포트 이중 바인드** — 고정포트(현 handoff)에서 **연속(back-to-back) 재구성** 시 발생. → **결정적 index 포트 or OS 할당(`:0`)** 으로 근절(내가 넣은 port-free 폴링은 TOCTOU라 근본해결 아님).
2. **NodeSet 슬라이스 동시 append** — 팬아웃 시. → index 쓰기 또는 mutex.
3. **공유 genesis 템플릿 변형** — 절대 embed 원본을 mutate 금지; overlay는 항상 새 바이트 반환(현재 `MergeOverride` OK) — 규약으로 고정.
4. **단일 로그/결과 파일 동시 쓰기** — 수집 goroutine 다수. → 채널 단일 writer.
5. **obs 버스 publish 중 구독자 수정** — 현재 `sync.Mutex`(bus.go:17)로 보호됨. 확장 시 락 규약 유지.
6. **procman 맵 동시 접근** — mutex 존재. remote PID(ssh) 추적으로 확장 시 동일 규약.
7. **ctx 미취소로 인한 고아 노드** — errgroup+ctx+procman.StopAll 로 실패 시 전 노드 확정 종료(12 상태검증과 연결).

### E-4. 원칙
- **소유권 단일화**: 각 파일/포트/PID는 소유자 1명(세션·배치기·procman)이 직렬 관리, 나머지는 채널/불변데이터로 접근 → 락 표면 최소화.
- **팬아웃-팬인 + 상한**: errgroup으로 팬아웃, semaphore로 상한, index/채널로 팬인 → 락 없는 결과수집.
- **취소 우선**: 모든 장기작업 ctx 종속 → 실패 조기전파·자원회수(35 성능·37 안정성).

---

## F. 우선순위 로드맵 (영향×비용)

1. **[기반] `session` 패키지(D-1)** — 이후 모든 항목(로그·키·결과·fingerprint·디버깅)의 정본. 선행 필수.
2. **[기반] Placement/Port 배치기(D-4)** — 고정포트 근절 → 동시성·연속실행의 전제(E-3-1).
3. **[핵심] TestSpec DSL + 해석기(D-2)** — 37의 본체(shell→go). session/placement 위에 구축.
4. **[안정] 동시 기동 + 헬스복구(E-1, D-6)** — errgroup+semaphore+ctx, 진단 게이트로 flaky 대응.
5. **[재사용] fingerprint 컨트롤러(D-5)** + capability 매트릭스(D-3) — 연속·체인별 테스트.
6. **[연결] MCP·dashboard 결과연동(31,33,34)** — 세션 아티팩트를 소비.

> 결론: 현재 구조는 **플러그인·consensus·driver 기반이 견고**하나, (i)테스트 계층의 선언성, (ii)런타임 아티팩트 정본, (iii)오케스트레이션 동시성·복구 3축이 비어 37의 목적(확장성·안정성·디버깅)을 아직 못 채운다. §F 순서로 세우면 유연성과 안전성을 함께 확보할 수 있다.
