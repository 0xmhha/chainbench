# 케이스 1 — gwemix 단독 체인 구성

> 목표: `gwemix` 만으로 wemix(wpoa) 체인을 세우고 블록을 만들게 한다.
> 상태: ⚠️ **절차 확정, 자동화 미구현** — 부트스트랩 계약이 케이스 2 실증으로 확립됐다(§5).
> 프레임워크에 standalone 오케스트레이터가 없을 뿐, 재료는 모두 있다.
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

## 5. 확정된 절차

케이스 2 실증으로 부트스트랩 계약이 확인됐다([README §1b](README.md#1b-governance-etcd-부트스트랩--2-페이즈-순서-실증-확인)).
wemix 단독은 그 계약에서 **후계 체인만 빼면** 된다 — 포크도, wbft 검증자도 없다.

```
── 페이즈 A: 부트스트랩 (프로듀서 단독) ──
 A1  wemix-config 작성 → gwemix wemix genesis → 전 노드 datadir init
 A2  프로듀서(BP1) 1대만 기동
 A3  gwemix wemix deploy-governance  (IPC)
 A4  admin.etcdInit()                (IPC)
 A5  verify: admin.wemixInfo.etcd.cluster 가 비어 있지 않을 것
 A6  프로듀서 종료
── 페이즈 B: 운영 ──
 B1  전 BP 기동
 B2  풀메시 연결 (admin_addPeer)
 B3  블록 전진 확인
```

케이스 2와의 유일한 차이:

| | 케이스 1 (wemix 단독) | 케이스 2 (핸드오프) |
|---|---|---|
| genesis | croissant 섹션 **없음**, `croissantBlock` 없음 | croissant 섹션 + 포크 블록 병합 |
| 노드 | wemix BP 만 | wemix 프로듀서 + wbft 검증자 |
| 페이즈 B 신원 | BP 는 이미 프로듀서로 unlock 됨 | **검증자에도** keystore + unlock 필요 |
| 종료 조건 | 블록이 계속 전진 | 포크 블록을 후계자가 봉인 |

**BP 를 2대 이상 두려면** 각 BP 를 `poa.Member` 로 등록해야 한다(참조 구현은 `wemix_bp_1`,
`wemix_bp_2` 2대). 등록되지 않은 노드는 etcd 클러스터에 조인하지 못한다.

---

## 6. 관측된 사실 (케이스 2 실증에서)

```
# 프로듀서 단독 + deploy-governance + etcdInit 후
governance: 0x0caff82e8d4fc5c2e0eb6d01739169a3912c1286
registry:   0xb803d9c283624c4a7f8a63c6df4381976fd5b300
staking:    0xd41db91511476705f4d371482c7f107d85b725f8
etcd:       {"cluster":"producer=https://127.0.0.1:30011",
             "leader":{"name":"producer"}, "members":[{"name":"producer",...}]}
miners:     "producer/up/*"
```

주의할 두 가지:

- **`admin.etcdInit()` 은 성공해도 `null` 을 반환한다.** 반환값으로 판정하면 안 되고
  `admin.wemixInfo.etcd.cluster` 를 봐야 한다.
- **`admin.etcdIsReady` 는 이 go-wemix 빌드에 없다**(`TypeError: Object has no member`).
  설계 §3.3 이 리더게이트 프로브로 지목한 이름이므로, 구현 시 위 프로브로 대체한다.

---

## 7. 자동화에 남은 일

재료(§3의 4·5·10·11)는 전부 있고, **이들을 §5 순서로 엮는 오케스트레이터만 없다.**

| # | 작업 |
|---|---|
| 1 | wemix 전용 로직을 핸드오프 CLI 에서 `internal/consensus/poa` 오케스트레이터로 승격 (config 조립·키 배치·IPC 대기·2단계 부트스트랩·verify) |
| 2 | `setup` 이 `bootstrap.type == "governance-etcd"` 를 보고 그 오케스트레이터로 분기 |
| 3 | `chain up --case wemix` 를 §5 순서로 구현 |
| 4 | standalone 프로파일(`profiles/wemix-standalone.yaml`) 정의 — 포크 없는 wemix 설정 |
| 5 | `supervisor.Deps.LeaderGate` 를 `admin.wemixInfo.etcd.cluster` 프로브로 배선 |
