# Web 데이터 인벤토리 (2026-10-25)

[Web 데이터 모델 계획](web-data-model-plan.md) 3-1단계의 결과물이다. Web 서버가 화면에 주는 데이터를 API 하나씩 열어, 필드마다 JSON 타입, 비어 있을 수 있는지, 단위, 원본, 화면이 실제로 읽는지를 적었다. 3-3단계에서 도메인 모델을 정하고 3-4단계에서 계약을 정할 때 이 문서를 출발점으로 쓴다.

## 이 문서를 읽는 법

세 영역으로 나눴다. 관측(네트워크, 지표, 로그, 스트림), 작업과 실행 기록(계획, 작업, 기록), 설정 입력과 계정(계약 스키마, 문서, 작업 공간, 자격 증명, 계정)이다. 각 영역은 API마다 응답 Go 타입, 원본 데이터, 갱신 방식, 그 API를 쓰는 화면, 필드 표를 적고 문제를 문장으로 덧붙였다. 영역 끝에는 서버가 가지고 있지만 화면에 오지 않는 데이터와, 화면이 응답을 받아 스스로 계산하는 값을 따로 모았다.

근거는 `path:line`으로 달았다. 줄 번호는 커밋 `a90c814b` 기준이므로 이후 수정으로 몇 줄씩 밀렸을 수 있다.

## 3-1단계에서 아직 끝나지 않은 것

계획의 완료 조건은 둘이다. 화면이 쓰는 모든 응답 필드가 인벤토리 한 줄로 거슬러 올라가야 하고, 서버에 있지만 화면에 오지 않는 데이터마다 노출할지를 정해야 한다. 앞의 것은 이 문서로 채웠다. 뒤의 것은 각 영역 끝에 후보만 모았고 아직 정하지 않았다. 노출 여부는 3-3단계에서 정했고 [도메인 모델](web-domain-model.md)의 "노출 결정"에 적었다.

## 어떻게 확인했나

세 영역을 각각 코드를 읽어 정리했다. 그 가운데 아래 "확인한 문제"에 적은 것은 해당 코드를 다시 직접 열어 확인했다. 나머지 영역 본문의 주장은 코드 읽기에서 나온 것이고, 하나하나 다시 확인하지는 않았다. 본문은 처음에 실제 서버의 응답과 맞춰 보지 않고 썼다. 첫 대조 결과는 아래 "실제 표본으로 확인한 것"에 있다.

실제 표본이 부족하다는 사실도 이번에 확인했다. 지난 확인에 쓰던 런타임 디렉터리는 지워졌고, 남은 것은 git이 무시하는 `chainbench-out/` 아래 옛 형식 `status.json` 42개뿐이다. 그 안에는 로그 줄이 없고, 실패한 세션이 없으며, 지표 시계열은 표본이 두 개 남짓이다.

## 고친 결함

인벤토리를 만들다 실제 서버에서만 드러나는 결함 셋을 찾았고, #466에서 고쳤다. 셋 다 화면 확인에 쓰던 mock 서버가 실제 서버 규칙을 흉내 내지 않아서 놓쳤다.

1. 작업 상세가 기록을 `web:<proc>/<run>` 참조로 찾았는데, 기록 저장소는 그 참조를 해시한 `session-<hash>`를 키로 쓴다. 요청이 늘 404가 되어 원인 구역, 시험 대장, 시간축이 실제 서버에서 한 번도 나타나지 않았다. 이제 서버가 `web:` 참조도 받는다.
2. 원인 구역이 로그를 `limit=200`으로 요청했는데 로그 API는 100을 넘으면 400으로 거절한다. 이제 `logQuery`가 100에서 자른다.
3. `RedactWeb`이 등록된 계정 개인키를 가리지 않았다. 이제 `0x`가 있는 형태와 없는 형태, `accountKey`라는 필드 이름을 모두 가린다.

## 확인한 문제

아래는 코드를 직접 열어 확인했지만 아직 고치지 않은 문제다. 대부분 모델이나 계약을 정할 때 같이 풀어야 하므로 3-3, 3-4단계로 넘긴다.

- **노드 상태가 늘 "unknown"이다.** `/networks`는 저장된 PID만 보고 상태를 정하지 않고 `state := "unknown"`으로 고정한다(`internal/app/web_networks.go`의 `Networks` 반복문). 저장된 PID는 살아 있다는 증거가 아니므로 이 값 자체는 정직하지만, 실제 상태를 관측하는 길은 지금 모니터 쪽에만 있다. 어느 상태를 누가 관측해 화면에 주는지를 3-3단계에서 정한다.
- **누락 사유를 한 문자열로 합쳐 보낸다.** 서버 내부에는 노드, 출처, 사유가 따로 있는데(`webMonitorGap`), 응답으로 나갈 때 `"<node> <source> <reason>"`로 이어 붙인다(`internal/app/web_monitor_read.go`의 커버리지 병합). 사유 자체에 공백이 들어가는 경우가 있어서 화면의 `gapText`(`web/src/labels.mjs`)가 잘못 읽는다. `"node1 metrics block_height not_exposed"`는 지표 이름이 빠지고 코드가 번역되지 않은 채 "node1 · not_exposed"가 되고, `"archive deleted_by_admin <runId>"`는 실행 ID를 사유 코드로 읽는다. 응답에 필드를 나눠 싣는 것이 맞다.
- **단계 시각이 `0001-01-01T00:00:00Z`로 나갈 수 있다.** 단계의 `startedAt`, `finishedAt`이 `time.Time`에 `omitempty`를 단 값인데(`internal/app/web_job_contract.go`), Go의 `omitempty`는 구조체인 `time.Time`을 빼지 않는다. 단계는 시작할 때 목록에 들어가므로 0년 시각이 나가는 것은 실행 중인 단계의 `finishedAt`이다. 같은 구조체를 직렬화해 확인했다([도메인 모델](web-domain-model.md)의 "표본으로 확인한 것"). 모델은 이 필드를 값이 없으면 빼기로 정했다.
- **목록 API마다 페이지 규칙이 다르다.** 공통 `pageOf`는 기본 25, 최대 100이다(`internal/dashboard/paging.go`). 작업 목록은 기본 50, 최대 200이다(`internal/dashboard/jobs.go`). 로그는 최대 100이다. `JobPanel`은 작업 목록을 `api('jobs')`로 첫 페이지만 읽어 충돌을 확인하는데, 목록이 최신순이라 진행 중인 작업이 빠질 가능성은 낮지만 0은 아니다.
- **취소 사유를 입력할 곳이 없다.** 서버는 `CancelWithMessage`로 사유를 받아 `cancelMessage`에 저장하고 작업 상세도 그것을 보여 주는데, 두 취소 버튼(`JobCard.svelte`, `MonitorPanel.svelte`)은 빈 본문 `{}`만 보낸다.
- **작업 시작의 Idempotency-Key가 계획 ID와 같다.** `JobPanel`의 `start`가 `plan.id`를 키로 보낸다. 같은 계획으로 다시 시작하면 새 작업 대신 예전 작업이 돌아온다. 의도한 동작인지는 3-4단계에서 계약으로 적는다.
- **OpenAPI와 응답 모양이 다른 곳이 있다.** `/api/v1/chain-presets`는 맨 배열을 돌려주는데 OpenAPI는 `{items}`라고 적는다. `Document` 응답 스키마에는 `caseRefs`가 없고 `additionalProperties: false`라서, 실행 정의 문서의 실제 응답이 스키마를 어긴다(입력 스키마 `DocumentInput`에는 있다).

영역 본문에는 이 밖에도 코드 읽기로만 나온 후보가 있다. 같은 지표가 두 이름(`block_height`와 `chain_head_block`, `peer_count`와 `p2p_peers`)으로 나오는 것, PID가 0인 노드가 조용히 빠지는 것, 오류 응답 모양이 다섯 가지인 것, 기록 64 MiB 상한이 500을 낼 수 있는 것, 생산자 기본값 4가 네 곳에 있는 것, 역할 이름이 `admin`과 `administrator`로 갈린 것 등이다. 3-2단계에서 실제 표본으로 하나씩 확인한다.

## 실제 표본으로 확인한 것 (2026-10-25)

3-2단계의 첫 수집이다. 로컬에서 stablenet 4노드(gstable 1.1.0, 커밋 `0937ac5c`)를 Web API로만 띄웠다. 일부러 실패하는 시험과 통과하는 시험을 한 번씩 돌린 뒤 노드를 남겨 두고, 30분 동안 1분마다 네트워크, 스냅샷, 작업 목록을, 5분마다 지표, 관측, 노드별 로그를 받았다. 10분째에 node4를 멈추고 20분째에 다시 켰다. SSE는 처음 2분을 받았다.

