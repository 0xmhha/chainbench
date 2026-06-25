#!/usr/bin/env bash
# tests/regression/lib/accessors.sh — 환경-중립 접근자 (테스트 본문이 쓰는 유일한 환경 API)
#
# 테스트 본문은 물리값(URL/키/IP) 대신 논리 인덱스만 쓴다. 활성 프로파일(tests/env/profile.sh)이
# 채운 CB_* 배열/변수를 여기서 해석한다.
#
# 불변식: 원시 private key를 반환하는 함수(acct_pk 류)는 두지 않는다.
#         서명은 tx_send_as 로 활성 백엔드에 위임하고, 백엔드만 키를 (필요 시) 내부에서 다룬다.

[[ -n "${_CB_ACCESSORS_LOADED:-}" ]] && return 0
_CB_ACCESSORS_LOADED=1

# node <idx> → 타깃 문자열 ("1" | "@stablenet-bp1")
node() { printf "${CB_TARGET_FMT}" "$1"; }

# acct_addr <idx> → 계정 주소(공개)
acct_addr() { printf '%s' "${CB_ACCT_ADDR[$1]:-}"; }

# validator_addr <idx> → 검증자 주소(공개)
validator_addr() { printf '%s' "${CB_VALIDATOR_ADDR[$1]:-}"; }

# validator_target <idx> → 검증자의 홈 노드 타깃 (= 동일 인덱스 노드)
validator_target() { node "$1"; }

# tx_send_as <acct_idx> <to> <value> [data] [gas] [type]
#   서명을 활성 백엔드(CB_SIGN_BACKEND)로 위임한다. 원시 키는 호출자/테스트에 노출되지 않는다.
#   stdout 으로 txHash(0x..)만 반환.
#   백엔드 구현: W6 — cb_sign_client_cast / cb_sign_node_keystore
tx_send_as() {
  "cb_sign_${CB_SIGN_BACKEND}" "$@"
}

# tx_send_fee_delegate_as <sender_idx> <feepayer_idx> <to> <value> [data] [gas]
#   수수료 위임 tx. 백엔드 구현: W7 — cb_fd_client_cast / cb_fd_node_keystore
tx_send_fee_delegate_as() {
  "cb_fd_${CB_SIGN_BACKEND}" "$@"
}
