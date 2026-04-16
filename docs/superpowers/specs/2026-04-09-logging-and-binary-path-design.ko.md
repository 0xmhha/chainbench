# 설계: 로깅 통합 & 바이너리 경로 설정

**날짜:** 2026-04-09
**상태:** 승인됨, 구현 계획 수립 대기 중
**범위:** `chainbench` 리포지토리 (bash CLI + MCP 서버)

## 1. 배경

`lib/cmd_start.sh`, `lib/common.sh`, `mcp-server/src/tools/`에 대한 코드 리뷰에서 두 가지 결함이 발견됨:

1. **로그 로테이션이 선언만 되고 구현되지 않음.** `cmd_start.sh:200-213`에서 `_logrot_bin`, `_log_max_size`, `_log_max_files` 변수를 선언하지만, 실제 노드 실행은 단순히 `nohup ... >> "${node_log}" 2>&1 &`로 수행함. 주석에서 참조하는 `bin/logrot` 바이너리는 리포에 존재하지 않음. 프로파일 필드 `logging.rotation`, `logging.max_size`, `logging.max_files`는 아무런 효력이 없음. 노드 로그는 무한히 증가함.
2. **바이너리 경로 설정에 깨끗한 런타임 override 채널이 없음.** `chain.binary_path`를 변경하려면 git-tracked 프로파일 YAML을 직접 편집하거나, `chainbench_profile_set` MCP 도구를 호출(동일 YAML에 기록)해야 함. CLI 플래그도 없고, 환경변수 경로(프로파일 로더가 사전 export된 환경변수를 덮어씀)도 없고, 머신-로컬 오버레이도 없음.

`logrot` 바이너리 자체는 이미 업스트림에 존재함: `go-wemix/cmd/logrot/main.go`는 `github.com/charlanxcc/logrot` 패키지의 3줄짜리 래퍼로, 인터페이스는 `logrot <file> <size> <count>`. `logrot`은 **파일 감시(file-watching) 로테이터**로, 디스크 상의 지정된 파일을 모니터링하다가 주어진 크기 제한을 초과하면 로테이션하며 `count`개의 백업 파일을 유지함. stdin에서 로그 데이터를 읽지 않음. 업스트림 스크립트에서 사용하는 `> file 2>&1 | logrot file 10M 5 &` 파이프 패턴은 **라이프사이클 결합**을 제공: gstable이 `>`를 통해 파일에 직접 쓰고, logrot이 같은 파일을 감시하며, gstable이 종료되면 파이프가 닫혀 logrot도 함께 종료됨. 형제 리포 `stable-net/test/local-test/script/run_gstable.sh:92`에서 이 호출 패턴을 확인할 수 있음. go-stablenet도 동일 소스 패키지로 `logrot`을 빌드할 수 있음.

## 2. 목표

1. `profiles/*.yaml`의 `logging.*` 설정이 실제로 효력을 발휘하도록 로그 로테이션 복원.
2. `gstable` 옆, git-root `build/bin/`, 또는 `cmd/logrot/main.go` 소스로부터 온디맨드 빌드를 통해 `logrot`을 자동 탐색.
3. 바이너리 경로 해석에 대한 명확한 우선순위 체인 제공: CLI 플래그 → 환경변수 → 로컬 오버레이 → 프로파일 YAML → 자동 감지.
4. `_cb_set_var` 환경변수 override 버그 수정: 사전 export된 `CHAINBENCH_*` 변수가 `load_profile`에 의해 존중되도록 함.
5. 머신-로컬 오버레이 파일(`state/local-config.yaml`, git-ignored)에 기록하는 `chainbench config` CLI 서브커맨드 추가.
6. MCP 원자적 init 추가: `chainbench_init` / `chainbench_start` / `chainbench_restart` / `chainbench_node_start`에 `binary_path?` 파라미터 추가 + 오버레이에 기록하는 신규 `chainbench_config_set` 도구.
7. 이전 리팩토링의 잔여 `.bak` 파일을 첫 번째 커밋에서 제거.

## 3. 비목표

- `logrot`을 직접 작성하지 않음. 업스트림 Go 패키지를 재사용.
- `chainbench_profile_set` 대체 안 함. 팀 공유 프로파일 편집은 기존 MCP 도구를 통해 계속 작동함. 오버레이는 추가 레이어.
- MCP 서버 테스트 또는 TypeScript 레이어용 테스트 프레임워크 추가 안 함.
- `lib/adapters/` 또는 `lib/chain_adapter.sh`의 체인 어댑터 추상화 확장 안 함.
- 로그 포맷 또는 `cmd_log.sh`의 timeline/anomaly/search 동작 변경 안 함.
- `chainbench_config_get` MCP 도구 추가 안 함. `chainbench_profile_get({ name: "active" })`로 이미 적용 중인 병합 값을 확인 가능.

## 4. 아키텍처

### 4.1 설정 우선순위 체인

모든 `CHAINBENCH_*` 변수(특히 `BINARY_PATH`와 `LOGROT_PATH`)는 다음 순서로 해석됨:

