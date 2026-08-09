# 케이스 1 — gwemix 단독 체인 구성

> 목표: `gwemix` 만으로 wemix(wpoa) 체인을 세우고 블록을 만들게 한다.
> 상태: ❌ **미지원** — 프레임워크에 standalone 부트스트랩 경로가 없다. 아래는 *무엇이 필요한지*의 명세다.
> 공통 절차·변곡점은 [README.md](README.md) 참조.

---

## 1. 전제

| 항목 | 값 |
|---|---|
| 바이너리 | `<chain>/go-wemix/build/bin/gwemix` (`make gwemix USE_ROCKSDB=NO`) |
| 부트스트랩 | **`governance-etcd`** — 다른 세 케이스와 근본적으로 다르다 |
| 합의 family | `poa` (wpoa) |
| chain id | 8285 |
| RPC 네임스페이스 | `wemix` |
| etcd | **바이너리 내장** — 별도 프로세스 없음, 포트는 `p2p+1` 로 파생 |

---

## 2. 왜 다른가 — genesis 에 검증자가 없다

static 체인(wbft·stablenet)은 genesis 가 검증자셋을 담고 있어 **기동 즉시** 합의가 성립한다.
wemix 는 그렇지 않다:

```
gwemix wemix genesis     → 검증자 없는 genesis (거버넌스 컨트랙트도 alloc 에 없음)
       ↓ init + launch
    노드는 뜨지만 블록 생성 주체가 없음   ← 여기서 멈추면 "떠 있지만 죽은 체인"
       ↓ deploy-governance (IPC)
    GovRegistry/GovStaking/Governance 배포 + 멤버 등록
       ↓ admin.etcdInit() (IPC)
    etcd 클러스터 형성 → 리더 선출
       ↓
    비로소 블록 생성 시작
```

즉 **기동 후 2개의 IPC 단계**가 성공해야 체인이 산다. 이 두 단계가 케이스 1·2의 핵심이자
현재의 실패 지점이다.

---

## 3. 필요한 절차 (미구현)

| # | 단계 | 현재 상태 |
|---|---|---|
| 1 | resolve-chain (`wemix`) | ✅ 매니페스트 존재, `bootstrap.type="governance-etcd"` 선언됨 |
| 2 | resolve-binary | ✅ |
| 3 | load-preset | ✅ |
| 4 | **wemix-config 작성** (`poa.Config`: 멤버·스테이크·거버넌스 env·alloc) | ⚠️ 타입은 있으나 **핸드오프 프로파일에서만 조립**(`buildPoAConfig`) — standalone 입력원 없음 |
| 5 | **base-genesis 생성** (`gwemix wemix genesis --data <cfg> --genesis <template> --out`) | ✅ `poa.GenerateGenesis` 존재. **템플릿은 go-wemix 저장소의 `wemix/scripts/genesis-template.json`** (chainbench 내장 `internal/chains/wemix/genesis.json` 은 `__CHAIN_ID__` 치환용이라 이 명령에 넣으면 실패) |
| 6 | allocate / plan | ✅ 단, `place` 는 etcd 포트를 `p2p+1` 로 예약만 함 |
| 7 | provision keys (nodekey → `geth/`, static-nodes, keystore) | ⚠️ `provisionKeysFn` 이 **핸드오프 CLI 안에 하드코딩** — 재사용 불가 |
| 8 | init + launch (`--mine --unlock --miner.etherbase`) | ✅ |
| 9 | wait-ipc | ⚠️ 핸드오프 CLI 내부 헬퍼 |
| 10 | **deploy-governance** | ✅ `poa.DeployGovernance` (2-인자 형태 — 3-인자는 gwemix 버그) |
| 11 | **etcd-init** | ✅ `poa.EtcdInit` **단, 결과 미검증** |
| 12 | **verify-etcd** | ❌ **없음** — 이 단계가 없어서 실패가 성공으로 보고된다 |
| 13 | health-gate (블록 전진) | ✅ 일반 게이트만; etcd 리더 게이트는 seam 만 존재 |

