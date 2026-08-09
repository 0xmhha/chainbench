# 케이스 2 — gwemix → gwbft 하드포크 핸드오프

> 목표: gwemix(wpoa)가 포크 블록까지 블록을 만들고, 그 이후로는 **같은 체인**을 gwbft 검증자들이
> 이어서 생성한다.
> 상태: ⚠️ **부분** — 기동·거버넌스 배포·메시 연결까지 성공. **etcd 클러스터 형성 실패**로 포크 미도달.
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

## 5. 실측 결과 (2026-08-09, 2회 연속 동일)

성공한 것:

```
handoff wemix -> wbft at croissant block 20; 5 nodes; launching...
  node1  http://127.0.0.1:40010  pid=…      ← go-wemix 프로듀서
  node2..5 http://127.0.0.1:400x0 pid=…     ← go-wbft 검증자
governance deployed, etcd initialized, mesh wired.
```

`admin.wemixInfo` 확인 — **거버넌스 배포는 진짜 성공**:

```
governance: 0x0caff8…  registry: 0xb803d9…  staking: 0xd41db9…
nodes: [{name:"producer", addr:"0xf9593d…", enode:"a172e9…", port:30010}]
miners: "producer/up"   modifiedblock: 9
```

실패한 것 — **etcd**:

```
etcd:  {"cluster":"", "members":null}      ← 클러스터가 비어 있음
admin.etcdInit()  →  null                  ← 아무것도 하지 않음
self.miner: false                          ← 마이너로 승격되지 않음

로그: ERROR etcd failed to start error="cannot fetch cluster info from peer urls: …"  (×30)
      INFO  etcd join failed  name=producer error="not found"
```

결과: **프로듀서가 블록 10에서 정지**(`eth_blockNumber` = `0xa`), 포크 블록 20 미도달 →
`handoff not observed within 150s`.

부수 관측: `Unavailable modules in HTTP API list unavailable=[wemix]` — 기동 직후에는 wemix RPC
네임스페이스가 없다(거버넌스 배포 전이므로 예상된 동작이나, 진단 시 혼동하기 쉬움).

---

## 6. 분석

| 관측 | 판정 |
|---|---|
| 거버넌스 배포 성공 | chainbench 경로는 정상 |
| `etcdInit()` → `null`, 클러스터 미형성 | **go-wemix 와의 계약 문제**. 반환값·전제조건이 이 빌드에서 다르다 |
| `upgrade run` 이 "etcd initialized" 출력 | **chainbench 결함** — 결과를 검증하지 않음(README §5-3). `chain up` 은 verify-etcd 로 해소 |
| `admin.etcdIsReady` 부재 | **설계 문서 결함** — 설계 §3.3 이 지목한 프로브가 이 빌드에 없음 |
| 재시도 없음 | **chainbench 결함** — 재시도는 e2e 헬퍼(`runGovHandoff`)에만 있고 CLI 에는 없음 |

`wemix4-port-tracker.md` 는 이 부트스트랩이 "간헐적으로" 실패한다고 기록하지만, **이 환경에서는
2회 연속 재현**되었다. 간헐이 아니라 결정적일 가능성이 있고, 그렇다면 재시도로는 해결되지 않는다.

---

## 7. 착수 순서 (제안)

1. ~~verify-etcd 단계 추가~~ ☑ 완료 — `chain up --case wemix-wbft` 가 클러스터를 폴링 확인하고, 비면 증거와 함께 실패한다.
2. `supervisor.Deps.LeaderGate` 를 같은 프로브로 구현·배선(T3.2b 잔여) → `AlignJoinGap` 이 살아난다.
3. `upgrade run` 에 `MaxAttempts`/`Backoff` 배선 — 재시도 시 datadir 삭제는 supervisor 가 이미 한다.
4. 위 증거를 근거로 `etcdInit` 전제조건을 go-wemix 쪽에서 확인(멤버 상태? 마이너 승격 순서? 블록 높이?).
5. 그 뒤에야 T5.2(신규 엔진에 핸드오프 배선)를 진행한다 — 지금 배선하면 깨진 위에 얹는 셈이다.
