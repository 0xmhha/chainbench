# 케이스 4 — gstable 단독 체인 구성

> 목표: `gstable` 만으로 stablenet 체인을 세우고 블록을 만들게 한다.
> 상태: ✅ **동작 확인** — 라이브 e2e 5종 + CI(mock/attach) 커버. 네 케이스 중 가장 검증도가 높다.
> 공통 절차·변곡점은 [README.md](README.md) 참조.

---

## 1. 전제

| 항목 | 값 |
|---|---|
| 바이너리 | `<chain>/go-stablenet/build/bin/gstable` (`make gstable`) |
| 부트스트랩 | `static` — genesis 에 검증자셋이 들어 있어 기동 즉시 합의 |
| 합의 family | `wbft` (stablenet 은 wbft 합의 + stable coin 정책) |
| chain id | 8283 |
| RPC 네임스페이스 | `istanbul` |
| 프리셋 | `keys/preset` (5노드; 검증자 4 + BLS/PoP + alloc) |
| 최소 노드 | 검증자 4 (BFT 진행) |

---

## 2. 절차

`bootstrap.type = static` 이므로 공통 파이프라인 1~10을 그대로 탄다.

| # | 단계 | 이 케이스에서 벌어지는 일 |
|---|---|---|
| 1 | resolve-chain | `registry.Get("stablenet")` → 매니페스트(engine_field `anzeon`, hardforks `istanbul,boho`) |
| 2 | resolve-binary | `--binary <gstable>` |
| 3 | load-preset | `keys/preset/metadata.json` → 검증자 4 주소 + BLS + `extraData` + alloc |
| 4 | allocate | `place` 가 노드별 p2p/http/ws/auth 포트 산출 + 용량 검증 |
| 5 | genesis | `PresetGenesisSource` → 템플릿 `internal/chains/stablenet/genesis.json` 에 프리셋 검증자셋 치환 |
| 6 | assemble-plan | `engine.AssemblePlan` → 노드별 datadir·config 경로·launch args |
| 7 | provision | genesis + per-node `config.toml` 물질화(upload-if-absent) |
| 8 | init-datadir | 노드별 `gstable init` |
| 9 | launch | `--nodekey`, 검증자는 `--unlock`/`--password`/`--miner.etherbase` |
| 10 | health-gate | 블록 전진(head ≥ 1) |

**중요(T4.4b 실측):** wbft 계열 검증자셋과 `extraData` 는 **프리셋에 baked** 되어 있다.
코드가 RLP 로 계산하지 않으므로 랜덤 키만으로는 유효 genesis 를 만들 수 없다 —
검증자셋의 출처는 항상 프리셋이고, `keyreg` 랜덤 키는 노드 신원/계정용이다.

---

## 3. 이 케이스의 변곡점

| 변곡점 | 설정 | 예 |
|---|---|---|
| 검증자 수 | `--validators` / spec `topology.bp` | `--validators 4` |
| 엔드포인트 | `--endpoints` / `topology.en` | 비생성 RPC 노드 추가 |
| 노드별 배치 | `setup --topology <yaml>` | 노드별 role·sync_mode·bootnode |
| 저장 방식 | `sync_mode` | `full`\|`snap`\|`archive` |
| **Boho 하드포크 블록** | `--set genesis.overrides.bohoBlock=N` | 지연 포크 시나리오 |
| 계정 Extra 비트 | `--genesis-overlay internal/chains/stablenet/overlays/account-extra.json` | authorized/blacklisted 계정 상태 |
| 프리셋 크기 | `validator set --nodes 6 --validators 6` | 5노드 초과 네트워크 |
| 포트 대역 | `ports.base_p2p`/`base_rpc` + step | step 제약(§2.6) 준수 |
| 아티팩트 | `--artifact-root` | 세션 저장 위치 |

### stablenet 고유

| 항목 | 값 |
|---|---|
| 시스템 컨트랙트 | native-coin adapter `0x…1000`, account manager `0x…B00003` |
| 하드포크 | `istanbul`, `boho` |
| capability | `process`, `rpc`, `ws`, `consensus` |

---

## 4. 실행

### 4.1 점검용 CLI (단계별)

```sh
CHAIN=/Users/0xtopaz/work/github/0xmhha/chain
chainbench chain up --case stablenet \
  --binary $CHAIN/go-stablenet/build/bin/gstable \
  --data-dir /tmp/cb-stablenet
```

단계마다 PASS/FAIL 이 찍힌다. 특정 단계까지만 보고 싶으면:

```sh
chainbench chain up --case stablenet --binary <gstable> --data-dir /tmp/x --stop-after provision
ls /tmp/x            # genesis.json + node<N>/config.toml 확인
```

### 4.2 DSL 스펙 실행 (엔진 경로)

```sh
chainbench run --chain stablenet --binary <gstable> --keys keys/preset \
  --artifact-root /tmp/out examples/specs/smoke-rpc-reads.json
```

### 4.3 레거시 setup 경로

```sh
chainbench setup --chain stablenet --launch --binary <gstable> --data-dir /tmp/x --validators 4
chainbench status --data-dir /tmp/x
chainbench stop   --data-dir /tmp/x
```

---

## 5. 검증 근거 (2026-08-09)

| 테스트 | 무엇을 증명 |
|---|---|
| `TestPresetGenesisSource_Live_GstableInit` | 프리셋 genesis 로 실 `gstable init` 통과 |
| `TestBuildEnv_Live_Stablenet` | allocator 할당 포트로 실 4노드 기동·헬스통과·teardown 고아 0 |
| `TestRunSpec_Live_Stablenet` | 실 RPC 대상 spec 실행(sendTx·chainId·blockNumber) |
| `TestEngine_Live_FullRun` | `Engine.Run` 한 번으로 기동→실행→teardown→session 저장 |
| `TestEngine_Live_NewVocabulary` | 바인딩·`read`·`faucet`·`logs`·`gasPrice`·`rpcCall`·`wsSubscribe`·`stopNode`/`startNode` |

모두 `GSTABLE_BIN` 게이트. CI 는 바이너리 부재로 clean-skip.

---

## 6. 알려진 제약

| 제약 | 내용 |
|---|---|
| IPC 경로 길이 | geth IPC 유닉스 소켓은 **104자 제한** → `--artifact-root`/`--data-dir` 를 짧게(`/tmp/x`) |
| 블록 웜업 | wbft 블록 생성까지 최대 ~35s → 헬스게이트 타임아웃을 넉넉히 |
| 검증자셋 출처 | 프리셋 baked(§2) — 랜덤 키만으로 genesis 생성 불가 |
