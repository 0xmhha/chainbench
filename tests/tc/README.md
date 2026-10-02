# tests/tc — DSL 테스트 케이스

케이스는 JSON 한 파일에 **무엇을 세우고 무엇을 확인할지**를 함께 적는다. 세울 망은
`chainPreset` 에 이름으로 부르고, 확인할 것은 `steps` 에 적는다.

## 1. 어디에 무엇이 있나

1단은 체인이고, `common/` 만 체인이 아니다 — 어느 체인에서도 같은 목적으로 돌 수 있는 것들이다.

```
tests/tc/
├── common/         69건   세 체인 공통. {node,tx,fee,contract,rpc,fault}/
├── go-stablenet/  111건   regression/ · post-v1.0.0-change/ · testnet/ · hardfork/ · vocabulary/
├── go-wbft/         8건
├── go-wemix/        7건
└── basic/           2건   공통으로 가지 않고 남은 둘
```

케이스 하나하나가 무엇을 보는지는 **파일 안의 `description`** 이 적는다. 훑어보려면:

```sh
bin/chainbench test list tests/tc/common
```

`common/` 에 무엇이 왜 있는지는 [`common/README.md`](common/README.md) 가 적는다.

## 2. 돌리는 법

[`HOW-TO-USE.md`](HOW-TO-USE.md) 하나만 보면 된다. 무엇을 먼저 갖춰야 하는지, 명령이 왜
그렇게 생겼는지, 판정을 어떻게 읽는지, 그리고 **케이스 197건의 실행 명령**이 거기 있다.

```sh
bin/chainbench run <케이스.json> --workspace-dir ~/cbw/one/<이름> --binary "$GSTABLE"
```

## 3. 케이스를 더할 때

**파일 이름.** `common/` 은 `CT-<영역>-<번호>-<간략설명>.json`, 나머지는
`<두자리 번호>-<간략설명>.json` 이다. 한 번호에서 갈라진 변형은 `01b`, `12c` 처럼 뒤에
글자를 붙인다.

**파일 안의 `id` 는 이름과 별개다.** 문서와 검사가 `id` 로 케이스를 부르므로, 파일 이름을
바꿔도 `id` 는 그대로 둔다.

**`description` 에 무엇을 왜 그렇게 재는지 적는다.** 이 저장소의 케이스 설명은 단언 목록이
아니라, 그 단언을 고른 이유와 한때 틀렸던 재는 방법을 적는다. 그것이 다음 사람이 같은
실수를 피하는 유일한 길이다.

**환경을 부르는 방법.** `"chainPreset": "<id>"` 로 적으면 `internal/dsl.ReadFiles` 가 케이스
파일의 디렉터리부터 위로 올라가며 `<id>.json` 과 `chain-preset/<id>.json` 을 찾는다. 그래서
`presets/chain/` 하나를 모든 깊이의 케이스가 공유한다. 새 망 모양이 필요하면 거기에 preset 을
더한다 — [`presets/chain/README.md`](../../presets/chain/README.md).

**더하기 전에 돌려 본다.**

```sh
bin/chainbench validate <새 케이스.json>     # 망을 세우지 않고 문법과 참조만 푼다
bin/chainbench run <새 케이스.json> --workspace-dir ~/cbw/new --binary "$GSTABLE" --plan
```

**목록을 다시 만든다.** 케이스를 더하거나 지웠으면 `HOW-TO-USE.md` 의 명령 목록을 갱신한다.

```sh
python3 scripts/gen-case-commands.py --write
```

## 4. 함께 있는 문서

| 문서 | 무엇 |
|---|---|
| [`HOW-TO-USE.md`](HOW-TO-USE.md) | 돌리는 법 + 케이스별 실행 명령 197건 |
| [`common/README.md`](common/README.md) | 공통 69건이 무엇이고 무엇이 왜 빠졌나 |
| [`CHAIN-BRINGUP.md`](CHAIN-BRINGUP.md) | 체인을 세우는 네 갈래와 핸드오프 |
| [`SPECS.md`](SPECS.md) | 케이스가 지키는 규약과, 끝내 옮기지 못한 여덟 건과 그 이유 |
| [`../../env/docker/README.md`](../../env/docker/README.md) | 체인 바이너리가 없는 기계에서 돌리는 길. 원격 서버 행세를 하는 컨테이너 15대 |
| [`../../docs/tc/common/`](../../docs/tc/common/) | CT 명세 원본(Confluence 사본)과 CT↔DSL 대응 |
| [`../../docs/dev/legacy-port-audit/`](../../docs/dev/legacy-port-audit/) | 레거시 대비 포팅 감사 |

DSL 이 표현하기 어려운 것(노드 생명주기, 동기화, 합의 정지)은 `tests/e2e/` 의 Go 테스트가
맡는다. 어느 레거시 테스트가 그쪽으로 갔는지는 위 감사 문서가 적는다.
