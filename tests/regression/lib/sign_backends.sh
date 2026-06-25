#!/usr/bin/env bash
# tests/regression/lib/sign_backends.sh — 서명 백엔드 (W6/W7)
#
# accessors.sh의 tx_send_as / tx_send_fee_delegate_as 가 활성 프로파일의 CB_SIGN_BACKEND에 따라
# 아래 함수로 디스패치한다. 백엔드만 (필요 시) 키를 다루며, 키를 stdout으로 반환하지 않는다.
#
#   client_cast   — 로컬: 공개 테스트 키로 클라이언트 서명(cast). 키는 비밀 아님.
#   node_keystore — 폐쇄망(방식 B): 노드 keystore unlock + eth_sendTransaction(노드측 서명).
#                   키가 서버 밖으로 나오지 않음.

[[ -n "${_CB_SIGN_BACKENDS_LOADED:-}" ]] && return 0
_CB_SIGN_BACKENDS_LOADED=1

# ----------------------------------------------------------------------------
# client_cast (local)
# ----------------------------------------------------------------------------

# cb_sign_client_cast <acct_idx> <to> <value> [data] [gas] [type] [tip] [feecap]
cb_sign_client_cast() {
  local idx="$1" to="$2" value="$3" data="${4:-}" gas="${5:-21000}" type="${6:-dynamic}" tip="${7:-}" feecap="${8:-}"
  local pk="${CB_ACCT_PK[$idx]:-}"
  if [[ -z "$pk" ]]; then
    printf '[ERROR] cb_sign_client_cast: no private key for account index %s\n' "$idx" >&2
    return 1
  fi
  # EOA 전송은 어느 노드든 raw tx를 받으므로 기본 노드(1)로 보낸다.
  send_raw_tx "$(node 1)" "$pk" "$to" "$value" "$data" "$gas" "$type" "$tip" "$feecap"
}

# ----------------------------------------------------------------------------
# node_keystore (closednet, 방식 B)
# ----------------------------------------------------------------------------

# cb_sign_node_keystore <acct_idx> <to> <value> [data] [gas] [type] [tip] [feecap]
# 계정의 홈 노드에서 unlock 후 eth_sendTransaction(노드측 서명). 원시 키를 만지지 않는다.
# 홈 노드 가정: 계정 i = 노드 i 의 keystore (W13에서 인프라 확인).
cb_sign_node_keystore() {
  local idx="$1" to="$2" value="$3" data="${4:-}" gas="${5:-21000}" type="${6:-dynamic}" tip="${7:-}" feecap="${8:-}"
  local from tgt
  from=$(acct_addr "$idx")
  tgt=$(node "$idx")
  if [[ -z "$from" ]]; then
    printf '[ERROR] cb_sign_node_keystore: no address for account index %s\n' "$idx" >&2
    return 1
  fi

  unlock_account "$tgt" "$from" "$CB_VALIDATOR_KEYSTORE_PASSWORD" 600 || \
    printf '[WARN]  cb_sign_node_keystore: unlock failed for %s on %s (계정이 이 노드 keystore에 없을 수 있음)\n' "$from" "$tgt" >&2

  local base_fee priority_fee max_fee
  base_fee=$(get_base_fee "$tgt")
  priority_fee=$(get_priority_fee "$tgt")
  [[ -n "$tip" ]] && priority_fee="$tip"
  if [[ -n "$feecap" ]]; then max_fee="$feecap"; else max_fee="$(( base_fee + priority_fee ))"; fi

  local fields="\"from\":\"${from}\""
  [[ -n "$to" ]] && fields="${fields},\"to\":\"${to}\""
  fields="${fields},\"value\":\"$(dec_to_hex "$value")\",\"gas\":\"$(dec_to_hex "$gas")\""
  if [[ -n "$data" ]]; then
    [[ "$data" != 0x* ]] && data="0x$data"
    fields="${fields},\"data\":\"${data}\""
  fi
  case "$type" in
    legacy)
      fields="${fields},\"gasPrice\":\"$(dec_to_hex "$max_fee")\""
      ;;
    accesslist)
      fields="${fields},\"type\":\"0x1\",\"gasPrice\":\"$(dec_to_hex "$max_fee")\",\"accessList\":[]"
      ;;
    dynamic|*)
      fields="${fields},\"type\":\"0x2\",\"maxFeePerGas\":\"$(dec_to_hex "$max_fee")\",\"maxPriorityFeePerGas\":\"$(dec_to_hex "$priority_fee")\""
      ;;
  esac

  local response hash
  response=$(rpc "$tgt" "eth_sendTransaction" "[{${fields}}]") || return 1
  hash=$(json_get "$response" "result")
  if [[ -z "$hash" || "$hash" == "null" ]]; then
    local err
    err=$(printf '%s' "$response" | jq -r '.error.message // "unknown"' 2>/dev/null)
    printf '[ERROR] cb_sign_node_keystore: send failed (from=%s tgt=%s): %s\n' "$from" "$tgt" "$err" >&2
    return 1
  fi
  printf '[INFO]  txHash=%s\n' "$hash" >&2
  printf '%s\n' "$hash"
}