```
1. CLI 플래그           --binary-path / --logrot-path         (일회성, 서브커맨드별)
2. 환경변수             CHAINBENCH_BINARY_PATH, ...           (쉘 세션)
3. 로컬 오버레이        state/local-config.yaml               (머신-로컬, git-ignored)
4. 프로파일 YAML        profiles/<name>.yaml + inherits 체인  (팀 공유, git-tracked)
5. 자동 감지            git-root/build/bin → PWD/build/bin
                        → dirname($BINARY)/logrot
                        → go build ./cmd/logrot
                        → $PATH
6. 미발견               log_warn + 우아한 fallback (로그에 대해 단순 `>>` append)
```

현재 코드는 `_cb_set_var`가 무조건 `export`하여 사전 export된 환경변수를 덮어쓰기 때문에 #2와 #3이 #4로 합쳐져 버림. 수정: env-first — `CHAINBENCH_*` 변수가 이미 설정되어 있고 비어있지 않으면 `_cb_set_var`가 조기 리턴. 사용자는 `CHAINBENCH_PROFILE_ENV_OVERRIDE=0`으로 opt-out 가능.

### 4.2 프로파일 병합 레이어

```
profiles/<name>.yaml + inherits 체인
        │ load_with_inheritance()
        ▼
병합된 프로파일 (메모리 내)
        │ deep_merge(state/local-config.yaml) 존재 시     <-- 신규
        ▼
최종 병합 JSON (CHAINBENCH_PROFILE_JSON에 기록)
        │ _cb_export_profile_vars() + env-first 가드
        ▼
export된 CHAINBENCH_* 환경변수
```

오버레이 병합은 `lib/profile.sh`의 `_cb_python_merge_yaml` Python 블록 내부에서 수행됨. `CHAINBENCH_DIR`이 Python 스크립트의 새 인자로 전달됨.

### 4.3 Logrot 탐색 (하이브리드 + 자동 빌드)

```
resolve_logrot(binary_path, explicit_logrot_path):
  1. explicit_logrot_path가 설정되어 있고 실행 가능하면  → 사용
  2. dirname(binary_path)/logrot가 실행 가능하면          → 사용
  3. <git-root>/build/bin/logrot가 실행 가능하면           → 사용
  4. <git-root>/cmd/logrot/main.go가 존재하고
       `go` 커맨드가 사용 가능하면
       → `go build -o <git-root>/build/bin/logrot ./cmd/logrot` 실행
         state/logrot-build.log에 로그 기록
       → 성공 시, 새로 빌드된 바이너리 사용
  5. logrot이 $PATH에 있으면                              → 사용 (경고 포함)
  6. 그 외                                                → 빈 문자열 리턴
                                                            + log_warn
```

6단계는 치명적 오류가 아님. 리턴이 비어있으면 `cmd_start.sh`는 단순 `>>` append로 fallback. 프로파일에서 `logging.rotation: false`면 탐색 자체를 단축(short-circuit)함.

### 4.4 노드 실행 파이프라인

**현재 (logrot 선언만 되고 사용 안 됨):**

```bash
nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
# _logrot_bin, _log_max_size, _log_max_files 선언만 되고 사용 안 됨
```

**교체 — 별도 프로세스 모델:**

`logrot`은 파일 감시(file-watching) 로테이터이며 stdin에서 읽지 않음. 두 가지 실행 모델을 검토:

| 모델 | 패턴 | PID 추적 |
|---|---|---|
| **파이프 (업스트림 패턴)** | `nohup gstable ... > file 2>&1 \| logrot file 10M 5 &` | `$!` = logrot PID (헬스체크에 부적합) |
| **별도 프로세스** | gstable `> file 2>&1 &` 후 `logrot file 10M 5 &` | 첫 번째 커맨드의 `$!` = gstable PID (정확) |

업스트림 stable-net 테스트 스크립트는 라이프사이클 결합을 위해 파이프 모델을 사용 (gstable 종료 시 파이프 닫힘 → logrot 종료). 그러나 chainbench는 `node stop/start` 작업을 위한 노드별 PID 추적이 필요함. **별도 프로세스 모델**이 기존 PID 의미론을 보존:

```bash
# gstable 실행 — PID 추적은 현재 코드와 동일
nohup "${launch_cmd[@]}" >> "${node_log}" 2>&1 &
local node_pid=$!
disown "${node_pid}" 2>/dev/null || true

# logrot을 동반 파일 감시 프로세스로 실행 (가용 시)
if [[ -n "${LOGROT_BIN}" ]]; then
  nohup "${LOGROT_BIN}" "${node_log}" \
    "${CHAINBENCH_LOG_MAX_SIZE}" "${CHAINBENCH_LOG_MAX_FILES}" &
  local logrot_pid=$!
  disown "${logrot_pid}" 2>/dev/null || true
  log_info "  logrot watching ${node_log} (PID ${logrot_pid})"
fi
```

**트레이드오프**: logrot의 라이프사이클이 더 이상 파이프를 통해 gstable과 결합되지 않음. gstable이 crash하면 logrot은 `cmd_stop.sh`가 `pkill -f "logrot.*node.*log"` (이미 `cmd_stop.sh:55`에 존재)로 정리할 때까지 계속 실행됨. 정상 워크플로에서 `cmd_stop.sh`는 항상 `cmd_start.sh` 전에 실행되므로 이는 수용 가능.

