# 케이스 2 — gwemix → gwbft 하드포크 핸드오프

> 목표: gwemix(wpoa)가 포크 블록까지 블록을 만들고, 그 이후로는 **같은 체인**을 gwbft 검증자들이
> 이어서 생성한다.
> 상태: ✅ **절차 검증 완료** — 2026-08-09 실 바이너리로 블록 100 인계 확인(§5). 단, **`chain up` 자동화는 아직 옛 순서**라 그대로 돌리면 실패한다(§6).
> 공통 절차·변곡점은 [README.md](README.md) 참조.

---

## 1. 전제

| 항목 | 값 |
|---|---|
| from 바이너리 | `<chain>/go-wemix/build/bin/gwemix` (프로듀서, 포크 전) |
| to 바이너리 | `<chain>/go-wbft/build/bin/gwemix` (검증자, 포크 후) |
| genesis 템플릿 | `<chain>/go-wemix/wemix/scripts/genesis-template.json` |
| 프로파일 | `profiles/wemix-upgrade.yaml` (golden) |
| 포크 | `croissant` @ block 20 |
| 역할 | 프로듀서 1 + 검증자 4 (**검증자 ≥ 4 강제**) |
| network id | 8285 (**전 노드 균일**) |

> **type-1 업그레이드**다: 프로듀서와 후계자가 **처음부터 서로 다른 바이너리로 동시에** 뜬다.
> 바이너리를 중간에 바꾸는 type-2(같은 체인, fork 전 교체)는 `supervisor.ForkSwaps` 소관이며 미배선이다.

---

## 2. 절차

| # | 단계 | 하는 일 | 상태 |
|---|---|---|---|
| 1 | load-profile | `profiles/wemix-upgrade.yaml` → 역할·포크·포트·거버넌스 env·검증자셋 | ✅ |
| 2 | load-preset | `keys/preset`; `identities.plan_order: [5,1,2,3,4]` 로 플랜 순서↔프리셋 노드 매핑 | ✅ |
| 3 | wemix-config | 프로듀서 멤버 1 + 거버넌스 env + alloc → `<data>/wemix-config.json` | ✅ |
| 4 | base-genesis | `gwemix wemix genesis --data <cfg> --genesis <template> --out <base>` | ✅ |
| 5 | fork-prereqs | base 에 `chainId`, `petersburgBlock=0` 주입 | ✅ |
| 6 | build-plan | to-chain genesis 를 만들어 **`croissant` 섹션을 데이터로 추출** → base 에 병합 + `croissantBlock` 설정 → preflight | ✅ |
| 7 | genesis-overlay *(선택)* | `--genesis-overlay` 로 `croissant.wBFT.useNCP` 등 깊은 병합 | ✅ |
| 8 | provision-keys | 노드별 nodekey 를 **바이너리별 instance 디렉터리**에 배치(from=`geth/`, to=`gwemix/`), static-nodes, 프로듀서 keystore 복사 | ✅ |
| 9 | init+launch | 노드별 **자기 바이너리로** init 후 동시 기동 | ✅ |
| 10 | wait-ready | 각 노드 HTTP 준비 대기 | ✅ |
| 11 | wire-mesh | `admin_addPeer` 로 N×N 메시 | ✅ |
| 12 | **deploy-governance** | 프로듀서 IPC 로 거버넌스 배포(2-인자 형태) | ✅ |
| 13 | **etcd-init** | `attach --exec admin.etcdInit()` | ⚠️ **호출은 되나 클러스터 미형성** |
| 14 | **verify-etcd** | `admin.wemixInfo.etcd.cluster` 가 비었는지 폴링 확인 | ✅ **신규**(`chain up` 경로). 기존 `upgrade run` 에는 없어 13의 실패가 성공으로 보고됐다 |
| 15 | await-handoff | 검증자에서 head > fork, `block[fork+1].miner != 프로듀서` 확인 | ✅ 로직은 있음 |