**결론:** 재료(4·5·10·11)는 있는데 **standalone 오케스트레이터가 없다**. 지금 이들을 호출하는
유일한 경로는 `chainbench upgrade run`(케이스 2)뿐이고, 그 안에 wemix 전용 로직이 CLI 계층에
묶여 있다.

---

## 4. 변곡점 (구현 시 노출해야 할 것)

| 변곡점 | 어디에 | 값 |
|---|---|---|
| 거버넌스 env | `poa.Env` | `ballot_duration_{min,max}`, `staking_{min,max}`, `max_idle_block_interval`, `block_creation_time`, `block_reward_amount`, `max_priority_fee_per_gas`, `reward_distribution[4]`, `max_base_fee`, `block_gas_limit`, `base_fee_max_change_rate`, `gas_target_percentage` |
| 멤버(생산자) | `poa.Member` | `addr`(unlock 가능 계정), `stake`, `id`(devp2p 공개키), `ip`, `port`, `bootnode` |
| 역할 계정 | `poa.Config` | `staker`, `ecosystem`, `maintenance`, `feeCollector` |
| alloc | `poa.Account[]` | 멤버 + 필요한 계정의 초기 잔액 |
| genesis 템플릿 | `--template` | **go-wemix 저장소의 것**(§3-5) |
| 하드포크 | 템플릿 config | `istanbul`, `pangyo`, `applepie`, `brioche` |
| etcd | (파생) | 포트 `p2p+1`, 클러스터 토큰은 바이너리 내장 |
| 조인 슬롯 | `supervisor.JoinGap(N)` | ≤11→7s, ≤23→11s, ≤41→17s, else 23s |

프로파일 예시는 `profiles/wemix-upgrade.yaml` 의 `producers.governance` 블록이 그대로 참고가 된다.

---

## 5. 현재 확인된 차단 사유

케이스 2 실행에서 관측한 것이 그대로 적용된다(케이스 1은 그 부분집합):

```
# admin.wemixInfo — 거버넌스는 배포됐다
governance: 0x0caff82e8d4fc5c2e0eb6d01739169a3912c1286
registry:   0xb803d9c283624c4a7f8a63c6df4381976fd5b300
staking:    0xd41db91511476705f4d371482c7f107d85b725f8
nodes:      [{name:"producer", addr:"0xf9593d…", miner:false}]
miners:     "producer/up"

# 그런데 etcd 는 비어 있다
etcd: {"cluster":"", "members":null}

# admin.etcdInit() 반환값
null

# 프로듀서 로그
ERROR etcd failed to start  error="cannot fetch cluster info from peer urls: …"   (×30)
INFO  etcd join failed      name=producer error="not found"
```

`etcdInit` 이 클러스터를 만들지 못하고 **조인 모드로 떨어진다**. 결과적으로 블록 생성이 멈춘다.

추가 관측:
- `admin.etcdIsReady` 는 **이 go-wemix 빌드에 존재하지 않는다** (`TypeError: Object has no member`).
  설계 §3.3 이 리더게이트 프로브로 지목한 이름이므로, 구현 시 `admin.wemixInfo.etcd.cluster` 로 대체해야 한다.
- `self.miner == false` — 멤버로는 등록됐으나 마이너로 승격되지 않았다. etcdInit 실패와 인과가 얽혀 있다.

---

## 6. 착수 순서 (제안)

1. **`chain up --case wemix` 에 verify-etcd 단계 추가** — 실패를 실패로 보이게 한다(가장 싸고 효과 큼).
2. wemix 전용 로직을 CLI 에서 **`internal/consensus/poa` 오케스트레이터로 승격** — config 조립·키 배치·IPC 대기·2단계 부트스트랩.
3. `supervisor.Deps.LeaderGate` 구현체를 `admin.wemixInfo.etcd.cluster` 기준으로 배선(T3.2b 잔여).
4. 그 위에서 standalone 프로파일(`profiles/wemix-standalone.yaml`) 정의.
5. `etcdInit` 이 null 을 반환하는 원인 규명 — 이는 chainbench 가 아니라 **go-wemix 와의 계약** 문제이므로, 위 1번이 만들어 줄 증거 위에서 판단한다.