**보존되는 핵심 불변량**: gstable 실행 후의 `$!`는 gstable의 PID — 모든 기존 PID 기반 작업 (`node stop`, `node start`, 헬스체크)이 수정 없이 동작.

### 4.5 컴포넌트 경계

```
chainbench.sh
  └─ lib/cmd_<subcommand>.sh를 source

lib/common.sh
  ├─ resolve_binary()                   (기존)
  ├─ resolve_logrot()                   (신규)
  ├─ _cb_build_logrot_from_source()     (신규)
  └─ _cb_parse_runtime_overrides()      (신규)

lib/profile.sh
  ├─ load_profile()                     (기존, 인터페이스 불변)
  ├─ _cb_set_var()                      (수정, env-first)
  └─ _cb_python_merge_yaml()            (수정, 오버레이 병합)

lib/cmd_init.sh      (수정: 상단에 런타임 override 파서)
lib/cmd_start.sh     (수정: 파서 + logrot + 실행 파이프라인)
lib/cmd_restart.sh   (수정: override를 init/start로 전달)
lib/cmd_node.sh      (수정: node start 내부에 파서)
lib/cmd_stop.sh      (수정: logrot 정리 메시지)
lib/cmd_config.sh    (신규, ~250-350줄)
    └─ get / set / unset / list, state/local-config.yaml 읽기/쓰기

mcp-server/src/utils/exec.ts            (수정: shellEscapeArg)
mcp-server/src/tools/lifecycle.ts       (수정: +binary_path)
mcp-server/src/tools/node.ts            (수정: +binary_path)
mcp-server/src/tools/config.ts          (신규: chainbench_config_set)
mcp-server/src/tools/schema.ts          (수정: 문서 섹션)
mcp-server/src/index.ts                 (수정: config 도구 등록)

profiles/default.yaml                   (수정: +chain.logrot_path)
.gitignore                              (수정: +오버레이 + 빌드 로그)
README.md                               (수정: CLI 참조, 스키마, MCP, 문제해결)
setup.sh                                (수정: 다음 단계 안내)
```

## 5. 상세 컴포넌트 변경 사항

### 5.1 `lib/common.sh`

신규 함수 3개. 기존 `resolve_binary`, `get_node_port`, `is_truthy`, `require_cmd`, 색상/로깅 헬퍼는 변경 없음.

**`resolve_logrot(binary_path, explicit_logrot_path)`** — §4.3 구현. stdout에 절대경로를 출력하거나 빈 문자열. 운영 로그 메시지는 `log_info` / `log_warn`을 통해 stderr로 출력.

**`_cb_build_logrot_from_source(git_root)`** — `resolve_logrot` 4단계에서 호출되는 내부 헬퍼. `<git_root>/cmd/logrot/main.go` 확인 → `go` 사용 가능 여부 확인 → `git_root`에서 `go build -o build/bin/logrot ./cmd/logrot` 실행, 빌드 출력을 `state/logrot-build.log`로 리다이렉트. 성공 시 빌드된 경로를 리턴, 실패 시 빈 문자열.

**`_cb_parse_runtime_overrides(remaining_array_ref, ...args)`** — override를 수락하는 각 커맨드에서 호출되는 공유 CLI 플래그 파서. bash nameref 사용. `--binary-path`, `--binary-path=`, `--logrot-path`, `--logrot-path=`를 소비하고, 해당하는 `CHAINBENCH_*` 환경변수를 export, 알 수 없는 플래그는 `remaining` 배열에 추가하여 커맨드가 처리.

빈 값과 비절대 경로는 `log_error`와 exit 1로 즉시 실패.

### 5.2 `lib/profile.sh`

**`_cb_set_var` env-first 가드:**

```bash
_cb_set_var() {
  local var_name="$1"
  local field="$2"
  local default="${3:-}"

  if [[ "${CHAINBENCH_PROFILE_ENV_OVERRIDE:-1}" == "1" ]] \
      && [[ -n "${!var_name+x}" ]] \
      && [[ -n "${!var_name}" ]]; then
    return 0
  fi

  local value
  value="$(_cb_jq_get "$json_file" "$field" "$default")"
  export "${var_name}=${value}"
}
```

`-n "${!var_name+x}"` 테스트는 "설정되어 있는가"를, `-n "${!var_name}"`는 "비어있지 않은가"를 확인. 사용자가 override를 리셋하려면 `export CHAINBENCH_BINARY_PATH=""`로 하면 프로파일 값으로 fall through됨.

**`_cb_python_merge_yaml`에서의 오버레이 병합:**

```python
merged = load_with_inheritance(PROFILE_PATH, PROFILES_ROOT)

overlay_path = os.path.join(CHAINBENCH_DIR, "state", "local-config.yaml")
if os.path.isfile(overlay_path):
    overlay = load_yaml_file(overlay_path)
    if overlay:
        # 오버레이에서 'inherits'는 경고 후 삭제; 오버레이는 순수 override 레이어.
        if 'inherits' in overlay:
            print("WARN: local-config.yaml: 'inherits' field ignored", file=sys.stderr)
            overlay = {k: v for k, v in overlay.items() if k != 'inherits'}
        merged = deep_merge(merged, overlay)

print(json.dumps(merged, ensure_ascii=False))
```