**포크 섹션이 상수가 아닌 이유(설계 원칙):** `croissant` 설정은 이 패키지에 하드코딩되지 않는다.
to-chain 의 자기 genesis 템플릿에서 **데이터로 추출**해 from-chain genesis 에 병합한다
(`genesis.ExtractConfigSection` → `SetConfigSection`). 그래서 체인 쌍이 바뀌어도 코드는 그대로다.

---

## 3. 이 케이스의 변곡점

프로파일(`profiles/wemix-upgrade.yaml`)이 사실상 전체 변곡점 표다.

| 변곡점 | 프로파일 키 | 비고 |
|---|---|---|
| 체인 쌍 | `upgrade.from` / `upgrade.to` | from 매니페스트의 `upgrade.to_chain` 과 일치해야 함 |
| 포크 | `upgrade.at_fork` / `upgrade.fork_block` | golden 은 `croissant` @ 20 |
| network id | `upgrade.network_id` | **균일 필수** — go-wemix 는 chain id 와 독립적으로 자기 id 를 기본값으로 쓴다 |
| 역할 수 | `roles.producers` / `roles.validators` | 검증자 **≥ 4** (BFT 3f+1) |
| 키 매핑 | `identities.plan_order` | 플랜은 프로듀서 우선, 프리셋은 node5 가 프로듀서 |
| 프로듀서 계정 | `producers.members[]` | **검증자와 disjoint 여야 함** — 겹치면 etcd 조인 실패로 프로듀서가 정지 |
| 스테이크 | `producers.stake` | |
| 거버넌스 env | `producers.governance.*` | 케이스 1 §4 와 동일 항목 |
| 검증자셋 | `validators.addresses` + `bls_public_keys` + `extra_data` | 정렬된 목록 |
| 카운슬 | `validators.members` | anzeon/거버넌스 카운슬 |
| 데이터 루트 | `data.directory` | |
| 포트 | `ports.base_p2p`/`step_p2p`/`base_rpc`/`step_rpc` | step 제약(README §2.6) |
| 노드 옵션 | `nodes.verbosity`/`gcmode`/`cache` | |
| genesis 오버레이 | CLI `--genesis-overlay` | `useNCP`/`targetValidators`/`stabilizingStakersThreshold`, `govNCP.params.ncps` |

### 프로파일이 인코딩한 "한 번씩 조용히 깨졌던" 조건

1. network id 가 전 노드 균일
2. 프로듀서와 검증자가 disjoint
3. 검증자 ≥ 4
4. `petersburgBlock` 존재 + `croissant` 섹션/블록 쌍 (`genesis.ValidateForks`·preflight 가 강제)

---

## 4. 실행

### 4.1 점검용 CLI (단계별)

```sh
CHAIN=/Users/0xtopaz/work/github/0xmhha/chain
chainbench chain up --case wemix-wbft \
  --profile profiles/wemix-upgrade.yaml \
  --from-binary $CHAIN/go-wemix/build/bin/gwemix \
  --to-binary   $CHAIN/go-wbft/build/bin/gwemix \
  --template    $CHAIN/go-wemix/wemix/scripts/genesis-template.json \
  --data-dir /tmp/cb-handoff
```

실측 출력(2026-08-09) — **`verify-etcd` 가 원인을 지목한다**:

```
OK    build-plan         (0s)     5 node(s); fork section "croissant" merged; preflight passed
OK    launch             (1.758s) 5 node(s); producer http://127.0.0.1:40010
OK    wire-mesh          (561ms)  5 endpoint(s) meshed
OK    deploy-governance  (5.729s) deploy-governance returned success (effect is checked by verify-etcd)
OK    etcd-init          (92ms)   admin.etcdInit() returned without error (effect is checked by verify-etcd)
FAIL  verify-etcd        (1m31s)  governance is deployed (0x0caff82e…) but the etcd cluster stayed
                                  empty for 1m30s — admin.etcdInit() did not form it; the producer
                                  will stall before the fork (self.miner=false, miners="producer/up")
SKIP  await-fork
```

