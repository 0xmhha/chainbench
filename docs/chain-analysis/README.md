# 체인 바이너리 분석 — 무엇이 여기 있고, 언제 다시 뽑는가

> **[정본]** 체인 바이너리의 CLI 표면(command · subcommand · flag)과 그 배선.
> 체인의 실행 옵션에 대한 질문은 **체인 소스를 읽기 전에 여기부터** 본다.
> 여기 없으면 그때 소스를 읽고, 읽은 결과를 여기에 남긴다.

## 왜 이 문서가 먼저인가

2026-08-22, wemix 기동을 디버깅하며 `--consensusmethod` 가 무엇인지 찾으려고 go-wemix 소스를
grep 했다. 답은 이미 `gwemix/cli-graph.md:176` 에 있었다 — 플래그 정의 위치(`cmd/utils/flags.go:840`),
값 매핑(1=PoW, 2=PoA, 3=ETCD, 4=PBFT), 배선(`SetWemixConfig`)까지. **분석이 없어서가 아니라
찾을 수 없어서** 같은 일을 두 번 했다. 그래서 이 README 가 생겼고, `docs/README.md` 인덱스에
등재됐고, 각 산출물이 **기준 체인 커밋**을 적는다.

## 두 가지 질문, 두 가지 산출물

체인 실행 옵션에는 성격이 다른 두 질문이 있고, 각각 다른 방법이 맞다.

| 질문 | 산출물 | 방법 | 갱신 |
|---|---|---|---|
| **무엇을 받는가** (전체 표면) | `cli-surface.txt` · `cli-flags.txt` | **빌드된 바이너리의 `--help` 순회** | 스크립트 1회 실행 |
| **그 플래그가 무엇을 하는가** (배선) | `cli-graph.md` | **바이너리에 실제로 들어가는 파일 집합의 AST 그래프** — flag 정의 → setter → config 필드 | 손, 체인이 크게 바뀔 때 |
| RPC/메트릭 표면 | `rpc-metrics-graph.md` | 위와 같음 | 손 |

**바이너리가 자기 표면의 정본이다.** 소스에는 빌드 태그로 빠지는 코드, 죽은 플래그, 등록되지
않는 명령이 있다. `--help` 순회는 *빌드된 그것*이 받는 것만 보여주므로 이 질문에 대해서는
소스보다 정확하다.

**AST 는 다른 질문에 답한다.** `--help` 는 `--consensusmethod` 가 존재한다고만 말하고,
그것이 `params.ConsensusMethod` 를 바꾼다는 것은 말하지 못한다. 그 배선은 **바이너리에 실제로
컴파일되는 파일 목록**(`go list -deps` 로 main 패키지에서 뽑는다)을 AST 로 파싱해 caller/callee 를
따라가야 나온다 — grep 이 아니라. `cli-graph.md` 가 그 결과다.

## 재생성

```sh
CHAIN=~/work/github/0xmhha/chain
scripts/chain-analysis/capture-cli.sh $CHAIN/go-stablenet $CHAIN/go-stablenet/build/bin/gstable docs/chain-analysis/gstable
scripts/chain-analysis/capture-cli.sh $CHAIN/go-wbft      $CHAIN/go-wbft/build/bin/gwemix      docs/chain-analysis/gwbft
scripts/chain-analysis/capture-cli.sh $CHAIN/go-wemix     $CHAIN/go-wemix/build/bin/gwemix     docs/chain-analysis/gwemix
```

각 `cli-surface.txt` 머리에 **체인 커밋과 캡처 시각**이 박힌다. 지금 빌드하는 커밋과 다르면
그 파일은 낡은 것이고, 거기서 파생된 판단도 낡은 것이다 — 신선도를 사람의 기억이 아니라
파일이 말하게 하는 것이 요점이다.

## 현재 기준 (2026-08-22 캡처)

| 체인 | 바이너리 | 체인 커밋 | 명령 표면 | 플래그 | **우리가 방출 가능** |
|---|---|---|---|---:|---:|
| stablenet | `gstable` | `0937ac5c9` | 80 | 179 | **42 (24%)** |
| wbft | `gwemix`(go-wbft) | `7af50e45d` | 78 | 177 | **42 (24%)** |
| wemix | `gwemix`(go-wemix) | `1350376a6` | 58 | 195 | **49 (25%)** |

"방출 가능"은 `internal/core/launchopt` 의 dialect 테이블에 있는 키다. **테이블에 없는 키는
오류다** — `Args.Set` 이 `dialect %s does not support %q` 로 거절하므로 raw 통과 경로가 없다.
즉 나머지 ~75% 는 지금 **표현 자체가 불가능**하고, 지원하려면 dialect 테이블을 늘려야 한다.

이 수치가 "DSL 로 체인 구성을 자유롭게 지원한다"의 현재 위치다. 무엇을 먼저 늘릴지는
`cli-flags.txt` 와 launchopt 키를 차집합으로 놓고 고르면 된다.

## 우리 저장소의 코드 분석은 어디에

같은 작업을 반복하지 않기 위한 다른 절반이다.

| 대상 | 수단 | 형태 |
|---|---|---|
| 패키지 계층·의존 방향 | `internal/arch` | **테스트** — `layers.md` 표를 파싱해 검사하므로 문서와 검사가 어긋날 수 없다 |
| 패키지 그래프·fan-in/out | [`dev/architecture/code-graph.md`](../dev/architecture/code-graph.md) | 문서(이력) |
| 관심사별 소유 모듈 | [`dev/architecture/module-responsibilities.md`](../dev/architecture/module-responsibilities.md) | 문서(현행 설계) |

원칙은 하나다: **한 번 뽑은 분석은 재생성 가능한 형태로 남기고, 신선도를 스스로 말하게 한다.**
검사로 만들 수 있으면 검사로(가장 강함), 아니면 스크립트 산출물로, 그것도 아니면 기준 커밋을
적은 문서로.
