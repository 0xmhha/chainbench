# Web 도메인 모델 (2026-10-26)

[Web 데이터 모델 계획](web-data-model-plan.md) 3-3단계의 결과물이다. [데이터 인벤토리](web-data-inventory.md)는
지금 서버가 무엇을 주는지 적었고, 이 문서는 화면이 앞으로 무엇을 받아야 하는지 정한다. 3-4단계는 이 문서를
Go 응답 구조체와 OpenAPI로 옮기고, 3-5단계는 이 모델을 SSE로 실어 보낸다.

## 왜 이 문서가 필요한가

인벤토리에서 본 문제는 대부분 같은 뿌리에서 나왔다. 서버가 엔진의 내부 모양을 거의 그대로 내보내고, 화면이
그것을 다시 해석한다. 노드 상태는 세 곳에서 다르게 정해지고, 누락 사유는 문자열 하나로 합쳐져 화면이 다시
쪼개며, 실행 상태를 나타내는 어휘가 다섯 벌이다. 실제 표본에서도 `/networks`는 노드를 늘 `unknown`이라고 하는데
`/observations`는 정확한 상태를 알고 있었다. 화면을 다시 만들기 전에, 화면이 받는 개념 하나하나의 필드와 타입과
출처를 한 곳에 정해 두어야 같은 해석을 여러 화면이 따로 하지 않는다.

이 문서는 두 가지를 바꾸지 않는다. 엔진의 디스크 형식(PascalCase 결과 파일, `chain-record.json` 등)은 그대로
두고 대시보드가 읽어서 이 모델로 옮긴다. 그리고 Web은 엔진을 다시 구현하지 않고 감싸기만 한다.

## 공통 규칙

**이름.** 필드는 camelCase로 쓴다. 엔진 결과 파일의 PascalCase(`Expected`, `Actual`, `Index`)는 서버가 옮긴다.

**시각.** RFC 3339 UTC 문자열이고 나노초까지 쓸 수 있다. 값이 없으면 필드를 뺀다. `0001-01-01T00:00:00Z` 같은
0년 시각은 어떤 응답에도 나가지 않는다. Go에서는 `time.Time` 필드에 `omitzero`를 달거나 포인터로 바꾼다.
저장소의 Go 버전(1.26)은 `omitzero`를 지원한다.

**없는 값.** 필수가 아닌 필드는 값이 없을 때 뺀다. `null`은 "다음 쪽이 없다" 같은 커서 값에만 쓴다. 목록 필드는
비어 있어도 `[]`로 보내고 빼지 않는다.

**큰 정수.** 체인에서 읽은 수량(블록 번호, 잔액, nonce, gas, 단언의 기대값과 실제값)은 10진 문자열로 보낸다.
지금 엔진이 실제값을 문자열로 바꾸는 방식을 계약으로 굳히고, 기대값도 같은 규칙으로 맞춘다. 표본의 실패한
단언은 기대값이 숫자 `0`, 실제값이 문자열 `"7"`이었다. 지표 표본 값은 예외로 JSON 숫자로 둔다. 게이지 값이고
2^53을 넘을 일이 없으며, 차트가 바로 숫자로 쓴다. 스트림의 `version`처럼 서버가 세는 카운터도 숫자로 둔다.

**식별자.** 네트워크 ID와 작업 ID는 지금처럼 32자 hex다. 노드는 네트워크 안의 이름(`node1`)인 `id`와, 네트워크
밖에서 노드를 가리키는 `ref`(`<networkId>.<id>`)를 둘 다 갖는다. 화면은 `ref`를 만들지 않고 서버가 준 값을
그대로 쓴다. 지금은 로그 API만 `ref`를 요구해서 화면이 두 값을 이어 붙인다.

**어휘의 주인.** 상태, 사유, 역할, 작업 이름 같은 열거값은 서버 계약이 목록을 준다. 화면은 그 목록에 없는 값을
받으면 코드를 그대로 보이고 "알 수 없는 값"으로 표시한다. 한국어 문장은 화면이 만든다. 지금 서버가 보내는
한국어 `controlLimits` 문장은 사유 코드로 바꾼다.

**경로와 식별 해시.** 런타임 절대 경로(`dataPath`, `partialEffects` 안의 경로)와 장비 식별 해시는 화면에 꼭 필요한
곳에만 둔다. 아래 개념 표에서 각각 정한다.

