# Code graph — AST-measured package structure

> **측정 문서.** §2~§3 은 **2026-10-22 (`f9e46b2d`)** 재측정이다. §4 는 완료된
> launchopt 리팩토링의 **[이력]** 이다. 측정은 다시 뽑으면 갱신되지만, 뽑지 않은
> 사이의 코드 변경은 반영되지 않는다 — 어긋나면 코드가 이긴다.
>
> Generated from `scripts/inventory/code-graph` (go/ast, no build required).
> Regenerate with `go run ./scripts/inventory/code-graph . > graph.json` after
> structural refactors; **do not hand-edit the numbers here.**
>
> 같은 수치 위에서 중복·패키지 문서·함수 크기까지 본 검토는
> `code-health-review-2026-09-10.md` (2026-09-24 삭제) 에 있다.
> 호출(선택자) 그래프는 별도 도구다 — `../codegraph/`(지운 문서).
>
> 측정 이력: 08-27 = 75/268 · 09-07 = 65/209 · 09-10 = 70/225 · 09-11 = 70/225 ·
> 09-29 = 75/251 · 10-22 = 80/266.

## 1. Method

The tool parses every non-test `.go` file under `cmd/`, `internal/`, `scripts/`
and `tests/` and emits:

- **Nodes** — packages, with file/line counts, exported surface size, fan-in and
  fan-out (module-internal only).
- **Edges** — import relations annotated with the **exported symbols the
  importer actually references** (selector-expression count), so an edge shows
  *which part* of a package's surface is consumed, not just that it is imported.
- **Violations** — layering breaks against the coarse layer buckets in
  `layerOf` (the per-package placement is `layers.md` §3, which `internal/arch`
  enforces).

## 2. Measured shape — 2026-10-22 (80 packages, 266 edges, 94,866 lines)

`violations: 0`. 09-29 측정의 6건은 층이 깨진 것이 아니라 도구가 `internal/preset` 을
못 본 것이었다 — `layerOf` 에 분기가 없어 `other` 로 떨어지고, core → other 가
역방향으로 보였다. `preset` 을 L1(core) 버킷에 넣고, 이미 없는 `testspec`·`testkit`
분기를 지금 있는 `testsupport` 로 바꾸자 사라졌다. 배치의 정본은 `layers.md` §3 이고
그것을 기계로 강제하는 `internal/arch` 도 위반 0 으로 통과한다.

| Layer | Packages | Lines | Outbound edges land in |
|---|---|---|---|
| core (`internal/core/*`, `resource`, `preset`) | 28 | 24,625 | core 55 · domain 1 |
| orchestration (`chainsetup`, `chainsetup/verb`, `testengine`, `testhelper`, `mcp`, `dashboard`) | 6 | 34,055 | core 58 · other 8 · domain 4 · orchestration 4 · usecase 2 |
| usecase (`app`) | 1 | 17,504 | core 20 · orchestration 4 · domain 3 · other 1 |
| entry (`cmd/*`, incl. the per-group `*cmd` packages) | 19 | 5,724 | entry 32 · usecase 16 · core 6 · orchestration 4 · domain 3 · other 1 |
| other (`dsl/*`, `feature`, `nodemonitor`, `webcli`) | 6 | 5,511 | core 8 · domain 1 · other 1 · usecase 1 · orchestration 1 |
| domain (`chains/*`, `consensus/*`, `accounts`, `validatorset`) | 11 | 4,556 | core 18 · domain 11 |
| tests (`arch`, `tests/e2e`, `tests/webui/fixtures/*`) | 4 | 1,760 | core 1 |
| scripts (`scripts/inventory/*`) | 4 | 1,105 | tests 2 |
| support (`testsupport`) | 1 | 26 | — |

09-29 와 비교해 가장 크게 바뀐 것은 `app` 이다. 2,794줄에서 17,504줄로 늘었고,
늘어난 몫의 대부분이 Web UI 의 유스케이스다. 층의 모양은 그대로다 — 표면은 여전히
app 을 지나고, core 는 아래로만 향한다.

