# 케이스 4 — gstable 단독 체인 구성

> 목표: `gstable` 만으로 stablenet 체인을 세우고 블록을 만들게 한다.
> 상태: ✅ **동작 확인** — 라이브 e2e 5종 + CI(mock/attach) 커버. 네 케이스 중 가장 검증도가 높다.
> 공통 절차·변곡점은 [README.md](README.md) 참조.

> **명령 표면 정정 (2026-09-11).** 본문 서술은 그때의 기록이다. **현재 표면은 `chain` 하나다** —
> `net` 명령군과 `chain up --case`, `chainbench setup` 은 모두 없다. 대응: `net up`/`chain up --case X`
> → `chain up --chain X`, `net <verb>` → `chain <verb>`, `--data-dir` → `--workspace-dir`,
> `--stop-after provision` → `--stage deploy`. 아래 실행 예시는 현재 표면으로 고쳐 두었다.
> 근거: `chainbench chain --help` · `cmd/chainbench/chaincmd/up.go`.

---

## 1. 전제

| 항목 | 값 |
|---|---|
| 바이너리 | `<chain>/go-stablenet/build/bin/gstable` (`make gstable`) |
| 부트스트랩 | `static` — genesis 에 검증자셋이 들어 있어 기동 즉시 합의 |
| 합의 family | `wbft` (stablenet 은 wbft 합의 + stable coin 정책) |
| chain id | 8283 |
| RPC 네임스페이스 | `istanbul` |
| 프리셋 | `presets/keys` (5노드; 검증자 4 + BLS/PoP + alloc) |
| 최소 노드 | 검증자 4 (BFT 진행) |

---

## 2. 절차

`bootstrap.type = static` 이므로 공통 파이프라인 1~10을 그대로 탄다.

> 아래 표의 단계 이름은 옛 이름이다. 지금 이름은 README §1 의 표로 읽는다(allocate → place,
> assemble-plan → build, provision → deploy, init-datadir → init, launch → start).
> 2026-09-30 확인: genesis 는 `genesis.PresetSource` 가 만들고, `engine.AssemblePlan` 은 코드에 없다
> (argv 조립은 `chain build`). config 파일은 워크스페이스에 `config_node<N>.toml` 로 떨어진다.

| # | 단계 | 이 케이스에서 벌어지는 일 |
|---|---|---|
| 1 | resolve-chain | `registry.Get("stablenet")` → 매니페스트(engine_field `anzeon`, hardforks `istanbul,boho`) |
| 2 | resolve-binary | `--binary <gstable>` |
| 3 | load-preset | `presets/keys/metadata.json` → 검증자 4 주소 + BLS + `extraData` + alloc |
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
| 검증자 수 | `--bp` / chain-preset `topology` | `--bp 4` |
| 엔드포인트 | `--en` / chain-preset `topology` | 비생성 RPC 노드 추가 |
| 노드별 배치 | `chain up --topology <yaml>` (예: `examples/topology.yaml`) | 노드별 role·sync_mode·bootnode |
| 저장 방식 | `sync_mode` | `full`\|`snap`\|`archive` |
| **Boho 하드포크 블록** | `--set bohoBlock=N` | 지연 포크 시나리오 |
| 계정 Extra 비트 | `--overlay internal/chains/stablenet/overlays/account-extra.json` | authorized/blacklisted 계정 상태 |
| 프리셋 크기 | `validator set --nodes 6 --validators 6 --out <dir>` | 5노드 초과 네트워크 |
| 포트 대역 | server set / `workspace-config` 의 포트 대역 | step 제약(README §2.6) 준수 |
| 아티팩트 | `run --artifact-root` (기본 `~/.chainbench/sessions`) | 세션 저장 위치 |

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
chainbench chain up --chain stablenet \
  --binary $CHAIN/go-stablenet/build/bin/gstable \
  --workspace-dir /tmp/cb-stablenet
```

단계마다 `단계: 결과` 한 줄이 찍히고(README §4 의 실제 출력), 실패하면 그 단계에서 멈춘다.
특정 단계까지만 보고 싶으면:

```sh
chainbench chain up --chain stablenet --binary <gstable> --workspace-dir /tmp/x --stage deploy
ls /tmp/x            # genesis.json + config_node<N>.toml + chain-record.json 확인
```

### 4.2 DSL 스펙 실행 (엔진 경로)

```sh
chainbench run --chain stablenet --binary <gstable> --keys presets/keys \
  --artifact-root /tmp/out examples/specs/smoke-rpc-reads.json
```

### 4.3 상태 확인과 정지

```sh
chainbench status --workspace-dir /tmp/x
chainbench stop   --workspace-dir /tmp/x
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

> **2026-09-30 확인.** 위 표는 2026-08-09 기록이다. 지금 있는 이름은
> `TestPresetGenesisSource_Live_GstableInit`(`internal/core/genesis`)·`TestRunSpec_Live_Stablenet`·
> `TestSuite_Live_FullRun`·`TestSuite_Live_NewVocabulary`(모두 `internal/testengine`)이고, 뒤 둘이
> `TestEngine_Live_*` 의 새 이름이다(`testengine.RunSuite` 가 chainsetup 으로 조립한다).
> `TestBuildEnv_Live_Stablenet` 은 코드에 없다. 같은 게이트로 `TestSuite_Live_AccountLabels`·
> `TestSuite_Live_DeclaredAccounts` 가 더 있다.

---

## 6. 알려진 제약

| 제약 | 내용 |
|---|---|
| IPC 경로 길이 | geth IPC 유닉스 소켓은 **104자 제한** → `--artifact-root`/`--workspace-dir` 를 짧게(`/tmp/x`) |
| 블록 웜업 | wbft 블록 생성까지 최대 ~35s → 헬스게이트 타임아웃을 넉넉히 |
| 검증자셋 출처 | 프리셋 baked(§2) — 랜덤 키만으로 genesis 생성 불가 |