`--etcd-timeout` 으로 대기창을 조절한다. 이 단계는 **폴링**한다 — etcdInit 직후의 첫 읽기는
`method handler crashed` 로 실패할 수 있어(노드가 거버넌스 상태를 아직 배선 중), 한 번 읽고
판정하면 일시적 현상을 결함으로 보고하게 된다.

### 4.2 기존 CLI (단일 명령)

```sh
chainbench upgrade run \
  --profile profiles/wemix-upgrade.yaml --preset keys/preset \
  --from-binary $CHAIN/go-wemix/build/bin/gwemix \
  --to-binary   $CHAIN/go-wbft/build/bin/gwemix \
  --template    $CHAIN/go-wemix/wemix/scripts/genesis-template.json \
  --data-dir /tmp/hand --wait 150
```

### 4.3 상태 확인

```sh
chainbench chain status --data-dir /tmp/cb-handoff     # 높이·피어·엔진·etcd 클러스터
# 또는 직접
$CHAIN/go-wemix/build/bin/gwemix attach /tmp/hand/node1/gwemix.ipc \
  --exec 'JSON.stringify(admin.wemixInfo)'
```

---

## 5. 검증된 절차 (2026-08-09 실 바이너리)

참조 구현 `wemix4/envs/default/bootstrap.sh` 의 순서를 그대로 재현해 **핸드오프 성공**을 확인했다.
공통 부트스트랩 계약은 [README §1b](README.md#1b-governance-etcd-부트스트랩--2-페이즈-순서-실증-확인).

### 5.1 절차 — 실행 가능한 스크립트

순서 자체가 요점이라 산문이 아니라 스크립트로 둔다. **실제로 실행해 성공한 그대로**다.

```sh
CHAIN_DIR=/Users/0xtopaz/work/github/0xmhha/chain \
PROFILE=profiles/wemix-upgrade.yaml \
  scripts/chain-setup/handoff-wemix-wbft.sh /tmp/handoff
```

스크립트가 하는 일:

| 구간 | 내용 |
|---|---|
| 준비 | `chain up --stop-after launch` 로 genesis 생성 + 전 노드 datadir init, **각 노드 실행 인자를 `ps` 로 캡처**, 전부 종료(datadir 유지) |
| 페이즈 A | 프로듀서 1대만 기동 → `deploy-governance` → `admin.etcdInit()` → **`admin.wemixInfo.etcd.cluster` 확인(비면 즉시 중단)** → 프로듀서 종료 |
| 페이즈 B | 검증자에 keystore 복사 + `--unlock`/`--password`/`--miner.etherbase` 부여 → 전체 기동 → `admin_addPeer` 풀메시 → 상태 출력 |

확인은 **검증자에서** 한다 — 프로듀서는 포크 직전에서 멈추는 것이 정상이다.

```sh
curl -s -X POST -H 'Content-Type: application/json' \
  --data '{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["0x64",false]}' \
  http://127.0.0.1:40020        # fork_block=100 → 0x64 의 miner 가 검증자여야 한다
```

> **zsh 주의**: `nohup $CMD` 는 동작하지 않는다(zsh 는 `$var` 를 단어 분리하지 않는다).
> 스크립트는 `nohup $(cat file)` 형태를 쓴다. 직접 옮겨 쓸 때 주의.

### 5.1b 검증자 신원 매핑

프로파일 `identities.plan_order: [5,1,2,3,4]` 가 플랜 순서 ↔ 프리셋 노드를 잇는다.
페이즈 B에서 각 검증자에 넣어야 할 주소가 이것이다.

| 플랜 노드 | 프리셋 노드 | 주소 | 역할 | 포트(http) |
|---|---|---|---|---|
| node1 | preset node5 | `0xf9593d…6984` | 프로듀서(go-wemix) | 40010 |
| node2 | preset node1 | `0xc17d49…f9d8` | 검증자(go-wbft) | 40020 |
| node3 | preset node2 | `0x2493a8…8d3c` | 검증자 | 40030 |
| node4 | preset node3 | `0x8c4a10…7764` | 검증자 | 40040 |
| node5 | preset node4 | `0x8eb790…39a6` | 검증자 | 40050 |

### 5.2 관측 결과

**페이즈 A5 — etcd 클러스터 형성 (단독이라 성공):**

```json
{"cluster":"producer=https://127.0.0.1:30011",
 "leader":{"id":"31f7d7811eac700f","name":"producer"},
 "members":[{"name":"producer","peerUrls":"https://127.0.0.1:30011",
             "clientUrls":"http://localhost:30012"}]}
miners: "producer/up/*"          ← '*' 가 리더 표식
```

**페이즈 B — 포크 인계:**

```
block  99 (0x63)  miner = 0xf9593d…   ← go-wemix 프로듀서, 포크 직전 마지막
block 100 (0x64)  miner = 0xc17d49…   ← go-wbft 검증자 1  (인계)
block 101 (0x65)  miner = 0x2493a8…   ← go-wbft 검증자 2  (라운드로빈)

프로듀서(40010) head = 0x63 에서 정지    ← 정상: 포크를 넘어 생성하지 않는다
검증자(40020)  head = 0x8a 계속 전진
```

---

## 6. 왜 기존 경로는 실패했는가

실패 원인은 셋이고, **전부 chainbench 쪽**이다. 각각 변수 하나만 바꿔 확인했다.

| # | 원인 | 증상 | 확인 방법 |
|---|---|---|---|
| 1 | **전체 기동 상태에서 `etcdInit`** | `etcd.cluster` 가 계속 빈 문자열, 로그에 `etcd join failed: not found` | 단독 기동으로 바꾸니 즉시 클러스터 형성 |
| 2 | **검증자에 keystore·`--unlock`·`--miner.etherbase` 없음** | 포크 블록에서 `Commit new sealing work number=100` 만 반복 | keystore 배치 + unlock 후 봉인 성공 |
| 3 | **재기동 후 메시 미연결** | 검증자 `peers=1`(프로듀서만), `currentRoundChanges.count=1`(자기 것만) | `admin_addPeer` 풀메시 후 `peers=4`, 즉시 포크 통과 |

배제된 가설:

- **포크 블록이 20이라 너무 이르다** — 100으로 바꿔도 동일하게 실패했다. 다만 20은 별개 문제가
  있다: 1초 블록이면 20초 만에 프로듀서가 포크에서 멈춰 거버넌스 배포 영수증조차 못 받는다
  (`context deadline exceeded`). 참조값 100을 쓰는 게 맞다.
- **`admin.etcdInit()` 이 `null` 을 반환한다** — 실패 신호가 아니다. **성공해도 `null`** 이다.
  판정은 반드시 `admin.wemixInfo.etcd.cluster` 로 해야 한다.

2·3 은 **static 경로(`engine.armSpecs`)가 이미 올바르게 하는 것**을 핸드오프 경로가 빠뜨린 것이다.

---

## 7. 자동화에 남은 일

절차는 확정됐고, `chain up --case wemix-wbft` 를 그 순서로 재구성하면 된다.

| # | 작업 | 내용 |
|---|---|---|
| 1 | 2-페이즈 재구성 | init all → 프로듀서 단독 기동 → deploy-governance → etcdInit → verify-etcd → 프로듀서 종료 → 전체 기동 → 메시 → 포크 대기 |
| 2 | 검증자 신원 | keystore 배치 + `--unlock`/`--password`/`--miner.etherbase` (`armSpecs` 와 동일) |
| 3 | 메시 위치 이동 | 최종 기동 뒤로 |
| 4 | 프로파일 | `fork_block: 20` → `100` |
| 5 | supervisor 배선 | `Deps.LeaderGate` 를 `admin.wemixInfo.etcd.cluster` 프로브로 (T3.2b 잔여) |
| 6 | `upgrade run` | **같은 순서가 됐다**(P6.3, 2026-08-28): `chain up --case handoff` 와 `upgrade run` 은 `consensus/upgrade.Handoff` 한 본문 위의 두 표면이다 |
