# DSL 15노드 구성 — 첫 제출물 (07 전달서에 대한 분석 응답)

작성일: 2026-09-08
용도: 07-dsl-15-node-refactoring-handoff.md 의 §10 첫 제출물. **코드 변경 전** 검토용.
근거: 현재 HEAD 소스의 AST 파싱 그래프(`docs/dev/codegraph/`)와 세 갈래 데이터-흐름 추적.

상태 표기(07 §8): [설계됨] [파싱] [전달됨=중간 구조체까지] [동작] [단위검증] [라이브검증].
"필드가 있거나 구조체로 전달된다"는 이유만으로 동작으로 판정하지 않는다.

---

## 1. 현재 HEAD DSL topology 통계

| 항목 | 수량 |
|---|---:|
| 전체 case | 189 |
| count 축약 형식 | 188 |
| `topology.nodes[]` 상세 | 1 |
| 4노드 | 171 |
| 5노드 | 15 |
| 15노드 | 3 |
| 노드별 `launch` scope | 0 |
| 노드별 `config` scope | 0 |
| 다중 바이너리(>1) | 3 |

15노드 스펙 셋 — 모두 count 축약형:
- `go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json` = 13 bp / 1 pn / 1 en
- `go-wbft/chain-up/02-wbft-chain-up-15.json` = 13 bp / 1 pn / 1 en
- `go-wemix/chain-up/02-wemix-chain-up-15.json` = 13 bp / 2 en

07 의 분석 시점(189/170/15/3)과 사실상 동일하다. 노드별 launch/config 는 여전히 0건.

## 2. AST 호출·데이터 흐름 그래프