## 상태 어휘

상태는 두 축으로 나눈다. 일이 어디까지 진행했는지를 말하는 **실행 상태**와, 노드 프로세스가 지금 어떤지를 말하는
**프로세스 상태**다. 둘을 한 어휘로 합치면 "노드가 실패했다"와 "작업이 실패했다"가 같은 낱말이 되어 화면이 다시
갈라야 한다.

### 실행 상태

작업, 단계, 실행 기록, 테스트, 스텝이 모두 이 어휘 하나를 쓴다. 값은 `pending`, `running`, `cancelling`,
`succeeded`, `failed`, `blocked`, `skipped`, `cancelled`, `interrupted`, `unknown` 열 가지다. 끝 상태는
`succeeded`, `failed`, `blocked`, `skipped`, `cancelled`, `interrupted`이고, 서버가 각 객체에 `terminal`
불리언을 같이 실어 화면이 끝 상태 목록을 다시 적지 않게 한다.

엔진의 다섯 어휘는 이렇게 옮긴다.

| 지금 어휘 (정의한 곳) | 지금 값 | 모델 값 |
|---|---|---|
| `WebJob.state` (`internal/app/web_job_contract.go`) | `accepted`, `running`, `cancelling`, `succeeded`, `failed`, `cancelled`, `interrupted` | `accepted`는 `pending`, 나머지는 같은 이름 |
| `WebJobPhase.state` (같은 파일) | `running`, `succeeded`, `failed`, `cancelled` | 같은 이름. 작업이 `interrupted`로 끝나면 남은 `running` 단계도 `interrupted`로 바꾼다 |
| `session.TestStatus` (`internal/core/session/session.go`) | `pass`, `fail`, `blocked`, `skip` | `succeeded`, `failed`, `blocked`, `skipped` |
| `session.StepState` (`internal/core/session/composition.go`) | `running`, `done`, `failed` | `running`, `succeeded`, `failed` |
| `collector.RunStatus` (`internal/core/collector/store.go`) | `running`, `succeeded`, `failed`, `skipped` | 같은 이름 |
| `WebRun.state` (`internal/app/web_history_capture.go`) | 작업 상태 일곱 가지, 또는 `succeeded`, `failed`, `unknown` | 위 규칙으로 옮긴 값 |

서버 재시작으로 작업이 `interrupted`가 될 때 단계가 `running`으로 남는 문제(인벤토리 "상태 문자열이 정의된 곳")는
이 규칙의 둘째 줄로 없앤다.

### 프로세스 상태

노드에만 쓴다. 값은 `running`, `stopped`, `conflict`, `unknown` 네 가지이고, 왜 그렇게 판정했는지는 `reason` 코드로
따로 준다. 지금 `process.Ownership`의 여섯 값과 Web 덮어쓰기는 이렇게 옮긴다.

| 지금 값 | 모델 `state` | 모델 `reason` |
|---|---|---|
| `running` | `running` | 없음 |
| `unrecorded_running` | `running` | `unrecorded_pid_found` |
| `stopped` | `stopped` | `no_matching_process` |
| `missing` | `stopped` | `recorded_pid_absent` |
| `ownership_mismatch` | `conflict` | `argv_mismatch`, `unrecorded_argv_mismatch`, `ledger_mismatch`, `binary_binding_unavailable` 가운데 하나 |
| `unknown` | `unknown` | `probe_unavailable`, `discovery_unavailable`, `machine_unavailable`, `binary_launch_unavailable` 가운데 하나 |

`/networks`, `/snapshot`, 스트림이 모두 이 관측 결과를 쓴다. 관측한 적이 없는 노드는 `unknown`과 `not_observed`다.
지금처럼 목록만 `unknown`으로 고정하는 일은 없어진다.

재확인 결과의 `process`(`running`, `stopped`, `unknown`)는 이 어휘의 부분집합이라 그대로 쓴다. 재확인의 `data`
(`present`, `absent`, `unknown`)와 `outcome`(`absent`, `present`, `unverified`), 작업의 `nodeDisposition`(`unknown`,
`retained`, `cleaned`, `cleanup_failed`)은 뜻이 달라 각자의 어휘로 둔다.

## 개념별 모델