`CHAINBENCH_DIR`이 Python 블록의 세 번째 인자로 전달됨 (`profile_path`와 `_CB_PROFILES_DIR` 이후).

**새 변수 export:**

```bash
_cb_set_var CHAINBENCH_LOGROT_PATH ".chain.logrot_path" ""
```

`_cb_export_profile_vars` 내부 `chain.*` 블록에 추가.

### 5.3 `lib/cmd_init.sh`, `lib/cmd_start.sh`, `lib/cmd_restart.sh`, `lib/cmd_node.sh`

각 파일 상단에 파서 블록 추가 (의존성 source 직후, `load_profile` 이전):

```bash
_CB_INIT_REMAINING=()
_cb_parse_runtime_overrides _CB_INIT_REMAINING "$@"
set -- "${_CB_INIT_REMAINING[@]}"
```

파일이 source될 때 상호 오염을 방지하기 위해 배열 이름을 커맨드별로 다르게 명명 (`_CB_INIT_REMAINING`, `_CB_START_REMAINING` 등).

`cmd_node.sh`의 경우, 파서는 `$1`(노드 번호)이 소비된 **후에** `_cb_node_cmd_start` **내부에서** 실행되므로, 남은 인자는 override 플래그뿐.

`cmd_restart.sh`는 내부적으로 `cmd_init.sh`와 `cmd_start.sh`를 호출함. 최상위 파서가 export한 환경변수가 source된 커맨드를 통해 자연스럽게 전파됨. 단계별 재파싱 불필요.

### 5.4 `lib/cmd_start.sh` logrot 통합

`resolve_binary` 리턴 후, 실행 루프 전:

```bash
LOGROT_BIN=""
if is_truthy "${CHAINBENCH_LOG_ROTATION:-true}"; then
  LOGROT_BIN="$(resolve_logrot "${BINARY}" "${CHAINBENCH_LOGROT_PATH:-}")" || LOGROT_BIN=""
fi

if [[ -z "${LOGROT_BIN}" ]] && is_truthy "${CHAINBENCH_LOG_ROTATION:-true}"; then
  log_warn "logrot not available — logs will grow unbounded. Set chain.logrot_path or build <git-root>/cmd/logrot."
fi
```

`_start_launch_node` 내부에서 200-213번 줄의 죽은 선언(`_logrot_bin`, `_log_max_size`, `_log_max_files`)을 삭제하고, `nohup` 호출을 §4.4의 별도 프로세스 모델로 교체. 기존 `nohup ... >> "${node_log}" 2>&1 &` 라인은 유지하고, 조건부 `logrot` 실행을 동반 감시 프로세스로 추가.

### 5.5 `lib/cmd_config.sh` (신규)

4개 서브커맨드:

- `chainbench config list` — `state/local-config.yaml`의 전체 내용을 출력하거나, 파일이 없으면 `(empty)` 출력.
- `chainbench config get <field>` — dot-notation 조회. 오버레이를 먼저 검색. 필드가 없으면 `(not found)` 출력 후 exit 1.
- `chainbench config set <field> <value>` — `value`를 JSON으로 먼저 파싱 시도; 실패 시 문자열로 취급. 오버레이를 로드하고, nested-set 헬퍼로 수정한 뒤, 원자적으로 기록 (임시 파일 + rename).
- `chainbench config unset <field>` — 필드를 제거. 빈 부모 dict은 cascade 정리.

필드 검증: `^[a-zA-Z0-9_][a-zA-Z0-9_.]*$` (`..` 불가, 선행 점 불가, 경로 탐색 불가). Python의 `yaml.safe_load`가 태그 주입을 차단.

구현 스타일은 `lib/profile.sh`의 기존 인라인 Python 패턴과 일치. YAML 직렬화는 `mcp-server/src/tools/schema.ts:181`의 `jsonToYaml` 헬퍼를 따름.

### 5.6 `mcp-server/src/utils/exec.ts`

신규 헬퍼:

