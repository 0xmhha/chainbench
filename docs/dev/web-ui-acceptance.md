# Web UI 인수 검증 계약

> **[현행 설계]** — 확정된 WEB-01–14의 성공 증거를 구체화한다. 이 문서는 검증 runner의
> 구현 계약이며 현재 제품이나 runner가 완성됐다는 보고가 아니다.

[동작 명세](web-ui-spec.md), [OpenAPI](web-ui.openapi.yaml),
[Seed](architecture/chainbench-web-ui.seed.yaml)의 같은 WEB ID를 사용한다.
구현은 먼저 `tests/webui/verify.sh`와 전용 fixture 준비·해제 도구를 만들어야 한다.
Seed의 `verify_command`는 프로젝트 루트에서 아래 인터페이스를 호출한다.

```sh
bash tests/webui/verify.sh --criterion WEB-01 --require-live --output chainbench-out/web-ui-acceptance
```

`--criterion`은 WEB-01부터 WEB-14 중 하나다. 해당 기준에 필요한 모든 시나리오를
수행한다. 증거 생성은 같은 명령에 `--capture`를 추가하여 각 기준 디렉터리의
`result.json`, `evidence.json`을 새로 만든다. 기본 명령은 소스/SPA fingerprint가 일치하는
동결된 증거를 검사하고, 별도 소스 snapshot에서 새 invocation으로 전체 live 시나리오를
독립 재실행한다. 기존 증거와 소스는 재검증 중 바뀌면 안 된다. Capture가 없거나 stale이면
기본 명령은 실패하며, 기존 결과를 읽기만 하고 live PASS를 출력하지 않는다.

fixture/runtime 기본 루트는 `/private/tmp/chainbench-web-ui-e2e`이며, 격리 검증 환경은
허용된 scratch 경로를 `WEBUI_RUNTIME_ROOT`로 지정한다. 결과는 workspace 상대
`chainbench-out/web-ui-acceptance/WEB-NN/`에 둔다. 로컬과 전용 SSH 환경은 runner가
설치하고 실제 연결 정보를 fixture manifest에 기록한다. 기존 외부 운영 서버를 사용하지 않는다.

성공은 exit 0과 `WEB-NN PASS` 문자열, 검증된 두 결과 파일을 함께 요구한다. 필수 환경·
바이너리·브라우저·assertion·coverage 항목이 없거나 skip이면 exit nonzero와 incomplete다.
일부 유닛 테스트나 mock만 실행하고 live PASS를 출력해서는 안 된다. 현재 runner가 없는
상태도 미완료다. 아티팩트가 존재한다는 사실만으로 완료 판정하지 않는다.

## 결과와 증거 형식

`result.json`은 criterion, invocationId, startedAt/finishedAt, outcome(pass/fail/incomplete),
requiredScenarios, executedScenarios, skippedScenarios, failedAssertions, evidenceDigest를 포함한다.
outcome=pass는 required 전체 실행·skip 0·실패 0·증거 검증을 모두 충족할 때만 가능하다.

`evidence.json`은 같은 criterion/invocationId와 각 시나리오의 다음 자료를 연결한다.

- mode(personal/team), transport(local/SSH), ownership(owned/attached), 사용자 역할과 관측 시간.
- 수행한 명령·브라우저 동작·API 요청/응답, 검증한 기대 상태와 실제 상태 및 assertion 결과.
- 최종 소스/SPA digest, 서버/프론트 build ID·기준 커밋·계약 버전, 바이너리 checksum·체인 정체성·버전/커밋·OS/아키텍처.
- fixture manifest와 실제 target fingerprint, 관련 job/plan/run/session 식별자.
- 이번 실행의 로그·캡처·스크린샷·coverage·노드 상태 자료의 상대 경로와 digest.

비밀정보는 증거에 저장하지 않는다. 이전 실행 자료를 현재 증거로 재사용하지 않는다.
필수 시나리오 집합은 아래 표와 DSL/체인 coverage에서 유도하고 구현자가 실제 지원 항목을
제외하여 denominator를 줄일 수 없다. 별도의 검증기가 invocation·digest·필수 집합과
관측 assertion을 검사해야 하며 runner의 출력 문자열만 신뢰하지 않는다.

## WEB 기준별 필수 관측