표의 "출처"는 지금 어느 응답의 어느 필드에서 이 값을 얻는지, 또는 서버가 새로 채워야 하는지를 적는다. "새로"라고
적은 필드는 3-4단계에서 서버가 만든다. 필수가 "아니오"인 필드는 값이 없으면 뺀다.

### 네트워크 `Network`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `id` | string | 예 | `/networks` `items[].id` |
| `workspaceId` | string | 예 | `items[].workspaceId` |
| `chain` | string | 예 | 새로. `chain-record.json`의 `chain` |
| `ownership` | string | 예 | `items[].ownership` (`owned` 등) |
| `nodes` | `Node[]` | 예 | `items[].nodes` |
| `collector` | `CollectorStatus` | 아니오 | 새로. 모니터가 켜진 네트워크만 |
| `version` | number | 예 | `items[].version` |

### 노드 `Node`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `id` | string | 예 | `nodes[].id` (`node1`) |
| `ref` | string | 예 | 새로. `<networkId>.<id>` |
| `networkId` | string | 예 | `nodes[].networkId` |
| `role` | string | 예 | `nodes[].role` (`bp`, `en`, `pn`) |
| `server` | string | 예 | 새로. 서버 세트의 서버 이름. `chain-record.json`의 `server` |
| `syncMode` | string | 아니오 | 새로. `chain-record.json`의 `syncMode` |
| `ports` | object | 예 | 새로. `{p2p, http, ws, metrics}` 가운데 열린 것만, 숫자 |
| `observation` | `NodeObservation` | 예 | `/observations` `nodes[]`의 상태 필드들 |
| `controls` | `{name, allowed, reason?}[]` | 예 | `supportedControls`와 `controlLimits`를 합친다. `reason`은 코드 |
| `dataPath` | string | 아니오 | `nodes[].dataPath`. 노드 상세에서만 쓴다 |

`hostIdentity`는 노드에 두지 않는다. 서버 이름이 같은 일을 하고, 장비 해시는 다중 호스트 판정용이라 서버 안에만
있으면 된다.

### 노드 관측 `NodeObservation`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `state` | string | 예 | 위 프로세스 상태 |
| `reason` | string | 아니오 | `observationReason`. `running`이면 없다 |
| `detail` | string | 아니오 | 새로. 관측이 실패한 원인 오류 문장. 비밀 제거 |
| `pid` | number | 아니오 | `observedPid`. 0이면 뺀다 |
| `executable` | string | 아니오 | 새로. `chain observe`의 `ready`, `unavailable`, `unknown` |
| `observedAt` | string | 예 | `observedAt` |

### 수집 상태 `CollectorStatus`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `firstCollected` | string | 아니오 | 새로. 모니터 `state.json` |
| `lastCollected` | string | 아니오 | 새로. 같은 곳 |
| `intervalSeconds` | number | 예 | 지금 `coverage.sampleIntervalSeconds` (5) |
| `remoteLogs` | string | 아니오 | 새로. SSH 대상이면 `attached` 또는 `detached` |

### 지표 계열 `MetricSeries`와 표본

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `metric` | string | 예 | 새로. 개념 이름 `block_height`, `peer_count`, `txpool_pending`, `txpool_queued` |
| `source` | string | 예 | `series[].source` (`rpc`, `metrics`) |
| `canonical` | boolean | 예 | 새로. 아래 정본 규칙 |
| `nodeId` | string | 예 | `series[].nodeId` |
| `unit` | string | 예 | `series[].unit` |
| `samples` | `{time, value}[]` | 예 | `series[].samples`, `value`는 숫자 |

같은 개념의 출처가 둘이면 RPC 쪽을 정본으로 한다. 블록 높이는 `eth_blockNumber`(지금 이름 `block_height`), 피어
수는 `net_peerCount`(지금 이름 `peer_count`)가 정본이다. RPC 포트는 노드를 다루는 데 언제나 필요하지만 metrics 포트는
꺼져 있을 수 있기 때문이다. Prometheus의 `chain_head_block`과 `p2p_peers`는 `metric`을 같은 개념 이름으로 바꾸고
`source: "metrics"`, `canonical: false`로 보낸다. 화면의 `failureMetrics`처럼 두 이름 가운데 있는 쪽을 고르는
계산은 `canonical`을 보는 것으로 바뀐다. 정본이 비는 구간에 보조 출처를 대신 쓸지는 화면이 정한다.