# ----------------------------------------------------------------------------
# 수수료 위임 (fee delegation) — sender+feepayer 이중 서명 (fee_delegate.py)
# 키는 env(FD_SENDER_PK/FD_FEE_PAYER_PK)로만 주입한다. argv 금지.
# ----------------------------------------------------------------------------

_cb_fd_invoke() {
  # _cb_fd_invoke <rpc_url> <sender_pk> <feepayer_pk> <to> <value> <gas> [data]
  local url="$1" s_pk="$2" f_pk="$3" to="$4" value="$5" gas="$6" data="${7:-}"
  local helper="${CHAINBENCH_DIR}/tests/regression/lib/fee_delegate.py"
  local args=(send --rpc "$url" --to "$to" --value "$value" --gas "$gas")
  [[ -n "$data" ]] && args+=(--data "$data")
  local out
  out=$(FD_SENDER_PK="$s_pk" FD_FEE_PAYER_PK="$f_pk" python3 "$helper" "${args[@]}" 2>&1) || {
    printf '[ERROR] fee_delegate send failed\n' >&2
    return 1
  }
  printf '%s' "$out" | jq -r '.txHash // empty'
}

# cb_fd_client_cast <sender_idx> <feepayer_idx> <to> <value> [data] [gas]
cb_fd_client_cast() {
  local s_idx="$1" f_idx="$2" to="$3" value="$4" data="${5:-}" gas="${6:-21000}"
  local s_pk f_pk url
  s_pk="${CB_ACCT_PK[$s_idx]:-}"; f_pk="${CB_ACCT_PK[$f_idx]:-}"
  if [[ -z "$s_pk" || -z "$f_pk" ]]; then
    printf '[ERROR] cb_fd_client_cast: missing key(s) for sender %s / feepayer %s\n' "$s_idx" "$f_idx" >&2
    return 1
  fi
  url=$(get_node_url "$(node 1)") || return 1
  _cb_fd_invoke "$url" "$s_pk" "$f_pk" "$to" "$value" "$gas" "$data"
}

# cb_fd_node_keystore <sender_idx> <feepayer_idx> <to> <value> [data] [gas]
# 방식 B 예외2: 위임 tx는 이중 서명이라 eth_sendTransaction 단독 불가.
#   1순위(권장): gstable이 위임 tx 노드측 서명(personal_signTransaction)을 지원하면 그쪽 — W13 확인.
#   현재: secret store(closednet.keys)의 키로 격리 클라이언트 서명. 키는 서브셸 내부에서만 로드, 출력 금지.
cb_fd_node_keystore() {
  local s_idx="$1" f_idx="$2" to="$3" value="$4" data="${5:-}" gas="${6:-21000}"
  local keyfile="${CB_SECRET_DIR}/closednet.keys"
  if [[ ! -f "$keyfile" ]]; then
    printf '[ERROR] cb_fd_node_keystore: 위임 서명 키가 필요. %s 준비 필요(secret.example 참고). 방식B 위임-서명 RPC 지원 시 그쪽 우선.\n' "$keyfile" >&2
    return 1
  fi
  local s_pk f_pk url
  s_pk=$(awk -v i="$s_idx" '$1==i{print $2; exit}' "$keyfile")
  f_pk=$(awk -v i="$f_idx" '$1==i{print $2; exit}' "$keyfile")
  if [[ -z "$s_pk" || -z "$f_pk" ]]; then
    printf '[ERROR] cb_fd_node_keystore: secret store에 sender %s / feepayer %s 키 없음\n' "$s_idx" "$f_idx" >&2
    return 1
  fi
  url=$(get_node_url "$(node "$s_idx")") || return 1
  _cb_fd_invoke "$url" "$s_pk" "$f_pk" "$to" "$value" "$gas" "$data"
}
