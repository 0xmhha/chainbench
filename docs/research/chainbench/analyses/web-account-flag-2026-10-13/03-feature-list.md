# 기능별 영향

로그인 여부와 실행 주체를 분리하면 정상 Web 기능을 계속 제공할 수 있다. 각 행의 수정 범위는 구현 제안이다.

| 기능 | 현재 연결 | 무계정 모드에서 필요한 처리 |
|---|---|---|
| 처음 접속과 로그인 | `web/src/LoginPanel.svelte:10-17`, `internal/dashboard/security.go:184-185` | `accountsEnabled`로 분기하고 로컬 실행 문맥을 자동 발급한다. |
| 공유 구성과 Workspace | `internal/dashboard/deployment.go:28-40`, `web/src/DeploymentEditor.svelte:66` | 같은 API에 로컬 authenticator를 전달한다. 자료 조회 조건은 로그인 여부와 분리한다. |
| 구성 자료와 바이너리 | `web/src/AssetManager.svelte:6-17`, `web/src/ManifestManager.svelte:7-11` | 로컬 운영자의 권한과 CSRF를 제공한다. 업로드 검사는 유지한다. |
| 테스트 정의와 작업 제출 | `web/src/DSLEditor.svelte:51-52`, `web/src/JobPanel.svelte:25-26`, `web/src/JobPanel.svelte:96` | 실행 주체가 있어야 초기 자료 조회와 쓰기를 사용할 수 있다. |
| 작업 실행과 취소 | `internal/app/web_job_execution.go:54-80`, `web/src/JobCard.svelte:10` | 실행 중 권한 함수도 로컬 주체를 허용해야 한다. 소유권 검사는 유지한다. |
| SSH와 테스트 서명 키 | `internal/app/deployment_store.go:397-406`, `internal/app/deployment_job_credentials.go:38-53` | 같은 로컬 ID로 저장하고 복호화한다. 다른 계정의 키는 자동 승계하지 않는다. |
| 모니터링과 SSE | `internal/dashboard/security.go:36-43`, `web/src/job-stream.svelte.js:17-22` | 자동 발급한 브라우저 문맥도 스트림 권한 검사에 연결한다. |
| 결과 조회와 삭제 | `web/src/HistoryPanel.svelte:13`, `web/src/HistoryPanel.svelte:38-42` | 로컬 운영자의 권한으로 기존 기능을 사용한다. 서버가 권한을 결정한다. |
| 사용자 관리 | `web/src/SettingsPanel.svelte:16`, `internal/dashboard/security.go:192-194` | 계정 없는 모드에서는 숨기고 해당 API도 비활성화한다. |
| 감사 기록 | `internal/dashboard/audit.go:60-78`, `web/src/AuditPanel.svelte:12` | concrete WebAuth 대신 인증 함수와 감사 소스를 주입할 수 있도록 바꾼다. |

Web 로그인 계정을 끄는 것과 SSH 로그인 또는 체인 트랜잭션 서명 키를 없애는 것은 다른 요구다. 원격 실행과 서명이 필요한 기능은 계속 자격증명을 사용한다. (`internal/app/web_access_contract.go:26-30`, `internal/app/deployment_job_credentials.go:38-53`)

기존 `legacy-observation`은 정상 제어 API를 등록하지 않는다. 따라서 위 기능을 제공하는 계정 없는 모드의 별칭으로 사용해서는 안 된다. (`cmd/chainbench-dashboard/main.go:85-112`, `cmd/chainbench-dashboard/main.go:130-176`)