`docs/dev/codegraph/` 에 커밋했다. 패키지 65개, 크로스패키지 호출 엣지 1382개(`codegraph.json`).
파이프라인 배선(`pipeline.md` mermaid): 두 표면이 app 으로 모이고(mcp→app 204, cli/*→app),
app→chainsetup·testengine·dsl, testengine→chainsetup·dsl·dsl/interp·testhelper,
testhelper→dsl/interp·dsl/assert. app 우회 표면 엣지 없음.

## 3. count / node-table / Blueprint 지원 비교

세 형식이 노드별로 무엇을 물질화까지 끌고 가는지. (증거: `internal/chainsetup/steps_compose.go`,
`internal/core/node/topology.go`, `internal/core/blueprint/`)

| 노드별 필드 | count | node-table | Blueprint |
|---|---|---|---|
| role | 동작(카운트) | 동작 | 동작 |
| binary | 표현 불가 | 동작 | 파싱→**폐기**(placement 에 binary 안 실림) |
| sync mode | EN 일괄 하나 | 동작 | 동작 |
| bootnode | 암묵(poa 최상위 producer) | 동작(Entry.Bootnode) | **필드 없음** |
| index/label | 파생 | 동작 | 파생 |
| launch | 표현 불가 | **표현 불가** | 파싱→**폐기**(ResolvedNode.Launch 미소비) |
| config | 표현 불가 | 표현 불가 | 표현 불가 |
| server(배치) | 표현 불가 | **필드 없음** | 필드 있으나 **거부**(§7) |
| validator subset | 앞 N=validator | 역할로만 | 파싱→키링까지(단 §5 참조) |
| genesis/alloc/governance | 플래그로만 | 플래그로만 | 파싱→**폐기**(genesis 는 NetGenesisIn 플래그로만) |

핵심: **Blueprint 는 선언은 넓지만 대부분 파싱만 되고 버려진다.** 실제로 흐르는 것은 nodekey,
validator 집합, role, syncmode, peering 뿐이다. genesis·alloc·governance·binary·launch·server 는
resolve 까지 갔다가 소비되지 않는다. genesis 는 두 경로(up·step) 모두 `NetGenesisIn` 플래그로만
만든다. **표면 도달성**: topology·blueprint 는 **MCP 로 도달 불가**(chain_place 는 validators/
endpoints/peering 만). blueprint 필드의 **e2e/docker 커버리지 0**(tests/ 에 blueprint 참조 없음).

## 4. 체인별 canonical 15노드 모델 — 근거 (정본 선택은 사용자 결정)

저장소가 서로 다른 값을 기록한다. 코드는 어느 분할도 강제하지 않는다.

| 출처 | BP | PN | EN | 근거 |
|---|--:|--:|--:|---|
| env/docker/README(산문, stale) | 4 | 0 | 11 | `env/docker/README.md:40,59` — 경로도 없어진 `tests/cases/...` |
| stablenet 현행 DSL(실행됨) | 13 | 1 | 1 | 33-stablenet-chain-up-15:17-21, istanbul_getValidators=13 |
| wbft 현행 DSL(실행됨) | 13 | 1 | 1 | 02-wbft-chain-up-15:17-21 |
| wemix 현행 DSL(실행됨) | 13 | 0 | 2 | 02-wemix-chain-up-15:18-20 |
| R6 라이브(2026-09-02) | 14 | 0 | 1 | worklist:507 |

코드가 강제하는 사실:
- **PN 은 wbft-family(stablenet, wbft)만 지원, poa(wemix)는 거부**한다(`peering.go:64-73`,
  `poa.go:155-166` — etcd 가 그 자리 차지). 그래서 wemix 는 15 = bp + en 로만 쪼갤 수 있다.
- **validator 집합 = BP 집합.** genesis 는 배치의 BP 역할 노드에서 validator/BLS/member 를 뽑고
  (`genesis/source.go:122-138`, `poa/genesis_source.go:168-188`), pn·en 은 validator 가 아니다.
- **bootnode 는 역할이 아니라 속성**이다(`topology.go:32-43` Entry.Bootnode, 최대 1개). 잔재
  `boot` 역할 문자열은 남아 있으나 "bp 의 속성으로 접어야 한다"고 코드 주석이 유보(`node.go:50-53`).
  count-form 은 poa 가 최상위 인덱스 producer 를 boot 로 암묵 선택한다(`poa.go:87-90`).
- **함대 용량 = 15 호스트 × 4 슬롯 = 60 슬롯.** 15노드는 15 슬롯이면 되고 15대가 꼭 필요하지
  않다(4대에도 들어감). 15대가 필요한 건 `--all-servers` 로 서버당 하나씩 펼칠 때뿐.

## 5. `keys.nodekeys.validators` 불변식 (WA21/E) — **중대 발견**

데이터 흐름(hop, file:line):

```
DSL keys.nodekeys.validators (spec_v2.go:143)
 → Spec.EnvKeys (spec.go:49)
 → compose keysValidators (compose.go:85,93) → NetUpIn.KeysValidators (compose.go:171)
 → NetKeysIn.Validators (verbs_up.go:195) → ws.Keys(KeysOpts.Validators) (verbs_steps.go:107)
 → GeneratedKeys.Validators (steps_compose.go:107-111) → 메타데이터 Network.Validators/Members
   (generate.go:243 promote)
```

genesis 는 이 경로와 **별도**다:

```
genesis.Request{Validators: w.state.Validators(=배치 BP 수), Nodes: placed} (steps_compose.go:823)
 → wbft: NetworkForNodes(BP 인덱스) (source.go:122-138, preset.go:155)
 → validator/BLS/member = 배치의 BP 노드
```

**결론: wbft up 경로에서 `keys.nodekeys.validators` 는 genesis validator 집합을 바꾸지 못한다.**
전체 노드만큼 신원을 생성하고 그중 카운트만큼을 **디스크 메타데이터 목록**에 validator 로 적을
뿐이며, genesis 는 그 목록을 읽지 않고 BP 역할에서 다시 뽑는다. 즉 WA21/E 가 의도한 "generate 시
키 스텝의 --validators 와 동등"이 실제로는 성립하지 않는다. **필드는 [전달됨]까지지 [동작]이 아니다.**
제가 붙인 단위 테스트는 값이 `NetUpIn.KeysValidators` 까지 가는 것만 확인했다 — 07 §3.3 이 경고한
바로 그 함정이다.

07 §3.3 여섯 질문 답:
1. keys 스텝의 Validators = 생성 신원 중 앞 N개를 메타데이터 validator 목록으로 승격(생성 수와 별개).
2. genesis validator 세트는 **배치 BP 수**에서 파생(keys-validators 아님).
3. `keys.validators != BP 수`: 거부 없이 조용히 무시. wbft 는 항상 BP 전원을 validator 로.
4. 음수: 거부 아니라 "전부"로 흡수(`steps_compose.go:107`). 과대: 신선 생성 때만 거부, 상한이
   **전체 노드 수** 기준(BP 수 아님), 기존 ring 재사용 시 검사 skip.
5. "validator 수" 단일 권위 = 배치 BP 수. count/node-table/blueprint 셋 다 여기로 수렴. 단
   keys.validators 는 같은 이름의 **네 번째 입력**으로 다른 값을 가질 수 있는 애매점.
6. "bp인데 validator 아님"은 정상 경로에서 안 생김(unlock/etherbase 도 genesis 도 BP 역할에서 파생).

## 6. DSL / Blueprint 통합 방식 제안

07 §5.3 의 세 선택지:
1. 모든 DSL 에 `topology.nodes[]` 직접 작성 — 15노드 × 189파일 중복. §5.1 이 금하는 방향.
2. 공통 Blueprint 를 `env.blueprint` 로 참조 — 중복 없음, 재사용, fingerprint 반영.
3. 공통 Blueprint + 테스트별 DSL override.

**권장: 2/3(공통 Blueprint 참조 + override).** 단 지금은 불가능하다 — §3 이 보이듯 **Blueprint 가
대부분 파싱만 되고 버려진다.** 공통 Blueprint 로 15노드를 정의하려면 먼저 blueprint 물질화를
완성해야 한다(genesis·alloc·binary·launch, 필요시 per-node server). 그전까지는 노드별 binary/sync/
bootnode 를 실제로 물질화하는 유일한 형식이 **node-table** 인데, 그것도 per-node server·launch·
config 는 못 한다. 즉 통합 설계의 선결 조건은 "형식 선택"이 아니라 "**Blueprint 소비 완성**"이다.

## 7. 노드별 서버 배치 — 현재 미배선 구간

- 거부 지점: `internal/chainsetup/steps_compose.go:276` — `blueprintPlacements` 가 node.Server 를
  `"per-node server placement is not wired yet (N3)"` 로 거부.
- 그 위: `Node.Server`(blueprint.go:74)는 **검증도 resolve 도 안 됨**(ResolvedNode 에 Server 필드
  없음). node-table 형식엔 아예 Server 필드가 없다(topology.go). count 는 노드별 주소 개념이 없다.
- 아래: 거부를 넘겨도 resource 계층이 못 받는다. `resource.Request` 는 Role+Label 만 싣고
  (`pool.go:93-98`), `Inventory.Assign` 은 호스트를 슬롯 순서로 채운다(`inventory.go:145-182`).
  "이 노드를 저 호스트에" 채널이 없다. 배선된 건 역방향뿐 — 자동 배치 후 `Record.Server` 에 어느
  호스트에 앉았는지 기록(`steps_compose.go:345-352`).
- 판정: **어느 형식으로도 특정 노드를 특정 서버에 배치할 수 없다**(선언→resolve→resource 전 구간 미배선).

## 8. 단계별 구현·검증 계획 (제안 — 승인 후 착수)

사용자 결정(§9)이 선행해야 하는 것: canonical 모델, keys.validators 위상.

- 단계 A. **canonical 15노드 모델 확정**(사용자 결정) → 체인별 표를 문서에 고정.
- 단계 B. **keys.validators 위상 결정**(사용자 결정) → (b1) genesis 가 존중하게 배선하거나, (b2)
  불일치를 사전 검증에서 거부하거나(07 §5.6), (b3) 필드를 제거. 결정 전 확장 금지.
- 단계 C. **Blueprint 소비 완성** — resolve 까지 온 genesis·alloc·binary·launch 를 실제로 소비.
  각 필드마다 단위 + docker 라이브 게이트.
- 단계 D. **per-node server 배치 배선** — resource.Request 에 호스트 지정 채널 추가, Inventory 가
  존중. §7 의 세 구간(선언·resolve·resource) 모두.
- 단계 E. **canonical Blueprint + 테스트별 override** — 공통 15노드 1개, 각 테스트는 차이만.
- 단계 F. **검증** — 07 §7 의 20개 게이트(파싱·역할·bootnode·노드별 binary/sync/launch/config·
  15 nodekey·validator=BP·무충돌 배치·용량 부족 사전 거부·provenance·proxied 라우팅·local/remote/
  docker·CLI=MCP·실패 증적·fingerprint 재사용·최종 report). 각 게이트에 실행 증적.

이 계획은 대량 DSL 수정 이전 단계다. "topology 4→15 일괄 변경"은 하지 않는다(07 §1).

## 9. 사용자 결정이 필요한 항목

1. **canonical 15노드 역할 분할**(체인별). 4+11 vs 13/1/1 vs 13/0/2 vs 14/1 — 코드는 강제 안 함.
   제약: wemix 는 pn 불가(bp+en 만), validator=bp.
2. **stablenet/wbft 의 pn 개수**(코드는 proxied 면 pn≥1 만 요구).
3. **wemix 의 bp/en 분할**(validator=producer 수라 BFT 정족수에 직결).
4. **`keys.nodekeys.validators` 위상** — genesis 가 존중(배선)? 불일치 거부? 필드 제거? 현재는
   조용히 무효. (07 §3.3·§10.5 가 임의 결론 금지로 지정한 항목.)
5. **bootnode 명시 여부** — count-form 은 암묵 선택. 특정 노드 고정은 node-table 필요. `boot`
   역할을 bp 속성으로 접을지도 미결.
6. **함대 배치 정책** — 15노드를 15대에 하나씩(`--all-servers`) vs 소수 서버 슬롯으로.
7. **per-node server 배치를 지원할지** — 지원하면 단계 D 필요(resource 계층 변경).
8. **문서 stale 정정 범위**(§ 아래) — 모델 결정과 별개.

## 10. 문서 드리프트 (07 §8)

- `env/docker/README.md` — 4+11 산문, 없는 `tests/cases/...` 경로, gen-env.sh 가 안 만드는
  `server-set-wemix.yaml` 언급.
- `tests/tc/README.md` — 실제 통계와 대조 필요.
- `docs/dev/server-set.md` — 경로 비교.
- `docs/dev/chainbench-worklist.md` — WA21/E 를 "[전달됨]이지 [동작]/[라이브검증] 아님"으로 정정
  (§5 발견 반영), blueprint 노드별 server [미배선] 명기.

---

부록: 근거 그래프는 `docs/dev/codegraph/codegraph.json`. 추적 3건(keys.validators / per-node
server / canonical 모델)의 file:line 증거는 이 문서 각 절에 요약돼 있다.

---

## 11. 사용자 결정 (2026-09-08 확정)

07 §9 항목에 대한 사용자 결정. 이 결정이 §12 의 계획을 확정한다.

1. **keys/genesis** — genesis 는 세 경우를 다룬다: (a) genesis 에 bp(validator)와 그 키가 이미
   있으면 **그 키를 재사용**, (b) genesis 에 bp 는 있는데 키가 없는 경우, (c) genesis 에 bp 가 없어
   새로 생성. 반복 테스트 효율을 위해 **서버에 키가 이미 있으면 genesis 가 그 키를 계속 사용**한다.
2. **canonical 모델** — 기본은 **bp 를 최대로, pn·en 은 1개씩**. DSL node-table 로 노드별 역할을
   선언하면 그에 따른다. 이때 **몇 번 노드가 어떤 역할인지까지 DSL 에서 지정 가능**해야 한다
   (테스트 절차에서 특정 노드의 동작을 개별 설정하기 위해).
3. **pn 개수** — 기본 1, DSL 상세 설정 시 변경 가능.
4. **wemix** — **bootnode 노드가 반드시 필요**하고, 그 노드가 **etcd 서버를 운영**하여 이를 통해
   노드 간 연결이 이뤄진다. 이 전제로 설계한다.
5. **bootnode = 역할** — bootnode 노드가 **가장 먼저 설정·실행**되어야 하므로 역할로 둔다. DSL 에서
   변경 가능하고, **bootnode 를 2개 지정하면 DSL 문법 검증에서 에러**.
6. **동적 노드 수** — 노드는 **서버당 하나**가 기본이고, 노드가 서버보다 많으면 **port 를 순환
   분배**해 배치한다. "15"는 server-set 가용 서버가 15대라 나온 수일 뿐, **가용 서버가 늘거나 줄면
   노드 수도 그에 맞춰 동적으로 변한다**(15 고정 아님).
7. **per-node server** — 6번과 연관된 것으로 본다: 배치는 6번의 자동 분배(서버당 하나, 초과 시 port
   순환)를 따른다. 운영자가 특정 노드를 특정 서버에 직접 지정하는 방식은 채택하지 않는다.
   *(이 해석이 아니면 사용자에게 재문의 — §12 확인 항목.)*
8. **문서** — 관련 모든 문서가 정정 범위. **모든 작업 완료 후** 문서 정리를 진행한다.

## 12. 결정 반영 — refined 모델과 구현 계획

### 확정된 canonical 모델

- **크기: 서버셋 가용 수에서 동적 산출.** 하드코딩 15 금지. 노드 수 = 가용 서버 수(기본, 서버당
  하나), 노드가 서버보다 많으면 port 순환 분배.
- **기본 역할 분배**: pn 1 + en 1 + 나머지 전부 bp. wemix(poa)는 pn 불가 → bootnode 1 + en 1 +
  나머지 bp. validator 집합 = bp 집합(불변).
- **bootnode = 역할**(first-class), 가장 먼저 실행. wemix 는 bootnode 가 etcd 운영. 2개 지정 시
  문법 에러.
- **DSL node-table**: 노드 index 별 역할(bp/pn/en/bootnode)을 명시 지정 가능. count-form 은 위
  기본 분배를 동적 크기로 적용.

### 현재 코드와의 간극 (구현 대상)

- **bootnode 를 역할로 승격** — 현재 `boot` 는 폐기 예정 잔재(node.go:50-53)이고 bootnode 는 속성
  (Entry.Bootnode). 결정 5·4 는 반대로 **first-class 역할 + 실행 순서 최우선 + wemix etcd**를 요구.
  topology/peering/genesis/실행순서(poa BringUpPhases) 전반 변경.
- **동적 크기** — count-form 기본이 고정 수가 아니라 server-set 가용 수를 읽어 산출. port 순환은
  이미 resource 슬롯에 있으나(슬롯>1), "노드>서버 시 순환"을 기본 경로로 배선.
- **키 재사용(genesis 3-case)** — 서버에 키/genesis 가 있으면 재사용, 없으면 생성. 현재 generate
  경로는 매번 생성 가능성 — 재사용 판정 배선.
- **DSL per-node role+index** — node-table 에 대체로 있음. bootnode 역할 + 2개 에러 검증 추가.
- **keys.validators** — 결정 1 의 키 재사용 관점으로 재정의(단순 카운트 승격이 아니라 genesis×키
  존재의 3-case). §5 의 "무효" 상태를 이 결정에 맞게 배선하거나 제거.

### 착수 전 확인 항목 (사용자)

- (C1) §11.7 해석: per-node server 는 자동 분배(6번)만, 운영자 직접 지정 없음 — 맞나?
- (C2) §11.1 case (b): genesis 에 validator 주소는 있는데 키가 없으면 — 에러로 볼지, 키를 만들
  수 없으니(주소 고정) 거부인지. 재사용/생성만 있고 (b)는 오류로 두는 게 맞나?
- (C3) bootnode 역할 승격 범위: 세 체인 공통으로 role 추가하되 etcd 운영은 wemix 만 — 맞나?

## 13. C1·C2·C3 확정 (2026-09-08)

- **C1 정정**: node→server 는 server-set 순서로 **순차 할당**(node1→server1, node2→server2…).
  운영자가 임의 서버를 직접 고르는 방식은 아니다. 대신 **DSL 이 노드 index 별로 role·binary·
  key·genesis·config 를 미리 지정**할 수 있어야 한다(테스트 절차에서 특정 노드 동작을 개별 설정).
- **C2 정정**: genesis 에 validator 주소가 있는데 키가 서버에 없으면 → **키를 새로 생성하고
  genesis 의 validator 를 그 주소로 변경**한다(에러 아님). DSL 에 "서버에 존재할 키·config"를 미리
  선언해 두고, 그 파일이 있으면 사용, 없으면(지금 케이스) 생성 + genesis 갱신. 목적은 반복 테스트
  효율(genesis 에 bp/validator 를 미리 고정).
- **C3 확정**: bootnode 역할은 세 체인 공통 추가, etcd 운영은 wemix 만. **bootnode 의 enode 정보를
  다른 노드의 config 에 넣어** 각 노드가 누구에게 바로 연결할지 알게 한다.

## 14. 구체 구현 계획 (§6.4 제출물 — 검토 후 착수)

각 단계는 단위 테스트 + docker fleet 라이브 게이트를 갖는다. 재사용을 우선하고 축약 경로를 남긴다.

- **S1. bootnode 역할 승격 (토대).** `internal/core/node` 에 bootnode 를 first-class 역할로.
  peering RoleSupport 세 패밀리 허용, 2개 지정 시 문법/Validate 에러. 실행 순서 최우선(poa
  BringUpPhases 일반화). **bootnode enode → 다른 노드 config 의 static-nodes/bootnodes 로 전파**
  (`nodeconfig`). wemix 는 bootnode 가 etcd 운영(기존 poa boot 로직을 역할에 연결).
  대상: node/role.go·topology.go·peering.go, consensus/poa, chainsetup steps, nodeconfig.
- **S2. 동적 크기.** count-form 기본을 server-set 가용 수에서 산출(15 하드코딩 제거). 기본 역할 =
  bootnode/pn 1 + en 1 + 나머지 bp(wemix: pn 없음). 노드>서버면 슬롯 port 순환. 대상: chainsetup
  allocate/steps_compose, resource.
- **S3. 노드 index 별 DSL 필드.** node-table 에 per-node key·config 추가(binary·role·sync 는 있음).
  genesis 는 네트워크 단위 파일 재사용으로 취급. 대상: dsl spec_v2·schema, testengine compose,
  chainsetup keys/config 스텝.
- **S4. 키 재사용 + genesis 3-case (C2).** 선언된 키/config 파일이 서버에 있으면 재사용, 없으면
  생성 + genesis validator 갱신. keys.validators 를 이 관점으로 재정의(§5 무효 상태 해소). 대상:
  chainsetup keys 스텝·core/keyring·genesis.
- **S5. canonical env/blueprint + 테스트별 override.** 공통 노드 정의 1곳, 각 테스트는 차이만.
  (S1~S4 로 blueprint/node-table 소비가 완성된 뒤.)
- **S6. 검증.** 07 §7 의 20 게이트를 local·remote·docker 로. 각 게이트 실행 증적.

### 착수 순서 제안

S1(bootnode 역할)이 나머지의 토대다. S1 → S2 → S3 → S4 → S5 → S6. 각 단계는 별도 커밋/PR 가능.
대량 DSL 수정(S5)은 S1~S4 완료 후에만.

## 15. 최종 확정 모델 (2026-09-08, W1·W2·W3)

체인 코드 검증(go-wemix/go-wbft/go-stablenet) 후 통합 모델로 확정.

**검증된 메커니즘**: 체인에 별개 pn/proxy 역할 플래그는 없다. bootnode = 표준 geth —
허브가 discovery 를 켜고, 다른 노드가 `--bootnodes=<허브 enode>` 로 가리킨다. chainbench
nodeconfig 에 이미 `--bootnodes`·`--nodiscover` 가 있어 배선만 하면 된다(체인 변경 불필요).
wemix etcd 는 어느 member(producer)나 init 가능(`etcdInit` 전제 = `self != nil`), 나머지는 join.

**확정 모델 — 세 체인 공통, 동적**:
- 노드 수 = server-set 가용 서버 수(동적). 15대면 15노드, 6대면 6노드…
- **마지막 노드(index = 서버 수) = PN**. PN 은 bootnode 옵션(discovery)을 켜고, 그 enode 를 모든
  노드의 static-nodes 에 기본 포함(+ discovery 모드면 `--bootnodes` 로도 제공).
- EN 1개, 나머지 전부 BP. 예) 15 = 13 bp + 1 pn + 1 en, 6 = 4 bp + 1 pn + 1 en.
- validator 집합 = BP(pn·en 제외).
- **wemix(poa)도 bp/en/pn 지원으로 재정리**한다(현행 poa 는 pn 거부 — 이를 바꾼다). 별도 boot
  역할은 두지 않는다. producer 들이 etcd 클러스터를 이룬다(seed = 최상위 인덱스 BP, 내부 지정,
  pn 아님). pn 은 non-validator 로 p2p 디스커버리 허브만.
- DSL node-table 은 노드 index 별 role·binary·key·config 를 지정 가능. count-form 은 위 기본을
  동적 크기로 적용.

**현행 코드와의 간극(구현 대상)**:
- poa SupportsRole 에 pn 추가(bp/en/pn). peering proxied 를 wemix 에도 허용. "etcd 가 proxy 자리
  차지" 주석 근거 폐기(pn=p2p 디스커버리, etcd=producer 합의로 층이 다름).
- poa bootPlacement/etcd-seed 를 "최상위 producer(BP)"로(마지막 노드가 pn 이 되므로 pn 이 아닌 BP).
  genesis bootNodeId 도 그 producer 로. pn 은 member 에서 제외(en 과 동일).
- pn = bootnode 배선: discovery 켜기 + pn enode 를 static-nodes 기본 포함, discovery 모드 --bootnodes.
- 동적 크기: count-form 기본 = 서버 수, pn=마지막, en=1, 나머지 bp. 노드>서버면 슬롯 port 순환.
- 키 재사용(genesis 3-case), keys.validators 재정의.

**라이브 검증 필요(fleet)**: wemix 에서 non-producer pn 이 producer 들의 etcd 와 공존하며 정상
동작하는지. 코드상 가능해 보이나 실동작 확인 필요.

## 16. 최종 구현 계획 (검토 후 착수)

- **S1. poa pn 지원 + peering proxied 통일.** poa SupportsRole bp/en/pn, bootPlacement/etcd-seed 를
  최상위 BP 로, pn 을 member 제외. 단위 + wemix docker 라이브(공존 검증).
- **S2. pn = bootnode 배선.** pn discovery on + enode 를 전 노드 static-nodes 기본 포함(+ --bootnodes).
  세 체인.
- **S3. 동적 크기.** count-form 기본 = 서버 수, pn=마지막·en=1·나머지 bp. 노드>서버 port 순환.
- **S4. DSL node-table per-node(role·binary·key·config, index 지정) + count-form 동적 기본.**
- **S5. 키 재사용 + genesis 3-case.**
- **S6. canonical env + 테스트별 override.**
- **S7. 검증(07 §7 20 게이트, local/remote/docker, CLI=MCP).**

## 17. 위험 #1 해소 (사용자 검증, 2026-09-08)

**wemix non-producer pn 허브 — 동작 가능 확인.** 모든 gwemix 는 etcd 를 내장하고, 초기에 init 하는
노드를 중심으로 etcd 클러스터가 형성된다. pn 을 먼저 설정하면 pn 을 중심으로 노드가 연결되고,
**producer 가 아니어도 노드 간 연결정보(디스커버리)를 제공하는 노드로 동작할 수 있다.** §15 의
"proxy 자리를 etcd 가 차지" 근거로 pn 을 거부하던 것은 폐기한다.

**함의 — bring-up 순서**: pn(디스커버리 허브)이 먼저 뜨고, 이후 producer 들이 떠서 그중 하나가
etcd 를 init(seed)하고 나머지가 join 한다. etcd 멤버십(합의 층)은 producer 들 사이의 일이고, pn 은
non-member 로 p2p 디스커버리(연결 층)만 맡는다 — 두 층은 분리된다. S1 의 poa BringUpPhases 는 이
순서(pn-first → etcd-seed producer → 나머지)로 조립하고, chainbench 오케스트레이션에서의 통합
동작을 docker 라이브로 확인한다(체인 능력은 사용자가 검증; S1 은 그 배선을 검증).

**남은 항목은 구현 디테일(논리 블로커 아님)**: (2) discovery 모드 전환 — static 기본 포함 경로는
안전, discovery 경로는 검증 필요. (4) per-node key/config 재사용의 원격/docker 파일 존재 확인.
(5) keys.validators/genesis 3-case 의 세부(case-b 생성+genesis 갱신). 각 항목은 해당 단계에서
단위·라이브 게이트로 처리한다.