```typescript
export function shellEscapeArg(arg: string): string {
  // POSIX 단일 따옴표 이스케이프, 내장 따옴표는 '\'' 처리
  return `'${arg.replace(/'/g, "'\\''")}'`;
}
```

`runChainbench`는 계속 단일 커맨드 문자열을 받음. `args`에 사용자 제어 값을 포함하는 모든 호출자는 `shellEscapeArg`를 거쳐야 함.

### 5.7 `mcp-server/src/tools/lifecycle.ts` 및 `node.ts`

`chainbench_init`, `chainbench_start`, `chainbench_restart`, `chainbench_node_start`에 `binary_path?: string` 추가. 검증 헬퍼:

```typescript
function validateBinaryPath(binary_path: string | undefined): string | null {
  if (binary_path === undefined) return null;
  if (!binary_path.startsWith("/")) return "binary_path must be an absolute path.";
  if (binary_path.length === 0)     return "binary_path must not be empty.";
  return null;
}
```

`binary_path`가 있으면 `runChainbench` args 문자열에 `--binary-path ${shellEscapeArg(binary_path)}`를 추가.

### 5.8 `mcp-server/src/tools/config.ts` (신규)

단일 도구:

```typescript
server.tool(
  "chainbench_config_set",
  "머신-로컬 오버레이(state/local-config.yaml)에 필드를 기록합니다. " +
  "영구적이지만 git-ignored. chain.binary_path 같은 머신별 경로에 사용하세요. " +
  "git-tracked 프로파일 YAML을 편집하는 chainbench_profile_set과는 다릅니다.",
  {
    field: z.string().describe("Dot-notation 경로 (예: 'chain.binary_path')"),
    value: z.string().describe("값. 유효한 JSON이면 JSON으로 파싱, 아니면 문자열."),
  },
  async ({ field, value }) => {
    if (!/^[a-zA-Z0-9_][a-zA-Z0-9_.]*$/.test(field)) {
      return errorResponse("Invalid field path");
    }
    const result = runChainbench(
      `config set ${shellEscapeArg(field)} ${shellEscapeArg(value)}`
    );
    return { content: [{ type: "text" as const, text: formatResult(result) }] };
  }
);
```

`mcp-server/src/index.ts`에서 `registerConfigTools(server)`로 등록.

### 5.9 프로파일 및 문서 업데이트

- `profiles/default.yaml` — `chain:` 블록에 `logrot_path: ""`를 기존 스타일의 한국어 주석과 함께 추가.
- `.gitignore` — `state/local-config.yaml` 및 `state/logrot-build.log` 추가.
- `README.md` — CLI 참조에 `chainbench config` 섹션 추가, 프로파일 스키마에 `chain.logrot_path` 문서화, MCP 도구 표에 `chainbench_config_set` 추가, logrot 문제해결 항목 추가, Quick Start 주석(73번 줄)을 `chainbench config set chain.binary_path` 권장으로 업데이트.
- `setup.sh` — "다음 단계" 블록을 `profiles/default.yaml` 편집 대신 `chainbench config set` 권장으로 변경, `bash tests/unit/run.sh` 실행 안내 추가.
- `mcp-server/src/tools/schema.ts` — `SECTION_DOCS.chain`에 `chain.logrot_path` 언급 추가, `SECTION_DOCS.logging`에 logrot 통합이 실제로 작동함을 반영.

## 6. 데이터 흐름 워크스루

### 6.1 일회성 CLI override

```
$ chainbench init --profile default --binary-path /opt/gstable-rc1/build/bin/gstable

chainbench.sh       : CHAINBENCH_PROFILE=default; subcommand=init
cmd_init.sh         : _cb_parse_runtime_overrides
                      → export CHAINBENCH_BINARY_PATH=/opt/gstable-rc1/build/bin/gstable
load_profile        : _cb_python_merge_yaml → 병합 JSON
                      → _cb_set_var가 CHAINBENCH_BINARY_PATH 이미 설정 확인 → /opt/... 보존
resolve_binary      : /opt/gstable-rc1/build/bin/gstable 리턴
cmd_init.sh         : state/current-profile-merged.json에 영속화
                      (기존 22-29번 줄 동작)
```

### 6.2 영구적 머신-로컬 설정

```
$ chainbench config set chain.binary_path /opt/gstable/build/bin/gstable
$ chainbench config set chain.logrot_path /opt/gstable/build/bin/logrot
$ chainbench init

cmd_config.sh set   : 중첩된 chain 블록으로 state/local-config.yaml 생성
load_profile        : Python 블록이 프로파일 위에 state/local-config.yaml을 deep-merge
                      → 병합 chain.binary_path = /opt/gstable/build/bin/gstable
_cb_set_var         : 환경변수 미설정 → 병합된 값을 export
resolve_binary      : /opt/gstable/build/bin/gstable
```

### 6.3 Logrot 자동 빌드

전제조건: `/opt/gstable/build/bin/gstable` 존재, `/opt/gstable/build/bin/logrot` 부재, `/opt/gstable/cmd/logrot/main.go` 존재, `go`가 `$PATH`에 있음.

```
resolve_logrot      :
  단계 1 (프로파일 chain.logrot_path)       → 비어있음, 건너뜀
  단계 2 (dirname(BINARY)/logrot)          → 미발견, 건너뜀
  단계 3 (git-root/build/bin/logrot)       → 미발견, 건너뜀
  단계 4 (git-root/cmd/logrot/main.go)     → 발견
    _cb_build_logrot_from_source
      → cd /opt/gstable && go build -o build/bin/logrot ./cmd/logrot
      → stdout+stderr → state/logrot-build.log
      → 성공, /opt/gstable/build/bin/logrot 리턴
  log_info "Built logrot from source: /opt/gstable/build/bin/logrot"
_start_launch_node  :
  nohup gstable ... >> node1.log 2>&1 &        # gstable이 파일에 기록
  node_pid=$!                                    # gstable PID (정확)
  nohup logrot node1.log 10M 5 &                # 동반 파일 감시 프로세스
  logrot_pid=$!
```

### 6.4 MCP 원자적 init

```
사용자: "Initialize the chain with the gstable binary at /opt/gstable-rc1/build/bin/gstable"

LLM → chainbench_init({
  profile: "default",
  binary_path: "/opt/gstable-rc1/build/bin/gstable"
})

lifecycle.ts        :
  validateBinaryPath("/opt/...")  → ok
  args = `init --profile default --quiet --binary-path '/opt/gstable-rc1/build/bin/gstable'`
  runChainbench(args, { cwd: project_root })
