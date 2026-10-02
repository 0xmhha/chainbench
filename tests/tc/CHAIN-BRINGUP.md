# 체인 구성 — 선언으로 세우고, 같은 실행기로 돌린다

체인을 어떻게 세울지는 **chain-preset** 이 선언한다. 케이스는 그 이름을 부를 뿐이고,
실행기는 체인 이름으로 분기하지 않는다. 갈래마다 다른 것은 preset 하나다.

프리셋이 무엇을 담고 무엇을 담지 않는지는
[`presets/chain/README.md`](../../presets/chain/README.md) 가 적는다. 여기서는 **체인을
세우는 네 갈래**가 지금 어떤 모양인지만 본다.

## 네 갈래

| 갈래 | 대표 preset | binary | 세우는 방식 |
|---|---|---|---|
| go-stablenet | `stablenet-bp4` 외 19개 | `gstable` | 워크스페이스 단계 |
| go-wbft | `wbft-bp4` 외 6개 | `gwbft` | 워크스페이스 단계 |
| go-wemix (poa) | `wemix-bp4` 외 5개 | `gwemix` | 워크스페이스 단계 — poa 가 governance → etcd → join 2단계 부트스트랩을 더 탄다 |
| go-wemix → go-wbft | `wemix-to-wbft`, `wemix-to-wbft-bp2` | 둘 다 | 워크스페이스 단계 + 포크 지점에서 바이너리 교체 |

케이스는 `chainPreset` 에 이름만 적는다.

```json
{ "schemaVersion": "2", "kind": "case", "id": "basic-tx-send",
  "chainPreset": "stablenet-bp4-en1",
  "steps": [ ... ] }
```

## 핸드오프 갈래가 다른 점

바이너리를 하나가 아니라 **이름 붙은 여럿**으로 선언하고, `upgrade` 가 언제 누구에게
넘길지를 적는다.

```json
"binaries": {
  "default": "${GWEMIX_BIN:-gwemix}",
  "next":    { "binary": "${GWBFT_BIN:-gwbft}", "chain": "wbft" }
},
"upgrade": { "fork": "croissant", "at": 20, "from": "default", "to": "next", "style": "concurrent" },
"topology": { "nodes": [
  { "index": 1, "role": "en", "binary": "next" },
  { "index": 5, "role": "bp" }
]}
```

**별도의 조립기를 타지 않는다.** 보통 네트워크와 같은 워크스페이스 단계로 세우고, 포크를
`GenesisFork` 로 얹을 뿐이다. 그래서 `upgrade` 가 있는 preset 도 같은 명령으로 돈다
(`stablenet-bp4-en1-default-upgrade`, `stablenet-restart-at-boho` 도 같은 모양이다).

## 실행

```sh
# 오프라인 검증 — 체인을 세우지 않고 문법과 env 참조만 푼다
bin/chainbench validate tests/tc/common/node/CT-NODE-001-startup-block-production.json

# 세우고, 돌리고, 내린다
bin/chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json \
  --workspace-dir ~/cbw/bringup --binary <체인 바이너리>

# 남겨 두고 들여다보려면
bin/chainbench run <케이스> --workspace-dir ~/cbw/bringup --binary <바이너리> --keep-up
bin/chainbench chain show   --workspace-dir ~/cbw/bringup
bin/chainbench chain stop   --workspace-dir ~/cbw/bringup
```

케이스마다 어떤 바이너리를 주는지, 환경변수가 더 필요한 preset 이 무엇인지는
[`HOW-TO-USE.md`](HOW-TO-USE.md) 에 케이스별로 적혀 있다.

## 바이너리 이름

preset 은 `${GWBFT_BIN:-gwbft}` 처럼 환경변수로 덮어쓸 수 있게 적는다. go-wbft 의 make
타깃이 `gwemix` 라는 이름으로 빌드하므로(저장소가 go-wemix 에서 갈라져 나왔다), wbft 갈래는
보통 이렇게 준다.

```sh
export GWBFT_BIN=<go-wbft>/build/bin/gwemix
```

핸드오프 갈래는 `GWEMIX_BIN` 과 `GWBFT_BIN` 두 빌드만 있으면 된다. wemix genesis 는
chainbench 에 내장된 템플릿을 `poa.PrepareTemplate` 으로 치환해 넘기므로, 예전에 쓰던
`GOWEMIX_TEMPLATE`(go-wemix 저장소의 자체 템플릿)는 **지금 아무 코드도 읽지 않는다**
(2026-09-30 확인).

## 기록

`presets/chain/wemix-upgrade.yaml` 과 `wemix-upgrade-15.yaml` 은 **실행 입력이 아니다.**
핸드오프를 라이브로 통과시켰을 때의 환경(생산자 1 + 검증자 4, 포크 20번 블록)을 값까지
적어 둔 기록이고, 어떤 코드도 이 파일을 열지 않는다. 그때 조용한 실패를 냈던 조건들
(network_id 를 노드마다 다르게 두면 안 된다, 생산자와 검증자가 겹치면 etcd 에 합류하지
못한다)이 파일 머리말에 적혀 있다.