### 누락 구간 `Gap`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `from`, `to` | string | 예 | `coverage.gaps[]` |
| `scope` | string | 예 | 새로. `network` 또는 `node` |
| `nodeId` | string | 아니오 | 지금 문자열의 첫 낱말. `scope`가 `node`일 때만 |
| `source` | string | 예 | 지금 문자열의 둘째 낱말. `collector`, `rpc`, `metrics`, `logs`, `archive` |
| `code` | string | 예 | 사유 코드 하나. 인벤토리 "누락 사유 목록" 표의 값과 `node_stopped`. 옛 `reason` 문자열과 겹치지 않게 이름을 `code`로 정했다 |
| `metric` | string | 아니오 | `not_exposed`일 때 어느 지표인지 |
| `runId` | string | 아니오 | `deleted_by_admin`일 때 어느 실행인지 |

지금의 `"<node> <source> <reason>"` 문자열(`reason`)은 화면이 새 필드로 옮겨 간 뒤 없앤다. 보관본(`gaps.jsonl`)에는 이미 노드, 출처, 사유가 따로 있으므로
서버는 합치지 않고 그대로 옮기면 된다. 사유 코드에 붙어 있던 지표 이름과 실행 ID는 `metric`, `runId`로 뺀다.

노드가 멈춰 PID가 0이면 수집기는 지금 사유 없이 건너뛴다. 표본에서 node4는 610초 동안 표본이 없었는데 누락 구간은
하나도 없었다. 모델에서는 이 구간을 `scope: "node"`, `source: "collector"`, `reason: "node_stopped"`인 누락으로
기록한다. 새 사유 코드다.

### 로그 줄 `LogLine`과 로그 쪽

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `time` | string | 예 | `entries[].time` |
| `timeSource` | string | 예 | 새로. 줄마다 `source`(노드가 찍은 시각) 또는 `collection`(수집 시각) |
| `level` | string | 아니오 | `entries[].level`. 읽지 못하면 뺀다 |
| `text` | string | 예 | `entries[].text`, 비밀 제거 |
| `nodeId` | string | 예 | 지금은 쪽 전체의 `nodeId`. 여러 노드를 한 쪽에 합칠 수 있게 줄로 내린다 |

로그 쪽은 `{lines, nextCursor, total, gaps}`이고 `gaps`는 위 `Gap` 모양이다. 한 쪽의 최대 줄 수는 100이다.
표본에서 node4 하나가 30분에 11,363줄을 썼다. 목록 API의 페이지 규칙(지금 기본 25와 50, 최대 100과 200이 섞여 있다)은
3-4단계에서 하나로 맞추고, 로그만 최대 100을 따로 둔다.