```

나머지 흐름은 §6.1과 동일.

### 6.5 logrot을 전혀 사용할 수 없는 경우의 Fallback

전제조건: logrot이 어디에도 없음, 소스 없음, `go` 커맨드 없음.

```
resolve_logrot      : 단계 1-5 모두 miss → log_warn + "" 리턴
cmd_start.sh        : LOGROT_BIN="" → log_warn "logrot not available ..."
_start_launch_node  :
  nohup gstable ... >> node1.log 2>&1 &      # 단순 append, 기존 동작 불변
node_pid            : 이전과 동일하게 $!
```

체인은 정상적으로 시작됨; 사용자는 시작 중 경고 한 줄을 확인.

### 6.6 우선순위 해석 표

| CLI 플래그 | 환경변수 | 오버레이 | 프로파일 | 최종값 |
|-----------|---------|---------|---------|-------|
| `/a`      | -       | -       | `/d`    | `/a`  |
| -         | `/b`    | -       | `/d`    | `/b`  |
| -         | -       | `/c`    | `/d`    | `/c`  |
| -         | -       | -       | `/d`    | `/d`  |
| `/a`      | `/b`    | `/c`    | `/d`    | `/a`  |
| -         | `/b`    | `/c`    | -       | `/b`  |
| `/a`      | -       | -       | -       | `/a`  |
| -         | -       | -       | `""`    | auto  |

이 표는 `tests/unit/tests/common-resolve-binary.sh` 및 `profile-env-override.sh`의 단위 테스트 케이스에 직접 매핑됨.

## 7. 에러 핸들링

### 7.1 런타임 override 파싱

| 실패 모드 | 동작 | 메시지 |
|---|---|---|
| 플래그 값 누락 (인자 끝의 `--binary-path`) | exit 1 | `"--binary-path requires a value (absolute path)"` |
| 비절대 경로 값 | exit 1 | `"--binary-path must be an absolute path (got: 'gstable')"` |
| 값 경로가 존재하지 않음 | 즉시 에러 아님; `resolve_binary`에서 포착 | `resolve_binary`가 경고 후 fallback |
| 알 수 없는 플래그 | 커맨드로 패스스루 | (커맨드가 처리) |

### 7.2 오버레이 병합

| 실패 모드 | 동작 | 메시지 |
|---|---|---|
| 파일 없음 | no-op | 없음 |
| 빈 파일 | no-op | 없음 |
| 잘못된 YAML | `load_profile` 실패, exit 1 | `"ERROR: local-config.yaml parse failed: <세부내용>"` |
| 오버레이에 `inherits` 필드 | 경고 후 제거 | `"local-config.yaml: 'inherits' field ignored in overlays"` |
| 타입 불일치 (예: `validators: "four"`) | `_cb_validate_profile_json`으로 전파 | 기존 검증 경로 |

### 7.3 `_cb_set_var` env-first 엣지 케이스

| 상태 | 현재 | 수정 후 |
|---|---|---|
| env 비어있음, 프로파일에 값 있음 | 프로파일 값 | **변경 없음** |
| env 설정됨 (비어있지 않음), 프로파일에 값 있음 | 프로파일이 env 덮어씀 (버그) | **env 보존** |
| env 설정됐으나 빈 문자열 | 프로파일이 덮어씀 | 프로파일이 덮어씀 (명시적 리셋 UX) |
| `CHAINBENCH_PROFILE_ENV_OVERRIDE=0`, env 설정, 프로파일에 값 있음 | 프로파일이 덮어씀 | 프로파일이 덮어씀 (opt-out) |

### 7.4 Logrot 탐색

| 단계 | 실패 모드 | 동작 |
|---|---|---|
| 1 프로파일 경로 | 실행 불가 | 경고, 진행 |
| 2-3 이웃/git-root | 없음 | 조용히 진행 |
| 4 자동 빌드 | `cmd/logrot/main.go` 없음 | 조용히 진행 |
| 4 자동 빌드 | `go` 사용 불가 | `log_info`, 진행 |
| 4 자동 빌드 | `go build` 실패 | `log_warn "logrot build failed, see state/logrot-build.log"`, 진행 |
| 5 `$PATH` | 없음 | 조용히 진행 |
| 최종 | 모두 miss | 단일 `log_warn`, 빈 문자열 리턴, `cmd_start.sh`는 단순 `>>`로 fallback |

`logging.rotation: false`는 전체 탐색 체인을 단축(short-circuit). 로테이션 비활성화 시 `resolve_logrot` 호출 없음.

### 7.5 `cmd_config.sh`

| 실패 모드 | 동작 |
|---|---|
| 서브커맨드 인자 누락 | 사용법 출력 + exit 1 |
| 유효하지 않은 필드 경로 (빈 값, 이중 점, 선행 점) | `log_error` + exit 1 |
| YAML-unsafe 값 | `yaml.safe_load`가 위험한 태그 거부 |
| `state/` 쓰기 불가 | `log_error` + exit 1 |
| 존재하지 않는 필드에 `unset` | `log_warn` + exit 0 |
| 존재하지 않는 필드에 `get` | `(not found)` + exit 1 |
| 쓰기 중 중단 | 임시 파일 + 원자적 `rename`으로 원본 보존 |

### 7.6 MCP

| 실패 모드 | 동작 |
|---|---|
| `binary_path`가 절대경로 아님 | 에러 응답, CLI 호출 안 함 |
| 전달 값에 쉘 메타문자 포함 | `shellEscapeArg`가 연결 전에 이스케이프 |
| `chainbench_config_set`의 잘못된 필드 | CLI 호출 전 클라이언트 측 정규식으로 거부 |

### 7.7 Logrot 동반 프로세스 라이프사이클

`logrot`은 로그 파일을 감시하는 별도의 백그라운드 프로세스로 실행됨 (gstable에 파이프 연결되지 않음). 라이프사이클 영향:

| 시나리오 | 동작 |
|---|---|
| `logrot`이 즉시 crash | gstable에 영향 없음. 로그는 무한 증가 (logrot 없는 fallback과 동일). `cmd_start.sh` 헬스체크는 gstable PID만 확인. |
| gstable이 정상 종료 | logrot은 계속 실행 (파이프 결합 없음). `cmd_stop.sh:55` `pkill -f "logrot.*node.*log"`로 정리. |
| gstable이 예기치 않게 crash | logrot이 (이제 정적인) 파일을 계속 감시. 다음 `cmd_stop.sh` 또는 `cmd_start.sh`의 stale-PID 가드에서 정리. |
| 이전 실행의 잔여 logrot | `cmd_stop.sh:55` `pkill -f "logrot.*node.*log"`가 안전망 제공. |
| `node stop <N>` 호출 | 저장된 PID로 gstable 종료. 해당 노드의 logrot은 다음 `cmd_stop.sh`까지 유지. 수용 가능 — 정적 파일에 대한 logrot은 유휴 상태이며 무해. |

## 8. 테스팅

### 8.1 디렉토리 구조

```
tests/
├── regression/              (기존, 변경 없음)
└── unit/                    (신규)
    ├── run.sh
    ├── lib/assert.sh
    ├── fixtures/
    │   ├── mock-gstable
    │   ├── mock-go/go
    │   └── profiles/
    └── tests/
        ├── smoke-meta.sh
        ├── common-resolve-binary.sh
        ├── common-resolve-logrot.sh
        ├── common-parse-overrides.sh
        ├── profile-env-override.sh
        ├── profile-overlay-merge.sh
        ├── cmd-config.sh
        └── smoke-logrot-integration.sh