```mermaid
flowchart TD
    entry[entry: cmd/chainbench + per-group *cmd · -mcp · -dashboard]
    usecase[usecase: app]
    orch[orchestration: chainsetup · chainsetup/verb · testengine · testhelper · mcp · dashboard]
    other[other: dsl · dsl/interp · dsl/assert · feature · nodemonitor · webcli]
    domain[domain: chains · consensus · accounts · validatorset]
    core[core: 28 packages incl. resource · preset]
    entry -->|32| entry
    entry -->|16| usecase
    entry -->|4| orch
    entry -->|6| core
    usecase -->|20| core
    usecase -->|3| domain
    usecase -->|4| orch
    orch -->|58| core
    orch -->|8| other
    orch -->|4| domain
    orch -->|2| usecase
    other -->|8| core
    domain -->|18| core
    domain -->|11| domain
    core -->|55| core
```

**표면은 app 을 지난다.** `cmd/*` 에서 `app` 으로 가는 엣지가 16개, 274 refs 이고,
`mcp -> app` 은 317 refs 다. MCP 가 app 을 우회하지 않는다는 규칙(architecture-v2 §2)이
그래프에 그대로 보이며, `internal/arch` 가 이것을 기계로 강제한다.

**Highest fan-in (the load-bearing contracts)**

| Package | in | out | lines |
|---|---:|---:|---:|
| `internal/app` | 19 | 28 | 17,504 |
| `internal/core/node` | 19 | **0** | 1,474 |
| `internal/core/registry` | 18 | 1 | 1,135 |
| `internal/preset` | 15 | 3 | 553 |
| `cmd/chainbench/surface` | 13 | 1 | 96 |
| `internal/core/filestore` | 13 | **0** | 297 |
| `internal/core/keyring` · `core/process` · `core/session` | 9 | 2 / 6 / 2 | 391 / 1,781 / 2,835 |

많이 의존되면서 아무것도 의존하지 않는 둘(`core/node`·`core/filestore`)이 프리미티브
층이다. `cmd/chainbench/surface` 는 이번 측정에서 out 1 이 생겼다.

**Highest fan-out (the assemblers):** `app` (28), `testengine` (26),
`chainsetup` (24), `cmd/chainbench` (17), `testhelper` (10).
`app` 은 in 19 / out 28 인 허브인데, 유스케이스 층이라 의도한 모양이다.

## 3. "노드가 어디서 도는가" — 한 소유자로 수렴했다

08-27 측정에서 이 질문에 답하던 패키지는 여섯이었다(`core/netmap`·`netmap`·
`netmap/internal/serverset`·`core/portplan`·`core/place`·`core/netreg`).
**그 여섯은 모두 없다.** R 트랙의 통폐합이 끝나 `internal/resource` 하나가 소유하고,
attached-network 레지스트리는 `core/session` 으로 흡수됐다.

| Package | Layer | Lines | Exported | in / out | What it answers |
|---|---|---:|---|---|---|
| `internal/resource` | core | 3,806 | 26f / 49t | 7 / 4 | 서버 세트·인벤토리·포트 슬롯·배치·풀·opener·localmap·workspace-config·baseline |
| `internal/core/node` | core | 1,474 | 18f / 18t | 19 / **0** | 노드 어휘: label · role · Map · placement · peering · enode · reservation |
| `internal/core/session` | core | 2,835 | 35f / 30t | 9 / 2 | 세션·아티팩트, 그리고 attached-network 레지스트리(옛 `core/netreg`) |
| `internal/core/blueprint` | core | 1,228 | 6f / 18t | 2 / 5 | 네트워크 선언 문서(N1) — 배치와 키를 한 장에 |

