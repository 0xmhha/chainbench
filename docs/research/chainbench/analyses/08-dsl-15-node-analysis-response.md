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
