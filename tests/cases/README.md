# 체인 구성 케이스 — 선언으로 세우고, 같은 어휘로 확인한다

네 갈래의 체인 구성을 DSL 로 선언한다. 어느 갈래든 **같은 문법, 같은 실행기**를 쓴다.
갈래마다 다른 것은 `env/` 아래의 선언 하나뿐이고, 실행기는 체인 이름으로 분기하지 않는다.

| 갈래 | env | 케이스 | 구성 방식 |
|---|---|---|---|
| go-stablenet | `env/stablenet.env.json` | `stablenet/chain-up.json` | 워크스페이스 단계(`net up`) |
| go-wbft 단독 | `env/wbft.env.json` | `wbft/chain-up.json` | 워크스페이스 단계 |
| go-wemix 단독 | `env/wemix.env.json` | `wemix/chain-up.json` | 워크스페이스 단계 — 패밀리가 선언한 2-페이즈 부트스트랩(governance → etcd → join) |
| go-wemix → go-wbft | `env/wemix-wbft.env.json` | `wemix-wbft/handoff.json` | `upgrade` 선언 → 핸드오프 본문(`consensus/upgrade.Handoff`) |

실행기가 보는 것은 선언의 **모양**이다. `upgrade` 블록이 있으면 혼합 바이너리 핸드오프로
조립하고, 없으면 워크스페이스 단계로 조립한다. 어느 쪽이든 조립이 끝나면 같은 attach
엔진이 케이스를 돌린다.

## 실행

```sh
# 오프라인 검증 (env 참조를 풀어 문법을 확인한다)
chainbench validate tests/cases/*/*.json

# 구성 + 실행: 선언한 네트워크를 워크스페이스에 세우고 케이스를 돌린 뒤 내린다
chainbench run --workspace-dir /tmp/cb-stablenet tests/cases/stablenet/chain-up.json

# 바이너리가 PATH 에 없으면 덮어쓴다 (단일 바이너리 갈래)
chainbench run --workspace-dir /tmp/cb-stablenet --binary /path/to/gstable tests/cases/stablenet/chain-up.json

# 네트워크를 남겨 두고 살펴보려면
chainbench run --workspace-dir /tmp/cb-wemix --keep-up tests/cases/wemix/chain-up.json
```

바이너리 이름은 선언 안에서 환경 변수로 덮어쓸 수 있다: `${GWBFT_BIN:-gwbft}` 처럼.
go-wbft 의 make 타깃은 `gwemix` 라는 이름의 바이너리를 만들므로, wbft 갈래는 보통
`GWBFT_BIN=/path/to/go-wbft/build/bin/gwemix` 을 준다.

핸드오프 갈래는 go-wemix 저장소의 **자체** genesis 템플릿이 필요하다:
`GOWEMIX_TEMPLATE=/path/to/go-wemix/wemix/scripts/genesis-template.json`.
chainbench 에 내장된 wemix 템플릿은 치환용이라 바이너리가 거부한다.

## 검증 상태 (2026-08-28)

| 갈래 | 오프라인 validate | 라이브 |
|---|---|---|
| stablenet | ✅ | ✅ 이 머신의 gstable 로 `run --workspace-dir` 통과 |
| wbft | ✅ | ☐ gwbft 빌드 없음 |
| wemix | ✅ | ☐ gwemix 빌드 없음 |
| wemix → wbft | ✅ | ☐ 두 바이너리와 템플릿 없음 |

라이브로 확인되지 않은 갈래는 선언과 실행기 경로만 검증된 상태다. 바이너리가 준비되면
위 명령을 그대로 돌리고 이 표를 갱신한다.
