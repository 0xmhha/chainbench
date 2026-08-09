# 케이스 3 — gwbft 단독 체인 구성

> 목표: `gwbft` 만으로 wbft 체인을 세우고 블록을 만들게 한다.
> 상태: ✅ **동작 확인** — 2026-08-09 라이브 실행으로 확인(§5).
> 공통 절차·변곡점은 [README.md](README.md) 참조.

---

## 1. 전제

| 항목 | 값 |
|---|---|
| 바이너리 | `<chain>/go-wbft/build/bin/gwemix` (`make gwemix`) |
| 부트스트랩 | `static` |
| 합의 family | `wbft` |
| chain id | 8284 |
| RPC 네임스페이스 | `istanbul` |
| 프리셋 | `keys/preset` |
| 최소 노드 | 검증자 4 |

> ⚠️ **바이너리 이름 함정**: go-wbft 의 make 타깃은 `gwemix` 라 **산출물 이름도 `gwemix`** 다.
> 그런데 chainbench 매니페스트의 `binary` 는 `gwbft` 이므로 PATH 조회로는 찾지 못한다.
> **`--binary <절대경로>` 를 반드시 준다.** (go-wemix 도 `gwemix` 를 만들므로 경로로만 구분된다.)

---

## 2. 절차

케이스 4(stablenet)와 **동일한 static 파이프라인**이다. 다른 것은 플러그인·템플릿·chain id 뿐.

| # | 단계 | 이 케이스에서 벌어지는 일 |
|---|---|---|
| 1 | resolve-chain | `registry.Get("wbft")` → engine_field `croissant`, hardforks `istanbul…croissant` |
| 2 | resolve-binary | **`--binary <go-wbft/build/bin/gwemix>`** (이름 불일치 때문에 필수) |
| 3 | load-preset | 검증자 4 + BLS/PoP + alloc |
| 4 | allocate | 포트 배치 + 용량 검증 |
| 5 | genesis | 템플릿 `internal/chains/wbft/genesis.json` + 프리셋 치환. 검증자셋은 **`croissant.init.validators`**(genesis config)에 들어간다 |
| 6 | assemble-plan | 노드별 spec |
| 7 | provision | genesis + config |
| 8 | init-datadir | `gwemix init` |
| 9 | launch | `--nodekey`, 검증자 `--unlock`/`--miner.etherbase` |
| 10 | health-gate | 블록 전진 |

**wbft 의 검증자셋 위치(중요):** 헤더 `extraData` 가 아니라 **genesis config 의 `croissant.init.validators`** 다.
그래서 `extraData` 는 평범한 32바이트 vanity 이고 **istanbul RLP 인코딩이 필요 없다** — 이것이
`keys generate` 로 임의 크기 프리셋을 만들 수 있는 이유다(`docs/dev/keys-generate.md`).

---

## 3. 이 케이스의 변곡점

| 변곡점 | 설정 | 예 |
|---|---|---|
| 검증자 수 | `--validators` / `topology.bp` | n=6 쿼럼 케이스는 6노드 프리셋 필요 |
| 프리셋 생성 | `keys generate --nodes 6 --validators 6 --bootnode <go-wbft/build/bin/bootnode> --binary <gwemix> --out /tmp/preset6` | 커밋 프리셋은 5노드가 상한 |
| **useNCP 거버넌스** | `--genesis-overlay` 로 `croissant.wBFT.{useNCP,targetValidators,stabilizingStakersThreshold}` + `govContracts.govNCP.params.ncps` | staking 기반 검증자 선정을 켠다 |
| 하드포크 블록 | `--set genesis.overrides.<fork>Block=N` | |
| 에폭 | genesis `croissant.wBFT.epochLength` (overlay) | 에폭 전이 관측 |
| 저장 방식 | `sync_mode` | `snap` 동기화 케이스 |
| 포트/배치/원격 | README §2.6·2.7 | |

### wbft 고유

| 항목 | 값 |
|---|---|
| 하드포크 | `istanbul`, `pangyo`, `applepie`, `brioche`, `croissant` |
| 검증자셋 소스 | `croissant.init.validators` (genesis config) |
| 기본 검증자 선정 | `useNCP=false` → 순수 staking(`GovStaking` 상위) |
| capability | `process`, `rpc`, `ws`, `consensus` |

---

## 4. 실행

### 4.1 점검용 CLI

```sh
CHAIN=/Users/0xtopaz/work/github/0xmhha/chain
chainbench chain up --case wbft \
  --binary $CHAIN/go-wbft/build/bin/gwemix \
  --data-dir /tmp/cb-wbft
```

### 4.2 DSL 스펙 실행 (실측 확인된 경로)

```sh
cat > /tmp/wbft-smoke.json <<'EOF'
{"schemaVersion":"1","id":"wbft-smoke","chain":{"name":"wbft","binary":"gwbft"},
 "assertions":[{"assert":"chainId","expected":8284},
               {"assert":"blockNumber","compare":"GreaterOrEqual","expected":1}]}
EOF

chainbench run --chain wbft \
  --binary $CHAIN/go-wbft/build/bin/gwemix \
  --keys keys/preset --artifact-root /tmp/out /tmp/wbft-smoke.json
```

### 4.3 레거시 setup 경로

```sh
chainbench setup --chain wbft --launch --binary $CHAIN/go-wbft/build/bin/gwemix \
  --data-dir /tmp/x --validators 4
```

---

## 5. 검증 근거 (2026-08-09)

`chainbench run --chain wbft --binary <go-wbft/build/bin/gwemix> --keys keys/preset` 로
4노드 wbft 네트워크를 기동하고 스펙을 실행:

```
SEQ  ID          STATUS
1    wbft-smoke  pass

pass=1 fail=0 blocked=0 skip=0
```

`chainId == 8284`, `blockNumber ≥ 1` 이 실 RPC 에서 통과 — 즉 **genesis 생성 → 기동 → 합의 성립 →
RPC 응답**의 전 구간이 동작한다. 게이트된 회귀 테스트로는 아직 고정되지 않았다(§6).

---

## 6. 남은 일

| 항목 | 내용 |
|---|---|
| 라이브 e2e 고정 | stablenet 과 달리 `GWBFT_BIN` 게이트 회귀 테스트가 없다 — 위 실행을 테스트로 고정할 것 |
| n=6 쿼럼 | WBFT-012/013 은 6노드 프리셋 필요(`keys generate`) |
| useNCP 시나리오 | 오버레이로 켤 수 있으나 golden 프로파일 없음 |
