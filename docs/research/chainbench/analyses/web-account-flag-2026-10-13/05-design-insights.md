# 계정 선택을 구현할 때의 설계

## 1. 로그인 계정과 실행 주체는 서로 다르다

### 현재 구현

`DeploymentAuthenticator`는 HTTP 요청에서 `DeploymentActor`를 만든다. actor에는 ID와 역할만 있다. 쓰기는 ID가 비어 있지 않고 역할이 admin 또는 operator일 때 허용된다. (`internal/dashboard/deployment.go:17`, `internal/app/deployment_store.go:27-33`)

```go
type DeploymentAuthenticator func(*http.Request) (app.DeploymentActor, error)
```

계정이 없어도 서버가 actor를 제공할 수 있는 구조다. 단, 이를 요청 JSON이나 브라우저가 지정하게 만들면 안 된다. 현재 actor도 신뢰하는 서버 층에서 공급한다. (`internal/app/deployment_store.go:26-29`)

### 제안

`-accounts=true|false`를 추가한다. true이면 기존 WebAuth를 사용한다. false이면 같은 data root에서 안정적으로 유지되는 예약된 로컬 actor ID를 사용한다. 서버 내부 역할은 `admin`, 화면에 보이는 역할은 기존 계약과 같은 `administrator`로 맞춘다. 기존 변환도 이를 수행한다. (`internal/dashboard/security.go:21-26`)

브라우저마다 임의 actor를 만들지 않는다. actor ID를 기동 때마다 바꾸지도 않는다. ID가 달라지면 자격증명과 작업 소유자가 달라진다. (`internal/app/deployment_store.go:397`, `internal/app/web_jobs.go:287-288`)

기본값은 별도의 결정이다. 미지정 때 무계정을 원하는 경우 false, 기존 실행 명령을 보존하는 경우 true다. 분석에서는 선택을 강제하지 않는다.

## 2. HTTP 보호와 계정 로그인을 분리한다

### 현재 구현

`WithWebSecurity`는 쿠키 인증 외에도 Origin 검사, CSRF 검사, 감사, 응답에서 민감정보 가리기, 요청 ID, 스트림 권한을 묶어서 설정한다. 옵션 전체를 생략하면 이 기능들도 빠진다. (`internal/dashboard/security.go:33-53`, `internal/dashboard/security.go:65-104`, `internal/dashboard/security.go:121-127`)

각 정상 Web API도 별도로 인증 함수를 부른다. `authenticate=nil`은 계정 없는 모드가 아니라 거절을 뜻한다. (`internal/dashboard/deployment.go:33-40`, `internal/dashboard/jobs.go:19-25`, `internal/dashboard/history.go:19-25`)

### 제안

계정 로그인과 사용자 관리만 선택 기능으로 둔다. 공통 HTTP 보호는 항상 설치한다. 무계정 브라우저에는 로그인 입력 없이 쿠키와 CSRF를 자동 발급할 수 있다. 이 쿠키는 등록된 사용자 계정이 아니라, 같은 출처에서 연 브라우저의 요청을 구분하는 임시 문맥이다.

Origin과 CSRF를 계속 검사하고, 서버가 로컬 actor를 결정한다. 이벤트 발행자의 `CHAINBENCH_PUBLISHER_TOKEN`은 별도 권한이므로 계정 플래그와 독립적으로 유지한다. (`cmd/chainbench-dashboard/main.go:86-93`, `internal/dashboard/security.go:74-104`)

무계정 모드의 주소 검사는 루프백으로 제한하는 방안을 권장한다. 정상 Web은 로컬 프로세스와 SSH 작업을 실행할 수 있고, 현재 personal 모드도 루프백 전용이다. 외부 주소에서 무계정 제어를 원한다면 별도 접근 정책을 정해야 한다. (`internal/dashboard/mode.go:24-31`, `internal/webcli/runner.go:48-59`)

## 3. 작업 실행 중 검사에도 같은 정책을 공급한다

### 현재 구현

HTTP에서 통과한 작업도 실행 직전 권한을 다시 확인한다. chain engine과 자격증명 접근 역시 권한 함수를 사용한다. (`internal/app/web_job_execution.go:54-80`, `internal/app/web_chain_engine.go:118-125`, `internal/app/deployment_job_credentials.go:41-42`)

기존 `WebAuth.AuthorizeJobActor`는 계정 목록에 없는 actor를 거절한다. 무계정 actor를 이 함수에 보내면 작업이 실패한다. (`internal/app/web_auth.go:201-208`)

### 제안

무계정 전용 authorizer가 예약된 로컬 ID와 허용된 역할을 검사하도록 한다. 같은 함수를 deployment store, engine, jobs에 공급한다. nil로 검사를 없애는 것보다 허용 대상이 코드에 드러나는 방식이 적합하다.

로그아웃과 계정 권한 철회를 구분하는 기존 동작은 true 모드에서 보존한다. 로그아웃은 브라우저 세션을 끊고, 계정 권한 철회는 작업의 후속 실행을 막는다. (`internal/app/web_auth.go:199-208`)

## 4. 화면은 계정 사용 여부와 실행 문맥을 분리한다

### 현재 구현

로그인 화면은 auth/config가 404이면 세션을 undefined로 둔다. 이는 legacy 관측의 처리다. App은 화면을 만들 수 있지만 여러 컴포넌트는 `if(webSession)` 때문에 자료를 읽지 않는다. (`web/src/LoginPanel.svelte:10-15`, `web/src/App.svelte:107-113`, `web/src/JobPanel.svelte:96`)

작업 화면은 세션의 사용자 역할로 편집 권한을 판단한다. API 클라이언트는 세션의 CSRF를 보낸다. 따라서 로그인 화면만 숨겨도 작업 화면은 정상 동작하지 않는다. (`web/src/JobPanel.svelte:25-26`, `web/src/web-api.mjs:6-12`)

