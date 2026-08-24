# 체인 구성 절차 — 공통 파이프라인과 변곡점

> 목적: 테스트를 돌리기 전에 **체인을 어떻게 세우는가**를 단계로 확정하고, 각 단계에서
> **무엇을 바꿀 수 있는가(변곡점)** 를 한자리에 모은다. 케이스별 상세는 아래 4개 문서로 분리한다.
> 검증 기준일: 2026-08-09. 실측은 이 저장소 `main` + `/Users/0xtopaz/work/github/0xmhha/chain` 빌드 산출물.

| 케이스 | 문서 | 현재 상태 |
|---|---|---|
| gwemix 단독 | [case-1-wemix.md](case-1-wemix.md) | ⚠️ **절차 확정, 자동화 미구현** — 수동 절차는 §1b 로 확립 |
| gwemix → gwbft 하드포크 핸드오프 | [case-2-wemix-to-wbft.md](case-2-wemix-to-wbft.md) | ✅ **수동 절차 검증 완료** (블록 100 인계 확인) / 자동화는 미구현 |
| gwbft 단독 | [case-3-wbft.md](case-3-wbft.md) | ✅ **동작 확인** (라이브) |
| gstable 단독 | [case-4-stablenet.md](case-4-stablenet.md) | ✅ **동작 확인** (라이브·CI 게이트) |

**다음 작업을 이어받는다면 → [next-automation.md](next-automation.md)** (컨텍스트 + 남은 작업 리스트)