```mermaid
flowchart TD
    cmd[cmd: resourcecmd · chaincmd · suitecmd]
    app[app]
    setup[chainsetup · chainsetup/verb]
    engine[testengine]
    res[internal/resource<br/>LoadSet · Access · Opener · Allocation · Baseline]
    nd[core/node<br/>Label · Map · Endpoints · peering · enode]
    proc[core/process]
    rem[core/remote<br/>AddrMap · Credentials]
    fs[core/filestore]

    cmd -->|3| res
    cmd -->|110| app
    app -->|162| res
    setup -->|106| res
    engine -->|12| res
    res -->|29| nd
    res -->|51| rem
    res -->|17| proc
    res -->|5| fs
```

엣지 라벨은 참조한 공개 심볼의 수(refs)다. 엣지가 이름보다 많이 말하는 것:

- **`resource` 는 `core/node` 의 어휘를 쓴다**(`Map`·`Label`·`Endpoints`, 29 refs).
  인벤토리가 배치 어휘에 의존하는 방향은 08-27 과 같고, architecture-v2 V2.4 가
  인벤토리를 그 모듈 *안에* 둔 결정이 지금 컴파일된 모양이다.
- **`app -> resource` 가 162 refs 로 가장 굵다.** 09-29 에는 `chainsetup` 쪽(67)이
  가장 굵었다. Web UI 가 서버 세트와 포트를 app 을 거쳐 다루면서 순서가 바뀌었다.
  구성 단계(`chainsetup`·`chainsetup/verb`, 106)도 여전히 자원 소유자를 통해서만
  서버·포트를 만진다. 그것이 이 통폐합의 목적이었다.
- `consensus/upgrade` 는 이제 없다. 09-29 그림의 `upg -> res` 엣지도 함께 사라졌다.
- `resource` 는 3,806줄로 core 에서 `core/session`(2,835) 보다 크다. 다음 통폐합
  검토 대상이라면 여기다.

## 4. Launch-argument assembly — 완료된 리팩토링 *(2026-08 이력)*

> **[이력] 이 절은 현재 상태를 말하지 않는다.** 여기 나오는 `internal/core/launchopt`·
> `internal/engine`·`internal/core/pipeline/setup`·`internal/chains/wemix/deploy` 는
> **모두 존재하지 않는다.** 실행 옵션 조립은 `internal/core/nodeconfig` 로 단일화됐다
> (`internal/core/nodeconfig/launchopt.go`). 그때의 설계 근거는
> `../archive/chain-binary-flag-graph.md` (2026-09-24 삭제) §3.3 이다.

그래프가 "실행 인자가 5곳에 흩어져 있다"는 주장(worklist T7.3/T7.4)을 정확한 지점으로
고정했던 기록이다.

| # | Site | What it contributed |
|---|---|---|
| 1 | `core/nodeconfig.LaunchArgs` | `--datadir --config --port --http --http.port --ws --ws.port` |
| 2 | `engine/launcher.go armSpecs` | identity: `--nodekey --unlock --password --miner.etherbase` |
| 3 | `consensus/{poa,wbft}.StartFlags` | family/role: `--mine --allow-insecure-unlock --rpc.*` |
| 4 | `consensus/upgrade.LaunchArgs` | handoff variant: adds `--http.addr --authrpc.port --networkid` |
| 5 | `chainsetup extraArgs` | handoff account/namespace: `--nat --http.api --unlock ...` |

**Gate note (그때의 결론, 지금도 유효한 교훈).** "byte-identical" 게이트는 구조적으로
만족 불가였다 — 레거시 argv 는 Identity 와 Mining 플래그를 두 번 교차로 낸다
(`--allow-insecure-unlock … --mine … --nodekey --unlock`). 실제로 쓴 게이트는 레거시
조립에 대한 **flag-pair equality** 와 정규 순서 스냅샷이다. geth 계열의 플래그 파싱은
위치 독립이므로 pair equality 가 의미상의 계약이다.
