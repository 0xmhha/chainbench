# node script

> 출처: Confluence [node script](https://wemade.atlassian.net/wiki/spaces/platfomDev/pages/2611871801) (페이지 ID 2611871801, 버전 4, 최종 수정 2026-04-13)  
> 상위 페이지: [StableNet] Test  
> 가져온 날짜: 2026-09-18 (본문은 Confluence markdown 변환 결과를 그대로 옮김)

원본 페이지는 두 부분이다. 첫 코드 블록은 사용법 설명이고, 두 번째는 "node_ctrl.sh"라는 접기(expand) 블록 안의 스크립트 본문이다. 페이지에는 같은 이름의 첨부 파일 `node_ctrl.sh`(10,906바이트)도 붙어 있는데, 이 저장소에는 [attachments/node_ctrl.sh](attachments/node_ctrl.sh)로 넣었다. 첨부 파일과 아래 본문은 같은 스크립트다. 1st Test Script 페이지의 압축 파일에 든 `hardfork/node_ctrl.sh`(14,873바이트)는 이보다 나중 버전이라 내용이 다르다.

---

```
---

**기본 형식**
```bash
./node_ctrl.sh <user> <password> <command> <all | 서버번호>
```
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
| `del` | kill.sh 실행 → 종료 확인 → /data/stableNet/gstable 폴더 삭제 |
| `del_main` | kill.sh 실행 → 종료 확인 → /data/stableNet/mainnet 폴더 삭제 |
| `del_test` | kill.sh 실행 → 종료 확인 → /data/stableNet/testnet 폴더 삭제 |

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
./node_ctrl.sh <user> <pw> download all /data/stableNet/gstable/gstable.log /tmp/logs/

# 전체 서버 데이터 디렉토리 삭제 (private)
./node_ctrl.sh <user> <pw> del all

# 전체 서버 데이터 디렉토리 삭제 (testnet)
./node_ctrl.sh <user> <pw> del_test all
``` 

---

** 사전 준비**
- Mac 전용 스크립트입니다
- `sshpass` 미설치 시 스크립트 실행 시 자동으로 Homebrew를 통해 설치됩니다
- Homebrew가 없다면 먼저 설치 필요: https://brew.sh
```

**node_ctrl.sh** (원본에서는 접기 블록)

```
#!/bin/bash
# node_ctrl.sh - 원격 우분투 서버 노드 제어 스크립트 (Mac용)
# 사용법: ./node_ctrl.sh <user> <password> <command> <all|number>

# ===================== 설정 =====================
REMOTE_SCRIPT_DIR="/data/stableNet/script"
KILL_WAIT=1      # kill 후 프로세스 종료 대기 시간 (초)
RUN_WAIT=1       # run 후 프로세스 기동 대기 시간 (초)

IP_LIST=(
  "172.21.132.1"
  "172.21.132.2"
  "172.21.132.3"
  "172.21.132.4"
  "172.21.132.5"
  "172.21.132.6"
  "172.21.132.7"
  "172.21.132.8"
  "172.21.132.9"
  "172.21.132.10"
  "172.21.132.11"
  "172.21.132.12"
  "172.21.132.13"
  "172.21.132.14"
  "172.21.132.15"
)

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
  echo "    del             - kill.sh 실행 → 종료 확인 → /data/stableNet/gstable 폴더 삭제
    del_main        - kill.sh 실행 → 종료 확인 → /data/stableNet/mainnet 폴더 삭제
    del_test        - kill.sh 실행 → 종료 확인 → /data/stableNet/testnet 폴더 삭제"
  echo ""
  echo "  예시:"
  echo "    $0 ubuntu pw kill all"
  echo "    $0 ubuntu pw restart all"
  echo "    $0 ubuntu pw upload all /tmp/bin/gstable /data/bin/"
  echo "    $0 ubuntu pw upload 5  /tmp/bin/gstable /data/bin/"
  echo "    $0 ubuntu pw download all /data/stableNet/gstable/gstable.log /tmp/logs/"
  echo "    $0 ubuntu pw download 5  /data/stableNet/gstable/gstable.log /tmp/logs/"
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
    -P 10022 \
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
    -p 10022 \
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
  local scp_flags="-P 10022 -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o LogLevel=ERROR -o PubkeyAuthentication=no -o PreferredAuthentications=password"
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
    delete_remote "$ip" "$REMOTE_USER" "$PW" "/data/stableNet/gstable" "del"
  elif [ "$CMD" = "del_main" ]; then
    delete_remote "$ip" "$REMOTE_USER" "$PW" "/data/stableNet/mainnet" "del_main"
  elif [ "$CMD" = "del_test" ]; then
    delete_remote "$ip" "$REMOTE_USER" "$PW" "/data/stableNet/testnet" "del_test"
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