점검용 CLI: `chainbench chain`. 케이스별 절차를 **단계 단위로 실행·중단·검증**한다 — [§4](#4-cli-로-직접-점검) 참조.

---

## 1. 공통 파이프라인

체인 종류와 무관하게 아래 순서다. 다른 것은 **부트스트랩 유형** 하나뿐이며, 그것이 매니페스트의
`bootstrap.type`으로 선언된다.

| # | 단계 | 하는 일 | 소유 패키지 |
|---|---|---|---|
| 1 | **resolve-chain** | 체인 플러그인 해석(매니페스트: chain_id·binary·family·hardforks·capabilities) | `core/registry` |
| 2 | **resolve-binary** | 실행할 노드 바이너리 확정(명시 경로 > PATH) | `cmd` |
| 3 | **load-preset** | 키셋 로드(nodekey·keystore·주소·BLS/PoP·alloc·검증자셋) | `core/keys` · `core/keyreg` |
| 4 | **allocate** | 노드별 host/port 배치 + **용량 사전검증**(min validators, 포트 대역) | `core/place` |
| 5 | **genesis** | genesis 바이트 산출(4모드 §2.5) | `core/genesis` · `engine.GenesisSource` |
| 6 | **assemble-plan** | 배치+genesis+역할 → `setup.Plan` (노드별 datadir·config·args) | `engine.AssemblePlan` |
| 7 | **provision** | genesis·per-node config 물질화(upload-if-absent; 로컬 FS 또는 SSH) | `core/provision` |
| 8 | **init-datadir** | 노드별 `init` (각자 자기 바이너리로) | `core/driver` |
| 9 | **launch** | 노드 기동 + PID 추적 | `core/driver` · `core/procman` |
| 10 | **health-gate** | 건강 판정(블록 전진 / etcd 리더 / 포크 도달) | `core/supervisor` |
| 11 | *(bootstrap)* | `bootstrap.type` 이 `governance-etcd` 인 체인만: 거버넌스 배포 + etcd 초기화 | `consensus/poa` |
| 12 | **teardown** | SIGTERM→SIGKILL→고아 검증, datadir 삭제는 **별개 연산** | `core/procman` |

```
bootstrap.type = "static"          bootstrap.type = "governance-etcd"
  (wbft, stablenet)                  (wemix)
  1..10 을 그대로 한 번               §1b 의 2-페이즈 순서 (전혀 다르다)
```

**핵심 비대칭:** `static` 체인은 **genesis에 검증자셋이 이미 들어 있어** 기동 즉시 합의가 성립한다.
`governance-etcd` 체인은 **genesis에 검증자가 없고**, 기동 후 거버넌스 컨트랙트를 배포하고
etcd 클러스터를 형성해야 비로소 블록을 만든다. 실패하면 체인은 "떠 있지만 멈춘" 상태가 된다.

---

## 1b. governance-etcd 부트스트랩 — 2-페이즈 순서 (실증 확인)

> 근거: 참조 구현 `stablenet/packages/chainbench/tests/wemix4/envs/default/bootstrap.sh`
> + 2026-08-09 실 바이너리 재현. **순서를 지키지 않으면 etcd 클러스터가 형성되지 않는다.**

```
── 페이즈 A: 부트스트랩 (프로듀서 단독) ────────────────────────────
 A1  genesis 준비 + 전 노드 datadir init
 A2  프로듀서 1대만 기동            ← 다른 노드는 절대 띄우지 않는다
 A3  deploy-governance  (IPC)
 A4  admin.etcdInit()   (IPC)
 A5  verify-etcd: admin.wemixInfo.etcd.cluster 가 비어 있지 않은지 확인
 A6  프로듀서 종료
── 페이즈 B: 운영 (전체) ──────────────────────────────────────────
 B1  전 노드 기동 (프로듀서 + 후계 검증자)
 B2  풀메시 연결 (admin_addPeer)     ← 재기동 뒤이므로 다시 해야 한다
 B3  안정화 대기(~30s) 후 블록 전진 확인
```

**A2가 핵심이다.** 전체 네트워크가 뜬 상태에서 `etcdInit` 을 호출하면 클러스터가 만들어지지 않는다.
피어가 보이면 노드가 *bootstrap* 이 아니라 *join* 경로로 들어가는 것으로 보이며, 로그의
`etcd join failed: not found` 와 일치한다. 단독 상태에서는 즉시 형성된다:

```jsonc
// A5 에서 관측 (단독)                          // 전체 기동 상태에서 관측
{"cluster":"producer=https://127.0.0.1:30011",  {"cluster":"", "members":null}
 "leader":{"name":"producer"},
 "members":[{"name":"producer", ...}]}
miners: "producer/up/*"   ← 리더 표식 '*'        miners: "producer/up"
```

**`admin.etcdInit()` 의 반환값은 판정 근거가 아니다** — 성공해도 `null` 을 반환한다.
반드시 A5 로 클러스터 상태를 확인해야 한다.

### 후계 검증자에 필요한 것 (페이즈 B)

BFT 검증자는 **자기 서명키가 열려 있어야** 봉인할 수 있다. static 경로(`engine.armSpecs`)는
검증자마다 아래를 하지만, 핸드오프 경로는 **프로듀서에만** 해서 포크 후 인계가 실패했다.

- datadir 에 **keystore 배치**
- `--unlock <addr>` · `--password <file>` · `--miner.etherbase <addr>`

이게 없으면 WBFT 는 라운드만 돌고(`Commit new sealing work number=<fork>` 반복) 진행하지 못한다.

### 메시는 최종 기동 뒤에 (페이즈 B2)

A6 에서 프로듀서를 껐다가 B1 에서 다시 켜므로, **B1 뒤에 메시를 다시 연결**해야 한다.
빠뜨리면 검증자들이 프로듀서하고만 연결되어(`peers=1`) 서로의 합의 메시지를 못 받고,
ROUND-CHANGE 가 자기 것만 쌓인다(`currentRoundChanges.count=1`).

---

## 2. 변곡점 카탈로그

각 단계에서 바꿀 수 있는 것과, **어디에 쓰는지**.

### 2.1 체인 선택 (단계 1)

| 변곡점 | 설정 위치 | 값 | 비고 |
|---|---|---|---|
| 내장 체인 | `--chain` / spec `chain.name` | `stablenet` \| `wbft` \| `wemix` | 기본 `stablenet` |
| 외부 체인 | `setup --manifest <json> --genesis-template <json>` | 프로젝트 제공 매니페스트 | 내장 family 위에 얹음 |
| 적용성 | spec `applicableChains` | `"wbft,stablenet"` | 미적용 체인은 SKIP |
| 기능 요구 | spec `requires` | `["rpc","ws","process"]` | 미충족은 SKIP(fail 아님) |

### 2.2 바이너리 (단계 2)

| 변곡점 | 설정 위치 | 비고 |
|---|---|---|
| 노드 바이너리 | `--binary <path>` (없으면 매니페스트 `binary` 를 PATH 에서) | **주의**: go-wbft 의 `make gwemix` 산출물 이름은 `gwemix` 인데 매니페스트는 `gwbft` → **`--binary` 명시 필수** |
| 혼합 바이너리 | spec `chain.binaries` / 핸드오프 `--from-binary`/`--to-binary` | 노드별 다른 바이너리(type-1 업그레이드) |
| 포크 전 교체 | `supervisor.Options.ForkSwaps` | type-2 하드포크. **구현체 미배선**(선언 시 오류) |

### 2.3 토폴로지 · 역할 (단계 4)

| 변곡점 | 설정 위치 | 값 |
|---|---|---|
| 개수 | `--validators` / `--endpoints`, spec `topology.bp`/`topology.en` | 정수 |
| 명시 배치 | `setup --topology <yaml>` | 노드별 `{index, role, sync_mode, bootnode}` |
| 역할 | `bp`(=validator) \| `en`(=endpoint) \| `boot` | 도메인 어휘는 `bp`/`en` |
| 저장 방식 | `sync_mode` | `full`(기본) \| `snap` \| `archive` |
| 용량 하한 | `place.Capacity.MinValidators` | BFT 진행 최소; 핸드오프는 **검증자 ≥ 4** 강제 |

### 2.4 키 · 신원 (단계 3)

| 변곡점 | 설정 위치 | 비고 |
|---|---|---|
| 프리셋 경로 | `--keys` / `--keys-dir` (기본 `keys/preset`) | 5노드 커밋본 |
| 프리셋 생성 | `chainbench validator set --nodes N --validators V --bootnode <path> --binary <path> --out <dir>` | 5노드 초과 네트워크용 |
| 계정 잔액 | `validator set --balance <0x-hex>` | genesis alloc |
| 키스토어 암호 | `validator set --password` (기본 `1`) | |
| **중요 제약** | wbft 계열 검증자셋·BLS 는 **프리셋에 baked** | 랜덤 keyreg 키만으로는 유효 genesis 불가(T4.4b) |

### 2.5 genesis (단계 5) — 4모드

| 모드 | 어떻게 | 쓰는 곳 |
|---|---|---|
| ① existing | 기존 genesis 파일 그대로 | 외부 체인 재현 |
| ② build | 플러그인 템플릿 + 프리셋 치환(`genesis.Build`) | 기본 경로 |
| ③ template+override | ② + `--set genesis.overrides.<key>=<v>` / `--genesis-overlay <json>` | 하드포크 블록 조정, 계정 Extra 비트 |
| ④ upgrade-inherit | from-chain genesis 를 받아 fork 섹션만 병합 | 핸드오프(케이스 2) |

- `--set genesis.overrides.bohoBlock=10` → 지연 하드포크
- `--genesis-overlay '{"capabilities":[...],"genesis":{...}}'` → 깊은 병합 + capability 광고
- spec `chain.genesisOverlay` / `hardforks` → DSL 에서 동일 효과

### 2.6 배치 · 포트 (단계 4)

| 변곡점 | 값 | 비고 |
|---|---|---|
| 모드 | `LocalStepped` \| `LocalOSAssigned` \| `RemotePerHost` | 엔진 기본은 Stepped; 설계 권고 기본은 OS 할당(이중바인드 근절) |
| 포트 기준 | `ports.base_p2p`/`step`, `ports.base_rpc`/`step` | **step 제약**: p2p_step ≥ 2(etcd=p2p+1 예약), rpc_step ≥ 3(ws=http+1, auth=http+2) |
| 원격 | `RemotePerHost` = 동일 포트 + 서버별 IP | 1서버 1노드 |

### 2.7 실행 위치 (단계 7·9)

| 변곡점 | 설정 위치 |
|---|---|
| 로컬 | 기본 |
| 원격 SSH | `setup --remote-host --remote-user --remote-port`, `CHAINBENCH_REMOTE_PASS` |
| SSH 키인증 | `CHAINBENCH_REMOTE_KEY_FILE` / `_PASSPHRASE` (0600 강제) |
| 접속정보 파일 | `remote-server-config.yaml` (**gitignore**, `.sample` 만 추적) |

### 2.8 헬스 게이트 (단계 10)

| 변곡점 | 설정 위치 | 상태 |
|---|---|---|
| 블록 전진 | `engine.NewBlockAdvanceGate(target, timeout)` | ✅ 배선됨 |
| etcd 리더 | `supervisor.Options.LeaderGate` + `Deps.LeaderGate` | ⚠️ **seam 만 존재, 구현체 미배선** — 요청 시 오류 |
| 조인 슬롯 정렬 | `Options.AlignJoinGap` → `JoinWindow(N)` | ✅ 데드라인 파생 |
| 재시도 | `Options.MaxAttempts` + `Backoff` | ✅ 재시도 시 datadir 삭제로 stale etcd 정리 |
| 실패 분류 | `supervisor.Classify` | ✅ 5종 방출 |

### 2.9 관측 (단계 9 이후)

| 변곡점 | 설정 위치 |
|---|---|
| 세션 아티팩트 | `--artifact-root` (기본 `chainbench-out`) |
| 라이브 대시보드 | `run --dashboard <chainbench-dashboard URL>` |
| 로그 tail | 로컬 파일 / 원격 `driver.RemoteLogReader`(SSH `tail -c +N`) |
| chainstate | `chainstate/chainstate.jsonl` + obs 미러 |

---

## 3. 사전 준비 — 바이너리 빌드

```sh
CHAIN=/Users/0xtopaz/work/github/0xmhha/chain

(cd $CHAIN/go-stablenet && make gstable)          # -> build/bin/gstable
(cd $CHAIN/go-wbft      && make gwemix)           # -> build/bin/gwemix  (== "gwbft")
(cd $CHAIN/go-wbft      && go build -o build/bin/bootnode ./cmd/bootnode)
(cd $CHAIN/go-wemix     && make gwemix USE_ROCKSDB=NO)  # -> build/bin/gwemix
```

빌드 확인(2026-08-09): gstable `1.1.0-stable`, go-wbft `gwemix`, go-wemix `gwemix` 모두 생성됨.

> **이름 충돌 주의**: go-wbft 와 go-wemix 는 **둘 다 `gwemix` 라는 이름**의 바이너리를 만든다.
> 경로로 구분해야 하며, 이 문서들은 항상 전체 경로로 표기한다.

---

## 4. CLI 로 직접 점검

```sh
chainbench chain cases                      # 4개 케이스와 현재 지원 상태
chainbench chain steps --case wbft          # 그 케이스의 단계 + 변곡점
chainbench chain up    --case wbft --binary <path> --data-dir /tmp/x   # 단계별 실행
chainbench chain up    --case wbft ... --stop-after provision          # 특정 단계까지만
chainbench chain status --data-dir /tmp/x   # 살아있는 네트워크 상태(높이·피어·엔진·etcd)
chainbench chain down   --data-dir /tmp/x   # 종료(고아 0 검증)
```

**케이스별 실행 진입점** — 어디까지 자동화됐는지가 다르다:

| 케이스 | 진입점 | 비고 |
|---|---|---|
| stablenet | `chain up --case stablenet --binary <gstable>` | 전 단계 자동 |
| wbft | `chain up --case wbft --binary <go-wbft/build/bin/gwemix>` | 전 단계 자동. **`--binary` 절대경로 필수**(이름이 `gwemix`) |
| wemix-wbft | `scripts/chain-setup/handoff-wemix-wbft.sh <data-dir>` | `chain up` 은 아직 옛 순서라 실패. 스크립트가 2-페이즈 순서를 수행 |
| wemix | 없음 — [case-1 §5](case-1-wemix.md) 의 수동 절차 | 오케스트레이터 미구현 |

`chain up` 은 **단계마다 PASS/FAIL 과 소요시간을 출력**하고, 실패하면 거기서 멈춘다.
어느 단계가 깨졌는지가 곧 답이 되도록 만든 것이 이 명령의 목적이다.

```
OK    resolve-chain      (0s)      stablenet: family wbft, chain id 8283, bootstrap static
OK    load-preset        (0s)      5 node identities, 4 validators from keys/preset
OK    allocate           (0s)      4 node(s); node1 p2p=30300 http=8600
OK    genesis            (0s)      6273 bytes, 4 validator(s) substituted
OK    provision          (1ms)     genesis + 4 config(s) under /tmp/x
OK    launch             (1.834s)  4 node(s) up; node1 http://127.0.0.1:8600
OK    health-gate        (3.022s)  head 2 on http://127.0.0.1:8600
```

세 가지 결과를 구분한다: **OK** / **FAIL**(구현됐는데 깨짐) / **TODO**(아직 안 만듦).
FAIL 과 TODO 를 나누는 이유는 후속 작업이 서로 다르기 때문이다.

`chain steps --case <id>` 는 단계와 **변곡점 표**를 함께 출력한다 — 이 문서 §2 와 같은 내용이
코드에서 나오므로, 문서와 구현이 조용히 어긋나지 않는다.

---

## 5. 현재 확인된 결함

문서화 과정에서 실측으로 드러난 것.

| # | 결함 | 영향 | 조치 |
|---|---|---|---|
| 1 | **부트스트랩을 전체 기동 상태에서 실행한다** | etcd 클러스터 미형성 → 체인 정지 | §1b 의 2-페이즈 순서로 재구성 필요 (**미구현**) |
| 2 | **후계 검증자에 keystore·`--unlock`·`--miner.etherbase` 가 없다** | 포크 후 인계 실패(라운드만 돎) | static 경로와 동일하게 배치 필요 (**미구현**) |
| 3 | **재기동 후 메시 재연결이 없다** | 검증자끼리 미연결 → 쿼럼 불가 | 페이즈 B1 뒤로 이동 필요 (**미구현**) |
| 4 | **`setup` 이 `governance-etcd` 부트스트랩을 실행하지 않는다** | 케이스 1 자동화 불가 | 매니페스트가 타입만 선언, `setup.Launch` 에 분기 없음 (**미구현**) |
| 5 | `upgrade run` 이 부트스트랩 성공을 검증하지 않는다 | 실패해도 "etcd initialized" 출력 | `chain up` 의 **verify-etcd 로 해소**. `upgrade run` 은 그대로 |
| 6 | `admin.etcdIsReady` 가 이 go-wemix 빌드에 없다 | 설계 §3.3 의 리더게이트 프로브 이름이 틀림 | `admin.wemixInfo.etcd.cluster` 로 대체 확정 |
| 7 | golden 프로파일 `fork_block: 20` 이 너무 이르다 | 프로듀서가 포크에서 멈춰 거버넌스 배포 영수증도 못 받음 | 참조값 **100** 으로 (**미구현**) |
| 8 | `upgrade run` 에 재시도 없음 | e2e 헬퍼에만 있음 | (**미구현**) |

1·2·3·7 이 케이스 2를 막던 실제 원인이고, 넷 다 chainbench 쪽 문제다.
6 만 go-wemix 와의 계약 차이이며, 2·3 은 static 경로가 이미 올바르게 하는 것을 핸드오프 경로가
빠뜨린 것이다.

- [`cli-steps.md`](cli-steps.md) — **CLI 기준 3체인 구동 절차 대조.** 공통 9스텝 · 체인별 함정 · wemix 갭 6개와 그것이 `net genesis`/`net start` 두 스텝으로 흡수되는 방식.
