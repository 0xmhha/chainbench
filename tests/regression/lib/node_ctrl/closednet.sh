#!/usr/bin/env bash
# tests/regression/lib/node_ctrl/closednet.sh — 폐쇄망 노드제어 백엔드 (W8)
#
# 폐쇄망 Ubuntu 서버군을 SSH로 제어한다. 테스트 하네스가 런타임에 필요한 것은 ensure(전체
# running 보장)뿐이다. 업로드/스위치 등 운영 명령은 별도 운영 도구(node_ctrl.sh)의 몫.
#
# ★ 비밀 경계:
#   - SSH 자격증명/호스트는 secret store에서만 읽는다(LLM·커밋에 노출 금지).
#       ${CB_SECRET_DIR}/closednet.ssh    : CB_SSH_USER, CB_SSH_PASSWORD, CB_SSH_PORT,
#                                           REMOTE_SCRIPT_DIR(기본 /data/stableNet/script),
#                                           RUN_SCRIPT(기본 run_testnet.sh)
#       ${CB_SECRET_DIR}/closednet.hosts  : 한 줄당 노드 IP (내부망 IP — 결정#2로 secret)
#   - 비밀값을 echo/log 하지 않는다. sshpass 비번은 env로 전달(argv 노출 회피).
#
# ★ W13 인프라 확인 대상: 원격 스크립트 경로/이름, sudo 방식, gstable 프로세스명.

[[ -n "${_CB_NODECTRL_CLOSEDNET_LOADED:-}" ]] && return 0
_CB_NODECTRL_CLOSEDNET_LOADED=1

_cb_cn_load_creds() {
  local ssh_file="${CB_SECRET_DIR}/closednet.ssh"
  if [[ ! -f "$ssh_file" ]]; then
    printf '[ERROR] node_ctrl/closednet: %s 없음. secret.example/closednet.ssh.example 참고.\n' "$ssh_file" >&2
    return 1
  fi
  # shellcheck disable=SC1090
  source "$ssh_file"
  : "${CB_SSH_PORT:=10022}"
  : "${REMOTE_SCRIPT_DIR:=/data/stableNet/script}"
  : "${RUN_SCRIPT:=run_testnet.sh}"
  if [[ -z "${CB_SSH_USER:-}" || -z "${CB_SSH_PASSWORD:-}" ]]; then
    printf '[ERROR] node_ctrl/closednet: CB_SSH_USER/CB_SSH_PASSWORD 미설정\n' >&2
    return 1
  fi
}

_cb_cn_hosts() {
  local hosts_file="${CB_SECRET_DIR}/closednet.hosts"
  [[ -f "$hosts_file" ]] || { printf '[ERROR] node_ctrl/closednet: %s 없음\n' "$hosts_file" >&2; return 1; }
  grep -vE '^\s*(#|$)' "$hosts_file"
}

# 비번을 argv가 아니라 env(SSHPASS)로 전달 → 프로세스 목록 노출 회피.
_cb_cn_ssh() {
  local host="$1" cmd="$2"
  SSHPASS="$CB_SSH_PASSWORD" sshpass -e ssh \
    -p "$CB_SSH_PORT" \
    -o StrictHostKeyChecking=no -o ConnectTimeout=10 -o LogLevel=ERROR \
    -o PubkeyAuthentication=no -o PreferredAuthentications=password \
    "${CB_SSH_USER}@${host}" "$cmd"
}

_cb_cn_is_running() {
  local host="$1"
  _cb_cn_ssh "$host" "ps aux | grep gstable | grep -v grep > /dev/null 2>&1 && echo yes || echo no" 2>/dev/null
}

_cb_cn_run() {
  local host="$1"
  _cb_cn_ssh "$host" "echo '${CB_SSH_PASSWORD}' | sudo -S bash ${REMOTE_SCRIPT_DIR}/${RUN_SCRIPT}" >/dev/null 2>&1
}

# cb_nodectrl_closednet_ensure
# stopped 노드를 병렬 기동 후 전체 running이 될 때까지 대기(최대 60초).
cb_nodectrl_closednet_ensure() {
  _cb_cn_load_creds || return 1
  local hosts
  hosts=$(_cb_cn_hosts) || return 1

  local max_wait=60 elapsed=0
  while (( elapsed < max_wait )); do
    local stopped=0 pids=()
    local host
    while IFS= read -r host; do
      [[ -z "$host" ]] && continue
      if [[ "$(_cb_cn_is_running "$host")" != "yes" ]]; then
        printf '[INFO]  node %s stopped — starting\n' "$host" >&2
        _cb_cn_run "$host" &
        pids+=($!)
        stopped=$(( stopped + 1 ))
      fi
    done <<< "$hosts"

    if (( stopped == 0 )); then
      (( elapsed > 0 )) && printf '[INFO]  all closednet nodes running (elapsed %ds)\n' "$elapsed" >&2
      return 0
    fi
    local p
    for p in "${pids[@]}"; do wait "$p" 2>/dev/null || true; done
    sleep 1
    elapsed=$(( elapsed + 1 ))
  done

  printf '[WARN]  some closednet nodes still stopped after %ds — proceeding\n' "$max_wait" >&2
}