### 작업 `Job`과 단계 `Phase`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `id` | string | 예 | `id` |
| `operation` | string | 예 | `operation` |
| `state` | string | 예 | 실행 상태 |
| `terminal` | boolean | 예 | 새로 |
| `needsCheck` | boolean | 예 | 새로. 지금 화면 `jobSummary`가 계산하는 재확인 필요 여부 |
| `workspaceId`, `planId`, `actorId` | string | 예 | 같은 이름 |
| `createdAt` | string | 예 | `createdAt` |
| `finishedAt` | string | 아니오 | 새로. 끝 상태가 된 시각 |
| `phases` | `Phase[]` | 예 | `phases` |
| `retention` | string | 예 | `retention` |
| `nodeDisposition` | string | 예 | `nodeDisposition` |
| `cancel` | `{reason, message?}` | 아니오 | `cancelReason`, `cancelMessage` |
| `error` | `{code, message, requestId?}` | 아니오 | `error` |
| `runIds` | string[] | 예 | `runIds`. 값을 그대로 `/history/{runId}`에 넣을 수 있다(#466) |
| `partialEffects` | string[] | 예 | `partialEffects`, 경로 제거 |
| `unresolvedResources` | string[] | 예 | `unresolvedResources` |
| `reconciliation` | object | 아니오 | `reconciliation` |
| `credentialRefs` | string[] | 예 | `credentialRefs` |
| `requestId` | string | 아니오 | `requestId` |

| `Phase` 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `name` | string | 예 | `phases[].name` |
| `state` | string | 예 | 실행 상태 |
| `startedAt` | string | 예 | `phases[].startedAt`. 단계는 시작할 때 목록에 들어가므로 언제나 있다 |
| `finishedAt` | string | 아니오 | `phases[].finishedAt`. 실행 중이면 뺀다 |
| `message` | string | 아니오 | `phases[].message` |
| `failure` | `{state, detail, nodeId?}` | 아니오 | 새로. `chain-record.json` 단계 기록의 `failed`, `detail`과 실패한 노드 |

`partialEffects`의 문장에는 지금 런타임 절대 경로와 키 자료 경로가 그대로 있다. 모델에서는 서버가 경로를 작업 공간
기준 상대 경로로 줄여 보낸다. 화면이 경로를 읽어 무엇을 하지는 않기 때문이다.

### 실행 기록 `Run`

| 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `id` | string | 예 | `id` (`job-<hex>`, `session-<hex>`) |
| `kind` | string | 예 | `summary.kind` (`job`, `engine-session`) |
| `jobId` | string | 아니오 | `jobId` |
| `workspaceId`, `chain`, `actorId` | string | 아니오 | 같은 이름. 빈 문자열 대신 뺀다 |
| `state` | string | 예 | 실행 상태 |
| `startedAt` | string | 아니오 | `startedAt` |
| `finishedAt` | string | 아니오 | `finishedAt` |
| `missing` | string[] | 예 | `summary.missingDimensions`. 응답 때 다시 계산한다 |
| `counts` | `{succeeded, failed, blocked, skipped}` | 아니오 | `summary.counts` |
| `tests` | `Test[]` | 예 | `summary.tests`와 `summary.results`를 합친다 |
| `nodes` | `{id, role}[]` | 예 | 새로. 캡처한 `env.json`의 `nodes[]` |
| `observations` | `{networkId, from, to?}` | 아니오 | `summary.observations` |
| `captureGaps` | string[] | 예 | `summary.captureGaps` |
| `fingerprints` | object | 예 | `fingerprints` |

표본의 두 세션 기록은 `finishedAt`이 있는데 `missingDimensions`가 `["finishedAt"]`이었다. 캡처할 때 정한 값을 상세
응답이 다시 계산하지 않아서다. 모델에서는 `missing`을 응답할 때 계산한다. 작업 기록 하나와 세션 기록 하나가 같은
테스트 작업을 두 행으로 나타내는 문제는 목록 모델에서 다룬다. 이 문서는 행 하나의 모양만 정한다.

### 테스트 `Test`, 단언 `Assertion`, 스텝 `Step`

화면이 `tests/<NNN>_<id>/` 디렉터리 이름 규칙과 PascalCase 결과 파일을 직접 읽는 일(`run-timeline.mjs`의 `testRows`)을
없앤다. 서버가 결과 파일을 읽어 테스트마다 아래 모양으로 준다.

| `Test` 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `seq` | number | 예 | `status.json` `seq` |
| `id` | string | 예 | `status.json` `id` |
| `state` | string | 예 | `status.json` `result`를 실행 상태로 |
| `reason` | string | 아니오 | `status.json` `reason` |
| `startedAt`, `finishedAt` | string | 아니오 | `status.json` |
| `env` | string | 아니오 | `status.json` `env` |
| `assertions` | `Assertion[]` | 예 | `assert.json` |
| `steps` | `Step[]` | 예 | `steps.json` |
| `postActions` | `{name, ok, detail?}[]` | 예 | `postaction.json` |
| `artifacts` | `{kind, ref, node?}[]` | 예 | `artifacts.json` `refs` |

| `Assertion` 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `assert` | string | 예 | `Assert` |
| `nodeId` | string | 아니오 | `On`. 비어 있으면 `Provenance.on` |
| `compare` | string | 아니오 | `Provenance.compare` |
| `expected` | string | 예 | `Expected`. 숫자여도 10진 문자열로 바꾼다 |
| `actual` | string | 아니오 | `Actual` |
| `passed` | boolean | 예 | `Pass` |
| `detail` | string | 아니오 | `Source` |
| `provenance` | object | 아니오 | `Provenance` 원본 |

표본의 단언은 `ID`와 `On`이 빈 문자열이고 같은 값이 `Provenance` 안에 있었다. 그래서 `nodeId`는 `On`이 비면
`Provenance.on`에서 가져온다.

| `Step` 필드 | 타입 | 필수 | 출처 |
|---|---|---|---|
| `index` | number | 예 | `Index` |
| `type` | string | 예 | `Type` |
| `nodeId` | string | 아니오 | `On` |
| `signer`, `hash` | string | 아니오 | `Signer`, `Hash`. 비면 뺀다 |
| `nonce`, `gas` | string | 아니오 | `Nonce`, `Gas`를 10진 문자열로. 0이거나 비면 뺀다 |
| `error` | string | 아니오 | `Error` |
| `deployment` | object | 아니오 | `Deployment` |

영수증 `Receipt` 원본은 모델에 넣지 않는다. 원본 JSON을 보여 주는 내보내기에만 남긴다.

### 이벤트 `Event`

`/api/v1/events` 스트림 하나가 모든 실시간 변화를 싣는다(계획 6절의 3). 모든 이벤트는 `{type, version, cursor,
observedAt}`를 갖고, 종류에 따라 하나를 더 싣는다.

| `type` | 더 싣는 것 | 지금 |
|---|---|---|
| `resync_required` | `reason` | 있다 |
| `job` | `job: Job` | 있다 |
| `node` | `node: Node` (관측이 바뀐 노드만) | 없다 |
| `metrics` | `networkId`, `samples: {metric, source, nodeId, time, value}[]` (한 수집 회차) | 없다 |
| `gap` | `networkId`, `gap: Gap` (열리거나 닫힌 누락) | 없다 |
| `logs` | `networkId`, `lines: LogLine[]` (노드마다 회차당 최대 줄 수 제한) | 없다 |

지금 1초마다 오는 `observation` 이벤트는 내용이 없으므로 없애고, 연결 유지는 SSE 주석 줄로 한다. 끊긴 동안 놓친
이벤트는 지금처럼 `resync_required`를 받고 스냅숏으로 다시 맞춘다. 로그를 회차당 몇 줄까지 보낼지는 3-5단계에서
정한다.

## 노출 결정

인벤토리 세 영역 끝의 "있는데 화면에 오지 않는 데이터"마다 모델에 넣을지 정했다. 이미 응답에 오는데 화면이 읽지
않을 뿐인 것(작업의 `requestId`, 자격증명의 `createdAt`, 바이너리 증거, 감사 기록의 `jobId` 등)은 모델에 그대로
두고 화면 단계(3-7)에서 쓸지 정한다. 아래는 응답에 새로 넣거나 넣지 않기로 정한 것이다.

**모델에 넣는다.**

- 노드의 포트, 서버 이름, 동기화 방식. 노드 상세에 필요하고 비밀이 아니다.
- `chain observe`의 `executable`과 관측이 실패한 원인 문장(`NodeObservation.detail`).
- 수집 상태: 첫 수집 시각, 마지막 수집 시각, 원격 로그 수집이 붙어 있는지. 화면이 누락 구간으로 짐작하던 것이다.
- 로그 줄마다의 시각 출처.
- 결과 파일의 나머지: 성공한 단언과 출처, 스텝의 해시와 서명자와 nonce, `postaction.json`, `artifacts.json`.
- 실행에 쓰인 노드 목록(`env.json`의 `nodes[]`).
- 단계 실패의 분류와 상세(`chain-record.json` 단계 기록의 `failed`, `detail`).
- 실패 증거 `observations/`. 계획 6절의 5대로 비밀 검사를 통과한 뒤에만 넣는다. 노드 로그 전체는 크기 한도를 다시
  정해야 하므로 3-5단계에서 함께 정한다.
- 디스크 프리셋을 읽다가 버린 파일과 이유, 가져오기 원문의 만료 시각. 3-6단계에서 넣는다.
- 문서 revision 목록. 3-6단계에서 넣는다.

**뒤로 미룬다.**

- 세션 chainstate의 생산자 참여와 포크 감지. 표본에 시각이 없고 `seq`만 있어 시간축에 놓을 수 없다. 엔진이 시각을
  남기게 된 뒤 다시 정한다.

**넣지 않는다.**

- 프로세스 명령줄(`args`, 원장의 실행 인자). 키 파일 경로나 비밀번호가 섞일 수 있고, 화면에 이것이 필요한 일이 없다.
- 노드의 `logPath`, `configPath`, 장비 식별 해시. 절대 경로와 장비 정보이고 화면이 쓸 곳이 없다.
- 세션의 `keys/`, `logs/`, `nodes/` 디렉터리. `keys/`는 비밀이고, 나머지는 로그 보관본과 실패 증거가 대신한다.
- 영수증 원본, 엔진 수집기의 옛 버스 이벤트와 `fields`, 상태 기계의 `LogRec`, 생명주기 상태 숫자. 화면이 쓸 곳이
  없거나 프로세스 메모리에만 있다.
- 워크스페이스 이력. 서버가 저장하지 않는다.

## 표본으로 확인한 것

계획의 끝나는 조건은 3-2단계의 실제 표본이 모두 이 모델로 옮겨지는 것이다. `tests/webui/web-domain-model.test.mjs`는
`tests/webui/fixtures/stablenet-sample/`의 응답을 이 문서의 규칙대로 옮기는 참고 변환을 담고 있고, 옮긴 결과가 필수
필드와 타입을 지키는지, 표본에서 본 문제가 모델에서 사라지는지 확인한다. 이 변환은 화면 코드가 아니다. 3-4단계에서
서버가 같은 일을 하면 지운다.

변환에서 확인한 것은 다음과 같다.

- 네트워크, 노드, 관측, 지표, 누락, 로그, 작업, 실행 기록, 테스트, 단언, 스텝으로 표본 전부를 옮길 수 있다.
  "새로"라고 적은 필드(포트, 서버 이름, 수집 상태 등)는 지금 응답에 없으므로 변환이 채우지 않고, 테스트는 그 필드가
  지금 비어 있다는 사실을 따로 적는다.
- 멈춘 node4의 610초 구멍은 `node_stopped` 누락 하나로 나타난다. 지금 응답에는 이 누락이 없으므로 참고 변환은 표본 간격으로 구멍을 찾는다. 서버는 PID가 0인 것을 알고 있으므로 간격을 볼 필요가 없다.
- 블록 높이와 피어 수는 개념마다 정본 계열이 노드당 하나다.
- 단언의 기대값과 실제값이 모두 10진 문자열이 된다.
- 세션 기록의 `missing`에서 `finishedAt`이 빠진다.

3-2단계에서 확인하지 못한 `0001-01-01` 시각은 코드와 직렬화 실험으로 정리했다. 단계는 시작할 때 목록에 들어가므로
(`internal/app/web_chain_engine.go`, `internal/app/web_compose_cli.go`) 시작하지 않은 단계는 응답에 없다. 대신 실행 중인
단계의 `finishedAt`이 0년 시각으로 나간다. 같은 구조체를 Go 1.26으로 직렬화해 `"finishedAt":"0001-01-01T00:00:00Z"`가
나오는 것을 확인했다. 표본은 모든 작업이 끝난 뒤에 받아서 이 값이 담기지 않았다.

## 아직 정하지 않은 것과 단점

- 히스토리 목록에서 테스트 작업 하나가 작업 행과 세션 행 둘로 나오는 문제는 목록 모델을 정할 때 다룬다. 둘을 합칠지,
  묶어서 보일지에 따라 `Run.id` 규칙이 달라진다.
- 설정 입력(문서, 작업 공간, 자격 증명, 계획)의 모델은 3-6단계에서 정한다. 이 문서는 관측과 작업과 실행 기록만 다룬다.
- 로그 이벤트의 양 제한과 실패 증거의 크기 한도는 3-5단계로 넘겼다.
- 모델을 서버가 만들게 되면 지금 화면 코드 여러 곳(`gapText`, `failureMetrics`, `testRows`, `jobSummary`,
  `parseHistoryJSON`)이 한꺼번에 바뀐다. 화면을 다시 만들기 전에 서버를 먼저 바꾸면 그 사이 기존 화면이 깨지므로,
  3-4단계는 새 필드를 더하고 옛 필드를 바로 지우지 않는 순서로 간다. 그만큼 한동안 응답에 같은 값이 두 모양으로 있다.
- `controlLimits`의 한국어 문장을 코드로 바꾸면 화면이 문장을 새로 가져야 한다. 지금 서버 문장을 그대로 옮기면 된다.
