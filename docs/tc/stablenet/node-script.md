# node script

> 출처: Confluence [node script](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2611871801) (페이지 ID 2611871801, 버전 4, 최종 수정 2026-04-13)  
> 상위 페이지: [StableNet] Test  
> 가져온 날짜: 2026-09-18  
> 보안 정리: 2026-09-28. 원본 스크립트에 박혀 있던 환경 의존 값만 `server-set.yaml` /
> `workspace-config.yaml` 로 옮겼다. 스크립트 구조와 실행 방식
> (`<user> <password> <command> <target>`)은 원본 그대로 유지한다. 두 설정 파일은
> `.gitignore` 로 추적 제외되며 커밋되지 않는다.

원본 페이지는 두 부분이다. 첫 코드 블록은 사용법 설명이고, 두 번째는 "node_ctrl.sh"라는 접기(expand) 블록 안의 스크립트 본문이다. 원본 페이지에는 같은 이름의 첨부 파일 `node_ctrl.sh`도 있었으나 이 저장소에는 두지 않았다(스크립트 내용은 아래 본문에 정리된 형태로 남아 있다). 1st Test Script 페이지의 압축 파일에 든 `hardfork/node_ctrl.sh`는 이보다 나중 버전이라 내용이 다르다.

---

## 설정 파일 (하드코딩됐던 값은 여기에 둔다)

원본 스크립트에 박혀 있던 서버 목록·포트·원격 경로를 아래 두 파일로 옮겼다. 두 파일은
chainbench 가 그대로 쓰는 설정 파일이며, 각 필드의 의미는 리포지토리 루트의
`*.sample.yaml` 이 문서화한다(여기서 재설명하지 않는다). 두 파일은 `docs/tc/stablenet/`
아래에 두며 `.gitignore` 로 추적 제외된다. 리포지토리에는 `*.sample.yaml` 만 추적된다.

아래 표의 "node_ctrl.sh 가 읽는 값"은 이 레거시 스크립트가 실제로 참조하는 하위 집합이다.
`server-set.yaml` 의 `ssh.user`·`ssh.password_file` 등 나머지 필드는 chainbench 가
사용하는 지원 필드로, 스키마대로 채워 둔다. node_ctrl.sh 자체는 계정·비밀번호를 실행
인자로 받는다(`<user> <password> ...`).

| 파일 | node_ctrl.sh 가 읽는 값 | 참조 형식 |
|---|---|---|
| `server-set.yaml` | 호스트 목록(15대), SSH 포트 | `server-set.sample.yaml`, `docs/dev/server-set.md` |
| `workspace-config.yaml` | 원격 대상의 `dataRoot`(원격 스크립트 위치·삭제 대상 경로 계산 기준) | `workspace-config.sample.yaml`, `docs/dev/config-files-guide.md` |

---

```
---

**기본 형식**
```bash
./node_ctrl.sh <user> <password> <command> <all | 서버번호>
```
- 계정·비밀번호는 실행 인자로 받는다(파일에 저장하지 않음)
- 호스트 목록과 SSH 포트는 `server-set.yaml`, 원격 경로는 `workspace-config.yaml` 에서 읽는다
- 서버 번호는 IP 끝자리 기준 (1 ~ 15)
- `all` 지정 시 전체 서버(15대)에 순차 실행

---

**주요 커맨드**

| command | 동작 |
|---|---|
| `kill` | 노드 프로세스 종료 |
| `run` | 프라이빗 노드 기동 |
| `run_mainnet` | 메인넷 기동 |
| `run_testnet` | 테스트넷 기동 |
| `restart` | kill → 종료 확인 → run (자동 검증 포함) |
| `restart_mainnet` | kill → 종료 확인 → run_mainnet |
| `restart_testnet` | kill → 종료 확인 → run_testnet |
| `init` | 초기화 |
| `upload` | 로컬 파일 → 원격 서버 전송 |
| `download` | 원격 서버 파일 → 로컬 수신 |
| `del` | kill.sh 실행 → 종료 확인 → `<dataRoot>/gstable` 폴더 삭제 |
| `del_main` | kill.sh 실행 → 종료 확인 → `<dataRoot>/mainnet` 폴더 삭제 |
| `del_test` | kill.sh 실행 → 종료 확인 → `<dataRoot>/testnet` 폴더 삭제 |

---

**사용 예시**
```bash
# 전체 노드 시작 (private)
./node_ctrl.sh <user> <pw> run all