```

`run.sh`는 `tests/*.sh`를 순회하며, 각각을 서브쉘에서 실행하고, pass/fail 합계를 보고. 테스트가 하나라도 실패하면 exit code 비정상.

### 8.2 어설션 헬퍼

`tests/unit/lib/assert.sh`는 `assert_eq`, `assert_neq`, `assert_empty`, `assert_nonempty`, `assert_file_exists`, `assert_contains`, `assert_exit_code`를 export. 외부 의존성 없음. 실패 시 간단한 이유를 출력하고 exit 1.

### 8.3 파일별 케이스 커버리지

**`common-resolve-binary.sh`** (기존 동작에 대한 회귀 방지):
1. 유효하고 실행 가능한 명시적 경로
2. 실행 불가한 명시적 경로 → 자동 감지 fallback
3. git-root/build/bin/$bin
4. PWD/build/bin/$bin
5. $PATH 히트 + 경고
6. 완전 miss → exit 1

**`common-resolve-logrot.sh`**:
1. 명시적 `chain.logrot_path`
2. `dirname($BINARY)/logrot`
3. `git-root/build/bin/logrot`
4. 소스로부터 빌드 (mock `go` 성공)
5. 소스 존재하나 `go` 사용 불가 → 다음 단계
6. 빌드 실패 → 경고, 다음 단계
7. `$PATH` 히트
8. 모두 miss → 빈 문자열 + 경고

`fixtures/mock-go/go`의 mock `go` 바이너리는 인자를 검사하여 `build -o <out> ./cmd/logrot`을 감지하면 `<out>`에 간단한 실행 파일을 기록. 테스트는 `resolve_logrot` 호출 전에 `fixtures/mock-go`를 `PATH`에 추가.

**`common-parse-overrides.sh`**:
1. `--binary-path /x init` → env + 나머지
2. `--binary-path=/x init` → 동일
3. `init --binary-path /x` → 동일
4. `--logrot-path /y --binary-path /x` → 둘 다 export
5. 알 수 없는 플래그 패스스루
6. 값 누락 → exit 2

**`profile-env-override.sh`**:
1. env 설정, 프로파일에 값 있음 → env 승
2. env 빈 문자열, 프로파일에 값 있음 → 프로파일 승 (리셋 UX)
3. env 미설정 → 프로파일 승
4. `CHAINBENCH_PROFILE_ENV_OVERRIDE=0` + env 설정 → 프로파일 승 (opt-out)

**`profile-overlay-merge.sh`**:
1. 오버레이 파일 없음 → 프로파일만
2. 오버레이가 리프 필드를 override
3. 오버레이가 새 필드 추가
4. 잘못된 오버레이 → `load_profile` exit 1
5. `inherits`가 있는 오버레이 → 무시 + 경고
6. deep merge가 형제 필드를 보존

**`cmd-config.sh`**:
1. `set`이 파일 생성
2. `get`이 값 리턴
3. 두 번째 `set`이 이전 필드를 보존
4. `unset`이 대상 필드만 제거
5. 빈 부모 dict cascade 정리
6. `list`가 오버레이 전체 출력
7. 유효하지 않은 필드 (`..`) → exit 1
8. 없는 필드에 `get` → exit 1
9. JSON 값 `[1,2]` → 배열로 저장
10. 원자적 쓰기 (임시 파일 + rename, 파일 존재 확인으로 간접 검증)

**`smoke-logrot-integration.sh`** (mock 기반):
- `gstable` 실행 대신 쉘 루프(`while echo "block $i"; do ((i++)); sleep 0.01; done >> "$logfile"`)로 로그 파일에 지속적으로 append.
- `logrot "$logfile" 1K 3`을 동반 감시 프로세스로 실행 (§4.4와 일치하는 별도 프로세스).
- 5초 내에 `$logfile.1`이 나타나는지 검증 (logrot이 파일을 로테이션함).
- 검증 후 두 프로세스 모두 정리.

### 8.4 실행 시간 목표

- `bash tests/unit/run.sh` 전체 < 10초
- 개별 단위 테스트 < 2초
- `smoke-logrot-integration.sh` < 5초

### 8.5 회귀 테스트

`tests/regression/` 스크립트는 사용자에 의해 수정 중. 이 스펙은 추가/수정하지 않음. 수동 스모크 검증: 각 phase 후 `chainbench init && chainbench start && chainbench status`로 기존 동작 유지 확인.

### 8.6 CI

GitHub Actions 워크플로 없음. 추가는 범위 밖. `setup.sh`에 `bash tests/unit/run.sh` 수동 실행 안내 출력.

## 9. 구현 단계

9개 커밋, 각각 독립적으로 검증 가능.

| 단계 | 커밋 메시지 | 범위 |
|---|---|---|
| 0 | `chore: remove stale .bak files from previous refactor` | `.bak` 파일 3개 삭제 |
| 1 | `test: add unit test runner scaffolding` | `tests/unit/run.sh`, `lib/assert.sh`, 메타 스모크 테스트 |
| 2 | `test(common): pin down resolve_binary behavior with unit tests` | `common-resolve-binary.sh` + mock-gstable fixture |
| 3 | `feat(cli): add --binary-path / --logrot-path runtime overrides` | `_cb_parse_runtime_overrides` + init/start/restart/node의 호출부 + 파서 테스트 |
| 4 | `fix(profile): respect env vars and merge local overlay during profile load` | `_cb_set_var` env-first, Python 블록의 오버레이 병합, `.gitignore`, 테스트 파일 2개 |
| 5 | `feat(cli): add 'chainbench config' subcommand for local overlay` | `lib/cmd_config.sh` + 테스트 |
| 6 | `feat(logging): integrate logrot with hybrid discovery and auto-build` | `resolve_logrot`, `_cb_build_logrot_from_source`, `cmd_start.sh` 실행 파이프라인, `profiles/default.yaml`, `cmd_stop.sh` 메시지, mock-go fixture, logrot 테스트 |
| 7 | `feat(mcp): add binary_path override to lifecycle tools and config_set tool` | `shellEscapeArg`, `lifecycle.ts`, `node.ts`, `config.ts`, `schema.ts`, `index.ts` |
| 8 | `docs: document logrot integration and chainbench config command` | `README.md`, `setup.sh`, 나머지 주석 |

단계 의존성은 엄격히 선형. 긴급 롤백 지점:

- **Phase 3 완료 후:** 런타임 override 채널이 end-to-end로 동작 (오버레이 없음, logrot 없음), 사용자가 CLI를 통해 임의 바이너리를 지정할 수 있음.
- **Phase 6 완료 후:** 로그 로테이션이 완전 복원. 주요 가치 전달. MCP와 문서는 미완이지만 CLI는 완성.

## 10. 미결 사항 (구현 시 결정)

- **`node stop <N>` 시 Logrot 정리:** 현재 `node stop`은 PID로 gstable을 종료하지만 해당 노드의 동반 logrot 프로세스는 중지하지 않음. 고아 logrot은 무해(정적 파일에 대해 유휴)하며 `cmd_stop.sh`에서 정리됨. 노드별 logrot 정리가 필요하면 pids.json에 `pid`와 함께 `logrot_pid`를 저장하고 `node stop` 시 둘 다 종료. Phase 6 구현 중 결정.
- **`cmd_log.sh`에서의 로테이션 파일 glob:** `node1.log.1`, `node1.log.2` 등이 존재할 수 있게 되면 `log search`에 포함시켜야 하는지? 현재 범위는 비목표이지만, `_cb_log_all_log_files`의 한 줄 확장이 가치 있을 수 있음. Phase 6 리뷰 중 결정.
- **`cmd_restart.sh` override 전파:** 현재 `cmd_restart.sh` 구조를 Phase 3에서 검사해야 함. `bash chainbench.sh`로 자식 커맨드를 재실행하면 환경변수가 자동 전파됨. source하면 이미 전파됨. 추측 대신 검증.

이것들은 설계 결정이 아닌 구현 시점의 결정이며, 스펙 승인을 차단하지 않음.
