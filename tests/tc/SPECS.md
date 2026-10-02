# 케이스 정의서의 규약과, 아직 쓰지 못한 케이스

`tests/tc` 의 케이스는 원래 `internal/testkit` 에 Go 함수로 등록돼 있던 것을 DSL 정의서로
옮긴 것이다. 그 이관은 끝났고 레거시 등록부는 2026-09-06 에 사라졌다. 이 문서에 남은 것은
두 가지다. 케이스가 지키는 **규약**과, 끝내 옮기지 못한 **케이스와 그 이유**다.

돌리는 법은 여기 없다. [`HOW-TO-USE.md`](HOW-TO-USE.md) 에 케이스마다 적혀 있다.

## 규약

- **파일 하나가 케이스 하나다.** 파일 이름은 `<번호>-<간략설명>.json`(공통 케이스는
  `CT-<영역>-<번호>-...`)이고, 번호는 원래 명세의 번호를 그대로 쓰므로 연속이 아니다.
- **파일 안의 `id` 는 파일 이름과 별개다.** 문서와 검사가 `id` 로 케이스를 부른다.
  `cmd/chainbench/validate_test.go` 가 저장소 전체에서 `id` 가 겹치지 않는지 본다. 세션이
  테스트를 `id` 로 기록하므로, 겹치면 하나가 다른 하나를 덮어쓴다.
- **`requires` 로 필요한 것을 선언한다.** 지금 쓰이는 값은 접두사로 갈린다 — 노드 기능
  (`rpc`·`ws`·`consensus`·`process`), 배포된 컨트랙트(`contract:govCouncil` 등),
  하드포크(`fork:boho`), 엔진(`engine:anzeon`), 체인 계열(`family:wbft`),
  precompile(`precompile:secp256r1`), EVM 버전(`evm:shanghai`), 실행 위치(`target:remote`),
  그리고 genesis 오버레이가 심는 상태(`account-extra`·`short-expiry`).
- **못 도는 체인은 `skipsOn` 으로 적는다.** 이름을 적은 체인에서 skip 하지 않으면 실패하고,
  적지 않은 체인에서 skip 해도 실패한다. 빈 목록은 "어디서든 돌아야 한다" 는 뜻이다.
  (`applicableChains` 는 v2 스키마에 아직 있지만 지금 쓰는 케이스가 하나도 없다.)

## 이관하지 않은 것과 그 이유

여덟 건이 남았다. 문법이 모자라서가 아니라, **DSL 이 잴 대상이 아니거나 바이너리가 그
기능을 갖고 있지 않아서**다. 가짜로 만들어 통과시키지 않고 그대로 둔다.

`TestSpecDoc_BlockedCasesHaveNoSpec`(`internal/testengine/specdoc_test.go`)이 이 표를
읽는다. 여기 막혔다고 적힌 케이스에 정의서가 생기면 테스트가 실패한다. 한 번 낡은 적이
있어서 — 막혔다고 적힌 19건 중 15건에 이미 정의서가 있었다 — 주장을 기계가 지키게 했다.

| 케이스 | 왜 막혀 있나 | 열리는 조건 |
|---|---|---|
| `zero-address-transfer-blocked` · `precompile-transfer-blocked` | accounts SDK 의 **클라이언트측 정적 가드**(제출 전 거부)를 재는 케이스다. DSL 의 `sendTx` 는 노드로 직행하므로 그 가드를 태우지 못한다. 같은 이름으로 다른 것을 재게 된다 | 없다. SDK 가드는 DSL 이 잴 대상이 아니다 |
| `external-value-transfer` · `external-fee-delegated-transfer` | 조작자가 밖에서 넣어 주는 자금 있는 키가 필요하다. `newAccount` 는 일회용 키를 **만들** 뿐이었다 | **수단은 생겼다.** `accounts.<이름>.keyFile` 이 `${VAR}` 로 가리킨 키 파일을 읽는다(`presets/chain/stablenet-testnet-funded.json` 이 그렇게 쓴다). 케이스를 아직 쓰지 않았을 뿐이다 |
| `p256-precompile-active` · `p256-rejects-invalid` · `p256-inactive-before-boho` | **라이브 반증.** gstable 빌드의 `0x100` 이 valid·corrupt·short 세 벡터 모두 `"0x"` 를 돌려준다. P256VERIFY 가 탑재돼 있지 않다. `-active` 는 그대로 실패하고, `-rejects-invalid` 는 precompile 이 없어서 우연히 통과할 뿐이라 재는 것이 없다 | Boho/P256 을 탑재한 gstable 바이너리. 체인팀 확인이 필요하다 |
| `govminter-code-changes-at-boho` | **라이브 반증.** chainbench 의 genesis 빌더는 `bohoBlock=10` 이어도 GovMinter 를 처음부터 최종본으로 굽는다. 블록 1 과 latest 의 `getCode` 가 같다(38250 hex, md5 일치). "v1→v2 코드 교체" 라는 신호 자체가 이 구성에 없다 | 지연 포크 넷에 v1 코드를 굽는 genesis 소스 |

`tipcap-underpriced-rejected` 는 2026-08 에 같은 이유로 보류했다가 삭제했다. 당시 gstable
빌드가 제출 시점에 최소 팁을 강제하지 않아 `tipCap=1 wei` 가 그대로 채굴됐다. 지금 그
자리는 공통 케이스 `CT-FEE-001` 이 메운다. `txpool.nolocals` 로 노드를 띄워 RPC 제출의 local
면제를 끄고 재는 방식이다.