### 제안하는 계약

공개 config에 `accountsEnabled`를 추가한다. 실행 문맥을 얻는 공통 API는 계정 모드에서는 현재 로그인 결과를 반환하고, 무계정 모드에서는 로컬 actor와 CSRF를 자동 제공한다. API 이름은 구현할 때 정한다. 기존 auth/me를 확장하거나 별도 access/context를 둘 수 있다.

```json
{
  "accountsEnabled": false,
  "bootstrapRequired": false
}
```

계정 모드 여부는 로그인, 로그아웃, bootstrap, 사용자 관리 표시만 결정한다. 실행 문맥은 목록 조회와 권한 표시를 결정한다. 계정 없는 모드에서도 사용자 관리만 숨기고 자격증명, 감사, 자료 관리 기능은 유지한다. 현재 설정 화면은 사용자 관리와 감사를 같은 조건 안에 둬서 이 부분도 나눠야 한다. (`web/src/SettingsPanel.svelte:15-18`)

초기 조회와 권한 검토 대상은 App, LoginPanel, SettingsPanel, AssetManager, ChainPresetEditor, ChainSurfacePanel, CredentialManager, DeploymentEditor, DSLEditor, HistoryPanel, JobDetail, JobPanel, ManifestManager, MonitorPanel, NodeDetail, UserManager, AuditPanel, JobCard다. 세션의 기존 형태를 임시로 재사용하면 변화량을 줄일 수 있지만, 그 객체가 실제 계정인지 로컬 문맥인지는 명시적으로 구분한다.

## 5. 모드 전환은 로그인보다 소유권 문제다

### 현재 구현

자격증명은 생성자의 ID를 OwnerID로 갖는다. 목록 조회도 admin 예외 없이 본인 자료만 반환한다. AES-GCM 암호화의 인증 데이터에는 credential ID와 owner ID가 포함된다. (`internal/app/deployment_store.go:397-418`)

```go
encrypted := s.aead.Seal(nonce, nonce, b, []byte(m.ID+":"+m.OwnerID))
```

복호화도 같은 owner ID를 쓴다. OwnerID만 바꾸면 복호화가 실패한다. Web 로그인 없이 사용하더라도 SSH와 테스트 서명 키는 계속 필요하다. (`internal/app/deployment_job_credentials.go:38-53`, `internal/app/web_access_contract.go:26-30`)

재시작 복구는 기존 작업의 actor ID로 권한을 확인한다. 계정 모드에서 만든 작업을 무계정 모드가 자동 승계한다고 가정해서는 안 된다. (`internal/app/web_job_reconcile.go:195-196`)

### 제안

기존 데이터를 삭제하거나 owner를 암묵적으로 변경하지 않는다. 공유 문서와 결과는 현재 계약대로 보존하고, 다른 actor의 credential과 계획은 자동 승계하지 않는다. 모드별 data root를 권장하거나, 같은 root의 모드 변경을 기록하고 명시적인 이전 절차를 제공하는 방안을 택한다.

기존 진행 중 작업과 미해결 리소스가 있을 때 모드 변경을 어떻게 처리할지도 구현 범위에 포함한다. 초기 구현에서는 변경을 거절하고 정리 또는 다른 data root 사용을 안내하는 방식이 단순하다. 이것은 제안이며 현재 동작이 아니다.

## 수정 범위와 검증 기준

| 경계 | 수정할 위치 | 검증할 결과 |
|---|---|---|
| 실행 옵션 | `cmd/chainbench-dashboard/main.go:25-38` | 플래그 true/false와 주소 조합이 정한 계약대로 동작한다. |
| HTTP와 인증 | `internal/dashboard/security.go:31`, `internal/dashboard/deployment.go:17` | 무계정 첫 접속은 로그인 없이 열리고, 쓰기에는 공통 보호가 적용된다. |
| 실행 권한 | `internal/app/web_auth.go:201`, `internal/app/web_job_execution.go:54` | 계획, 제출, 실행, SSH 직전 모두 같은 로컬 정책을 쓴다. |
| 감사 | `internal/dashboard/audit.go:60-78` | WebAuth 없이도 요청과 작업을 기록하고 조회한다. |
| 화면 문맥 | `web/src/LoginPanel.svelte:10`, `web/src/App.svelte:34`, `web/src/web-api.mjs:6` | 모든 목록, 편집, 업로드, 작업, snapshot과 SSE가 초기화된다. |
| 소유권과 복구 | `internal/app/deployment_store.go:397-418`, `internal/app/web_job_reconcile.go:195-196` | 재시작해도 로컬 ID가 유지되고 다른 계정의 키를 승계하지 않는다. |

true 모드에서는 기존 401/403, viewer 제한, 사용자 관리, 세션 철회, CSRF 테스트를 그대로 유지한다. 해당 계약은 기존 테스트에 있다. (`internal/dashboard/security_test.go:47-104`, `internal/dashboard/security_test.go:149-183`, `internal/dashboard/assets_security_test.go:54-76`, `tests/webui/browser_web10.mjs:24-57`)

false 모드에서는 계정 등록 없이 첫 접속, 전체 목록과 계약 조회, 자료 업로드, 계획 검토와 제출, 작업 취소, 결과 삭제, 자격증명 저장과 사용, 새로고침과 재시작, 감사와 SSE를 검증한다. 사용자 API는 노출하지 않고 계정 화면도 표시하지 않는다. 다른 출처에서 온 쓰기 요청과 CSRF 없는 요청은 계속 거절해야 한다.

이번 작업은 이 구현을 하기 위한 분석이다. 기존 인증을 우회하는 임시 코드는 추가하지 않았다.
