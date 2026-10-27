# Web 작업 인자와 CLI 명령줄

Web에서 작업을 계획할 때 보내는 인자(`POST /api/v1/plans`의 `arguments`)가 `chainbench` 명령줄에서 무엇이 되는지
작업마다 적는다. 아래 표는 작업 계약(`GET /api/v1/contracts/jobs`, `internal/app/web_jobs_contract.go`)에서 만든다.
계약과 이 문서가 다르면 `TestWebJobArgumentsDocumentMatchesTheContract`가 실패하고,
`CHAINBENCH_UPDATE_DOCS=1 go test ./internal/app -run TestWebJobArgumentsDocumentMatchesTheContract`로 다시 쓴다.

계약은 세 가지를 함께 묶는다.

- 작업은 표에 있는 인자만 읽는다. 표에 없는 인자를 보내면 계획이 `<작업> does not read argument <이름>`으로
  거절된다. 무시되는 값이 적용된 것처럼 보이지 않게 하려는 것이다.
- 숫자 인자의 범위도 계약이 정한다. 0은 값을 비운 것으로 보고 엔진 기본값을 쓴다.
- 플래그가 적힌 인자는 그 플래그를 실제 명령줄에 더한다. `TestEveryContractFlagReachesTheCommandLine`이 작업이 쓰는
  명령줄 함수로 이것을 확인한다. 플래그가 "없음"인 인자는 엔진이 계획이나 실행에서 쓰지만 명령줄로 넘기지 않는다.

표에 없는 것은 엔진이 스스로 정하는 플래그다. `--workspace-dir`, `--keys`, `--server-set`, `--workspace-config`,
`--server-logins`, `--artifact-root`, `--read-only-keys`, `--keep-up`, `--plan`, `--json`이 여기에 든다. 어디서 오는지는
[데이터 인벤토리](web-data-inventory.md)의 "Web 입력에서 CLI 플래그로"에 있다.

## 작업별 표

<!-- generated from app.JobsContract: begin -->
### `chain.setup` 설정 생성 · 노드 초기화

명령: `chainbench chain up --stage init`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 내장 manifest면 --chain <id>로, 등록한 외부 manifest면 고정한 파일을 --manifest와 --genesis-template으로 넘깁니다. |
| `assetId` | 바이너리 | `--binary` | 검증한 바이너리를 대상에 둔 경로를 넘깁니다. |
| `serverRef` | 서버 | `--server` | 서버 이름을 그대로 넘깁니다. |
| `allServers` | 모든 서버 | `--all-servers` | 켜면 붙입니다. |
| `validators` | 블록 생산자 수 | `--bp` | 체인 구성을 고르지 않았을 때 넘깁니다. 비우면 4입니다. |
| `chainPresetRef` | 체인 구성 | `--chain-preset` | 고정한 revision의 내용을 파일로 써서 넘깁니다. |

### `chain.deploy` 노드 구축 · 실행

명령: `chainbench chain up --stage start`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 내장 manifest면 --chain <id>로, 등록한 외부 manifest면 고정한 파일을 --manifest와 --genesis-template으로 넘깁니다. |
| `assetId` | 바이너리 | `--binary` | 검증한 바이너리를 대상에 둔 경로를 넘깁니다. |
| `serverRef` | 서버 | `--server` | 서버 이름을 그대로 넘깁니다. |
| `allServers` | 모든 서버 | `--all-servers` | 켜면 붙입니다. |
| `validators` | 블록 생산자 수 | `--bp` | 체인 구성을 고르지 않았을 때 넘깁니다. 비우면 4입니다. |
| `chainPresetRef` | 체인 구성 | `--chain-preset` | 고정한 revision의 내용을 파일로 써서 넘깁니다. |

### `node.start` 노드 시작

명령: `chainbench node start --index N`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다. |
| `assetId` | 바이너리 | 없음 | 명령에는 넘기지 않습니다. 계획에서 검증합니다. |
| `serverRef` | 서버 | 없음 | 명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다. |
| `validators` | 블록 생산자 수 | 없음 | 명령에는 넘기지 않습니다. 적으면 기록된 생산자 수와 같아야 합니다. |

### `node.stop` 노드 정지

명령: `chainbench node stop --index N`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다. |
| `assetId` | 바이너리 | 없음 | 명령에는 넘기지 않습니다. 계획에서 검증합니다. |
| `serverRef` | 서버 | 없음 | 명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다. |
| `validators` | 블록 생산자 수 | 없음 | 명령에는 넘기지 않습니다. 적으면 기록된 생산자 수와 같아야 합니다. |

### `node.restart` 노드 재실행 · 자료 보존

명령: `chainbench chain restart --node N`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다. |
| `assetId` | 바이너리 | 없음 | 명령에는 넘기지 않습니다. 계획에서 검증합니다. |
| `serverRef` | 서버 | 없음 | 명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다. |
| `validators` | 블록 생산자 수 | 없음 | 명령에는 넘기지 않습니다. 적으면 기록된 생산자 수와 같아야 합니다. |

