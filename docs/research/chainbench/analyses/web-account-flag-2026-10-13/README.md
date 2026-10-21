# Web 계정 선택 플래그 분석

분석일: 2026-10-13. 대상: `662fff77382eb031df687d371a8bc8f63cad581f`.
추적 중인 소스와 화면 파일 전체를 AST로 파싱한 뒤, 백엔드와 프런트엔드를 두 주제로 나누어 코드 근거를 확인했다.
본문과 그래프는 위 커밋에서 진행한 구현 전 분석이다. 당시에는 제품 코드와 실행 중인 서버를 변경하지 않았다.

이후 사용자 요청으로 `-accounts`를 구현했고 기본값은 `false`로 확정했다. `-accounts=true`는 기존 계정 기능을 켠다. 무계정 모드에서는 로컬 브라우저 문맥을 자동 제공하며, CSRF와 감사 기록을 유지한다. 현재 사용법은 저장소 README의 Dashboard 절에 있다. 아래 그래프는 구현 전 커밋의 자료로 보존한다.

## 결론

계정 사용 여부를 Boolean 플래그로 선택할 수 있다. 엔진이 필요로 하는 것은 로그인 계정 자체보다 `DeploymentActor`와 실행 권한 검사 함수다. 계정 없는 모드에서도 서버가 고정된 로컬 실행 주체를 공급하면 배포, 테스트, 노드 제어, 결과 조회를 재사용할 수 있다. (`internal/dashboard/deployment.go:17`, `internal/app/deployment_store.go:27-33`, `internal/app/web_chain_engine.go:118-125`)

로그인 검사만 빼는 수정으로는 충분하지 않다. 백엔드는 HTTP 요청과 실행 중 권한을 각각 검사하고, 화면은 `webSession`으로 자료 조회와 버튼 표시를 결정한다. (`internal/dashboard/security.go:81-104`, `internal/app/web_job_execution.go:49-80`, `web/src/JobPanel.svelte:25-26`, `web/src/JobPanel.svelte:96`)

분석 당시 제안했고 이후 구현한 명령은 다음과 같다.

```sh
./bin/chainbench-dashboard -accounts=true   # 현재 계정 기능 유지
./bin/chainbench-dashboard -accounts=false  # 로그인 없이 정상 Web 기능 사용
```

기본값은 `false`로 확정했다. 외부 주소에서 계정을 사용하는 기존 서비스는 `-accounts=true`를 명시해야 한다.

무계정 모드는 루프백에서 사용하는 단일 로컬 운영자로 설계하는 것이 적합하다. 현재 personal 모드도 루프백만 허용한다. 외부 접속까지 무계정으로 허용하는 요구는 별도로 정해야 한다. (`internal/dashboard/mode.go:24-31`)

## 읽는 순서

| 자료 | 내용 |
|---|---|
| [05-design-insights.md](05-design-insights.md) | 변경 범위, 권장 계약, 모드 전환과 검증 기준 |
| [02-diagrams.md](02-diagrams.md) | 현재 인증 경로와 제안 구조 |
| [01-code-graph.md](01-code-graph.md) | 전체 AST 범위, 수치와 해석 한계 |
| [03-feature-list.md](03-feature-list.md) | 기능별 변경 영향 |
| [04-history-evolution.md](04-history-evolution.md) | 현재 결합이 생긴 이력 |
| [graph/go-packages.svg](graph/go-packages.svg) | Go 패키지 전체의 import 관계 |
| [graph/web-files.svg](graph/web-files.svg) | 웹 소스 전체의 파일 import 관계 |
| [graph/source-graph.json](graph/source-graph.json) | 파일별 AST 통계, 선언, import, 구문상 호출, 오류 위치 |
| [graph/go-package-graph.json](graph/go-package-graph.json) | 패키지와 선언, 참조하는 외부 공개 심볼 |

검증 범위와 재생성 절차는 [graph/README.md](graph/README.md)에 적었다. 기존 날짜의 그래프는 덮어쓰지 않았다.
