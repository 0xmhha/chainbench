# wemix4 → chainbench(Go) 마이그레이션 절차 검토

> 근거 문서: `tests/wemix4/docs/test-execution-review.md` (사용자 작성 인수인계 검토서) 및
> `tests/wemix4/` 전체(README, envs/default/{run,bootstrap}.sh, node_env.json, lib/*.sh,
> genesis/genesis_main_test.md, docs/{node,tx,wbft,gov,rpc}.md).
> 목적: bash/SSH 기반 wemix4 e2e 스위트를 현재 chainbench Go 프레임워크에서 재현하기 위한 절차 검토.

---

## 1. wemix4 실행 모델 (재현해야 할 3가지 특성)

wemix4는 **하나의 연속된 체인**에서 세 단계를 모두 수행한다. 이것이 마이그레이션의 핵심 제약이다.

### 1-1. 단일 연속 체인 (wemix → 업그레이드 → wbft)
| 구간 | 블록 | 바이너리/합의 | 내용 |
|------|------|--------------|------|
| ① wemix(wpoa) | 0 ~ 99 | gwemix3 / wpoa | `wemix_bp_1~2` 생성, wemix 거버넌스 배포(`deploy_gov`+etcdInit), pre-croissant TX 5종 |
| ② 업그레이드 | 50 / 100 | — | BriocheBlock=50(리워드 halving), CroissantBlock=100(wbft 하드포크), 거버넌스 4종 엔진 배포, wpoa 데이터 마이그레이션 |
| ③ wbft | 100 ~ | gwemix4 / WBFT | `wbft_bp_1~7` 인계, BLS committed seal, epoch=10, **staking 기반 validator 선정** |

### 1-2. Stateful Phase 순서 (테스트 간 체인 상태 공유)
`envs/default/run.sh`의 Phase 1~17-Q는 **한 체인 위에서 순차 실행**되며 상태를 공유한다. 특히 GOV 계열:
- Phase 10: GOV-003(A 등록) → GOV-010(B~F,H 등록·Stabilization 해제) → GOV-005(G 등록, validator 6→7)
- Phase 12: GOV-009(H가 non-NCP라 미선정 → 등록 후 선정, G 탈락)
- Phase 15: GOV-023/004(unstake→credential)
- Phase 17-Q: WBFT-012가 `WBFT_012_KEEP=1`로 wbft-013과 "G 제거 상태" 공유

즉 GOV staking 흐름은 **선행 등록 상태(A~G 누적)** 에 의존한다. 재실행 시 체인 리셋 필수(`kill all && del all`).

### 1-3. 전체 설정 (genesis_main_test.md)
`chainId=111133, BriocheBlock=50, CroissantBlock=100, EpochLength=10, TargetValidators=7,`
`StabilizingStakersThreshold=5, UseNCP=true`, Init.Validators=7(wbft_bp coinbase),
govNCP.ncps=7(operator=OP_A~G), GovConfig{minStaking, unbonding 15/5, changeFeeDelay=5}.

핵심: **validator 선정은 staking 기반**이며, 현재는 `useNCP=true`+govNCP(=operator) 레지스트리로
**permissioned("약속된 validator만 등록")**. operator(OP_X)가 validator coinbase(VAL_X)를 staker로 등록
(`registerStaker(amount, VAL, OP, 0, BLS, PoP)`)한다. 추후 public(useNCP=false, 순수 staking)이 목표.

---

## 2. chainbench 현재 상태 매핑

| wemix4 구성요소 | chainbench Go 대응 | 상태 |
|---|---|---|
| 15서버 토폴로지(node_env.json) | driver(local/remote) + `internal/core/topology` + preset | 있음 (역할 레이아웃 재구성 필요) |
| `.credentials`/`env.conf`/`accounts.env` | `keys/preset` + `profiles/*.yaml` + genesis 템플릿 | 있음 |
| `bootstrap.sh`(genesis→setup→init→gov배포→run→pre-croissant→블록100대기) | `chainbench upgrade run` (`internal/consensus/upgrade`) | 있음 (**minimal 설정**) |
| `genesis_main_test.md` | `internal/chains/wbft/genesis.json`(템플릿) + 프로파일 | 있음 (**설정 격차**) |
| `staker_register.sh`(OP가 VAL 등록) | e2e 헬퍼 `registerStakerVia` 등 | 있음 |
| `lib/common.sh`/`node_ctrl.sh` | `internal/core/rpc`, `driver`, `procman` | 있음 |
| Phase 1~17 stateful 순서 | — | **없음** (포팅은 테스트별 독립 네트워크) |
| 87개 테스트 스크립트 | ~82개 Go e2e 포팅 | 대부분 완료 (독립 실행) |

**현재 포팅 방식**: GOV e2e 14개가 각각 `runGovHandoff`로 **독립 handoff 네트워크**를 부팅한다.
설정은 minimal(1 producer + 4 validator, fork@20, targetValidators=1, useNCP 없음).

---

## 3. 핵심 격차 (3가지)

1. **설정 충실도 격차**
   - 현 handoff: 1+4 노드, fork@20, `targetValidators=1`, `useNCP` 없음, chainId 템플릿.
   - wemix4: 2 producer + 7 validator(+5 EN +1 PN), fork@100, `targetValidators=7`, `threshold=5`,
     `useNCP=true`, govNCP=7 operator, brioche@50, chainId=111133.
   - GOV-005/009/010/006/007/008/017은 이 full 설정(useNCP + 7 validator)이 전제.

2. **Stateful 연속 체인 부재**
   - wemix4: 한 체인에서 A~G 누적 등록 등 상태 공유.
   - chainbench: 테스트마다 새 네트워크 → 상태 비의존 테스트(TX/RPC/WBFT 기본)는 무방하나,
     GOV staking 순서 의존(003→010→005→009…)은 **각 테스트가 선행 상태를 자체 재구성**해야 함.

3. **3-part 통합(wemix run/upgrade/wbft run) 미표현**
   - 현 handoff는 upgrade+wbft-run은 다루나, wemix(wpoa) 구간 테스트/pre-croissant TX는
     별도 취급. wemix4는 이를 하나의 체인 타임라인으로 연결.

---

## 4. 마이그레이션 절차 (단계별)

### 단계 A — 설정 계층: wemix4-fidelity 프로파일/genesis
1. `profiles/wemix-upgrade-full.yaml`(신규) 또는 기존 프로파일 옵션 확장:
   producers=2, validators=7, fork_block=100(+brioche 50), chainId=111133.
2. wbft genesis에 `useNCP`, `targetValidators`, `stabilizingStakersThreshold` 주입 경로 확보.
   - 표준 wbft 체인은 `--genesis-overlay`(`{"genesis":{"config":{"croissant":{"wBFT":{…}}}}}`)로 이미 가능(검증됨).
   - **handoff(upgrade run)에도 동일 오버레이/설정 주입 경로가 필요** (현재 handoff는 오버레이 미지원 → 추가 대상).
3. govNCP.ncps = operator 집합을 설정으로 주입(= wemix4의 "NCP=OP_A~G 자동 파생" 대응).

### 단계 B — Preset: validator/operator 신원 분리
1. `chainbench validator set`는 현재 validator/non-validator 구분만 지원(operator 역할 없음).
   - wemix4는 각 노드가 **coinbase(VAL_X) + 별도 operator(OP_X)** 를 가짐.
   - 확장: `--operators N`(또는 메타데이터에 operator 키 세트) 추가, govNCP=operator 매핑 생성.
2. 필요한 funded 신원: 7 validator coinbase + 7 operator (+ EN/producer). 8~15 노드 규모 preset 생성.

### 단계 C — Stateful GOV 시나리오 러너
GOV staking 순서 의존을 재현하는 두 방식 중 택1:
- **(권장) 시나리오 러너**: full-config 업그레이드 체인 **한 번 부팅** 후, Phase 10~15 순서대로
  GOV-003→010→005→009→011~014→020~023→004를 **하나의 테스트 함수 안에서 순차 assert**.
  - 장점: 비싼 handoff 1회, wemix4의 stateful 흐름 그대로 재현, validator 6→7 성장 등 자연스러움.
  - 형태: `cmd/chainbench/upgrade_gov_scenario_e2e_test.go`(신규) — `runGovHandoff`(full 설정) 1회 →
    등록 시퀀스 → 단계별 검증.
- (대안) 테스트별 독립 유지 + 각자 선행 상태 재구성: 현재 방식 연장이나 useNCP 7-validator를
  매 테스트 재구성 → 비용·복잡도 큼.

### 단계 D — 테스트 재작성/재배치
1. **GOV-005/009/010**: 단계 A~C 기반 full-config 업그레이드 체인에서 재작성.
   - PR #169(독립 wbft + useNCP 축소 3/2)를 **업그레이드 컨텍스트 + 문서값(7/5)** 으로 교체.
   - 프레이밍을 "staking 기반 validator 선정(현재 permissioned)"으로 정정.
2. **wemix(wpoa) 구간**: pre-croissant TX 5종 / wpoa 블록 검증이 필요한 케이스는 producer 구간에서 수행.
3. **상태 비의존 케이스(TX/RPC/WBFT 기본, 이미 포팅됨)**: 변경 불필요(독립 네트워크 유지).

### 단계 E — 검증
- 각 단계 라이브 검증(handoff full 설정 boot → 블록 생성 → 등록 → validator 성장/제외).
- 비-e2e 스위트·시크릿 스캔 유지.

---

## 5. 테스트별 마이그레이션 분류

| 분류 | 케이스 | 마이그레이션 |
|------|--------|--------------|
| 상태 비의존 (완료·유지) | TX-*, RPC 기본, WBFT-001/002/009/010/011/012/013, NODE-004/006/007 | 변경 없음(독립 네트워크) |
| upgrade 컨텍스트 (완료) | NODE-001/002, GOV-001/002, GOV-012/013/014/020/021/022/023 등 | 현행 handoff 유지(설정 충실도만 선택 상향) |
| **full useNCP + stateful (재작성 대상)** | **GOV-003/010/005/009**, GOV-006/007/008(NCP 거버넌스), GOV-017(긴급) | 단계 A~D: full-config 시나리오 러너로 재작성 |
| deferred/n-a | RPC-008(brioche 설정), GOV-024(n/a) | 별도 |

---

## 6. 권장 접근 & 리스크

- **권장**: 단계 A(handoff 오버레이 주입) → B(operator preset) → C(GOV 시나리오 러너) 순으로,
  full-config 업그레이드 체인 **1개**를 부팅해 GOV staking Phase를 순차 재현.
  기존 독립 포팅(상태 비의존)은 그대로 두어 회귀 위험 최소화.
- **리스크**:
  1. handoff에 useNCP/7-validator 설정 시 **합의 전이 안정성**(validator 세트 변동 중 quorum) — 라이브 검증 필수.
  2. handoff의 flaky etcd(기존 procman 재시도로 완화됨).
  3. operator/coinbase 분리 preset은 keys generate 확장 필요.
- **비변경 원칙**: 공유 genesis(`internal/chains/wbft/genesis.json`)는 오버레이로만 조정, 다른 e2e 회귀 방지.

---

## 7. 다음 실행 단위(제안)

1. handoff(`upgrade run`)에 `--genesis-overlay` 주입 경로 추가 (단계 A-2).
2. `keys generate --operators` 로 validator+operator preset 생성 (단계 B).
3. full-config handoff로 GOV-005/009/010 시나리오 러너 재작성 (단계 C·D) — PR #169 대체.