# 전체 노드 종료
./node_ctrl.sh <user> <pw> kill all

# 전체 노드 재시작 (private)
./node_ctrl.sh <user> <pw> restart all

# 5번 서버만 메인넷으로 재시작
./node_ctrl.sh <user> <pw> restart_mainnet 5

# 바이너리 전체 배포
./node_ctrl.sh <user> <pw> upload all /tmp/bin/gstable /data/bin/

# 특정 서버(5번)에만 배포
./node_ctrl.sh <user> <pw> upload 5 /tmp/bin/gstable /data/bin/

# 전체 서버 로그 수집 (파일명_끝자리IP 로 저장됨)
./node_ctrl.sh <user> <pw> download all <dataRoot>/gstable/gstable.log /tmp/logs/

# 전체 서버 데이터 디렉토리 삭제 (private)
./node_ctrl.sh <user> <pw> del all

# 전체 서버 데이터 디렉토리 삭제 (testnet)
./node_ctrl.sh <user> <pw> del_test all
``` 

---

** 사전 준비**
- Mac 전용 스크립트입니다
- 같은 디렉터리에 `server-set.yaml` 과 `workspace-config.yaml` 이 있어야 합니다
  (각각 루트의 `*.sample.yaml` 을 복사해 실제 값을 채웁니다)
- `yq`(설정 파일 파싱) 필요 — 미설치 시 스크립트가 자동 설치를 시도합니다
- `sshpass` 미설치 시 스크립트 실행 시 자동으로 Homebrew를 통해 설치됩니다
- Homebrew가 없다면 먼저 설치 필요: https://brew.sh
```

**node_ctrl.sh** (원본에서는 접기 블록)

> 원본 대비 바뀐 곳은 "설정" 블록뿐이다(IP_LIST·포트·경로를 설정 파일에서 읽음).
> 나머지 함수와 실행 흐름은 원본 그대로다. 실제 서버에 대한 실행 검증은 하지 않았다.