이 표본을 처음 저장소에 넣었다가, 2026-10-27에 같은 방법으로 새 서버(#471~#475 반영)에서 다시 받아 바꿨다. 새 표본에는 실행 중인 작업 응답 두 개가 더 있다. `tests/webui/fixtures/stablenet-sample/`에는 API마다 대표 응답 하나, node4가 멈췄던 구간, 30분 지표 계열 전체가 있다. 런타임 절대 경로는 `/RUNTIME`으로, 장비 식별 해시와 계정 해시와 CSRF 토큰은 1, 2, 3을 64번 이은 값으로 바꿨다. `tests/webui/stablenet-sample.test.mjs`가 바꾼 값이 남지 않았는지 확인하고, 모든 응답을 화면의 해독 함수에 통과시킨다. 아래 사실 가운데 고친 것(누락 사유 없는 구멍, 섞인 타입, 0년 시각)은 새 표본에서 고쳐진 모습으로, 아직 남은 것(목록의 `unknown`, 내용 없는 SSE)은 그대로 확인한다.

- **노드 상태가 API마다 다르다.** `/observations`는 node4를 `running`으로 보였다가, 멈춘 동안은 `stopped`와 `observationReason: "no_matching_process"`로, 다시 켠 뒤에는 새 PID와 `running`으로 정확히 보였다. 같은 시점에 `/networks`와 `/snapshot`은 네 노드 모두 줄곧 `unknown`이었다. 실제 상태를 관측하는 길은 이미 있고, 다른 두 응답이 그것을 쓰지 않는다.
- **멈춘 노드의 지표 구멍에 누락 사유가 없다.** node4의 지표는 610초 동안 표본이 없는데(다른 노드 372개, node4 251개), 응답의 `coverage.gaps`에는 수집 시작 전 구간 하나뿐이다. 수집기가 PID 0인 노드를 사유 없이 건너뛰기 때문이다(`internal/app/web_monitor.go`의 수집 반복문). 화면은 이 구간을 정상으로 그린다. 인벤토리 본문의 "PID 0인 노드가 조용히 빠진다"는 주장을 이것으로 확인했다.
- **같은 지표가 두 이름으로 온다.** 한 응답에 노드마다 `block_height`와 `chain_head_block`, `peer_count`와 `p2p_peers`가 함께 있다. 표본 간격은 5초다.
- **노드 ID 형식이 API마다 다르다.** `/networks`와 `/observations`는 `node1`을 주고, 로그 API는 `<networkId>.node1`을 요구한다. `node1`로 부르면 404이고 본문은 JSON `{code, message, requestId}`다. 화면은 두 값을 이어 붙여 부른다.
- **SSE `observation` 이벤트는 1초마다 오지만 내용이 없다.** 본문은 `cursor`, `version`, `observedAt`, 전달 실패 수뿐이고 노드 상태나 지표는 들어 있지 않다. 2분 동안 `observation` 120개와 처음의 `resync_required` 하나가 왔다. 계획 6절에서 정한 "한 스트림으로 지표, 로그, 노드 상태를 보낸다"는 아직 구현되지 않은 상태다.
- **로그 한 쪽은 100줄이다.** node4는 30분에 11,363줄이었다. 레벨은 `INFO`와 `WARN`만 나왔다.
- **실행 기록의 시험 결과는 타입이 섞여 있다.** `assert.json`의 `Expected`는 숫자 `0`, `Actual`은 문자열 `"6"`이다. `ID`와 `On`은 빈 문자열이고 같은 값이 `Provenance` 안에 있다. 세션에는 끝 시각이 없어 `summary.missingDimensions`가 `["finishedAt"]`이다.
- **진행 문장에 구조화된 정보가 묻혀 있다.** 작업의 `partialEffects`에 "nodemonitor r1 node1 READY: alive, rpc ready, advancing"처럼 노드 준비 상태가 문장으로만 있고, 바이너리 경로가 런타임 절대 경로로 나온다.
- **단계 시각의 `0001-01-01`은 표본에 담기지 않았다.** 끝난 작업의 단계는 모두 시작과 끝 시각이 있었다. 이 값은 실행 중인 단계에서만 나오는데, 표본은 작업이 모두 끝난 뒤에 받았다. 직렬화 실험으로 따로 확인했다.

## 관측

이 절은 네트워크와 노드, 노드의 현재 상태, 지표, 로그, 실시간 스트림을 다룬다. 서버 쪽 시작점은 `internal/dashboard/jobs.go`(목록, 관측, 스냅샷, 이벤트), `internal/dashboard/monitor.go`(지표, 로그), `internal/dashboard/server.go`와 `sessions.go`(옛 경로)이다. 모든 `/api/` 경로와 `/events`는 `internal/dashboard/security.go:44-138`의 보안 처리기를 지난다. 이 처리기는 로그인 쿠키를 요구하고, 응답 본문 전체를 `deployments.RedactWeb`로 한 번 더 가린다(`security.go:53`). 오류 응답은 거의 모두 `http.Error`가 만든 평문(`text/plain`)이고, `deploymentError`가 오류 종류에 따라 404/403/409/500, 그 밖은 422를 고른다(`internal/dashboard/deployment.go:325-360`).

공통 시간 형식: Go `time.Time`은 RFC 3339(나노초 포함 가능) 문자열로 나간다. 이 영역에는 문자열로 나가는 큰 정수가 없다. 자세한 내용은 아래 "큰 정수 처리" 절에 적는다.

---

### `GET /api/v1/networks`

- **응답 Go 타입**: 익명 `map[string]any{"items": []app.WebNetwork, "nextCursor": nil}` (`internal/dashboard/jobs.go:49-52`). 원소 타입은 `app.WebNetwork`(`internal/app/web_networks.go:50-56`)와 `app.WebNode`(`web_networks.go:33-49`)이다.
- **원본 데이터**: `<deploymentRoot>/networks/<workspaceId>/chain-record.json`이 있는 workspace만 고른다(`web_networks.go:62-71`). 노드 표는 CLI 하위 프로세스 `chainbench chain status --json`으로 읽는다(`web_networks.go:75`, 인터페이스 설명 `web_networks.go:19-20`). 서버 신원은 `web-target.json`에서 읽는다(`web_networks.go:80-84`). 노드 프로세스는 확인하지 않고 `state`를 언제나 `"unknown"`으로 채운다(`web_networks.go:94`, `:101`).
- **갱신 방식**: 서버는 요청마다 workspace 수만큼 CLI를 돌려 새로 만든다. 캐시는 없다. 화면은 열 때 한 번 부른다(`NodeDetail.svelte:24`, `JobDetail.svelte:95`, `DSLEditor.svelte:25`, `JobPanel.svelte:73`, `:78`). 주기 갱신은 없다.
- **쓰는 화면**: `web/src/NodeDetail.svelte`, `web/src/JobDetail.svelte`, `web/src/JobPanel.svelte`, `web/src/DSLEditor.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[]` | array | 빈 배열일 수 있다 | | workspace 목록 중 chain-record.json이 있는 것 | 예 (`NodeDetail.svelte:24`) |
| `items[].id` | string | 아니오 | workspace id | `w.ID` (`web_networks.go:85`) | 예 (`NodeDetail.svelte:25`) |
| `items[].workspaceId` | string | 아니오 | id와 같은 값 | `w.ID` (`web_networks.go:85`) | 예 (`NodeDetail.svelte:27`, `JobDetail.svelte:95`, `DSLEditor.svelte:25`) |
| `items[].ownership` | string | 아니오 | 언제나 `"owned"` | 상수 (`web_networks.go:85`) | 예 (`NodeDetail.svelte:48`) |
| `items[].version` | number | 아니오 | workspace revision 정수 | `w.Revision` (`web_networks.go:85`) | 아니오 |
| `items[].nodes[]` | array | 빈 배열일 수 있다 | | CLI `chain status`의 `Nodes` | 예 (`NodeDetail.svelte:15`) |
| `items[].nodes[].id` | string | 아니오 | 노드 label(`node1` 또는 이름 label) | `ns.NodeLabel()` (`web_networks.go:101`, `internal/core/node/record.go:74-79`) | 예 (`NodeDetail.svelte:15`, `DSLEditor.svelte:25`) |
| `items[].nodes[].networkId` | string | 아니오 | workspace id | `w.ID` | 아니오 |
| `items[].nodes[].role` | string | 아니오 | `bp`, `en`, `pn` 등 | `ns.Role` | 예 (`NodeDetail.svelte:47`) |
| `items[].nodes[].hostIdentity` | string | 빈 문자열일 수 있다 | 서버 신원 문자열 | `web-target.json`의 `HostIdentity` | 예 (`NodeDetail.svelte:52`) |
| `items[].nodes[].dataPath` | string | 빈 문자열일 수 있다 | 대상 서버의 절대 경로 | `ns.DataDir` | 예 (`NodeDetail.svelte:53`) |
| `items[].nodes[].pid` | number | 0일 수 있다 | 기록된 PID. 생존 증거가 아니다 | `ns.PID` | 예 (`NodeDetail.svelte:50`) |
| `items[].nodes[].observedPid` | number | omitempty, 이 경로에서는 늘 빠진다 | | 채우지 않는다 | 이 경로에서는 해당 없음 |
| `items[].nodes[].observationReason` | string | omitempty, 이 경로에서는 늘 빠진다 | | 채우지 않는다 | 이 경로에서는 해당 없음 |
| `items[].nodes[].binaryAssetId`, `binarySHA256` | string | omitempty, 이 경로에서는 늘 빠진다 | | 채우지 않는다 | 이 경로에서는 해당 없음 |
| `items[].nodes[].state` | string | 아니오 | 언제나 `"unknown"` | 상수 (`web_networks.go:94`) | 예 (`NodeDetail.svelte:49`) |
| `items[].nodes[].supportedControls` | array of string | 빈 배열일 수 있다 | `node.start` 등 | 기록된 PID와 per-node binary 유무로 계산 (`web_networks.go:87-100`) | 예 (`node-view.mjs:14`) |
| `items[].nodes[].controlLimits` | object(string→string) | 빈 객체일 수 있다 | 한국어 문장 | `webControlLimits` (`internal/app/web_control_limits.go:11-37`) | 예 (`node-view.mjs:14`, `:16`) |
| `items[].nodes[].observedAt` | string | 아니오 | RFC 3339. 실제 관측 시각이 아니라 응답을 만든 시각이다 | `time.Now()` (`web_networks.go:101`) | 이 경로 값은 관측 전 문구로 가려진다 (`NodeDetail.svelte:49`) |
| `nextCursor` | null | 언제나 null | | 상수 | 아니오 |

- **문제**: `state`는 관측 없이 언제나 `"unknown"`인데, 같은 응답의 `supportedControls`는 기록된 PID가 0보다 크고 per-node binary가 없으면 `node.start`, `node.stop`, `node.restart`, (en/pn이면) `node.reset`을 내놓는다(`web_networks.go:87-100`). 서버 주석(`web_networks.go:58-59`)과 OpenAPI 설명은 "기록된 PID는 생존 증거가 아니다"라고 하므로 서로 맞지 않는다. 계획 단계에서 서버가 다시 검사하는지는 이 영역에서 확인하지 않았다. OpenAPI(`docs/dev/web-ui.openapi.yaml:2413-2440`)는 `items`를 `type: object`로만 적어 노드 구조를 밝히지 않고, `nextCursor`도 적지 않는다. CLI가 실패하면 그 오류가 `deploymentError`로 가서 대부분 422 평문이 된다(`deployment.go:326`). workspace 하나라도 CLI 실패하면 목록 전체가 실패한다(`web_networks.go:75-78`).

### `GET /api/v1/workspaces/{workspaceId}/networks`

- **응답 Go 타입**: 위와 같은 `{"items": []app.WebNetwork, "nextCursor": nil}` (`internal/dashboard/jobs.go:39-48`).
- **원본 데이터**: `jobs.Networks()` 전체를 읽은 뒤 서버 메모리에서 `WorkspaceID`로 거른다(`jobs.go:40-46`).
- **갱신 방식**: 요청마다 새로 만든다.
- **쓰는 화면**: 없다. `web/src`에서 `workspaces/.../networks`를 부르는 곳이 없다(grep `api/v1/workspaces` 결과는 export뿐, `DeploymentEditor.svelte:65`).

필드는 `GET /api/v1/networks`와 같다.

- **문제**: OpenAPI(`web-ui.openapi.yaml:1180-1186`)는 요약을 "Read actual owned and attached network state"라고 적지만, 실제 값은 관측하지 않은 기록이고(`state`가 늘 `unknown`) `ownership`은 늘 `owned`이다. 화면은 이 경로 대신 전체 목록을 받아 브라우저에서 workspace로 거른다(`JobDetail.svelte:95`, `DSLEditor.svelte:25`).

### `GET /api/v1/networks/{networkId}/observations`

- **응답 Go 타입**: `app.WebNetwork` (`internal/app/web_networks.go:50-56`). 처리기는 `internal/dashboard/jobs.go:53-60`, 계산은 `internal/app/web_node_observations.go:19-163`.
- **원본 데이터**: `chain-record.json`을 직접 읽고(`web_node_observations.go:28-38`), `web-target.json`, 분산 배치면 `web-targets.json`(`:56-66`)을 읽는다. 호출자 본인의 자격증명으로 대상 서버를 확인한 뒤(`:90-104`) CLI 하위 프로세스 `chainbench chain observe --json --workspace-dir <dir> --server-logins <logins>`를 돌린다(`:189`). CLI 쪽 판정은 `internal/chainsetup/observe_nodes.go:26-50`이 `process.ObserveLaunch`, `process.ObserveVacancy`(`internal/core/process/ownership.go:55-137`)로 한다. 원장 대조는 `verifyWebRecordLedger`(`web_node_observations.go:76`)이고, per-node binary가 있으면 등록 자산 근거를 `observedWebBinary`로 찾는다(`:130-141`).
- **갱신 방식**: 사용자가 누를 때만 부른다. `NodeDetail.svelte:36`, `:57`의 "노드 상태 확인" 버튼과 `JobPanel.svelte:98`이다. 서버 요청 제한 시간은 30초다(`jobs.go:56`). 결과는 저장하지 않는다.
- **쓰는 화면**: `web/src/NodeDetail.svelte`, `web/src/JobPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `id`, `workspaceId` | string | 아니오 | workspace id | `workspace.ID` (`web_node_observations.go:47`) | 예 (`JobPanel.svelte:161`) |
| `ownership` | string | 아니오 | 늘 `"owned"` | 상수 (`:47`) | 아니오(NodeDetail은 목록 값을 읽는다, `NodeDetail.svelte:48`) |
| `version` | number | 아니오 | workspace revision | `workspace.Revision` | 아니오 |
| `nodes[].id` | string | 아니오 | 노드 label | `ns.NodeLabel()` (`:49`) | 예 (`NodeDetail.svelte:16`, `JobPanel.svelte:161`) |
| `nodes[].role` | string | 아니오 | | `ns.Role` | 예 (`NodeDetail.svelte:47`) |
| `nodes[].hostIdentity` | string | 빈 문자열일 수 있다 | | 노드 서버별 기록 기계 (`:73-75`) | 예 (`NodeDetail.svelte:52`) |
| `nodes[].dataPath` | string | 빈 문자열일 수 있다 | | `ns.DataDir` | 예 (`NodeDetail.svelte:53`) |
| `nodes[].pid` | number | 0일 수 있다 | 기록 PID | `ns.PID` | 예 (`NodeDetail.svelte:50`, `JobPanel.svelte:161`) |
| `nodes[].observedPid` | number | omitempty. running, unrecorded_running일 때만 값이 있다 | 실제 PID | `LaunchVerdict.PID` (`:126`, `ownership.go:46`) | 예 (`NodeDetail.svelte:50`, `JobPanel.svelte:161`) |
| `nodes[].observationReason` | string | omitempty. running이면 보통 빠진다 | 사유 코드(아래 "노드 상태 어휘" 절) | `seen.Reason` 또는 Web 덮어쓰기 (`:49`, `:123`, `:127`, `:136`, `:143`) | 예. NodeDetail은 코드를 그대로 보이고(`NodeDetail.svelte:50`), JobPanel은 한국어로 바꾼다(`JobPanel.svelte:18`, `:161`) |
| `nodes[].binaryAssetId` | string | omitempty | 등록 자산 id | `binding.Evidence.ID` (`:139`) | 예 (`NodeDetail.svelte:54`, `JobPanel.svelte:161`) |
| `nodes[].binarySHA256` | string | omitempty | 소문자 hex 64자 | `binding.Evidence.SHA256` (`:140`) | 예 (`NodeDetail.svelte:54`, `JobPanel.svelte:161`) |
| `nodes[].state` | string | 아니오 | 아래 "노드 상태 어휘" 절 | CLI 판정 또는 Web 덮어쓰기 (`:125`, `:123`, `:135`) | 예 (`NodeDetail.svelte:49`, `JobPanel.svelte:161`) |
| `nodes[].supportedControls` | array of string | 빈 배열일 수 있다 | | 관측 결과로 계산 (`:144-155`) | 예 (`node-view.mjs:14`) |
| `nodes[].controlLimits` | object | 빈 객체일 수 있다 | 한국어 문장 | `webControlLimits` (`:159-161`) | 예 (`node-view.mjs:16`) |
| `nodes[].observedAt` | string | 아니오 | RFC 3339. 관측 실패 노드는 응답을 만든 시각 | `time.Now()` (`:49`, `:128`) | 예 (`NodeDetail.svelte:49`, `JobPanel.svelte:161`) |

- **문제**: 관측 도중 실패해도 200으로 답하고 모든 노드를 `unknown`/`probe_unavailable`로 남긴다. 그 지점은 다섯 곳이다. 대상 조회 실패(`web_node_observations.go:92-95`), 대상 확인 실패(`:98-101`), 기록 대상과 실제 대상 불일치(`:102-104`), CLI 실패(`:109-112`), 분산 배치인데 `web-targets.json`이 없을 때(`:64-66`)이다. 화면은 어느 단계가 실패했는지 알 수 없다. 서버 권한은 `canEdit`(admin, operator)인데(`internal/app/web_jobs.go:135-143`, `web_chain_engine.go:118-126`), `NodeDetail.svelte:57`은 viewer에게도 버튼을 보이므로 viewer는 403 평문을 받는다. OpenAPI `Node` 스키마(`web-ui.openapi.yaml:3080-3132`)와 다른 점은 셋이다. 스키마에는 `ports`가 있지만 `WebNode`에는 없다. `observationReason` enum에 CLI가 내는 `machine_unavailable`(`observe_nodes.go:30`)이 없다. `state` enum의 `starting`, `failed`는 이 코드에서 만들어지지 않는다. `web-target.json`이 없으면 409(`ErrDeploymentConflict`, `:39-42`)로 답한다.

### `GET /api/v1/networks/{networkId}/metrics`

- **응답 Go 타입**: `app.WebMetrics` (`internal/app/web_monitor_read.go:75-79`), 안의 `WebMetricSeries`(`:66-73`), `WebMetricSample`(`:61-64`), `WebCoverage`(`:48-59`), `WebCoverageGap`(`:42-46`). 처리기는 `internal/dashboard/monitor.go:36-39`, 질의 해석은 `monitor.go:47-76`.
- **원본 데이터**: `<deploymentRoot>/observations/<networkId>/samples-YYYYMMDD.jsonl`(UTC 날짜별, `web_monitor_read.go:147-148`, `web_monitor.go:336-339`)과 `gaps.jsonl`, 수집 위치 파일 `state.json`(`web_monitor_read.go:316-363`). 이 파일들은 서버 안 수집기 `WebMonitor.Run`이 5초마다 쓴다(`web_monitor.go:22`, `:99-110`, `cmd/chainbench-dashboard/main.go:189-195`). 수집기는 노드별로 RPC `eth_blockNumber`, `net_peerCount`(`web_monitor.go:262-273`)와 Prometheus endpoint `/debug/metrics/prometheus`(`web_monitor.go:279`, `internal/core/collector/metrics.go:21`)를 직접 HTTP로 읽는다. 노드 목록은 매 회차 CLI `chain status --json`으로 읽는다(`web_monitor.go:145`).
- **질의 두 형태**:
  - `?from=&to=` (RFC 3339): 비우면 끝은 지금, 시작은 끝에서 2시간 전이다. 최대 31일이다(`web_monitor_read.go:121-134`). 거꾸로 된 구간은 400 평문 `to precedes from`이다(`monitor.go:65-67`).
  - `?runId=`: history 실행의 job 창을 쓴다(`internal/app/web_monitor_archive.go:32-54`). `from`/`to`와 함께 주면 400이다(`:36-38`). 창은 job 생성 시각부터 마지막 phase 종료까지이고, 진행 중이면 끝을 지금으로 한다(`internal/app/web_history_observations.go:21-62`). 실행 창은 31일 제한을 받지 않는다(`web_monitor_archive.go:52`). 다른 네트워크의 실행이면 404다(`:49-51`).
  - 그 밖에 `nodeId`(label), `metric`으로 서버가 거를 수 있다(`web_monitor_read.go:149`). `limit`은 1~100만 받는데 지표에는 쓰이지 않는다(`monitor.go:68-74`).
- **갱신 방식**: 서버는 요청 때 보관 파일을 처음부터 훑어 계산한다. 화면 쪽은 다음과 같다. `NodeMetrics.svelte`는 범위가 바뀔 때와, 최근 N분 프리셋이면 5초마다 다시 부른다(`NodeMetrics.svelte:57`, `:59`). `NodeDetail.svelte:34`는 열 때 최근 10분을 한 번 부른다. `FailureMetrics.svelte:24`, `CauseBench.svelte:30`은 구간이 바뀔 때 부른다. `HistoryPanel.svelte:66`, `:72`는 `runId` 형태로 상세를 열거나 버튼을 누를 때 부른다.
- **쓰는 화면**: `web/src/NodeMetrics.svelte`(MonitorPanel 안), `web/src/NodeDetail.svelte`, `web/src/FailureMetrics.svelte`(RunTimeline 안), `web/src/CauseBench.svelte`(JobDetail 안), `web/src/RecorderStrip.svelte`, `web/src/Readings.svelte`, `web/src/HistoryPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `networkId` | string | 아니오 | | 경로 값 | 아니오 |
| `series[]` | array | 빈 배열일 수 있다 | node, source, name 조합으로 정렬 | samples 파일 (`web_monitor_read.go:145-182`) | 예 (`metric-chart.mjs:9`) |
| `series[].nodeId` | string | 아니오 | 노드 label만(`node1`). network 접두어가 없다 | 표본 `n` | 예 (`metric-chart.mjs:9`, `NodeMetrics.svelte:40`, `HistoryPanel.svelte:67`, `node-view.mjs:4`) |
| `series[].name` | string | 아니오 | 아래 "지표 이름표" 절 | 표본 `m` | 예 (`metric-chart.mjs:9`, `:100`, `:126`) |
| `series[].unit` | string | omitempty | `blocks`, `peers`, `transactions` | 표본 `u` | 예 (`metric-chart.mjs:31`, `:100`) |
| `series[].source` | string | 아니오 | `rpc` 또는 `metrics` | 표본 `s` | 예 (`metric-chart.mjs:31`, `:100`, `NodeDetail.svelte:51`) |
| `series[].labels` | object | omitempty. 늘 빠진다 | | 채우지 않는다 | 아니오 |
| `series[].samples[]` | array | 빈 배열은 나오지 않는다(표본이 있어야 계열이 생긴다) | 시간순 | 표본 | 예 (`metric-chart.mjs:10`) |
| `series[].samples[].time` | string | 아니오 | RFC 3339. 수집 회차 시각, 다운샘플이면 구간 시작 시각 | 표본 `t` (`web_monitor_read.go:159`, `:385`) | 예 (`metric-chart.mjs:21`, `recorder.mjs:27`) |
| `series[].samples[].value` | number 또는 null | 타입은 null을 허용하지만 실제로는 null이 나오지 않는다 | float64 JSON 숫자. 다운샘플이면 구간 평균 | 표본 `v` (`web_monitor_read.go:158-159`) | 예 (`metric-chart.mjs:22`, `recorder.mjs:29`) |
| `coverage.from`, `coverage.to` | string | 아니오 | RFC 3339 UTC | 해석한 창 | 예 (`NodeMetrics.svelte:67`) |
| `coverage.sampleIntervalSeconds` | number | 아니오 | 초. 늘 5 | `m.interval` (`web_monitor_read.go:189`) | 예 (`NodeMetrics.svelte:19`, `:67`, `metric-chart.mjs:16`, `recorder.mjs:19`) |
| `coverage.complete` | boolean | 아니오 | 누락 구간이 하나도 없으면 true | `len(gaps)==0` | 예 (`NodeMetrics.svelte:66-67`, `HistoryPanel.svelte:139`) |
| `coverage.gaps[]` | array | 빈 배열일 수 있다 | | 아래 "누락 사유 목록" 절 | 예 (`metric-chart.mjs:48`, `:90`, `RecorderStrip.svelte:22`, `HistoryPanel.svelte:139`) |
| `coverage.gaps[].from`, `.to` | string | 아니오 | RFC 3339 | 누락 기록 | 예 (`metric-chart.mjs:49`, `:92`) |
| `coverage.gaps[].reason` | string | 아니오 | `"<node> <source> <code>"`를 공백으로 이은 문자열 | `web_monitor_read.go:357`, `:360` | 예 (`labels.mjs:26-33`, `RecorderStrip.svelte:22`) |
| `coverage.downsampled` | boolean | 아니오 | 한 계열이라도 720점을 넘으면 true | `web_monitor_read.go:176-179` | 예 (`NodeMetrics.svelte:67`) |
| `coverage.bucketSeconds` | number | omitempty, 원본 표본이면 빠진다 | 초 | `(to-from)/720` | 예 (`metric-chart.mjs:16`, `NodeMetrics.svelte:19`, `:67`) |
| `coverage.timestampSource` | string | 지표에서는 늘 `"collection"` | | 상수 (`:189`) | 아니오(지표 쪽) |

- **문제**: 기록 PID가 0인 노드는 수집에서 아예 빠진다(`web_monitor.go:187-189`). 그래서 정지된 노드는 표본도 누락 구간도 없고, `complete`가 true로 나올 수 있다. 같은 개념이 두 이름으로 나간다. 블록 높이는 `block_height`(rpc)와 `chain_head_block`(metrics), 피어 수는 `peer_count`(rpc)와 `p2p_peers`(metrics)이다. 화면이 별칭 표로 하나를 고른다(`metric-chart.mjs:115-129`). 누락 사유가 구조 없이 문자열 하나로 합쳐져 나가서 화면이 다시 쪼갠다(`labels.mjs:26-33`, `RecorderStrip.svelte:22`). `<metric> not_exposed`와 `deleted_by_admin <runId>`는 낱말이 하나 더 붙어 화면 해석이 어긋난다(아래 "누락 사유 목록" 절). `WebMetricSample.Value`가 포인터라 OpenAPI는 null을 허용하지만 서버는 null을 만들지 않는다. 화면의 null 처리(`metric-chart.mjs:22`, `recorder.mjs:29`)는 쓰이지 않는 길이다. CLI `chain status`가 실패한 회차는 누락 기록 없이 오류만 돌려주고 끝나며(`web_monitor.go:161-163`), 15초가 지나야 읽기 시점에 `collector_stale`이 보인다(`web_monitor_read.go:331-333`). `limit`이 100을 넘으면 지표 요청도 400이 된다(`monitor.go:68-74`). OpenAPI 설명(`web-ui.openapi.yaml:1569`)은 실제 동작과 대체로 맞다.

### `GET /api/v1/nodes/{nodeId}/logs`

- **응답 Go 타입**: `app.WebLogs` (`internal/app/web_monitor_read.go:88-96`), 원소 `app.WebLogEntry`(`:81-86`). 처리기 `internal/dashboard/monitor.go:40-43`, 계산 `web_monitor_read.go:195-281`. 검색과 레벨은 `internal/app/web_logsearch.go`.
- **원본 데이터**: `nodeId`는 `"<networkId>.<label>"`이다. 마지막 `.`에서 자른다(`web_monitor_read.go:196-200`). label이 실제 노드인지 CLI `chain status`로 확인한다(`:204-214`). 줄은 `<deploymentRoot>/observations/<networkId>/logs/<label>-YYYYMMDD.jsonl`에서 읽는다(`:240`). 이 파일은 두 수집기가 쓴다. 로컬 노드는 5초 수집기가 노드 로그 파일을 직접 읽고(`web_monitor.go:198`, `web_monitor_log.go:103-190`), SSH 노드는 운영자가 시작한 `network.monitor` 작업이 그 운영자의 SSH 연결로 읽는다(`internal/app/web_network_monitor.go:184-242`, `web_monitor_remote.go:46-128`). 저장 전에 가린다(`web_monitor_log.go:194-210`).
- **질의**: `from`, `to`, `runId`는 지표와 같은 규칙이다(`web_monitor_read.go:229`). `q`는 가려진 본문에 대한 대소문자 무시 부분 문자열이고 200자를 넘으면 400이다(`web_logsearch.go:76-90`). `level`은 쉼표로 이은 `CRIT,ERROR,WARN,INFO,DEBUG,TRACE`이고 `EROR`, `DBUG`, `TRCE`도 받는다(`web_logsearch.go:33-67`). 모르는 레벨은 400 `unknown log level "X"`이다(`monitor.go:51-54`). `limit`은 1~100, 기본 100이다(`monitor.go:68-74`, `web_monitor_read.go:232-235`). `cursor`는 `<index>-<fromNano>-<toNano>[-<filter지문>]` 형식이고, 발급 때와 같은 `q`와 `level`에서만 받는다(`web_monitor_read.go:221-228`, `:285-311`).
- **갱신 방식**: 서버는 요청마다 창 안의 날짜 파일을 처음부터 끝까지 훑고 일치 줄 수를 센다. 화면은 입력이 바뀔 때 노드마다 첫 쪽을 따로 부르고(`LogViewer.svelte:61-78`, `:96`), 스크롤이 끝에 닿으면 `nextCursor`로 다음 쪽을 부른다(`:80-93`, `:130-132`). 주기 갱신은 없다. `CauseBench.svelte:19-24`는 구간이 바뀔 때 한 번 부르고 다음 쪽은 읽지 않는다.
- **쓰는 화면**: `web/src/LogViewer.svelte`(NodeDetail, NodeMetrics, RunTimeline, HistoryPanel, CauseBench 안), `web/src/CauseBench.svelte`, 그 결과를 받는 `web/src/RecorderStrip.svelte`(`events`, `otherEvents`).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `nodeId` | string | 아니오 | `<networkId>.<label>` | 경로 값 | 아니오 |
| `runId` | string | omitempty. 늘 빠진다 | | 채우지 않는다(`web_monitor_read.go:236`) | 아니오 |
| `entries[]` | array | 빈 배열일 수 있다 | 노드 로그 순서(파일에 쓰인 순서) | logs 파일 | 예 (`LogViewer.svelte:69`, `CauseBench.svelte:23`) |
| `entries[].time` | string | 아니오 | RFC 3339 UTC. 노드 로그의 시각이 있으면 그 시각, 없으면 수집 시각 | 줄 `t` (`web_monitor_log.go:215-231`) | 예 (`log-view.mjs:38`, `LogViewer.svelte:174`, `recorder.mjs:11`) |
| `entries[].text` | string | 빈 문자열일 수 있다 | 가린 본문. 64 KiB에서 자르고 `[truncated N bytes]`를 붙인다 | 줄 `x` (`web_monitor_log.go:206-208`) | 예 (`LogViewer.svelte:177`, `:184`) |
| `entries[].level` | string | omitempty. 레벨 접두어가 없는 줄은 빠진다 | `CRIT`~`TRACE` | 읽을 때 정규식으로 판정 (`web_monitor_read.go:244`, `web_logsearch.go:31`, `:42-48`) | 예 (`LogViewer.svelte:174-175`, `recorder.mjs:12`, `RecorderStrip.svelte:23-24`) |
| `total` | number | 아니오 | 창 전체에서 필터와 일치한 줄 수 | `matched` (`web_monitor_read.go:278`) | 예 (`LogViewer.svelte:71`) |
| `coverage.from`, `.to` | string | 아니오 | RFC 3339 | 창 | 예. 빈 결과 문구에서 쓴다 (`LogViewer.svelte:167`) |
| `coverage.sampleIntervalSeconds` | number | 로그에서는 늘 0 | | 채우지 않는다 (`:279`) | 아니오 |
| `coverage.complete` | boolean | 아니오 | | `len(gaps)==0` | 아니오 |
| `coverage.gaps[]` | array | 빈 배열일 수 있다 | `logs` 출처 중 이 노드 것, 그리고 `collector`, `archive` 출처 | `web_monitor_read.go:265-267` | 예 (`LogViewer.svelte:35-37`, `:161-165`) |
| `coverage.downsampled` | boolean | 늘 false | | | 아니오 |
| `coverage.timestampSource` | string | 아니오 | `source`, `collection`, `mixed`. 반환된 쪽의 줄만 보고 정한다 | `web_monitor_read.go:271-279` | 예 (`LogViewer.svelte:38`, `:45-50`) |
| `nextCursor` | string 또는 null | 마지막 쪽이면 null | 위 커서 형식 | `web_monitor_read.go:252-255` | 예 (`LogViewer.svelte:70`, `:89`) |

- **문제**: (첫 문제는 #466에서 고쳤다. 이제 `logQuery`가 `LOG_PAGE_MAX` 100에서 자른다.) `CauseBench.svelte:20`은 `limit=200`을 보내는데 서버는 100을 넘으면 400 `limit must be between 1 and 100`으로 거절한다(`monitor.go:68-74`). 선택 노드의 오류·경고 요청이 실패하면 `Promise.all` 전체가 실패해 지표도 오지 않고 오류 문구가 뜬다(`CauseBench.svelte:29-37`). 다른 노드 요청은 `.catch(()=>[])`로 조용히 빈 목록이 된다(`:32`). 실제 브라우저에서 돌려 보지는 않았고 코드만 대조했다. 다른 문제도 있다. 로그 응답의 누락 구간은 rpc, metrics 출처를 빼므로(`web_monitor_read.go:265-267`) 노드가 RPC에 답하지 않던 구간은 로그 화면에 나타나지 않는다. `timestampSource`는 이번 쪽에 담긴 줄만으로 정하므로 쪽마다 바뀔 수 있다(`:258`, `:271-277`). 줄마다의 출처(`WebLogEntry.TimestampSource`, `json:"-"`)는 나가지 않는다(`:85`). 요청마다 CLI `chain status`를 한 번 돌리므로(`:204`), "모든 노드" 보기는 노드 수만큼 CLI를 돌린다. 화면은 노드별 쪽을 받아 브라우저에서 시간순으로 합치는데(`log-view.mjs:32-39`), 노드마다 100줄씩만 받은 상태라 합친 목록의 뒤쪽은 다른 노드의 줄이 아직 오지 않은 순서일 수 있다. OpenAPI(`web-ui.openapi.yaml:1626`)의 설명은 동작과 맞지만 `Logs` 스키마의 `runId`는 실제로 나오지 않는다.

### `GET /api/v1/snapshot`

- **응답 Go 타입**: `app.WebJobSnapshot`(`internal/app/web_job_observation.go:14-20`)에 `busDeliveryDropsTotal`, `jobDeliveryDropsTotal`을 덧붙인 익명 구조체(`internal/dashboard/snapshot.go:58-62`).
- **원본 데이터**: 작업 저장소 메모리(`s.state.Jobs`)와, `GET /api/v1/networks`와 같은 `Networks()` 결과(`web_job_observation.go:95-139`). 네트워크 노드의 `supportedControls`와 `controlLimits`는 비운다(`:117-126`). `busDeliveryDropsTotal`은 옛 이벤트 버스의 전달 실패 수(`snapshot.go:53-56`), `jobDeliveryDropsTotal`은 작업 구독자 전달 실패 수(`web_job_observation.go:75-80`, `:86-91`)다.
- **질의**: `workspaceId`, `jobId`만 받고 그 밖은 400 `invalid snapshot scope`(`snapshot.go:14-19`). `jobId`가 없는 작업이면 404.
- **갱신 방식**: `job-stream.svelte.js:17`이 스트림을 열기 전, 그리고 끊김, 버전 건너뜀, 재시작을 알아챘을 때 다시 부른다(`job-stream.svelte.js:10-42`, `job-observation.mjs:14-31`). 화면은 질의 없이 부른다.
- **쓰는 화면**: `web/src/job-stream.svelte.js`를 통해 `web/src/MonitorPanel.svelte`, `web/src/Dashboard.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `version` | number | 아니오 | uint64. 감사 기록 길이 | `len(s.state.Audit)` (`web_job_observation.go:101`) | 예 (`job-observation.mjs:3`, `MonitorPanel.svelte:48`) |
| `cursor` | string | 아니오 | `<32자 hex 인스턴스>:<version>` | `streamCursor` (`:46-48`) | 예 (`job-stream.svelte.js:22`, `job-observation.mjs:4`) |
| `jobs[]` | array | 빈 배열일 수 있다 | 생성 시각 내림차순 | 작업 저장소. 필드는 작업 영역 문서가 다룬다 | 예 (`Dashboard.svelte:15`, `MonitorPanel.svelte:24`) |
| `networks[]` | array of `WebNetwork` | 빈 배열일 수 있다 | `GET /api/v1/networks`와 같다. 단 `supportedControls`는 `[]`, `controlLimits`는 `{}` | `Networks()` | 예 (`Dashboard.svelte:18`, `MonitorPanel.svelte:22`) |
| `networks[].nodes[].state` | string | 아니오 | 늘 `"unknown"` | `web_networks.go:94` | 예 (`dashboard-view.mjs:9-10`, `Dashboard.svelte:62`, `MonitorPanel.svelte:61`) |
| `networks[].nodes[].pid` | number | 0일 수 있다 | | | 예 (`MonitorPanel.svelte:61`) |
| `networks[].nodes[].id`, `.role` | string | 아니오 | | | 예 (`Dashboard.svelte:62`, `MonitorPanel.svelte:45`, `:61`) |
| `networks[].ownership` | string | 아니오 | 늘 `owned` | | 예 (`Dashboard.svelte:61`, `MonitorPanel.svelte:61`) |
| `observedAt` | string | 아니오 | RFC 3339. 스냅샷을 만든 시각 | `time.Now()` | 예 (`job-observation.mjs:10`, `job-stream.svelte.js:21`, `MonitorPanel.svelte:60`) |
| `busDeliveryDropsTotal` | number | 아니오 | uint64 누적 | `s.bus.Dropped()` | 예 (`job-stream.svelte.js:21`, `MonitorPanel.svelte:53`) |
| `jobDeliveryDropsTotal` | number | 아니오 | uint64 누적 | `s.feed.dropped` | 예 (`job-stream.svelte.js:21`, `MonitorPanel.svelte:53`) |

- **문제**: 노드 `state`가 늘 `unknown`이라 `Dashboard.svelte`의 집계(`dashboard-view.mjs:7-12`, `Dashboard.svelte:20`, `:30`, `:59-61`)는 실행 0, 정지 0, 기타 전체로만 나온다. `Dashboard.svelte:57`은 "작업이 마지막으로 기록한 상태입니다"라고 적지만 실제로는 서버가 넣은 상수이다. 스냅샷 안의 `networks`는 이벤트로 갱신되지 않으므로(아래 `/api/v1/events` 참고) 스냅샷을 다시 받을 때까지 그대로다. 스냅샷 하나를 만들 때마다 workspace 수만큼 CLI `chain status`가 돈다. 작업이 많이 커밋되면 세 번까지 다시 시도하고 그래도 맞지 않으면 409다(`web_job_observation.go:96-138`).

### `GET /api/v1/events` (SSE)

- **응답 Go 타입**: 데이터는 `app.WebJobChange`(`internal/app/web_job_observation.go:22-33`)와 heartbeat용 익명 map(`internal/dashboard/snapshot.go:134-135`). 처리기 `snapshot.go:65-143`.
- **원본 데이터**: 작업 저장소가 커밋할 때마다 만드는 변경 기록(`web_job_observation.go:60-82`). 최근 128건을 보관해 다시 보낸다(`:12`, `:71-74`, `:176-180`). 구독자는 64명까지다(`:147-149`).
- **질의와 헤더**: `cursor`, `workspaceId`, `jobId`만 받는다(`snapshot.go:66-72`). `Last-Event-ID` 헤더가 `cursor`보다 앞선다(`:86-89`). 범위를 주면 다른 작업의 `job_changed`는 `job`을 지우고 `version_changed`로 바꿔 보낸다(`:116-121`).
- **갱신 방식**: 서버는 커밋 때 즉시 보내고, 1초마다 heartbeat를 보낸다(`snapshot.go:105`, `:130-141`). 매 이벤트와 heartbeat마다 로그인을 다시 확인하고 실패하면 닫는다(`:107`, `:113`, `:131`). 화면은 스냅샷 뒤에 `?cursor=`로 연결하고, 오류 시 2초 뒤 스냅샷부터 다시 한다(`job-stream.svelte.js:22-28`). 마지막 관측에서 5초가 지나면 stale로 표시한다(`MonitorPanel.svelte:18`).
- **쓰는 화면**: `web/src/job-stream.svelte.js`, `web/src/job-observation.mjs`, 이를 쓰는 `web/src/MonitorPanel.svelte`, `web/src/Dashboard.svelte`.

SSE 이벤트 종류:

| SSE `event:` | `id:` 줄 | 데이터 필드 | 보내는 곳 | 화면 처리 |
|---|---|---|---|---|
| `job_changed` | 있다(`cursor`) | `type`, `version`, `cursor`, `job`(작업 전체, 계획 원문 제외), `observedAt` | 커밋된 작업이 있을 때 (`web_job_observation.go:66-70`) | 버전이 정확히 하나 늘면 작업을 목록 맨 앞에 넣는다 (`job-observation.mjs:23-30`) |
| `version_changed` | 있다 | `type`, `version`, `cursor`, `observedAt`. `job` 없음 | 작업이 사라진 커밋, 또는 범위 밖 작업 (`web_job_observation.go:65`, `snapshot.go:118-121`) | 버전만 올린다 (`job-observation.mjs:30`) |
| `resync_required` | 있다 | `type`, `version`, `cursor`, `reason`(`initial_snapshot_required`, `cursor_invalid`, `server_restarted`, `cursor_expired`), `missingFrom`, `missingTo`(만료일 때만), `observedAt` | 구독 시작 때 커서를 받아 줄 수 없을 때 (`web_job_observation.go:152-174`) | 스냅샷을 다시 받는다. 만료면 건너뛴 건수를 문구로 보인다 (`job-stream.svelte.js:33`) |
| `observation` (heartbeat) | 없다 | `version`, `cursor`, `jobDeliveryDropsTotal`, `observedAt` | 1초마다 (`snapshot.go:134-138`) | 서버 버전이 앞서 있으면 다시 맞춘다. 마지막 관측 시각과 드롭 수를 바꾼다 (`job-observation.mjs:21`, `job-stream.svelte.js:34-35`) |

데이터 필드 표(공통 `WebJobChange`):

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `type` | string | 아니오 | 위 네 값 | | 아니오. SSE 이벤트 이름을 대신 쓴다 (`job-stream.svelte.js:32`) |
| `version` | number | 아니오 | uint64 | | 예 (`job-observation.mjs:3`, `:21-24`) |
| `cursor` | string | 아니오 | `<인스턴스>:<version>` | | 예 (`job-observation.mjs:4`, `:30`) |
| `job` | object | omitempty | 작업 | | 예 (`job-observation.mjs:27-28`) |
| `reason` | string | omitempty | | | 예 (`job-observation.mjs:20`) |
| `missingFrom`, `missingTo` | number | omitempty | | | 예 (`job-stream.svelte.js:33`) |
| `observedAt` | string | 아니오 | RFC 3339 | | 예 (`job-stream.svelte.js:34`) |
| `jobDeliveryDropsTotal` (heartbeat만) | number | heartbeat에만 있다 | | `StreamPosition` | 예 (`job-stream.svelte.js:35`) |

- **문제**: 이 스트림은 작업 변화만 나른다. 노드 상태, 지표, 로그, 네트워크 목록의 변화는 오지 않는다. 그래서 화면의 지표는 5초 폴링(`NodeMetrics.svelte:57`), 로그는 손으로 새로 고침, 노드 상태는 버튼 관측에 기댄다. OpenAPI(`web-ui.openapi.yaml:1542-1563`)는 이벤트 이름과 페이로드를 설명문으로만 적고 스키마가 없다. heartbeat 이름이 옛 `/events`의 heartbeat와 같은 `observation`인데 데이터 모양이 다르다.

### `GET /events` (옛 SSE)

- **응답 Go 타입**: 데이터 메시지는 `collector.Event`(`internal/core/collector/event.go:33-41`), heartbeat는 익명 map(`internal/dashboard/server.go:139`). 처리기 `server.go:116-159`.
- **원본 데이터**: 대시보드 프로세스 안의 이벤트 버스(`cmd/chainbench-dashboard/main.go:91`). 버스에 넣는 길은 `POST /api/events`(`server.go:173-194`)뿐이고, 그것을 부르는 쪽은 `--dashboard` URL을 받은 CLI 실행이다(`cmd/chainbench/suitecmd/run.go:271`, `cmd/chainbench/lifecyclecmd/verify.go:32`). Web 작업 엔진은 `--dashboard`를 넘기지 않는다(`internal/app`, `internal/webcli`에서 grep 결과 없음).
- **갱신 방식**: 버스 이벤트는 즉시, heartbeat는 1초마다 보낸다(`server.go:129-143`). 로그인이 풀리면 닫는다(`server.go:136`, `:148`, `security.go:36`). 화면은 앱이 열릴 때 한 번 연결하고, 최근 200건을 메모리에 둔다(`App.svelte:66-79`).
- **쓰는 화면**: `web/src/App.svelte`(연결 표시, 모니터링 화면의 이벤트 타임라인), `web/src/Dashboard.svelte`(`busDrops`, `connected`를 props로 받는다, `App.svelte:119`).

| SSE `event:` | 데이터 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| 이름 없음(`message`) | `time` | string | 아니오(버스가 채운다) | `Event.Time` | 예 (`App.svelte:83`, `:131`) |
| | `phase` | string | 아니오 | `setup`, `verify`, `test` (`event.go:13-17`) | 예 (`App.svelte:131`) |
| | `kind` | string | 아니오 | `info`, `progress`, `result`, `error` 등 열린 문자열 (`event.go:21-28`) | 예 (`App.svelte:76`, `:131`) |
| | `network` | string | omitempty | | 예 (`App.svelte:131`) |
| | `node` | number | omitempty. 0이면 빠진다 | 노드 index | 예. `node${e.node}`로 이름을 지어 낸다 (`App.svelte:131`) |
| | `message` | string | 빈 문자열일 수 있다 | | 예 (`App.svelte:131`) |
| | `fields` | object | omitempty | 이벤트별 구조 자료 | 아니오 |
| `observation` | `busDeliveryDropsTotal` | number | 아니오 | `s.bus.Dropped()` | 예 (`App.svelte:71`) |
| | `observedAt` | string | 아니오 | `time.Now()` | 예 (`App.svelte:71`, `:41`) |

- **문제**: 데이터 메시지에 `id:`가 없어 끊긴 사이의 이벤트를 다시 받을 수 없다. `fields`는 화면이 읽지 않으므로, 테스트 엔진이 보내는 chainstate 이벤트(`message: "chainstate"`, `fields`에 heights, peers, bpParticipation, forked, `internal/testengine/collect.go:130-143`)는 화면에 "chainstate"라는 글자로만 보인다. 노드 이름을 `node${index}`로 지어 내므로 이름 label을 쓴 노드와 맞지 않는다. OpenAPI에는 `/events` 경로가 없다(`web-ui.openapi.yaml`에서 `^  /events` grep 결과 없음).

### `GET /api/runs` (옛 경로)

- **응답 Go 타입**: `[]collector.RunRecord` (`internal/core/collector/store.go:26-35`), 처리기 `internal/dashboard/server.go:162-169`.
- **원본 데이터**: 대시보드의 메모리 저장소 `collector.NewMemStore()`(`cmd/chainbench-dashboard/main.go:93`). 대시보드 코드 안에서 이 저장소에 `SaveRun`을 부르는 곳이 없다(grep 결과는 `FileStore` 내부뿐, `internal/core/collector/filestore.go:40`, `:49`). 그래서 응답은 늘 `[]`이다.
- **갱신 방식**: 화면은 앱이 열릴 때와 옛 이벤트의 `kind`가 `result`일 때 부른다(`App.svelte:57-59`, `:65`, `:76`). 대시보드의 "새로고침"도 이것을 부른다(`App.svelte:119`).
- **쓰는 화면**: `web/src/App.svelte`, `web/src/Dashboard.svelte`(`:71-73`).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `[]` (최상위 배열) | array | 늘 빈 배열이다 | | MemStore | 예 (`Dashboard.svelte:71`) |
| `[].id`, `.phase`, `.chain`, `.network` | string | | | | 예 (`Dashboard.svelte:73`) |
| `[].status` | string | | `running`, `succeeded`, `failed`, `skipped` (`store.go:12-20`) | | 예 (`Dashboard.svelte:73`) |
| `[].started_at`, `[].ended_at` | string | `ended_at`는 omitempty이지만 time.Time이라 빠지지 않고 0시각이 나간다 | snake_case | | 아니오 |
| `[].result` | object | omitempty | | | 아니오 |

- **문제**: OpenAPI(`web-ui.openapi.yaml:1935-1966`)는 `{items: [...]}`로 적지만 실제는 맨 배열이다. 필드 이름이 snake_case라 다른 v1 응답(camelCase)과 다르다. `Dashboard.svelte:23`과 `App.svelte:85`의 `runOk`는 `success`, `pass`를 찾지만 실제 상태 값은 `succeeded`이고, 둘 다 쓰이지 않는다. 늘 빈 배열이므로 대시보드의 "엔진 실행 인덱스" 표(`Dashboard.svelte:72-73`)는 나타나지 않는다.

### `GET /api/sessions`, `GET /api/sessions/{id}`, `GET /api/sessions/{id}/chainstate` (+ `{proc}/{id}` 형태)

- **응답 Go 타입**: 목록은 `[]string`(`internal/dashboard/sessions.go:15-27`). 상세는 `session.json` 바이트를 그대로 쓴다(`sessions.go:30-47`). chainstate는 `[]json.RawMessage`(`sessions.go:51-70`)이고 한 줄의 모양은 `testengine.chainstateRecord`(`internal/testengine/chainstate_sink.go:16-22`)이다.
- **원본 데이터**: `--artifact-root` 아래 세션 디렉터리(`session.List`), `<root>/<id>/session.json`(`internal/core/session/read.go:145-148`), `<root>/<id>/environments/*/chainstate/*.jsonl`(`read.go:150-162`). chainstate 파일은 테스트 엔진 수집기가 표본마다 한 줄씩 쓴다(`internal/testengine/collect.go:60`, `chainstate_sink.go:44-56`). 두 마디 id는 `{proc}/{id}`로 읽는다(`sessions.go:95-108`).
- **갱신 방식**: 목록은 앱이 열릴 때와 옛 `result` 이벤트 때 부른다(`App.svelte:60-62`, `:65`, `:76`). 상세와 chainstate는 화면이 부르지 않는다(`web/src`에서 `/chainstate` grep 결과 없음).
- **쓰는 화면**: 목록만 `web/src/App.svelte:61`이 받아 `sessions` 상태에 넣고, 그 값을 어디에도 그리지 않는다(`App.svelte:38`, `:61` 외 사용처 없음).

chainstate 한 줄:

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `[].seq` | number | 아니오 | 1부터 늘어나는 표본 번호. 시각이 없다 | `chainstateSink.seq` | 아니오 |
| `[].heights` | object(string→number) | omitempty | 노드 이름별 블록 높이. uint64를 JSON 숫자로 쓴다 | `collector.Chainstate.Heights` | 아니오 |
| `[].peers` | object(string→number) | omitempty | 노드 이름별 피어 수 | `Chainstate.Peers` | 아니오 |
| `[].bpParticipation` | object(string→number) | omitempty | 생산자 주소별, 최근 창 안의 블록 수 | `tallyBP` (`internal/core/collector/collector_impl.go:220-230`) | 아니오 |
| `[].forked` | boolean | 아니오 | 같은 높이에서 다른 해시를 본 적이 있으면 true(한 번 true면 계속 true) | `recordSample` (`collector_impl.go:170-184`) | 아니오 |

- **문제**: OpenAPI(`web-ui.openapi.yaml:1967-2060`)는 목록을 `{items: [...]}`로 적지만 실제는 문자열 맨 배열이다. 상세는 404일 때 평문 `404 page not found`이다(`sessions.go:41-43`). chainstate 표본에는 시각이 없어 Web 지표 보관본과 시간축으로 맞출 수 없다.

---

### 노드 상태 어휘와 출처

`state` 값은 세 곳에서 정해진다.

1. **목록과 스냅샷**: 언제나 `unknown`이다(`internal/app/web_networks.go:94`).
2. **관측 경로, CLI 판정**: `process.Ownership`(`internal/core/process/ownership.go:23-41`)이다. `running`(기록한 실행 그대로 살아 있음), `missing`(기록 PID가 없어짐), `ownership_mismatch`(PID가 다른 실행에 재사용됨), `unknown`(판정 불가), `stopped`(PID 기록이 없고 같은 실행도 없음), `unrecorded_running`(PID 기록은 없는데 같은 실행이 돌고 있음).
3. **관측 경로, Web 덮어쓰기**: 원장이 맞지 않으면 `ownership_mismatch`/`ledger_mismatch`(`web_node_observations.go:122-124`), per-node binary 근거가 없으면 `ownership_mismatch`/`binary_binding_unavailable`(`:134-137`), 대상 서버를 확인하지 못하면 `unknown`/`probe_unavailable`(`:49`).

`observationReason` 값과 만드는 곳:

| 코드 | 만드는 곳 | 뜻 |
|---|---|---|
| `probe_unavailable` | `ownership.go:56`, `web_node_observations.go:49` | PID 생존이나 실행 인자를 읽지 못했다. Web 쪽은 관측 자체를 못 했을 때의 기본값이다 |
| `no_recorded_pid` | `ownership.go:58` | `ObserveLaunch`에 PID 0이 들어왔다(실제로는 `ObserveVacancy`로 가므로 드물다) |
| `recorded_pid_absent` | `ownership.go:67` | 기록 PID가 살아 있지 않다 |
| `argv_mismatch` | `ownership.go:76` | 기록 PID의 실행 인자가 기록과 다르다 |
| `discovery_unavailable` | `ownership.go:93` | PID 기록 없는 노드의 후보 프로세스를 끝까지 확인하지 못했다 |
| `unrecorded_pid_found` | `ownership.go:120` | 기록에 없는 같은 실행을 찾았다 |
| `unrecorded_argv_mismatch` | `ownership.go:128` | 같은 데이터 경로나 설정을 쓰는 다른 실행이 있다 |
| `no_matching_process` | `ownership.go:136` | 같은 실행이 없다(stopped) |
| `machine_unavailable` | `internal/chainsetup/observe_nodes.go:30` | CLI가 노드의 기계에 닿지 못했다 |
| `ledger_mismatch` | `web_node_observations.go:123` | 실행 기록과 원장이 맞지 않는다 |
| `binary_binding_unavailable` | `web_node_observations.go:136` | per-node binary의 등록 근거가 없다 |
| `binary_launch_unavailable` | `web_node_observations.go:143` | 기록된 바이너리를 대상에서 실행할 수 있는지 확인되지 않았다(`Executable != "ready"`) |

화면 쪽 어휘는 두 벌이다. `JobPanel.svelte:17-18`이 상태와 사유를 한국어로 바꾸고, `labels.mjs:35-42`의 `STATES`가 `StatusBadge`용 낱말을 준다. `STATES`에는 `missing`, `ownership_mismatch`, `unrecorded_running`이 없어 코드가 그대로 보인다. `JobPanel.svelte:18`에는 `machine_unavailable`이 없어 "관측할 수 없음"으로 뭉뚱그려진다. `NodeDetail.svelte:50`은 사유 코드를 바꾸지 않고 그대로 보인다.

### 누락 사유 목록과 만드는 곳

보관 형식은 `{from, to, n(node), s(source), r(reason)}`(`internal/app/web_monitor.go:63-69`)이다. API는 이를 `"<n> <s> <r>"`를 공백으로 이은 `reason` 하나로 바꾸고 앞뒤 공백을 지운다(`internal/app/web_monitor_read.go:357`).

| 보관 `r` | `s` | 노드 붙음 | 만드는 곳 | 언제 |
|---|---|---|---|---|
| `collector_stopped` | collector | 아니오 | `web_monitor.go:177` | 수집기 프로세스가 새로 떠서 지난 회차 이후를 보증할 수 없을 때 |
| `collector_error` | collector | 아니오 | `web_monitor.go:207` | samples 파일 쓰기 실패 |
| `collector_error` | logs | 예 | `web_monitor.go:201`, `web_monitor_remote.go:92` | 로그 읽기 오류 |
| `remote_log_collection_unavailable` | logs | 예 | `web_monitor.go:194` | SSH 대상인데 `network.monitor` 작업이 돌고 있지 않을 때 |
| `rpc_disabled` | rpc | 예 | `web_monitor.go:260` | 노드에 HTTP RPC 포트가 없을 때 |
| `rpc_unavailable` | rpc | 예 | `web_monitor.go:268` | 2초 안에 높이나 피어 수를 얻지 못했을 때 |
| `metrics_disabled` | metrics | 예 | `web_monitor.go:276` | metrics 포트가 없을 때 |
| `metrics_unavailable` | metrics | 예 | `web_monitor.go:281` | Prometheus 읽기 실패 |
| `<지표이름> not_exposed` | metrics | 예 | `web_monitor.go:287` | 바이너리가 그 gauge를 내놓지 않을 때 |
| `log_unavailable` | logs | 예 | `web_monitor_log.go:112` | 로그 파일이 없을 때 |
| `log_rotated` | logs | 예 | `web_monitor_log.go:121`, `:128` | inode나 첫 256바이트가 바뀌었을 때 |
| `log_truncated` | logs | 예 | `web_monitor_log.go:133` | 파일이 읽은 위치보다 작아졌을 때 |
| `log_line_oversized` | logs | 예 | `web_monitor_log.go:164` | 한 번 읽기(1 MiB) 안에 줄바꿈이 없을 때 |
| `deleted_by_admin <runId>` | archive | 아니오 | `web_monitor_archive.go:71` | 관리자가 history 실행을 지웠을 때 |
| `collector_stale` | collector | 아니오 | `web_monitor_read.go:332` (읽을 때 계산, 저장 안 함) | 마지막 회차가 15초(3회차)보다 오래되었을 때 |
| `not_collected` | collector | 아니오 | `web_monitor_read.go:339` (읽을 때 계산) | 창의 시작이 첫 수집보다 앞설 때 |
| `archive corrupt_record <N>` | (API 문자열 자체) | 아니오 | `web_monitor_read.go:360` (읽을 때 계산) | 보관 줄을 해석하지 못했을 때 |

진행 중인 누락은 `state.json`의 `open`, `remoteOpen`에 있다가 끝나야 `gaps.jsonl`에 옮겨진다(`web_monitor.go:71-85`, `:209-213`, `:221-246`). API는 둘을 합쳐 보인다(`web_monitor_read.go:327`).

화면 해석(`labels.mjs:26-33`)과 어긋나는 곳이 둘이다. `node1 metrics txpool_pending not_exposed`는 마지막 낱말 `not_exposed`를 코드로 읽는데, 사전(`labels.mjs:5-19`)에 없어 코드가 그대로 보이고 어느 지표인지는 `gapSummary`가 덧붙이지 않는다. `archive deleted_by_admin r123`은 마지막 낱말 `r123`을 코드로 읽어 실행 id가 사유 자리에 나온다. `RecorderStrip.svelte:22`는 첫 낱말이 `node`로 시작하거나 `.`을 가지면 노드로 여기므로, `faucet` 같은 이름 label의 누락은 네트워크 전체 누락으로 잘못 보인다.

### 지표 이름표

| 이름 | 단위 | source | 원본 | 뜻 |
|---|---|---|---|---|
| `block_height` | blocks | rpc | `eth_blockNumber` (`web_monitor.go:264`, `:271`) | 노드의 head 블록 높이 |
| `peer_count` | peers | rpc | `net_peerCount` (`web_monitor.go:265`, `:272`) | 연결된 피어 수 |
| `chain_head_block` | blocks | metrics | Prometheus gauge (`web_monitor.go:29`) | head 블록 높이(geth gauge) |
| `p2p_peers` | peers | metrics | Prometheus gauge (`web_monitor.go:30`) | 피어 수 |
| `txpool_pending` | transactions | metrics | Prometheus gauge (`web_monitor.go:31`) | 실행 대기 트랜잭션 수 |
| `txpool_queued` | transactions | metrics | Prometheus gauge (`web_monitor.go:32`) | 순서 대기 트랜잭션 수 |

수집 주기는 5초(`web_monitor.go:22`), 기본 창은 2시간, 최대 창은 31일, 계열당 720점을 넘으면 같은 너비 구간의 평균으로 줄인다(`web_monitor_read.go:22-27`, `:373-400`). Prometheus 값은 label을 버리고 같은 이름의 마지막 값을 쓴다(`internal/core/collector/metrics.go:47-51`). 화면은 `txpool_queued`를 실패 계기판(`metric-chart.mjs:115-119`)에서 쓰지 않고, 모니터링 차트에서만 그린다.

### 큰 정수 처리

- 지표 값은 모두 `float64`로 보관하고 JSON 숫자로 나간다(`web_monitor.go:54-61`, `web_monitor_read.go:61-64`). RPC 높이와 피어 수는 `uint64`로 받은 뒤 `float64`로 바꾼다(`internal/core/rpc/client.go:135`, `:153`, `web_monitor.go:271-272`). 2^53을 넘는 높이는 정밀도를 잃지만 현실의 블록 높이에서는 문제가 되지 않는다. 문자열로 나가는 큰 정수는 이 영역에 없다. wei나 `big.Int`도 없다.
- 스냅샷과 이벤트의 `version`, `busDeliveryDropsTotal`, `jobDeliveryDropsTotal`, `missingFrom`, `missingTo`는 `uint64`를 JSON 숫자로 쓴다. 화면은 `Number.isSafeInteger`로 `version`을 검사한다(`job-observation.mjs:3`).
- 세션 chainstate의 `heights`는 `uint64`를 JSON 숫자로 쓴다(`chainstate_sink.go:18`).
- 로그 커서 안의 시각은 UnixNano 정수를 문자열에 넣는다(`web_monitor_read.go:285-291`). 화면은 커서를 해석하지 않고 그대로 돌려준다.

### 디스크 보관본과 API의 차이

배치 위치는 `--deployment-root`이다(`cmd/chainbench-dashboard/main.go:189`). 보관소는 `<root>/observations/`이고, 이름 검사와 경로 규칙은 `internal/core/session/observationstore.go:16-35`에 있다.

| 파일 | 줄 모양 | API에 나오는 것 | API에 나오지 않는 것 |
|---|---|---|---|
| `<root>/observations/<net>/samples-YYYYMMDD.jsonl` | `{t,n,m,u,s,v}` (`web_monitor.go:54-61`) | 모두(창, 노드, 지표로 거르고, 720점 초과면 평균) | 없음. 단 다운샘플된 응답에는 원래 표본이 없다 |
| `<root>/observations/<net>/gaps.jsonl` | `{from,to,n,s,r}` (`web_monitor.go:63-69`) | 합친 구간과 이은 문자열 `reason` | `n`, `s`, `r`의 구분. 지표 응답에서는 logs 출처, 로그 응답에서는 rpc와 metrics 출처가 빠진다 |
| `<root>/observations/<net>/logs/<label>-YYYYMMDD.jsonl` | `{t,c,ts,x}` (`web_monitor_log.go:33-38`) | `t`→`time`, `x`→`text`, 레벨은 읽을 때 계산 | `c`(수집 시각), 줄마다의 `ts`(시각 출처). 응답에는 이번 쪽 전체의 출처 하나만 나온다 |
| `<root>/observations/<net>/state.json` | `webMonitorCursor` (`web_monitor.go:74-85`) | `open`, `remoteOpen` 누락 구간(합쳐서), `firstCollected`는 `not_collected` 계산에, `lastCollected`는 `collector_stale` 계산에 쓰인다 | `firstCollected`, `lastCollected`, `remoteLast` 자체, 로그 위치(`offsets`, `files`, `heads`, `skipping`, `inSecret`) |
| `<root>/networks/<ws>/chain-record.json` | `chainsetup.State` (`internal/chainsetup/state.go:35` 이하) | 노드 id, role, dataDir, PID | 포트(`p2p`, `http`, `ws`, `auth`, `metrics`, `etcd`, `etcdClient`, `internal/core/node/node.go:58-74`), `host`, `server`, `logPath`, `configPath`, `syncMode`, `binary`, `args`, `chain`, `compositionId`, `bp` 수, genesis 경로 등 |
| `<root>/networks/<ws>/web-target.json`, `web-targets.json` | `resource.Inspection`, 서버별 기계 목록 | `hostIdentity` | `transport`(local/ssh) 등 나머지 검사 결과, 서버 이름과 기계의 대응 |

---

### 있는데 화면에 오지 않는 데이터

- **세션 chainstate(`chainstate.jsonl`)**: 테스트 실행마다 `<artifactRoot>/<session>/environments/*/chainstate/chainstate.jsonl`에 `heights`, `peers`, `bpParticipation`, `forked`가 쌓인다(`internal/testengine/collect.go:60`, `chainstate_sink.go:16-56`). `GET /api/sessions/{id}/chainstate`가 내주지만 화면은 부르지 않는다. `internal/app`에도 `chainstate`를 읽는 코드가 없어 v1 history 응답에도 나오지 않는다. 생산자 참여(`bpParticipation`)와 포크 감지(`forked`)는 Web 5초 수집기가 전혀 모으지 않는 값이라, 작업 상세의 원인 영역(`CauseBench.svelte`, `RecorderStrip.svelte`)이나 history 상세에 쓰일 수 있다. 다만 표본에 시각이 없어(`seq`만 있다) 지금 모양으로는 시간축에 놓을 수 없다.
- **엔진 쪽 수집기(`internal/core/collector`)**: 테스트 실행 중 `collector.Snapshot()`이 가진 `Chainstate`와 블록 생산자, head 해시 기록(`collector_impl.go:140-184`)은 옛 버스 이벤트로만 나가고(`collect.go:130-143`), Web 작업은 `--dashboard`를 넘기지 않아 그마저 오지 않는다. 같은 수집기의 로그 tail(`collector_impl.go:110-116`)과 `LogMatch`(파일, 줄 범위, 바이트 위치, `collector.go:24-31`)도 Web 로그 보관본과 따로 논다. 단언이 어느 로그 줄에 걸렸는지는 세션 산출물에 있을 수 있으나 이 영역 API로는 나오지 않는다.
- **생명주기 상태(`internal/core/lifecycle`)**: CLI `chain up` 비교 경로는 `At`(끝난 위치)과 `Failed`(설계된 실패 상태 이름)를 결과에 담는다(`internal/chainsetup/verb/compare.go:30-39`, `:72-77`). 관측 API는 이 값을 다루지 않는다. 작업 영역에서 쓰는지는 `internal/app`에서 `.Failed`를 찾지 못했다는 것까지만 확인했다. 노드나 네트워크가 지금 생명주기의 어느 단계에 있는지는 화면이 알 수 없다.
- **노드 기록의 포트와 경로**: `chain-record.json`의 포트, `host`, `logPath`, `configPath`, `syncMode`, `args`, `server`는 수집기가 쓰지만 API로 나오지 않는다(위 표). OpenAPI `Node.ports`가 자리를 마련해 두었으나 구현이 없다. 노드 상세의 "RPC" 칸(`NodeDetail.svelte:51`)은 지표 값만 보인다.
- **CLI 관측의 나머지**: `chain observe`는 노드마다 `executable`(`ready`, `unavailable`, `unknown`)과 `recordedPid`, `server`를 준다(`internal/chainsetup/observe_nodes.go:11-21`). Web은 `executable`을 사유 하나로 줄이고(`web_node_observations.go:142-148`) 나머지는 버린다.
- **관측 실패 단계**: 관측 경로가 조용히 멈춘 다섯 지점(위 observations 문제)의 원인 오류는 버려진다.
- **수집 상태**: 마지막 수집 시각(`lastCollected`), 첫 수집 시각, SSH 로그 수집 작업이 지금 붙어 있는지(`WebMonitor.remote`, `web_monitor_remote.go:19-39`)는 API로 나오지 않는다. 화면은 누락 구간이 있는지로만 짐작한다.
- **로그 줄의 수집 시각과 줄별 시각 출처**: 보관본의 `c`, `ts`(위 표).
- **옛 버스 이벤트의 `fields`**: 화면이 읽지 않는다(`App.svelte:131`).
- **`txpool_queued`**: 수집하고 모니터링 차트에는 그리지만, 실패 원인 계기판과 기록계 띠에는 넣지 않는다(`metric-chart.mjs:115-119`).

### 화면이 직접 계산하는 것

| 함수(파일:줄) | 무엇을 계산하나 | 서버 읽기 모델 후보인가 |
|---|---|---|
| `gapText` (`web/src/labels.mjs:26-33`) | 이은 `reason` 문자열을 노드, 코드로 다시 쪼개고 한국어로 바꾼다 | 강한 후보다. 서버가 `{node, source, code, detail}`로 나누어 주면 `not_exposed`와 `deleted_by_admin`의 해석 오류가 사라진다 |
| `RecorderStrip.svelte:22`의 `gaps` | 누락 사유의 첫 낱말이 이 노드인지 정규식으로 판정한다 | 후보다. 위와 같은 구조화로 해결된다 |
| `failureMetrics` (`web/src/metric-chart.mjs:115-129`) | `block_height`/`chain_head_block`, `peer_count`/`p2p_peers` 중 있는 쪽을 고른다 | 후보다. 서버가 개념 이름(높이, 피어)과 출처를 따로 주면 된다 |
| `chartModel` (`metric-chart.mjs:8-32`) | 2배 주기보다 벌어진 표본 사이에서 선을 끊는다 | 그리기 계산이라 화면에 둔다. 끊는 기준은 서버의 누락 구간과 겹치므로 서버가 구간을 정확히 주면 단순해진다 |
| `latestValues` (`metric-chart.mjs:58-62`), `latestSamples` (`web/src/node-view.mjs:3-8`) | 노드별 마지막 값과 노드 사이 차이(뒤처짐) | 후보다. "노드별 최신 값" 읽기 모델이 있으면 노드 상세와 모니터링이 같은 값을 쓴다 |
| `measureBetween`, `readingRows` (`web/src/recorder.mjs:40-65`), `WRONG` 규칙(`:34`) | 같은 시각에 다른 모든 노드보다 높이가 낮거나, 피어가 적거나, 대기 tx가 많은 값을 이상으로 표시한다 | 후보다. 분석 규칙이 화면에만 있어 history 비교나 다른 화면과 공유되지 않는다 |
| `cursorDefaults` (`recorder.mjs:10-17`) | 첫 경고와 첫 오류로 커서 A, B를 정한다 | 화면 동작이라 화면에 둔다. 다만 첫 오류 시각은 서버 요약으로 줄 수 있다 |
| `RecorderStrip.svelte:15-20`의 `otherBlock` | 다른 노드 중 가장 높은 블록을 시각별로 고른다 | 후보다(네트워크 최고 높이 계열) |
| `mergeByTime` (`web/src/log-view.mjs:32-39`) | 노드별 로그 쪽을 시간순으로 합친다 | 후보다. 서버가 여러 노드를 한 커서로 합쳐 주지 않아 화면의 합친 순서가 쪽 경계에서 불완전하다 |
| `stampText` (`web/src/LogViewer.svelte:45-50`), `gaps`/`gapWords` (`:35-37`) | 노드별 시각 출처와 누락 사유를 합친다 | 위 다중 노드 읽기가 생기면 함께 옮길 수 있다 |
| `nodeTally` (`web/src/dashboard-view.mjs:7-12`) | 노드 상태별 수를 센다 | 서버 상태가 늘 `unknown`이라 지금은 의미 없는 계산이다. 실제 관측 상태가 생기면 서버 집계 후보다 |
| `loadRunNodes` (`web/src/HistoryPanel.svelte:62-69`) | 실행 창의 지표 계열에서 노드 목록을 뽑는다 | 후보다. 실행이 쓴 노드 목록은 서버가 알고 있으므로 history 상세가 주는 편이 맞다. 지금은 표본이 없는 노드가 목록에서 빠진다 |
| `NodeMetrics.svelte:40`의 `allNodes`, `MonitorPanel.svelte:45` | 지표 계열과 스냅샷에서 각각 노드 목록을 만든다 | 두 목록이 다를 수 있다(지표가 없는 노드). 서버 노드 목록 하나로 맞추는 편이 낫다 |
| `nodeControls` (`node-view.mjs:13-20`) | 서버가 준 제어 목록과 제한 사유, 실행 중 작업을 합친다 | 실행 중 작업 판정(`NodeDetail.svelte:31`, `MonitorPanel.svelte:25`)이 화면마다 따로 있어 서버 쪽 판정 후보다 |
| `applyJobChange`, `snapshotState` (`web/src/job-observation.mjs:8-31`) | 스트림 버전 순서를 검사하고 작업 목록을 갱신한다 | 프로토콜 처리라 화면에 둔다 |
| `rangedEvents` (`web/src/App.svelte:83`) | 옛 이벤트를 모니터링 시간 범위로 거른다 | 이 연결에서 받은 200건만 대상이라 서버 보관이 없으면 옮길 것이 없다 |
| `valueAt` (`web/src/FailureMetrics.svelte:35-38`), `nearestPoints` (`metric-chart.mjs:67-75`) | 실패 시각에 가장 가까운 표본을 2주기 안에서 고른다 | 후보다. 원인 영역 세 화면이 같은 규칙을 따로 쓴다 |

## 작업과 실행 기록

이 영역은 세 저장소에서 읽는다. 작업과 계획은 서버 데이터 루트의 `jobs.json` 한 파일에 계획, 작업, 멱등 키, 감사 기록이 함께 들어 있다(`internal/core/session/jobstore.go:19`, `internal/app/web_jobs.go:29-34`). 실행 기록은 `history.json` 한 파일에 캡처 사본, 삭제 묘비, 감사 기록이 들어 있고 64 MiB를 넘으면 읽기와 쓰기를 모두 거절한다(`internal/core/session/historystore.go:22-36`, `internal/app/web_history_contract.go:77-81`). 엔진 세션 원본은 Web이 실행한 것은 `<root>/sessions` 아래에(`internal/app/web_test_run.go:406-408`), CLI가 만든 것은 별도 artifact root 아래에 있다(`internal/app/web_history.go:101-104`).

라우트는 `internal/dashboard/jobs.go:32-145`와 `internal/dashboard/history.go:35-83`에 등록된다. 모든 라우트가 로그인을 요구하고 `Cache-Control: no-store`를 붙인다(`jobs.go:17-28`, `history.go:17-33`). 오류 응답은 보안 래퍼가 `{code, message, requestId, ...}` JSON으로 바꾸며, 400·409·422만 서버가 적은 이유를 그대로 싣고 나머지는 상태 문구만 싣는다(`internal/dashboard/security.go:276-340`). 역할 문자열은 사용자 계정의 `administrator`를 작업 쪽 행위자 역할 `admin`으로 바꿔 쓴다(`security.go:22-25`). 화면은 계정 역할 `administrator`로 비교한다(`web/src/JobCard.svelte:13`, `web/src/HistoryPanel.svelte:15`).

### `POST /api/v1/plans`

- **응답 Go 타입**: `app.WebPlan` (`internal/app/web_job_contract.go:72-86`). 요청은 `app.WebPlanInput` (`web_job_contract.go:12-24`). 성공은 201 (`internal/dashboard/jobs.go:61-68`).
- **원본 데이터**: 엔진 `Prepare`가 만든 `WebPreparedJob`(`web_job_contract.go:62-70`)에서 계산한다. `WebChainEngine.Prepare`가 대상 기계를 검사하고 단계 이름, 바뀌는 내용, 필요한 접근, 자원 주장(claim)을 정한다(`internal/app/web_chain_engine.go:511-540`). 테스트 작업은 `chainbench run ... --plan` 하위 프로세스로 구성 계획을 받는다(`internal/app/web_plan_cli.go:14-40`). 결과는 `jobs.json`에 `savedWebPlan{public,input,prepared}`로 저장된다(`internal/app/web_jobs.go:19-23`, `web_jobs.go:208`).
- **갱신 방식**: 화면이 "계획 만들기"를 누를 때 한 번 부른다(`web/src/JobPanel.svelte:135-138`). 계획은 만든 뒤 15분 동안만 유효하다(`web_jobs.go:198`).
- **쓰는 화면**: `web/src/JobPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `id` | string | 아니오 | 32자 hex (`deployment_store.go:165-171`) | `deploymentID()` (`web_jobs.go:198`) | 예 (`JobPanel.svelte:132`, `:140`) |
| `actorId` | string | 아니오 | 계정 ID | 요청한 행위자 | 아니오 |
| `workspaceId` | string | 아니오 | | 요청 본문 | 아니오 |
| `operation` | string | 아니오 | 작업 ID(아래 `contracts/jobs` 목록) | 요청 본문, 검증은 엔진 | 아니오 |
| `documentRefs[]` | array of `{id, revision}` | 요청이 비우면 null 가능 | | 요청 본문 그대로 | 예 (`JobPanel.svelte:167`) |
| `targetFingerprint` | string | 아니오 | 64자 hex SHA-256 (`web_jobs.go:146-151`) | `prepared.fingerprint` | 아니오 |
| `expiresAt` | string | 아니오 | RFC 3339, 생성 시각 + 15분 | 서버 시계 | 예 (`JobPanel.svelte:170`) |
| `phases[]` | array of string | 엔진이 nil을 주면 null | 단계 이름 | `prepared.phases` (`web_chain_engine.go:522-531`) | 예 (`JobPanel.svelte:170`) |
| `changes[]` | array of string | 엔진이 nil을 주면 null | 사람이 읽는 영어 문장 | `prepared.changes` (`web_chain_engine.go:533-538`) | 예 (`JobPanel.svelte:169`) |
| `resources[]` | array of string | 아니오(빈 배열로 시작) | `"<hostIdentity>:<dataPath>"` 또는 `"<hostIdentity>:process:<executable>"` | claim에서 문자열로 만든다 (`web_jobs.go:199-205`) | 예 (`JobPanel.svelte:170`) |
| `requiredAccess[]` | array of string | 엔진이 nil을 주면 null | 예 `"local filesystem and process access"` | `web_chain_engine.go:532` | 예 (`JobPanel.svelte:165`) |
| `validation` | object | 아니오 | 항상 `{"valid":true,"errors":[],"warnings":[]}` | 하드코딩 (`web_jobs.go:198`) | 아니오 |
| `chainSurfaces[]` | array of `{chainId, assetId, fingerprint, contractVersion}` | 아니오(nil이면 빈 배열로 바꾼다, `web_jobs.go:195-197`) | | `app.WebChainSurfaceRef` (`web_chain_surface.go:113-118`) | 예 (`JobPanel.svelte:168`) |

- **문제**: `validation`은 언제나 유효로 고정된 값이라 정보가 없다. 실패는 4xx 오류로만 온다(`web_jobs.go:173-191`). 화면은 계획에 넣은 자격증명 연결을 응답이 아니라 자기가 보낸 요청(`planInput`)에서 다시 보여 준다(`JobPanel.svelte:135`, `:166`). 응답에 그 값이 없기 때문이다. 부족한 SSH 접근은 422 본문의 `details.missingAccess`로 오고(`internal/dashboard/deployment.go:334-339`) 화면이 읽는다(`JobPanel.svelte:137`). `resources` 문자열은 claim의 `ports`를 빼고 만들기 때문에 포트만 주장한 claim은 `"host:"`로 끝난다(`web_jobs.go:204`). 계획의 `phases`, `changes`, `requiredAccess`가 null이 될 수 있는지는 엔진마다 다르다. 체인 엔진은 늘 채우지만 인터페이스가 이를 보장하지 않는다.

### `GET /api/v1/plans/{planId}/conflicts`

- **응답 Go 타입**: 익명 맵 `{"items": []app.WebResourceConflict, "observedAt": time.Time}` (`internal/dashboard/jobs.go:69-72`). 항목 타입은 `app.WebResourceConflict` (`internal/app/web_resource_holds.go:13-21`).
- **원본 데이터**: 메모리의 `jobs.json` 상태. 저장된 계획의 claim과 다른 작업들의 claim이 겹치는지 계산한다(`web_resource_holds.go:25-39`, `:47-95`, 겹침 판정 `web_job_contract.go:37-58`). 끝난 작업도 정리가 확인되지 않았으면 자원을 잡고 있다고 본다(`web_resource_holds.go:41-42`, `:101`). 실행 바이너리 claim은 네트워크 기록이 아직 실행 중이라고 적은 바이너리로 판단한다(`web_resource_holds.go:68-83`).
- **갱신 방식**: 계획을 만든 직후, 실행 직전, "충돌 다시 확인" 클릭, 작업 목록 새로고침 때 부른다(`JobPanel.svelte:73`, `:132`, `:138`, `:140`). 서버는 부를 때마다 새로 계산한다.
- **쓰는 화면**: `web/src/JobPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[]` | array | 빈 배열 가능, null 아님 (`web_resource_holds.go:48`) | | `resourceConflicts` | 예 (`JobPanel.svelte:132`) |
| `items[].jobId` | string | 아니오 | 32자 hex | 자원을 잡은 작업 | 예 (`JobPanel.svelte:170`) |
| `items[].workspaceId` | string | 아니오 | | 그 작업 | 예 (`JobPanel.svelte:170`) |
| `items[].actorId` | string | 아니오 | | 그 작업 | 예 (`JobPanel.svelte:170`) |
| `items[].operation` | string | 아니오 | | 그 작업 | 예 (`JobPanel.svelte:170`) |
| `items[].state` | string | 아니오 | 작업 상태 문자열 | 그 작업 | 예 (`JobPanel.svelte:170`) |
| `items[].nodeDisposition` | string | 아니오 | 노드 처리 문자열 | 그 작업 | 예 (`JobPanel.svelte:170`) |
| `items[].resources[]` | array of `WebResourceClaim` | 아니오 | | 겹친 claim | 예 (`JobPanel.svelte:170`) |
| `items[].resources[].hostIdentity` | string | 아니오 | 물리 호스트 식별자 | 엔진 claim (`web_job_contract.go:30-35`) | 예 |
| `items[].resources[].dataPath` | string | omitempty, 실행 바이너리 claim이면 없음 | 절대 경로 | 엔진 claim | 예 |
| `items[].resources[].ports[]` | array of number | omitempty | 1-65535 | 엔진 claim | 예 |
| `items[].resources[].executable` | string | omitempty | 슬래시 없는 파일 이름 (`web_jobs.go:156`) | 엔진 claim | 예 |
| `observedAt` | string | 아니오 | RFC 3339 나노초, UTC | 서버 시계 | 예 (`JobPanel.svelte:132`, `:170`) |

- **문제**: 이 응답만 `items` 옆에 `nextCursor` 대신 `observedAt`을 단다. 계획이 만료되면 409, 다른 사람 계획이면 404이다(`web_resource_holds.go:31-37`). 실행(`POST /jobs`)이 자원 충돌로 거절될 때 오는 409 본문에도 같은 모양의 `details.holders`가 실리지만(`internal/dashboard/deployment.go:327-332`) 화면은 그것을 읽지 않고 이 엔드포인트를 다시 부른다(`JobPanel.svelte:140`). 같은 자료를 두 길로 받는다.

### `POST /api/v1/jobs`

- **응답 Go 타입**: `app.WebJob` (`internal/app/web_job_contract.go:96-120`). 요청은 `{planId}` 익명 구조체와 `Idempotency-Key` 헤더 (`internal/dashboard/jobs.go:73-82`). 새 작업이든 재요청이든 202로 답한다.
- **원본 데이터**: 저장된 계획. 서버는 계획 소유자와 만료를 확인하고(`internal/app/web_jobs.go:279-294`), 엔진 `Prepare`를 다시 돌려 지문과 claim이 그대로인지 보고(`web_jobs.go:238-247`), 같은 잠금 안에서 다시 확인한 뒤 자원 충돌을 검사해 `accepted` 작업을 저장한다(`web_jobs.go:251-271`). 실행은 서버 소유의 별도 컨텍스트에서 이어진다(`web_jobs.go:272-275`).
- **갱신 방식**: "검토한 계획 실행" 클릭 때 한 번(`JobPanel.svelte:140`).
- **쓰는 화면**: `web/src/JobPanel.svelte` (응답의 `id`만 읽고 상세 화면으로 이동한다).

필드 표는 아래 `GET /api/v1/jobs/{jobId}`와 같다. 이 응답에서 `state`는 `accepted`, `phases`·`runIds`·`partialEffects`·`unresolvedResources`는 빈 배열, `nodeDisposition`은 `unknown`이다(`web_jobs.go:268`).

- **Idempotency-Key 처리**: 헤더가 비었거나 128바이트를 넘으면 400(`web_jobs.go:223-225`, `ErrWebMalformedRequest`는 `deployment.go:349-350`에서 400). 키는 `sha256(actorId + ":" + key)`로, 본문은 `sha256(planId)`로 저장한다(`web_jobs.go:229`, `:212-215`). 같은 키와 같은 계획이면 저장된 작업을 그대로 돌려주고, 같은 키에 다른 계획이면 409이다(`web_jobs.go:280-285`). 키 기록은 `jobs.json`에 남아 재시작 뒤에도 유지된다(`web_jobs.go:32`, `:76-80`). 화면은 계획 ID를 키로 쓴다(`JobPanel.svelte:58-61`, `:140`). 그래서 같은 계획으로 두 번 누르면 두 번째는 이미 끝난 작업이라도 같은 작업을 돌려받는다. 다시 실행하려면 계획을 새로 만들어야 한다.
- **문제**: 만료된 계획, 엔진 없음, 지문 변경, 다른 키 재사용이 모두 같은 409 `ErrDeploymentConflict`로 온다(`web_jobs.go:245-246`, `:282`, `:290-291`). 자원 충돌만 `details.holders`가 붙어 구분된다. 화면은 이유를 메시지 문자열로만 보여 준다.

### `GET /api/v1/jobs`

- **응답 Go 타입**: 익명 맵 `{"items": []app.WebJob, "nextCursor": string|null}` (`internal/dashboard/jobs.go:83-125`).
- **원본 데이터**: 메모리의 `jobs.json` 상태를 `createdAt` 내림차순으로 정렬한 사본(`internal/app/web_jobs.go:317-326`).
- **갱신 방식**: JobPanel은 열 때와 작업을 바꾼 뒤 부른다(`JobPanel.svelte:73`, `:78`). NodeDetail은 열 때 그 workspace의 활성 작업을 끝까지 페이지를 넘겨 읽는다(`web/src/NodeDetail.svelte:31`, `web/src/pages.mjs:3-13`).
- **쓰는 화면**: `web/src/JobPanel.svelte`, `web/src/NodeDetail.svelte`.
- **쿼리**: `limit` 1-200, 기본 50, 범위를 벗어나면 400 `invalid limit`(`jobs.go:85-93`). `cursor`는 앞 페이지 마지막 작업 ID이고 없는 ID면 400(`jobs.go:94-119`). `workspaceId`, `state`, `actorId`는 정확히 같은 값만 남긴다(`jobs.go:107`).

필드 표는 `GET /api/v1/jobs/{jobId}`와 같다.

- **문제**: JobPanel은 `limit`도 `cursor`도 없이 부르므로 최신 50개만 보이고 다음 페이지를 읽지 않는다(`JobPanel.svelte:73`, `:173`). 커서가 필터를 적용하기 전 순서에서 찾아지므로, 필터를 바꾸고 이전 커서를 쓰면 다른 결과의 중간부터 읽을 수 있다(`jobs.go:99-109`). 모든 로그인 사용자(viewer 포함)가 모든 작업을 읽는다. 소유자 검사는 취소와 재확인에만 있다(`web_jobs.go:308-316`, `web_job_execution.go:188`).

### `GET /api/v1/jobs/{jobId}`

- **응답 Go 타입**: `app.WebJob` (`internal/app/web_job_contract.go:96-120`). 단계는 `app.WebJobPhase` (`web_job_contract.go:88-94`), 재확인 결과는 `app.WebReconciliation`과 `app.WebNodeCheck` (`internal/app/web_job_reconcile.go:20-33`).
- **원본 데이터**: `jobs.json`의 작업 사본(`web_jobs.go:308-316`). 단계는 엔진이 보고할 때마다 기록된다(`internal/app/web_job_execution.go:18-47`, 엔진 쪽 `web_chain_engine.go:772-792`). 끝 상태와 노드 처리, 세션 참조는 실행이 끝날 때 기록된다(`web_job_execution.go:121-164`).
- **갱신 방식**: 상세 화면은 열 때 한 번 읽고(`web/src/JobDetail.svelte:25`, `:105`), 그 뒤로는 `/api/v1/snapshot`과 `/api/v1/events` 스트림의 같은 작업 객체를 우선 쓴다(`JobDetail.svelte:19-23`). 스트림 이벤트의 `job`은 매 커밋마다 `detachedWebJob`으로 만든 같은 `WebJob` 전체이고(`internal/app/web_job_observation.go:60-70`), 스냅샷의 `jobs[]`도 같은 타입이다(`internal/dashboard/snapshot.go:25-36`). 그래서 스트림이 연결된 동안 이 엔드포인트의 응답과 모양이 같다. 취소나 재확인 버튼 뒤에도 다시 읽는다(`JobDetail.svelte:158`, `JobCard.svelte:16`).
- **쓰는 화면**: `web/src/JobDetail.svelte`, `web/src/JobCard.svelte`, `web/src/CasualtyEntry.svelte`, `web/src/PhaseWatches.svelte`(단계 행), `web/src/job-view.mjs`. 같은 객체를 스트림으로 받는 화면은 `web/src/Dashboard.svelte`, `web/src/MonitorPanel.svelte`, `web/src/dashboard-view.mjs`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `id` | string | 아니오 | 32자 hex, 라우트도 이 형식만 받는다 (`web/src/routes.mjs:4`) | `deploymentID()` (`web_jobs.go:268`) | 예 (`JobDetail.svelte:113`) |
| `actorId` | string | 아니오 | | 시작한 행위자 | 예 (`JobCard.svelte:13`, `MonitorPanel.svelte:26`) |
| `workspaceId` | string | 아니오 | | 계획 입력 | 예 (`JobDetail.svelte:93`) |
| `planId` | string | 아니오 | 32자 hex | 계획 | 아니오 |
| `operation` | string | 아니오 | 작업 ID | 계획 입력 | 예 (`CasualtyEntry.svelte:35`) |
| `state` | string | 아니오 | 아래 상태 목록 | 실행 흐름 | 예 (`job-view.mjs:11`, `CasualtyEntry.svelte:33`) |
| `phases[]` | array of `WebJobPhase` | 빈 배열 가능, null 아님 (`web_jobs.go:268`) | | 엔진 보고 | 예 (`JobDetail.svelte:30`, `JobCard.svelte:22`) |
| `phases[].name` | string | 아니오 | 계획의 단계 이름 중 하나만 허용 (`web_job_execution.go:26-34`) | 엔진 | 예 (`PhaseWatches.svelte:16`) |
| `phases[].state` | string | 아니오 | `running`, `succeeded`, `failed`, `cancelled`(끝날 때 덮어씀) | 엔진, 끝 처리 (`web_job_execution.go:146-151`) | 예 (`job-view.mjs:9`) |
| `phases[].startedAt` | string | 비지 않는다. 단계는 시작할 때 목록에 들어간다 | RFC 3339 나노초, UTC | 엔진 (`web_chain_engine.go:778`) | 예 (`job-view.mjs:67`) |
| `phases[].finishedAt` | string | 실행 중이면 `"0001-01-01T00:00:00Z"` | RFC 3339 나노초, UTC | 엔진 (`web_chain_engine.go:781`) 또는 끝 처리 (`web_job_execution.go:149`) | 예 (`job-view.mjs:59`, `:67`) |
| `phases[].message` | string | omitempty | 엔진 오류 문장, 등록된 비밀 제거 (`web_job_execution.go:19`) | 엔진 | 예 (`JobCard.svelte:22`, `job-view.mjs:44`) |
| `createdAt` | string | 아니오 | RFC 3339 나노초, UTC | 수락 시각 | 예 (`CasualtyEntry.svelte:38`, `job-view.mjs:58`) |
| `retention` | string | 아니오 | `retain` 또는 `cleanup`. 취소 이유가 `user`가 아니면 `retain`으로 바뀐다 (`web_job_execution.go:243-245`) | 계획 입력 | 아니오 |
| `nodeDisposition` | string | 아니오 | 아래 상태 목록 | 실행·정리·재확인 | 예 (`JobCard.svelte:21`, `job-view.mjs:16`) |
| `cancelReason` | string | omitempty | `user`, `credential_revoked`, `account_revoked` | 취소 경로 (`web_job_execution.go:194`, `:257-262`) | 예 (`JobDetail.svelte:155`, `job-view.mjs:47`) |
| `cancelMessage` | string | omitempty | 최대 500바이트, 비밀 제거 (`web_job_execution.go:175-178`) | 취소 요청 본문 | 예 (`JobDetail.svelte:155`) |
| `credentialRefs[]` | array of string | 빈 배열 가능, null 아님 | 개인 자격증명 ID, 정렬 (`web_job_lifecycle.go:47-60`) | 계획의 SSH·계정 키 연결 | 아니오 |
| `requestId` | string | omitempty | 요청 ID | 수락한 HTTP 요청 (`web_job_lifecycle.go:30-37`) | 아니오 |
| `reconciliation` | object | omitempty, 재확인 전에는 없음 | | `Reconcile` (`web_job_reconcile.go:44-88`) | 예 (`JobCard.svelte:27`) |
| `reconciliation.checkedAt` | string | 아니오 | RFC 3339 나노초 | 서버 시계 | 예 (`JobCard.svelte:27`) |
| `reconciliation.outcome` | string | 아니오 | `absent`, `present`, `unverified` | `web_job_reconcile.go:148-155`, 오류면 `unverified` (`:66-68`) | 예 (`JobCard.svelte:14`, `:27`) |
| `reconciliation.nodes[]` | array of `{nodeId, process, data}` | 빈 배열 가능 | `process`: `running`·`stopped`·`unknown`, `data`: `present`·`absent`·`unknown` | `chain-record.json`의 노드마다 SSH·로컬 관측 (`web_job_reconcile.go:120-147`, `internal/core/process/ownership.go:27`, `:37`) | 예 (`JobCard.svelte:27`) |
| `reconciliation.message` | string | omitempty | 영어 문장 | `web_job_reconcile.go:104`, `:150` | 예 (`JobCard.svelte:27`) |
| `runIds[]` | array of string | 빈 배열 가능, null 아님 | `"web:<process>/<run>"` 형식의 엔진 세션 참조 (`internal/app/web_test_run.go:362-368`) | 테스트 실행·연결 실행 결과 (`web_test_run.go:352`, `web_test_attach.go:231`) | 예 (`JobCard.svelte:23`, `JobDetail.svelte:98`) |
| `partialEffects[]` | array of string | 빈 배열 가능 | 영어 문장, 비밀 제거 | 엔진 결과와 정리 결과 (`web_job_execution.go:96`, `:111-113`) | 예 (`job-view.mjs:14`, `JobCard.svelte:25`) |
| `unresolvedResources[]` | array of string | 빈 배열 가능 | 경로나 `host:path` 문자열 | 엔진 결과, 재시작 때 계획의 `resources`를 더함 (`web_jobs.go:95`) | 예 (`job-view.mjs:15`, `MonitorPanel.svelte:57`) |
| `error` | object(string 맵) | omitempty | `{code, message, requestId?}` | 끝 처리 (`web_jobs.go:94`, `web_job_execution.go:155-157`, `:220-225`) | 예 (`job-view.mjs:13`, `JobCard.svelte:24`) |
| `error.code` | string | 아니오 | `server_restarted`, `engine_error`, `plan_invalid`, `access_revoked` | 같은 곳 | 아니오 |
| `error.message` | string | 아니오 | 영어 문장 | 같은 곳 | 예 |

#### 상태 문자열이 정의된 곳과 화면이 다시 적은 곳

작업 `state`는 `accepted`, `running`, `cancelling`, `succeeded`, `failed`, `cancelled`, `interrupted` 일곱 가지다. 끝 상태 네 개는 `terminalWebJob`에 있고(`web_job_contract.go:137-139`), 나머지는 수락(`web_jobs.go:268`), 실행 시작(`web_job_execution.go:65-71`), 취소 요청(`web_job_execution.go:238`), 재시작 처리(`web_jobs.go:89-97`)에서 쓴다. 서버가 재시작하면 끝나지 않은 작업은 모두 `interrupted`가 되는데 그 작업의 `phases[].state`는 고치지 않으므로 `running` 단계가 남는다(`web_jobs.go:92-96`). 화면은 끝 상태를 `job-view.mjs:3`, `MonitorPanel.svelte:25`, `HistoryPanel.svelte:17`에, 활성 상태를 `JobPanel.svelte:57`, `NodeDetail.svelte:31`에, 주의할 상태를 `dashboard-view.mjs:4`에, 일곱 가지와 `unknown`을 히스토리 필터 선택지로 `HistoryPanel.svelte:107`에 다시 적었다.

단계 `state`는 엔진이 `running`, `succeeded`, `failed`만 보고할 수 있고(`web_job_execution.go:32`), 작업이 끝날 때 아직 `running`인 단계를 작업의 끝 상태(`succeeded`, `failed`, `cancelled`)로 덮어쓴다(`web_job_execution.go:146-151`).

`nodeDisposition`은 `unknown`(수락 시, 재시작, 대상 단계 실패, 재확인 미확인), `retained`(기본 결과, 철회, 재확인 `present`), `cleaned`(정리 성공, 재확인 `absent`), `cleanup_failed`(정리 실패) 네 가지다. 정의는 흩어져 있다(`web_jobs.go:93`, `:268`, `web_job_execution.go:106`, `:137-142`, `web_job_reconcile.go:75-82`, `web_partial_effects.go:18-23`, `web_chain_engine.go:815-825`, `web_chain_execution.go:21`, `web_test_run.go:255`, `web_test_attach.go:170`, `web_network_monitor.go:185`). 화면은 `job-view.mjs:16`에서 `unknown`만 따로 본다.

재확인 `outcome`은 `absent`, `present`, `unverified`이고 정의는 `web_job_reconcile.go:16-19`, `:148-155`에 있다. 화면은 세 값의 한국어 문장을 `JobCard.svelte:14`에 따로 적었다. 노드 `process`와 `data` 값은 `web_job_reconcile.go:27-33`에 주석으로만 있다.

상태를 그림과 한국어로 바꾸는 표는 `labels.mjs:35-42`와 `status-icon.mjs:3-10` 두 곳이다. `labels.mjs`에는 `cleaned`, `cleanup_failed`, `present`, `unverified`의 한국어가 없어서 `StatusBadge`가 영어 코드만 보여 준다(`StatusBadge.svelte:11`).

- **문제**: `WebJobPhase.StartedAt`과 `FinishedAt`은 `time.Time`에 `omitempty`를 달았지만 Go는 구조체 값을 비었다고 보지 않으므로 `"0001-01-01T00:00:00Z"`가 나간다(`web_job_contract.go:91-92`). 화면은 `job-view.mjs:22-26`에서 이 값을 "기록 없음"으로 다시 해석하지만 `run-timeline.mjs:7`의 `at()`과 `JobDetail.svelte:33`의 `Date.parse`는 이 값을 유효한 시각으로 받는다. `phaseSegments`는 음수 길이를 걸러서 피하지만(`run-timeline.mjs:72-73`) 같은 값을 두 방식으로 읽는 셈이다. OpenAPI `Job.runIds`는 항목 패턴을 `^[A-Za-z0-9_.-]+$`로 두었는데 실제 값에는 `:`와 `/`가 있다(`docs/dev/web-ui.openapi.yaml:254-257`, `:3370-3372`). OpenAPI `Error.code` 열거값에는 작업 오류 코드 네 개(`server_restarted` 등)가 없다(`web-ui.openapi.yaml:2555-2572`). OpenAPI는 `reconciliation`을 `additionalProperties: true`로만 적었다(`web-ui.openapi.yaml:3366-3369`).

### `POST /api/v1/jobs/{jobId}/cancel`

- **응답 Go 타입**: `app.WebJob`. 요청은 `{message}` (`internal/dashboard/jobs.go:136-145`). 202로 답한다.
- **원본 데이터**: 작업 상태를 `cancelling`으로 저장하고 실행 컨텍스트를 끊는다(`web_job_execution.go:235-254`). 이미 끝난 작업이면 바꾸지 않고 그대로 돌려준다(`web_job_execution.go:191-193`). 시작한 사람과 관리자만 취소할 수 있다(`web_job_execution.go:188-190`).
- **갱신 방식**: 버튼 클릭(`JobCard.svelte:29`, `MonitorPanel.svelte:30`).
- **쓰는 화면**: `web/src/JobCard.svelte`, `web/src/MonitorPanel.svelte`.

필드 표는 `GET /api/v1/jobs/{jobId}`와 같다.

- **문제**: 이 엔드포인트는 멱등 키를 받지 않는다. 같은 요청을 다시 보내도 상태가 이미 `cancelling`이면 같은 기록을 한 번 더 남기고 같은 결과를 돌려준다(`web_job_execution.go:194`). 서버는 취소 이유 문장(`message`)을 받지만 두 화면 모두 빈 객체 `{}`를 보내므로 화면에서 시작한 취소에는 `cancelMessage`가 생기지 않는다(`JobCard.svelte:29`, `MonitorPanel.svelte:30`).

### `POST /api/v1/jobs/{jobId}/reconcile`

- **응답 Go 타입**: `app.WebJob` (`internal/dashboard/jobs.go:130-135`). 200으로 답한다.
- **원본 데이터**: 계획 payload의 `controlDir/chain-record.json`을 읽고 기록된 노드마다 프로세스와 데이터 디렉터리를 관측한다(`web_job_reconcile.go:93-157`). 결과에 따라 `nodeDisposition`과 `unresolvedResources`를 바꾼다(`web_job_reconcile.go:72-83`). 로컬 기계의 중단된 작업은 서버 시작 때 한 번 자동으로 확인한다(`web_job_reconcile.go:177-198`).
- **갱신 방식**: "실제 상태 확인" 클릭(`JobCard.svelte:28`).
- **쓰는 화면**: `web/src/JobCard.svelte`.

필드 표는 `GET /api/v1/jobs/{jobId}`와 같다. 바뀌는 것은 `reconciliation`, `nodeDisposition`, `unresolvedResources`다.

- **문제**: 화면이 버튼을 보이는 조건(`job-view.mjs:16`)과 서버가 받아들이는 조건(`web_job_reconcile.go:58`)이 같다. 다만 둘을 따로 적었다. 끝나지 않은 작업에 부르면 일반 오류로 422가 난다.

### `GET /api/v1/contracts/jobs`

- **응답 Go 타입**: `app.WebJobsContract` (`internal/app/web_jobs_contract.go:16-18`), 항목은 `app.WebOperation` (`web_jobs_contract.go:7-12`).
- **원본 데이터**: 코드에 고정된 작업 목록 10개의 사본(`web_jobs_contract.go:20-34`). `network.monitor`와 `test.attach`는 상수에서 온다(`web_network_monitor.go:18`, `web_test_attach.go:22`).
- **갱신 방식**: JobPanel과 NodeDetail이 열 때 한 번(`JobPanel.svelte:78`, `NodeDetail.svelte:30`).
- **쓰는 화면**: `web/src/JobPanel.svelte`, `web/src/NodeDetail.svelte`(→ `node-view.mjs`의 `nodeControls`).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `operations[]` | array | 아니오 | | `webOperations` | 예 (`JobPanel.svelte:23`) |
| `operations[].id` | string | 아니오 | `chain.setup` 등 | 같은 곳 | 예 (`JobPanel.svelte:24`, `:153`) |
| `operations[].label` | string | 아니오 | 한국어 이름 | 같은 곳 | 예 (`JobPanel.svelte:153`) |
| `operations[].description` | string | omitempty | 한국어 설명 | 같은 곳 | 예 (`JobPanel.svelte:153`, title 속성) |
| `operations[].scope` | string | 아니오 | `chain` 또는 `test` | 같은 곳 | 예 (`JobPanel.svelte:23`) |

- **문제**: 작업 단계 이름(`new`, `place`, `keys`, ...)과 그 뜻은 이 계약에 없다. 단계 목록은 계획 응답에만 오고(`web_chain_engine.go:522-531`) 한국어 이름이 없다. OpenAPI는 작업 ID를 열거형으로 다시 적었다(`web-ui.openapi.yaml:3191-3203`). 목록이 바뀌면 두 곳을 같이 고쳐야 한다.

### `GET /api/v1/history`

- **응답 Go 타입**: `app.WebRunPage` (`internal/app/web_history_contract.go:34-37`), 항목은 `app.WebRun` (`web_history_contract.go:11-26`). 쿼리는 `app.WebHistoryQuery` (`web_history_contract.go:28-32`, 파싱 `internal/dashboard/history.go:87-107`).
- **원본 데이터**: 요청마다 먼저 `Sync`를 돈다(`history.go:28-31`). `Sync`는 `jobs.json`의 모든 작업을 `job-<jobId>` 기록으로 다시 만들고(`internal/app/web_history_capture.go:11-37`), 두 세션 루트의 엔진 세션을 `session-<hash>` 기록으로 캡처해(`internal/app/web_history.go:90-134`) `history.json`에 저장한다. 목록은 저장된 캡처에서 필터하고 `startedAt` 문자열 내림차순으로 정렬한다(`web_history.go:136-179`). 각 행에는 작업 기간으로 계산한 `summary.observations`가 붙는다(`internal/app/web_history_observations.go:64-93`).
- **갱신 방식**: 히스토리 화면은 로그인 직후 숨겨진 상태로도 한 번 읽고(`App.svelte:116`, `HistoryPanel.svelte:42-48`), 새로고침, 필터 적용, "다음 기록 더 보기" 때 다시 읽는다(`HistoryPanel.svelte:101-123`). 응답 본문은 텍스트로 받아 `parseHistoryJSON`으로 파싱한다(`HistoryPanel.svelte:36`).
- **쓰는 화면**: `web/src/HistoryPanel.svelte`.
- **쿼리**: `search`(기록 JSON 전체에서 대소문자 무시 부분 일치, `web_history.go:189-192`), `chain`, `workspaceId`, `actorId`, `state`(정확히 일치), `caseId`(`summary.tests[].id` 일치, `:193-200`), `from`·`to`(RFC 3339 나노초, `startedAt` 기준), `cursor`(직전 페이지 마지막 `id`), `limit` 1-200 기본 50. 쿼리 해석 실패는 400 `invalid history query`, 범위·커서 오류는 400 `invalid history cursor or filters`(`history.go:36-46`). 별도 검색 엔드포인트는 없고 `search` 쿼리가 그 역할이다.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[]` | array | 빈 배열 가능 | | 캡처 | 예 (`HistoryPanel.svelte:37`) |
| `items[].id` | string | 아니오 | `job-<32hex>` 또는 `session-<32hex>` (`web_history.go:83-86`) | | 예 (`HistoryPanel.svelte:118`) |
| `items[].jobId` | string | omitempty, CLI 세션은 없음 | 32자 hex | 작업 또는 `runIds`로 연결된 작업 (`web_history_capture.go:32-34`) | 아니오 |
| `items[].workspaceId` | string | 빈 문자열 가능(연결 없는 세션) | | 작업 | 예 (`HistoryPanel.svelte:119`) |
| `items[].chain` | string | 빈 문자열 가능 | 체인 이름 | 계획 payload `binary.chain` (`web_history_capture.go:25`) | 예 (`HistoryPanel.svelte:119`) |
| `items[].actorId` | string | omitempty | | 작업 | 예 (`HistoryPanel.svelte:25`) |
| `items[].state` | string | 아니오 | 작업 상태 일곱 가지 또는 세션 판정 `succeeded`·`failed`·`unknown` | 아래 WebRun 계산 | 예 (`HistoryPanel.svelte:119`) |
| `items[].startedAt` | string | omitempty | 작업 기록은 RFC 3339 나노초, 세션은 `session.json`의 RFC 3339 초 단위 | `web_history_capture.go:25`, `:70` | 예 (`HistoryPanel.svelte:119`) |
| `items[].finishedAt` | string | 목록에서는 항상 없음 | | 상세에서만 채운다 | 아니오 |
| `items[].sessionRefs[]` | array of string | 빈 배열 가능 | `web:` 접두 또는 CLI 세션 ID | | 아니오(목록) |
| `items[].fingerprints` | object string→string | 빈 객체 가능 | `target`, `binary`, `environments/<env>/env.json` 키 | `web_history_capture.go:25-28`, `:73-84` | 아니오(목록) |
| `items[].summary` | object | 아니오 | 기록 종류별로 다름. 아래 상세 표 | | 예 (`HistoryPanel.svelte:26`, `:118`, `:120`) |
| `items[].artifactRefs[]` | array of string | 빈 배열 가능 | 캡처한 파일 이름 | | 아니오 |
| `nextCursor` | string 또는 null | null 가능 | 마지막 항목 `id` | `web_history.go:174-177` | 예 (`HistoryPanel.svelte:37`) |

- **문제**: 목록 요청마다 모든 세션 디렉터리를 다시 훑고, Web 세션은 연결된 작업이 끝날 때까지 매번 다시 캡처한다(`web_history.go:116-126`). 캡처 하나가 8 MiB까지이고 전체 스냅샷이 64 MiB 한도이므로(`internal/core/session/history_capture.go:72`, `historystore.go:27-36`) 큰 세션이 몇 개 쌓이면 `Sync`가 실패해 모든 히스토리 요청이 500 `history capture unavailable`이 된다. 이 동작은 코드로 추정했고 실행해 확인하지 않았다. 테스트 작업 하나가 `job-<id>`와 `session-<hash>` 두 행으로 따로 나오고, 둘을 잇는 값은 세션 행의 `jobId`와 작업 행의 `sessionRefs`뿐이다. 정렬이 두 시각 형식을 문자열로 비교하므로 같은 초 안에서는 순서가 시간과 어긋날 수 있다(`web_history.go:153-158`). OpenAPI `Run.id`, `jobId`의 패턴은 실제 값과 맞지만 `chain`과 `workspaceId`를 필수로 적어 빈 문자열을 허용하는지 밝히지 않는다(`web-ui.openapi.yaml:3543-3570`).

### `GET /api/v1/history/{runId}`

- **응답 Go 타입**: `app.WebRun` (`internal/app/web_history_contract.go:11-26`). `coverage`는 `*app.WebCoverage`(모니터링 영역 소관).
- **원본 데이터**: 캡처 사본(`web_history.go:204-212`)에 `Detail`이 작업 기간의 끝 시각과 지표 보관 범위를 더한다(`web_history.go:216-233`). 기간은 연결된 작업의 `createdAt`부터 마지막 단계 `finishedAt`까지다(`web_history_observations.go:40-62`).
- **갱신 방식**: 히스토리 화면에서 행을 누를 때(`HistoryPanel.svelte:51-58`). 작업 상세 화면은 열 때 `job.runIds`의 각 항목으로 이 엔드포인트를 부른다(`JobDetail.svelte:97-104`). 둘 다 본문을 텍스트로 받아 `parseHistoryJSON`으로 파싱한다.
- **쓰는 화면**: `web/src/HistoryPanel.svelte`, `web/src/JobDetail.svelte`, `web/src/RunTimeline.svelte`, `web/src/TestLedger.svelte`, `web/src/CasualtyEntry.svelte`, `web/src/CauseBench.svelte`(테스트 행의 시각만), `web/src/run-timeline.mjs`, `web/src/job-view.mjs`.

목록 표의 필드에 더해 다음이 있다. `summary`는 `map[string]any`이라 Go 타입이 없고 기록 종류에 따라 키가 다르다.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `finishedAt` | string | omitempty. 연결된 작업이 없거나 끝 시각이 없으면 없음 | RFC 3339 나노초 | 마지막 단계 `finishedAt` (`web_history.go:225`) | 예 (`HistoryPanel.svelte:137`, `RunTimeline.svelte:21`) |
| `coverage` | object | omitempty | 모니터링 영역 | `WebMonitor.Metrics` (`web_history.go:226-231`) | 예 (`HistoryPanel.svelte:137`) |
| `summary.kind` | string | 아니오 | `job` 또는 `engine-session` | `web_history_capture.go:25`, `:70` | 아니오 |
| `summary.operation` | string | 작업 기록만 | | 작업 | 예 (`HistoryPanel.svelte:118`) |
| `summary.phases[]` | array of `WebJobPhase` | 작업 기록만 | 0년 시각 포함 | 작업 | 아니오(원본 JSON 상자에만) |
| `summary.partialEffects[]`, `summary.unresolvedResources[]`, `summary.nodeDisposition`, `summary.error` | | 작업 기록만, `error`는 null 가능 | | 작업 | 아니오 |
| `summary.command` | string | 세션 기록만 | 엔진 명령 이름 | `session.json`의 `command` | 예 (`HistoryPanel.svelte:118`) |
| `summary.tests[]` | array of `session.TestResult` | 세션 기록만, null 가능 | | `session.json`의 `tests` (`internal/core/session/read.go:23-35`) | 예 (`run-timeline.mjs:16`) |
| `summary.tests[].seq` | number | 아니오 | 1부터 | | 예 (`run-timeline.mjs:17`) |
| `summary.tests[].id` | string | 아니오 | 케이스 ID | | 예 (`run-timeline.mjs:17`) |
| `summary.tests[].status` | string | 아니오 | `pass`, `fail`, `blocked`, `skip` (`internal/core/session/session.go:17-24`) | | 예 (`run-timeline.mjs:27`) |
| `summary.tests[].startedAt`, `finishedAt` | string | omitempty. 옛 기록과 판정 없는 테스트는 없음 (`read.go:29-34`) | RFC 3339 나노초 UTC (`record_impl.go:111-116`) | | 예 (`run-timeline.mjs:23`, `:28-29`) |
| `summary.tests[].presetId`, `env` | string | omitempty | | | 아니오 |
| `summary.counts` | object `{pass, fail, blocked, skip}` | 세션 기록만 | 정수 | `session.json`의 `summary` (`read.go:38-43`) | 예 (`RunTimeline.svelte:24`, `JobDetail.svelte:40`) |
| `summary.captureGaps[]` | array of string | 세션 기록만 | 영어 문장 | `CapturedSession.Gaps` (`history_capture.go:56`, `:73`, `:91`) | 예 (`HistoryPanel.svelte:142`) |
| `summary.results` | object 파일이름→JSON | 세션 기록만 | 키는 `tests/<NNN>_<id>/<file>`. `spec.json`과 JSON이 아닌 파일은 뺀다 (`web_history_capture.go:85-87`) | 캡처한 파일 | 예 (`run-timeline.mjs:15-20`) |
| `summary.missingDimensions[]` | array of string | 아니오 | 아래 WebRun 계산 | | 예 (`HistoryPanel.svelte:138`) |
| `summary.observations` | object `{networkId, from, to?}` | 연결된 작업이 있고 모니터가 켜졌을 때만 | `from`·`to`는 RFC 3339 나노초 | 작업 기간 (`web_history_observations.go:77-81`) | 예 (`RunTimeline.svelte:18-21`, `JobDetail.svelte:67-70`, `HistoryPanel.svelte:62-72`) |

#### WebRun의 `state`와 `missingDimensions` 계산

작업 기록은 작업의 `state`를 그대로 쓰고 `missingDimensions`를 `["test-results","metrics","logs"]`로 둔다(`web_history_capture.go:25`). 세션 기록은 `session.json`의 테스트 판정으로 계산한다(`web_history_capture.go:46-61`). 테스트가 없으면 `unknown`, `fail`이나 `blocked`가 하나라도 있으면 `failed`, 그 밖의 값이 있으면 `unknown`, 모두 `pass`·`skip`이면 `succeeded`다. `web:` 세션인데 연결된 작업이 없으면 `unknown`으로 바꾸고 `jobId`를 빠진 항목에 더하며, 연결된 작업이 아직 끝나지 않았으면 그 작업 상태를 쓴다(`web_history_capture.go:62-69`). 세션 기록의 `missingDimensions`는 처음부터 `["metrics","logs","finishedAt"]`이고 연결된 작업이 없으면 `actorId`, `workspaceId`, `chain`, `binary`를 더한다(`web_history_capture.go:47`, `:71-72`). 응답 직전에 작업 기간이 계산되면 `metrics`와 `logs`를 뺀다(`web_history_observations.go:82-90`).

엔진은 세션 전체의 끝 시각을 기록하지 않는다. `session.Result`에는 `startedAt`만 있고(`read.go:14-20`), 저장할 때도 `startedAt`만 RFC 3339 초 단위로 쓴다(`internal/core/session/session_impl.go:234-242`). 그래서 세션 기록에는 언제나 `finishedAt`이 빠졌다고 적힌다. 반면 `Detail`은 연결된 작업이 끝났으면 작업 단계로 `finishedAt`을 채운다(`web_history.go:221-225`). 그 결과 히스토리 상세는 "종료 시각"을 보여 주면서 바로 아래에 "자료 없음: finishedAt"을 같이 보여 줄 수 있다(`HistoryPanel.svelte:137-138`). `missingDimensions`는 캡처 시점에 고정되고 상세 응답이 이 값을 다시 계산하지 않기 때문이다. 연결이 없는 CLI 세션은 `RunTimeline`이 마지막 테스트의 `finishedAt`을 축의 끝으로 쓴다(`RunTimeline.svelte:20-21`).

#### 엔진 세션 파일과 캡처 허용 목록

세션 디렉터리 구조는 `session.json`, `tests/<NNN>_<id>/`, `environments/<env>/`, `keys/`, `logs/`, `nodes/`이다(`session_impl.go:18-28`). 캡처는 허용 목록 방식이다. 들어가는 디렉터리는 루트, `tests`, `tests/<x>`, `environments`, `environments/<x>`, `environments/<x>/chainstate`뿐이고 나머지는 통째로 건너뛴다(`history_capture.go:126-131`, `:59-64`). 들어가는 파일은 다음과 같다(`history_capture.go:133-146`, 이름 상수 `record_impl.go:12-19`).

| 파일 | 형식과 필드 | 캡처 | `summary.results`에 들어가나 | 화면이 읽는 곳 |
|---|---|---|---|---|
| `session.json` | camelCase `{id, command, startedAt, tests[], summary}` (`read.go:14-43`). 끝 시각 없음 | 예. 없거나 `id`가 비면 캡처 실패 (`history_capture.go:106-108`) | 아니오. 대신 `summary.tests`, `summary.counts`, `summary.command`로 펼친다 | `run-timeline.mjs:16`, `RunTimeline.svelte:24` |
| 케이스 디렉터리의 `status.json` | camelCase `{id, seq, result, env?, reason?, presetId?, startedAt?, finishedAt?}` (`record_impl.go:98-108`) | 예 | 예 | `reason`만 (`run-timeline.mjs:20`, `:31`) |
| 케이스 디렉터리의 `steps.json` | PascalCase 배열 `[{Index, Type, On, Signer, Nonce, Gas, Hash, Receipt, Deployment?, Error}]` (`internal/core/session/record.go:5-20`). `Deployment` 안쪽만 camelCase (`record.go:24-32`) | 예 | 예 | `Error`가 있는 항목의 `Index`, `Type`, `On`, `Error` (`run-timeline.mjs:22`) |
| 케이스 디렉터리의 `assert.json` | PascalCase 배열 `[{ID, Source, On, Assert, Expected, Actual, Pass, Provenance}]` (`record.go:37-46`) | 예 | 예 | `Pass === false`인 항목의 `ID`, `On`, `Source`, `Assert`, `Expected`, `Actual` (`run-timeline.mjs:21`) |
| 케이스 디렉터리의 `postaction.json` | PascalCase 배열 `[{Name, OK, Detail}]` (`record.go:50-54`) | 예 | 예 | 아니오 |
| 케이스 디렉터리의 `artifacts.json` | camelCase `{refs:[{kind, ref, node?}]}` (`record.go:62-79`) | 예 | 예 | 아니오 |
| 케이스 디렉터리의 `spec.json` | 원래 케이스 선언을 비밀 제거해 저장 (`record_impl.go:63-67`) | 예 | 아니오 (`web_history_capture.go:85`) | 아니오. 비교의 선언 지문에만 쓴다 (`web_history_compare.go:100-114`) |
| 케이스 디렉터리의 `env-ref` | JSON이 아닌 평문 환경 ID (`record_impl.go:53-58`) | 예 | 아니오(JSON이 아니라서) | 아니오 |
| `environments/*/env.json` | camelCase `{envId, fingerprint, dataPath, nodes[]}` (`internal/core/session/environment_impl.go:153-159`), `nodes[]`는 `node.Node`(`internal/core/node/node.go:78`) | 예 | 아니오. `fingerprint`만 `fingerprints["environments/<env>/env.json"]`로 올린다 (`web_history_capture.go:80-84`) | 아니오(원본 JSON 상자의 지문만, `HistoryPanel.svelte:145`) |
| `environments/*/chainstate/*.jsonl` | 줄마다 `{seq, heights?, peers?, bpParticipation?, forked}` (`internal/testengine/chainstate_sink.go:16-22`). `heights`는 노드→uint64 | 예 | 아니오 | 아니오. 내보내기 파일에만 있다 |

파일 하나는 2 MiB, 세션 전체는 8 MiB까지이고 넘으면 `captureGaps`에 적고 뺀다(`history_capture.go:72-75`). 링크와 잘못된 JSON도 빼고 적는다(`history_capture.go:55-58`, `:90-93`). 캡처한 JSON은 `UseNumber`로 읽어 비밀 키 이름을 `[REDACTED]`로 바꾼 뒤 다시 직렬화하므로 들여쓰기는 사라지지만 숫자 자릿수는 그대로 남는다(`web_history_capture.go:94-131`, `web_history_contract.go:90-94`).

`run-timeline.mjs`는 테스트 디렉터리 이름을 `tests/${seq를 세 자리로 채움}_${id}/`로 다시 만들어 `summary.results`의 키를 찾는다(`run-timeline.mjs:10`, `:17-20`). 엔진 쪽 규칙은 `fmt.Sprintf("%03d_%s", seq, id)`이다(`read.go:141-143`). 같은 규칙을 화면이 따로 적었다. `testRows`는 PascalCase 필드를 camelCase 행으로 바꾸고(`run-timeline.mjs:21-22`), 시각이 둘 다 있을 때만 길이를 계산하며(`:23`, `:30`), 실패와 막힘을 앞으로 정렬한다(`:37-38`).

#### 큰 정수

assert의 `Actual`은 대부분 십진 문자열이다. 블록 번호, 체인 ID, 피어 수, 잔액은 읽을 때 문자열로 바꾼다(`internal/testhelper/read.go:309-334`, `internal/dsl/interp/binding.go:180-185`). `Expected`는 케이스 선언에서 온다. 선언의 `assertions`는 `[]map[string]any`로 표준 디코더가 읽으므로(`internal/dsl/spec.go:76`, `internal/dsl/parse_v2.go:23-27`) 숫자 리터럴로 쓴 큰 값은 엔진에서 이미 float64가 되어 정밀도를 잃는다. 문자열로 쓴 값은 그대로 남는다. 영수증 `Receipt`는 RPC 응답을 그대로 `map[string]any`로 읽은 것이라(`internal/testhelper/outcome.go:179-184`) 수량은 `0x` hex 문자열이다. `StepResult.Nonce`와 `Deployment.nonce`는 uint64 숫자로 나간다(`record.go:10`, `:26`).

서버는 이 값들을 `UseNumber`로 보존해 JSON 숫자 리터럴로 내보낸다. 화면은 응답을 `response.text()`로 받아 `parseHistoryJSON`으로 파싱한다(`HistoryPanel.svelte:36`, `:54`, `JobDetail.svelte:101`). 이 함수는 안전하지 않은 정수와 무한값을 만나면 reviver의 `context.source`(원래 리터럴 문자열)로 바꾸고, 그 기능이 없는 브라우저에서는 "정확한 수치 표시 불가" 문장으로 바꾼다(`web/src/history-json.js:3-9`). 소수점이 있는 값은 그대로 숫자로 남는다. `assertDiff`는 기대값과 실제값이 둘 다 정수 문자열일 때만 `BigInt`로 차이를 계산한다(`run-timeline.mjs:79-88`). 그래서 기대값이 `1e+21`처럼 지수로 직렬화되면 차이를 계산하지 않는다. 비교 응답은 `.json()`으로 읽지만(`HistoryPanel.svelte:78`) 큰 정수가 없는 응답이다.

- **문제**: (#466에서 고쳤다. 이제 `Get`이 `web:` 참조를 캡처 ID로 바꿔 찾고, OpenAPI 설명도 고쳤다. 아래는 고치기 전의 기록이다.) 작업 상세 화면은 `job.runIds`의 값(`"web:<process>/<run>"`)을 그대로 `GET /api/v1/history/{runId}`에 넣는다(`JobDetail.svelte:98-102`). 그런데 히스토리 캡처의 키는 `historySessionID("web:...")`가 만든 `session-<hash>`이고(`web_history.go:83-86`, `:115`), `Get`은 이 키로만 찾는다(`web_history.go:204-210`). 코드대로라면 이 요청은 404가 되고, 실패는 `catch(() => null)`로 삼켜져 `runs`가 빈 배열이 된다(`JobDetail.svelte:102-103`). 그러면 작업 상세의 원인 영역, 테스트 기록, 실행 타임라인이 나오지 않는다. 실행해서 확인하지는 않았고, 이 경로를 검사하는 테스트를 `tests/webui`에서 찾지 못했다. 히스토리 화면으로 가는 링크는 작업 ID로 검색하므로(`JobCard.svelte:23`) 이 문제와 상관없이 동작한다. OpenAPI의 `runId` 패턴도 `:`와 `/`를 허용하지 않는다(`web-ui.openapi.yaml:254-257`).

### `POST /api/v1/history/compare`

- **응답 Go 타입**: `app.WebComparison` (`internal/app/web_history_contract.go:39-65`). 요청은 `{runIds}` (`internal/dashboard/history.go:52-61`).
- **원본 데이터**: 캡처한 기록과 파일. 판정은 `summary.tests`, 단언은 `assert.json`의 `ID`·`Pass`, 선언은 `spec.json`의 해시, 길이와 지표는 작업 기간과 지표 보관소에서 계산한다(`internal/app/web_history_compare.go:23-126`, `:158-178`, `:183-250`).
- **갱신 방식**: 2-4개를 고르고 "선택한 실행 비교"를 누를 때(`HistoryPanel.svelte:76-79`). viewer도 이 POST만은 쓸 수 있다(`internal/dashboard/security.go:111-112`).
- **쓰는 화면**: `web/src/HistoryPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `runIds[]` | array of string | 아니오 | 요청 순서 | 요청 | 예 (`HistoryPanel.svelte:126`) |
| `comparable` | boolean | 아니오 | | 계산 | 예 (`HistoryPanel.svelte:124`) |
| `limitations[]` | array of string | 빈 배열 가능 | 영어 문장 | 계산 | 예 (`HistoryPanel.svelte:125`) |
| `results[].caseId` | string | 아니오 | | `summary.tests[].id` | 예 (`HistoryPanel.svelte:128`) |
| `results[].statuses` | object runId→string | 아니오 | `pass` 등 | 판정 | 예 (`HistoryPanel.svelte:128`) |
| `results[].complete` | boolean | 아니오 | | 계산 | 아니오 |
| `results[].assertions` | object 단언ID→(runId→boolean) | 빈 객체 가능 | | `assert.json` | 예 (`HistoryPanel.svelte:129`) |
| `durations` | object runId→number 또는 null | null 값 가능 | 초 | 작업 기간 (`web_history_compare.go:187-194`) | 예 (`HistoryPanel.svelte:127`) |
| `metrics[]` | array `{name, unit?, source, values}` | 빈 배열 가능 | | 지표 보관소 | 예 (`HistoryPanel.svelte:131`) |
| `metrics[].values` | object runId→`{samples, min?, max?, last?}` | 표본 없는 실행은 키가 없음 | float64 | 같은 곳 | 예 (`HistoryPanel.svelte:131`, `:74`) |

- **문제**: 세션 기록은 연결된 작업이 끝났어도 세션 자체에 끝 시각이 없으므로, 길이는 언제나 작업 기간으로 대신한다. 연결 없는 CLI 세션은 `durations`가 null이다. 같은 실행의 `job-` 행과 `session-` 행을 같이 고르면 작업 행에는 테스트가 없어 "No common test cases"가 된다(`web_history_compare.go:121-124`). 한계 문장이 영어라 화면이 한국어로 바꾸지 못한다.

### `GET /api/v1/history/{runId}/export`

- **응답 Go 타입**: `app.webHistoryCapture` `{run, files}` (`web_history_contract.go:67-70`)를 직렬화하고 다시 비밀을 제거한 바이트(`web_history.go:235-248`). `Content-Disposition: attachment`(`history.go:62-72`).
- **원본 데이터**: 캡처 사본 전체. `files`는 허용 목록으로 캡처한 모든 파일의 문자열 본문이다.
- **갱신 방식**: "결과 내보내기" 클릭(`HistoryPanel.svelte:80-87`).
- **쓰는 화면**: `web/src/HistoryPanel.svelte`(blob 다운로드만, 파싱하지 않음).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `run` | object `WebRun` | 아니오 | `Detail`이 아닌 캡처 원본이라 `finishedAt`, `coverage`, `observations`가 없다 | 캡처 | 아니오 |
| `files` | object 파일이름→string | 아니오 | 작업 기록은 `job.json` 하나 (`web_history.go:98`) | 캡처 | 아니오 |

- **문제**: 내보내기 파일의 `run`과 화면이 본 상세 응답이 다르다(작업 기간, 끝 시각이 없다). `env.json`과 `chainstate` 파일은 이 경로로만 밖에 나온다.

### `DELETE /api/v1/history/{runId}`

- **응답 Go 타입**: 본문 없음, 204 (`internal/dashboard/history.go:73-83`).
- **원본 데이터**: 관리자만(`security.go:125-128`, `web_history.go:251-253`). 끝난 기록이거나 연결 없는 CLI 세션이어야 하고, 연결된 작업도 끝나 있어야 한다(`web_history.go:260-271`). 그 실행 기간에만 속한 지표·로그를 먼저 지우고, Web 소유 세션 디렉터리를 지운 뒤, 캡처를 묘비로 바꾼다(`web_history.go:272-287`, `web_history_observations.go:95-123`).
- **갱신 방식**: 상세에서 "보관 결과 삭제"와 확인 클릭(`HistoryPanel.svelte:88-98`, `:134`, `:143`).
- **쓰는 화면**: `web/src/HistoryPanel.svelte`.

- **문제**: 화면은 끝 상태 네 개일 때만 삭제 버튼을 켠다(`HistoryPanel.svelte:17`, `:134`). 서버는 `unknown` 상태의 연결 없는 CLI 세션도 지울 수 있게 하는데(`web_history.go:260-265`) 화면은 그 기록의 삭제 버튼을 꺼 둔다. 지표 삭제가 일부만 되면 500과 고정 문장이 온다(`history.go:75-77`). 작업 기록(`job-<id>`)을 지워도 `jobs.json`의 작업은 남으므로 다음 `Sync`에서 묘비 때문에 다시 생기지 않을 뿐 작업 목록에는 계속 나온다(`web_history.go:96`).

### 있는데 화면에 오지 않는 데이터

- **실패 증거 `observations/`**: 테스트가 실패하면 엔진이 `tests/<NNN>_<id>/observations/` 아래에 `health.json`(노드 상태 보고), `processes.json`(프로세스 원장 사본), `node<N>.log`(노드 로그 전체), `gather-problems.txt`(수집 실패 이유)를 남긴다(`internal/testengine/failuredata.go:39-105`, `internal/core/session/record_impl.go:158-172`). 캡처 허용 목록이 `tests/<x>/observations`를 디렉터리로 허용하지 않으므로 이 파일들은 히스토리에 들어오지 않는다(`history_capture.go:126-131`). 작업 상세의 원인 영역(`JobDetail.svelte:117-121`)이 실시간 로그 보관소 대신 이 증거를 보여 줄 수 있다. 노드 로그는 수 MiB가 될 수 있어 지금의 8 MiB·64 MiB 한도와 같이 다시 정해야 한다.
- **세션의 `logs/`, `nodes/`, `keys/`**: 디렉터리째 제외된다(`session_impl.go:18-24`, `history_capture.go:126-131`). `keys/`는 의도된 제외다.
- **chainstate와 env.json 본문**: 캡처는 되지만 상세 응답의 `summary.results`에는 `tests/` 파일만 들어가서(`web_history_capture.go:85-87`) 높이·피어 수 추이(`chainstate_sink.go:16-22`)와 환경의 노드 표(`environment_impl.go:153-159`)는 내보내기 파일에만 있다. 실행 타임라인의 축 위에 높이 추이를 그릴 수 있는 자료다.
- **steps.json의 나머지 필드**: 화면은 실패한 단계의 `Index`, `Type`, `On`, `Error`만 쓴다(`run-timeline.mjs:22`). 성공한 단계의 `Hash`, `Receipt`, `Signer`, `Nonce`, `Gas`, `Deployment`(배포 주소 기대값과 실제값, 바이트코드 해시)는 원본 JSON 상자에서만 보인다(`HistoryPanel.svelte:144`).
- **assert.json의 `Provenance`와 성공한 단언**: 출처(RPC 메서드와 원시 응답, 로그 파일과 줄)는 화면에 오지만 읽지 않는다(`run-timeline.mjs:21`). `postaction.json`과 `artifacts.json`(테스트가 실제로 쓴 genesis·config·명령·배포의 참조)도 읽지 않는다.
- **프로세스 원장**: 네트워크 데이터 디렉터리의 `process.json`에 PID, 라벨, 바이너리, 실행 명령줄, 데이터 디렉터리, 호스트, 리비전이 있다(`internal/core/process/process.go:12-30`, `internal/core/process/ledger.go:15`, `:189`). 작업 API는 이것을 내보내지 않고 재확인 결과의 `process` 한 단어로 줄인다(`web_job_reconcile.go:134-137`).
- **chain-record의 단계 기록과 실행 입력 해시**: `chain-record.json`의 `steps`는 단계마다 `{done, result, detail, err, failed, startedAt, at}`을 가진다(`internal/core/session/composition.go:70-83`, `internal/chainsetup/state.go:120-125`). `failed`는 분류된 실패 상태 이름이다. `launchInputs`는 대상 경로별 입력 파일 해시이고 `statePath`는 구성 기계가 멈춘 위치다(`internal/chainsetup/state.go:96-119`). 작업의 `phases[].message`는 이 중 오류 문장만 전달한다(`web_chain_engine.go:782-785`). 단계 실패 화면이 `failed` 분류와 `detail`을 쓸 수 있다.
- **lifecycle 상태와 LogRec**: 구성 단계의 실패 분류는 `lifecycle.Status`(영역·단계·슬롯을 담은 uint32, `internal/core/lifecycle/status.go:5-40`)로 계산되고(`internal/chainsetup/failure_status.go:20-60`), 상태 기계는 최근 20개 메시지의 처리 경로를 `LogRec{Time, What, Processed, Original, Dest}`로 들고 있다(`internal/core/statemachine/logrec.go:19-25`, `machine.go:294`). 둘 다 Web API로 나오지 않는다. 메모리에만 있는 `LogRec`가 프로세스 밖에 저장되는지는 확인하지 못했다.
- **작업 쪽 숨은 필드**: `credentialRefs`, `requestId`, `retention`, `planId`, `error.code`는 응답에 있지만 화면이 읽지 않는다. `jobs.json`의 감사 기록(`web_jobs.go:25-28`, `:118`)과 히스토리 감사 기록(`web_history_contract.go:71-76`)은 API로 나오지 않는다. 계획의 `claims` 원본과 엔진 payload(`web_job_contract.go:62-70`)도 나오지 않는다.

### 화면이 직접 계산하는 것

- `jobSummary` (`web/src/job-view.mjs:8-18`): 활성 여부, 실패한 단계, 재확인 필요 여부를 계산한다. 끝 상태 목록과 재확인 조건을 서버(`web_job_contract.go:137-139`, `web_job_reconcile.go:58`)와 따로 적었으므로 서버 읽기 모델의 `active`, `needsCheck` 필드로 옮길 후보다.
- `failureHeadline` (`job-view.mjs:34-53`): 실패한 테스트, 노드, 기대값과 실제값으로 첫 문장을 만든다. 한국어 문장이라 화면에 두되, "어느 실행의 어느 테스트의 어느 단언이 대표 실패인가"를 고르는 부분은 서버가 정할 수 있다.
- `jobDuration`, `phaseRows` (`job-view.mjs:57-70`): 0년 시각을 빈 값으로 바꾸고 길이를 계산한다. 서버가 빈 시각을 생략하면(`web_job_contract.go:91-92`를 포인터나 문자열로) 이 보정이 필요 없다. 이전 후보다.
- `testRows` (`web/src/run-timeline.mjs:14-39`): 테스트 디렉터리 이름 규칙을 다시 만들고 PascalCase 결과 파일을 읽어 실패한 단언과 단계를 모은다. 엔진의 파일 규칙(`read.go:141-143`)을 화면이 알아야 하므로 서버가 테스트별 `{failedAsserts, stepErrors, reason, durationMs}`를 주는 쪽으로 옮길 1순위 후보다.
- `verdictSegments`, `placeOnAxis`, `phaseSegments`, `testLogSpan` (`run-timeline.mjs:41-77`): 비율과 축 위치 계산이다. 표시 계산이라 화면에 남겨도 된다. `testLogSpan`의 앞뒤 5초 여유는 화면 정책이다(`JobDetail.svelte:60`, `RunTimeline.svelte:15`).
- `assertDiff` (`run-timeline.mjs:83-88`): 정수 문자열 두 개의 차이를 `BigInt`로 계산한다. 서버가 큰 정수를 문자열로 정규화해 주면 화면 계산은 그대로 두어도 된다.
- `parseHistoryJSON` (`web/src/history-json.js:3-9`): 큰 정수 리터럴을 문자열로 바꾼다. 서버가 `Expected`, `Actual`, `Nonce` 같은 값을 처음부터 문자열로 내보내면 필요 없어진다.
- `nodeInMessage` (`job-view.mjs:89-93`): 단계 실패 문장에서 노드 이름을 찾아낸다. 엔진은 실패한 노드를 구조화된 값으로 알지 못하고 문장에만 적으므로, 서버가 단계 실패에 노드를 필드로 붙이는 쪽이 낫다.
- `attentionJobs` (`web/src/dashboard-view.mjs:16-22`): 실패·중단 작업과 실패한 단계를 고른다. `jobSummary`와 같은 계산을 다른 이름으로 한 번 더 한다.
- `JobDetail.svelte`의 `failed`, `failingRuns`, `endedAt`, `failureAt` (`JobDetail.svelte:40-42`, `:77-78`): 실행 기록의 `counts`와 단계 끝 시각으로 실패 시점을 정한다. 서버의 작업 기간 계산(`web_history_observations.go:40-62`)과 같은 일을 화면이 다시 한다.
- `HistoryPanel.svelte`의 `remember` (`HistoryPanel.svelte:24-28`): 지금까지 받은 페이지에서 체인, 실행자, 케이스 선택지를 모은다. 받은 페이지에만 의존하므로 선택지가 불완전하다. 서버가 필터 선택지를 따로 주는 후보다.
- `HistoryPanel.svelte:62-72`의 실행 노드 목록: 실행의 노드 이름을 지표 시계열의 `nodeId`에서 거꾸로 얻는다. 캡처한 `env.json`의 `nodes[]`에 이미 노드 표가 있으므로(`environment_impl.go:153-159`) 서버가 상세 응답에 노드 목록을 넣는 쪽으로 옮길 후보다.

## 설정 입력과 계정

### 이 영역 전체에 걸리는 공통 사항

**오류 응답은 보안 계층이 한 번 더 감싼다.** `/api/` 아래 모든 응답은 `secureResponse`를 지난다(`internal/dashboard/security.go:53`). 상태가 400 이상이면 원래 본문을 버리고 `{code, message, requestId}`를 새로 만든다(`security.go:338-374`). 원래 본문의 이유가 살아남는 것은 **GET/HEAD가 아닌 요청**이면서 상태가 400, 409, 422일 때뿐이다(`security.go:53`의 `explain`, `security.go:343`). 이때 원래 본문이 JSON이면 `message`, `fieldErrors`, `errors`, `details`만 위로 올리고 나머지 키는 `details` 아래로 옮긴다(`security.go:347-363`). 원래 본문이 평문이면 2048바이트까지 `message`에 넣는다(`security.go:365-369`). 그래서 다음 세 가지가 생긴다.

- GET 요청의 오류는 언제나 상태 문구(`"Not Found"`, `"Unprocessable Entity"` 등)만 남는다. 예를 들어 `GET /documents/{id}?revision=abc`는 `"invalid document revision"`을 만들지만(`internal/dashboard/deployment.go:369`) 화면에는 `422: Unprocessable Entity`만 간다.
- 428(`If-Match required`, `deployment.go:297`)과 401, 403, 404, 500은 POST여도 이유가 사라진다.
- 서버 핸들러는 `http.Error`로 JSON 문자열을 평문 Content-Type으로 쓰지만(`deployment.go:330-345`), `secureResponse.WriteHeader`가 400 이상이면 `application/json`으로 바꾼다(`security.go:280-282`).

**화면의 오류 해석은 한 곳이다.** `errorMessage`(`web/src/api-error.mjs:4-13`)는 `fieldErrors`와 `errors`를 이어 붙여 `"path: message"`로 만들고, 둘 다 없으면 `message`, 그다음 `code`를 쓴다. `responseError`(`api-error.mjs:17-30`)는 `status`, `details`, 경로 붙은 `errors` 배열을 Error 객체에 붙인다. `requestId`는 어디서도 읽지 않는다. 그래서 사용자가 감사 기록과 오류를 맞춰 볼 방법이 화면에 없다.

**오류 본문 모양은 엔드포인트마다 다섯 가지다.** 이 영역에서 실제로 나오는 모양을 정리하면 아래와 같다.

| 모양 | 어디서 나오나 | 경로가 있나 |
|---|---|---|
| `{message, fieldErrors:[{path,code,message}]}` | `WebFieldError`를 돌려주는 문서 저장(`deployment.go:341-346`) | 있다. `/name`, `/contractVersion`, `/assetRefs`, `/presetRefs`, `/caseRefs`, `/content` 중 하나(`internal/app/deployment.go:62-82`) |
| `{message, details:{holders}}` / `{message, details:{missingAccess}}` | 409 바쁨, 422 접근 없음(`deployment.go:327-340`) | 없다 |
| 200 응답 안의 `{valid, errors:[{path,code,message}], warnings}` | `documents/validate`, `documents/import`의 `validation` | 있다. 다만 핸들러마다 다르다(아래) |
| 422 응답 `{valid:false, errors:[{path,code,message}]}` | `test-cases/import`(`internal/dashboard/testcase.go:66-68`) | 있다. `valid`는 보안 계층이 `details.valid`로 옮긴다 |
| 200 응답 안의 `{valid, errors:[string]}` | `manifests/validate`(`internal/dashboard/manifests.go:140-145`) | 없다. 문자열 배열이다 |
| 평문 | 나머지 거의 전부(`"invalid deployment input"`, `"select exactly one kind and one file"` 등) | 없다 |

**목록 쪽 나누기.** `writePage`(`internal/dashboard/paging.go:52-59`)는 id 문자열 순으로 정렬하고 `{items, nextCursor}`를 돌려준다. 기본 25개, 1~100개, `nextCursor`는 마지막 쪽에서 `null`이다(`paging.go:14-49`). 화면은 `allItems`(`web/src/pages.mjs:3-13`)로 100개씩 끝까지 읽는다. 감사 기록만 커서가 숫자 오프셋이고 시간 역순이다(`internal/app/web_audit.go:56-85`). `manifests`, `manifest-assets`, `chain-presets`는 쪽을 나누지 않는다.

**성공 응답도 다시 직렬화된다.** 성공 본문은 `RedactWeb`이 JSON으로 읽고 `password`, `passwordHash`, `privateKey`, `private_key`, `passphrase`, `setupToken`, `key_passphrase` 키의 값을 `"[REDACTED]"`로 바꾼 뒤 다시 직렬화한다(`internal/app/web_redaction.go:16-71`, `security.go:294`). `UseNumber`로 읽으므로 큰 정수는 숫자 그대로 남는다. 객체 키 순서는 Go `map` 직렬화 순서(알파벳순)로 바뀐다. `GET /contracts/chain-preset`만 이 키 치환을 끈다(`security.go:57-59`).

**화면은 로그인 직후 거의 모든 편집기를 한꺼번에 띄운다.** `App.svelte`는 `/chains`, `/tests`, `/settings` 화면을 `hidden` 속성으로 숨길 뿐 조건부로 그리지 않는다(`web/src/App.svelte:112-115`). 그래서 `ChainPresetEditor`, `DeploymentEditor`, `DSLEditor`, `SettingsPanel`(그 안의 `CredentialManager`, `UserManager`, `AuditPanel`, `AssetManager`, `ManifestManager`)이 세션이 생기면 경로와 상관없이 각자 목록을 읽는다. `JobPanel`만 `/chains`나 `/tests`일 때 그린다(`App.svelte:114`).

---

### `GET /api/v1/contracts/deployment`

- **응답 Go 타입**: `map[string]any` = `ContractWithDigest(map[string]app.DeploymentField)` (`internal/dashboard/deployment.go:81-84`, `internal/app/web_contract_digest.go:22-34`, `internal/app/deployment_contract.go:12-21`).
- **원본 데이터**: `resource.Set`과 `resource.WorkspaceConfig`의 Go 구조체 태그를 reflect로 읽어 만든다(`deployment_contract.go:25-64`, `65-106`). 디스크를 읽지 않는다. 요청마다 새로 계산한다.
- **갱신 방식**: `DeploymentEditor`가 세션이 생기면 한 번 읽고(`web/src/DeploymentEditor.svelte:66`), 저장이 409 "engine contract changed"로 거절되면 다시 읽는다(`DeploymentEditor.svelte:57`).
- **쓰는 화면**: `web/src/DeploymentEditor.svelte`, `web/src/DeploymentFields.svelte`, `web/src/schema-problems.mjs`, `web/src/contract-draft.mjs`.

필드가 많은 스키마라 최상위 구조와 반복 단위만 적는다.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `server-set` | object(DeploymentField) | 아니오 | 스키마 조각 | `resource.Set`에서 `dataRoot`, `ssh.password*`, `ssh.key_file`, `ssh.key_passphrase_file`, `pool.hosts[].ssh`를 지운 것(`deployment_contract.go:26-45`) | 예 (`DeploymentEditor.svelte:24`, `83`) |
| `workspace-config` | object(DeploymentField) | 아니오 | 스키마 조각 | `resource.WorkspaceConfig`, `inputs.mode`와 `execution.chain`에 enum을 붙임(`deployment_contract.go:49-62`) | 예 (같은 곳) |
| `*.type` | string | 아니오 | `object`/`array`/`boolean`/`integer`/`string` | `deployment_contract.go:69-105` | 예 (`DeploymentFields.svelte:21-62`) |
| `*.properties` / `*.required` / `*.items` / `*.additionalProperties` / `*.enum` | object/array | 예(omitempty) | | `deployment_contract.go:14-20` | 예 (`DeploymentFields.svelte:24-55`, `schema-problems.mjs:8-18`) |
| `*.minimum` / `*.maximum` | number | 예(omitempty, 포인터) | 포트 대역 base·step, `ssh.port`에만 붙음 | `deployment_contract.go:80-86` | 예 (`DeploymentFields.svelte:60`, `schema-problems.mjs:19-21`) |
| `x-contractDigest` | string | 아니오 | SHA-256 hex | `web_contract_digest.go:31` | 예 (`contract-draft.mjs:5`) |
| `x-contractVersion` | string | 아니오 | `"2"` | `web_contract_digest.go:32` | 예 (`contract-draft.mjs:5`) |

- **문제**: `DeploymentField`에는 `title`, `default`, `minLength`, `pattern`이 없다(`deployment_contract.go:12-21`). 그래서 `schema-problems.mjs:23-26`의 `minLength`, `pattern` 검사는 이 계약에 대해서는 한 번도 걸리지 않는다. 필드 이름이 사람 말이 아니라 YAML 키(`p2pBase` 등)로만 보이고, 기본값은 `DeploymentFields.svelte:9-16`이 타입만 보고 `0`, `''`, `false`, enum 첫 값으로 지어낸다. 포트 `base`에 0이 기본으로 들어간다. `pool.hosts`의 원소는 `resource.HostSpec`인데 엔진은 문자열 줄임 표기도 받는다. 그래서 화면은 `typeof h === 'string'`을 따로 처리한다(`DeploymentEditor.svelte:68`, `JobPanel.svelte:154`). 공유 문서의 `ssh.user`, `ssh.port`는 계약에 남아 있는데(`internal/resource/serverset.go:76-78`), 개인 자격증명에도 `sshUser`가 있어(`internal/app/deployment_store.go:78`) SSH 사용자를 정하는 곳이 두 곳이다. 실제 접속은 자격증명 쪽 `SSHUser`를 쓴다(`internal/app/deployment_job_credentials.go:68`).

### `GET /api/v1/contracts/chain-preset?chain=`

- **응답 Go 타입**: `map[string]any`. `chain`이 있으면 `app.ChainPresetContract(chain)`, 없으면 `dsl.SchemaV2` 원문을 `ContractWithDigest`로 감싼다(`internal/dashboard/chainpreset.go:26-50`, `internal/app/chainpreset.go:66-104`).
- **원본 데이터**: 내장 DSL v2 스키마(`internal/dsl/schema/v2.schema.json`)에 레지스트리의 체인 manifest dialect 옵션(`dialect.OptionSchemas`)과 `nodeconfig.ConfigOptionSchemas()`를 덧붙인다(`chainpreset.go:98-102`). 요청마다 계산한다.
- **갱신 방식**: 프리셋을 고를 때(`web/src/ChainPresetEditor.svelte:119`), 공유 구성을 불러올 때(`ChainPresetEditor.svelte:61`, `73`), 저장이 계약 변경으로 거절될 때(`ChainPresetEditor.svelte:88`). DSL 편집기는 열린 문서의 체인이 바뀔 때마다(`web/src/DSLEditor.svelte:26`), `JobPanel`은 node.swap 설정 스키마가 필요할 때(`web/src/JobPanel.svelte:70`).
- **쓰는 화면**: `ChainPresetEditor.svelte`, `DSLEditor.svelte`, `JobPanel.svelte`, `chain-contract.mjs`, `DSLField.svelte`.

큰 스키마라 최상위와 이 영역에서 바꾸는 부분만 적는다.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `$defs.envSpec` | object | 아니오 | JSON Schema | `v2.schema.json` | 예 (`ChainPresetEditor.svelte:220`) |
| `$defs.envSpec.properties.chain` | object `{const}` | 아니오 | 체인 id | `chainpreset.go:82` | 아니오(값은 문서에서 읽음) |
| `$defs.envSpec.properties.topology.oneOf[0].properties.{bp,en,pn}` | object | 아니오 | `minimum`, `default`, `title` | `chainpreset.go:84-88`. bp 기본 4, en·pn 기본 0 | 예 (`chain-contract.mjs:5-10`) |
| `...topology.oneOf[0].properties.syncMode` | object | 아니오 | enum `full/snap/archive`, 기본 `full` | `chainpreset.go:88` | 예 (`chain-contract.mjs:11-12`) |
| `...topology.oneOf[1].properties.nodes.items.properties.role.enum` | array | 아니오 | `bp/en/pn` | `chainpreset.go:92` | 예 (`chain-contract.mjs:8`) |
| `$defs.envSpec.properties.launch` | object | 아니오 | 범위 `all/bp/en/pn/node<N>` → dialect 옵션 | `chainpreset.go:98-100` | 예 (`DSLField` 경유) |
| `$defs.envSpec.properties.config.additionalProperties` | object | 아니오 | 설정 키 스키마 | `chainpreset.go:101-102` | 예 (`JobPanel.svelte:37`) |
| `x-contractDigest`, `x-contractVersion` | string | 아니오 | | `web_contract_digest.go:31-32` | 예 (`contract-draft.mjs:5`) |

- **문제**: 알 수 없는 체인은 400 `"unsupported chain"`인데(`chainpreset.go:38-40`), GET이므로 화면에는 `400: Bad Request`만 간다. `ChainPresetEditor.choose`는 그마저 버리고 한국어 고정 문구로 바꾼다(`ChainPresetEditor.svelte:120`). 역할 `bp/en/pn`, 기본 bp 4, syncMode 목록이 Go 코드 안에 문자열로 박혀 있다(`chainpreset.go:84-95`). 같은 목록이 `launch`와 `config`의 `propertyNames` 정규식에도 따로 적혀 있다(`chainpreset.go:100`, `102`). 화면 쪽 `chain-contract.mjs`는 이 계약에서 읽으므로 중복이 아니지만, Go 안에서 세 번 반복된다.

### `GET /api/v1/contracts/dsl`

- **응답 Go 타입**: `map[string]any` = `ContractWithDigest(DSLContract()의 schema)` (`internal/dashboard/testcase.go:51-59`, `internal/app/testcase.go:29-155`).
- **원본 데이터**: `dsl.SchemaV2`에 `testhelper.EditorVocabulary()`가 등록한 동작·단언·읽기의 매개변수 스키마를 끼워 넣는다. 요청마다 계산한다.
- **갱신 방식**: `DSLEditor`가 세션이 생기면 한 번(`DSLEditor.svelte:51`), 저장이 계약 변경으로 거절되면 다시(`DSLEditor.svelte:30`).
- **쓰는 화면**: `DSLEditor.svelte`, `DSLStatementEditor.svelte`, `DSLField.svelte`, `dsl-form.js`, `dsl-tree.mjs`.

큰 스키마라 최상위 구조만 적는다. 최상위 키는 `$schema`, `$id`, `title`, `description`, `oneOf`, `$defs`, `x-contractDigest`, `x-contractVersion`이다. `$defs`에는 원래의 `envSpec`, `keySource`, `caseSpec`, `statement`, `doStatement`, `expectStatement`, `presetValue`에 더해 `action-<이름>`, `assertion-<이름>`, `reader-<이름>`, `v1Do`, `v1Assert`, `v1Spec`, `inlinePreset`, `assertion-rpc`가 들어간다(`testcase.go:38-153`). 반복 단위는 `$defs`의 문장 스키마 하나다.

- **문제**: 이 계약은 체인을 모른다. 체인별 `launch`, `config` 어휘는 `contracts/chain-preset?chain=`에서 따로 받는다(`DSLEditor.svelte:26`). 그래서 DSL 편집기 한 화면이 계약 두 개를 동시에 들고 있고, 저장 때 보내는 `contractDigest`는 DSL 계약 것뿐이다(`DSLEditor.svelte:30`). 체인 계약이 바뀌어도 케이스 저장은 거절되지 않는다.

### `GET /api/v1/vocabulary`

- **응답 Go 타입**: `app.TestCaseContract` (`internal/app/testcase.go:22-26`), 원소 `testhelper.VocabularyEntry` (`internal/testhelper/editor_contract.go:15-21`).
- **원본 데이터**: `DSLContract()`의 첫 반환값(`internal/dashboard/testcase.go:46-49`). 등록된 builtin 목록이다.
- **갱신 방식**: `DSLEditor`가 세션이 생기면 한 번(`DSLEditor.svelte:51`).
- **쓰는 화면**: `DSLEditor.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `contractVersion` | string | 아니오 | `"2"` | `testcase.go:154` | 아니오 |
| `grammarContractId` | string | 아니오 | `"dsl"` | `testcase.go:154` | 아니오 |
| `entries[]` | array | 아니오 | | `testhelper.EditorVocabulary` | 길이만 읽음 (`DSLEditor.svelte:78`) |
| `entries[].name` / `.kind` / `.contractId` / `.schemaRef` | string | 아니오 | kind는 `action/assertion/reader` | `editor_contract.go:28-30` | 아니오 |
| `entries[].prerequisites` | array | 예(omitempty) | | `editor_contract.go:20` | 아니오 |

- **문제**: 화면은 개수 표시에만 쓴다. 문장 선택지는 `contracts/dsl`의 `$defs.doStatement.oneOf`에서 다시 만든다. `prerequisites`(그 문장이 요구하는 능력)는 서버에 있지만 화면에 오지 않는다.

### `GET /api/v1/contracts/access`

- **응답 Go 타입**: `app.WebAccessContract` (`internal/app/web_access_contract.go:14-17`), 원소 `app.WebChoice` (`web_access_contract.go:6-10`).
- **원본 데이터**: 코드에 박힌 `webRoles`, `webCredentialKinds`(`web_access_contract.go:19-29`).
- **갱신 방식**: `SettingsPanel`이 세션이 생기면 한 번(`web/src/SettingsPanel.svelte:11`).
- **쓰는 화면**: `SettingsPanel.svelte` → `CredentialManager.svelte`(kinds), `UserManager.svelte`(roles).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `roles[].id` | string | 아니오 | `viewer/operator/administrator` | `web_access_contract.go:20-22` | 예 (`UserManager.svelte:7`, `21`) |
| `roles[].label` | string | 아니오 | 한국어 | 같은 곳 | 예 (`UserManager.svelte:7`) |
| `roles[].description` | string | 예(omitempty) | | 같은 곳 | 아니오 |
| `credentialKinds[].id` | string | 아니오 | `private-key/password/account-key` | `web_access_contract.go:26-28` | 예 (`CredentialManager.svelte:8`, `56`) |
| `credentialKinds[].label` | string | 아니오 | | 같은 곳 | 예 |
| `credentialKinds[].description` | string | 예 | | 같은 곳 | 아니오 |

- **문제**: 이 계약이 "화면이 목록을 따로 들지 않게" 하려고 있지만(`web_access_contract.go:12-13`), 화면은 여전히 역할 id를 문자열로 비교한다(아래 "하드코딩 기본값과 중복 어휘"). 서버 안에서도 같은 역할이 두 이름이다. 세션과 사용자 저장소는 `administrator`, `DeploymentActor`는 `admin`이다. 변환은 `WebAuthenticator` 한 곳에서 한다(`internal/dashboard/security.go:22-25`). `LoginPanel`(`web/src/LoginPanel.svelte:35`)과 `DeploymentEditor`(`DeploymentEditor.svelte:74`)는 라벨 대신 원래 id를 보여 준다. 이 계약에는 `x-contractDigest`가 없다.

### `GET /api/v1/contracts/manifest`

- **응답 Go 타입**: `map[string]any{contractVersion, families, dialects, protocols}` (`internal/dashboard/manifests.go:59-61`).
- **원본 데이터**: `registry.FamilyNames()`, `nodeconfig.DialectNames()`, `registry.Names()`.
- **갱신 방식**: 해당 없음.
- **쓰는 화면**: 없음. `web/src` 전체에서 `contracts/manifest`를 부르는 곳이 없다(grep 결과 0건).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `contractVersion` | string | 아니오 | `"2"` | `manifests.go:60` | 아니오 |
| `families[]` / `dialects[]` / `protocols[]` | array of string | 레지스트리에 따라 | | 레지스트리 | 아니오 |

- **문제**: `ManifestManager`는 외부 manifest를 고를 때 이 목록 대신 파일을 그대로 올리고 서버 검증에 맡긴다. 다른 계약과 달리 `x-contractDigest`가 없다.

### `GET /api/v1/contracts/{contractId}` (그 밖의 id)

- **응답 Go 타입**: 없음. 404 평문 `"unknown contract; this server serves: ..."` (`internal/dashboard/server.go:68-73`).
- **문제**: OpenAPI는 "message names the contracts this server serves"라고 적지만(`docs/dev/web-ui.openapi.yaml:294`), GET이라 보안 계층이 `"Not Found"`로 바꾼다(`security.go:343`). 실제로 그 목록은 사용자에게 가지 않는다.

### `GET /api/v1/chain-presets`

- **응답 Go 타입**: `[]app.ChainPresetInfo` (`internal/app/chainpreset.go:17-22`). **배열 그대로**이고 `{items}`로 감싸지 않는다(`internal/dashboard/chainpreset.go:16-24`).
- **원본 데이터**: 서버 플래그 `-chain-presets`(기본 `presets/chain`, `cmd/chainbench-dashboard/main.go:32`) 디렉터리의 `*.json`을 요청마다 읽는다. 파싱 실패, 미등록 체인, `attach` 프리셋, 엔진 검증 실패는 조용히 빠진다(`chainpreset.go:41-50`). id 중복은 500이다.
- **갱신 방식**: `ChainPresetEditor`가 마운트될 때 한 번(`ChainPresetEditor.svelte:99-104`).
- **쓰는 화면**: `ChainPresetEditor.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `[].id` | string | 아니오 | | 프리셋 파일 `id` | 예 (`ChainPresetEditor.svelte:109`, `184`) |
| `[].chain` | string | 아니오 | | 프리셋 파일 `chain` | 예 (`ChainPresetEditor.svelte:119`, `184`) |
| `[].description` | string | 예(빈 문자열) | | 프리셋 파일 | 예 (`ChainPresetEditor.svelte:188`) |
| `[].content` | object | 아니오 | 프리셋 원문 JSON | 파일 바이트 | 예 (`ChainPresetEditor.svelte:110-111`) |

- **문제**: OpenAPI는 `{items: [...]}`를 약속한다(`docs/dev/web-ui.openapi.yaml:2188-2207`). 실제는 배열이다. 화면은 응답이 실패해도 아무 말 없이 돌아간다(`ChainPresetEditor.svelte:101`). 디스크 프리셋이 왜 목록에서 빠졌는지(검증 실패, 미등록 체인) 서버가 알지만 알려 주지 않는다.

### `GET /api/v1/chains/{chainId}/surface?assetId=`

- **응답 Go 타입**: `app.WebChainSurface` (`internal/app/web_chain_surface.go:99-110`), 원소 `WebChainCommand`(`73-80`), `WebChainOption`(`82-95`), `WebSurfaceSupport`(`47-52`), `WebSurfaceSource`(`54-60`), `WebBinaryFingerprint`(`62-71`).
- **원본 데이터**: 서버 플래그 `-chain-analysis`(기본 `docs/chain-analysis`)의 체인 문서 카탈로그(`LoadChainSurface`)와 엔진 매핑. `assetId`가 있으면 등록 바이너리를 실제로 실행해 help를 읽는다(`web_chain_surface.go:462-509`). 바이너리 실행은 편집 권한이 있어야 하고(`web_chain_surface.go:463-465`), 요청 하나에 45초 제한이 있다(`internal/dashboard/surface.go:27`).
- **갱신 방식**: `ChainSurfacePanel`이 체인이 바뀔 때와 바이너리를 고를 때(`web/src/ChainSurfacePanel.svelte:17-21`, `27`).
- **쓰는 화면**: `ChainSurfacePanel.svelte`(`ChainPresetEditor.svelte:235` 안에 들어감).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `chainId` / `dialect` / `contractVersion` / `fingerprint` | string | 아니오 | | 카탈로그, dialect | 예 (`ChainSurfacePanel.svelte:31`) |
| `verification` | string | 아니오 | `documented/verified/stale/incompatible` | `web_chain_surface.go:174`, `309-315` | 예 (`ChainSurfacePanel.svelte:31`) |
| `sources[].path` / `.kind` | string | 아니오 | | 카탈로그 | 예 (`ChainSurfacePanel.svelte:34`) |
| `sources[].digest` | string | 아니오 | | | 아니오 |
| `sources[].chainCommit` | string | 예(omitempty) | | | 예 |
| `sources[].capturedAt` | string | 예(포인터, omitempty) | RFC 3339 | | 예 |
| `binary` | object | 예(포인터, omitempty) | `assetId`가 있을 때만 | 바이너리 탐침 | 예 (`ChainSurfacePanel.svelte:32`) |
| `binary.checksum` / `.chainId` / `.verifiedAt` | string | 아니오 | verifiedAt은 RFC 3339 | | 아니오 |
| `commands[]` | array | 아니오(빈 배열) | | `web_chain_surface.go:270-297` | 아니오 |
| `options[].id` / `.spelling` / `.hidden` / `.scope` / `.commandPath` | | 아니오 | | 카탈로그 | 예 (`ChainSurfacePanel.svelte:40`) |
| `options[].optionKey` / `.schemaRef` / `.contractId` | string | 예(omitempty) | | 엔진 매핑 | 아니오 |
| `options[].aliases` / `.boolean` / `.emission` | | 아니오 | | | 아니오 |
| `options[].support.status` | string | 아니오 | `supported/engine-unmapped/binary-absent/unverified` | `web_chain_surface.go:330-356` | 예 (`ChainSurfacePanel.svelte:12-14`, `40`) |
| `options[].support.selectable` / `.reason` | | 아니오 | | | 예 (`ChainSurfacePanel.svelte:40`) |
| `options[].support.evidenceRefs` | array | 아니오 | | | 아니오 |
| `limitations[]` | array of string | 아니오(빈 배열) | 영어 문장 | | 예 (`ChainSurfacePanel.svelte:33`) |

- **문제**: 상태 문자열이 서버에 정해진 목록으로 선언되어 있지 않고 코드 곳곳의 리터럴이다. 화면의 `status-icon.mjs:3-10`은 `engine-unmapped`를 어느 묶음에도 넣지 않아 "알 수 없음" 아이콘이 붙는다. `ChainSurfacePanel.svelte:15`의 `icon` 함수는 정의만 되어 있고 템플릿에서 쓰이지 않는다. 체인 문서가 없으면 404(`web_chain_surface.go:467-469`)인데 GET이라 `"Not Found"`만 간다. 옵션은 화면이 상태별 개수를 직접 센다(`ChainSurfacePanel.svelte:12-14`).

### `GET /api/v1/documents?kind=`

- **응답 Go 타입**: `{items: []app.DeploymentDocument, nextCursor}` (`internal/dashboard/deployment.go:97-99`, `internal/app/deployment_store.go:37-43`).
- **원본 데이터**: 웹 데이터 디렉터리(서버 플래그 `-deployment-root`, 기본 `chainbench-out/web-ui`)의 `deployment.json` 안 `documents` 맵. 각 문서의 **최신 revision만** 나온다(`deployment_store.go:258-273`). 프로세스 메모리에 올려 둔 상태를 읽는다.
- **갱신 방식**: `DeploymentEditor`(전체 종류를 읽고 화면에서 거름, `DeploymentEditor.svelte:33`), `ChainPresetEditor`(`kind=chain-preset`, `ChainPresetEditor.svelte:48`), `DSLEditor`(`kind=case`, `DSLEditor.svelte:28`, 저장할 때 `kind=chain-preset`도, `DSLEditor.svelte:35`, `47`), `JobPanel`(전체, `JobPanel.svelte:78`). 각 화면은 마운트 때와 저장 직후에 다시 읽는다.
- **쓰는 화면**: 위 네 파일.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | 32자 hex | `deploymentID()` (`deployment_store.go:165-171`) | 예 (`DeploymentEditor.svelte:78`) |
| `items[].revision` | number | 아니오 | 1부터 | `deployment_store.go:230` | 예 (`DeploymentEditor.svelte:78`) |
| `items[].kind` | string | 아니오 | `server-set/workspace-config/chain-preset/case/run-definition` | 저장 입력 | 예 (`DeploymentEditor.svelte:33`, `JobPanel.svelte:38`) |
| `items[].name` | string | 아니오 | | 저장 입력 | 예 (`DeploymentEditor.svelte:78`, `ChainPresetEditor.svelte:176`) |
| `items[].contractVersion` | string | 아니오 | `"2"` | 저장 입력 | 아니오 |
| `items[].content` | object | 아니오 | 엔진 선언 원문 | 저장 입력(`json.RawMessage`) | 예 (`JobPanel.svelte:38`, `DSLEditor.svelte:38`) |
| `items[].assetRefs` | array of string | 아니오(`[]`) | 32자 hex | 저장 입력 | 예 (`JobPanel.svelte:138`) |
| `items[].presetRefs` | array `{id,revision}` | 예(omitempty, case만) | | 저장 입력 | 예 (`dsl-form.js:148`) |
| `items[].caseRefs` | array `{id,revision}` | 예(omitempty, run-definition만) | | 저장 입력 | 예 (`JobPanel.svelte:42`) |
| `items[].updatedAt` | string | 아니오 | RFC 3339 UTC | `deployment_store.go:230` | 아니오 |
| `items[].updatedBy` | string | 아니오 | 사용자 id | 같은 곳 | 아니오 |
| `nextCursor` | string \| null | 예 | 문서 id | `paging.go:44-47` | 예 (`pages.mjs:10`) |

- **문제**: OpenAPI는 `search` 쿼리를 적지만(`docs/dev/web-ui.openapi.yaml:699`) 서버는 `kind`만 읽는다(`deployment.go:98`). OpenAPI의 종류 enum에는 `manifest`가 있는데(`web-ui.openapi.yaml:696`) 저장 검증은 그것을 `"unsupported deployment document kind"`로 거절한다(`internal/app/deployment.go:165-166`). OpenAPI `Document` 스키마에는 `caseRefs`가 없고 `additionalProperties:false`다(`web-ui.openapi.yaml:2947-2968`). 목록이 내용 전체를 실어 나르므로 `DeploymentEditor`와 `JobPanel`은 모든 문서의 본문을 받는다. 화면이 필요한 것은 일부 종류뿐이다(`DeploymentEditor.svelte:33`). 지난 revision 목록을 주는 엔드포인트가 없다.

### `POST /api/v1/documents`

- **요청 Go 타입**: `app.DeploymentDocumentInput` (`internal/app/deployment.go:15-31`). **응답**: `app.DeploymentDocument`, 201, `ETag: "<revision>"` (`internal/dashboard/deployment.go:116-128`, `318-319`).
- **원본 데이터**: 요청 본문을 엔진 검증(`DeploymentStore.ValidateDocument`, `internal/app/deployment_presets.go:13-29`)에 통과시킨 뒤 `deployment.json`에 새 문서로 쓴다(`deployment_store.go:201-233`). 쓰기는 임시 파일, fsync, rename이다(`internal/core/session/webstore.go:65-`).
- **갱신 방식**: 클릭. 서버는 쓰는 순간 revision 1을 만든다.
- **쓰는 화면**: `DeploymentEditor.svelte:54`, `ChainPresetEditor.svelte:80`, `DSLEditor.svelte:30`, `39`(공유되지 않은 프리셋을 자동 등록), `JobPanel.svelte:44`(run-definition).

요청 필드: `kind`, `name`, `contractVersion`(반드시 `"2"`), `content`(object), `assetRefs`(선언 안의 `asset:<id>`와 순서까지 같아야 함, `internal/app/web_document_assets.go:91-105`), `presetRefs`(case만), `caseRefs`(run-definition만), `contractDigest`(선택, 저장하지 않음). 모르는 필드는 400 `"invalid deployment input"`이다(`deployment.go:281-293`).

- **문제**: 오류 모양이 저장과 검증에서 다르다. 저장은 `fieldErrors`에 진짜 경로(`/name`, `/assetRefs` 등)를 담는다(`deployment.go:341-346`). 계약 digest가 낡으면 409 `"revision changed; reload before editing: the engine contract changed..."`이다(`internal/app/web_contract_digest.go:17`). 화면은 이 문구를 정규식 `/engine contract changed/`로 알아낸다(`web/src/contract-draft.mjs:8-10`). 문자열이 바뀌면 화면 흐름이 끊긴다. 409 문구 앞머리가 `ErrDeploymentConflict`의 "revision changed; reload before editing"이라 계약 변경인데도 revision 충돌처럼 읽힌다. `DeploymentEditor`는 저장 오류의 `e.errors`를 필드에 붙이지 않고 문구만 보인다(`DeploymentEditor.svelte:31`, `54-60`). 엔진 검증기는 내용 안의 위치를 모르므로 내용 오류는 늘 `/content` 한 칸이다(`internal/app/deployment.go:42-45`). 다만 `ContentErrorPath`가 오류 문장 앞머리에서 경로를 뽑는 길이 있는데(`internal/app/content_error_path.go:16-26`), 저장 경로는 그것을 쓰지 않는다.

### `GET /api/v1/documents/{id}?revision=`

- **응답 Go 타입**: `app.DeploymentDocument`, `ETag` 포함 (`deployment.go:129-132`, `363-373`).
- **원본 데이터**: `deployment.json`의 그 문서 이력 배열. `revision`이 없거나 0이면 최신이다(`deployment_store.go:281-300`).
- **갱신 방식**: 워크스페이스가 고정한 revision이 목록(최신만 있음)에 없을 때(`DeploymentEditor.svelte:36`, `47`), 공유 구성·케이스를 열 때(`ChainPresetEditor.svelte:58`, `DSLEditor.svelte:29`), 케이스가 고정한 프리셋 revision을 읽을 때(`DSLEditor.svelte:29`), 작업 화면이 워크스페이스 문서를 맞출 때(`JobPanel.svelte:85`), 노드 상세(`web/src/NodeDetail.svelte:29`).
- **쓰는 화면**: 위 파일들.

필드는 목록의 원소와 같다.

- **문제**: 숫자가 아닌 `revision`은 일반 오류(`errors.New`)라 422로 나가고(`deployment.go:369`, `deployment.go:326`), 범위를 벗어난 revision은 404다(`deployment_store.go:288-290`). 둘 다 GET이라 이유가 지워진다. 목록이 최신만 주므로 화면이 고정 revision을 하나씩 따로 읽는다. `DeploymentEditor.refresh`는 워크스페이스마다 문서마다 순차로 요청한다(`DeploymentEditor.svelte:36`).

### `PATCH /api/v1/documents/{id}`

- **요청 Go 타입**: `app.DeploymentDocumentInput`, 헤더 `If-Match: "<현재 revision>"` 필수. **응답**: `app.DeploymentDocument` 200 (`deployment.go:133-149`).
- **원본 데이터**: 이력 배열 끝에 새 revision을 붙인다. 현재 revision과 다르면 409, kind를 바꾸면 `"document kind cannot change"` 422다(`deployment_store.go:214-224`).
- **갱신 방식**: 클릭.
- **쓰는 화면**: `DeploymentEditor.svelte:54`, `ChainPresetEditor.svelte:80`, `DSLEditor.svelte:30`.

- **문제**: 서버와 OpenAPI 모두 `PATCH`만 있다(`deployment.go:133`). `If-Match`가 없으면 428인데 보안 계층이 이유를 지운다. 문서를 지우는 엔드포인트도 없다.

### `POST /api/v1/documents/validate`

- **요청 Go 타입**: 종류에 따라 셋 중 하나가 받는다. **응답**: 200 `{valid, contractVersion, errors[], warnings[]}`.
- **원본 데이터**: 핸들러가 감싸기 순서로 이어져 있다. 등록 순서(`cmd/chainbench-dashboard/main.go:97`, `132`)에 따라 `WithChainPresets`가 먼저 `s.validateDocument`를 정하고, `WithDeployments`가 그것을 감싸고(`internal/dashboard/deployment.go:57-75`), `WithTestCases`가 다시 감싼다(`internal/dashboard/testcase.go:72-112`). 최종 등록은 `server.go:64-66`.
  1. `kind=case`이고 `presetRefs`가 없으면 테스트 케이스 핸들러가 받는다. 오류 경로는 언제나 `/content`, 형식 오류는 `/`다(`testcase.go:104-110`).
  2. `kind=chain-preset`이면 프리셋 핸들러가 받는다. 경로는 `ContentErrorPath(err)`로 뽑고, 형식 오류는 `/`, code는 `unsupported`다(`chainpreset.go:69-79`). 이 핸들러는 `presetRefs`, `caseRefs`, `contractDigest` 필드를 몰라서 보내면 400이다(`chainpreset.go:52-64`).
  3. 그 밖(`server-set`, `workspace-config`, `run-definition`, `presetRefs`가 있는 case)은 배치 핸들러가 받는다. 오류 경로는 **`WebFieldError.Path`를 무시하고** 언제나 `/content`다(`deployment.go:85-96`).
- **갱신 방식**: 클릭(검증 버튼, 저장 전 검사).
- **쓰는 화면**: `DeploymentEditor.svelte:26`, `58`, `ChainPresetEditor.svelte:155-160`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `valid` | boolean | 아니오 | | 핸들러 | 예 (`DeploymentEditor.svelte:27`, `ChainPresetEditor.svelte:229`) |
| `contractVersion` | string | 아니오 | `"2"` | 핸들러 상수 | 아니오 |
| `errors[].path` | string | 아니오 | JSON pointer | 위 1~3 | `ChainPresetEditor`만 (`ChainPresetEditor.svelte:20`, `230`). `DeploymentEditor`는 버림 (`DeploymentEditor.svelte:27`) |
| `errors[].code` | string | 아니오 | `invalid/unsupported` | 위 1~3 | 아니오 |
| `errors[].message` | string | 아니오 | 엔진 영어 문장 | 엔진 오류 | 예 |
| `warnings[]` | array of string | 아니오 | 영어 고정 문장 | 핸들러별 상수(`deployment.go:95`, `chainpreset.go:79`, `testcase.go:110`) | `ChainPresetEditor`만 (`ChainPresetEditor.svelte:231`) |

- **문제**: 같은 경로 하나가 세 핸들러로 갈라지고, 경로 정밀도와 받는 필드가 다르다. 배치 핸들러는 저장 때라면 `/name`이나 `/assetRefs`를 알려 줄 오류를 검증 때는 `/content`로 뭉갠다. `DeploymentEditor`는 그 경로마저 `''`로 바꿔 버리고(`DeploymentEditor.svelte:27`), 화면 필드 경로는 `content.pool.hosts[0].addr` 같은 점 표기라(`DeploymentFields.svelte:28`, `48`) 서버의 JSON pointer(`/content/...`)와 맞지 않는다. `ChainPresetEditor`는 `/content/...` pointer를 그대로 필드 키로 쓴다(`ChainPresetEditor.svelte:20-27`). 같은 앱 안에 경로 표기가 두 가지다. `ChainPresetEditor.validate`는 `contractDigest`를 보내지 않고 `contractVersion`만 보낸다(`ChainPresetEditor.svelte:157`).
- **3-6단계에서 고친 것**: 세 핸들러가 모두 `app.WebFieldPath`로 같은 위치를 돌려준다. 엔진의 `resource` 검증이 오류에 필드 위치(`resource.FieldError`)를 싣고, 서버 세트와 workspace-config 오류는 `/content/pool/hosts/1/addr`처럼 그 필드를 가리킨다. 문장은 바뀌지 않아 CLI 출력은 그대로다. `DeploymentEditor`는 이 pointer를 스키마를 따라 화면 경로(`content.pool.hosts[1].addr`)로 바꿔 해당 칸에 오류를 붙인다. 계약 변경 409는 `details.reason: "contract_changed"`를 싣고 문장 앞머리의 "revision changed"를 뗐으며, 화면은 문구 정규식 대신 이 값으로 알아본다.

### `GET /api/v1/documents/{id}/export?format=`

- **응답**: 원문 바이트, `Content-Type: application/octet-stream` (`deployment.go:150-163`). `format`은 없거나 `json`이면 저장한 JSON 그대로, `yaml`이면 YAML로 바꾼다(`internal/app/deployment.go:195-212`).
- **원본 데이터**: 최신 revision(이 핸들러는 `?revision=`도 받는다, `deployment.go:151`, `363-373`). 내보내기 전에 다시 검증한다.
- **갱신 방식**: 클릭.
- **쓰는 화면**: `DeploymentEditor.svelte:64`. 화면은 `api()`로 받아 JSON으로 파싱한 뒤 다시 `JSON.stringify`해 내려받는다.

- **문제**: `ChainPresetEditor`와 `DSLEditor`는 이 엔드포인트를 쓰지 않고 브라우저가 들고 있는 문서를 직접 내려받는다(`ChainPresetEditor.svelte:94-97`, `DSLEditor.svelte:95`). 서버 내보내기는 검증을 거치지만 화면 내보내기는 거치지 않는다.

### `GET /api/v1/documents/{id}/bundle`

- **응답**: JSON `{format:"chainbench-web-bundle", documents:[{kind,name,contractVersion,assetRefs,content}]}`, 첨부 파일 헤더 (`deployment.go:164-173`, `internal/app/document_bundle.go:291-340`).
- **원본 데이터**: 최신 revision. run-definition이면 그 케이스들을, 케이스면 고정한 프리셋 revision을 함께 싣는다.
- **갱신 방식**: 클릭.
- **쓰는 화면**: `DSLEditor.svelte:96`.

- **문제**: 이 묶음에는 `presetRefs`, `caseRefs`가 빠진다. 다시 가져올 때 서버가 같은 묶음 안 위치로 다시 연결한다(`document_bundle.go:118-170`).

### `POST /api/v1/documents/import`

- **요청 Go 타입**: `app.DocumentImportInput{filename, format, source, kind?}` (`internal/app/document_import.go:15-20`). **응답**: `app.DocumentImportPreview` 200 (`document_import.go:32-39`).
- **원본 데이터**: 요청의 `source` 문자열을 JSON이나 YAML(여러 문서 스트림 가능)로 읽는다(`document_bundle.go:27-92`). 묶음이면 64개까지. 원문은 AES-GCM으로 암호화해 `deployment.json`의 `imports`에 15분 동안 둔다(`document_import.go:124-142`). 서버 SSH 비밀 필드는 공유 선언에서 지운다(`document_bundle.go:175-189`).
- **갱신 방식**: 파일을 고를 때.
- **쓰는 화면**: `web/src/BundleImport.svelte:18`, `ChainPresetEditor.svelte:68`(`kind:'chain-preset'`를 붙여 묶음 해석을 끔).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `previewId` | string | 아니오 | 32자 hex | `document_import.go:66` | 예 (`BundleImport.svelte:23`) |
| `validation.valid` | boolean | 아니오 | | `document_import.go:120` | 예 (`BundleImport.svelte:47-48`, `ChainPresetEditor.svelte:70`) |
| `validation.errors[].path` | string | 아니오 | `/documents/<i>` 또는 `/documents/<i>/content`, 단일 문서는 `/content` | `document_bundle.go:101`, `192-230` | `BundleImport`만 (`BundleImport.svelte:53`). `ChainPresetEditor`는 버림 (`ChainPresetEditor.svelte:70`) |
| `validation.errors[].code` / `.message` | string | 아니오 | | | message만 읽음 |
| `validation.warnings` | array | 아니오(`[]`) | | | 아니오 |
| `redactedDocuments[]` | array of DeploymentDocumentInput | 예(무효면 `[]`) | | `document_import.go:121-123` | 예 (`BundleImport.svelte:49-50`, `ChainPresetEditor.svelte:71`) |
| `changes[]` | array of string | 아니오 | 영어 문장 | `document_bundle.go:185`, `210` | 예 (`BundleImport.svelte:55`, `ChainPresetEditor.svelte:178`) |
| `privateBindingsRequired[]` | array of string | 아니오 | `ssh.password` 류와 `ssh:<server>` 두 형식 | `document_bundle.go:180-184`, `250-251` | 예 (`BundleImport.svelte:56`) |
| `sourcePreserved` | boolean | 아니오 | 늘 `true` | `document_import.go:66` | 아니오 |

- **문제**: `privateBindingsRequired`에 형식이 다른 두 종류가 섞인다. 하나는 지운 필드 이름(`ssh.password`, 묶음이면 `/documents/0/ssh.password`), 다른 하나는 바인딩이 필요한 서버(`ssh:server1`)다. 화면은 둘을 구분하지 않고 쉼표로 이어 보인다(`BundleImport.svelte:56`). 입력 오류(형식 오류, 종류 불일치)는 미리보기가 아니라 422 평문으로 온다(`document_import.go:59-77`). 같은 "가져올 수 없음"이 응답 모양 두 가지로 나뉜다. 서버는 `server-set`의 `ssh` 비밀을 지워 주지만, 저장 검증은 같은 필드를 거절한다(`internal/app/deployment.go:110-116`). 그래서 가져오기로만 그런 파일을 받아들일 수 있다.

### `POST /api/v1/documents/import/commit`

- **요청 Go 타입**: `app.DocumentImportCommit{previewId, baseRevisions?}` (`document_import.go:40-43`). **응답**: `{items: []app.DeploymentDocument, nextCursor: null}` 201 (`deployment.go:108-115`).
- **원본 데이터**: 서버가 보관한 미리보기의 `redactedDocuments`를 다시 검증해 한 번에 저장한다(`document_import.go:148-215`). 미리보기는 만든 사람만, 한 번만, 15분 안에만 확정할 수 있다.
- **갱신 방식**: 클릭.
- **쓰는 화면**: `BundleImport.svelte:23`.

- **문제**: 화면은 `baseRevisions`를 보내지 않으므로 같은 이름 문서를 새 revision으로 잇지 못하고 늘 새 문서를 만든다. 만료와 중복 확정은 409 `"revision changed; reload before editing"`로 온다(`document_import.go:158-160`). 실제 원인과 다른 문구다. 내부 검증 실패는 모두 같은 문장 `"import declaration no longer satisfies shared document validation"`이라 어느 문서인지 알 수 없다(`document_import.go:171`, `227-241`).

### `POST /api/v1/test-cases/import`

- **요청 Go 타입**: `app.TestCaseInput{content, presets?}` (`internal/app/testcase.go:158-162`). **응답**: `app.TestCasePrepared` 200 (`testcase.go:163-170`), 실패는 422 `{valid:false, errors:[{path,code,message}]}` (`internal/dashboard/testcase.go:60-71`).
- **원본 데이터**: 요청 본문만. 노드 키 문자열을 먼저 거르고(`internal/app/web_document_secrets.go:12-46`), v1이면 실행 의미가 같을 때만 v2로 옮긴다(`testcase.go:175-248`). 디스크를 쓰지 않는다.
- **갱신 방식**: 파일을 고를 때, 검증 버튼, 공유 케이스를 열 때(viewer 제외), 저장이 계약 변경으로 거절될 때(`DSLEditor.svelte:29`, `30`, `53`, `70`).
- **쓰는 화면**: `DSLEditor.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `content` | object | 아니오 | 옮긴 뒤의 선언 | `testcase.go:247` | 예 (`DSLEditor.svelte:53`, `70`) |
| `original` | object | 아니오 | 보낸 그대로 | 같은 곳 | 아니오 |
| `migrated` | boolean | 아니오 | | | 예 (`DSLEditor.svelte:53`) |
| `semanticFingerprint` | string | 아니오 | | `testCaseSemantics` | 예 (`DSLEditor.svelte:29`, `53`, `70`) |
| `contractVersion` | string | 아니오 | `"2"` | | 아니오 |
| `migrationMessage` | string | 예(omitempty) | 영어 문장 | `testcase.go:206`, `209` | 예 (`DSLEditor.svelte:53`) |

- **문제**: 오류 경로는 `ContentErrorPath`로 뽑지만, 이 엔드포인트의 요청 본문은 `{content}`이므로 pointer `/content`는 요청 기준으로는 맞고 문서 기준으로는 한 단계 위다. 화면은 `/content`로 시작하는 오류만 문장 편집기에 넘긴다(`DSLEditor.svelte:50`). `valid:false`는 보안 계층이 `details.valid`로 옮기므로 최상위에서는 사라진다(`security.go:352-363`).

### `GET /api/v1/workspaces` · `POST /api/v1/workspaces` · `GET /api/v1/workspaces/{id}` · `PATCH /api/v1/workspaces/{id}`

- **요청 Go 타입**: `app.DeploymentWorkspaceInput{name, documents:[{id,revision}]}` (`deployment_store.go:51-55`). **응답**: `app.DeploymentWorkspace` (`deployment_store.go:57-62`), 목록은 `{items, nextCursor}`. PATCH는 `If-Match` 필수(`deployment.go:174-210`).
- **원본 데이터**: `deployment.json`의 `workspaces` 맵. 저장할 때 서버 세트 하나와 workspace-config 하나를 정확히 고정했는지 본다(`deployment_store.go:302-330`). 워크스페이스는 이력을 남기지 않고 덮어쓴다(`deployment_store.go:355-356`).
- **갱신 방식**: 편집기 마운트와 저장 직후(`DeploymentEditor.svelte:34`, `62`), `BundleImport` 확정 후 만들기(`BundleImport.svelte:35`), `CredentialManager`(`CredentialManager.svelte:11`), `JobPanel`(`JobPanel.svelte:78`), 노드 상세(`NodeDetail.svelte:28`), 히스토리 필터(`HistoryPanel.svelte:45`), 앱 머리말 이름표(`App.svelte:45`, 첫 100개만).
- **쓰는 화면**: 위 파일들.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `id` | string | 아니오 | 32자 hex | `deployment_store.go:343` | 예 |
| `name` | string | 아니오 | | 입력 | 예 (`DeploymentEditor.svelte:93`, `CredentialManager.svelte:9`) |
| `revision` | number | 아니오 | | `deployment_store.go:355` | 예 (`DeploymentEditor.svelte:46`, `93`) |
| `documents[].id` / `.revision` | string/number | 아니오 | 정확히 2개 | 입력 | 예 (`DeploymentEditor.svelte:47-49`, `JobPanel.svelte:138`) |

- **문제**: 문서 두 개 중 어느 것이 서버 세트인지 응답이 말하지 않는다. 화면이 문서를 읽어 `kind`로 가린다(`DeploymentEditor.svelte:49`, `JobPanel.svelte:51`). `App.svelte:45`는 `limit=100` 첫 쪽만 읽어 101번째 워크스페이스부터 이름이 비어 보일 수 있다. 서버 세트가 둘이면 `"duplicate server-set"`, revision이 범위를 벗어나면 409로 온다(`deployment_store.go:309-322`).

### `GET /api/v1/workspaces/{id}/export`

- **응답**: JSON `{format, workspace, documents[], privateBindingsRequired[]}`, 첨부 헤더 (`deployment.go:185-194`, `document_bundle.go:260-287`).
- **원본 데이터**: 워크스페이스가 고정한 revision들.
- **갱신 방식**: 클릭.
- **쓰는 화면**: `DeploymentEditor.svelte:65`(Blob으로 받음).

- **문제**: 여기서 `privateBindingsRequired`는 `ssh:<server>` 형식만 담는다(`document_bundle.go:276-278`). 같은 이름 필드가 가져오기 미리보기에서는 다른 형식도 섞는다.

### `GET /api/v1/workspaces/{id}/credential-bindings` · `PUT /api/v1/workspaces/{id}/credential-bindings`

- **요청(PUT)**: `{serverRef, credentialId}` (`deployment.go:265-278`). **응답**: 둘 다 `map[string]string`, 키는 서버 이름, 값은 자격증명 id (`deployment_store.go:457-465`).
- **원본 데이터**: `deployment.json`의 `bindings[사용자 id][workspace id][server]`. 본인 것만 읽는다. PUT은 서버가 그 워크스페이스에 있고, 자격증명이 본인 것이며 폐기되지 않았고, `account-key`가 아닌지 본다(`internal/app/deployment_access.go:57-100`).
- **갱신 방식**: 워크스페이스를 고를 때(`DeploymentEditor.svelte:50`), 연결 버튼(`DeploymentEditor.svelte:105`), 작업 계획 직전(`JobPanel.svelte:138`).
- **쓰는 화면**: `DeploymentEditor.svelte`, `JobPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `<serverName>` | string | 예(빈 객체) | 자격증명 id | 바인딩 맵 | 예 (`DeploymentEditor.svelte:107`, `JobPanel.svelte:138`) |

- **문제**: 바인딩을 지우는 방법이 없다. 자격증명을 폐기해도 맵에는 남는다(`deployment_store.go:237-255`). 서버 이름이 키라 서버 세트 revision이 바뀌어 이름이 사라져도 남는다.

### `GET /api/v1/credentials` · `POST /api/v1/credentials` · `GET /api/v1/credentials/{id}` · `DELETE /api/v1/credentials/{id}` · `POST /api/v1/credentials/{id}/check`

- **요청(POST)**: `app.DeploymentCredentialInput{label, kind, sshUser, password?, privateKey?, passphrase?, accountKey?}` (`deployment_store.go:75-85`). **응답**: 목록은 `{items: []credentialUsage, nextCursor}`(`internal/dashboard/deployment.go:19-25`, `211-221`), POST와 단건 GET은 `app.DeploymentCredential` (`deployment_store.go:65-72`), DELETE는 204, check는 `resource.AccessCheck` (`internal/resource/access_check.go:17-23`).
- **원본 데이터**: `deployment.json`의 `credentials` 맵. 비밀은 AES-GCM 암호문, 추가 인증 데이터는 `id:ownerId`(`deployment_store.go:398-407`). 키는 `<deployment-root>/credential.key`(0600, 없으면 새로 만들되 기존 데이터가 있으면 거절, `internal/core/session/webstore.go:23-57`). 목록의 `activeJobs`는 작업 저장소에서 계산한다(`deployment.go:213-215`). check는 실제 SSH 접속으로 `chainbench resource access --json`을 돌린다(`deployment_access.go:107-158`).
- **갱신 방식**: 설정 화면 마운트와 저장·폐기 직후(`CredentialManager.svelte:11-12`, `18`, `27`), 배포 편집기 마운트(`DeploymentEditor.svelte:35`), 테스트 연결 작업에서 account-key 목록(`JobPanel.svelte:47`). check는 클릭.
- **쓰는 화면**: `CredentialManager.svelte`, `DeploymentEditor.svelte`, `JobPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | 32자 hex | `deployment_store.go:397` | 예 (`CredentialManager.svelte:35`) |
| `items[].ownerId` | string | 아니오 | | | 아니오 |
| `items[].label` | string | 아니오 | | 입력 | 예 (`CredentialManager.svelte:37`) |
| `items[].kind` | string | 아니오 | `private-key/password/account-key` | 입력 | 예 (`CredentialManager.svelte:37`, `DeploymentEditor.svelte:104`) |
| `items[].revoked` | boolean | 아니오 | | | 예 (`CredentialManager.svelte:38`) |
| `items[].createdAt` | string | 아니오 | RFC 3339 UTC | | 아니오 |
| `items[].bindings[].workspaceId` / `.serverRef` | string | 예(빈 배열) | | 바인딩 맵 역색인(`deployment_store.go:475-493`) | 예 (`CredentialManager.svelte:42-44`) |
| `items[].activeJobs[]` | array of string | 예(빈 배열) | 작업 id | 작업 저장소 | 길이만 (`CredentialManager.svelte:25`, `39`) |
| check `reachable` | boolean | 아니오 | | CLI 출력 | 아니오 |
| check `authenticated` | boolean | 아니오 | | CLI 출력 | 예 (`CredentialManager.svelte:46`, `DeploymentEditor.svelte:108`) |
| check `hostIdentity` | string | 아니오 | | CLI 출력 | 예 (`DeploymentEditor.svelte:108`) |
| check `allowedOperations[]` | array | 아니오 | `deploy` 등 | CLI 출력 | 예 |
| check `reason` | string | 예(omitempty) | | CLI 출력 | 예 |

- **문제**: 입력 모양 검사는 `validCredentialShape` 한 줄짜리 참·거짓이라(`deployment_access.go:19-32`) 어떤 필드가 틀렸는지 없이 `"invalid private credential"` 422로 온다(`deployment_store.go:389-391`). 단건 GET은 화면에서 쓰지 않는다. viewer는 보안 계층에서 `/api/v1/credentials`로 시작하는 경로 전체가 403이다(`security.go:102-105`). check는 `authenticated:false`면 감사 기록 없이 200으로 끝난다(`deployment_access.go:152-154`).

### `GET /api/v1/manifests` · `GET /api/v1/manifests/{id}` · `POST /api/v1/manifests/validate` · `POST /api/v1/manifests` · `POST /api/v1/manifests/{id}/setup`

- **요청**: validate와 저장은 `app.ManifestInput{manifest, template}` (`internal/app/manifests.go:20-23`), setup은 `{assetId}`. **응답**: 목록 `{items: []app.ManagedManifest}`(쪽 없음, `internal/dashboard/manifests.go:62-65`), 단건과 저장은 `app.ManagedManifest` (`manifests.go:25-29`), validate는 `{valid, errors:[string], contractVersion}` (`manifests.go:135-146`), setup은 `app.ManifestSetup` (`internal/app/manifest_setup.go:98-103`).
- **원본 데이터**: 내장 플러그인(`registry.Names()`)과 `<deployment-root>/manifests/<sha256>.json`(`internal/core/session/manifeststore.go:16-56`). 외부 manifest의 id는 `{manifest, template}` JSON의 SHA-256이다(`manifests.go:96-101`). setup은 `chainbench chain up --stage init --bp 4 ...`를 돌린 뒤 `chain status`를 읽는다(`manifest_setup.go:107-158`).
- **갱신 방식**: `ManifestManager` 마운트, 자료 업로드 뒤(`assetRevision`), 저장 직후(`web/src/ManifestManager.svelte:10-11`, `14`). `JobPanel` 마운트(`JobPanel.svelte:78`). validate, 저장, setup은 클릭.
- **쓰는 화면**: `ManifestManager.svelte`, `JobPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | 내장은 체인 id, 외부는 64자 hex | `manifests.go:109-135` | 예 (`ManifestManager.svelte:25`) |
| `items[].source` | string | 아니오 | `builtin/external` | | 예 (`ManifestManager.svelte:25`) |
| `items[].manifest` | object | 아니오 | `registry.Manifest` snake_case | 파일 또는 플러그인 | 예 |
| `items[].manifest.id` / `.consensus_family` / `.dialect` / `.protocol` | string | protocol은 omitempty | | `internal/core/registry/manifest.go:20-74` | 예 (`ManifestManager.svelte:28`, `30`, `JobPanel.svelte:38`) |
| `items[].manifest.chain_id` | number | 아니오 | int64를 JSON 숫자로 | `registry/manifest.go:25` | 예 (`ManifestManager.svelte:28`) |
| `items[].template` | string | 예(빈 문자열) | genesis 템플릿 원문 | | 예 (`ManifestManager.svelte:15`) |
| validate `valid` / `errors[]` | boolean / array of string | | | `manifests.go:140-145` | 예 (`ManifestManager.svelte:13`) |
| setup `id` | string | 아니오 | setup 디렉터리 이름 | `manifest_setup.go:126` | 예 (`ManifestManager.svelte:42`) |
| setup `manifestId` / `binary.*` | | 아니오 | | | 아니오 |
| setup `state` | object(`chainsetup.State`) | 아니오 | | `chain status` | `state.chain`만 (`ManifestManager.svelte:42`) |

- **문제**: validate의 `errors`만 문자열 배열이다. 다른 모든 검증 응답은 `{path,code,message}` 객체 배열이다. 저장(`POST /manifests`)은 같은 엔진 오류를 422 평문으로 돌려준다. 그래서 같은 검증 실패가 검증 버튼에서는 200 `valid:false`로, 저장 버튼에서는 422로 온다. 목록은 쪽을 나누지 않고, 호출마다 모든 외부 manifest를 다시 검증한다(`manifests.go:137-158`). 외부 manifest 하나가 깨지면 목록 전체가 실패한다. "체인"을 `manifest.protocol || manifest.id`로 정하는 규칙을 화면이 여러 곳에서 되풀이한다(`ManifestManager.svelte:30`, `JobPanel.svelte:38-39`, `61`). setup의 노드 4개는 Go 코드 상수다(`manifest_setup.go:152`).

### `GET /api/v1/manifest-assets`

- **응답**: `{items: [{id, chain, sha256, commit}]}`, 쪽 없음 (`internal/dashboard/manifests.go:70-92`).
- **원본 데이터**: 서버 플래그 `-manifest-assets` JSON 파일에서 읽은 바이너리(`cmd/chainbench-dashboard/main.go:139-149`)와, 업로드한 자산 중 `binary` 영수증.
- **갱신 방식**: `ManifestManager`, `ChainSurfacePanel`(`ChainSurfacePanel.svelte:21`), `JobPanel`(`JobPanel.svelte:78`) 마운트.
- **쓰는 화면**: 위 세 파일.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | | | 예 (`ManifestManager.svelte:30`, `ChainSurfacePanel.svelte:27`) |
| `items[].chain` | string | 아니오 | | | 예 (`ChainSurfacePanel.svelte:21`, `JobPanel.svelte:61`) |
| `items[].sha256` | string | 아니오 | 64자 hex | | 예 (`ManifestManager.svelte:30`, `JobPanel.svelte:60`) |
| `items[].commit` | string | 예 | | | 아니오 |

- **문제**: 같은 업로드 바이너리가 `/assets`에서는 `checksum`, 여기서는 `sha256`이라는 다른 이름으로 나온다. 서버 플래그로 넣은 바이너리는 `/assets`에 나오지 않는다. 같은 개념인 "등록 바이너리"가 두 목록으로 갈린다.

### `GET /api/v1/assets` · `GET /api/v1/assets/{id}` · `POST /api/v1/assets` (multipart)

- **요청(POST)**: multipart, 값 `kind` 하나와 파일 `file` 하나 (`internal/dashboard/manifests.go:105-134`). **응답**: `app.WebAsset` (`internal/app/web_assets.go:25-33`), 목록은 `{items, nextCursor}`.
- **원본 데이터**: `<deployment-root>/assets/<id>/{payload, receipt.json}`(`internal/core/session/assetstore.go:24-179`). 바이너리는 업로드할 때 실제로 실행해 `version`과 help를 읽고 맞는 체인을 하나만 찾는다(`web_assets.go:66-108`). 그 밖의 파일은 형식과 비밀 키 이름을 검사한다(`web_assets.go:132-196`). 목록은 읽을 때마다 모든 영수증의 체크섬을 다시 계산한다(`web_assets.go:198-232`).
- **갱신 방식**: `AssetManager` 마운트와 업로드 뒤, 새로고침 버튼(`web/src/AssetManager.svelte:9-12`, `26`). `ChainPresetEditor`와 `DSLEditor`는 `assetRevision`이 바뀔 때(`ChainPresetEditor.svelte:49`, `DSLEditor.svelte:52`).
- **쓰는 화면**: `AssetManager.svelte`, `ChainPresetEditor.svelte`, `DSLEditor.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | 32자 hex | `session.NewAssetID` | 예 (`AssetManager.svelte:39`) |
| `items[].kind` | string | 아니오 | `binary/configuration/template/material` | 입력 | 예 (`AssetManager.svelte:39`, `ChainPresetEditor.svelte:35`) |
| `items[].name` | string | 아니오 | 파일 이름 | 입력 | 예 (`AssetManager.svelte:39`) |
| `items[].checksum` | string | 아니오 | SHA-256 hex | 스테이징 | 예 (`AssetManager.svelte:39`, `ChainPresetEditor.svelte:190`) |
| `items[].bytes` | number | 아니오 | 바이트, 최대 256 MiB | 스테이징 | 예 (`AssetManager.svelte:39`, MiB로 환산) |
| `items[].compatibility` (binary) | object | 아니오 | `chain, version, commit, helpDigest, os, architecture, inspection` | `web_assets.go:102` | `chain, os, architecture`만 (`AssetManager.svelte:39`) |
| `items[].compatibility` (그 밖) | object | 아니오 | `format, inspection, executionReady:false` | `web_assets.go:112` | `format`만 (`AssetManager.svelte:39`, `ChainPresetEditor.svelte:35`) |
| `items[].uploaderId` | string | 아니오 | 사용자 id | | 아니오 |

- **문제**: `compatibility`는 `map[string]any`라 종류에 따라 키가 다르고 계약이 없다. 업로드 실패 이유는 영어 평문이고 POST라 그대로 보인다. 크기 한도(256 MiB, 16 MiB)와 허용 형식은 화면 문구에 한 번 더 적혀 있다(`AssetManager.svelte:33`, Go 쪽 `web_assets.go:20-21`, `150-171`). 자산을 지우는 엔드포인트가 없다. `AssetManager`만 `webApi`를 쓰지 않고 `fetch`를 직접 부른다(`AssetManager.svelte:10`, `17`).

### `GET /api/v1/auth/config`

- **응답**: `{bootstrapRequired: bool, accountsEnabled: bool}` (`internal/dashboard/security.go:211-213`). 로그인 없이 열린다(`security.go:81`).
- **원본 데이터**: `WebAuth` 메모리 상태(사용자가 없고 setup 토큰이 있는지, 계정 모드인지).
- **갱신 방식**: 앱 시작 때와 `/auth/me`가 401을 낼 때(`web/src/web-access.mjs:6`, `LoginPanel.svelte:16-21`).
- **쓰는 화면**: `web-access.mjs` → `LoginPanel.svelte`.

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `bootstrapRequired` | boolean | 아니오 | | `internal/app/web_auth.go:377` | 예 (`web-access.mjs:17`) |
| `accountsEnabled` | boolean | 아니오 | | `web_auth.go:128` | 예 (`web-access.mjs:10`) |

- **문제**: 404면 화면은 "계정 없음, 세션 없음"으로 본다(`web-access.mjs:7`). legacy 관측 모드를 뜻하지만 응답이 그 이유를 말하지 않는다.

### `POST /api/v1/auth/local` · `POST /api/v1/bootstrap` · `POST /api/v1/auth/login` · `GET /api/v1/auth/me` · `POST /api/v1/auth/logout`

- **요청**: bootstrap `{username, password, setupToken}`, login `{username, password}` (`security.go:165-195`). **응답**: 앞의 넷은 `app.WebSession` (`web_auth.go:30-36`)과 `HttpOnly; SameSite=Strict` 쿠키 `chainbench_session`(`security.go:143-146`). logout은 204.
- **원본 데이터**: 사용자는 `<deployment-root>/users.json`(bcrypt 해시, `internal/core/session/accountstore.go:20`), setup 토큰은 `<deployment-root>/setup.token`(`accountstore.go:50`). 세션은 메모리에만 있고 재시작하면 사라진다. 수명 12시간(`web_auth.go:50`). 계정 없는 모드의 사용자는 고정 `local-web-operator`, 역할 `administrator`(`web_auth.go:130-132`).
- **갱신 방식**: 시작 때 한 번, 로그인 클릭, 그리고 세션이 있는 동안 **2초마다** `/auth/me`(`LoginPanel.svelte:21`).
- **쓰는 화면**: `LoginPanel.svelte`, `web-access.mjs`. `csrfToken`은 모든 화면의 쓰기 요청이 쓴다(`web-api.mjs:9`).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `user.id` | string | 아니오 | 64자 hex 또는 `local-web-operator` | | 예 (`UserManager.svelte:21`) |
| `user.username` | string | 아니오 | `^[A-Za-z0-9_.-]{1,64}$` | `web_auth.go:64` | 예 (`LoginPanel.svelte:35`) |
| `user.role` | string | 아니오 | `viewer/operator/administrator` | | 예 (여러 곳) |
| `user.active` | boolean | 아니오 | | | 아니오 |
| `csrfToken` | string | 아니오 | 64자 hex | `web_auth.go:187` | 예 (`web-api.mjs:9`) |
| `expiresAt` | string | 아니오 | RFC 3339 UTC | | 아니오 |
| `mode` | string | 아니오 | `personal/team` | 서버 플래그 `-web-mode` | 예 (`LoginPanel.svelte:35`) |
| `accountsEnabled` | boolean | 아니오 | | | 예 (`SettingsPanel.svelte:17`) |

- **문제**: 로그인 실패는 401 `"invalid login"`이지만 401이라 보안 계층이 `"Unauthorized"`로 바꾼다. bootstrap의 토큰 불일치는 403, 사용자가 이미 있으면 409 `"revision changed; reload before editing"`이다(`web_auth.go:196-201`). 두 경우 모두 이유가 엉뚱하거나 사라진다. 화면은 만료 시각을 모르므로 2초 polling으로 401을 기다린다.

### `GET /api/v1/users` · `POST /api/v1/users` · `PATCH /api/v1/users/{userId}`

- **요청**: POST `{username, password, role}`, PATCH `{role?, password?, active?}` (`security.go:219-261`). **응답**: 목록 `{items: []app.WebUser, nextCursor}`, 단건 `app.WebUser` (`web_auth.go:20-25`).
- **원본 데이터**: `users.json`. 역할을 viewer로 낮추거나 끄면 그 사용자의 진행 중 작업을 먼저 철회한다(`security.go:248-259`). 마지막 활성 관리자를 없애는 변경은 409다(`web_auth.go:345-354`).
- **갱신 방식**: 설정 화면 마운트와 변경 직후(`UserManager.svelte:9-12`).
- **쓰는 화면**: `UserManager.svelte`(관리자이고 계정 모드일 때만, `SettingsPanel.svelte:16-17`).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | 64자 hex | `web_auth.go:182` | 예 (`UserManager.svelte:18`, `21`) |
| `items[].username` | string | 아니오 | | | 예 (`UserManager.svelte:19`) |
| `items[].role` | string | 아니오 | | | 예 (`UserManager.svelte:21`) |
| `items[].active` | boolean | 아니오 | | | 예 (`UserManager.svelte:20`, `22`) |

- **문제**: 사용자 이름 중복과 마지막 관리자 보호가 모두 `ErrDeploymentConflict`라 409 문구가 `"revision changed; reload before editing"`이다(`web_auth.go:175-179`, `352`). 화면 비밀번호 칸은 최소 12자만 막고(`UserManager.svelte:28`) 최대 72바이트와 사용자 이름 정규식은 서버만 안다(`web_auth.go:171`). 화면은 비밀번호 바꾸기를 제공하지 않는다(PATCH의 `password`를 쓰는 곳이 없다).

### `GET /api/v1/audit?from=&to=&limit=&cursor=`

- **응답 Go 타입**: `app.WebAuditPage{items: []WebAuditRecord, nextCursor: *string}` (`internal/app/web_audit.go:17-38`).
- **원본 데이터**: 계정 저장소, 배치 저장소(`deployment.json`의 `audit`), 작업, 히스토리, manifest(`manifest-audit.jsonl`)의 기록을 매번 모두 읽어 합친다(`internal/dashboard/audit.go:60-102`, `web_audit.go:45-90`).
- **갱신 방식**: 설정 화면 마운트, 기간 조회, 더 보기(`web/src/AuditPanel.svelte:8-13`).
- **쓰는 화면**: `AuditPanel.svelte`(관리자만).

| 필드(JSON 경로) | JSON 타입 | 비어 있을 수 있나 | 단위·형식 | 원본 | 화면이 읽나 |
|---|---|---|---|---|---|
| `items[].id` | string | 아니오 | | 저장소별 | 예 (`AuditPanel.svelte:29`, key) |
| `items[].actorId` | string | 아니오 | 사용자 id 또는 `unauthenticated` | | 예 (id 그대로) |
| `items[].operation` | string | 아니오 | 라우트 패턴 또는 `document.save` 등 | `security.go:63-66`, `deployment_store.go:231` | 예 |
| `items[].target` | string | 아니오 | `users/<id>` 같은 경로 | `audit.go:32-56` | 예 |
| `items[].requestId` | string | 예(omitempty) | 24자 hex | | 예 |
| `items[].jobId` | string | 예(omitempty) | | | 아니오 |
| `items[].result` | string | 아니오 | HTTP 상태 숫자 문자열 또는 저장소 결과 | | 예 (`AuditPanel.svelte:15`) |
| `items[].time` | string | 아니오 | RFC 3339 | | 예 |
| `nextCursor` | string \| null | 예 | 숫자 오프셋 | `web_audit.go:85-` | 예 |

- **문제**: `operation` 어휘가 저장소마다 다르다(HTTP 라우트 패턴, `document.save`, `credential.bind`, `asset.upload` 등). `result`도 HTTP 상태와 저장소 결과가 한 칸에 섞여 화면이 정규식으로 거절 여부를 추측한다(`AuditPanel.svelte:15`). 사용자 이름이 아니라 id만 온다. 다른 목록과 달리 커서가 오프셋이라 새 기록이 들어오면 다음 쪽이 밀린다.

---

### Web 입력에서 CLI 플래그로

**배경.** 작업 계획 요청 `POST /api/v1/plans`의 본문은 `WebPlanInput`이다(`internal/app/web_job_contract.go:12-24`). 그 안의 `arguments`는 `webChainArguments`로 엄격하게 읽는다. 모르는 필드가 있으면 `"select manifest, binary and server; arbitrary command arguments are unavailable"`로 거절한다(`internal/app/web_chain_engine.go:140-148`). 아래는 화면이 `arguments`를 어떻게 만드는지(`web/src/JobPanel.svelte:138`)와, 엔진이 그 값을 `chainbench` 하위 명령의 어떤 플래그로 바꾸는지다.

**화면이 만드는 `arguments`.** `JobPanel.preparePlan`은 작업 종류마다 다음을 보낸다(`JobPanel.svelte:138`).

- `network.monitor`: `{serverRef}` 또는 `{allServers:true}`. 서버 선택 상자의 특수 값 `'*all*'`을 `allServers`로 바꾼다.
- 그 밖 공통: `manifestId`, `assetId`, 그리고 `serverRef` 또는 `allServers`.
- `test.run`, `test.attach`: `runDefinitionRef` 또는 `caseRefs`, 그리고 기본 외 이름 바이너리가 있으면 `binaryAssets`.
- `node.swap`: `configAssetId` 또는 `configOverrides`(둘 중 하나), 그리고 `replacementAssetId`.
- 그 밖의 `node.*`: 추가 인자 없음. 노드는 `nodeIds:[nodeId]`로 따로 간다.
- `chain.setup`, `chain.deploy`: 공유 프리셋을 골랐으면 `chainPresetRef`, 아니면 `validators`.

**엔진의 기본값과 검사.** `validators`가 0이면 4로 채운다(`web_chain_engine.go:166-168`). 1~128 밖이면 거절한다(`web_chain_engine.go:169-171`). `chainPresetRef`는 setup, deploy에만, `validators`와 함께 쓸 수 없다(`web_chain_engine.go:163-165`). `allServers`와 `serverRef`는 함께 쓸 수 없고, test.run은 `allServers`를 받지 않는다(`internal/app/web_chain_servers.go:19-45`).

작업 종류별 대응은 아래와 같다. 같은 축(입력 → 플래그)을 나란히 비교하므로 표로 적는다.

| 작업 | 실행하는 명령 | `arguments` 필드 → CLI 플래그 | 엔진이 스스로 정하는 플래그 |
|---|---|---|---|
| `chain.setup`, `chain.deploy` | `chainbench chain up` (`internal/app/web_compose_cli.go:141-180`) | `allServers` → `--all-servers`, `serverRef` → `--server <이름>` (`web_compose_cli.go:143-148`). `manifestId`가 외부면 `--manifest <고정 파일> --genesis-template <고정 파일>`, 내장이면 `--chain <id>` (`149-157`). `chainPresetRef` → 그 revision 내용을 `<control>/chain-preset.json`에 써서 `--chain-preset` (`158-163`). 없으면 `validators` → `--bp N` (`164-166`). `assetId` → 검증한 바이너리 경로(SSH면 대상 경로)를 `--binary` (`web_chain_engine.go:270-276`) | `--workspace-dir <deployment-root>/networks/<workspace id>`, `--stage init`(setup) 또는 `start`(deploy) (`internal/app/web_chain_execution.go:204-208`), `--keys <승인한 키 스냅숏>`, `--server-set <control>/server-set.yaml`, `--workspace-config <control>/workspace-config.yaml`, `--server-logins <임시 비밀 디렉터리>/server-set.yaml` |
| 프리셋 계획(setup·deploy의 계획 단계) | `chainbench chain up --chain-preset <파일> --plan --json` (`internal/app/web_plan_cli.go:44-74`) | 위와 같은 값 중 `--chain`, `--binary`, `--keys`, `--workspace-config`, `--server-set`, `--server` | `--workspace-dir <임시>` |
| `node.start` | `chainbench node start --index N` (`web_chain_execution.go:123`) | `nodeIds[0]` → 기록에서 찾은 노드 번호 `--index` | `--workspace-dir`, `--server-logins` (`web_compose_cli.go:302-312`) |
| `node.stop` | `chainbench node stop --index N` (`web_chain_execution.go:185`) | 같음 | 같음 |
| `node.restart` | `chainbench chain restart --node N` (`web_chain_execution.go:170`) | 같음 | 같음 |
| `node.reset` | `chainbench node reset --index N` (`web_chain_execution.go:178`) | 같음 | 같음 |
| `node.swap` | `chainbench node swap --index N --purpose web-reviewed-config [--binary P] [--config k=v ...]` (`web_compose_cli.go:315-324`) | `replacementAssetId` → 스테이징한 경로를 `--binary`. `configOverrides`(문자열 맵) 또는 `configAssetId`의 내용 → 기록된 launch와 맞춘 `--config` 목록 (`internal/app/web_config_change.go:17-`) | 같음 |
| `test.run` | 계획 `chainbench run <cases...> --json --suite ... --plan`, 실행은 같은 명령에서 `--plan` 없이 (`internal/app/web_test_run_cli.go:32-75`, `web_plan_cli.go:17-42`) | `caseRefs` 또는 `runDefinitionRef` → 고정 revision 내용을 `cases/001.json`…로 써서 위치 인자로. `binaryAssets` → `--binary-for name=경로` (`web_test_run_cli.go:60-67`). `serverRef` → `--server`. `assetId` → `--binary`와 `--binary-for default=경로` | `--workspace-dir`, `--chain`(바이너리 증거의 체인), `--keys`, `--artifact-root <deployment-root>/sessions`, `--workspace-config`, `--server-set`, `--server-logins`, `--read-only-keys`, `--keep-up` (`internal/app/web_test_run.go:234-244`) |
| `test.attach` | `chainbench run <cases...> --json --attach --workspace-dir <control> --no-node-control --read-only-keys --artifact-root <...> [--server-logins]` (`web_test_run_cli.go:81-94`) | `caseRefs`/`runDefinitionRef` → 위치 인자. `accountBindings`(`arguments` 밖) → account-key 비밀을 임시 파일로 써서 케이스의 `accounts.<label>.keyFile`을 그 경로로 바꿈 (`internal/app/web_test_attach.go:79-110`) | 위 고정 플래그 |
| `network.monitor` | 하위 명령을 띄우지 않고 노드마다 `chainbench resource inspect --json --request <파일> [--server-logins]`로 로그를 읽음 (`internal/app/web_target_cli.go:133-137`, `281-295`) | `serverRef`/`allServers` → 어느 서버의 노드를 읽을지 (`internal/app/web_network_monitor.go:31`, `200-219`) | 없음 |
| 자격증명 확인(작업 아님) | `chainbench resource access --json --server-set <logins> --server-logins <logins> --server <이름> --data-root <dataRoot>` (`deployment_access.go:145`) | 요청 `serverRef` → `--server` | `--data-root`는 workspace-config에서 |
| manifest setup(작업 아님) | `chainbench chain up --workspace-dir <setup> --stage init --bp 4 --keys <-manifest-keys> --binary <경로> (--chain id \| --manifest --genesis-template)` (`manifest_setup.go:151-158`) | 요청 `assetId` → `--binary` | `--bp 4` 고정 |

화면에서 오지 않는 `WebPlanInput` 필드 가운데 `retention`은 CLI 플래그가 아니다. `cleanup`이면 작업이 끝난 뒤 엔진이 `chainbench chain stop`, `chain rm`을 부른다(`web_chain_engine.go:819-822`, `internal/app/web_job_execution.go:85`). test.run은 늘 `--keep-up`으로 돌린 뒤 이 정리로 지운다.

### `chain up` 플래그 가운데 Web이 표현하지 못하는 것

**배경.** `chain up`의 플래그는 `cmd/chainbench/chaincmd/up.go:116-147`에 있고, 공통 서버 플래그(`cmd/chainbench/resourcecmd/serverflags.go:22-28`)와 대상 플래그(`cmd/chainbench/chaincmd/chain.go:65-72`)가 붙는다. Web이 직접 넘기는 것은 위 표의 `--workspace-dir`, `--stage`, `--keys`, `--binary`, `--server-set`, `--workspace-config`, `--server`, `--all-servers`, `--chain`, `--manifest`, `--genesis-template`, `--chain-preset`, `--bp`, `--server-logins`, `--plan`, `--json`이다. 나머지를 프리셋 계약(DSL v2 `envSpec`, `internal/dsl/schema/v2.schema.json`)이 같은 뜻으로 표현할 수 있는지, 그리고 Web의 프리셋 합성이 그 필드를 받아들이는지를 따로 봤다. Web 합성은 `blueprint`, `manifest`, `genesisTemplate`, `upgrade`, `accounts`, `attach`, 바이너리별 genesis, 바이너리 둘 이상, 기본 외 키 소스, 경로·포트 launch 키를 거절한다(`internal/app/web_preset_composition.go:78-121`). `chain up --chain-preset`은 프리셋이 답하는 플래그를 함께 주면 거절한다(`up.go:153-154`, `167-171`).

| `chain up` 플래그 | 뜻 | 프리셋 계약으로 표현되나 | Web 합성이 받아들이나 |
|---|---|---|---|
| `--en`, `--pn` | 엔드포인트, 프록시 수 | 된다. `topology.en`, `topology.pn` | 된다. 화면 입력은 ChainPresetEditor의 개수 칸뿐이고, 작업 화면에서 프리셋 없이 고를 때는 bp만 있다(`JobPanel.svelte:156`) |
| `--endpoint-syncmode` | 엔드포인트 동기화 방식 | 된다. `topology.syncMode` 또는 노드 표의 `sync` | 된다 |
| `--topology` | 노드별 배치 YAML | 된다. `topology.nodes[]` | 된다. 단 노드별 `binary`, `key`, `config` 파일은 등록 자산이 아니면 거절(`web_preset_composition.go:200`) |
| `--binaries name=path` | 이름 바이너리 경로 | 된다. `binaries` | 거절. 기본 바이너리 하나만 허용(`web_preset_composition.go:102-109`). test.run은 `binaryAssets`로 따로 표현 |
| `--blueprint` | 배치와 키를 한 문서로 | 된다. `blueprint` | 거절(`web_preset_composition.go:78`) |
| `--peering` | `mesh`/`proxied` | 된다. `peering`(그룹 선언). pn이 있으면 엔진이 proxied로 정함(`internal/testengine/compose.go:375-382`) | 거절 목록에 없으므로 받아들이는 것으로 읽힌다. 실행해 확인하지는 않았다 |
| `--keys-source` | `preset/generate/declared` | 된다. `keys.nodekeys.source` | 거절. `keyPreset`과 `presets/keys`만 허용(`web_preset_composition.go:110-115`) |
| `--chain-id` | manifest 체인 id 덮기 | 같은 이름의 필드가 없다. 엔진은 이 값을 실행 입력(`in.ChainID`)에서만 받는다(`compose.go:361`). `genesis.set`으로 체인 id 키를 덮을 수 있는지는 확인하지 못했다 | 표현할 길 없음 |
| `--set k=v` | genesis 설정 키 덮기 | 된다. `genesis.set`, `hardforks` | 받아들임(거절 목록에 없음) |
| `--overlay` | genesis 덮어쓰기 JSON 파일 | 된다. `genesis.overlay`(파일 대신 객체) | 받아들임 |
| `--genesis-existing` | 완성 genesis 사용 | 된다. `genesis.mode:"existing"` + `genesis.ref` | `asset:<id>`인 JSON template 자산만(`web_preset_composition.go:85-99`) |
| `--launch-opt` | launch 옵션 | 된다. `launch.all` 등 범위별 | 받아들임. 경로·포트 키는 거절(`web_preset_composition.go:116-121`) |
| `--docker` | 서버가 로컬 docker 컨테이너 | 안 된다. 프리셋에 해당 필드가 없다 | 표현할 길 없음 |
| `--target`, `--remote-host`, `--remote-user`, `--remote-port`, `--target-dir` | 서버 세트 없이 대상 지정 | 된다. `target` 문자열 | Web은 늘 서버 세트와 workspace-config로 대상을 정한다. 프리셋의 `target`을 Web이 거절하는지는 확인하지 못했다 |
| `--server-index` | 번호로 서버 선택 | 안 된다 | 이름(`serverRef`)으로만 고른다. 같은 일을 이름으로 할 수 있다 |
| `--stage deploy` | 산출물만 쓰고 init 전 멈춤 | 해당 없음(실행 입력) | Web은 `init`과 `start`만 쓴다(`web_chain_execution.go:204-208`) |
| `--bootnode` | 폐기됨 | 해당 없음 | 해당 없음 |

### `run`(suite) 플래그 가운데 Web이 표현하지 못하는 것

**배경.** Web이 부르는 명령은 최상위 `chainbench run`이고 `suitecmd.NewRun()`이 만든다(`cmd/chainbench/root.go:44`). 플래그는 `cmd/chainbench/suitecmd/run.go:189-227`에 있다. Web이 넘기는 것은 `--json`, `--suite`, `--workspace-dir`, `--chain`, `--binary`, `--binary-for`, `--keys`, `--artifact-root`, `--workspace-config`, `--server-set`, `--server`, `--server-logins`, `--read-only-keys`, `--keep-up`, `--plan`, 그리고 연결 작업의 `--attach`, `--no-node-control`이다.

| `run` 플래그 | 뜻 | 프리셋 계약으로 표현되나 | Web에서는 |
|---|---|---|---|
| `--chain-preset` | 모든 케이스를 다른 프리셋으로 | 해당 없음(케이스가 프리셋을 이름으로 부르거나 품음) | 케이스가 고정한 프리셋 revision을 그대로 쓴다. 바꿔 끼울 수 없다 |
| `--chain-presets` (반복) | 같은 묶음을 여러 프리셋에서 | 해당 없음 | 표현할 길 없음 |
| `--wait-blocks` | 시작 전 높이 대기 | 안 된다. `genesis`의 멈춤 높이 설명은 있으나(`v2.schema.json:126`) 대기 높이 필드는 없다 | 표현할 길 없음 |
| `--node-monitor-timeout` | 준비 관문 대기 시간 | 안 된다 | 기본값만 쓴다 |
| `--rpc`, `--allow-chain-mismatch` | 주소로 살아 있는 네트워크에 붙기 | `attach.rpc`로는 된다 | Web은 자기가 배포·기록한 네트워크에만 붙는다(`--attach --workspace-dir`). 프리셋 목록은 `attach` 프리셋을 뺀다(`internal/app/chainpreset.go:48`) |
| `--keys-source generate` | 새 키 생성 | 된다. `keys.nodekeys.source:"generate"` | 거절(`web_preset_composition.go:110-115`). 늘 `--read-only-keys` |
| `--bp` | 생산자 수 덮기 | 된다. `topology.bp` | test.run은 케이스가 정한 배치만 쓴다(`web_chain_engine.go:302-304`) |
| `--chain-id` | genesis 체인 id 덮기 | `chain up`과 같다. 같은 이름 필드 없음 | 표현할 길 없음 |
| `--network-id` | devp2p 네트워크 id | 된다. `launch.all.networkid`(엔진도 이 값을 launch 옵션으로 바꾼다, `compose.go:347-350`) | 프리셋에 쓰면 된다 |
| `--launch-opt` | launch 옵션 | 된다. `launch` | 케이스·프리셋 안에 쓰면 된다 |
| `--docker` | docker 서버 | 안 된다 | 표현할 길 없음 |
| `--server-index`, `--all-servers` | 서버 선택 | 안 된다 | test.run은 서버 하나만(`web_chain_servers.go:30-32`) |
| `--dashboard` | 실행 이벤트를 대시보드로 보냄 | 해당 없음 | Web은 쓰지 않는다 |
| `--no-skips` | 건너뛴 케이스를 실패로 | 안 된다 | 표현할 길 없음. 건너뛴 케이스도 통과로 끝날 수 있다 |
| `--target`류(대상 플래그) | 이 명령에는 서버 플래그만 있다 | 해당 없음 | 해당 없음 |

### 하드코딩 기본값과 Go·JS 사이 중복 어휘

**역할.** 서버는 `contracts/access`로 역할 목록을 주지만, 화면 곳곳이 문자열을 직접 비교한다. `'administrator'`와 `'operator'`를 `AssetManager.svelte:6`, `ManifestManager.svelte:7`, `DeploymentEditor.svelte:17`, `SettingsPanel.svelte:16`이, `'viewer'`를 `ChainPresetEditor.svelte:11`, `ChainSurfacePanel.svelte:11`, `DSLEditor.svelte:29`, `80`, `JobPanel.svelte:28`, `SettingsPanel.svelte:15`, `UserManager.svelte:6`(새 사용자 기본 역할)이 비교한다. 서버 안에서도 `canEdit`(`deployment_store.go:32-34`), `ReadWebAudit`(`web_audit.go:46`), `CreateUser`(`web_auth.go:278`), `Save`(`manifests.go:86`), `SetupManifest`(`manifest_setup.go:109`), `POST /assets`(`internal/dashboard/manifests.go:106`)는 `admin`을, 보안 계층(`security.go:102-125`)과 계정 저장소(`web_auth.go:345-350`)는 `administrator`를 비교한다.

**자격증명 종류.** 라벨은 계약에서 오지만 분기 로직은 `'account-key'`, `'private-key'`, `'password'` 리터럴을 쓴다(`CredentialManager.svelte:7`, `15-16`, `57-63`, `DeploymentEditor.svelte:104`, `JobPanel.svelte:47`). 기본 선택도 `'private-key'`다(`CredentialManager.svelte:7`). 서버는 `validCredentialShape`에서 같은 문자열로 분기한다(`deployment_access.go:23-30`).

**문서 종류.** `server-set`, `workspace-config`, `chain-preset`, `case`, `run-definition`이 화면 리터럴이다(`DeploymentEditor.svelte:29`, `33`, `BundleImport.svelte:29-30`, `JobPanel.svelte:38-42`, `51`, `ChainPresetEditor.svelte:48`, `DSLEditor.svelte:28`). 서버에는 이 목록을 주는 계약이 없고 `validateDeploymentContent`의 `switch`가 사실상의 목록이다(`internal/app/deployment.go:103-167`).

**자산 종류.** `AssetManager.svelte:8`의 `names`가 `binary/configuration/template/material`을 들고, 서버는 `UploadAsset`의 `switch`가 같은 목록이다(`web_assets.go:47-53`). 크기 한도도 두 곳이다(`AssetManager.svelte:33`, `web_assets.go:20-21`).

**작업과 노드 어휘.** 작업 목록은 `contracts/jobs`에서 오지만 화면은 `'chain.setup'`(기본값), `'test.run'`, `'test.attach'`, `'node.swap'`, `'network.monitor'`, `'node.reset'`, `operation.startsWith('node.')`를 직접 쓴다(`JobPanel.svelte:14`, `31-36`, `55`). node.reset 대상 역할 `['en','pn']`이 화면에 박혀 있다(`JobPanel.svelte:55`). 활성 작업 상태 `['accepted','running','cancelling']`도 화면 리터럴이다(`JobPanel.svelte:57`). 관측 상태 이름표 `observationNames`가 화면에 있다(`JobPanel.svelte:17`). chain up 단계 이름 `new, place, keys, genesis, config, build, deploy, init, start`는 Go가 CLI 기록을 읽으려고 따로 들고 있다(`web_chain_execution.go:204-207`).

**노드 수와 역할.** 생산자 기본 4가 네 곳에 있다. CLI `--bp` 기본(`up.go:124`, `run.go:216`), 엔진 `validators` 기본(`web_chain_engine.go:166-168`), 프리셋 계약 `topology.bp.default`(`internal/app/chainpreset.go:85`), 화면 `validators=$state(4)`(`JobPanel.svelte:14`). 범위 1~128은 엔진(`web_chain_engine.go:169-171`)과 화면 입력(`JobPanel.svelte:156`)에 둘 다 있다. manifest setup은 `--bp 4`를 따로 고정한다(`manifest_setup.go:152`), 화면 문구도 "노드 네 개"라고 적는다(`ManifestManager.svelte:21`). 역할 `bp/en/pn`은 프리셋 계약 안에서 세 번 쓰인다(`chainpreset.go:84-102`). 화면 기본 노드 `'node1'`(`JobPanel.svelte:14`).

**기본 문서.** 새 서버 풀은 `{version:2, pool:{hosts:[{name:'local', addr:'127.0.0.1'}], slots:1, ports:{p2p:{base:31000, step:10}, rpc:{base:8600, step:10}}}}`, 새 대상 경로는 `{version:1, dataRoot:'/data/chainbench', paths:{binaries…logs}, control:{artifactRoot:'chainbench-out'}, inputs:{mode:'generated'}, execution:{chain:'fresh'}}`로 화면이 짓는다(`DeploymentEditor.svelte:40`). 버전 2와 1은 `resource.SupportedVersion`(`internal/resource/serverset.go:232`), `SupportedWorkspaceVersion`(`internal/resource/workspaceconfig.go:271`)과 같고, 목적 디렉터리 여덟 개는 `Purpose*` 상수(`workspaceconfig.go:205-212`)와 같다. 계약에 enum으로 실려 오는 값을 화면이 다시 적는 셈이다. 새 테스트는 `{schemaVersion:"2", kind:"case", id:"new-test", chainPreset:{chain:"stablenet", binaries:{default:"gstable"}, topology:{bp:4}}, steps:[{expect:"blockNumber", is:1}]}`로 짓는다(`DSLEditor.svelte:81`). 체인 이름과 바이너리 이름은 레지스트리 manifest 값(`internal/core/registry/manifest.go:19-22`)을 화면이 복사한 것이다. 화면 이름 기본값 `'서버 풀'`, `'팀 workspace'`, `'가져온 workspace'`, `'SSH 로그인'`도 화면에 있다.

**프리셋 키 경로.** Web 합성은 키 참조로 `presets/keys`만 허용한다(`web_preset_composition.go:112`). 서버 플래그 `-manifest-keys` 기본값도 `presets/keys`다(`cmd/chainbench-dashboard/main.go:36`). 두 값이 독립된 문자열이다.

**표면 상태.** `verified/incompatible/stale/documented`와 `supported/engine-unmapped/binary-absent/unverified`는 Go 리터럴이고(`web_chain_surface.go:174-356`), 화면은 `status-icon.mjs:3-10`에 일부만 옮겨 적었다.

### 비밀 경계

**쓰기 전용으로 받는 값.** 자격증명의 `password`, `privateKey`, `passphrase`, `accountKey`는 POST로만 받고, 어떤 응답도 돌려주지 않는다(`deployment_store.go:64-72`, `385-433`). 화면은 저장 시도 뒤 성공 여부와 관계없이 입력 칸을 비운다(`CredentialManager.svelte:13`, `17`). 사용자 비밀번호와 bootstrap `setupToken`도 쓰기 전용이고, 화면은 요청 뒤 칸을 비운다(`LoginPanel.svelte:22`, `UserManager.svelte:11`).

**암호화해 두는 값.** 자격증명 비밀은 AES-256-GCM으로, 추가 인증 데이터 `id:ownerId`와 함께 `deployment.json`에 둔다(`deployment_store.go:398-407`). 가져오기 원문은 미리보기 id와 사용자 id를 추가 인증 데이터로 암호화해 둔다(`document_import.go:124-129`). 키 파일은 `credential.key`이고 권한이 0600이 아니면 서버가 열지 않는다(`internal/core/session/webstore.go:53-55`). 폐기한 자격증명도 암호문은 남는다. 응답 가림(`RedactWeb`)이 그 값을 알아야 하기 때문이다(`deployment_store.go:235-255`).

**계약에서 지운 필드.** 서버 세트 계약에서 `dataRoot`, `ssh.password`, `ssh.password_file`, `ssh.key_file`, `ssh.key_passphrase_file`, `pool.hosts[].ssh`를 지운다(`deployment_contract.go:26-45`).

**저장을 거절하는 값.** 서버 세트의 `ssh` 블록에 위 네 필드가 있으면 거절한다(`internal/app/deployment.go:110-116`). 서버 호스트 객체에 `name`, `addr` 말고 다른 키가 있으면 거절한다(`deployment.go:131-140`). 프리셋과 케이스에서 노드 키 참조(`topology.nodes[].key`)나 서명 키 참조(`key`, `feePayerKey`, `senderKey`, `authorityKey`가 가리키는 `values`)에 32자 이상 hex가 있으면 거절한다(`internal/app/web_document_secrets.go:52-125`). 자산 파일에 `PRIVATE KEY` 문자열이 있거나 `password`, `privateKey`, `mnemonic`, `nodekey`, `blsSecretKey` 같은 키가 있으면 거절한다(`web_assets.go:132-196`).

**가져오기에서 떼어 내는 값.** 서버 세트 `ssh`의 비밀 네 필드는 지우고 `changes`, `privateBindingsRequired`로 알린다(`document_bundle.go:175-189`).

**실행 때만 꺼내는 값.** 작업과 접속 확인은 바인딩한 자격증명을 `<deployment-root>/job-logins/job-*/secrets/host<N>-{password,key,passphrase}`(0600)로 쓰고, 그 경로를 담은 서버 세트를 `--server-logins`로 넘긴 뒤 명령이 끝나면 디렉터리를 지운다(`web_compose_cli.go:187-204`, `internal/app/deployment_job_serverset.go:20-123`). account-key는 연결 작업 때 임시 파일로 쓴다(`web_test_attach.go:79-91`). 자격증명이나 계정이 철회되면 실행 중인 CLI를 100ms 간격 확인으로 끊는다(`web_target_cli.go:151-182`).

**응답에서 가리는 값.** 모든 성공 응답과 400·409·422 이유에서 `password` 류 키의 값을 가리고, 등록된 모든 사용자의 SSH 비밀 문자열(8자 이상이면 부분 일치도)을 지운다(`web_redaction.go:16-80`). 이때 넣는 비밀은 `Password`, `PrivateKey`, `Passphrase`와 격리된 노드 키뿐이고, **`AccountKey`는 문자열 목록에 들어가지 않는다**(`web_redaction.go:33`). #466에서 고쳤다. 이제 계정 키를 `0x`가 있는 형태와 없는 형태로 넣고, `accountKey`, `account_key` 필드 이름도 가린다. 엔진 오류 문장에 계정 키 원문이 섞이는 경우가 있다면 지워지지 않는다. 그런 경로가 실제로 있는지는 확인하지 못했다. `contracts/chain-preset`은 dialect 스키마의 `password` 속성이 파일 경로 옵션이라 키 치환을 끈다(`security.go:54-59`).

**접근 범위.** 자격증명과 바인딩은 관리자에게도 본인 것만 보인다(`deployment_store.go:411-422`, `456-465`). viewer는 `/api/v1/credentials` 아래 전체가 403이다(`security.go:102-105`). 감사 기록은 경로 값 중 서버 id 문자 집합 밖의 것을 `redacted`로 적는다(`internal/dashboard/audit.go:22-56`).

### 있는데 화면에 오지 않는 데이터

- **문서 이력.** 서버는 문서마다 모든 revision을 `updatedAt`, `updatedBy`와 함께 들고 있다(`deployment_store.go:96-104`, `230`). 목록은 최신만, 단건은 번호로만 준다. revision 목록과 차이를 보여 줄 엔드포인트가 없어서, 공유 편집기의 "누가 언제 바꿨나"를 화면이 보일 수 없다. 목록에 이미 오는 `updatedAt`, `updatedBy`조차 화면은 읽지 않는다.
- **워크스페이스 이력.** 저장할 때 덮어써서 서버에도 지난 revision이 남지 않는다(`deployment_store.go:355-356`). 감사 기록 `workspace.save`만 남는다.
- **자격증명 생성 시각과 소유자.** `createdAt`, `ownerId`가 오지만 읽지 않는다. `activeJobs`는 개수만 쓰고 어떤 작업인지 링크하지 않는다(`CredentialManager.svelte:39`).
- **바이너리 증거.** `/assets`의 binary `compatibility.version`, `commit`, `helpDigest`, `inspection`과 `uploaderId`, 표면의 `binary.checksum`, `binary.verifiedAt`, manifest setup의 `binary` 증거가 오지만 읽지 않는다. 바이너리 상세 화면에 쓸 수 있다.
- **표면의 명령 목록과 근거.** `commands[]`(어느 엔진 use case가 어느 하위 명령을 부르는지), 옵션의 `aliases`, `emission`, `optionKey`, `evidenceRefs`가 오지만 읽지 않는다. ChainSurfacePanel이 "이 옵션을 프리셋 launch 어느 키로 쓰는지"를 보여 줄 재료다.
- **어휘의 선행 조건.** `vocabulary.entries[].prerequisites`가 있지만 화면은 개수만 쓴다. 문장 선택지 옆에 "이 체인 능력이 있어야 함"을 보일 수 있다.
- **manifest 계약.** `contracts/manifest`의 `families`, `dialects`, `protocols`를 아무도 읽지 않는다. 외부 manifest 가져오기 전에 선택지를 보여 줄 수 있다.
- **디스크 프리셋이 빠진 이유.** `ChainPresets`는 파싱 실패, 미등록 체인, 검증 실패 파일을 조용히 버린다(`internal/app/chainpreset.go:41-50`).
- **가져오기 원문 보존 여부.** `sourcePreserved`(늘 참)와, 서버가 15분 동안 원문을 암호화해 둔다는 사실은 ChainPresetEditor 문구에만 있고(`ChainPresetEditor.svelte:178`) 만료 시각은 오지 않는다(`document_import.go:141`).
- **오류의 요청 id.** 모든 오류 본문의 `requestId`와 `X-Request-ID` 헤더를 화면이 버린다(`api-error.mjs:4-13`). 감사 기록과 맞춰 볼 수 있는 유일한 열쇠다.
- **세션 만료 시각.** `expiresAt`가 오지만 화면은 2초 polling으로 만료를 알아챈다(`LoginPanel.svelte:21`).
- **감사 기록의 작업 id.** `jobId`가 오지만 읽지 않는다. 작업 상세로 이을 수 있다.

### 화면이 직접 계산하는 것

- `schemaProblems` (`web/src/schema-problems.mjs:5-28`): 배치 계약으로 필수·enum·범위를 미리 검사한다. 서버 검증은 위치를 모르므로 이 계산이 화면에서 유일하게 필드를 짚는다. 서버가 `resource` 검증 결과에 경로를 실어 주게 되면 읽기 모델로 옮길 후보다. 지금 `minLength`, `pattern` 분기는 배치 계약에 그런 키가 없어 죽은 코드다.
- `DeploymentFields.defaultValue` (`web/src/DeploymentFields.svelte:9-16`): 계약에 `default`가 없어서 타입으로 기본값을 지어낸다. 계약이 `default`를 싣는 쪽으로 옮길 후보다.
- `draftMeta`, `contractChanged`, `changedDefinitions` (`web/src/contract-draft.mjs:4-24`): 409 문구를 정규식으로 알아보고 두 계약을 JSON 문자열로 비교해 바뀐 정의를 고른다. 서버가 409 본문 `details`에 "어떤 계약의 어떤 정의가 바뀌었나"를 실으면 화면 계산이 필요 없다. 문구 정규식 의존은 먼저 없애야 할 위험이다.
- `topologyChoices`, `draftChain` (`web/src/chain-contract.mjs:4-26`): 프리셋 계약의 `oneOf`에서 역할·개수·동기화 선택지를 뽑는다. 계약에서 읽으니 중복은 아니지만 스키마 구조(`oneOf` 순서, `nodes` 유무)에 기대므로, 서버가 `topology` 선택지를 평평한 목록으로 주면 단단해진다.
- `caseChain`, `caseAttaches`, `keyFileAccounts`, `declaredBinaries` (`web/src/dsl-form.js:143-187`): 케이스와 고정 프리셋을 합쳐 체인, 연결 여부, 키 파일 계정, 이름 바이너리를 계산한다. 프리셋을 `documents.find(d => d.id === ref.id)`로 찾아 revision을 무시하므로 최신 프리셋과 고정 revision이 다르면 틀린다. 서버가 이미 `CasePresets`(`internal/app/deployment_presets.go:35`)로 같은 해석을 하므로 문서 목록 응답에 "해석된 체인·바이너리·계정"을 싣는 읽기 모델로 옮길 강한 후보다.
- `caseAssetRefs`, `finishedGenesisRef` (`dsl-form.js:119-126`): 선언에서 `asset:<id>`를 뽑아 `assetRefs`를 만든다. 서버 `webDocumentAssetRefs`(`internal/app/web_document_assets.go:26-89`)와 같은 계산이라, 두 구현이 어긋나면 저장이 "assetRefs must match" 422로 거절된다. 서버가 저장 때 직접 채우면 화면 계산을 없앨 수 있다. ChainPresetEditor도 `genesis.ref`에서 같은 계산을 한다(`ChainPresetEditor.svelte:34-36`).
- `presetIds`, `sameDeclaration`, `pinPresets` (`dsl-form.js:129-141`, `DSLEditor.svelte:33-43`): 케이스가 이름으로 부르는 프리셋을 공유 문서에서 찾고, 키 순서를 무시한 비교로 같은 내용을 고르며, 없으면 새로 등록한다. 서버의 저장이 `presetRefs`를 스스로 고정하는 쪽으로 옮길 후보다.
- 체인 결정 `manifest.protocol || manifest.id` (`ManifestManager.svelte:30`, `JobPanel.svelte:38-39`, `61`): manifest 응답에 해석된 체인 필드를 실으면 된다.
- 서버 세트 문서 찾기 (`DeploymentEditor.svelte:49`, `67`, `JobPanel.svelte:51-52`): 워크스페이스의 두 문서를 읽어 `kind`로 가르고 `pool.hosts`에서 서버 이름을 `h.name || h.addr`로 뽑는다. 엔진의 `resource.Set.ByName`과 같은 규칙인지는 확인하지 못했다. 워크스페이스 응답이 서버 이름 목록을 실어 주면 된다.
- 표면 옵션 집계 (`ChainSurfacePanel.svelte:12-14`): 상태별 개수와 필터. 화면 몫으로 두어도 된다.
- 자산 크기 MiB 환산과 체크섬 줄임 (`AssetManager.svelte:39`): 표시용이라 화면 몫으로 둔다.
- 감사 결과 거절 판정 (`AuditPanel.svelte:15`): `result`가 HTTP 상태인지 저장소 결과인지 정규식으로 짐작한다. 서버가 `outcome`(허용/거절)을 따로 실으면 없앨 수 있다.
- `declarationDiff` (`web/src/dsl-tree.mjs:34-48`): v1 이전 전후 차이를 화면이 계산한다. 서버가 `TestCasePrepared`에 `original`을 이미 실어 주므로(`internal/app/testcase.go:165`) 차이도 서버가 줄 수 있다.