### `node.swap` 설정·바이너리 교체 · 재실행

명령: `chainbench node swap --index N --purpose web-reviewed-config`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다. |
| `assetId` | 바이너리 | 없음 | 명령에는 넘기지 않습니다. 계획에서 검증합니다. |
| `serverRef` | 서버 | 없음 | 명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다. |
| `validators` | 블록 생산자 수 | 없음 | 명령에는 넘기지 않습니다. 적으면 기록된 생산자 수와 같아야 합니다. |
| `replacementAssetId` | 교체할 바이너리 | `--binary` | 검증한 교체 바이너리를 대상에 둔 경로를 넘깁니다. |
| `configOverrides` | 바꿀 설정 | `--config` | 기록된 실행 옵션과 맞춘 key=value를 하나씩 넘깁니다. |
| `configAssetId` | 설정 파일 | `--config` | 등록 설정 파일의 값을 key=value로 바꿔 하나씩 넘깁니다. |

### `node.reset` 비생산자 초기화 · 정지 상태로 유지

명령: `chainbench node reset --index N`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다. |
| `assetId` | 바이너리 | 없음 | 명령에는 넘기지 않습니다. 계획에서 검증합니다. |
| `serverRef` | 서버 | 없음 | 명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다. |
| `validators` | 블록 생산자 수 | 없음 | 명령에는 넘기지 않습니다. 적으면 기록된 생산자 수와 같아야 합니다. |

### `network.monitor` SSH 노드 로그 수집 · 취소할 때까지

명령: `chainbench resource inspect --request <노드마다>`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `serverRef` | 서버 | 없음 | 이 서버의 기록된 노드 로그를 읽습니다. 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 서버 풀의 모든 서버에서 기록된 노드 로그를 읽습니다. |

### `test.run` 저장한 DSL 케이스 실행

명령: `chainbench run <케이스 파일...> --suite`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. --chain은 바이너리를 검증한 체인입니다. |
| `assetId` | 바이너리 | `--binary` | 검증한 바이너리 경로를 --binary와 --binary-for default=로 넘깁니다. |
| `serverRef` | 서버 | `--server` | 서버 이름을 그대로 넘깁니다. |
| `caseRefs` | 케이스 | 없음 | 고정한 revision의 케이스를 파일로 써서 위치 인자로 넘깁니다. |
| `runDefinitionRef` | 실행 목록 | 없음 | 실행 목록이 고정한 케이스를 순서대로 파일로 써서 위치 인자로 넘깁니다. |
| `binaryAssets` | 추가 바이너리 | `--binary-for` | 이름마다 검증한 바이너리 경로를 name=path로 넘깁니다. |
| `noSkips` | 건너뛴 케이스를 실패로 | `--no-skips` | 켜면 붙입니다. |
| `waitBlocks` | 시작 전 대기 높이 | `--wait-blocks` | 높이를 그대로 넘깁니다. |
| `nodeMonitorTimeoutSeconds` | 노드 준비 대기(초) | `--node-monitor-timeout` | 초를 붙여(300s) 넘깁니다. |

### `test.attach` 기록된 네트워크에 연결해 케이스 실행

명령: `chainbench run <케이스 파일...> --attach --no-node-control --read-only-keys`

| 인자 | 이름 | 플래그 | 명령줄에서 |
|---|---|---|---|
| `manifestId` | 체인 manifest | 없음 | 명령에는 넘기지 않습니다. 계획에서 이 manifest로 바이너리를 검증합니다. |
| `assetId` | 바이너리 | 없음 | 명령에는 넘기지 않습니다. 기록된 바이너리와 같아야 합니다. |
| `serverRef` | 서버 | 없음 | 명령에는 넘기지 않습니다. 기록에서 이 서버의 노드를 찾고, 이 서버에 연결한 내 로그인을 --server-logins 파일에 넣습니다. |
| `allServers` | 모든 서버 | 없음 | 명령에는 넘기지 않습니다. 여러 서버에 걸친 기록을 서버마다 확인합니다. |
| `caseRefs` | 케이스 | 없음 | 고정한 revision의 케이스를 파일로 써서 위치 인자로 넘깁니다. |

<!-- generated from app.JobsContract: end -->

## 아직 Web에서 표현할 수 없는 CLI 인자

`chain up`의 `--docker`와 `--chain-id`는 2026-10-27에 Web에서 열기로 정했고, 서버 세트 모델과 genesis 처리를 확인한 뒤
다음 변경에서 연다. 체인 프리셋으로 표현할 수 있는 인자(`--en`, `--pn`, `--topology` 등)는 프리셋으로 고른다.
`run --chain-presets`처럼 여러 실행을 묶는 인자와 서버 세트 없이 대상을 정하는 인자(`--target` 등)는 이번에 열지
않았고, 열지 말자고 정한 것도 아니다. 판단 근거는 인벤토리의 "`chain up` 플래그 가운데 Web이 표현하지 못하는 것"과
"`run`(suite) 플래그 가운데 Web이 표현하지 못하는 것"에 있다.