```
#!/bin/bash
# node_ctrl.sh - 원격 우분투 서버 노드 제어 스크립트 (Mac용)
# 사용법: ./node_ctrl.sh <user> <password> <command> <all|number>

# ===================== 설정 =====================
# 서버 목록·포트·경로는 하드코딩하지 않고 아래 두 설정 파일에서 읽는다.
# 두 파일은 이 스크립트와 같은 디렉터리에 있어야 하며 커밋되지 않는다.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_SET="${SCRIPT_DIR}/server-set.yaml"
WORKSPACE_CONFIG="${SCRIPT_DIR}/workspace-config.yaml"

KILL_WAIT=1      # kill 후 프로세스 종료 대기 시간 (초)
RUN_WAIT=1       # run 후 프로세스 기동 대기 시간 (초)

# yq 확인 (설정 파일 파싱용)
if ! command -v yq &>/dev/null; then
  echo "[알림] yq가 설치되어 있지 않습니다. 자동 설치를 시도합니다..."
  if ! command -v brew &>/dev/null; then
    echo "[오류] Homebrew가 없습니다: https://brew.sh"
    exit 1
  fi
  brew install yq
fi

[ -f "$SERVER_SET" ] || { echo "[오류] server-set.yaml 이 없습니다: ${SERVER_SET}"; exit 1; }
[ -f "$WORKSPACE_CONFIG" ] || { echo "[오류] workspace-config.yaml 이 없습니다: ${WORKSPACE_CONFIG}"; exit 1; }

# 호스트 목록 (server-set.yaml, 소비 순서 그대로)
IFS=$'\n' read -r -d '' -a IP_LIST < <(yq '.pool.hosts[].addr' "$SERVER_SET" && printf '\0')

# SSH 포트 (server-set.yaml)
SSH_PORT="$(yq '.ssh.port' "$SERVER_SET")"

# 원격 데이터 루트 (workspace-config.yaml). 원격 스크립트 위치·삭제 대상 경로의 기준.
DATA_ROOT="$(yq '.dataRoot' "$WORKSPACE_CONFIG")"
REMOTE_SCRIPT_DIR="${DATA_ROOT}/script"

VALID_COMMANDS=("kill" "init" "run" "run_mainnet" "run_testnet" "restart" "restart_mainnet" "restart_testnet" "upload" "download" "del" "del_main" "del_test")
# ================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

usage() {
  echo ""
  echo "사용법:"
  echo "  $0 <user> <password> <command> <all|number>"
  echo "  $0 <user> <password> upload <all|number> <local_file> <remote_path>"
  echo "  $0 <user> <password> download <all|number> <remote_file> <local_path>"
  echo ""
  echo "  command 목록:"
  echo "    kill            - kill.sh 실행"
  echo "    init            - init.sh 실행"
  echo "    run             - run.sh 실행"
  echo "    run_mainnet     - run_mainnet.sh 실행"
  echo "    run_testnet     - run_testnet.sh 실행"
  echo "    restart         - kill.sh → 종료 확인 → run.sh → 기동 확인"
  echo "    restart_mainnet - kill.sh → 종료 확인 → run_mainnet.sh → 기동 확인"
  echo "    restart_testnet - kill.sh → 종료 확인 → run_testnet.sh → 기동 확인"
  echo "    upload          - 로컬 파일을 원격 서버로 전송"
  echo "    download        - 원격 서버 파일을 로컬로 수신 (all 시 파일명_끝자리IP로 저장)"
  echo "    del             - kill.sh 실행 → 종료 확인 → ${DATA_ROOT}/gstable 폴더 삭제
    del_main        - kill.sh 실행 → 종료 확인 → ${DATA_ROOT}/mainnet 폴더 삭제
    del_test        - kill.sh 실행 → 종료 확인 → ${DATA_ROOT}/testnet 폴더 삭제"
  echo ""
  echo "  예시:"
  echo "    $0 ubuntu pw kill all"
  echo "    $0 ubuntu pw restart all"
  echo "    $0 ubuntu pw upload all /tmp/bin/gstable /data/bin/"
  echo "    $0 ubuntu pw upload 5  /tmp/bin/gstable /data/bin/"
  echo "    $0 ubuntu pw download all ${DATA_ROOT}/gstable/gstable.log /tmp/logs/"
  echo "    $0 ubuntu pw download 5  ${DATA_ROOT}/gstable/gstable.log /tmp/logs/"
  echo "    $0 ubuntu pw del all"
  echo ""
  echo "  사용 가능한 서버 번호: 1 ~ ${#IP_LIST[@]}"
  echo ""
  exit 1
}

check_sshpass() {
  if ! command -v sshpass &>/dev/null; then
    echo -e "${YELLOW}[알림] sshpass가 설치되어 있지 않습니다. 자동 설치를 시도합니다...${NC}"
    if ! command -v brew &>/dev/null; then
      echo -e "${RED}[오류] Homebrew가 설치되어 있지 않습니다.${NC}"
      echo "  Homebrew 설치: https://brew.sh"
      exit 1
    fi
    brew install sshpass
    if ! command -v sshpass &>/dev/null; then
      echo -e "${RED}[오류] sshpass 설치에 실패했습니다.${NC}"
      exit 1
    fi
    echo -e "${GREEN}[완료] sshpass 설치 성공${NC}"
  fi
}

scp_exec() {
  local direction="$1"   # upload | download
  local ip="$2"
  local user="$3"
  local pw="$4"
  local src="$5"
  local dst="$6"
  sshpass -p "$pw" scp \
    -P "$SSH_PORT" \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -o LogLevel=ERROR \
    -o PubkeyAuthentication=no \
    -o PreferredAuthentications=password \
    "$src" "$dst"
}

ssh_exec() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  local cmd="$4"
  sshpass -p "$pw" ssh \
    -p "$SSH_PORT" \
    -o StrictHostKeyChecking=no \
    -o ConnectTimeout=10 \
    -o LogLevel=ERROR \
    -o PubkeyAuthentication=no \
    -o PreferredAuthentications=password \
    "${user}@${ip}" \
    "$cmd"
}

is_running() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  ssh_exec "$ip" "$user" "$pw" \
    "ps aux | grep gstable | grep -v grep > /dev/null 2>&1 && echo 'yes' || echo 'no'"
}

run_remote() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  local script_name="$4"

  echo -e "${CYAN}[$ip]${NC} ${script_name} 실행 중..."
  local output
  output=$(ssh_exec "$ip" "$user" "$pw" \
    "echo '${pw}' | sudo -S bash ${REMOTE_SCRIPT_DIR}/${script_name} 2>&1")
  local exit_code=$?

  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] 오류 발생 (exit code: ${exit_code})${NC}"
    echo "$output" | sed "s/^/  [$ip] /"
    return 1
  fi

  echo -e "${GREEN}[$ip] 완료${NC}"
  return 0
}

restart_remote() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  local run_script="${4:-run.sh}"

  # 1. kill
  echo -e "${CYAN}[$ip]${NC} [restart] kill.sh 실행 중..."
  local output
  output=$(ssh_exec "$ip" "$user" "$pw" \
    "echo '${pw}' | sudo -S bash ${REMOTE_SCRIPT_DIR}/kill.sh 2>&1")
  local exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] kill.sh 오류 (exit code: ${exit_code})${NC}"
    echo "$output" | sed "s/^/  [$ip] /"
    return 1
  fi

  # 2. 프로세스 종료 확인
  echo -e "${CYAN}[$ip]${NC} [restart] 프로세스 종료 대기 (${KILL_WAIT}초)..."
  sleep "$KILL_WAIT"
  local status
  status=$(is_running "$ip" "$user" "$pw")
  if [ "$status" = "yes" ]; then
    echo -e "${RED}[$ip] [restart] 오류: kill 후에도 gstable 프로세스가 종료되지 않았습니다.${NC}"
    return 1
  fi
  echo -e "${CYAN}[$ip]${NC} [restart] 프로세스 종료 확인"

  # 3. run
  echo -e "${CYAN}[$ip]${NC} [restart] ${run_script} 실행 중..."
  output=$(ssh_exec "$ip" "$user" "$pw" \
    "echo '${pw}' | sudo -S bash ${REMOTE_SCRIPT_DIR}/${run_script} 2>&1")
  exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] run.sh 오류 (exit code: ${exit_code})${NC}"
    echo "$output" | sed "s/^/  [$ip] /"
    return 1
  fi

  # 4. 프로세스 기동 확인
  echo -e "${CYAN}[$ip]${NC} [restart] 프로세스 기동 대기 (${RUN_WAIT}초)..."
  sleep "$RUN_WAIT"
  status=$(is_running "$ip" "$user" "$pw")
  if [ "$status" = "no" ]; then
    echo -e "${RED}[$ip] [restart] 오류: run 후 gstable 프로세스가 기동되지 않았습니다.${NC}"
    return 1
  fi

  echo -e "${GREEN}[$ip] [restart] 완료 (종료 확인 → 기동 확인)${NC}"
  return 0
}

upload_remote() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  local local_file="$4"
  local remote_path="$5"

  if [ ! -e "$local_file" ]; then
    echo -e "${RED}[$ip] 오류: 로컬 파일/디렉토리를 찾을 수 없습니다: ${local_file}${NC}"
    return 1
  fi

  local filename
  filename=$(basename "$local_file")
  local tmp_path="/tmp/${filename}"

  echo -e "${CYAN}[$ip]${NC} 업로드 중: ${local_file} → ${user}@${ip}:${remote_path}"

  # 1단계: /tmp/ 에 업로드
  local scp_flags="-P ${SSH_PORT} -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o LogLevel=ERROR -o PubkeyAuthentication=no -o PreferredAuthentications=password"
  if [ -d "$local_file" ]; then
    sshpass -p "$pw" scp $scp_flags -r "$local_file" "${user}@${ip}:${tmp_path}"
  else
    sshpass -p "$pw" scp $scp_flags "$local_file" "${user}@${ip}:${tmp_path}"
  fi
  local exit_code=$?

  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] 업로드 오류 (exit code: ${exit_code}) 퍼미션 에러 확인${NC}"
    return 1
  fi

  # 2단계: sudo로 최종 경로로 이동
  ssh_exec "$ip" "$user" "$pw" \
    "echo '${pw}' | sudo -S mv -f '${tmp_path}' '${remote_path}' 2>&1"
  exit_code=$?

  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] sudo mv 오류 (exit code: ${exit_code})${NC}"
    return 1
  fi

  echo -e "${GREEN}[$ip] 업로드 완료${NC}"
  return 0
}

delete_remote() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  local target_path="$4"
  local label="$5"

  # 1. kill
  echo -e "${CYAN}[$ip]${NC} [${label}] kill.sh 실행 중..."
  local output
  output=$(ssh_exec "$ip" "$user" "$pw" \
    "echo '${pw}' | sudo -S bash ${REMOTE_SCRIPT_DIR}/kill.sh 2>&1")
  local exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] kill.sh 오류 (exit code: ${exit_code})${NC}"
    echo "$output" | sed "s/^/  [$ip] /"
    return 1
  fi

  # 2. 프로세스 종료 확인
  echo -e "${CYAN}[$ip]${NC} [${label}] 프로세스 종료 대기 (${KILL_WAIT}초)..."
  sleep "$KILL_WAIT"
  local status
  status=$(is_running "$ip" "$user" "$pw")
  if [ "$status" = "yes" ]; then
    echo -e "${RED}[$ip] [${label}] 오류: kill 후에도 gstable 프로세스가 종료되지 않았습니다.${NC}"
    return 1
  fi
  echo -e "${CYAN}[$ip]${NC} [${label}] 프로세스 종료 확인"

  # 3. 폴더 삭제
  echo -e "${CYAN}[$ip]${NC} [${label}] ${target_path} 삭제 중..."
  output=$(ssh_exec "$ip" "$user" "$pw" \
    "echo '${pw}' | sudo -S rm -rf ${target_path} 2>&1")
  exit_code=$?
  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] [${label}] 폴더 삭제 오류 (exit code: ${exit_code})${NC}"
    echo "$output" | sed "s/^/  [$ip] /"
    return 1
  fi

  echo -e "${GREEN}[$ip] [${label}] 완료 (종료 확인 → 폴더 삭제)${NC}"
  return 0
}

download_remote() {
  local ip="$1"
  local user="$2"
  local pw="$3"
  local remote_file="$4"
  local local_path="$5"
  local rename="$6"   # yes | no

  mkdir -p "$local_path"

  local filename
  filename=$(basename "$remote_file")
  local last_octet="${ip##*.}"

  local dest_file
  if [ "$rename" = "yes" ]; then
    local name="${filename%.*}"
    local ext="${filename##*.}"
    if [ "$name" = "$ext" ]; then
      dest_file="${local_path}/${name}_${last_octet}"
    else
      dest_file="${local_path}/${name}_${last_octet}.${ext}"
    fi
  else
    dest_file="${local_path}/${filename}"
  fi

  echo -e "${CYAN}[$ip]${NC} 다운로드 중: ${user}@${ip}:${remote_file} → ${dest_file}"
  scp_exec download "$ip" "$user" "$pw" \
    "${user}@${ip}:${remote_file}" \
    "$dest_file"
  local exit_code=$?

  if [ $exit_code -ne 0 ]; then
    echo -e "${RED}[$ip] 다운로드 오류 (exit code: ${exit_code})${NC}"
    return 1
  fi
  echo -e "${GREEN}[$ip] 다운로드 완료: ${dest_file}${NC}"
  return 0
}

# ===================== 인자 검증 =====================
if [ $# -lt 4 ]; then
  echo -e "${RED}[오류] 인자가 부족합니다.${NC}"
  usage
fi

REMOTE_USER="$1"
PW="$2"
CMD="$3"
TARGET="$4"
ARG1="$5"   # upload: local_file  / download: remote_file
ARG2="$6"   # upload: remote_path / download: local_path

# 명령어 유효성 검사
VALID=false
for c in "${VALID_COMMANDS[@]}"; do
  if [ "$c" = "$CMD" ]; then
    VALID=true
    break
  fi
done

if [ "$VALID" = false ]; then
  echo -e "${RED}[오류] 알 수 없는 명령어: ${CMD}${NC}"
  usage
fi

# upload/download 추가 인자 검사
if [ "$CMD" = "upload" ] || [ "$CMD" = "download" ]; then
  if [ -z "$ARG1" ] || [ -z "$ARG2" ]; then
    echo -e "${RED}[오류] upload/download 는 파일 경로 인자가 필요합니다.${NC}"
    usage
  fi
fi

check_sshpass

# ===================== 실행 =====================
execute() {
  local ip="$1"
  local rename="$2"
  if [ "$CMD" = "restart" ]; then
    restart_remote "$ip" "$REMOTE_USER" "$PW" "run.sh"
  elif [ "$CMD" = "restart_mainnet" ]; then
    restart_remote "$ip" "$REMOTE_USER" "$PW" "run_mainnet.sh"
  elif [ "$CMD" = "restart_testnet" ]; then
    restart_remote "$ip" "$REMOTE_USER" "$PW" "run_testnet.sh"
  elif [ "$CMD" = "del" ]; then
    delete_remote "$ip" "$REMOTE_USER" "$PW" "${DATA_ROOT}/gstable" "del"
  elif [ "$CMD" = "del_main" ]; then
    delete_remote "$ip" "$REMOTE_USER" "$PW" "${DATA_ROOT}/mainnet" "del_main"
  elif [ "$CMD" = "del_test" ]; then
    delete_remote "$ip" "$REMOTE_USER" "$PW" "${DATA_ROOT}/testnet" "del_test"
  elif [ "$CMD" = "upload" ]; then
    upload_remote "$ip" "$REMOTE_USER" "$PW" "$ARG1" "$ARG2"
  elif [ "$CMD" = "download" ]; then
    download_remote "$ip" "$REMOTE_USER" "$PW" "$ARG1" "$ARG2" "$rename"
  else
    run_remote "$ip" "$REMOTE_USER" "$PW" "${CMD}.sh"
  fi
}

if [ "$TARGET" = "all" ]; then
  echo -e "${YELLOW}=== 전체 서버 (${#IP_LIST[@]}대) ${CMD} 실행 ===${NC}"
  if [[ "$CMD" =~ ^(run|run_testnet|run_mainnet|restart|restart_testnet|restart_mainnet)$ ]]; then
    for (( i=${#IP_LIST[@]}-1; i>=0; i-- )); do
      execute "${IP_LIST[$i]}" "yes"
    done
  else
    for ip in "${IP_LIST[@]}"; do
      execute "$ip" "yes"
    done
  fi
  echo -e "${YELLOW}=== 전체 완료 ===${NC}"

else
  if ! [[ "$TARGET" =~ ^[0-9]+$ ]]; then
    echo -e "${RED}[오류] 타겟은 'all' 또는 숫자(IP 끝자리)여야 합니다.${NC}"
    usage
  fi

  FOUND=false
  for ip in "${IP_LIST[@]}"; do
    LAST_OCTET="${ip##*.}"
    if [ "$LAST_OCTET" = "$TARGET" ]; then
      echo -e "${YELLOW}=== ${ip} ${CMD} 실행 ===${NC}"
      execute "$ip" "no"
      FOUND=true
      break
    fi
  done

  if [ "$FOUND" = false ]; then
    echo -e "${RED}[오류] IP 끝자리가 ${TARGET}인 서버를 찾을 수 없습니다.${NC}"
    echo "  사용 가능한 번호: 1 ~ ${#IP_LIST[@]}"
    exit 1
  fi
fi
```
