# 인수인계 — governance-etcd 부트스트랩 자동화

> 새 세션이 이 문서만 읽고 이어서 작업할 수 있도록 정리한 컨텍스트다.
> 작성: 2026-08-10 · 기준 커밋: `2082f94` (main)
>
> **한 줄 요약**: wemix 계열 체인을 세우는 *절차*는 실증으로 확정됐다. 그 절차대로
> **자동화하는 코드가 아직 없다.** 그게 남은 일 전부다.

---

## 1. 왜 이 일이 남았나

사용자 요구는 네 가지 방식으로 체인을 구성하고 **동작하게** 만드는 것이었다.

| 케이스 | 요구 | 현재 |
|---|---|---|
| 2.1 gwemix 단독 | 구성 + 동작 | ⚠️ 절차 확정, **자동화 없음** |
| 2.2 gwemix→gwbft 하드포크 | 포크 후 gwbft가 블록 생성 | ⚠️ 절차 실증 완료, **자동화는 옛 순서라 실패** |
| 2.3 gwbft 단독 | 구성 + 동작 | ✅ `chain up --case wbft` |
| 2.4 gstable 단독 | 구성 + 동작 | ✅ `chain up --case stablenet` |

지금까지 한 것은 **문서화와 진단 도구**(#227, #229)이고, 2.1·2.2를 실제로 돌아가게 만드는
코드는 아직 쓰지 않았다. 수동 스크립트(`scripts/chain-setup/handoff-wemix-wbft.sh`)로
2.2가 동작하는 것은 확인했다.

---

## 2. 확정된 사실 — 다시 알아내지 말 것

이 절의 내용은 실 바이너리 실험으로 확인한 것이다. 재발견 비용이 크니 그대로 신뢰해도 된다.

### 2.1 부트스트랩은 2-페이즈다

```
── 페이즈 A: 프로듀서 단독 ──────────────
 A1  genesis + 전 노드 datadir init
 A2  프로듀서 1대만 기동        ← 다른 노드가 하나라도 떠 있으면 실패한다
 A3  deploy-governance  (IPC)
 A4  admin.etcdInit()   (IPC)
 A5  admin.wemixInfo.etcd.cluster 가 비어 있지 않은지 확인
 A6  프로듀서 종료
── 페이즈 B: 전체 ──────────────────────
 B1  전 노드 기동
 B2  풀메시 재연결 (admin_addPeer)   ← 새 프로세스이므로 반드시 다시
 B3  블록 전진 확인
```

근거: 참조 구현 `stablenet/packages/chainbench/tests/wemix4/envs/default/bootstrap.sh`
+ 실 바이너리 재현.

### 2.2 실패 원인 3가지 (각각 변수 하나만 바꿔 분리 확인)

| # | 원인 | 증상 |
|---|---|---|
| 1 | 전체 기동 상태에서 `etcdInit` | `etcd.cluster` 가 계속 `""`, 로그 `etcd join failed: not found` |
| 2 | 후계 검증자에 keystore·`--unlock`·`--miner.etherbase` 없음 | 포크 블록에서 `Commit new sealing work number=<fork>` 무한 반복 |
| 3 | 메시가 최종 기동보다 먼저 | 검증자 `peers=1`, `currentRoundChanges.count=1`(자기 것만) |

**2·3 은 static 경로(`engine.armSpecs`, `internal/engine/launcher.go:195`)가 이미 올바르게
하는 것**을 핸드오프 경로가 빠뜨린 것이다. 새로 설계할 필요 없이 그 코드를 따라가면 된다.

### 2.3 성공했을 때의 관측값

```jsonc
// A5 (프로듀서 단독)
{"cluster":"producer=https://127.0.0.1:30011",
 "leader":{"id":"31f7d7811eac700f","name":"producer"},
 "members":[{"name":"producer","peerUrls":"https://127.0.0.1:30011"}]}
miners: "producer/up/*"        // '*' 가 리더 표식

// B3 (포크 인계, fork_block=100)
block  99 (0x63)  miner = 0xf9593d…   go-wemix 프로듀서 — 포크 직전 마지막
block 100 (0x64)  miner = 0xc17d49…   go-wbft 검증자 (인계)
block 101 (0x65)  miner = 0x2493a8…   go-wbft 검증자 (라운드로빈)
```

### 2.4 함정 — 앞서 잘못 판단했던 것들

| 오판 | 사실 |
|---|---|
| `admin.etcdInit()` 이 `null` 을 반환하니 실패다 | **성공해도 `null`** 이다. 반환값은 판정 근거가 아니며 `admin.wemixInfo.etcd.cluster` 만이 증거다 |
| 포크 블록 20 이 너무 일러서 실패한다 | 100 으로 바꿔도 **동일하게 실패**했다. 원인이 아니다. 다만 20 은 별개 문제가 있다 — 1초 블록이면 20초 만에 프로듀서가 포크에서 멈춰 거버넌스 배포가 영수증조차 못 받는다(`context deadline exceeded`). 참조값 100 을 쓸 것 |
| 설계 §3.3 의 리더게이트 프로브 `admin.etcdIsReady` | **이 go-wemix 빌드에 없다**(`TypeError: Object has no member`). `admin.wemixInfo.etcd.cluster` 로 대체 확정 |

### 2.5 환경 특이사항

| 항목 | 내용 |
|---|---|
| 바이너리 이름 충돌 | go-wbft 와 go-wemix **둘 다 `gwemix`** 를 만든다. 경로로만 구분되며, wbft 매니페스트의 `binary` 는 `gwbft` 라 PATH 조회로는 못 찾는다 → `--binary` 절대경로 필수 |
| genesis 템플릿 | `gwemix wemix genesis` 에는 **go-wemix 저장소의** `wemix/scripts/genesis-template.json` 을 줘야 한다. chainbench 내장 `internal/chains/wemix/genesis.json` 은 `__CHAIN_ID__` 치환용이라 넣으면 파싱 실패 |
| IPC 경로 길이 | geth IPC 유닉스 소켓은 **104자 제한** → data dir 를 짧게(`/tmp/handoff`) |
| 셸 | 이 머신 기본 셸은 **zsh**. `nohup $CMD` 는 동작하지 않는다(zsh 는 `$var` 를 단어 분리하지 않음). `nohup $(cat file)` 또는 `${=CMD}` 를 쓸 것 |
| 바이너리 빌드 | `make gstable` / `make gwemix`(go-wbft) / `make gwemix USE_ROCKSDB=NO`(go-wemix). `CHAIN_DIR` 기본값은 `~/work/github/0xmhha/chain` |

### 2.6 검증자 신원 매핑 (핸드오프)

프로파일 `identities.plan_order: [5,1,2,3,4]` 가 플랜 순서 ↔ 프리셋 노드를 잇는다.

| 플랜 노드 | 프리셋 노드 | 주소 | 역할 | http |
|---|---|---|---|---|
| node1 | preset node5 | `0xf9593d…6984` | 프로듀서(go-wemix) | 40010 |
| node2 | preset node1 | `0xc17d49…f9d8` | 검증자(go-wbft) | 40020 |
| node3 | preset node2 | `0x2493a8…8d3c` | 검증자 | 40030 |
| node4 | preset node3 | `0x8c4a10…7764` | 검증자 | 40040 |
| node5 | preset node4 | `0x8eb790…39a6` | 검증자 | 40050 |

---

## 3. 남은 작업

### A. 케이스 2 자동화 — `chain up --case wemix-wbft` 를 2-페이즈로 (우선)

| # | 작업 | 손댈 곳 |
|---|---|---|
| A1 | 단계 목록을 2-페이즈 순서로 재정의 | `internal/chainsetup/cases.go` → `handoffSteps()` |
| A2 | 실행기를 새 순서로 재배열 | `internal/chainsetup/handoff.go` → `RunHandoff` |
| A3 | 프로듀서 단독 기동 / 전체 기동을 분리 | `internal/chainsetup/handoff_driver.go` → `liveHandoff.Launch`(:151) 를 `LaunchProducer` + `LaunchAll` 로 |
| A4 | 검증자 keystore 배치 + `--unlock`/`--password`/`--miner.etherbase` | `liveHandoff.provisionKeys`(:219) 와 `extraArgs`. **`engine.armSpecs`(`internal/engine/launcher.go:195`) 가 하는 그대로** |
| A5 | 메시 연결을 최종 기동 뒤로 | `liveHandoff.WireMesh`(:168) 호출 위치 |
| A6 | 프로듀서 종료 단계 추가(A6) | 신규. `procman.StopOne` 재사용 |
| A7 | 프로파일 `fork_block: 20` → `100` | `profiles/wemix-upgrade.yaml` |

**완료 기준**: `chain up --case wemix-wbft` 가 **전 단계 OK** 로 끝나고, 포크 블록의 `miner` 가
검증자 주소일 것. 수동 스크립트와 동일 결과가 나오면 된다.

> 주의: `handoffSteps()` 의 `Implemented` 플래그와 `RunHandoff` 의 실제 구현이 어긋나면
> `TestCases_AreWellFormed` 가 잡는다(Supported 인 케이스에 미구현 단계가 있으면 실패).

### B. 케이스 1 자동화 — `chain up --case wemix`

A 가 끝난 뒤. 같은 2-페이즈를 쓰되 후계 체인이 없다.

| # | 작업 |
|---|---|
| B1 | wemix 전용 로직을 CLI 계층에서 `internal/consensus/poa` 오케스트레이터로 승격 (config 조립·키 배치·IPC 대기·2-페이즈·verify) |
| B2 | `setup` 이 `bootstrap.type == "governance-etcd"` 를 보고 그 오케스트레이터로 분기 (현재 분기 자체가 없다) |
| B3 | `RunWemix`(`internal/chainsetup/wemix.go`) 를 실제 구현으로 — 지금은 3단계만 하고 나머지는 `NotImplemented` 를 정직하게 보고한다 |
| B4 | standalone 프로파일(`profiles/wemix-standalone.yaml`) — 포크 없는 설정 |
| B5 | BP 2대 이상이면 각 BP 를 `poa.Member` 로 등록해야 한다(미등록 노드는 etcd 조인 불가) |

**완료 기준**: `chain up --case wemix` 가 전 단계 OK, 블록이 계속 전진.

### C. supervisor 잔여 (T3.2b)

`Deps.LeaderGate` 는 seam 만 있고 구현체가 없다. §2.4대로 프로브를 `admin.wemixInfo.etcd.cluster`
로 하여 배선하면 `Options.AlignJoinGap`(`JoinWindow(N)`) 도 비로소 의미를 갖는다.
`Deps.SwapBinary`(type-2 포크)도 미배선 — 선언 시 오류를 내도록만 되어 있다.

### D. 병행 트랙 — DSL 이관 (별개, 막힌 것 없음)

레거시 134개 중 18개 이관 완료(#228). 남은 106개: `system-contracts`(46) · `accounts`(35) ·
`gas-policy`(17) · `hardfork`(8). 표현력 블로커는 #225 에서 모두 해소됐고 작업량만 남았다.
상세: `tests/specs/README.md`, `docs/dev/legacy-retirement-plan.md` §4.4.

이관 불가로 남긴 4건(순서·산술·조건부대기·토폴로지 참조)도 같은 문서에 이유와 함께 있다.

---

## 4. 지금 바로 쓸 수 있는 것

```sh
CHAIN=~/work/github/0xmhha/chain

# 동작하는 두 케이스
chainbench chain up --case stablenet --binary $CHAIN/go-stablenet/build/bin/gstable --data-dir /tmp/s
chainbench chain up --case wbft      --binary $CHAIN/go-wbft/build/bin/gwemix       --data-dir /tmp/w

# 핸드오프 — 수동 스크립트가 유일한 수단 (chain up 은 아직 실패한다)
CHAIN_DIR=$CHAIN scripts/chain-setup/handoff-wemix-wbft.sh /tmp/handoff

# 진단
chainbench chain cases                     # 케이스별 지원 상태
chainbench chain steps --case wemix-wbft   # 단계 + 변곡점
chainbench chain status --data-dir /tmp/x
chainbench chain down   --data-dir /tmp/x
```

푸시 전 CI 게이트 4개를 로컬에서 그대로 돌릴 것 — `golangci-lint` 는 `go vet` 이 잡지 않는
`unused` 를 잡는다(한 번 놓쳐 CI 를 깨뜨린 적 있다):

```sh
gofmt -l cmd internal tests   # 빈 출력이어야 함
go vet ./... && go build ./...
go test -race ./...
golangci-lint run
```

---

## 5. 관련 문서

| 문서 | 내용 |
|---|---|
| `docs/dev/chain-setup/README.md` | 공통 파이프라인, **§1b 2-페이즈 계약**, 변곡점 카탈로그, 확인된 결함 8건 |
| `docs/dev/chain-setup/case-1-wemix.md` | wemix 단독 절차(도출), 재료 현황, 자동화 잔여 |
| `docs/dev/chain-setup/case-2-wemix-to-wbft.md` | 핸드오프 절차(실증), 원인 3건, 배제된 가설 |
| `docs/dev/chain-setup/case-3-wbft.md`, `case-4-stablenet.md` | 동작하는 두 케이스 |
| `scripts/chain-setup/handoff-wemix-wbft.sh` | 성공한 순서 그대로의 실행 스크립트 |
| `docs/dev/chainbench-worklist.md` | 전체 작업 트래커(T0~T6). 진행 상황의 **단일 정본** |
| `tests/specs/README.md` | DSL 이관 현황·규약·이관 불가 사유 |

참조 구현(외부, 읽기 전용):
`/Users/0xtopaz/work/github/stablenet/packages/chainbench/tests/wemix4/`
— `envs/default/bootstrap.sh`(순서), `lib/init_wemix_gov.sh`(단독 부트스트랩),
`envs/default/node_env.json`(토폴로지), `genesis/genesis_main_test.md`(genesis 설정),
`env.conf.sample`(포크·에폭·검증자 파라미터). 원격 `deploy_gov.sh` 는 이 트리에 없다.