| 기준 | 반드시 실행할 시나리오와 성공 상태 | 추가 증거 |
|---|---|---|
| WEB-01 | 세 내장 체인의 프리셋/필수·선택 필드로 유효 구성 생성; 상속·기본/적용값 구분; 미지원 선택지는 제외하고 직접 입력도 거절 | 프리셋·옵션·필드 coverage, 엔진 검증 결과, 폼 캡처 |
| WEB-02 | 기존 구성 import→구조 편집→export→실제 셋업; 선언 의미 보존; 잘못된 필드와 unknown 확장 오류; revision 충돌 거절 | 변환 전후 선언·정규화 비교, 오류 경로, 생성 파일·셋업 결과 |
| WEB-03 | 두 운영자의 공유 server-set/workspace 편집과 개인 SSH binding; 포트·경로·배치 검증 및 접근 권한 차이 | 공유 revision, 비밀 제거 export, 각 사용자 접근 결과 |
| WEB-04 | 내장 매니페스트와 기존 family의 외부 매니페스트 검증·셋업; 미지원 family/dialect·잘못된 binary 정체성 거절 | registry/manifest coverage, 엔진 적용 결과와 부정 사례 |
| WEB-05 | 기준 파서/스키마의 모든 문법·내장 do/expect/reader·인자 구조 편집/round-trip/실행; 노드·변수 참조와 v1 migration; invalid/unknown 거절 | 등록→인자 schema→편집→의미 비교→실행 coverage의 누락 0 |
| WEB-06 | 전용 SSH 대상에 구성·호환 바이너리 배포와 체인 구축; 실제 PID/RPC·파일 checksum 확인; 중간 SSH 실패의 부분 효과 기록 | target/binary provenance, 배포 파일·상태, 단계·부분 실패 자료 |
| WEB-07 | 소유 노드 시작/정지·설정/바이너리 교체와 재시작·비생산 노드 reset의 실제 결과; reset 후 stopped; 생산/attach UI·API 거절 | 교체 전후 checksum/PID/RPC·datadir 상태, 접근/소유 판정 |
| WEB-08 | 실제 실행/노드 상태·metric 차트·시간 연계 로그; 브라우저 종료/로그아웃 뒤 계속 실행; 재접속 snapshot 복원; SSE gap/drop/stale/수집 실패 | 시간·단위·출처/coverage, cursor/버전, 차트·연계 로그 캡처 |
| WEB-09 | 검색/필터/상세·호환 비교·불가 이유·안전 export; 재시작 후 보관; 관리자 삭제와 비관리자 거절; active/shared/live 자료 보호·부분 삭제 실패 표시 | Run/session archive, 비교/보관/삭제 전후 소유 자료와 감사 기록 |
| WEB-10 | 개인/팀 로그인·bootstrap 제한·세 역할 UI와 직접 API; CSRF/소유 검사; 타 사용자 secret 조회/사용 거절; 로그/export/오류 비밀 누출 0 | 권한 행렬과 민감 값 탐지 결과, 암호화 저장/세션 관측 |
| WEB-11 | 다른 workspace 별칭의 동일 호스트/path/port 자원 요청 충돌; 현재 job/실행자 표시; 독립 자원 병행; 테스트 내부 제어와 외부 제어 충돌 | canonical footprint·동시 요청·잠금 및 진행 상태 |
| WEB-12 | 실제 작업 중 UI 서버 재시작; unfinished=interrupted와 상태 재확인; 기록 보존; 배포/reset 자동 재실행 0; 사용자 명시 재실행 | 재시작 전후 job/node·부작용 횟수·새 plan/job ID |
| WEB-13 | 정상 종료/일반 취소 × 기본 retain/선택 cleanup; 실행자/관리자 취소; cleanup 실패; 계정/credential 철회 중 retain 우선과 후속 credential 사용 차단; attach 보호 | 취소 사유·nodeDisposition·호출 시점·실제 노드/기록·부분 효과 |
| WEB-14 | 실제 Go 서버에서 새 SPA build·경로 이동/새로고침; 기존 CLI/MCP·DSL/session/읽기 API와 mandatory checks 통과; 개인/팀·로컬/SSH 인수 증거 누락 0 | build ID/asset digest, 회귀 로그, 전체 WEB 검증 보고서 |

WEB-14는 자기 자신의 보고서를 요구하여 순환하지 않는다. WEB-01–13의 최신 완전한 결과와
WEB-14의 별도 build/회귀 assertion을 종합한 최종 보고서를 생성한다.
필수 정적 검증은 `make check`, `go test -race ./...`, `go vet -tags e2e ./...`,
Web 빌드는 `npm --prefix web run build`다. 프론트 계약/편집·실제 브라우저 검증도 runner가
연결해야 한다. 기존 Go 테스트가 skip되면 필요한 Web 시나리오를 별도로 실행하거나 미완료다.

## 체인 명령·옵션 증거

[체인 분석](../chain-analysis/README.md)의 세 체인 × 네 산출물과 현행
`internal/core/nodeconfig/launchopt.go`·모듈 validator·기존 usecase를 기준 집합으로 둔다.
체인별 coverage는 문서 경로/digest/캡처 커밋, command/subcommand 경로와 위치 인자,
global/local 범위, OptionKey·철자·boolean/값 형태, schema/default/제약, hidden 검증,
엔진 mapping과 selectable 여부·제외 이유를 연결한다. 없는 허용값을 추정하여 enum으로 만들지 않는다.

실제 바이너리는 version/help 캡처와 안전한 검사를 격리 수행해 fingerprint와 연결한다.
`scripts/chain-analysis/capture-cli.sh`, `verify-docs.sh`와
`TestDialectSpellingsExistInTheBinaries`를 활용한다. 스냅샷 테스트만으로 선택 바이너리의
지원 여부를 보장하지 않는다. command-local scope·숨김 등록·부재도 검증한다.

필수 부정 사례는 두 gwemix의 잘못된 체인 연결, stale 계약/바이너리, `--docroot`처럼
정의되지만 부재한 플래그, 다른 dialect의 옵션, boolean/값 혼동, chainId/networkid 혼동,
metrics 비활성 상태의 endpoint 입력, 엔진 미매핑 binary 명령의 실행 거절이다.
RPC/metric은 각 체인의 namespace·단위·수집 지점과 실제 자료를 대조한다.
미매핑 binary 항목은 이유와 함께 분류하고 현행 엔진 지원 항목은 폼 coverage에서 누락하지 않는다.
